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
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"time"
)

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// BinCons1ProposalInfo represents a state machine for binary consensus that randomly proposes 0 or 1 for each binary consensus.
type BinCons1ProposalInfo struct {
	AbsStateMachine
	AbsRandSMNotSupported

	perm []bool

	binConsPercentOnes int // number of proposals that will be 1 vs 0 (randomly chosen) for testing
	rand               *rand.Rand
}

// NewBinCons1ProposalInfo generates a new BinCons1ProposalInfo object.
func NewBinCons1ProposalInfo(binConsPercentOnes int, seed int64, numParticipants int, numNodes int) *BinCons1ProposalInfo {
	// rand := rand.New(rand.NewSource(atomic.AddInt64(&binconsSeed, 1)))
	randlocal := rand.New(rand.NewSource(seed))

	var perm []bool
	if false && numParticipants == numNodes { // We use a random permutation to choose the number of ones exactly (otherwise we just choose randomly)
		perm = make([]bool, numParticipants)
		numOnes := int(float64(numParticipants) * (float64(binConsPercentOnes) / 100))
		for i := 0; i < numOnes; i++ {
			perm[i] = true
		}
	}
	return &BinCons1ProposalInfo{rand: randlocal, binConsPercentOnes: binConsPercentOnes, perm: perm}
}

func (spi *BinCons1ProposalInfo) shuffleOnes() {
	spi.rand.Shuffle(len(spi.perm), func(i, j int) {
		spi.perm[i], spi.perm[j] = spi.perm[j], spi.perm[i]
	})
}

// Init initalizes the object.
func (spi *BinCons1ProposalInfo) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
}

// GetInitialState returns []byte{0}.
func (spi *BinCons1ProposalInfo) GetInitialState() []byte {
	return []byte{0}
}

func (spi *BinCons1ProposalInfo) StatsString(testDuration time.Duration) string {
	return ""
}

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *BinCons1ProposalInfo) GetProposal() {
	// use a random value
	var binVal types.BinVal
	if spi.perm != nil {
		spi.shuffleOnes()
		if spi.perm[spi.GeneralConfig.TestIndex] {
			binVal = 1
		}
	} else {
		if spi.rand.Intn(100) < spi.binConsPercentOnes {
			binVal = 1
		}
	}
	logging.Info("propose", binVal, spi.index.Index, spi.GeneralConfig.TestIndex)
	w := messagetypes.NewBinProposeMessage(spi.index, binVal)
	spi.AbsGetProposal(w)
}

func checkBinary(dec []byte) error {
	if len(dec) != 1 {
		return fmt.Errorf("not a binary decided value: %v", dec)
	}
	switch dec[0] {
	case 0:
	case 1:
	default:
		return fmt.Errorf("not a binary decided value: %v", dec)
	}
	return nil
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *BinCons1ProposalInfo) ValidateProposal(proposer sig.Pub, dec []byte) error {
	return checkBinary(dec)
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *BinCons1ProposalInfo) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	return []byte{1 - originProposal[0]}
}

// StartIndex is called when the previous consensus index has finished.
func (spi *BinCons1ProposalInfo) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &BinCons1ProposalInfo{AbsStateMachine: spi.AbsStateMachine,
		perm:               spi.perm,
		binConsPercentOnes: spi.binConsPercentOnes,
		rand:               spi.rand}

	ret.AbsStartIndex(nxt)
	return ret
}

// HasDecided is called after the index nxt has decided.
func (spi *BinCons1ProposalInfo) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, decision []byte) {
	if err := checkBinary(decision); err != nil {
		panic(err)
	}
	spi.AbsHasDecided(nxt, decision)
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *BinCons1ProposalInfo) DoneClear() {
	spi.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *BinCons1ProposalInfo) DoneKeep() {
	spi.AbsDoneKeep()
	// nothing to do
}

// CheckDecisions ensures that only binary values were decided.
func (spi *BinCons1ProposalInfo) CheckDecisions(decs [][]byte) (outOforderErrors, errs []error) {
	for _, dec := range decs {
		if err := spi.ValidateProposal(nil, dec); err != nil {
			errs = append(errs, err)
		}
	}
	return
}
