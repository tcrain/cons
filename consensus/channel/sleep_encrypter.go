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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"golang.org/x/crypto/nacl/secretbox"
	"sync"
	"time"
)

const (
	encryptTime = 689
	decryptTime = 1413
)

type SleepEncrypter struct {
	externalPub        sig.Pub
	myPriv             sig.Priv // local nodes private
	isEncrypted        bool     // if the messages are encrypted
	failedEncryptSetup bool     // set to true if encryption setup failed

	shouldWatForPub bool
	mutex           sync.Mutex
	cond            *sync.Cond
}

func NewSleepEcrypter(myPriv sig.Priv,
	shouldWaitForPub bool,
	externalPub sig.Pub) *SleepEncrypter {

	ret := SleepEncrypter{
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
	}

	ret.cond = sync.NewCond(&ret.mutex)
	return &ret
}

func (enc *SleepEncrypter) FailedGetPub() {
	enc.mutex.Lock()
	enc.failedEncryptSetup = true
	enc.cond.Broadcast()
	enc.mutex.Unlock()
}

func (enc *SleepEncrypter) GotPub(pubBytes []byte, randBytes []byte) error {
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

	return nil
}

func (enc *SleepEncrypter) GetRandomBytes() (ret [24]byte) {
	return
}

func (enc *SleepEncrypter) GetMyPubBytes() []byte {
	pubBytes, err := enc.myPriv.GetPub().GetRealPubBytes()
	if err != nil {
		panic(err)
	}
	return pubBytes
}

func (enc *SleepEncrypter) WaitForPub() (err error) {
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

func (enc *SleepEncrypter) Decode(msg []byte, includeSize bool) ([]byte, error) {

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

		out = append(out, nxtMsg[8+secretbox.Overhead:]...)
		time.Sleep(decryptTime)
	}
	return out, nil
}

func (enc *SleepEncrypter) GetExternalPub() sig.Pub {
	return enc.externalPub
}

func (enc *SleepEncrypter) Encode(msg []byte, includeSize bool) []byte {

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

		size += 8
		nxt := make([]byte, 8)
		ret = append(ret, nxt...)
		size += secretbox.Overhead

		ret = append(ret, append(make([]byte, secretbox.Overhead), nxtMsg...)...)
		time.Sleep(encryptTime)
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
