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
/*
Package storage contains objects for storing decided value and consensus state to disk, or in memory if desired.
*/
package storage

type ConsensusIDType int

const (
	ConsensusIndexKey ConsensusIDType = iota
	HashKey
)

// StoreInterface is the interface used by the consensus to store decidede values and consensus state to disk (or memory).
type StoreInterface interface {
	// Range go through the keys in the order they were written, calling f on them.
	// It stops if f returns false.
	Range(f func(key interface{}, state, decision []byte) bool)
	Read(key interface{}) (state, decision []byte, err error)   // Read returns the values stored for the consensus decision at index key. See Write for the description of what the values represent.
	Contains(key interface{}) bool                              // Contains returns true if key is stored in the map.
	Close() error                                               // Close closes the storage.
	Clear() error                                               // Clear deletes all keys and values from the storage.
	Write(key interface{}, state []byte, decision []byte) error // Write stores for consensus index key the state of the consensus and the decided value. The state should be all the messages needed to reply the consensus and decide the same value. Writing the same key twice overwrites the old value.
	DeleteFile() error                                          // DeleteFile removes the storage file used for this item.
}

type diskStoreInterface interface {
	Range(f func(key interface{}, value []byte) bool)
	ContainsKey(key interface{}) bool
	ReadKey(key interface{}) (value []byte, err error)
	WriteKey(key interface{}, value []byte) error
	Close() error
	Clear() error
}
