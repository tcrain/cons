package bitid

import (
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"sort"
)

func uvarintEncode(items sort.IntSlice, writer io.Writer) (n int, err error) {
	var n1 int
	n1, err = utils.EncodeUvarint(uint64(len(items)), writer)
	n += n1
	if err != nil {
		return
	}
	for _, nxt := range items {
		n1, err = utils.EncodeUvarint(uint64(nxt), writer)
		n += n1
		if err != nil {
			return
		}
	}
	return
}

func GetBitIDFuncs(id types.BitIDType) (FromIntFunc, NewBitIDFunc) {
	switch id {
	case types.BitIDBasic:
		return NewBitIDFromInts, (&BitID{}).New
	case types.BitIDMulti:
		return NewMultiBitIDFromInts, (&MultiBitID{}).New
	case types.BitIDSlice:
		return NewSliceBitIDFromInts, (&SliceBitID{}).New
	case types.BitIDUvarint:
		return NewUvarintBitIDFromInts, (&UvarintBitID{}).New
	case types.BitIDP:
		return NewPbitidFromInts, (&Pbitid{}).New
	case types.BitIDChoose:
		return NewChooseBIDFromInts, (&ChooseBid{}).New
	default:
		panic(id)
	}
}
