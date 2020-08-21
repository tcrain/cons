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
	"github.com/tcrain/cons/consensus/types"
	"io"
)

type AssetInterface interface {
	// GetOwner() sig.Pub
	GetID() types.HashBytes
	types.EncodeInterface
}

// SortedAssests is a list of assets that can be sorted.
type SortedAssets []AssetInterface

func (sa SortedAssets) Less(i, j int) bool {
	return bytes.Compare(sa[i].GetID(), sa[j].GetID()) < 0
}

func (sa SortedAssets) Len() int {
	return len(sa)
}

func (sa SortedAssets) Swap(i, j int) {
	sa[i], sa[j] = sa[j], sa[i]
}

type BasicAsset struct {
	ID types.HashBytes
	// Owner sig.Pub
}

func NewBasicAsset(id types.HashBytes) *BasicAsset {
	return &BasicAsset{ID: id}
}

// New returns a new empty asset object.
func (ba *BasicAsset) New() types.EncodeInterface {
	return &BasicAsset{
		// Owner: ba.Owner.New(),
	}
}

// Encode the the asset to the writer.
func (ba *BasicAsset) Encode(writer io.Writer) (n int, err error) {
	// if n, err = ba.Owner.DoEncode(writer); err != nil {
	//	return
	// }

	var n1 int
	n1, err = ba.ID.Encode(writer)
	n += n1
	return
}

// Decode the asset from the reader.
func (ba *BasicAsset) Decode(reader io.Reader) (n int, err error) {
	// if n, err = ba.Owner.Decode(reader); err != nil {
	//	return
	// }
	var n1 int
	ba.ID, n1, err = types.DecodeHash(reader)
	n += n1
	return
}

// Marshal the asset.
func (ba *BasicAsset) MarshalBinary() (data []byte, err error) {
	return types.MarshalBinaryHelper(ba)
}

// Unmarshal the asset.
func (ba *BasicAsset) UnmarshalBinary(data []byte) error {
	return types.UnmarshalBinaryHelper(ba, data)
}

// Return the hash id of the asset.
func (ba *BasicAsset) GetID() types.HashBytes {
	return ba.ID
}

// Return the public key owner of the asset.
//func (ba *BasicAsset) GetOwner() sig.Pub {
//	return ba.Owner
//}
