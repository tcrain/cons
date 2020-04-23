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

type SignedMessage interface {
	GetSignedHash() types.HashBytes
	GetSignedMessage() []byte
}

type BasicSignedMessage []byte

func (bsm BasicSignedMessage) GetSignedHash() types.HashBytes {
	return types.GetHash(bsm)
}
func (bsm BasicSignedMessage) GetSignedMessage() []byte {
	return bsm
}

// MultipleSignedMessage represents a message with a list of signatures for that message
type MultipleSignedMessage struct {
	messages.InternalSignedMsgHeader                      // the specific message
	Index                            types.ConsensusIndex // the cons index of the message
	newPubFunc                       func() Pub           // The local public key
	Hash                             types.HashBytes      // The hash of the signed part of the message as bytes
	hashString                       types.HashStr        // The hash of the signed part of the message as a string
	Msg                              []byte               // The signed part of the message
	SigItems                         []*SigItem           // A list of signatures
	// EncryptPub                       Pub                  // If this message was encrypted
}

// NewMultipleSignedMsg generates a new multiple signed msg object given the input, it does not compute the hash fields.
func NewMultipleSignedMsg(index types.ConsensusIndex, pub Pub, internalMsg messages.InternalSignedMsgHeader) *MultipleSignedMessage {
	return &MultipleSignedMessage{
		newPubFunc:              pub.New,
		Index:                   index,
		InternalSignedMsgHeader: internalMsg}
}

// ShallowCopy generates a new multiple signed msg object that is a shallow copy of the original.
func (sm *MultipleSignedMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MultipleSignedMessage{
		InternalSignedMsgHeader: sm.InternalSignedMsgHeader.ShallowCopy(),
		Index:                   sm.Index.ShallowCopy(),
		newPubFunc:              sm.newPubFunc,
		Hash:                    sm.Hash,
		hashString:              sm.hashString,
		Msg:                     sm.Msg,
		SigItems:                sm.SigItems}
}

// GetIndex returns the consensus index for this message.
func (sm *MultipleSignedMessage) GetIndex() types.ConsensusIndex {
	return sm.Index
}

// GetSignedMessage returns the bytes of the signed part of the message
func (sm *MultipleSignedMessage) GetSignedMessage() []byte {
	return sm.Msg
}

// GetSignedHash returns the hash of the signed part of the message message as bytes (see issue #22)
func (sm *MultipleSignedMessage) GetSignedHash() types.HashBytes {
	return sm.Hash
}

// GetHashString returns the hash of the message as a string
func (sm *MultipleSignedMessage) GetHashString() types.HashStr {
	if sm.hashString == "" {
		sm.hashString = types.HashStr(sm.Hash)
	}
	return sm.hashString
}

// GetSigCount returns the number of times this message has been signed (can be greater than the number of signatures when using threshold signatures)
func (sm *MultipleSignedMessage) GetSigCount() (count int) {
	for _, sigItem := range sm.SigItems {
		if sigItem.WasEncrypted {
			return 1 // encrypted messages signed by a single key
		}
		count += sigItem.Pub.GetSigMemberNumber()
	}
	return
}

// GetSigItems returns the list of signatures of the message
func (sm *MultipleSignedMessage) GetSigItems() []*SigItem {
	return sm.SigItems
}

// SetSigItems sets the list of signatures for the message
func (sm *MultipleSignedMessage) SetSigItems(si []*SigItem) {
	sm.SigItems = si
}

