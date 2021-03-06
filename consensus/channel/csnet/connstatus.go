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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"sync"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
)

type connDetailsConnected struct {
	channelinterface.ConnDetails
	isConnected bool
}

// Connstats tracks and maintains open connections to other nodes
// Each node may use multiple addresses, in this case the addresses are stored as a list, where the first element of the list
// is the address that identifies the node.
// The method AddPendingSend creates a new NetSendCon object, in TCP these objects maintain connections,
// in udp they just keep the address, and use MainChannel.NetRecvConn to actually send the messages.
// In TCP whenever there is a error with a connection it is closed and the connection count is reduced.
// In UDP, given that it is connectionless, the connection count is always the number of addresses added.
// The connection list is updated in the following ways (TODO clean this up):
// - When a connection is made from this node to an external node (TCP or UDP) through this object sendCons is updated
// - When NetPortListenerTCP received a connection from an external node, recvCons is updated (recvCons is allways empty for UDP)
// - When a NetConnection connection has an error (TCP only), the connection will be removed from either sendCons or recvCons
type ConnStatus struct {
	sendCons                []channelinterface.SendChannel                 // List of connections for sending messages to (created from this local node)
	cons                    map[sig.PubKeyStr]connDetailsConnected         // Map from a connected node's pub key to its addresses (for sendCons)
	consPendingReconnection map[sig.PubKeyStr]bool                         // Nodes that have been disconnected, but will be started as soon as a timer runs out
	removedMap              map[sig.PubKeyStr]channelinterface.NetNodeInfo // Map from a non-connected node's public key to its addresses
	// List of removed nodes addresses (should correspond to the values of removedMap). Theses nodes will be retired to connect in order of the list.
	removed               []channelinterface.NetNodeInfo
	activeSendConnections int // number of send connections successfully made

	// Map of connections from external nodes that will send messages to this node (i.e. opposite of sendCons)
	recvCons map[channelinterface.NetConInfo]*rcvConTime

	closeChan     chan channelinterface.ChannelCloseType // channel used to stop the state and shutdown all connections
	cond          *sync.Cond                             // for concurrency control
	mutex         sync.RWMutex                           // **
	isClosed      bool                                   // used during closing
	udpMsgCountID uint64                                 // we give each udp packet an incremented id
	nwType        types.NetworkProtocolType              // TCP or UDP

	ctx       context.Context      // context used to terminate pending TCP dial connections
	ctxCancel context.CancelFunc   // cancel function
	wgChan    chan *sync.WaitGroup // for waiting for threads to close
	myWg      sync.WaitGroup

	udpMsgPool *udpMsgPool // used for allocating buffers to UDP connections
}

type rcvConTime struct { // Information for when a connection made by an external node was established.
	con     channelinterface.SendChannel
	rcvTime time.Time
}

// NewConnStatus creates and initialized a ConnStatus object
func NewConnStatus(nwType types.NetworkProtocolType) *ConnStatus {
	cs := &ConnStatus{}
	cs.nwType = nwType
	cs.cond = sync.NewCond(&cs.mutex)
	cs.closeChan = make(chan channelinterface.ChannelCloseType, 1)
	cs.cons = make(map[sig.PubKeyStr]connDetailsConnected)
	cs.consPendingReconnection = make(map[sig.PubKeyStr]bool)
	cs.recvCons = make(map[channelinterface.NetConInfo]*rcvConTime)
	cs.removedMap = make(map[sig.PubKeyStr]channelinterface.NetNodeInfo)
	cs.udpMsgPool = newUdpMsgPool()
	cs.ctx, cs.ctxCancel = context.WithCancel(context.Background())

	if cs.nwType == types.UDP {
		go cs.checkKeepAliveLoop()
	}
	cs.runWaitThread()

	return cs
}

func (cs *ConnStatus) runWaitThread() {
	cs.wgChan = make(chan *sync.WaitGroup, 10)
	cs.myWg.Add(1)
	ctx, _ := context.WithCancel(cs.ctx)
	doneChan := ctx.Done()
	go func() {
		for true {
			select {
			case wg := <-cs.wgChan:
				wg.Wait()
			case <-doneChan:
				cs.myWg.Done()
				return
			}
		}
	}()
}

