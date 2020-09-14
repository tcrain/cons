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

package rbbcast1

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

func TestRbBcast1Basic(t *testing.T) {
	to := types.TestOptions{}
	cons.RunBasicTests(to, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, []int{}, t)
}

func TestRbBcast1CollectBcast(t *testing.T) {
	to := types.TestOptions{}
	to.CollectBroadcast = types.Commit
	cons.RunBasicTests(to, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, []int{}, t)
}

func TestRbBcast1RandMC(t *testing.T) {
	cons.RunRandMCTests(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, false, nil, t)
}

func TestRbBcast1Byz(t *testing.T) {
	// TODO
	cons.RunByzTests(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1MemStore(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMemstoreTest(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1MultiSig(t *testing.T) {
	cons.RunMultiSigTests(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1P2p(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunP2pNwTests(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1FailDisk(t *testing.T) {
	cons.RunFailureTests(types.TestOptions{}, types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1CausalBasic(t *testing.T) {
	cons.RunBasicTests(types.TestOptions{OrderingType: types.Causal},
		types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1CausalFailDisk(t *testing.T) {
	cons.RunFailureTests(types.TestOptions{OrderingType: types.Causal},
		types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, nil, t)
}

func TestRbBcast1CausalRandMC(t *testing.T) {
	cons.RunRandMCTests(types.TestOptions{OrderingType: types.Causal},
		types.RbBcast1Type, &RbBcast1{}, RbBcast1Config{}, false, nil, t)
}
