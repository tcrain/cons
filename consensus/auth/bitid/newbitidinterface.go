package bitid

import (
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"sort"
)

type BIDIter interface {
	NextID() (nxt int, err error) // For iterating through the bitid, an iterator is created using NewBitIDIterator(), returns an error if the iterator has traversed all items
	Done()                        // Done is called when the iterator is no longer needed
}

type NewBitIDInterface interface {
	New() NewBitIDInterface        // New allocates a new bit id.
	DoMakeCopy() NewBitIDInterface // DoMakeCopy returns a copy of the bit id.
	// GetNumItems() int                                         // Returns the number of items in the bitID
	// CheckBitID(int) bool                              // Returns true if the argument is in the bid
	GetItemList() sort.IntSlice                       // Returns the list of items of the bitid
	GetStr() string                                   // Gets the string representation of the bitid
	NewIterator() BIDIter                             // Returns a new iterator of the bit id
	GetBasicInfo() (min, max, count, uniqueCount int) // Returns the smallest element, the largest element, and the total number of elements
	SetInitialSize(int)                               // Allocate the expected initial size
	AppendItem(int)                                   // Append an item at the end of the bitID (must be bigger than all existing items)
	AllowsDuplicates() bool                           // AllowDuplicates returns true if the bit ID allows duplicate elements
	Done()                                            // Done is called when this item is no longer needed

	Encode(writer io.Writer) (n int, err error)
	Decode(reader io.Reader) (n int, err error)
	Deserialize(msg *messages.Message) (n int, err error)
}

var NewBitIDFuncs = []FromIntFunc{NewBitIDFromInts, NewMultiBitIDFromInts, NewSliceBitIDFromInts, NewUvarintBitIDFromInts}

type NewBitIDFunc func() NewBitIDInterface
type FromIntFunc func(slice sort.IntSlice) NewBitIDInterface
type DoneFunc func(NewBitIDInterface) // Called when the input is finished.

// GetNewItemsHelper returns a bitID of all the items in b2 and not in b1
// (does not return duplicates)
func GetNewItemsHelper(b1, b2 NewBitIDInterface, newFunc NewBitIDFunc) NewBitIDInterface {
	ret := newFunc()
	_, _, b2Size, _ := b2.GetBasicInfo()
	ret.SetInitialSize(b2Size)
	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() {
		iter1.Done()
		iter2.Done()
	}()

	lastAdded := -1

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			if lastAdded != nxt2 {
				lastAdded = nxt2
				ret.AppendItem(nxt2)
			}
			nxt2, err2 = iter2.NextID()
		case nxt1 < nxt2:
			nxt1, err1 = iter1.NextID()
		default:
			val := nxt1
			for nxt1 == val && err1 == nil {
				nxt1, err1 = iter1.NextID()
			}
			val = nxt2
			for nxt2 == val && err2 == nil {
				nxt2, err2 = iter2.NextID()
			}
		}
	}
	for err2 == nil {
		if lastAdded != nxt2 {
			lastAdded = nxt2
			ret.AppendItem(nxt2)
		}
		nxt2, err2 = iter2.NextID()
	}
	return ret
}

// HasNewItemsHelper returns true if b2 has at least one item not in b1.
// Duplicates are only checked once.
func HasNewItemsHelper(b1, b2 NewBitIDInterface) bool {
	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() { iter1.Done(); iter2.Done() }()

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			return true
		case nxt1 < nxt2:
			nxt1, err1 = iter1.NextID()
		default:
			val := nxt1
			for nxt1 == val && err1 == nil {
				nxt1, err1 = iter1.NextID()
			}
			val = nxt2
			for nxt2 == val && err2 == nil {
				nxt2, err2 = iter2.NextID()
			}
		}
	}
	if err2 != nil {
		return false
	}
	return true
}

// HasNewItemsBothHelper returns true if b3 has at least one item not in b1 and not in b2
func HasNewItemsBothHelper(b1, b2, b3 NewBitIDInterface) bool {
	iter1, iter2, iter3 := b1.NewIterator(), b2.NewIterator(), b3.NewIterator()
	defer func() { iter1.Done(); iter3.Done() }()

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	nxt3, err3 := iter3.NextID()
	for (err1 == nil || err2 == nil) && err3 == nil {
		switch {
		case (err1 != nil || nxt1 > nxt3) && (err2 != nil || nxt2 > nxt3):
			return true
		case nxt1 < nxt3 && err1 == nil:
			nxt1, err1 = iter1.NextID()
		case nxt2 < nxt3 && err2 == nil:
			nxt2, err2 = iter2.NextID()
		default:
			val := nxt1
			for nxt1 == val && err1 == nil {
				nxt1, err1 = iter1.NextID()
			}
			val = nxt2
			for nxt2 == val && err2 == nil {
				nxt2, err2 = iter2.NextID()
			}
			val = nxt3
			for nxt3 == val && err3 == nil {
				nxt3, err3 = iter3.NextID()
			}
		}
	}
	if err3 != nil {
		return false
	}
	return true
}

