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

package bincons1

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

var binTO = types.TestOptions{BinConsPercentOnes: 50, IncludeProofs: true}

func TestBinCons1Basic(t *testing.T) {
	cons.RunBasicTests(binTO, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1SleepBasic(t *testing.T) {
	myTO := binTO
	myTO.SleepCrypto = true
	cons.RunBasicTests(myTO, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1Byz(t *testing.T) {
	cons.RunByzTests(binTO, types.BinCons1Type, &BinCons1{}, Config{}, nil, t)
}

func TestBinCons1MemStore(t *testing.T) {
	cons.RunMemstoreTest(binTO, types.BinCons1Type, &BinCons1{}, Config{}, nil, t)
}

func TestBinCons1MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(binTO, types.BinCons1Type, &BinCons1{}, Config{}, nil, t)
}

func TestBinCons1MultiSig(t *testing.T) {
	cons.RunMultiSigTests(binTO, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1SleepMultiSig(t *testing.T) {
	to := binTO
	to.SleepCrypto = true
	to.MemCheckerBitIDType = types.BitIDSlice
	to.SigBitIDType = types.BitIDChoose
	cons.RunMultiSigTests(to, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1P2p(t *testing.T) {
	cons.RunP2pNwTests(binTO, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1SleepP2p(t *testing.T) {
	to := binTO
	to.SleepCrypto = true
	cons.RunP2pNwTests(to, types.BinCons1Type, &BinCons1{}, Config{}, []int{}, t)
}

func TestBinCons1FailDisk(t *testing.T) {
	cons.RunFailureTests(binTO, types.BinCons1Type, &BinCons1{}, Config{}, nil, t)
}
