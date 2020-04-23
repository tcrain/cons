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

package sig

import (
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// var testStr = "this is a test string"
// var testVal uint32 = 32

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

type MultiSignTestMsg struct {
	Round  uint32
	BinVal types.BinVal
}

func NewMultiSignTestMsg(pub Pub) *MultiSignTestMsg {
	return &MultiSignTestMsg{}
}

func (*MultiSignTestMsg) GetSignType() types.SignType {
	return types.NormalSignature
}

func (scm *MultiSignTestMsg) ShallowCopy() messages.InternalSignedMsgHeader {
	return &MultiSignTestMsg{scm.Round, scm.BinVal}
}

func (scm *MultiSignTestMsg) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(scm.GetID())
}

func (scm *MultiSignTestMsg) NeedsSMValidation(msgIndex types.ConsensusIndex, proposalIdx int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

func (scm *MultiSignTestMsg) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {
	// Now the round
	bytesWritten, _ = (*messages.MsgBuffer)(m).AddUint32(scm.Round)

	// Now the bin value
	signEndOffset = (*messages.MsgBuffer)(m).AddBin(scm.BinVal, false)
	bytesWritten++
	signEndOffset++

	// End of signed message
	return
}

func (scm *MultiSignTestMsg) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {
	// Get the round
	scm.Round, bytesRead, err = (*messages.MsgBuffer)(m).ReadUint32()
	if err != nil {
		return
	}

	// Get the bin val
	scm.BinVal, err = (*messages.MsgBuffer)(m).ReadBin(false)
	bytesRead++
	if err != nil {
		return
	}
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()

	return
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (scm *MultiSignTestMsg) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return scm
}

func (scm *MultiSignTestMsg) GetID() messages.HeaderID {
	// use any header since we just use this message for testing
	return messages.HdrAuxProof
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// type SignTestMsg struct {
// 	SignedMessage
// 	Proposal []byte
// }

// func NewSignTestMsg(pub Pub, priv Priv) *SignTestMsg {
// 	return &SignTestMsg{
// 		SignedMessage: SignedMessage{
// 			Pub: pub.New(),
// 			Priv: priv}}
// }

// func (scm *SignTestMsg) Serialize(m *messages.Message) (int, error) {
// 	l, sizeOffset, offset := messages.WriteInitialHeader(scm.GetID(), 0, (*messages.MsgBuffer)(m))

// 	// Size of the proposal
// 	l1, _ := (*messages.MsgBuffer)(m).AddUint32(uint32(len(scm.Proposal)))
// 	l += l1

// 	// Now the proposal
// 	l1, endoffset := (*messages.MsgBuffer)(m).AddBytes(scm.Proposal)
// 	if l1 != len(scm.Proposal) {
// 		panic("error writing proposal")
// 	}

// 	l += l1
// 	endoffset += l1

// 	// End of signed message
// 	return scm.Sign(m, l, offset, endoffset, sizeOffset)
// }

// func (scm *SignTestMsg) GetBytes(m *messages.Message) ([]byte, error) {
// 	size, err := (*messages.MsgBuffer)(m).PeekUint32()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
// }

// func (scm *SignTestMsg) Deserialize(m *messages.Message) (int, error) {
// 	l, size, offset, err := messages.ReadInitialHeader(scm.GetID(), (*messages.MsgBuffer)(m))
// 	if err != nil {
// 		return 0, err
// 	}
// 	// Length of the proposal
// 	plen, err := (*messages.MsgBuffer)(m).ReadUint32()
// 	if err != nil {
// 		return 0, err
// 	}
// 	l += 4

// 	// Get the proposal
// 	proposal, err := (*messages.MsgBuffer)(m).ReadBytes(int(plen))
// 	if err != nil {
// 		return 0, err
// 	}
// 	scm.Proposal = proposal
// 	l+= int(plen)
// 	endoffset := (*messages.MsgBuffer)(m).GetReadOffset()

// 	return scm.DeserSign(m, l, offset, endoffset, size)
// }

// func (scm *SignTestMsg) GetID() messages.HeaderID {
// 	return messages.HdrMvInit
// }

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////
