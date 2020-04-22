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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// MvInitSupportMessage is used by the coordinator of a round of multi-value consensus to send
// its proposal with the additional information about the previous proposal it points to.
// It implements messages.MsgHeader
type MvInitSupportMessage struct {
	Proposal       []byte             // The proposal
	ByzProposal    []byte             // Alternative proposal for byzantine nodes
	SupportedIndex types.ConsensusInt // the index supported by this init
	SupportedHash  types.HashBytes    // the hash of the init message supported
}

// NewMvInitMessage creates a new mv init message
func NewMvInitSupportMessage() *MvInitSupportMessage {
	return &MvInitSupportMessage{}
}

// ShallowCopy makes a shallow copy of the message
func (mvi *MvInitSupportMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvInitSupportMessage{Proposal: mvi.Proposal,
		SupportedIndex: mvi.SupportedIndex,
		SupportedHash:  mvi.SupportedHash}
}

// GetSignType returns types.NormalSignature
func (*MvInitSupportMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// NeedsSMValidation returns the proposal bytes to be validated by the state machine.
func (mvi *MvInitSupportMessage) NeedsSMValidation(msgIndex types.ConsensusIndex, proposalIdx int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	if proposalIdx != 0 {
		panic("proposalIdx must be 0")
	}

	switch msgIndex.Index.(type) {
	case types.ConsensusID:
		if len(mvi.Proposal) == 0 {
			panic("should have handled nil proposal earlier")
		}
	default:
		panic("unsupported")
	}
	idx = msgIndex.ShallowCopy()
	idx.Index, idx.FirstIndex = mvi.SupportedIndex, mvi.SupportedIndex
	return idx, mvi.Proposal, nil
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mvi *MvInitSupportMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mvi.GetID()}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mvi *MvInitSupportMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// The proposal
	v, _ := (*messages.MsgBuffer)(m).AddSizeBytes(mvi.Proposal)
	bytesWritten += v

	// The supported index
	v, _ = (*messages.MsgBuffer)(m).AddConsensusID(mvi.SupportedIndex)
	bytesWritten += v

	// The supported hash
	if len(mvi.SupportedHash) != types.GetHashLen() {
		err = types.ErrInvalidHash
		return
	}
	v, signEndOffset = (*messages.MsgBuffer)(m).AddBytes(mvi.SupportedHash)
	bytesWritten += v
	signEndOffset += v

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mvi *MvInitSupportMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal
	mvi.Proposal, bytesRead, err = (*messages.MsgBuffer)(m).ReadSizeBytes()
	if err != nil {
		return
	}

	if len(mvi.Proposal) == 0 {
		err = types.ErrNilProposal
		return
	}

	// Get the supported index
	var br int
	mvi.SupportedIndex, br, err = (*messages.MsgBuffer)(m).ReadConsensusIndex()
	if err != nil {
		return
	}
	bytesRead += br

	// Get the hash
	hashLen := types.GetHashLen()
	mvi.SupportedHash, err = (*messages.MsgBuffer)(m).ReadBytes(hashLen)
	if err != nil {
		return
	}
	bytesRead += hashLen

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetID returns the header id for this header
func (mvi *MvInitSupportMessage) GetID() messages.HeaderID {
	return messages.HdrMvInitSupport
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mvi *MvInitSupportMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mvi
}
