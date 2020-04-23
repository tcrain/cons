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

// MvEchoMessage is used during multi-value consensus to echo the MvInitMessage by all participants.
// It implements messages.MsgHeader
type MvEchoMessage struct {
	ProposalHash types.HashBytes // The hash of the proposal being echo'd
	Round        types.ConsensusRound
}

// NewMvEchoMessage creates a new mv echo message
func NewMvEchoMessage() *MvEchoMessage {
	return &MvEchoMessage{}
}

func (mve *MvEchoMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvEchoMessage{ProposalHash: mve.ProposalHash, Round: mve.Round}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (mve *MvEchoMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// GetSignType return types.NormalSignature
func (*MvEchoMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mve *MvEchoMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mve.GetID(), Round: mve.Round}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mve *MvEchoMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the proposal hash
	if len(mve.ProposalHash) != types.GetHashLen() {
		err = messages.ErrInvalidHash
		return
	}
	v, _ := (*messages.MsgBuffer)(m).AddBytes(mve.ProposalHash)
	if v != len(mve.ProposalHash) {
		panic("error writing proposal hash")
	}
	bytesWritten += v

	v, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(mve.Round)
	bytesWritten += v
	signEndOffset += v

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mve *MvEchoMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal hash
	bytesRead = types.GetHashLen()
	mve.ProposalHash, err = (*messages.MsgBuffer)(m).ReadBytes(bytesRead)
	if err != nil {
		return
	}

	// the round
	var br int
	mve.Round, br, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	if err != nil {
		return
	}
	bytesRead += br
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetID returns the header id for this header
func (mve *MvEchoMessage) GetID() messages.HeaderID {
	return messages.HdrMvEcho
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mve *MvEchoMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mve
}

/////////////////////////////////////////////////////////////////////////////////////////////////
// Timeout message
/////////////////////////////////////////////////////////////////////////////////////////////////

// MvEchoMessageTimeout is sent internally to indicate that timeout has passed
// for a round while waiting for MvEchoMessage
type MvEchoMessageTimeout types.ConsensusRound

// PeekHeaders is unused since it is a local message.
func (MvEchoMessageTimeout) PeekHeaders(*messages.Message,
	types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	panic("unused")
}

// GetMsgID is unused since it is a local message.
func (tm MvEchoMessageTimeout) GetMsgID() messages.MsgID {
	panic("unused")
}

// Serialize is unused since it is a local message.
func (tm MvEchoMessageTimeout) Serialize(*messages.Message) (int, error) {
	panic("unused")
}

// GetBytes is unused since it is a local message.
func (tm MvEchoMessageTimeout) GetBytes(*messages.Message) ([]byte, error) {
	panic("unused")
}

// Deserialize is unused since it is a local message.
func (tm MvEchoMessageTimeout) Deserialize(*messages.Message, types.ConsensusIndexFuncs) (int, error) {

	panic("unused")
}

// GetID returns the header id for this header.
func (tm MvEchoMessageTimeout) GetID() messages.HeaderID {
	return messages.HdrMvEchoTimeout
}
