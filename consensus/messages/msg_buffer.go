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
	"encoding/binary"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"math/big"

	"github.com/tcrain/cons/config"
)

var encoding = config.Encoding

// MsgBuffer is for serializing a message.
// Read/get and write/put operations can be performed on the buffer.
// Read operations read based on the index of the previous read, where each
// operation incraments the index.
// E.g. the first ReadUint32 reads bytes 0-3, and the follow reads 3-7
// Peek operations do not change the index.
// Write operations write to the end of the buffer.
// ReadAt/WriteAt operations read/write at a specific index in the buffer.
type MsgBuffer struct {
	tmpBuf      []byte // temporary buffer for writing varints
	buff        []byte // The actual bytes
	readOffset  int    // The index where the next read/get operation will be performed, each read/get operation increments this by the amount of bytes read
	writeOffset int    // The index where the next write/put operation will be performed, each write/put operation increments this by the amount of bytes written
}

// ToMsgBuffer creates a MsgBuffer object from the bytes, the read offset
// is set to the beginning and the writeoffset is set to the end
func ToMsgBuffer(buff []byte) *MsgBuffer {
	return &MsgBuffer{
		buff:        buff,
		writeOffset: len(buff),
	}
}

// NewMsgBuffer creates a new empty buffer
func NewMsgBuffer() *MsgBuffer {
	return &MsgBuffer{
		buff: make([]byte, 0, 64),
	}
}

// NewMsgBuffer creates a new buffer using the following: "make([]byte, i, n)"
func NewMsgBufferSize(i int, n int) *MsgBuffer {
	return &MsgBuffer{
		buff:        make([]byte, i, n),
		writeOffset: i,
	}
}

// Write appends p to the end of the buffer and returns len(p)
func (mb *MsgBuffer) Write(p []byte) (n int, err error) {
	mb.buff = append(mb.buff, p...)
	mb.writeOffset += len(p)
	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}
	return len(p), nil
}

// WriteByte appends p to the end of the buffer
func (mb *MsgBuffer) WriteByte(p byte) error {
	mb.buff = append(mb.buff, p)
	mb.writeOffset++
	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}
	return nil
}

// WriteAt writes p to the buffer at offset and returns len(p)
func (mb *MsgBuffer) WriteAt(offset int, p []byte) (n int, err error) {
	if len(mb.buff) < offset+len(p) {
		return 0, types.ErrNotEnoughBytes
	}
	copy(mb.buff[offset:offset+len(p)], p)
	return len(p), nil
}

// AddHeaderID encodes v and appends it to the end of the buffer.
// It returns 4 and the offset of where v was written in the buffer.
func (mb *MsgBuffer) AddHeaderID(v HeaderID) (int, int) {
	return mb.AddUint32(uint32(v))
}

// AddConsensusRound encodes the round and appends to the end of the buffer.
// It returns the number of bytes written and where v was written in the buffer.
func (mb *MsgBuffer) AddConsensusRound(v types.ConsensusRound) (int, int) {
	return mb.AddUint32(uint32(v))
}

// AddConsensusID encodes the id and appends to the end of the buffer.
// It returns the number of bytes written and where v was written in the buffer.
func (mb *MsgBuffer) AddConsensusID(v types.ConsensusID) (int, int) {
	byt, err := v.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return mb.AddBytes(byt)
}

// AddUvarint writes a unsigned variable size integer.
// It returns the number of bytes written and where v was written in the buffer.
func (mb *MsgBuffer) AddUvarint(v uint64) (n int, offest int) {
	if len(mb.tmpBuf) == 0 {
		mb.tmpBuf = make([]byte, binary.MaxVarintLen64)
	}
	n = binary.PutUvarint(mb.tmpBuf, v)
	return mb.AddBytes(mb.tmpBuf[:n])
}

// AddUint32 encodes v and appends it to the end of the buffer.
// It returns 4 and the offset of where v was written in the buffer.
func (mb *MsgBuffer) AddUint32(v uint32) (int, int) {
	off := mb.writeOffset
	err := binary.Write(mb, encoding, v)
	if err != nil {
		panic(err)
	}
	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}

	return 4, off
}

// ResetOffest sets the read offset to 0, so the following read will read from
// the beginning of the buffer
func (mb *MsgBuffer) ResetOffset() {
	mb.readOffset = 0
}

