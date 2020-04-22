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
	"fmt"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type newTestInterface func(bool, int) (storeTestInterface, error)

type storeTestInterface interface {
	getDiskStore() *Diskstore
	testWrite(key int, val []byte) error
	testRead(key int) ([]byte, error)
	getKey(int) interface{}
}

///////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////

type uint64TestStore DiskStoreUint64

func newUint64TestStore(useSnappy bool, bufferSize int) (storeTestInterface, error) {
	ret, err := InitDiskStoreUint64("diskstore_test.store", useSnappy, bufferSize)
	return (*uint64TestStore)(ret), err
}

func (ds *uint64TestStore) getDiskStore() *Diskstore {
	return &ds.Diskstore
}

func (ds *uint64TestStore) getKey(key int) interface{} {
	return uint64Marshaller(key)
}

func (ds *uint64TestStore) testWrite(key int, val []byte) error {
	return (*DiskStoreUint64)(ds).WriteKey(uint64(key), val)
}

func (ds *uint64TestStore) testRead(key int) ([]byte, error) {
	return (*DiskStoreUint64)(ds).ReadKey(uint64(key))
}

func TestStoreUint64(t *testing.T) {
	runTestStore(newUint64TestStore, t)
}

///////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////

type hashTestStore DiskStoreHash

func newHashTestStore(useSnappy bool, bufferSize int) (storeTestInterface, error) {
	ret, err := InitDiskStoreHash("diskstorehash_test.store", useSnappy, bufferSize)
	return (*hashTestStore)(ret), err
}

func (ds *hashTestStore) getDiskStore() *Diskstore {
	return &ds.Diskstore
}

func (ds *hashTestStore) getKey(key int) interface{} {
	return types.HashStr(types.GetHash([]byte(fmt.Sprint(key))))
}

func (ds *hashTestStore) testWrite(key int, val []byte) error {
	return (*DiskStoreHash)(ds).WriteKey(ds.getKey(key).(types.HashStr), val)
}

func (ds *hashTestStore) testRead(key int) ([]byte, error) {
	return (*DiskStoreHash)(ds).ReadKey(ds.getKey(key).(types.HashStr))
}

func TestStoreHash(t *testing.T) {
	runTestStore(newHashTestStore, t)
}

///////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////

func runTestStore(allocFunc newTestInterface, t *testing.T) {
	openType = os.O_RDWR | os.O_CREATE

	ds, err := allocFunc(false, 0)
	assert.Nil(t, err)
	err = ds.getDiskStore().Clear()
	assert.Nil(t, err)

	// Write some keys
	aval := []byte("test")
	err = ds.testWrite(0, aval)
	assert.Nil(t, err)
	err = ds.testWrite(1, aval)
	assert.Nil(t, err)

	// Read them
	val, err := ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)
	val, err = ds.testRead(1)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	// Close the store
	err = ds.getDiskStore().Close()
	assert.Nil(t, err)

	// Open the store
	ds, err = allocFunc(false, 0)
	assert.Nil(t, err)

	// Read the keys
	val, err = ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	val, err = ds.testRead(1)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	// Corrupt a key
	err = ds.getDiskStore().corruptValue(ds.getKey(0), 'a')
	assert.Nil(t, err)
	val, err = ds.testRead(0)
	assert.NotNil(t, err)

	// Corrupt another key
	err = ds.getDiskStore().corruptHash(ds.getKey(1), 'a')
	assert.Nil(t, err)
	val, err = ds.testRead(1)
	assert.NotNil(t, err)

	// Fix 0 key
	err = ds.testWrite(0, aval)
	assert.Nil(t, err)
	val, err = ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	// Close the store
	err = ds.getDiskStore().Close()
	assert.Nil(t, err)

	// Open the store
	ds, err = allocFunc(false, 0)
	assert.Nil(t, err)

	// Read the keys
	val, err = ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	val, err = ds.testRead(1)
	assert.Nil(t, val)
	assert.Nil(t, err)
}

