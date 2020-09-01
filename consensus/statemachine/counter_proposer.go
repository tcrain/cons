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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messagetypes"
)

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// CounterProposalInfo represents a state machine for multi-value consensus where each consensus (that doesn't decided a nil value) proposes the next integer.
type CounterProposalInfo struct {
	AbsStateMachine
	AbsRandSM
	// our proposal count (doesnt increment when nil is decided)
	proposalIndex types.ConsensusInt
}

// NewCounterProposalInfo generates a new CounterProposalInfo object.
func NewCounterProposalInfo(useRand bool, initRandBytes [32]byte) *CounterProposalInfo {

	return &CounterProposalInfo{AbsRandSM: NewAbsRandSM(initRandBytes, useRand)}
}

// Init initalizes the object.
func (spi *CounterProposalInfo) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsRandSM.AbsRandInit(gc)
	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
}

// HasDecided is called after the index nxt has decided.
func (spi *CounterProposalInfo) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, decision []byte) {
	spi.AbsHasDecided(nxt, decision)
	if len(decision) != 0 {
		buf := bytes.NewReader(decision)
		var err error
		_, err = spi.RandHasDecided(proposer, buf, true)
		if err != nil {
			logging.Error("invalid mv vrf proof", err)
		}

		// we only incrament the counter if we dont decide nil
		// initial proposal will be 1
		logging.Info("Incrementing counter", spi.proposalIndex, spi.index)
		spi.proposalIndex++
	} else {
		logging.Info("Not incrementing counter", spi.proposalIndex, spi.index)
	}
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *CounterProposalInfo) DoneClear() {
	spi.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *CounterProposalInfo) DoneKeep() {
	spi.AbsDoneKeep()
	// nothing to do
}

// GetInitialState returns 0 encoded as a uvarint.
func (spi *CounterProposalInfo) GetInitialState() []byte {
	// use the index
	proposalValue := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(proposalValue, uint64(0))

	return proposalValue[:n]
}

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *CounterProposalInfo) GetProposal() {
	buff := bytes.NewBuffer(nil)
	spi.RandGetProposal(buff)

	// use the index
	proposalValue := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(proposalValue, uint64(spi.proposalIndex))
	buff.Write(proposalValue[:n])

	logging.Info("support proposal", spi.proposalIndex, "my index", spi.index)
	w := messagetypes.NewMvProposeMessage(spi.index, buff.Bytes())
	if generalconfig.CheckFaulty(spi.index, spi.GeneralConfig) {
		logging.Info("get byzantine proposal", spi.proposalIndex, spi.gc.TestIndex)
		w.ByzProposal = spi.GetByzProposal(w.Proposal, spi.GeneralConfig)
	}

	spi.AbsGetProposal(w)
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *CounterProposalInfo) ValidateProposal(proposer sig.Pub, dec []byte) error {
	if len(dec) == 0 {
		return nil // no decided value
	}
	buf := bytes.NewReader(dec)
	_, err := spi.RandHasDecided(proposer, buf, false)
	if err != nil {
		return err
	}

	v, err := binary.ReadUvarint(buf)
	if err != nil {
		return err
	}
	if spi.AbsStateMachine.needsConcurrent == 0 && v != uint64(spi.proposalIndex) {
		return fmt.Errorf("out of order proposal")
	}
	return nil
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *CounterProposalInfo) GetByzProposal(originProposal []byte,
	_ *generalconfig.GeneralConfig) (byzProposal []byte) {

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

// StartIndex is called when the previous consensus index has finished.
func (spi *CounterProposalInfo) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &CounterProposalInfo{}
	*ret = *spi
	ret.AbsStartIndex(nxt)
	ret.RandStartIndex(spi.randBytes)

	logging.Infof("my id %v generate next my counter %v, my index %v, nxt idx %v, created from %v", spi.GeneralConfig.TestIndex, spi.proposalIndex, spi.index, nxt, spi.createdFrom)
	return ret
}

func (spi *CounterProposalInfo) StatsString(testDuration time.Duration) string {
	_ = testDuration
	return fmt.Sprintf("Got to counter %v out of %v instances", spi.proposalIndex, spi.index)
}

// CheckDecisions ensure each value decided incraments the value by 1 (except for nil decisions).
func (spi *CounterProposalInfo) CheckDecisions(decs [][]byte) (outOfOrderErrors, errs []error) {
	var proposalIndex types.ConsensusInt
	for i, dec := range decs {
		// if nil was decided then the counter was not incremented
		if len(dec) == 0 {
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
			outOfOrderErrors = append(outOfOrderErrors, fmt.Errorf("got an invalid decision %v, expected %v, idx %v",
				v, proposalIndex, i+1))
		}
		proposalIndex++
	}
	return
}
