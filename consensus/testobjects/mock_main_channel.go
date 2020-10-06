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
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
)

var mci = []channelinterface.NetConInfo{{"nw", "mock"}}

type MockMainChannel struct {
	selfMsgs   []*deserialized.DeserializedItem
	proposals  []*deserialized.DeserializedItem
	msgs       [][]byte
	msgHeaders []messages.MsgHeader
}

func (mmc *MockMainChannel) WaitUntilAtLeastSendCons(int) error {
	return nil
}
func (mmc *MockMainChannel) SetStaticNodeList(map[sig.PubKeyStr]channelinterface.NetNodeInfo) {}
func (mmc *MockMainChannel) StartMsgProcessThreads()                                          {}
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

func (mmc *MockMainChannel) GetProposals() []*deserialized.DeserializedItem {
	ret := mmc.proposals
	mmc.proposals = nil
	return ret
}

func (mmc *MockMainChannel) GetSelfMessages() []*deserialized.DeserializedItem {
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

func (mmc *MockMainChannel) SendToSelf(deser []*deserialized.DeserializedItem, timeout time.Duration) channelinterface.TimerInterface {
	if timeout > 0 {
		// TODO why no concurrency safety here?
		return time.AfterFunc(timeout, func() { mmc.selfMsgs = append(mmc.selfMsgs, deser...) })
	}
	mmc.selfMsgs = append(mmc.selfMsgs, deser...)
	return nil
}
func (mmc *MockMainChannel) SendHeader(headers []messages.MsgHeader, _, _ bool,
	_ channelinterface.NewForwardFuncFilter, _ bool, _ stats.ConsNwStatsInterface) {
	for _, nxt := range headers {
		if nxt != nil {
			_, ok := nxt.(*messagetypes.ConsMessage)
			if !ok {
				mmc.msgHeaders = append(mmc.msgHeaders, nxt)
			}
		}
	}
}
func (mmc *MockMainChannel) Send(buff []byte, _, toSelf bool,
	_ channelinterface.NewForwardFuncFilter, _ bool, _ stats.ConsNwStatsInterface) {

	if toSelf {
		mmc.msgs = append(mmc.msgs, buff)
	}
}
func (mmc *MockMainChannel) SendAlways(buff []byte, toSelf bool, _ channelinterface.NewForwardFuncFilter,
	_ bool, _ stats.ConsNwStatsInterface) {

	if toSelf {
		mmc.msgs = append(mmc.msgs, buff)
	}
}
func (mmc *MockMainChannel) ComputeDestinations(_ channelinterface.NewForwardFuncFilter) []channelinterface.SendChannel {
	return nil
}
func (mmc *MockMainChannel) SendToPub(headers []messages.MsgHeader, _ sig.Pub, countStats bool,
	consStats stats.ConsNwStatsInterface) error {

	mmc.SendHeader(headers, false, true, nil, countStats, consStats)
	return nil
}
func (mmc *MockMainChannel) SendTo(_ []byte, _ channelinterface.SendChannel, _ bool, _ stats.ConsNwStatsInterface) {
}
func (mmc *MockMainChannel) AddExternalNode(_ channelinterface.NetNodeInfo) {
}
func (mmc *MockMainChannel) RemoveExternalNode(_ channelinterface.NetNodeInfo) {
}
func (mmc *MockMainChannel) MakeConnections([]sig.Pub) (errs []error)            { return }
func (mmc *MockMainChannel) RemoveConnections([]sig.Pub) (errs []error)          { return }
func (mmc *MockMainChannel) MakeConnectionsCloseOthers([]sig.Pub) (errs []error) { return }
func (mmc *MockMainChannel) HasProposal(p *deserialized.DeserializedItem) {
	mmc.proposals = append(mmc.proposals, p)
}
func (mmc *MockMainChannel) Recv() (*channelinterface.RcvMsg, error) {
	return nil, nil
}
func (mmc *MockMainChannel) Close() {
}
func (mmc *MockMainChannel) GotRcvConnection(*channelinterface.SendRecvChannel) {
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
func (mmc *MockMainChannel) ReprocessMessageBytes(msg []byte) {}
func (mmc *MockMainChannel) GetBehaviorTracker() channelinterface.BehaviorTracker {
	return nil
}
func (mmc *MockMainChannel) GetStats() stats.NwStatsInterface {
	return nil
}
