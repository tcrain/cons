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

package types

import (
	"fmt"
)

// NetworkPropagationType defines how messages are propagated throughout the network.
type NetworkPropagationType int

const (
	AllToAll         NetworkPropagationType = iota // Every node broadcasts its messages to every other node.
	P2p                                            // Each node is connected to a fixed set of neighbords to which it sends and fowards messages.
	Random                                         // Each time a node sends or forwards a message it choises a random set of nodes.
	RequestForwarder                               // Forward only to those who have requested to get messages.
)

// String returns the network type as a readable string.
func (ct NetworkPropagationType) String() string {
	switch ct {
	case AllToAll:
		return "AllToAll"
	case P2p:
		return "P2P"
	case Random:
		return "Random"
	case RequestForwarder:
		return "RequestForwarder"
	default:
		return fmt.Sprintf("NetworkPropagationType%d", ct)
	}
}

// NetworkProtocolType defines the internet protocol nodes use for communication.
// Currently supports UDP and TCP (see issue https://github.com/tcrain/cons/issues/13 for UDP).
type NetworkProtocolType int

const (
	UDP NetworkProtocolType = iota // Nodes communicate using UDP
	TCP                            // Nodes communicate using TCP
)

// NetworkProtocolTypes lists all the posilbe network protocol type
var NetworkProtocolTypes = []NetworkProtocolType{UDP, TCP}

// String returns the network protocol type as a readable string.
func (ct NetworkProtocolType) String() string {
	switch ct {
	case UDP:
		return "udp"
	case TCP:
		return "tcp"
	default:
		return fmt.Sprintf("ConType%d", ct)
	}
}
