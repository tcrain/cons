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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/consensus/channel"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
)

// NetMainChannel stores the state of the network connections for the consensus
type NetMainChannel struct {
	channel.AbsMainChannel
	myPriv    sig.Priv                      // local node private key
	myConInfo []channelinterface.NetConInfo // List of addresses that this node will open connections on
	myPubStr  sig.PubKeyStr                 // Pub key string of this node
	// allNodesInfo    [][]channelinterface.NetConInfo // List of all nodes
	maxConnCount    int             // The total number of external nodes that should be connected to (note each node may have multiple connections)
	connStatus      *ConnStatus     // Object to track and maintains open connections to other nodes
	netPortListener NetPortListener // Object that listens for connections.
	encryptChannels bool            // if channels should be encrypted

	nodeMap        map[sig.PubKeyStr]*channelinterface.ConMapItem // List of all connection information
	nodeList       []*channelinterface.ConMapItem
	nodePubList    []sig.Pub // Same as connection list
	mutex          sync.Mutex
	processMsgLoop // for processing messages
	staticNodeList map[sig.PubKeyStr]channelinterface.NetNodeInfo
}

// NewNetMainChannel creates a net NetMainChannel object.
// Ports will be opened on the addresses of myConInfo (if they are 0, then it will bind to any availalbe port).
// connCount is the number of open connections that this node will make and maintain to external nodes for sending messages.
// msgDropPercent is the percent of received messages to drop randomly (for testing).
// It stores the connection objects for ths consensus and returns the addresses that have been bound to locally.
// Only static methods of ConsItem will be used.
func NewNetMainChannel(
	myPriv sig.Priv,
	myConInfo channelinterface.NetNodeInfo,
	maxConnCount int,
	connectionType types.NetworkProtocolType,
	// connStatus *ConnStatus,
	numMsgProcessThreads int,
	consItem consinterface.ConsItem,
	bt channelinterface.BehaviorTracker,
	encryptChannels bool,
	msgDropPercent int,
	stats stats.NwStatsInterface) (*NetMainChannel, []channelinterface.NetConInfo) {

	tp := &NetMainChannel{}
	tp.Init(myConInfo, consItem, bt, msgDropPercent, numMsgProcessThreads, stats)
	tp.myPriv = myPriv
	tp.encryptChannels = encryptChannels
	tp.nodeMap = make(map[sig.PubKeyStr]*channelinterface.ConMapItem)

	tp.connStatus = NewConnStatus(connectionType)

	tp.maxConnCount = maxConnCount

	// Start the listen thread
	var err error
	tp.netPortListener, tp.MyConInfo.AddrList, err = NewNetPortListener(myConInfo.AddrList, tp.connStatus, tp)
	if err != nil {
		panic(err)
	}
	tp.myPubStr, err = myConInfo.Pub.GetPubString()
	if err != nil {
		panic(err)
	}

	tp.initProcessMsgLoop(tp)

	// Start the thread that connects to external servers
	tp.runConnectionloop()

	return tp, tp.MyConInfo.AddrList
}

// StartMsgProcessThreads starts the threads that will process the messages.
func (tp *NetMainChannel) StartMsgProcessThreads() {
	// multiple threads for processing the reconstructed messages
	tp.runProcessMsgLoop()
}

func (tp *NetMainChannel) WaitUntilAtLeastSendCons(n int) error {
	return tp.connStatus.WaitUntilAtLeastSendCons(n)
}

// GetLocalNodeConnectionInfo returns the addresses that this node is listening on.
func (tp *NetMainChannel) GetLocalNodeConnectionInfo() channelinterface.NetNodeInfo {
	return tp.MyConInfo
}

// SendToPub sends buff to the node associated with the public key (if it exists), it returns an error if pub is not found
// in the list of connections
func (tp *NetMainChannel) SendToPub(headers []messages.MsgHeader, pub sig.Pub, countStats bool) error {
	if tp.IsInInit {
		return nil
	}

	var nwStats stats.NwStatsInterface
	if countStats {
		nwStats = tp.Stats
	}
	sndMsg, err := messages.CreateMsg(headers)
	if err != nil {
		panic(err)
	}
	buff := sndMsg.GetBytes()

	pStr, err := pub.GetPubString()
	if err != nil {
		return err
	}
	if pStr == tp.myPubStr {
		tp.SendToSelfInternal(buff)
		return nil
	}
	// Check if we are connected to this node
	tp.mutex.Lock()
	conItem, ok := tp.nodeMap[pStr]
	if !ok {
		if itm, ok := tp.staticNodeList[pStr]; ok {
			tp.addExternalNode(itm)
			conItem = tp.nodeMap[pStr]
		} else {
			tp.mutex.Unlock()
			return types.ErrConnDoesntExist
		}
	}
	if conItem.ConCount == 0 { // make the connection
		tp.makeConnection(pStr, conItem)
		// TODO how/when/where to close connections made in this way??
	}
	tp.mutex.Unlock()

	return tp.connStatus.SendToPub(buff, pub, nwStats)
}