// Close unblocks anyone waiting on WaitUntilFewerSendCons and closes any connections.
func (cs *ConnStatus) Close() {
	// This should only be called once
	cs.mutex.Lock()
	if cs.isClosed {
		panic("should not be called multiple times")
	}
	cs.isClosed = true
	cs.mutex.Unlock()

	cs.ctxCancel()      // Unblock TCP connections in progress
	close(cs.closeChan) // This will force the select to return in WaitUntilFewerSendCons
	cs.cond.Broadcast()

	// Close all our connections
	for _, nsc := range cs.recvCons {
		err := nsc.con.Close(channelinterface.EndTestClose)
		if err != nil {
			logging.Error(err)
		}
	}
	for _, nsc := range cs.sendCons {
		err := nsc.Close(channelinterface.EndTestClose)
		if err != nil {
			logging.Error(err)
		}
	}
	// wait for our wait group
	cs.mutex.Lock()
	cs.myWg.Wait()
	cs.mutex.Unlock()
}

////////////////////////////////////////////////////////////////////////////////////////////////
// Public methods to add or remove connections
/////////////////////////////////////////////////////////////////////////////////////////////////

// addPendingSendDontClose adds a connection to the pending list, returns an error if the connection is already pending.
// It does not close the connection if it already exists
func (cs *ConnStatus) addPendingSendDontClose(conInfo channelinterface.NetNodeInfo) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}

	ret := cs.internalAddPendingSend(conInfo)
	cs.cond.Broadcast()
	return ret
}

// addPendingSend adds a connection to the pending list.
// It disconnects the connection and adds it to the list to be reconnected if it exists.
func (cs *ConnStatus) addPendingSend(conInfo channelinterface.NetNodeInfo) error {
	// First we remove the connection in case it exists
	err := cs.removePendingSend(conInfo.Pub)
	if err != nil {
		logging.Info(err)
	}

	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}

	ret := cs.internalAddPendingSend(conInfo)
	cs.cond.Broadcast()
	return ret
}

// RemovePendingSend removes a connection to the pending list, returns an error if the connection is already pending
// Should be call to permanately remove a connection.
func (cs *ConnStatus) removePendingSend(pub sig.Pub) error {
	cs.mutex.Lock()
	if cs.isClosed {
		cs.mutex.Unlock()
		return types.ErrClosingTime
	}

	conn, err := cs.removeSendConnectionInternal(pub, false)
	cs.mutex.Unlock() // unlock first because closing the connection may take the lock

	if err != nil {
		logging.Info(err)
	}
	if conn != nil {
		err = conn.Close(channelinterface.CloseDuringTest)
	}

	// cs.cond.Broadcast()
	return err
}

// removeRecvConnection removes the connection from the list of recv connections.
func (cs *ConnStatus) removeRecvConnection(conInfo channelinterface.NetConInfo, wg *sync.WaitGroup) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}
	cs.wgChan <- wg

	if _, ok := cs.recvCons[conInfo]; !ok {
		return types.ErrConnDoesntExist
	}
	delete(cs.recvCons, conInfo)
	return nil
}

// addRecvConnection adds the connection to the list of received connections
func (cs *ConnStatus) addRecvConnection(conInfo channelinterface.NetConInfo, conn channelinterface.SendChannel) (err error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}

	if cs.nwType == types.TCP {
		if _, ok := cs.recvCons[conInfo]; ok {
			return types.ErrConnAlreadyExists
		}
		// with UDP even if it exists, we update it with the new connection since it is just the addresses
	}
	cs.recvCons[conInfo] = &rcvConTime{con: conn, rcvTime: time.Now()}
	return nil
}

////////////////////////////////////////////////////////
// Checking connection state
////////////////////////////////////////////////////////

// SendConnCount returns the sum of connections and pending connections origination from this node
func (cs *ConnStatus) SendConnCount() int {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return 0
	}

	return len(cs.cons) + len(cs.removed) + len(cs.consPendingReconnection)
}

// RecvConnCount returns the sum of connections and pending connections originating from external nodes
func (cs *ConnStatus) RecvConnCount() int {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return 0
	}

	return len(cs.recvCons)
}

