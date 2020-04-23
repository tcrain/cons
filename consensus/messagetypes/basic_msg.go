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

// basicMessage is a partial implementation of messages.MsgHeader for messages that only hold their id and no extra information.
// It is different than other messages in that it only serializes the main Index, and not the first and additional indices.
type basicMessage struct {
	cid types.ConsensusIndex
}

// PeekHeader returns the indices related to the messages without impacting m.
func (basicMessage) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	if index, err = messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m)); err != nil {
		return
	}
	index.Index = index.FirstIndex
	index.FirstIndex = nil
	return
}

// serializeID appends a serialized header to the message m, and returns the size of bytes written
func (npm *basicMessage) serializeID(m *messages.Message, id messages.HeaderID) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(npm.cid.Index, nil, id, 0, (*messages.MsgBuffer)(m))
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

// GetIndex returns the consensus index for this message.
func (npm *basicMessage) GetIndex() types.ConsensusIndex {
	return npm.cid
}

// GetBytes returns the bytes that make up the message
func (npm *basicMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// deserializeID deserialzes a header into the object, returning the number of bytes read
func (npm *basicMessage) deserializeID(m *messages.Message, id messages.HeaderID,
	unmarFunc types.ConsensusIDUnMarshaler) (int, error) {

	_, idx, _, size, _, err := messages.ReadHeaderHead(id, unmarFunc, (*messages.MsgBuffer)(m))
	npm.cid = types.ConsensusIndex{
		Index: idx,
	}
	return int(size), err
}
