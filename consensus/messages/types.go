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

// HeaderID Each message type has a its own header id, this will identify serialized messages
type HeaderID uint32

// MsgID: Each message has a MsgID, that may differentiate it between messages of the same type
// for example a binary messages have the same MsgID for the same round (for both 1 and 0)
// but have different MsgIDs for different rounds this is only used internally and not serialized
// The idea is so we can cout how many messages of each type we have gotten.
// For example if we want to have 'n-t' messages of type 'aux' for round 1 of binary consensus,
// we will use the MsgID to count these, since they will have the same MsgID even if they
// support different binary values.
// Note it is importat that MsgID is comparable with == (i.e. if it returns a pointer
// to a new object then every message header will have a different MsgID)
// TODO maybe this check should happen in a method instead like 'CheckEqual'
type MsgID interface {
	IsMsgID() bool                             // IsMsgID to satisfy the interface and returns true
	ToBytes(index types.ConsensusIndex) []byte // Returns the byte representation of MsgID
	// CheckEqual(MsgID) bool
}

// BasicMsgID implements the MsgID interface as a HeaderID.
// So every message header with the same HeaderID will have equal MsgIDs.
type BasicMsgID HeaderID

// IsMsgID to satisfy the interface and returns true
func (BasicMsgID) IsMsgID() bool {
	return true
}

func (bm BasicMsgID) ToBytes(index types.ConsensusIndex) []byte {
	m := NewMsgBuffer()
	m.AddConsensusID(index.Index)
	m.AddHeaderID(HeaderID(bm))
	return m.GetRemainingBytes()
}
