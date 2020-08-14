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
package mvcons2

import (
	"bytes"
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
	"github.com/tcrain/cons/consensus/utils"
	"sort"
)

type roundMvState struct {
	initTimeOutState   cons.TimeoutState // the timeouts concerning the init message
	echoTimeOutState   cons.TimeoutState // the timeouts concerning the echo message
	commitTimeOutState cons.TimeoutState // the timeouts concerning the commit message
	sentEcho           bool              // true if an echo message was sent
	sentCommit         bool              // true if an echo message was sent
}

// MvCons2 is a simple rotating-coordinator-based multi-value consensus.
// Assuming the coordinator is correct and the timeout configured correctly, termination happens in 3 message steps.
// Otherwise a "view change" happens where the next coordinator proposes a value (it can be a new value if it has proof
// that no value was decided previously, otherwise it must be a value that was previously proposed).
// The consensus executes in round each with the following steps.
// (1) an init message broadcast from the leader
// (2) an all to all echo message containing the hash of the init if the init is a valid message
// (3) an all to all commit message
// If a timeout runs out between (1) and (2), then a node broadcasts an echo message with a zero hash.
// Following the first round, nodes only support valid init/echo messages as follows:
// (1) n-t commit messages supporting 0 from the previous round, then an echo with any hash is valid
// (2) n-t echo messages supporting a value from the previous round, then an echo with this value is valid
// Furthermore we only pass to the this round when one of these conditions is valid.
//
// decision (termination)
// we decide when n-t non-zero commit
//
// safety
// - if we get n-t commit 0, then no-one decided - because no one got n-t different value commit by majority
// - if we don't get n-t commit 0, then a non-faulty must have gotten n-t same echo, so we support this
// - we can only get n-t echo/commit of a single value (by 2/3 majority)
//
// livness
// - leader changes each round by rotating coordinator
// (a) we go from init to echo on timeout (if we have no echo to send we send nothing)
// (b) we go from echo to commit on timeout
// (c) we proceed to next round if we get n-t echo and commit, where either n-t commit 0, or n-t same echo.
// (d) if we are leader for round then we use the messages that passed condition (c) to send init proofs
// == a, b are always true by timer, c by follows:
// if a non-faulty gets n-t same echo it sends commit with a val,
// otherwise it sends commit with a 0.
// Now all non-faulty will get n-t commit 0, or n-t echo, since non-faulty send all info on timeout (or include proofs).
// Now all will have proofs for the next round.
// Will eventually reach a round with fast enough and correct leader by increasing timeout => termination
// If a non-faulty node terminated in a previous round, then all will eventually terminate as it broadcasts all msgs
// received so far which is enough for all nodes to terminate.
type MvCons2 struct {
	cons.AbsConsItem
	cons.AbsMVRecover
	myProposal          *messagetypes.MvProposeMessage                       // My proposal for this round
	round               types.ConsensusRound                                 // the current consensus round
	roundState          map[types.ConsensusRound]roundMvState                // state for each round
	skipTimeoutRound    types.ConsensusRound                                 // will not wait on timeouts until this round
	decisionHash        types.HashStr                                        // the hash of the decided value
	decisionHashBytes   types.HashBytes                                      // the hash of the decided value
	decisionRound       types.ConsensusRound                                 // the round of decision
	decisionInitMsg     *messagetypes.MvInitMessage                          // the actual decided value (should be same as proposal if nonfaulty coord)
	decisionPub         sig.Pub                                              // the pub of the decisionInitMsg
	zeroHash            types.HashBytes                                      // if we don't receive an init message after the timeout we just send a zero hash to let others know we didn't receive anything
	includeProofs       bool                                                 // if true then we should include proofs with our echo/commit messages
	validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem // hashes of init messages that have been validated by the state machine
	initMessageByRound  map[types.ConsensusRound]cons.DeserSortVRF           // map from round to init hash

	initTimer   channelinterface.TimerInterface // timer for the init message for the current round
	echoTimer   channelinterface.TimerInterface // timer for the echo messages for the current round
	commitTimer channelinterface.TimerInterface // timer for the commit messages for the current round
	// priv            sig.Priv             // the local nodes private key
	myProposalRound       types.ConsensusRound // round that this nodes sends a proposal
	lastProposalBroadcast bool                 // true when broadcast the latest proposal
}

