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
	"crypto/sha512"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/sign/eddsa"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/kyber/v3/util/random"
)

// edtype is the type of eddsa signature to use,
// if using edwards25519 group they can both be verified using eddsa, but
// schnorr creates a new random for each signature
type edtype int

const (
	eddsaType   edtype = iota
	schnorrType        // if using the edwards25519 group, can be verified using eddsa
)

///////////////////////////////////////////////////////////////////////////////////////
// Private keys
///////////////////////////////////////////////////////////////////////////////////////

// EDpriv represents the EDDSA private key object
type Edpriv struct {
	priv        *eddsa.EdDSA    // The private key object
	privSchnorr kyber.Scalar    // The private key object when schnorr signatures are used
	pub         *Edpub          // The public key object
	index       sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
	privEdtype  edtype          // The type of signature
}

// Clean does nothing
func (priv *Edpriv) Clean() {
}

// Returns key that is used for signing the sign type.
func (priv *Edpriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	if signType == types.CoinProof {
		return nil, types.ErrCoinProofNotSupported
	}
	return priv, nil
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (priv *Edpriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	switch priv.privEdtype {
	case schnorrType:
		byt, err := sig.EdSuite.Point().Mul(priv.privSchnorr, pub.(*Edpub).pub).MarshalBinary()
		if err != nil {
			panic(err)
		}
		return sha512.Sum512_256(append(byt, config.InitRandBytes[:]...))
	case eddsaType:
		byt, err := sig.EddsaGroup.Point().Mul(priv.priv.Secret, pub.(*Edpub).pub).MarshalBinary()
		if err != nil {
			panic(err)
		}
		return sha512.Sum512_256(append(byt, config.InitRandBytes[:]...))
	default:
		panic(priv.privEdtype)
	}
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *Edpriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.pub = priv.pub.ShallowCopy().(*Edpub)
	return &newPriv
}

// NewSig returns an empty sig object of the same type.
func (priv *Edpriv) NewSig() sig.Sig {
	return &Edsig{}
}

// New creates an empty EDDSA private key object
func (priv *Edpriv) New() sig.Priv {
	return &Edpriv{privEdtype: priv.privEdtype}
}

// GetBaseKey returns the same key.
func (priv *Edpriv) GetBaseKey() sig.Priv {
	return priv
}

// Evaluate is for generating VRFs and not supported for ed.
func (priv *Edpriv) Evaluate(m sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	panic("unsupported")
}

// GetPub returns the coreesponding EDDSA public key object
func (priv *Edpriv) GetPub() sig.Pub {
	return priv.pub
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (priv *Edpriv) SetIndex(index sig.PubKeyIndex) {
	priv.index = index
	priv.pub.SetIndex(index)
}

// NewEcpriv creates a new random EDDSA private key object
func NewEdpriv() (sig.Priv, error) {
	priv := eddsa.NewEdDSA(random.New())
	return &Edpriv{
		privEdtype: eddsaType,
		priv:       priv,
		pub: &Edpub{
			useIndex: sig.UsePubIndex,
			pub:      priv.Public,
		},
	}, nil
}

// NewEcpriv creates a new random schnorr private key object
func NewSchnorrpriv() (sig.Priv, error) {
	priv := key.NewKeyPair(sig.EdSuite)
	p, err := NewSchnorrpub(priv.Public)
	if err != nil {
		return nil, err
	}
	return &Edpriv{
		privEdtype:  schnorrType,
		privSchnorr: priv.Private,
		pub:         p,
	}, nil
}

func NewSchnorrprivFrom(secret kyber.Scalar) sig.Priv {
	pb, err := NewSchnorrpub(sig.EdSuite.Point().Mul(secret, nil))
	if err != nil {
		panic(err)
	}
	return &Edpriv{
		privEdtype:  schnorrType,
		privSchnorr: secret,
		pub:         pb,
	}
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *Edpriv) GenerateSig(header sig.SignedMessage, proof sig.VRFProof, signType types.SignType) (*sig.SigItem, error) {
	if proof != nil {
		panic("vrf not supported by ED")
	}
	m := messages.NewMessage(nil)
	if signType == types.CoinProof {
		panic("coin proof only supported for ")
	}

	_, err := priv.GetPub().Serialize(m) // priv.SerializePub(m)
	if err != nil {
		return nil, err
	}
	si, err := priv.Sign(header)
	if err != nil {
		return nil, err
	}
	_, err = si.Serialize(m)
	if err != nil {
		return nil, err
	}

	return &sig.SigItem{
		Pub:      priv.GetPub(),
		Sig:      si,
		SigBytes: m.GetBytes()}, nil
}

// Sign signs a message and returns the signature.
func (priv *Edpriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	var asig []byte
	var err error
	switch priv.privEdtype {
	case eddsaType:
		asig, err = priv.priv.Sign(msg.GetSignedMessage())
	case schnorrType:
		asig, err = schnorr.Sign(sig.EdSuite, priv.privSchnorr, msg.GetSignedMessage())
	}
	if err != nil {
		return nil, err
	}
	return &Edsig{sig: asig, sigEdtype: priv.privEdtype}, nil
}
