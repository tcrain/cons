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
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"testing"
)

// genMultipleOutput takes as input the last asset of acc, outputs an output for each receiver.
func genMultipleOutput(acc *AssetAccount, receivers []sig.Pub) (inputs, outputs []AssetInterface) {
	inputs = acc.Assets[len(acc.Assets)-1:]
	outputs = make([]AssetInterface, len(receivers))
	for i := range receivers {
		w := bytes.NewBuffer(nil)
		_, err := utils.EncodeUvarint(uint64(i), w)
		if err != nil {
			panic(err)
		}
		outputs[i] = NewBasicAsset(types.GetHash(append(inputs[0].GetID(), w.Bytes()...)))
	}
	return
}

func checkMultipleOutputFunc(account *AssetAccount, inputs, outputs []AssetInterface) error {
	if len(account.Assets) == 0 || len(inputs) != 1 {
		return errInvalidAssetCount
	}
	if !bytes.Equal(account.Assets[len(account.Assets)-1].GetID(), inputs[0].GetID()) {
		return fmt.Errorf("invalid asset")
	}
	for i, nxt := range outputs {
		w := bytes.NewBuffer(nil)
		_, err := utils.EncodeUvarint(uint64(i), w)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(nxt.GetID(), types.GetHash(append(inputs[0].GetID(), w.Bytes()...))) {
			return fmt.Errorf("invalid asset output")
		}
	}
	return nil
}

// genConsumeInput takes any number of inputs and gives no outputs.
func genConsumeInput(acc *AssetAccount) (inputs, outputs []AssetInterface) {
	inputs = acc.Assets
	return
}

func checkConsumeOutputFunc(account *AssetAccount, inputs, outputs []AssetInterface) error {
	if len(outputs) != 0 {
		return fmt.Errorf("should have consumed outputs")
	}
	return nil
}

func TestAccountTableConsumeAssets(t *testing.T) {
	priv, err := ec.NewEcpriv()
	assert.Nil(t, err)
	pub := priv.GetPub()

	tab := NewAccountTable()

	acc, err := tab.GetAccount(pub)
	assert.Nil(t, err)

	asts := genAssetRange(1, 10, false)
	for _, as := range asts {
		acc.AddAsset(as)
	}
	count, err := tab.GetAccountInfo(pub)
	assert.Nil(t, err)
	assert.Equal(t, len(asts), count)

	inputs, outputs := genConsumeInput(acc)
	at, err := tab.GenerateAssetTransfer(priv, inputs, outputs, nil, checkConsumeOutputFunc)
	assert.Nil(t, err)

	// validate the output
	err = tab.ValidateTransfer(at, checkConsumeOutputFunc, true)
	assert.Nil(t, err)

	// consume it
	tab.ConsumeTransfer(at)

	// it should no longer be valid
	err = tab.ValidateTransfer(at, checkConsumeOutputFunc, true)
	assert.Equal(t, errTooManyAssets, err)
}

func TestAccountTableOutputAssets(t *testing.T) {
	accCount := 10
	privs := make([]sig.Priv, accCount)
	pubs := make([]sig.Pub, accCount)
	var err error
	for i := range privs {
		privs[i], err = ec.NewEcpriv()
		assert.Nil(t, err)
		pubs[i] = privs[i].GetPub()
	}

	tab := NewAccountTable()

	acc, err := tab.GetAccount(pubs[0])
	assert.Nil(t, err)

	asts := genAssetRange(1, 11, false)
	for _, as := range asts {
		acc.AddAsset(as)
	}
	count, err := tab.GetAccountInfo(pubs[0])
	assert.Nil(t, err)
	assert.Equal(t, len(asts), count)

	for i, pub := range pubs {
		acc, err := tab.GetAccount(pub)
		assert.Nil(t, err)
		inputs, outputs := genMultipleOutput(acc, pubs[i+1:])
		at, err := tab.GenerateAssetTransfer(privs[i], inputs, outputs, pubs[i+1:], checkMultipleOutputFunc)
		assert.Nil(t, err)

		// validate the output
		err = tab.ValidateTransfer(at, checkMultipleOutputFunc, true)
		assert.Nil(t, err)

		// consume it
		tab.ConsumeTransfer(at)

		// it should no longer be valid
		err = tab.ValidateTransfer(at, checkConsumeOutputFunc, true)
		switch i {
		case 1:
			assert.Equal(t, errTooManyAssets, err)
		default:
			assert.Equal(t, errAssetNotFound, err)
		}
	}

	// check the number of assets
	for i, pub := range pubs {
		assetCount, err := tab.GetAccountInfo(pub)
		assert.Nil(t, err)
		switch {
		case i == 0:
			// started with 10, consumed 1
			assert.Equal(t, 9, assetCount)
		default:
			// added i, consumed i
			assert.Equal(t, i-1, assetCount)
		}
	}
}