// GetConsType returns the type of consensus this instance implements.
func (sc *MvCons2) GetConsType() types.ConsType {
	return types.MvCons2Type
}

// GenerateNewItem creates a new cons item.
func (*MvCons2) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc, gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &MvCons2{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.includeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.zeroHash = types.GetZeroBytesHashLength()
	newItem.roundState = make(map[types.ConsensusRound]roundMvState)
	newItem.roundState = make(map[types.ConsensusRound]roundMvState)
	newItem.validatedInitHashes = make(map[types.HashStr]*channelinterface.DeserializedItem)
	newItem.initMessageByRound = make(map[types.ConsensusRound]cons.DeserSortVRF)
	newItem.InitAbsMVRecover(index)

	return newItem
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *MvCons2) GetCommitProof() []messages.MsgHeader {
	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t

	commitMsg := messagetypes.NewMvCommitMessage()
	commitMsg.Round = sc.decisionRound
	commitMsg.ProposalHash = sc.decisionHashBytes

	// Add sigs
	prfMsgSig, err := sc.ConsItems.MsgState.SetupSignedMessage(commitMsg, false, nmt, sc.ConsItems.MC)
	if err != nil {
		panic(err)
	}
	return []messages.MsgHeader{prfMsgSig}
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *MvCons2) SetNextConsItem(consinterface.ConsItem) {
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *MvCons2) PrevHasBeenReset() {
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *MvCons2) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *MvCons2) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrMvInit returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
// (2) HdrMvEcho returns n-t, n for the thresholds.
// (3) HdrMvCommit returns n-t, n for the thresholds.
func (sc *MvCons2) GetBufferCount(hdr messages.MsgIDHeader, _ *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrMvInit:
		if config.MvBroadcastInitForBufferForwarder { // This is an all to all broadcast
			return 1, 1, nil, types.ErrDontForwardMessage
		}
		return 1, 1, hdr.GetMsgID(), nil // otherwise we propagate it through gossip
	case messages.HdrMvEcho, messages.HdrMvCommit:
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
// The valid headers are HdrMvInit, HdrMvEcho, HdrMvCommit, HdrMvRequestRecover.
func (*MvCons2) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrMvInit:
		var internalMsg messages.InternalSignedMsgHeader
		internalMsg = messagetypes.NewMvInitMessage()
		if gc.PartialMessageType != types.NoPartialMessages {
			// if partials are used then MvInit must always construct into combined messages
			// TODO add test where a combined message is sent with a different internal header type
			internalMsg = messagetypes.NewCombinedMessage(internalMsg)
		}
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, internalMsg), nil
	case messages.HdrMvEcho:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewMvEchoMessage()), nil
	case messages.HdrMvCommit:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewMvCommitMessage()), nil
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
func (sc *MvCons2) ShouldCreatePartial(headerType messages.HeaderID) bool {
	if sc.PartialMessageType != types.NoPartialMessages && headerType == messages.HdrMvInit {
		return true
	}
	return false
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *MvCons2) HasDecided() bool {
	if sc.decisionInitMsg != nil {
		return true
	}
	return false
}

// GetDecision returns the decided value as a byte slice.
func (sc *MvCons2) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	if sc.decisionInitMsg != nil {
		if len(sc.decisionInitMsg.Proposal) == 0 {
			panic(sc.Index)
		}
		return sc.decisionPub, sc.decisionInitMsg.Proposal,
			types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
	}
	panic("should have decided")
}

// passTimers is used to say the timers are complete for the current round.
func (sc *MvCons2) passTimers(round types.ConsensusRound) {
	roundState := sc.roundState[round]
	roundState.initTimeOutState = cons.TimeoutPassed
	roundState.echoTimeOutState = cons.TimeoutPassed
	roundState.commitTimeOutState = cons.TimeoutPassed
	sc.roundState[round] = roundState
}

