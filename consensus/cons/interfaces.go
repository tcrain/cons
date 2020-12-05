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
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/messages"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// var binconsSeed int64

type ConsStateInterface interface {
	Start()
	// ProcessMessage should be called when a message sent from an external node is ready to be processed.
	// It returns a list of messages to be sent back to the sender of the original message (if any).
	ProcessMessage(rcvMsg *channelinterface.RcvMsg) (returnMsg [][]byte, returnErrs []error)

	// ProcessLocalMessage should be called when a message sent from the local node is ready to be processed.
	ProcessLocalMessage(rcvMsg *channelinterface.RcvMsg)

	// SMStatsString prints the statistics of the state machine.
	SMStatsString(testDuration time.Duration) string
	// SMStats returns the stats object of the state machine.
	SMStats() consinterface.SMStats

	// Collect is called when the process is terminating
	Collect()
}

// ConsInitSate is used as an input to the consensus initalization, and provoides extra configuration details.
type ConsInitState struct {
	IncludeProofs bool // If true messages should include a list of current signautures received for that message.

	// These will only be set when running tests.
	Id uint32 // The id of the node.
	N  uint32 // The number of nodes participating.
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// BinConsInterface is an interface for sending binary consensus messages.
// Multivalue to binary consensus reductions may use it.
type BinConsInterface interface {
	consinterface.ConsItem
	// CheckRound checks for the given round if enough messages have been received to progress to the next round
	// and return true if it can.
	CheckRound(nmt int, t int, round types.ConsensusRound,
		mainChannel channelinterface.MainChannel) bool
	// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
	CanSkipMvTimeout() bool
	// GetBinDecided returns -1 if not decided, or the decided value, and the round decided.
	GetBinDecided() (int, types.ConsensusRound)
	// GetMVInitialRoundBroadcast returns the type of binary message that the multi-value reduction should broadcast for the initial round.
	GetMVInitialRoundBroadcast(val types.BinVal) messages.InternalSignedMsgHeader
}

// GetMvMsgRound is a helper method to the the round from either a MvInitMessage, PartialMessage, MvCommitMessage, MvEchoMessage
func GetMvMsgRound(deser *deserialized.DeserializedItem) (round types.ConsensusRound) {
	switch v := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.MvInitMessage:
		round = v.Round
	case *messagetypes.PartialMessage:
		round = v.Round
	case *messagetypes.MvEchoMessage:
		round = v.Round
	case *messagetypes.MvCommitMessage:
		round = v.Round
	case *messagetypes.MvEchoHashMessage:
		round = v.Round
	default:
		panic("invalid msg type")
	}
	return
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// ConfigOptions is an interface that returns the set of valid configurations for a given consensus implementation.
// Each consensus implementation should create an implementation of this interface so tests can be run on the valid configs.
type ConfigOptions interface {
	// ComputeSigAndEncoding(options types.TestOptions) (noSignatures, encryptChannels bool)

	// GetIsMV returns true if the consensus is multi-value or false if binary.
	GetIsMV() bool
	// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
	RequiresStaticMembership() bool

	// GetStopOnCommitTypes returns the types to test when to terminate.
	GetStopOnCommitTypes(optionType GetOptionType) []types.StopOnCommitType
	// GetCoinTypes returns the types of coins allowed.
	GetCoinTypes(optionType GetOptionType) []types.CoinType
	// GetByzTypes returns the fault types to test.
	GetByzTypes(optionType GetOptionType) []types.ByzType
	// GetStateMachineTypes returns the types of state machines to test.
	GetStateMachineTypes(GetOptionType) []types.StateMachineType
	// GetOrderingTypes returns the types of ordering supported by the consensus.
	GetOrderingTypes(GetOptionType) []types.OrderingType
	// GetStateMachineTypes  returns the types of state machines supported by the consensus
	GetSigTypes(GetOptionType) []types.SigType
	// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
	GetUsePubIndexTypes(GetOptionType) []bool
	// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
	GetIncludeProofsTypes(GetOptionType) []bool
	// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
	GetMemberCheckerTypes(GetOptionType) []types.MemberCheckerType
	// GetRandMemberCheckerTypes returns the types of random member checkers supported by the consensus
	GetRandMemberCheckerTypes(GetOptionType) []types.RndMemberType
	// GetUseMultiSigTypes() []bool
	// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
	GetRotateCoordTypes(GetOptionType) []bool
	// GetAllowSupportCoinTypes returns the values for if the the consensus supports sending messages supporting the coin
	// (for randomized binary consensus) or not or both.
	GetAllowSupportCoinTypes(GetOptionType) []bool
	// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
	// when using total ordering or not or both.
	GetAllowConcurrentTypes(GetOptionType) []bool
	// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
	// directly to the leader.
	GetCollectBroadcast(GetOptionType) []types.CollectBroadcastType
	// GetBroadcastFunc returns the broadcast function for the given byzantine type
	GetBroadcastFunc(types.ByzType) consinterface.ByzBroadcastFunc
	// GetAllowNoSignatures returns true if the consensus can run without signatures
	GetAllowNoSignatures(GetOptionType) []bool
	// GetAllowsNonMembers returns true if there can be replicas that are not members of the consensus.
	GetAllowsNonMembers() bool
}
