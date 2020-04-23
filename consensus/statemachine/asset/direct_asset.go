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
	"github.com/tcrain/cons/consensus/utils"
)

// DirectAsset is a one-to-one input output
type DirectAsset struct {
	BasicAsset
}

func NewDirectAsset() AssetInterface {
	return &DirectAsset{}
}

func (da *DirectAsset) New() types.EncodeInterface {
	return &DirectAsset{
		BasicAsset: *da.BasicAsset.New().(*BasicAsset)}
}

// Create initial direct asset should be called when the asset is first created,
// asset name is a unique name for the asset, instanceID the unique id of the system
// (e.g. config.CSID).
func CreateInitialDirectAsset(assetName []byte, instanceID []byte) *DirectAsset {

	return &DirectAsset{BasicAsset: *NewBasicAsset(types.GetHash(utils.JoinBytes(assetName, instanceID)))}
}

func GenDirectAssetTransfer(priv sig.Priv, participants []sig.Pub, table *AccountTable) (tr *AssetTransfer) {
	pub := priv.GetPub()
	acc, err := table.GetAccount(pub)
	if err != nil {
		panic(err)
	}
	if len(acc.Assets) == 0 {
		return
	}
	// We send the asset to ourself.
	// We do this so everyone has an asset for the experiment. // TODO make something more insteresting.

	recipiants := make([]sig.Pub, 1)
	inputs := acc.Assets
	for i := range acc.Assets[:1] {
		recipiants[i] = priv.GetPub()
	}
	_, outputs := GenDirectAssetOutput(acc, acc.Assets)
	tr, err = table.GenerateAssetTransfer(priv, inputs, outputs, recipiants, CheckDirectOutputFunc)
	if err != nil {
		panic(err)
	}
	return
}

// genMultipleOutput takes as input the last asset of acc, outputs an output for each receiver.
func GenDirectAssetOutput(acc *AssetAccount, assets []AssetInterface) (inputs []AssetInterface, outputs []AssetInterface) {

	inputs = make([]AssetInterface, len(assets))
	outputs = make([]AssetInterface, len(assets))

	for i, nxt := range assets {
		inputs[i] = nxt
		outputs[i] = &DirectAsset{BasicAsset: *NewBasicAsset(types.GetHash(nxt.GetID()))}
	}
	return
}

func CheckDirectOutputFunc(account *AssetAccount, inputs []AssetInterface, outputs []AssetInterface) error {
	if len(inputs) != len(outputs) {
		return errInvalidAssetCount
	}
	for i, nxt := range inputs {
		_ = nxt.(*DirectAsset)        // sanity check
		_ = outputs[i].(*DirectAsset) // sanity check
		if !bytes.Equal(types.GetHash(nxt.GetID()), outputs[i].GetID()) {
			return errInvalidOutputAsset
		}
	}
	return nil
}
