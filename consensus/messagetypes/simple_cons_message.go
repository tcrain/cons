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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// ScMsgID implements messages.MsgID.
// It is used to differentiate between multi-value consensus messages of different types by comparing them.
type ScMsgID sig.PubKeyID

// IsMsgID to satisfy the interface and returns true
func (ScMsgID) IsMsgID() bool {
	return true
}

func (sc ScMsgID) ToBytes(index types.ConsensusIndex) []byte {
	m := messages.NewMsgBuffer()
	m.AddConsensusID(index.Index)
	m.AddBytes([]byte(sc))
	return m.GetRemainingBytes()
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// SimpleConsMessage is the message type used for the simple cons test.
// The message itself includes the public key (for fun).
// It implements messages.MsgHeader
type SimpleConsMessage struct {
	MyPub sig.Pub // The local nodes public key.
}

// NewSimpleConsMessage creates a new simple cons message
func NewSimpleConsMessage(pub sig.Pub) *SimpleConsMessage {
	return &SimpleConsMessage{pub}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (scm *SimpleConsMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	return
}

// GetSignType returns types.NormalSignature
func (*SimpleConsMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

func (scm *SimpleConsMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &SimpleConsMessage{MyPub: scm.MyPub}
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (scm *SimpleConsMessage) GetMsgID() messages.MsgID {
	id, err := scm.MyPub.GetPubID()
	if err != nil {
		panic(err)
	}
	return ScMsgID(id)
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (scm *SimpleConsMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddUint32(uint32(scm.GetID()))

	var l1 int
	l1, err = scm.MyPub.Serialize(m)
	bytesWritten += l1
	if err != nil {
		return
	}
	signEndOffset = (*messages.MsgBuffer)(m).GetWriteOffset()

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (scm *SimpleConsMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Main message
	var id messages.HeaderID
	id, bytesRead, err = (*messages.MsgBuffer)(m).ReadHeaderID()
	if err != nil {
		return
	}
	if id != messages.HdrSimpleCons {
		err = types.ErrWrongMessageType
		return
	}

	var l1 int
	scm.MyPub = scm.MyPub.New()
	l1, err = scm.MyPub.Deserialize(m, types.NilIndexFuns)
	if err != nil {
		return
	}
	bytesRead += l1

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetID returns the header id for this header
func (scm *SimpleConsMessage) GetID() messages.HeaderID {
	return messages.HdrSimpleCons
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (scm *SimpleConsMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return scm
}
