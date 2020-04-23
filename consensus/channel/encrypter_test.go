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

package channel

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"sync"
	"testing"
)

var msg = []byte("This is a message")

func genMsg(size int, t *testing.T) []byte {
	ret := make([]byte, size)
	n, err := rand.Read(ret)
	assert.Nil(t, err)
	assert.Equal(t, size, n)
	return ret
}

func TestEncrypterBasic(t *testing.T) {
	for _, nxtPriv := range []func() (sig.Priv, error){ec.NewEcpriv, ed.NewEdpriv, bls.NewBlspriv} {
		runBasicTest(nxtPriv, t)
	}
}

func runBasicTest(newPriv func() (sig.Priv, error), t *testing.T) {
	enc1, enc2, _ := encryptSetup(newPriv, t)

	var includeSize bool
	for i := 1; i < 20; i++ {
		includeSize = !includeSize
		myMsg := genMsg(1<<uint(i), t)

		cyt := enc1.Encode(myMsg, includeSize)
		assert.NotEqual(t, myMsg, cyt)
		assert.True(t, len(myMsg) < len(cyt))

		// fmt.Println(len(myMsg), len(cyt))

		cyt2 := enc2.Encode(myMsg, includeSize)
		assert.NotEqual(t, myMsg, cyt2)
		assert.True(t, len(myMsg) < len(cyt2))

		// be sure the encryptions are different
		assert.NotEqual(t, cyt, cyt2)

		ct, err := enc2.Decode(cyt, includeSize)
		assert.Nil(t, err)
		assert.Equal(t, myMsg, ct)

		ct2, err := enc1.Decode(cyt2, includeSize)
		assert.Nil(t, err)
		assert.Equal(t, myMsg, ct2)
	}
}

func TestEncrypterFail(t *testing.T) {
	enc1, enc2, _ := encryptSetup(ec.NewEcpriv, t)

	myMsg := msg
	cyt := enc1.Encode(myMsg, true)
	assert.NotEqual(t, myMsg, cyt)
	assert.True(t, len(myMsg) < len(cyt))

	cyt2 := enc2.Encode(myMsg, true)
	assert.NotEqual(t, myMsg, cyt2)
	assert.True(t, len(myMsg) < len(cyt2))

	// be sure the encryptions are different
	assert.NotEqual(t, cyt, cyt2)

	// make them invalid
	cyt[12] += 1
	cyt2[12] += 1

	_, err := enc2.Decode(cyt, true)
	assert.Error(t, err)

	_, err = enc1.Decode(cyt2, true)
	assert.Error(t, err)
}

func TestEncryptBadRand(t *testing.T) {
	enc1, enc2, _ := encryptSetup(ec.NewEcpriv, t)

	// make some new random bytes
	randomBytes := &[24]byte{}
	n, err := rand.Read(randomBytes[:])
	assert.Nil(t, err)
	assert.Equal(t, n, len(randomBytes))

	// change the random bytes
	pubBytes, err := enc1.myPriv.GetPub().GetRealPubBytes()
	assert.Nil(t, err)
	err = enc2.GotPub(pubBytes, randomBytes[:])
	assert.Nil(t, err)

	// the decryption should fail
	cyt := enc1.Encode(msg, true)
	assert.NotEqual(t, msg, cyt)
	assert.True(t, len(msg) < len(cyt))
	_, err = enc2.Decode(cyt, true)
	assert.Error(t, err)

	// also in reverse
	cyt2 := enc2.Encode(msg, true)
	assert.NotEqual(t, msg, cyt2)
	assert.True(t, len(msg) < len(cyt2))
	_, err = enc2.Decode(cyt2, true)
	assert.Error(t, err)
}

func encryptSetup(newPrivFunc func() (sig.Priv, error), t assert.TestingT) (enc1, enc2 *Encrypter,
	randomBytes *[24]byte) {

	priv1, err := newPrivFunc()
	assert.Nil(t, err)

	priv2, err := newPrivFunc()
	assert.Nil(t, err)

	enc1 = NewEcrypter(priv1, false, priv2.GetPub())
	enc2 = NewEcrypter(priv2, true, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := enc2.WaitForPub()
		assert.Nil(t, err)
		wg.Done()
	}()
	pubBytes, err := priv1.GetPub().GetRealPubBytes()
	assert.Nil(t, err)
	rndBytes := enc1.GetRandomBytes()
	err = enc2.GotPub(pubBytes, rndBytes[:])
	assert.Nil(t, err)
	wg.Wait()

	assert.Equal(t, enc1.mySharedSecret, enc2.otherSharedSecret)
	assert.Equal(t, enc1.otherSharedSecret, enc2.mySharedSecret)

	return
}

var encryptMsgSize = sig.EncryptMsgSize

// BenchmarkEncrypt measures the time to encrypt a message
func BenchmarkEncrypt(b *testing.B) {
	encMsg := make([]byte, encryptMsgSize)
	n, err := rand.Read(encMsg)
	assert.Nil(b, err)
	assert.Equal(b, n, encryptMsgSize)

	enc1, _, _ := encryptSetup(ec.NewEcpriv, b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cyt := enc1.Encode(encMsg, true)
		_ = cyt
	}
}

// BenchmarkDecrypt measures the time to decrypt a message.
func BenchmarkDecrypt(b *testing.B) {
	encMsg := make([]byte, encryptMsgSize)
	n, err := rand.Read(encMsg)
	assert.Nil(b, err)
	assert.Equal(b, n, encryptMsgSize)

	enc1, enc2, _ := encryptSetup(ec.NewEcpriv, b)
	cyt := enc1.Encode(encMsg, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		res, err := enc2.Decode(cyt, true)
		assert.Nil(b, err)
		_ = res
	}
}
