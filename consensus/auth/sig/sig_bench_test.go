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
package sig

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/blake2b"
	"testing"
)

const hashSize = 1000

func genMsg() []byte {
	hashMsg := make([]byte, hashSize)
	n, err := rand.Read(hashMsg)
	if err != nil || n != hashSize {
		panic(err)
	}
	return hashMsg
}

func BenchmarkHashSha256(b *testing.B) {
	hashMsg := genMsg()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := sha256.Sum256(hashMsg)
		_ = hash
	}
}

func BenchmarkHashSha512(b *testing.B) {
	hashMsg := genMsg()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := sha512.Sum512_256(hashMsg)
		_ = hash
	}
}

func BenchmarkHashBlake2(b *testing.B) {
	hashMsg := genMsg()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hf, err := blake2b.New256(nil)
		if err != nil {
			panic(err)
		}
		hf.Write(hashMsg)
		hash := hf.Sum(nil)
		_ = hash
	}
}