// CheckIntersectionHelper returns the items in both bitid (allows duplicates)
func CheckIntersectionHelper(b1, b2 NewBitIDInterface) sort.IntSlice {
	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() { iter1.Done(); iter2.Done() }()

	var ret sort.IntSlice

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			nxt2, err2 = iter2.NextID()
		case nxt1 < nxt2:
			nxt1, err1 = iter1.NextID()
		default:
			ret = append(ret, nxt1)
			nxt2, err2 = iter2.NextID()
			nxt1, err1 = iter1.NextID()
		}
	}
	return ret
}

// HasIntersectionHelper returns true if there is at least one item commit to both bitids
func HasIntersectionHelper(b1, b2 NewBitIDInterface) bool {
	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() { iter1.Done(); iter2.Done() }()

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			nxt2, err2 = iter2.NextID()
		case nxt1 < nxt2:
			nxt1, err1 = iter1.NextID()
		default:
			return true
		}
	}
	return false
}

// IntersectionCountHelper returns the number of intersecting items (allows duplicates).
func HasIntersectionCountHelper(b1, b2 NewBitIDInterface) int {
	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() { iter1.Done(); iter2.Done() }()

	var ret int
	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			nxt2, err2 = iter2.NextID()
		case nxt1 < nxt2:
			nxt1, err1 = iter1.NextID()
		default:
			ret++
			nxt2, err2 = iter2.NextID()
			nxt1, err1 = iter1.NextID()
		}
	}
	return ret
}

type SafeSubType int

const (
	NotSafe                 SafeSubType = iota // subtract any two bit ids
	NonIntersecting                            // fail if the one being subtracted is not contained in the outter one
	NonIntersectingAndEmpty                    // also fail is the resulting bid id would be empty
)

// SubHelper subtracts b2 from b1 (allows duplicates).
// If safeSub is true then all b1 must contain all items in b2.
// If freeB1 is non nil, then it will be called on b1.
func SubHelper(b1, b2 NewBitIDInterface, safeSub SafeSubType, newFunc NewBitIDFunc, freeB1 DoneFunc) (NewBitIDInterface, error) {
	_, _, b1Size, _ := b1.GetBasicInfo()
	_, _, b2Size, _ := b2.GetBasicInfo()
	switch safeSub {
	case NonIntersectingAndEmpty:
		if b2Size >= b1Size {
			return nil, types.ErrInvalidBitIDSub
		}
	case NonIntersecting:
		if b2Size > b1Size {
			return nil, types.ErrInvalidBitIDSub
		}
	}

	iter1 := b1.NewIterator()
	iter2 := b2.NewIterator()
	defer func() {
		iter1.Done()
		iter2.Done()
		if freeB1 != nil {
			freeB1(b1)
		}
	}()

	ret := newFunc()
	ret.SetInitialSize(b1Size)
	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 > nxt2:
			if safeSub != NotSafe {
				return nil, types.ErrInvalidBitIDSub
			}
			nxt2, err2 = iter2.NextID()
		case nxt2 > nxt1:
			ret.AppendItem(nxt1)
			nxt1, err1 = iter1.NextID()
		default:
			nxt1, err1 = iter1.NextID()
			nxt2, err2 = iter2.NextID()
		}
	}
	if safeSub != NotSafe && err1 != nil && err2 == nil {
		return nil, types.ErrInvalidBitIDSub
	}
	for err1 == nil {
		ret.AppendItem(nxt1)
		nxt1, err1 = iter1.NextID()
	}
	return ret, nil
}

// findHelper returns true if v is in b1
func findHepler(v int, b1 NewBitIDInterface) bool {
	iter := b1.NewIterator()
	defer iter.Done()
	nxt, err := iter.NextID()
	for err == nil {
		if nxt == v {
			return true
		}
	}
	return false
}

// toSliceHelper returns the slice of elements in the bid
func toSliceHelper(b1 NewBitIDInterface) sort.IntSlice {
	_, _, size, _ := b1.GetBasicInfo()
	ret := make(sort.IntSlice, 0, size)
	iter := b1.NewIterator()
	defer iter.Done()

	nxt, err := iter.NextID()
	for err == nil {
		ret = append(ret, nxt)
		nxt, err = iter.NextID()
	}
	return ret
}

