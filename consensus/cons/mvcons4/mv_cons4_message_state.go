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

package mvcons4

import (
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// MvCons4Message state stores the message of MvCons
type MessageState struct {
	index types.ConsensusIndex
	gs    *globalMessageState
	gc    *generalconfig.GeneralConfig
}

// NewMvCons4MessageState generates a new MvCons4MessageStateObject.
func NewMvCons4MessageState(gc *generalconfig.GeneralConfig) *MessageState {
	ret := &MessageState{
		gc: gc,
	}
	return ret
}

// New inits and returns a new MvCons4MessageState object for the given consensus index.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	ret := &MessageState{
		index: idx,
		gs:    sms.gs,
		gc:    sms.gc,
	}
	return ret
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The valid message types are
// (1) HdrEvent - a message containing an event
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *deserialized.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*deserialized.DeserializedItem, error) {

	_, _ = hdrFunc, gc
	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}

	// all messages are unique and must be signed by only a single node
	if err := sig.CheckSingleSupporter(deser.Header); err != nil {
		return nil, err
	}

	switch deser.HeaderType {
	case messages.HdrEventInfo:
		switch v := deser.Header.(type) {
		case *sig.MultipleSignedMessage: // check if it is a signed message
			// check we dont have too many remote dependencies (at most n-1)
			if len(v.InternalSignedMsgHeader.(*messagetypes.EventMessage).Event.RemoteAncestors) >= mc.MC.GetMemberCount() {
				return nil, types.ErrInvalidRemoteAncestorCount
			}

			// check it is a valid msg in the graph
			if err := sms.gs.checkMsg(v, mc); err != nil {
				return nil, err
			}
			// we should only have a single signature
			sigs := v.GetSigItems()
			if len(sigs) != 1 {
				return nil, types.ErrInvalidSigCount
			}
			// check the signature and member
			if err := consinterface.CheckMember(mc, deser.Index, sigs[0], v, sms.gc); err != nil {
				return nil, err
			}
			deser.NewMsgIDSigCount, deser.NewTotalSigCount = 1, 1
			// the message is valid, store it in the message state
			if err := sms.gs.storeMsg(v, false); err != nil {
				return nil, err
			}
			return []*deserialized.DeserializedItem{deser}, nil
		default:
			panic("only signed messages supported")
		}
	case messages.HdrIdx:
		if sms.gc.MvCons4BcastType == types.Normal {
			panic("should not reach")
		}
		w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.IndexMessage)
		if len(w.Indices) > mc.MC.GetMemberCount() {
			return nil, types.ErrInvalidPubIndex
		}
		return []*deserialized.DeserializedItem{deser}, nil
	default:
		panic(fmt.Sprint("invalid msg type ", deser.HeaderType))
	}
}

// GetMsgState returns the serialized state of all the valid messages received
// since the previous index decided until either this index decided or all messages
// following the previous index if this index has not yet decided.
func (sms *MessageState) GetMsgState(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc,
	mc *consinterface.MemCheckers) ([]byte, error) {

	return sms.getMsgState(priv, localOnly, false, bufferCountFunc, mc)
}

func (sms *MessageState) getMsgState(priv sig.Priv, localOnly, fromNoProgress bool,
	bufferCountFunc consinterface.BufferCountFunc,
	mc *consinterface.MemCheckers) ([]byte, error) {

	if localOnly {
		panic("unsupported")
	}
	_ = priv
	_, _ = bufferCountFunc, mc
	msgs := sms.gs.getConsensusIndexMessages(sms.index)
	if fromNoProgress && sms.gc.MvCons4BcastType != types.Normal { // if it a no progress message then we include an indices messages to trigger a sync
		msgs = append(msgs, sms.gs.getMyIndicesMsg(false, mc))
	}
	ret, err := messages.SerializeHeaders(msgs)
	if err != nil {
		return nil, err
	}
	return ret.GetBytes(), nil
}

// GetIndex returns the consensus index.
func (sms *MessageState) GetIndex() types.ConsensusIndex {
	return sms.index
}

// SetupSigned message takes a messages.InternalSignedMsgHeader, serializes the message appending signatures
// creating and returning a new *sig.MultipleSignedMessage.
// If generateMySig is true, the serialized message will include the local nodes signature.
// Up to addOtherSigs number of signatures received so far for this message will additionally be appended.
func (sms *MessageState) SetupSignedMessage(hdr messages.InternalSignedMsgHeader, generateMySig bool,
	addOthersSigsCount int, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error) {

	_, _, _ = generateMySig, addOthersSigsCount, mc
	if msg := sms.gs.getMsg(types.HashStr(hdr.GetMsgID().ToBytes(sms.index))); msg != nil {
		return msg, nil
	}
	return nil, types.ErrMsgNotFound
}

// SetupUnsignedMessage is unsupported
func (sms *MessageState) SetupUnsignedMessage(hdr messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (*sig.UnsignedMessage, error) {

	_, _ = hdr, mc
	panic("unsupported")
}

// SetupSignedMessageDuplicates is unsupported
func (sms *MessageState) SetupSignedMessagesDuplicates(combined *messagetypes.CombinedMessage, hdrs []messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (combinedSigned *sig.MultipleSignedMessage, partialsSigned []*sig.MultipleSignedMessage, err error) {

	_, _, _ = combined, hdrs, mc
	panic("unsupported")
}

// GetSigCountMsg returns the number of signatures received for this message.
func (sms *MessageState) GetSigCountMsg(hsh types.HashStr) int {
	if sms.gs.getMsg(hsh) != nil {
		return 1
	}
	return 0
}

// GetSigCountMsgHeader returns the number of signatures received for this header.
func (sms *MessageState) GetSigCountMsgHeader(header messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (int, error) {

	_ = mc
	if v, ok := header.(*messagetypes.EventMessage); ok {
		return sms.GetSigCountMsg(types.HashStr(v.GetEventInfoHash())), nil
	}
	return 0, types.ErrMsgNotFound
}

// GetSigCountMsgID returns the number of sigs for this message's MsgID (see messages.MsgID).
func (sms *MessageState) GetSigCountMsgID(msgID messages.MsgID) int {
	return sms.GetSigCountMsg(types.HashStr(msgID.ToBytes(sms.index)))
}

// GetSigCountMsgIDList returns list of received messages that have msgID for their MsgID and how many
// signatures have been received for each.
func (sms *MessageState) GetSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {

	if msg := sms.gs.getMsg(types.HashStr(msgID.ToBytes(sms.index))); msg != nil {
		return []consinterface.MsgIDCount{{MsgHeader: msg, Count: 1}}
	}
	return nil
}

// GetThreshSig is no supported.
func (sms *MessageState) GetThreshSig(hdr messages.InternalSignedMsgHeader, threshold int,
	mc *consinterface.MemCheckers) (*sig.SigItem, error) {

	_, _, _ = hdr, threshold, mc
	panic("unsupported")
}

// GetCoinVal is unsupported.
func (sms *MessageState) GetCoinVal(hdr messages.InternalSignedMsgHeader, threshold int,
	mc *consinterface.MemCheckers) (coinVal types.BinVal, ready bool, err error) {

	_, _, _ = hdr, threshold, mc
	panic("unsupported")
}
