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

package csnet

import (
	"fmt"
	"net"

	"github.com/tcrain/cons/consensus/channelinterface"
)

// NetPortListener maintains the local open network ports
// In TCP it handles newly received connections and informs ConnStatus
// In UDP it handles all sending and receiving of messages given UDP is connectionless
type NetPortListener interface {
	addExternalNode(conns channelinterface.NetNodeInfo)
	removeExternalNode(conns channelinterface.NetNodeInfo)
	prepareClose()
	finishClose()
}

// NewNetPortListener creates a new NetPortListener on the given local addresses
// If the addresses are given in a format like '{address}:0' it will bind on any free port
// It returns the list of local addresses that the connections were made on
func NewNetPortListener(connInfos []channelinterface.NetConInfo, connStatus *ConnStatus, netMainChannel *NetMainChannel) (NetPortListener, []channelinterface.NetConInfo, error) {
	if len(connInfos) < 1 {
		return nil, nil, fmt.Errorf("Must have at least 1 conn info")
	}

	_, err := net.ResolveUDPAddr(connInfos[0].Nw, connInfos[0].Addr)
	if err == nil {
		return NewNetPortListenerUDP(connInfos, netMainChannel)
	}
	if len(connInfos) != 1 {
		return nil, nil, fmt.Errorf("Must only have one listen address for tcp")
	}
	return NewNetPortListenerTCP(connInfos[0], connStatus, netMainChannel)
}
