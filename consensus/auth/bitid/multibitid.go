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
	"encoding/binary"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"sort"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/utils"
)

type MultiBitID struct {
	itemList        sort.IntSlice
	encode          []byte
	str             string
	uniqueItemCount int // number of unique items in the BitID
}

func NewMultiBitIDFromInts(items sort.IntSlice) NewBitIDInterface {
	ret, err := CreateMultiBitIDFromInts(items)
	utils.PanicNonNil(err)
	return ret
}

func (bid *MultiBitID) New() NewBitIDInterface {
	return &MultiBitID{}
}
func (bid *MultiBitID) DoMakeCopy() NewBitIDInterface {
	ret := &MultiBitID{}
	ret.itemList = make(sort.IntSlice, len(bid.itemList))
	copy(ret.itemList, bid.itemList)
	ret.encode = make([]byte, len(bid.encode))
	copy(ret.encode, bid.encode)
	ret.str = bid.str
	ret.uniqueItemCount = bid.uniqueItemCount
	return ret
}

func (bid *MultiBitID) NewIterator() BIDIter {
	return &sliceBitIDIter{
		items: bid.itemList,
	}
}

// GetBasicInfo returns the smallest element, the largest element, and the total number of elements
func (bid *MultiBitID) GetBasicInfo() (min, max, count, uniqueCount int) {
	if len(bid.itemList) == 0 {
		return
	}
	return bid.itemList[0], bid.itemList[len(bid.itemList)-1], len(bid.itemList), bid.uniqueItemCount
}

// SetInitialSize allocates the expected initial size
func (bid *MultiBitID) SetInitialSize(v int) {
	bid.itemList = make(sort.IntSlice, 0, v)
}

// Done is called when this item is no longer needed
func (bid *MultiBitID) Done() {
	panic("TODO")
}

// AppendItem appends an item at the end of the bitID (must be bigger than all existing items)
func (bid *MultiBitID) AppendItem(v int) {
	if len(bid.itemList) == 0 || v != bid.itemList[len(bid.itemList)-1] {
		bid.uniqueItemCount++
	}
	bid.itemList = append(bid.itemList, v)
}

func (bid *MultiBitID) Encode(writer io.Writer) (n int, err error) {
	return utils.EncodeHelper(bid.DoEncode(), writer)
}
func (bid *MultiBitID) Deserialize(msg *messages.Message) (n int, err error) {
	var buff []byte
	n, buff, err = utils.DecodeHelperMsg((*messages.MsgBuffer)(msg))
	if err != nil {
		return
	}
	err = bid.doDecode(buff)
	return
}
func (bid *MultiBitID) Decode(reader io.Reader) (n int, err error) {
	var buff []byte
	n, buff, err = utils.DecodeHelper(reader)
	if err != nil {
		return
	}
	err = bid.doDecode(buff)
	return
}

// AllowDuplicates returns true.
func (bid *MultiBitID) AllowsDuplicates() bool {
	return true
}

func (bid *MultiBitID) NextID(iter *BitIDIterator) (nxt int, err error) {
	if iter.iterIdx >= len(bid.itemList) {
		return 0, types.ErrNoItems
	}
	nxt = bid.itemList[iter.iterIdx]
	iter.iterIdx++
	return
}

func (bid *MultiBitID) GetStr() string {
	// panic(1)
	if bid.str == "" {
		if bid.encode == nil {
			bid.DoEncode()
		}
		bid.str = string(bid.encode)
	}
	return bid.str
}

func (bid *MultiBitID) MakeCopy() BitIDInterface {
	newItemList := make([]int, len(bid.itemList))
	copy(newItemList, bid.itemList)
	return &MultiBitID{
		uniqueItemCount: bid.uniqueItemCount,
		itemList:        newItemList}
}

func (bid *MultiBitID) DoEncode() []byte {
	if len(bid.itemList) == 0 {
		return nil
	}
	if bid.encode != nil {
		return bid.encode
	}
	// seperate into duplicates and non-duplicates
	nonDuplicates, duplicates := utils.GetDuplicates(bid.itemList)

	// encode the non duplicates as a bit id
	prebids, err := CreateBitIDFromInts(nonDuplicates)
	if err != nil {
		panic(err)
	}
	prebid := prebids.DoEncode()

	// encode the duplicates
	enc := make([]byte, 0, len(duplicates)*binary.MaxVarintLen32)
	putIn := make([]byte, binary.MaxVarintLen64)
	for _, nxt := range duplicates {
		pos := binary.PutUvarint(putIn, uint64(nxt))
		enc = append(enc, putIn[:pos]...)
	}

	// put them together
	// First byte is the id, next 4 bytes are the size of the Simple bitid
	// then the duplicates
	fullEnc := make([]byte, 5+len(prebid)+len(enc))
	fullEnc[0] = byte(types.BitIDMulti)
	pos := 1
	config.Encoding.PutUint32(fullEnc[pos:], uint32(len(prebid)))
	pos += 4
	pos += copy(fullEnc[pos:], prebid)
	pos += copy(fullEnc[pos:], enc)

	if pos != len(fullEnc) {
		panic("bad encode")
	}

	bid.encode = fullEnc

	return fullEnc
}

