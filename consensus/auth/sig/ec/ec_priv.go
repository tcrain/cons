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
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

///////////////////////////////////////////////////////////////////////////////////////
// Private key
///////////////////////////////////////////////////////////////////////////////////////

// Ecpriv represents the ECDSA private key object
type Ecpriv struct {
	// priv  *ecdsa.PrivateKey // The private key object
	priv  *p256.PrivateKey
	pub   *Ecpub          // The public key object
	index sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
}

// Clean does nothing
func (priv *Ecpriv) Clean() {
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (priv *Ecpriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	ecPub := pub.(*Ecpub)
	x, y := ecCurve.ScalarMult(ecPub.pub.X, ecPub.pub.Y, priv.priv.D.Bytes())
	return sha512.Sum512_256(bytes.Join([][]byte{x.Bytes(), y.Bytes(), config.InitRandBytes[:]}, nil))
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *Ecpriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.pub = newPriv.pub.ShallowCopy().(*Ecpub)
	return &newPriv
}

// NewSig returns an empty sig object of the same type.
func (priv *Ecpriv) NewSig() sig.Sig {
	return &Ecsig{}
}

// GetBaseKey returns the same key.
func (priv *Ecpriv) GetBaseKey() sig.Priv {
	return priv
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (priv *Ecpriv) SetIndex(index sig.PubKeyIndex) {
	priv.index = index
	priv.pub.SetIndex(index)
}

// New creates an empty ECDSA private key object
func (priv *Ecpriv) New() sig.Priv {
	return &Ecpriv{}
}

// GetPub returns the coreesponding ECDSA public key object
func (priv *Ecpriv) GetPub() sig.Pub {
	return priv.pub
}

// Evaluate is for generating VRFs.
func (priv *Ecpriv) Evaluate(m sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	var prf []byte
	index, prf = priv.priv.Evaluate(m.GetSignedMessage())
	return index, VRFProof(prf)
}

// NewEcpriv creates a new random ECDSA private key object
func NewEcpriv() (sig.Priv, error) {
	priv, pub := p256.GenerateKey()
	return &Ecpriv{
		priv: priv.(*p256.PrivateKey),
		pub: &Ecpub{
			pub: pub.(*p256.PublicKey)},
	}, nil
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *Ecpriv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof, signType types.SignType) (*sig.SigItem, error) {
	return sig.GenerateSigHelper(priv, header, vrfProof, signType)
}

// Returns key that is used for signing the sign type.
func (priv *Ecpriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	if signType == types.CoinProof {
		return nil, types.ErrCoinProofNotSupported
	}
	return priv, nil
}

// Sign signs a message and returns the signature.
func (priv *Ecpriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	hash := msg.GetSignedHash()
	if len(hash) != 32 {
		return nil, types.ErrInvalidHashSize
	}
	r, s, err := ecdsa.Sign(rand.Reader, priv.priv.PrivateKey, hash)
	if err != nil {
		return nil, err
	}

	return &Ecsig{R: r, S: s}, nil
}

/////////////// Remove below since we dont need to serialize the private key //////////////////

func (priv *Ecpriv) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(priv.GetID())
}

func (priv *Ecpriv) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, priv.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the pub
	l1, err := priv.pub.Serialize(m)
	if err != nil {
		return 0, err
	}
	l += l1
	// now the priv
	l1, _ = (*messages.MsgBuffer)(m).AddBigInt(priv.priv.D)
	l += l1
	// update the size
	err = (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}

	return l, nil
}

func (priv *Ecpriv) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// PeekHeader returns nil.
func (Ecpriv) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

func (priv *Ecpriv) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(priv.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	priv.pub = &Ecpub{}
	l1, err := priv.pub.Deserialize(m, types.NilIndexFuns)
	if err != nil {
		return 0, err
	}
	l += l1
	d, l1, err := (*messages.MsgBuffer)(m).ReadBigInt()
	if err != nil {
		return 0, err
	}
	l += l1
	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}
	priv.priv = &p256.PrivateKey{PrivateKey: &ecdsa.PrivateKey{}}
	pub, err := priv.pub.getPub()
	if err != nil {
		return l, err
	}
	priv.priv.PublicKey = *pub
	priv.priv.D = d
	return l, err
}

func (priv *Ecpriv) GetID() messages.HeaderID {
	return messages.HdrEcpriv
}
