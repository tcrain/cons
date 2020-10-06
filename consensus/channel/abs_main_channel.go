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

/*
This package contains the core network sending and receiving functionalities, see subpackage csnet for TCP and UDP implementations.
*/
package channel

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
)

// AbsMainChannel partially implements channelinterface.MainChannel
// it implements the methods that are network type agnostic
type AbsMainChannel struct {
	NumMsgProcessThreads int                                    // number of threads for processing messages
	MyConInfo            channelinterface.NetNodeInfo           // Connection info of the local node
	InternalChan         chan *channelinterface.RcvMsg          // Used to send messages from the receiving threads to the main consensus thread in Recv
	CloseChannel         chan channelinterface.ChannelCloseType // Used to perform shutdown
	Proposal             []*deserialized.DeserializedItem       // The proposal for the next consensus instance, should be nil once processes by the current instance
	SelfMessages         *[]*messages.Message                   // List of messages sent by the local node to itself, pending to be processed
	IsInInit             bool                                   // While true system is recovering, and messages should not be sent
	ConsItem             consinterface.ConsItem                 // The consensus state
	MemberCheckerState   consinterface.ConsStateInterface       // Tracking which public keys are valid for what consensus instance
	BehaviorTracker      channelinterface.BehaviorTracker       // Tracking the bad behavior of the connections
	Ticker               *time.Ticker                           // Ticker will tick at generalconfig.Timeoutrecvms, waking up the main consensus thread to be sure some action is eventually taken even when no messages are received
	Stats                stats.NwStatsInterface                 // Tracks network statistics
	// for testing
	msgDropPercent int // Will randomly drop this percentage of messages
	// rand *rand.Rand
	ReprocessCount int32    // Tracks the number of messages that failed inital deserialization becase the system state was not ready (likely from a future consensus instance), and are being reprocessed in a seperate go routine
	DoneLoop       chan int // Used to indicate when we have exited the main loop

	SelfMsgCount int32 // used atomically by threads keeping track of the number of self messages

	proposalDuringInit []*deserialized.DeserializedItem // The proposal is stored here during initalization, as during initilaization we don't resend proposal for decided indecies, until we reach one that is undecided

}

func (tp *AbsMainChannel) SetMemberCheckerState(memberCheckerState consinterface.ConsStateInterface) {
	tp.MemberCheckerState = memberCheckerState
}

