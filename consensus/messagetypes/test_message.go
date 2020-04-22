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
package messagetypes

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/messages"
)

var testBytes = make([]byte, config.TestMsgSize)

// var testStr = string(testBytes)

func init() {
	_, err := rand.Read(testBytes)
	if err != nil {
		panic(err)
	}
	// testStr = string(testBytes)
}

var testVal uint32 = 99
var tstMsgSize = 12 + len(testBytes) + types.GetHashLen() // TODO dont use static size

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type badTestMessage1 int // Has invalid string

func (tm badTestMessage1) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

func (tm badTestMessage1) Serialize(m *messages.Message) (int, error) {
	// first size
	(*messages.MsgBuffer)(m).AddUint32(uint32(tstMsgSize))
	// then id
	_, offset := (*messages.MsgBuffer)(m).AddUint32(uint32(tm.GetID()))
	// then message (string is missing last 4 bytes)
	(*messages.MsgBuffer)(m).AddBytes(testBytes[:len(testBytes)-4])
	_, endoffset := (*messages.MsgBuffer)(m).AddUint32(testVal)
	endoffset += 4
	_, _, err := m.ComputeHash(offset, endoffset, true)
	if err != nil {
		return 0, err
	}
	return tstMsgSize - 4, nil
}

func (tm badTestMessage1) CreateCopy() messages.MsgHeader {
	return tm
}

// PeekHeader returns the indices related to the messages without impacting m.
func (badTestMessage1) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	panic("TODO/unused")
}

func (tm badTestMessage1) GetBytes(m *messages.Message) ([]byte, error) {
	return (*messages.MsgBuffer)(m).ReadBytes(tstMsgSize)
}

func (tm badTestMessage1) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	return 0, nil
}

func (tm badTestMessage1) GetID() messages.HeaderID {
	return messages.HdrTest
}

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type badTestMessage2 int // Not a full message

func (tm badTestMessage2) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

func (tm badTestMessage2) Serialize(m *messages.Message) (int, error) {
	// first size
	(*messages.MsgBuffer)(m).AddUint32(uint32(tstMsgSize))
	// then id
	(*messages.MsgBuffer)(m).AddUint32(uint32(tm.GetID()))
	// missing the rest of the message
	(*messages.MsgBuffer)(m).AddUint32(testVal)
	return 12, nil
}

// PeekHeader returns the indices related to the messages without impacting m.
func (badTestMessage2) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {
	panic("TODO/unused")
}

func (tm badTestMessage2) CreateCopy() messages.MsgHeader {
	return tm
}

func (tm badTestMessage2) GetBytes(m *messages.Message) ([]byte, error) {
	return (*messages.MsgBuffer)(m).ReadBytes(tstMsgSize)
}

func (tm badTestMessage2) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	return 0, nil
}

func (tm badTestMessage2) GetID() messages.HeaderID {
	return messages.HdrTest
}

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type badTestMessage3 int // Has a bad hash

func (tm badTestMessage3) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

// PeekHeader returns the indices related to the messages without impacting m.
func (badTestMessage3) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	panic("TODO/unused")
}

func (tm badTestMessage3) Serialize(m *messages.Message) (int, error) {
	// first size
	(*messages.MsgBuffer)(m).AddUint32(uint32(tstMsgSize))
	// then id
	_, offset := (*messages.MsgBuffer)(m).AddUint32(uint32(tm.GetID()))
	// then message
	(*messages.MsgBuffer)(m).AddBytes(testBytes)
	_, endoffset := (*messages.MsgBuffer)(m).AddUint32(testVal)
	endoffset += 4
	// bad hash starts 1 char late
	_, l, err := m.ComputeHash(offset+1, endoffset, true)
	if err != nil {
		return 0, err
	}
	if (12 + len(testBytes) + l) != tstMsgSize {
		panic("size is wrong")
	}
	return tstMsgSize, nil
}

func (tm badTestMessage3) CreateCopy() messages.MsgHeader {
	return tm
}

func (tm badTestMessage3) GetBytes(m *messages.Message) ([]byte, error) {
	return (*messages.MsgBuffer)(m).ReadBytes(tstMsgSize)
}

func (tm badTestMessage3) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	return 0, nil
}

