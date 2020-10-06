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

// MvEchoMessage is used during multi-value consensus to echo the MvInitMessage by all participants,
// plus an additional hash.
// It implements messages.MsgHeader
type MvEchoHashMessage struct {
	ProposalHash types.HashBytes // The hash of the proposal being echo'd
	RandHash     types.HashBytes
	Round        types.ConsensusRound
}

// NewMvEchoMessage creates a new mv echo message
func NewMvEchoHashMessage() *MvEchoHashMessage {
	return &MvEchoHashMessage{}
}

func (mve *MvEchoHashMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MvEchoHashMessage{ProposalHash: mve.ProposalHash,
		Round:    mve.Round,
		RandHash: mve.RandHash}
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (mve *MvEchoHashMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// GetSignType return types.NormalSignature
func (*MvEchoHashMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// GetMsgID returns the MsgID for this specific header, in this case a MvMsgID (see MsgID definition for more details)
func (mve *MvEchoHashMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: mve.GetID(), Round: mve.Round}
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (mve *MvEchoHashMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the proposal hash
	if len(mve.ProposalHash) != types.GetHashLen() {
		panic(types.ErrInvalidHash)
	}
	var v int
	v, err = (*messages.MsgBuffer)(m).Write(mve.ProposalHash)
	if err != nil {
		return
	}
	bytesWritten += v

	if len(mve.RandHash) != types.GetHashLen() {
		panic(types.ErrInvalidHash)
	}
	v, err = (*messages.MsgBuffer)(m).Write(mve.RandHash)
	if err != nil {
		return
	}
	bytesWritten += v

	v, signEndOffset = (*messages.MsgBuffer)(m).AddConsensusRound(mve.Round)
	bytesWritten += v
	signEndOffset += v

	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (mve *MvEchoHashMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the proposal hash
	hashLen := types.GetHashLen()
	mve.ProposalHash, err = (*messages.MsgBuffer)(m).ReadBytes(hashLen)
	if err != nil {
		return
	}
	bytesRead += hashLen
	mve.RandHash, err = (*messages.MsgBuffer)(m).ReadBytes(hashLen)
	if err != nil {
		return
	}
	bytesRead += hashLen

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
func (mve *MvEchoHashMessage) GetID() messages.HeaderID {
	return messages.HdrMvEchoHash
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (mve *MvEchoHashMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return mve
}
