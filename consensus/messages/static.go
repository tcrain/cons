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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
)

func GetBytesHelper(m *Message) ([]byte, error) {
	size, err := (*MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*MsgBuffer)(m).ReadBytes(int(size))
}

func SerializeInternalSignedHeader(index types.ConsensusIndex,
	hdr InternalSignedMsgHeader, m *Message) (bytesWritten, signOffset, signEndOffset, sizeOffset int, err error) {

	var l1 int
	bytesWritten, sizeOffset, signOffset = WriteHeaderHead(index.FirstIndex, index.AdditionalIndices, hdr.GetID(), 0, (*MsgBuffer)(m))
	l1, signEndOffset, err = hdr.SerializeInternal(m)
	bytesWritten += l1
	return
}

func DeserialzeInternalSignedHeader(hdr InternalSignedMsgHeader, unmarFunc types.ConsensusIndexFuncs,
	m *Message) (index types.ConsensusIndex, bytesRead, signOffset,
	signEndOffset int, size uint32, err error) {

	var l1 int
	var firstIndex types.ConsensusID
	var additionalIndices []types.ConsensusID
	l1, firstIndex, additionalIndices, size, signOffset, err = ReadHeaderHead(hdr.GetID(),
		unmarFunc.ConsensusIDUnMarshaler, (*MsgBuffer)(m))
	bytesRead += l1
	if err != nil {
		return
	}
	if unmarFunc.ConsensusIDUnMarshaler != nil {
		index, err = unmarFunc.ComputeConsensusID(firstIndex, additionalIndices)
	}
	l1, signEndOffset, err = hdr.DeserializeInternal(m)
	bytesRead += l1
	return
}

// PeekHeaderHead is a helper method used before deserialization,
// it returns the indices of the message.
func PeekHeaderHead(unmarFunc types.ConsensusIndexFuncs, m *MsgBuffer) (index types.ConsensusIndex, err error) {

	defer func() {
		if err != nil {
			panic(err) // TODO remove this
		}
	}()

	offset := indexIndex
	var v int
	var firstIndex types.ConsensusID
	if firstIndex, v, err = m.PeekConsIDAt(offset, unmarFunc.ConsensusIDUnMarshaler); err != nil {
		return
	}
	offset += v

	var numAddIds uint64
	if numAddIds, v, err = m.PeekUvarintAt(offset); err != nil {
		return
	}
	offset += v

	if numAddIds > config.MaxAdditionalIndices {
		err = types.ErrTooManyAdditionalIndices
		return
	}

	var additionalIndices []types.ConsensusID
	for i := 0; i < int(numAddIds); i++ {
		var nxt types.ConsensusID
		if nxt, v, err = m.PeekConsIDAt(offset, unmarFunc.ConsensusIDUnMarshaler); err != nil {
			return
		}
		offset += v
		additionalIndices = append(additionalIndices, nxt)
	}
	return unmarFunc.ComputeConsensusID(firstIndex, additionalIndices)
}

// ReadHeaderHead is a helper method used during MsgHeader.Serialize to read the first
// part of the header to message m. It reads the HeaderID and header size according
// to the message format described in the root documentation of this package.
// It checks the HeaderID is correct as well as the CsID.
// It returns the number of bytes read, the size of the full header,
// and the offset in the message where the header starts.
// IMPORTANT: if unmarFunc is nil it expects not to have any indecies,
// which meas it will not include CsID either.
func ReadHeaderHead(id HeaderID, unmarFunc types.ConsensusIDUnMarshaler, m *MsgBuffer) (l int,
	index types.ConsensusID, additionalIndices []types.ConsensusID, size uint32, offset int, err error) {

	// Initial header
	// Msg size
	size, l, err = m.ReadUint32()
	if err != nil {
		return
	}
	// Main message
	offset = m.GetReadOffset()
	var br int

	// first read the header id
	var msgid HeaderID
	msgid, br, err = m.ReadHeaderID()
	if err != nil {
		return
	}
	l += br
	if id != msgid {
		err = types.ErrWrongMessageType
		return
	}

	//  now the consensus index
	if unmarFunc != nil {
		index, br, err = m.ReadConsensusID(unmarFunc)
		l += br
		if err != nil {
			return
		}
		// Any additional indices?
		var addInCount uint64
		addInCount, br, err = m.ReadUvarint()
		l += br
		if err != nil {
			return
		}
		switch {
		case addInCount > config.MaxAdditionalIndices:
			err = types.ErrTooManyAdditionalIndices
			return
		case addInCount > 0:
			additionalIndices = make([]types.ConsensusID, int(addInCount))
			for i := range additionalIndices {
				additionalIndices[i], br, err = m.ReadConsensusID(unmarFunc)
				l += br
				if err != nil {
					return
				}
			}
		}

		// now the unique id
		var csid []byte
		csid, err = m.ReadBytes(len(config.CsID))
		if err != nil {
			return
		}
		l += len(config.CsID)
		if string(csid) != config.CsID {
			err = types.ErrWrongMessageType
			return
		}
	}

	return
}

// headerIndex is the index of the header id in the message
var headerIndex = 4

// indexIndex is the index of the consensus index in the message
var indexIndex = 8

// WriteHeaderHead is a helper method used during MsgHeader.Serialize to write the first
// part of the header to message m. It reads the HeaderID and header size according
// to the message format described in the root documentation of this package.
// It writes the size of the header, the HeaderID and the CsID.
// It returns the number of bytes written, the offset in the message where the size is stored,
// and the offset in the message where the header starts.
// Important: if index is nil, it will not write any indecies and will not include the CsID.
func WriteHeaderHead(index types.ConsensusID, additionalIndecies []types.ConsensusID, id HeaderID,
	size uint32, m *MsgBuffer) (l int, sizeOffset int, offset int) {

	// Initial header
	l, sizeOffset = m.AddUint32(size) // we will put the size here

	// SignedItem messaged starts here
	var l1 int

	l1, offset = m.AddHeaderID(id)
	l += l1

	if index != nil {
		l1, _ = m.AddConsensusID(index)
		l += l1
		if len(additionalIndecies) > config.MaxAdditionalIndices {
			panic(types.ErrTooManyAdditionalIndices)
		}
		l1, _ = m.AddUvarint(uint64(len(additionalIndecies)))
		l += l1
		for _, nxt := range additionalIndecies {
			l1, _ = m.AddConsensusID(nxt)
			l += l1
		}

		l1, _ = m.Write([]byte(config.CsID))
		l += l1
	}

	return l, sizeOffset, offset
}
