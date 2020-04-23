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

package bls

import (
	"crypto/cipher"
	// "errors"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
)

// NewBLSMSKeyPair creates a new key pair that is based on the new multi-sigs from https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
// this code is modified from https://go.dedis.ch/kyber/v3/blob/master/sign/bls/bls.go
// TODO check if hasing the right objects
// output is public key, private key, public key * hashed public key, hashed public key
func NewBLSMSKeyPair(suite pairing.Suite, random cipher.Stream) (kyber.Scalar, kyber.Point, kyber.Point, kyber.Scalar) {
	x := suite.G2().Scalar().Pick(random)
	X := suite.G2().Point().Mul(x, nil)

	hpk, err := hashScalar(suite, X)
	if err != nil {
		panic(err)
	}

	newPk := X.Clone().Mul(hpk, X)
	return x, X, newPk, hpk
}

// NewBLSMSKeyPairFrom is the same as NewBLSMSKeyPair except uses secret as the key.
func NewBLSMSKeyPairFrom(secret kyber.Scalar, suite pairing.Suite, random cipher.Stream) (kyber.Scalar, kyber.Point, kyber.Point, kyber.Scalar) {
	X := suite.G2().Point().Mul(secret, nil)

	hpk, err := hashScalar(suite, X)
	if err != nil {
		panic(err)
	}

	newPk := X.Clone().Mul(hpk, X)
	return secret, X, newPk, hpk
}

// GetNewPubFromPubBLSMulti multiplies the hash of a public key by its hash
// This is based on https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
// it returns the new public key and the hash
func GetNewPubFromPubBLSMulti(suite pairing.Suite, pub kyber.Point) (kyber.Point, kyber.Scalar, error) {
	hpk, err := hashScalar(suite, pub)
	if err != nil {
		return nil, nil, err
	}

	newPk := pub.Clone().Mul(hpk, pub)
	return newPk, hpk, nil
}

func blsmsFromBytes(b []byte) (kyber.Point, kyber.Point, kyber.Scalar, error) {
	var pub = blssuite.G2().Point()
	if err := pub.UnmarshalBinary(b); err != nil {
		return nil, nil, nil, err
	}
	hpk, err := hashScalar(blssuite, pub)
	if err != nil {
		return nil, nil, nil, err
	}

	newPk := pub.Clone().Mul(hpk, pub)
	return pub, newPk, hpk, nil
}

// SignBLSMS signs a message, returning the signature and the serialzed bytes
// it is based on https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
func SignBLSMS(suite pairing.Suite, x kyber.Scalar, hpk kyber.Scalar, msg []byte) (kyber.Point, []byte, error) {
	HM := hashToPoint(suite, msg)
	xHM := HM.Mul(x, HM)
	newSig := xHM.Mul(hpk, xHM)
	b, err := newSig.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	return newSig, b, nil
}

// TODO is this valid hash for H1 (https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html)?
func hashScalar(suite pairing.Suite, first kyber.Point, points ...kyber.Point) (kyber.Scalar, error) {
	h := suite.Hash()
	s, err := first.MarshalBinary()
	if err != nil {
		return nil, err
	}
	h.Write(s)
	for _, point := range points {
		s, err := point.MarshalBinary()
		if err != nil {
			return nil, err
		}
		h.Write(s)
	}
	ret := suite.G1().Scalar().SetBytes(h.Sum(nil))
	return ret, nil
}
