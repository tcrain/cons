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
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/utils"
)

// send info is used internally that tracks a message and it's sender address.
type sendinfo struct {
	buff []byte
	addr channelinterface.NetConInfo
}

// NetPortListenerUDP maintains local UDP ports
// For each port it runs a recveive and send loop
type NetPortListenerUDP struct {
	netMainChannel *NetMainChannel
	conns          []*net.UDPConn                // list of local open UDP ports
	connInfos      []channelinterface.NetConInfo // the addresses of the conns
	closed         int32                         // accessed atomically to indicate when the connections are closed
	sendChan       chan sendinfo                 // the send threads loop on this, taking out messages and sending them
	packetChan     chan packet                   // multi-packet messages are sent on this channel to be reconstructed by the reconstuct packet thread

	sendLock          sync.Mutex     // concurrency control
	wgReadGroup       sync.WaitGroup // wait for the read threads to finish
	wgConstructPacket sync.WaitGroup // wait for the reconstruct packet thread to finsih
	udpMsgPool        *udpMsgPool

	connMap *UDPConnMap // concurrent safe map of addresses to connection objects

	addConnInfoChannel    chan connInfoItem
	removeConnInfoChannel chan connInfoItem
}

type connInfoItem struct {
	ni      channelinterface.NetNodeInfo
	retChan chan bool
}

// NewNetPortListenerUDP creates a new NetPortListenerUDP object and starts loops for recving and sending on the opened ports.
// All network sends and receive will happen through this object.
// NetConnectionUDP objects will go through this object to perform their sends.
// Each node may have multiple UDP addresses, this allows it to send and receive over multiple connections.
// Large messages are split into multiple packets then sent over the multiple connections per node.
// When messages are received, if they are multi-packet then they are reconstructed, and NetMainChannel.ProcessMessage is called.
// If the addresses input to this function are given in a format like '{address}:0' it will bind on any free port.
// It returns the list of local addresses that the connections were made on.
func NewNetPortListenerUDP(connInfos []channelinterface.NetConInfo, netMainChannel *NetMainChannel) (*NetPortListenerUDP, []channelinterface.NetConInfo, error) {
	// First check the addresses are vaild
	for _, connInfo := range connInfos {
		_, err := net.ResolveUDPAddr(connInfo.Nw, connInfo.Addr)
		if err != nil {
			return nil, nil, err
		}
	}

	nrc := &NetPortListenerUDP{}
	nrc.netMainChannel = netMainChannel
	nrc.connInfos = connInfos
	nrc.sendChan = make(chan sendinfo, config.InternalBuffSize)

	nrc.conns = make([]*net.UDPConn, len(connInfos))
	nrc.connMap = NewUDPConnMap()
	nrc.udpMsgPool = nrc.netMainChannel.connStatus.udpMsgPool

	// one thread for reconstructing all the multi-packet messages
	nrc.packetChan = make(chan packet, config.InternalBuffSize)
	nrc.addConnInfoChannel = make(chan connInfoItem)
	nrc.removeConnInfoChannel = make(chan connInfoItem)

	for idx, connInfo := range connInfos {
		addr, err := net.ResolveUDPAddr(connInfo.Nw, connInfo.Addr)
		if err != nil {
			// should have been checked in loop above
			panic(err)
		}

		var conn *net.UDPConn
		logging.Infof("Calling UDP listen on addr %v, Idx %v", addr, idx)
		for i := 0; i < config.ConnectRetires; i++ {
			conn, err = net.ListenUDP(connInfo.Nw, addr)
			if err == nil {
				nrc.conns[idx] = conn
				break
			}
			logging.Error(err)
			// we try again in case connection is still closing
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(err)
		}
		logging.Infof("Done UDP listen on addr %v, Idx %v", addr, idx)
		nrc.sendToSelf(idx, nrc.conns[idx]) // sanity check
		nrc.runSendLoop(nrc.conns[idx])
		nrc.runReadLoop(nrc.conns[idx])
	}
	nrc.constructUDPPacketLoop()

	return nrc, nrc.connInfos, nil
}

