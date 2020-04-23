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

// ConsMessage is a header that indicates it will be followed by a message dealing with consensus.
// It implements messages.MsgHeader
type ConsMessage struct {
	basicMessage
}

func NewConsMessage() *ConsMessage {
	return &ConsMessage{}
}

// GetID returns the header id for this header
func (cm *ConsMessage) GetID() messages.HeaderID {
	return messages.HdrConsMsg
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (cm *ConsMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(cm.GetID())
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cm *ConsMessage) Serialize(m *messages.Message) (int, error) {
	return cm.basicMessage.serializeID(m, cm.GetID())
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cm *ConsMessage) Deserialize(m *messages.Message, _ types.ConsensusIndexFuncs) (int, error) {
	return cm.basicMessage.deserializeID(m, cm.GetID(), nil)
}
