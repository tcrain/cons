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
package cons

import (
	"testing"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
)

func BenchmarkSimpleCons(b *testing.B) {
	to := types.TestOptions{ // No fails
		StateMachineType:   types.TestProposer,
		MaxRounds:          config.MaxBenchRounds,
		FailRounds:         0,
		NumFailProcs:       0,
		NumTotalProcs:      config.ProcCount,
		ClearDiskOnRestart: false,
		NetworkType:        types.AllToAll,
	}
	BenchSimpleConsTrueMember(to, b)
}

func BenchmarkSimpleConsP2p(b *testing.B) {
	to := types.TestOptions{ // No fails
		StateMachineType:   types.TestProposer,
		MaxRounds:          config.MaxBenchRounds,
		FailRounds:         0,
		NumFailProcs:       0,
		NumTotalProcs:      config.ProcCount,
		ClearDiskOnRestart: false,
		NetworkType:        types.AllToAll,
		FanOut:             config.FanOut,
	}
	BenchSimpleConsTrueMember(to, b)
}

func BenchSimpleConsTrueMember(to types.TestOptions, b *testing.B) {
	BenchSimpleCons(to, b)
}

// BenchSimpleCons runs a local benchmark for SimpleCons using the given configuration options.
func BenchSimpleCons(to types.TestOptions, b *testing.B) {
	// Run the bench
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initItem := &SimpleCons{}
		RunConsType(initItem, SimpleConsConfig{}.GetBroadcastFunc(types.NonFaulty),
			SimpleConsConfig{}, to, b)
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
