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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/coinproof"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"io"
	"time"
)

type PartPub struct {
	edPub *Edpub
	*EdThresh
}

func (pub *PartPub) gets() sig.CoinProofPubInterface {
	return pub
}

func (pub *PartPub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *PartPub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// ProofToHash is for validating VRFs and not supported for ed.
func (pub *PartPub) ProofToHash(sig.SignedMessage, sig.VRFProof) (index [32]byte, err error) {
	panic("unsupported")
}

// NewVRFProof returns an empty VRFProof object, and is not supported for ed.
func (pub *PartPub) NewVRFProof() sig.VRFProof {
	panic("unsupported")
}

func (pub *PartPub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	p, err := pub.edPub.FromPubBytes(b)
	if err != nil {
		return nil, err
	}
	return &PartPub{edPub: p.(*Edpub)}, nil
}

func (pub *PartPub) SetIndex(index sig.PubKeyIndex) {
	if pub.index != index {
		panic("should not change index for partial pub")
	}
	pub.edPub.SetIndex(index)
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *PartPub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.index
}

func NewEdPartPub(index sig.PubKeyIndex, point kyber.Point, edThresh *EdThresh) *PartPub {
	// NewEdpub(partPub)
	if index != edThresh.index {
		panic("invalid index")
	}
	edPub, err := NewSchnorrpub(point)
	if err != nil {
		panic(err)
	}
	edPub.SetIndex(index)
	return &PartPub{
		edPub:    edPub,
		EdThresh: edThresh}
}

func (pub *PartPub) GetSigMemberNumber() int {
	return 1
}

var edPartialVerifyTime = 671000 * time.Nanosecond

func (pub *PartPub) VerifySig(msg sig.SignedMessage, asig sig.Sig) (bool, error) {
	return pub.edPub.VerifySig(msg, asig)
}

// CheckSignature validates the signature with the public key.
// If the message type is a coin message, it is verified using CoinProof
func (pub *PartPub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	var err error
	signType := msg.GetSignType()
	if signType == types.CoinProof { // sanity check
		if sigItem.CoinProof == nil {
			panic("should have coin proof")
		}
	} else if sigItem.CoinProof != nil {
		panic("should not have coin proof")
	}
	if sigItem.CoinProof != nil { // Check if the coin proof is valid
		if thrsh, ok := sigItem.Pub.(sig.CoinProofPubInterface); ok {
			if err = thrsh.CheckCoinProof(msg, sigItem.CoinProof); err != nil {
				return err
			}
		} else {
			panic("should have caught this earlier")
		}
		return nil
	}

	valid, err := pub.VerifySig(msg, sigItem.Sig)
	if err != nil {
		return err
	}
	if !valid {
		return types.ErrInvalidSig
	}
	return nil
}

func (pub *PartPub) GetPubID() (sig.PubKeyID, error) {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.edPub.GetPubID()
}

func (pub *PartPub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.GetPubBytes()
}

func (pub *PartPub) GetPubBytes() (sig.PubKeyBytes, error) {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.edPub.GetPubBytes()
}

func (pub *PartPub) GetPubString() (sig.PubKeyStr, error) {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.edPub.GetPubString()
}

func (pub *PartPub) New() sig.Pub {
	return &PartPub{EdThresh: nil, edPub: pub.edPub.New().(*Edpub)}
}
func (pub *PartPub) DeserializeCoinProof(m *messages.Message) (coinProof *coinproof.CoinProof, size int, err error) {
	coinProof = coinproof.EmptyCoinProof(sig.EdSuite)
	size, err = coinProof.Deserialize(m, types.NilIndexFuns)
	return
}

func (pub *PartPub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.CoinProof {
		var n int
		var l1 int
		var err error
		newPub := pub.New()
		l1, err = newPub.Deserialize(m, types.NilIndexFuns)
		if err != nil {
			return nil, n, err
		}
		n += l1
		var coinProof *coinproof.CoinProof
		if coinProof, l1, err = pub.DeserializeCoinProof(m); err != nil {
			return nil, l1, err
		}
		n += l1
		return &sig.SigItem{Pub: newPub, CoinProof: coinProof}, n, nil
	} else {
		return pub.edPub.DeserializeSig(m, signType)
	}
}

func (pub *PartPub) Encode(io.Writer) (n int, err error) {
	panic("unused")
}

func (pub *PartPub) Decode(io.Reader) (n int, err error) {
	panic("unused")
}

func (pub *PartPub) Serialize(m *messages.Message) (int, error) {
	return pub.edPub.Serialize(m)
}

// PeekHeader returns nil.
func (PartPub) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}
func (pub *PartPub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	return pub.edPub.Deserialize(m, unmarFunc)
}
func (pub *PartPub) GetBytes(*messages.Message) ([]byte, error) {
	panic("unused")
}
func (pub *PartPub) GetID() messages.HeaderID {
	return messages.HdrSchnorrpub
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////

// EdPartialPriv represents the ECDSA private key object.
type PartialPriv struct {
	*EdThresh
	edPriv *Edpriv
}

// Clean does nothing
func (priv *PartialPriv) Clean() {
}

// NewSig returns an empty sig object of the same type.
func (priv *PartialPriv) NewSig() sig.Sig {
	return &Edsig{}
}

// SetIndex sets the index of the node represented by this key in the consensus participants.
func (priv *PartialPriv) SetIndex(index sig.PubKeyIndex) {
	if priv.index != index || priv.edPriv.index != index || priv.edPriv.pub.index != index {
		panic(fmt.Sprintf("index for threshold keys set during creation, expected %v, set %v", priv.index, index))
	}
}

// Evaluate is for generating VRFs and not supported for ed.
func (priv *PartialPriv) Evaluate(sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	panic("unsupported")
}

// GetBaseKey returns the key as a normal Schnorr private key.
func (priv *PartialPriv) GetBaseKey() sig.Priv {
	return NewSchnorrprivFrom(priv.secret)
}

// New creates an empty ECDSA private key object.
func (priv *PartialPriv) New() sig.Priv {
	return &PartialPriv{}
}

// GetPub returns the coreesponding ECDSA public key object.
func (priv *PartialPriv) GetPub() sig.Pub {
	return priv.GetPartialPub()
}

// GetEdThresh returns the EdThresh object.
func (priv *PartialPriv) GetEdThresh() *EdThresh {
	return priv.EdThresh
}

// NewEdPartialPriv creates a new partial priv given the thrsh structure.
func NewEdPartPriv(thrsh *EdThresh) (sig.Priv, error) {
	p := NewSchnorrprivFrom(thrsh.secret).(*Edpriv)
	p.SetIndex(thrsh.index)
	return &PartialPriv{EdThresh: thrsh,
		edPriv: p}, nil
}

// ComputeSharedSecret is unsupported.
func (priv *PartialPriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	return priv.edPriv.ComputeSharedSecret(pub.(*PartPub).edPub)
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *PartialPriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.sharedPub = priv.sharedPub.ShallowCopy().(*Edpub)
	newPriv.partPub = priv.partPub.ShallowCopy().(*PartPub)
	return &newPriv
}

// Sign signs a message and returns the signature.
func (priv *PartialPriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	return priv.edPriv.Sign(msg)
}

// Returns key that is used for signing the sign type.
func (priv *PartialPriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return priv, nil
}

// GenerateSig signs a message and returns the SigItem object containing the signature.
func (priv *PartialPriv) GenerateSig(header sig.SignedMessage, proof sig.VRFProof,
	signType types.SignType) (*sig.SigItem, error) {

	if proof != nil {
		panic("vrf not supported by ED")
	}
	if signType != types.CoinProof { // just a normal ed sig
		return priv.edPriv.GenerateSig(header, proof, signType)
	}
	var coinProof *coinproof.CoinProof
	var err error
	m := messages.NewMessage(nil)
	_, err = priv.GetPub().Serialize(m) // priv.SerializePub(m)
	if err != nil {
		return nil, err
	}
	coinProof, err = coinproof.CreateCoinProof(sig.EdSuite, priv.sharedPub.pub, header.GetSignedMessage(), priv.secret)
	if err != nil {
		return nil, err
	}
	if _, err = coinProof.Serialize(m); err != nil {
		return nil, err
	}
	return &sig.SigItem{
		Pub:       priv.edPriv.GetPub(),
		Sig:       nil,
		CoinProof: coinProof,
		SigBytes:  m.GetBytes()}, nil
}
