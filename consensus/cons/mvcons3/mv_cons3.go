/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
/*
Implementation of multi-valued consensus in the likeness of PBFT.
*/
package mvcons3

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

/* TODO: this may require infinite concurrent not yet instances running, but currently there is a max threshold set
by config.KeepFuture. To fix should always allow new instances once we have received n-t messages from the current instance. */

// what type of decision has happened (if any)
type decisionType int

const (
	undecided    decisionType = iota // the index has not yet decided
	decidedNil                       // the index has decided nil
	decidedValue                     // the index has decided a value
)

// MvCons3 is a rotating-coordinator-based multi-value consensus, where the coordinator rotates for each index.
// Instead of executing a single consensus index in rounds, multiple indecies are run at the same time even
// if they have not yet terminated.
// An index consists of an init message from the coordinator, and an all to all echo broadcast of the hash of the init.
// The init message contains the hash and index of the most recent valid index that the coordinator knows of.
// An index is considered valid if it has n-t echo messages supporting the init.
// The consensus executes in round each with the following steps.
// (1) an init message broadcast from the leader
// (2) an all to all echo message containing the hash of the init if the init is a valid message
// An init message is valid if it points to the largest previous index that this is valid
// If a timeout runs out during the index, the node passes on to the next index.
// Once an index is in the path of three larger valid indecies then it is considered committed.
//
// liveness (assuming synchrony) - the coordinator will gather the n-t signatures
// for the largest round known by all non-faulty, all will receive this, and commit the
// index pointed to by three previous

// how to implement synchrony?
// - skip timeouts up to the index where you have n-t supports
// - if we receive an init from a future index we don't support it until the timeout has passed
// TODO is this enough?

// decision (termination)
// we decide when supported by 3 indecies

// safety
// at most one init per index gets n-t echos (by 2/3 marjority)
// once a node know an index is valid it will only support that or larger indecies
// - TODO show diagram cannot support two different indecies longer than 2 supports

// instance says decided when has three later indecies pointing to it
// when a later instance decides, set previous ones to undecided

// an init is valid if it points to the largest index that this node has n-t inits
// if it is valid then the node will broadcast an echo supporting it

// note that an init may not have proofs when received, so we must reject it in the message state?
// or when an index becomes valid can check future indecies

type MvCons3 struct {
	cons.AbsConsItem
	cons.AbsMVRecover
	supportDepth int
	supportItems []types.ConsensusInt
	myProposal   *messagetypes.MvProposeMessage // My proposal for this round

	sentEcho           bool               // true if an echo message was sent
	sentEchoHash       types.HashBytes    // the hash sent in the echo
	sentEchoSupportIdx types.ConsensusInt // the index supported by the echo

	hasDecided          decisionType                                         // if this index has decided or not
	echoHash            types.HashStr                                        // the hash of the decided init msg
	echoHashBytes       types.HashBytes                                      // the hash of the decided init msg
	echoHashSupport     types.ConsensusInt                                   // the index that the sent echo supports
	echoInitMsg         *messagetypes.MvInitSupportMessage                   // the actual decided value (should be same as proposal if nonfaulty coord)
	echoInitProposer    sig.Pub                                              // the public key of the echoInitMsg signer
	myInitMsg           *messagetypes.MvInitSupportMessage                   // my proposal
	myProof             messages.MsgHeader                                   // proof for my proposal
	initTimeOutState    cons.TimeoutState                                    // the timeouts concerning the init message
	initTimer           channelinterface.TimerInterface                      // timer for the init message for the current round
	prevTimeoutState    cons.TimeoutState                                    // state of the timer of the previous consensus index
	prevItem            *MvCons3                                             // previous consensus index
	nextItem            *MvCons3                                             // next consensus index
	initialHash         types.HashBytes                                      // the initial hash
	supportIndexHasInit bool                                                 // set to true once the index we support has received its init message
	validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem // hashes of init messages that have been validated by the state machine

	zeroHash      types.HashBytes // if we don't receive an init message after the timeout we just send a zero hash to let others know we didn't receive anything
	includeProofs bool            // if true then we should include proofs with our echo/commit messages
}

// SetInitialState sets the value that is supported by the inital index (1).
func (sc *MvCons3) SetInitialState(value []byte) {
	sc.initialHash = types.GetHash(value)
}

