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
	"context"
	"fmt"
	"net"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
)

var encoding = config.Encoding

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewNetConnection creates new connection to conInfo.
// For TCP conInfos must contain a single element, it creates and maintains a connection
// For UDP multiple addresses can be used for a node. When sending to the node, messages will be sent in a round robin order, cycling through the addresses.
// (UDP doesn't maintain any connections, it uses NetPortListenerUDP accessed through NetMainChannel (TODO cleanup))
func NewNetSendConnection(conInfo channelinterface.NetNodeInfo, context context.Context,
	connStatus *ConnStatus, netMainChannel *NetMainChannel) (channelinterface.SendChannel, error) {
	if len(conInfo.AddrList) < 1 {
		return nil, fmt.Errorf("Must have at least 1 conn info")
	}
	_, err := net.ResolveUDPAddr(conInfo.AddrList[0].Nw, conInfo.AddrList[0].Addr)
	if err == nil {
		nsc := newNetConnectionUDP(conInfo, netMainChannel, true)
		return nsc, nil
	}
	// nsc.sendRecvChan = &channelinterface.SendRecvChannel{
	// 	ReturnChan: nsc,
	// 	ConnectionInfo: conInfo[0]}
	return newNetConnectionTCP(conInfo, context, connStatus, netMainChannel)
}