// stopTimers is used to stop running timers.
func (sc *MvCons2) stopTimers() {
	if sc.initTimer != nil {
		sc.initTimer.Stop()
		sc.initTimer = nil
	}
	if sc.echoTimer != nil {
		sc.echoTimer.Stop()
		sc.echoTimer = nil
	}
	if sc.commitTimer != nil {
		sc.commitTimer.Stop()
		sc.commitTimer = nil
	}
	sc.StopRecoverTimeout()
}

// GotProposal takes the proposal, and broadcasts it if it is the leader.
func (sc *MvCons2) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {

	sc.AbsGotProposal()
	sc.myProposal = hdr.(*messagetypes.MvProposeMessage)
	if sc.myProposal.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	return sc.broadcastProposal(mainChannel)
}

func (sc *MvCons2) broadcastProposal(mainChannel channelinterface.MainChannel) error {
	round := sc.myProposalRound
	proposal, proof := sc.getProposal(round)
	sc.lastProposalBroadcast = true
	var newMsg messages.InternalSignedMsgHeader
	if proposal != nil { // we broadcast our own proposal
		initMsg := messagetypes.NewMvInitMessage()
		initMsg.Proposal = proposal
		initMsg.Round = round
		initMsg.ByzProposal = sc.myProposal.ByzProposal
		newMsg = initMsg
		logging.Infof("Sending non nil proposal round %v, index %v", round, sc.Index)
	} else { // we must support a proposal from the previous round
		logging.Infof("Supporting previous proposal round %v, index %v", round, sc.Index)
	}
	sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

	sc.broadcastInit(newMsg, proof, mainChannel)

	return nil
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *MvCons2) CanStartNext() bool {
	return true
}

// Start allows GetProposalIndex to return true.
func (sc *MvCons2) Start() {
	sc.AbsConsItem.AbsStart()
	if sc.round == 0 {
		err := sc.startRound(sc.MainChannel)
		if err != nil {
			panic(err)
		}
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *MvCons2) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created. // TODO
func (sc *MvCons2) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1),
		nil, nil, true
}

