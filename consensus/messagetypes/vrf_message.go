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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// VRFProofMessage contains a VRFProof.
type VRFProofMessage struct {
	Index    types.ConsensusIndex
	VRFProof sig.VRFProof
}

// NewVRFProofMessage creates a new simple cons message
func NewVRFProofMessage(index types.ConsensusIndex, vrf sig.VRFProof) *VRFProofMessage {
	return &VRFProofMessage{VRFProof: vrf, Index: index}
}

// GetMsgID returns the MsgID for this specific header.
func (vrm *VRFProofMessage) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(vrm.GetID())
}

// GetBytes returns the bytes that make up the message
func (vrm *VRFProofMessage) GetBytes(m *messages.Message) ([]byte, error) {
	return messages.GetBytesHelper(m)
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (vrm *VRFProofMessage) Serialize(m *messages.Message) (bytesWritten int, err error) {
	// bytesWritten, _ = (*messages.MsgBuffer)(m).AddUint32(uint32(scm.GetID()))
	var sizeOffset int
	bytesWritten, sizeOffset, _ = messages.WriteHeaderHead(vrm.Index.Index, nil, vrm.GetID(),
		0, (*messages.MsgBuffer)(m))

	var l1 int
	l1, err = vrm.VRFProof.Encode((*messages.MsgBuffer)(m))
	bytesWritten += l1
	if err != nil {
		return
	}

	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(bytesWritten))
	return
}

// PeekHeader returns the indices related to the messages without impacting m.
func (*VRFProofMessage) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex,
	err error) {

	index, err = messages.PeekHeaderHead(unmarFunc, (*messages.MsgBuffer)(m))
	if err != nil {
		return
	}
	index.Index = index.FirstIndex
	index.FirstIndex = nil
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (vrm *VRFProofMessage) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (bytesRead int, err error) {
	var size uint32
	var index types.ConsensusID
	bytesRead, index, _, size, _, err = messages.ReadHeaderHead(vrm.GetID(), unmarFunc.ConsensusIDUnMarshaler, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}
	vrm.Index = types.ConsensusIndex{
		Index:         index,
		UnmarshalFunc: unmarFunc,
	}

	var l1 int
	vrm.VRFProof = vrm.VRFProof.New()
	l1, err = vrm.VRFProof.Decode((*messages.MsgBuffer)(m))
	bytesRead += l1
	if err != nil {
		return
	}

	if size != uint32(bytesRead) {
		err = types.ErrInvalidMsgSize
		return
	}

	return
}

// GetIndex returns the consensus index for this message.
func (vrm *VRFProofMessage) GetIndex() types.ConsensusIndex {
	return vrm.Index
}

// GetID returns the header id for this header
func (vrm *VRFProofMessage) GetID() messages.HeaderID {
	return messages.HdrVrfProof
}
