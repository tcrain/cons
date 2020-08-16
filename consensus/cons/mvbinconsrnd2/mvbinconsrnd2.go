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

package mvbinconsrnd2

import (
	"bytes"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/binconsrnd2"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

type MvBinConsRnd2 struct {
	cons.AbsConsItem
	cons.AbsMVRecover
	binCons             cons.BinConsInterface                                // the binary consensus object
	myProposal          *messagetypes.MvProposeMessage                       // My proposal for this bcast
	decisionHash        types.HashStr                                        // the hash of the decided value
	decisionHashBytes   types.HashBytes                                      // the hash of the decided value
	decisionInitMsg     *messagetypes.MvInitMessage                          // the actual decided value (should be same as proposal if nonfaulty coord)
	decisionPub         sig.Pub                                              // the pub of the decisionInitMsg
	sortedInitHashes    cons.DeserSortVRF                                    // valid init messages sorted by VRFID, used when random member selection is enabled
	validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem // hashes of init messages that have been validated by the state machine
	sentEcho            bool
	sentCommit          bool
	includeProofs       bool // if true then we should include proofs with our commit messages
	// priv                sig.Priv             // the local nodes private key

	initTimer        channelinterface.TimerInterface // timer for the init message
	initTimeoutState cons.TimeoutState               // the timeout concerning the init message
}

// GetConsType returns the type of consensus this instance implements.
func (sc *MvBinConsRnd2) GetConsType() types.ConsType {
	return types.MvBinConsRnd2Type
}

// checkInitTimeoutPassed returns true if the init timeout has already expired, or bincons is past round 1
func (sc *MvBinConsRnd2) checkInitTimeoutPassed() bool {
	if sc.initTimeoutState == cons.TimeoutPassed {
		return true
	}
	return false
}

// GenerateNewItem creates a new cons item.
func (*MvBinConsRnd2) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &MvBinConsRnd2{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.includeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.validatedInitHashes = make(map[types.HashStr]*channelinterface.DeserializedItem)
	newItem.InitAbsMVRecover(index)

	binConsItem := &binconsrnd2.BinConsRnd2{}
	newItem.binCons = binConsItem.GenerateNewItem(index, items, mainChannel,
		prevItem, broadcastFunc, gc).(cons.BinConsInterface)

	return newItem
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *MvBinConsRnd2) GetCommitProof() []messages.MsgHeader {

	// the commit comes from the bin cons
	return sc.binCons.GetCommitProof()
	/*	t := sc.ConsItems.MC.MC.GetFaultCount()
		nmt := sc.ConsItems.MC.MC.GetMemberCount() - t

		commitMsg := messagetypes.NewMvCommitMessage()
		commitMsg.ProposalHash = sc.decisionHashBytes

		// Add sigs
		prfMsgSig, err := sc.ConsItems.MsgState.SetupSignedMessage(commitMsg, false, nmt, sc.ConsItems.MC)
		if err != nil {
			panic(err)
		}
		return []messages.MsgHeader{prfMsgSig}
	*/
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *MvBinConsRnd2) SetNextConsItem(consinterface.ConsItem) {
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *MvBinConsRnd2) PrevHasBeenReset() {
}

// HasValidStarted returns true if the cons has received a valid proposal, or the binary reduction has terminated.
func (sc *MvBinConsRnd2) HasValidStarted() bool {
	dec, _ := sc.binCons.GetBinDecided()
	return len(sc.validatedInitHashes) > 0 || dec >= 0
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *MvBinConsRnd2) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *MvBinConsRnd2) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrMvInit returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
// (2) HdrMvEcho returns n-t, n for the thresholds.
// (3) HdrMvCommit returns n-t, n for the thresholds.
func (sc *MvBinConsRnd2) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig,
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
	}
	return sc.binCons.GetBufferCount(hdr, gc, memberChecker)
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// The valid headers are HdrMvInit, HdrMvEcho, HdrMvCommit, HdrMvRequestRecover.
func (*MvBinConsRnd2) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig,
	headerType messages.HeaderID) (messages.MsgHeader, error) {

	var signMsg bool
	var internalMsg messages.InternalSignedMsgHeader

	switch headerType {
	case messages.HdrMvInit:
		internalMsg = messagetypes.NewMvInitMessage()
		if gc.PartialMessageType != types.NoPartialMessages {
			// if partials are used then MvInit must always construct into combined messages
			// TODO add test where a combined message is sent with a different internal header type
			internalMsg = messagetypes.NewCombinedMessage(internalMsg)
			return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, internalMsg), nil
		}
		if gc.NetworkType == types.RequestForwarder || !gc.NoSignatures {
			signMsg = true
		}
	case messages.HdrMvEcho:
		signMsg = !gc.NoSignatures
		internalMsg = messagetypes.NewMvEchoMessage()
	case messages.HdrMvCommit:
		signMsg = !gc.NoSignatures
		internalMsg = messagetypes.NewMvCommitMessage()
	case messages.HdrMvRequestRecover:
		return messagetypes.NewMvRequestRecoverMessage(), nil
	case messages.HdrPartialMsg:
		if gc.PartialMessageType != types.NoPartialMessages {
			return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewPartialMessage()), nil
		}
	}
	if internalMsg == nil {
		return (&binconsrnd2.BinConsRnd2{}).GetHeader(emptyPub, gc, headerType)
	}
	if signMsg {
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, internalMsg), nil
	}
	return sig.NewUnsignedMessage(types.ConsensusIndex{}, emptyPub, internalMsg), nil
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *MvBinConsRnd2) ShouldCreatePartial(headerType messages.HeaderID) bool {
	if sc.PartialMessageType != types.NoPartialMessages && headerType == messages.HdrMvInit {
		return true
	}
	return false
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *MvBinConsRnd2) HasDecided() bool {
	dec, _ := sc.binCons.GetBinDecided()
	switch dec {
	case 0:
		return true
	case 1:
		if sc.decisionInitMsg != nil {
			return true
		}
	}
	return false
}

