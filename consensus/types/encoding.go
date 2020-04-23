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

import (
	"bytes"
	"encoding"
	"io"
)

type BasicEncodeInterface interface {
	Encode(writer io.Writer) (n int, err error)
	Decode(reader io.Reader) (n int, err error)
}

type EncodeInterface interface {
	BasicEncodeInterface
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	New() EncodeInterface
}

func MarshalBinaryHelper(item EncodeInterface) (data []byte, err error) {
	buff := bytes.NewBuffer(nil)
	if _, err := item.Encode(buff); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func UnmarshalBinaryHelper(item EncodeInterface, data []byte) error {
	reader := bytes.NewReader(data)
	n, err := item.Decode(reader)
	if err != nil {
		return err
	}
	if n != len(data) {
		return ErrInvalidMsgSize
	}
	return nil
}