// HasReceivedProposal returns true if the cons has received a valid proposal.
func (sc *MvCons2) HasValidStarted() bool {
	return len(sc.validatedInitHashes) > 0
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// The following are the valid message types:
// messages.HdrMvInit is the leader proposal, once this is received an echo is sent containing the hash, and starts the echo timeoutout.
// messages.HdrMvInitTimeout the init timeout is started in GotProposal, if we don't receive a hash before the timeout, we support 0 in bin cons.
// messages.HdrMvEcho is the echo message, when these are received we run CheckEchoState.
// messages.HdrMvEchoTimeout once the echo timeout runs out, without receiving n-t equal echo messages, start the commit timeout, otherwise sending a commit message.
// messages.HdrMvCommit is the commit message, when these are received we run CheckCommitState.
// messages.HdrMvCommitTimeout once the commit timeout runs out, without receiving n-t equal commit messages, we move to the next round if possible, otherwise we decide.
// messages.HdrMvRequestRecover a node terminated bin cons with 1, but didn't get the init message, so if we have it we send it.
// messages.HdrMvRecoverTimeout if a node terminated bin cons with 1, but didn't get the init mesage this timeout is started, once it runs out, we ask other nodes to send the init message.
func (sc *MvCons2) ProcessMessage(
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
	case messages.HdrMvInitTimeout, messages.HdrMvEchoTimeout, messages.HdrMvCommitTimeout:
		if !isLocal {
			panic("should be local")
		}
		// Drop timeouts if already decided
		if sc.decisionHashBytes != nil {
			return true, false
		}
	}
	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t
	switch deser.HeaderType {
	case messages.HdrMvRecoverTimeout:
		if !isLocal {
			panic("should be local")
		}
		if sc.decisionInitMsg == nil {
			// we have decided, but not received the init message after a timeout so we request it from neighbour nodes.
			logging.Infof("Requesting mv init recover for index %v", sc.Index)
			sc.BroadcastRequestRecover(sc.PreHeaders, sc.decisionHashBytes, sc.ConsItems.FwdChecker, sc.MainChannel,
				sc.ConsItems)
		}
		return true, false
	case messages.HdrMvInitTimeout:
		round := (types.ConsensusRound)(deser.Header.(messagetypes.MvInitMessageTimeout))
		roundState := sc.roundState[round]
		roundState.initTimeOutState = cons.TimeoutPassed
		logging.Infof("Got init timeout for round %v, index %v", round, sc.Index)
		// Start echo timeout
		roundState = sc.startEchoTimeout(round, t, roundState, sc.MainChannel)
		sc.roundState[round] = roundState
		sc.checkProgress(round, t, nmt, sc.MainChannel)
		return true, false
	case messages.HdrMvEchoTimeout:
		// if we haven't already send a commit msg then we didnt receive enough equal echo messages after the init message.
		round := (types.ConsensusRound)(deser.Header.(messagetypes.MvEchoMessageTimeout))
		roundState := sc.roundState[round]
		roundState.echoTimeOutState = cons.TimeoutPassed
		logging.Infof("Got echo timeout for round %v, index %v", round, sc.Index)
		// start commit timeout
		roundState = sc.startCommitTimeout(round, t, roundState, sc.MainChannel)
		sc.roundState[round] = roundState

		sc.checkProgress(round, t, nmt, sc.MainChannel)
		return true, false
	case messages.HdrMvCommitTimeout:
		// if we didn't already decide then we didnt receive enough echo messages after the init message.
		round := (types.ConsensusRound)(deser.Header.(messagetypes.MvCommitMessageTimeout))
		roundState := sc.roundState[round]
		roundState.commitTimeOutState = cons.TimeoutPassed
		sc.roundState[round] = roundState
		logging.Infof("Got commit timeout for round %v, index %v", round, sc.Index)

		sc.checkProgress(round, t, nmt, sc.MainChannel)
		return true, false
	case messages.HdrMvInit:
		// Store it in a map of the proposals
		round := cons.GetMvMsgRound(deser)
		hashStr := types.HashStr(types.GetHash(deser.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal))
		initMsgsByRound := sc.initMessageByRound[round]
		if len(initMsgsByRound) != 0 || sc.validatedInitHashes[hashStr] != nil {
			// we still process this message because it may be the message we commit
			logging.Info("Received multiple inits in mv index %v, round %v", sc.Index, round)
		}
		sc.initMessageByRound[round] = append(initMsgsByRound, deser)
		sc.validatedInitHashes[hashStr] = deser
		w := deser.Header.(*sig.MultipleSignedMessage)

		// sanity checks to ensure the init message comes from the coordinator
		err := consinterface.CheckMemberCoord(sc.ConsItems.MC, round, w.SigItems[0], w) // sanity check
		if err != nil {
			panic("should have handled this in GotMsg")
		}
		logging.Infof("Got an mv init message of len %v, round %v, index %v",
			len(w.GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal), round, sc.Index)

		sc.checkProgress(round, t, nmt, sc.MainChannel)
		// send any recovers that migt have requested this init msg
		sc.SendRecover(sc.validatedInitHashes, sc.InitHeaders, sc.ConsItems)
		return true, true
	case messages.HdrMvEcho, messages.HdrMvCommit:
		// check if we have enough echos to decide
		round := cons.GetMvMsgRound(deser)
		sc.checkProgress(round, t, nmt, sc.MainChannel)
		return true, true
	case messages.HdrMvRequestRecover: // a node terminated bin cons, but didn't receive the init message
		sc.GotRequestRecover(sc.validatedInitHashes, deser, sc.InitHeaders, senderChan, sc.ConsItems)
		return false, false
	default:
		panic("unknown msg type")
	}
}

