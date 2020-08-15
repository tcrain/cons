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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// MvRequestRecoverMessage is used when multi-value consensus has completed, but only
// a hash was received, so the full proposal must be requested.
// It implements messages.MsgHeader
type MvRequestRecoverMessage struct {
	Index        types.ConsensusIndex
	ProposalHash types.HashBytes // The hash of the proposal to recover
}

// NewMvRequestRecoverMessage creates a new mv request recover message
func NewMvRequestRecoverMessage() *MvRequestRecoverMessage {
	return &MvRequestRecoverMessage{}
}

// GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
func (mrm *MvRequestRecoverMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(mrm.GetID())
}

// GetIndex returns the consensus index for this message.
func (mrm *MvRequestRecoverMessage) GetIndex() types.ConsensusIndex {
	return mrm.Index
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mrm *MvRequestRecoverMessage) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(mrm.Index.FirstIndex, mrm.Index.AdditionalIndices, mrm.GetID(), 0, (*messages.MsgBuffer)(m))

	// Now the proposal hash
	if len(mrm.ProposalHash) != types.GetHashLen() {
		return 0, types.ErrInvalidHash
	}
	l1, _ := (*messages.MsgBuffer)(m).AddBytes(mrm.ProposalHash)
	if l1 != len(mrm.ProposalHash) {
		panic("error writing proposal hash")
	}
	l += l1

	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

// GetBytes returns the bytes that make up the message
func (mrm *MvRequestRecoverMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// PeekHeader returns the indices related to the messages without impacting m.
func (*MvRequestRecoverMessage) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mrm *MvRequestRecoverMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	l, index, addIdx, size, _, err := messages.ReadHeaderHead(mrm.GetID(), unmarFunc.ConsensusIDUnMarshaler, (*messages.MsgBuffer)(m))
	if err != nil {
		return l, err
	}
	if mrm.Index, err = unmarFunc.ComputeConsensusID(index, addIdx); err != nil {
		return l, err
	}

	// Get the proposal hash
	hlen := types.GetHashLen()
	proposalHash, err := (*messages.MsgBuffer)(m).ReadBytes(hlen)
	if err != nil {
		return 0, err
	}
	mrm.ProposalHash = proposalHash
	l += hlen

	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for this header
func (mrm *MvRequestRecoverMessage) GetID() messages.HeaderID {
	return messages.HdrMvRequestRecover
}

// /////////////////////////////////////////////////////////////////////////////////////////////////
// //
// /////////////////////////////////////////////////////////////////////////////////////////////////

// type MvRecoverMessageTimeout uint32

// func (tm MvRecoverMessageTimeout) GetMsgID() messages.MsgID {
// 	panic("unused")
// }

// func (tm MvRecoverMessageTimeout) Serialize(m *messages.Message) (int, error) {
// 	return 0, nil
// }

// func (tm MvRecoverMessageTimeout) CreateCopy() messages.MsgHeader {
// 	return tm
// }

// func (tm MvRecoverMessageTimeout) GetBytes(m *messages.Message) ([]byte, error) {
// 	return nil, nil
// }

// func (tm MvRecoverMessageTimeout) Deserialize(m *messages.Message) (int, error) {
// 	return 0, nil
// }

// func (tm MvRecoverMessageTimeout) GetID() messages.HeaderID {
// 	return messages.HdrMvRecoverTimeout
// }
