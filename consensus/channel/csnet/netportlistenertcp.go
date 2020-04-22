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
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
)

// NetPortListenerTCP maintains local listening TCP addresses
// Whenever a connection is added or dropped it informs ConnStatus, who then handles the connection
type NetPortListenerTCP struct {
	netMainChannel *NetMainChannel
	listener       net.Listener
	connInfo       channelinterface.NetConInfo
	connStatus     *ConnStatus
	closed         int32
	wgListener     sync.WaitGroup
}

// GetConnInfo returns the local listening addresses
func (nsc *NetPortListenerTCP) GetConnInfo() channelinterface.NetConInfo {
	return nsc.connInfo
}

// NewNetPortListenerTCP creates a new NetPortListenerTCP object and starts listening on the address connInfo
// When a new connection is made, it informs ConnStatus.
// If the addresses are given in a format like '{address}:0' it will bind on any free port
// It returns the list of local addresses that the connections were made on
func NewNetPortListenerTCP(connInfo channelinterface.NetConInfo, connStatus *ConnStatus, netMainChannel *NetMainChannel) (*NetPortListenerTCP, []channelinterface.NetConInfo, error) {
	nrc := &NetPortListenerTCP{}
	nrc.netMainChannel = netMainChannel
	nrc.connStatus = connStatus
	nrc.connInfo = connInfo

	var ln net.Listener
	var err error
	logging.Info("Calling TCP listen on ", connInfo)
	for i := 0; i < config.ConnectRetires; i++ {
		ln, err = net.Listen(connInfo.Nw, connInfo.Addr)
		if err == nil {
			break
		}
		logging.Error(err)
		// we try again in case connection is still closing
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		panic(err)
	}
	logging.Info("Done TCP listen on ", connInfo)
	nrc.connInfo = channelinterface.UpdateNw(nrc.connInfo, ln.Addr())

	nrc.listener = ln
	nrc.runListenLoop()

	return nrc, []channelinterface.NetConInfo{nrc.connInfo}, nil
}

func (nsc *NetPortListenerTCP) addExternalNode(conn channelinterface.NetNodeInfo) {
	// nothing for tcp
}

func (nsc *NetPortListenerTCP) removeExternalNode(conn channelinterface.NetNodeInfo) {
	// nothing for tcp
}

// loop that listens for incoming connections
func (nrc *NetPortListenerTCP) runListenLoop() {
	nrc.wgListener.Add(1)
	go func() {
		defer nrc.wgListener.Done()
		for {
			conn, err := nrc.listener.Accept()
			if err != nil {
				// no data race here?, b/c close listener close method should invoke memory barrier?
				if atomic.LoadInt32(&nrc.closed) == 0 {
					panic(err)
				}
				return
			}
			err = newNetConnectionTCPAlreadyConnected(conn, nrc.connStatus, nrc.netMainChannel)
			if err != nil {
				logging.Info(err)
				continue
			}
			// src := &channelinterface.SendRecvChannel{
			// 	MainChan: nsc,
			// 	ReturnChan: nsc,
			// 	ConnectionInfo: nsc.nci[0]}
			// nrc.netMainChannel.GotRcvConnection(src)
		}
	}()
}

// // GetType returns channelinterface.TCP
// func (nsc *NetPortListenerTCP) GetType() channelinterface.NetworkProtocolType {
// 	return channelinterface.TCP
// }

// Close closes the listening ports (note it does not affect any active connections)
func (nrc *NetPortListenerTCP) prepareClose() {
	if atomic.CompareAndSwapInt32(&nrc.closed, 0, 1) {
		// t := time.Time{}
		// t.Add(time.Duration(1))
		// nrc.listener.(*net.TCPListener).SetDeadline(t)
		err := nrc.listener.Close()
		if err != nil {
			panic(err)
		}
		nrc.wgListener.Wait()
	}
}

func (nrc *NetPortListenerTCP) finishClose() {
}
