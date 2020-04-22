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
	"github.com/tcrain/cons/consensus/auth/coinproof"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

// EdThresh keys threshold keys that should be generated using a
// distributed key generation protocol.
// They support normal EDDSA signatures, plus the use of threshold
// coin proofs for generating random coins.
type EdThresh struct {
	n         int          // the number of participants
	t         int          // the number of signatures needed for the threshold
	secret    kyber.Scalar // the partial secret key for this node
	partPub   *PartPub     // the partial public key for this node
	sharedPub *Edpub       // the threshold public key
	index     sig.PubKeyIndex
}

func NewEdThresh(index sig.PubKeyIndex, dSh *DSSShared) *EdThresh {

	var secret kyber.Scalar
	var public kyber.Point
	if index < sig.PubKeyIndex(dSh.NumParticipants) {
		secret = dSh.MemberScalars[index]
		public = dSh.MemberPoints[index]
	} else {
		secret = dSh.NonMemberScalars[int(index)-dSh.NumParticipants]
		public = dSh.NonMemberPoints[int(index)-dSh.NumParticipants]
	}
	// partPub := sig.EdSuite.Point().Mul(secret, nil)
	edThresh := &EdThresh{
		n:      dSh.NumParticipants,
		t:      dSh.NumThresh,
		secret: secret,
		index:  index}
	edThresh.partPub = NewEdPartPub(index, public, edThresh)
	sp, err := NewEdThresholdPub(dSh.SharePoint, dSh.NumThresh)
	if err != nil {
		panic(err)
	}
	edThresh.sharedPub = sp

	return edThresh
}

func (et *EdThresh) GetT() int {
	return et.t
}

func (et *EdThresh) GetN() int {
	return et.n
}

func (et *EdThresh) GetPartialPub() sig.Pub {
	return et.partPub
}

func (et *EdThresh) GetSharedPub() sig.Pub {
	return et.sharedPub
}

func (et *EdThresh) CheckCoinProof(msg sig.SignedMessage, prf *coinproof.CoinProof) error {
	return prf.Validate(et.partPub.edPub.pub, et.sharedPub.pub, msg.GetSignedMessage())
}

// CombineProofs combines the given proofs and returns the resulting coin values.
// The proofs are expected to have already been validated by CheckCoinProof.
func (et *EdThresh) CombineProofs(items []*sig.SigItem) (coinVal types.BinVal, err error) {
	suite := sig.EdSuite
	var shares []*share.PubShare
	for i := 0; i < et.GetT(); i++ {
		shares = append(shares, &share.PubShare{
			I: int(items[i].Pub.GetIndex()),
			V: items[i].CoinProof.GetShare(),
		})
	}
	var recovered kyber.Point
	recovered, err = share.RecoverCommit(suite, shares, et.GetT(), et.GetN())
	if err != nil {
		return
	}
	h := suite.Hash()
	if _, err = recovered.MarshalTo(h); err != nil {
		return
	}
	res := h.Sum(nil)
	return types.BinVal(res[0] % 2), nil
}
