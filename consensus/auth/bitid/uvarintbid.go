package bitid

import (
	"bytes"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
	"sort"
)

type UvarintBitID struct {
	min, max, size, uniqueCount int
	hasIteration                bool
	maxSizeIdx                  int // index in arr where the number of items of max is counted
	maxCount                    int // number of items of max
	arr                         []byte
	iter                        uvarintBitIDIter
}

type uvarintBitIDIter struct {
	buff            bytes.Reader
	arr             []byte
	previous, count int
	started         bool
}

// const sizeBytes = 4 // start with uint64

func allocUvarintArr(size int) int {
	return 3 * size // TODO better estimate?
}

func NewUvarintBitIDFromInts(items sort.IntSlice) NewBitIDInterface {
	arr := make([]byte, 0, allocUvarintArr(len(items)))
	writer := bytes.NewBuffer(arr)

	var n, n1, n2 int
	var min, max, maxSizeIdx, maxSizeCount, unique int
	size := len(items)
	if size > 0 {
		min, max = items[0], items[len(items)-1]
		prev := items[0]
		count := 0
		unique = 1
		for _, nxt := range items[1:] {
			if nxt == prev {
				count++
			} else {
				unique++
				n1, n2 = writeValCount(prev, count, writer)
				n += n1 + n2
				count = 0
				prev = nxt
			}
		}
		n1, n2 = writeValCount(prev, count, writer)
		maxSizeIdx = n + n1
		maxSizeCount = count
	}
	return &UvarintBitID{
		min:         min,
		uniqueCount: unique,
		max:         max,
		maxSizeIdx:  maxSizeIdx,
		maxCount:    maxSizeCount,
		size:        size,
		arr:         writer.Bytes()}
}

func writeValCount(v, count int, writer io.Writer) (int, int) {
	n1, err := utils.EncodeUvarint(uint64(v), writer)
	utils.PanicNonNil(err)
	n2, err := utils.EncodeUvarint(uint64(count), writer)
	utils.PanicNonNil(err)
	return n1, n2
}

// Done is called when the iterator is no longer needed
func (iter *uvarintBitIDIter) Done() {
	iter.started = false
	iter.previous, iter.count = 0, 0
	(&iter.buff).Reset(iter.arr)
}

func (iter *uvarintBitIDIter) NextID() (nxt int, err error) {
	if iter.count > 0 {
		iter.count--
		nxt = iter.previous
		return
	}
	var i uint64
	i, _, err = utils.ReadUvarintByteReader(&iter.buff)
	if err != nil {
		return
	}
	iter.previous = int(i)
	var count uint64
	count, _, err = utils.ReadUvarintByteReader(&iter.buff)
	if err != nil {
		return
	}
	iter.count = int(count)
	return iter.previous, nil
}

func (bid *UvarintBitID) New() NewBitIDInterface {
	return &UvarintBitID{}
}

func (bid *UvarintBitID) writeMaxSizeCount() {
	if !bid.hasIteration {
		if bid.size > 0 {
			_, bid.arr = utils.AppendUvarint(uint64(bid.maxCount), bid.arr[:bid.maxSizeIdx])
		}
		bid.hasIteration = true
	}
}

func (bid *UvarintBitID) DoMakeCopy() NewBitIDInterface {
	bid.writeMaxSizeCount()
	cpy := make([]byte, len(bid.arr))
	copy(cpy, bid.arr)
	return &UvarintBitID{
		min:         bid.min,
		max:         bid.max,
		size:        bid.size,
		uniqueCount: bid.uniqueCount,
		maxSizeIdx:  bid.maxSizeIdx,
		maxCount:    bid.maxCount,
		arr:         cpy,
	}
}

// Returns true if the argument is in the bid
func (bid *UvarintBitID) CheckBitID(v int) bool {
	return findHepler(v, bid)
}

// Returns the list of items of the bitid
func (bid *UvarintBitID) GetItemList() sort.IntSlice {
	return toSliceHelper(bid)
}

