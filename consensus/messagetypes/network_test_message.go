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
	"fmt"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/consensus/messages"
)

type NetworkTestMessage struct {
	ID    uint32
	Index types.ConsensusID
}

func (tm *NetworkTestMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(tm.GetID())
}

func (tm *NetworkTestMessage) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, offset := messages.WriteHeaderHead(tm.Index, nil, tm.GetID(), 0, (*messages.MsgBuffer)(m))
	// the msg
	var l1 int
	if tm.Index.(types.ConsensusInt)%2 == 0 {
		l1, _ = (*messages.MsgBuffer)(m).AddBytes(testBytes)
		l += l1
	}
	// a test uint
	l1, _ = (*messages.MsgBuffer)(m).AddUint32(testVal)
	l += l1
	l1, endoffset := (*messages.MsgBuffer)(m).AddUint32(tm.ID)
	endoffset += l1
	l += l1
	_, _, err := m.ComputeHash(offset, endoffset, true)
	if err != nil {
		return 0, err
	}
	l += types.GetHashLen()
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

func (tm *NetworkTestMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

func (tm *NetworkTestMessage) GetIndex() types.ConsensusID {
	return tm.Index
}

// PeekHeader returns the indices related to the messages without impacting m.
func (NetworkTestMessage) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	return messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
}

func (tm *NetworkTestMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	l, index, _, size, offset, err := messages.ReadHeaderHead(tm.GetID(), unmarFunc.ConsensusIDUnMarshaler,
		(*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}
	tm.Index = index
	if tm.Index.(types.ConsensusInt)%2 == 0 {
		val, err := (*messages.MsgBuffer)(m).ReadBytes(len(testBytes))
		if err != nil {
			return 0, err
		}
		l += len(testBytes)
		if !bytes.Equal(val, testBytes) {
			return 0, ErrInvalidStringMsg
		}
	}
	val2, br, err := (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}
	l += br
	if val2 != testVal {
		return 0, fmt.Errorf("not equal int: %v, %v", val2, testVal)
	}
	val3, br, err := (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return 0, err
	}
	l += br
	tm.ID = val3
	endoffset := (*messages.MsgBuffer)(m).GetReadOffset()
	// end of main msg

	// now the hash
	_, _, err = m.CheckHash(offset, endoffset, true)
	if err != nil {
		return 0, err
	}
	l += types.GetHashLen()
	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}
	return l, nil
}

func (tm NetworkTestMessage) GetID() messages.HeaderID {
	return messages.HdrNetworkTest
}
