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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func TestGenConfig(t *testing.T) {
	a := make(map[types.TestOptions]bool)
	i := NewAllConfigIter(AllTestConfig, types.TestOptions{})

	hasNxt := true
	var nxt types.TestOptions
	for hasNxt {
		nxt, hasNxt = i.Next()
		assert.False(t, a[nxt])
		a[nxt] = true
	}
	assert.Equal(t, AllTestConfig.ComputeMaxConfigs(), len(a))

	i = NewSingleIter(AllTestConfig, types.TestOptions{})
	a = make(map[types.TestOptions]bool)
	hasNxt = true
	for hasNxt {
		nxt, hasNxt = i.Next()
		assert.False(t, a[nxt])
		a[nxt] = true
	}
	// assert.Equal(t, SimpleConfig.ComputeMaxConfigs(), len(a))

	to := types.TestOptions{}
	to.StateMachineType = types.CounterProposer
	to.NumTotalProcs = 10
	i = NewSingleIter(AllTestConfig, to)
	j, err := NewTestOptIter(AllOptions, StandardMvConfig{}, i)
	assert.Nil(t, err)
	a = make(map[types.TestOptions]bool)

	prv, hasNxt := j.Next()
	for hasNxt {
		nxt, hasNxt = j.Next()
		assert.False(t, a[nxt])
		a[nxt] = true

		t.Log(prv.StringDiff(nxt))
		prv = nxt
	}

}