// Gets the string representation of the bitid
func (bid *UvarintBitID) GetStr() string {
	bid.writeMaxSizeCount()
	return string(bid.arr)
}
func (bid *UvarintBitID) NewIterator() BIDIter {
	bid.writeMaxSizeCount() // be sure the max size is written
	var ret *uvarintBitIDIter
	if bid.iter.started {
		ret = &uvarintBitIDIter{}
	} else {
		ret = &bid.iter
	}
	ret.arr = bid.arr
	ret.Done()
	ret.started = true
	return ret
	//return &uvarintBitIDIter{
	//	buff:     *bytes.NewReader(bid.arr),
	// }
}

// AllowDuplicates returns true.
func (bid *UvarintBitID) AllowsDuplicates() bool {
	return true
}

// Done is called when this item is no longer needed
func (bid *UvarintBitID) Done() {
	bid.arr = bid.arr[:0]
	bid.min, bid.max, bid.size, bid.uniqueCount, bid.maxSizeIdx, bid.maxCount = 0, 0, 0, 0, 0, 0
	(&bid.iter).Done()
	bid.hasIteration = false
}

// Returns the smallest element, the largest element, and the total number of elements
func (bid *UvarintBitID) GetBasicInfo() (min, max, count, uniqueCount int) {
	return bid.min, bid.max, bid.size, bid.uniqueCount
}

// Allocate the expected initial size
func (bid *UvarintBitID) SetInitialSize(v int) {
	size := allocUvarintArr(v)
	switch cap(bid.arr) >= size {
	case true:
		// Dont need to allocate
	case false:
		// bid.arr = make([]byte, 0, size)
	}
}

// Append an item at the end of the bitID (must be bigger than all existing items)
func (bid *UvarintBitID) AppendItem(v int) {
	if v < bid.max {
		panic("can only allow larger items")
	}
	if bid.hasIteration {
		panic("cannot add to a bid being iterated")
	}
	bid.size++
	if bid.size == 1 { // first item, set min
		bid.min = v
	} else if bid.max != v { // new item, but not the first
		// write the count of the previous
		_, bid.arr = utils.AppendUvarint(uint64(bid.maxCount), bid.arr)
	}
	if bid.size == 1 || bid.max != v { // new item
		_, bid.arr = utils.AppendUvarint(uint64(v), bid.arr)
		bid.max = v
		bid.maxCount = 0
		bid.uniqueCount++
		bid.maxSizeIdx = len(bid.arr)
	} else { // repeated item
		if v != bid.max {
			panic("invalid order")
		}
		bid.maxCount++
	}
}

func (bid *UvarintBitID) Encode(writer io.Writer) (n int, err error) {
	bid.writeMaxSizeCount() // be sure the max size is written
	var n1 int
	n1, err = utils.EncodeUvarint(uint64(len(bid.arr)), writer)
	n += n1
	if err != nil {
		return
	}
	n1, err = writer.Write(bid.arr)
	n += n1
	return
}
func (bid *UvarintBitID) construct() error {
	// newArr := make([]byte, len(bid.arr))
	// copy(newArr, bid.arr)
	// bid.arr = newArr
	buff := bytes.NewBuffer(bid.arr)
	var n int
	prev := -1
	for n < len(bid.arr) {
		nxt, n1, err := utils.ReadUvarintByteReader(buff)
		if err != nil {
			return err
		}
		n += n1
		if nxt > math.MaxUint32 || int(nxt) <= prev {
			return types.ErrInvalidBitID
		}
		if prev != int(nxt) {
			bid.uniqueCount++
		}
		if bid.size == 0 {
			bid.min = int(nxt)
		}
		prev = int(nxt)
		bid.max = int(nxt)
		bid.maxSizeIdx = n

		count, n1, err := utils.ReadUvarintByteReader(buff)
		if err != nil {
			return err
		}
		n += n1
		if count > math.MaxUint32 {
			return types.ErrInvalidBitID
		}
		bid.maxCount = int(count)
		bid.size += bid.maxCount + 1
	}
	return nil
}
func (bid *UvarintBitID) Decode(reader io.Reader) (n int, err error) {
	n, bid.arr, err = utils.DecodeHelper(reader)
	if err != nil {
		return
	}
	err = bid.construct()
	return
}
func (bid *UvarintBitID) Deserialize(msg *messages.Message) (n int, err error) {
	n, bid.arr, err = messages.DecodeHelperMsg((*messages.MsgBuffer)(msg))
	if err != nil {
		return
	}
	err = bid.construct()
	return
}
