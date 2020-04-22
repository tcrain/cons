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
	//"github.com/golang/snappy"
	"bufio"
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"github.com/tcrain/cons/consensus/types"
	"hash/adler32"
	"io"
	"math"
	"os"

	"github.com/tcrain/cons/consensus/logging"
)

var openType = os.O_APPEND | os.O_RDWR | os.O_CREATE

// Diskstore represents a simple key/value disk storage.
// It works using an append only file. Writing a key twice will
// append the new value to the end of the file, the old value will
// still exist, but be unaccessble. The keys and a pointer to where the value
// is stored on disk is kept in memory. Values are stored with their hashes
// and the hash is verified when the value is read.
// Writes can be buffered in memory if a bufferSize > 0 is used as input to
// the Init function.
// It is meant to be efficient for storing consensus decisions for example,
// where we store something once and keep it forever.
// TODO fix the error types.
type Diskstore struct {
	name          string                   // the file name
	useSnappy     bool                     // if snappy compression should be used
	keyMap        map[interface{}][2]int64 // map from key to location on disk and size of value stored
	orderedKeys   []interface{}            // list of keys in the order they were written
	file          *os.File                 // the actual file
	pos           int64                    // the current position in the file
	bufferSize    int                      // the size of the in memory buffer for writing
	writer        *bufio.Writer            // if bufferSize > 0 then this is used to buffer writes
	keySize       int                      // size of the key in bytes
	unmarshalFunc func([]byte) (interface{}, error)
}

// Initialized the diskstore for file name, if to use snappy compression, and the size of the write buffer
// (use 0 for unbuffered). If the file exists, it checks the stored values are valid and loads
// meta-data into memory.
func (ds *Diskstore) Init(unmarshalFunc func([]byte) (interface{}, error), keySize int, name string, useSnappy bool, bufferSize int) error {
	ds.name = name
	ds.useSnappy = useSnappy
	ds.pos = -1
	ds.keyMap = make(map[interface{}][2]int64)
	ds.bufferSize = bufferSize
	ds.keySize = keySize
	ds.unmarshalFunc = unmarshalFunc
	return ds.readFile(openType)
}

// InitClear is the same as Init, expect it deletes any stored keys and values.
func (ds *Diskstore) InitClear(unmarshalFunc func([]byte) (interface{}, error), keySize int, name string, useSnappy bool, bufferSize int) error {
	ds.name = name
	ds.useSnappy = useSnappy
	ds.pos = -1
	ds.keyMap = make(map[interface{}][2]int64)
	ds.orderedKeys = nil
	ds.bufferSize = bufferSize
	ds.keySize = keySize
	ds.unmarshalFunc = unmarshalFunc
	return ds.Clear()
}

// Contains returns true is the key is contained in the map.
// It returns false if a 0 length value has been stored for the key.
func (ds *Diskstore) Contains(key interface{}) bool {
	if val, ok := ds.keyMap[key]; ok {
		return val[1] > 0
	}
	return false
}

// Range go through the keys in the order they were written, calling f on them.
// It stops if f returns false.
func (ds *Diskstore) Range(f func(key interface{}, value []byte) bool) {
	for _, k := range ds.orderedKeys {
		v, err := ds.Read(k)
		if err != nil {
			panic(err)
		}
		if len(v) == 0 {
			continue
		}
		if !f(k, v) {
			return
		}
	}
}

// Read returns the value stored for key.
// Returns an error if the key was not found.
func (ds *Diskstore) Read(key interface{}) ([]byte, error) {
	if val, ok := ds.keyMap[key]; ok {
		if ds.writer != nil && ds.writer.Buffered() > 0 {
			err := ds.writer.Flush()
			if err != nil {
				logging.Error(err)
			}
		}
		ds.pos = -1

		pos, err := ds.file.Seek(val[0], io.SeekStart)
		if pos != val[0] || err != nil {
			return nil, fmt.Errorf("Could not seek")
		}
		// Get the hash
		nextHash := make([]byte, 4)
		n, err := ds.file.Read(nextHash)
		if err != nil || n != 4 {
			return nil, fmt.Errorf("Error reading value hash")
		}

		// Read the value
		ret := make([]byte, val[1])
		n, err = ds.file.Read(ret)
		if err != nil || int64(n) != val[1] {
			return nil, fmt.Errorf("Error reading")
		}

		// Check the hash
		hasher := adler32.New()
		_, err = hasher.Write(ret)
		if err != nil {
			panic(err)
		}
		if bytes.Compare(nextHash, hasher.Sum(nil)) != 0 {
			return nil, fmt.Errorf("Invalid hash")
		}

		return ret, nil
	}
	return nil, nil
}