// GetBytes returns the bytes that make up the message
func (sm *MultipleSignedMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// PeekHeader returns the indices related to the messages without impacting m.
func (sm *MultipleSignedMessage) PeekHeaders(m *messages.Message,
	unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (sm *MultipleSignedMessage) Serialize(m *messages.Message) (int, error) {
	bytesWritten, signOffset, signEndOffset, sizeOffset, err := messages.SerializeInternalSignedHeader(sm.Index,
		sm.InternalSignedMsgHeader, m)
	if err != nil {
		return 0, err
	}
	return sm.Sign(m, bytesWritten, signOffset, signEndOffset, sizeOffset)
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (sm *MultipleSignedMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (n int, err error) {
	return sm.DeserializeEncoded(FromMessage(m), unmarFunc)
}

// DeserializeEncoded deserialzes a header into the object, returning the number of bytes read
func (sm *MultipleSignedMessage) DeserializeEncoded(m EncodedMsg, unmarFunc types.ConsensusIndexFuncs) (n int, err error) {
	var bytesRead, signOffset, signEndOffset int
	var size uint32
	sm.Index, bytesRead, signOffset, signEndOffset, size, err = messages.DeserialzeInternalSignedHeader(
		sm.InternalSignedMsgHeader, unmarFunc, m.Message)
	if err != nil {
		return
	}
	return sm.DeserSign(m, bytesRead, signOffset, signEndOffset, size)
}

// Sign takes as input a message m, it's lenght l, the part of the message to sign (from offset to endOffset) and where
// within the message the size should be stored (sizeOffset).
// If the sm object has a private key, then it uses that to sign the message.
// If the sm object has any signatures (in SigItems) those are also added to the message
// It returns the total lenght of the message including the signatures
func (sm *MultipleSignedMessage) Sign(m *messages.Message, l, offset, endOffset, sizeOffset int) (int, error) {
	// If we have a private key we sign with that
	// Oterwise we use the exisiting sigs in SigItmes
	if sm.Index.FirstIndex == nil {
		panic("should only sign messages with indices")
	}

	var err error
	sm.Msg, err = (*messages.MsgBuffer)(m).GetSlice(offset, endOffset)
	if err != nil {
		return l, err
	}
	sm.Hash, _, err = m.CheckHash(offset, endOffset, false)
	if err != nil {
		return l, err
	}
	if sm.Hash == nil || sm.Msg == nil {
		panic("should call FinishSerialize before sign")
	}
	if len(sm.SigItems) == 0 {
		return l, nil
	}
	// Include any already serialized sigs in case they are included
	for _, sigItem := range sm.SigItems {
		if sigItem.WasEncrypted {
			continue
		}
		l1, _ := (*messages.MsgBuffer)(m).AddBytes(sigItem.SigBytes)
		l += l1
	}
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return l, err
	}
	return l, nil
}

// DeserSign takes as input a signed message m, the index in the message where signatures are placed l, the start
// and end indecies of the signed part of the message (offset and endOffest) and size the total size of the message
// It fills in the Hash and SigItems fields of the sm object and returns the size of the message.
func (sm *MultipleSignedMessage) DeserSign(m EncodedMsg, l, offset, endOffset int, size uint32) (int, error) {
	var err error
	sm.Msg, err = (*messages.MsgBuffer)(m.Message).GetSlice(offset, endOffset)
	if err != nil {
		return 0, err
	}
	sm.Hash, _, err = m.CheckHash(offset, endOffset, false)
	if err != nil {
		return 0, err
	}

	if len(sm.SigItems) != 0 { // sanity check
		panic("should not have sigs")
	}

	/*	if m.WasEncrypted {
		if m.Pub == nil {
			panic("should have pub key after encrypt")
		}
		sm.EncryptPub = m.Pub
	}*/

	// Get the sigs
	for uint32(l) < size {
		startSigoffset := (*messages.MsgBuffer)(m.Message).GetReadOffset()

		sigItem, l1, err := sm.newPubFunc().DeserializeSig(m.Message, sm.GetSignType())
		l += l1
		if err != nil {
			return 0, err
		}

		endSigoffset := (*messages.MsgBuffer)(m.Message).GetReadOffset()
		sigItem.SigBytes, err = (*messages.MsgBuffer)(m.Message).GetSlice(startSigoffset, endSigoffset)
		if err != nil {
			return 0, err
		}
		sm.SigItems = append(sm.SigItems, sigItem)
	}
	if len(sm.SigItems) < 1 {
		return 0, types.ErrNoValidSigs
	}
	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}