// AddBinInt encodes v and appends it to the end of the buffer.
// It returns the number of bytes written, and the offset of where it was written.
func (mb *MsgBuffer) AddBigInt(v *big.Int) (int, int) {
	buff, err := v.GobEncode()
	if err != nil {
		panic(err)
	}
	off := mb.writeOffset
	mb.AddUint32(uint32(len(buff)))
	mb.AddBytes(buff)

	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}

	return len(buff) + 4, off
}

// RemainingLen returns the number of bytes from the current read offset to the end of the buffer.
func (mb *MsgBuffer) RemainingLen() int {
	return len(mb.buff) - mb.readOffset
}

// WriteUint32 encodes and overwrites the beggning of the message with v
func (mb *MsgBuffer) WriteUint32AtStart(v uint32) error {
	return mb.WriteUint32At(0, v)
}

// WriteUint32At encodes v and overwrites the bytes at offset
func (mb *MsgBuffer) WriteUint32At(offset int, v uint32) error {
	if len(mb.buff) < offset+4 {
		return types.ErrNotEnoughBytes
	}
	encoding.PutUint32(mb.buff[offset:offset+4], v)
	return nil
}

// func (mb *MsgBuffer) WriteUint64AtStart(v uint64) error {
// 	if len(mb.buff) < 4 {
// 		return utils.ErrNotEnoughBytes
// 	}
// 	encoding.PutUint64(mb.buff[0:8], v)
// 	return nil
// }

// WriteSizeBytes writes a slice of bytes to be read by ReadSizeBytes (it contains a serialized number followed by the bytes)
// It returns the number of bytes written and the offset where the values were written in the buffer.
func (mb *MsgBuffer) AddSizeBytes(buff []byte) (int, int) {
	l, off := mb.AddUint32(uint32(len(buff)))
	l1, _ := mb.AddBytes(buff)
	return l + l1, off
}

// AddBytes append v to the end of the buffer.
// It reutrns len(v) and the offset where v was written in the buffer.
func (mb *MsgBuffer) AddBytes(v []byte) (int, int) {
	off := mb.writeOffset
	_, err := mb.Write(v)
	if err != nil {
		panic(err)
	}

	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}

	return len(v), off
}

// AddByte appends v to the end of the buffer.
// It returns the offset where v was written in the buffer.
func (mb *MsgBuffer) AddByte(v byte) int {
	off := mb.writeOffset
	err := mb.WriteByte(v)
	if err != nil {
		panic(err)
	}

	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}

	return off
}

// AddBool appends a boolean value v to the end of the buffer.
// It returns the offset where v was written to the buffer.
func (mb *MsgBuffer) AddBool(v bool) int {
	var b byte
	if v {
		b = 1
	}
	return mb.AddByte(b)
}

// AddBin appends binary value v to the end of the buffer.
// It returns the offset where v was written to the buffer.
func (mb *MsgBuffer) AddBin(v types.BinVal, allowCoin bool) int {
	switch v {
	case 0:
	case 1:
	case 2:
		if allowCoin {
			break
		}
		fallthrough
	default:
		panic("tried to write a non binary value")
	}
	return mb.AddByte(byte(v))
}

// ReadBigInt reads a big integer from the buffer,
// It returns the big int, and the new read index
func (mb *MsgBuffer) ReadBigInt() (*big.Int, int, error) {
	l, br, err := mb.ReadUint32()
	if err != nil {
		return nil, 0, err
	}
	offset := mb.readOffset
	mb.readOffset += int(l)
	if len(mb.buff) < mb.readOffset {
		mb.readOffset = offset
		return nil, 0, types.ErrNotEnoughBytes
	}
	z := &big.Int{}
	err = z.GobDecode(mb.buff[offset:mb.readOffset])
	return z, br + int(l), err
}

// ReadHeaderID reads a HeaderID from the buffer.
// It returns the value read, the number of bytes read, and any errors.
// It also increments the read offset.
func (mb *MsgBuffer) ReadHeaderID() (HeaderID, int, error) {
	hd, br, err := mb.ReadUint32()
	return HeaderID(hd), br, err
}

