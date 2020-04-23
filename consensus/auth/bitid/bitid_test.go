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

package bitid

import (
	// "fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIterateBitID(t *testing.T) {
	t1Slices := []sort.IntSlice{{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 11, 12, 13, 14, 15, 16, 5000}, {1}}
	for _, t1 := range t1Slices {
		byt1, err := CreateMultiBitIDFromInts(t1)
		assert.Nil(t, err)

		iter := NewBitIDIterator()
		var iteritems sort.IntSlice
		for nxt, err := byt1.NextID(iter); err == nil; {
			iteritems = append(iteritems, nxt)
			nxt, err = byt1.NextID(iter)
		}
		assert.Equal(t, t1, iteritems)
	}

	t1Slices = []sort.IntSlice{{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 5000}, {1}}
	for _, t1 := range t1Slices {
		byt2, err := CreateBitIDFromInts(t1)
		assert.Nil(t, err)

		iter := NewBitIDIterator()
		var iteritems sort.IntSlice
		for nxt, err := byt2.NextID(iter); err == nil; {
			iteritems = append(iteritems, nxt)
			nxt, err = byt2.NextID(iter)
		}
		assert.Equal(t, t1, iteritems)
	}
}

func TestPBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 1, 2, 3, 4, 5, 6, 7, 29, 29, 30, 32, 37, 100, 100, 100, 100, 200, 300, 5000}
	byt1, err := CreatePbitidFromInts(t1)
	assert.Nil(t, err)
	assert.Equal(t, t1, byt1.GetItemList())
	assert.Equal(t, len(t1), byt1.GetNumItems())

	t1 = sort.IntSlice{6, 6, 6, 6, 7, 29, 29, 30, 32, 37, 100, 100, 100, 100, 200, 300, 5000}
	byt1, err = CreatePbitidFromInts(t1)
	assert.Nil(t, err)
	assert.Equal(t, t1, byt1.GetItemList())
	assert.Equal(t, len(t1), byt1.GetNumItems())

	// Test create from bytes
	byt2, err := CreatePbitIDFromBytes(byt1.buff[1:])
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), byt2.GetItemList())
	assert.Equal(t, byt1.GetNumItems(), byt2.GetNumItems())

	// Test copy from bytes
	byt3 := byt1.MakeCopy()
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), byt3.GetItemList())
	assert.Equal(t, byt1.GetNumItems(), byt3.GetNumItems())

	// Test decode
	// return
	// TODO finish test
	byt2, err = DecodePbitid(byt1.Encode())
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), byt2.GetItemList())
	assert.Equal(t, byt1.GetNumItems(), byt2.GetNumItems())
}

func TestEncodeMultiBitID(t *testing.T) {
	// test encoding
	t1 := sort.IntSlice{0, 1, 2, 3, 4, 5, 6, 7, 5000}
	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	enc := byt1.Encode()
	decBitID, err := DecodeMultiBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), decBitID.GetItemList(), t1)

	// test encoding
	t1 = sort.IntSlice{0, 1, 2, 3, 4, 5, 6, 7}
	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	enc = byt1.Encode()
	decBitID, err = DecodeMultiBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), decBitID.GetItemList(), t1)

	// test encoding with duplicates
	t1 = sort.IntSlice{0, 1, 1, 1, 2, 3, 3, 3, 4, 5, 6, 7}
	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	enc = byt1.Encode()
	decBitID, err = DecodeMultiBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), decBitID.GetItemList(), t1)

	// test encoding with duplicates
	t1 = sort.IntSlice{0, 0, 0, 1, 2, 2, 2, 3, 4, 5, 6, 7, 10, 10, 5000}
	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	enc = byt1.Encode()
	decBitID, err = DecodeMultiBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, byt1.GetItemList(), decBitID.GetItemList(), t1)
}

func TestBadMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 1, 1, 1, 2, 10, 3, 3, 3, 4, 5, 6, 7}
	_, err := CreateMultiBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{-1, 0, 1, 1, 1, 2, 3, 3, 3, 4, 5, 6, 7}
	_, err = CreateMultiBitIDFromInts(t1)
	assert.NotNil(t, err)
}

func TestMergeMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 1, 2, 3, 4, 4}
	t2 := sort.IntSlice{0, 2, 2, 13}

	megList := make(sort.IntSlice, len(t1)+len(t2))
	pos := copy(megList, t1)
	copy(megList[pos:], t2)
	sort.Sort(megList)

	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err := CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	mrgbyt := MergeMultiBitID(byt1, byt2)
	assert.Equal(t, mrgbyt.GetItemList(), megList)

	mrgbyt = MergeMultiBitIDNoDup(byt1, byt2)
	assert.Equal(t, mrgbyt.GetItemList(), sort.IntSlice{0, 1, 2, 3, 4, 13})
}

func TestGetNewItemsMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 0, 1, 2, 3, 4, 4}
	t2 := sort.IntSlice{0, 2, 2, 13, 13}

	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err := CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	newItems := byt1.GetNewItems(byt2)
	assert.Equal(t, sort.IntSlice{13}, newItems.GetItemList())

	t1 = sort.IntSlice{0}
	t2 = sort.IntSlice{0, 2, 2, 13}

	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err = CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	newItems = byt1.GetNewItems(byt2)
	assert.Equal(t, sort.IntSlice{2, 13}, newItems.GetItemList())
}

func TestSubMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 0, 1, 2, 3, 4, 4}
	t2 := sort.IntSlice{0, 2, 2, 13}

	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err := CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	subBid := SubMultiBitID(byt1, byt2)
	assert.Equal(t, sort.IntSlice{0, 0, 1, 3, 4, 4}, subBid.GetItemList())

	t1 = sort.IntSlice{0, 2}
	t2 = sort.IntSlice{0, 2, 2, 13}

	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err = CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	subBid = SubMultiBitID(byt1, byt2)
	assert.Equal(t, sort.IntSlice{}, subBid.GetItemList())

	t1 = sort.IntSlice{0, 2, 2, 13, 14, 15, 16}
	t2 = sort.IntSlice{0, 2, 2, 13}

	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err = CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	subBid = SubMultiBitID(byt1, byt2)
	assert.Equal(t, sort.IntSlice{14, 15, 16}, subBid.GetItemList())
}

func TestCheckIntersectionMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 1, 2, 3, 4, 4}
	t2 := sort.IntSlice{0, 2, 2, 13}

	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err := CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	intersect := byt1.CheckIntersection(byt2)
	assert.Equal(t, sort.IntSlice{0, 2}, intersect)

	t1 = sort.IntSlice{0, 0, 1, 2, 3, 4, 4}
	t2 = sort.IntSlice{0, 0, 2, 2, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13}

	byt1, err = CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)
	byt2, err = CreateMultiBitIDFromInts(t2)
	assert.Nil(t, err)

	intersect = byt1.CheckIntersection(byt2)
	assert.Equal(t, sort.IntSlice{0, 0, 2}, intersect)
}

func TestCheckMultiBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 1, 2, 3, 4, 4}

	byt1, err := CreateMultiBitIDFromInts(t1)
	assert.Nil(t, err)

	for _, nxt := range t1 {
		assert.True(t, byt1.CheckBitID(nxt))
	}

	t2 := sort.IntSlice{0, 0, 3, 5, 6, 6, 6}
	iter := NewBitIDIterator()
	for _, nxt := range t2 {
		if !byt1.CheckBitID(nxt) {
			assert.True(t, byt1.AddBitID(nxt, true, iter))
		} else {
			// assert.False(t, byt1.AddBitID(nxt, false, iter))
			assert.True(t, byt1.AddBitID(nxt, true, iter))
		}
	}

	for _, nxt := range t2 {
		assert.True(t, byt1.CheckBitID(nxt))
	}

	megList := make(sort.IntSlice, len(t1)+len(t2))
	pos := copy(megList, t1)
	copy(megList[pos:], t2)
	sort.Sort(megList)

	assert.Equal(t, megList, byt1.GetItemList())
}

