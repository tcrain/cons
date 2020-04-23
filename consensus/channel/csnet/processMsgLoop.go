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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
	"sync"
)

// toProcessInfo is used internally for tracking a received message, and where it was from.
type toProcessInfo struct {
	buff         []byte
	wasEncrypted bool
	encrypter    sig.Pub
	// sharedSecret [32]byte
	retChan *channelinterface.SendRecvChannel
}

type processMsgLoop struct {
	wgProcessMsgLoop sync.WaitGroup     // wait for the process message threads to finish
	pendingToProcess chan toProcessInfo // messages that are ready to be processed for consensus
	netMainChannel   *NetMainChannel    // the main channel
	closeProcessChan chan bool
}

func (nrc *processMsgLoop) initProcessMsgLoop(netMainChannel *NetMainChannel) {
	nrc.pendingToProcess = make(chan toProcessInfo, config.InternalBuffSize)
	nrc.netMainChannel = netMainChannel
	nrc.closeProcessChan = make(chan bool, 1)
}

func (nrc *processMsgLoop) stopProcessThreads() {
	close(nrc.closeProcessChan)
	// start a new thread that just empties the channel
	go func() {
		for {
			if _, ok := <-nrc.pendingToProcess; !ok {
				return
			}
		}
	}()
	// wait for the process message loops to exit
	nrc.wgProcessMsgLoop.Wait()
}

func (nrc *processMsgLoop) closeProcessMsgLoop() {
	// Close the process message loops
	close(nrc.pendingToProcess)
}

// runProcessMsgLoop runs in a loop, processing messages that have been constructed from packets.
func (nrc *processMsgLoop) runProcessMsgLoop() {
	for i := 0; i < nrc.netMainChannel.NumMsgProcessThreads; i++ {
		nrc.wgProcessMsgLoop.Add(1)
		go func() {
			defer nrc.wgProcessMsgLoop.Done()
			for {
				select {
				case item, ok := <-nrc.pendingToProcess:
					if !ok {
						return
					}
					// process the msg
					nrc.netMainChannel.ProcessMessage(messages.NewMessage(item.buff), item.wasEncrypted,
						item.encrypter, item.retChan)
				case <-nrc.closeProcessChan:
					return
				}
			}
		}()
	}
}
