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

package currency

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
)

var InitUniqueName = types.GetHash([]byte("should be some unique name"))

// InitBlock represents the initial balances and participants in the currency.
type InitBlock struct {
	InitPubs       []sig.Pub       // The initial public keys.
	InitBalances   []uint64        // The initial balances.
	InitUniqueName types.HashBytes // Should be a unique string.
	NewPubFunc     func() sig.Pub  // Used during deserialization, should return an empty public key of the type being used.
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
	ib.InitBalances = make([]uint64, int(pubCount))
	for i := range ib.InitPubs {
		ib.InitPubs[i] = ib.NewPubFunc()
		nxtN, err = ib.InitPubs[i].Decode(reader)
		n += nxtN
		if err != nil {
			return
		}
		ib.InitBalances[i], nxtN, err = utils.ReadUint64(reader)
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
		nxtN, err = utils.EncodeUint64(ib.InitBalances[i], writer)
		n += nxtN
		if err != nil {
			return
		}
	}
	return
}
