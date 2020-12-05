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

package channelinterface

import (
	"github.com/tcrain/cons/consensus/deserialized"
	"sync/atomic"
)

// used by GetNextConnCounter
var connCounter uint64

// ChannelCloseType dictates the reason for closing a channel
// Currently all the close types do the same thing
type ChannelCloseType int

const (
	// EndTestClose must be used when the test is finished and everything is shutting down.
	// It should only be called from a thread that does not process consensus messages, otherwise it can block.
	EndTestClose ChannelCloseType = iota
	// CloseDuringTest can be used to close channels during the test from threads that process consensus messages.
	CloseDuringTest
)

// GetNextConnCounter is a atomic interger called each time a connection is created to assign
// it a unique id
func GetNextConnCounter() uint64 {
	return atomic.AddUint64(&connCounter, 1)
}

// RcvMsg is the struct returned by the Recv call of MainChannel
type RcvMsg struct {
	CameFrom     int                              // For debugging to see where this message was processed
	Msg          []*deserialized.DeserializedItem // A list of deserialized messages
	SendRecvChan *SendRecvChannel                 // The channel that the messages were received from (or nil if it is not availble)
	IsLocal      bool                             // True if the messages were sent from the local node
}

// ConnDetails stores the address and a connection
type ConnDetails struct {
	Addresses NetNodeInfo
	Conn      SendChannel
}

type ConMapItem struct {
	NI             NetNodeInfo
	Idx            int
	IsConnected    bool
	NewCon         bool // used in MakeConnectionsCloseOthers ( no longer used)
	ConCount       int  // how many instances are using this connection, when set to 0 connection is closed
	AddressUnknown bool // is true if we don't know the connection address
}
