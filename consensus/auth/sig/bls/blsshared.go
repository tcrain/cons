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
	"github.com/tcrain/cons/consensus/logging"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

// BlsShared represents precomputed information used to create
// Bls threshold keys.
type BlsShared struct {
	NumParticipants int
	NumThresh       int
	SharedPub       kyber.Point
	PubPoints       []kyber.Point
	PriScalars      []kyber.Scalar
}

// temporary internal representation
type keyStruct struct {
	pub  kyber.Point
	priv kyber.Scalar
}

// NewBLShared creates a BlsShared object for creating BLS threshold keys.
// It generates private values centrally so is only for testing.
func NewBlsShared(numParticipants, numThresh int) *BlsShared {
	logging.Warning("Generating unsafe shared threshold keys for testing")
	secret := blssuite.G1().Scalar().Pick(blssuite.RandomStream())
	priPoly := share.NewPriPoly(blssuite.G2(), numThresh, secret, blssuite.RandomStream())
	priShares := priPoly.Shares(numParticipants)
	pubPoly := priPoly.Commit(blssuite.G2().Point().Base())
	keys := make([]keyStruct, numParticipants)

	for i := 0; i < numParticipants; i++ {
		// pubPoints[i] = pubPoly.Eval(i).V
		// priScalars[i] = priShares[i].V
		keys[i] = keyStruct{pubPoly.Eval(i).V, priShares[i].V}
	}

	// we cannot sort because indecies must not be changed for the combine function
	// sort by public key
	// sort.Slice(keys, func(i, j int) bool {
	// 	fst, err := keys[i].pub.MarshalBinary()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	snd, err := keys[j].pub.MarshalBinary()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return bytes.Compare(fst, snd) == -1
	// })

	pubPoints := make([]kyber.Point, numParticipants)
	priScalars := make([]kyber.Scalar, numParticipants)
	for i, nxt := range keys {
		pubPoints[i] = nxt.pub
		priScalars[i] = nxt.priv
	}

	return &BlsShared{
		NumParticipants: numParticipants,
		NumThresh:       numThresh,
		PriScalars:      priScalars,
		SharedPub:       pubPoly.Commit(),
		PubPoints:       pubPoints}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////
// Marshalling
////////////////////////////////////////////////////////////////////////////////////////////////

// BlsSharedMarshaled is a partially marshalled version of BlsShared.
// It can then be input to the go json marshaller for example.
type BlsSharedMarshal struct {
	NumParticipants int
	NumThresh       int
	SharedPub       []byte
	PubPoints       [][]byte
	PriScalars      [][]byte
}

// PartialMartial partially martials a BlsShared object into a DSSSharedMarshaled object,
// which can then be mashaled into json for example.
// The index is the index of the secret key to marshall, all others secrets are nil.
func (bs *BlsShared) PartialMarshal() (ret BlsSharedMarshal, err error) {
	ret.PubPoints = make([][]byte, len(bs.PubPoints))
	ret.PriScalars = make([][]byte, len(bs.PriScalars))
	for i := 0; i < bs.NumParticipants; i++ {
		ret.PubPoints[i], err = bs.PubPoints[i].MarshalBinary()
		if err != nil {
			return
		}
		ret.PriScalars[i], err = bs.PriScalars[i].MarshalBinary()
		if err != nil {
			return
		}
	}
	ret.SharedPub, err = bs.SharedPub.MarshalBinary()
	if err != nil {
		return
	}
	ret.NumParticipants = bs.NumParticipants
	ret.NumThresh = bs.NumThresh
	return
}

// PartialUnmartial takes a BlsSharedMarshalled object, unmarshals it, and returns a BlsShared object.
func (bsm BlsSharedMarshal) PartialUnmarshal() (ret *BlsShared, err error) {
	ret = &BlsShared{}
	ret.PubPoints = make([]kyber.Point, len(bsm.PubPoints))
	ret.PriScalars = make([]kyber.Scalar, len(bsm.PubPoints))
	for i := 0; i < bsm.NumParticipants; i++ {
		ret.PubPoints[i] = blssuite.G2().Point()
		err = ret.PubPoints[i].UnmarshalBinary(bsm.PubPoints[i])
		if err != nil {
			return
		}
		ret.PriScalars[i] = blssuite.G2().Scalar()
		err = ret.PriScalars[i].UnmarshalBinary(bsm.PriScalars[i])
		if err != nil {
			return
		}
	}
	ret.SharedPub = blssuite.G2().Point()
	err = ret.SharedPub.UnmarshalBinary(bsm.SharedPub)
	if err != nil {
		return
	}
	ret.NumParticipants = bsm.NumParticipants
	ret.NumThresh = bsm.NumThresh
	return
}