func (tp *NetMainChannel) RemoveConnections(pubs []sig.Pub) (errs []error) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	for _, nxt := range pubs {
		pStr, err := nxt.GetPubString()
		if err != nil {
			panic(err)
		}
		if pStr == tp.myPubStr {
			continue
		}

		if item, ok := tp.nodeMap[pStr]; ok {
			tp.removeConnection(pStr, item)
		} else {
			logging.Error("unknown pub to remove connection")
			errs = append(errs, types.ErrPubNotFound)
		}
	}
	return errs
}

func (tp *NetMainChannel) removeConnection(pStr sig.PubKeyStr, conItem *channelinterface.ConMapItem) {
	if pStr == tp.myPubStr {
		return // we don't connect to ourselves
	}

	if !conItem.IsConnected && !conItem.AddressUnknown {
		if conItem.ConCount != 0 {
			panic(conItem.ConCount)
		}
		return
	}
	conItem.ConCount--
	switch {
	case conItem.ConCount > 0:
		return
	case conItem.ConCount < 0:
		panic("removed a connection multiple times")
	}

	if conItem.AddressUnknown {
		if conItem.IsConnected {
			panic(conItem.IsConnected)
		}
		return
	}

	// Now we end the connection since we have no one using it
	conItem.IsConnected = false

	err := tp.connStatus.removePendingSend(conItem.NI.Pub)
	if err != nil && err != types.ErrClosingTime {
		logging.Error(err)
	}

}

func (tp *NetMainChannel) makeConnection(pStr sig.PubKeyStr, conItem *channelinterface.ConMapItem) {
	if pStr == tp.myPubStr {
		return // we don't connect to ourselves
	}

	conItem.ConCount++
	if conItem.IsConnected {
		return
	}
	if conItem.AddressUnknown {
		return
	}

	conItem.IsConnected = true

	//tp.nodeMap[pStr] = conItem
	//tp.nodeList[conItem.Idx] = conItem
	// tp.nodePubList[conItem.Idx] = conItem.NI.Pub

	err := tp.connStatus.addPendingSendDontClose(conItem.NI)
	if err != nil && err != types.ErrClosingTime && err != types.ErrConnAlreadyExists {
		logging.Error(err)
	}
}

// SetStaticNodeList can optionally set an initial read only list of nodes in the network.
func (tp *NetMainChannel) SetStaticNodeList(staticNodeList map[sig.PubKeyStr]channelinterface.NetNodeInfo) {
	tp.staticNodeList = staticNodeList
}

// MakeConnections will connect to the nodes given by the pubs.
// This should be called after AddExternalNodes with a subset of the nodes
// added there.
func (tp *NetMainChannel) MakeConnections(pubs []sig.Pub) (errs []error) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	for _, nxt := range pubs {
		pStr, err := nxt.GetPubString()
		if err != nil {
			panic(err)
		}
		if tp.myPubStr == pStr {
			continue
		}

		if item, ok := tp.nodeMap[pStr]; ok {
			tp.makeConnection(pStr, item)
			continue
		} else if item, ok := tp.staticNodeList[pStr]; ok {
			tp.addExternalNode(item)
			tp.makeConnection(pStr, tp.nodeMap[pStr])
			continue
		} else {
			logging.Error("unknown pub to connect to")

			// We add it to the list with AddressUnknown=true, so then we will connect when we know the address
			// through call to AddExternalNode
			idx := len(tp.nodeList)
			newItm := &channelinterface.ConMapItem{NI: channelinterface.NetNodeInfo{Pub: nxt},
				Idx: idx, AddressUnknown: true}
			tp.nodeList = append(tp.nodeList, newItm)
			tp.nodePubList = append(tp.nodePubList, newItm.NI.Pub)
			tp.nodeMap[pStr] = newItm
			tp.makeConnection(pStr, newItm)

			// we return an error
			errs = append(errs, types.ErrPubNotFound)
		}
	}
	return errs
}

// SendTo sends a message on the given SendChannel.
func (tp *NetMainChannel) SendTo(buff []byte, dest channelinterface.SendChannel, countStats bool) {
	if tp.IsInInit {
		return
	}

	var nwStats stats.NwStatsInterface
	if countStats {
		nwStats = tp.Stats
	}
	tp.connStatus.SendTo(buff, dest, nwStats)
}

