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

/*
In order to describe which nodes have signed a multi-signature, the indecies of the sorted list of consensus
participants public keys are encoded into binary objects called BitIDs, this is still in development.
*/
package bitid

import (
	"github.com/tcrain/cons/consensus/types"
	"sort"

	"github.com/tcrain/cons/config"
)

type BitIDInterface interface {
	MakeCopy() BitIDInterface
	Encode() []byte
	GetNumItems() int                                         // Returns the number of items in the bitID
	GetNewItems(BitIDInterface) BitIDInterface                // Returns a bitID of all the items in the agrument bitid and not in the bitid calling the function (does not return duplicates)
	HasNewItems(BitIDInterface) bool                          // Returns true if the argument bitid has at least one item not in the bitid calling the function
	CheckIntersection(BitIDInterface) sort.IntSlice           // Return the items in both bitid (allows duplicates)
	CheckBitID(int) bool                                      // Returns true if the argument is in the bid
	AddBitID(id int, allowDup bool, iter *BitIDIterator) bool // Adds the argument to the bid and returns if successful (if allowDup is true then always returns true)
	GetItemList() sort.IntSlice                               // Returns the list of items of the bitid
	GetStr() string                                           // Gets the string representation of the bitid
	HasIntersection(BitIDInterface) bool                      // Returns true if there is at least one item commit to both bitids
	// GetBuff() []byte

	// For iteration
	NextID(iter *BitIDIterator) (nxt int, err error) // For iterating through the bitid, an iterator is created using NewBitIDIterator(), returns an error if the iterator has traversed all items
}

type BitIDType int

const (
	BitIDBasic BitIDType = iota
	BitIDSingle
	BitIDMulti
	BitIDP
)

type BitIDIterator struct {
	iterIdx    int
	currentVal int
	prevVal    bool
}

func NewBitIDIterator() *BitIDIterator {
	return &BitIDIterator{prevVal: true}
}

func AddBitIDType(dst BitIDInterface, src BitIDInterface, allowDup bool) {
	iter := NewBitIDIterator()
	internalIter := NewBitIDIterator()

	for i, err := src.NextID(iter); err == nil; i, err = src.NextID(iter) {
		dst.AddBitID(i, allowDup, internalIter)
	}
}

func CreateBitIDTypeFromInts(items sort.IntSlice) (BitIDInterface, error) {
	if !config.AllowMultiMerge {
		return CreateBitIDFromInts(items)
	}
	return CreateMultiBitIDFromInts(items)
}

func DecodeBitIDType(buff []byte) (BitIDInterface, error) {
	t := BitIDType(buff[0])
	switch t {
	case BitIDMulti, BitIDP:
		if !config.AllowMultiMerge {
			return nil, types.ErrInvalidBitIDEncoding
		}
	}
	switch t {
	case BitIDBasic:
		return DecodeBitID(buff)
	case BitIDSingle:
		return DecodeBitID(buff)
	case BitIDMulti:
		return DecodeMultiBitID(buff)
	case BitIDP:
		return DecodePbitid(buff)
	default:
		return nil, types.ErrInvalidBitIDEncoding
	}
}

func SubBitIDType(b1 BitIDInterface, b2 BitIDInterface) BitIDInterface {
	if !config.AllowMultiMerge {
		return SubBitID(b1.(*BitID), b2.(*BitID))
	}
	return SubMultiBitID(b1.(*MultiBitID), b2.(*MultiBitID))
}

func MergeBitIDType(b1 BitIDInterface, b2 BitIDInterface, allowDuplicates bool) BitIDInterface {
	if !config.AllowMultiMerge {
		if allowDuplicates {
			panic("cant allow duplicates without multimerge enabled")
		}
		return MergeBitID(b1.(*BitID), b2.(*BitID))
	}
	if allowDuplicates {
		return MergeMultiBitID(b1.(*MultiBitID), b2.(*MultiBitID))
	}
	return MergeMultiBitIDNoDup(b1.(*MultiBitID), b2.(*MultiBitID))
}

func MergeBitIDListType(allowDuplicates bool, bList ...BitIDInterface) BitIDInterface {
	if !config.AllowMultiMerge {
		return MergeBitIDList(bList...)
		// if len(bList) == 0 {
		// 	return nil
		// }
		// mrg := bList[0]
		// for _, nxt := range bList[1:] {
		// 	mrg = MergeBitID(mrg.(*BitID), nxt.(*BitID))
		// }
		// return mrg
	}
	if allowDuplicates {
		return MergeMultiBitIDList(bList...)
	}
	panic("unsupported")
	// return MergeMultiBitIDNoDup(b1.(*MultiBitID), b2.(*MultiBitID))
}
