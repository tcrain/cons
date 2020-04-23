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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/coinproof"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"go.dedis.ch/kyber/v3"
	"math"
)

// Blspriv represents a BLS private key object
type PartPriv struct {
	*BlsThrsh
	blsPriv *Blspriv
}

// Clean does nothing
func (priv *PartPriv) Clean() {
}

// Shallow copy makes a copy of the object without following pointers.
func (priv *PartPriv) ShallowCopy() sig.Priv {
	newPriv := *priv
	newPriv.partPub = priv.partPub.ShallowCopy().(*PartPub)
	newPriv.sharedPub = priv.sharedPub.ShallowCopy().(*SharedPub)
	return &newPriv
}

func NewBlsPartPriv(thrsh *BlsThrsh) (sig.Priv, error) {
	ret := &PartPriv{BlsThrsh: thrsh}
	ret.blsPriv = NewBlsprivFrom(ret.secret).(*Blspriv)
	return ret, nil
}

// NewSig returns an empty sig object of the same type.
func (priv *PartPriv) NewSig() sig.Sig {
	return &PartSig{}
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (priv *PartPriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	return priv.blsPriv.ComputeSharedSecret(pub)
}

func (priv *PartPriv) Evaluate(m sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	if priv.blsPriv == nil {
		priv.blsPriv = NewBlsprivFrom(priv.secret).(*Blspriv)
	}
	return priv.blsPriv.Evaluate(m)
}

func (priv *PartPriv) SetIndex(idx sig.PubKeyIndex) {
	if idx != priv.idx {
		panic(fmt.Sprintf("index for threshold keys set during creation, expected %v, set %v", priv.idx, idx))
	}
}

// GetBaseKey returns threshold key as a normal BSL key.
func (priv *PartPriv) GetBaseKey() sig.Priv {
	return NewBlsprivFrom(priv.secret)
}

// New creates an empty BLS private key object
func (priv *PartPriv) New() sig.Priv {
	return &PartPriv{BlsThrsh: priv.BlsThrsh}
}

// Returns key that is used for signing the sign type.
func (priv *PartPriv) GetPrivForSignType(types.SignType) (sig.Priv, error) {
	return priv, nil
}

// GetPub returns the coresponding BLS public key object
func (priv *PartPriv) GetPub() sig.Pub {
	return priv.GetPartialPub()
}

// Sign signs a message and returns the signature.
func (priv *PartPriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	return priv.PartialSign(msg)
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *PartPriv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof,
	_ types.SignType) (*sig.SigItem, error) {

	m := messages.NewMessage(nil)
	if err := sig.CheckSerVRF(vrfProof, m); err != nil {
		return nil, err
	}
	_, err := priv.GetPartialPub().Serialize(m) // priv.SerializePub(m)
	if err != nil {
		return nil, err
	}
	si, err := priv.PartialSign(header)
	if err != nil {
		return nil, err
	}
	_, err = si.Serialize(m)
	if err != nil {
		return nil, err
	}

	return &sig.SigItem{
		VRFProof: vrfProof,
		Pub:      priv.GetPub(),
		Sig:      si,
		SigBytes: m.GetBytes()}, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////

const sharedIDBytes = 2 + 9
const sharedIDString = "blsshared"

type SharedPub struct {
	Blspub
	memberCount int
	id          sig.PubKeyID
}

func NewSharedPub(point kyber.Point, memberCount int) *SharedPub {
	buff := bytes.NewBuffer(nil)
	if memberCount > math.MaxUint16 {
		panic("member count too large")
	}
	_, err := utils.EncodeUint16(uint16(memberCount), buff)
	if err != nil {
		panic(err)
	}
	if _, err = buff.WriteString(sharedIDString); err != nil {
		panic(err)
	}
	if buff.Len() != sharedIDBytes {
		panic("error encoding int")
	}
	return &SharedPub{
		Blspub:      Blspub{pub: point, newPub: point},
		memberCount: memberCount,
		id:          sig.PubKeyID(buff.Bytes()),
	}
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *SharedPub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	valid, err := pub.Blspub.VerifySig(msg, sigItem.Sig)
	if err != nil {
		return err
	}
	if !valid {
		return types.ErrInvalidSig
	}
	return nil
}

// GetPubID returns the id for this pubkey.
// Given that there is only one threshold pub per consensus it returns PubKeyID("blssharedpub").
func (pub *SharedPub) GetPubID() (sig.PubKeyID, error) {
	return pub.id, nil
}

// GetSigMemberNumber returns the number of nodes represented by this BLS pub key
func (pub *SharedPub) GetSigMemberNumber() int {
	return pub.memberCount
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *SharedPub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////

type PartPub struct {
	Blspub
	idx sig.PubKeyIndex
}

// Shallow copy makes a copy of the object without following pointers.
func (bpp *PartPub) ShallowCopy() sig.Pub {
	newPub := *bpp
	return &newPub
}

// func (pub *BlsPartPub) FromPubBytes(b PubKeyBytes) (Pub, error) {
// 	newPub, err := pub.Blspub.FromPubBytes(b)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BlsPartPub{
// 		Blspub: newPub,
// 		idx:
// }

func NewBlsPartPub(idx sig.PubKeyIndex, p kyber.Point) *PartPub {
	pub := Blspub{pub: p, newPub: p}
	pub.SetIndex(idx)
	return &PartPub{
		Blspub: pub,
		idx:    idx}
}

// New generates an empty Blspub object
func (bpp *PartPub) New() sig.Pub {
	return &PartPub{}
}

func (bpp *PartPub) SetIndex(index sig.PubKeyIndex) {
	bpp.idx = index
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (bpp *PartPub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return bpp.idx
}

// CheckSignature validates the partial threshold signature with the public key, it returns an error if a coin proof is included.
// Coin messages are verified as partial threshold signatures.
func (bpp *PartPub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	if sigItem.CoinProof != nil {
		return types.ErrCoinProofNotSupported
	}
	valid, err := bpp.VerifySig(msg, sigItem.Sig)
	if err != nil {
		return err
	}
	if !valid {
		return types.ErrInvalidSig
	}
	return nil
}

func (bpp *PartPub) VerifySig(msg sig.SignedMessage, asig sig.Sig) (bool, error) {
	switch v := asig.(type) {
	case *PartSig:
		v.idx = bpp.idx
		if sig.SleepValidate {
			sig.DoSleepValidation(blsVerifyTime)
			return true, nil
		}
		if err := VerifyPartialSig(msg.GetSignedMessage(), bpp, asig); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, types.ErrInvalidSigType
	}
}

func (bpp *PartPub) DeserializeCoinProof(_ *messages.Message) (coinProof *coinproof.CoinProof, size int, err error) {
	panic("unsupported")
}

// DeserializeSig takes a message and returns a BLS public key object and partial signature object as well as the number of bytes read
func (bpp *PartPub) DeserializeSig(m *messages.Message, _ types.SignType) (*sig.SigItem, int, error) {
	ret, l, err := sig.DeserVRF(bpp, m)
	if err != nil {
		return nil, l, err
	}
	ht, err := m.PeekHeaderType()
	if err != nil {
		return nil, l, err
	}
	switch ht {
	case messages.HdrBlssig: // this was signed by the shared threshold key
		// When using the BLS shared threshold key, we don't include the serialized public
		// key with the signature as there is only one threshold key
		ret.Sig = &Blssig{}
		l1, err := ret.Sig.Deserialize(m, types.NilIndexFuns)
		if err != nil {
			return ret, l, err
		}
		l += l1
		// The id is the number
		id, err := (*messages.MsgBuffer)(m).ReadBytes(sharedIDBytes)
		if err != nil {
			return ret, l, err
		}
		l += sharedIDBytes
		ret.Pub = &SharedPub{id: sig.PubKeyID(id)}
		return ret, l, err
	case messages.HdrBlspub:
		ret.Pub = bpp.New()
		l1, err := ret.Pub.Deserialize(m, types.NilIndexFuns)
		l += l1
		if err != nil {
			return nil, l, err
		}

		ret.Sig = &PartSig{}
		l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
		l += l1
		return ret, l, err
	default:
		return nil, 0, types.ErrInvalidHeader
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////

type PartSig struct {
	Blssig
	idx sig.PubKeyIndex
}

// New creates an empty sig object
func (sig *PartSig) New() sig.Sig {
	return &PartSig{}
}
