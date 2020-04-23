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
	"github.com/tcrain/cons/consensus/types"
	// "fmt"
	"sort"

	"github.com/tcrain/cons/consensus/utils"
)

type Pbitid struct {
	buff     []byte // 1st bit is the encoding type
	str      string // buff as a string
	n        int
	numItems int
}

func appendZero(buff []byte, currentIndex int) ([]byte, int) {
	// pos is + 1 because the first byte describes the encoding type
	if (currentIndex/8)+1 >= len(buff) {
		buff = append(buff, 0)
	}
	currentIndex++
	return buff, currentIndex
}

func appendOne(buff []byte, currentIndex int) ([]byte, int) {
	// pos is + 1 because the first byte describes the encoding type
	pos := (currentIndex / 8) + 1
	if pos >= len(buff) {
		buff = append(buff, 0)
	}
	buff[pos] = (1 << uint(currentIndex%8)) | buff[pos]
	currentIndex++
	return buff, currentIndex
}

func DecodePbitid(buff []byte) (*Pbitid, error) {
	if len(buff) < 1 {
		return nil, types.ErrInvalidBitID
	}
	switch buff[0] {
	case byte(BitIDP):
		buff = utils.TrimZeros(buff, 1)
		return &Pbitid{buff: buff, numItems: -1}, nil
	default:
		panic(1) // TODO
	}
}

func CreatePbitIDFromBytes(arr []byte) (*Pbitid, error) {
	if len(arr) >= (1<<15-1)/8 {
		return nil, types.ErrTooLargeBitID
	}

	encodeBuff := make([]byte, len(arr)+1)
	copy(encodeBuff[1:], arr)
	encodeBuff = utils.TrimZeros(encodeBuff, 1)

	return &Pbitid{buff: encodeBuff, numItems: -1}, nil
}

func CreatePbitidFromInts(items sort.IntSlice) (*Pbitid, error) {
	if len(items) == 0 {
		enc := make([]byte, 1)
		enc[0] = byte(BitIDP)
		return &Pbitid{buff: enc, numItems: -1}, nil
	}
	end := items[len(items)-1]
	if end < 0 || end >= (1<<15-1) {
		return nil, types.ErrUnsortedBitID
	}

	buff := make([]byte, 1, (end+8)/8+1)
	var currentIndex int

	var prev int
	for _, nxt := range items {
		for prev < nxt {
			buff, currentIndex = appendZero(buff, currentIndex)
			prev++
		}
		buff, currentIndex = appendOne(buff, currentIndex)
	}
	return &Pbitid{
		buff:     buff,
		numItems: len(items)}, nil
}

func (bid *Pbitid) MakeCopy() BitIDInterface {
	buff := make([]byte, len(bid.buff))
	copy(buff, bid.buff)
	return &Pbitid{
		buff:     buff,
		str:      bid.str,
		n:        bid.n,
		numItems: bid.numItems}
}
func (bid *Pbitid) Encode() []byte {
	bid.buff[0] = byte(BitIDP) // TODO
	return bid.buff
}
func (bid *Pbitid) GetNumItems() int {
	if bid.numItems == -1 {
		bid.numItems = 0
		iter := NewBitIDIterator()
		_, err := bid.NextID(iter)
		for ; err == nil; _, err = bid.NextID(iter) {
			bid.numItems++
		}
	}
	return bid.numItems
}
func (bid *Pbitid) GetNewItems(other BitIDInterface) BitIDInterface {
	newItems := getNewItems(bid, other.(*Pbitid), false)
	ret, err := CreatePbitidFromInts(newItems)
	if err != nil {
		panic(err)
	}
	return ret
}
func (bid *Pbitid) HasNewItems(other BitIDInterface) bool {
	return len(getNewItems(bid, other.(*Pbitid), true)) > 0
}
func (bid *Pbitid) CheckIntersection(other BitIDInterface) sort.IntSlice {
	otherIter := NewBitIDIterator()
	bidIter := NewBitIDIterator()
	var items sort.IntSlice
	otherpbid := other.(*Pbitid)
	v, err := nextIntersection(bid, otherpbid, bidIter, otherIter)
	for ; err != nil; v, err = nextIntersection(bid, otherpbid, bidIter, otherIter) {
		items = append(items, v)
	}
	return items
}
func (bid *Pbitid) CheckBitID(val int) bool {
	iter := NewBitIDIterator()
	nxt, err := bid.NextID(iter)
	for ; err == nil; nxt, err = bid.NextID(iter) {
		if nxt == val {
			return true
		}
		if nxt > val {
			return false
		}
	}
	return false
}
func (bid *Pbitid) AddBitID(id int, allowDup bool, iter *BitIDIterator) bool {
	panic(1)
}