// GetDecision returns the decided value as a byte slice.
func (sc *MvBinConsRnd2) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	var retIdx types.ConsensusIndex
	switch v := sc.Index.Index.(type) {
	case types.ConsensusInt:
		retIdx = types.SingleComputeConsensusIDShort(v - 1)
	}

	dec, _ := sc.binCons.GetBinDecided()
	switch dec {
	case 1:
		if sc.decisionInitMsg != nil {
			if len(sc.decisionInitMsg.Proposal) == 0 {
				panic(sc.Index)
			}
			return sc.decisionPub, sc.decisionInitMsg.Proposal, retIdx
		}
	case 0:
		return sc.decisionPub, nil, retIdx
	}
	panic("should have decided")
}

// GotProposal takes the proposal, and broadcasts it if it is the leader.
func (sc *MvBinConsRnd2) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {

	sc.AbsGotProposal()
	sc.myProposal = hdr.(*messagetypes.MvProposeMessage)
	if sc.myProposal.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	initMsg := messagetypes.NewMvInitMessage()
	initMsg.Proposal = sc.myProposal.Proposal
	initMsg.ByzProposal = sc.myProposal.ByzProposal
	logging.Infof("Sending proposal, index %v", sc.Index)
	sc.broadcastInit(initMsg, mainChannel)

	return nil
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *MvBinConsRnd2) CanStartNext() bool {
	return true
}

// Start allows GetProposalIndex to return true.
func (sc *MvBinConsRnd2) Start() {
	sc.AbsConsItem.AbsStart()
	logging.Infof("Starting MvBinConsRnd2 index %v", sc.Index)
	if sc.CheckMemberLocal() {
		initMsg := messagetypes.NewMvInitMessage()
		// Check if we are the proposer
		_, _, err := consinterface.CheckCoord(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC, 0, initMsg.GetMsgID())
		if err == nil {
			sc.NeedsProposal = true
			logging.Info("I am coordinator for index", sc.Index, sc.GeneralConfig.TestIndex)
		} else {
			logging.Info("I am NOT coordinator for index", sc.Index, sc.GeneralConfig.TestIndex, err)
		}
		// Start the init timer
		sc.initTimer = cons.StartInitTimer(0, sc.ConsItems, sc.MainChannel)
		sc.initTimeoutState = cons.TimeoutSent
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *MvBinConsRnd2) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created. // TODO
func (sc *MvBinConsRnd2) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1),
		nil, nil, true
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// The following are the valid message types:
// messages.HdrMvInit is the leader proposal, once this is received an echo is sent containing the hash, and starts the echo timeoutout.
// messages.HdrMvEcho is the echo message, when these are received we run CheckEchoState.
// messages.HdrMvRequestRecover a node terminated bin cons with 1, but didn't get the init message, so if we have it we send it.
// messages.HdrMvRecoverTimeout if a node terminated bin cons with 1, but didn't get the init mesage this timeout is started, once it runs out, we ask other nodes to send the init message.
func (sc *MvBinConsRnd2) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (bool, bool) {

	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t
	switch deser.HeaderType {
	case messages.HdrMvInitTimeout:
		sc.initTimeoutState = cons.TimeoutPassed
		sc.checkProgress(t, nmt, sc.MainChannel)
		return false, false
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
		return false, false
	case messages.HdrMvInit:
		// w := deser.Header.(*sig.MultipleSignedMessage)
		// sanity checks to ensure the init message comes from the coordinator
		err := consinterface.CheckMemberCoordHdr(sc.ConsItems.MC, 0, deser.Header) // sanity check
		if err != nil {
			panic("should have handled this in GotMsg")
		}
		// Store it in a map of the proposals
		hashStr := types.HashStr(types.GetHash(
			deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal))
		if len(sc.validatedInitHashes) > 0 {
			logging.Infof("Recived duplicate proposals from coord at index %v", sc.Index)
		}
		sc.validatedInitHashes[hashStr] = deser
		sc.sortedInitHashes = append(sc.sortedInitHashes, deser)
		logging.Info("Got an mv init message of len", len(
			deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal))

		sc.checkProgress(t, nmt, sc.MainChannel)
		// send any recovers that migt have requested this init msg
		sc.SendRecover(sc.validatedInitHashes, sc.InitHeaders, sc.ConsItems)
		return true, true
	case messages.HdrMvEcho, messages.HdrMvCommit:
		// check if we have enough echos to decide
		if cons.GetMvMsgRound(deser) != 0 {
			panic("should have caught this in msg state")
		}
		sc.checkProgress(t, nmt, sc.MainChannel)
		return true, true
	case messages.HdrMvRequestRecover: // a node terminated bin cons, but didn't receive the init message
		sc.GotRequestRecover(sc.validatedInitHashes, deser, sc.InitHeaders, senderChan, sc.ConsItems)
		return false, false
	default: // This will be a bin cons message, so process it in the binary consensus
		_, forward := sc.binCons.ProcessMessage(deser, isLocal, senderChan)
		sc.checkProgress(t, nmt, sc.MainChannel)
		// TODO don't always return true?
		return true, forward
	}
}

