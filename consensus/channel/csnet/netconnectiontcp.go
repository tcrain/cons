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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channel"
	"io"
	"net"
	"sync"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// newNetConnectionTCP is called when we want to make a connection to an external node
func newNetConnectionTCP(conInfo channelinterface.NetNodeInfo, ctx context.Context,
	connStatus *ConnStatus, netMainChannel *NetMainChannel) (*NetConnectionTCP, error) {

	nsc := &NetConnectionTCP{}
	nsc.mutex.Lock()
	defer nsc.mutex.Unlock()

	if netMainChannel.encryptChannels {
		nsc.encrypter = channel.GenerateEncrypter(netMainChannel.myPriv, false, conInfo.Pub)
	}
	// nsc.myConInfo = myConInfo
	nsc.connStatus = connStatus
	nsc.netMainChannel = netMainChannel
	nsc.nci = conInfo.AddrList
	nsc.pub = conInfo.Pub

	if len(nsc.nci) != 1 {
		return nil, fmt.Errorf("must have 1 conn info for TCP")
	}

	nsc.sendChan = make(chan []byte, config.InternalBuffSize)
	nsc.conns = make([]net.Conn, len(nsc.nci))
	err := nsc.connectSend(ctx)

	return nsc, err
}

// newNetConnectionTCPAlreadyConnected is called when an external node has connected to us
func newNetConnectionTCPAlreadyConnected(conn net.Conn, connStatus *ConnStatus, netMainChannel *NetMainChannel) error {
	nsc := &NetConnectionTCP{}

	nsc.cond = sync.NewCond(&nsc.mutex)

	if netMainChannel.encryptChannels {
		nsc.encrypter = channel.GenerateEncrypter(netMainChannel.myPriv, true, nil)
	}
	// we need the lock because after connStatus.addRecvConnection is called successfully,
	// it may call Close, so we need to be all set up before this
	nsc.mutex.Lock()
	defer nsc.mutex.Unlock()

	addr := conn.RemoteAddr()
	connInfo := channelinterface.NetConInfo{
		Addr: addr.String(),
		Nw:   addr.Network()}
	_, err := net.ResolveUDPAddr(connInfo.Nw, connInfo.Addr)
	if err == nil {
		panic("should not receive udp connections")
	}

	if netMainChannel.BehaviorTracker.CheckShouldReject(connInfo) {
		err = conn.Close()
		if err != nil {
			logging.Info(err)
		}
		return fmt.Errorf("not accepting connections from %v", connInfo)
	}
	nsc.nci = []channelinterface.NetConInfo{connInfo}
	nsc.sendChan = make(chan []byte, config.InternalBuffSize)
	nsc.netMainChannel = netMainChannel
	nsc.connStatus = connStatus
	nsc.isRecvConn = true
	nsc.conns = []net.Conn{conn}

	err = connStatus.addRecvConnection(connInfo, nsc)
	if err != nil {
		// if there was an error nsc is not used, so we just close the connection
		// directly and let nsc be garbage collected
		err2 := conn.Close()
		if err2 != nil {
			logging.Info(err2)
		}
		return err
	}

	// we need to receive return messages
	nsc.wgTCPConn.Add(2)
	go nsc.loopSend(conn, connInfo)
	go nsc.loopRecv(conn, connInfo)

	// nsc.sendRecvChan = &channelinterface.SendRecvChannel{
	// 	ReturnChan: nsc,
	// 	ConnectionInfo: nsc.nci}
	return nil
}

// NetConnection represents a connection to an external node
// TCP connections are maintained here, constatus is updated when an error happens on an existing connection
// UDP connections just store the address (since it is connectionless), the operations just call the connections NetPortListenerUDP (TODO cleanup)
type NetConnectionTCP struct {
	nci            []channelinterface.NetConInfo // the address of the connection
	isRecvConn     bool                          // this is true when the connection came from an external node
	connStatus     *ConnStatus                   // pointer to ConnStatus object
	conns          []net.Conn                    // the connection
	sendChan       chan []byte                   // to send to the send loop
	netMainChannel *NetMainChannel               // pointer to main chan
	closed         bool                          // set to true when closed is called, read/updated under the lock
	mutex          sync.Mutex                    // concurrency control
	cond           *sync.Cond                    // for when we are an encrypted receive connection waiting for someone to connect
	wgTCPConn      sync.WaitGroup                // when closing wait for all the threads to finish
	pub            sig.Pub                       // the public key if a send connection
	encrypter      channel.EncryptInterface
}

// GetType returns channelinterface.TCP
func (nsc *NetConnectionTCP) GetType() types.NetworkProtocolType {
	return types.TCP
}

