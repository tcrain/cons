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
package mvcons4

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/graph"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/storage"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

type MvCons4 struct {
	cons.AbsConsItem
	gs                *globalMessageState
	myProposal        *messagetypes.MvProposeMessage // My proposal for this round
	sentProposal      bool
	broadcastType     types.MvCons4BcastType
	finishedLastRound bool

	initialHash types.HashBytes
	hasDecided  bool
	prev        *MvCons4
	rand        *rand.Rand
}

// SetInitialState sets the value that is supported by the inital index (1).
func (sc *MvCons4) SetInitialState(value []byte, store storage.StoreInterface) {
	sc.initialHash = types.GetHash(value)
	if sc.Index.Index.(types.ConsensusInt) != 1 {
		panic("should be called on index 1")
	}
	if sc.prev != nil {
		panic("initial prev should be nil")
	}
	n := sc.ConsItems.MC.MC.GetMemberCount()
	nmt := n - sc.ConsItems.MC.MC.GetFaultCount()
	participantCount := len(sc.ConsItems.MC.MC.GetAllPubs())
	sc.gs = initGlobalMessageState(n, nmt, participantCount, sc.GeneralConfig.TestIndex, sc.initialHash, store,
		sc.ConsItems.MC.MC.GetStats(), sc.GeneralConfig)
	sc.ConsItems.MsgState.(*MessageState).gs = sc.gs
}

// GetCommitProof returns nil for MvCons4. It generates the proofs itself since it
// piggybacks rounds with consensus instances.
func (sc *MvCons4) GetCommitProof() (proof []messages.MsgHeader) {
	return
}

// GetConsType returns the type of consensus this instance implements.
func (sc *MvCons4) GetConsType() types.ConsType {
	return types.MvCons4Type
}

// GetPrevCommitProof is unsupported
func (sc *MvCons4) GetPrevCommitProof() (coordPub sig.Pub, proof []messages.MsgHeader) {
	return
}

// Start allows GetProposalIndex to return true.
func (sc *MvCons4) Start(finishedLastRound bool) {
	sc.Started = true
	logging.Infof("Starting consensus index %v", sc.Index)

	sc.AbsConsItem.AbsStart()
	sc.finishedLastRound = finishedLastRound
	logging.Infof("Got local start multivalue index %v", sc.Index)
	if sc.CheckMemberLocal() {
		sc.NeedsProposal = true
	}
}

