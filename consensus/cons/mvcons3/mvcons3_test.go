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

package mvcons3

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

var mvTO = types.TestOptions{MCType: types.TrueMC, SigType: types.TBLS, IncludeProofs: true,
	CollectBroadcast: types.Commit}

func TestMvCons3Basic(t *testing.T) {
	cons.RunBasicTests(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, nil, t)
}

func TestMvCons3Byz(t *testing.T) {
	cons.RunByzTests(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, []int{}, t)
}

func TestMvCons3MemStore(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMemstoreTest(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, nil, t)
}

func TestMvCons3MsgDrop(t *testing.T) {
	cons.RunMsgDropTest(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, nil, t)
}

func TestMvCons3MultiSig(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMultiSigTests(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, []int{}, t)
}

func TestMvCons3SleepMultiSig(t *testing.T) {
	to := mvTO
	to.SleepCrypto = true
	cons.RunMultiSigTests(to,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, []int{}, t)
}

func TestMvCons3P2p(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunP2pNwTests(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, []int{}, t)
}

func TestMvCons3FailDisk(t *testing.T) {
	cons.RunFailureTests(mvTO,
		types.MvCons3Type, &MvCons3{}, MvCons3Config{}, []int{}, t)
}
