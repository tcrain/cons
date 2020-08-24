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

package types

import (
	"fmt"
	"golang.org/x/crypto/blake2b"
	"hash"
)

// SigType defines the type of signatures used
type SigType int

const (
	EC SigType = iota
	ED
	SCHNORR
	EDCOIN
	BLS
	TBLS
	TBLSDual
	CoinDual
	QSAFE
)

var AllSigTypes = []SigType{EC, ED, SCHNORR, EDCOIN, BLS, TBLS, TBLSDual, CoinDual, QSAFE}
var ThrshSigTypes = []SigType{TBLS, TBLSDual, CoinDual}
var EncryptChannelsSigTypes = []SigType{EC, ED, SCHNORR, EDCOIN, BLS, TBLS, TBLSDual, CoinDual}
var VRFSigTypes = []SigType{EC, BLS}
var MultiSigTypes = []SigType{BLS}
var CoinSigTypes = append(ThrshSigTypes, EDCOIN)

func (ct SigType) String() string {
	switch ct {
	case EC:
		return "EC"
	case EDCOIN:
		return "EDCOIN"
	case BLS:
		return "BLS"
	case ED:
		return "ED"
	case SCHNORR:
		return "SCHNORR"
	case TBLS:
		return "TBLS"
	case CoinDual:
		return "CoinDual"
	case TBLSDual:
		return "TBLSDual"
	case QSAFE:
		return "QSAFE"
	default:
		return fmt.Sprintf("SigType%d", ct)
	}
}

func GetNewHash() hash.Hash {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	return h
	// return sha512.New()
}

func GetHash(v []byte) HashBytes {
	// h := sha512.Sum512_256(v)
	hf, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	hf.Write(v)
	out := hf.Sum(nil)
	if len(out) != GetHashLen() {
		panic(len(out))
	}
	return out
}

// GetZeroBytesHashLength returns a byte slice of 0s the length of a hash
// The slice returned should not be edited.
func GetZeroBytesHashLength() HashBytes {
	return zeroHash
}

var zeroHash = make([]byte, GetHashLen())

func GetHashLen() int {
	return 32
}

type BoolSetting []bool

var WithTrue BoolSetting = []bool{true}
var WithFalse = []bool{false}
var WithBothBool = []bool{true, false}

type SignType int

const (
	NormalSignature SignType = iota
	SecondarySignature
	CoinProof
)

type UseSignaturesType int

const (
	UseSignatures = iota
	NoSignatures
	ConsDependentSignatures
)

func (ust UseSignaturesType) String() string {
	switch ust {
	case UseSignatures:
		return "UseSignatures"
	case NoSignatures:
		return "NoSignatures"
	case ConsDependentSignatures:
		return "ConsDependentSignatures"
	default:
		return fmt.Sprintf("UseSignaturesType%d", ust)
	}
}

type EncryptChannelsType int

const (
	NonEncryptedChannels = iota
	EncryptedChannels
	ConsDepedentChannels
)

func (ect EncryptChannelsType) String() string {
	switch ect {
	case EncryptedChannels:
		return "EncryptedChannels"
	case NonEncryptedChannels:
		return "NoSignatures"
	case ConsDepedentChannels:
		return "ConsDependentChannels"
	default:
		return fmt.Sprintf("EncryptChannelsType%d", ect)
	}
}

type BitIDType int

const (
	BitIDUvarint BitIDType = iota
	BitIDSlice
	BitIDP
	BitIDMulti
	BitIDBasic
	BitIDSingle
)

func (bt BitIDType) String() string {
	switch bt {
	case BitIDMulti:
		return "BitIDMulti"
	case BitIDSlice:
		return "BitIDSlice"
	case BitIDUvarint:
		return "BitIDUvarint"
	case BitIDBasic:
		return "BitIDBasic"
	case BitIDP:
		return "BitIDP"
	case BitIDSingle:
		return "BitIDSingle"
	default:
		panic(bt)
	}
}
