package bitid

import (
	"github.com/tcrain/cons/consensus/utils"
	"sort"
)

func MergeMultiBitID(bid1 *MultiBitID, bid2 *MultiBitID) *MultiBitID {
	// newItemList := make(sort.IntSlice, len(bid1.itemList) + len(bid2.itemList))
	// n := copy(newItemList, bid1.itemList)
	// copy(newItemList[n:], bid2.itemList)
	// sort.Sort(newItemList)
	newItemList := utils.SortSorted(bid1.itemList, bid2.itemList)

	return &MultiBitID{
		itemList: newItemList}
}

func MergeMultiBitIDList(bidList ...BitIDInterface) *MultiBitID {
	// newItemList := make(sort.IntSlice, len(bid1.itemList) + len(bid2.itemList))
	// n := copy(newItemList, bid1.itemList)
	// copy(newItemList[n:], bid2.itemList)
	// sort.Sort(newItemList)
	items := make([]sort.IntSlice, len(bidList))
	for i, bid := range bidList {
		items[i] = bid.GetItemList()
	}

	return &MultiBitID{
		itemList: utils.SortSortedList(items...)}
}

func MergeMultiBitIDNoDup(bid1 *MultiBitID, bid2 *MultiBitID) *MultiBitID {
	var err error
	if bid1 == nil {
		bid1, err = CreateMultiBitIDFromInts(nil)
		if err != nil {
			panic(err)
		}
	}
	if bid2 == nil {
		bid2, err = CreateMultiBitIDFromInts(nil)
		if err != nil {
			panic(err)
		}
	}
	nonDupList := utils.SortSortedNoDuplicates(bid1.itemList, bid2.itemList)

	// newItemList := make(sort.IntSlice, len(bid1.itemList) + len(bid2.itemList))
	// if len(newItemList) == 0 {
	//	return &MultiBitID{}
	// }
	// n := copy(newItemList, bid1.itemList)
	// copy(newItemList[n:], bid2.itemList)
	// sort.Sort(newItemList)

	// nonDupList := make(sort.IntSlice, 1, len(newItemList))
	// nonDupList[0] = newItemList[0]
	// for _, nxt := range newItemList[1:] {
	//	if nxt != nonDupList[len(nonDupList)-1] {
	//		nonDupList = append(nonDupList, nxt)
	//	}
	// }
	return &MultiBitID{
		itemList: nonDupList}
}

// SubMultiBitID assumes bid1 and bid2 are already valid to subtract
// bid 2 is the smaller one
func SubMultiBitID(bid1 *MultiBitID, bid2 *MultiBitID) *MultiBitID {
	newItemList := make(sort.IntSlice, 0, len(bid1.itemList))
	partialItemList := bid2.itemList
	for _, nxt := range bid1.itemList {
		idx := sort.SearchInts(partialItemList, nxt)
		rest := idx
		if idx < len(partialItemList) && partialItemList[idx] == nxt {
			// we found it so it is removed
			rest++
		} else {
			newItemList = append(newItemList, nxt)
		}
		if rest < len(partialItemList) {
			partialItemList = partialItemList[rest:]
		} else {
			partialItemList = nil
		}
	}
	return &MultiBitID{
		itemList: newItemList}
}
