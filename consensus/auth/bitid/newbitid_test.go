package bitid

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sort"
	"testing"
)

func TestAddSet(t *testing.T) {
	testAddSet(NewSliceBitIDFromInts, t)
	testAddSet(NewUvarintBitIDFromInts, t)
	testAddSet(NewMultiBitIDFromInts, t)
	testAddSet(NewBitIDFromInts, t)
	testAddSet(NewUvarintBitIDFromInts, t)
}

func testAddSet(intFunc FromIntFunc, t *testing.T) {
	count := 10

	items := make([]NewBitIDInterface, count)
	ints := make(sort.IntSlice, count)
	for i := range items {
		items[i] = intFunc(sort.IntSlice{i})
		ints[i] = i
	}
	b1, err := AddHelperSet(true, true, intFunc(nil).New, items...)
	assert.Nil(t, err)
	assert.Equal(t, ints, b1.GetItemList())

	items = make([]NewBitIDInterface, count)
	intsDub := make(sort.IntSlice, 0, count)
	for i := range items {
		items[i] = intFunc(sort.IntSlice{i, i})
		intsDub = append(intsDub, i, i)
	}
	if intFunc(nil).AllowsDuplicates() {
		b1, err = AddHelperSet(true, false, intFunc(nil).New, items...)
		assert.Nil(t, err)
		assert.Equal(t, intsDub, b1.GetItemList())
		b1, err = AddHelperSet(false, true, intFunc(nil).New, items...)
		assert.NotNil(t, err)
	}
	b1, err = AddHelperSet(false, false, intFunc(nil).New, items...)
	assert.Nil(t, err)
	assert.Equal(t, ints, b1.GetItemList())
}

func TestAdd(t *testing.T) {
	testAdd(NewSliceBitIDFromInts, NewSliceBitIDFromInts, t)
	testAdd(NewUvarintBitIDFromInts, NewUvarintBitIDFromInts, t)
	testAdd(NewMultiBitIDFromInts, NewMultiBitIDFromInts, t)
	testAdd(NewBitIDFromInts, NewBitIDFromInts, t)
	testAdd(NewUvarintBitIDFromInts, NewSliceBitIDFromInts, t)
}

func testAdd(intFunc1, intFunc2 FromIntFunc, t *testing.T) {

	s1 := sort.IntSlice{1, 2, 8, 10}
	s2 := sort.IntSlice{1, 2, 8, 10}
	s3 := utils.SortSorted(s1, s2)
	s4 := utils.SortSortedNoDuplicates(s1, s2)
	bid1 := intFunc1(s1)
	bid2 := intFunc2(s2)

	bid3, err := AddHelper(bid1, bid2, true, true, bid1.New, nil)
	assert.Equal(t, types.ErrIntersectingBitIDs, err)
	bid3, err = AddHelper(bid1, bid2, true, true, bid1.New, nil)
	assert.Equal(t, types.ErrIntersectingBitIDs, err)

	bid3, err = AddHelperSet(true, true, bid1.New, bid1, bid2)
	assert.Equal(t, types.ErrIntersectingBitIDs, err)
	bid3, err = AddHelper(bid1, bid2, true, true, bid1.New, nil)
	assert.Equal(t, types.ErrIntersectingBitIDs, err)

	bid3, err = AddHelper(bid1, bid2, true, false, bid1.New, nil)
	assert.Nil(t, err)
	switch bid3.AllowsDuplicates() {
	case true:
		checkSli(t, s3, bid3)
	case false:
		checkSli(t, s4, bid3)
	}
	bid3, err = AddHelper(bid1, bid2, false, false, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s1, bid3)
	bid3, err = AddHelper(bid1, bid2.New(), false, false, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s1, bid3)
	bid3, err = AddHelper(bid1.New(), bid2, false, false, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s1, bid3)
	bid3, err = AddHelper(bid1.New(), bid2.New(), false, false, bid1.New, nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(bid3.GetItemList()))

	s1 = sort.IntSlice{2, 8, 9, 10, 20, 24}
	s2 = sort.IntSlice{1, 2, 8, 10}
	s3 = utils.SortSorted(s1, s2)
	s4 = utils.SortSortedNoDuplicates(s1, s2)
	bid1 = intFunc1(s1)
	bid2 = intFunc2(s2)

	bid3, err = AddHelper(bid1, bid2, true, false, bid1.New, nil)
	assert.Nil(t, err)
	switch bid3.AllowsDuplicates() {
	case true:
		checkSli(t, s3, bid3)
	case false:
		checkSli(t, s4, bid3)
	}
	bid3, err = AddHelper(bid1, bid2, false, false, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s4, bid3)

	s1 = sort.IntSlice{9, 10, 20, 24}
	s2 = sort.IntSlice{1, 2, 8}
	s3 = utils.SortSorted(s1, s2)
	bid1 = intFunc1(s1)
	bid2 = intFunc2(s2)

	bid3, err = AddHelper(bid1, bid2, true, true, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s3, bid3)
	bid3, err = AddHelper(bid1, bid2, false, true, bid1.New, nil)
	assert.Nil(t, err)
	checkSli(t, s4, bid3)

}

