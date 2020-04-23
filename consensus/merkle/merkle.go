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
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"hash"
)

// TODO add an actual use for this package, i.e. so we can check the inclusion of an item without getting the whole tree.

type MerkleRoot types.HashBytes   // Represents the root node of a merkle tree
type NewHashFunc func() hash.Hash // Function that returns a blank hash object.

// GetHash is a interface that is used for the objects used to compute their merkle root.
type GetHash interface {
	GetSignedHash() types.HashBytes // Should return the hash of the object.
}

// Returns the Merkle root of the items given the tree width and the hash function.
func ComputeMerkleRoot(allItems []GetHash, width int, newFunc NewHashFunc) MerkleRoot {
	return MerkleRoot(recCompute(allItems, 0, 0, width, newFunc))
}

func recCompute(allItems []GetHash, myIndex, depth int, width int, newFunc NewHashFunc) types.HashBytes {
	if utils.Exp(width, depth) >= len(allItems) {
		if myIndex >= len(allItems) {
			return nil
		}
		return allItems[myIndex].GetSignedHash()
	}

	childItems := make([]GetHash, width)
	for i := 0; i < width; i++ {
		childItems[i] = recCompute(allItems, (myIndex*width)+i, depth+1, width, newFunc)
	}
	return computeParent(childItems, newFunc)
}

func computeParent(items []GetHash, newFunc NewHashFunc) types.HashBytes {
	hsh := newFunc()

	for _, nxt := range items {
		hsh.Write(nxt.GetSignedHash())
	}
	return hsh.Sum(nil)
}