// startRound is called once the previous round is completed, and the next round needs to start
func (sc *MvCons2) startRound(mainChannel channelinterface.MainChannel) error {

	logging.Infof("Starting round %v, index %v", sc.round, sc.Index)

	// stop the timers for the current round
	sc.stopTimers()

	// if memberChecker.CheckMemberBytes(sc.index, sc.Pub.GetPubString()) != nil {
	// Check if we are the coord, if so send coord message
	// Otherwise start timeout if we havent already received a proposal
	if sc.CheckMemberLocal() {
		initMsg := messagetypes.NewMvInitMessage()
		initMsg.Round = sc.round
		_, _, err := consinterface.CheckCoord(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC, sc.round, initMsg.GetMsgID())
		if err == nil {
			sc.NeedsProposal = true
			sc.myProposalRound = sc.round
			sc.lastProposalBroadcast = false
			logging.Info("I am coordinator for round", sc.round)
		}
	}
	// start the init timeout
	roundState := sc.roundState[sc.round]
	if roundState.initTimeOutState == cons.TimeoutNotSent {
		// Start the init timer
		sc.initTimer = cons.StartInitTimer(sc.round, sc.ConsItems, sc.MainChannel)
		roundState.initTimeOutState = cons.TimeoutSent
	} else if roundState.initTimeOutState == cons.TimeoutPassed {
		// start the echo timeout immediately
		t := sc.ConsItems.MC.MC.GetFaultCount()
		roundState = sc.startEchoTimeout(sc.round, t, roundState, mainChannel)
	}
	sc.roundState[sc.round] = roundState

	if !sc.lastProposalBroadcast && sc.AbsConsItem.GotProposal &&
		sc.myProposalRound == sc.round { // Broadcast the proposal if we have received it

		if err := sc.broadcastProposal(mainChannel); err != nil {
			return err
		}
	}
	return nil
}

