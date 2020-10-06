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

package mvcons2

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

func TestMvCons2Basic(t *testing.T) {
	cons.RunBasicTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, nil, t)
}

func TestMvCons2RandMC(t *testing.T) {
	// we dont run MvCons2 with local rand because it won't terminate since processes
	// stop as soon as they decide, but others might not decide that round
	cons.RunRandMCTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, false, []int{}, t)
}

func TestMvCons2Byz(t *testing.T) {
	cons.RunByzTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, []int{}, t)
}

func TestMvCons2MemStore(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMemstoreTest(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, nil, t)
}

func TestMvCons2MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, nil, t)
}

func TestMvCons2MultiSig(t *testing.T) {
	cons.RunMultiSigTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, []int{}, t)
}

func TestMvCons2P2p(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunP2pNwTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, nil, t)
}

func TestMvCons2FailDisk(t *testing.T) {
	cons.RunFailureTests(types.TestOptions{}, types.MvCons2Type, &MvCons2{}, MvCons2Config{}, nil, t)
}