func TestBadBitID(t *testing.T) {
	t1 := sort.IntSlice{0, 0, 1, 2, 3}
	_, err := CreateBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{0, -1, 2, 3}
	_, err = CreateBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{5, 1, 2, 3}
	_, err = CreateBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{-1, 2, 3}
	_, err = CreateBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{1, 2, -3}
	_, err = CreateBitIDFromInts(t1)
	assert.NotNil(t, err)

	t1 = sort.IntSlice{-1, 1, 2, 3}
	_, err = CreateBitIDFromInts(t1)
	assert.NotNil(t, err)
}

func TestMergeBitID(t *testing.T) {
	// TODO check if works for both endienness
	// should be ok since only maniuplating bytes?

	// test encoding
	t1 := sort.IntSlice{0, 1, 2, 3, 4, 5, 6, 7, 5000}
	byt1, err := CreateBitIDFromInts(t1)
	assert.Nil(t, err)
	enc := byt1.Encode()
	decBitID, err := DecodeBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, decBitID.Buff, byt1.Buff)
	assert.Equal(t, decBitID.GetItemList(), byt1.GetItemList())

	// all 8
	t1 = sort.IntSlice{0, 1, 2, 3, 4, 5, 6, 7}

	byt1, err = CreateBitIDFromInts(t1)
	assert.Nil(t, err)
	assert.Equal(t, (7+7)/8, len(byt1.Buff))
	assert.Equal(t, 255, int(byt1.Buff[0]))
	assert.NotEqual(t, byt1.Buff[len(byt1.Buff)-1], 0)

	arr1 := byt1.GetItemList()
	assert.Equal(t, t1, arr1)
	assert.Equal(t, len(t1), byt1.GetNumItems())

	assert.Equal(t, len(byt1.CheckIntersection(byt1)), byt1.GetNumItems())
	assert.Equal(t, byt1.CheckIntersection(byt1), byt1.GetItemList())

	for _, i := range t1 {
		assert.True(t, byt1.CheckBitID(i))
	}

	// testencoding
	enc = byt1.Encode()
	decBitID, err = DecodeBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, decBitID.Buff, byt1.Buff)
	assert.Equal(t, decBitID.GetItemList(), byt1.GetItemList())

	// add later 8
	t1later := sort.IntSlice{8, 9, 10, 11, 12, 13, 14, 15}
	byt1later, err := CreateBitIDFromInts(t1later)
	assert.Nil(t, err)
	assert.Equal(t, (15+7)/8, len(byt1later.Buff))
	assert.Equal(t, 255, int(byt1later.Buff[1]))
	assert.NotEqual(t, byt1later.Buff[len(byt1later.Buff)-1], 0)

	// test new items
	newItems := byt1.GetNewItems(byt1later)
	assert.Equal(t, t1later, newItems.GetItemList())

	// merge
	byt1merge := MergeBitID(byt1, byt1later)
	assert.Equal(t, (15+7)/8, len(byt1merge.Buff))
	assert.Equal(t, 255, int(byt1merge.Buff[1]))
	assert.NotEqual(t, byt1merge.Buff[len(byt1merge.Buff)-1], 0)

	// testencoding
	enc = byt1merge.Encode()
	decBitID, err = DecodeBitID(enc)
	assert.Nil(t, err)
	assert.Equal(t, decBitID.Buff, byt1merge.Buff)
	assert.Equal(t, decBitID.GetItemList(), byt1merge.GetItemList())

	// sub
	byt1sub := SubBitID(byt1merge, byt1)
	assert.Equal(t, byt1later.Buff, byt1sub.Buff)

	// merge 1 by 1
	iter := NewBitIDIterator()
	for _, i := range t1later {
		assert.False(t, byt1.CheckBitID(i))
		assert.True(t, byt1.AddBitID(i, false, iter))
		assert.NotEqual(t, byt1.Buff[len(byt1.Buff)-1], 0)
		assert.True(t, byt1.CheckBitID(i))

		assert.Equal(t, len(byt1.CheckIntersection(byt1)), byt1.GetNumItems())
		assert.Equal(t, byt1.CheckIntersection(byt1), byt1.GetItemList())
	}

	assert.Equal(t, append(t1, t1later...), byt1.GetItemList())
	assert.Equal(t, append(t1, t1later...), byt1merge.GetItemList())
	assert.Equal(t, len(byt1.CheckIntersection(byt1merge)), byt1.GetNumItems())
	assert.Equal(t, byt1.CheckIntersection(byt1merge), byt1.GetItemList())
}

