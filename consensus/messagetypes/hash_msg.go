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

// HashMessage is used during multi-value consensus to echo the MvInitMessage by all participants.
// It implements messages.MsgHeader
type HashMessage struct {
	TheHash types.HashBytes // The hash of the proposal being echo'd
}

// NewHashMessage creates a new mv echo message
func NewHashMessage() *HashMessage {
	return &HashMessage{}
}

func (hm *HashMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &HashMessage{TheHash: hm.TheHash}
}

// GetSignType returns types.NormalSignature
func (*HashMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (hm *HashMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	return
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (hm *HashMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(hm.GetID())
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (hm *HashMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the proposal hash
	if len(hm.TheHash) != types.GetHashLen() {
		err = messages.ErrInvalidHash
		return
	}
	v, signEndOffset := (*messages.MsgBuffer)(m).AddBytes(hm.TheHash)
	if v != len(hm.TheHash) {
		panic("error writing hash")
	}
	bytesWritten += v
	signEndOffset += v

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (hm *HashMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal hash
	bytesRead = types.GetHashLen()
	hm.TheHash, err = (*messages.MsgBuffer)(m).ReadBytes(bytesRead)
	if err != nil {
		return
	}
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (hm *HashMessage) GetID() messages.HeaderID {
	return messages.HdrHash
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (hm *HashMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return hm
}
