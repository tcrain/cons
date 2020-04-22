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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

type AbsCausalStateMachine struct {
	absGeneralStateMachine
	// outputs []types.ConsensusID
	// lastProposal    types.ConsensusInt                   // the consensus index to stop at
	parHashes      []types.ConsensusID // parent hashes
	consumedHashes []types.ConsensusID // consumed hashes from the parents
	childIndices   []sig.ConsIDPub
	// so we use this so we only send finished once to the end
	setDone *bool
}

func (spi *AbsCausalStateMachine) AbsGetInitialFirstIndex(initialState []byte) types.ConsensusID {
	return types.ConsensusHash(types.GetHash(initialState))
}

// AbsInit initalizes the object.
func (spi *AbsCausalStateMachine) AbsInit(index types.ConsensusIndex, gc *generalconfig.GeneralConfig,
	childIndices []sig.ConsIDPub, mainChannel channelinterface.MainChannel,
	doneChan chan channelinterface.ChannelCloseType) {

	var setDone bool
	spi.setDone = &setDone
	spi.childIndices = childIndices
	spi.index = index
	spi.GeneralConfig = gc
	spi.mainChannel = mainChannel
	spi.doneChan = doneChan
	var closed bool
	spi.closed = &closed
	spi.decided = true
}

func (spi *AbsCausalStateMachine) AbsHasDecided(proposer sig.Pub, index types.ConsensusIndex,
	decision []byte, childIndices []sig.ConsIDPub, done bool) {

	_, _ = proposer, decision

	if spi.decided {
		panic("decided twice")
	}
	spi.decided = true
	if index.Index != spi.index.Index {
		panic(fmt.Sprintf("Got out of order decided %v, expected %v", index, spi.index))
	}
	spi.childIndices = childIndices
	spi.CheckOK()
	if *spi.closed {
		return
	}

	logging.Infof("Got decided for index %v", index)
	if done && !*spi.setDone {

		// Address so we only do this once
		*spi.setDone = true

		*spi.closed = true
		spi.GeneralConfig.Stats.DoneRecording()
		logging.Infof("Got last decided %v, time to exit", index)
		spi.doneChan <- channelinterface.ChannelCloseType(spi.GeneralConfig.TestIndex)
		// spi.endChan <- channelinterface.EndTestClose
	}
	return

}

func (spi *AbsCausalStateMachine) CheckOK() {
	// check the child indicies changed
	for _, nxt := range spi.childIndices { // sanity check TODO remove
		for _, nxtIn := range spi.consumedHashes {
			if nxt.ID.(types.ConsensusHash) == nxtIn.(types.ConsensusHash) {
				panic("produced same output")
			}
		}
	}
}

func (spi *AbsCausalStateMachine) AbsStartIndex(consumedIndices []types.ConsensusID,
	parentSMs []consinterface.CausalStateMachineInterface, startRecordingStats bool) {

	if startRecordingStats {
		spi.GeneralConfig.Stats.StartRecording(spi.GeneralConfig.CPUProfile, spi.GeneralConfig.MemProfile,
			spi.GeneralConfig.TestIndex, spi.GeneralConfig.TestID)
	}

	if spi.doneClear {
		panic(fmt.Sprint("called start index on cleared SM", spi.index))
	}
	parHashes := make([]types.ConsensusID, len(parentSMs))
	for i, nxt := range parentSMs {
		parHashes[i] = nxt.GetIndex().Index
	}
	spi.parHashes = parHashes

	// Your index is generated from the consumed indices
	var err error
	var newIdx types.ConsensusIndex
	if newIdx, err = types.GenerateParentHash(consumedIndices[0], consumedIndices[1:]); err != nil {
		panic(err)
	}
	spi.index = newIdx
	spi.consumedHashes = consumedIndices
	spi.childIndices = nil
	spi.CheckOK()

	spi.doneClear = false
	spi.doneKeep = false
	spi.decided = false
}

func (spi *AbsCausalStateMachine) AbsGetDependentItems() (childIndices []sig.ConsIDPub) {
	if !spi.decided {
		panic("tried to get child hashes before decision")
	}
	return spi.childIndices
}

// AbsGetProposal is called when a consensus index is ready for a proposal.
func (spi *absGeneralStateMachine) AbsGetProposal(hdr messages.MsgHeader) {
	//if spi.index > spi.lastProposal+spi.needsConcurrent {
	//	return
	//}
	di := &channelinterface.DeserializedItem{
		Index:          spi.GetIndex(),
		HeaderType:     hdr.GetID(),
		Header:         hdr,
		IsDeserialized: true,
		IsLocal:        types.LocalMessage}

	spi.mainChannel.HasProposal(di)
}
