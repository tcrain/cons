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
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"time"
)

const initialCounterString = "some initial unique counter string"

// CausalCounterProposerInfo represents a state machine for causal consensus that just creates a single output in sequential order.
type CausalCounterProposerInfo struct {
	AbsCausalStateMachine
	AbsRandSM
	// our proposal count
	proposalIndex uint64
	lastProposal  uint64
	// my pub
	myPub sig.Pub
	// who makes the proposals
	proposer sig.Pub
}

// NewCausalCounterProposerInfo generates a new CausalCounterProposerInfo object.
func NewCausalCounterProposerInfo(myPub, proposer sig.Pub, useRand bool, initRandBytes [32]byte) *CausalCounterProposerInfo {

	return &CausalCounterProposerInfo{AbsRandSM: NewAbsRandSM(initRandBytes, useRand),
		myPub:    myPub,
		proposer: proposer}
}

// Init initalizes the object. Called once on just the very initial object.
func (spi *CausalCounterProposerInfo) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt,
	memberCheckerState consinterface.ConsStateInterface, mainChannel channelinterface.MainChannel,
	doneChan chan channelinterface.ChannelCloseType) {

	spi.lastProposal = uint64(lastProposal)
	spi.AbsRandSM.AbsRandInit(gc)

	myIndex, err := types.GenerateParentHash(spi.GetInitialFirstIndex(), nil)
	if err != nil {
		panic(err)
	}

	spi.AbsInit(myIndex, gc,
		[]sig.ConsIDPub{
			sig.ConsIDPub{ID: types.ConsensusHash(types.GetHash(spi.GetInitialState())),
				Pub: spi.proposer}}, mainChannel, doneChan)
}

// StartInit is called on the init state machine to start the program.
func (spi *CausalCounterProposerInfo) StartInit(memberCheckerState consinterface.ConsStateInterface) {
	mc, _, _, err := memberCheckerState.GetMemberChecker(spi.index)
	if err != nil {
		panic(err)
	}
	_, err = mc.MC.CheckFixedCoord(spi.myPub)
	switch err {
	case types.ErrNoFixedCoord:
		panic("must use fixed coord")
	case nil:
		spi.getProposal()
	}
}

// FailAfter sets the last consensus index to run.
func (spi *CausalCounterProposerInfo) FailAfter(end types.ConsensusInt) {
	spi.lastProposal = uint64(end)
}

// HasDecided is called each time a consensus decision takes place, given the index and the decided vaule.
// Proposer is the public key of the node that proposed the decision.
// It returns a list of causally dependent StateMachines that result from the decisions.
// It panics if the proposal is invalid (should have been checked in validate).
func (spi *CausalCounterProposerInfo) HasDecided(proposer sig.Pub, index types.ConsensusIndex,
	decision []byte) []sig.ConsIDPub {

	buf := bytes.NewReader(decision)
	var err error
	_, err = spi.RandHasDecided(proposer, buf, true)
	if err != nil {
		logging.Error("invalid mv vrf proof", err)
		panic(err)
	}
	counterBytes := make([]byte, buf.Len())
	if n, err := buf.Read(counterBytes); err != nil || n != len(counterBytes) {
		panic("buffer error")
	}
	counter, n := binary.Uvarint(counterBytes)
	if n != len(counterBytes) {
		panic("err counter")
	}

	if spi.proposalIndex != counter {
		panic(fmt.Sprintf("decided %v, expected %v", counter, spi.proposalIndex))
	}

	// we only increment the counter if we dont decide nil
	// initial proposal will be 1
	logging.Info("Incrementing counter", spi.proposalIndex, spi.index)
	spi.proposalIndex++

	// The output is the hash of the new counter value.
	buff := bytes.NewBuffer(nil)
	if _, err := utils.EncodeUvarint(spi.proposalIndex, buff); err != nil {
		panic(err)
	}
	outputs := []sig.ConsIDPub{
		sig.ConsIDPub{ID: types.ConsensusHash(types.GetHash(buff.Bytes())),
			Pub: proposer}}

	end := spi.proposalIndex == spi.lastProposal
	spi.AbsHasDecided(proposer, index, decision, outputs, end)

	pStr, err := proposer.GetPubString()
	if err != nil {
		panic(err)
	}
	myStr, err := spi.myPub.GetPubString()
	if err != nil {
		panic(err)
	}
	if pStr == myStr {
		// there is a single proposer, make the next proposal if not finished
		if spi.proposalIndex < spi.lastProposal {
			spi.getProposal()
		}
	}

	return outputs
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *CausalCounterProposerInfo) DoneClear() {
	spi.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *CausalCounterProposerInfo) DoneKeep() {
	spi.AbsDoneKeep()
	// nothing to do
}

// GetInitialFirstIndex returns the hash of some unique string for the first instance of the state machine.
func (spi *CausalCounterProposerInfo) GetInitialFirstIndex() types.ConsensusID {
	return spi.AbsGetInitialFirstIndex([]byte(initialCounterString))
}

