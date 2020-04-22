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
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"math/bits"
	mathrand "math/rand"
	"testing"
)

func TestCTRSource(t *testing.T) {
	var key [32]byte
	n, err := rand.Read(key[:])
	assert.Equal(t, len(key), n)
	assert.Nil(t, err)

	var nonce [16]byte
	n = copy(nonce[:], config.CsID)
	assert.Equal(t, 16, n)

	// Create some randoms
	src, err := NewSeededCTRSource(key, nonce)
	assert.Nil(t, err)
	var rndList []uint64
	var oneCount int
	for i := 0; i < 100; i++ {
		rndList = append(rndList, src.Uint64())
		oneCount += bits.OnesCount64(rndList[i])
	}
	// Note this next assert may fail because of randomness
	assert.True(t, oneCount/100 <= 64/2+1, oneCount/100 >= 64/2-1)

	// Create the same randoms again
	src, err = NewSeededCTRSource(key, nonce)
	assert.Nil(t, err)
	var rndList2 []uint64
	for i := 0; i < 100; i++ {
		rndList2 = append(rndList2, src.Uint64())
	}

	// Be sure the values are equal
	assert.Equal(t, rndList, rndList2)

	// Be sure it works as a source
	newRand := mathrand.New(src)
	for i := 0; i < 100; i++ {
		assert.True(t, newRand.Intn(100) < 100)
	}
}