// NeedsConcurrent returns 1.
func (sc *MvBinConsRnd2) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// checkProgress checks if we should perform an action
func (sc *MvBinConsRnd2) checkProgress(t, nmt int, mainChannel channelinterface.MainChannel) {

	if sc.HasDecided() {
		return
	}
	msgState := sc.ConsItems.MsgState.(*MessageState)

	// check if we should send an echo
	if !sc.sentEcho {
		for hash := range sc.validatedInitHashes {
			sc.sentEcho = true
			// hashBytes := messages.GetHash(msg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal)
			hashBytes := types.HashBytes(hash)
			sc.broadcastEcho(hashBytes, mainChannel)
			break
		}
	}

	// send commit if needed
	if !sc.sentCommit {
		// if we got n-t echos then we can send a commit msg
		if commitHash := msgState.getSupportedEchoHash(); commitHash != nil {
			sc.sentCommit = true
			sc.broadcastCommit(nmt, commitHash, mainChannel)
		} else if commitHash, count := msgState.getSupportedCommitHash(); count > t {
			// Otherwise if we got at least t+1 commits
			sc.sentCommit = true
			sc.broadcastCommit(nmt, commitHash, mainChannel)
		}
	}

	// check if we can deliver
	if commitHash, count := msgState.getSupportedCommitHash(); count >= nmt {

		// if it is a non-zero hash then we decide
		if sc.decisionHashBytes != nil && !bytes.Equal(sc.decisionHashBytes, commitHash) {
			panic("committed two different hashes")
		}
		if sc.decisionHashBytes == nil {
			// let the binary know 1 is valid
			msgState.BinConsMessageStateInterface.SetMv1Valid(sc.ConsItems.MC)

			logging.Infof("Delivering index %v", sc.Index)
			sc.ConsItems.MC.MC.GetStats().AddFinishRound(0, false)
			sc.decisionHashBytes = commitHash
			sc.decisionHash = types.HashStr(commitHash)
		}
		// request recover if needed
		if initMsg := sc.validatedInitHashes[sc.decisionHash]; initMsg == nil {
			// we haven't yet received the init message for the hash, so we request it from other nodes after a timeout
			sc.StartRecoverTimeout(sc.Index, mainChannel)
		} else {
			// we have the init message so we decide
			sc.decisionInitMsg = initMsg.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.MvInitMessage)
			sc.decisionPub = sig.GetSingleSupporter(initMsg.Header)
			logging.Infof("Have decision init message index %v", sc.Index)
			sc.stopTimers()
		}
	}

	// If the echo timeout passed, and we haven't delivered a value we can support 0
	if sc.checkInitTimeoutPassed() && sc.decisionHashBytes == nil {
		// check if we have not supported any value for round 1
		// (the first argument is 0 here because we the structure keeps track of having sent proposal for the round+1)
		if !msgState.BinConsMessageStateInterface.SentProposal(0, true, sc.ConsItems.MC) {
			auxMsg := sc.binCons.GetMVInitialRoundBroadcast(0)
			if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) { // check if we are a member for this type of message
				// We support 0 since we have not gotten a proposal
				logging.Infof("Supporting 0 for index %v, since not enough echos received", sc.Index)
				sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, !sc.NoSignatures,
					sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, nil)
				// cons.BroadcastBin(nil, sc.ByzType, sc.binCons, auxMsg, mainChannel)
			}
		}
	}
	// Check if bincons has made progress
	if dec, _ := sc.binCons.GetBinDecided(); dec == -1 {

		var round types.ConsensusRound = 1
		for sc.binCons.CheckRound(nmt, t, round, mainChannel) {
			round++
		}
	}
	if dec, _ := sc.binCons.GetBinDecided(); dec > -1 {
		sc.stopTimers()
	}
}

