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
package types

import "fmt"

// StorageType is the type of storage to use, either on disk or in memory.
type StorageType int

const (
	Diskstorage StorageType = iota // Store decided values to disk.
	Memstorage                     // Store decided values in memory.
)

// String returns the human readable string of the storage type.
func (st StorageType) String() string {
	switch st {
	case Diskstorage:
		return "Diskstorage"
	case Memstorage:
		return "Memstorage"
	default:
		return fmt.Sprintf("StorageType%d", st)
	}
}
