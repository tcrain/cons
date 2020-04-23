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

package asset

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"sort"
)

type AssetAccount struct {
	Pub    sig.Pub
	Assets SortedAssets
}

func NewAssetAccount(pub sig.Pub) *AssetAccount {
	return &AssetAccount{Pub: pub}
}

// AddAsset adds the asset to the account.
func (aa *AssetAccount) AddAsset(asset AssetInterface) {
	aa.Assets = append(aa.Assets, asset)
	sort.Sort(aa.Assets)

	// Sanity check TODO remove
	for i := 1; i < len(aa.Assets); i++ {
		if bytes.Equal(aa.Assets[i-1].GetID(), aa.Assets[i].(AssetInterface).GetID()) {
			panic("duplicate assets")
		}
	}
}

// ConsumeAssets removes the assets from the account.
func (aa *AssetAccount) ConsumeAssets(assetIDs []types.HashBytes) {

	// assets must be sorted
	sorted := make(types.SortedHashBytes, len(assetIDs))
	copy(sorted, assetIDs)
	sort.Sort(sorted)

	var nxtIdx int
	consumedIdxs := make([]int, len(assetIDs))
	for i, nxt := range sorted {
		// We only need to check the remaining assets since sorted
		accountAssets := aa.Assets[nxtIdx:]
		// Check if the asset is owned by this account
		idx := sort.Search(len(accountAssets), func(n int) bool {
			return bytes.Compare(nxt, accountAssets[n].GetID()) <= 0
		})
		if idx == len(accountAssets) {
			panic(errAssetNotFound)
		}
		if !bytes.Equal(nxt, accountAssets[idx].GetID()) {
			panic(errAssetNotFound)
		}
		nxtIdx += idx
		consumedIdxs[i] = nxtIdx
	}

	remainAssets := make(SortedAssets, 0, len(aa.Assets)-len(assetIDs))
	var consumeIdx int
	for i, nxt := range aa.Assets {
		if consumeIdx < len(consumedIdxs) && i == consumedIdxs[consumeIdx] {
			consumeIdx++
		} else {
			remainAssets = append(remainAssets, nxt)
		}
	}
	// sanity check
	if len(remainAssets) != len(aa.Assets)-len(assetIDs) {
		panic("bad consume")
	}

	aa.Assets = remainAssets
}

type sortedIndexHash []indexHash

func (a sortedIndexHash) Len() int      { return len(a) }
func (a sortedIndexHash) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortedIndexHash) Less(i, j int) bool {
	return bytes.Compare(a[i].HashBytes, a[j].HashBytes) < 0
}

type indexHash struct {
	types.HashBytes
	idx int
}

// CheckAssets returns the assets corresponding to the hashes at the account.
// Otherwise if the account does not have all the assets an error is returned.
func (aa *AssetAccount) CheckAssets(toCheck []types.HashBytes) (ret []AssetInterface, err error) {

	if len(toCheck) == 0 {
		return
	}
	if len(aa.Assets) < len(toCheck) {
		return nil, errTooManyAssets
	}

	ret = make([]AssetInterface, len(toCheck))

	// assets must be sorted
	sortedCheck := make(sortedIndexHash, len(toCheck))
	for i, nxt := range toCheck {
		sortedCheck[i] = indexHash{idx: i, HashBytes: nxt}
	}
	sort.Sort(sortedCheck)

	if !sort.IsSorted(sortedCheck) { // sanity check TODO remove
		panic(errAssetsNotSorted)
	}

	accountAssets := aa.Assets
	for i, nxt := range sortedCheck {
		// Check if the asset is owned by this account
		idx := sort.Search(len(accountAssets), func(n int) bool {
			return bytes.Compare(nxt.HashBytes, accountAssets[n].GetID()) <= 0
		})
		if idx == len(accountAssets) {
			return nil, errAssetNotFound
		}
		if !bytes.Equal(nxt.HashBytes, accountAssets[idx].GetID()) {
			return nil, errAssetNotFound
		}
		ret[nxt.idx] = accountAssets[idx]
		// We only need to check the remaining assets since sorted
		accountAssets = accountAssets[idx:]

		// Check if the asset is unique in the to Check list
		remain := sortedCheck[i+1:]
		idx = sort.Search(len(remain), func(n int) bool {
			return bytes.Compare(nxt.HashBytes, remain[n].HashBytes) <= 0
		})
		if idx != len(remain) && bytes.Equal(nxt.HashBytes, remain[idx].HashBytes) {
			return nil, errAssetNotUnique
		}
	}
	return
}
