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

package mvcons4

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

var mvTO = types.TestOptions{MCType: types.TrueMC, EncryptChannels: true}

func TestMvCons4BasicNormal(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Normal
	cons.RunBasicTests(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}

func TestMvCons4BasicDirect(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Direct
	cons.RunBasicTests(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}

func TestMvCons4BasicIndices(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Indices
	cons.RunBasicTests(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}

func TestMvCons4Byz(t *testing.T) {
	// TODO
	//	cons.RunByzTests(mvTO,
	//	types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}

func TestMvCons4MemStore(t *testing.T) {
	if !config.RunAllTests {
		return
	}

	cons.RunMemstoreTest(mvTO,
		types.MvCons4Type, &MvCons4{}, Config{}, nil, t)
}

func TestMvCons4MsgDropNormal(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Normal
	cons.RunMsgDropTest(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{0}, t)
}

func TestMvCons4MsgDropDirect(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Direct
	cons.RunMsgDropTest(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{0}, t)
}

func TestMvCons4MsgDropIndices(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Indices
	cons.RunMsgDropTest(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{0}, t)
}

func TestMvCons4FailDisk(t *testing.T) {
	cons.RunFailureTests(mvTO,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}

func TestMvCons4P2p(t *testing.T) {
	to := mvTO
	to.MvCons4BcastType = types.Normal
	cons.RunP2pNwTests(to,
		types.MvCons4Type, &MvCons4{}, Config{}, []int{}, t)
}
