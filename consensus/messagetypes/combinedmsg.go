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

package messagetypes

import (
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"sort"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
)

// CombinedMsg is a full message constructed from partial messages
// It implements messages.InternalSignedMsgHeader
type CombinedMessage struct {
	messages.InternalSignedMsgHeader
	Round            types.ConsensusRound
	FullMsgHash      types.HashBytes
	PartialMsgHashes []types.HashBytes
}

// NewCombinedMessage returns a new CombinedMessage for the given header and partial message.
func NewCombinedMessageFromPartial(partial *PartialMessage, fullMsg messages.InternalSignedMsgHeader) *CombinedMessage {
	return &CombinedMessage{InternalSignedMsgHeader: fullMsg,
		Round:            partial.Round,
		FullMsgHash:      partial.FullMsgHash,
		PartialMsgHashes: partial.PartialMsgHashes}
}

// NewCombinedMessage returns a new CombinedMessage for the given internal message header type.
func NewCombinedMessage(fullMsg messages.InternalSignedMsgHeader) *CombinedMessage {
	return &CombinedMessage{InternalSignedMsgHeader: fullMsg}
}

// GenerateCombinedMessage creates the bytes of the full message from a list of partial messages.
// It assumes the partial messages have already been verified and have the same meta-data.
func GenerateCombinedMessageBytes(pms []*PartialMessage) (*messages.Message, error) {
	var size int
	for _, nxt := range pms {
		if nxt == nil {
			return nil, types.ErrNotEnoughPartials
		}
	}
	if len(pms) != len(pms[0].PartialMsgHashes) {
		return nil, types.ErrIncorrectNumPartials
	}

	sort.Slice(pms, func(i, j int) bool {
		return pms[i].PartialMsgIndex < pms[j].PartialMsgIndex
	})

	for i, nxt := range pms {
		if i != nxt.PartialMsgIndex {
			return nil, types.ErrInvalidPartialIndecies
		}
		size += len(nxt.PartialMsg)
	}

	full := make([]byte, 0, size)
	for _, nxt := range pms {
		full = append(full, nxt.PartialMsg...)
	}
	return messages.NewMessage(full), nil
}

// GenerateCombinedMessage takes a PartialMessage that was one of the inputs to GenerateCombinedMessageBytes, the output of that function,
// and a function that will return a message header from the headerID. It outputs the CombinedMessage created from this.
func GenerateCombinedMessage(pm *PartialMessage, fullMsg *messages.Message,
	hdrFunc func(sig.Pub, *generalconfig.GeneralConfig, messages.HeaderID) (messages.MsgHeader, error),
	gc *generalconfig.GeneralConfig, pub sig.Pub) (*CombinedMessage, error) {

	ht, err := fullMsg.PeekHeaderType()
	if err != nil {
		return nil, err
	}

	hdr, err := hdrFunc(pub, gc, ht)
	if err != nil {
		return nil, err
	}

	// must be a signed message type
	sm, ok := hdr.(*sig.MultipleSignedMessage)
	if !ok {
		return nil, types.ErrInvalidHeader
	}
	fullMsgHdr := sm.GetBaseMsgHeader()

	_, _, _, _, _, err = messages.ReadHeaderHead(ht, nil, (*messages.MsgBuffer)(fullMsg))
	if err != nil {
		return nil, err
	}
	_, _, err = fullMsgHdr.DeserializeInternal(fullMsg)
	if err != nil {
		return nil, err
	}

	return &CombinedMessage{
		FullMsgHash:             pm.FullMsgHash,
		PartialMsgHashes:        pm.PartialMsgHashes,
		Round:                   pm.Round,
		InternalSignedMsgHeader: fullMsgHdr}, nil
}

// ShallowCopy returns a shallow copy of the message
func (mvi *CombinedMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &CombinedMessage{
		FullMsgHash:             mvi.FullMsgHash,
		PartialMsgHashes:        mvi.PartialMsgHashes,
		Round:                   mvi.Round,
		InternalSignedMsgHeader: mvi.InternalSignedMsgHeader.ShallowCopy()}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mvi *CombinedMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	bytesWritten, signEndOffset = headerPartialSerialze(m, mvi.FullMsgHash, mvi.Round, mvi.PartialMsgHashes)
	logging.Errorf("combine full hashes", bytesWritten, signEndOffset, mvi.FullMsgHash, mvi.PartialMsgHashes)

	// serialize the InternalSignedMsgHeader
	var l1, sizeOffset int
	l1, _, _, sizeOffset, err = messages.SerializeInternalSignedHeader(types.ConsensusIndex{}, mvi.InternalSignedMsgHeader, m)
	bytesWritten += l1
	if err != nil {
		return
	}
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l1))

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mvi *CombinedMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	mvi.FullMsgHash, mvi.Round, mvi.PartialMsgHashes, bytesRead, signEndOffset, err = headerPartialDeserialize(m)
	if err != nil {
		return
	}

	// deserialize the InternalSignedMsgHeader
	var l1 int
	_, l1, _, _, _, err = messages.DeserialzeInternalSignedHeader(mvi.InternalSignedMsgHeader, types.NilIndexFuns, m)
	bytesRead += l1
	return
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mvi *CombinedMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mvi.InternalSignedMsgHeader.GetBaseMsgHeader()
}

// // GetID returns the header id for this header
// func (scm *CombinedMessage) GetID() messages.HeaderID {
// 	return scm.InternalSignedMsgHeader.GetID()
// }

// // GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
// func (mvi *CombinedMessage) GetMsgID() messages.MsgID {
// 	return mvi.InternalSignedMsgHeader.GetMsgID()
// }
