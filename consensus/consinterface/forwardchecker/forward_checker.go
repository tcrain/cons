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
ForwardCheckers keep track of successfully processes consensus messages and decide if they should be forwarded on the network or not.
*/
package forwardchecker

import (
	"fmt"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/utils"
)

// internalForwardChecker contains some additional helper functions for the forward checker.
type internalForwardChecker interface {
	consinterface.AdditionalForwardCheckerOps
	// getSpecificForwardListFunc is used by the buffer forwarder. Count is the number of times the function has been called
	// for the specific message ID that is being sent.
	// RandomForwarder just returns the normal random function.
	// P2P forwarder loops through the P2P networks created by the option testoptions.AdditionalP2PNetworks, by the given count.
	getSpecificForwardListFunc(count int) channelinterface.NewForwardFuncFilter
	// newInternal creates a new internalForwardChecker for the consensus index.
	newInternal(idx types.ConsensusIndex, pubs sig.PubList) internalForwardChecker
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// buffMsg tracks a set of messaged to be forwarded, based on their messages.MsgID.
// (i.e. there will be one of these object per unique messages.MsgID)
type buffMsg struct {
	msgID messages.MsgID                       // the msgID
	msgs  []*channelinterface.DeserializedItem // the set of messages

	count         int       // number of signatures the message has
	sendTime      time.Time // last time the message was send
	forwardCount  int       // number of times the message has been forwarded with at least end count messages
	previousCount int       // number of signatures there were the last time it was sent
	totalForwards int       // total number of times this message has been forwarded

	nextThreshold    int // next forward threshold
	initialThreshold int // initial forward threshold
	endThreshold     int // threshold needed to be done with the message
	maxPossible      int // max possible number
}

// newBufferForwarder creates a new absBufferForwarder using the internalForwardChecker
func newBufferForwarder(internal internalForwardChecker) *absBufferForwarder {
	return &absBufferForwarder{
		internalForwardChecker: internal,
		buffMsgMap:             make(map[messages.MsgID]*buffMsg)}
}

// absBufferForwarder contains the message tracking for forwards that only forward a message once
// they have recieved a threshold of messages with the same MsgID.
type absBufferForwarder struct {
	idx types.ConsensusIndex
	internalForwardChecker
	buffMsgMap  map[messages.MsgID]*buffMsg
	buffMsgList []*buffMsg
}

// New creates a new buffer ForwardChecker for the consensus index. It will be always be called on an "initialForwardChecker" that is given as input to
// MemberCheckerState.Init. A buffer forwarder forwards a message once it has recieved a threshold of messages with the same MsgID.
func (fwd *absBufferForwarder) New(idx types.ConsensusIndex, participants, allPubs sig.PubList) consinterface.ForwardChecker {
	f := newBufferForwarder(fwd.internalForwardChecker.newInternal(idx, participants))
	f.idx = idx
	return f
}

// CheckForward is called each time a message is successfully processes by ConsState, sndRcvChan is the channel who sent the message, shouldForward is given by the consensus
// object after the message is processesed, ConsItem, memberChecker, and messageState all correspond to the consensus instance this message is from.
// It keeps track of the messages received so far based on messages.MsgID.
func (fwd *absBufferForwarder) CheckForward(
	sndRcvChan *channelinterface.SendRecvChannel,
	msg *channelinterface.DeserializedItem,
	shouldForward bool,
	isProposalMessage bool,
	endThreshold, maxPossible, sigCount int,
	msgID messages.MsgID,
	memberChecker *consinterface.MemCheckers) {

	if msg.IsLocal == types.LocalMessage {
		panic(msg.Header)
	}

	switch w := msg.Header.(type) {
	case *sig.MultipleSignedMessage: // we only forward signed consensus messags

		minThreshold := utils.Min(endThreshold, fwd.internalForwardChecker.GetFanOut())

		var bMsg *buffMsg
		var ok bool
		// check if this MsgID has been received already
		if bMsg, ok = fwd.buffMsgMap[msgID]; !ok {
			bMsg = &buffMsg{
				msgID:            msgID,
				nextThreshold:    minThreshold,
				initialThreshold: minThreshold,
				endThreshold:     endThreshold,
				maxPossible:      maxPossible,
				sendTime:         time.Now()}
			fwd.buffMsgMap[msgID] = bMsg
			fwd.buffMsgList = append(fwd.buffMsgList, bMsg)
		}
		var found bool
		// check if you already received this message
		// TODO maybe do this check in a map for speed?
		for _, oldMsg := range bMsg.msgs {
			if oldW, ok := oldMsg.Header.(*sig.MultipleSignedMessage); ok {
				if w.GetHashString() == oldW.GetHashString() {
					found = true
				}
			}
		}
		if !found {
			// add the new message to the list of received
			bMsg.msgs = append(bMsg.msgs, msg)
		}
		// Add to the count based on how many signatures this message has
		// bMsg.count = messageState.GetSigCountMsgID(w.GetMsgID())
		bMsg.count = sigCount

	default:
		// we should only get signed messages here
		panic(w)
	}
}

// GetNextForwardItem returns the next message to be forwarded.
// Messages are forwareded when they have reached a threshold based on their result of ContItem.GetBufferCount, or if they have not reached
// the next threshold after a timeout. It returns nil for msgs if there are no messages ready to be forwarded.
func (fwd *absBufferForwarder) GetNextForwardItem(stats stats.NwStatsInterface) (
	msgs []*channelinterface.DeserializedItem,
	forwardFunc channelinterface.NewForwardFuncFilter) {

	for _, nxt := range fwd.buffMsgList {
		if nxt == nil {
			continue
		}

		if nxt.endThreshold < 1 {
			// messages with endThreshold < 1 are only sent in their inital broadcast and not forwarded
			// this means that message with endthreshold = 1 should be globally broadcast at the beginning
			// TODO maybe do this differently
			panic(1)
			continue
		}

		// We only forward the message if we have reached a threshold, or if a timeout has passed without reaching the threshold
		passedTimeout := time.Since(nxt.sendTime) > config.ForwardTimeout*time.Millisecond
		if passedTimeout {
			stats.BufferForwardTimeout()
		}
		if (nxt.count >= nxt.nextThreshold && nxt.count > nxt.previousCount) || passedTimeout {

			var info strings.Builder
			info.WriteString(fmt.Sprintf("MsgID: %v, ", nxt.msgID))
			info.WriteString(fmt.Sprintf("Passed thrsh: %v >= %v, Got new: %v > %v, Passed timeout: %v, ",
				nxt.count, nxt.nextThreshold, nxt.count, nxt.previousCount, passedTimeout))

			nxt.sendTime = time.Now()
			nxt.previousCount = nxt.count

			if nxt.forwardCount >= 1 {
				// we have already forwarded this message after it reached its end threshold, so we are done with it
				continue
			}

			if nxt.previousCount >= nxt.endThreshold {
				// we have reached the end threshold
				nxt.forwardCount++
			}

			info.WriteString(fmt.Sprintf("Pass endThresh: %v >= %v, ", nxt.previousCount, nxt.endThreshold))

			// If we passed the current forward threshold then we go to the next
			if nxt.count >= nxt.nextThreshold {
				info.WriteString(fmt.Sprintf("PassedNxtThresh: %v >= %v, ", nxt.count, nxt.nextThreshold))
				if nxt.nextThreshold >= nxt.endThreshold || nxt.count >= nxt.endThreshold {

					info.WriteString(fmt.Sprintf(
						"PassedEndThrsh: %v/%v >= %v, ", nxt.nextThreshold, nxt.count, nxt.endThreshold))
					// if we passed the end threshold, then our next threshold is just the max possible
					nxt.nextThreshold = nxt.maxPossible
				} else {
					// otherwise we incrament the threshold by the a multiple of the fan out
					// TODO make this configurable???
					nxt.nextThreshold = utils.Min(nxt.nextThreshold*fwd.internalForwardChecker.GetFanOut(), nxt.endThreshold)
					//nxt.nextThreshold = utils.Min(nxt.nextThreshold*2-1, nxt.endThreshold)
				}
				info.WriteString(fmt.Sprintf("Set NxtThrsh: %v, ", nxt.nextThreshold))
			}

			// sendCount := fwd.internalForwardChecker.GetFanOut()

			// if nxt.endThreshold == 1 {
			// 	// TODO what should this be?
			// 	sendCount = sendCount * 2
			// 	// panic(1)
			// }
			nxt.totalForwards++

			info.WriteString(fmt.Sprintf("TotalForwards: %v\n", nxt.totalForwards))
			// fmt.Println(info.String())

			// get the forward function for the index
			// TODO may want to send to different number of nodes for different ways of propogation
			forwardFunc = fwd.internalForwardChecker.getSpecificForwardListFunc(nxt.totalForwards)
			// set the return value to be this message
			msgs = nxt.msgs
			return
		}
	}
	return
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// absDirectForwarder is a forwarder that forwards a message directly once
// it is processed.
type absDirectForwarder struct {
	internalForwardChecker
	msg        *channelinterface.DeserializedItem
	sndRcvChan *channelinterface.SendRecvChannel
}

func newDirectForwarder(internal internalForwardChecker) *absDirectForwarder {
	return &absDirectForwarder{
		internalForwardChecker: internal}
}

// New creates a new forwarder for consensus index that forwards a message directly once
// it is processed.
func (fwd *absDirectForwarder) New(idx types.ConsensusIndex, participants, allPubs sig.PubList) consinterface.ForwardChecker {
	return newDirectForwarder(fwd.internalForwardChecker.newInternal(idx, participants))
}

// CheckForward takes a successfully processes message and prepares it to be forwarded
// if forwarding is enabled.
// It expects GetNextForwardItem to be called before CheckForward is called again.
func (fwd *absDirectForwarder) CheckForward(
	sndRcvChan *channelinterface.SendRecvChannel,
	msg *channelinterface.DeserializedItem,
	shouldForward bool,
	isProposalMessage bool,
	endThreshold, maxPossible, sigCount int,
	msgID messages.MsgID,
	memberChecker *consinterface.MemCheckers) {

	if fwd.msg != nil || fwd.sndRcvChan != nil {
		panic("should have forwarded")
	}
	if fwd.internalForwardChecker.ShouldForward(shouldForward, isProposalMessage) {
		if len(msg.Message.GetBytes()) == 0 { // sanity check
			panic("should not send nil message")
		}
		fwd.msg = msg
		fwd.sndRcvChan = sndRcvChan
	}
}

// GetNextFowardItem returns the last message that was called with CheckForward
// if forwarding is enabled.
func (fwd *absDirectForwarder) GetNextForwardItem(stats stats.NwStatsInterface) (
	msg []*channelinterface.DeserializedItem,
	forwardFunc channelinterface.NewForwardFuncFilter) {

	if fwd.msg != nil {
		msg = []*channelinterface.DeserializedItem{fwd.msg}
	}
	fwd.msg = nil
	fwd.sndRcvChan = nil

	forwardFunc = fwd.internalForwardChecker.GetNewForwardListFunc()

	return
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// NewAllToAllForwarder creates a new all to all forwarder, these forwarders are for
// all to all networks and do no forwarding.
func NewAllToAllForwarder() consinterface.ForwardChecker {
	return &AllToAllForwarder{}
}

// AllToAllForwarder is the forwarder used for all to all network patterns,
// no messages are forwarded since it is not needed for all to all.
type AllToAllForwarder struct {
	pubs sig.PubList
}

// New creates a new AllToAllForwarder for the consensus index. It will be always be called on an "initialForwardChecker" that is given as input to
// MemberCheckerState.Init.
func (fwd *AllToAllForwarder) New(idx types.ConsensusIndex, participants, allPubs sig.PubList) consinterface.ForwardChecker {
	ret := NewAllToAllForwarder().(*AllToAllForwarder)
	ret.pubs = participants
	return ret
}

func (fwd *AllToAllForwarder) NewForwardFunc() sig.PubList {
	return fwd.pubs
}

func (fwd *AllToAllForwarder) newInternal(idx types.ConsensusID) internalForwardChecker {
	panic("not used")
}

// CheckForward does nothing for the AllToAllForwarder.
func (fwd *AllToAllForwarder) CheckForward(
	sndRcvChan *channelinterface.SendRecvChannel,
	msg *channelinterface.DeserializedItem,
	shouldForward bool,
	isProposalMessage bool,
	endThreshold, maxPossible, sigCount int,
	msgID messages.MsgID,
	memberChecker *consinterface.MemCheckers) {

	// no forwarding in all to all
}

// GetNextForwardItem always returns a nil result for the AllToAllForwarder.
func (fwd *AllToAllForwarder) GetNextForwardItem(stats stats.NwStatsInterface) (
	msg []*channelinterface.DeserializedItem,
	forwardFunc channelinterface.NewForwardFuncFilter) {

	// no forwarding in all to all
	return
}

// ShouldForward always returns false for the AllToAll forwarder.
func (fwd *AllToAllForwarder) ShouldForward(progress bool, isProposalMessage bool) bool {
	// no forwarding in all to all
	return false
}

func (fwd *AllToAllForwarder) GetFanOut() int {
	// no forwarding in all to all
	return 0
}

func (fwd *AllToAllForwarder) getSpecificForwardListFunc(count int) channelinterface.NewForwardFuncFilter {
	return channelinterface.ForwardAllPub
}

// GetForwardList is called before a message is broadcast, it takes as input the list of all nodes in the system.
// It returns the list of public keys to which the message should be sent.
func (fwd *AllToAllForwarder) GetNewForwardListFunc() channelinterface.NewForwardFuncFilter {
	return channelinterface.ForwardAllPub
}

// GetHalfHalfForwardListFunc is the same as GetNewForwardListFunc except is returns functions that broadcast
// to half of the participants.
func (fwd *AllToAllForwarder) GetHalfHalfForwardListFunc() (firstHalf, secondHalf channelinterface.NewForwardFuncFilter) {
	return channelinterface.ForwardAllPubFrontHalf, channelinterface.ForwardAllPubBackHalf
}

// GetNoProgressForwardFunc is called before a no progress message update is sent.
func (fwd *AllToAllForwarder) GetNoProgressForwardFunc() channelinterface.NewForwardFuncFilter {
	return channelinterface.ForwardAllPub // TODO send to a fraction for scalability?
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// Creates a new peer to peer forwarder.
// If bufferForwarder is true, then it buffers messages before forwarding them as described in
// the buffer forwarder functions.
// If requestForwarder is true then this is being used for a local random member checker.
func NewP2PForwarder(requestForwarder bool, fanOut int, bufferForwarder bool, forwardPubs []sig.PubList) consinterface.ForwardChecker {
	if requestForwarder && bufferForwarder {
		panic("can't have both requestForwarder and bufferForwarder")
	}
	p2p := newP2PForwarder(requestForwarder, fanOut, bufferForwarder, forwardPubs)
	switch bufferForwarder {
	case true:
		return newBufferForwarder(p2p)
	default:
		return newDirectForwarder(p2p)
	}
}

// P2PForwarder is used for p2p network types, where node is connected to only fanOut neighbours.
// Messages will always be forwarded to these negibors.
// TODO when using P2PForwarder + BufferForwarder, having messages with endthreshold <= 1, will not
// be sent to everyone, because the inital broadcast will only be sent to the neighbours, then noone will forward it (see BufferForwarder).
type P2PForwarder struct {
	forwardPubs      []sig.PubList
	requestForwarder bool
	fanOut           int
	bufferFowarder   bool
	fwdFuncs         []channelinterface.NewForwardFuncFilter
	// participants     sig.PubList
}

func newP2PForwarder(requestForwarder bool, fanOut int, bufferForwarder bool, forwardPubs []sig.PubList) *P2PForwarder {
	var forwardFuncs []channelinterface.NewForwardFuncFilter
	if bufferForwarder {
		if len(forwardPubs) == 0 {
			panic("must have forward pubs for buffer forwarder")
		}
		forwardFuncs = make([]channelinterface.NewForwardFuncFilter, len(forwardPubs))
		// we have a forward function for each set of forwardPubs
		for i, nxt := range forwardPubs {
			forwardFuncs[i] = func(allConnections []sig.Pub) (destinations sig.PubList,
				sendToRecv, sendToSend bool, sendRange channelinterface.SendRange) {

				sendRange = channelinterface.FullSendRange
				destinations = nxt
				return
			}
		}
	}

	return &P2PForwarder{requestForwarder: requestForwarder, fanOut: fanOut, bufferFowarder: bufferForwarder,
		forwardPubs: forwardPubs, fwdFuncs: forwardFuncs}
}

func (fwd *P2PForwarder) newInternal(idx types.ConsensusIndex, participants sig.PubList) internalForwardChecker {
	return &P2PForwarder{requestForwarder: fwd.requestForwarder, fanOut: fwd.fanOut, forwardPubs: fwd.forwardPubs,
		bufferFowarder: fwd.bufferFowarder, fwdFuncs: fwd.fwdFuncs} //, participants: participants}
}

// GetFanOut returns the number of neighbors the node is connected to.
func (fwd *P2PForwarder) GetFanOut() int {
	return fwd.fanOut
}

// ShouldForward returns the same value as progress (i.e. only forward a message if it made progress towards a consensus decision.
func (fwd *P2PForwarder) ShouldForward(progress bool, isProposalMessage bool) bool {
	if fwd.requestForwarder { // requestForwarder we only forward proposal messages
		return progress && isProposalMessage
	}
	return progress
}

func (fwd *P2PForwarder) getSpecificForwardListFunc(count int) channelinterface.NewForwardFuncFilter {
	return fwd.fwdFuncs[count%len(fwd.fwdFuncs)]
}

// GetForwardList is called before a message is broadcast, it takes as input the list of all nodes in the system.
// It returns the list of public keys to which the message should be sent.
func (fwd *P2PForwarder) GetNewForwardListFunc() channelinterface.NewForwardFuncFilter {
	if fwd.bufferFowarder {
		return fwd.fwdFuncs[0]
	}
	return channelinterface.BaseP2pForward
}

// GetHalfHalfForwardListFunc is the same as GetNewForwardListFunc except is returns functions that broadcast
// to half of the participants.
func (fwd *P2PForwarder) GetHalfHalfForwardListFunc() (firstHalf, secondHalf channelinterface.NewForwardFuncFilter) {
	if fwd.bufferFowarder {
		return func(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
				sendRange channelinterface.SendRange) {
				return fwd.forwardPubs[0], false, false, channelinterface.FrontHalfSendRange
			}, func(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
				sendRange channelinterface.SendRange) {
				return fwd.forwardPubs[0], false, false, channelinterface.FrontHalfSendRange
			}
	}

	return channelinterface.BaseP2pForwardFrontHalf, channelinterface.BaseP2pForwardBackHalf
}

// GetNoProgressForwardFunc is called before a no progress message update is sent.
func (fwd *P2PForwarder) GetNoProgressForwardFunc() channelinterface.NewForwardFuncFilter {
	return channelinterface.BaseP2pNoProgress
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

var rndFwdSeed int64 // for testing

// RandomForwarder will forward messages to a random set of nodes.
type RandomForwarder struct {
	fanOut                            int
	rand                              *rand.Rand
	fwdFunc                           channelinterface.NewForwardFuncFilter
	fwdFuncFrontHalf, fwdFuncBackHalf channelinterface.NewForwardFuncFilter
	participants                      sig.PubList
}

// NewRandomForwarder creates a new random forwarder.
// If bufferForwarder is true, then it buffers messages before forwarding them as described in
// the buffer forwarder functions.
func NewRandomForwarder(bufferForwarder bool, fanOut int) consinterface.ForwardChecker {
	randlocal := rand.New(rand.NewSource(atomic.AddInt64(&rndFwdSeed, 1)))
	switch bufferForwarder {
	case true:
		return newBufferForwarder(&RandomForwarder{
			fanOut: fanOut,
			rand:   randlocal})
	default:
		return &absDirectForwarder{
			internalForwardChecker: &RandomForwarder{
				fanOut: fanOut,
				rand:   randlocal}}
	}
}

func (fwd *RandomForwarder) newInternal(idx types.ConsensusIndex, participants sig.PubList) internalForwardChecker {
	return &RandomForwarder{
		participants: participants,
		fanOut:       fwd.fanOut,
		rand:         fwd.rand}
}

// GetFanOut returns the number of nodes messages are forwarded to.
func (fwd *RandomForwarder) GetFanOut() int {
	return fwd.fanOut
}

// GetForwardList is called before a message is broadcast, it takes as input the list of all nodes in the system.
// It returns the list of public keys to which the message should be sent.
func (fwd *RandomForwarder) GetNewForwardListFunc() channelinterface.NewForwardFuncFilter {
	if fwd.fwdFunc == nil {
		fwd.fwdFunc = fwd.genFwdFunc(channelinterface.FullSendRange)
	}
	return fwd.fwdFunc
}

func (fwd *RandomForwarder) genFwdFunc(sndRng channelinterface.SendRange) channelinterface.NewForwardFuncFilter {
	return func(allPubs []sig.Pub) (pubs sig.PubList, sendToRecv, sendToSend bool,
		sendRange channelinterface.SendRange) {

		perm := utils.GenRandPerm(fwd.fanOut, len(allPubs), fwd.rand)
		ret := make([]sig.Pub, fwd.fanOut)
		for i, v := range perm {
			ret[i] = allPubs[v]
		}
		return ret, false, false, sndRng
	}
}

// GetHalfHalfForwardListFunc is the same as GetNewForwardListFunc except is returns functions that broadcast
// to half of the participants.
func (fwd *RandomForwarder) GetHalfHalfForwardListFunc() (firstHalf, secondHalf channelinterface.NewForwardFuncFilter) {
	if fwd.fwdFuncBackHalf == nil || fwd.fwdFuncFrontHalf == nil {
		fwd.fwdFuncFrontHalf = fwd.genFwdFunc(channelinterface.FrontHalfSendRange)
		fwd.fwdFuncBackHalf = fwd.genFwdFunc(channelinterface.BackHalfSendRange)
	}
	return fwd.fwdFuncFrontHalf, fwd.fwdFuncFrontHalf
}

// ShouldForward returns the same value as progress (i.e. only forward a message if it made progress towards a consensus decision.
func (fwd *RandomForwarder) ShouldForward(progress bool, isProposalMessage bool) bool {
	return progress
}

// GetNoProgressForwardFunc is called before a no progress message update is sent.
func (fwd *RandomForwarder) GetNoProgressForwardFunc() channelinterface.NewForwardFuncFilter {
	return fwd.GetNewForwardListFunc()
}

// TODO use semi-random here based on a configuration
func (fwd *RandomForwarder) getSpecificForwardListFunc(count int) channelinterface.NewForwardFuncFilter {
	return fwd.GetNewForwardListFunc()
	/*	if fwd.fwdFunc == nil {
			// Here the forward function returns a random permutation of the channels of lenght count
			fwd.fwdFunc = func(count int, sendChans []channelinterface.SendChannel) []channelinterface.SendChannel {
				if count >= len(sendChans) {
					return sendChans
				}
				perm := fwd.rand.Perm(len(sendChans))
				toSend := make([]channelinterface.SendChannel, count)
				for i := 0; i < count; i++ {
					toSend[i] = sendChans[perm[i]]
				}
				return toSend
			}
		}
		return func(sendChans []channelinterface.SendChannel) []channelinterface.SendChannel {
			return fwd.fwdFunc(count, sendChans)
		}
	*/
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////