// AddHelper adds the items in b2 to b1.
// If allowsDuplicates is false then duplicates will only be counted once.
// If errorOnDuplicate it true then an error will be returned if a duplicate is found.
// If freeB1 is non-nil, then it will be called on b1.
func AddHelper(b1, b2 NewBitIDInterface, allowDuplicates, errorOnDuplicate bool,
	newFunc NewBitIDFunc, freeB1 DoneFunc) (NewBitIDInterface, error) {

	defer func() {
		if freeB1 != nil {
			freeB1(b1)
		}
	}()

	_, _, b1Size, _ := b1.GetBasicInfo()
	_, _, b2Size, _ := b2.GetBasicInfo()
	lastAdded := -1

	iter1, iter2 := b1.NewIterator(), b2.NewIterator()
	defer func() { iter1.Done(); iter2.Done() }()

	ret := newFunc()
	ret.SetInitialSize(b1Size + b2Size)

	nxt1, err1 := iter1.NextID()
	nxt2, err2 := iter2.NextID()
	for err1 == nil && err2 == nil {
		switch {
		case nxt1 >= nxt2:
			if errorOnDuplicate && lastAdded == nxt2 {
				return nil, types.ErrIntersectingBitIDs
			}
			if allowDuplicates || lastAdded != nxt2 {
				ret.AppendItem(nxt2)
				lastAdded = nxt2
			}
			nxt2, err2 = iter2.NextID()
		default:
			if errorOnDuplicate && lastAdded == nxt1 {
				return nil, types.ErrIntersectingBitIDs
			}
			if allowDuplicates || lastAdded != nxt1 {
				ret.AppendItem(nxt1)
				lastAdded = nxt1
			}
			nxt1, err1 = iter1.NextID()
		}
	}
	for err1 == nil {
		if errorOnDuplicate && lastAdded == nxt1 {
			return nil, types.ErrIntersectingBitIDs
		}
		if allowDuplicates || lastAdded != nxt1 {
			ret.AppendItem(nxt1)
			lastAdded = nxt1
		}
		nxt1, err1 = iter1.NextID()
	}
	for err2 == nil {
		if errorOnDuplicate && lastAdded == nxt2 {
			return nil, types.ErrIntersectingBitIDs
		}
		if allowDuplicates || lastAdded != nxt2 {
			ret.AppendItem(nxt2)
			lastAdded = nxt2
		}
		nxt2, err2 = iter2.NextID()
	}
	return ret, nil
}

type iterVal struct {
	val  int
	iter BIDIter
}

func remIter(idx int, iv []iterVal) []iterVal {
	iv[idx] = iv[len(iv)-1]
	return iv[:len(iv)-1]
}

func getMinIdx(iv []iterVal) int {
	ret := iv[0]
	idx := 0
	for i, nxt := range iv {
		if nxt.val < ret.val {
			ret = nxt
			idx = i
		}
	}
	return idx
}

// AddHelper adds the items in the bitids together.
// If allowsDuplicates is false then duplicates will only be counted once.
// If errorOnDuplicate it true then an error will be returned if a duplicate is found.
// If freeB1 is non-nil, then it will be called on b1.
func AddHelperSet(allowDuplicates, errorOnDuplicate bool,
	newFunc NewBitIDFunc, bids ...NewBitIDInterface) (NewBitIDInterface, error) {

	iters := make([]BIDIter, len(bids))
	for i, nxt := range bids {
		iters[i] = nxt.NewIterator()
	}
	defer func() {
		for _, nxt := range iters {
			nxt.Done()
		}
	}()

	idxs := make([]iterVal, 0, len(bids))
	for _, nxt := range iters {
		if v, err := nxt.NextID(); err == nil {
			idxs = append(idxs, iterVal{
				val:  v,
				iter: nxt,
			})
		}
	}

	prev := -1
	ret := newFunc()
	for len(idxs) > 0 {
		nxt := getMinIdx(idxs)
		val := idxs[nxt].val
		if val == prev {
			if errorOnDuplicate {
				return nil, types.ErrIntersectingBitIDs
			}
			if allowDuplicates {
				ret.AppendItem(val)
			}
		} else {
			ret.AppendItem(val)
		}
		if nxtV, err := idxs[nxt].iter.NextID(); err != nil {
			idxs = remIter(nxt, idxs)
		} else {
			idxs[nxt].val = nxtV
		}
		prev = val
	}
	return ret, nil
}
