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

// CoinPreMessage is used by random binary consensus.
// It is sent before the coin to change a t+1 threshold coin to an n-t one
// It implements messages.MsgHeader
type CoinPreMessage struct {
	Round types.ConsensusRound // The round within the consensus instance
}

func (cm *CoinPreMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &CoinPreMessage{
		Round: cm.Round}
}

// NewCoinPreMessage creates a new empty CoinPreMessage
func NewCoinPreMessage() *CoinPreMessage {
	return &CoinPreMessage{}
}

// GetSignType returns the SignType that was used for this message during allocation.
func (cm *CoinPreMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (cm *CoinPreMessage) GetMsgID() messages.MsgID {
	return BinMsgID{HdrID: cm.GetID(), Round: cm.Round}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (cm *CoinPreMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cm *CoinPreMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(cm.Round)
	signEndOffset += bytesWritten

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cm *CoinPreMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	cm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (cm *CoinPreMessage) GetID() messages.HeaderID {
	return messages.HdrCoinPre
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (cm *CoinPreMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return cm
}
