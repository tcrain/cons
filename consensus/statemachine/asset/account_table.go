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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"strings"
)

type AccountTable struct {
	accountMap map[sig.PubKeyStr]*AssetAccount
	stats      AssetStats
}

func NewAccountTable() *AccountTable {
	return &AccountTable{
		accountMap: make(map[sig.PubKeyStr]*AssetAccount),
	}
}

// GetAccount returns the account object for the given public key.
// A new account is allocated if one is not found.
func (at *AccountTable) GetAccount(pub sig.Pub) (*AssetAccount, error) {
	str, err := pub.GetPubString()
	if err != nil {
		return nil, err
	}
	if acc, ok := at.accountMap[str]; ok {
		return acc, nil
	}
	acc := NewAssetAccount(pub)
	at.accountMap[str] = acc
	return acc, nil
}

// GetAccount info returns the number of assets held by the account.
func (at *AccountTable) GetAccountInfo(pub sig.Pub) (assetCount int, err error) {
	acc, err := at.GetAccount(pub)
	if err != nil {
		return
	}
	return len(acc.Assets), nil
}

func (at *AccountTable) String() string {
	var b strings.Builder
	b.WriteString("----- Account table ------\n")
	for _, acc := range at.accountMap {
		byt, err := acc.Pub.GetPubBytes()
		if err != nil {
			panic(err)
		}
		b.WriteString(fmt.Sprintf("|  (%v) assets: %v\n", byt[:5], len(acc.Assets)))
	}
	b.WriteString("----- End account table -----\n")
	return b.String()
}

var errSignatureInvalid = fmt.Errorf("invalid signature")

// ValidateTransfer returns an error if the asset transfer is not valid. If validate sig is false then
// the transfer's signatures will not be validated.
func (at *AccountTable) ValidateTransfer(tr *AssetTransfer, checkFunc CheckTransferOutputFunc, validateSig bool) (err error) {
	defer func() {
		if err == nil {
			at.stats.PassedValidations++
		} else {
			at.stats.FailedValidations++
		}
	}()

	if err = tr.CheckFormat(); err != nil {
		return
	}

	var acc *AssetAccount
	acc, err = at.GetAccount(tr.Sender)
	if err != nil {
		return
	}
	var inputAssets []AssetInterface
	inputAssets, err = acc.CheckAssets(tr.Inputs)
	if err != nil {
		return
	}
	if validateSig {
		var valid bool
		valid, err = tr.Sender.VerifySig(tr, tr.Signature)
		if !valid {
			err = errSignatureInvalid
			return
		}
		if err != nil {
			return
		}
	}

	if err = checkFunc(acc, inputAssets, tr.Outputs); err != nil {
		return
	}
	return
}

func (at *AccountTable) ConsumeTransfer(tr *AssetTransfer) {
	acc, err := at.GetAccount(tr.Sender)
	if err != nil {
		panic(err)
	}
	// Consume the inputs
	acc.ConsumeAssets(tr.Inputs)

	// Add the outputs
	for i, nxt := range tr.Outputs {
		addAcc, err := at.GetAccount(tr.Receivers[i])
		if err != nil {
			panic(err)
		}
		addAcc.AddAsset(nxt)
	}

	at.stats.AssetConsumedCount += uint64(len(tr.Inputs))

	return
}

// GenerateAssetTransfer generates an asset transfer given the inputs.
// An error is returned if the generated transfer would be invalid.
func (at *AccountTable) GenerateAssetTransfer(sender sig.Priv, inputAssets,
	outPuts []AssetInterface, recipiants []sig.Pub, checkFunc CheckTransferOutputFunc) (tx *AssetTransfer, err error) {

	defer func() {
		if err == nil {
			at.stats.AssetGeneratedCount++
		} else {
			at.stats.FailedAssetGeneratedCount++
		}
	}()

	sendHashes := make([]types.HashBytes, len(inputAssets))
	for i, nxt := range inputAssets {
		sendHashes[i] = nxt.GetID()
	}

	tx = &AssetTransfer{Sender: sender.GetPub(),
		Inputs:    sendHashes,
		Outputs:   outPuts,
		Receivers: recipiants}

	err = tx.encodeSignedBytes()
	if err != nil {
		return
	}
	asig, err := sender.Sign(tx)
	if err != nil {
		return
	}
	tx.Signature = asig

	err = tx.CheckFormat()
	if err != nil {
		return
	}

	err = at.ValidateTransfer(tx, checkFunc, false)
	return
}
