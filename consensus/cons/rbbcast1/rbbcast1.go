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

package rbbcast1

import (
	"bytes"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

type RbBcast1 struct {
	cons.AbsConsItem
	cons.AbsMVRecover
	myProposal          *messagetypes.MvProposeMessage                   // My proposal for this bcast
	decisionHash        types.HashStr                                    // the hash of the decided value
	decisionHashBytes   types.HashBytes                                  // the hash of the decided value
	decisionInitMsg     *messagetypes.MvInitMessage                      // the actual decided value (should be same as proposal if nonfaulty coord)
	decisionPub         sig.Pub                                          // the pub of the decisionInitMsg
	validatedInitHashes map[types.HashStr]*deserialized.DeserializedItem // hashes of init messages that have been validated by the state machine
	sentEcho            bool
	// priv                sig.Priv             // the local nodes private key
}

// GetConsType returns the type of consensus this instance implements.
func (sc *RbBcast1) GetConsType() types.ConsType {
	return types.RbBcast1Type
}

// GenerateNewItem creates a new cons item.
func (*RbBcast1) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &RbBcast1{AbsConsItem: newAbsItem}

	// newItem.priv = gc.Priv
	items.ConsItem = newItem
	newItem.validatedInitHashes = make(map[types.HashStr]*deserialized.DeserializedItem)
	newItem.InitAbsMVRecover(index, gc)

	return newItem
}

// GetPrevCommitProof returns a signed message header that counts at the commit message for the previous consensus.
// This should only be called after DoneKeep has been called on this instance.
// cordPub is the expected public key of the coordinator of the current round (used for collect broadcast)
func (sc *RbBcast1) GetPrevCommitProof() (cordPub sig.Pub, proof []messages.MsgHeader) {
	cordPub = cons.GetCoordPubCollectBroadcastEnd(0, sc.ConsItems, sc.GeneralConfig)
	_, proof = sc.AbsConsItem.GetPrevCommitProof()
	return
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *RbBcast1) GetCommitProof() []messages.MsgHeader {
	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t

	commitMsg := messagetypes.NewMvEchoMessage()
	commitMsg.ProposalHash = sc.decisionHashBytes

	// Add sigs
	prfMsgSig, err := sc.ConsItems.MsgState.SetupSignedMessage(commitMsg, false, nmt, sc.ConsItems.MC)
	if err != nil {
		panic(err)
	}
	return []messages.MsgHeader{prfMsgSig}
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *RbBcast1) SetNextConsItem(consinterface.ConsItem) {
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *RbBcast1) PrevHasBeenReset() {
}

// HasReceivedProposal returns true if the cons has received a valid proposal.
func (sc *RbBcast1) HasValidStarted() bool {
	return len(sc.validatedInitHashes) > 0
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *RbBcast1) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *RbBcast1) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrMvInit returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
// (2) HdrMvEcho returns n-t, n for the thresholds.
func (sc *RbBcast1) GetBufferCount(hdr messages.MsgIDHeader, _ *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrMvInit:
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
// The valid headers are HdrMvInit, HdrMvEcho, HdrMvCommit, HdrMvRequestRecover.
func (*RbBcast1) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
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
func (sc *RbBcast1) ShouldCreatePartial(headerType messages.HeaderID) bool {
	if sc.PartialMessageType != types.NoPartialMessages && headerType == messages.HdrMvInit {
		return true
	}
	return false
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *RbBcast1) HasDecided() bool {
	if sc.decisionInitMsg != nil {
		return true
	}
	return false
}

// GetDecision returns the decided value as a byte slice.
func (sc *RbBcast1) GetDecision() (sig.Pub, []byte, types.ConsensusIndex, types.ConsensusIndex) {
	if sc.decisionInitMsg != nil {
		if len(sc.decisionInitMsg.Proposal) == 0 {
			panic(sc.Index)
		}
		var retIdx, futureIdx types.ConsensusIndex
		switch v := sc.Index.Index.(type) {
		case types.ConsensusInt:
			retIdx = types.SingleComputeConsensusIDShort(v - 1)
			futureIdx = types.SingleComputeConsensusIDShort(v + 1)
		}
		return sc.decisionPub, sc.decisionInitMsg.Proposal, retIdx, futureIdx
	}
	panic("should have decided")
}

// GotProposal takes the proposal, and broadcasts it if it is the leader.
func (sc *RbBcast1) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {

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
func (sc *RbBcast1) CanStartNext() bool {
	return true
}

// Start allows GetProposalIndex to return true.
func (sc *RbBcast1) Start() {
	sc.AbsConsItem.AbsStart()
	logging.Infof("Starting RbBcast1 index %v", sc.Index)
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
	}

}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *RbBcast1) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created. // TODO
func (sc *RbBcast1) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1),
		nil, nil, sc.GeneralConfig.AllowConcurrent > 0
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
func (sc *RbBcast1) ProcessMessage(
	deser *deserialized.DeserializedItem,
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
		// Store it in a map of the proposals
		hashStr := types.HashStr(types.GetHash(deser.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal))
		sc.validatedInitHashes[hashStr] = deser
		w := deser.Header.(*sig.MultipleSignedMessage)
		if cons.GetMvMsgRound(deser) != 0 {
			panic("should have caught this in msg state")
		}

		// sanity checks to ensure the init message comes from the coordinator
		err := consinterface.CheckMemberCoord(sc.ConsItems.MC, 0, w.SigItems[0], w) // sanity check
		if err != nil {
			panic("should have handled this in GotMsg")
		}
		logging.Infof("Got an mv init message of len %v, index %v",
			len(w.GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal), sc.Index)

		sc.checkProgress(t, nmt, sc.MainChannel)
		// send any recovers that migt have requested this init msg
		sc.SendRecover(sc.validatedInitHashes, sc.InitHeaders, sc.ConsItems)
		return true, true
	case messages.HdrMvEcho:
		// check if we have enough echos to decide
		if cons.GetMvMsgRound(deser) != 0 {
			panic("should have caught this in msg state")
		}
		sc.checkProgress(t, nmt, sc.MainChannel)
		return true, true
	case messages.HdrMvRequestRecover: // a node terminated bin cons, but didn't receive the init message
		sc.GotRequestRecover(sc.validatedInitHashes, deser, sc.InitHeaders, senderChan, sc.ConsItems)
		return false, false
	default:
		panic("unknown msg type")
	}
}

