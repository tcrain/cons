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

// BVMessage1 is the type of messages used by the binary consensus
// It implements messages.MsgHeader
type BVMessage1 struct {
	Round types.ConsensusRound // The round within the consensus instance
	Stage byte
}

func (bvm *BVMessage1) ShallowCopy() messages.InternalSignedMsgHeader {
	return &BVMessage1{
		Stage: bvm.Stage,
		Round: bvm.Round}
}

// NewBVMessage1 creates a new empty BVMessage1
func NewBVMessage1() *BVMessage1 {
	return &BVMessage1{}
}

// GetSignType retursn types.NormalSignature
func (*BVMessage1) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (bvm *BVMessage1) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: bvm.GetID(), Round: bvm.Round, Stage: bvm.Stage}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (bvm *BVMessage1) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (bvm *BVMessage1) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddConsensusRound(bvm.Round)
	if bvm.Round == 0 {
		panic("0 round invalid")
	}

	signEndOffset = (*messages.MsgBuffer)(m).AddByte(bvm.Stage)
	signEndOffset++
	bytesWritten++

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (bvm *BVMessage1) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	bvm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	if bvm.Round == 0 {
		err = types.ErrInvalidRound
		return
	}

	bvm.Stage, err = (*messages.MsgBuffer)(m).ReadByte()
	if err != nil {
		return
	}
	bytesRead++
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (bvm *BVMessage1) GetID() messages.HeaderID {
	return messages.HdrBV1
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (bvm *BVMessage1) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return bvm
}