// Init is called to initalize the state of the AbsMainChannelObject
func (tp *AbsMainChannel) Init(myConInfo channelinterface.NetNodeInfo, consItem consinterface.ConsItem,
	bt channelinterface.BehaviorTracker, msgDropPercent int, numMsgProcessThreads int,
	stats stats.NwStatsInterface) {

	if numMsgProcessThreads == 0 {
		panic("must have at least 1 message process thread")
	}
	tp.NumMsgProcessThreads = numMsgProcessThreads
	tp.MyConInfo = myConInfo
	tp.BehaviorTracker = bt
	tp.ConsItem = consItem
	tp.CloseChannel = make(chan channelinterface.ChannelCloseType, 1)
	tp.InternalChan = make(chan *channelinterface.RcvMsg, config.InternalBuffSize)

	slf := make([]*messages.Message, 0, 10)
	tp.SelfMessages = new([]*messages.Message)
	*tp.SelfMessages = slf
	tp.Ticker = time.NewTicker(config.Timeoutrecvms * time.Millisecond)
	tp.Stats = stats
	tp.DoneLoop = make(chan int)

	// for testing
	tp.msgDropPercent = msgDropPercent
	if tp.msgDropPercent > 0 {
		// tp.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
}

// GetStats returns the nw stats object
func (tp *AbsMainChannel) GetStats() stats.NwStatsInterface {
	return tp.Stats
}

// StartInit is called when the system is starting and is recovering from disk
// At this point any calls to Send/SendToSelf should not send any messages as the system is just replying events
func (tp *AbsMainChannel) StartInit() {
	tp.IsInInit = true
}

func (tp *AbsMainChannel) InitInProgress() bool {
	return tp.IsInInit
}

// EndInit is called once recovery is finished
func (tp *AbsMainChannel) EndInit() {
	tp.IsInInit = false
	if tp.Proposal != nil {
		panic("should not yet have a proposal")
	}

	if len(tp.proposalDuringInit) > 0 { // TODO how should this be done
		tp.Proposal = append(tp.Proposal, tp.proposalDuringInit[len(tp.proposalDuringInit)-1])
		tp.proposalDuringInit = nil
	}
}

// GetBehaviorTracker returns the BehaviorTracker object
func (tp *AbsMainChannel) GetBehaviorTracker() channelinterface.BehaviorTracker {
	return tp.BehaviorTracker
}

// HasProposal should be called by the state machine when it is ready with its proposal for the next round of consensus.
// It should be called after ProposalInfo object interface (package consinterface) method HasDecided had been called for the previous consensus instance.
func (tp *AbsMainChannel) HasProposal(msg *deserialized.DeserializedItem) {
	if tp.IsInInit {
		tp.proposalDuringInit = append(tp.proposalDuringInit, msg)
		return
	}
	if len(tp.Proposal) > 1 {
		// This can happen if a recover is received after you make a proposal
		// and before you process the proposal
		logging.Warning("Got unexpected proposal")
	}
	tp.Proposal = append(tp.Proposal, msg)
}

// ChannelTimer is returned from SendToSelf.
// The process that creates the timer is responsible for stopping it before the test returns.
type ChannelTimer struct {
	abs   *AbsMainChannel
	timer *time.Timer
}

// Stop should be called when the timer is no longer needed.
func (ct *ChannelTimer) Stop() bool {
	if ct.timer.Stop() {
		atomic.AddInt32(&ct.abs.SelfMsgCount, -1)
	}
	return true
}

// SendToSelf sends a deserialzed message to the current processes after a timeout, it returns the timer
// or nil if timeout <= 0, this method is concurrent safe.
func (tp *AbsMainChannel) SendToSelf(deser []*deserialized.DeserializedItem, timeout time.Duration) channelinterface.TimerInterface {
	if timeout > 0 {
		atomic.AddInt32(&tp.SelfMsgCount, 1)
		return &ChannelTimer{abs: tp,
			timer: time.AfterFunc(timeout, func() {
				/*			defer func() {
							if r := recover(); r != nil {
								logging.Info("recovered panic", r)
							}
						}() */
				tp.processLocalMsg(deser)
				atomic.AddInt32(&tp.SelfMsgCount, -1)
			})}
	}
	tp.processLocalMsg(deser)
	return nil
}

func (tp *AbsMainChannel) processLocalMsg(deser []*deserialized.DeserializedItem) {
	for _, di := range deser {
		ret, err := consinterface.CheckLocalDeserializedMessage(di, tp.ConsItem, tp.MemberCheckerState)
		switch err {
		// case types.ErrIndexTooOld:
		//	logging.Error(err)
		//		continue
		case types.ErrInvalidIndex, types.ErrIndexTooOld: // ok
			logging.Warning("Error from local message", err)
		case nil:
			break
		default:
			logging.Error(err)
			panic(err)
			// continue
		}
		tp.InternalChan <- &channelinterface.RcvMsg{CameFrom: 2, Msg: ret, SendRecvChan: nil, IsLocal: true}
	}
}

// SendToSelf internal can be called by the main thread (usually from within the Send function) to send a serialized message
// to the local node, it is not concurrent safe
func (tp *AbsMainChannel) SendToSelfInternal(buff []byte) {
	if tp.IsInInit {
		return
	}

	msg := messages.NewMessage(buff)
	slf := append(*tp.SelfMessages, msg)
	*tp.SelfMessages = slf
}

// ProcessMessage is called each time a message has been received from the network,
// sndRcvChan is the channel that the message was recieved from or nil
// It tries to deserialze the message then send it to the consensus state
// If deserialzation fails because it is from a future consensus instance then the message
// will be placed in the to-be-reprocesses queue
// The return values are currently unused
func (tp *AbsMainChannel) ProcessMessage(msg *messages.Message, wasEncrypted bool, encrypter sig.Pub,
	sndRcvChan *channelinterface.SendRecvChannel) (bool, []error) {
	// for testing
	if tp.msgDropPercent > 0 && rand.Intn(100) <= tp.msgDropPercent {
		// Drop the msg (for testing)
		return false, nil
	}

	encodedMsg := sig.NewEncodedMsg(msg.GetBytes(), wasEncrypted, encrypter)

	items, err := consinterface.UnwrapMessage(encodedMsg, tp.MemberCheckerState.GetConsensusIndexFuncs(),
		types.NonLocalMessage, tp.ConsItem, tp.MemberCheckerState)

	return tp.afterMessageProcess(items, err, sndRcvChan, false, tp.MemberCheckerState)
}

func (tp *AbsMainChannel) afterMessageProcess(items []*deserialized.DeserializedItem, errs []error,
	sndRcvChan *channelinterface.SendRecvChannel, isRepreocess bool, _ consinterface.ConsStateInterface) (bool, []error) {

	// logging.Infof("Error unwraping message at connection %v: %v", sndRcvChan.ConnectionInfo, err)
	for _, err := range errs {
		if err == nil {
			panic("Should not have nil errors")
		}
		if sndRcvChan != nil {
			connInfos := sndRcvChan.ReturnChan.GetConnInfos()
			if tp.BehaviorTracker.GotError(err, connInfos.AddrList[0]) {
				logging.Errorf("Closing connections from %v due to error %v", connInfos, err)
				sndRcvChan.Close(channelinterface.CloseDuringTest)
				return true, []error{err}
			}
		}
	}
	if len(items) > 0 {
		cameFrom := 1
		if isRepreocess {
			cameFrom = 11
		}
		tp.InternalChan <- &channelinterface.RcvMsg{CameFrom: cameFrom, Msg: items, SendRecvChan: sndRcvChan, IsLocal: false}
	}
	return false, errs
}

// Reprocess is called on messages that were unable to be deserialized upon first reception, it is safe to be called by many threads
func (tp *AbsMainChannel) ReprocessMessage(rcvMsg *channelinterface.RcvMsg) {
	val := atomic.AddInt32(&tp.ReprocessCount, 1)
	if val > config.MaxMsgReprocessCount { // Too many messages
		atomic.AddInt32(&tp.ReprocessCount, -1)
		return
	}
	go func() {
		for _, deser := range rcvMsg.Msg {
			var items []*deserialized.DeserializedItem
			var errors []error
			if deser.IsDeserialized {
				items = []*deserialized.DeserializedItem{deser}
			} else {
				var err error
				items, err = consinterface.FinishUnwrapMessage(deser.IsLocal, deser.Header, deser.Message,
					tp.MemberCheckerState.GetConsensusIndexFuncs(),
					tp.ConsItem, tp.MemberCheckerState)
				if err != nil {
					// items = nil
					errors = []error{err}
				}
			}
			tp.afterMessageProcess(items, errors, rcvMsg.SendRecvChan, true, nil)
		}
		atomic.AddInt32(&tp.ReprocessCount, -1)
	}()
}

// ReprocessMessageBytes is called on messages that have already been received and need to be reprocesses.
// It is safe to be called by many threads.
func (tp *AbsMainChannel) ReprocessMessageBytes(msg []byte) {
	val := atomic.AddInt32(&tp.ReprocessCount, 1)
	if val > config.MaxMsgReprocessCount { // Too many messages
		atomic.AddInt32(&tp.ReprocessCount, -1)
		return
	}
	go func() {
		defer func() {
			atomic.AddInt32(&tp.ReprocessCount, -1)
		}()
		msg := messages.NewMessage(msg)
		_, err := msg.PopMsgSize()
		if err != nil {
			logging.Warning("Error reprocessing message: ", err)
			return
		}
		deserItems, errors := consinterface.UnwrapMessage(sig.FromMessage(msg),
			types.IntIndexFuns, types.NonLocalMessage,
			tp.ConsItem, tp.MemberCheckerState)
		if errors != nil {
			// we may have errors since the members have changed
			logging.Warning("Error reprocessing messages: ", errors)
		}
		// reprocess the messages
		for _, deser := range deserItems {
			var items []*deserialized.DeserializedItem
			var errors []error
			if deser.IsDeserialized {
				items = []*deserialized.DeserializedItem{deser}
			} else {
				var err error
				items, err = consinterface.FinishUnwrapMessage(deser.IsLocal, deser.Header, deser.Message,
					tp.MemberCheckerState.GetConsensusIndexFuncs(),
					tp.ConsItem, tp.MemberCheckerState)
				if err != nil {
					// items = nil
					errors = []error{err}
				}
			}
			tp.afterMessageProcess(items, errors, nil, true, nil)
		}
	}()
}

// Recv should be called as the main consensus loop every time the node is ready to process a message.
// It is expected to be called one at a time (not concurrent safe).
// It will return utils.ErrTimeout after a timeout to ensure progress.
func (tp *AbsMainChannel) Recv() (*channelinterface.RcvMsg, error) {
	for {
		// First check if a proposal message is available
		if len(tp.Proposal) > 0 {
			// _, err := tp.Proposal.PopMsgSize()
			// if err != nil {
			// 	panic(err)
			// }
			// items, errors := consinterface.UnwrapMessage(tp.Proposal, true, tp.ConsItem, tp.MemberCheckerState)
			// if errors != nil {
			// 	panic(errors)
			// }
			logging.Info("Got a proposal message")
			nxtProposal := tp.Proposal[0]
			tp.Proposal = tp.Proposal[1:]
			ret := &channelinterface.RcvMsg{CameFrom: 3, Msg: []*deserialized.DeserializedItem{nxtProposal},
				SendRecvChan: nil, IsLocal: true}
			// tp.Proposal = nil
			return ret, nil
		}
		// Next check if any messages sent from the local node are pending
		if len(*tp.SelfMessages) > 0 {
			logging.Info("Got a message sent locally")
			msg := (*tp.SelfMessages)[0]
			sfl := (*tp.SelfMessages)[1:]
			*tp.SelfMessages = sfl
			_, err := msg.PopMsgSize()
			if err != nil {
				panic(err)
			}
			items, errors := consinterface.UnwrapMessage(sig.FromMessage(msg),
				tp.MemberCheckerState.GetConsensusIndexFuncs(), types.LocalMessage, tp.ConsItem, tp.MemberCheckerState)
			if errors != nil {
				// Local message had an error, is ok?
				logging.Info("Error processing local message: ", errors)
				if len(items) == 0 {
					continue
				}
			}
			return &channelinterface.RcvMsg{CameFrom: 4, Msg: items, SendRecvChan: nil, IsLocal: true}, nil
		}

		select {
		case <-tp.CloseChannel: // Check if it is time to exit
			// tp.ticker.Stop() // do this in child classes
			// let the channel know the main loop is exiting
			close(tp.DoneLoop)
			return nil, types.ErrClosingTime
		case msg := <-tp.InternalChan: // Check if there is a message availalbe
			return msg, nil
		case <-tp.Ticker.C: // Otherwise return after a timeout
			return &channelinterface.RcvMsg{CameFrom: 5}, types.ErrTimeout
		}
	}
}
