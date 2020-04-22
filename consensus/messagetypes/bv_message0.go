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

func GetBVMessageInfo(header messages.InternalSignedMsgHeader) (binVal types.BinVal,
	round types.ConsensusRound, stage byte) {

	switch h := header.(type) {
	case *BVMessage0:
		return 0, h.Round, h.Stage
	case *BVMessage1:
		return 1, h.Round, h.Stage
	default:
		panic(header)
	}
}

func CreateBVMessage(binVal types.BinVal, round types.ConsensusRound) messages.InternalSignedMsgHeader {
	return CreateBVMessageStage(binVal, round, 0)
}

func CreateBVMessageStage(binVal types.BinVal, round types.ConsensusRound,
	stage byte) messages.InternalSignedMsgHeader {

	var bvMsg messages.InternalSignedMsgHeader
	switch binVal {
	case 0:
		msg := NewBVMessage0()
		msg.Round = round
		msg.Stage = stage
		bvMsg = msg
	case 1:
		msg := NewBVMessage1()
		msg.Round = round
		msg.Stage = stage
		bvMsg = msg
	default:
		panic(binVal)
	}
	return bvMsg
}

// BVMessage0 is the type of messages used by the binary consensus
// It implements messages.MsgHeader
type BVMessage0 struct {
	Round types.ConsensusRound // The round within the consensus instance
	Stage byte                 // The stage within the consensus this message is for
}

func (bvm *BVMessage0) ShallowCopy() messages.InternalSignedMsgHeader {
	return &BVMessage0{
		Stage: bvm.Stage,
		Round: bvm.Round}
}

// NewBVMessage0 creates a new empty BVMessage0
func NewBVMessage0() *BVMessage0 {
	return &BVMessage0{}
}

// GetSignType returns type.NormalSignature
func (*BVMessage0) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (bvm *BVMessage0) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: bvm.GetID(), Round: bvm.Round, Stage: bvm.Stage}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (bvm *BVMessage0) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (bvm *BVMessage0) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	if bvm.Round == 0 {
		panic("should not have 0 round")
	}
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddConsensusRound(bvm.Round)

	signEndOffset = (*messages.MsgBuffer)(m).AddByte(bvm.Stage)
	bytesWritten++
	signEndOffset++

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (bvm *BVMessage0) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	bvm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	if bvm.Round == 0 {
		err = types.ErrInvalidRound
		return
	}
	//scm.Stage
	bvm.Stage, err = (*messages.MsgBuffer)(m).ReadByte()
	if err != nil {
		return
	}
	bytesRead++
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (bvm *BVMessage0) GetID() messages.HeaderID {
	return messages.HdrBV0
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (bvm *BVMessage0) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return bvm
}