// AddExternalNode should be called with the list of addresses that a single external node will have.
// In UDP a node might split packets over multiple connections, so this lets us know this list for each node.
func (nrc *NetPortListenerUDP) addExternalNode(conns channelinterface.NetNodeInfo) {
	nrc.sendLock.Lock()
	defer nrc.sendLock.Unlock()

	if atomic.LoadInt32(&nrc.closed) == 1 {
		return
	}

	itm := connInfoItem{conns, make(chan bool, 1)}
	nrc.addConnInfoChannel <- itm
	<-itm.retChan
}

// RemoveExternalNode should be called when we will no longer receive messages from an external node,
// it should be called with the list of addresses that a single external node will have.
func (nrc *NetPortListenerUDP) removeExternalNode(conns channelinterface.NetNodeInfo) {
	nrc.sendLock.Lock()
	defer nrc.sendLock.Unlock()

	if atomic.LoadInt32(&nrc.closed) == 1 {
		return
	}

	itm := connInfoItem{conns, make(chan bool, 1)}
	nrc.removeConnInfoChannel <- itm
	<-itm.retChan
}

// GetLocalNodeConnectionInfo returns the list of addresses that this local node is listening on.
// With UDP we may use multiple addresses so we can split sending and receiving over them.
func (nsc *NetPortListenerUDP) GetConnInfos() []channelinterface.NetConInfo {
	return nsc.connInfos
}

// send to self to be sure channel is up
func (nrc *NetPortListenerUDP) sendToSelf(connIdx int, conn *net.UDPConn) {
	var err error
	var n int
	logging.Info("Calling send to self on ", conn.LocalAddr())
	for i := 0; i < 10; i++ {
		tmpaddr := conn.LocalAddr()
		nwaddr, err := net.ResolveUDPAddr(tmpaddr.Network(), tmpaddr.String())
		addr := &net.UDPAddr{}
		addr.IP = net.ParseIP("127.0.0.1") // TODO should get exernal IP here?
		addr.Port = nwaddr.Port
		msg := []byte("test")
		conn.SetDeadline(time.Now().Add(time.Second))
		n, err = conn.WriteTo(msg, addr)
		if err != nil || n != len(msg) {
			logging.Error(err)
			time.Sleep(1 * time.Second)
			continue
		}
		buff := make([]byte, len(msg))
		n, err = conn.Read(buff)
		if err != nil || !bytes.Equal(buff, msg) {
			continue
		}
		nrc.connInfos[connIdx] = channelinterface.NetConInfoFromAddr(addr)
		logging.Info("Done send to self on ", conn.LocalAddr())
		conn.SetDeadline(time.Time{})
		return
	}
	panic(err)
}

// runSenLoop waits for messages to send, then sends them on connection
func (nrc *NetPortListenerUDP) runSendLoop(conn *net.UDPConn) {
	// connections
	connBuff := make(map[channelinterface.NetConInfo]*net.UDPAddr)

	ma := utils.NewMovingAvg(config.UDPMovingAvgWindowSize, config.MaxPacketSize) // TODO what should init value be?
	go func() {
		for {
			item, okChan := <-nrc.sendChan
			if !okChan {
				return
			}
			// send the msg
			buff := item.buff
			var addr *net.UDPAddr
			var ok bool
			if addr, ok = connBuff[item.addr]; !ok {
				var err error
				addr, err = net.ResolveUDPAddr(item.addr.Network(), item.addr.String())
				if err != nil {
					panic(err)
				}
				connBuff[item.addr] = addr
			}
			n, err := conn.WriteToUDP(buff, addr)
			if err != nil || n != len(buff) {
				// nrc.netMainChannel.BehaviorTracker.GotError(err, item.addr)
				logging.Infof("Got a write error %v, wrote %v out of %v, for %v", err, n, len(buff), item.addr)
			}

			// Dont worry about status update messages
			if len(buff) == 1 {
				continue
			}
			if config.UDPMaxSendRate > 0 { // 0 value means no sent rate limit
				avgMsgSize := ma.AddElement(len(buff))

				// the number of packets we could send to meet the max send rate
				if avgMsgSize > 0 {
					maxPack := config.UDPMaxSendRate / avgMsgSize
					if maxPack > 0 {
						sleepTime := time.Second / time.Duration(maxPack)
						time.Sleep(sleepTime)
					}
				}
			}
		}
	}()
}

