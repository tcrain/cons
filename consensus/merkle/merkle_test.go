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

package merkle

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/types"
	"hash"
	"testing"
)

func computeHsh(hsh hash.Hash, v []byte) []byte {
	hsh.Write(v)
	return hsh.Sum(nil)
}

func TestComputeMerkleRoot(t *testing.T) {

	newHashFunc := types.GetNewHash
	numItems := 4
	width := 2

	l := computeHsh(newHashFunc(), append(computeHsh(newHashFunc(), []byte(fmt.Sprint(0))),
		computeHsh(newHashFunc(), []byte(fmt.Sprint(1)))...))
	r := computeHsh(newHashFunc(), append(computeHsh(newHashFunc(), []byte(fmt.Sprint(2))),
		computeHsh(newHashFunc(), []byte(fmt.Sprint(3)))...))
	root := MerkleRoot(computeHsh(newHashFunc(), append(l, r...)))

	items := make([]GetHash, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = types.HashBytes(computeHsh(newHashFunc(), []byte(fmt.Sprint(i))))
	}

	assert.Equal(t, root, ComputeMerkleRoot(items, width, newHashFunc))
}
