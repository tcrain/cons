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
// It is used to differentiate between CoinMessages of different rounds by comparing them, for example, during
// consensus we can count n-t different CoinMessages of each round (even if the CoinMessages have different
// binary values, the BinMsgID should be equal for the same round).
type CoinMsgID struct {
	HdrID messages.HeaderID
	Round types.ConsensusRound
}

// IsMsgID to satisfy the interface and returns true
func (CoinMsgID) IsMsgID() bool {
	return true
}

func (bm CoinMsgID) ToBytes(index types.ConsensusIndex) []byte {
	m := messages.NewMsgBuffer()
	m.AddConsensusID(index.Index)
	m.AddHeaderID(bm.HdrID)
	m.AddConsensusRound(bm.Round)
	return m.GetRemainingBytes()
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// CoinMessage is used by random binary consensus.
// Once we receive n-t of these for the threshold signature, the coin value of the round is revealed.
// To ensure that coin is unpredictable, be sure to change config.CsID each time consensus is run
// (this is included in all signed messages).
// It implements messages.MsgHeader
type CoinMessage struct {
	Round    types.ConsensusRound // The round within the consensus instance
	signType *types.SignType      // make it a pointer so we are sure we assign this or else crash
}

func (cm *CoinMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &CoinMessage{
		Round: cm.Round}
}

// NewCoinMessage creates a new empty CoinMessage
func NewCoinMessage() *CoinMessage {
	return &CoinMessage{}
}

// GetSignType returns the SignType that was used for this message during allocation.
func (cm *CoinMessage) GetSignType() types.SignType {
	return types.CoinProof
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (cm *CoinMessage) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: cm.GetID(), Round: cm.Round}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (cm *CoinMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cm *CoinMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(cm.Round)
	signEndOffset += bytesWritten

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cm *CoinMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	cm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (cm *CoinMessage) GetID() messages.HeaderID {
	return messages.HdrCoin
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (cm *CoinMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return cm
}
