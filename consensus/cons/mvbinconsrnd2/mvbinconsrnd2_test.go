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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

// getBinConsStateMachineTypes returns a list of the valid state machine types for multi-value consensus given the configuration.
func getMvConsStateMachineTypes() []types.StateMachineType {
	if config.RunAllTests {
		return types.MultivalueProposerTypes
	}
	return []types.StateMachineType{types.CounterProposer}
}

var baseTO = types.TestOptions{EncryptChannels: true, NoSignatures: true, SigType: types.EDCOIN,
	CoinType: types.StrongCoin2Type, StopOnCommit: types.NextRound}
var fixedCoinTO = types.TestOptions{
	CoinType: types.FlipCoinType, StopOnCommit: types.NextRound}
var causalTO = types.TestOptions{OrderingType: types.Causal, EncryptChannels: true, NoSignatures: true,
	CoinType: types.FlipCoinType, StopOnCommit: types.NextRound}

func TestMvBinConsRnd2Basic(t *testing.T) {
	cons.RunBasicTests(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2BasicSleep(t *testing.T) {
	to := baseTO
	to.SleepCrypto = true
	cons.RunBasicTests(to, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2RandMC(t *testing.T) {
	cons.RunRandMCTests(fixedCoinTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, true,
		[]int{}, t)
}

func TestMvBinConsRnd2RandMCSleep(t *testing.T) {
	to := fixedCoinTO
	to.SleepCrypto = true
	cons.RunRandMCTests(to, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, true,
		[]int{}, t)
}

func TestMvBinConsRnd2Byz(t *testing.T) {
	cons.RunByzTests(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2MemStore(t *testing.T) {
	cons.RunMemstoreTest(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, nil, t)
}

func TestMvBinConsRnd2MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, nil, t)
}

func TestMvBinConsRnd2MultiSig(t *testing.T) {
	cons.RunMultiSigTests(fixedCoinTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2P2p(t *testing.T) {
	cons.RunP2pNwTests(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, nil, t)
}

func TestMvBinConsRnd2FailDisk(t *testing.T) {
	cons.RunFailureTests(baseTO, types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, nil, t)
}

func TestMvBinConsRnd2CausalBasic(t *testing.T) {
	cons.RunBasicTests(causalTO,
		types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2CausalBasicSleep(t *testing.T) {
	to := causalTO
	to.SleepCrypto = true
	cons.RunBasicTests(to,
		types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2CausalFailDisk(t *testing.T) {
	cons.RunFailureTests(causalTO,
		types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, []int{}, t)
}

func TestMvBinConsRnd2CausalRandMC(t *testing.T) {
	cons.RunRandMCTests(causalTO,
		types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, true, []int{}, t)
}

func TestMvBinConsRnd2CausalRandMCSleep(t *testing.T) {
	to := causalTO
	to.SleepCrypto = true
	cons.RunRandMCTests(to,
		types.MvBinConsRnd2Type, &MvBinConsRnd2{}, Config{}, true, []int{}, t)
}
