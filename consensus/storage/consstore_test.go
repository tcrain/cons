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
	"bytes"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func TestStoreInterface(t *testing.T) {
	var testKey types.ConsensusInt = 123
	var testStateBytes = []byte("testStateBytesString")
	var testDecisionBytes = []byte("testDecisionBytesString")
	memStore := NewConsMemStore(ConsensusIndexKey)
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, memStore, t)

	diskStore, err := NewConsDiskStore("diskstore_test.store", ConsensusIndexKey, 0)
	if err != nil {
		t.Error(err)
	}
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, diskStore, t)

	testKey = 1234
	testStateBytes = []byte("")
	testDecisionBytes = []byte("")
	memStore = NewConsMemStore(ConsensusIndexKey)
	if err != nil {
		t.Error(err)
	}
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, memStore, t)

	diskStore, err = NewConsDiskStore("diskstore_test.store", ConsensusIndexKey, 0)
	if err != nil {
		t.Error(err)
	}
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, diskStore, t)

	testKey = 1235
	testStateBytes = []byte("1")
	testDecisionBytes = []byte("")
	memStore = NewConsMemStore(ConsensusIndexKey)
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, memStore, t)

	diskStore, err = NewConsDiskStore("diskstore_test.store", ConsensusIndexKey, 0)
	if err != nil {
		t.Error(err)
	}
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, diskStore, t)

	testKey = 1236
	testStateBytes = []byte("")
	testDecisionBytes = []byte("1")
	memStore = NewConsMemStore(ConsensusIndexKey)
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, memStore, t)

	diskStore, err = NewConsDiskStore("diskstore_test.store", ConsensusIndexKey, 0)
	if err != nil {
		t.Error(err)
	}
	storeInterfaceTest(testKey, testStateBytes, testDecisionBytes, diskStore, t)
}

func storeInterfaceTest(testKey types.ConsensusInt, testStateBytes []byte, testDecisionBytes []byte, si StoreInterface, t *testing.T) {
	s, d, err := si.Read(testKey)
	if err != nil {
		t.Error(err)
	}
	if s != nil || d != nil {
		t.Error("s and d should be nil:", s, d)
	}

	err = si.Write(testKey, testStateBytes, testDecisionBytes)
	if err != nil {
		t.Error(err)
	}

	s, d, err = si.Read(testKey)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(s, testStateBytes) {
		t.Errorf("Read state %v, should be %v", s, testStateBytes)
	}
	if !bytes.Equal(d, testDecisionBytes) {
		t.Errorf("Read decision %v, should be %v", d, testDecisionBytes)
	}

	err = si.Clear()
	if err != nil {
		t.Error(err)
	}

	s, d, err = si.Read(testKey)
	if err != nil {
		t.Error(err)
	}
	if s != nil || d != nil {
		t.Error("s and d should be nil:", s, d)
	}

	err = si.Close()
	if err != nil {
		t.Error(err)
	}
}