// PeekUvarintAt returns the uvarint at the current read offset plus offset.
// It returns the uint64 and the number of bytes read
func (mb *MsgBuffer) PeekUvarintAt(offset int) (v uint64, n int, err error) {
	if mb.readOffset+offset >= len(mb.buff) {
		err = types.ErrNotEnoughBytes
		return
	}
	startOffset := mb.readOffset
	mb.readOffset += offset
	startRead := mb.readOffset
	v, err = binary.ReadUvarint(mb)
	n = mb.readOffset - startRead
	mb.readOffset = startOffset
	return
}

// ReadUvarint reads an unsigned variable sized integer from the buffer.
// It returns the value read, the number of bytes read and any errors.
func (mb *MsgBuffer) ReadUvarint() (v uint64, n int, err error) {
	if mb.readOffset >= len(mb.buff) {
		err = types.ErrNotEnoughBytes
		return
	}
	start := mb.readOffset
	v, err = binary.ReadUvarint(mb)
	n = mb.readOffset - start
	return
}

// ReadConsensusRound reads a consensus round from the buffer.
// It returns the value read, the number of bytes read, and any errors.
// It also increments the read offset.
func (mb *MsgBuffer) ReadConsensusRound() (types.ConsensusRound, int, error) {
	cr, br, err := mb.ReadUint32()
	return types.ConsensusRound(cr), br, err
}

// ReadConsensusIndex reads a consensus index from the buffer.
// It returns the value read, the number of bytes read, and any errors.
// It also increments the read offset.
func (mb *MsgBuffer) ReadConsensusIndex() (types.ConsensusInt, int, error) {
	cr, br, err := mb.ReadUint64()
	return types.ConsensusInt(cr), br, err
}

// ReadConsensusID reads a consensus id from the buffer.
// It returns the value read, the number of bytes read, and any errors.
// It also increments the read offset.
func (mb *MsgBuffer) ReadConsensusID(unmarFunc types.ConsensusIDUnMarshaler) (types.ConsensusID, int, error) {
	ret, n, err := unmarFunc(mb.buff[mb.readOffset:])
	mb.readOffset += n
	return ret, n, err
}

// ReadUint32 reads a uint32 from the buffer.
// It returns the value read, the number of bytes read, and any errors.
// It also increments the read offset.
func (mb *MsgBuffer) ReadUint32() (uint32, int, error) {
	offset := mb.readOffset
	mb.readOffset += 4
	if len(mb.buff) < mb.readOffset {
		mb.readOffset = offset
		return 0, 0, types.ErrNotEnoughBytes
	}
	v := encoding.Uint32(mb.buff[offset:mb.readOffset])
	return v, 4, nil
}

// PeekUint32 returns the uint32 at the current read offset
func (mb *MsgBuffer) PeekUint32() (uint32, error) {
	return mb.PeekUint32At(0)
}

// PeekUint32 returns the uint32 at the current offset plus offs
func (mb *MsgBuffer) PeekUint32At(offs int) (uint32, error) {
	offset := mb.readOffset + offs
	end := offset + 4
	if len(mb.buff) < end {
		return 0, types.ErrNotEnoughBytes
	}
	v := encoding.Uint32(mb.buff[offset:end])
	return v, nil
}

// PeekUint64 returns the uint64 at the current read offset
func (mb *MsgBuffer) PeekUint64() (uint64, error) {
	return mb.PeekUint64At(0)
}

// PeekUint64At returns the uint64 at the current read offset plus offs
func (mb *MsgBuffer) PeekUint64At(offs int) (uint64, error) {
	offset := mb.readOffset + offs
	end := offset + 8
	if len(mb.buff) < end {
		return 0, types.ErrNotEnoughBytes
	}
	v := encoding.Uint64(mb.buff[offset:end])
	return v, nil
}

// PeekConsIDAt gets the consensusID at the given offset using the unmarshal function.
// It returns the id and the number of bytes read.
func (mb *MsgBuffer) PeekConsIDAt(offs int, unmarFunc types.ConsensusIDUnMarshaler) (types.ConsensusID, int, error) {
	offset := mb.readOffset + offs
	return unmarFunc(mb.buff[offset:])
}

// AddUint64 encodes v and appends it to the end of the buffer.
// It returns 8 and the index where v was written
func (mb *MsgBuffer) AddUint64(v uint64) (int, int) {
	off := mb.writeOffset
	err := binary.Write(mb, encoding, v)
	if err != nil {
		panic(err)
	}
	if mb.writeOffset != len(mb.buff) {
		panic("offset")
	}

	return 8, off
}

