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
	"crypto/sha512"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"golang.org/x/crypto/nacl/secretbox"
	"math"
	"sync"
	"sync/atomic"
)

type Encrypter struct {
	sendSequenceID     uint64
	externalPub        sig.Pub
	myPriv             sig.Priv // local nodes private
	isEncrypted        bool     // if the messages are encrypted
	failedEncryptSetup bool     // set to true if encryption setup failed

	mySharedSecret [32]byte // shared secret

	otherSharedSecret [32]byte // shared secret

	encryptSequenceNumber uint64 // number of messages encrypted
	randomBytes           [24]byte
	nonce                 [24]byte
	shouldWatForPub       bool
	mutex                 sync.Mutex
	cond                  *sync.Cond
}

func NewEcrypter(myPriv sig.Priv,
	shouldWaitForPub bool,
	externalPub sig.Pub) *Encrypter {

	ret := Encrypter{
		shouldWatForPub:    shouldWaitForPub,
		externalPub:        externalPub,
		myPriv:             myPriv,
		failedEncryptSetup: false,
	}

	if shouldWaitForPub {
		if externalPub != nil {
			panic("must not include pub or random bytes if waiting")
		}
	} else {
		if externalPub == nil {
			panic("must include pub and random bytes if not waiting")
		}
		randomBytes := [24]byte{}
		n, err := rand.Read(randomBytes[:])
		if err != nil || n != 24 {
			panic("error reading random bytes")
		}
		ret.randomBytes = randomBytes

		// We take the hash of the secret and the random bytes to get a new key
		ss := ret.myPriv.ComputeSharedSecret(externalPub)
		ret.mySharedSecret = sha512.Sum512_256(append(ss[:], randomBytes[:]...))
		// the others shared secret is the hash of that
		ret.otherSharedSecret = sha512.Sum512_256(ret.mySharedSecret[:])

		// ret.sharedSecret = ret.myPriv.ComputeSharedSecret(ret.externalPub)
	}

	ret.cond = sync.NewCond(&ret.mutex)
	return &ret
}

func (enc *Encrypter) FailedGetPub() {
	enc.mutex.Lock()
	enc.failedEncryptSetup = true
	enc.cond.Broadcast()
	enc.mutex.Unlock()
}

func (enc *Encrypter) GotPub(pubBytes []byte, randBytes []byte) error {
	defer func() {
		enc.cond.Broadcast()
		enc.mutex.Unlock()
	}()
	enc.mutex.Lock()
	var err error
	enc.externalPub, err = enc.myPriv.GetPub().FromPubBytes(pubBytes)
	if err != nil {
		enc.failedEncryptSetup = true
		return err
	}
	if len(randBytes) != 24 {
		err = types.ErrNotEnoughBytes
		enc.failedEncryptSetup = true
		return err
	}
	copy(enc.randomBytes[:], randBytes)

	// We take the hash of the secret and the random bytes to get a new key
	ss := enc.myPriv.ComputeSharedSecret(enc.externalPub)
	enc.otherSharedSecret = sha512.Sum512_256(append(ss[:], randBytes[:]...))
	enc.mySharedSecret = sha512.Sum512_256(enc.otherSharedSecret[:])
	return nil
}

func (enc *Encrypter) GetRandomBytes() [24]byte {
	return enc.randomBytes
}

func (enc *Encrypter) GetMyPubBytes() []byte {
	pubBytes, err := enc.myPriv.GetPub().GetRealPubBytes()
	if err != nil {
		panic(err)
	}
	return pubBytes
}

func (enc *Encrypter) WaitForPub() (err error) {
	enc.mutex.Lock()
	for enc.externalPub == nil && !enc.failedEncryptSetup {
		enc.cond.Wait()
	}
	if enc.failedEncryptSetup {
		err = types.ErrInvalidPub
	}
	enc.mutex.Unlock()
	return
}

func createNonce(id uint64) (nonce *[24]byte) {
	nonce = &[24]byte{}
	config.Encoding.PutUint64(nonce[:], id)
	return
}

func createNonceFromBytes(msg []byte) (nonce *[24]byte, err error) {
	if len(msg) < 8 {
		err = types.ErrNotEnoughBytes
		return
	}
	nonce = &[24]byte{}
	copy(nonce[:], msg[:8])
	return
}

func (enc *Encrypter) Decode(msg []byte, includeSize bool) ([]byte, error) {

	if includeSize {
		if len(msg) < 4 {
			return nil, types.ErrInvalidMsgSize
		}
		if config.Encoding.Uint32(msg) != uint32(len(msg)-4) {
			return nil, types.ErrInvalidMsgSize
		}
		msg = msg[4:]
	}

	if len(msg) == 0 {
		return nil, types.ErrInvalidMsgSize
	}

	var out []byte
	for len(msg) > 0 {

		nxtMsg := msg[:utils.Min(config.MaxEncryptSize+8+secretbox.Overhead, len(msg))]
		msg = msg[utils.Min(config.MaxEncryptSize+8+secretbox.Overhead, len(msg)):]

		nonce, err := createNonceFromBytes(nxtMsg)
		if err != nil {
			return nil, err
		}
		// msg = msg[8:]
		var ok bool
		out, ok = secretbox.Open(out, nxtMsg[8:], nonce, &enc.otherSharedSecret)
		if !ok {
			return nil, types.ErrDecryptionFailed
		}
	}
	return out, nil
}

func (enc *Encrypter) GetExternalPub() sig.Pub {
	return enc.externalPub
}

func (enc *Encrypter) Encode(msg []byte, includeSize bool) []byte {

	if len(msg) == 0 {
		panic("cannot encode nil message")
	}

	size := uint32(len(msg))
	var ret []byte
	if includeSize {
		ret = make([]byte, 4)
	}

	for len(msg) > 0 {
		nxtMsg := msg[:utils.Min(config.MaxEncryptSize, len(msg))]
		msg = msg[utils.Min(config.MaxEncryptSize, len(msg)):]

		id := atomic.AddUint64(&enc.sendSequenceID, 1)
		if id == math.MaxUint64 {
			panic("should not reach")
		}

		nonce := createNonce(id)

		size += 8
		nxt := make([]byte, 8)
		config.Encoding.PutUint64(nxt, id)
		ret = append(ret, nxt...)
		size += secretbox.Overhead

		ret = secretbox.Seal(ret, nxtMsg, nonce, &enc.mySharedSecret)
	}
	if includeSize {
		config.Encoding.PutUint32(ret, size)
		size += 4
	}
	if size != uint32(len(ret)) {
		panic("error encoding")
	}
	return ret
}
