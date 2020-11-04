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

package cons

import (
	"fmt"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"strconv"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

// GetConsType returns the type of consensus this instance implements.
func (sc *SimpleCons) GetConsType() types.ConsType {
	return types.SimpleConsType
}

// SimpleCons is just for testing the consensus functionality.
// Each round each node sends a single SimpleConsMessage containing its public key,
// the instance is complete when a node has received the SimpleConsMessage from all nodes.
type SimpleCons struct {
	AbsConsItem
	id          messagetypes.ScMsgID   // the id of this node (its public key)
	n           uint32                 // the total number of nodes
	decided     bool                   // true when messages from all n nodes has been received
	idset       map[sig.PubKeyStr]bool // keeps track of the pub keys of the messages received from different nodes so far
	gotProposal bool                   // used for sanity checks
	// priv        sig.Priv               // this nodes private key
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *SimpleCons) GetBinState(localOnly bool) ([]byte, error) {
	msg, err := messages.CreateMsg(sc.PreHeaders)
	if err != nil {
		panic(err)
	}
	mm, err := sc.ConsItems.MsgState.GetMsgState(sc.ConsItems.MC.MC.GetMyPriv(), localOnly,
		sc.GetBufferCount, sc.ConsItems.MC)
	if err != nil {
		return nil, err
	}
	_, err = messages.AppendHeader(msg, (messagetypes.ConsBinStateMessage)(mm))
	if err != nil {
		return nil, err
	}
	return msg.GetBytes(), nil
}

// GetProposeHeaderID returns the HeaderID messages.HdrPropose that will be input to GotProposal.
func (sc *SimpleCons) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrPropose
}

// NeedsConcurrent returns 1.
func (sc *SimpleCons) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// HasReceivedProposal panics because SimpleCons has no proposals.
func (sc *SimpleCons) HasValidStarted() bool {
	panic("unused")
}

// GenerateNewItem creates a new simple cons item.
func (*SimpleCons) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &SimpleCons{AbsConsItem: newAbsItem}

	is := gc.Eis.(ConsInitState)
	newItem.id = messagetypes.ScMsgID(strconv.Itoa(int(is.Id)))
	newItem.n = is.N
	newItem.idset = make(map[sig.PubKeyStr]bool)
	newItem.gotProposal = false
	newItem.decided = false

	return newItem
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *SimpleCons) ShouldCreatePartial(messages.HeaderID) bool {
	return false
}

// GetCommitProof returns nil as SimpleCons has no proofs.
func (sc *SimpleCons) GetCommitProof() []messages.MsgHeader {
	return nil
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *SimpleCons) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// Start allows GetProposalIndex to return true.
func (sc *SimpleCons) Start(finishedLastRound bool) {
	_ = finishedLastRound
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *SimpleCons) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, canStartNext bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil,
		nil, sc.GeneralConfig.AllowConcurrent > 0
}

// SetInitialState does noting for this algorithm.
func (sc *SimpleCons) SetInitialState([]byte) {}

// GotProposal takes the proposal, creates a SimpleConsMessage and broadcasts it.
func (sc *SimpleCons) GotProposal(_ messages.MsgHeader, mainChannel channelinterface.MainChannel) error {

	// sanity checks
	if sc.gotProposal {
		panic(fmt.Sprint("got multiple proposals for same cons", sc.Index))
	}
	if !sc.ConsItems.MC.MC.IsReady() {
		panic("should be ready")
	}
	sc.gotProposal = true

	// Check if we are a member of this consensus
	if consinterface.CheckMemberLocal(sc.ConsItems.MC) {
		// setup a SimpleConsMessage to be broadcast
		w := messagetypes.NewSimpleConsMessage(sc.ConsItems.MC.MC.GetMyPriv().GetPub())
		w.MyPub = sc.ConsItems.MC.MC.GetMyPriv().GetPub()
		if sc.CheckMemberLocalMsg(w) { // check if we are a member for this message type

			sc.Broadcast(nil, w, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
				mainChannel, nil)
		}
	}
	return nil
}

