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

// ConsBinStateMessage represents a set of many consensus messages all appended together.
// It is used during recovery to send blocks of messages.
// It implements messages.MsgHeader
type ConsBinStateMessage []byte

// GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
func (cm ConsBinStateMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(cm.GetID())
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cm ConsBinStateMessage) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, cm.GetID(), 0, (*messages.MsgBuffer)(m))

	// then the message
	l1, _ := (*messages.MsgBuffer)(m).AddBytes(cm)
	l += l1

	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

// GetBytes returns the bytes that make up the message
func (cm ConsBinStateMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cm ConsBinStateMessage) Deserialize(m *messages.Message, _ types.ConsensusIndexFuncs) (int, error) {
	_, _, _, size, _, err := messages.ReadHeaderHead(cm.GetID(), nil, (*messages.MsgBuffer)(m))
	return int(size), err
}

// PeekHeader returns the indices related to the messages without impacting m.
func (ConsBinStateMessage) PeekHeaders(m *messages.Message,
	unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

// GetID returns the header id for this header
func (cm ConsBinStateMessage) GetID() messages.HeaderID {
	return messages.HdrBinState
}

// GetIndex returns 0.
func (cm ConsBinStateMessage) GetIndex() types.ConsensusInt {
	return 0
}
