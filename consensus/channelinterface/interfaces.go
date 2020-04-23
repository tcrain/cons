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

/*
This package contains higher level network abstractions that will run on top and along side of the channel and csnet packages.
*/
package channelinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/stats"
	"time"

	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

/////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////

// BehaviorTracker objects track bad behaviors of connections.
// The functions must be safe to be accesses concurrently.
type BehaviorTracker interface {
	// GotError should be called whenever there was an error on the connection conInfo.
	// For example it sent an incorrect/duplicate message
	// Or there was a problem with the connection itself
	// Returns true if requestions from the connection should be rejected.
	GotError(err error, conInfos interface{}) bool
	// CheckShouldReject returns true if messages or connections from the address of conInfo should be rejected.
	CheckShouldReject(conInfo interface{}) bool
	// RequestedOldDecisions should be called when conInfo requested an old decided value, returns true if
	// the request should be rejected.
	RequestedOldDecisions(conInfo interface{}) bool
}

type ForwardFuncFilter func(sendChans []SendChannel) []SendChannel
type NewForwardFuncFilter func(allConnections []sig.Pub) (destinations sig.PubList,
	sendToRecv, sendToSend bool, sendRange SendRange)
type MsgConstructFunc func(pieces int) (toSelf *DeserializedItem, toOthers [][]byte)

// SendRange is used as a percentage of the recipients returned from NewForwardFuncFilter to send to.
type SendRange struct {
	From, Until int
}

func (sc SendRange) ComputeIndicies(sliLen int) (startIndex, endIndex int) {
	startIndex = int(float64(sliLen) * (float64(sc.From) / 99))
	endIndex = int(float64(sliLen) * (float64(sc.Until) / 99))
	return
}

// FullSendRange means to send to all recipients returnd from NewForwardFuncFilter
var FullSendRange = SendRange{
	From:  0,
	Until: 99,
}

// FrontHalfSendRange means to send to the first half of recipients returned from NewForwardFuncFilter
var FrontHalfSendRange = SendRange{
	From:  0,
	Until: 50,
}

// BackHalfSendRange means to send to the second half of recipients returned from NewForwardFuncFilter
var BackHalfSendRange = SendRange{
	From:  50,
	Until: 99,
}

// TimerInterface is returned from SendToSelf.
// The process that creates the timer is responsible for stopping it before the test returns.
type TimerInterface interface {
	// Stop should be called when the timer is no longer needed.
	Stop() bool
}

