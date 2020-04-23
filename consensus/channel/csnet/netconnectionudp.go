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
	"github.com/tcrain/cons/config"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/types"
)

// NetConnection represents a connection to an external node
// TCP connections are maintained here, constatus is updated when an error happens on an existing connection
// UDP connections just store the address (since it is connectionless), the operations just call the connections NetPortListenerUDP (TODO cleanup)
type NetConnectionUDP struct {
	nci            channelinterface.NetNodeInfo // the addresses
	netMainChannel *NetMainChannel              // pointer to main channel
	sndIdx         uint32                       // we go through the addresses round robin for balancing the sending over all the connections
	closeChan      chan bool
	mutex          sync.Mutex
	closed         int32
	// TODO allow encryption over UDP
}

func newNetConnectionUDP(nci channelinterface.NetNodeInfo, netMainChannel *NetMainChannel, isSendConnection bool) *NetConnectionUDP {
	ret := &NetConnectionUDP{nci: nci, netMainChannel: netMainChannel, sndIdx: uint32(rand.Intn(len(nci.AddrList)))}
	if isSendConnection {
		ret.mutex.Lock()
		ret.closeChan = make(chan bool, 1)
		ticker := time.NewTicker(config.RcvConUDPUpdate * time.Millisecond)
		var tickCount int
		netMainChannel.netPortListener.(*NetPortListenerUDP).sendChan <- sendinfo{buff: []byte{1},
			addr: nci.AddrList[0]}
		go func() {
			/*			defer func() { // TODO clean this up, here we can panic because we send on a close channel (line 229 of net_main_channel)
						if r := recover(); r != nil {
							logging.Info(r)
							ret.mutex.Unlock()
						}
					}()*/
			for {
				select {
				case <-ret.closeChan:
					netMainChannel.netPortListener.(*NetPortListenerUDP).sendChan <- sendinfo{buff: []byte{0},
						addr: nci.AddrList[0]}
					ticker.Stop()
					ret.mutex.Unlock()
					return
				case <-ticker.C:
					tickCount++
					// send keep alive
					netMainChannel.netPortListener.(*NetPortListenerUDP).sendChan <- sendinfo{buff: []byte{2},
						addr: nci.AddrList[0]}
				}
			}
		}()
	}
	return ret
}

func newNetConnectionUDPAddr(nci channelinterface.NetConInfo, netMainChannel *NetMainChannel) *NetConnectionUDP {
	return &NetConnectionUDP{nci: channelinterface.NetNodeInfo{
		AddrList: []channelinterface.NetConInfo{nci}}, netMainChannel: netMainChannel, sndIdx: 0}
}

// GetType returns channelinterface.UDP
func (nsc *NetConnectionUDP) GetType() types.NetworkProtocolType {
	return types.UDP
}

// GetLocalNodeConnectionInfo returns the addresses for this connection
func (nsc *NetConnectionUDP) GetConnInfos() channelinterface.NetNodeInfo {
	return nsc.nci
}

// Close shuts down the connection
func (nsc *NetConnectionUDP) Close(closeType channelinterface.ChannelCloseType) error {
	// UDP conns use the main listen channel
	if atomic.CompareAndSwapInt32(&nsc.closed, 0, 1) && nsc.closeChan != nil {
		close(nsc.closeChan)
		nsc.mutex.Lock()
		nsc.mutex.Unlock()
	}
	return nil
}

// Send sends a byte slice to the node addressed by this connection
func (nsc *NetConnectionUDP) Send(buff []byte) (err error) {
	res := atomic.AddUint32(&nsc.sndIdx, 1)
	// TODO fix this, too much indirection, should clean up udp
	nsc.netMainChannel.netPortListener.(*NetPortListenerUDP).sendChan <- sendinfo{buff, nsc.nci.AddrList[res%uint32(len(nsc.nci.AddrList))]}
	return nil
}
