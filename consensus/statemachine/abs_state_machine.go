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

package statemachine

import (
	"fmt"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

type AbsStateMachine struct {
	absGeneralStateMachine
	startedRecordingStats bool
	doneCount             *types.ConsensusInt
	lastProposal          types.ConsensusInt   // the consensus index to stop at
	needsConcurrent       types.ConsensusInt   // additional concurrent indecies to run as needed by the consensus
	createdFrom           []types.ConsensusInt // the previous index from which this SM was created
	isCoord               bool                 // if this node is the coordinator for this round
}

// GetDone returns the done status of this SM.
func (spi *AbsStateMachine) GetDone() types.DoneType { // TODO cleanup
	if spi.doneKeep {
		return types.DoneKeep
	}
	if spi.doneClear {
		return types.DoneClear
	}
	return types.NotDone
}

func (spi *AbsStateMachine) GetStartedRecordingStats() bool {
	return spi.startedRecordingStats
}

func (spi *AbsStateMachine) AbsDoneKeep() {
	if spi.doneKeep {
		panic(fmt.Sprint("called done keep twice", spi.index))
	}
	if spi.doneClear {
		panic(fmt.Sprint("called done clear after done keep", spi.index))
	}
	spi.doneKeep = true
	if spi.index.Index.(types.ConsensusInt) == 0 { // We don't Increments the initiation object
		return
	}
	*spi.doneCount++

	if !*spi.closed && *spi.doneCount >= spi.lastProposal {
		*spi.closed = true
		spi.GeneralConfig.Stats.DoneRecording()
		logging.Infof("Got last decided %v, time to exit", spi.index.Index)
		spi.doneChan <- channelinterface.ChannelCloseType(spi.GeneralConfig.TestIndex)
		// spi.endChan <- channelinterface.EndTestClose
	}
}

func (spi *AbsStateMachine) AbsDoneClear() {
	if spi.doneClear {
		panic(fmt.Sprint("called done clear twice", spi.index))
	}
	if spi.doneKeep {
		panic(fmt.Sprint("called done keep after done clear", spi.index))
	}
	spi.doneClear = true
}

// AbsInit initalizes the object.
func (spi *AbsStateMachine) AbsInit(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt,
	needsConcurrent types.ConsensusInt, mainChannel channelinterface.MainChannel,
	doneChan chan channelinterface.ChannelCloseType) {

	var doneCount types.ConsensusInt
	spi.doneCount = &doneCount
	spi.GeneralConfig = gc
	spi.mainChannel = mainChannel
	spi.doneChan = doneChan
	spi.lastProposal = lastProposal
	var closed bool
	spi.closed = &closed
	spi.needsConcurrent = needsConcurrent
	spi.index = types.SingleComputeConsensusIDShort(types.ConsensusInt(0))
}

// FailAfter sets the last consensus index to run.
func (spi *AbsStateMachine) FailAfter(end types.ConsensusInt) {
	spi.lastProposal = end
}

// IsDone returns true is the test has finished for the state machine.
func (spi *AbsStateMachine) IsDone() bool {
	return *spi.closed
}

// GetIndex returns the current consensus index.
func (spi *AbsStateMachine) GetIndex() types.ConsensusIndex {
	return spi.index
}

// HasDecided returns true if the SM has decided.
func (spi *AbsStateMachine) GetDecided() bool {
	return spi.decided
}

// AbsHasDecided is called after the index nxt has decided, it sends the next proposal to the consensus.
func (spi *AbsStateMachine) AbsHasDecided(nxt types.ConsensusInt, dec []byte) (shouldReturn bool) {
	_ = dec
	if spi.decided {
		panic("decided twice")
	}
	spi.decided = true
	if *spi.closed {
		shouldReturn = true
		return
	}
	if nxt != spi.index.Index {
		panic(fmt.Sprintf("Got out of order decided %v, expected %v", nxt, spi.index))
	}
	logging.Infof("Got decided for index %v", nxt)
	return
}

// AbsGetProposal is called when a consensus index is ready for a proposal.
func (spi *AbsStateMachine) AbsGetProposal(hdr messages.MsgHeader) {
	//if spi.index > spi.lastProposal+spi.needsConcurrent {
	//	return
	//}
	if *spi.closed {
		return
	}
	di := &channelinterface.DeserializedItem{
		Index:          spi.index,
		HeaderType:     hdr.GetID(),
		Header:         hdr,
		IsDeserialized: true,
		IsLocal:        types.LocalMessage}

	spi.mainChannel.HasProposal(di)
}

// Collect is called when the item is about to be garbage collected and is used as a sanity check.
func (spi *AbsStateMachine) Collect() {
	if !spi.doneKeep && !spi.doneClear {
		panic("should have called done keep or done clear before garbage collection")
	}
}

// CheckStartStatsRecording is called before allocating an index to check if stats recording should start.
func (spi *AbsStateMachine) CheckStartStatsRecording(index types.ConsensusInt) {
	if index > types.ConsensusInt(spi.GeneralConfig.WarmUpInstances) {
		spi.startedRecordingStats = true
		spi.GeneralConfig.Stats.StartRecording(spi.GeneralConfig.CPUProfile, spi.GeneralConfig.MemProfile,
			spi.GeneralConfig.TestIndex, spi.GeneralConfig.TestID)
	}
}

// AbsStartIndex is called when the previous consensus index has finished.
func (spi *AbsStateMachine) AbsStartIndex(nxt types.ConsensusInt) {
	if nxt != spi.index.Index.(types.ConsensusInt)+1 {
		logging.Infof("out of order indecies, started %v after %v", spi.index, nxt)
	}
	spi.CheckStartStatsRecording(nxt)

	if spi.doneClear {
		panic(fmt.Sprint("called start index on cleared SM", spi.index))
	}
	newCreatedFrom := make([]types.ConsensusInt, len(spi.createdFrom)+1)
	copy(newCreatedFrom, spi.createdFrom)
	newCreatedFrom[len(newCreatedFrom)-1] = nxt
	spi.createdFrom = newCreatedFrom
	spi.doneClear = false
	spi.doneKeep = false
	spi.decided = false

	spi.index = types.SingleComputeConsensusIDShort(nxt)
}
