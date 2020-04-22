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

/*type randBytes []byte

func (rb randBytes) GetSignedHash() types.HashBytes {
	return types.GetHash(rb)
}
func (rb randBytes) GetSignedMessage() []byte {
	return rb
}*/

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// MvCons1ProposalInfo represents a state machine for multi-value consensus where each consensus proposes a random string.
type MvCons1ProposalInfo struct {
	AbsStateMachine
	AbsRandSM
	rand              *rand.Rand
	proposalSizeBytes int // size of the proposals in bytes
}

// NewMvCons1ProposalInfo generates a new MvCons1ProposalInfo object.
func NewMvCons1ProposalInfo(useRand bool, initRandBytes [32]byte,
	proposalSizeBytes int, seed int64) *MvCons1ProposalInfo {

	// rand := rand.New(rand.NewSource(atomic.AddInt64(&binconsSeed, 1)))
	randlocal := rand.New(rand.NewSource(seed))
	return &MvCons1ProposalInfo{
		AbsRandSM: NewAbsRandSM(initRandBytes, useRand),
		rand:      randlocal, proposalSizeBytes: proposalSizeBytes}
}

// Init initializes the object.
func (spi *MvCons1ProposalInfo) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt,
	needsConcurrent types.ConsensusInt, mainChannel channelinterface.MainChannel,
	doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsRandSM.AbsRandInit(gc)
	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
}

// HasDecided is called after the index nxt has decided.
func (spi *MvCons1ProposalInfo) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, decision []byte) {
	spi.AbsHasDecided(nxt, decision)

	buf := bytes.NewReader(decision)
	var err error
	_, err = spi.RandHasDecided(proposer, buf, true)
	if err != nil {
		logging.Error("invalid mv vrf proof", err)
	}

	if buf.Len() != spi.proposalSizeBytes {
		logging.Warning("invalid mv proposal length", buf.Len(), spi.proposalSizeBytes)
		return
	}
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *MvCons1ProposalInfo) DoneClear() {
	spi.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *MvCons1ProposalInfo) DoneKeep() {
	spi.AbsDoneKeep()
	// nothing to do
}

// GetInitialState returns []byte("initial state").
func (spi *MvCons1ProposalInfo) GetInitialState() []byte {
	return []byte("initial state")
}

func (spi *MvCons1ProposalInfo) StatsString(testDuration time.Duration) string {
	return ""
}

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *MvCons1ProposalInfo) GetProposal() {

	buff := bytes.NewBuffer(nil)
	spi.RandGetProposal(buff)
	// make a proposal, just some bytes
	proposalValue := make([]byte, spi.proposalSizeBytes)
	spi.rand.Read(proposalValue)
	_, err := buff.Write(proposalValue)
	if err != nil {
		panic(err)
	}

	w := messagetypes.NewMvProposeMessage(spi.index, buff.Bytes())
	if generalconfig.CheckFaulty(spi.GetIndex(), spi.GeneralConfig) {
		logging.Info("get byzantine proposal", spi.GetIndex(), spi.GeneralConfig.TestIndex)
		w.ByzProposal = spi.GetByzProposal(w.Proposal, spi.GeneralConfig)
	}
	spi.AbsGetProposal(w)
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *MvCons1ProposalInfo) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	n := spi.GetRndNumBytes()

	// Create the new proposal
	buff := bytes.NewBuffer(nil)
	// Copy the rand bytes
	if _, err := buff.Write(originProposal[:n]); err != nil {
		panic(err)
	}
	// Generate new bytes
	proposalValue := make([]byte, spi.proposalSizeBytes)
	spi.rand.Read(proposalValue)
	if _, err := buff.Write(proposalValue); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *MvCons1ProposalInfo) ValidateProposal(proposer sig.Pub, dec []byte) error {
	if len(dec) == 0 {
		return nil // no decided value
	}

	buf := bytes.NewReader(dec)
	_, err := spi.RandHasDecided(proposer, buf, false)
	if err != nil {
		return err
	}

	if buf.Len() != spi.proposalSizeBytes {
		return fmt.Errorf("got an invalid lenght decision %v, expected %v", len(dec), spi.proposalSizeBytes)
	}
	return nil
}

// StartIndex is called when the previous consensus index has finished.
func (spi *MvCons1ProposalInfo) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &MvCons1ProposalInfo{}
	*ret = *spi

	ret.AbsStartIndex(nxt)
	return ret
}

// CheckDecisions ensure each value decided is a string of length proposalSizeBytes
func (spi *MvCons1ProposalInfo) CheckDecisions(decs [][]byte) (outOfOrderErrors, errs []error) {
	for _, dec := range decs {
		if len(dec) == 0 {
			continue
		}
		buf := bytes.NewReader(dec)
		err := spi.RandCheckDecision(buf)
		if err != nil {
			errs = append(errs, err)
		}

		if buf.Len() != spi.proposalSizeBytes {
			errs = append(errs, fmt.Errorf("got an invalid lenght decision %v, expected %v", len(dec), spi.proposalSizeBytes))
		}
	}
	return
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////
