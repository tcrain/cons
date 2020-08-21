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
	// "sync/atomic"
	"encoding/binary"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"math/bits"
	"sort"

	"github.com/tcrain/cons/consensus/utils"
	// unsafe "unsafe"
)

// A BitID stores a set of indecies as an array of bytes
type BitID struct {
	Buff       []byte        // This are the encoded indecies
	encodeBuff []byte        // This is the same as buff with an extra byte at the beginning to indicate the encoding
	str        string        // This is Buff stored as a string
	itemList   sort.IntSlice // This is a sorted list of the unencoded indecies
	numItems   int           // This is the count of indecies stored
	min, max   int
}

func (bid *BitID) New() NewBitIDInterface {
	return &BitID{numItems: -1}
}
func (bid *BitID) DoMakeCopy() NewBitIDInterface {
	return bid.MakeCopy().(*BitID)
}
func (bid *BitID) NewIterator() BIDIter {
	ret := NewBitIDIterator()
	ret.bid = bid
	return ret
}

// Returns the smallest element, the largest element, and the total number of elements
func (bid *BitID) GetBasicInfo() (min, max, count, uniqueCount int) {
	bid.GetNumItems() // populate the items
	return bid.min, bid.max, bid.numItems, bid.numItems
}

// Allocate the expected initial size
func (bid *BitID) SetInitialSize(v int) {
	bid.encodeBuff = make([]byte, v/8+1+1)
	bid.Buff = bid.encodeBuff[1:]
}

// Append an item at the end of the bitID (must be bigger than all existing items)
func (bid *BitID) AppendItem(v int) {
	bid.AddBitID(v, false, nil)
}

func (bid *BitID) Encode(writer io.Writer) (n int, err error) {
	return utils.EncodeHelper(bid.DoEncode(), writer)
}
func (bid *BitID) Decode(reader io.Reader) (n int, err error) {
	var buff []byte
	n, buff, err = utils.DecodeHelper(reader)
	if err != nil {
		return
	}
	err = bid.doDecode(buff)
	return
}

func get1Count(buff []byte) (min, max, count int) {
	var foundMin bool
	for i, b := range buff {
		if !foundMin {
			z := bits.TrailingZeros8(b) // TODO does endian matter here?
			if z < 8 {
				min = i*8 + z
				foundMin = true
			}
		}
		z := bits.LeadingZeros8(b)
		if z < 8 {
			max = i*8 + (7 - z)
		}
		count += bits.OnesCount8(b)
		// for _, j := range bitItems {
		// 	if b&j > 0 {
		// 		count++
		// 	}
		// }
	}
	return
}

func (bid *BitID) NextID(iter *BitIDIterator) (nxt int, err error) {
	for true {
		bytID := iter.iterIdx / 8
		if bytID >= len(bid.Buff) {
			iter.started = false
			return 0, types.ErrNoItems
		}
		switch (1 << uint(iter.iterIdx%8)) & bid.Buff[bytID] {
		case 0:
		default:
			nxt = iter.iterIdx
			iter.iterIdx++
			return
		}
		iter.iterIdx++
	}
	panic("should not reach")
}

func (bid *BitID) MakeCopy() BitIDInterface {
	if bid.GetNumItems() == 0 {
		return &BitID{}
	}
	encodeBuff := make([]byte, len(bid.encodeBuff))
	copy(encodeBuff, bid.encodeBuff)
	return &BitID{
		Buff:       encodeBuff[1:],
		encodeBuff: encodeBuff,
		str:        bid.str,
		numItems:   bid.numItems,
		min:        bid.min,
		max:        bid.max}
}

func (bid *BitID) DoEncode() []byte {
	if len(bid.Buff) <= bid.GetNumItems()*4 {
		// generalconfig.Encoding.PutUint32(bid.encodeBuff[1:], uint32(bid.GetNumItems()))
		return bid.encodeBuff
	}

	// smaller to just encode the values as ints directly
	enc := make([]byte, 1, bid.GetNumItems()*binary.MaxVarintLen32+1)
	enc[0] = byte(types.BitIDBasic)

	putIn := make([]byte, binary.MaxVarintLen64)
	for _, nxt := range bid.GetItemList() {
		pos := binary.PutUvarint(putIn, uint64(nxt))
		enc = append(enc, putIn[:pos]...)
	}
	return enc
}

