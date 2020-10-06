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

package channelinterface

import (
	// "fmt"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/deserialized"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

func NewConInfoList(conType types.NetworkProtocolType) []NetConInfo {
	var infos []NetConInfo
	var count int
	switch conType {
	case types.TCP:
		count = 1
	case types.UDP:
		count = config.UDPConnCount
	default:
		panic("invalid conn type")
	}
	for i := 0; i < count; i++ {
		infos = append(infos, NetConInfo{Addr: ":0", Nw: conType.String()})
	}
	return infos
}

// RunChannelTest is used to test network channels, where there all the channels have already been connected to eachother
// loops is the number of times to perform an all to all message broadcast
func RunChannelTest(testProcs []MainChannel, loops int, t *testing.T) {
	var wg1 sync.WaitGroup
	closeChannel := make(chan int, len(testProcs))

	start := time.Now()
	for i := 0; i < len(testProcs); i++ {
		go runLoop(i, testProcs[i], len(testProcs), loops, &wg1, t, closeChannel)
		wg1.Add(1)
	}

	wg1.Wait()
	t.Log("test took", time.Now().Sub(start))
	for i := 0; i < len(testProcs); i++ {
		testProcs[i].Close()
	}
}

// runLoop should then be called for each channel in the RunChannelTest
func runLoop(id int, tp MainChannel, numChannels, loops int, wg *sync.WaitGroup, t *testing.T, closeChannel chan int) {
	hdrs := make([]messages.MsgHeader, 1)

	var tstMsgTimeout messagetypes.TestMessageTimeout
	hdrsTo := make([]messages.MsgHeader, 1)
	hdrsTo[0] = tstMsgTimeout
	var wgResend sync.WaitGroup
	doneChan := make(chan int)

	for i := types.ConsensusInt(1); i < types.ConsensusInt(loops+1); i++ {
		tstMsg := &messagetypes.NetworkTestMessage{uint32(id), i}
		hdrs[0] = tstMsg
		buff, err := messages.CreateMsg(hdrs)
		assert.Nil(t, err)
		b := buff.GetBytes()
		if i != types.ConsensusInt(loops) {
			// we must send and receive the message from everyone in each loop
			// except the last where we force it to wait for the timeout and send there
			m := make([]byte, len(b))
			copy(m, b)
			tp.Send(m, false, true, ForwardAllPub, false, nil)
		}

		buffTo, err := messages.CreateMsg(hdrsTo)
		assert.Nil(t, err)
		deser := []*deserialized.DeserializedItem{
			{
				Index:          types.ConsensusIndex{Index: i, FirstIndex: i},
				HeaderType:     tstMsgTimeout.GetID(),
				Header:         tstMsgTimeout,
				IsDeserialized: true,
				IsLocal:        types.LocalMessage,
				Message:        sig.FromMessage(buffTo)}}
		tp.SendToSelf(deser, 101*time.Millisecond)

		for j := 0; j < numChannels+numChannels; j++ {
			rcvMsg, err := tp.Recv()
			if err != nil {
				if err == types.ErrTimeout {
					j--
					// resend the message on timeout
					m := make([]byte, len(b))
					copy(m, b)
					tp.Send(m, false, true, ForwardAllPub, false, nil)
					continue
				} else {
					t.Error(err)
				}
			}
			if len(rcvMsg.Msg) != 1 {
				t.Error("should only be single messages")
			}
			for _, msg := range rcvMsg.Msg {
				if msg == nil {
					panic("nil msg")
				}

				if msg.Index.Index.(types.ConsensusInt) > types.ConsensusInt(loops) {
					t.Error("Got invalid index", msg.Index, loops)
				}
				if rcvMsg.SendRecvChan != nil {
					// send the msg on the return channel
					fromID := msg.Header.(*messagetypes.NetworkTestMessage).ID
					if fromID != uint32(id) {
						newhdrs := make([]messages.MsgHeader, 1)
						newhdrs[0] = msg.Header
						// rcvMsg.SendRecvChan.ReturnChan.SendReturn(messages.CreateMsg(i, newhdrs).GetBytes(), rcvMsg.SendRecvChan.ConnectionInfo)
						mm, err := messages.CreateMsg(newhdrs)
						assert.Nil(t, err)
						rcvMsg.SendRecvChan.MainChan.SendTo(mm.GetBytes(), rcvMsg.SendRecvChan.ReturnChan, false, nil)
					}
				}

			}
		}

	}
	// let all the resend loops finish
	close(doneChan)
	wgResend.Wait()
	// tell the experiment thread you are done
	wg.Done()

	// wait until it closes our channel
	for {
		_, err := tp.Recv()
		if err == types.ErrClosingTime {
			break
		}
	}
	<-closeChannel
}

func forwardAll(sendChans []SendChannel) []SendChannel {
	return sendChans
}