// runReadLoop loops on a UDP connection, reading packets and sending them to be reconstucted (for mult-packet messages) or processed
func (nrc *NetPortListenerUDP) runReadLoop(conn *net.UDPConn) {
	// bt := nrc.netMainChannel.BehaviorTracker
	// buff := make([]byte, config.MaxTotalPacketSize)
	buff := nrc.udpMsgPool.getPacket()

	// pendingMsgs := make(map[channelinterface.NetConInfo][]byte)
	nrc.wgReadGroup.Add(1)
	go func() {
		for {
			n, addr, err := conn.ReadFrom(buff)
			if err != nil {
				if atomic.LoadInt32(&nrc.closed) == 0 {
					panic(err)
				}
				nrc.wgReadGroup.Done()
				// panic(3)
				return
			}
			nci := channelinterface.NetConInfo{
				Nw:   addr.Network(),
				Addr: addr.String()}
			// id := nci.GetID()
			/*			if bt.CheckShouldReject(nci) {
							logging.Infof("Dropping msg from %v", nci)
							delete(pendingMsgs, nci)
							panic(1)
							continue
						}
			*/
			// if it is a single byte message then it is a keep alive connection
			if n == 1 {
				// panic(1)
				netConn := nrc.getUDPConnFromMap(nci)
				var err error
				switch buff[0] {
				case 1, 2:
					err = nrc.netMainChannel.connStatus.addRecvConnection(netConn.nci.AddrList[0], netConn)
				case 0:
					err = nrc.netMainChannel.connStatus.removeRecvConnection(netConn.nci.AddrList[0])
				default:
					logging.Error("received invalid single byte UDP msg", buff[0])
				}
				if err != nil {
					logging.Info(err)
				}
				continue
			}
			// read the header
			pack, err := processUDPPacket(buff[:n], nrc.udpMsgPool)
			if err != nil {
				// panic(err)
				logging.Errorf("Error processing packet: %v", err)
				continue
			}
			switch pack.packetCount {
			case 0:
				panic("should have returned error earlier")
			case 1: // if it is a single packet message then we just process it directly
				msgbuff, err := popMsgSize(pack.buff)
				if err != nil {
					// nrc.udpMsgPool.donePacket(pack.buff)
					// panic(5)
					logging.Error(err)
					continue
				}
				// Send the message to be processeed
				// the packet will be collected later by normal GC
				src := &channelinterface.SendRecvChannel{
					MainChan:   nrc.netMainChannel,
					ReturnChan: nrc.getUDPConnFromMap(nci)}
				nrc.netMainChannel.pendingToProcess <- toProcessInfo{buff: msgbuff,
					wasEncrypted: false, encrypter: nil, retChan: src}
				// nrc.pendingToProcess <- toProcessInfo{msgbuff, nrc.getUDPConnFromMap(nci)}
			default: // it is a multi-packet message so we have to join it with the rest of the packets, this is handeled by the packet reconstruction thread.
				pack.connInfo = nci
				nrc.packetChan <- pack
				// Since the message continues to use the buffer we need to allocate a new one
				buff = nrc.udpMsgPool.getPacket()
			}
		}
	}()
}

func (nrc *NetPortListenerUDP) finishClose() {
	// Close the send loop
	close(nrc.sendChan)
}

// Close closes the UDP connections.
func (nrc *NetPortListenerUDP) prepareClose() {
	if !atomic.CompareAndSwapInt32(&nrc.closed, 0, 1) {
		// closed by someone else
		panic("should only be called once")
	}
	nrc.sendLock.Lock()
	defer nrc.sendLock.Unlock()

	for _, conn := range nrc.conns {
		err := conn.SetDeadline(time.Now())
		if err != nil {
			logging.Info(err)
		}
		err = conn.Close()
		if err != nil {
			logging.Info(err)
		}
		// nrc.closeChan <- closeType
	}

	// need all the read threads to finish before closing pendingToProcess channel and packetChan because
	// the read threads send on these channels
	nrc.wgReadGroup.Wait()

	// Now close the packetChan
	close(nrc.packetChan)
	// wait for the packet loop thread to finish
	nrc.wgConstructPacket.Wait()

}

