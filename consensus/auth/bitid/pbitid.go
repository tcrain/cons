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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math/bits"
	"sort"
)

type Pbitid struct {
	buff                            []byte // 1st bit is the encoding type
	str                             string // buff as a string
	lastIdx                         int    // index in the last byte of the last 1
	numItems, min, max, uniqueCount int
	iter                            pbitidIter
}

func (bid *Pbitid) New() NewBitIDInterface {
	return &Pbitid{}
}

// append count 0s starting from currentIndex
func appendZero(buff []byte, count, currentIndex int) ([]byte, int) {
	// TODO multiple ZEROS
	currentIndex += count
	for currentIndex/8 >= len(buff) {
		buff = append(buff, 0)
	}
	return buff, currentIndex
}

// append 1 at current index
func appendOne(buff []byte, currentIndex int) ([]byte, int) {
	// TODO multiple ONES
	pos := currentIndex / 8
	if pos >= len(buff) {
		buff = append(buff, 0)
	}
	buff[pos] = (1 << uint(currentIndex%8)) | buff[pos]
	currentIndex++
	return buff, currentIndex
}

func NewPbitidFromInts(items sort.IntSlice) NewBitIDInterface {
	if len(items) == 0 {
		return &Pbitid{}
	}
	end := items[len(items)-1]
	if end < 0 || end >= (1<<32-1) {
		panic(types.ErrUnsortedBitID)
	}

	buff := make([]byte, 0, (end+8)/8)
	ret := &Pbitid{
		buff: buff,
		min:  items[0]}

	for _, nxt := range items {
		ret.AppendItem(nxt)
	}
	if ret.numItems != len(items) {
		panic("invalid num items")
	}
	if ret.max != items[len(items)-1] {
		panic("invalid max")
	}
	return ret
}
func (bid *Pbitid) AppendItem(nxt int) {
	count := nxt - bid.max
	if count < 0 {
		panic("invalid order")
	}
	if bid.numItems == 0 {
		bid.min = nxt
	}
	bid.buff, bid.lastIdx = appendZero(bid.buff, count, bid.lastIdx)
	bid.buff, bid.lastIdx = appendOne(bid.buff, bid.lastIdx)
	if bid.max != nxt || bid.numItems == 0 {
		bid.uniqueCount++
	}
	bid.numItems++
	bid.max = nxt
}

// SetInitialSize is unused.
func (bid *Pbitid) SetInitialSize(int) {
	// TODO?
}

// DoMakeCopy returns a copy of the bit id.
func (bid *Pbitid) DoMakeCopy() NewBitIDInterface {
	buff := make([]byte, len(bid.buff))
	copy(buff, bid.buff)
	return &Pbitid{
		buff:        buff,
		str:         bid.str,
		min:         bid.min,
		max:         bid.max,
		lastIdx:     bid.lastIdx,
		uniqueCount: bid.uniqueCount,
		numItems:    bid.numItems}
}
func (bid *Pbitid) Encode(writer io.Writer) (n int, err error) {
	return utils.EncodeHelper(bid.DoEncode(), writer)
}
func (bid *Pbitid) Decode(reader io.Reader) (n int, err error) {
	n, bid.buff, err = utils.DecodeHelper(reader)
	if err != nil {
		return
	}
	bid.construct()
	return
}
func (bid *Pbitid) Deserialize(msg *messages.Message) (n int, err error) {
	n, bid.buff, err = utils.DecodeHelper((*messages.MsgBuffer)(msg))
	if err != nil {
		return
	}
	bid.construct()
	return
}

func (bid *Pbitid) DoEncode() []byte {
	return bid.buff
}
func (bid *Pbitid) construct() {
	if len(bid.buff) == 0 {
		bid.numItems = 0
		return
	}
	var gotMin bool
	prev0 := true
	bid.numItems = 0
	var maxIdx int
	for _, b := range bid.buff {
		if !gotMin {
			z := bits.TrailingZeros8(b) // TODO does endian matter here?
			if z < 8 {
				bid.min = maxIdx + z
				gotMin = true
			}
		}
		for i := 0; i < 8; i++ {
			switch b&(1<<i) != 0 {
			case true:
				bid.numItems++
				if prev0 {
					bid.uniqueCount++
					bid.max = maxIdx
				} else {
					prev0 = false
				}
			case false:
				maxIdx++
				prev0 = true
			}
		}
	}
}

// GetBasicInfo returns the smallest element, the largest element, and the total number of elements
func (bid *Pbitid) GetBasicInfo() (min, max, count, uniqueCount int) {
	return bid.min, bid.max, bid.numItems, bid.uniqueCount
}

func (bid *Pbitid) GetItemList() sort.IntSlice {
	return toSliceHelper(bid)
}
func (bid *Pbitid) GetStr() string {
	if bid.str == "" {
		bid.str = string(bid.DoEncode())
	}
	return bid.str
}

// CheckBitID returns true if the argument is in the bid
func (bid *Pbitid) CheckBitID(int) bool {
	panic("unused")
}

// Done is called when this item is no longer needed
func (bid *Pbitid) Done() {
	bid.buff = bid.buff[:0]
	bid.uniqueCount = 0
	bid.numItems = 0
	bid.max = 0
	bid.lastIdx = 0
	bid.min = 0
	bid.str = ""
}

// AllowDuplicates returns true.
func (bid *Pbitid) AllowsDuplicates() bool {
	return true
}

// NewIterator Returns a new iterator of the bit id.
func (bid *Pbitid) NewIterator() BIDIter {
	var ret *pbitidIter
	if bid.iter.started {
		ret = &pbitidIter{}
	} else {
		ret = &bid.iter
	}
	ret.started = true
	ret.buff = bid.buff
	return ret
}

type pbitidIter struct {
	buff       []byte
	iterIdx    int
	currentVal int
	prevVal    bool
	started    bool
}

// stops at each index (whether it exists or not) (next ID stops at each value that exists in the bitid)
// this should not be used in conjunction with NextID
// iter.iterIdx will be 1 bit past the end of the current value
func (iter *pbitidIter) nextIndex() (nxt int, err error) {
	for true {
		bytID := iter.iterIdx / 8
		if bytID >= len(iter.buff) {
			return 0, types.ErrNoItems
		}
		v := (1 << uint(iter.iterIdx%8)) & iter.buff[bytID]
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

// Done is called when the iterator is no longer needed.
func (iter *pbitidIter) Done() {
	iter.buff = nil
	iter.prevVal = false
	iter.started = false
	iter.currentVal = 0
	iter.iterIdx = 0
}

// NextID is for iterating through the bitid, an iterator is created using NewBitIDIterator(),
//returns an error if the iterator has traversed all items
func (iter *pbitidIter) NextID() (nxt int, err error) {
	for true {
		bytID := iter.iterIdx / 8
		if bytID >= len(iter.buff) {
			return 0, types.ErrNoItems
		}
		v := (1 << uint(iter.iterIdx%8)) & iter.buff[bytID]
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
