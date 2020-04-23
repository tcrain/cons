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

// MvCommitMessage is used during multi-value consensus to Commit the MvInitMessage by all participants.
// It implements messages.MsgHeader
type MvCommitMessage struct {
	ProposalHash types.HashBytes // The hash of the proposal being Commit'd
	Round        types.ConsensusRound
}

// NewMvCommitMessage creates a new mv Commit message
func NewMvCommitMessage() *MvCommitMessage {
	return &MvCommitMessage{}
}

func (mvc *MvCommitMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvCommitMessage{ProposalHash: mvc.ProposalHash, Round: mvc.Round}
}

// NeedsCoinProof returns false.
func (mvc *MvCommitMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (mvc *MvCommitMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mvc *MvCommitMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mvc.GetID(), Round: mvc.Round}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mvc *MvCommitMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the proposal hash
	if len(mvc.ProposalHash) != types.GetHashLen() {
		err = messages.ErrInvalidHash
		return
	}
	v, _ := (*messages.MsgBuffer)(m).AddBytes(mvc.ProposalHash)
	if v != len(mvc.ProposalHash) {
		panic("error writing proposal hash")
	}
	bytesWritten += v

	v, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(mvc.Round)
	bytesWritten += v
	signEndOffset += v

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mvc *MvCommitMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal hash
	bytesRead = types.GetHashLen()
	mvc.ProposalHash, err = (*messages.MsgBuffer)(m).ReadBytes(bytesRead)
	if err != nil {
		return
	}

	// the round
	var br int
	mvc.Round, br, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	bytesRead += br
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (mvc *MvCommitMessage) GetID() messages.HeaderID {
	return messages.HdrMvCommit
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mvc *MvCommitMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mvc
}

/////////////////////////////////////////////////////////////////////////////////////////////////
// Timeout message
/////////////////////////////////////////////////////////////////////////////////////////////////

// MvCommitMessageTimeout is sent internally to indicate that timeout has passed
// for a round while waiting for MvCommitMessage
type MvCommitMessageTimeout types.ConsensusRound

// PeekHeaders is unused since it is a local message.
func (MvCommitMessageTimeout) PeekHeaders(*messages.Message,
	types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	panic("unused")
}

// GetMsgID is unused since it is a local message.
func (tm MvCommitMessageTimeout) GetMsgID() messages.MsgID {
	panic("unused")
}

// Serialize is unused since it is a local message.
func (tm MvCommitMessageTimeout) Serialize(*messages.Message) (int, error) {
	panic("unused")
}

// GetBytes is unused since it is a local message.
func (tm MvCommitMessageTimeout) GetBytes(*messages.Message) ([]byte, error) {
	panic("unused")
}

// Deserialize is unused since it is a local message.
func (tm MvCommitMessageTimeout) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {
	panic("unused")
}

// GetID returns the header id for this header.
func (tm MvCommitMessageTimeout) GetID() messages.HeaderID {
	return messages.HdrMvCommitTimeout
}
