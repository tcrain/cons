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

package messages

import (
	"github.com/tcrain/cons/consensus/types"
)

// AppendCopyMsgHeader appends the input into a new slice.
func AppendCopyMsgHeader(slice []MsgHeader, elems ...MsgHeader) []MsgHeader {
	newSlice := make([]MsgHeader, len(slice), len(slice)+len(elems))
	copy(newSlice, slice)
	return append(newSlice, elems...)
}

type MsgIDHeader interface {
	GetID() HeaderID // GetID returns the header id for this header
	GetMsgID() MsgID // GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
}

// MsgHeader is a component of a messsage that contains an id (an encoded 4 byte integer), followed by any data
// The interface allows for serialization and deserialzation of the message
type MsgHeader interface {
	MsgIDHeader
	SeralizeHeader
	// PeekHeader returns the indices related to the messages without impacting m.
	PeekHeaders(m *Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error)
	Deserialize(m *Message, unmarFunc types.ConsensusIndexFuncs) (int, error) // Deserialize deserialzes a header into the object, returning the number of bytes read
	GetBytes(m *Message) ([]byte, error)                                      // GetBytes returns the bytes that make up the header
}

// SerializeHeader implements the serialize method
type SeralizeHeader interface {
	Serialize(m *Message) (int, error) // Serialize appends a serialized header to the message m, and returns the size of bytes written
}

type InternalSignedMsgHeader interface {
	MsgIDHeader
	// GetBaseMsgHeader returns the header pertaning to the message contents (in most cases just directly returns the same header).
	GetBaseMsgHeader() InternalSignedMsgHeader
	// SerializeInternal serializes the header contents, returning the number of bytes written and the end of the part of the message that should be signed.
	SerializeInternal(m *Message) (bytesWritten, signEndOffset int, err error)
	// DeserializeInternal deserializes the header contents, returning the number of bytes read and the end of the signed part of the message.
	DeserializeInternal(m *Message) (bytesRead, signEndOffset int, err error)
	// ShallowCopy preforms a shallow copy of the header.
	ShallowCopy() InternalSignedMsgHeader
	// NeedsSMValidation should return a byte slice if the message should be validated by the state machine before being processed,
	// otherwise it should return nil.
	// If proposal is non-nil idx should be the state machine index that will be used to validate this message.
	// It takes as input the current index.
	// If the message has multiple proposals then proposalIdx is the index of the proposal being requested.
	// If causal ordering is used then the idx returned is not used.
	NeedsSMValidation(msgIndex types.ConsensusIndex, proposalIdx int) (idx types.ConsensusIndex, proposal []byte, err error)
	// NeedsCoinProof returns true if this message must include a proof for a coin message.
	GetSignType() types.SignType
}

func IsProposalHeader(msgIndex types.ConsensusIndex, hdr InternalSignedMsgHeader) bool {
	_, pr, err := hdr.NeedsSMValidation(msgIndex, 0)
	if err != nil {
		panic(err)
	}
	return len(pr) > 0
}

func SerializeHelper(headerID HeaderID, item types.BasicEncodeInterface, m *Message) (n int, err error) {
	l, sizeOffset, _ := WriteHeaderHead(nil, nil, headerID, 0, (*MsgBuffer)(m))
	n += l

	l, err = item.Encode((*MsgBuffer)(m))
	if err != nil {
		return
	}
	n += l

	// now update the size
	err = (*MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(n))
	if err != nil {
		return
	}
	return
}

func DeserializeHelper(headerID HeaderID, item types.BasicEncodeInterface,
	m *Message) (n int, err error) {

	l, _, _, size, _, err := ReadHeaderHead(headerID, nil, (*MsgBuffer)(m))
	if err != nil {
		return
	}
	n += l

	l, err = item.Decode((*MsgBuffer)(m))
	if err != nil {
		return
	}
	n += l

	if size != uint32(n) {
		err = types.ErrInvalidMsgSize
		return
	}

	return
}