// ComputeDestinations returns the list of destinations given the forward filter function.
// If unconnected pubs are found to send to, then it returns them as unknown pubs, and a nil destList.
func (cs *ConnStatus) ComputeDestinations(forwardFunc channelinterface.NewForwardFuncFilter,
	allPubs []sig.Pub) (destList []channelinterface.SendChannel, unknownPubs []sig.Pub) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return
	}

	dests, sendToRecv, sendToSend, sendRange := forwardFunc(allPubs)
	// Need to make a copy of the list since forwardFunc may return the cs.sendCons list directly, which may be modified by ConnStatus
	listSize := len(dests)
	if sendToRecv {
		listSize += len(cs.recvCons)
	}
	if sendToSend {
		listSize += len(cs.sendCons)
	}
	destList = make([]channelinterface.SendChannel, 0, listSize)
	for _, p := range dests {
		pStr, err := p.GetPubString()
		if err != nil {
			panic(err)
		}
		if con, ok := cs.cons[pStr]; ok {
			destList = append(destList, con.Conn)
		} else {
			if !cs.checkConPending(pStr) {
				unknownPubs = append(unknownPubs, p)
			}
		}
	}
	if len(unknownPubs) > 0 {
		return nil, unknownPubs
	}

	if sendToRecv {
		for _, con := range cs.recvCons {
			destList = append(destList, con.con)
		}
	}
	if sendToSend {
		for _, con := range cs.sendCons {
			destList = append(destList, con)
		}
	}
	// take the send range
	if sendRange != channelinterface.FullSendRange {
		startIdx, endIdx := sendRange.ComputeIndicies(len(destList))
		destList = destList[startIdx:endIdx]
	}

	return
}

// WaitUntilFewerSendCons blocks until there are at least n connections, or an error
func (cs *ConnStatus) WaitUntilAtLeastSendCons(n int) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for cs.activeSendConnections < n {
		if cs.isClosed {
			return types.ErrClosingTime
		}
		cs.cond.Wait()
	}
	if cs.isClosed {
		return types.ErrClosingTime
	}
	return nil
}

// WaitUntilFewerSendCons blocks until there are less than n connections, or an error
func (cs *ConnStatus) waitUntilFewerSendCons(n int) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for len(cs.cons) >= n || len(cs.removed) == 0 {
		select {
		case <-cs.closeChan:
			return types.ErrClosingTime
		default:
			cs.cond.Wait()
		}
	}
	select {
	case <-cs.closeChan:
		return types.ErrClosingTime
	default:
		return nil
	}
}

///////////////////////////////////////////////////
// Public sending fucntions
///////////////////////////////////////////////////

// SendTo sends the byte slice to the destination channel.
// TODO should try to connect to on connection not existing?
func (cs *ConnStatus) SendTo(buff []byte, dest channelinterface.SendChannel, stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface) {
	buff = utils.CopyBuf(buff)
	if config.LatencySend > 0 {
		time.AfterFunc(time.Millisecond*time.Duration(rand.Intn(config.LatencySend)), func() {
			cs.internalSendTo(buff, dest, stats, consStats)
		})
	} else {
		cs.internalSendTo(buff, dest, stats, consStats)
	}
}

// SendToPub sends buff to the node associated with the public key (if it exists), it returns an error if pub is not found
// in the list of connections.
// TODO should try to connect to on connection not existing?
func (cs *ConnStatus) SendToPub(buff []byte, pub sig.Pub, stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface) error {

	pStr, err := pub.GetPubString()
	if err != nil {
		return err
	}
	cs.mutex.Lock()
	if cs.isClosed {
		cs.mutex.Unlock()
		return types.ErrClosingTime
	}
	conn, ok := cs.cons[pStr]
	cs.mutex.Unlock()
	if ok {
		cs.SendTo(buff, conn.Conn, stats, consStats)
		return nil
	}
	return types.ErrPubNotFound
}

// SendToPubList sends buf to the list of pub keys if connections to them exist
func (cs *ConnStatus) SendToPubList(buf []byte, pubList []sig.PubKeyStr, stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface) (errs []error) {
	var destList []channelinterface.SendChannel

	buf = utils.CopyBuf(buf)

	cs.mutex.Lock()
	if cs.isClosed {
		cs.mutex.Unlock()
		return []error{types.ErrClosingTime}
	}

	for _, pStr := range pubList {
		if nxt, ok := cs.cons[pStr]; ok {
			destList = append(destList, nxt.Conn)
		} else {
			errs = append(errs, types.ErrConnDoesntExist)
		}
	}

	cs.internalSendFunc(buf, destList, false, false,
		channelinterface.FullSendRange, stats, consStats, false)
	cs.mutex.Unlock()
	return errs
}