type MainChannel interface {
	// MakeConnections creates connections to the nodes at the list of pubs.
	// The pubs must have been added previously by AddExternalNode.
	// If a connection is added multiple times it increments a counter for that connection.
	MakeConnections(pubs []sig.Pub) (errs []error)
	// RemoveConnections removes connections to the list of pubs.
	// The pubs should have been added previously with MakeConnections.
	// If the connection has been added multiple times this decrements the counter,
	// and closes the connection when the counter reaches 0.
	RemoveConnections(pubs []sig.Pub) (errs []error)
	// GetConnections returns the list of connections that the node is currently connected to or trying to connect to.
	// GetConnections() (ret sig.PubKeyStrList)
	// MakeConnectionsCloseOther will connect to the nodes given by the pubs.
	// Any existing connections not in pubs will be closed.
	// This should be called after AddExternalNodes with a subset of the nodes
	// added there.
	// MakeConnectionsCloseOthers(pubs []sig.Pub) (errs []error)
	// SendToSelf sends a message to the current processes after a timout, it returns the timer
	// this method is concurrent safe.
	// The timer should either fire or be closed before the program exits.
	// I.E. it should be closed during the Collect() method called on consensus items.
	SendToSelf(deser []*DeserializedItem, timeout time.Duration) TimerInterface
	// ComputeDestinations returns the list of destinations given the forward filter function.
	ComputeDestinations(forwardFunc NewForwardFuncFilter) []SendChannel
	// Send sends a message to the outgoing connections
	// toSelf should be true if the message should also be received by the local node
	// the forwardChecker function will receive all current connections, and should return either all or a subset
	// of them based on forwarding rules, the set of channels returnd will be sent the message.
	// For example if it returns the input list, then the send is a broadcast to all nodes
	// IsProposal should be true if the message is a proposal message.
	// This method is not concurrent safe.
	Send(buff []byte, isProposal, toSelf bool, forwardChecker NewForwardFuncFilter, countStats bool)
	// SendHeader is the same as Send except it take a messasges.MsgHeader instead of a byte slice.
	// This should be used in the consensus implementations.
	// IsProposal should be true if the message is a proposal message.
	SendHeader(headers []messages.MsgHeader, isProposal, toSelf bool, forwardChecker NewForwardFuncFilter, countStats bool)
	// SendAlways is the same as Send, except the message will be sent even if the consensus is in initialization.
	// This is just used to request the state from neighbour nodes on initialization.
	SendAlways(buff []byte, toSelf bool, forwardChecker NewForwardFuncFilter, countStats bool)
	// SendTo sends buff to dest
	SendTo(buff []byte, dest SendChannel, countStats bool)
	// SendToPub sends buff to the node associated with the public key (if it exists), it returns an error if pub is not found
	// in the list of connections
	SendToPub(headers []messages.MsgHeader, pub sig.Pub, countStats bool) error
	// HasProposal should be called by the state machine when it is ready with its proposal for the next round of consensus.
	// It should be called after ProposalInfo object interface (package consinterface) method HasDecided had been called for the previous consensus instance.
	HasProposal(*DeserializedItem)
	// Recv should be called as the main consensus loop every time the node is ready to process a message.
	// It is expected to be called one at a time (not concurrent safe).
	// It will return utils.ErrTimeout after a timeout to ensure progress.
	Recv() (*RcvMsg, error)
	// Close should be called when the system is shutting down
	Close()
	// GotRcvConnection(rcvcon *SendRecvChannel)
	// CreateSendConnection adds the list of connections to the list of nodes to send messages to
	// CreateSendConnection(NetNodeInfo) error
	// StartMsgProcessThreads starts the threads that will process the messages.
	StartMsgProcessThreads()

	// AddExternalNode should be called with the list of addresses that a single external node will have.
	// In UDP a node might split packets over multiple connections, so this lets us know this list for each node.
	// In TCP this does nothing. // TODO should do something?
	AddExternalNode(NetNodeInfo)
	// RemoveExternalNode should be called when we will no longer receive messages from an external node,
	// it should be called with the list of addresses that a single external node will have.
	RemoveExternalNode(NetNodeInfo)
	// WaitUntilFewerSendCons blocks until there are at least n connections, or an error.
	WaitUntilAtLeastSendCons(n int) error
	// GetLocalNodeConnectionInfo returns the list of local addresses that this node is listening for messages(udp)/connections(tcp) on
	GetLocalNodeConnectionInfo() NetNodeInfo
	// StartInit is called when the system is starting and is recovering from disk
	// At this point any calls to Send/SendToSelf should not send any messages as the system is just replying events
	StartInit()
	// EndInit is called once recovery is finished
	EndInit()
	// InitInProgress returns true if StartInit has been called and EndInit has not.
	InitInProgress() bool
	// Reprocess is called on messages that were unable to be deserialized upon first reception, it is called by concurrent threads
	ReprocessMessage(*RcvMsg)
	// GetBehaviorTracker returns the BehaviorTracker object
	GetBehaviorTracker() BehaviorTracker
	// GetStats returns the stats object being used
	GetStats() stats.NwStatsInterface
}

// SendChannel represents a connection to an external node
// An object implementing MainChannel will keep a SendChannel for each open connection it has
type SendChannel interface {
	// ConnectSend() error
	// Close closes a connection
	Close(closeType ChannelCloseType) error
	Send([]byte) error
	// SendReturn([]byte, NetConInfo) error
	// GetConnID() string
	GetConnInfos() NetNodeInfo
	// ConnectRecv(chan RcvMsg, NetConInfo) error
	GetType() types.NetworkProtocolType
}

func ForwardAll(sendChans []SendChannel) []SendChannel {
	return sendChans
}

func ForwardAllPub(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv,
	sendToSend bool, sendRange SendRange) {

	return allPubs, false, false, FullSendRange
}

func ForwardAllPubFrontHalf(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv,
	sendToSend bool, sendRange SendRange) {

	return allPubs, false, false, FrontHalfSendRange
}

func ForwardAllPubBackHalf(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv,
	sendToSend bool, sendRange SendRange) {

	return allPubs, false, false, BackHalfSendRange
}

func BaseP2pForward(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
	sendRange SendRange) {
	return nil, true, false, FullSendRange
}

func BaseP2pForwardFrontHalf(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
	sendRange SendRange) {
	return nil, true, false, FrontHalfSendRange
}

func BaseP2pForwardBackHalf(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
	sendRange SendRange) {
	return nil, true, false, BackHalfSendRange
}

func BaseP2pNoProgress(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
	sendRange SendRange) {
	return nil, false, true, FullSendRange
}
