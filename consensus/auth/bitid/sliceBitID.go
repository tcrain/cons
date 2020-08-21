package bitid

import (
	"bytes"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
	"sort"
)

type SliceBitID struct {
	items             sort.IntSlice
	nonDuplicateCount int
}

type sliceBitIDIter struct {
	items   sort.IntSlice
	idx     int
	started bool
}

// Done is called when the iterator is no longer needed
func (iter *sliceBitIDIter) Done() {
	iter.started = false
}

// NextID is for iterating through the bitid, an iterator is created using NewBitIDIterator(), returns an error if the iterator has traversed all items
func (iter *sliceBitIDIter) NextID() (nxt int, err error) {
	if iter.idx >= len(iter.items) {
		err = types.ErrNoItems
		return
	}
	nxt = iter.items[iter.idx]
	iter.idx++
	return
}

// NewSliceBitIDFromInts allocates a SliceBitID from the int slice
func NewSliceBitIDFromInts(from sort.IntSlice) NewBitIDInterface {
	return &SliceBitID{
		nonDuplicateCount: utils.GetUniqueCount(from),
		items:             from}
}

// New allocates a new empty bit id.
func (bid *SliceBitID) New() NewBitIDInterface {
	return &SliceBitID{}
}

// Done is called when this item is no longer needed
func (bid *SliceBitID) Done() {
	panic("TODO")
}

// DoMakeCopy returns a copy of the bit id.
func (bid *SliceBitID) DoMakeCopy() NewBitIDInterface {
	cpy := make(sort.IntSlice, len(bid.items))
	copy(cpy, bid.items)
	return &SliceBitID{
		nonDuplicateCount: bid.nonDuplicateCount,
		items:             cpy}
}

// Returns true if the argument is in the bid
func (bid *SliceBitID) CheckBitID(v int) bool {
	idx := bid.items.Search(v)
	if idx < len(bid.items) && bid.items[idx] == v {
		return true
	}
	return false
}

// Returns the list of items of the bitid
func (bid *SliceBitID) GetItemList() sort.IntSlice {
	return bid.items
}

// Gets the string representation of the bitid
func (bid *SliceBitID) GetStr() string {
	writer := bytes.NewBuffer(nil)
	_, err := bid.Encode(writer)
	utils.PanicNonNil(err)
	return string(writer.Bytes())
}
func (bid *SliceBitID) NewIterator() BIDIter {
	return &sliceBitIDIter{
		items: bid.items,
	}
}

// AllowDuplicates returns true.
func (bid *SliceBitID) AllowsDuplicates() bool {
	return true
}

// Returns the smallest element, the largest element, and the total number of elements
func (bid *SliceBitID) GetBasicInfo() (min, max, count, uniqueItemCount int) {
	if len(bid.items) == 0 {
		return
	}
	return bid.items[0], bid.items[len(bid.items)-1], len(bid.items), bid.nonDuplicateCount
}

// Allocate the expected initial size
func (bid *SliceBitID) SetInitialSize(v int) {
	bid.items = make(sort.IntSlice, 0, v)
}

// Append an item at the end of the bitID (must be bigger than all existing items)
func (bid *SliceBitID) AppendItem(v int) {
	if len(bid.items) == 0 || v != bid.items[len(bid.items)-1] {
		bid.nonDuplicateCount++
	}
	bid.items = append(bid.items, v)
}

func (bid *SliceBitID) Encode(writer io.Writer) (n int, err error) {
	return uvarintEncode(bid.items, writer)
}

func (bid *SliceBitID) Decode(reader io.Reader) (n int, err error) {
	var n1 int
	var numItems uint64
	numItems, n1, err = utils.ReadUvarint(reader)
	n += n1
	if err != nil {
		return
	}
	if numItems > math.MaxUint16 {
		err = types.ErrTooLargeBitID
		return
	}
	bid.items = make(sort.IntSlice, numItems)
	prev := -1
	for i := 0; i < int(numItems); i++ {
		var nxt uint64
		nxt, n1, err = utils.ReadUvarint(reader)
		n += n1
		if err != nil {
			return
		}
		val := int(nxt)
		if val < 0 {
			err = types.ErrUnsortedBitID
			return
		}
		if val != prev {
			bid.nonDuplicateCount++
			prev = val
		}
		bid.items[i] = val
	}
	return
}
