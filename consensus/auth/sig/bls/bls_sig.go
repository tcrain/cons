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

package bls

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/random"
	"io"
)

// Blssig represent a bls signature object
type Blssig struct {
	sig      kyber.Point // the signature is a point on the ecCurve
	sigBytes []byte      // the serialized signature
}

// adds or subtracts two signatures
func makeBlsSig(isSub bool, sig1, sig2 *Blssig) (*Blssig, error) {
	s1 := sig1.sig.Clone()
	s2 := sig2.sig
	if isSub {
		s1 = s1.Sub(s1, s2)
	} else {
		s1 = s1.Add(s1, s2)
	}
	sigByt, err := s1.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &Blssig{s1, sigByt}, nil
}

// MergeSig combines two signatures, it assumes the sigs are valid to be merged
func (sig1 *Blssig) MergeSig(sig2 sig.MultiSig) (sig.MultiSig, error) {
	return makeBlsSig(false, sig1, sig2.(*Blssig))
}

// SubSig removes sig2 from sig1, it assumes sig 1 already contains sig2
func (sig1 *Blssig) SubSig(sig2 sig.MultiSig) (sig.MultiSig, error) {
	return makeBlsSig(true, sig1, sig2.(*Blssig))
}

// GetMsgID returns the message id for a bls signature message
func (sig *Blssig) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(sig.GetID())
}

// GetRand returns a random binary from the signature if supported.
func (sig *Blssig) GetRand() types.BinVal {
	// TODO this is only for coin using threshold signatures, should cleanup
	return types.BinVal(types.GetHash(sig.sigBytes)[0] % 2)
}

// GetSigBytes returns the bytes of the serialized signature
func (sig *Blssig) GetSigBytes() []byte {
	return sig.sigBytes
}

// New creates an empty sig object
func (sig *Blssig) New() sig.Sig {
	return &Blssig{}
}

// Corrupt invalidates the signature
func (sig *Blssig) Corrupt() {
	sig.sig = sig.sig.Pick(random.New())
	var err error
	sig.sigBytes, err = sig.sig.MarshalBinary()
	if err != nil {
		panic(err)
	}
}

func (sig *Blssig) Encode(writer io.Writer) (n int, err error) {
	if len(sig.sigBytes) != blssigmarshalsize {
		return 0, types.ErrInvalidMsgSize
	}
	return writer.Write(sig.sigBytes)
}
func (sig *Blssig) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, blssigmarshalsize)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	sig.sigBytes = buf

	s2 := blssuite.G1().Point()
	if err = s2.UnmarshalBinary(buf); err != nil {
		return
	}
	sig.sig = s2
	return
}

// Serialize marshalls the signature and adds the bytes to the message
// returns the number of bytes written and any error
func (sig *Blssig) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, sig.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the sig
	(*messages.MsgBuffer)(m).AddBytes(sig.sigBytes)
	l += len(sig.sigBytes)

	// now update the size
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

// GetBytes takes the serialized bls signature in a message and returns the signautre as a byte slice
func (sig *Blssig) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// PeekHeader returns nil.
func (Blssig) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// Deserialize takes a message and fills in the fields of the sig oject from the bytes of the message
// it returns the number of bytes read
func (sig *Blssig) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
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

	s2 := blssuite.G1().Point()
	if err := s2.UnmarshalBinary(buff); err != nil {
		return 0, err
	}
	sig.sig = s2

	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for the BLS sig message
func (sig *Blssig) GetID() messages.HeaderID {
	return messages.HdrBlssig
}
