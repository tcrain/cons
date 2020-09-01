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
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"time"
)

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// SimpleProposalInfo represents the statemachine object for the SimpleCons protocol, only this state machine can be used
// when testing this object.
type SimpleProposalInfo struct {
	AbsStateMachine
	AbsRandSMNotSupported
}

// NewSimpleProposalInfo creates an empty SimpleProposalInfo object.
func NewSimpleProposalInfo() *SimpleProposalInfo {
	return &SimpleProposalInfo{}
}

func (spi *SimpleProposalInfo) StatsString(testDuration time.Duration) string {
	_ = testDuration
	return ""
}

// Init initalizes the simple proposal object state.
func (spi *SimpleProposalInfo) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
}

// GetInitialState returns []byte("initial state")
func (spi *SimpleProposalInfo) GetInitialState() []byte {
	return []byte("initial state")
}

// HasDecided is called after the index nxt has decided.
func (spi *SimpleProposalInfo) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, decision []byte) {
	_ = proposer
	spi.AbsHasDecided(nxt, decision)
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *SimpleProposalInfo) DoneClear() {
	spi.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *SimpleProposalInfo) DoneKeep() {
	spi.AbsDoneKeep()
	// nothing to do
}

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *SimpleProposalInfo) GetProposal() {
	// The message type is fixed for this consensus type.
	w := messagetypes.NewSimpleConsProposeMessage(spi.index)
	spi.AbsGetProposal(w)
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *SimpleProposalInfo) GetByzProposal(originProposal []byte,
	_ *generalconfig.GeneralConfig) (byzProposal []byte) {

	// No byzantine supported, just return the original proposal
	return originProposal
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *SimpleProposalInfo) ValidateProposal(proposer sig.Pub, dec []byte) error {
	_ = proposer
	if string(dec) != fmt.Sprintf("simpleCons%v", spi.index.Index) {
		return fmt.Errorf("got an invalid decision %v", string(dec))
	}
	return nil
}

// StartIndex is called when the previous consensus index has finished.
func (spi *SimpleProposalInfo) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &SimpleProposalInfo{AbsStateMachine: spi.AbsStateMachine}
	ret.AbsStartIndex(nxt)
	return ret
}

// CheckDecisions verifies that the decided values were valid.
func (spi *SimpleProposalInfo) CheckDecisions(decs [][]byte) (outOforderErrors, errs []error) {
	for i, dec := range decs {
		i := i + 1
		if string(dec) != fmt.Sprintf("simpleCons%v", i) {
			errs = append(errs, fmt.Errorf("got an invalid decision %v", string(dec)))
		}
	}
	return
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////