// stopTimers is used to stop running timers
func (sc *MvBinConsRnd2) stopTimers() {
	if sc.initTimer != nil {
		sc.initTimer.Stop()
		sc.initTimer = nil
	}
	sc.StopRecoverTimeout()
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// broadcastInit broadcasts an int message
func (sc *MvBinConsRnd2) broadcastInit(initMsg *messagetypes.MvInitMessage,
	mainChannel channelinterface.MainChannel) {

	var forwardFunc channelinterface.NewForwardFuncFilter
	if config.MvBroadcastInitForBufferForwarder { // we change who we broadcast to depending on the configuration
		forwardFunc = channelinterface.ForwardAllPub // we broadcast the init message to all nodes directly
	} else {
		forwardFunc = sc.ConsItems.FwdChecker.GetNewForwardListFunc() // we propoagte the init message using gossip
	}
	var signMsg bool
	if sc.NetworkType == types.RequestForwarder || !sc.NoSignatures {
		signMsg = true
	}
	sc.BroadcastFunc(nil, sc.ConsItems, initMsg, signMsg, forwardFunc,
		mainChannel, sc.GeneralConfig, sc.CommitProof...)

	// BroadcastMvBinConsRnd2(nil, sc.ByzType, sc, forwardFunc, initMsg, sc.CommitProof, mainChannel)
}

// broadcastEcho broadcasts an echo message
func (sc *MvBinConsRnd2) broadcastEcho(proposalHash []byte,
	mainChannel channelinterface.MainChannel) {

	newMsg := messagetypes.NewMvEchoMessage()
	newMsg.ProposalHash = proposalHash

	// if sc.CheckMemberLocalMsg(newMsg.GetMsgID()) { // only send the message if we are a participant of consensus
	cordPub := cons.GetCoordPubCollectBroadcast(0, sc.ConsItems, sc.GeneralConfig)
	sc.BroadcastFunc(cordPub, sc.ConsItems, newMsg, !sc.NoSignatures,
		sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		mainChannel, sc.GeneralConfig, nil)

	//BroadcastMvBinConsRnd2(cordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(), newMsg,
	//	nil, mainChannel)
	//}
}

// broadcastCommit broadcasts a commit message
func (sc *MvBinConsRnd2) broadcastCommit(nmt int, proposalHash []byte,
	mainChannel channelinterface.MainChannel) {

	newMsg := messagetypes.NewMvCommitMessage()
	newMsg.ProposalHash = proposalHash

	if sc.CheckMemberLocalMsg(newMsg.GetMsgID()) { // only send the message if we are a participant of consensus

		// Check if we should include proofs and who to broadcast to based on the BroadcastCollect settings
		includeProofs, nxtCoordPub := cons.CheckIncludeEchoProofs(0, sc.ConsItems,
			sc.includeProofs, sc.GeneralConfig)
		var prfMsgs []*sig.MultipleSignedMessage
		var err error

		if includeProofs {
			prfMsgs, err = sc.ConsItems.MsgState.(*MessageState).GenerateProofs(newMsg.GetID(), nmt, 0,
				1, sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
			if err != nil {
				// we may get an error here in case we sent the commit from commits instead of echos
				logging.Error(err)
			}
		}
		prfs := make([]messages.MsgHeader, len(prfMsgs))
		for i, nxt := range prfMsgs {
			prfs[i] = nxt
		}
		sc.BroadcastFunc(nxtCoordPub, sc.ConsItems, newMsg, !sc.NoSignatures,
			sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
			mainChannel, sc.GeneralConfig, prfs...)

		//BroadcastMvBinConsRnd2(nxtCoordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		//	newMsg, []messages.MsgHeader{proofMsg}, mainChannel)
	}
}

// SetInitialState does noting for this algorithm.
func (sc *MvBinConsRnd2) SetInitialState([]byte) {}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*MvBinConsRnd2) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewMvBinConsRnd2MessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *MvBinConsRnd2) Collect() {
	sc.AbsConsItem.Collect()
	sc.binCons.Collect()
	sc.stopTimers()
	sc.StopRecoverTimeout()
}