// HasValidStarted is unsupported
func (sc *MvCons4) HasValidStarted() bool {
	panic("unsupported")
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *MvCons4) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// SetNextConsItem is unused.
func (sc *MvCons4) SetNextConsItem(next consinterface.ConsItem) {
	_ = next
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index,
// it is unused.
func (sc *MvCons4) PrevHasBeenReset() {}

// ResetState resets the stored state for the current consensus index, and prepares for the new consensus index given by the input.
func (*MvCons4) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	if gc.AllowConcurrent > 0 {
		panic("MvCons4 does not allow concurrent")
	}
	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &MvCons4{
		AbsConsItem: newAbsItem,
	}
	newItem.broadcastType = gc.MvCons4BcastType

	if index.Index.(types.ConsensusInt) != 1 {
		newItem.gs = prevItem.(*MvCons4).gs
		items.MsgState.(*MessageState).gs = newItem.gs
		newItem.prev = prevItem.(*MvCons4)
		newItem.rand = prevItem.(*MvCons4).rand
		newItem.broadcastType = prevItem.(*MvCons4).broadcastType
		newItem.gs.startedIdx(index, items.MC.MC)
	} else {
		if prevItem != nil {
			panic("prev should be nil for initial index")
		}
		newItem.rand = rand.New(rand.NewSource(int64(gc.TestIndex)))
	}
	items.ConsItem = newItem
	//if !newItem.CheckMemberLocal() {
	//	panic("all participants must be members")
	//}

	return newItem
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *MvCons4) GetBinState(localOnly bool) ([]byte, error) {
	msg, err := messages.CreateMsg(sc.PreHeaders)
	if err != nil {
		panic(err)
	}
	bs, err := sc.ConsItems.MsgState.(*MessageState).GetMsgState(sc.ConsItems.MC.MC.GetMyPriv(), localOnly,
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

// GetCustomRecoverMsg is called when there is no progress after a timeout.
// It returns an IndexRecoverMsg
func (sc *MvCons4) GetCustomRecoverMsg(createEmpty bool) messages.MsgHeader {
	if createEmpty {
		return messagetypes.NewIndexRecoverMsg(types.ConsensusIndex{})
	} else {
		return sc.gs.getRecoverMsg()
	}
}

// GetRecoverMsgType returns messages.HdrIdxRecover.
func (sc *MvCons4) GetRecoverMsgType() messages.HeaderID {
	return messages.HdrIdxRecover
}

// ForwardOldIndices returns true since events from old indices may be needed for future decisions.
func (sc *MvCons4) ForwardOldIndices() bool {
	return true
}

// ProcessCustomRecoveryMessage handles recovery messages of type IndexRecoverMessage.
func (sc *MvCons4) ProcessCustomRecoveryMessage(item *deserialized.DeserializedItem,
	sendChan *channelinterface.SendRecvChannel) {

	rcvMsg := item.Header.(*messagetypes.IndexRecoverMsg)
	msgs, byts := sc.gs.getRecoverReply(rcvMsg.Indices, rcvMsg.MissingDependencies)
	if sc.MvCons4BcastType != types.Normal { // if it a no progress message then we include an indices messages to trigger a sync
		msgs = append(msgs, sc.gs.getMyIndicesMsg(false, sc.ConsItems.MC))
	}
	if len(byts) > 0 || len(msgs) > 0 {
		sndMsg, err := messages.CreateMsgFromBytes(byts)
		utils.PanicNonNil(err)
		sndMsg, err = messages.AppendHeaders(sndMsg, msgs)
		utils.PanicNonNil(err)
		sts := sc.gs.getStats(sc.ConsItems.MC.MC)
		sendChan.MainChan.SendTo(sndMsg.GetBytes(), sendChan.ReturnChan, sts.IsRecordIndex(),
			sts)
	}
}

// GetProposeHeaderID returns the HeaderID messages.HdrMvPropose that will be input to GotProposal.
func (sc *MvCons4) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrEventInfo returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
func (sc *MvCons4) GetBufferCount(hdr messages.MsgIDHeader, _ *generalconfig.GeneralConfig,
	_ *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrEventInfo:
		if config.MvBroadcastInitForBufferForwarder {
			panic("unsupported")
		}
		return 1, 1, hdr.GetMsgID(), nil // otherwise we propagate it through gossip
	default:
		return 0, 0, nil, types.ErrInvalidHeader
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// The valid headers are HdrEventInfo.
func (*MvCons4) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrEventInfo:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewEventMessage()), nil
	case messages.HdrIdx:
		if gc.MvCons4BcastType == types.Normal {
			return nil, types.ErrInvalidHeader
		}
		hdr := messagetypes.NewIndexMessage() // we don't need to sign index messages since they are just ids
		if gc.EncryptChannels {
			return sig.NewUnsignedMessage(types.ConsensusIndex{}, emptyPub, hdr), nil
		}
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
			hdr), nil
	default:
		return nil, types.ErrInvalidHeader
	}
}

// ShouldCreatePartial returns false.
func (sc *MvCons4) ShouldCreatePartial(headerType messages.HeaderID) bool {
	_ = headerType

	return false
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *MvCons4) HasDecided() bool {
	sc.checkDecidedInternal()
	return sc.hasDecided
}

// GetDecision returns the decided value of the consensus. It should only be called after HasDecided returns true.
// Proposer is nil, prvIdx is the current index - 1,
// futureFixed is the first larger index that this decision does not depend, i.e.
// the current index + 1.
func (sc *MvCons4) GetDecision() (proposer sig.Pub, decision []byte, prvIdx, futureFixed types.ConsensusIndex) {
	prvIdx = types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
	futureFixed = types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) + 1)
	dec, _ := sc.gs.getDecision(sc.Index)

	for _, nxt := range dec {
		for _, d := range nxt {
			decision = append(decision, d...)

			// TODO for now just return the first non-nil decision
			decision = d
			return
		}
	}
	return
}

// GotProposal takes the proposal, and broadcasts it.
func (sc *MvCons4) GotProposal(hdr messages.MsgHeader, _ channelinterface.MainChannel) error {
	sc.myProposal = hdr.(*messagetypes.MvProposeMessage)
	if sc.myProposal.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	sc.ConsItems.MC.MC.GetStats().BroadcastProposal()

	sc.gs.addProposal(sc.Index, sc.myProposal.Proposal)
	// sc.gs.addCreateMessageOnIndices(make([]graph.IndexType, sc.ConsItems.MC.MC.GetMemberCount()))
	// sc.doBroadcast(nil, mainChannel)
	// if sc.Index.Index.(types.ConsensusInt) == 1 {
	switch sc.broadcastType {
	case types.Normal:
		sc.checkBroadcastNormal()
	case types.Direct:
		sc.broadcastEventInfoFromEvent(nil, sc.MainChannel)
	case types.Indices:
		sc.broadcastEventInfoFromIndices(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), true, nil,
			sc.gs.getMyIndices(), graph.IndexType(sc.getRandReceiver(false).GetIndex()))
	default:
		panic(sc.broadcastType)
	}
	// }
	return nil
}

