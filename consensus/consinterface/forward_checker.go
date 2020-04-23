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

package consinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
)

// ForwardChecker keeps track of successfully processes consensus messages and decides if they should be forwarded on the network or not.
// There is one created for each consensus instance.
// Each time a valid message is processes it will be sent to the forward checker through CheckForward.
// ConsState will check for messages to be forwared by calling GetNextForwardItem.
// GetNextForwardItem is called repetably on a timeout for the most recent consensus instance.
// It is also called each time a message is received on the instance that the message corresponds to.
// (TODO maybe also call old consensus instances on a timeout to help other nodes progress faster?)
// ForwardChecker should be called by cons_state and net_main_channel but only from a single thread, so no synchronization needed (TODO be sure this remains true)
type ForwardChecker interface {
	AdditionalForwardCheckerOps
	// forwardchecker.internalForwardChecker
	// CheckForward is called each time a message is successfully processes by ConsState, sndRcvChan is the channel who sent the message, progress is given by the consensus
	// object after the message is processesed and is true if the message made progress in the consensus.
	// The inputs endThreshold, maxPossible int, msgID messages.MsgID should come from item.GetBufferCount called on the msg.
	// The input sigCount is the number of signatures received for this MsgID so for for this consensus.
	// The input memberChecker correspond to the consensus instance this message is from.
	// The forward checker should then keep this message if it should be forwarded later, messages are only forwarded when they are returned from GetNextForwardItem
	// Is propose message should be true if the message is a proposal message.
	CheckForward(sndRcvChan *channelinterface.SendRecvChannel, msg *channelinterface.DeserializedItem, shouldForward bool,
		isProposalMessage bool, endThreshold, maxPossible, sigCount int,
		msgID messages.MsgID, memberChecker *MemCheckers)
	// GetNextForward item is called by ConsState after a consensus message is successfully processed and after a timeout.
	// If there is a message to be forwareded, it should return that.
	// It will be called in a loop until msg is nil. The forwardFunc function takes the list of nodes the node is connected to and returns the set
	// of nodes to forward the message to. The returned message and function are used as input to MainChannel.Send.
	GetNextForwardItem(stats.NwStatsInterface) (msg []*channelinterface.DeserializedItem, forwardFunc channelinterface.NewForwardFuncFilter)
	// New creates a new ForwardChecker for the consensus index. It will be always be called on an "initialForwardChecker" that is given as input to
	// MemberCheckerState.Init. ParticipantCount is the number of nodes in the system.
	New(idx types.ConsensusIndex, participants, allPubs sig.PubList) ForwardChecker
}

type AdditionalForwardCheckerOps interface {
	// ShouldForward determines is forwarding is necessary based on if a message has made progress
	// towards a decision in consensus. For example in all to all communication
	// pattern it should return false since forwarding is not necessary.
	// Is proposal messages is true if the message is a proposal.
	ShouldForward(progress bool, isProposalMessage bool) bool
	// GetFanOut returns the number of nodes messages are forwarded to.
	GetFanOut() int

	// GetForwardList is called before a message is broadcast, it takes as input the list of all nodes in the system.
	// It returns the list of public keys to which the message should be sent.
	GetNewForwardListFunc() channelinterface.NewForwardFuncFilter

	// GetHalfHalfForwardListFunc is the same as GetNewForwardListFunc except is returns functions that broadcast
	// to half of the participants.
	GetHalfHalfForwardListFunc() (firstHalf, secondHalf channelinterface.NewForwardFuncFilter)

	// GetNoProgressForwardFunc is called before a noprogress message is broadcast.
	GetNoProgressForwardFunc() channelinterface.NewForwardFuncFilter
}

// GetForwardAllFunc returns the forward function that sends to all connections.
// It is used in case a consensus wants to send a special message to all connections at once.
func GetForwardAllFunc() func(sendChans []channelinterface.SendChannel) []channelinterface.SendChannel {
	return channelinterface.ForwardAll
}