// Send sends a message,
// the forwardChecker function will receive all current connections, and should return either all or a subset
// of them based on forwarding rules.
// It returns any destinations that are not known, but were returned by the forward checker.
// This is called by NetMainChannel.Send
func (cs *ConnStatus) Send(buff []byte,
	forwardChecker channelinterface.NewForwardFuncFilter,
	allPubs []sig.Pub,
	stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface) []sig.Pub {

	// we copy the buf because
	buff = utils.CopyBuf(buff)
	cs.mutex.Lock()
	if cs.isClosed {
		cs.mutex.Unlock()
		return nil
	}

	destList, sendToRcv, sendToSend, sendRange := forwardChecker(allPubs)
	dests := make([]channelinterface.SendChannel, 0, len(destList))
	var unknownPubs []sig.Pub
	for _, p := range destList {
		pStr, err := p.GetPubString()
		if err != nil {
			panic(err)
		}

		if con, ok := cs.cons[pStr]; ok {
			dests = append(dests, con.Conn)
		} else {
			if !cs.checkConPending(pStr) {
				unknownPubs = append(unknownPubs, p)
			}
		}
	}

	if config.LatencySend > 0 {
		cs.mutex.Unlock()
		// the func must take the lock itself later
		time.AfterFunc(time.Millisecond*time.Duration(rand.Intn(config.LatencySend)), func() {
			cs.internalSendFunc(buff, dests, sendToRcv, sendToSend, sendRange, stats, consStats, true)
		})
	} else {
		cs.internalSendFunc(buff, dests, sendToRcv, sendToSend, sendRange, stats, consStats, false)
		cs.mutex.Unlock()
	}
	return unknownPubs
}

////////////////////////////////////////////////////////////////////////////////////////////////
// Internal methods should only be called withing the package
/////////////////////////////////////////////////////////////////////////////////////////////////

// internalSendTo sends buf to dest
func (cs *ConnStatus) internalSendTo(buff []byte, dest channelinterface.SendChannel, stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface) {

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isClosed {
		return
	}

	if dest.GetType() == types.UDP { // udp we send directly
		sendUDP(buff, cs.udpMsgPool, []channelinterface.SendChannel{dest}, nil, nil,
			&cs.udpMsgCountID, channelinterface.FullSendRange, stats, consStats)
		return
	}

	// TCP we check if we have the connection
	// First check if it is a send connection
	if pub := dest.GetConnInfos().Pub; pub != nil {
		pstr, err := pub.GetPubString()
		if err != nil {
			panic(err)
		}
		if conn, ok := cs.cons[pstr]; ok { // otherwise for tcp we have to check if we are still connected to the address
			if stats != nil {
				stats.Send(len(buff))
			}
			if consStats != nil {
				consStats.ConsSend(len(buff))
			}
			err := conn.Conn.Send(buff)
			if err != nil {
				logging.Error(err)
			}
			return
		}
	}

	// Next check if it is a received connection
	if conn, ok := cs.recvCons[dest.GetConnInfos().AddrList[0]]; ok {
		if stats != nil {
			stats.Send(len(buff))
		}
		if consStats != nil {
			consStats.ConsSend(len(buff))
		}
		err := conn.con.Send(buff)
		if err != nil {
			logging.Error(err)
		}
		return
	}
	logging.Warning("Tried to send on a closed channel", dest.GetConnInfos().AddrList[0])
}

