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

package mvcons1

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type MvBinCons1Config struct {
	cons.StandardMvConfig
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (MvBinCons1Config) GetStopOnCommitTypes(optionType cons.GetOptionType) []types.StopOnCommitType {
	switch optionType {
	case cons.AllOptions:
		return []types.StopOnCommitType{types.Immediate, types.SendProof, types.NextRound}
	case cons.MinOptions:
		return []types.StopOnCommitType{types.Immediate}
	default:
		panic(optionType)
	}
}

type MvBinConsRnd1Config struct {
	cons.StandardMvConfig
}

// GetCoinTypes returns the types of coins allowed.
func (MvBinConsRnd1Config) GetCoinTypes(optionType cons.GetOptionType) []types.CoinType {
	switch optionType {
	case cons.AllOptions:
		return types.StrongCoins
	case cons.MinOptions:
		return []types.CoinType{types.StrongCoin1Type}
	default:
		panic(optionType)
	}
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (MvBinConsRnd1Config) GetStopOnCommitTypes(optionType cons.GetOptionType) []types.StopOnCommitType {
	switch optionType {
	case cons.AllOptions:
		return []types.StopOnCommitType{types.Immediate, types.SendProof, types.NextRound}
	case cons.MinOptions:
		return []types.StopOnCommitType{types.Immediate}
	default:
		panic(optionType)
	}
}

// GetAllowSupportCoinTypes returns the values for if the the consensus supports sending messages supporting the coin
// (for randomized binary consensus) or not or both.
func (MvBinConsRnd1Config) GetAllowSupportCoinTypes(gt cons.GetOptionType) []bool {
	switch gt {
	case cons.AllOptions:
		return types.WithBothBool
	case cons.MinOptions:
		return types.WithFalse
	default:
		panic(gt)
	}
}

// GetSigTypes return the types of signatures supported by the consensus
func (MvBinConsRnd1Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
	switch gt {
	case cons.AllOptions:
		return types.CoinSigTypes
	case cons.MinOptions:
		return []types.SigType{types.EDCOIN}
	default:
		panic(gt)
	}
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (MvBinConsRnd1Config) GetUsePubIndexTypes(gt cons.GetOptionType) []bool {
	return types.WithTrue
}

func GetConf() cons.ConfigOptions {
	return MvBinConsRnd1Config{}
}