func TestEmptyBitID(t *testing.T) {

	// empty
	var t2 sort.IntSlice
	byt2, err := CreateBitIDFromInts(t2)
	assert.Nil(t, err)
	assert.Nil(t, byt2.Buff)

	arr2 := byt2.GetItemList()
	assert.Equal(t, len(t2), len(arr2))
	assert.Equal(t, 0, byt2.GetNumItems())
}

func TestLargeBitID(t *testing.T) {

	// some randoms
	rand.Seed(1234)
	allt3 := (sort.IntSlice)(rand.Perm(10000))
	t3 := make(sort.IntSlice, 1000)
	copy(t3, allt3[:1000])
	remaint3 := make(sort.IntSlice, len(allt3)-len(t3))
	copy(remaint3, allt3[1000:])

	t3.Sort()
	remaint3.Sort()

	// all items
	allt3.Sort()
	allbyt, err := CreateBitIDFromInts(allt3)
	assert.Nil(t, err)
	assert.Equal(t, (7+allt3[len(allt3)-1])/8, len(allbyt.Buff))
	assert.NotEqual(t, allbyt.Buff[len(allbyt.Buff)-1], 0)

	// create the first 10000
	byt3, err := CreateBitIDFromInts(t3)
	assert.Nil(t, err)
	assert.Equal(t, (7+t3[len(t3)-1])/8, len(byt3.Buff))
	assert.NotEqual(t, byt3.Buff[len(byt3.Buff)-1], 0)
	assert.Equal(t, len(byt3.CheckIntersection(allbyt)), byt3.GetNumItems())
	assert.Equal(t, byt3.CheckIntersection(allbyt), byt3.GetItemList())

	// test get new
	newItems := allbyt.GetNewItems(byt3)
	assert.Equal(t, sort.IntSlice{}, newItems.GetItemList())

	arr3 := byt3.GetItemList()
	assert.True(t, sort.IsSorted((sort.IntSlice)(arr3)))
	assert.Equal(t, t3, arr3)
	assert.Equal(t, len(t3), byt3.GetNumItems())

	for _, i := range t3 {
		assert.True(t, byt3.CheckBitID(i))
	}

	// add the remaining 1 by one
	iter := NewBitIDIterator()
	for _, i := range remaint3 {
		assert.False(t, byt3.CheckBitID(i))
		assert.True(t, byt3.AddBitID(i, false, iter))
		assert.True(t, byt3.CheckBitID(i))
		assert.NotEqual(t, byt3.Buff[len(byt3.Buff)-1], 0)

		assert.Equal(t, len(byt3.CheckIntersection(allbyt)), byt3.GetNumItems())
		assert.Equal(t, byt3.CheckIntersection(allbyt), byt3.GetItemList())
	}
	assert.Equal(t, allt3, byt3.GetItemList())
	assert.Equal(t, len(allt3), byt3.GetNumItems())

	// Do the creation from the bytes
	cpy := make([]byte, len(byt3.Buff))
	copy(cpy, byt3.Buff)
	byt4, err := CreateBitIDFromBytes(byt3.Buff)
	assert.Nil(t, err)
	assert.Equal(t, byt3.Buff, byt4.Buff)
	assert.Equal(t, byt3.GetStr(), byt4.GetStr())
	assert.Equal(t, byt3.GetItemList(), byt4.GetItemList())
	assert.Equal(t, len(allt3), byt4.GetNumItems())

	// Test new
	n1 := sort.IntSlice{1, 2}
	n2 := sort.IntSlice{0, 2}

	nb1, err := CreateBitIDFromInts(n1)
	assert.Nil(t, err)
	nb2, err := CreateBitIDFromInts(n2)
	assert.Nil(t, err)

	newItems = nb1.GetNewItems(nb2)
	assert.Equal(t, sort.IntSlice{0}, newItems.GetItemList())
}