// internalSendFunc sends buff to the given destinations.
func (cs *ConnStatus) internalSendFunc(buff []byte,
	destList []channelinterface.SendChannel,
	sendToRcv,
	sendToSend bool,
	sendRange channelinterface.SendRange,
	stats stats.NwStatsInterface,
	consStats stats.ConsNwStatsInterface,
	shouldLock bool) {

	// check we actually have send destinations
	if len(destList) == 0 {
		if (!sendToSend || len(cs.sendCons) == 0) && (!sendToRcv || len(cs.recvCons) == 0) {
			return
		}
	}

	if shouldLock {
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

	}

	if cs.isClosed {
		return
	}

	if cs.nwType == types.UDP {
		var conMap map[channelinterface.NetConInfo]*rcvConTime
		if sendToRcv {
			conMap = cs.recvCons
		}
		var sendCons []channelinterface.SendChannel
		if sendToSend {
			sendCons = cs.sendCons
		}
		sendUDP(buff, cs.udpMsgPool, destList, sendCons, conMap, &cs.udpMsgCountID, sendRange, stats, consStats)
	} else {
		var numSends int
		if sendRange != channelinterface.FullSendRange && len(destList) > 0 {
			startIdx, endIdx := sendRange.ComputeIndicies(len(destList))
			destList = destList[startIdx:endIdx]
		}
		for _, conn := range destList {
			numSends++
			err := conn.Send(buff)
			if err != nil {
				logging.Error(err)
			}
		}
		if sendToRcv {
			stopIdx := len(cs.recvCons)
			if sendRange != channelinterface.FullSendRange {
				startIdx, endIdx := sendRange.ComputeIndicies(len(cs.recvCons))
				stopIdx = endIdx - startIdx
			}
			var i int
			for _, conn := range cs.recvCons {
				if i == stopIdx {
					break
				}
				numSends++
				err := conn.con.Send(buff)
				if err != nil {
					logging.Error(err)
				}
			}
		}
		if sendToSend {
			sndCons := cs.sendCons
			if sendRange != channelinterface.FullSendRange && len(sndCons) > 0 {
				startIdx, endIdx := sendRange.ComputeIndicies(len(sndCons))
				sndCons = sndCons[startIdx:endIdx]
			}
			for _, conn := range sndCons {
				numSends++
				err := conn.Send(buff)
				if err != nil {
					logging.Error(err)
				}
			}
		}

		l := len(buff)
		if stats != nil {
			stats.Broadcast(l, numSends)
		}
		if consStats != nil {
			consStats.ConsBroadcast(l, numSends)
		}

	}
}

func (cs *ConnStatus) internalAddPendingSend(conInfo channelinterface.NetNodeInfo) error {
	pubStr, err := conInfo.Pub.GetPubString()
	if err != nil {
		panic(err)
	}
	_, ok1 := cs.removedMap[pubStr]
	_, ok2 := cs.cons[pubStr]
	if ok1 || ok2 {
		return types.ErrConnAlreadyExists
	}
	cs.removedMap[pubStr] = conInfo
	cs.removed = append(cs.removed, conInfo)
	return nil
}

// MakeNextSendConnection connects and adds the input to the list of send connections
func (cs *ConnStatus) makeNextSendConnection(netMainChannel *NetMainChannel, bt channelinterface.BehaviorTracker) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}

	l := len(cs.removed)
	for i := 0; i < l; i++ {
		item := cs.removed[0]
		pubStr, err := item.Pub.GetPubString()
		if err != nil {
			panic(err)
		}
		delete(cs.removedMap, pubStr)
		cs.removed = cs.removed[1:]

		if bt.CheckShouldReject(item.AddrList[0]) {
			// Dont connect, just add it to the back of the list
			err := cs.internalAddPendingSend(item)
			if err != nil {
				panic(err)
			}
			logging.Infof("Rejecting connection to %v", item)
			continue
		}

		logging.Info("Connecting to", item)
		nsc, err := NewNetSendConnection(item, cs.ctx, cs, netMainChannel)
		if err != nil {
			logging.Error("Invalid conn item ", item)
			continue
		}
		cs.addSendConnection(item, nsc)
		// nsc.ConnectSend()

		cs.cond.Broadcast()
		return nil
	}
	return types.ErrNoRemovedCons
}

func (cs *ConnStatus) addSendConnection(conInfo channelinterface.NetNodeInfo, conn channelinterface.SendChannel) {
	pubStr, err := conInfo.Pub.GetPubString()
	if err != nil {
		panic(err)
	}
	if _, ok := cs.cons[pubStr]; !ok {
		if cs.nwType == types.UDP {
			cs.activeSendConnections++
			cs.cond.Broadcast()
		}
		cs.cons[pubStr] = connDetailsConnected{
			isConnected: cs.nwType == types.UDP,
			ConnDetails: channelinterface.ConnDetails{Addresses: conInfo, Conn: conn}}

		cs.sendCons = append(cs.sendCons, conn)
	} else {
		panic(types.ErrConnAlreadyExists)
	}
}

