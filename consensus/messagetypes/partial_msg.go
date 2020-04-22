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
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

/*
Partial messages are implemented as follows:
- A normal consensus message (should be a proposal message) is created as usual, except with no index.
- It is then broken into pieces (TODO implement this as erasure codes, currently this is only a direct slicing).
- Each piece is a PartialMessage object which is used as a normal InternalSignedMsgHeader.
- Additionally each partial message contains the hash of the full message, and a list of hashes of all the partial messages,
- and the consensus round that this message belongs to (this means we can have one partial message per round per process).
- Only this part of the message is signed - thus all partial messages will have the same signature.
- Since each piece is used as a normal IternalSignedMessage the signed part also contains the index of the consensus.
- Once enough pieces are recieved in the message state object, the partial messages are combined, verifying the hashes and signature.
- A CombinedMessage is created which acts as a normal consensus message, using MsgHdrID to distingush it as normal.

- // TODO fix for erasure
*/

// PartialMessage is used by the coordinator of a round of multi-value consensus to send
// its proposal.
// It implements messages.InternalSignedMsgHeader
type PartialMessage struct {
	FullMsgHash types.HashBytes
	// Round is the consensus round, since we only have the index in the normal header, and we need the round for distingusing the proposals
	Round            types.ConsensusRound
	PartialMsgHashes []types.HashBytes
	PartialMsg       []byte
	PartialMsgIndex  int
}

// NewPartialMessage creates a new partial message
func NewPartialMessage() *PartialMessage {
	return &PartialMessage{}
}

// GeneratePartialMessageDirect generates a partial message by evenly splitting it into pieces pieces.
// Round is the round of the proposal message in the consensus to distinguish the message from other rounds of the same index.
func GeneratePartialMessageDirect(hdr messages.InternalSignedMsgHeader,
	round types.ConsensusRound, pieces int) ([]messages.InternalSignedMsgHeader, error) {

	// serialize the InternalSignedMsgHeader
	m := messages.NewMessage(nil)
	l1, _, _, sizeOffset, err := messages.SerializeInternalSignedHeader(types.ConsensusIndex{}, hdr, m)
	if err != nil {
		return nil, err
	}
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l1))
	if err != nil {
		return nil, err
	}
	logging.Error("bytes", m.GetBytes())
	return GeneratePartialMessages(m.GetBytes(), round, utils.SplitBytes(m.GetBytes(), pieces)), nil
}

// GeneratePartialMessage creates a set of partial messages given the full message and its parts
func GeneratePartialMessages(fullMsg []byte, round types.ConsensusRound, partialMessages [][]byte) []messages.InternalSignedMsgHeader {
	fullMsgHash := types.GetHash(fullMsg)
	pmHashes := make([]types.HashBytes, len(partialMessages))
	for i, pm := range partialMessages {
		pmHashes[i] = types.GetHash(pm)
	}
	ret := make([]messages.InternalSignedMsgHeader, len(partialMessages))
	for i, pm := range partialMessages {
		nxt := NewPartialMessage()
		nxt.FullMsgHash = fullMsgHash
		nxt.PartialMsgHashes = pmHashes
		nxt.PartialMsg = pm
		nxt.PartialMsgIndex = i
		nxt.Round = round
		ret[i] = nxt
	}
	return ret
}

// NeedsSMValidation returns nil, since these messages do not need to be validated by the state machine.
func (pm *PartialMessage) NeedsSMValidation(types.ConsensusIndex, int) (idx types.ConsensusIndex,
	proposal []byte, err error) {
	return
}

// GetSignType returns types.NormalSignature
func (pm *PartialMessage) GetSignType() types.SignType {
	return types.NormalSignature
}

// ShallowCopy returns a shallow copy of the message
func (pm *PartialMessage) ShallowCopy() messages.InternalSignedMsgHeader {
	return &PartialMessage{
		FullMsgHash:      pm.FullMsgHash,
		PartialMsgHashes: pm.PartialMsgHashes,
		PartialMsg:       pm.PartialMsg,
		PartialMsgIndex:  pm.PartialMsgIndex,
		Round:            pm.Round}
}

// GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
// TODO
func (pm *PartialMessage) GetMsgID() messages.MsgID {
	return MvMsgID{HdrID: pm.GetID(), Round: pm.Round}
}

