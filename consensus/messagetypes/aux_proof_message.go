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

// BinMsgID implements messages.MsgID.
// It is used to differentiate between AuxProofMessages of different rounds by comparing them, for example, during
// consensus we can count n-t different AuxProofMessages of each round (even if the AuxProofMessages have different
// binary values, the BinMsgID should be equal for the same round).
type BinMsgID struct {
	HdrID messages.HeaderID
	Round types.ConsensusRound
	Stage byte
}

// IsMsgID to satisfy the interface and returns true
func (BinMsgID) IsMsgID() bool {
	return true
}

// ToMsgIDInfo converts the MsgID to a MsgIDInfo
func (bm BinMsgID) ToMsgIDInfo() messages.MsgIDInfo {
	return messages.MsgIDInfo{
		HeaderID: uint32(bm.HdrID),
		Round:    uint32(bm.Round),
		Extra:    bm.Stage,
	}
}

func (bm BinMsgID) ToBytes(index types.ConsensusIndex) []byte {
	m := messages.NewMsgBuffer()
	m.AddConsensusID(index.Index)
	m.AddHeaderID(bm.HdrID)
	m.AddConsensusRound(bm.Round)
	m.AddByte(bm.Stage)
	return m.GetRemainingBytes()
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// AuxProofMessage is the type of messages used by the binary consensus
// It implements messages.MsgHeader
type AuxProofMessage struct {
	Round     types.ConsensusRound // The round within the consensus instance
	BinVal    types.BinVal         // The supported binary value
	AllowCoin bool                 // Allows BinVal to take the value 0, 1, or Coin
}

// GetSignType returns types.Secondary signature for round 1, types.NormalSignature otherwise.
func (apm *AuxProofMessage) GetSignType() types.SignType {
	if apm.Round == 0 {
		return types.SecondarySignature
	}
	return types.NormalSignature
}

func (apm *AuxProofMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &AuxProofMessage{
		AllowCoin: apm.AllowCoin,
		Round:     apm.Round,
		BinVal:    apm.BinVal}
}

// NewAuxProofMessage creates a new empty AuxProofMessage
func NewAuxProofMessage(allowNotCoin bool) *AuxProofMessage {
	return &AuxProofMessage{AllowCoin: allowNotCoin}
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (apm *AuxProofMessage) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: apm.GetID(), Round: apm.Round}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (apm *AuxProofMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (apm *AuxProofMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddConsensusRound(apm.Round)

	// Now the bin value
	signEndOffset = (*messages.MsgBuffer)(m).AddBin(apm.BinVal, apm.AllowCoin)
	bytesWritten++
	signEndOffset++

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (apm *AuxProofMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	apm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}

	// Get the bin val
	apm.BinVal, err = (*messages.MsgBuffer)(m).ReadBin(apm.AllowCoin)
	if err != nil {
		return
	}
	// if scm.Round <= 1 && scm.BinVal == types.Coin {
	// Coin values only valid from round 2 onward
	//	err = types.ErrInvalidRound
	//	return
	//}

	bytesRead++
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (apm *AuxProofMessage) GetID() messages.HeaderID {
	return messages.HdrAuxProof
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (apm *AuxProofMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return apm
}

/////////////////////////////////////////////////////////////////////////////////////////////////
// Timeout message
/////////////////////////////////////////////////////////////////////////////////////////////////

// AuxProofMessageTimeout is sent internally to indicate that timeout has passed
// for a round while waiting for AuxProofMessage
type AuxProofMessageTimeout types.ConsensusRound

// GetMsgID is unused since it is a local message.
func (tm AuxProofMessageTimeout) GetMsgID() messages.MsgID {
	panic("unused")
}

// PeekHeaders is unused since it is a local message.
func (AuxProofMessageTimeout) PeekHeaders(*messages.Message,
	types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	panic("unused")
}

// Serialize is unused since it is a local message.
func (tm AuxProofMessageTimeout) Serialize(*messages.Message) (int, error) {
	panic("unused")
}

// GetBytes is unused since it is a local message.
func (tm AuxProofMessageTimeout) GetBytes(*messages.Message) ([]byte, error) {
	panic("unused")
}

// Deserialize is unused since it is a local message.
func (tm AuxProofMessageTimeout) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	panic("unused")
}

// GetID returns the header id for this header.
func (tm AuxProofMessageTimeout) GetID() messages.HeaderID {
	return messages.HdrAuxProofTimeout
}
