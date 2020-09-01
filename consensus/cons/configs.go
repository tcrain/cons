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
	"github.com/tcrain/cons/consensus/types"
)

type StandardMvConfig struct{}

// GetCoinTypes returns the types of coins allowed.
func (StandardMvConfig) GetCoinTypes(optionType GetOptionType) []types.CoinType {
	return []types.CoinType{types.NoCoinType}
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (StandardMvConfig) GetStopOnCommitTypes(optionType GetOptionType) []types.StopOnCommitType {
	return []types.StopOnCommitType{types.Immediate}
}

// GetBroadcastFunc returns the broadcast function for the given byzantine type
func (StandardMvConfig) GetBroadcastFunc(bt types.ByzType) consinterface.ByzBroadcastFunc {
	switch bt {
	case types.NonFaulty:
		return consinterface.NormalBroadcast
	case types.BinaryBoth:
		return BroadcastBinBoth
	case types.BinaryFlip:
		return BroadcastBinFlip
	case types.Mute:
		return BroadcastMute
	case types.HalfHalfNormal:
		return BroadcastHalfHalfNormal
	case types.HalfHalfFixedBin:
		return BroadcastHalfHalfFixedBin
	default:
		panic(bt)
	}
}

// GetByzTypes returns the fault types to test.
func (StandardMvConfig) GetByzTypes(optionType GetOptionType) []types.ByzType {
	return types.AllByzTypes
}

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (StandardMvConfig) GetAllowNoSignatures(gt GetOptionType) []bool { //[]types.UseSignaturesType {
	return types.WithFalse // []types.UseSignaturesType{types.UseSignatures, types.ConsDependentSignatures}
}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (StandardMvConfig) GetCollectBroadcast(GetOptionType) []types.CollectBroadcastType {
	return types.AllCollectBroadcast
}

// GetIsMV returns true.
func (StandardMvConfig) GetIsMV() bool {
	return true
}

// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
func (StandardMvConfig) RequiresStaticMembership() bool {
	return false
}

// GetStateMachineTypes returns the types of state machines to test.
func (StandardMvConfig) GetStateMachineTypes(GetOptionType) []types.StateMachineType {
	return types.AllProposerTypes
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
func (StandardMvConfig) GetOrderingTypes(gt GetOptionType) []types.OrderingType {
	switch gt {
	case AllOptions:
		return types.BothOrders
	case MinOptions:
		return []types.OrderingType{types.Total}
	default:
		panic(gt)
	}
}

// GetSigTypes return the types of signatures supported by the consensus
func (StandardMvConfig) GetSigTypes(gt GetOptionType) []types.SigType {
	switch gt {
	case AllOptions:
		return types.AllSigTypes
	case MinOptions:
		return []types.SigType{types.EC}
	default:
		panic(gt)
	}
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (StandardMvConfig) GetUsePubIndexTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithTrue
	default:
		panic(gt)
	}
}

// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (StandardMvConfig) GetIncludeProofsTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
func (StandardMvConfig) GetMemberCheckerTypes(gt GetOptionType) []types.MemberCheckerType {
	switch gt {
	case AllOptions:
		return types.AllMC
	case MinOptions:
		return []types.MemberCheckerType{types.TrueMC}
	default:
		panic(gt)
	}
}

// GetRandMemberCheckerTypes returns the types of random member checkers supported by the consensus
func (StandardMvConfig) GetRandMemberCheckerTypes(gt GetOptionType) []types.RndMemberType {
	switch gt {
	case AllOptions:
		return types.AllRandMemberTypes
	case MinOptions:
		return []types.RndMemberType{types.NonRandom}
	default:
		panic(gt)
	}
}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (StandardMvConfig) GetRotateCoordTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetAllowSupportCoinTypes returns the values for if the the consensus supports sending messages supporting the coin
// (for randomized binary consensus) or not or both.
func (StandardMvConfig) GetAllowSupportCoinTypes(gt GetOptionType) []bool {
	return types.WithFalse
}

// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
// when using total ordering or not or both.
func (StandardMvConfig) GetAllowConcurrentTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

//////////////////////////////////////////////////////////////////////
// Binary config
//////////////////////////////////////////////////////////////////////

type StandardBinConfig struct{}

// GetCoinTypes returns the types of coins allowed.
func (StandardBinConfig) GetCoinTypes(optionType GetOptionType) []types.CoinType {
	return []types.CoinType{types.NoCoinType}
}

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (StandardBinConfig) GetAllowNoSignatures(gt GetOptionType) []bool { // types.UseSignaturesType {
	// return []types.UseSignaturesType{types.UseSignatures, types.ConsDependentSignatures}
	return types.WithFalse
}

// GetIsMV returns false.
func (StandardBinConfig) GetIsMV() bool {
	return false
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (StandardBinConfig) GetStopOnCommitTypes(optionType GetOptionType) []types.StopOnCommitType {
	switch optionType {
	case AllOptions:
		return []types.StopOnCommitType{types.Immediate, types.SendProof, types.NextRound}
	case MinOptions:
		return []types.StopOnCommitType{types.Immediate}
	default:
		panic(optionType)
	}
}

// GetBroadcastFunc returns the broadcast function for the given byzantine type
func (StandardBinConfig) GetBroadcastFunc(bt types.ByzType) consinterface.ByzBroadcastFunc {
	switch bt {
	case types.NonFaulty:
		return consinterface.NormalBroadcast
	case types.BinaryBoth:
		return BroadcastBinBoth
	case types.BinaryFlip:
		return BroadcastBinFlip
	case types.Mute:
		return BroadcastMute
	case types.HalfHalfNormal:
		return BroadcastHalfHalfNormal
	case types.HalfHalfFixedBin:
		return BroadcastHalfHalfFixedBin
	default:
		panic(bt)
	}
}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (StandardBinConfig) GetCollectBroadcast(GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full}
}

// GetStateMachineTypes returns the types of state machines to test.
func (StandardBinConfig) GetStateMachineTypes(GetOptionType) []types.StateMachineType {
	return types.BinaryProposerTypes
}

// GetByzTypes returns the fault types to test.
func (StandardBinConfig) GetByzTypes(optionType GetOptionType) []types.ByzType {
	return types.AllByzTypes
}

// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
func (StandardBinConfig) RequiresStaticMembership() bool {
	return false
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
func (StandardBinConfig) GetOrderingTypes(gt GetOptionType) []types.OrderingType {
	switch gt {
	case AllOptions:
		return types.BothOrders
	case MinOptions:
		return []types.OrderingType{types.Total}
	default:
		panic(gt)
	}
}

// GetSigTypes return the types of signatures supported by the consensus
func (StandardBinConfig) GetSigTypes(gt GetOptionType) []types.SigType {
	switch gt {
	case AllOptions:
		return types.AllSigTypes
	case MinOptions:
		return []types.SigType{types.EC}
	default:
		panic(gt)
	}
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (StandardBinConfig) GetUsePubIndexTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithTrue
	default:
		panic(gt)
	}
}

// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (StandardBinConfig) GetIncludeProofsTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
func (StandardBinConfig) GetMemberCheckerTypes(gt GetOptionType) []types.MemberCheckerType {
	switch gt {
	case AllOptions:
		return types.AllMC
	case MinOptions:
		return []types.MemberCheckerType{types.TrueMC}
	default:
		panic(gt)
	}
}

// GetRandMemberCheckerTypes returns the types of random member checkers supported by the consensus
func (StandardBinConfig) GetRandMemberCheckerTypes(gt GetOptionType) []types.RndMemberType {
	return []types.RndMemberType{types.NonRandom}
}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (StandardBinConfig) GetRotateCoordTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetAllowSupportCoinTypes returns the values for if the the consensus supports sending messages supporting the coin
// (for randomized binary consensus) or not or both.
func (StandardBinConfig) GetAllowSupportCoinTypes(gt GetOptionType) []bool {
	return types.WithFalse
}

// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
// when using total ordering or not or both.
func (StandardBinConfig) GetAllowConcurrentTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

//////////////////////////////////////////////////////////////////////
// simple cons config
//////////////////////////////////////////////////////////////////////

type SimpleConsConfig struct{}

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (SimpleConsConfig) GetAllowNoSignatures(gt GetOptionType) []bool { //[]types.UseSignaturesType {
	return types.WithFalse // []types.UseSignaturesType{types.UseSignatures, types.ConsDependentSignatures}
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (SimpleConsConfig) GetStopOnCommitTypes(optionType GetOptionType) []types.StopOnCommitType {
	return []types.StopOnCommitType{types.Immediate}
}

// GetCoinTypes returns the types of coins allowed.
func (SimpleConsConfig) GetCoinTypes(optionType GetOptionType) []types.CoinType {
	return []types.CoinType{types.NoCoinType}
}

// GetBroadcastFunc returns the broadcast function for the given byzantine type
func (SimpleConsConfig) GetBroadcastFunc(types.ByzType) consinterface.ByzBroadcastFunc {
	return consinterface.NormalBroadcast
}

// GetIsMV returns false.
func (SimpleConsConfig) GetIsMV() bool {
	return false
}

// GetStateMachineTypes returns the types of state machines to test.
func (SimpleConsConfig) GetStateMachineTypes(GetOptionType) []types.StateMachineType {
	return []types.StateMachineType{types.TestProposer}
}

// GetByzTypes returns the fault types to test.
func (SimpleConsConfig) GetByzTypes(optionType GetOptionType) []types.ByzType {
	return []types.ByzType{types.NonFaulty}
}

// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
func (SimpleConsConfig) RequiresStaticMembership() bool {
	return true
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
func (SimpleConsConfig) GetOrderingTypes(gt GetOptionType) []types.OrderingType {
	return []types.OrderingType{types.Total}
}

// GetSigTypes return the types of signatures supported by the consensus
func (SimpleConsConfig) GetSigTypes(gt GetOptionType) []types.SigType {
	switch gt {
	case AllOptions:
		return types.AllSigTypes
	case MinOptions:
		return []types.SigType{types.EC}
	default:
		panic(gt)
	}
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (SimpleConsConfig) GetUsePubIndexTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithTrue
	default:
		panic(gt)
	}
}

// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (SimpleConsConfig) GetIncludeProofsTypes(gt GetOptionType) []bool {
	return types.WithFalse
}

// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
func (SimpleConsConfig) GetMemberCheckerTypes(gt GetOptionType) []types.MemberCheckerType {
	switch gt {
	case AllOptions:
		return []types.MemberCheckerType{types.TrueMC, types.CurrentTrueMC}
	case MinOptions:
		return []types.MemberCheckerType{types.TrueMC}
	default:
		panic(gt)
	}
}

// GetRandMemberCheckerTypes returns the types of random member checkers supported by the consensus
func (SimpleConsConfig) GetRandMemberCheckerTypes(gt GetOptionType) []types.RndMemberType {
	return []types.RndMemberType{types.NonRandom}
}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (SimpleConsConfig) GetRotateCoordTypes(gt GetOptionType) []bool {
	return types.WithFalse
}

// GetAllowSupportCoinTypes returns the values for if the the consensus supports sending messages supporting the coin
// (for randomized binary consensus) or not or both.
func (SimpleConsConfig) GetAllowSupportCoinTypes(gt GetOptionType) []bool {
	return types.WithFalse
}

// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
// when using total ordering or not or both.
func (SimpleConsConfig) GetAllowConcurrentTypes(gt GetOptionType) []bool {
	switch gt {
	case AllOptions:
		return types.WithBothBool
	case MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (SimpleConsConfig) GetCollectBroadcast(GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full}
}

// AllTestConfig contains all options.
var AllTestConfig = OptionStruct{
	StopOnCommitTypes:      types.AllStopOnCommitTypes,
	OrderingTypes:          []types.OrderingType{types.Total, types.Causal},
	ByzTypes:               types.AllByzTypes,
	StateMachineTypes:      types.AllProposerTypes,
	SigTypes:               types.AllSigTypes,
	UsePubIndexTypes:       types.WithBothBool,
	IncludeProofsTypes:     types.WithBothBool,
	MemberCheckerTypes:     types.AllMC,
	RandMemberCheckerTypes: types.AllRandMemberTypes,
	RotateCoordTypes:       types.WithBothBool,
	AllowSupportCoinTypes:  types.WithBothBool,
	CollectBroadcast:       types.AllCollectBroadcast,
	AllowConcurrentTypes:   []types.ConsensusInt{0, 5},
	EncryptChannelsTypes:   types.WithBothBool,
	AllowNoSignaturesTypes: types.WithBothBool,
	CoinTypes:              types.AllCoins,
	UseFixedCoinPresets:    types.WithBothBool,
}

var BasicTestConfigs = ReplaceNilFields(OptionStruct{
	StateMachineTypes:    []types.StateMachineType{types.CurrencyTxProposer},
	MemberCheckerTypes:   []types.MemberCheckerType{types.CurrentTrueMC},
	StopOnCommitTypes:    []types.StopOnCommitType{types.Immediate},
	ByzTypes:             []types.ByzType{types.NonFaulty},
	IncludeProofsTypes:   types.WithFalse,
	EncryptChannelsTypes: types.WithFalse,
	UsePubIndexTypes:     types.WithTrue,
}, AllTestConfig)

// SingleSMTest is the same as AllTestConfig with the state machine types replaced to one type.
var SingleSMTest = ReplaceNilFields(OptionStruct{StateMachineTypes: []types.StateMachineType{types.BinaryProposer,
	types.CounterProposer, types.CausalCounterProposer},
	ByzTypes: []types.ByzType{types.NonFaulty}},
	AllTestConfig)

var ByzTest = ReplaceNilFields(OptionStruct{StateMachineTypes: []types.StateMachineType{types.BinaryProposer,
	types.CounterProposer, types.CausalCounterProposer},
	ByzTypes: types.AllByzTypes},
	AllTestConfig)

// StandardTotalOrderConfig contains options for a normal total order test.
var StandardTotalOrderConfig = OptionStruct{
	StopOnCommitTypes:      []types.StopOnCommitType{types.Immediate},
	OrderingTypes:          []types.OrderingType{types.Total},
	SigTypes:               []types.SigType{types.EC},
	StateMachineTypes:      []types.StateMachineType{types.CounterProposer},
	UsePubIndexTypes:       types.WithTrue,
	IncludeProofsTypes:     types.WithFalse,
	MemberCheckerTypes:     []types.MemberCheckerType{types.TrueMC},
	RandMemberCheckerTypes: []types.RndMemberType{types.NonRandom},
	RotateCoordTypes:       types.WithFalse,
	AllowSupportCoinTypes:  types.WithFalse,
	CollectBroadcast:       []types.CollectBroadcastType{types.Full},
	AllowConcurrentTypes:   []types.ConsensusInt{0},
	EncryptChannelsTypes:   types.WithTrue,
	AllowNoSignaturesTypes: types.WithFalse,
	CoinTypes:              []types.CoinType{types.NoCoinType},
	UseFixedCoinPresets:    types.WithFalse,
}

// RandTermTotalOrderTest creates tests for radomized termination consensus items.
var RandTermTotalOrderTest = ReplaceNilFields(OptionStruct{
	SigTypes:              types.ThrshSigTypes,
	UsePubIndexTypes:      types.WithTrue,
	IncludeProofsTypes:    types.WithBothBool,
	AllowSupportCoinTypes: types.WithBothBool,
}, StandardTotalOrderConfig)

var ThrsSigTst = ReplaceNilFields(OptionStruct{SigTypes: types.ThrshSigTypes, UsePubIndexTypes: types.WithTrue},
	StandardTotalOrderConfig)

var RndMemCheckTest = ReplaceNilFields(OptionStruct{
	MemberCheckerTypes:     []types.MemberCheckerType{types.TrueMC},
	RandMemberCheckerTypes: types.AllRandMemberTypes}, StandardTotalOrderConfig)