// getProposal returns the valid proposal for this round
func (sc *MvCons2) getProposal(round types.ConsensusRound) (proposal []byte, proofMsg messages.MsgHeader) {

	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t

	var err error
	if round > 0 {
		// proofs will be either echo or commit from the previous round
		proofMsg, err = sc.ConsItems.MsgState.(*MessageState).GenerateProofs(nmt, round-1, 0,
			sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
		if err != nil {
			panic(err) // we should always have a valid proof msg here
		}
	}
	if round == 0 { // first round the proof is the local proposal
		proposal = sc.myProposal.Proposal
		// the proof comes from the previous consensus round
		switch len(sc.CommitProof) {
		case 0: // no proofs
		case 1:
			proofMsg = sc.CommitProof[0]
		default:
			panic("should not have multiple proof messages")
		}
	} else {
		if proofMsg.GetID() == messages.HdrMvEcho {
			// if it is an mvEcho message, then we support that directly with the proofs,
			// we don't need to create a new init message
			// so we keep proposal nil
			logging.Info("echo proof")
		} else if proofMsg.GetID() == messages.HdrMvCommit {
			if sc.myProposal == nil {
				logging.Infof("Advanced to round %v without local proposal %v", sc.round, sc.Index)
			} else {
				proposal = sc.myProposal.Proposal
			}
		} else {
			panic("invalid proof msg")
		}
	}
	return
}

// isInitValid returns the valid init message received for the round, otherwise nil if there is none
// If the zero hash is valid for the round it returns that as well.
func (sc *MvCons2) isInitValid(round types.ConsensusRound, nmt int) (
	zeroHashValid bool, validInit *channelinterface.DeserializedItem) {

	msgState := sc.ConsItems.MsgState.(*MessageState)
	initMsgs := sc.initMessageByRound[round]
	sort.Sort(initMsgs)
	for _, initMsg := range initMsgs {
		hashStr := types.HashStr(types.GetHash(initMsg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal))
		if sc.validatedInitHashes[hashStr] == nil {
			// the init message was not validated
			// return nil
			panic(hashStr)
		}
		if round == 0 { // init for round 0 is always valid
			return true, initMsg
		}
		// an init is valid if we have n-t 0 commits from the previous round
		if _, err := msgState.checkMvProofs(round-1, nmt, sc.ConsItems.MC.MC.GetMyPriv().GetPub(),
			sc.ConsItems.MC, false, true); err == nil {

			return true, initMsg
		}
	}
	if round == 0 {
		return true, nil
	}
	if _, err := msgState.checkMvProofs(round-1, nmt, sc.ConsItems.MC.MC.GetMyPriv().GetPub(),
		sc.ConsItems.MC, false, true); err == nil {

		return true, nil
	}
	return false, nil
}

// NeedsConcurrent returns 1.
func (sc *MvCons2) NeedsConcurrent() types.ConsensusInt {
	return 1
}

func (sc *MvCons2) checkProgress(round types.ConsensusRound, t, nmt int, mainChannel channelinterface.MainChannel) {
	// Check from the current round of the consensus to the round of the message for progress
	min := utils.MinConsensusRound(round, sc.round)
	max := utils.MaxConsensusRound(round, sc.round)
	for r := min; r <= max; r++ {
		sc.checkProgressRound(r, t, nmt, mainChannel)
	}
	// Check if we can advance in rounds past that of the message
	for sc.round > round {
		round = sc.round
		sc.checkProgressRound(sc.round, t, nmt, mainChannel)
	}
}

// checkProgress checks if we should perform an action
func (sc *MvCons2) checkProgressRound(round types.ConsensusRound, t, nmt int, mainChannel channelinterface.MainChannel) {

	if sc.HasDecided() {
		return
	}
	msgState := sc.ConsItems.MsgState.(*MessageState)

	// check if we can skip timers because of the reception of more advanced messages
	roundState := sc.roundState[round]
	echoCount, commitCount := msgState.GetValidMessageCount(round)
	if echoCount >= nmt { // if the echo count of the round is large enough we skip the init timer
		roundState.initTimeOutState = cons.TimeoutPassed
		roundState = sc.startEchoTimeout(round, t, roundState, mainChannel)
	}
	if commitCount >= nmt { // if the commit count of the round is large enough we skip the echo timer
		roundState.echoTimeOutState = cons.TimeoutPassed
		roundState = sc.startCommitTimeout(round, t, roundState, mainChannel)
	}
	sc.roundState[round] = roundState          // we store round state because passTimers will modify it
	if commitCount > nmt || echoCount >= nmt { // for all previous rounds we skip all timeouts
		for r := sc.round; r < round; r++ {
			sc.passTimers(r)
		}
	}

	// check if we should send an echo
	if round == sc.round { //  only send messages for the current round
		roundState = sc.roundState[round]
		if !roundState.sentEcho {
			// if we have a valid init message then we don't care about the timeout (if we are not selecting random members)
			var hash types.HashBytes
			zeroHashValid, initMsg := sc.isInitValid(round, nmt)
			selectRandMembers := consinterface.ShouldWaitForRndCoord(sc.ConsItems.MC.MC.RandMemberType())
			if initMsg != nil && // have a valid init message and either (a) or (b)
				((selectRandMembers && roundState.initTimeOutState == cons.TimeoutPassed) || // (a) the init timeout has passed and we are selecting random members
					!selectRandMembers) { // (b) we are not selecting random members
				logging.Infof("Sending non-zero echo message for round %v, index%v", round, sc.Index)
				hash = types.GetHash(initMsg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal)
			} else if hash = msgState.getSupportedEchoHash(round); hash != nil {
				// we dont have an init, but we got enough echos to support that
			} else if roundState.initTimeOutState == cons.TimeoutPassed && zeroHashValid {
				// if the timeout has passed and no valid init then we send an echo with a zero message
				// and we are round 0, or have enough zero messages from the previous round
				logging.Infof("Sending ZERO echo message for round %v, index%v", round, sc.Index)
				hash = types.GetZeroBytesHashLength()
			}
			if hash != nil { // we got a hash to send
				roundState.sentEcho = true
				roundState = sc.startEchoTimeout(round, t, roundState, mainChannel)
				sc.broadcastEcho(nmt, hash, round, mainChannel)
			}
		}

		// check if we should send a commit
		if !roundState.sentCommit {
			//  if we have n-t echos of the same type, then we don't care about the timeout
			echoHash := msgState.getSupportedEchoHash(round)
			if echoHash == nil && roundState.commitTimeOutState == cons.TimeoutPassed {
				// if the cons timeout passed and we have no echo to support, then we support 0
				echoHash = types.GetZeroBytesHashLength()
				logging.Infof("Sending ZERO commit message for round %v, index%v", round, sc.Index)
			} else if echoHash != nil {
				logging.Infof("Sending non-zero commit message for round %v, index%v", round, sc.Index)
			}
			if echoHash != nil {
				roundState = sc.startCommitTimeout(round, t, roundState, mainChannel)
				roundState.sentCommit = true
				sc.broadcastCommit(nmt, echoHash, round, mainChannel)
			}
		}
		sc.roundState[round] = roundState
	}

	// check if we can decide
	if commitHash := msgState.getSupportedCommitHash(round); commitHash != nil {

		// if it is a non-zero hash then we decide
		if !bytes.Equal(types.GetZeroBytesHashLength(), commitHash) {
			if sc.decisionHashBytes != nil && !bytes.Equal(sc.decisionHashBytes, commitHash) {
				panic("committed two different hashes")
			}
			if sc.decisionHashBytes == nil {
				logging.Infof("Deciding on round %v, index %v", round, sc.Index)
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, false)
				sc.decisionHashBytes = commitHash
				sc.decisionRound = round
				sc.decisionHash = types.HashStr(commitHash)
			}
			// request recover if needed
			if initMsg := sc.validatedInitHashes[sc.decisionHash]; initMsg == nil {
				// we haven't yet received the init message for the hash, so we request it from other nodes after a timeout
				sc.StartRecoverTimeout(sc.Index, mainChannel)
			} else {
				// we have the init message so we decide
				sc.decisionInitMsg = initMsg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage)
				sc.decisionPub = initMsg.Header.(*sig.MultipleSignedMessage).SigItems[0].Pub
				logging.Infof("Have decision init message index %v", sc.Index)

				// stop the timers
				sc.stopTimers()
			}
		}
	}

	// check if we can go to the next round
	if round == sc.round && sc.decisionHashBytes == nil {
		if roundState.sentCommit && roundState.commitTimeOutState == cons.TimeoutPassed {
			// We need n-t commit messages
			_, commitCount := msgState.GetValidMessageCount(sc.round)
			if commitCount >= nmt {
				if _, err := msgState.checkMvProofs(round, nmt, sc.ConsItems.MC.MC.GetMyPriv().GetPub(),
					sc.ConsItems.MC, false, false); err == nil {

					// advance to the next round
					sc.round += 1
					logging.Infof("Advance to next round %v, index %v", sc.round, sc.Index)
					if err = sc.startRound(mainChannel); err != nil {
						panic(err)
					}
				}
			}
		}
	}
}

