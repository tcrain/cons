// +build !windows

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

package qsafe

import (
	"fmt"
	"github.com/open-quantum-safe/liboqs-go/oqs"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
)

// QsafeSig is the object representing an qsafe signature
type QsafeSig struct {
	sigBytes   []byte
	algDetails oqs.SignatureDetails
}

// GetMsgID returns the message ID for qsafe sig header
func (sig *QsafeSig) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(sig.GetID())
}

// New creates a new blank qsafe signature object
func (sig *QsafeSig) New() sig.Sig {
	return &QsafeSig{algDetails: sig.algDetails}
}

// PeekHeader returns the indices related to the messages without impacting m.
func (*QsafeSig) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return messages.PeekHeaderHead(types.NilIndexFuns, (*messages.MsgBuffer)(m))
}

// Corrupt invlidates an qsafe signature object
func (sig *QsafeSig) Corrupt() {
	sig.sigBytes[13] += 1
}

func (sig *QsafeSig) Encode(writer io.Writer) (n int, err error) {
	n, err = utils.EncodeUint16(uint16(len(sig.sigBytes)), writer)
	if err != nil {
		return
	}
	var nNxt int
	nNxt, err = writer.Write(sig.sigBytes)
	n += nNxt
	return
}
func (sig *QsafeSig) Decode(reader io.Reader) (n int, err error) {
	var v uint16
	var nNxt int
	v, n, err = utils.ReadUint16(reader)
	// TODO check max size before allocating
	if int(v) > sig.algDetails.MaxLengthSignature {
		return n, fmt.Errorf("signature too many bytes")
	}
	buf := make([]byte, v)
	nNxt, err = reader.Read(buf)
	n += nNxt
	if err != nil {
		return
	}
	if nNxt != len(buf) {
		panic(n)
	}
	sig.sigBytes = buf
	return n, err
}

// Serialize the signature into the message, and return the nuber of bytes written
func (sig *QsafeSig) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, sig.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the sig
	(*messages.MsgBuffer)(m).AddBytes(sig.sigBytes)
	l += len(sig.sigBytes)

	// now update the size
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	// _, l1 := (*messages.MsgBuffer)(m).AddBigInt(sig.r)
	// _, l2 := (*messages.MsgBuffer)(m).AddBigInt(sig.s)
	// err := (*messages.MsgBuffer)(m).WriteUint32At(offset, uint32(l1+l2))
	// if err != nil {
	//	panic(err)
	// }
	return l, nil
}

// GetBytes returns the bytes of the signature from the message
func (sig *QsafeSig) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Derserialize  updates the fields of the qsafe signature object from m, and returns the number of bytes read
func (sig *QsafeSig) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	l, _, _, size, _, err := messages.ReadHeaderHead(sig.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	buff, err := (*messages.MsgBuffer)(m).ReadBytes(int(size) - l)
	if err != nil {
		return 0, err
	}
	sig.sigBytes = buff
	l += len(buff)

	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for qsafe signatures
func (sig *QsafeSig) GetID() messages.HeaderID {
	return messages.HdrQsafeSig
}