func (tm badTestMessage3) GetID() messages.HeaderID {
	return messages.HdrTest
}

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type testMessage int

// PeekHeader returns the indices related to the messages without impacting m.
func (testMessage) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {
	panic("TODO/unused")
}

func (tm testMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

func (tm testMessage) Serialize(m *messages.Message) (int, error) {
	// first size
	(*messages.MsgBuffer)(m).AddUint32(uint32(tstMsgSize))
	// then id
	_, offset := (*messages.MsgBuffer)(m).AddUint32(uint32(tm.GetID()))
	// then message
	(*messages.MsgBuffer)(m).AddBytes(testBytes)
	_, endoffset := (*messages.MsgBuffer)(m).AddUint32(testVal)
	endoffset += 4

	// then size
	_, l, err := m.ComputeHash(offset, endoffset, true)
	if err != nil {
		return 0, err
	}
	if (12 + len(testBytes) + l) != tstMsgSize {
		panic("size is wrong")
	}
	return tstMsgSize, nil
}

func (tm testMessage) CreateCopy() messages.MsgHeader {
	return tm
}

func (tm testMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return (*messages.MsgBuffer)(m).ReadBytes(tstMsgSize)
}

var ErrInvalidStringMsg = fmt.Errorf("not equal string")

func (tm testMessage) Deserialize(m *messages.Message, _ types.ConsensusIndexFuncs) (int, error) {
	// first size
	_, _, err := (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}

	// id
	offset := (*messages.MsgBuffer)(m).GetReadOffset()
	id, _, err := (*messages.MsgBuffer)(m).ReadHeaderID()
	if err != nil {
		return 0, err
	}
	if id != messages.HdrTest {
		return 0, types.ErrWrongMessageType
	}
	// the msg
	val, err := (*messages.MsgBuffer)(m).ReadBytes(len(testBytes))
	if err != nil {
		return 0, err
	}
	if !bytes.Equal(testBytes, val) {
		return 0, ErrInvalidStringMsg
	}
	val2, _, err := (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}
	if val2 != testVal {
		return 0, fmt.Errorf("not equal int: %v, %v", val2, testVal)
	}

	// check the hash
	endoffset := (*messages.MsgBuffer)(m).GetReadOffset()
	_, l, err := m.CheckHash(offset, endoffset, true)
	if err != nil {
		return 0, err
	}
	if 12+len(testBytes)+l != tstMsgSize {
		return 0, types.ErrInvalidMsgSize
	}
	return tstMsgSize, nil
}

func (tm testMessage) GetID() messages.HeaderID {
	return messages.HdrTest
}

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

// TestMessageTimeout is used in the csnet package for testing.
type TestMessageTimeout int

const tstMsgTimeoutSize = 8 // TODO no static size

func (tm TestMessageTimeout) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

func (tm TestMessageTimeout) Serialize(m *messages.Message) (int, error) {
	// first size
	(*messages.MsgBuffer)(m).AddUint32(uint32(tstMsgTimeoutSize))
	// then id
	(*messages.MsgBuffer)(m).AddUint32(uint32(tm.GetID()))
	return 4, nil
}

func (tm TestMessageTimeout) CreateCopy() messages.MsgHeader {
	return tm
}

// PeekHeader returns the indices related to the messages without impacting m.
func (TestMessageTimeout) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

func (tm TestMessageTimeout) GetBytes(m *messages.Message) ([]byte, error) {
	return (*messages.MsgBuffer)(m).ReadBytes(8)
}

func (tm TestMessageTimeout) Deserialize(m *messages.Message, _ types.ConsensusIndexFuncs) (int, error) {
	// first size
	size, _, err := (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}
	if size != tstMsgTimeoutSize {
		return 0, types.ErrInvalidMsgSize
	}
	// then id
	id, _, err := (*messages.MsgBuffer)(m).ReadHeaderID()
	if err != nil {
		return 0, err
	}
	if id != messages.HdrTestTimeout {
		return 0, types.ErrWrongMessageType
	}
	return tstMsgTimeoutSize, nil
}

func (tm TestMessageTimeout) GetID() messages.HeaderID {
	return messages.HdrTestTimeout
}
