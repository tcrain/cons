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

package cons

import (
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"sort"
)

// checkForward is called after a message is processed, it checks if a message should be fowarded
// using forwardChecker.CheckForward(), this queues the message to be forwarded if needed,
// it is then acutally forwarded in the call to sendForward.
func checkForward(sndRcvChan *channelinterface.SendRecvChannel,
	msg *channelinterface.DeserializedItem,
	shouldForward bool,
	item consinterface.ConsItem,
	memberCheckerState consinterface.ConsStateInterface,
	mainChannel channelinterface.MainChannel) {

	memberChecker, messageState, forwardChecker, err := memberCheckerState.GetMemberChecker(msg.Index)
	if err != nil {
		panic("invalid idx")
	}

	switch w := msg.Header.(type) {
	case *sig.MultipleSignedMessage: // we only forward signed consensus messags

		if _, ok := w.InternalSignedMsgHeader.(*messagetypes.CombinedMessage); ok {
			logging.Info("Combined messages are not forwarded since they are forwarded as partials")
		} else {
			// Check how many times this message should be received before being forwarded
			endThreshold, maxPossible, msgID, err := item.GetBufferCount(msg.Header,
				memberCheckerState.GetGeneralConfig(), memberChecker)
			if err != nil || endThreshold == 0 {
				logging.Infof("Not forwarding msg type: %s, err: %v, endThresh %v", msg.Header.GetID(), err, endThreshold)
			} else {
				// count how many signatures this message has
				sigCount := messageState.GetSigCountMsgID(w.GetMsgID())
				forwardChecker.CheckForward(sndRcvChan, msg, shouldForward,
					messages.IsProposalHeader(w.Index, w.InternalSignedMsgHeader), endThreshold,
					maxPossible, sigCount, msgID, memberChecker)
			}
		}
	default:
		// dont forward unsigned messages
		logging.Infof("Not forwarding msg type: %v", msg.HeaderType)
	}

	sendForward(msg.Index, memberCheckerState, mainChannel)
}

// sendForward loops through the messages to be forwarded for the consensus index, sending them, as given by
// forwardChecker.GetNextForwardItem() until no messages remain to be forwarded.
func sendForward(idx types.ConsensusIndex, memberCheckerState consinterface.ConsStateInterface,
	mainChannel channelinterface.MainChannel) (sent bool) {

	mc, msgState, forwardChecker, err := memberCheckerState.GetMemberChecker(idx)
	if err != nil {
		panic("invalid idx")
	}
	// mainChannel.GetStats()
	// get the next item to be fowraded
	msgs, forwardFunc := forwardChecker.GetNextForwardItem(mainChannel.GetStats())
	// loop until no items remain
	for ; msgs != nil; msgs, forwardFunc = forwardChecker.GetNextForwardItem(mainChannel.GetStats()) {
		for _, msg := range msgs {
			if msg.Index.Index != idx.Index {
				panic("should have same index")
			}
			item, err := memberCheckerState.GetConsItem(msg.Index)
			if err != nil {
				panic("err")
			}
			toSend, err := messages.CreateMsg(item.GetPreHeader())
			if err != nil {
				panic(err)
			}
			gc := memberCheckerState.GetGeneralConfig()
			if gc.IncludeCurrentSigs { // We include all sigs we have seen so far
				switch w := msg.Header.(type) {
				case *sig.MultipleSignedMessage:
					sigNum, _, _, err := item.GetBufferCount(w, gc, mc)
					if err != nil || sigNum < 1 {
						panic(fmt.Sprint("should have already filtered invalid forward messages", err, sigNum))
					}
					// w.SetSigItems(nil)                                // First clear the sigs
					msg.Header, err = msgState.SetupSignedMessage(w.InternalSignedMsgHeader, false, sigNum, mc) // Now add all we have seen
					if err != nil && err != types.ErrNotEnoughSigs {
						logging.Error(err)
						continue
					}
					if msg.Header == nil {
						panic(err)
					}
				}
				_, err = messages.AppendHeader(toSend, msg.Header)
				if err != nil {
					panic(err)
				}
			} else { // Otherwise we just forward the message as is
				messages.AppendMessage(toSend, msg.Message.Message)
			}
			sent = true
			// count how many signatures this message has
			// send the message
			mainChannel.Send(toSend.GetBytes(),
				messages.IsProposalHeader(msg.Index, msg.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader),
				false, forwardFunc, mc.MC.GetStats().IsRecordIndex(), mc.MC.GetStats())
		}
	}
	return
}

// CheckForwardProposal is called by multivalue consensus. It keeps track of sorted proposals by their
// VRFs and says the forward the ones with the minimum VRFs seen so far.
func CheckForwardProposal(deser *channelinterface.DeserializedItem,
	hashStr types.HashStr, decisionHash types.HashStr, sortedInitHashesIn DeserSortVRF,
	items *consinterface.ConsInterfaceItems) (sortedInitHashes DeserSortVRF, shouldForward bool) {

	if _, ok := deser.Header.(*sig.MultipleSignedMessage); !ok { // We are not using signatures
	} else { // We are using signatures, forward by VRF score if enabled
		sortedInitHashes = append(sortedInitHashesIn, deser)
		sort.Sort(sortedInitHashes)
		if hashStr == decisionHash || sortedInitHashes[0] == deser { // only forward if it is the most likely leader we have seen
			shouldForward = true
			items.MC.MC.GetStats().ProposalForward()
		}
	}
	return
}