// GetCommitProof returns nil for MvCons3. It generates the proofs itself since it
// piggybacks rounds with consensus instances.
func (sc *MvCons3) GetCommitProof() []messages.MsgHeader {
	// unused
	return nil
}

// GetConsType returns the type of consensus this instance implements.
func (sc *MvCons3) GetConsType() types.ConsType {
	return types.MvCons3Type
}

// GetPrevCommitProof returns a signed message header that counts at the commit message for the previous consensus.
// This should only be called after DoneKeep has been called on this instance.
func (sc *MvCons3) GetPrevCommitProof() []messages.MsgHeader {
	if len(sc.supportItems) == 0 {
		return nil
	}

	// First go backwards to find the most recent decided
	nxt := sc
	for nxt.hasDecided != decidedValue {
		nxt = nxt.prevItem
	}

	// Now have to go forward 3 support items to get the index that made us commit
	for i := 2; i >= 0; i-- {
		supportIndx := nxt.Index.Index
		for nxt.supportDepth != i || nxt.echoInitMsg == nil || nxt.echoInitMsg.SupportedIndex != supportIndx {
			nxt = nxt.nextItem
		}
	}

	if nxt.myProof == nil {
		supportIndex, _, proof, _ := nxt.getMostRecentSupportRec(true)
		if supportIndex != 0 && proof == nil {
			panic("should have a proof")
		}
		nxt.myProof = proof
	}
	return []messages.MsgHeader{nxt.myProof}
}

// Start allows GetProposalIndex to return true.
func (sc *MvCons3) Start() {
	sc.AbsConsItem.AbsStart()

	if err := sc.startCons(); err != nil {
		panic(err)
	}
}

// HasReceivedProposal returns true if the cons has received a valid proposal.
func (sc *MvCons3) HasReceivedProposal() bool {
	return len(sc.validatedInitHashes) > 0
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *MvCons3) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		if !sc.supportIndexHasInit {
			_, _, _, sc.supportIndexHasInit = sc.getMostRecentSupport(false)
		}
		if sc.supportIndexHasInit {
			return types.SingleComputeConsensusIDShort(sc.myInitMsg.SupportedIndex), true
		}
	}
	return types.ConsensusIndex{}, false
}

// getMostRecentSupport returns the index, hash, and proof of the most recent index that a valid init message and proofs.
// This is called when sending the init to get the index being supported,
// and when sending the echo to check the init is the most recent support.
// HasInit returns true if the index has an init message
func (sc *MvCons3) getMostRecentSupport(generateProofs bool) (index types.ConsensusInt, hash types.HashBytes,
	prfMsg messages.MsgHeader, hasInit bool) {

	_ = sc.Index.Index.(types.ConsensusInt) // sanity check
	if sc.Index.Index.IsInitIndex() {
		if sc.initialHash == nil {
			panic("should have set initial hash")
		}
		return types.ConsensusInt(0), sc.initialHash, nil, true
	}
	return sc.prevItem.getMostRecentSupportRec(generateProofs)
}

func (sc *MvCons3) getMostRecentSupportRec(generateProofs bool) (index types.ConsensusInt,
	hash types.HashBytes, prfMsg messages.MsgHeader, hasInit bool) {
	if sc.hasDecided != decidedNil && sc.echoHashBytes != nil { // TODO fix gc so only collect when can support previous
		if generateProofs {
			nmt := sc.ConsItems.MC.MC.GetMemberCount() - sc.ConsItems.MC.MC.GetFaultCount()
			var err error
			prfMsg, err = sc.ConsItems.MsgState.(*MessageState).checkMvProofs(nmt, sc.ConsItems.MC, true)
			if err != nil {
				panic(fmt.Sprint(err, sc.Index, sc.echoHashBytes))
			}
		}
		return sc.Index.Index.(types.ConsensusInt), sc.echoHashBytes, prfMsg, sc.echoInitMsg != nil
	}
	if sc.Index.Index == types.ConsensusInt(1) {
		if sc.initialHash == nil {
			panic("should have set initial hash")
		}
		return 0, sc.initialHash, nil, true
	}
	index, hash, prfMsg, hasInit = sc.prevItem.getMostRecentSupportRec(generateProofs)
	// return
	// if we haven't sent an echo then we haven't supported anything more recent
	// or if we decided nil
	// or if our timeout passed
	if !sc.sentEcho || sc.hasDecided == decidedNil || sc.initTimeOutState == cons.TimeoutPassed {
		return // so just return the recursive call
	}
	if sc.sentEchoSupportIdx == index { // if we sent an echo supporting the most recent committed we sent that
		// the proof msg is nil because we have only sent the echo and not received n-t of them yet
		return sc.Index.Index.(types.ConsensusInt), sc.sentEchoHash, nil, sc.echoInitMsg != nil
	}
	return // return the recursive call
}

