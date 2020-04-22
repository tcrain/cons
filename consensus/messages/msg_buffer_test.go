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
package messages

import (
	"github.com/tcrain/cons/consensus/types"
	"io"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUint32(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 uint32 = math.MaxUint32
	var v2 uint32

	mb.AddUint32(v1)
	mb.AddUint32(v2)
	mb.AddUint32(v1)

	c1, _, err := mb.ReadUint32()
	if err != nil {
		t.Error(err)
	}
	if c1 != v1 {
		t.Error("Got not equal")
	}

	c2, _, err := mb.ReadUint32()
	if err != nil {
		t.Error(err)
	}
	if c2 != v2 {
		t.Error("Got not equal")
	}

	_, _, err = mb.ReadUint64()
	if err == nil {
		t.Error("Should have out of bounds error")
	}
}

func TestUvarint(t *testing.T) {
	mb := NewMsgBuffer()

	mb.AddUvarint(0)
	for i := uint64(1); i <= 64; i = i * 2 {
		mb.AddUvarint((1 << i) - 1)
	}

	v, _, err := mb.ReadUvarint()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), v)
	for i := uint64(1); i <= 64; i = i * 2 {
		v, _, err := mb.ReadUvarint()
		assert.Nil(t, err)
		assert.Equal(t, uint64((1<<i)-1), v)
	}

	_, _, err = mb.ReadUvarint()
	assert.Equal(t, types.ErrNotEnoughBytes, err)
}

func TestBin(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 types.BinVal = 1
	var v2 types.BinVal

	mb.AddBin(v1, false)
	mb.AddBin(v2, false)
	mb.AddBin(v1, false)

	c1, err := mb.ReadBin(false)
	if err != nil {
		t.Error(err)
	}
	if c1 != v1 {
		t.Error("Got not equal")
	}

	c2, err := mb.ReadBin(false)
	if err != nil {
		t.Error(err)
	}
	if c2 != v2 {
		t.Error("Got not equal")
	}

	_, _, err = mb.ReadUint64()
	if err == nil {
		t.Error("Should have out of bounds error")
	}

	mb = NewMsgBuffer()
	v1 = 3

	mb.AddByte(byte(v1))

	c1, err = mb.ReadBin(false)
	assert.Nil(t, err)
	assert.Equal(t, c1, types.BinVal(1))
}

func TestBinAllowCoin(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 types.BinVal = 1
	var v2 types.BinVal

	mb.AddBin(v1, true)
	mb.AddBin(v2, true)
	mb.AddBin(types.Coin, true)
	mb.AddBin(v1, true)

	c1, err := mb.ReadBin(true)
	if err != nil {
		t.Error(err)
	}
	if c1 != v1 {
		t.Error("Got not equal")
	}

	c2, err := mb.ReadBin(true)
	if err != nil {
		t.Error(err)
	}
	if c2 != v2 {
		t.Error("Got not equal")
	}

	c3, err := mb.ReadBin(true)
	if err != nil {
		t.Error(err)
	}
	if c3 != types.Coin {
		t.Error("Got not equal")
	}

	_, _, err = mb.ReadUint64()
	if err == nil {
		t.Error("Should have out of bounds error")
	}

	mb = NewMsgBuffer()
	v1 = 3

	mb.AddByte(byte(v1))

	c1, err = mb.ReadBin(true)
	assert.Nil(t, err)
	assert.Equal(t, c1, types.BinVal(1))
}

func TestBool(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 bool = true
	var v2 bool

	mb.AddBool(v1)
	mb.AddBool(v2)
	mb.AddBool(v1)

	c1, err := mb.ReadBool()
	if err != nil {
		t.Error(err)
	}
	if c1 != v1 {
		t.Error("Got not equal")
	}

	c2, err := mb.ReadBool()
	if err != nil {
		t.Error(err)
	}
	if c2 != v2 {
		t.Error("Got not equal")
	}

	_, _, err = mb.ReadUint64()
	if err == nil {
		t.Error("Should have out of bounds error")
	}

	mb = NewMsgBuffer()

	mb.AddByte(byte(3))

	c1, err = mb.ReadBool()
	assert.Nil(t, err)
	assert.Equal(t, c1, true)
}

func TestBigInt(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 = &big.Int{}
	var v2 *big.Int

	v1.SetBytes([]byte("Abigrandomnumberfrombytesiswrittenhere"))

	mb.AddBigInt(v1)
	mb.AddBigInt(v2)

	c1, _, err := mb.ReadBigInt()
	if err != nil {
		t.Error(err)
	}
	if c1.Cmp(v1) != 0 {
		t.Error("Got not equal")
	}

	c2, _, err := mb.ReadBigInt()
	if err != nil {
		t.Error(err)
	}
	if c2.Cmp(&big.Int{}) != 0 {
		t.Error("Got not equal")
	}

}

func TestUint64(t *testing.T) {
	mb := NewMsgBuffer()
	var v1 uint64 = math.MaxUint64
	var v2 uint64

	mb.AddUint64(v1)
	mb.AddUint64(v2)
	mb.AddUint64(v1)

	c1, _, err := mb.ReadUint64()
	if err != nil {
		t.Error(err)
	}
	if c1 != v1 {
		t.Error("Got not equal")
	}

	c2, _, err := mb.ReadUint64()
	if err != nil {
		t.Error(err)
	}
	if c2 != v2 {
		t.Error("Got not equal")
	}

}

func TestWrite(t *testing.T) {
	mb := NewMsgBuffer()

	someString := "10jklscka;jskdfj;eiafnankjbavljjadfio;eajfia;lcmnda;3903r3u 8jjf3;lkjf0 93 jfqj kqlfm lkjfq fij;lkaknca;lkjfieaofj"

	mb.AddBytes([]byte(someString))

	result, err := mb.ReadBytes(len(someString))
	if err != nil {
		t.Error(err)
	}
	if string(result) != someString {
		t.Error("Got not equal")
	}

	mb.ResetOffset()
	into := make([]byte, 10)
	n, err := mb.Read(into)
	assert.Equal(t, 10, n)
	assert.Nil(t, err)
	assert.Equal(t, someString[:10], string(into))

	remain := make([]byte, len(someString))
	n, err = mb.Read(remain)
	assert.Equal(t, err, io.EOF)
	assert.Equal(t, len(someString)-10, n)
	assert.Equal(t, someString, (string(into) + string(remain))[:len(someString)])

	mb.AddBytes([]byte(someString))

	_, err = mb.ReadBytes(len(someString) + 1)
	if err == nil {
		t.Error("Should have gotten out of bounds error")
	}
}