// GetInitialState returns 0 encoded as a uvarint.
func (spi *CausalCounterProposerInfo) GetInitialState() []byte {
	// use the index
	proposalValue := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(proposalValue, uint64(0))

	return proposalValue[:n]
}

// GetProposal is used internally
func (spi *CausalCounterProposerInfo) getProposal() {
	buff := bytes.NewBuffer(nil)
	spi.RandGetProposal(buff)

	// use the index
	proposalValue := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(proposalValue, uint64(spi.proposalIndex))
	buff.Write(proposalValue[:n])

	logging.Info("support proposal", spi.proposalIndex, "my index", spi.index)
	childHashes := spi.GetDependentItems()

	addIds := make([]types.ConsensusID, len(childHashes[1:]))
	for i, nxt := range childHashes[1:] {
		addIds[i] = nxt.ID
	}

	parIdx, err := types.GenerateParentHash(childHashes[0].ID, addIds)
	if err != nil {
		panic(err)
	}

	w := messagetypes.NewMvProposeMessage(parIdx, buff.Bytes())
	logging.Info("get proposal", spi.proposalIndex, spi.gc.TestIndex, parIdx)
	if generalconfig.CheckFaulty(spi.index, spi.GeneralConfig) {
		logging.Info("get byzantine proposal", spi.proposalIndex, spi.gc.TestIndex, parIdx)
		w.ByzProposal = spi.GetByzProposal(w.Proposal, spi.GeneralConfig)
	}

	spi.AbsGetProposal(w)
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *CausalCounterProposerInfo) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	n := spi.GetRndNumBytes()
	buf := bytes.NewReader(originProposal[n:])

	v, err := binary.ReadUvarint(buf)
	if err != nil {
		panic(err)
	}

	// Create the new proposal
	buff := bytes.NewBuffer(nil)
	// Copy the rand bytes
	if _, err := buff.Write(originProposal[:n]); err != nil {
		panic(err)
	}

	if _, err := utils.EncodeUvarint(v+1, buff); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *CausalCounterProposerInfo) ValidateProposal(proposer sig.Pub, dec []byte) error {
	buf := bytes.NewReader(dec)
	_, err := spi.RandHasDecided(proposer, buf, false)
	if err != nil {
		return err
	}

	v, err := binary.ReadUvarint(buf)
	if err != nil {
		return err
	}
	if v != uint64(spi.proposalIndex) {
		return fmt.Errorf("out of order proposal")
	}
	return nil
}

// GenerateNewSM is called on this init SM to generate a new SM given the items to be consumed.
// It should just generate the item, it should not change the state of any of the parentSMs.
// This will be called on the initial CausalStateMachineInterface passed to the system
func (*CausalCounterProposerInfo) GenerateNewSM(consumedIndices []types.ConsensusID,
	parentSMs []consinterface.CausalStateMachineInterface) consinterface.CausalStateMachineInterface {
	if len(parentSMs) != 1 {
		panic("should only have 1 parent state machine")
	}
	parent := parentSMs[0].(*CausalCounterProposerInfo)
	if !parent.decided {
		panic("should be decided")
	}
	parDeps := parent.GetDependentItems()
	if len(parDeps) != 1 {
		panic("should only have 1 dep")
	}
	if len(consumedIndices) != 1 {
		panic("should only consume a single index")
	}
	if consumedIndices[0] != parDeps[0].ID {
		panic("should consume parent index")
	}
	ret := &CausalCounterProposerInfo{}
	*ret = *parent
	startRecordingStats := parent.proposalIndex == config.WarmUpInstances
	ret.AbsStartIndex(consumedIndices, parentSMs, startRecordingStats)
	logging.Infof("generating new causal counter sm from counter %v", parent.proposalIndex)
	return ret
}

func (spi *CausalCounterProposerInfo) StatsString(testDuration time.Duration) string {
	return fmt.Sprintf("Got to counter %v for hash %v", spi.proposalIndex, spi.index)
}

// GetDependentItems returns a  list of items dependent from this SM.
// This list must be the same as the list returned from HasDecided
func (spi *CausalCounterProposerInfo) GetDependentItems() []sig.ConsIDPub {
	return spi.AbsGetDependentItems()
}

// CheckDecisions ensure each value decided incraments the value by 1 (except for nil decisions).
func (spi *CausalCounterProposerInfo) CheckDecisions(root *utils.StringNode) (errs []error) {
	var proposalIndex types.ConsensusInt
	nxt := root.Children
	for nxt != nil {
		if len(nxt) != 1 {
			errs = append(errs, fmt.Errorf("should have total order for counter"))
		}

		// if nil was decided then the counter was not incremented
		dec := []byte(nxt[0].Value)
		if len(dec) == 0 {
			errs = append(errs, types.ErrNilProposal)
			continue
		}
		buf := bytes.NewReader(dec)
		err := spi.RandCheckDecision(buf)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// otherwise be sure we decided the next possible counter value
		v, err := binary.ReadUvarint(buf)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if v != uint64(proposalIndex) {
			errs = append(errs, fmt.Errorf("got an invalid decision %v, expected %v",
				v, proposalIndex))
		}
		proposalIndex++

		nxt = nxt[0].Children
	}
	return
}