func (sc *MvCons3) passTimer() {
	if sc.initTimeOutState != cons.TimeoutPassed {
		sc.initTimeOutState = cons.TimeoutPassed
		// go back recursively until timeout finding one where the timeout is passed
		if sc.prevItem != nil {
			sc.prevItem.passTimer()
		}
		// if the previous timeout has passed, then we let the next know that all timeouts have passed up to it
		if sc.nextItem != nil && sc.prevTimeoutState == cons.TimeoutPassed {
			sc.nextItem.prevTimeoutPassed()
		}
	}
}

func (sc *MvCons3) prevTimeoutPassed() {
	if sc.prevTimeoutState != cons.TimeoutPassed {
		sc.prevTimeoutState = cons.TimeoutPassed
		sc.checkProgress()
	}
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *MvCons3) SetNextConsItem(next consinterface.ConsItem) {
	sc.nextItem = next.(*MvCons3)
	sc.NextItem = next
	if sc.initTimeOutState == cons.TimeoutPassed {
		sc.nextItem.prevTimeoutPassed()

		// TODO
		//-- prevtimeout passed should mean we check progress (DONE) -- but now should remove excessive recursive calls
		//-- how do we be sure we dont broadcast an echo for an older index after we have sent and echo for a further ahead index?
	}
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *MvCons3) PrevHasBeenReset() {
	sc.prevItem = nil
	sc.prevTimeoutPassed() // TODO what to do here?
}

// ResetState resets the stored state for the current consensus index, and prepares for the new consensus index given by the input.
func (*MvCons3) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &MvCons3{AbsConsItem: newAbsItem}
	if prevItem != nil {
		newItem.prevItem = prevItem.(*MvCons3)
	}
	items.ConsItem = newItem
	newItem.includeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.zeroHash = types.GetZeroBytesHashLength()

	newItem.validatedInitHashes = make(map[types.HashStr]*channelinterface.DeserializedItem)
	newItem.InitAbsMVRecover(index)

	return newItem
}

func (sc *MvCons3) PrintState() string {
	return fmt.Sprintf("index %v, prevTimeout: %v, initTimeout %v, supportDepth %v, myProposal: %v, hasDecided %v, sentEcho: %v, echoHash %v, echoInitMsg: %v\n",
		sc.Index, sc.prevTimeoutState, sc.initTimeOutState, sc.supportDepth, sc.myProposal != nil, sc.hasDecided,
		sc.sentEcho, sc.echoHashBytes != nil, sc.echoInitMsg != nil)
}

// stopTimers is used to stop running timers.
func (sc *MvCons3) stopTimers() {
	if sc.initTimer != nil {
		sc.initTimer.Stop()
		sc.initTimer = nil
	}
	sc.StopRecoverTimeout()
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *MvCons3) GetBinState(localOnly bool) ([]byte, error) {
	msg, err := messages.CreateMsg(sc.PreHeaders)
	if err != nil {
		panic(err)
	}
	bs, err := sc.ConsItems.MsgState.GetMsgState(sc.ConsItems.MC.MC.GetMyPriv(), localOnly,
		sc.GetBufferCount, sc.ConsItems.MC)
	if err != nil {
		return nil, err
	}
	_, err = messages.AppendHeader(msg, (messagetypes.ConsBinStateMessage)(bs))
	if err != nil {
		return nil, err
	}
	return msg.GetBytes(), nil
}

