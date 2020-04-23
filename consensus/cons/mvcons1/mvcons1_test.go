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
	"testing"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

// getBinConsStateMachineTypes returns a list of the valid state machine types for multi-value consensus given the configuration.
func getMvConsStateMachineTypes() []types.StateMachineType {
	// return []types.StateMachineType{types.CurrencyTxProposer}
	if config.RunAllTests {
		return types.MultivalueProposerTypes
	}
	return []types.StateMachineType{types.CounterProposer}
}

func TestMvBinCons1Basic(t *testing.T) {
	cons.RunBasicTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, []int{}, t)
}

func TestMvBinCons1RandMC(t *testing.T) {
	cons.RunRandMCTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, true,
		[]int{}, t)
}

func TestMvBinCons1Byz(t *testing.T) {
	cons.RunByzTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, nil, t)
}

func TestMvBinCons1MemStore(t *testing.T) {
	cons.RunMemstoreTest(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, nil, t)
}

func TestMvBinCons1MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, nil, t)
}

func TestMvCons1MultiSig(t *testing.T) {
	cons.RunMultiSigTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, []int{}, t)
}

func TestMvCons1P2p(t *testing.T) {
	cons.RunP2pNwTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, nil, t)
}

func TestMvCons1FailDisk(t *testing.T) {
	cons.RunFailureTests(types.TestOptions{}, types.MvBinCons1Type, &MvCons1{}, MvBinCons1Config{}, nil, t)
}

func TestMvBinConsRnd1Basic(t *testing.T) {
	cons.RunBasicTests(types.TestOptions{SigType: types.EDCOIN, CoinType: types.StrongCoin2Type}, types.MvBinConsRnd1Type, &MvCons1{}, MvBinConsRnd1Config{}, nil, t)
}

func TestMvBinConsRand1Byz(t *testing.T) {
	cons.RunByzTests(types.TestOptions{SigType: types.EDCOIN, CoinType: types.StrongCoin2Type}, types.MvBinConsRnd1Type, &MvCons1{}, MvBinConsRnd1Config{}, nil, t)
}
