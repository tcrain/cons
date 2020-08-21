package bitid

import (
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
	DoMakeCopy() NewBitIDInterface // Do make copy returns a copy of the bit id.
	// GetNumItems() int                                         // Returns the number of items in the bitID
	CheckBitID(int) bool                 // Returns true if the argument is in the bid
	GetItemList() sort.IntSlice          // Returns the list of items of the bitid
	GetStr() string                      // Gets the string representation of the bitid
	NewIterator() BIDIter                // Returns a new iterator of the bit id
	GetBasicInfo() (min, max, count int) // Returns the smallest element, the largest element, and the total number of elements
	SetInitialSize(int)                  // Allocate the expected initial size
	AppendItem(int)                      // Append an item at the end of the bitID (must be bigger than all existing items)
	AllowsDuplicates() bool              // AllowDuplicates returns true if the bit ID allows duplicate elements
	Done()                               // Done is called when this item is no longer needed

	Encode(writer io.Writer) (n int, err error)
	Decode(reader io.Reader) (n int, err error)
}

var NewBitIDFuncs = []FromIntFunc{NewBitIDFromInts, NewMultiBitIDFromInts, NewSliceBitIDFromInts, NewUvarintBitIDFromInts}

type NewBitIDFunc func() NewBitIDInterface
type FromIntFunc func(slice sort.IntSlice) NewBitIDInterface
type DoneFunc func(NewBitIDInterface) // Called when the input is finished.

// GetNewItemsHelper returns a bitID of all the items in b2 and not in b1
// (does not return duplicates)
func GetNewItemsHelper(b1, b2 NewBitIDInterface, newFunc NewBitIDFunc) NewBitIDInterface {
	ret := newFunc()
	_, _, b2Size := b2.GetBasicInfo()
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

// HasNewItemsHelper returns true if b2 has at least one item not in b1
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
	_, _, b1Size := b1.GetBasicInfo()
	_, _, b2Size := b2.GetBasicInfo()
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
	_, _, size := b1.GetBasicInfo()
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

	_, _, b1Size := b1.GetBasicInfo()
	if b1Size == 0 {
		return b2.DoMakeCopy(), nil
	}
	_, _, b2Size := b2.GetBasicInfo()
	if b2Size == 0 {
		return b1.DoMakeCopy(), nil
	}
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
