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
package config

import "golang.org/x/crypto/nacl/secretbox"

// CsID is a unique value that should be changed each time the consensus is run.
// This is included in each message (see doc.go) so that the verifier of the signature
// knows that the message was signed to be used uniquely for this run of consensus.
// TODO cleaner way to do this, should this be bigger?
// Put these bytes after the size and header id
// Change every run so we cant reuse signed messages
// how many bytes should this be?
const CsID = "01234567" //"89abcdefghijklmnopqrstuv"

const RandID = "0123456789abcdefghijklmnopqrstuv"

var InitRandBytes [32]byte

func init() {
	if len(RandID) != 32 {
		panic(len(RandID))
	}
	if copy(InitRandBytes[:], RandID) != 32 {
		panic("invalid copy")
	}
}

const (
	CoinSeed      = 98766
	LocalCoinSeed = 9821
)

const EncryptOverhead = secretbox.Overhead + 8
const MaxEncryptSize = 1600
