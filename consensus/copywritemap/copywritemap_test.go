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
package copywritemap

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func assertMapContains(t *testing.T, m *CopyWriteMap, key interface{}, value interface{}) {
	v, ok := m.Read(key)
	assert.True(t, ok)
	assert.Equal(t, value, v)
}

func assertNotInMap(t *testing.T, m *CopyWriteMap, key interface{}) {
	v, ok := m.Read(key)
	assert.Nil(t, v)
	assert.False(t, ok)
}

func rangeCheck(t *testing.T, m *CopyWriteMap, count int) {
	var c int
	m.Range(func(k, v interface{}) bool {
		c++
		assert.Equal(t, k, v)
		return true
	})
	assert.Equal(t, count, c)
}

func TestCopyWriteMap(t *testing.T) {
	m1 := NewCopyWriteMap()
	m1.Write(1, 1)
	assertMapContains(t, m1, 1, 1)

	m2 := m1.Copy()
	assertMapContains(t, m1, 1, 1)
	assertMapContains(t, m2, 1, 1)

	m2.Write(2, 2)
	assertMapContains(t, m1, 1, 1)
	assertMapContains(t, m2, 1, 1)
	assertNotInMap(t, m1, 2)
	assertMapContains(t, m2, 2, 2)

	m3 := m2.Copy()
	m3.Write(3, 3)
	assertMapContains(t, m1, 1, 1)
	assertMapContains(t, m2, 1, 1)
	assertMapContains(t, m3, 1, 1)
	assertNotInMap(t, m1, 2)
	assertMapContains(t, m2, 2, 2)
	assertMapContains(t, m3, 2, 2)
	assertNotInMap(t, m1, 3)
	assertNotInMap(t, m2, 3)
	assertMapContains(t, m3, 3, 3)
	rangeCheck(t, m1, 1)
	rangeCheck(t, m2, 2)
	rangeCheck(t, m3, 3)

	m3.Delete(1)
	assertMapContains(t, m1, 1, 1)
	assertMapContains(t, m2, 1, 1)
	assertNotInMap(t, m3, 1)
	assertNotInMap(t, m1, 2)
	assertMapContains(t, m2, 2, 2)
	assertMapContains(t, m3, 2, 2)
	assertNotInMap(t, m1, 3)
	assertNotInMap(t, m2, 3)
	assertMapContains(t, m3, 3, 3)
	rangeCheck(t, m1, 1)
	rangeCheck(t, m2, 2)
	rangeCheck(t, m3, 2)

	m1.DoneKeep()
	assert.Equal(t, doneKeep, m1.done)
	assertMapContains(t, m2, 1, 1)
	assertNotInMap(t, m3, 1)
	assertMapContains(t, m2, 2, 2)
	assertMapContains(t, m3, 2, 2)
	assertNotInMap(t, m2, 3)
	assertMapContains(t, m3, 3, 3)

	m2.DoneKeep()
	assert.Equal(t, doneKeep, m2.done)
	assertNotInMap(t, m3, 1)
	assertMapContains(t, m3, 2, 2)
	assertMapContains(t, m3, 3, 3)

	m3.DoneKeep()
	assert.Equal(t, doneKeep, m3.done)

	m4 := m3.Copy()
	m5 := m3.Copy()

	for i := 1; i <= 3; i++ {
		assertMapContains(t, m4, i, i)
		assertMapContains(t, m5, i, i)
	}
	m4.Write(4, 4)
	assertMapContains(t, m4, 4, 4)
	assertNotInMap(t, m5, 4)

	m5.Write(5, 5)
	assertMapContains(t, m5, 5, 5)
	assertNotInMap(t, m4, 5)

	m5.Write(5, 6)
	assertMapContains(t, m5, 5, 6)
	assertNotInMap(t, m4, 5)
	m4.Write(5, 3)
	assertMapContains(t, m5, 5, 6)
	assertMapContains(t, m4, 5, 3)

	m6 := m5.Copy()

	m4.DoneClear()
	assert.Equal(t, doneClearFinish, m4.done)
	assertNotInMap(t, m5, 4)

	m5.DoneClear()
	assert.Equal(t, doneClearStart, m5.done)

	m6.DoneClear()
	assert.Equal(t, doneClearFinish, m5.done)
	assert.Equal(t, doneClearFinish, m6.done)
}