func (bid *Pbitid) GetItemList() sort.IntSlice {
	iter := NewBitIDIterator()
	var items sort.IntSlice
	nxt, err := bid.NextID(iter)
	for ; err == nil; nxt, err = bid.NextID(iter) {
		items = append(items, nxt)
	}
	return items
}
func (bid *Pbitid) GetStr() string {
	if bid.str == "" {
		bid.str = string(bid.Encode())
	}
	return bid.str
}
func (bid *Pbitid) HasIntersection(other BitIDInterface) bool {
	if _, err := nextIntersection(bid, other.(*Pbitid), NewBitIDIterator(), NewBitIDIterator()); err != nil {
		return true
	}
	return false
}

func getNewItems(bid *Pbitid, other *Pbitid, earlyStop bool) sort.IntSlice {
	var items sort.IntSlice
	otherIter := NewBitIDIterator()
	bidIter := NewBitIDIterator()

	nxtBid, errBid := bid.NextID(bidIter)
	for {
		nxtOther, errOther := other.NextID(otherIter)
		if errOther != nil {
			return items
		}
		for {
			if errBid != nil || nxtBid > nxtOther {
				if len(items) == 0 || items[len(items)-1] != nxtOther {
					items = append(items, nxtOther)
				}
				if earlyStop {
					return items
				}
				break
			}
			nxtBid, errBid = bid.NextID(bidIter)
		}

	}
}

func nextIntersection(small *Pbitid, big *Pbitid, smallIter *BitIDIterator, bigIter *BitIDIterator) (int, error) {
	var bigVal, smallVal int
	var err error
	bigVal, err = big.NextID(bigIter)
	if err != nil {
		return 0, err
	}
	for {
		smallVal, err = small.NextID(smallIter)
		if err != nil {
			return 0, err
		}
		if bigVal == smallVal {
			return smallVal, nil
		}
		if smallVal > bigVal {
			big, small = small, big
			bigVal = smallVal
		}
	}
}

// stops at each index (whether it exists or not) (next ID stops at each value that exists in the bitid)
// this should not be used in conjunction with NextID
// iter.iterIdx will be 1 bit past the end of the current value
func (bid *Pbitid) nextIndex(iter *BitIDIterator) (nxt int, err error) {
	for true {
		bytID := iter.iterIdx/8 + 1
		if bytID >= len(bid.buff) {
			return 0, types.ErrNoItems
		}
		v := (1 << uint(iter.iterIdx%8)) & bid.buff[bytID]
		switch v {
		case 0:
			nxt = iter.currentVal
			iter.currentVal++
			iter.iterIdx++
			iter.prevVal = false
			return
		default:
			nxt = iter.currentVal
			iter.iterIdx++
			iter.prevVal = true
			return
		}
	}
	panic("should not reach")
}

// For iteration
func (bid *Pbitid) NextID(iter *BitIDIterator) (nxt int, err error) {
	for true {
		bytID := iter.iterIdx/8 + 1
		if bytID >= len(bid.buff) {
			return 0, types.ErrNoItems
		}
		v := (1 << uint(iter.iterIdx%8)) & bid.buff[bytID]
		switch v {
		case 0:
			iter.currentVal++
			iter.iterIdx++
			iter.prevVal = false
		default:
			nxt = iter.currentVal
			iter.iterIdx++
			iter.prevVal = true
			return
		}
	}
	panic("should not reach")
}
