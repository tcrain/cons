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

package ed

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
)

const edSigSize = 64

// Edsig is the object representing an EDDSA signature
type Edsig struct {
	sig       []byte
	sigEdtype edtype
}

// NewEdSig returns an empty EDDSA signature object
func NewEdSig() *Edsig {
	return &Edsig{sigEdtype: eddsaType}
}

// NewEdSig returns an empty schnorr signature object
func NewSchnorrSig() *Edsig {
	return &Edsig{sigEdtype: schnorrType}
}

// GetMsgID returns the message ID for EDDSA sig header
func (sig *Edsig) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(sig.GetID())
}

// New returns a empty EDDSA signature object of the same type as sig
func (sig *Edsig) New() sig.Sig {
	return &Edsig{sigEdtype: sig.sigEdtype}
}

// Corrupt invalidates the signature
func (sig *Edsig) Corrupt() {
	index := len(sig.sig) / 2
	sig.sig[index]++
}

func (sig *Edsig) Encode(writer io.Writer) (n int, err error) {
	return writer.Write(sig.sig)
}
func (sig *Edsig) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, edSigSize)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	sig.sig = buf

	return
}

// GetRand returns a random binary from the signature if supported.
func (sig *Edsig) GetRand() types.BinVal {
	// TODO is this correct?
	binVal := types.BinVal(types.GetHash(sig.sig)[0] % 2)
	return binVal
}

// Serialize the signature into the message, and return the nuber of bytes written
func (sig *Edsig) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, sig.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the sig
	(*messages.MsgBuffer)(m).AddBytes(sig.sig)
	l += len(sig.sig)

	// now update the size
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

func (sig *Edsig) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// PeekHeader returns nil.
func (Edsig) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// GetBytes returns the bytes of the signature from the message
func (sig *Edsig) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmar func
	l, _, _, size, _, err := messages.ReadHeaderHead(sig.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	buff, err := (*messages.MsgBuffer)(m).ReadBytes(int(size) - l)
	if err != nil {
		return 0, err
	}
	l += len(buff)
	sig.sig = buff

	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for the EDDSA signature object type
func (sig *Edsig) GetID() messages.HeaderID {
	switch sig.sigEdtype {
	case eddsaType:
		return messages.HdrEdsig
	case schnorrType:
		return messages.HdrSchnorrsig
	default:
		panic("invalid pub type")
	}
}
