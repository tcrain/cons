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

type MvBinConsRnd2Config struct {
	cons.StandardMvConfig
}

// GetStopOnCommitTypes returns the types to test when to terminate.
func (MvBinConsRnd2Config) GetStopOnCommitTypes(optionType cons.GetOptionType) []types.StopOnCommitType {
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
func (MvBinConsRnd2Config) GetCoinTypes(optionType cons.GetOptionType) []types.CoinType {
	switch optionType {
	case cons.AllOptions:
		return types.StrongCoins
	case cons.MinOptions:
		return []types.CoinType{types.StrongCoin2Type}
	default:
		panic(optionType)
	}
}

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (MvBinConsRnd2Config) GetAllowNoSignatures(gt cons.GetOptionType) []bool { // []types.UseSignaturesType {
	switch gt {
	case cons.AllOptions:
		// return []types.UseSignaturesType{types.ConsDependentSignatures, types.UseSignatures, types.NoSignatures}
		return types.WithBothBool
	case cons.MinOptions:
		// return []types.UseSignaturesType{types.UseSignatures}
		return types.WithTrue
	default:
		panic(gt)
	}
}

// GetSigTypes return the types of signatures supported by the consensus
func (MvBinConsRnd2Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
	switch gt {
	case cons.AllOptions:
		// return types.CoinSigTypes
		return types.AllSigTypes // To allow for known coins
	case cons.MinOptions:
		return []types.SigType{types.EDCOIN}
	default:
		panic(gt)
	}

}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader. // TODO for mvbincons2
func (MvBinConsRnd2Config) GetCollectBroadcast(cons.GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full} // TODO
}

// GetUsePubIndex returns the values for if the consensus supports using pub index or not or both.
func (MvBinConsRnd2Config) GetUsePubIndexTypes(_ cons.GetOptionType) []bool {
	return types.WithTrue
}

// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (MvBinConsRnd2Config) GetRotateCoordTypes(_ cons.GetOptionType) []bool {
	return types.WithFalse
}

func GetConf() cons.ConfigOptions {
	return MvBinConsRnd1Config{}
}