func (bid *BitID) doDecode(buff []byte) error {
	if len(buff) < 1 {
		return types.ErrInvalidBitID
	}
	switch buff[0] {
	case byte(types.BitIDSingle):
		// we construct the itemList lazily
		buff = utils.TrimZeros(buff, 1)
		bid.Buff = buff[1:]
		bid.encodeBuff = buff
		bid.numItems = -1
		return nil
	case byte(types.BitIDBasic):
		var itemList sort.IntSlice
		pos := 1
		prv := -1
		for pos < len(buff) {
			v, n := binary.Uvarint(buff[pos:])
			vint := int(v)
			if vint < 0 || vint <= prv {
				// must be sorted and no duplicates
				return types.ErrUnsortedBitID
			}
			pos += n
			itemList = append(itemList, vint)
			prv = vint
		}
		if !sort.IsSorted(itemList) {
			return types.ErrInvalidBitID
		}
		return bid.fromInts(itemList)
	}
	return types.ErrInvalidBitID
}

func DecodeBitID(buff []byte) (*BitID, error) {
	ret := &BitID{}
	err := ret.doDecode(buff)
	return ret, err
}

// Done is called when this item is no longer needed
func (bid *BitID) Done() {
	panic("TODO")
}
func (bid *BitID) GetNumItems() int {
	if bid.numItems == -1 {
		bid.min, bid.max, bid.numItems = get1Count(bid.Buff)
	}
	return bid.numItems
}

var bitItems = [8]byte{1, 2, 4, 8, 16, 32, 64, 128}

// bitIDGetMaxLen returns the number of bytes needed to encode maxMembers
func bitIDGetMaxLen(maxMembers int) int {
	var extra int
	if maxMembers%8 != 0 {
		extra++
	}
	return (maxMembers / 8) + extra
}

// AllowDuplicates returns false.
func (bid *BitID) AllowsDuplicates() bool {
	return false
}

func (bid *BitID) GetStr() string {
	if bid.str == "" {
		bid.str = string(bid.DoEncode())
	}
	// if bid.str == "" && len(bid.Buff) > 0 {
	// 	bid.str = string(bid.Buff)
	// }
	return bid.str
}

func MergeBitIDList(bList ...BitIDInterface) *BitID {
	var maxLen int
	for _, bid := range bList {
		inbuffLen := len(bid.(*BitID).Buff)
		if maxLen < inbuffLen {
			maxLen = inbuffLen
		}
	}
	encBuff := make([]byte, maxLen+1)
	encBuff[0] = byte(types.BitIDSingle)
	buff := encBuff[1:]
	for i := 0; i < maxLen; i++ {
		for _, bid := range bList {
			inbuff := bid.(*BitID).Buff
			if i < len(inbuff) {
				buff[i] |= inbuff[i]
			}
		}
	}
	encBuff = utils.TrimZeros(encBuff, 1)
	buff = encBuff[1:]
	return &BitID{
		Buff:       buff,
		encodeBuff: encBuff,
		numItems:   -1}
}

func (bid *BitID) GetNewItems(otherBid BitIDInterface) BitIDInterface {
	bid2 := otherBid.(*BitID)
	encBuff := make([]byte, utils.Max(len(bid.Buff), len(bid2.Buff))+1)
	encBuff[0] = byte(types.BitIDSingle)
	buff := encBuff[1:]

	end := utils.Min(len(bid.Buff), len(bid2.Buff))
	for i := 0; i < end; i++ {
		buff[i] = (bid.Buff[i] ^ bid2.Buff[i]) & bid2.Buff[i]
	}
	if len(bid.Buff) < len(bid2.Buff) {
		copy(buff[end:], bid2.Buff[end:])
	}
	encBuff = utils.TrimZeros(encBuff, 1)
	buff = encBuff[1:]
	return &BitID{
		Buff:       buff,
		encodeBuff: encBuff,
		numItems:   -1}
}

func MergeBitID(bid1 *BitID, bid2 *BitID) *BitID {
	encBuff := make([]byte, utils.Max(len(bid1.Buff), len(bid2.Buff))+1)
	encBuff[0] = byte(types.BitIDSingle)
	buff := encBuff[1:]

	var longBid, shortBid []byte
	if len(bid1.Buff) > len(bid2.Buff) {
		longBid = bid1.Buff
		shortBid = bid2.Buff
	} else {
		longBid = bid2.Buff
		shortBid = bid1.Buff
	}

	for i, s := range shortBid {
		buff[i] = s | longBid[i]
	}

	copy(buff[len(shortBid):], longBid[len(shortBid):])
	encBuff = utils.TrimZeros(encBuff, 1)
	buff = encBuff[1:]
	return &BitID{
		Buff:       buff,
		encodeBuff: encBuff,
		numItems:   -1}
}

