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
Objects shared by some tests.
*/
package testobjects

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
)

var mci = []channelinterface.NetConInfo{{"nw", "mock"}}

type MockMainChannel struct {
	selfMsgs   []*channelinterface.DeserializedItem
	proposals  []*channelinterface.DeserializedItem
	msgs       [][]byte
	msgHeaders []messages.MsgHeader
}

func (mmc *MockMainChannel) WaitUntilAtLeastSendCons(n int) error {
	return nil
}

func (mmc *MockMainChannel) StartMsgProcessThreads() {}
func (mmc *MockMainChannel) GetMsgs() [][]byte {
	return mmc.msgs
}

func (mmc *MockMainChannel) GetFirstHeader() messages.MsgHeader {
	if len(mmc.msgHeaders) == 0 {
		return nil
	}
	item := mmc.msgHeaders[0]
	mmc.msgHeaders = mmc.msgHeaders[1:]
	return item
}

func (mmc *MockMainChannel) GetMessages() [][]byte {
	return mmc.msgs
}

func (mmc *MockMainChannel) GetProposals() []*channelinterface.DeserializedItem {
	ret := mmc.proposals
	mmc.proposals = nil
	return ret
}

func (mmc *MockMainChannel) GetSelfMessages() []*channelinterface.DeserializedItem {
	return mmc.selfMsgs
}

func (mmc *MockMainChannel) ClearMsgHeaders() {
	mmc.msgHeaders = nil
}

func (mmc *MockMainChannel) ClearMessages() {
	mmc.msgs = nil
}

func (mmc *MockMainChannel) ClearSelfMessages() {
	mmc.selfMsgs = nil
}

func (mmc *MockMainChannel) SendToSelf(deser []*channelinterface.DeserializedItem, timeout time.Duration) channelinterface.TimerInterface {
	if timeout > 0 {
		// TODO why no concurrency safety here?
		return time.AfterFunc(timeout, func() { mmc.selfMsgs = append(mmc.selfMsgs, deser...) })
	}
	mmc.selfMsgs = append(mmc.selfMsgs, deser...)
	return nil
}
func (mmc *MockMainChannel) SendHeader(headers []messages.MsgHeader, isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {
	for _, nxt := range headers {
		if nxt != nil {
			_, ok := nxt.(*messagetypes.ConsMessage)
			if !ok {
				mmc.msgHeaders = append(mmc.msgHeaders, nxt)
			}
		}
	}
}
func (mmc *MockMainChannel) Send(buff []byte, isProposal, toSelf bool,
	forwardChecker channelinterface.NewForwardFuncFilter, countStats bool) {

	if toSelf {
		mmc.msgs = append(mmc.msgs, buff)
	}
}
func (mmc *MockMainChannel) SendAlways(buff []byte, toSelf bool, forwardChecker channelinterface.NewForwardFuncFilter,
	countStats bool) {

	if toSelf {
		mmc.msgs = append(mmc.msgs, buff)
	}
}
func (mmc *MockMainChannel) ComputeDestinations(forwardFunc channelinterface.NewForwardFuncFilter) []channelinterface.SendChannel {
	return nil
}
func (mmc *MockMainChannel) SendToPub(headers []messages.MsgHeader, pub sig.Pub, countStats bool) error {
	mmc.SendHeader(headers, false, true, nil, countStats)
	return nil
}
func (mmc *MockMainChannel) SendTo(buff []byte, dest channelinterface.SendChannel, countStats bool) {
}
func (mmc *MockMainChannel) AddExternalNode(conn channelinterface.NetNodeInfo) {
}
func (mmc *MockMainChannel) RemoveExternalNode(conn channelinterface.NetNodeInfo) {
}
func (mmc *MockMainChannel) MakeConnections([]sig.Pub) (errs []error)            { return }
func (mmc *MockMainChannel) RemoveConnections([]sig.Pub) (errs []error)          { return }
func (mmc *MockMainChannel) MakeConnectionsCloseOthers([]sig.Pub) (errs []error) { return }
func (mmc *MockMainChannel) HasProposal(p *channelinterface.DeserializedItem) {
	mmc.proposals = append(mmc.proposals, p)
}
func (mmc *MockMainChannel) Recv() (*channelinterface.RcvMsg, error) {
	return nil, nil
}
func (mmc *MockMainChannel) Close() {
}
func (mmc *MockMainChannel) GotRcvConnection(rcvcon *channelinterface.SendRecvChannel) {
}
func (mmc *MockMainChannel) CreateSendConnection(channelinterface.NetNodeInfo) error {
	return nil
}
func (mmc *MockMainChannel) GetLocalNodeConnectionInfo() channelinterface.NetNodeInfo {
	return channelinterface.NetNodeInfo{AddrList: mci,
		Pub: nil}
}
func (mmc *MockMainChannel) StartInit() {
}
func (mmc *MockMainChannel) InitInProgress() bool {
	return false
}
func (mmc *MockMainChannel) EndInit() {
}
func (mmc *MockMainChannel) ReprocessMessage(*channelinterface.RcvMsg) {
}
func (mmc *MockMainChannel) GetBehaviorTracker() channelinterface.BehaviorTracker {
	return nil
}
func (mmc *MockMainChannel) GetStats() stats.NwStatsInterface {
	return nil
}
