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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sort"
	"testing"
)

func genBasicAsset(v uint64) *BasicAsset {
	w := bytes.NewBuffer(nil)
	if _, err := utils.EncodeUint64(v, w); err != nil {
		panic(err)
	}
	return &BasicAsset{
		ID: types.GetHash(w.Bytes()),
	}
}

func genIdRange(start, end uint64, doSort bool) (ret types.SortedHashBytes) {
	for i := start; i < end; i++ {
		ret = append(ret, genBasicAsset(i).GetID())
	}
	if doSort {
		sort.Sort(ret)
	}
	return
}

func genAssetRange(start, end uint64, doSort bool) (ret SortedAssets) {
	for i := start; i < end; i++ {
		ret = append(ret, genBasicAsset(i))
	}
	if doSort {
		sort.Sort(ret)
	}
	return
}

func TestAccount(t *testing.T) {

	account := &AssetAccount{
		Assets: genAssetRange(5, 10, true),
	}

	retErr := func(a interface{}, err error) error {
		return err
	}

	assert.Nil(t, retErr(account.CheckAssets(nil)))

	assert.Nil(t, retErr(account.CheckAssets(genIdRange(5, 10, true))))
	assert.Nil(t, retErr(account.CheckAssets(genIdRange(9, 10, true))))

	assert.Nil(t, retErr(account.CheckAssets(genIdRange(5, 10, false))))
	assert.Nil(t, retErr(account.CheckAssets(genIdRange(5, 10, true))))

	assert.Equal(t, errAssetNotFound, retErr(account.CheckAssets(genIdRange(4, 6, false))))
	assert.Equal(t, errAssetNotFound, retErr(account.CheckAssets(genIdRange(8, 11, true))))

	assert.Equal(t, errTooManyAssets, retErr(account.CheckAssets(genIdRange(5, 11, false))))

	asst := append(genIdRange(5, 8, true), genIdRange(6, 7, true)...)

	assert.Equal(t, errAssetNotUnique, retErr(account.CheckAssets(asst)))

	account.ConsumeAssets(genIdRange(5, 7, false))

	assert.Equal(t, errAssetNotFound, retErr(account.CheckAssets(genIdRange(5, 7, false))))
	assert.Nil(t, retErr(account.CheckAssets(genIdRange(7, 10, false))))

	account.ConsumeAssets(genIdRange(7, 10, false))

	assert.Equal(t, errTooManyAssets, retErr(account.CheckAssets(genIdRange(7, 8, false))))
}