func runTestStoreBuffer(allocFunc newTestInterface, t *testing.T) {
	openType = os.O_APPEND | os.O_RDWR | os.O_CREATE
	bufferSize := 1000

	ds, err := allocFunc(false, bufferSize)
	assert.Nil(t, err)
	err = ds.getDiskStore().Clear()
	assert.Nil(t, err)

	// Write some keys
	aval := []byte("test")
	err = ds.testWrite(0, aval)
	assert.Nil(t, err)
	err = ds.testWrite(1, aval)
	assert.Nil(t, err)

	// Read them
	val, err := ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)
	val, err = ds.testRead(1)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)

	// Close the store
	err = ds.getDiskStore().Close()
	assert.Nil(t, err)

	// Open the store
	ds, err = allocFunc(false, bufferSize)
	assert.Nil(t, err)

	// Read the keys
	val, err = ds.testRead(0)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)
	val, err = ds.testRead(1)
	assert.Nil(t, err)
	assert.Equal(t, val, aval)
}

func BenchmarkStoragetestWrite(b *testing.B) {
	openType = os.O_APPEND | os.O_RDWR | os.O_CREATE
	ds, err := InitDiskStoreUint64("diskstore_test.store", false, 0)
	assert.Nil(b, err)
	err = ds.Clear()
	assert.Nil(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = ds.WriteKey(uint64(i), []byte("blah"))
		assert.Nil(b, err)
	}
	err = ds.Close()
	assert.Nil(b, err)
}

func BenchmarkStorageBuffertestWrite(b *testing.B) {
	openType = os.O_APPEND | os.O_RDWR | os.O_CREATE
	ds, err := InitDiskStoreUint64("diskstore_test.store", false, 10000)
	assert.Nil(b, err)
	err = ds.Clear()
	assert.Nil(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = ds.WriteKey(uint64(i), []byte("blah"))
		assert.Nil(b, err)
	}
	err = ds.Close()
	assert.Nil(b, err)
}

var readPercentage = 0.1

func BenchmarkStorageReadtestWrite(b *testing.B) {
	openType = os.O_APPEND | os.O_RDWR | os.O_CREATE
	ds, err := InitDiskStoreUint64("diskstore_test.store", false, 0)
	assert.Nil(b, err)
	err = ds.Clear()
	assert.Nil(b, err)
	err = ds.WriteKey(uint64(0), []byte("blah"))
	assert.Nil(b, err)

	random := rand.New(rand.NewSource(0))
	writeCount := 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if random.Float64() > readPercentage {
			err = ds.WriteKey(uint64(writeCount), []byte("blah"))
			writeCount++
			assert.Nil(b, err)
		} else {
			key := uint64(random.Intn(writeCount))
			val, err := ds.ReadKey(key)
			assert.Nil(b, err)
			assert.Equal(b, val, []byte("blah"))
		}
	}
	err = ds.Close()
	assert.Nil(b, err)
}

func BenchmarkStorageBufferedReadtestWrite(b *testing.B) {
	openType = os.O_APPEND | os.O_RDWR | os.O_CREATE
	ds, err := InitDiskStoreUint64("diskstore_test.store", false, 1000)
	assert.Nil(b, err)
	err = ds.Clear()
	assert.Nil(b, err)
	err = ds.WriteKey(uint64(0), []byte("blah"))
	assert.Nil(b, err)

	random := rand.New(rand.NewSource(0))
	writeCount := 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if random.Float64() > readPercentage {
			err = ds.WriteKey(uint64(writeCount), []byte("blah"))
			writeCount++
			assert.Nil(b, err)
		} else {
			val, err := ds.ReadKey(uint64(random.Intn(writeCount)))
			assert.Nil(b, err)
			assert.Equal(b, val, []byte("blah"))
		}
	}
	err = ds.Close()
	assert.Nil(b, err)
}
