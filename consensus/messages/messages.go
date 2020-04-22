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
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/types"
)

type Any interface {
}

// Interface for using byte slice with the serialize method
type SeralizeBuff []byte

// Serialize appends a serialized header to the message m, and returns the size of bytes written.
func (buff SeralizeBuff) Serialize(m *Message) (int, error) {
	return (*MsgBuffer)(m).Write(buff)
}

///////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////

// SerializeSingleItem serializes a header by itself into a byte slice
func SerializeSingleItem(header MsgHeader) []byte {
	mb := NewMsgBuffer()
	_, err := header.Serialize((*Message)(mb))
	if err != nil {
		panic(err)
	}
	return mb.GetRemainingBytes()
}

// SerializeSingle serializes a header by itself into a byte slice
func CreateSingleMsgBytes(header SeralizeHeader) []byte {
	mb := NewMsgBufferSize(4, 8+4+64)

	msg, err := AppendSerialize((*Message)(mb), header)
	if err != nil {
		panic(err)
	}
	return msg.GetBytes()
}

// CreateMsgSingle creates a serialized message containing a single header.
func CreateMsgSingle(header MsgHeader) (*Message, error) {
	mb := NewMsgBufferSize(4, 8+4+64)

	return AppendHeader((*Message)(mb), header)
}

// AppendSerialize adds a serialized header to the end of msg and updates the size
func AppendSerialize(msg *Message, header SeralizeHeader) (*Message, error) {
	if header == nil {
		return msg, nil
	}

	_, err := header.Serialize(msg)
	if err != nil {
		return nil, err
	}

	err = (*MsgBuffer)(msg).WriteUint32AtStart(uint32(msg.Len() - 4))
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// AppendHeader adds a serialized header to the end of msg and updates the size
func AppendHeader(msg *Message, header MsgHeader) (*Message, error) {
	return AppendSerialize(msg, header)
}

// AppendMessage appends two messages together into a single message and upates the size
// Only front is updated
func AppendMessage(front *Message, end *Message) {
	(*MsgBuffer)(front).AddBytes(end.GetBytes())
	err := (*MsgBuffer)(front).WriteUint32AtStart(uint32(front.Len() - 4))
	if err != nil {
		panic(err)
	}
}

// Serialize seralizes a header
func SerializeHeader(header MsgHeader) (*Message, error) {
	mb := (*Message)(NewMsgBufferSize(0, 8+4+64))
	_, err := header.Serialize(mb)
	if err != nil {
		return nil, err
	}
	return mb, nil
}

// Serialize seralizes a header
func SerializeHeaders(headers []MsgHeader) (*Message, error) {
	mb := (*Message)(NewMsgBufferSize(0, 8+4+64))
	for _, hdr := range headers {
		_, err := hdr.Serialize(mb)
		if err != nil {
			return nil, err
		}
	}
	return mb, nil
}

// CreateMsg creates a serialzed messages from a set of headers
func CreateMsg(headers []MsgHeader) (*Message, error) {
	mb := NewMsgBufferSize(4, 8+4*len(headers)+64)

	return AppendHeaders((*Message)(mb), headers)
}

// AppendHeaders serialized the list of headers and appends them to the end of msg
func AppendHeaders(msg *Message, headers []MsgHeader) (*Message, error) {
	for _, h := range headers {
		if h == nil {
			continue
		}
		_, err := h.Serialize(msg)
		if err != nil {
			return nil, err
		}
	}

	err := (*MsgBuffer)(msg).WriteUint32AtStart(uint32(msg.Len() - 4))
	if err != nil {
		return nil, err
	}

	return msg, nil
}

///////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////

type Message MsgBuffer

func NewMessage(buff []byte) *Message {
	return (*Message)(ToMsgBuffer(buff))
}

func (m *Message) GetBytes() []byte {
	return (*MsgBuffer)(m).GetRemainingBytes()
}

func (m *Message) GetDebug() []byte {
	return m.buff
}

func (m *Message) ResetOffset() {
	(*MsgBuffer)(m).ResetOffset()
}

func (m *Message) Len() int {
	return (*MsgBuffer)(m).RemainingLen()
}

// func (m *Message) PopMsgIndex() (types.ConsensusInt, error) {
// 	return (*MsgBuffer)(m).Readtypes.ConsensusInt()
// }

func (m *Message) PeekMsgIndex() (types.ConsensusInt, error) {
	idx, err := (*MsgBuffer)(m).PeekUint64At(indexIndex)
	return types.ConsensusInt(idx), err
}

func (m *Message) PeekMsgConsID(unmarFunc types.ConsensusIDUnMarshaler) (cid types.ConsensusID, err error) {
	cid, _, err = (*MsgBuffer)(m).PeekConsIDAt(indexIndex, unmarFunc)
	return
}

func (m *Message) PopMsgSize() (int, error) {
	v, _, err := (*MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}
	if int(v) != m.Len() {
		return 0, types.ErrInvalidMsgSize
	}
	return int(v), err
}

func (m *Message) PopHeaderType() (int, error) {
	v, _, err := (*MsgBuffer)(m).ReadUint32()
	return int(v), err
}

func (m *Message) PeekHeaderType() (HeaderID, error) {
	v, err := (*MsgBuffer)(m).PeekUint32At(headerIndex)
	return HeaderID(v), err
}

func (m *Message) PeekHeaderTypeAt(offs int) (HeaderID, error) {
	v, err := (*MsgBuffer)(m).PeekUint32At(offs)
	return HeaderID(v), err
}

func (m *Message) ComputeHash(offset, endoffset int, includeHash bool) (types.HashBytes, int, error) {
	buff, err := (*MsgBuffer)(m).GetSlice(offset, endoffset)
	if err != nil {
		return nil, 0, err
	}
	hash := types.GetHash(buff)
	var l int
	if includeHash {
		l, _ = (*MsgBuffer)(m).AddBytes(hash)
	}
	return hash, l, nil
}

var ErrInvalidHash = fmt.Errorf("Invalid hash")

func (m *Message) CheckHash(offset int, endoffset int, includesHash bool) (types.HashBytes, int, error) {
	var hsh []byte
	var err error
	var hashLen int
	if includesHash {
		hashLen = types.GetHashLen()
		hsh, err = (*MsgBuffer)(m).ReadBytes(hashLen)
		if err != nil {
			return nil, 0, err
		}
	}
	buff, err := (*MsgBuffer)(m).GetSlice(offset, endoffset)
	if err != nil {
		return nil, 0, err
	}
	check := types.GetHash(buff)
	if includesHash {
		if !bytes.Equal(hsh, check) {
			return nil, 0, ErrInvalidHash
		}
	}
	return check, hashLen, nil
}

func GetMsgSizeLen() int {
	// uint32
	return 4
}
