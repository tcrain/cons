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
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"time"
)

type AbsMVRecover struct {
	sentRequestRecover  bool                                                  // if the node has asked for the leader's init message from other nodes (because it only received the hash)
	requestedRecover    map[types.HashStr][]*channelinterface.SendRecvChannel // external nodes that have asked for the leader's init message
	requestRecoverTimer channelinterface.TimerInterface                       // timer for requesting the leader's init message from other nodes
	index               types.ConsensusIndex
	gc                  *generalconfig.GeneralConfig
}

func (abs *AbsMVRecover) InitAbsMVRecover(index types.ConsensusIndex, gc *generalconfig.GeneralConfig) {
	abs.requestedRecover = make(map[types.HashStr][]*channelinterface.SendRecvChannel)
	abs.index = index
	abs.gc = gc
}

func (sc *AbsMVRecover) GotRequestRecover(validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem,
	deser *channelinterface.DeserializedItem, initHeaders []messages.MsgHeader, senderChan *channelinterface.SendRecvChannel,
	items *consinterface.ConsInterfaceItems) {

	w := deser.Header.(*messagetypes.MvRequestRecoverMessage)
	proposalHash := types.HashStr(w.ProposalHash)
	// add them to the list of nodes that requested the init message
	sc.requestedRecover[proposalHash] = append(sc.requestedRecover[proposalHash], senderChan)
	// send them back the value if we have it
	sc.SendRecover(validatedInitHashes, initHeaders, items)
}

func (sc *AbsMVRecover) StopRecoverTimeout() {
	if sc.requestRecoverTimer != nil {
		sc.requestRecoverTimer.Stop()
		sc.requestRecoverTimer = nil
	}
}

func (sc *AbsMVRecover) StartRecoverTimeout(index types.ConsensusIndex, channel channelinterface.MainChannel) {
	// we haven't yet received the init message for the hash, so we request it from other nodes after a timeout
	if sc.requestRecoverTimer == nil {
		logging.Info("Got echos before proposal, requesting recover after timer for index", index)
		deser := []*channelinterface.DeserializedItem{
			{
				Index:          index,
				HeaderType:     messages.HdrMvRecoverTimeout,
				IsDeserialized: true,
				IsLocal:        types.LocalMessage}}
		sc.requestRecoverTimer = channel.SendToSelf(deser, time.Duration(sc.gc.MvConsRequestRecoverTimeout)*time.Millisecond)
	}
}

// sendRecover sends the init message to anyone who has send a RequestRecover message
func (sc *AbsMVRecover) SendRecover(validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem,
	initHeaders []messages.MsgHeader, items *consinterface.ConsInterfaceItems) {

	if len(sc.requestedRecover) == 0 {
		return
	}
	// msgState := sc.MessageState.(*MvCons1MessageState)
	// msgState.mutex.Lock()
	// only send messages for which they asked for the hash of an init message received
	for hashStr, initMsg := range validatedInitHashes {
		if sendChans, ok := sc.requestedRecover[hashStr]; ok {
			msg, err := messages.CreateMsg(initHeaders)
			if err != nil {
				panic(err)
			}
			messages.AppendMessage(msg, initMsg.Message.Message)
			for _, sendChan := range sendChans {
				sendChan.MainChan.SendTo(msg.GetBytes(), sendChan.ReturnChan, items.MC.MC.GetStats().IsRecordIndex(),
					items.MC.MC.GetStats())
			}
			delete(sc.requestedRecover, hashStr)
		}
	}
	// msgState.mutex.Unlock()
}

// broadcastRequest broadcasts a message requesting the init message
func (sc *AbsMVRecover) BroadcastRequestRecover(preHeaders []messages.MsgHeader, hash []byte, forwardChecker consinterface.ForwardChecker,
	mainChannel channelinterface.MainChannel, items *consinterface.ConsInterfaceItems) {

	if !sc.sentRequestRecover {
		sc.sentRequestRecover = true
		newMsg := messagetypes.NewMvRequestRecoverMessage()
		newMsg.Index = sc.index
		newMsg.ProposalHash = hash
		mainChannel.SendHeader(messages.AppendCopyMsgHeader(preHeaders, newMsg),
			false, false, forwardChecker.GetNewForwardListFunc(),
			items.MC.MC.GetStats().IsRecordIndex(), items.MC.MC.GetStats()) // TODO use a different forward function??
	}
}