func (sc *MvCons4) checkBroadcastNormal() {
	if !sc.checkCreateEvent() { // check if we are a member
		return
	}
	if msgs := sc.gs.checkCreateEventAll2Al(sc.ConsItems.MC); len(msgs) > 0 {
		sc.checkDecidedInternal()                                      // check if the new message made us decide
		forwardFunc := sc.ConsItems.FwdChecker.GetNewForwardListFunc() // propagate message using gossip
		sts := sc.gs.getStats(sc.ConsItems.MC.MC)
		sc.MainChannel.SendHeader(messages.AppendCopyMsgHeader(sc.GetPreHeader(), msgs...),
			true, false, forwardFunc,
			sts.IsRecordIndex(), sts)
	}
}

func (sc *MvCons4) doBroadcast(fromMsg *messagetypes.EventMessage, mainChannel channelinterface.MainChannel) {
	switch sc.broadcastType {
	case types.Direct:
		sc.broadcastEventInfoFromEvent(fromMsg, mainChannel)
	case types.Indices:
		sc.checkBroadcastIndices(mainChannel)
	case types.Normal:
		sc.checkBroadcastNormal()
	default:
		panic(sc.broadcastType)
	}
}

func (sc *MvCons4) checkDecidedInternal() {
	if !sc.hasDecided && sc.gs.hasDecided(sc.Index, sc.ConsItems.MC) {
		sc.ConsItems.MC.MC.GetStats().AddCustomStartTime(sc.gs.GetStartTime(sc.Index))
		sc.hasDecided = true
		sc.SetDecided()
	}
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// The following are the valid message types:
// messages.HdrEventInfo is an event info message.
func (sc *MvCons4) ProcessMessage(
	deser *deserialized.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (progress bool, forward bool) {

	_, _ = isLocal, senderChan
	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	switch deser.HeaderType {
	case messages.HdrEventInfo:
		sc.checkDecidedInternal()
		switch sc.MvCons4BcastType {
		case types.Normal:
			sc.checkBroadcastNormal()
		default:
			sc.checkBroadcastIndices(sc.MainChannel)
		}
		return true, true
	case messages.HdrIdx:
		msg := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.IndexMessage)
		fromPub := sig.GetSingleSupporter(deser.Header)
		destID := graph.IndexType(fromPub.GetIndex())
		sc.gs.gotIndicesMsg(destID, msg)
		switch sc.MvCons4BcastType {
		case types.Normal:
			panic("should not reach")
		case types.Direct:
			sc.gs.addCreateMessageOnIndices(msg.Indices) // we bcast the indices when satisfied
			sc.checkBroadcastIndices(sc.MainChannel)     // check if indices satisfied
		case types.Indices:
			if msg.IsReply { // this is a reply from a sync done by a HdrIdx, so we will create a new sync when these Indices are satisfied
				if sc.gs.addCreateMessageOnIndices(msg.Indices) { // we have new local events, so we send them back, without creating a new local event
					sc.broadcastEventInfoFromIndices(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), false, senderChan,
						msg.Indices, destID)
				}
				sc.checkBroadcastIndices(sc.MainChannel)
			} else { // this a new msg for a sync, so we create a new local event and send it with the missing dependencies
				sc.broadcastEventInfoFromIndices(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), true, senderChan,
					msg.Indices, destID)
			}
		}
		return true, false
	default:
		panic("unknown msg type")
	}
}

// CanStartNext returns true
func (sc *MvCons4) CanStartNext() bool {
	return sc.gs.canStartNext(sc.Index)
}

// GetNextInfo will be called after CanStartNext returns true.
// prevIdx is the index that this consensus index will follow (i.e. idx - 1).
// proposer and preDecision are nil.
// hasInfo is false since this consensus does not support concurrent instances (it is built in).
func (sc *MvCons4) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil,
		nil, false
}

