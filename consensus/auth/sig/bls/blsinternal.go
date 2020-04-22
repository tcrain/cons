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
	"errors"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
)

// This code is modified from https://go.dedis.ch/kyber/v3/blob/master/sign/bls/bls.go
// The funcation represent the standard BLS signatures

// NewBLSKeyPair generates a new random BLS public/private key pair
func NewBLSKeyPair(suite pairing.Suite, random cipher.Stream) (kyber.Scalar, kyber.Point) {
	x := suite.G2().Scalar().Pick(random)
	X := suite.G2().Point().Mul(x, nil)
	return x, X
}

// SignBLS signs a message, and returns the signature unmarshalled, and marshalled
func SignBLS(suite pairing.Suite, x kyber.Scalar, msg []byte) (kyber.Point, []byte, error) {
	HM := hashToPoint(suite, msg)
	xHM := HM.Mul(x, HM)
	b, err := xHM.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	return xHM, b, nil
}

// MarshallSig marshalls a BLS signature
func MarshalSigBLS(sig kyber.Point) ([]byte, error) {
	return sig.MarshalBinary()
}

// Unmarshall sig unmarshalls a BLS signature
func UnmarshalSig(suite pairing.Suite, b []byte) (kyber.Point, error) {
	s := suite.G1().Point()
	err := s.UnmarshalBinary(b)
	return s, err
}

// VerifyBLS checks if a BLS signature is valid
func VerifyBLS(suite pairing.Suite, X kyber.Point, msg []byte, sig kyber.Point) error {
	HM := hashToPoint(suite, msg)
	left := suite.Pair(HM, X)
	right := suite.Pair(sig, suite.G2().Point().Base())
	if !left.Equal(right) {
		return errors.New("bls: invalid signature")
	}
	return nil
}

func hashToPoint(suite pairing.Suite, msg []byte) kyber.Point {
	h := suite.Hash()
	h.Write(msg)
	x := suite.G1().Scalar().SetBytes(h.Sum(nil))
	return suite.G1().Point().Mul(x, nil)
}