// GetProposeHeaderID returns the HeaderID messages.HdrMvPropose that will be input to GotProposal.
func (sc *MvCons3) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrMvInitSupport returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
// (2) HdrMvEcho returns n-t, n for the thresholds.
func (sc *MvCons3) GetBufferCount(hdr messages.MsgIDHeader, _ *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {
	switch hdr.GetID() {
	case messages.HdrMvInitSupport:
		if config.MvBroadcastInitForBufferForwarder { // This is an all to all broadcast
			return 1, 1, nil, types.ErrDontForwardMessage
		}
		return 1, 1, hdr.GetMsgID(), nil // otherwise we propagate it through gossip
	case messages.HdrMvEcho:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	case messages.HdrPartialMsg:
		if sc.PartialMessageType == types.NoPartialMessages {
			panic("should not have partials")
		}
		return 1, 1, hdr.GetMsgID(), nil
	default:
		return 0, 0, nil, types.ErrInvalidHeader
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// The valid headers are HdrMvInitSupport, HdrMvEcho, HdrMvCommit, HdrMvRequestRecover.
func (*MvCons3) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrMvInitSupport:
		var internalMsg messages.InternalSignedMsgHeader
		internalMsg = messagetypes.NewMvInitSupportMessage()
		if gc.PartialMessageType != types.NoPartialMessages {
			// if partials are used then MvInitSupport must always construct into combined messages
			// TODO add test where a combined message is sent with a different internal header type
			internalMsg = messagetypes.NewCombinedMessage(internalMsg)
		}
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, internalMsg), nil
	case messages.HdrMvEcho:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewMvEchoMessage()), nil
	case messages.HdrMvRequestRecover:
		return messagetypes.NewMvRequestRecoverMessage(), nil
	case messages.HdrPartialMsg:
		if gc.PartialMessageType != types.NoPartialMessages {
			return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewPartialMessage()), nil
		}
	}
	return nil, types.ErrInvalidHeader
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *MvCons3) ShouldCreatePartial(headerType messages.HeaderID) bool {
	if sc.PartialMessageType != types.NoPartialMessages && headerType == messages.HdrMvInitSupport {
		return true
	}
	return false
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *MvCons3) HasDecided() bool {
	switch sc.hasDecided {
	case undecided:
		return false
	case decidedNil:
		return true
	case decidedValue:
		return sc.echoInitMsg != nil
	}
	panic("invalid decisionType")
}

// GetDecision returns the decided value as a byte slice.
func (sc *MvCons3) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	switch sc.hasDecided {
	case decidedNil:
		// TODO: return a default value?
		return nil, nil, types.ConsensusIndex{}
	case decidedValue:
		return sc.echoInitProposer, sc.echoInitMsg.Proposal,
			types.SingleComputeConsensusIDShort(sc.echoInitMsg.SupportedIndex)
	}
	panic("should have decided")
}