// GetLocalNodeConnectionInfo returns the addresses for this connection
func (nsc *NetConnectionTCP) GetConnInfos() channelinterface.NetNodeInfo {
	return channelinterface.NetNodeInfo{AddrList: nsc.nci,
		Pub: nsc.pub}
}

// loopRecv is an internal function used by TCP that reads from the connection, and sends
// the data to NetMainChannel.InternalRcvMsg
func (nsc *NetConnectionTCP) loopRecv(conn net.Conn, connInfo channelinterface.NetConInfo) {
	sendRecvChan := &channelinterface.SendRecvChannel{
		MainChan:   nsc.netMainChannel,
		ReturnChan: nsc}

	var err error
	defer func() {
		nsc.netMainChannel.BehaviorTracker.GotError(err, connInfo)
		logging.Warningf("Closing (is recv connection %v) recv channel %v due to error %v", nsc.isRecvConn, connInfo, err)

		nsc.wgTCPConn.Done()
		err2 := nsc.Close(channelinterface.CloseDuringTest)
		if err2 != nil {
			logging.Warning(err2)
		}
	}()

	if nsc.isRecvConn && nsc.encrypter != nil {
		// The public key will always be the first message
		pubBuff, err := readFromConn(conn)
		if err != nil {
			nsc.encrypter.FailedGetPub()
			return
		}
		// Then the random bytes
		randBuff, err := readFromConn(conn)
		if err != nil {
			nsc.encrypter.FailedGetPub()
			return
		}
		err = nsc.encrypter.GotPub(pubBuff, randBuff)
		if err != nil {
			return
		}
	}

	for {
		var buff []byte
		buff, err = readFromConn(conn)
		if err != nil {
			return
		}

		if nsc.encrypter != nil {
			buff, err = nsc.encrypter.Decode(buff, false)
			if err != nil {
				return
			}
			size := encoding.Uint32(buff)
			buff = buff[4:]
			if size != uint32(len(buff)) {
				err = types.ErrInvalidMsgSize
				return
			}
			nsc.netMainChannel.pendingToProcess <- toProcessInfo{
				buff:         buff,
				retChan:      sendRecvChan,
				encrypter:    nsc.encrypter.GetExternalPub(),
				wasEncrypted: true,
			}
		} else {
			// TODO clean this up
			nsc.netMainChannel.pendingToProcess <- toProcessInfo{
				buff:    buff,
				retChan: sendRecvChan,
			}
		}
	}
}

// loopSend is an internal function for TCP connections that reads from an internal channel, and sends the data
// over the connection.
func (nsc *NetConnectionTCP) loopSend(conn net.Conn, connInfo channelinterface.NetConInfo) {
	// cis := []channelinterface.NetConInfo{connInfo}

	var err error
	var n int

	defer func() {
		nsc.wgTCPConn.Done()
		if err != nil {
			nsc.netMainChannel.BehaviorTracker.GotError(err, connInfo)
			logging.Warning("Got a write error %v, wrote %v, for %v", err, n, connInfo)
			err2 := nsc.Close(channelinterface.CloseDuringTest)
			if err2 != nil {
				logging.Warning(err2)
			}
		}
	}()

	if nsc.encrypter != nil {
		if nsc.isRecvConn {
			// wait until we receive the key
			if err = nsc.encrypter.WaitForPub(); err != nil {
				return
			}
		} else {
			// first we send our pub
			msg := messages.CreateSingleMsgBytes(messages.SeralizeBuff(nsc.encrypter.GetMyPubBytes()))
			if err = sendBytes(conn, msg); err != nil {
				return
			}
			// now the random bytes
			rndBytes := nsc.encrypter.GetRandomBytes()
			msg = messages.CreateSingleMsgBytes(messages.SeralizeBuff(rndBytes[:]))
			if err = sendBytes(conn, msg); err != nil {
				return
			}
		}
	}
	for {
		buff, ok := <-nsc.sendChan
		if !ok {
			// closed
			// nsc.wgTCPConn.Done()
			return
		}
		// check if we should encrypt the message
		if nsc.encrypter != nil {
			buff = nsc.encrypter.Encode(buff, true)
		}

		n, err = conn.Write(buff)
		if err != nil {
			return
		}
		if n != len(buff) {
			err = types.ErrInvalidMsgSize
			return
		}
	}

}

func sendBytes(conn net.Conn, buff []byte) error {
	n, err := conn.Write(buff)
	if err != nil {
		return err
	}
	if n != len(buff) {
		err = types.ErrInvalidMsgSize
		return err
	}
	return nil
}