// ComputeDestinations returns the list of destinations given the forward filter function.
func (tp *NetMainChannel) ComputeDestinations(forwardFunc channelinterface.NewForwardFuncFilter) []channelinterface.SendChannel {
	if tp.IsInInit {
		return nil
	}
	ret, unknownPubs := tp.connStatus.ComputeDestinations(forwardFunc, tp.nodePubList)
	for len(unknownPubs) > 0 {
		tp.MakeConnections(unknownPubs) // TODO how to remove connections made here?
		ret, unknownPubs = tp.connStatus.ComputeDestinations(forwardFunc, tp.nodePubList)
	}
	return ret
}

// SendHeader serializes the header then calls Send.
func (tp *NetMainChannel) SendHeader(headers []messages.MsgHeader, isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	sndMsg, err := messages.CreateMsg(headers)
	if err != nil {
		panic(err)
	}
	tp.Send(sndMsg.GetBytes(), isProposal, toSelf, forwardChecker, countStats)
}

// Send sends a message to the outgoing connections,
// toSelf should be true if the message should also be received by the local node,
// the forwardChecker function will receive all current connections, and should return either all or a subset
// of them based on forwarding rules, the set of channels returnd will be sent the message.
// For example if it returns the input list, then the send is a broadcast to all nodes
// This method is not concurrent safe.
func (tp *NetMainChannel) Send(buff []byte,
	isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	_ = isProposal
	if tp.IsInInit {
		return
	}
	tp.sendInternal(buff, toSelf, forwardChecker, countStats)
}

// SendAlways is the same as Send, except the message will be sent even if the consensus is in initialization.
// This is just used to request the state from neighbour nodes on initialization.
func (tp *NetMainChannel) SendAlways(buff []byte, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	tp.sendInternal(buff, toSelf, forwardChecker, countStats)
}

func (tp *NetMainChannel) sendInternal(buff []byte,
	toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	if toSelf {
		// n := atomic.AddInt32(&sndtoself, 1)
		//if len(buff) < 10 {
		//	panic(buff)
		// }
		tp.SendToSelfInternal(buff)
	}
	var nwStats stats.NwStatsInterface
	if countStats {
		nwStats = tp.Stats
	}
	tp.mutex.Lock()
	unknownPubs := tp.connStatus.Send(buff, forwardChecker, tp.nodePubList, nwStats)
	tp.mutex.Unlock()
	if len(unknownPubs) > 0 {
		tp.MakeConnections(unknownPubs) // TODO how to remove connections added here
	}
}

// Close closes the channels.
// This should only be called once
func (tp *NetMainChannel) Close() {
	close(tp.CloseChannel) // tell the main loop to exit, so we won't make new messages
	_, ok := <-tp.DoneLoop // wait for the main recv loop to exit
	if ok {
		panic("should only exit on close")
	}

	go func() {
		// Do this to ensure msgs dont block on internalChan if the send loop has already exited
		// TODO better way to do this
		for {
			select {
			case _, ok := <-tp.InternalChan:
				if !ok {
					return
				}
			}
		}
	}()

	// stop the process message threads
	tp.stopProcessThreads()

	// First the rcv con because it can block on sending if we close sends first
	tp.netPortListener.prepareClose()
	// Now the send cons
	tp.connStatus.Close()

	// wait for all the reprocess msg go routines to finish
	for atomic.LoadInt32(&tp.ReprocessCount) > 0 || atomic.LoadInt32(&tp.SelfMsgCount) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// close(tp.CloseChannel)
	tp.AbsMainChannel.Ticker.Stop()
	close(tp.InternalChan)

	// close the msg process
	tp.closeProcessMsgLoop()
	tp.netPortListener.finishClose()
}