// GotProposal takes the proposal, and broadcasts it if it is the leader.
func (sc *MvCons3) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {
	if hdr.(*messagetypes.MvProposeMessage).Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	sc.myInitMsg.Proposal = hdr.(*messagetypes.MvProposeMessage).Proposal
	sc.myInitMsg.ByzProposal = hdr.(*messagetypes.MvProposeMessage).ByzProposal
	sc.broadcastInit(sc.myInitMsg, sc.myProof, mainChannel)
	logging.Info("support index ", sc.myInitMsg.SupportedIndex, " my index", sc.Index)

	return nil
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// The following are the valid message types:
// messages.HdrMvInitSupport is the leader proposal, once this is received an echo is sent containing the hash, and starts the echo timeoutout.
// messages.HdrMvInitTimeout the init timeout is started in GotProposal, if we don't receive a hash before the timeout, we support 0 in bin cons.
// messages.HdrMvEcho is the echo message, when these are received we run CheckEchoState.
// messages.HdrMvRequestRecover a node terminated bin cons with 1, but didn't get the init message, so if we have it we send it.
// messages.HdrMvRecoverTimeout if a node terminated bin cons with 1, but didn't get the init message this timeout is started, once it runs out, we ask other nodes to send the init message.
func (sc *MvCons3) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (bool, bool) {

	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	switch deser.HeaderType {
	case messages.HdrMvRecoverTimeout:
		if !isLocal {
			panic("should be local")
		}
		if sc.echoInitMsg == nil {
			// we have decided, but not received the init message after a timeout so we request it from neighbour nodes.
			logging.Infof("Requesting mv init recover for index %v", sc.Index)
			sc.BroadcastRequestRecover(sc.PreHeaders, sc.echoHashBytes, sc.ConsItems.FwdChecker, sc.MainChannel,
				sc.ConsItems)
		}
		return true, false
	case messages.HdrMvInitTimeout:
		// mvc.initTimeOutState = cons.TimeoutPassed
		if !isLocal {
			panic("should be local")
		}
		sc.passTimer()
		logging.Infof("Got init timeout for index %v", sc.Index)
		// Start echo timeout
		sc.checkProgress()
		return true, false
	case messages.HdrMvInitSupport:
		w := deser.Header.(*sig.MultipleSignedMessage)
		round := types.ConsensusRound(sc.Index.Index.(types.ConsensusInt) - 1) // TODO how to do index instead of round properly?

		// Store it in a map of the proposals
		hashStr := types.HashStr(w.Hash)
		// hashStr := messages.HashStr(messages.GetSignedHash(nxt.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitSupportMessage).Proposal))
		if len(sc.validatedInitHashes) > 0 {
			// we still process this message because it may be the message we commit
			logging.Warningf("Received multiple inits in mv index %v, round %v", sc.Index, round)
		}
		sc.validatedInitHashes[hashStr] = deser

		// initMsg := w.GetBaseMsgHeader().(*messagetypes.MvInitSupportMessage)
		// sanity checks to ensure the init message comes from the coordinator
		// TODO fix way of getting coordpub
		// sanity checks to ensure the init message comes from the coordinator
		err := consinterface.CheckMemberCoord(sc.ConsItems.MC, round, w.SigItems[0], w)
		if err != nil {
			panic("should have handled this in GotMsg")
		}

		logging.Infof("Got an mv init message of len %v, index %v",
			len(w.GetBaseMsgHeader().(*messagetypes.MvInitSupportMessage).Proposal), sc.Index)
		sc.checkProgress()
		// send any recovers that migt have requested this init msg
		sc.SendRecover(sc.validatedInitHashes, sc.InitHeaders, sc.ConsItems)
		return true, true
	case messages.HdrMvEcho:
		// check if we have enough echos to decide
		sc.checkProgress()
		return true, true
	case messages.HdrMvRequestRecover: // a node terminated bin cons, but didn't receive the init message
		sc.GotRequestRecover(sc.validatedInitHashes, deser, sc.InitHeaders, senderChan, sc.ConsItems)
		return false, false
	default:
		panic("unknown msg type")
	}
}

// startRound is called once the previous consensus index has either timed out or supported an echo msg
func (sc *MvCons3) startCons() error {

	if sc.prevItem != nil { // sanity check
		if sc.prevItem.initTimeOutState != cons.TimeoutPassed {
			panic(fmt.Sprintf("sanity check, %v, idx %v, prv idx %v", sc.prevItem.initTimeOutState, sc.Index, sc.prevItem.Index))
		}
	}

	// if memberChecker.CheckMemberBytes(mvc.index, mvc.Pub.GetPubString()) != nil {
	// Check if we are the coord, if so send coord message
	// Otherwise start timeout if we havent already received a proposal
	// TODO fix way of getting coordpub
	if sc.CheckMemberLocal() {
		initMsg := messagetypes.NewMvInitSupportMessage()
		_, _, err := consinterface.CheckCoord(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC,
			types.ConsensusRound(sc.Index.Index.(types.ConsensusInt)-1), initMsg.GetMsgID())
		if err == nil {
			logging.Info("I am coordinator")
			sc.NeedsProposal = true
			supportIndex, supportHash, proof, supportIndexHasInit := sc.getMostRecentSupport(true)
			if supportIndex != 0 && proof == nil {
				panic("should have a proof")
			}
			logging.Info("Sending init for index", sc.Index, supportIndex)
			sc.supportIndexHasInit = supportIndexHasInit
			initMsg.SupportedIndex = supportIndex
			initMsg.SupportedHash = supportHash
			if supportIndex != sc.Index.Index.(types.ConsensusInt)-1 {
				logging.Warningf("Have to skip an init support, supporting %v, at index %v", supportIndex, sc.Index)
			}
			sc.myInitMsg = initMsg
			sc.myProof = proof
		}
	}
	// start the init timeout
	if sc.initTimeOutState == cons.TimeoutNotSent {
		sc.initTimeOutState = cons.TimeoutSent
		deser := []*channelinterface.DeserializedItem{
			{
				Index:          sc.Index,
				HeaderType:     messages.HdrMvInitTimeout,
				IsDeserialized: true,
				IsLocal:        types.LocalMessage}}
		prvIdx, _, _, _ := sc.getMostRecentSupport(false) // TODO better way to decide timeout duration?
		sc.initTimer = sc.MainChannel.SendToSelf(deser,
			cons.GetMvTimeout(types.ConsensusRound(sc.Index.Index.(types.ConsensusInt)-prvIdx), sc.ConsItems.MC.MC.GetFaultCount()))
	}
	return nil
}

// func (sc *MvCons3) processInit(initMsg *messagetypes.MvInitSupportMessage, mainChannel channelinterface.MainChannel) {
// if this is the supported init then
// }

// checkPrevDecided is a sanity check
func (sc *MvCons3) checkPrevDecided() {
	if sc.prevItem != nil {
		if sc.prevItem.hasDecided == undecided {
			panic("failed sanity check")
			// bc.prevItem.decideUndecidedNil()
		}
	}
}

func (sc *MvCons3) addSupport(fromIndex, toIndex types.ConsensusInt, supportedHash types.HashBytes, depth int,
	supportIdxs []types.ConsensusInt) {

	if sc.Index.Index.(types.ConsensusInt) > fromIndex {
		panic("out of order from indecies")
	}
	if sc.Index.Index.(types.ConsensusInt) < toIndex {
		panic("out of order to indecies")
	}
	//if bc.hasDecided != undecided { // we have already decided so we don't need to update indecies
	//	return
	//}

	if sc.Index.Index.(types.ConsensusInt) > toIndex { // we are not supported, someone with a smaller index is
		if depth > 3 {
			if sc.hasDecided == decidedValue { // sanity check
				panic(fmt.Sprint("tried to decide nil after decided a value unsupported ",
					sc.Index, " from ", fromIndex, " to ", toIndex, " depth ", depth))
			}
			// since someone before us has been supported and we are not, we cannot be decided
			// TODO is this true?
			if sc.hasDecided == undecided {
				logging.Info("Decided nil index ", sc.Index)
				logging.Info("decide nil", sc.Index, "from", fromIndex, "depth", depth,
					"without something to support", "depth", sc.supportDepth+1)
				// sc.ConsItems.MC.MC.GetStats().AddFinishRound(types.ConsensusRound(fromIndex - sc.Index.Index.(types.ConsensusInt) - 1))
			}
			sc.hasDecided = decidedNil
		}
		// keep going to find correct index to support
		if sc.prevItem != nil {
			sc.prevItem.addSupport(fromIndex, toIndex, supportedHash, depth, supportIdxs)
		}
	} else { // we are supported
		if depth > sc.supportDepth { // if this is greater than a previous support then we update our info
			if depth >= 3 {
				if sc.hasDecided == decidedNil {
					panic(fmt.Sprint("tried to decide nil after decided a value", sc.Index, "from", fromIndex, "depth", depth))
				}
				if sc.hasDecided == undecided {
					logging.Info("Decided value index ", sc.Index)
					if sc.echoInitMsg == nil && sc.GeneralConfig.TestIndex == 9 {
						logging.Info("we decided", sc.Index, "from", fromIndex, "depth", depth,
							"without something to support", "depth", sc.supportDepth+1)
					} else if sc.GeneralConfig.TestIndex == 9 {
						logging.Info("we decided", sc.Index, "from", fromIndex, "depth", depth,
							"we now support", sc.echoInitMsg.SupportedIndex, "depth", sc.supportDepth+1)
					}
					sc.ConsItems.MC.MC.GetStats().AddFinishRound(types.ConsensusRound(
						supportIdxs[len(supportIdxs)-2]-sc.Index.Index.(types.ConsensusInt)), false)
					//types.ConsensusRound(fromIndex - sc.Index.Index.(types.ConsensusInt) - 1))
				}
				sc.hasDecided = decidedValue // we decided a value
			}
			// bc.supportedBy[fromIndex] = depth
			sc.supportDepth = depth
			sc.supportItems = supportIdxs

			// if we are supported, the the supported message must have the same hash
			// as our echo hash (got from n-m messages), otherwise something went wrong
			if sc.echoHashBytes != nil {
				if !bytes.Equal(sc.echoHashBytes, supportedHash) {
					panic("got multiple support hashes")
				}
			} else {
				// we only set the echo hash bytes when we get the hash from the echo supports
				// this is so we dont try to send support until we have enough proofs for a proof message
				// TODO also check in case we receive in the other order

				// bc.echoHashBytes = supportedHash
				// bc.echoHash = messages.HashStr(supportedHash)
			}

			if sc.echoInitMsg != nil {
				// let the others know they are supported
				if sc.prevItem != nil {
					sc.prevItem.addSupport(sc.Index.Index.(types.ConsensusInt), sc.echoInitMsg.SupportedIndex,
						sc.echoInitMsg.SupportedHash, sc.supportDepth+1, append(supportIdxs, sc.Index.Index.(types.ConsensusInt)))
				}
				if sc.supportDepth >= 3 {
					// if we decided and told the previous about their support, but they still haven't decided, then they
					// must decide nil
					// TODO is this needed because it is done above?
					sc.checkPrevDecided() // tell the previous undecided instances to decideNil
				}
			}
		}
	}
}

func (sc *MvCons3) printDecidedList(prevString string) string {
	prevString += fmt.Sprintf("%v,%v ", sc.Index, sc.hasDecided)
	if sc.prevItem == nil {
		return prevString
	}
	return sc.prevItem.printDecidedList(prevString)
}

// CanStartNext returns true if the timer for this instance has passed,
// or a if a later instance has received at least n-t messages
func (sc *MvCons3) CanStartNext() bool {
	// if mvc.initTimeOutState == cons.TimeoutPassed {
	if sc.echoHashBytes != nil && sc.echoInitMsg == nil {
		return false
	}
	if sc.echoInitMsg != nil || sc.initTimeOutState == cons.TimeoutPassed { // TODO do we need the above condition???
		return true
	}
	return false
}

// GetNextInfo will be called after CanStartNext returns true.
// prevIdx should be the index that this cosensus index will follow (normally this is just idx - 1).
// preDecision is either nil or the value that will be decided if a non-nil value is decided.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *MvCons3) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	if sc.echoInitMsg == nil {
		return types.ConsensusIndex{}, nil, nil, false
	}
	return types.SingleComputeConsensusIDShort(sc.echoInitMsg.SupportedIndex),
		sc.echoInitProposer, sc.echoInitMsg.Proposal, true
}

