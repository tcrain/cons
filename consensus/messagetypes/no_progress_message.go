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

// NoProgressMessage is sent when a node has made no progress towards consensus after a timeout.
// It implements messages.MsgHeader
type NoProgressMessage struct {
	TestID             int
	IsUnconsumedOutput bool // if true then have already decided this index so only want future information
	basicMessage
}

func NewNoProgressMessage(index types.ConsensusIndex, isUnconsumedOutput bool, testID int) *NoProgressMessage {
	return &NoProgressMessage{testID, isUnconsumedOutput, basicMessage{cid: index}}
}

// GetID returns the header id for this header
func (npm *NoProgressMessage) GetID() messages.HeaderID {
	return messages.HdrNoProgress
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (npm *NoProgressMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(npm.GetID())
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (npm *NoProgressMessage) Serialize(m *messages.Message) (int, error) {
	l, err := npm.basicMessage.serializeID(m, npm.GetID())
	if err != nil {
		return l, err
	}
	(*messages.MsgBuffer)(m).AddBool(npm.IsUnconsumedOutput)
	n, _ := (*messages.MsgBuffer)(m).AddUint32(uint32(npm.TestID))
	return l + n + 1, nil
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (npm *NoProgressMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	l, err := npm.basicMessage.deserializeID(m, npm.GetID(), unmarFunc.ConsensusIDUnMarshaler)
	if err != nil {
		return l, err
	}
	if npm.IsUnconsumedOutput, err = (*messages.MsgBuffer)(m).ReadBool(); err != nil {
		return l + 1, err
	}
	v, r, err := (*messages.MsgBuffer)(m).ReadUint32()
	npm.TestID = int(v)
	return l + r + 1, err
}
