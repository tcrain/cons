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
	types "github.com/tcrain/cons/consensus/types"
	"io"
)

type DiskStoreHash struct {
	Diskstore
}

func InitDiskStoreHash(name string, useSnappy bool, bufferSize int) (*DiskStoreHash, error) {
	ret := &DiskStoreHash{}
	hashLen := types.GetHashLen()
	unmarshalFunc := func(data []byte) (interface{}, error) {
		if len(data) < hashLen {
			return nil, io.ErrUnexpectedEOF
		}
		return types.HashStr(data[:hashLen]), nil
	}
	return ret, ret.Init(unmarshalFunc, hashLen, name, useSnappy, bufferSize)
}

func (ds *DiskStoreHash) WriteKey(key interface{}, val []byte) error {
	return ds.Write(key.(types.HashStr), val)
}

func (ds *DiskStoreHash) ReadKey(key interface{}) ([]byte, error) {
	return ds.Read(key.(types.HashStr))
}

func (ds *DiskStoreHash) ContainsKey(key interface{}) bool {
	return ds.Contains(key.(types.HashStr))
}
