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
	"github.com/tcrain/cons/config"
	"io"
)

type uint64Marshaller uint64

func (v uint64Marshaller) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 8)
	config.Encoding.PutUint64(data, uint64(v))
	return
}

type DiskStoreUint64 struct {
	Diskstore
}

func InitDiskStoreUint64(name string, useSnappy bool, bufferSize int) (*DiskStoreUint64, error) {
	ret := &DiskStoreUint64{}
	unmarshalFunc := func(data []byte) (interface{}, error) {
		if len(data) < 8 {
			return nil, io.ErrUnexpectedEOF
		}
		return uint64Marshaller(config.Encoding.Uint64(data)), nil
	}
	return ret, ret.Init(unmarshalFunc, 8, name, useSnappy, bufferSize)
}

func (ds *DiskStoreUint64) WriteKey(key interface{}, val []byte) error {
	return ds.Write(uint64Marshaller(key.(uint64)), val)
}

func (ds *DiskStoreUint64) ReadKey(key interface{}) ([]byte, error) {
	return ds.Read(uint64Marshaller(key.(uint64)))
}

func (ds *DiskStoreUint64) ContainsKey(key interface{}) bool {
	return ds.Contains(uint64Marshaller(key.(uint64)))
}
