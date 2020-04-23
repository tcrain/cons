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

// MvProposeMessage details a proposal for an instance of multi-value consensus.
// It implements messages.MsgHeader
type MvProposeMessage struct {
	Index       types.ConsensusIndex // Index is the index of consensus for the proposal
	Proposal    []byte               // The actual proposal
	ByzProposal []byte               // Alternative proposal for byzantine nodes
	localMsgInterface
}

// NewMvProposeMessage creates a new mv propose message
func NewMvProposeMessage(index types.ConsensusIndex, proposal []byte) *MvProposeMessage {

	return &MvProposeMessage{
		Index:    index,
		Proposal: proposal}
}

// GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
func (pm *MvProposeMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pm.GetID())
}

// GetIndex returns the consensus index of this message
func (pm *MvProposeMessage) GetIndex() types.ConsensusIndex {
	return pm.Index
}

// GetID returns the header id for this header
func (pm *MvProposeMessage) GetID() messages.HeaderID {
	return messages.HdrMvPropose
}
