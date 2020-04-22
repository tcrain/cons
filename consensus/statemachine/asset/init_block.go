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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
)

var InitUniqueName = types.GetHash([]byte(config.CsID))

// InitAssetBlock represents the initial assets.
type InitBlock struct {
	InitPubs       []sig.Pub // The initial public keys.
	InitAssets     []AssetInterface
	InitUniqueName types.HashBytes       // Should be a unique string.
	NewPubFunc     func() sig.Pub        // Used during deserialization, should return an empty public key of the type being used.
	NewAssetFunc   func() AssetInterface // Used to deserialize the assets
}

func NewInitBlock(initPubs []sig.Pub, initAssets []AssetInterface, initUniqueName string) *InitBlock {
	return &InitBlock{
		InitPubs:       initPubs,
		InitAssets:     initAssets,
		InitUniqueName: []byte(initUniqueName),
	}
}

func (ib *InitBlock) Decode(reader io.Reader) (n int, err error) {
	var nxtN int

	ib.InitUniqueName = make(types.HashBytes, types.GetHashLen())
	nxtN, err = reader.Read(ib.InitUniqueName)
	n += nxtN
	if err != nil {
		return
	}

	var pubCount uint16
	pubCount, nxtN, err = utils.ReadUint16(reader)
	n += nxtN
	if err != nil {
		return
	}
	ib.InitPubs = make([]sig.Pub, int(pubCount))
	ib.InitAssets = make([]AssetInterface, int(pubCount))
	for i := range ib.InitPubs {
		ib.InitPubs[i] = ib.NewPubFunc()
		nxtN, err = ib.InitPubs[i].Decode(reader)
		n += nxtN
		if err != nil {
			return
		}
		ib.InitAssets[i] = ib.NewAssetFunc()
		nxtN, err = ib.InitAssets[i].Decode(reader)
		n += nxtN
		if err != nil {
			return
		}
	}
	return
}

func (ib *InitBlock) Encode(writer io.Writer) (n int, err error) {
	if len(ib.InitPubs) > math.MaxUint16 {
		panic(1)
	}
	if !bytes.Equal(InitUniqueName, ib.InitUniqueName) {
		panic("should set unique name")
	}
	var nxtN int
	nxtN, err = writer.Write(InitUniqueName)
	n += nxtN
	if err != nil {
		return
	}

	nxtN, err = utils.EncodeUint16(uint16(len(ib.InitPubs)), writer)
	n += nxtN
	if err != nil {
		return
	}
	for i, pub := range ib.InitPubs {
		nxtN, err = pub.Encode(writer)
		n += nxtN
		if err != nil {
			return
		}
		nxtN, err = ib.InitAssets[i].Encode(writer)
		n += nxtN
		if err != nil {
			return
		}
	}
	return
}