// NeedsConcurrent returns 1.
func (sc *MvCons4) NeedsCompletionConcurrentProposals() types.ConsensusInt {
	return 1
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

func (sc *MvCons4) checkBroadcastIndices(mainChannel channelinterface.MainChannel) {

	for i := 0; i < sc.gs.checkCreateEvent(); i++ {
		switch sc.MvCons4BcastType {
		case types.Direct:
			sc.doBroadcast(nil, sc.MainChannel)
		case types.Indices:
			sc.doBroadcastIndices(mainChannel)
		default:
			panic("should not reach")
		}
	}
}

func (sc *MvCons4) doBroadcastIndices(mainChannel channelinterface.MainChannel) {
	msg := sc.gs.getMyIndicesMsg(false, sc.ConsItems.MC)
	// bcast to a random member
	receiver := sc.getRandReceiver(false)
	sts := sc.gs.getStats(sc.ConsItems.MC.MC)
	err := mainChannel.SendToPub([]messages.MsgHeader{msg}, receiver, sts.IsRecordIndex(),
		sts)
	if err != nil {
		logging.Error(err)
	}
}

func (sc *MvCons4) getRandReceiver(justMembers bool) sig.Pub {
	myIdx := sc.ConsItems.MC.MC.GetMyPriv().GetPub().GetIndex()
	var pubs []sig.Pub
	if justMembers {
		pubs = sc.ConsItems.MC.MC.GetParticipants()
	} else {
		pubs = sc.ConsItems.MC.MC.GetAllPubs()
	}
	receiver := myIdx
	for receiver == myIdx {
		receiver = sig.PubKeyIndex(sc.rand.Intn(len(pubs)))
	}
	return pubs[receiver]
}

func (sc *MvCons4) broadcastEventInfoFromEvent(prevMsg *messagetypes.EventMessage,
	mainChannel channelinterface.MainChannel) {

	myIdx := sc.ConsItems.MC.MC.GetMyPriv().GetPub().GetIndex()
	receiverPub := sc.getRandReceiver(false)
	receiver := receiverPub.GetIndex()
	var fromIdx sig.PubKeyIndex
	if prevMsg != nil { // the event that caused this new event to be created
		fromIdx = sig.PubKeyIndex(prevMsg.Event.LocalInfo.ID)
		if fromIdx == myIdx {
			return // we don't create events from our own messages
		}
	} else { // this event was created by receiving a proposal locally, so use the receiver as the fromIdx
		fromIdx = sc.getRandReceiver(true).GetIndex()
	}

	createEvent := sc.checkCreateEvent()
	msgs := append(sc.gs.createEventFromID(createEvent, sc.ConsItems.MC, myIdx, fromIdx, receiver),
		sc.gs.getMyIndicesMsg(false, sc.ConsItems.MC))
	sts := sc.gs.getStats(sc.ConsItems.MC.MC)
	err := mainChannel.SendToPub(msgs, receiverPub, sts.IsRecordIndex(), sts)
	if err != nil {
		logging.Error(err)
	}
	if createEvent {
		sc.checkDecidedInternal() // check if the new event made us decide
	}
}

func (sc *MvCons4) checkCreateEvent() bool {
	return sc.CheckMemberLocal() && !sc.finishedLastRound
}

// broadcastEcho broadcasts an echo message
func (sc *MvCons4) broadcastEventInfoFromIndices(
	myPub sig.Pub,
	createNewEvent bool,
	sendChan *channelinterface.SendRecvChannel,
	indices []graph.IndexType,
	destID graph.IndexType) {

	myIdx := myPub.GetIndex()
	receiver := myIdx
	for receiver == myIdx {
		receiver = sig.PubKeyIndex(sc.rand.Intn(sc.ConsItems.MC.MC.GetMemberCount()))
	}
	msgs := sc.gs.createEventFromIndices(createNewEvent && sc.checkCreateEvent(),
		sc.ConsItems.MC, myIdx, receiver, destID, indices)
	if createNewEvent {
		// logging.Info("created event", sc.gs.graph.GetIndices(), indices, myIdx)
		// if we are creating a new event, then we append an index message for when the node can trigger a new sync
		msgs = append(msgs, sc.gs.getMyIndicesMsg(true, sc.ConsItems.MC))
		sc.checkDecidedInternal() // see if the new event made us decide
	} else {
		// otherwise we are just sending missing events, so we don't want to trigger a new sync
	}
	if len(msgs) > 0 {
		sts := sc.gs.getStats(sc.ConsItems.MC.MC)
		if sendChan != nil {
			sndMsg, err := messages.CreateMsg(msgs)
			utils.PanicNonNil(err)
			sendChan.MainChan.SendTo(sndMsg.GetBytes(), sendChan.ReturnChan, sts.IsRecordIndex(),
				sts)
		} else {
			err := sc.MainChannel.SendToPub(msgs, sc.ConsItems.MC.MC.GetAllPubs()[destID],
				sts.IsRecordIndex(), sts)
			if err != nil {
				logging.Error(err)
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*MvCons4) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewMvCons4MessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *MvCons4) Collect() {
	sc.prev = nil
	sc.AbsConsItem.Collect()
	sc.gs.performGC(sc.Index)
}