func (nrc *NetPortListenerUDP) getUDPConnFromMap(nci channelinterface.NetConInfo) *NetConnectionUDP {
	if conn := nrc.connMap.Get(nci); conn != nil {
		return conn
	}
	// if we don't find it, we just make a new object
	// note this means that connections that aren't added through AddExternalNode will only have single connections
	return newNetConnectionUDPAddr(nci, nrc.netMainChannel)
}

// constructUDPPacketLoop is run in a single thread and is responsible for reconstructing multi-packet udp messages from
// all senders. It keeps a map of pieces of packets received so far per node.
func (nrc *NetPortListenerUDP) constructUDPPacketLoop() {

	pendingMsgs := make(map[channelinterface.NetConInfo]*packetConInfo)

	nrc.wgConstructPacket.Add(1)
	go func() {
		// addrConnMapLocal := nrc.addrConnMap.Load().(addrConnMapType)
		defer nrc.wgConstructPacket.Done()
		for {
			select {
			case toAdd := <-nrc.addConnInfoChannel:
				newUdpConn := newNetConnectionUDP(toAdd.ni, nrc.netMainChannel, false)

				pci := &packetConInfo{}
				for _, nxtConn := range toAdd.ni.AddrList {
					pendingMsgs[nxtConn] = pci
				}
				// update the conn addr map, and the pending msgs map
				nrc.connMap.AddConnection(toAdd.ni.AddrList, newUdpConn)

				if toAdd.retChan != nil {
					toAdd.retChan <- true
				}

			case toRemove := <-nrc.removeConnInfoChannel:
				// update the conn addr map, and the pending msgs map
				nrc.connMap.RemoveConnection(toRemove.ni.AddrList)
				for _, nxtConn := range toRemove.ni.AddrList {
					delete(pendingMsgs, nxtConn)
				}
				if toRemove.retChan != nil {
					toRemove.retChan <- true
				}

			case nxtPacket, ok := <-nrc.packetChan:
				if !ok { // channel has been closed
					return
				}
				var buffPackets *bufferedPackets // the buffered state for this message
				var buffPacketsIdx int           // the index of the buffered state for this connection

				pci := pendingMsgs[nxtPacket.connInfo]
				if pci == nil {
					pci = &packetConInfo{}
					pendingMsgs[nxtPacket.connInfo] = pci
				}

				// See if we already have a buffered state for this message
				for i, nxtBuffPacket := range pci.packets {
					if nxtBuffPacket != nil && nxtBuffPacket.msgCountID == nxtPacket.msgCountID {
						buffPackets = nxtBuffPacket
						buffPacketsIdx = i
						break
					}
				}
				if buffPackets == nil { // We didn't find a bufferd state, so make a new one
					buffPackets = newBufferedPackets(nxtPacket.msgCountID, nxtPacket.packetCount)
					pci.packets[pci.idx] = buffPackets // if there was an old unfinished message here, we drop it by replacing it with the new one
					buffPacketsIdx = pci.idx
					pci.idx = (pci.idx + 1) % keepPackets // increment the index
				}

				if buffPackets.addPacket(nxtPacket.buff, nxtPacket.idx) { // we got the whole message!
					pci.packets[buffPacketsIdx] = nil // gc

					buff, err := popMsgSize(buffPackets.buff)
					if err != nil {
						logging.Error(err)
						continue
					}
					// Send the message to be processeed
					src := &channelinterface.SendRecvChannel{
						MainChan:   nrc.netMainChannel,
						ReturnChan: nrc.getUDPConnFromMap(nxtPacket.connInfo)}
					nrc.netMainChannel.pendingToProcess <- toProcessInfo{buff: buff,
						wasEncrypted: false, encrypter: nil, retChan: src}
				}
				// we are done with the packet's buffer
				nrc.udpMsgPool.donePacket(nxtPacket.originBuff)
			}
		}
		// panic("should not reach here")

	}()
}
