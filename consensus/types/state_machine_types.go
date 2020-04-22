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
package types

import (
	"fmt"
)

// StateMachineType represents the object implemeting the StateMachineInterface for the application run by the consensus nodes
type StateMachineType int

const (
	BinaryProposer        StateMachineType = iota // BinCons1ProposalInfo object
	BytesProposer                                 // MvCons1ProposalInfo object
	CounterProposer                               // CounterProposalInfo object
	TestProposer                                  // SimpleProposalInfo object
	CounterTxProposer                             // SimpleTxProposer
	CurrencyTxProposer                            // SimpleCurrencyTxProposer object
	CausalCounterProposer                         // CausalCounterProposalInfo object
	DirectAssetProposer                           // AssetProposer using DirectAsset
	ValueAssetProposer                            // AssetProposer using ValueAsset
)

// ComputeNumRounds returns the number of consensus instances to run for the given state machine type.
func ComputeNumRounds(to TestOptions) uint64 {
	for _, nxt := range append(TotalOrderProposerTypes, TestProposer, CausalCounterProposer) {
		if to.StateMachineType == nxt {
			return to.MaxRounds
		}
	}
	for _, nxt := range CausalProposerTypes { // For causal each participant runs to.MaxRounds
		if to.StateMachineType == nxt {
			return uint64(to.NumTotalProcs-to.NumNonMembers) * to.MaxRounds
		}
	}
	panic("unknown SM type")
}

var AllProposerTypes = append(TotalOrderProposerTypes, CausalProposerTypes...)

var TotalOrderProposerTypes = append(BinaryProposerTypes, MultivalueProposerTypes...)

// BinaryProposerTypes are the StateMachineTypes that are valid to use with binary consensus
var BinaryProposerTypes = []StateMachineType{BinaryProposer}

// MultivalueProposerTypes are the StateMachineTypes that are valid to use with multivalue consensus
var MultivalueProposerTypes = []StateMachineType{BytesProposer, CounterProposer,
	CounterTxProposer, CurrencyTxProposer}

var CausalProposerTypes = []StateMachineType{ValueAssetProposer, DirectAssetProposer, CausalCounterProposer}

//var CausalProposerTypes = []StateMachineType{}

// var MultivalueProposerTypes = []StateMachineType{BytesProposer, CounterProposer, CounterTxProposer}

// var MultivalueProposerTypes = []StateMachineType{CurrencyTxProposer}

// MultivalueProposerTypesNoCurrency is the same as MultivalueProposerTypes
// except without CurrencyTxProposer
var MultivalueProposerTypesNoCurrency = []StateMachineType{BytesProposer, CounterProposer,
	CounterTxProposer}

// var MultivalueProposerTypes = []StateMachineType{CurrencyTxProposer}

func (smt StateMachineType) String() string {
	switch smt {
	case BinaryProposer:
		return "BinaryStateMachine"
	case BytesProposer:
		return "MvRandomStringStateMachine"
	case CounterProposer:
		return "CounterStateMachine"
	case TestProposer:
		return "SimpleProposalStateMachine"
	case CounterTxProposer:
		return "SimpleTxProposer"
	case CurrencyTxProposer:
		return "CurrencyTxProposer"
	case ValueAssetProposer:
		return "ValueAssetProposer"
	case DirectAssetProposer:
		return "DirectAssetProposer"
	case CausalCounterProposer:
		return "CausalCounterProposer"
	default:
		return fmt.Sprintf("StateMachineType%d", smt)
	}
}

func (smt StateMachineType) IsBinStateMachineType() bool {
	return inSmSlice(smt, BinaryProposerTypes)
}

func (smt StateMachineType) IsMvStateMachineType() bool {
	return inSmSlice(smt, MultivalueProposerTypes)
}

func inSmSlice(smt StateMachineType, sli []StateMachineType) bool {
	for _, nxt := range sli {
		if nxt == smt {
			return true
		}
	}
	return false
}
