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
	"testing"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

func BenchmarkBinCons1TCP(b *testing.B) {
	to := types.TestOptions{ // No fails
		MaxRounds:          config.MaxBenchRounds,
		FailRounds:         0,
		NumFailProcs:       0,
		NumTotalProcs:      config.ProcCount,
		ClearDiskOnRestart: false,
		NetworkType:        types.AllToAll,
		ConnectionType:     types.TCP,
		StateMachineType:   types.BinaryProposer,
	}
	BenchBinCons1TrueMember(to, b)
}

func BenchmarkBinCons1UDP(b *testing.B) {
	to := types.TestOptions{ // No fails
		MaxRounds:          config.MaxBenchRounds,
		FailRounds:         0,
		NumFailProcs:       0,
		NumTotalProcs:      config.ProcCount,
		ClearDiskOnRestart: false,
		NetworkType:        types.AllToAll,
		ConnectionType:     types.UDP,
		StateMachineType:   types.BinaryProposer,
	}
	BenchBinCons1TrueMember(to, b)
}

func BenchmarkBinCons1P2p(b *testing.B) {
	to := types.TestOptions{ // No fails
		MaxRounds:          config.MaxBenchRounds,
		FailRounds:         0,
		NumFailProcs:       0,
		NumTotalProcs:      config.ProcCount,
		ClearDiskOnRestart: false,
		NetworkType:        types.P2p,
		FanOut:             config.FanOut,
		StateMachineType:   types.BinaryProposer,
	}
	BenchBinCons1TrueMember(to, b)
}

// BenchBinCons1TrueMember runs a local benchmark for BinCons1 using a TrueMemberChecker and the given configuration options.
func BenchBinCons1TrueMember(to types.TestOptions, b *testing.B) {
	BenchBinCons1(to, b)
}

// BenchBinCons1 runs a local benchmark for BinCons1 using the given configuration options.
func BenchBinCons1(to types.TestOptions, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initItem := &BinCons1{}
		cons.RunConsType(initItem, Config{}.GetBroadcastFunc(types.NonFaulty), Config{}, to, b)
	}
}
