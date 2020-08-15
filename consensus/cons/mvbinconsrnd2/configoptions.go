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

package mvbinconsrnd2

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type Config struct {
	cons.StandardMvConfig
}

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (Config) GetAllowNoSignatures(gt cons.GetOptionType) []bool { // []types.UseSignaturesType {
	switch gt {
	case cons.AllOptions:
		// return []types.UseSignaturesType{types.ConsDependentSignatures, types.UseSignatures, types.NoSignatures}
		return types.WithBothBool
	case cons.MinOptions:
		// return []types.UseSignaturesType{types.NoSignatures}
		return types.WithTrue
	default:
		panic(gt)
	}
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
//func (RbBcast1Config) GetOrderingTypes(gt types.GetOptionType) []types.OrderingType {
//	return []types.OrderingType{types.Causal}
//}

// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (Config) GetIncludeProofsTypes(_ cons.GetOptionType) []bool {
	return types.WithFalse
}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
//func (Config) GetRotateCoordTypes(gt cons.GetOptionType) []bool {
//	return types.WithFalse
//}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (Config) GetStopOnCommitTypes(optionType cons.GetOptionType) []types.StopOnCommitType {
	switch optionType {
	case cons.AllOptions:
		return []types.StopOnCommitType{types.Immediate, types.SendProof, types.NextRound}
	case cons.MinOptions:
		return []types.StopOnCommitType{types.NextRound}
	default:
		panic(optionType)
	}
}

// GetCoinTypes returns the types of coins allowed.
func (Config) GetCoinTypes(optionType cons.GetOptionType) []types.CoinType {
	switch optionType {
	case cons.AllOptions:
		return types.StrongCoins
	case cons.MinOptions:
		return []types.CoinType{types.StrongCoin2Type}
	default:
		panic(optionType)
	}
}

// GetSigTypes return the types of signatures supported by the consensus
func (Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
	switch gt {
	case cons.AllOptions:
		return types.AllSigTypes
	case cons.MinOptions:
		return []types.SigType{types.EDCOIN}
	default:
		panic(gt)
	}
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (Config) GetUsePubIndexTypes(_ cons.GetOptionType) []bool {
	return types.WithTrue
}
