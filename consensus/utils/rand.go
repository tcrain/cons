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
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/tcrain/cons/config"
)

// SeededCTRSource is intended to be used as a source for the golang math.Rand objects.
// It uses an key and a nonce as the seed for the randoms using golang's AES CTR stream.
type SeededCTRSource struct {
	stream cipher.Stream
}

// NewSeededCTRSource create a need random source using key and nonce as the seed.
func NewSeededCTRSource(key [32]byte, nonce [16]byte) (*SeededCTRSource, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	ret := &SeededCTRSource{}
	ret.stream = cipher.NewCTR(block, nonce[:])
	return ret, nil
}

// Uint64 returns the next pseudo-random uint64.
func (src *SeededCTRSource) Uint64() uint64 {
	byt := make([]byte, 8)
	src.stream.XORKeyStream(byt, byt)
	return config.Encoding.Uint64(byt)
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (src *SeededCTRSource) Int63() int64 {
	ret := int64(src.Uint64())
	if ret < 0 {
		return -ret
	}
	return ret
}

// Seed is not supported, the source instead should be created with NewSeededCTRSource.
func (src *SeededCTRSource) Seed(seed int64) {
	panic("unused")
}