// ReadKey return the uint64 at the current offset, the number of bytes read, and any error.
// It also increments the read offset.
func (mb *MsgBuffer) ReadUint64() (uint64, int, error) {
	offset := mb.readOffset
	mb.readOffset += 8
	if len(mb.buff) < mb.readOffset {
		mb.readOffset = offset
		return 0, 0, types.ErrNotEnoughBytes
	}
	v := encoding.Uint64(mb.buff[offset:mb.readOffset])
	return v, 8, nil
}

// GetRemainingBytes returns the remaining bytes from the current read offset.
func (mb *MsgBuffer) GetRemainingBytes() []byte {
	return mb.buff[mb.readOffset:]
}

// GetReadOffset returns the current read offset
func (mb *MsgBuffer) GetReadOffset() int {
	return mb.readOffset
}

// GetWriteOffest returns the current write offset (this will always be the end of the buffer)
func (mb *MsgBuffer) GetWriteOffset() int {
	return mb.writeOffset
}

// ReadSizeBytes reads a slice of bytes written by AddSizeBytes, it returns the nubmer of bytes read (i.e. a serialized number followed by that many bytes)
func (mb *MsgBuffer) ReadSizeBytes() ([]byte, int, error) {
	size, br, err := mb.ReadUint32()
	if err != nil {
		return nil, 0, err
	}
	isize := int(size)
	ret, err := mb.ReadBytes(isize)
	if err != nil {
		return nil, 0, err
	}
	return ret, br + isize, nil
}

// Read implements io.Reader read
func (mb *MsgBuffer) Read(p []byte) (n int, err error) {
	if mb.readOffset+len(p) > len(mb.buff) {
		copy(p, mb.buff[mb.readOffset:len(mb.buff)])
		n = len(mb.buff) - mb.readOffset
		mb.readOffset = len(mb.buff)
		err = io.EOF
		return
	}
	copy(p, mb.buff[mb.readOffset:])
	mb.readOffset += len(p)
	n = len(p)
	return
}

// ReadBytes reads n bytes from the buffer and updates the read offset
func (mb *MsgBuffer) ReadBytes(n int) ([]byte, error) {
	offset := mb.readOffset
	mb.readOffset += n
	if len(mb.buff) < mb.readOffset {
		mb.readOffset = offset
		return nil, types.ErrNotEnoughBytes
	}
	return mb.buff[offset:mb.readOffset], nil
}

// var ErrNonBinValue = fmt.Errorf("Tried to read a non-binary value")

// ReadBool reads a boolean value from the buffer and increments the read offset.
// A bool is encoded as a byte, any non-zero value is considered true.
func (mb *MsgBuffer) ReadBool() (bool, error) {
	v, err := mb.ReadByte()
	if err != nil {
		return false, err
	}
	switch v {
	case 0:
		return false, nil
	default:
		return true, nil
	}
}

// ReadBin reads a binary value from the buffer and increments the read offset.
// A binVal is 1 byte, any non-zero value is 1.
func (mb *MsgBuffer) ReadBin(allowCoin bool) (types.BinVal, error) {
	v, err := mb.ReadByte()
	if err != nil {
		return 0, err
	}
	switch v {
	case 0:
		return 0, nil
	case byte(types.Coin):
		if allowCoin {
			return types.Coin, nil
		}
		fallthrough
	default:
		return 1, nil
	}
}

// ReadByte reads a byte from the buffer and increments the read offset
func (mb *MsgBuffer) ReadByte() (byte, error) {
	offset := mb.readOffset
	mb.readOffset++
	if len(mb.buff) < mb.readOffset {
		mb.readOffset = offset
		return 0, types.ErrNotEnoughBytes
	}
	return mb.buff[offset], nil
}

// GetSlice returns the slice of bytes from start to end in the buffer
func (mb *MsgBuffer) GetSlice(start, end int) ([]byte, error) {
	if start >= end || start < 0 || len(mb.buff) < end {
		return nil, types.ErrNotEnoughBytes
	}
	return mb.buff[start:end], nil
}

// GetSlice from returns the slice starting at start until the end of the buffer
func (mb *MsgBuffer) GetSliceFrom(start int) ([]byte, error) {
	if start < 0 || len(mb.buff) < start {
		return nil, types.ErrNotEnoughBytes
	}
	return mb.buff[start:], nil
}
