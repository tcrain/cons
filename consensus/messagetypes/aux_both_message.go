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

// AuxBothMessage is the type of messages used by the binary consensus
// It implements messages.MsgHeader
type AuxBothMessage struct {
	Round  types.ConsensusRound // The round within the consensus instance
	BinVal types.BinVal         // The supported binary value
}

func (apm *AuxBothMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &AuxBothMessage{
		Round:  apm.Round,
		BinVal: apm.BinVal}
}

// NewAuxBothMessage creates a new empty AuxBothMessage
func NewAuxBothMessage() *AuxBothMessage {
	return &AuxBothMessage{}
}

// GetSignType returns types.NormalSignature
func (*AuxBothMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (apm *AuxBothMessage) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: apm.GetID(), Round: apm.Round}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (apm *AuxBothMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,

	proposal []byte, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (apm *AuxBothMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddConsensusRound(apm.Round)
	if apm.Round == 0 {
		panic("round 0 invalid")
	}

	// Now the bin value
	signEndOffset = (*messages.MsgBuffer)(m).AddBin(apm.BinVal, true)
	bytesWritten++
	signEndOffset++

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (apm *AuxBothMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	apm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	if apm.Round == 0 {
		err = types.ErrInvalidRound
		return
	}

	// Get the bin val
	apm.BinVal, err = (*messages.MsgBuffer)(m).ReadBin(true)
	if err != nil {
		return
	}

	bytesRead++
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (apm *AuxBothMessage) GetID() messages.HeaderID {
	return messages.HdrAuxBoth
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (apm *AuxBothMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return apm
}
