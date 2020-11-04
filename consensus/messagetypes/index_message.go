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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/graph"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

// IndexRecoverMsg is used for MvCons4 recovery
type IndexRecoverMsg struct {
	IndexMessage
	basicMessage
}

// NewIndexMessage creates a new index recover message.
func NewIndexRecoverMsg(index types.ConsensusIndex) *IndexRecoverMsg {
	return &IndexRecoverMsg{basicMessage: basicMessage{
		cid: index}}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (npm *IndexRecoverMsg) Serialize(m *messages.Message) (int, error) {
	l, err := npm.basicMessage.serializeID(m, npm.GetID())
	if err != nil {
		return l, err
	}
	v, _, err := npm.SerializeInternal(m)
	return v + l, err
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (npm *IndexRecoverMsg) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	l, err := npm.basicMessage.deserializeID(m, npm.GetID(), unmarFunc.ConsensusIDUnMarshaler)
	if err != nil {
		return l, err
	}
	br, _, err := npm.DeserializeInternal(m)
	return br, err
}

// GetID returns the header id for this header
func (npm *IndexRecoverMsg) GetID() messages.HeaderID {
	return messages.HdrIdxRecover
}

// IndexMessage is used during multi-value consensus to send a graph based message to all other nodes.
// It implements messages.MsgHeader
type IndexMessage struct {
	Indices []graph.IndexType
	IsReply bool
}

// NewIndexMessage creates a new index message.
func NewIndexMessage() *IndexMessage {
	return &IndexMessage{}
}

func (hm *IndexMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &IndexMessage{Indices: hm.Indices}
}

// GetSignType returns types.NormalSignature
func (*IndexMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// NeedsSMValidation returns nil.
func (hm *IndexMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	return
}

// GetMsgID returns the MsgID for this specific header, in this case a BasicMsgID (see MsgID definition for more details)
func (hm *IndexMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(hm.GetID())
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (hm *IndexMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {

	var n int
	// write the number of indices
	if n, err = utils.EncodeUvarint(uint64(len(hm.Indices)), (*messages.MsgBuffer)(m)); err != nil {
		return
	}
	bytesWritten += n
	// write each index
	for _, nxt := range hm.Indices {
		if n, err = utils.EncodeUvarint(uint64(nxt), (*messages.MsgBuffer)(m)); err != nil {
			return
		}
		bytesWritten += n
	}
	// write if it is a reply message
	(*messages.MsgBuffer)(m).AddBool(hm.IsReply)
	bytesWritten++
	signEndOffset = (*messages.MsgBuffer)(m).GetWriteOffset()

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (hm *IndexMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {

	hm.Indices = nil
	var count uint64
	var n int
	if count, n, err = utils.ReadUvarint((*messages.MsgBuffer)(m)); err != nil {
		return
	}
	bytesRead += n
	if count > config.MaxMsgSize {
		err = types.ErrInvalidMsgSize
		return
	}
	for i := 0; i < int(count); i++ {
		var nxt uint64
		if nxt, n, err = utils.ReadUvarint((*messages.MsgBuffer)(m)); err != nil {
			return
		}
		bytesRead += n
		hm.Indices = append(hm.Indices, graph.IndexType(nxt))
	}
	if hm.IsReply, err = (*messages.MsgBuffer)(m).ReadBool(); err != nil {
		return
	}
	bytesRead++
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetID returns the header id for this header
func (hm *IndexMessage) GetID() messages.HeaderID {
	return messages.HdrIdx
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (hm *IndexMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return hm
}
