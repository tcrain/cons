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
package storage

import (
	"encoding/binary"
	"fmt"
	"github.com/tcrain/cons/consensus/types"
	"os"
)

///////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////

// ConsDiskstore is the interface to store decided values on disk.
type ConsDiskstore struct {
	ds       diskStoreInterface // The actual storage.
	fileName string
	keyType  ConsensusIDType
}

// NewConsDiskStore creates a new disk storage, at the file name in the current directory, for the
// give key type and buffer size in bytes.
func NewConsDiskStore(name string, keyType ConsensusIDType, bufferSize int) (*ConsDiskstore, error) {
	cs := &ConsDiskstore{}
	cs.fileName = name
	var err error
	switch keyType {
	case ConsensusIndexKey:
		cs.ds, err = InitDiskStoreUint64(name, false, bufferSize)
		cs.keyType = keyType
	case HashKey:
		cs.ds, err = InitDiskStoreHash(name, false, bufferSize)
		cs.keyType = keyType
	default:
		panic(fmt.Sprintf("invalid key type: %v", keyType))
	}
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// DeleteFile removes the storage file used for this item.
func (cs *ConsDiskstore) DeleteFile() error {
	return os.Remove(cs.fileName)
}

func (cs *ConsDiskstore) Range(f func(key interface{}, state, decision []byte) bool) {
	cs.ds.Range(func(key interface{}, value []byte) bool {
		stateLen := binary.BigEndian.Uint64(value)
		state := value[8 : stateLen+8]
		decision := value[8+stateLen:]
		return f(key, state, decision)
	})
}

// Contains returns true is the key is contained in the map.
func (cs *ConsDiskstore) Contains(key interface{}) bool {
	return cs.ds.ContainsKey(cs.checkKey(key))
}

// checkKey converts the key to the valid type or panics
func (cs *ConsDiskstore) checkKey(key interface{}) interface{} {
	var readKey interface{}
	switch cs.keyType {
	case ConsensusIndexKey:
		readKey = uint64(key.(types.ConsensusInt))
	case HashKey:
		readKey = types.HashStr(key.(types.ConsensusHash))
	default:
		panic(cs.keyType)
	}
	return readKey
}

// Read returns the values stored for the consensus decision at index key.
// See Write for the description of what the values represent.
func (cs *ConsDiskstore) Read(key interface{}) ([]byte, []byte, error) {
	readKey := cs.checkKey(key)
	buff, err := cs.ds.ReadKey(readKey)
	if err != nil || buff == nil {
		return nil, nil, err
	}
	stateLen := binary.BigEndian.Uint64(buff)
	state := buff[8 : stateLen+8]
	decision := buff[8+stateLen:]
	return state, decision, nil
}

// Close closes the file.
func (cs *ConsDiskstore) Close() error {
	return cs.ds.Close()
}

// Clear deletes all stored keys and values.
func (cs *ConsDiskstore) Clear() error {
	return cs.ds.Clear()
}

// Write stores for consensus index key the state of the consensus and the decided value.
// The state should be all the messages needed to reply the consensus and decide the same value.
// Writing the same key twice overwrites the old value.
func (cs *ConsDiskstore) Write(key interface{}, state []byte, decision []byte) error {
	buff := make([]byte, 8+len(state)+len(decision))
	binary.BigEndian.PutUint64(buff, uint64(len(state)))
	l := 8
	l += copy(buff[l:], state)
	l += copy(buff[l:], decision)

	if l != 8+len(state)+len(decision) {
		panic("invalid copy")
	}
	return cs.ds.WriteKey(cs.checkKey(key), buff)
}

///////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////

// memstoreItem is the internal represnetation of a consensus instance store
type memstoreItem struct {
	state    []byte
	decision []byte
}

// ConsMemstore represnets consensus storage in memory.
type ConsMemstore struct {
	keyType      ConsensusIDType
	items        map[interface{}]memstoreItem
	orderedItems []interface{}
}

// NewConsMemStore creates a new
func NewConsMemStore(keyType ConsensusIDType) *ConsMemstore {
	cs := &ConsMemstore{}
	cs.keyType = keyType
	cs.items = make(map[interface{}]memstoreItem)
	return cs
}

// Range go through the keys in the order they were written, calling f on them.
// It stops if f returns false.
func (cs *ConsMemstore) Range(f func(key interface{}, state, decision []byte) bool) {
	for _, k := range cs.orderedItems {
		s, d, err := cs.Read(k)
		if err != nil {
			panic(err)
		}
		if !f(k, s, d) {
			return
		}
	}
}

// DeleteFile returns nil.
func (cs *ConsMemstore) DeleteFile() error {
	return nil
}

// Contains returns true if key is stored in the map.
func (cs *ConsMemstore) Contains(key interface{}) bool {
	v1, v2, err := cs.Read(key)
	if err != nil {
		panic(err)
	}
	return len(v1) > 0 || len(v2) > 0
}

// Read returns the values stored for the consensus decision at index key.
// See Write for the description of what the values represent.
func (cs *ConsMemstore) Read(key interface{}) ([]byte, []byte, error) {
	switch cs.keyType { // sanity check
	case ConsensusIndexKey:
		_ = uint64(key.(types.ConsensusInt))
	case HashKey:
		_ = key.(types.ConsensusHash)
	default:
		panic(cs.keyType)
	}
	if v, ok := cs.items[key]; ok {
		return v.state, v.decision, nil
	}
	return nil, nil, nil
}

// Close does nothing for the memstore.
func (cs *ConsMemstore) Close() error {
	return nil
}

// Clear deletes all stored keys and values.
func (cs *ConsMemstore) Clear() error {
	cs.items = make(map[interface{}]memstoreItem)
	return nil
}

// Write stores for consensus index key the state of the consensus and the decided value.
// The state should be all the messages needed to reply the consensus and decide the same value.
// Writing the same key twice overwrites the old value.
func (cs *ConsMemstore) Write(key interface{}, state []byte, decision []byte) error {
	switch cs.keyType { // sanity check
	case ConsensusIndexKey:
		_ = uint64(key.(types.ConsensusInt))
	case HashKey:
		_ = key.(types.ConsensusHash)
	default:
		panic(cs.keyType)
	}
	if _, ok := cs.items[key]; !ok {
		cs.orderedItems = append(cs.orderedItems, key)
	}
	cs.items[key] = memstoreItem{state, decision}
	return nil
}
