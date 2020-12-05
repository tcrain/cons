package bitid

import (
	"bytes"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math/bits"
	"sort"
)

func NewChooseBIDFromInts(items sort.IntSlice) NewBitIDInterface {
	return &ChooseBid{
		sli: *(NewSliceBitIDFromInts(items).(*SliceBitID)),
	}
}

type ChooseBid struct {
	NewBitIDInterface
	sli SliceBitID
	str string
}

// New allocates a new bit id.
func (bid *ChooseBid) New() NewBitIDInterface {
	return &ChooseBid{}
}

// DoMakeCopy returns a copy of the bit id.
func (bid *ChooseBid) DoMakeCopy() NewBitIDInterface {
	return &ChooseBid{
		NewBitIDInterface: bid.NewBitIDInterface.DoMakeCopy(),
		sli:               *(bid.sli.DoMakeCopy().(*SliceBitID)),
	}
}

// Returns the list of items of the bitid
func (bid *ChooseBid) GetItemList() sort.IntSlice {
	if bid.NewBitIDInterface != nil {
		return bid.NewBitIDInterface.GetItemList()
	}
	return bid.sli.GetItemList()
}

// Gets the string representation of the bitid
func (bid *ChooseBid) GetStr() string {
	if bid.str == "" {
		buff := bytes.NewBuffer(nil)
		if _, err := bid.Encode(buff); err != nil {
			panic(err)
		}
		bid.str = buff.String()
	}
	return bid.str
}

// Returns a new iterator of the bit id
func (bid *ChooseBid) NewIterator() BIDIter {
	if bid.NewBitIDInterface != nil {
		return bid.NewBitIDInterface.NewIterator()
	}
	return bid.sli.NewIterator()
}

// Returns the smallest element, the largest element, and the total number of elements
func (bid *ChooseBid) GetBasicInfo() (min, max, count, uniqueCount int) {
	if bid.NewBitIDInterface != nil {
		return bid.NewBitIDInterface.GetBasicInfo()
	}
	return bid.sli.GetBasicInfo()
}

// Allocate the expected initial size
func (bid *ChooseBid) SetInitialSize(int) {
	// TODO
}

// Append an item at the end of the bitID (must be bigger than all existing items)
func (bid *ChooseBid) AppendItem(v int) {
	bid.sli.AppendItem(v)
}

// AllowDuplicates returns true.
func (bid *ChooseBid) AllowsDuplicates() bool {
	return true
}

// Done is called when this item is no longer needed
func (bid *ChooseBid) Done() {
	bid.sli.Done()
	if bid.NewBitIDInterface != nil {
		bid.NewBitIDInterface.Done()
		bid.NewBitIDInterface = nil
	}
}

func (bid *ChooseBid) Encode(writer io.Writer) (n int, err error) {
	if bid.NewBitIDInterface == nil {
		_, max, count, unique := bid.sli.GetBasicInfo()
		// estimated number of bytes used by the different bitid types
		pBidSize := ((max + count) >> 3) + 1
		uvarintSize := ((((64 - bits.LeadingZeros64(uint64(max))) >> 3) + 1) * unique) + unique
		switch pBidSize <= uvarintSize {
		case true:
			bid.NewBitIDInterface = NewPbitidFromInts(bid.sli.GetItemList())
		case false:
			bid.NewBitIDInterface = NewUvarintBitIDFromInts(bid.sli.GetItemList())
		}
	}
	var bidType types.BitIDType
	switch bid.NewBitIDInterface.(type) {
	case *Pbitid:
		bidType = types.BitIDP
	case *UvarintBitID:
		bidType = types.BitIDUvarint
	}
	if n, err = writer.Write([]byte{byte(bidType)}); err != nil {
		return n, err
	}

	n, err = bid.NewBitIDInterface.Encode(writer)
	return n + 1, err
}
func (bid *ChooseBid) Decode(reader io.Reader) (n int, err error) {
	var typ []byte
	_, typ, err = utils.ReadBytes(1, reader)
	if err != nil {
		return
	}
	switch types.BitIDType(typ[0]) {
	case types.BitIDP:
		bid.NewBitIDInterface = NewPbitidFromInts(nil)
	case types.BitIDUvarint:
		bid.NewBitIDInterface = NewUvarintBitIDFromInts(nil)
	default:
		return 1, types.ErrInvalidBitID
	}
	n, err = bid.NewBitIDInterface.Decode(reader)
	return n + 1, err
}
func (bid *ChooseBid) Deserialize(msg *messages.Message) (n int, err error) {
	var typ []byte
	_, typ, err = utils.ReadBytes(1, (*messages.MsgBuffer)(msg))
	if err != nil {
		return
	}
	switch types.BitIDType(typ[0]) {
	case types.BitIDP:
		bid.NewBitIDInterface = NewPbitidFromInts(nil)
	case types.BitIDUvarint:
		bid.NewBitIDInterface = NewUvarintBitIDFromInts(nil)
	default:
		return 1, types.ErrInvalidBitID
	}
	n, err = bid.NewBitIDInterface.Deserialize(msg)
	return n + 1, err
}