func TestGetNewItems(t *testing.T) {
	testGetNewItems(NewSliceBitIDFromInts, NewSliceBitIDFromInts, t)
	testGetNewItems(NewUvarintBitIDFromInts, NewUvarintBitIDFromInts, t)
	testGetNewItems(NewMultiBitIDFromInts, NewMultiBitIDFromInts, t)
	testGetNewItems(NewBitIDFromInts, NewBitIDFromInts, t)
	testGetNewItems(NewUvarintBitIDFromInts, NewSliceBitIDFromInts, t)
}

func testGetNewItems(intFunc1, intFunc2 FromIntFunc, t *testing.T) {

	// hasDuplicates := []bool{false, false, true}
	s1 := []sort.IntSlice{{1, 2, 8, 10}, {1, 2, 8, 10}, {4, 8, 8}}
	s2 := []sort.IntSlice{{1, 2, 8, 10}, {}, {2, 2, 4, 4, 7, 8, 10}}
	s3 := [][2]sort.IntSlice{{{}, {}}, {{}, {1, 2, 8, 10}}, {{2, 7, 10}, {}}}

	for i := range s1 {
		bid1 := intFunc1(s1[i])
		bid2 := intFunc2(s2[i])

		bid3 := GetNewItemsHelper(bid1, bid2, bid1.New)
		checkSli(t, s3[i][0], bid3)
		assert.Equal(t, len(s3[i][0]) > 0, HasNewItemsHelper(bid1, bid2))

		bid3 = GetNewItemsHelper(bid2, bid1, bid2.New)
		checkSli(t, s3[i][1], bid3)
		assert.Equal(t, len(s3[i][1]) > 0, HasNewItemsHelper(bid2, bid1))
	}
}

func TestHasNewItems(t *testing.T) {
	testHasNewItems(NewSliceBitIDFromInts, NewSliceBitIDFromInts, t)
	testHasNewItems(NewUvarintBitIDFromInts, NewUvarintBitIDFromInts, t)
	testHasNewItems(NewMultiBitIDFromInts, NewMultiBitIDFromInts, t)
	testHasNewItems(NewBitIDFromInts, NewBitIDFromInts, t)
	testHasNewItems(NewUvarintBitIDFromInts, NewSliceBitIDFromInts, t)
}

func testHasNewItems(intFunc1, intFunc2 FromIntFunc, t *testing.T) {

	// hasDuplicates := []bool{false, false, true}
	s1 := []sort.IntSlice{{1, 2, 8, 10}, {1, 2, 8, 10}, {4, 8, 8}, {}, {1, 2, 4}, {3}}
	s2 := []sort.IntSlice{{1, 2, 8, 10}, {}, {2, 2, 4, 4, 7, 8, 10}, {1}, {2, 4, 6, 6, 7}, {}}
	s3 := []sort.IntSlice{{1, 2, 8, 10}, {1, 2, 8, 10}, {2, 4, 4, 7, 8, 10}, {1, 2}, {0, 0, 1, 3}, {2}}
	s4 := []bool{false, false, false, true, true, true}

	for i := range s1 {
		bid1 := intFunc1(s1[i])
		bid2 := intFunc2(s2[i])
		bid3 := intFunc2(s3[i])

		assert.Equal(t, s4[i], HasNewItemsBothHelper(bid1, bid2, bid3))
		assert.Equal(t, s4[i], HasNewItemsBothHelper(bid2, bid1, bid3))
	}
}

