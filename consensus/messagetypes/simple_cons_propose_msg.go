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

// SimpleConsProposeMessage is used as a proposal for the simple cons test.
// The proposal is just the consensus index.
type SimpleConsProposeMessage struct {
	// basicMessage
	cid types.ConsensusIndex
	localMsgInterface
}

func NewSimpleConsProposeMessage(index types.ConsensusIndex) *SimpleConsProposeMessage {
	return &SimpleConsProposeMessage{cid: index} // basicMessage: basicMessage{index}}
}

// GetID returns the header id for this header
func (cm *SimpleConsProposeMessage) GetID() messages.HeaderID {
	return messages.HdrPropose
}

// GetMsgID returns the MsgID for this specific header, in this case a BinMsgID (see MsgID definition for more details)
func (cm *SimpleConsProposeMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(cm.GetID())
}