func (cs *ConnStatus) checkConPending(pStr sig.PubKeyStr) bool {
	if _, ok := cs.removedMap[pStr]; ok {
		return true
	}
	if _, ok := cs.consPendingReconnection[pStr]; ok {
		return true
	}
	return false
}

func (cs *ConnStatus) FinishedMakingConnection(pub sig.Pub) {
	pubStr, err := pub.GetPubString()
	utils.PanicNonNil(err)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if itm, ok := cs.cons[pubStr]; ok {
		if itm.isConnected {
			panic("tried to set connected twice")
		}
		cs.activeSendConnections++
		cs.cond.Broadcast()
	}
}

func (cs *ConnStatus) removeSendConnectionInternal(pub sig.Pub, allowReconnection bool) (conn channelinterface.SendChannel, err error) {
	pubStr, perr := pub.GetPubString()
	if perr != nil {
		return nil, perr
	}
	isPending := cs.checkConPending(pubStr)
	if !allowReconnection {
		delete(cs.consPendingReconnection, pubStr)
	}
	if infos, ok := cs.cons[pubStr]; ok {
		if infos.isConnected {
			cs.activeSendConnections--
		}
		delete(cs.cons, pubStr)
		if allowReconnection {
			// we add the connection back to the list after a second so we can reconnect
			cs.consPendingReconnection[pubStr] = true
			time.AfterFunc(config.RetryConnectionTimeout*time.Millisecond, func() {
				cs.mutex.Lock()
				defer cs.mutex.Unlock()
				if cs.isClosed {
					return
				}
				if _, ok := cs.consPendingReconnection[pubStr]; !ok {
					// we are no longer pending so forget it
					return
				}
				if _, ok := cs.removedMap[pubStr]; ok {
					// we were added to the list to be reconnected already
					return
				}
				if _, ok := cs.cons[pubStr]; ok {
					// we were already reconnected
					return
				}

				cs.removed = append(cs.removed, infos.Addresses)
				cs.removedMap[pubStr] = infos.Addresses
				cs.cond.Broadcast()
			})
		}

		for i, nxt := range cs.sendCons {
			nxtPubStr, perr := nxt.GetConnInfos().Pub.GetPubString()
			if perr != nil {
				panic(perr)
			}
			if nxtPubStr == pubStr {
				// replace the connection with the one at the end of the list
				l := len(cs.sendCons)
				// go cs.sendCons[i].Close(channelinterface.EndTestClose)
				cs.sendCons[i] = cs.sendCons[l-1]
				cs.sendCons[l-1] = nil
				cs.sendCons = cs.sendCons[:l-1]
				cs.cond.Broadcast()
				conn = nxt
				return
			}
		}
		panic("Didnt find con in list")
	}
	if !isPending {
		err = types.ErrConnDoesntExist
	}
	return
}

// removeSendConnection should be called when a connection to conInfo fails or is closed.
// The connection will be added to the list to be reconnected to.
func (cs *ConnStatus) removeSendConnection(pub sig.Pub, wg *sync.WaitGroup) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.isClosed {
		return types.ErrClosingTime
	}
	cs.wgChan <- wg

	_, err := cs.removeSendConnectionInternal(pub, true)
	return err
}

////////////////////////////////////////////////////////////////////////////////
// Threads
////////////////////////////////////////////////////////////////////////////////

// checkKeepAliveLoop checks if UDP connections from external nodes have sent heartbeats and if not after
// config.RcvConUDPTimeout then the connection is removed from the list of recvCons
func (cs *ConnStatus) checkKeepAliveLoop() {
	ticker := time.NewTicker(config.RcvConUDPTimeout * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			cs.mutex.Lock()
			if cs.isClosed {
				cs.mutex.Unlock()
				return
			}
			for k, c := range cs.recvCons {
				if time.Since(c.rcvTime) > config.RcvConUDPTimeout*time.Millisecond {
					logging.Error("ending UDP rcv conn because have not heard from it")
					_ = c.con.Close(channelinterface.CloseDuringTest)
					delete(cs.recvCons, k)
				}
			}
			cs.mutex.Unlock()
		case <-cs.closeChan:
			ticker.Stop()
			return
		}
	}
}
