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

package consinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"time"
)

// CausalStateMachineInterface represents a state machine for causally ordered applications.
// It is different than the normal state machine in several ways.
// First there is no request for proposals. The consensus is always ready to receive a proposal.
// Second, at decision, a new state machine is not created, it is only created when a proposal is received.
// Because of (1) and (2), it is the state machines responsibility to generate a new proposal (when ready)
// once the previous instance has decided.
// Third, a decision returns a set of consensus hashes.
// These are considered the outputs, that allow generation of new consensus instances, a new instance can take one
// or more output hashes from a previous decision.
// One of the key ideas of causal is that each output is "owned" by a single public key, thus
// if the member checker ensures that each output hash can be proposed by exactly one member then
// the system will then ensure that each hash is only included in one decided consensus instance and that
// all the inputs to a state machine are owned by a single public key.
// Fourth DoneKeep will always be called immediately after HasDecided.

type CausalStateMachineInterface interface {
	GeneralStateMachineInterface

	CheckOK()

	// Init is called to initialize the object, lastProposal is the number of consensus instances to run, after which a message should be sent
	// on doneChan telling the consensus to shut down.
	// If basic init is true, then the SM will just be used for checking stats and checking decisions.
	Init(gc *generalconfig.GeneralConfig, endAfter types.ConsensusInt, memberCheckerState ConsStateInterface,
		mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType, basicInit bool)

	// GenerateNewSM is called on this init SM to generate a new SM given the items to be consumed.
	// It should just generate the item, it should not change the state of any of the parentSMs.
	// This will be called on the initial CausalStateMachineInterface passed to the system
	GenerateNewSM(consumedIndices []types.ConsensusID, parentSMs []CausalStateMachineInterface) CausalStateMachineInterface

	// HasEchoed should return true if this SM has echoed a proposal
	// HasEchoed() bool
	// GetDependentItems returns a  list of items dependent from this SM.
	// This list must be the same as the list returned from HasDecided
	GetDependentItems() []sig.ConsIDPub

	// HasDecided is called each time a consensus decision takes place, given the index and the decided vaule.
	// Proposer is the public key of the node that proposed the decision.
	// Owners are the owners of the input consensus indices.
	// It returns a list of causally dependent ids and public keys of the owners of those ids,
	// and the decided value (in case the state machine wants to change it).
	HasDecided(proposer sig.Pub, index types.ConsensusIndex, owners []sig.Pub,
		decision []byte) (outputs []sig.ConsIDPub, updatedDecision []byte)

	// CheckDecisions is for testing and will be called at the end of the test with
	// a causally ordered tree of all the decided values, it should then check if the
	// decided values are valid.
	CheckDecisions(root *utils.StringNode) (errors []error)

	// FailAfter is for testing, once index is reached, it should send a message on doneChan (the input from Init), telling the consensus to shutdown.
	FailAfter(index types.ConsensusInt)

	// GetInitialFirstIndex returns the hash of some unique string for the first instance of the state machine.
	GetInitialFirstIndex() types.ConsensusID

	// StartInit is called on the init state machine to start the program.
	StartInit(memberCheckerState ConsStateInterface)
}

type SMStats interface {
	StatsString(testDuration time.Duration) string
}

type GeneralStateMachineInterface interface {
	// FinishedLastRound returns true if the last test index has finished.
	FinishedLastRound() bool
	// GetIndex returns the index for this consensus
	GetIndex() types.ConsensusIndex
	// ValidateProposal should return nil if the input proposal is valid, otherwise an error.
	// Proposer is the public key of the node that proposed the value.
	// It is called on the parent state machine instance of the proposal after it has decided.
	ValidateProposal(proposer sig.Pub, proposal []byte) error
	// GetByzProposal should generate a byzantine proposal based on the configuration
	GetByzProposal(originProposal []byte, gc *generalconfig.GeneralConfig) (byzProposal []byte)
	// GetInitialState returns the initial state of the program.
	GetInitialState() []byte
	// StatsString returns statistics for the state machine.
	StatsString(testDuration time.Duration) string
	// GetDecided returns true if the SM has decided.
	GetDecided() bool
	// GetSMStats returns the statistics object for the SM.
	GetSMStats() SMStats
	// GetRand returns 32 random bytes if supported by the state machine type.
	// It should only be called after HasDecided.
	GetRand() [32]byte
	// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
	DoneClear()
	// DoneKeep should be called if the instance of the state machine will be kept.
	DoneKeep()
	// Collect is called when the item is about to be garbage collected and is used as a sanity check.
	Collect()
	// EndTest is called when the test is finished
	EndTest()
}

// StateMachineInterface represents the object running the application on top of the consensus.
// It is responsible for sending proposals to the consensus, it should do this by calling spi.mainChannel.HasProposal().
// It should not call HasProposal for a consensus index until the previous index has completed.
type StateMachineInterface interface {
	GeneralStateMachineInterface

	// GetProposal is called when a consensus index is ready for a proposal.
	// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
	GetProposal()
	// HasDecided is called each time a consensus decision takes place, given the index and the decided vaule.
	// The indecies can be expected to be called in order starting at 1.
	// Proposer is the public key of the node that proposed the decision.
	HasDecided(proposer sig.Pub, index types.ConsensusInt, decision []byte)
	// StartIndex is called when the previous consensus index has finished.
	// It is called on the previous consensus index with the index of next state machine.
	// This might be called multiple times with multiple indicies as allowed by certain consensuses (e.g. mvcons3).
	// It should return the state machine instance for index index.
	StartIndex(index types.ConsensusInt) StateMachineInterface
	// FailAfter is for testing, once index is reached, it should send a message on doneChan (the input from Init), telling the consensus to shutdown.
	FailAfter(index types.ConsensusInt)
	// Init is called to initialize the object, lastProposal is the number of consensus instances to run, after which a message should be sent
	// on doneChan telling the consensus to shut down.
	// Need concurrent is the number of consensus indicies that need to be additionally run for the consensus to complete.
	// If basic init is true, then the SM will just be used for checking stats and checking decisions.
	Init(generalConfig *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
		mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType, basicInit bool) // To set state
	// GetDone returns the done status of this SM.
	GetDone() types.DoneType
	// CheckDecisions is for testing and will be called at the end of the test with the ordered list of all the decided values, it should then check if the
	// decided values are valid.
	// Out of order errors are caused by committing proposals that did not see the previous proposal, which may be allowed by some consensuses.
	CheckDecisions([][]byte) (outOfOrderErrors, errors []error)
	// CheckStartStatsRecording is called before allocating an index to check if stats recording should start.
	CheckStartStatsRecording(index types.ConsensusInt)
}
