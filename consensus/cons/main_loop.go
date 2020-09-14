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
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
)

// RunMainLoop is the loop where the consensus operates.
// Messages are taken from the netowrk, then processesed sequentially by ConsState.
func RunMainLoop(state ConsStateInterface, mc channelinterface.MainChannel) {
	// stats := mc.GetStats()
	for true {
		// Get a message from the network.
		rcvMsg, err := mc.Recv()
		if err == types.ErrClosingTime { // time to exit
			logging.Info("Exiting main loop")
			state.Collect()
			break
		} else if err == types.ErrTimeout { // a timeout passed with no message
			// Nothing since we check timeout in all cases in the cons state
		} else if err != nil { // an error, just log it and continue
			logging.Error(err)
			continue
		}
		if rcvMsg == nil {
			panic("should not be nil")
		}

		if rcvMsg.IsLocal { // process a local message
			state.ProcessLocalMessage(rcvMsg)
			continue
		}

		returnMsgs, errs := state.ProcessMessage(rcvMsg) // process an external message
		if rcvMsg.SendRecvChan != nil {
			for _, err = range errs {
				// If there was an error processing the message then we track it in the behaviour tracker.
				conInfos := rcvMsg.SendRecvChan.ReturnChan.GetConnInfos()
				logging.Warningf("Error on connection %v: %v", conInfos, err)
				if mc.GetBehaviorTracker().GotError(err, conInfos.AddrList[0]) {
					logging.Errorf("Error on connection %v: %v, closing", conInfos, err)
					// On error we close the connection
					rcvMsg.SendRecvChan.Close(channelinterface.CloseDuringTest)
				}
				// continue
			}
		}
		if rcvMsg.SendRecvChan != nil {
			// If there are any reply messages send them directly.
			for _, rmsg := range returnMsgs {
				if len(rmsg) == 0 {
					continue
				}
				// stats.Send(len(rmsg))
				// rcvMsg.SendRecvChan.ReturnChan.SendReturn(rmsg, rcvMsg.SendRecvChan.ConnectionInfo)
				rcvMsg.SendRecvChan.MainChan.SendTo(rmsg, rcvMsg.SendRecvChan.ReturnChan, true, nil)
			}
		}
		if rcvMsg.SendRecvChan == nil && len(returnMsgs) > 0 {
			panic("shouldnt get messages to send back here")
		}
	}
}
