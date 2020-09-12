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

package forwardchecker

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"time"
)

type forwardItem struct {
	sigItems     []*sig.SigItem
	di           *channelinterface.DeserializedItem
	receivedTime time.Time
}

// absDirectForwarder is a forwarder that forwards a message directly once
// it is processed.
type fixedBufferForwarder struct {
	internalForwardChecker
	pendingMesages map[types.HashStr]*forwardItem
	// sndRcvChan *channelinterface.SendRecvChannel
	proposalMsg        *channelinterface.DeserializedItem
	proposalSndRcvChan *channelinterface.SendRecvChannel
	gc                 *generalconfig.GeneralConfig
}

func newFixedBufferForwarder(internal internalForwardChecker, gc *generalconfig.GeneralConfig) *fixedBufferForwarder {
	return &fixedBufferForwarder{
		pendingMesages:         make(map[types.HashStr]*forwardItem),
		gc:                     gc,
		internalForwardChecker: internal}
}

// New creates a new forwarder for consensus index that forwards a message directly once
// it is processed.
func (fwd *fixedBufferForwarder) New(idx types.ConsensusIndex, participants, allPubs sig.PubList) consinterface.ForwardChecker {

	_ = allPubs
	return newFixedBufferForwarder(fwd.internalForwardChecker.newInternal(idx, participants), fwd.gc)
}

// CheckForward takes a successfully processes message and prepares it to be forwarded
// if forwarding is enabled.
// It expects GetNextForwardItem to be called before CheckForward is called again.
func (fwd *fixedBufferForwarder) CheckForward(
	sndRcvChan *channelinterface.SendRecvChannel,
	msg *channelinterface.DeserializedItem,
	shouldForward bool,
	isProposalMessage bool,
	endThreshold, maxPossible, sigCount int,
	msgID messages.MsgID,
	memberChecker *consinterface.MemCheckers) {

	_, _, _, _, _ = endThreshold, maxPossible, sigCount, msgID, memberChecker

	if isProposalMessage {
		if fwd.proposalMsg != nil || fwd.proposalSndRcvChan != nil {
			panic("should have forwarded")
		}
		if fwd.internalForwardChecker.ShouldForward(shouldForward, isProposalMessage) {
			if len(msg.Message.GetBytes()) == 0 { // sanity check
				panic("should not send nil message")
			}
			fwd.proposalMsg = msg
			fwd.proposalSndRcvChan = sndRcvChan
		}
	} else {
		w := msg.Header.(*sig.MultipleSignedMessage)

		fwdItem := fwd.pendingMesages[w.GetHashString()]
		if fwdItem == nil {
			fwdItem = &forwardItem{
				di:           msg.CopyBasic(),
				receivedTime: time.Now(),
			}
			fwdItem.di.Header = w.ShallowCopy().(*sig.MultipleSignedMessage)
			fwdItem.di.HeaderType = msg.HeaderType
			fwd.pendingMesages[w.GetHashString()] = fwdItem
		}
		fwdItem.sigItems = append(fwdItem.sigItems, w.SigItems...)
	}
}

// GetNextFowardItem returns the last message that was called with CheckForward
// if forwarding is enabled.
func (fwd *fixedBufferForwarder) GetNextForwardItem(_ stats.NwStatsInterface) (
	msg []*channelinterface.DeserializedItem,
	forwardFunc channelinterface.NewForwardFuncFilter) {

	forwardFunc = fwd.internalForwardChecker.GetNewForwardListFunc()
	if fwd.proposalMsg != nil {
		msg = []*channelinterface.DeserializedItem{fwd.proposalMsg}
		fwd.proposalMsg = nil
		fwd.proposalSndRcvChan = nil
	} else {
		for k, nxt := range fwd.pendingMesages {
			if time.Since(nxt.receivedTime) > time.Duration(fwd.gc.ForwardTimeout)*time.Millisecond {
				delete(fwd.pendingMesages, k)
				w := nxt.di.Header.(*sig.MultipleSignedMessage)
				w.SetSigItems(nxt.sigItems)
				m := messages.NewMessage(nil)
				if _, err := w.Serialize(m); err != nil {
					panic(err)
				}
				nxt.di.Message = sig.EncodedMsg{Message: m}
				msg = []*channelinterface.DeserializedItem{nxt.di}
				break
			}
		}
	}

	return
}

// ConsDecided sets all pending messages as ready to be sent.
func (fwd *fixedBufferForwarder) ConsDecided(stats.NwStatsInterface) {
	for _, nxt := range fwd.pendingMesages {
		nxt.receivedTime = nxt.receivedTime.Add(-time.Duration(fwd.gc.ForwardTimeout) * time.Millisecond)
	}

	return
}