// SubBitID assumes bid1 and bid2 are already valid to subtract
// bid 2 is the smaller one
func SubBitID(bid1 *BitID, bid2 *BitID) *BitID {
	// if len(bid1.Buff) == 0 {
	//	return &BitID{}
	// }

	encBuff := make([]byte, len(bid1.Buff)+1)
	encBuff[0] = byte(types.BitIDSingle)
	buff := encBuff[1:]
	minEnd := utils.Min(len(bid1.Buff), len(bid2.Buff))

	for i := 0; i < minEnd; i++ {
		// for i, s := range bid2.Buff {
		buff[i] = (bid2.Buff[i] & bid1.Buff[i]) ^ bid1.Buff[i]
	}

	if len(bid1.Buff) > len(bid2.Buff) {
		copy(buff[len(bid2.Buff):], bid1.Buff[len(bid2.Buff):])
	}
	encBuff = utils.TrimZeros(encBuff, 1)
	buff = encBuff[1:]
	return &BitID{
		Buff:       buff,
		encodeBuff: encBuff,
		numItems:   -1}
}

func innerAddLoop(i int, b byte, itemsList sort.IntSlice) sort.IntSlice {
	for k, j := range bitItems {
		if b&j > 0 {
			itemsList = append(itemsList, (i*8)+k)
		}
	}
	return itemsList
}

func (bid *BitID) HasNewItems(otherint BitIDInterface) bool {
	other := otherint.(*BitID)

	end := utils.Min(len(bid.Buff), len(other.Buff))
	for i := 0; i < end; i++ {
		if len(innerAddLoop(i, (bid.Buff[i]&other.Buff[i])^other.Buff[i], nil)) > 0 {
			return true
		}
	}
	for i := end; i < len(other.Buff); i++ {
		if len(innerAddLoop(i, other.Buff[i], nil)) > 0 {
			return true
		}
	}
	return false
}

func (bid *BitID) HasIntersection(otherint BitIDInterface) bool {
	other := otherint.(*BitID)
	end := utils.Min(len(bid.Buff), len(other.Buff))
	for i := 0; i < end; i++ {
		if len(innerAddLoop(i, bid.Buff[i]&other.Buff[i], nil)) > 0 {
			return true
		}
	}
	return false
}

func (bid *BitID) CheckIntersection(otherint BitIDInterface) sort.IntSlice {
	other := otherint.(*BitID)
	var itemsList sort.IntSlice
	end := utils.Min(len(bid.Buff), len(other.Buff))
	for i := 0; i < end; i++ {
		itemsList = innerAddLoop(i, bid.Buff[i]&other.Buff[i], itemsList)
	}
	return itemsList
}

func (bid *BitID) CheckBitID(ID int) bool {
	arr := bid.Buff
	idx := ID / 8
	if idx >= len(arr) {
		return false
	}

	v := arr[idx]
	mod := ID % 8
	switch mod {
	case 0:
		if v&1 > 0 {
			return true
		}
	case 1:
		if v&2 > 0 {
			return true
		}
	case 2:
		if v&4 > 0 {
			return true
		}
	case 3:
		if v&8 > 0 {
			return true
		}
	case 4:
		if v&16 > 0 {
			return true
		}
	case 5:
		if v&32 > 0 {
			return true
		}
	case 6:
		if v&64 > 0 {
			return true
		}
	case 7:
		if v&128 > 0 {
			return true
		}
	default:
		panic("invalid mod")
	}
	return false
}

func (bid *BitID) AddBitID(ID int, allowDup bool, _ *BitIDIterator) bool {
	if allowDup {
		panic("single bitid cant be used with duplicates")
	}
	if bid.CheckBitID(ID) {
		return false
	}
	idx := ID / 8
	if idx >= len(bid.Buff) {
		addition := make([]byte, idx+6-len(bid.Buff))
		bid.encodeBuff = append(bid.encodeBuff, addition...)
		bid.Buff = bid.encodeBuff[1:]
	}
	bid.str = "" // need to nil the string since we have a new encoding
	arr := bid.Buff
	mod := ID % 8
	switch mod {
	case 0:
		arr[idx] |= 1
	case 1:
		arr[idx] |= 2
	case 2:
		arr[idx] |= 4
	case 3:
		arr[idx] |= 8
	case 4:
		arr[idx] |= 16
	case 5:
		arr[idx] |= 32
	case 6:
		arr[idx] |= 64
	case 7:
		arr[idx] |= 128
	default:
		panic("bad mod")
	}
	// bid.Buff = arr
	// bid.Str = string(bid.Buff)
	// we construct the itemList lazily
	if bid.itemList != nil {
		bid.itemList, _ = utils.InsertInSortedSlice(bid.itemList, ID, 0)
	}
	if bid.numItems != -1 {
		bid.numItems++
		if bid.min > ID {
			bid.min = ID
		}
		if bid.max < ID {
			bid.max = ID
		}
	}
	return true
}