func headerPartialSerialze(m *messages.Message, fullMsgHash types.HashBytes, round types.ConsensusRound, partialMsgHashes []types.HashBytes) (bytesWritten, signEndOffset int) {
	var l1 int
	// First the MsgHash
	l1, _ = (*messages.MsgBuffer)(m).AddBytes(fullMsgHash)
	bytesWritten += l1
	if l1 != types.GetHashLen() {
		panic(fmt.Sprint("error writing hash ", l1, types.GetHashLen()))
	}

	// The round
	l1, _ = (*messages.MsgBuffer)(m).AddConsensusRound(round)
	bytesWritten += l1

	// Number of partial hashes
	l1, _ = (*messages.MsgBuffer)(m).AddUint32(uint32(len(partialMsgHashes)))
	bytesWritten += l1

	// Now the partial hashes
	hlen := types.GetHashLen()
	for _, hsh := range partialMsgHashes {
		l1, signEndOffset = (*messages.MsgBuffer)(m).AddBytes(hsh)
		bytesWritten += l1
		if l1 != hlen {
			panic("error writing hash")
		}
	}
	// endoffset is computed as the position at the end of the hashes
	signEndOffset += l1
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (pm *PartialMessage) SerializeInternal(m *messages.Message) (bytesWritten, signEndOffset int, err error) {

	bytesWritten, signEndOffset = headerPartialSerialze(m, pm.FullMsgHash, pm.Round, pm.PartialMsgHashes)
	logging.Error("partial full hashes", bytesWritten, signEndOffset, pm.FullMsgHash, pm.PartialMsgHashes)
	var l1 int
	// Now the partial message index in the hash list (we dont sign the following part of the message)
	l1, _ = (*messages.MsgBuffer)(m).AddUint32(uint32(pm.PartialMsgIndex))
	bytesWritten += l1

	// Now the partial message size
	l1, _ = (*messages.MsgBuffer)(m).AddSizeBytes(pm.PartialMsg)
	bytesWritten += l1

	return
}

func headerPartialDeserialize(m *messages.Message) (fullMsgHash types.HashBytes,
	round types.ConsensusRound, partialMsgHashes []types.HashBytes, bytesRead, signEndOffset int, err error) {

	// First the MsgHash
	hlen := types.GetHashLen()
	fullMsgHash, err = (*messages.MsgBuffer)(m).ReadBytes(hlen)
	bytesRead += hlen
	if err != nil {
		return
	}

	// the round
	var br int
	round, br, err = (*messages.MsgBuffer)(m).ReadConsensusRound()
	bytesRead += br
	if err != nil {
		return
	}
	// Number of hashes
	var nh uint32
	nh, br, err = (*messages.MsgBuffer)(m).ReadUint32()
	bytesRead += br
	if err != nil {
		return
	}

	// the hashes
	var hash types.HashBytes
	for i := 0; i < int(nh); i++ {
		// Get the hash
		hash, err = (*messages.MsgBuffer)(m).ReadBytes(hlen)
		bytesRead += hlen
		if err != nil {
			return
		}
		partialMsgHashes = append(partialMsgHashes, hash)
	}
	// this is the end of the signed part
	signEndOffset = (*messages.MsgBuffer)(m).GetReadOffset()
	return
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (pm *PartialMessage) DeserializeInternal(m *messages.Message) (bytesRead, signEndOffset int, err error) {

	pm.FullMsgHash, pm.Round, pm.PartialMsgHashes, bytesRead, signEndOffset, err = headerPartialDeserialize(m)
	if err != nil {
		return
	}

	// the partial msg index
	var idx uint32
	var br int
	idx, br, err = (*messages.MsgBuffer)(m).ReadUint32()
	bytesRead += br
	if err != nil {
		return
	}
	pm.PartialMsgIndex = int(idx)
	// check the index is valid
	if pm.PartialMsgIndex >= len(pm.PartialMsgHashes) || pm.PartialMsgIndex < 0 {
		err = types.ErrHashIdxOutOfRange
		return
	}

	// the partial msg
	var rlen int
	pm.PartialMsg, rlen, err = (*messages.MsgBuffer)(m).ReadSizeBytes()
	bytesRead += rlen
	if err != nil {
		return
	}

	// Check that the partial message matches the hash
	if !bytes.Equal(pm.PartialMsgHashes[pm.PartialMsgIndex], types.GetHash(pm.PartialMsg)) {
		err = types.ErrInvalidHash
		return
	}
	return

	// TODO
	// // A partial message should only be signed once (TODO maybe certain protocols want it to be signed multiple times?)
	// if len(scm.SigItems) > 1 {
	// 	return 0, utils.ErrInvalidSigType
	// }
}

// GetID returns the header id for this header
func (pm *PartialMessage) GetID() messages.HeaderID {
	return messages.HdrPartialMsg
}

// GetBaseMsgHeader returns the header pertaning to the message contents.
func (pm *PartialMessage) GetBaseMsgHeader() messages.InternalSignedMsgHeader {
	return pm
}

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////
