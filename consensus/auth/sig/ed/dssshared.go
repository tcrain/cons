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
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/logging"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

// CoinShared represents precomputed information used to create
// threshold secrets for generating random coins.
type CoinShared struct {
	NumParticipants  int
	NumThresh        int
	MemberScalars    []kyber.Scalar
	MemberPoints     []kyber.Point
	NonMemberScalars []kyber.Scalar
	NonMemberPoints  []kyber.Point
	SharePoint       kyber.Point
}

// NewCoinShared creates a CoinShared object for creating threshold coin keys.
// It generates private values centrally so is only for testing.
func NewCoinShared(numTotalNodes, numNonMembers, numThresh int) *CoinShared {
	logging.Warning("Generating unsafe shared threshold keys for testing")
	nbParticipants := numTotalNodes - numNonMembers
	suite := sig.EdSuite

	secret := suite.Scalar().Pick(suite.RandomStream())
	sharedPub := suite.Point().Mul(secret, suite.Point().Base())
	priPoly := share.NewPriPoly(suite, numThresh, secret, suite.RandomStream())
	priPolyShares := priPoly.Shares(nbParticipants)
	pubPoly := priPoly.Commit(suite.Point().Base()).Shares(nbParticipants)

	memberScalars := make([]kyber.Scalar, nbParticipants)
	memberPoints := make([]kyber.Point, nbParticipants)

	for _, nxt := range priPolyShares {
		memberScalars[nxt.I] = nxt.V
	}
	for _, nxt := range pubPoly {
		memberPoints[nxt.I] = nxt.V
	}

	nonMemberScalars := make([]kyber.Scalar, numNonMembers)
	nonMemberPoints := make([]kyber.Point, numNonMembers)
	for i := range nonMemberScalars {
		nonMemberScalars[i] = suite.Scalar().Pick(suite.RandomStream())
		nonMemberPoints[i] = suite.Point().Mul(nonMemberScalars[i], suite.Point().Base())
	}

	return &CoinShared{
		NumParticipants:  nbParticipants,
		NumThresh:        numThresh,
		SharePoint:       sharedPub,
		MemberScalars:    memberScalars,
		MemberPoints:     memberPoints,
		NonMemberScalars: nonMemberScalars,
		NonMemberPoints:  nonMemberPoints}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////
// Marshalling
////////////////////////////////////////////////////////////////////////////////////////////////

// CoinSharedMarshaled is a partially marshalled version of CoinShared.
// It can then be input to the go json marshaller for example.
// It doesn't include the PartSec field of CoinShared.
type CoinSharedMarshaled struct {
	NumParticipants  int
	NumThresh        int
	MemberScalars    [][]byte
	MemberPoints     [][]byte
	NonMemberScalars [][]byte
	NonMemberPoints  [][]byte
	SharePoint       []byte
}

// PartialUnmartial takes a CoinSharedMarshaled object, unmarshals it, and returns a CoinShared object.
func (dsm CoinSharedMarshaled) PartialUnMartial() (ret *CoinShared, err error) {
	suite := sig.EdSuite
	ret = &CoinShared{}
	ret.NumParticipants = dsm.NumParticipants
	ret.NumThresh = dsm.NumThresh

	ret.MemberPoints = make([]kyber.Point, len(dsm.MemberPoints))
	for i, nxt := range dsm.MemberPoints {
		ret.MemberPoints[i] = suite.Point()
		if _, err := ret.MemberPoints[i].UnmarshalFrom(bytes.NewBuffer(nxt)); err != nil {
			return nil, err
		}
	}

	ret.MemberScalars = make([]kyber.Scalar, len(dsm.MemberScalars))
	for i, nxt := range dsm.MemberScalars {
		ret.MemberScalars[i] = suite.Scalar()
		if _, err := ret.MemberScalars[i].UnmarshalFrom(bytes.NewBuffer(nxt)); err != nil {
			return nil, err
		}
	}

	ret.NonMemberPoints = make([]kyber.Point, len(dsm.NonMemberPoints))
	for i, nxt := range dsm.NonMemberPoints {
		ret.NonMemberPoints[i] = suite.Point()
		if _, err := ret.NonMemberPoints[i].UnmarshalFrom(bytes.NewBuffer(nxt)); err != nil {
			return nil, err
		}
	}

	ret.NonMemberScalars = make([]kyber.Scalar, len(dsm.NonMemberScalars))
	for i, nxt := range dsm.NonMemberScalars {
		ret.NonMemberScalars[i] = suite.Scalar()
		if _, err := ret.NonMemberScalars[i].UnmarshalFrom(bytes.NewBuffer(nxt)); err != nil {
			return nil, err
		}
	}

	ret.SharePoint = suite.Point()
	if _, err := ret.SharePoint.UnmarshalFrom(bytes.NewBuffer(dsm.SharePoint)); err != nil {
		return nil, err
	}

	return
}

// PartialMartial partially martials a CoinShared object into a CoinSharedMarshaled object,
// which can then be mashaled into json for example.
func (ds *CoinShared) PartialMarshal() (ret CoinSharedMarshaled, err error) {
	ret.NumParticipants = ds.NumParticipants
	ret.NumThresh = ds.NumThresh

	ret.MemberScalars = make([][]byte, len(ds.MemberScalars))
	for i, nxt := range ds.MemberScalars {
		ret.MemberScalars[i], err = nxt.MarshalBinary()
		if err != nil {
			return
		}
	}
	ret.MemberPoints = make([][]byte, len(ds.MemberPoints))
	for i, nxt := range ds.MemberPoints {
		ret.MemberPoints[i], err = nxt.MarshalBinary()
		if err != nil {
			return
		}
	}
	ret.NonMemberScalars = make([][]byte, len(ds.NonMemberScalars))
	for i, nxt := range ds.NonMemberScalars {
		ret.NonMemberScalars[i], err = nxt.MarshalBinary()
		if err != nil {
			return
		}
	}
	ret.NonMemberPoints = make([][]byte, len(ds.NonMemberPoints))
	for i, nxt := range ds.NonMemberPoints {
		ret.NonMemberPoints[i], err = nxt.MarshalBinary()
		if err != nil {
			return
		}
	}

	ret.SharePoint, err = ds.SharePoint.MarshalBinary()
	if err != nil {
		return
	}

	return
}