// connectSend is an internal function for TCP that tries to connect to the external node
// it start the internal read and send loops
func (nsc *NetConnectionTCP) connectSend(ctx context.Context) error {
	// UDP conns use the main listen channel
	if nsc.isRecvConn {
		panic("rcv conn sets up connections in NewNetSendConAlreadyConnected")
	}
	for i := 0; i < len(nsc.nci); i++ {
		// nsc.wgTCPConn.Add(2) // one thread loops recv, one loops send
		go func(idx int) {
			logging.Infof("Connecting to %v", nsc.nci)
			nsc.mutex.Lock()
			// defer nsc.mutex.Unlock()

			if nsc.closed {
				// nsc.wgTCPConn.Add(-2) // we didn't start the threads so -2
				nsc.mutex.Unlock()

				return
			}
			nsc.mutex.Unlock()

			// we do this outside of the lock because someone might want to remove the connection in the meantime
			var d net.Dialer
			ctx, _ = context.WithTimeout(ctx, config.TCPDialTimeout*time.Millisecond)
			conn, err := d.DialContext(ctx, nsc.nci[idx].Nw, nsc.nci[idx].Addr)
			// conn, err := net.DialTimeout(nsc.nci[idx].Nw, nsc.nci[idx].Addr, config.TCPDialTimeout*time.Millisecond)
			nsc.mutex.Lock()
			if nsc.closed {
				if err == nil {
					_ = conn.Close()
				}
				nsc.mutex.Unlock()
				return
			}

			nsc.conns[idx] = conn

			if err != nil {
				logging.Warning("Unable to connect to %v, err %v", nsc.nci, err)
				// nsc.wgTCPConn.Add(-2) // we didn't start the threads so -2
				// nsc.mutex.Unlock()

				err2 := nsc.closeInternal(channelinterface.CloseDuringTest)
				if err2 != nil {
					logging.Warning(err2)
				}
				nsc.mutex.Unlock()
				return
			}
			nsc.connStatus.FinishedMakingConnection(nsc.pub)
			logging.Infof("Connected to %v", nsc.nci)
			// loop the recv thread
			nsc.wgTCPConn.Add(2)
			go nsc.loopRecv(conn, nsc.nci[idx])
			// the loop send runs in this thread
			go nsc.loopSend(conn, nsc.nci[idx])
			nsc.mutex.Unlock()
		}(i)
	}

	return nil
}

// Close shuts down the connection
func (nsc *NetConnectionTCP) Close(closeType channelinterface.ChannelCloseType) error {
	nsc.mutex.Lock()
	defer nsc.mutex.Unlock()

	return nsc.closeInternal(closeType)
}

func (nsc *NetConnectionTCP) closeInternal(closeType channelinterface.ChannelCloseType) error {

	// This might be called multiple times
	if nsc.closed {
		return nil
	}
	nsc.closed = true
	// First remove it from the connstatus object so we don't send on this channel anymore
	// If it was closed by a fault, we must remove it
	// Otherwise it was closed by connStatus directly
	if closeType == channelinterface.CloseDuringTest {
		var err error
		if nsc.isRecvConn {
			err = nsc.connStatus.removeRecvConnection(nsc.nci[0], &nsc.wgTCPConn)
		} else {
			err = nsc.connStatus.removeSendConnection(nsc.pub, &nsc.wgTCPConn)
		}
		if err != nil {
			// We might get an error if the connection was removed all together from the pending connections list
			// i.e. if a node is no longer needed
			logging.Info("Got err when removing send connection:", err)
		}
	}

	for _, conn := range nsc.conns {
		if conn == nil {
			continue
		}
		err := conn.Close()
		if err != nil {
			logging.Info(err)
		}
		if c, typ := conn.(*net.TCPConn); typ {
			//c.CloseWrite()
			//c.CloseRead()
			//c.SetLinger(0)
			err = c.SetDeadline(time.Now())
			if err != nil {
				logging.Info(err)
			}
		} else {
			panic(typ)
		}
	}

	close(nsc.sendChan) // closing the chan causes select to return

	if closeType == channelinterface.EndTestClose { // we only wait for the threads to ext on final close
		// wait for the internal loops to exit
		nsc.wgTCPConn.Wait()
	}

	return nil
}

// Send sends a byte slice to the node addressed by this connection
func (nsc *NetConnectionTCP) Send(buff []byte) (err error) {

	//cpy := make([]byte, len(buff))
	//copy(cpy, buff)

	nsc.sendChan <- buff
	// nsc.sendChan <- cpy
	return nil
}

// readFromConn reads a message from a TCP connection
func readFromConn(conn net.Conn) ([]byte, error) {
	sizeBuff := make([]byte, messages.GetMsgSizeLen())
	_, err := io.ReadFull(conn, sizeBuff)
	if err != nil {
		return nil, err
	}
	v := encoding.Uint32(sizeBuff)
	if v > config.MaxMsgSize {
		return nil, types.ErrInvalidMsgSize
	}
	msgBuff := make([]byte, v)
	_, err = io.ReadFull(conn, msgBuff)
	if err != nil {
		return nil, err
	}
	return msgBuff, nil
}
