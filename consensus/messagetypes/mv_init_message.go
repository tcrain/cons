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

// MvMsgID implements messages.MsgID.
// It is used to differentiate between multi-value consensus messages of different types rounds by comparing them, for example, during
// consensus we can count n-t different MvEchoMessages of each round (even if the MvEchoMessages have different
// echo values, the MvMsgID should be equal for the same round).
type MvMsgID struct {
	HdrID messages.HeaderID
	Round types.ConsensusRound
}

// IsMsgID to satisfy the interface and returns true
func (MvMsgID) IsMsgID() bool {
	return true
}

func (bm MvMsgID) ToBytes(index types.ConsensusIndex) []byte {
	m := messages.NewMsgBuffer()
	m.AddConsensusID(index.Index)
	m.AddHeaderID(bm.HdrID)
	m.AddConsensusRound(bm.Round)
	return m.GetRemainingBytes()
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// MvInitMessage is used by the coordinator of a round of multi-value consensus to send
// its proposal.
// It implements messages.MsgHeader
type MvInitMessage struct {
	Proposal    []byte // The proposal
	ByzProposal []byte // Alternative proposal for byzantine nodes
	Round       types.ConsensusRound
}

// NewMvInitMessage creates a new mv init message
func NewMvInitMessage() *MvInitMessage {
	return &MvInitMessage{}
}

// NeedsSMValidation returns the proposal bytes to be validated by the SM.
// It takes as input the current index and returns the index -1.
// It panics if proposalIdx is not 0.
func (mvi *MvInitMessage) NeedsSMValidation(msgIndex types.ConsensusIndex, proposalIdx int) (idx types.ConsensusIndex,
	proposal []byte, err error) {

	if len(mvi.Proposal) == 0 {
		panic("should have handled nil proposal earlier")
	}

	switch idx := msgIndex.Index.(type) {
	case types.ConsensusInt:
		if proposalIdx != 0 {
			panic("proposal idx mut be 0 for single init proposals")
		}
		newIdx := msgIndex.ShallowCopy()
		newIdx.Index, newIdx.FirstIndex = idx-1, idx-1
		return newIdx, mvi.Proposal, nil
	default:
		return types.ConsensusIndex{}, mvi.Proposal, nil
	}
}

// GetSignType returns types.NormalSignature
func (*MvInitMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// ShallowCopy makes a shallow copy of the message
func (mvi *MvInitMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvInitMessage{Proposal: mvi.Proposal, Round: mvi.Round}
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mvi *MvInitMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mvi.GetID(), Round: mvi.Round}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mvi *MvInitMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// The proposal
	if len(mvi.Proposal) == 0 {
		panic("must not make a nil proposal")
	}
	v, _ := (*messages.MsgBuffer)(m).AddSizeBytes(mvi.Proposal)
	bytesWritten += v

	// The round
	v, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(mvi.Round)
	bytesWritten += v
	signEndOffset += v

	// End of signed message
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mvi *MvInitMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal
	mvi.Proposal, bytesRead, err = (*messages.MsgBuffer)(m).ReadSizeBytes()
	if err != nil {
		return
	}

	if len(mvi.Proposal) == 0 {
		err = types.ErrNilProposal
		return
	}

	// Get the round
	var br int
	mvi.Round, br, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	bytesRead += br

	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// GetID returns the header id for this header
func (mvi *MvInitMessage) GetID() messages.HeaderID {
	return messages.HdrMvInit
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mvi *MvInitMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mvi
}

/////////////////////////////////////////////////////////////////////////////////////////////////
// Timeout message
/////////////////////////////////////////////////////////////////////////////////////////////////

// MvInitMessageTimeout is sent internally to indicate that timeout has passed
// for a round while waiting for MvInitMessage
type MvInitMessageTimeout types.ConsensusRound

// PeekHeaders is unused since it is a local message.
func (MvInitMessageTimeout) PeekHeaders(*messages.Message,
	types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	panic("unused")
}

// GetMsgID is unused since it is a local message.
func (tm MvInitMessageTimeout) GetMsgID() messages.MsgID {
	panic("unused")
}

// Serialize is unused since it is a local message.
func (tm MvInitMessageTimeout) Serialize(*messages.Message) (int, error) {
	panic("unused")
}

// GetBytes is unused since it is a local message.
func (tm MvInitMessageTimeout) GetBytes(*messages.Message) ([]byte, error) {
	panic("unused")
}

// Deserialize is unused since it is a local message.
func (tm MvInitMessageTimeout) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	panic("unused")
}

// GetID returns the header id for this header.
func (tm MvInitMessageTimeout) GetID() messages.HeaderID {
	return messages.HdrMvInitTimeout
}
