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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/consensus/messages"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

type BlsThrsh struct {
	n      int             // the number of participants
	t      int             // the number of signatures needed for the threshold
	secret kyber.Scalar    // the partial secret key for this node
	idx    sig.PubKeyIndex // the index of this node in the list of sorted pub keys

	partPub   *PartPub   // the partial public key for this node
	sharedPub *SharedPub // the threshold public key
}

// NewBlsThrsh creates an object for a given member of a signature threshold scheme.
func NewBlsThrsh(n, t int, idx sig.PubKeyIndex, secret kyber.Scalar, pub kyber.Point, sharedPub kyber.Point) *BlsThrsh {
	if sig.GetUseMultisig() || sig.GetBlsMultiNew() || !sig.GetUsePubIndex() {
		panic(fmt.Sprintf("For bls thrsh must not use multi sig or bls multi new and must use put index, useMultiSig: %v, BlsMultiNew: %v, UsePubIndex %v",
			sig.GetUseMultisig(), sig.GetBlsMultiNew(), sig.GetUsePubIndex()))
	}
	sPub := NewSharedPub(sharedPub, t) // &SharedPub{Blspub: Blspub{pub: sharedPub, newPub: sharedPub}, memberCount: t}
	thrsh := &BlsThrsh{
		n:         n,
		t:         t,
		secret:    secret,
		idx:       idx,
		sharedPub: sPub}
	partPub := NewBlsPartPub(idx, pub)
	thrsh.partPub = partPub
	return thrsh
}

// NewBlsThrsh creates an object for a signature threshold scheme. The object created can combine signatures,
// but cannot generate its own partial signature as it is not a member of the scheme.
func NewNonMemberBlsThrsh(n, t int, idx sig.PubKeyIndex, sharedPub kyber.Point) *BlsThrsh {
	if sig.GetUseMultisig() || sig.GetBlsMultiNew() || !sig.GetUsePubIndex() {
		panic(fmt.Sprintf("For bls thrsh must not use multi sig or bls multi new and must use put index, useMultiSig: %v, BlsMultiNew: %v, UsePubIndex %v",
			sig.GetUseMultisig(), sig.GetBlsMultiNew(), sig.GetUsePubIndex()))
	}
	// Since we are not a member, we just make a normal BLS key
	sec, pub := NewBLSKeyPair(blssuite, blssuite.RandomStream())

	sPub := NewSharedPub(sharedPub, t) // &SharedPub{Blspub: Blspub{pub: sharedPub, newPub: sharedPub}, memberCount: t}
	thrsh := &BlsThrsh{
		n:         n,
		t:         t,
		secret:    sec,
		idx:       idx,
		sharedPub: sPub}
	partPub := NewBlsPartPub(idx, pub)
	thrsh.partPub = partPub
	return thrsh
}

// PartialSign creates a signature on the message that can also be used to create a shared signature.
func (bt *BlsThrsh) PartialSign(msg sig.SignedMessage) (sig.Sig, error) {
	sig, b, err := SignBLS(blssuite, bt.secret, msg.GetSignedMessage())
	if err != nil {
		return nil, err
	}
	return &PartSig{Blssig: Blssig{sig: sig, sigBytes: b}, idx: bt.idx}, nil
}

// CombinePartialSigs generates a shared signature from the list of partial signatures.
func (bt *BlsThrsh) CombinePartialSigs(ps []sig.Sig) (*sig.SigItem, error) {
	if len(ps) < bt.t {
		return nil, types.ErrNotEnoughPartials
	}
	pubShares := make([]*share.PubShare, bt.t)

	for i := 0; i < bt.t; i++ {
		switch v := ps[i].(type) {
		case *PartSig:
			pubShares[i] = &share.PubShare{I: int(v.idx), V: v.sig}
		default:
			return nil, types.ErrInvalidSigType
		}
	}
	commit, err := share.RecoverCommit(blssuite.G1(), pubShares, bt.t, bt.n)
	if err != nil {
		return nil, err
	}
	buff, err := commit.MarshalBinary()
	if err != nil {
		return nil, err
	}
	si := &Blssig{commit, buff}
	m := messages.NewMessage(nil)
	// When using the BLS shared threshold key, we don't include the serialized public
	// key with the signature as there is only one threshold key
	// first we write a 0 to indicate no VRFProof
	err = (*messages.MsgBuffer)(m).WriteByte(0)
	if err != nil {
		return nil, err
	}
	_, err = si.Serialize(m)
	if err != nil {
		return nil, err
	}
	pid, err := bt.sharedPub.GetPubID()
	if err != nil {
		panic(err)
	}
	n, err := (*messages.MsgBuffer)(m).Write([]byte(pid))
	if err != nil {
		panic(err)
	}
	if n != sharedIDBytes {
		panic(n)
	}
	return &sig.SigItem{
		Pub:      bt.sharedPub,
		Sig:      si,
		SigBytes: m.GetBytes()}, nil
}

func (bt *BlsThrsh) GetT() int {
	return bt.t
}

func (bt *BlsThrsh) GetN() int {
	return bt.n
}

func (bt *BlsThrsh) GetPartialPub() sig.Pub {
	return bt.partPub
}

func (bt *BlsThrsh) GetSharedPub() sig.Pub {
	return bt.sharedPub
}

func VerifyPartialSig(msg []byte, pubInt sig.Pub, sigInt sig.Sig) error {
	pp, b := pubInt.(*PartPub)
	if !b {
		return types.ErrInvalidPub
	}
	ps, b := sigInt.(*PartSig)
	if !b {
		return types.ErrInvalidSigType
	}

	return VerifyBLS(blssuite, pp.pub, msg, ps.sig)
}
