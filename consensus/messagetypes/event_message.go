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
	"github.com/tcrain/cons/consensus/graph"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// EventMessage is used during multi-value consensus to send a graph based message to all other nodes.
// It implements messages.MsgHeader
type EventMessage struct {
	Event     graph.EventInfo
	eventHash types.HashBytes
}

// NewEventMessage creates a new mv echo message
func NewEventMessage() *EventMessage {
	return &EventMessage{}
}

func (hm *EventMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &EventMessage{Event: hm.Event}
}

// GetSignType returns types.NormalSignature
func (*EventMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine
// (this is because we have concurrent proposals).
func (hm *EventMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	return
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (hm *EventMessage) GetMsgID() messages.MsgID {
	return messages.EventInfoMsgID{}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (hm *EventMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {

	b := bytes.NewBuffer(nil)
	if _, err = hm.Event.Encode(b); err != nil {
		return
	}
	hm.eventHash = b.Bytes()
	bytesWritten, signEndOffset = (*messages.MsgBuffer)(m).AddSizeBytes(b.Bytes())
	signEndOffset += bytesWritten

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (hm *EventMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	var buff []byte
	if buff, bytesRead, err = (*messages.MsgBuffer)(m).ReadSizeBytes(); err != nil {
		return
	}
	var n int
	if n, err = (&hm.Event).Decode(bytes.NewReader(buff)); err != nil {
		return
	}
	if n != len(buff) {
		err = types.ErrInvalidMsgSize
		return
	}
	hm.eventHash = types.GetHash(buff)
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetEventInfoHash returns the hash of the event info object.
func (hm *EventMessage) GetEventInfoHash() types.HashBytes {
	return hm.eventHash
}

// GetID returns the header id for this header
func (hm *EventMessage) GetID() messages.HeaderID {
	return messages.HdrEventInfo
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (hm *EventMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return hm
}