func TestInterection(t *testing.T) {
	testIntersection(NewSliceBitIDFromInts, NewSliceBitIDFromInts, t)
	testIntersection(NewUvarintBitIDFromInts, NewUvarintBitIDFromInts, t)
	testIntersection(NewMultiBitIDFromInts, NewMultiBitIDFromInts, t)
	testIntersection(NewBitIDFromInts, NewBitIDFromInts, t)
	testIntersection(NewUvarintBitIDFromInts, NewSliceBitIDFromInts, t)
}

func testIntersection(intFunc1, intFunc2 FromIntFunc, t *testing.T) {

	s1 := []sort.IntSlice{{1, 2, 8, 10}, {1, 2, 8, 10}, {4, 4, 8, 8}}
	s2 := []sort.IntSlice{{1, 2, 8, 10}, {}, {2, 2, 4, 4, 7, 8, 10}}
	s3 := []sort.IntSlice{{1, 2, 8, 10}, nil, {4, 4, 8}}

	for i := range s1 {
		bid1 := intFunc1(s1[i])
		bid2 := intFunc2(s2[i])
		tmp := s3[i]
		if !bid1.AllowsDuplicates() && len(s3[i]) > 0 {
			s3[i] = utils.SortSortedNoDuplicates(s3[i], nil)
		}

		bid3 := CheckIntersectionHelper(bid1, bid2)
		assert.Equal(t, s3[i], bid3)
		assert.Equal(t, len(s3[i]) > 0, HasIntersectionHelper(bid1, bid2))
		assert.Equal(t, len(s3[i]), HasIntersectionCountHelper(bid1, bid2))

		bid3 = CheckIntersectionHelper(bid2, bid1)
		assert.Equal(t, s3[i], bid3)
		assert.Equal(t, len(s3[i]) > 0, HasIntersectionHelper(bid2, bid1))
		assert.Equal(t, len(s3[i]), HasIntersectionCountHelper(bid2, bid1))
		s3[i] = tmp
	}
}

func TestSub(t *testing.T) {
	testSub(NewSliceBitIDFromInts, NewSliceBitIDFromInts, t)
	testSub(NewUvarintBitIDFromInts, NewUvarintBitIDFromInts, t)
	testSub(NewMultiBitIDFromInts, NewMultiBitIDFromInts, t)
	testSub(NewBitIDFromInts, NewBitIDFromInts, t)
	testSub(NewUvarintBitIDFromInts, NewSliceBitIDFromInts, t)
}

func testSub(intFunc1, intFunc2 FromIntFunc, t *testing.T) {

	s1 := []sort.IntSlice{{1, 2, 8, 10}, {1, 2, 8, 10}, {4, 8, 8}}
	s2 := []sort.IntSlice{{1, 2, 8, 10}, {}, {2, 2, 4, 4, 7, 8, 10}}
	// s3 := [][2]sort.IntSlice{{{}, {}}, {{1, 2, 8, 10}, {}}, {{8}, {2, 2, 4, 7, 10}}}
	//errs := [][2]error{{types.ErrInvalidBitIDSub, types.ErrInvalidBitIDSub}, {nil, types.ErrInvalidBitIDSub},
	//	{types.ErrInvalidBitIDSub, types.ErrInvalidBitIDSub}}

	for i := range s1 {
		tmp1 := s1[i]
		if !intFunc1(nil).AllowsDuplicates() && len(s1[i]) > 0 {
			s1[i] = utils.SortSortedNoDuplicates(s1[i], nil)
		}
		tmp2 := s2[i]
		if !intFunc2(nil).AllowsDuplicates() && len(s2[i]) > 0 {
			s2[i] = utils.SortSortedNoDuplicates(s2[i], nil)
		}
		bid1 := intFunc1(s1[i])
		bid2 := intFunc2(s2[i])
		s3 := []sort.IntSlice{utils.SubSortedSlice(s1[i], s2[i]), utils.SubSortedSlice(s2[i], s1[i])}

		bid3, err := SubHelper(bid1, bid2, NotSafe, bid1.New, nil)
		assert.Nil(t, err)
		checkSli(t, s3[0], bid3)
		bid3, err = SubHelper(bid2, bid1, NotSafe, bid2.New, nil)
		assert.Nil(t, err)
		checkSli(t, s3[1], bid3)

		var checkErr error
		if !utils.ContainsSlice(s1[i], s2[i]) || (utils.ContainsSlice(s1[i], s2[i]) && len(s1[i]) == len(s2[i])) {
			checkErr = types.ErrInvalidBitIDSub
		}
		bid3, err = SubHelper(bid1, bid2, NonIntersectingAndEmpty, bid1.New, nil)
		assert.Equal(t, checkErr, err)
		if checkErr == nil {
			checkSli(t, s3[0], bid3)
		}

		checkErr = nil
		if !utils.ContainsSlice(s2[i], s1[i]) || (utils.ContainsSlice(s2[i], s1[i]) && len(s1[i]) == len(s2[i])) {
			checkErr = types.ErrInvalidBitIDSub
		}
		bid3, err = SubHelper(bid2, bid1, NonIntersectingAndEmpty, bid2.New, nil)
		assert.Equal(t, checkErr, err)
		if checkErr == nil {
			checkSli(t, s3[1], bid3)
		}
		s1[i] = tmp1
		s2[i] = tmp2
	}
}