// Close flushes to the disk, sycs the file, and closes it.
func (ds *Diskstore) Close() error {
	var err error
	if ds.file != nil {
		if ds.writer != nil && ds.writer.Buffered() > 0 {
			err = ds.writer.Flush()
			if err != nil {
				logging.Error(err)
			}
		}
		err = ds.file.Sync()
		if err != nil {
			logging.Error(err)
		}
		err = ds.file.Close()
	}
	ds.file = nil
	ds.keyMap = nil
	return err
}

// Clear deletes all keys and values stored.
func (ds *Diskstore) Clear() error {
	if ds.file != nil {
		err := ds.file.Close()
		if err != nil {
			logging.Error(err)
		}
	}
	ds.writer = nil
	err := os.Remove(ds.name)
	if err != nil {
		return err
	}
	ds.keyMap = make(map[interface{}][2]int64)
	ds.orderedKeys = nil
	return ds.readFile(openType)
}

func (ds *Diskstore) internalWrite(val []byte) (int, error) {
	if ds.writer != nil {
		return ds.writer.Write(val)
	}
	return ds.file.Write(val)
}

// Write stores a key and associated value to disk.
// If the key exists, the value is overwritten.
func (ds *Diskstore) Write(key encoding.BinaryMarshaler, val []byte) error {
	if ds.pos == -1 {
		pos, err := ds.file.Seek(0, io.SeekEnd)
		if err != nil {
			err2 := ds.file.Close()
			if err2 != nil {
				logging.Error(err2)
			}
			return err
		}
		ds.pos = pos
	}
	sizeBytes := make([]byte, 8)

	keyBytes, err := key.MarshalBinary()
	if err != nil {
		return err
	}
	if len(keyBytes) != ds.keySize {
		return types.ErrInvalidKeySize
	}
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(val)))

	hasher := adler32.New()
	_, err = hasher.Write(val)
	if err != nil {
		panic(err)
	}
	hashBytes := hasher.Sum(nil)
	if len(hashBytes) != 4 {
		panic("Wrong length hash")
	}

	// Write key
	n, err := ds.internalWrite(keyBytes)
	if err != nil {
		logging.Error("Unable to write key", err)
		err2 := ds.file.Close()
		if err2 != nil {
			logging.Error(err2)
		}
		return err
	}
	ds.pos += int64(n)

	// Write size
	n, err = ds.internalWrite(sizeBytes)
	if err != nil {
		logging.Error("Unable to write key size", err)
		err2 := ds.file.Close()
		if err2 != nil {
			logging.Error(err2)
		}
		return err
	}
	ds.pos += int64(n)
	valuePosition := ds.pos

	// Write hash
	n, err = ds.internalWrite(hashBytes)
	if err != nil {
		logging.Error("Unable to write key hash", err)
		err2 := ds.file.Close()
		if err2 != nil {
			logging.Error(err2)
		}
		return err
	}
	ds.pos += int64(n)

	// Write value
	n, err = ds.internalWrite(val)
	if err != nil {
		logging.Error("Unable to write value", err)
		err2 := ds.file.Close()
		if err2 != nil {
			logging.Error(err2)
		}
		return err
	}

	if _, ok := ds.keyMap[key]; !ok {
		ds.orderedKeys = append(ds.orderedKeys, key)
	}

	ds.keyMap[key] = [2]int64{valuePosition, int64(len(val))}
	ds.pos += int64(n)

	return nil
}

