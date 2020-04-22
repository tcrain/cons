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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// MvInitMessage is used by the coordinator of a round of multi-value consensus to send
// multiple proposals using a single consensus item.
// It implements messages.MsgHeader
type MvMultiInitMessage struct {
	Proposals [][]byte // The proposals
}

// NewMvMultiInitMessage creates a new mv init message
func NewMvMultiInitMessage() *MvMultiInitMessage {
	return &MvMultiInitMessage{}
}

// NeedsSMValidation returns the proposal bytes at the given proposalIdx to be validated by the SM.
// It returns the same msgIndex taken an input for idx output.
func (mvi *MvMultiInitMessage) NeedsSMValidation(msgIndex types.ConsensusIndex, proposalIdx int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	switch {
	case proposalIdx < 0, proposalIdx >= len(mvi.Proposals):
		err = types.ErrInvalidIndex
		return
	}
	switch msgIndex.Index.(type) {
	case types.ConsensusHash:
		if len(mvi.Proposals[proposalIdx]) == 0 {
			panic("should have handled nil proposal earlier")
		}
		return msgIndex, mvi.Proposals[proposalIdx], nil
	default:
		panic("unsupported")
	}
}

// GetSignType returns types.NormalSignature
func (*MvMultiInitMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// ShallowCopy makes a shallow copy of the message
func (mvi *MvMultiInitMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvMultiInitMessage{Proposals: mvi.Proposals}
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mvi *MvMultiInitMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mvi.GetID()}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mvi *MvMultiInitMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddUvarint(uint64(len(mvi.Proposals)))
	if len(mvi.Proposals) == 0 {
		panic("must have at least one proposal")
	}
	var v int
	for _, p := range mvi.Proposals {
		// The proposal
		if len(p) == 0 {
			panic("must not make a nil proposal")
		}
		v, signEndOffset = (*messages.MsgBuffer)(m).AddSizeBytes(p)
		bytesWritten += v
		signEndOffset += v
	}
	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mvi *MvMultiInitMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	var numP uint64
	numP, bytesRead, err = (*messages.MsgBuffer)(m).ReadUvarint()
	switch {
	case numP == 0:
		err = types.ErrNilProposal
		return
	case numP > config.MaxAdditionalIndices:
		err = types.ErrTooManyAdditionalIndices
		return
	}
	mvi.Proposals = make([][]byte, int(numP))
	var v int
	for i := 0; i < int(numP); i++ {
		// Get the proposal
		mvi.Proposals[i], v, err = (*messages.MsgBuffer)(m).ReadSizeBytes()
		bytesRead += v
		if err != nil {
			return
		}
		if len(mvi.Proposals[i]) == 0 {
			err = types.ErrNilProposal
			return
		}
	}

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetID returns the header id for this header
func (mvi *MvMultiInitMessage) GetID() messages.HeaderID {
	return messages.HdrMvMultiInit
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mvi *MvMultiInitMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mvi
}
