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

package utils

import (
	"bytes"
	"encoding/binary"
	"github.com/tcrain/cons/consensus/types"
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBitAt(t *testing.T) {
	assert.Equal(t, types.BinVal(0), GetBitAt(0, 0))
	assert.Equal(t, types.BinVal(1), GetBitAt(63, 1<<63))
	assert.Equal(t, types.BinVal(0), GetBitAt(0, 1<<63))
	assert.Equal(t, types.BinVal(1), GetBitAt(0, 1))
}

func TestCheckOverflow(t *testing.T) {
	assert.False(t, CheckOverflow(math.MaxUint64-1, 1))
	assert.False(t, CheckOverflow(math.MaxUint64))
	assert.False(t, CheckOverflow(math.MaxUint64/2, math.MaxUint64/2))
	assert.False(t, CheckOverflow(math.MaxUint64/2, math.MaxUint64/2, 1))
	assert.True(t, CheckOverflow(math.MaxUint64/2, math.MaxUint64/2, 1, 1))
}

func TestGenList(t *testing.T) {
	assert.Equal(t, []int{0, 1, 2, 3, 4}, GenList(5))
	assert.Equal(t, []int{}, GenList(0))
	assert.Equal(t, []int{0}, GenList(1))
}

func TestMovingAvg(t *testing.T) {
	var v int
	ma := NewMovingAvg(10, 0)
	v = ma.AddElement(0)
	assert.Equal(t, 0, v)
	for i := 0; i < 5; i++ {
		v = ma.AddElement(20)
	}
	assert.Equal(t, 10, v)
	for i := 0; i < 5; i++ {
		v = ma.AddElement(20)
	}
	assert.Equal(t, 20, v)

	ma = NewMovingAvg(10, 10)
	for i := 0; i < 5; i++ {
		v = ma.AddElement(20)
	}
	assert.Equal(t, 15, v)
	for i := 0; i < 5; i++ {
		v = ma.AddElement(20)
	}
	assert.Equal(t, 20, v)
}

func TestSortSortedList(t *testing.T) {
	l1 := sort.IntSlice{1, 3, 5, 123, 144, 144}
	l2 := sort.IntSlice{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 5, 5, 5, 6, 7, 8}
	l3 := make(sort.IntSlice, len(l1)+len(l2))
	n := copy(l3, l1)
	copy(l3[n:], l2)
	sort.Sort(l3)

	l4 := SortSortedList(l1, l2)
	assert.Equal(t, l3, l4)

	l5 := SortSortedList(l1, nil)
	assert.Equal(t, l1, l5)

	l6 := SortSortedList(nil, l2)
	assert.Equal(t, l2, l6)

}

func TestSortSorted(t *testing.T) {
	l1 := sort.IntSlice{1, 3, 5, 123, 144, 144}
	l2 := sort.IntSlice{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 5, 5, 5, 6, 7, 8}
	l3 := make(sort.IntSlice, len(l1)+len(l2))
	n := copy(l3, l1)
	copy(l3[n:], l2)
	sort.Sort(l3)

	l4 := SortSorted(l1, l2)
	assert.Equal(t, l3, l4)

	l5 := SortSorted(l1, nil)
	assert.Equal(t, l1, l5)

	l6 := SortSorted(nil, l2)
	assert.Equal(t, l2, l6)
}

func TestSortSortedNoDuplicates(t *testing.T) {
	l1 := sort.IntSlice{1, 3, 5, 123, 144, 144}
	l1no := sort.IntSlice{1, 3, 5, 123, 144}
	l2 := sort.IntSlice{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 5, 5, 5, 6, 7, 8}
	l2no := sort.IntSlice{0, 1, 5, 6, 7, 8}
	l3no := sort.IntSlice{0, 1, 3, 5, 6, 7, 8, 123, 144}

	l4 := SortSortedNoDuplicates(l1, l2)
	assert.Equal(t, l3no, l4)

	l5 := SortSortedNoDuplicates(l1, nil)
	assert.Equal(t, l1no, l5)

	l6 := SortSortedNoDuplicates(nil, l2)
	assert.Equal(t, l2no, l6)
}

func TestInsertInSortedSlice(t *testing.T) {
	l1 := sort.IntSlice{1, 3, 5, 123, 144, 144}
	toInsert := []int{0, 124, 145}

	mySlice := make(sort.IntSlice, len(l1))
	copy(mySlice, l1)

	for _, nxt := range toInsert {
		l1 = append(l1, nxt)
		sort.Sort(l1)

		mySlice, _ = InsertInSortedSlice(mySlice, nxt, 0)

		assert.Equal(t, l1, mySlice)
	}
}

func TestCheckUnique(t *testing.T) {
	assert.True(t, CheckUnique(1))
	assert.True(t, CheckUnique(1, 2))
	assert.True(t, CheckUnique(1, 2, 3))

	assert.True(t, CheckUnique("a"))
	assert.True(t, CheckUnique("a", "aa"))
	assert.True(t, CheckUnique("a", "aa", "aaa"))

	assert.False(t, CheckUnique(1, 1))
	assert.False(t, CheckUnique(1, 2, 1))
	assert.False(t, CheckUnique(1, 2, 3, 1))

	assert.False(t, CheckUnique("a", "a"))
	assert.False(t, CheckUnique("a", "aa", "a"))
	assert.False(t, CheckUnique("a", "aa", "aaa", "a"))

	var lst []Any
	for i := 0; i < 100; i++ {
		lst = append(lst, i)
	}
	assert.True(t, CheckUnique(lst...))
	lst = append(lst, 0)
	assert.False(t, CheckUnique(lst...))
}

func TestGenRandPerm(t *testing.T) {
	rnd := rand.New(rand.NewSource(1234))

	until := 20
	for i := 1; i < until; i++ {
		for j := i; j < until; j++ {
			perm := GenRandPerm(i, j, rnd)
			assert.True(t, MaxIntSlice(perm...) < j)
			assert.Equal(t, i, len(perm))

			res := make([]Any, len(perm))
			for idx, val := range perm {
				res[idx] = val
			}
			assert.True(t, CheckUnique(res...))
		}
	}
}

func TestEncodeUvarint(t *testing.T) {
	byt := make([]byte, binary.MaxVarintLen64)
	b := bytes.NewBuffer(nil)
	for i := uint64(0); i < 1000000; i = i * 2 {
		var n int
		var err error
		for j := 0; j < 3; j++ {
			n, err = EncodeUvarint(i, b)
			assert.Nil(t, err)
			n1 := binary.PutUvarint(byt, i)
			assert.Equal(t, n, n1)
		}

		for j := 0; j < 3; j++ {
			v, n2, err := ReadUvarint(b)
			assert.Nil(t, err)
			assert.Equal(t, i, v)
			assert.Equal(t, n, n2)
		}

		i += 1
	}

}

func TestIncreaseCap(t *testing.T) {
	start := []byte("some string")

	other := IncreaseCap(start, cap(start)+20)
	assert.Equal(t, start, other)
	assert.Equal(t, cap(start)+20, cap(other))

	other = IncreaseCap(start, cap(start)-5)
	assert.Equal(t, start, other)
	assert.Equal(t, cap(start), cap(other))
}