func (bid *MultiBitID) doDecode(buff []byte) error {
	// First byte is the id, next 4 bytes are the size of the Simple bitid
	// then the duplicates
	if len(buff) == 0 {
		return nil
	}
	pos := 5
	if len(buff) < pos {
		return types.ErrInvalidBitID
	}
	// Check it has the correct encoding
	if buff[0] != byte(types.BitIDMulti) {
		return types.ErrInvalidBitIDEncoding
	}

	prebidEnd := int(config.Encoding.Uint32(buff[1:])) + pos
	if len(buff) < prebidEnd {
		return types.ErrInvalidBitID
	}
	prebid, err := DecodeBitID(buff[pos:prebidEnd])
	pos = prebidEnd
	if err != nil {
		return err
	}

	var duplicates sort.IntSlice
	for pos < len(buff) {
		v, n := binary.Uvarint(buff[pos:])
		pos += n
		duplicates = append(duplicates, int(v))
	}
	if !sort.IsSorted(duplicates) {
		return types.ErrInvalidBitID
	}

	// allItems := append(prebid.GetItemList(), duplicates...)
	// sort.Sort(allItems)
	preItems := prebid.GetItemList()
	bid.uniqueItemCount = len(preItems)
	allItems := utils.SortSorted(preItems, duplicates)

	bid.itemList = allItems
	bid.encode = buff
	bid.str = string(buff)
	return nil
}

func (bid *MultiBitID) GetNumItems() int {
	// we construct the itemList lazily
	return len(bid.GetItemList())
}

func (bid *MultiBitID) HasNewItems(otherint BitIDInterface) bool {
	other := otherint.(*MultiBitID)
	partialItemList := bid.itemList

	for _, nxt := range other.itemList {
		idx := sort.SearchInts(partialItemList, nxt)
		// rest := idx
		if idx < len(partialItemList) && partialItemList[idx] == nxt {
			// we found it so not new
			// rest++
		} else {
			// this is new
			return true
		}
		if idx < len(partialItemList) {
			partialItemList = partialItemList[idx:]
		} else {
			partialItemList = nil
		}
	}
	return false
}

// GetNewItems does not return duplicates
func (bid *MultiBitID) GetNewItems(otherint BitIDInterface) BitIDInterface {
	other := otherint.(*MultiBitID)
	var newItemList sort.IntSlice
	partialItemList := bid.itemList

	for _, nxt := range other.itemList {
		idx := sort.SearchInts(partialItemList, nxt)
		// rest := idx
		if idx < len(partialItemList) && partialItemList[idx] == nxt {
			// we found it so not new
			// rest++
		} else {
			// this is new
			if len(newItemList) == 0 {
				newItemList = append(newItemList, nxt)
			} else if newItemList[len(newItemList)-1] != nxt {
				newItemList = append(newItemList, nxt)
			}
		}
		if idx < len(partialItemList) {
			partialItemList = partialItemList[idx:]
		} else {
			partialItemList = nil
		}
	}
	return &MultiBitID{
		itemList: newItemList}
}

func (bid *MultiBitID) HasIntersection(otherint BitIDInterface) bool {
	other := otherint.(*MultiBitID)
	partialItemList := other.itemList
	for _, nxt := range bid.itemList {
		idx := sort.SearchInts(partialItemList, nxt)
		rest := idx
		if idx < len(partialItemList) && partialItemList[idx] == nxt {
			// we found it
			return true
		}
		if rest < len(partialItemList) {
			partialItemList = partialItemList[rest:]
		} else {
			// we checked all the second list
			break
		}
	}
	return false
}

func (bid *MultiBitID) CheckIntersection(otherint BitIDInterface) sort.IntSlice {
	other := otherint.(*MultiBitID)
	var newItemList sort.IntSlice
	partialItemList := other.itemList
	for _, nxt := range bid.itemList {
		idx := sort.SearchInts(partialItemList, nxt)
		rest := idx
		if idx < len(partialItemList) && partialItemList[idx] == nxt {
			// we found it
			newItemList = append(newItemList, nxt)
			rest++
		}
		if rest < len(partialItemList) {
			partialItemList = partialItemList[rest:]
		} else {
			// we checked all the second list
			break
		}
	}
	return newItemList
}

func (bid *MultiBitID) CheckBitID(ID int) bool {
	idx := sort.SearchInts(bid.itemList, ID)
	if idx < len(bid.itemList) && bid.itemList[idx] == ID {
		return true
	}
	return false
}

func (bid *MultiBitID) AddBitID(ID int, allowDup bool, iter *BitIDIterator) bool {
	if allowDup {
		bid.itemList, iter.iterIdx = utils.InsertInSortedSlice(bid.itemList, ID, iter.iterIdx)
		bid.encode = nil
		return true
	}
	var inserted bool
	bid.itemList, iter.iterIdx, inserted = utils.InsertIfNotFound(bid.itemList, ID, iter.iterIdx)
	if inserted {
		bid.encode = nil
	}
	return inserted
}

// CreateMultiBitIDFromInts does not clopy the int slice and may modify it when adding new items
func CreateMultiBitIDFromInts(items sort.IntSlice) (*MultiBitID, error) {
	if !sort.IsSorted(items) {
		return nil, types.ErrUnsortedBitID
	}
	if len(items) > 0 && items[0] < 0 {
		return nil, types.ErrInvalidBitID
	}
	return &MultiBitID{
		uniqueItemCount: utils.GetUniqueCount(items),
		itemList:        items}, nil
}

func (bid *MultiBitID) GetItemList() sort.IntSlice {
	return bid.itemList
}

func DecodeMultiBitID(buff []byte) (*MultiBitID, error) {
	ret := &MultiBitID{}
	err := ret.doDecode(buff)
	return ret, err
}