/*func (sc *SimpleCons) Broadcast(nxtCoordPub sig.Pub, auxMsg messages.InternalSignedMsgHeader,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel, additionalMsgs ...messages.MsgHeader) {

	DoConsBroadcast(nxtCoordPub, auxMsg, !sc.GeneralConfig.NoSignatures, additionalMsgs, forwardFunc,
		sc.ConsItems, mainChannel, sc.GeneralConfig)
}
*/
// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// It tracks from which nodes we have received the SimpleConsMessage so far, and once n are recieved, the "consensus" is finished.
func (sc *SimpleCons) ProcessMessage(
	deser *deserialized.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (bool, bool) {

	_, _ = isLocal, senderChan
	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic(fmt.Sprintf("got wrong idx, %v, expected %v", sc.Index, deser.Index))
	}
	if deser.HeaderType == messages.HdrSimpleCons {
		w := deser.Header.(*sig.MultipleSignedMessage)

		// A simple cons message should only be signed once
		if len(w.SigItems) > 1 || len(w.SigItems) <= 0 {
			return false, false
		}

		// Check that the pub key in the message corresponds to the pubkey that signed the message.
		// Just for testing, we dont actually use the pub key in the message, we only use the one with the signature.
		p1, err := w.SigItems[0].Pub.GetPubID()
		if err != nil {
			return false, false
		}
		p2, err := w.InternalSignedMsgHeader.GetBaseMsgHeader().(*messagetypes.SimpleConsMessage).MyPub.GetPubID()
		if err != nil {
			return false, false
		}
		if p1 != p2 {
			logging.Infof("Node %v tried to send pub key of %v in simple cons message", p1, p2)
			return false, false
		}

		// get the pub key of the signer of the message
		pubBytes, err := w.SigItems[0].Pub.GetPubString()
		if err != nil {
			return false, false
		}

		if _, ok := sc.idset[pubBytes]; ok {
			logging.Info("Got a repeat message")
			return false, false
		}

		// store it in the map
		sc.idset[pubBytes] = true
		l := len(sc.idset)
		count := sc.ConsItems.MC.MC.GetMemberCount()
		// if we got all n, then we have decided
		if l == count {
			if !sc.decided {
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(0, false)
			}
			sc.decided = true
		} else if l > count {
			panic("Got too many values")
		}
		return true, true
	}
	panic("got invalid message header")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrSimpleCons, which returns 1, 1 for the thresholds, meaning to forward it as soon as you see it, since each message is unique from each process.
func (sc *SimpleCons) GetBufferCount(hdr messages.MsgIDHeader,
	_ *generalconfig.GeneralConfig, _ *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrSimpleCons:
		return 1, 1, hdr.GetMsgID(), nil
	default:
		return 0, 0, nil, types.ErrInvalidHeader
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrSimpleCons are valid headerIDs.
func (SimpleCons) GetHeader(pub sig.Pub, _ *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrSimpleCons:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, pub, messagetypes.NewSimpleConsMessage(pub)), nil
	default:
		return nil, types.ErrInvalidHeader
	}
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *SimpleCons) CanStartNext() bool {
	return true
}

// GetDecision returns the decided value which is []byte(fmt.Sprintf("simpleCons%v", sc.Index)).
func (sc *SimpleCons) GetDecision() (sig.Pub, []byte, types.ConsensusIndex, types.ConsensusIndex) {
	if !sc.decided {
		panic("should have decided")
	}
	return nil, []byte(fmt.Sprintf("simpleCons%v", sc.Index.Index)),
		types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1),
		types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) + 1)
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *SimpleCons) HasDecided() bool {
	return sc.decided
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *SimpleCons) SetNextConsItem(consinterface.ConsItem) {
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *SimpleCons) PrevHasBeenReset() {
}

// AllowsOutOfOrderProposals returns false.
func (*SimpleCons) AllowsOutOfOrderProposals() bool {
	return false
}

// GenerateMessageState generates a new message state object given the inputs.
func (*SimpleCons) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return messagestate.NewSimpleMessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *SimpleCons) Collect() {
}
