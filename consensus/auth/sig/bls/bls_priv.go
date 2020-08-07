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
	"bytes"
	"crypto/sha512"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/random"
	"golang.org/x/crypto/blake2b"
)

///////////////////////////////////////////////////////////////////////////////////////
// Private key
///////////////////////////////////////////////////////////////////////////////////////

// Blspriv represents a BLS private key object
type Blspriv struct {
	priv kyber.Scalar // The scalar on the ecCurve representing the private key
	pub  *Blspub      // The corresponding public key object
	//index sig.PubKeyIndex      // The index of this key in the sorted list of public keys participating in a consensus
	//bitID bitid.BitIDInterface // The BitID representation of index
}

// Clean does nothing
func (priv *Blspriv) Clean() {
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (priv *Blspriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	var blsPub *Blspub
	switch v := pub.(type) {
	case *Blspub:
		blsPub = v
	case *PartPub:
		blsPub = &v.Blspub
	default:
		panic(v)
	}
	pt := blssuite.Point().Mul(priv.priv, blsPub.newPub)
	if sig.BlsMultiNew {
		pt = blssuite.Point().Mul(priv.pub.hpk, pt)
	}
	byt, err := pt.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return sha512.Sum512_256(append(byt, config.InitRandBytes[:]...))
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *Blspriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.pub = priv.pub.ShallowCopy().(*Blspub)
	return &newPriv
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (priv *Blspriv) SetIndex(index sig.PubKeyIndex) {
	priv.pub.SetIndex(index)
}

// Evaluate is for generating VRFs.
func (priv *Blspriv) Evaluate(m sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	s, err := priv.doSign(m)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBuffer(nil)
	if _, err := s.Encode(buff); err != nil {
		panic(err)
	}
	proof = VRFProof(buff.Bytes())
	hf, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	hf.Write(proof.(VRFProof))
	hf.Sum(index[:0])
	return
}

// Returns key that is used for signing the sign type.
func (priv *Blspriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	if signType == types.CoinProof {
		return nil, types.ErrCoinProofNotSupported
	}
	return priv, nil
}

// GetBaseKey returns the same key.
func (priv *Blspriv) GetBaseKey() sig.Priv {
	return priv
}

// New creates an empty BLS private key object
func (priv *Blspriv) New() sig.Priv {
	return &Blspriv{}
}

// GetPub returns the coresponding BLS public key object
func (priv *Blspriv) GetPub() sig.Pub {
	return priv.pub
}

func NewBlsprivFrom(secret kyber.Scalar) sig.Priv {
	var hpk kyber.Scalar
	var pub, newPub kyber.Point
	if sig.BlsMultiNew {
		_, pub, newPub, hpk = NewBLSMSKeyPairFrom(secret, blssuite, random.New())
	} else {
		pub = blssuite.G2().Point().Mul(secret, nil)
		newPub = pub
	}
	return &Blspriv{
		priv: secret,
		pub: &Blspub{
			pub:    pub,
			newPub: newPub,
			hpk:    hpk,
		},
	}

}

// NewBlspriv creates a new random BLS private key
func NewBlspriv() (sig.Priv, error) {
	var priv, hpk kyber.Scalar
	var pub, newPub kyber.Point
	if sig.BlsMultiNew {
		priv, pub, newPub, hpk = NewBLSMSKeyPair(blssuite, random.New())
	} else {
		priv, pub = NewBLSKeyPair(blssuite, random.New())
		newPub = pub
	}
	return &Blspriv{
		priv: priv,
		pub: &Blspub{
			pub:    pub,
			newPub: newPub,
			hpk:    hpk,
		},
	}, nil
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *Blspriv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof, signType types.SignType) (*sig.SigItem, error) {
	return sig.GenerateSigHelper(priv, header, true, vrfProof, signType)
}

// NewSig returns an empty sig object of the same type.
func (priv *Blspriv) NewSig() sig.Sig {
	return &Blssig{}
}

// Sign signs a message and returns the signature.
func (priv *Blspriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	return priv.doSign(msg)
}

func (priv *Blspriv) doSign(msg sig.SignedMessage) (sig.Sig, error) {
	var asig kyber.Point
	var b []byte
	var err error
	if sig.BlsMultiNew {
		asig, b, err = SignBLSMS(blssuite, priv.priv, priv.pub.hpk, msg.GetSignedMessage())
	} else {
		asig, b, err = SignBLS(blssuite, priv.priv, msg.GetSignedMessage())
	}
	if err != nil {
		return nil, err
	}
	return &Blssig{sig: asig, sigBytes: b}, nil
}