// NeedsConcurrent returns 1.
func (sc *RbBcast1) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// checkProgress checks if we should perform an action
func (sc *RbBcast1) checkProgress(t, nmt int, mainChannel channelinterface.MainChannel) {

	_ = t
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
			sc.broadcastEcho(nmt, hashBytes, mainChannel)
			break
		}
	}

	// check if we can decide
	if commitHash := msgState.getSupportedEchoHash(); commitHash != nil {

		// if it is a non-zero hash then we decide
		if sc.decisionHashBytes != nil && !bytes.Equal(sc.decisionHashBytes, commitHash) {
			panic("committed two different hashes")
		}
		if sc.decisionHashBytes == nil {
			logging.Infof("Deciding index %v", sc.Index)
			sc.ConsItems.MC.MC.GetStats().AddFinishRound(1, false)
			sc.decisionHashBytes = commitHash
			sc.decisionHash = types.HashStr(commitHash)
		}
		// request recover if needed
		if initMsg := sc.validatedInitHashes[sc.decisionHash]; initMsg == nil {
			// we haven't yet received the init message for the hash, so we request it from other nodes after a timeout
			sc.StartRecoverTimeout(sc.Index, mainChannel, sc.ConsItems.MC)
		} else {
			// we have the init message so we decide
			sc.decisionInitMsg = initMsg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage)
			sc.decisionPub = initMsg.Header.(*sig.MultipleSignedMessage).SigItems[0].Pub
			logging.Infof("Have decision init message index %v", sc.Index)
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// broadcastInit broadcasts an int message
func (sc *RbBcast1) broadcastInit(initMsg *messagetypes.MvInitMessage,
	mainChannel channelinterface.MainChannel) {

	sc.ConsItems.MC.MC.GetStats().BroadcastProposal()
	var forwardFunc channelinterface.NewForwardFuncFilter
	if config.MvBroadcastInitForBufferForwarder { // we change who we broadcast to depending on the configuration
		forwardFunc = channelinterface.ForwardAllPub // we broadcast the init message to all nodes directly
	} else {
		forwardFunc = sc.ConsItems.FwdChecker.GetNewForwardListFunc() // we propoagte the init message using gossip
	}
	sc.BroadcastFunc(nil, sc.ConsItems, initMsg, true, forwardFunc,
		mainChannel, sc.GeneralConfig, sc.CommitProof...)
	// BroadcastRbBcast1(nil, sc.ByzType, sc, forwardFunc, initMsg, sc.CommitProof, mainChannel)
}

// broadcastEcho broadcasts an echo message
func (sc *RbBcast1) broadcastEcho(nmt int, proposalHash []byte,
	mainChannel channelinterface.MainChannel) {

	_ = nmt
	newMsg := messagetypes.NewMvEchoMessage()
	newMsg.ProposalHash = proposalHash

	// see if we should broadcast directly to the next coordinator, or to all nodes
	nxtCoordPub := cons.GetNextCoordPubCollectBroadcast(0, sc.ConsItems, sc.GeneralConfig)
	sc.BroadcastFunc(nxtCoordPub, sc.ConsItems, newMsg, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		mainChannel, sc.GeneralConfig, nil)
}

// SetInitialState does noting for this algorithm.
func (sc *RbBcast1) SetInitialState([]byte) {}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*RbBcast1) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewRbBcast1MessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *RbBcast1) Collect() {
	sc.AbsConsItem.Collect()
	sc.StopRecoverTimeout()
}
