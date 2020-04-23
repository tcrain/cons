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

package ec

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sigMsg = []byte("sign this message")

// BenchmarkVerifyEC tests the cost to verify a EC signature
func BenchmarkSignEC(b *testing.B) {
	k, err := ecdsa.GenerateKey(ecCurve, rand.Reader)
	assert.Nil(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := types.GetHash(sigMsg)
		r, s, err := ecdsa.Sign(rand.Reader, k, hash)
		assert.Nil(b, err)
		_, err = asn1.Marshal(Ecsig{r, s})
		assert.Nil(b, err)
	}
}

// BenchmarkVerifyEC tests the cost to verify a EC signature
func BenchmarkVerifyEC(b *testing.B) {
	k, err := ecdsa.GenerateKey(ecCurve, rand.Reader)
	assert.Nil(b, err)
	hash := types.GetHash(sigMsg)
	r, s, err := ecdsa.Sign(rand.Reader, k, hash)
	assert.Nil(b, err)

	buff, err := asn1.Marshal(Ecsig{r, s})
	assert.Nil(b, err)
	b.Log("EC sig length", len(buff))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmp := Ecsig{}
		_, err = asn1.Unmarshal(buff, &tmp)
		assert.Nil(b, err)
		v := ecdsa.Verify(&k.PublicKey, hash, tmp.R, tmp.S)
		assert.True(b, v)
	}
}

// BenchmarkVerifyDSA tests the cost to verify a L1024N160 DSA signature
func BenchmarkVerifyDSA(b *testing.B) {
	params := &dsa.Parameters{}
	err := dsa.GenerateParameters(params, rand.Reader, dsa.L1024N160)
	assert.Nil(b, err)
	priv := &dsa.PrivateKey{
		PublicKey: dsa.PublicKey{
			Parameters: *params}}
	err = dsa.GenerateKey(priv, rand.Reader)
	assert.Nil(b, err)
	hash := types.GetHash(sigMsg)
	r, s, err := dsa.Sign(rand.Reader, priv, hash)
	assert.Nil(b, err)

	buff, err := asn1.Marshal(Ecsig{r, s})
	assert.Nil(b, err)
	b.Log("DSA sig length", len(buff))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := dsa.Verify(&priv.PublicKey, hash, r, s)
		assert.True(b, v)
	}
}

// BenchmarkVerifyRSA tests the cost to verify a 3072 bit RSA signature
func BenchmarkVerifyRSA(b *testing.B) {
	priv, err := rsa.GenerateKey(rand.Reader, 3072)
	assert.Nil(b, err)

	hash := sha256.Sum256(sigMsg)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
	assert.Nil(b, err)

	b.Log("RSA sig length", len(sig))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, hash[:], sig)
		assert.Nil(b, err)
	}

}
