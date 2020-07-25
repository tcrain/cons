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
	"github.com/open-quantum-safe/liboqs-go/oqs"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
)

const QsafeName = "DEFAULT"

///////////////////////////////////////////////////////////////////////////////////////
// Private key
///////////////////////////////////////////////////////////////////////////////////////

// QsafePriv represents the ECDSA private key object
type QsafePriv struct {
	// priv  *ecdsa.PrivateKey // The private key object
	priv  oqs.Signature
	pub   *QsafePub       // The public key object
	index sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (priv *QsafePriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	// use the KEM module, but need to generate custom keys for this
	panic("TODO")
}

// Clean garbage collects the c objects.
func (priv *QsafePriv) Clean() {
	priv.priv.Clean()
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *QsafePriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.pub = newPriv.pub.ShallowCopy().(*QsafePub)
	return &newPriv
}

// NewSig returns an empty sig object of the same type.
func (priv *QsafePriv) NewSig() sig.Sig {
	return &QsafeSig{algDetails: priv.priv.Details()}
}

// GetBaseKey returns the same key.
func (priv *QsafePriv) GetBaseKey() sig.Priv {
	return priv
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (priv *QsafePriv) SetIndex(index sig.PubKeyIndex) {
	priv.index = index
	priv.pub.SetIndex(index)
}

// New creates an empty ECDSA private key object
func (sig *QsafePriv) New() sig.Priv {
	return &QsafePriv{}
}

// GetPub returns the coreesponding ECDSA public key object
func (priv *QsafePriv) GetPub() sig.Pub {
	return priv.pub
}

// NewQsafePriv creates a new random ECDSA private key object
func NewQsafePriv() (sig.Priv, error) {
	var priv oqs.Signature
	priv.Init(QsafeName, nil)
	pubBytes, err := priv.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	return &QsafePriv{
		priv: priv,
		pub: &QsafePub{
			algDetails: priv.Details(),
			pubBytes:   pubBytes,
		}}, nil
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *QsafePriv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof, signType types.SignType) (*sig.SigItem, error) {
	return sig.GenerateSigHelper(priv, header, vrfProof, signType)
}

// Sign signs a message and returns the signature.
func (priv *QsafePriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	sigBytes, err := priv.priv.Sign(msg.GetSignedMessage())
	if err != nil {
		return nil, err
	}
	return &QsafeSig{sigBytes: sigBytes, algDetails: priv.priv.Details()}, nil
}

// Returns key that is used for signing the sign type.
func (priv *QsafePriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	if signType == types.CoinProof {
		return nil, types.ErrCoinProofNotSupported
	}
	return priv, nil
}