// corruptIndex corrupts the data stored at index in the file by writing val there.
// It is just for testing.
func (ds *Diskstore) corruptIndex(index int64, val byte) error {
	ds.pos = -1
	pos, err := ds.file.Seek(index, io.SeekStart)
	if err != nil || pos != index {
		return err
	}
	_, err = ds.internalWrite([]byte{val})
	return err
}

// corruptValue corrupts the value stored for key by writing corrupt at the first byte of the value.
func (ds *Diskstore) corruptValue(key interface{}, corrupt byte) error {
	if val, ok := ds.keyMap[key]; ok {
		return ds.corruptIndex(val[0]+5, corrupt)
	}
	return fmt.Errorf("Key not found")
}

// corruptHash corrupts the hash stored for key by writing corrupt as the first byte of the hash.
func (ds *Diskstore) corruptHash(key interface{}, corrupt byte) error {
	if val, ok := ds.keyMap[key]; ok {
		return ds.corruptIndex(val[0], corrupt)
	}
	return fmt.Errorf("Key not found")
}

// readFile opens the file, checks the format, and loads all the meta-data into memory.
func (ds *Diskstore) readFile(openType int) error {
	file, err := os.OpenFile(ds.name, openType, 0666)
	if err != nil {
		logging.Error("Error opening store: ", err)
		return err
	}
	stat, err := file.Stat()
	logging.Info(fmt.Sprintf("Fileinfo: %+v, %+v\n", stat, err))
	ds.file = file
	pos, err := file.Seek(0, io.SeekStart)
	if err != nil {
		err2 := file.Close()
		if err2 != nil {
			logging.Error(err2)
		}
		return err
	}

	if ds.bufferSize > 0 {
		ds.writer = bufio.NewWriter(file)
	}

	for true {
		nextPos := pos

		// Get the key
		nextInt := make([]byte, ds.keySize)
		n, err := file.Read(nextInt)
		if err != nil || n != ds.keySize {
			if n != 0 {
				logging.Error("Error reading key, got length: ", n)
			}
			break
		}

		key, err := ds.unmarshalFunc(nextInt)
		if err != nil {
			if err != nil {
				logging.Error(err)
			}
			break
		}

		nextPos += int64(n)
		// Get the size
		nextInt = make([]byte, 8)
		n, err = file.Read(nextInt)
		if err != nil || n != 8 {
			logging.Error("Error reading value size, got length: ", n)
			break
		}
		nextPos += int64(n)
		valuePosition := nextPos
		size := int64(binary.BigEndian.Uint64(nextInt))
		if size > math.MaxInt32 || size < 0 {
			logging.Error("invalid record size", size)
			break
		}

		// Get the hash
		nextHash := make([]byte, 4)
		n, err = file.Read(nextHash)
		if err != nil || n != 4 {
			logging.Error("Error reading value hash, got length: ", n)
			break
		}
		nextPos += int64(n)

		// Be sure you can get the whole value
		newPos, err := file.Seek(size, io.SeekCurrent)
		if newPos != nextPos+size {
			logging.Error("Error reading value, got position, expected: ", newPos, pos+size)
			break
		}
		newPos, err = file.Seek(-size, io.SeekCurrent)
		if err != nil || newPos != nextPos {
			panic(err)
		}

		value := make([]byte, size)
		n, err = file.Read(value)
		if err != nil || int64(n) != size {
			panic(err)
		}
		nextPos += int64(n)

		hasher := adler32.New()
		_, err = hasher.Write(value)
		if err != nil {
			panic(err)
		}
		if bytes.Compare(nextHash, hasher.Sum(nil)) != 0 {
			logging.Error("Invalid hash for key: ", key)
		} else {
			ds.keyMap[key] = [2]int64{valuePosition, size}
			ds.orderedKeys = append(ds.orderedKeys, key)
		}
		pos = nextPos
	}

	ds.pos = pos
	if pos > 0 {
		// TODO should truncate?
		//err = file.Truncate(pos)
		//if err != nil {
		//panic(err)
		//file.Close()
		//return err
		//}
	}
	return nil
}

//binary.BigEndian.PutUint32(idx, uint32(node.Properties.Index))
