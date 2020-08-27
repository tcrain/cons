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
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"sync/atomic"
	"time"
)

type DualNetMainChannel struct {
	*NetMainChannel
	Secondary *NetMainChannel
}

// NewNetMainChannel creates a net NetMainChannel object.
// Ports will be opened on the addresses of myConInfo (if they are 0, then it will bind to any availalbe port).
// connCount is the number of open connections that this node will make and maintain to external nodes for sending messages.
// msgDropPercent is the percent of received messages to drop randomly (for testing).
// It stores the connection objects for ths consensus and returns the addresses that have been bound to locally.
// Only static methods of ConsItem will be used.
func NewDualNetMainChannel(
	myPriv sig.Priv,
	myConInfo []channelinterface.NetNodeInfo,
	maxConnCount int,
	connectionType types.NetworkProtocolType,
	numMsgProcessThreads int,
	consItem consinterface.ConsItem,
	bt channelinterface.BehaviorTracker,
	encryptChannels bool,
	msgDropPercent int,
	stats stats.NwStatsInterface) *DualNetMainChannel {

	if len(myConInfo) != 2 {
		panic("must have 2 NetNodeInfo objects for DualNetMainChannel")
	}

	ret := &DualNetMainChannel{}

	nmc, _ := NewNetMainChannel(myPriv, myConInfo[0],
		maxConnCount, connectionType, numMsgProcessThreads, consItem,
		bt, encryptChannels, msgDropPercent, stats)
	ret.NetMainChannel = nmc

	secondary, _ := NewNetMainChannel(myPriv, myConInfo[1],
		maxConnCount, connectionType, numMsgProcessThreads, consItem,
		bt, encryptChannels, msgDropPercent, stats)
	// the secondary shares the same message process threads
	secondary.processMsgLoop = nmc.processMsgLoop
	secondary.AbsMainChannel.SelfMessages = ret.SelfMessages
	// *secondary.AbsMainChannel.SelfMessages = append(*secondary.AbsMainChannel.SelfMessages, nil)
	ret.Secondary = secondary

	return ret
}

// SendHeader calls SendHeader on the secondary channel if the message is a proposal,
// on the main channel otherwise.
func (tp *DualNetMainChannel) SendHeader(headers []messages.MsgHeader, isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {
	switch isProposal {
	case true:
		tp.Secondary.SendHeader(headers, isProposal, toSelf, forwardChecker, countStats)
	default:
		tp.NetMainChannel.SendHeader(headers, isProposal, toSelf, forwardChecker, countStats)
	}
}

func (tp *DualNetMainChannel) SetMemberCheckerState(memberCheckerState consinterface.ConsStateInterface) {
	tp.NetMainChannel.SetMemberCheckerState(memberCheckerState)
	tp.Secondary.SetMemberCheckerState(memberCheckerState)
}

// Send calls Send on the secondary channel if the message is a proposal,
// on the main channel otherwise.
func (tp *DualNetMainChannel) Send(buff []byte,
	isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	switch isProposal {
	case true:
		tp.Secondary.Send(buff, isProposal, toSelf, forwardChecker, countStats)
	default:
		tp.NetMainChannel.Send(buff, isProposal, toSelf, forwardChecker, countStats)
	}
}

// StartMsgProcessThreads starts the threads that will process the messages.
func (tp *DualNetMainChannel) StartMsgProcessThreads() {
	tp.NetMainChannel.StartMsgProcessThreads()
	// tp.Secondary.StartMsgProcessThreads()
}

// Close closes the channels.
// This should only be called once
func (tp *DualNetMainChannel) Close() {
	for _, nxt := range []*NetMainChannel{tp.NetMainChannel, tp.Secondary} {
		close(nxt.CloseChannel) // tell the main loop to exit, so we won't make new messages
	}

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

	for _, nxt := range []*NetMainChannel{tp.NetMainChannel, tp.Secondary} {
		// First the rcv con because it can block on sending if we close sends first
		nxt.netPortListener.prepareClose()
		// Now the send cons
		nxt.connStatus.Close()

		// wait for all the reprocess msg go routines to finish
		for atomic.LoadInt32(&nxt.ReprocessCount) > 0 || atomic.LoadInt32(&nxt.SelfMsgCount) > 0 {
			time.Sleep(10 * time.Millisecond)
		}

		// close(tp.CloseChannel)
		nxt.AbsMainChannel.Ticker.Stop()
		close(nxt.InternalChan)
	}

	// close the msg process
	tp.closeProcessMsgLoop()
	for _, nxt := range []*NetMainChannel{tp.NetMainChannel, tp.Secondary} {
		nxt.netPortListener.finishClose()
	}
}

/*// Close closes the channels.
// This should only be called once
func (tp *DualNetMainChannel) Close() {
	tp.NetMainChannel.Close()
	tp.closeSecondary()
}
*/
func (tp *DualNetMainChannel) closeSecondary() {
	close(tp.Secondary.CloseChannel) // tell the main loop to exit, so we won't make new messages

	go func() {
		// Do this to ensure msgs dont block on internalChan if the send loop has already exited
		// TODO better way to do this
		for {
			select {
			case _, ok := <-tp.Secondary.InternalChan:
				if !ok {
					return
				}
			}
		}
	}()

	// First the rcv con because it can block on sending if we close sends first
	tp.Secondary.netPortListener.prepareClose()
	// Now the send cons
	tp.Secondary.connStatus.Close()

	// wait for all the reprocess msg go routines to finish
	for atomic.LoadInt32(&tp.Secondary.ReprocessCount) > 0 || atomic.LoadInt32(&tp.Secondary.SelfMsgCount) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// close(tp.CloseChannel)
	tp.Secondary.AbsMainChannel.Ticker.Stop()
	close(tp.Secondary.InternalChan)

	// close the msg process
	// tp.Secondary.closeProcessMsgLoop()
	tp.Secondary.netPortListener.finishClose()
}

// StartInit is called when the system is starting and is recovering from disk
// At this point any calls to Send/SendToSelf should not send any messages as the system is just replying events
func (tp *DualNetMainChannel) StartInit() {
	tp.NetMainChannel.StartInit()
	tp.Secondary.StartInit()
}

// EndInit is called once recovery is finished
func (tp *DualNetMainChannel) EndInit() {
	tp.NetMainChannel.EndInit()
	tp.Secondary.EndInit()
}
