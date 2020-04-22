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
package binconsrnd4

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

var binTO = types.TestOptions{SigType: types.EDCOIN, BinConsPercentOnes: 50, EncryptChannels: true, CoinType: types.StrongCoin2Type, NoSignatures: true, StopOnCommit: types.NextRound, UseFixedCoinPresets: false}
var binTOLocalCoin = types.TestOptions{SigType: types.EC, BinConsPercentOnes: 50, EncryptChannels: true, CoinType: types.LocalCoinType, NoSignatures: true, StopOnCommit: types.NextRound}
var binTOStrongCoin1 = types.TestOptions{SigType: types.TBLS, BinConsPercentOnes: 50, EncryptChannels: true, CoinType: types.StrongCoin1Type, NoSignatures: true, StopOnCommit: types.NextRound}

// getBinConsStateMachineTypes returns a list of the valid state machine types for binary consensus given the configuration.
func getBinConsStateMachineTypes() []types.StateMachineType {
	if config.RunAllTests {
		return types.BinaryProposerTypes
	}
	return []types.StateMachineType{types.BinaryProposer}
}

func TestBinConsRnd4Basic(t *testing.T) {
	cons.RunBasicTests(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, []int{}, t)
}

func TestBinConsRnd4BasicLocalCoin(t *testing.T) {
	cons.RunBasicTests(binTOLocalCoin, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, []int{}, t)
}

func TestBinConsRnd4BasicStrongCoin1(t *testing.T) {
	cons.RunBasicTests(binTOStrongCoin1, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, []int{}, t)
}

func TestBinConsRnd4Byz(t *testing.T) {
	cons.RunByzTests(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, []int{}, t)
}

func TestBinConsRnd4MemStore(t *testing.T) {
	cons.RunMemstoreTest(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, nil, t)
}

func TestBinConsRnd4MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, nil, t)
}

func TestBinConsRnd4P2p(t *testing.T) {
	cons.RunP2pNwTests(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, nil, t)
}

func TestBinConsRnd4FailDisk(t *testing.T) {
	cons.RunFailureTests(binTO, types.BinConsRnd4Type, &BinConsRnd4{},
		Config{}, nil, t)
}
