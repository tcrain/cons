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

package coinproof

import (
	"bytes"
	"crypto/sha512"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/proof/dleq"
	"go.dedis.ch/kyber/v3/share"
	"testing"
)

func TestCoinProof(t *testing.T) {
	doTestCoinProof(edwards25519.NewBlakeSHA256Ed25519(), t)
}

func doTestCoinProof(suite dleq.Suite, t *testing.T) {

	msg := []byte("coin")

	sharedSecret := suite.Scalar().Pick(suite.RandomStream())
	sharedPub := suite.Point().Mul(sharedSecret, nil)
	localSecret := suite.Scalar().Pick(suite.RandomStream())
	localPub := suite.Point().Mul(localSecret, nil)

	prf, err := CreateCoinProof(suite, sharedPub, msg, localSecret)
	assert.Nil(t, err)

	assert.Nil(t, prf.Validate(localPub, sharedPub, msg))

	w := bytes.NewBuffer(nil)
	n, err := prf.Encode(w)
	assert.Nil(t, err)

	r := bytes.NewBuffer(w.Bytes())
	decPrf := EmptyCoinProof(suite)

	n1, err := decPrf.Decode(r)
	assert.Nil(t, err)
	assert.Equal(t, n, n1)

	assert.Nil(t, decPrf.Validate(localPub, sharedPub, msg))

}

func TestCoin(test *testing.T) {

	msg := []byte("coin")
	hash := sha512.New()
	hash.Write(msg)

	suite := edwards25519.NewBlakeSHA256Ed25519()

	secret := suite.Scalar().Pick(suite.RandomStream())
	group := suite
	coinHashScalar := group.Scalar().SetBytes(hash.Sum(nil))
	coinHashPoint := group.Point().Mul(coinHashScalar, nil)

	n := config.Thrshn
	t := config.Thrsht

	priPoly := share.NewPriPoly(group, t, secret, suite.RandomStream())
	pubPoly := priPoly.Commit(group.Point().Base())

	coinPoint := group.Point().Mul(priPoly.Secret(), coinHashPoint)

	var sharesPriv []*share.PubShare
	for _, x := range priPoly.Shares(n) {
		sharesPriv = append(sharesPriv, &share.PubShare{
			I: x.I,
			V: group.Point().Mul(x.V, coinHashPoint),
		})

		g := pubPoly.Commit() // Shared public key
		xi := x.V             // local private key

		prf, xG, xH, err := dleq.NewDLEQProof(suite, g, coinHashPoint, xi)
		assert.Nil(test, err)

		err = prf.Verify(suite, g, coinHashPoint, xG, xH)
		assert.Nil(test, err)

	}

	for i := 0; i < n-t; i++ {

		recoveredCoinPoint, err := share.RecoverCommit(group, sharesPriv[i:], t, n)
		assert.Nil(test, err)
		assert.True(test, coinPoint.Equal(recoveredCoinPoint))

		// assert.True(test, group.Point().Mul(hashScalar, pubPoly.Commit()).Equal(recovered2))
	}

	// group.Point().Mul(priPoly.Secret(), nil)
	// group.Point().

}