// runConnectionLoop ensures at least tp.connCount connections are open at all times.
func (tp *NetMainChannel) runConnectionloop() {
	go func() {
		for {
			err := tp.connStatus.waitUntilFewerSendCons(tp.maxConnCount)
			if err == types.ErrClosingTime {
				break
			} else if err != nil {
				// TODO here we should find new people to connect to
				logging.Error(err)
				continue
			}
			err = tp.connStatus.makeNextSendConnection(tp, tp.BehaviorTracker)
			if err == types.ErrClosingTime {
				break
			} else if err != nil {
				// TODO should look for new connections
				logging.Error(err)
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

// AddExternalNode adds the list of addresses for an external node.
// Connections and messages will then be accepted from this node.
func (tp *NetMainChannel) AddExternalNode(nodeInfo channelinterface.NetNodeInfo) {
	tp.netPortListener.addExternalNode(nodeInfo) // Add it to the port listener (so it knows those connections are valid)

	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	tp.addExternalNode(nodeInfo)
}

func (tp *NetMainChannel) addExternalNode(nodeInfo channelinterface.NetNodeInfo) {
	tp.netPortListener.addExternalNode(nodeInfo) // Add it to the port listener (so it knows those connections are valid)

	ps, err := nodeInfo.Pub.GetPubString()
	if err != nil {
		panic(err)
	}
	if tp.myPubStr == ps {
		return
	}
	if itm, ok := tp.nodeMap[ps]; ok { // if we already have the item then we must update it
		itm.NI = nodeInfo
		// newItm := &channelinterface.ConMapItem{NI: nodeInfo, Idx: itm.Idx, IsConnected: itm.IsConnected}
		// tp.nodeList[itm.Idx] = newItm
		// tp.nodePubList[  .Idx] = newItm.NI.Pub
		// tp.nodeMap[ps] = newItm
		if itm.AddressUnknown {
			itm.AddressUnknown = false
			itm.IsConnected = true
		}
		if itm.IsConnected { // update the connection
			if err := tp.connStatus.addPendingSend(nodeInfo); err != nil && err != types.ErrClosingTime {
				logging.Info(err)
			}
		}
	} else { // Otherwise add the new item
		idx := len(tp.nodeList)
		newItm := &channelinterface.ConMapItem{NI: nodeInfo, Idx: idx}
		tp.nodeList = append(tp.nodeList, newItm)
		tp.nodePubList = append(tp.nodePubList, newItm.NI.Pub)
		tp.nodeMap[ps] = newItm
	}
}

// RemoveExternalNode removes the external nodes list of addresses.
// Connections and messages will no longer be accepted from this node.
func (tp *NetMainChannel) RemoveExternalNode(nodeInfo channelinterface.NetNodeInfo) {
	// Remove from the list of connections
	tp.netPortListener.removeExternalNode(nodeInfo)

	tp.mutex.Lock()

	ps, err := nodeInfo.Pub.GetPubString()
	if err != nil {
		panic(err)
	}
	if itm, ok := tp.nodeMap[ps]; ok { // remove it from the list of connections
		switch itm.IsConnected {
		// if it is connected we disconnect it, but keep the status
		// we keep the status because the connection is needed by the consensus
		// so we will want to reconnect if we get the updated address
		case true:
			itm.IsConnected = false
			itm.AddressUnknown = true
		case false:
			for idx := itm.Idx + 1; idx < len(tp.nodeList); idx++ {
				nxtItm := tp.nodeList[idx] // get the item
				// if it is not connected we remove information about it
				nxtItm.Idx = idx - 1 // reduce its index by one
				nxtPs, err := nxtItm.NI.Pub.GetPubString()
				if err != nil {
					panic(err)
				}
				if tp.nodeMap[nxtPs] != nxtItm {
					panic("objects should be equal")
				}
				tp.nodeMap[nxtPs] = nxtItm  // update the map item
				tp.nodeList[idx-1] = nxtItm // place it at the lower index
				tp.nodePubList[idx-1] = nxtItm.NI.Pub
			}
			tp.nodeList = tp.nodeList[:len(tp.nodeList)-1]
			tp.nodePubList = tp.nodePubList[:len(tp.nodePubList)-1]
			delete(tp.nodeMap, ps)
		}
	}
	if len(tp.nodeList) != len(tp.nodeMap) {
		panic("invalid remove")
	}
	if len(tp.nodeList) != len(tp.nodePubList) {
		panic("invalid remove")
	}
	tp.mutex.Unlock()

	// Remove from the list of connected nodes (it is ok if it is not a connected node, it will just return an error)
	if err := tp.connStatus.removePendingSend(nodeInfo.Pub); err != nil {
		logging.Info(err)
	}
}

/*// CreateSendConnection adds a list of addresses for a node that the local node will try to connect to
// when the number of nodes it is connected to is less than connCount.
func (tp *NetMainChannel) CreateSendConnection(nodeInfo channelinterface.NetNodeInfo) error {
	panic("unused")
	return tp.connStatus.addPendingSend(nodeInfo)
}
*/
/*// InternalRcvMsg is called when a TCP connection receives a message, it deserializes the message and
// sends it to be processed by the consensus.
func (tp *NetMainChannel) InternalRcvMsg(buff []byte, sndRcvChan *channelinterface.SendRecvChannel) []error {
	msg := messages.NewMessage(buff)
	items, errors := consinterface.UnwrapMessage(msg, false, tp.ConsItem, tp.MemberCheckerState)
	for _, err := range errors {
		tp.BehaviorTracker.GotError(err, sndRcvChan.ReturnChan.GetConnInfos().AddrList[0])
	}
	if len(items) > 0 {
		if sndRcvChan == nil {
			panic(1)
		}
		// nrc.pendingToProcess <- toProcessInfo{buff, nrc.getUDPConnFromMap(nxtPacket.connInfo)}
		tp.InternalChan <- &channelinterface.RcvMsg{CameFrom: 6, Msg: items, SendRecvChan: sndRcvChan, IsLocal: false}
	}
	return errors
}
*/