func TestEncode(t *testing.T) {
	testEncode(NewSliceBitIDFromInts, t)
	testEncode(NewMultiBitIDFromInts, t)
	testEncode(NewUvarintBitIDFromInts, t)
	testEncode(NewBitIDFromInts, t)
}

func testEncode(intFunc1 FromIntFunc, t *testing.T) {
	sli := []sort.IntSlice{{}, {0}, {0, 100, 300}, {100, 122, 156, 199, 900}}

	for _, nxt := range sli {
		bid1 := intFunc1(nxt)
		checkSli(t, nxt, bid1)
		buff := bytes.NewBuffer(nil)

		n1, err := bid1.Encode(buff)
		assert.Nil(t, err)
		byt := buff.Bytes()
		buff2 := bytes.NewBuffer(byt[:len(byt)-1])

		bid2 := bid1.New()
		n2, err := bid2.Decode(buff)
		assert.Nil(t, err)
		assert.Equal(t, n1, n2)
		checkSli(t, nxt, bid2)

		bid3 := bid1.New()
		_, err = bid3.Decode(buff2)
		assert.NotNil(t, err)
	}
}

func checkSli(t *testing.T, sli sort.IntSlice, bid NewBitIDInterface) {
	itms := bid.GetItemList()
	if itms == nil {
		itms = sort.IntSlice{}
	}
	assert.Equal(t, sli, itms)
	min, max, count, unique := bid.GetBasicInfo()
	var sMin, sMax, sCount int
	sCount = len(sli)
	if sCount > 0 {
		sMin = sli[0]
		sMax = sli[sCount-1]
	}
	assert.Equal(t, sMin, min)
	assert.Equal(t, sMax, max)
	assert.Equal(t, sCount, count)
	assert.Equal(t, utils.GetUniqueCount(sli), unique)
}

func TestPool(t *testing.T) {
	checkPool(t, NewUvarintBitIDFromInts)
}

func checkPool(t *testing.T, intFunc FromIntFunc) {
	for _, concurrent := range []bool{false, true} {
		pool := NewBitIDPool(intFunc(nil).New, concurrent)
		sli := sort.IntSlice{1, 4, 6, 8}
		start := intFunc(sli)
		nxt, err := AddHelper(start, start, true, false, pool.Get, pool.Done)
		assert.Nil(t, err)
		for _, id := range utils.SortSorted(sli, sli) {
			single := pool.Get()
			single.AppendItem(id)
			nxt, err = SubHelper(nxt, single, NonIntersecting, pool.Get, pool.Done)
			assert.Nil(t, err)
			pool.Done(single)
		}
		min, max, count, unique := nxt.GetBasicInfo()
		assert.Equal(t, 0, min)
		assert.Equal(t, 0, max)
		assert.Equal(t, 0, count)
		assert.Equal(t, utils.GetUniqueCount(nxt.GetItemList()), unique)
		fmt.Println(nxt.GetItemList())
		pool.Done(nxt)
	}
}