// sendEchoTimeout starts the echo timeout
func (sc *MvCons2) startEchoTimeout(round types.ConsensusRound, t int, roundState roundMvState,
	mainChannel channelinterface.MainChannel) roundMvState {

	_ = t
	if sc.round != round { // only start the timeout for the current round
		return roundState
	}
	if roundState.echoTimeOutState == cons.TimeoutNotSent && !sc.HasDecided() {
		roundState.echoTimeOutState = cons.TimeoutSent
		sc.echoTimer = cons.StartEchoTimer(round, sc.ConsItems, mainChannel)
	}
	return roundState
}

// startCommitTimeout starts the commit timeout
func (sc *MvCons2) startCommitTimeout(round types.ConsensusRound, t int, roundState roundMvState,
	mainChannel channelinterface.MainChannel) roundMvState {
	if sc.round != round { // only start the timeout for the current round
		return roundState
	}
	if roundState.commitTimeOutState == cons.TimeoutNotSent && !sc.HasDecided() {
		roundState.commitTimeOutState = cons.TimeoutSent
		deser := []*channelinterface.DeserializedItem{
			{
				Index:          sc.Index,
				HeaderType:     messages.HdrMvCommitTimeout,
				Header:         (messagetypes.MvCommitMessageTimeout)(round),
				IsDeserialized: true,
				IsLocal:        types.LocalMessage}}
		sc.commitTimer = mainChannel.SendToSelf(deser, cons.GetMvTimeout(round, t))
	}
	return roundState
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// broadcastInit broadcasts an int message
func (sc *MvCons2) broadcastInit(newMsg messages.InternalSignedMsgHeader, proofMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	var forwardFunc channelinterface.NewForwardFuncFilter
	if config.MvBroadcastInitForBufferForwarder { // we change who we broadcast to depending on the configuration
		forwardFunc = channelinterface.ForwardAllPub // we broadcast the init message to all nodes directly
	} else {
		forwardFunc = sc.ConsItems.FwdChecker.GetNewForwardListFunc() // we propoagte the init message using gossip
	}
	sc.BroadcastFunc(nil, sc.ConsItems, newMsg, true, forwardFunc,
		mainChannel, sc.GeneralConfig, proofMsg)
	// BroadcastMv2(nil, sc.ByzType, sc, forwardFunc, newMsg, proofMsg, mainChannel)
}

// broadcastEcho broadcasts an echo message
func (sc *MvCons2) broadcastEcho(nmt int, proposalHash []byte, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) {

	newMsg := messagetypes.NewMvEchoMessage()
	newMsg.Round = round
	newMsg.ProposalHash = proposalHash
	sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

	if sc.CheckMemberLocalMsg(newMsg.GetMsgID()) { // only send the message if we are a participant of consensus
		var proofMsg messages.MsgHeader
		var err error
		if sc.includeProofs && round > 0 {
			// proofs will either be echos or commits from the previous round
			proofMsg, err = sc.ConsItems.MsgState.(*MessageState).GenerateProofs(nmt, round-1, 0,
				sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
			if err != nil {
				panic(err) // we should always have a valid proof msg here
			}
		}
		cordPub := cons.GetCoordPubCollectBroadcast(round, sc.ConsItems, sc.GeneralConfig)
		sc.BroadcastFunc(cordPub, sc.ConsItems, newMsg, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
			mainChannel, sc.GeneralConfig, proofMsg)
		// BroadcastMv2(cordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(), newMsg, proofMsg, mainChannel)
	}
}

// broadcastCommit broadcasts an commit message
func (sc *MvCons2) broadcastCommit(nmt int, proposalHash []byte, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) {

	newMsg := messagetypes.NewMvCommitMessage()
	newMsg.Round = round
	newMsg.ProposalHash = proposalHash
	sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

	if sc.CheckMemberLocalMsg(newMsg.GetMsgID()) { // only send the message if we are a participant of consensus

		// Check if we should include proofs and who to broadcast to based on the BroadcastCollect settings
		includeProofs, nxtCoordPub := cons.CheckIncludeEchoProofs(round, sc.ConsItems,
			sc.includeProofs, sc.GeneralConfig)

		var proofMsg messages.MsgHeader
		var err error
		var proofRound types.ConsensusRound
		if bytes.Equal(types.GetZeroBytesHashLength(), proposalHash) {
			// the proof here will be the n-t 0 commits from the previous round
			proofRound = round - 1
			if round == 0 { // no profs for zero hash in round 0
				includeProofs = false
			}
		} else {
			// the proof here will be the n-t echos from the current round
			proofRound = round
		}
		if includeProofs {
			proofMsg, err = sc.ConsItems.MsgState.(*MessageState).GenerateProofs(nmt, proofRound,
				0, sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
			if err != nil {
				panic(err) // we should always have a valid proof msg here
			}
		}
		sc.BroadcastFunc(nxtCoordPub, sc.ConsItems, newMsg, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
			mainChannel, sc.GeneralConfig, proofMsg)

		// BroadcastMv2(nxtCoordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(), newMsg,
		//	proofMsg, mainChannel)
	}

}

// SetInitialState does noting for this algorithm.
func (sc *MvCons2) SetInitialState([]byte) {}

// Collect is called when the item is being garbage collected.
func (sc *MvCons2) Collect() {
	sc.stopTimers()
	sc.StopRecoverTimeout()
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*MvCons2) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewMvCons2MessageState(gc)
}