// NeedsConcurrent returns 4.
func (sc *MvCons3) NeedsConcurrent() types.ConsensusInt {
	return 20
}

func (sc *MvCons3) getInitMessages() []*sig.MultipleSignedMessage {

	ret := make([]*sig.MultipleSignedMessage, 0, len(sc.validatedInitHashes))
	for _, deser := range sc.validatedInitHashes {
		// ret = append(ret, deser.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitSupportMessage))
		ret = append(ret, deser.Header.(*sig.MultipleSignedMessage))
	}
	return ret
}

// recSentEcho is called when we send an echo, and sets it so any lower indecies will not send an echo.
// TODO fix excesive recursive calls
func (sc *MvCons3) recSentEcho() {
	sc.sentEcho = true
	if sc.prevItem != nil {
		sc.prevItem.recSentEcho()
	}
}

// checkProgress checks if we should perform an action
func (sc *MvCons3) checkProgress() {

	// if mvc.HasDecided() {
	//return
	//}
	// t := mvc.MemberChecker.MC.GetFaultCount()
	// nmt := mvc.MemberChecker.MC.GetMemberCount() - t

	msgState := sc.ConsItems.MsgState.(*MessageState)
	if sc.Index.Index == types.ConsensusInt(1) && sc.prevTimeoutState != cons.TimeoutPassed { // sanity check
		panic("prev timeout should always be passed for index 1")
	}
	// we only send the echo once the previous index is done
	if !sc.sentEcho && sc.prevTimeoutState == cons.TimeoutPassed {
		for _, initMsg := range sc.getInitMessages() {
			// TODO don't create proofs in loop
			mostRecentSupportIndex, mostRecentHash, proof, _ := sc.getMostRecentSupport(sc.includeProofs)
			initSupportMsg := initMsg.InternalSignedMsgHeader.(*messagetypes.MvInitSupportMessage)
			if mostRecentSupportIndex == initSupportMsg.SupportedIndex && bytes.Equal(mostRecentHash, initSupportMsg.SupportedHash) {
				if mostRecentSupportIndex != types.ConsensusInt(0) && proof == nil && sc.includeProofs {
					panic("should have a proof")
				}
				logging.Infof("Got a valid init msg to support, sending echo for index %v, supporting index %v", sc.Index, mostRecentSupportIndex)
				sc.sentEcho = true
				sc.sentEchoHash = initMsg.GetSignedHash()
				sc.sentEchoSupportIdx = initSupportMsg.SupportedIndex

				// we don't want to send any echos for instances with lower indecies
				sc.recSentEcho()
				sc.broadcastEcho(initMsg.Hash, proof, sc.MainChannel)
			} else {
				logging.Infof("Got an invalid init message because of support, most recent index %v, msg support index %v, most recent hash %v, msg support hash %v",
					mostRecentSupportIndex, initSupportMsg.SupportedIndex, mostRecentHash, initSupportMsg.SupportedHash)
			}
		}
	}

	if echoHash := msgState.getSupportedEchoHash(); echoHash != nil {
		if sc.echoHashBytes != nil {
			if !bytes.Equal(sc.echoHashBytes, echoHash) {
				panic("got two different supported hashes")
			}
		} else {
			sc.echoHashBytes = echoHash
			sc.echoHash = types.HashStr(echoHash)

			// let the instances that support us know if they are valid or not

		}
	}

	// check if we can decide
	// decision is made through addSupport fucntion TODO do it here instead, why?, seems ok in addSupport?

	if !sc.HasDecided() && sc.echoHashBytes != nil {
		// request recover if needed
		if initMsg := sc.validatedInitHashes[sc.echoHash]; initMsg == nil {
			// we haven't yet received the init message for the hash, so we request it from other nodes after a timeout
			if sc.hasDecided == decidedValue {
				sc.StartRecoverTimeout(sc.Index, sc.MainChannel)
			}
		} else {
			// we have the init message so we let others know who we support
			if sc.echoInitMsg == nil {
				sc.echoInitMsg = initMsg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitSupportMessage)
				sc.echoInitProposer = initMsg.Header.(*sig.MultipleSignedMessage).SigItems[0].Pub
				logging.Infof("Have decision init message index %v", sc.Index)

				// stop the timers
				sc.stopTimers()

				// let the others know they are supported
				// we add the hash for a sanity check
				if sc.prevItem != nil {
					sc.prevItem.addSupport(sc.Index.Index.(types.ConsensusInt), sc.echoInitMsg.SupportedIndex,
						sc.echoInitMsg.SupportedHash, sc.supportDepth+1, sc.supportItems)
				}
			}
		}
	}

	if sc.echoHashBytes != nil {
		// we can pass our timer now that we have enough participated
		sc.passTimer()
	}

	if sc.prevItem != nil {
		sc.prevItem.checkProgress()
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// broadcastInit broadcasts an int message
func (sc *MvCons3) broadcastInit(initMsg *messagetypes.MvInitSupportMessage, proofMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	var forwardFunc channelinterface.NewForwardFuncFilter
	if config.MvBroadcastInitForBufferForwarder { // we change who we broadcast to depending on the configuration
		forwardFunc = channelinterface.ForwardAllPub // we broadcast the init message to all nodes directly
	} else {
		forwardFunc = sc.ConsItems.FwdChecker.GetNewForwardListFunc() // we propoagte the init message using gossip
	}
	sc.BroadcastFunc(nil, sc.ConsItems, initMsg, true, forwardFunc,
		mainChannel, sc.GeneralConfig, proofMsg)
	// BroadcastMv3(nil, sc.ByzType, sc, forwardFunc, initMsg, proofMsg, mainChannel)
}

// broadcastEcho broadcasts an echo message
func (sc *MvCons3) broadcastEcho(proposalHash []byte, proofMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	newMsg := messagetypes.NewMvEchoMessage()
	newMsg.ProposalHash = proposalHash

	// the next coordinator
	var coordPub sig.Pub
	var err error
	if sc.ConsItems.MC.MC.RandMemberType() != types.NonRandom { // TODO support membership changes for MVCons3??
		logging.Error("rand members not supported for MvCons3")
		err = types.ErrNotMember
	} else {
		if sc.GeneralConfig.CollectBroadcast != types.Full { // we broadcast to the parent
			msg := messagetypes.NewMvInitSupportMessage()
			_, coordPub, err = consinterface.CheckCoord(nil, sc.ConsItems.MC,
				types.ConsensusRound(sc.Index.Index.(types.ConsensusInt)), msg.GetMsgID())
		}
	}
	if err != nil {
		logging.Errorf("error getting next pub", err, sc.Index)
	}
	sc.BroadcastFunc(coordPub, sc.ConsItems, newMsg, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		mainChannel, sc.GeneralConfig, proofMsg)

	// BroadcastMv3(coordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(), newMsg, proofMsg, mainChannel)
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*MvCons3) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewMvCons3MessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *MvCons3) Collect() {
	sc.stopTimers()
	sc.StopRecoverTimeout()
}
