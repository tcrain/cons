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
	"flag"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
)

var Thrshblsshare *BlsShared
var Thrshblsshare2 *BlsShared

var thrshn = flag.Int("thrshn", config.Thrshn, "benchmark threshold signature n")
var thrsht = flag.Int("thrsht", config.Thrsht, "benchmark threshold signature t")
var thrshn2 = flag.Int("thrshn2", config.Thrshn2, "benchmark threshold signature n")
var thrsht2 = flag.Int("thrsht2", config.Thrsht2, "benchmark threshold signature t")

func init() {
	Thrshblsshare = NewBlsShared(*thrshn, *thrsht)
	Thrshblsshare2 = NewBlsShared(*thrshn2, *thrsht2)
}

func GetBlsPartPrivFunc() func() (sig.Priv, error) {
	index := sig.TestKeyIndex
	return func() (sig.Priv, error) {
		blsThresh := NewBlsThrsh(*thrshn, *thrsht, index, Thrshblsshare.PriScalars[index],
			Thrshblsshare.PubPoints[index], Thrshblsshare.SharedPub)
		index += 1
		return NewBlsPartPriv(blsThresh)
	}
}

func GetBlsPartPrivFunc2() func() (sig.Priv, error) {
	index := sig.TestKeyIndex
	return func() (sig.Priv, error) {
		blsThresh := NewBlsThrsh(*thrshn2, *thrsht2, index, Thrshblsshare2.PriScalars[index],
			Thrshblsshare2.PubPoints[index], Thrshblsshare2.SharedPub)
		index += 1
		return NewBlsPartPriv(blsThresh)
	}
}
