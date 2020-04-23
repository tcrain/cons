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

package sig

import (
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// UnsignedMessage represents a message with a list of signatures for that message
type UnsignedMessage struct {
	messages.InternalSignedMsgHeader                      // the specific message
	Index                            types.ConsensusIndex // the cons index of the message
	newPubFunc                       func() Pub           // The local public key
	Hash                             types.HashBytes      // The hash of the signed part of the message as bytes
	hashString                       types.HashStr        // The hash of the signed part of the message as a string
	Msg                              []byte               // The signed part of the message
	EncryptPubs                      []Pub                // The nodes who sent this message
}

// NewMultipleSignedMsg generates a new multiple signed msg object given the input, it does not compute the hash fields.
func NewUnsignedMessage(index types.ConsensusIndex, pub Pub, internalMsg messages.InternalSignedMsgHeader) *UnsignedMessage {
	return &UnsignedMessage{
		newPubFunc:              pub.New,
		Index:                   index,
		InternalSignedMsgHeader: internalMsg}
}

// ShallowCopy generates a new multiple signed msg object that is a shallow copy of the original.
func (sm *UnsignedMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &UnsignedMessage{
		InternalSignedMsgHeader: sm.InternalSignedMsgHeader.ShallowCopy(),
		Index:                   sm.Index.ShallowCopy(),
		newPubFunc:              sm.newPubFunc,
		Hash:                    sm.Hash,
		hashString:              sm.hashString,
		Msg:                     sm.Msg,
		EncryptPubs:             sm.EncryptPubs}
}

// GetIndex returns the consensus index for this message.
func (sm *UnsignedMessage) GetIndex() types.ConsensusIndex {
	return sm.Index
}

// GetSignedMessage returns the bytes of the signed part of the message
func (sm *UnsignedMessage) GetSignedMessage() []byte {
	return sm.Msg
}

// GetSignedHash returns the hash of the signed part of the message message as bytes (see issue #22)
func (sm *UnsignedMessage) GetSignedHash() types.HashBytes {
	return sm.Hash
}

// GetHashString returns the hash of the message as a string
func (sm *UnsignedMessage) GetHashString() types.HashStr {
	if sm.hashString == "" {
		sm.hashString = types.HashStr(sm.Hash)
	}
	return sm.hashString
}

// GetSigCount returns the number of times this message has been signed (can be greater than the number of signatures when using threshold signatures)
func (sm *UnsignedMessage) GetSigCount() (count int) {
	for _, pub := range sm.EncryptPubs {
		count++
		if pub.GetSigMemberNumber() != 1 {
			panic("shouldn't get non-signed message for non-single member pubs")
		}
	}
	return
}

// GetSigItems returns the list of signatures of the message
func (sm *UnsignedMessage) GetEncryptPubs() []Pub {
	return sm.EncryptPubs
}

// SetSigItems sets the list of signatures for the message
func (sm *UnsignedMessage) SetEncryptPubs(pubs []Pub) {
	sm.EncryptPubs = pubs
}

// GetBytes returns the bytes that make up the message
func (sm *UnsignedMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// PeekHeader returns the indices related to the messages without impacting m.
func (sm *UnsignedMessage) PeekHeaders(m *messages.Message,
	unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (sm *UnsignedMessage) Serialize(m *messages.Message) (int, error) {
	bytesWritten, offset, endOffset, sizeOffset, err := messages.SerializeInternalSignedHeader(sm.Index,
		sm.InternalSignedMsgHeader, m)
	if err != nil {
		return 0, err
	}

	sm.Msg, err = (*messages.MsgBuffer)(m).GetSlice(offset, endOffset)
	if err != nil {
		return bytesWritten, err
	}
	sm.Hash, _, err = m.CheckHash(offset, endOffset, false)
	if err != nil {
		return bytesWritten, err
	}
	if sm.Hash == nil || sm.Msg == nil {
		panic("should call FinishSerialize before sign")
	}

	var l int
	for _, nxt := range sm.EncryptPubs {
		if l, err = nxt.Serialize(m); err != nil {
			return bytesWritten, err
		}
		bytesWritten += l
	}
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(bytesWritten))
	if err != nil {
		return bytesWritten, err
	}

	return bytesWritten, nil
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (sm *UnsignedMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (n int, err error) {
	return sm.DeserializeEncoded(FromMessage(m), unmarFunc)
}

// DeserializeEncoded deserialzes a header into the object, returning the number of bytes read
func (sm *UnsignedMessage) DeserializeEncoded(m EncodedMsg, unmarFunc types.ConsensusIndexFuncs) (n int, err error) {
	var signOffset, signEndOffset int
	var size uint32
	sm.Index, n, signOffset, signEndOffset, size, err = messages.DeserialzeInternalSignedHeader(
		sm.InternalSignedMsgHeader, unmarFunc, m.Message)
	if err != nil {
		return
	}
	sm.Msg, err = (*messages.MsgBuffer)(m.Message).GetSlice(signOffset, signEndOffset)
	if err != nil {
		return
	}
	sm.Hash, _, err = m.CheckHash(signOffset, signEndOffset, false)
	if err != nil {
		return
	}

	for uint32(n) < size {
		pub := sm.newPubFunc()
		var l int
		l, err = pub.Deserialize(m.Message, unmarFunc)
		if err != nil {
			return
		}
		sm.EncryptPubs = append(sm.EncryptPubs, pub)
		n += l
	}

	return
}
