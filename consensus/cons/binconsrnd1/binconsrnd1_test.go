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

package binconsrnd1

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

// getBinConsStateMachineTypes returns a list of the valid state machine types for binary consensus given the configuration.
func getBinConsStateMachineTypes() []types.StateMachineType {
	if config.RunAllTests {
		return types.BinaryProposerTypes
	}
	return []types.StateMachineType{types.BinaryProposer}
}

// var binTO = types.TestOptions{SigType: types.TBLS, BinConsPercentOnes: 50, CoinType: types.StrongCoin1Type, IncludeProofs:true, StopOnCommit:types.SendProof, AllowSupportCoin:true}
var binTO = types.TestOptions{SigType: types.TBLSDual, BinConsPercentOnes: 50, CoinType: types.StrongCoin1Type, IncludeProofs: true, UseFixedSeed: false, UseFixedCoinPresets: false, StopOnCommit: types.SendProof}
var binTOKnown = types.TestOptions{SigType: types.EDCOIN, BinConsPercentOnes: 50, CoinType: types.KnownCoinType, UseFixedCoinPresets: false}
var binTOStrongCoin2 = types.TestOptions{UseFixedSeed: false, SigType: types.EDCOIN, BinConsPercentOnes: 50, CoinType: types.StrongCoin2Type}
var binTOStrongCoin2Echo = types.TestOptions{UseFixedSeed: false, SigType: types.EDCOIN, BinConsPercentOnes: 50, CoinType: types.StrongCoin2EchoType, EncryptChannels: true}
var binTOStrongCoin1Echo = types.TestOptions{UseFixedSeed: false, SigType: types.TBLSDual, BinConsPercentOnes: 50, CoinType: types.StrongCoin1EchoType, EncryptChannels: true}

func TestBinConsRnd1Basic(t *testing.T) {
	cons.RunBasicTests(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1SleepBasic(t *testing.T) {
	to := binTO
	to.SleepCrypto = true
	cons.RunBasicTests(to, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1BasicKnown(t *testing.T) {
	cons.RunBasicTests(binTOKnown, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1BasicStrongCoin1Echo(t *testing.T) {
	cons.RunBasicTests(binTOStrongCoin1Echo, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1SleepBasicStrongCoin1Echo(t *testing.T) {
	to := binTOStrongCoin1Echo
	to.SleepCrypto = true
	cons.RunBasicTests(to, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1BasicStrongCoin2(t *testing.T) {
	cons.RunBasicTests(binTOStrongCoin2, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1SleepBasicStrongCoin2(t *testing.T) {
	to := binTOStrongCoin2
	to.SleepCrypto = true
	to.NumMsgProcessThreads = 1
	cons.RunBasicTests(to, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1BasicStrongCoin2Echo(t *testing.T) {
	cons.RunBasicTests(binTOStrongCoin2Echo, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1SleepBasicStrongCoin2Echo(t *testing.T) {
	to := binTOStrongCoin2Echo
	to.SleepCrypto = true
	to.NumMsgProcessThreads = 1
	cons.RunBasicTests(to, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1Byz(t *testing.T) {
	cons.RunByzTests(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1MemStore(t *testing.T) {
	cons.RunMemstoreTest(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, nil, t)
}

func TestBinConsRnd1MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, nil, t)
}

func TestBinConsRnd1P2p(t *testing.T) {
	cons.RunP2pNwTests(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, []int{}, t)
}

func TestBinConsRnd1FailDisk(t *testing.T) {
	cons.RunFailureTests(binTO, types.BinConsRnd1Type, &BinConsRnd1{},
		Config{}, nil, t)
}
