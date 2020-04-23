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

package ec

import (
	"encoding/asn1"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math/big"
)

// Ecsig is the object representing an ECDSA signature
type Ecsig struct {
	R *big.Int
	S *big.Int
}

// GetMsgID returns the message ID for ECDSA sig header
func (sig *Ecsig) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(sig.GetID())
}

// New creates a new blank ECDSA signature object
func (sig *Ecsig) New() sig.Sig {
	return &Ecsig{}
}

// GetRand is unsupported.
func (sig *Ecsig) GetRand() types.BinVal {
	panic("unsupported")
}

// PeekHeader returns the indices related to the messages without impacting m.
func (*Ecsig) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	_ = unmarFunc // we want a nil unmarFunc
	return messages.PeekHeaderHead(types.NilIndexFuns, (*messages.MsgBuffer)(m))
}

// Corrupt invlidates an ECDSA signature object
func (sig *Ecsig) Corrupt() {
	sig.S.Add(big.NewInt(1), sig.S)
}

func (sig *Ecsig) Encode(writer io.Writer) (n int, err error) {
	buf, err := asn1.Marshal(*sig)
	if err != nil {
		return 0, err
	}
	n, err = utils.EncodeUint16(uint16(len(buf)), writer)
	if err != nil {
		return
	}
	var nNxt int
	nNxt, err = writer.Write(buf)
	n += nNxt
	return
}
func (sig *Ecsig) Decode(reader io.Reader) (n int, err error) {
	var v uint16
	var nNxt int
	v, n, err = utils.ReadUint16(reader)
	// TODO check max size before allocating
	if v > 2048 {
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
	_, err = asn1.Unmarshal(buf, sig)
	return n, err
}

// Serialize the signature into the message, and return the nuber of bytes written
func (sig *Ecsig) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, sig.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the sig
	buff, err := asn1.Marshal(*sig)
	if err != nil {
		return 0, err
	}
	(*messages.MsgBuffer)(m).AddBytes(buff)
	l += len(buff)

	// now update the size
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
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
func (sig *Ecsig) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Derserialize  updates the fields of the ECDSA signature object from m, and returns the number of bytes read
func (sig *Ecsig) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(sig.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	buff, err := (*messages.MsgBuffer)(m).ReadBytes(int(size) - l)
	if err != nil {
		return 0, err
	}
	_, err = asn1.Unmarshal(buff, sig)
	if err != nil {
		return 0, err
	}
	l += len(buff)

	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for ECDSA signatures
func (sig *Ecsig) GetID() messages.HeaderID {
	return messages.HdrEcsig
}
