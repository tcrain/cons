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

package rbbcast2

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

var baseTO = types.TestOptions{EncryptChannels: true, NoSignatures: true}
var causalTO = types.TestOptions{OrderingType: types.Causal}

func TestRbBcast2Basic(t *testing.T) {
	cons.RunBasicTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, []int{}, t)
}

func TestRbBcast2RandMC(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunRandMCTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, true,
		[]int{}, t)
}

func TestRbBcast2RandMCSleep(t *testing.T) {
	to := baseTO
	to.SleepCrypto = true
	cons.RunRandMCTests(to, types.RbBcast2Type, &RbBcast2{}, Config{}, true,
		[]int{}, t)
}

func TestRbBcast2Byz(t *testing.T) {
	// TODO
	cons.RunByzTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2MemStore(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMemstoreTest(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2MultiSig(t *testing.T) {
	cons.RunMultiSigTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2P2p(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunP2pNwTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2FailDisk(t *testing.T) {
	cons.RunFailureTests(baseTO, types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2CausalBasic(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunBasicTests(causalTO,
		types.RbBcast2Type, &RbBcast2{}, Config{}, []int{}, t)
}

func TestRbBcast2CausalBasicSleep(t *testing.T) {
	to := causalTO
	to.SleepCrypto = true
	cons.RunBasicTests(to,
		types.RbBcast2Type, &RbBcast2{}, Config{}, []int{}, t)
}

func TestRbBcast2CausalFailDisk(t *testing.T) {
	cons.RunFailureTests(causalTO,
		types.RbBcast2Type, &RbBcast2{}, Config{}, nil, t)
}

func TestRbBcast2CausalRandMC(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunRandMCTests(causalTO,
		types.RbBcast2Type, &RbBcast2{}, Config{}, true, []int{}, t)
}

func TestRbBcast2CausalRandMCSleep(t *testing.T) {
	to := causalTO
	to.SleepCrypto = true
	cons.RunRandMCTests(to,
		types.RbBcast2Type, &RbBcast2{}, Config{}, true, []int{}, t)
}
