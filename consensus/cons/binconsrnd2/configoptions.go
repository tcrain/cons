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

package binconsrnd2

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type Config struct {
	cons.StandardBinConfig
}

// ComputeSigAndEncoding returns whether signatures should be used or messages should be encoded
func (Config) ComputeSigAndEncoding(options types.TestOptions) (noSignatures, encryptChannels bool) {
	// return cons.NonSigComputeSigAndEncoding(options)
	return
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

// GetAllowNoSignatures returns true if the consensus can run without signatures
func (Config) GetAllowNoSignatures(gt cons.GetOptionType) []bool { // []types.UseSignaturesType {
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
func (Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
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
func (Config) GetUsePubIndexTypes(_ cons.GetOptionType) []bool {
	return types.WithTrue
}

// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (Config) GetRotateCoordTypes(_ cons.GetOptionType) []bool {
	return types.WithFalse
}