// CreateBitIDFromBytes does not copy the byte slice and may modify it when adding new items
func CreateBitIDFromBytes(arr []byte) (*BitID, error) {
	if len(arr) >= (1<<15-1)/8 {
		return nil, types.ErrTooLargeBitID
	}

	encodeBuff := make([]byte, len(arr)+1)
	encodeBuff[0] = byte(types.BitIDSingle)
	copy(encodeBuff[1:], arr)
	encodeBuff = utils.TrimZeros(encodeBuff, 1)
	// we construct the itemList lazily
	return &BitID{Buff: encodeBuff[1:],
		encodeBuff: encodeBuff,
		numItems:   -1}, nil
}

// var count int32

func (bid *BitID) fromInts(items sort.IntSlice) error {
	if len(items) == 0 {
		//n := atomic.AddInt32(&count, 1)
		enc := make([]byte, 1)
		enc[0] = byte(types.BitIDBasic)
		bid.encodeBuff = enc
		bid.numItems = -1
		return nil
	}
	end := items[len(items)-1]
	if end < 0 || end >= (1<<15-1) {
		return types.ErrUnsortedBitID
	}

	encRet := make([]byte, (end+8)/8+1)
	// generalconfig.Encoding.PutUint32(encRet[1:], uint32(len(items)))
	encRet[0] = byte(types.BitIDSingle)
	ret := encRet[1:]

	var itemsIdx int
	var idx int
	prv := -1
	for i := 7; i <= end+7; i += 8 {
		var cur byte
		for itemsIdx < len(items) && items[itemsIdx] <= i {
			nxt := items[itemsIdx]
			if nxt < 0 {
				return types.ErrUnsortedBitID
			}
			if nxt == prv {
				itemsIdx++
				continue
			}
			if nxt < prv {
				return types.ErrUnsortedBitID
			}
			switch nxt % 8 {
			case 0:
				cur++
			case 1:
				cur += 2
			case 2:
				cur += 4
			case 3:
				cur += 8
			case 4:
				cur += 16
			case 5:
				cur += 32
			case 6:
				cur += 64
			case 7:
				cur += 128
			default:
				panic("invalid mod")
			}
			prv = nxt
			itemsIdx++
		}
		ret[idx] = cur
		idx++
	}
	// // we make a copy of the item list because we will modify it if we add new items
	// cop := make([]int, len(items), cap(items))
	// copy(cop, items)
	bid.Buff = ret
	bid.numItems = len(items)
	bid.min = items[0]
	bid.max = items[len(items)-1]
	bid.encodeBuff = encRet
	bid.itemList = items
	return nil
}

func NewBitIDFromInts(items sort.IntSlice) NewBitIDInterface {
	ret, err := CreateBitIDFromInts(items)
	utils.PanicNonNil(err)
	return ret
}

// CreateBitIDFromInts does not clopy the int slice and may modify it when adding new items
func CreateBitIDFromInts(items sort.IntSlice) (*BitID, error) {
	ret := &BitID{}
	err := ret.fromInts(items)
	return ret, err
}

func (bid *BitID) GetItemList() sort.IntSlice {
	if bid.itemList != nil {
		return bid.itemList
	}
	// we construct the itemList lazily
	arr := bid.Buff
	// var items []int
	items := make([]int, 0, bid.GetNumItems())
	// items := make([]int, 0, len(arr)*8)

	for i, nxt := range arr {
		if nxt == 0 {
			continue
		}
		var count int
		i = i * 8
		if nxt&1 > 0 {
			items = append(items, i)
			count++
		}
		if nxt&2 > 0 {
			items = append(items, i+1)
			count++
		}
		if nxt&4 > 0 {
			items = append(items, i+2)
			count++
		}
		if nxt&8 > 0 {
			items = append(items, i+3)
			count++
		}
		if nxt&16 > 0 {
			items = append(items, i+4)
			count++
		}
		if nxt&32 > 0 {
			items = append(items, i+5)
			count++
		}
		if nxt&64 > 0 {
			items = append(items, i+6)
			count++
		}
		if nxt&128 > 0 {
			count++
			items = append(items, i+7)
		}
		if count == 0 {
			panic("should have had an item")
		}
	}
	bid.itemList = items
	bid.numItems = len(items)
	return items
}
