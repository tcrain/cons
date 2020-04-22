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
)

// CheckTransferOutputFunc is used to check if the given asset inputs and outputs for a transfer are valid.
type CheckTransferOutputFunc func(account *AssetAccount, inputs []AssetInterface, outputs []AssetInterface) error

// GenAssetTransferFunc is called during the experiment each time a new transfer asset is needed.
// It returns nil if there is no assets to transfer.
type GenAssetTransferFunc func(priv sig.Priv, participants []sig.Pub, table *AccountTable) (tr *AssetTransfer)

var errNoSender = fmt.Errorf("no sender")
var errNoSignature = fmt.Errorf("no signture")
var errNonUniqueAssets = fmt.Errorf("non-unique assets")

var errInvalidAssetCount = fmt.Errorf("invalid asset count")
var errInvalidAssetReceiverCount = fmt.Errorf("invalid asset receiver count")
var errAssetTooLarge = fmt.Errorf("encoded asset too large")
var errInvalidOutputAsset = fmt.Errorf("invalid output asset")
var errOverflow = fmt.Errorf("inputs would create overflow")
var errNotEnoughInputValue = fmt.Errorf("input value does not cover output")
var errZeroOutput = fmt.Errorf("cannoto have zero output value")

const MaxAssetSize = 10000 // TODO

var errAssetNotFound = fmt.Errorf("asset not found")
var errTooManyAssets = fmt.Errorf("too many assets")
var errAssetsNotSorted = fmt.Errorf("assets not sorted")
var errAssetNotUnique = fmt.Errorf("asset not unique")
