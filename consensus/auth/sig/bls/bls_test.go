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
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
	// "sort"
	// "fmt"

	"go.dedis.ch/kyber/v3/pairing/bn256"
	// "go.dedis.ch/kyber/v3/pairing"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/util/random"
)

func TestBLSMarshal(t *testing.T) {
	priv, pub := NewBLSKeyPair(blssuite, random.New())
	privByte, err := priv.MarshalBinary()
	assert.Nil(t, err)

	// priv2 := priv.Clone()
	priv2 := blssuite.G2().Scalar()
	err = priv2.UnmarshalBinary(privByte)
	assert.Nil(t, err)

	pubByte, err := pub.MarshalBinary()
	assert.Nil(t, err)

	// pub2 := pub.Clone()
	pub2 := blssuite.G2().Point()
	err = pub2.UnmarshalBinary(pubByte)
	assert.Nil(t, err)
}

var sigCount = 10

// include all pubs in the hash
func TestBLS3(t *testing.T) {
	msg := []byte("testmsg")
	suite := bn256.NewSuite()

	pubList := make([]kyber.Point, sigCount)
	oldPubList := make([]kyber.Point, sigCount)
	privList := make([]kyber.Scalar, sigCount)
	sigList := make([]kyber.Point, sigCount)

	for i := 0; i < sigCount; i++ {
		private, public := bls.NewKeyPair(suite, random.New())

		oldPubList[i] = public
		privList[i] = private
	}

	// var newPubs []kyber.Point
	for i := 0; i < sigCount; i++ {
		hpk, err := hashScalar(suite, oldPubList[i], oldPubList...)
		assert.Nil(t, err)
		newPk := oldPubList[i].Clone().Mul(hpk, oldPubList[i])

		// create the sig
		asig, err := bls.Sign(suite, privList[i], msg)
		assert.Nil(t, err)
		err = bls.Verify(suite, oldPubList[i], msg, asig)
		assert.Nil(t, err)

		s := suite.G1().Point()
		err = s.UnmarshalBinary(asig)
		assert.Nil(t, err)

		newSig := s.Mul(hpk, s)

		msig, err := newSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, newPk, msg, msig)
		assert.Nil(t, err)

		pubList[i] = newPk
		sigList[i] = newSig
	}

	msPub := pubList[0].Clone()
	msSig := sigList[0].Clone()
	sigbyt, err := sigList[0].MarshalBinary()
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, sigbyt)
	assert.Nil(t, err)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Add(pubList[i], msPub)
		msSig = msSig.Add(sigList[i], msSig)

		msSigByt, err := msSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)
	}

	msSigByt, err := msSig.MarshalBinary()
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, msSigByt)
	assert.Nil(t, err)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Sub(msPub, pubList[i])
		msSig = msSig.Sub(msSig, sigList[i])

		msSigByt, err = msSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)
	}

	pubByt, err := pubList[0].MarshalBinary()
	assert.Nil(t, err)
	msPubByt, err := msPub.MarshalBinary()
	assert.Nil(t, err)

	assert.Equal(t, pubByt, msPubByt)

	err = bls.Verify(suite, pubList[0], msg, msSigByt)
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, sigbyt)
	assert.Nil(t, err)

}

// multi-sig based on https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
func TestBLS2(t *testing.T) {
	msg := []byte("testmsg")
	suite := bn256.NewSuite()

	var pubList []kyber.Point
	// var privList []kyber.Scalar
	var sigList []kyber.Point

	for i := 0; i < sigCount; i++ {
		private, public := bls.NewKeyPair(suite, random.New())
		asig, err := bls.Sign(suite, private, msg)
		assert.Nil(t, err)
		err = bls.Verify(suite, public, msg, asig)
		assert.Nil(t, err)

		s := suite.G1().Point()
		err = s.UnmarshalBinary(asig)
		assert.Nil(t, err)

		// create the new public key
		hpk, err := hashScalar(suite, public)
		assert.Nil(t, err)

		newPk := public.Mul(hpk, public)

		// create the new sig
		newSig := s.Mul(hpk, s)

		msig, err := newSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, newPk, msg, msig)
		assert.Nil(t, err)

		pubList = append(pubList, newPk)
		sigList = append(sigList, newSig)
	}

	if len(pubList) == 0 || len(sigList) == 0 {
		panic("should not be nil")
	}
	msPub := pubList[0].Clone()
	msSig := sigList[0].Clone()
	sigbyt, err := sigList[0].MarshalBinary()
	assert.Nil(t, err)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Add(pubList[i], msPub)
		msSig = msSig.Add(sigList[i], msSig)
	}

	msSigByt, err := msSig.MarshalBinary()
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, msSigByt)
	assert.Nil(t, err)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Sub(msPub, pubList[i])
		msSig = msSig.Sub(msSig, sigList[i])

		msSigByt, err = msSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)
	}

	pubByt, err := pubList[0].MarshalBinary()
	assert.Nil(t, err)
	msPubByt, err := msPub.MarshalBinary()
	assert.Nil(t, err)

	assert.Equal(t, pubByt, msPubByt)

	err = bls.Verify(suite, pubList[0], msg, msSigByt)
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, sigbyt)
	assert.Nil(t, err)

}

func TestBLS(t *testing.T) {
	msg := []byte("testmsg")
	suite := bn256.NewSuite()

	var pubList []kyber.Point
	var sigList []kyber.Point
	for i := 0; i < sigCount; i++ {
		private, public := bls.NewKeyPair(suite, random.New())
		asig, err := bls.Sign(suite, private, msg)
		assert.Nil(t, err)
		err = bls.Verify(suite, public, msg, asig)
		assert.Nil(t, err)

		pubList = append(pubList, public)

		s2 := suite.G1().Point()
		err = s2.UnmarshalBinary(asig)
		assert.Nil(t, err)
		sigList = append(sigList, s2)

		msig, err := s2.MarshalBinary()
		assert.Nil(t, err)
		err = bls.Verify(suite, public, msg, msig)
		assert.Nil(t, err)
	}
	if len(pubList) == 0 || len(sigList) == 0 {
		panic("should not be nil")
	}
	msPub := pubList[0].Clone()
	msSig := sigList[0].Clone()
	sigbyt, err := sigList[0].MarshalBinary()
	assert.Nil(t, err)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Add(pubList[i], msPub)
		msSig = msSig.Add(sigList[i], msSig)

		msSigByt, err := msSig.MarshalBinary()
		assert.Nil(t, err)
		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)

		msSig = msSig.Add(sigList[i], msSig)
		msPub = msPub.Add(pubList[i], msPub)

		msSigByt, err = msSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)
	}

	msSigByt, err := msSig.MarshalBinary()
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, msSigByt)
	assert.Nil(t, err)

	msSigOld := msSig.Clone()
	msPubOld := msPub.Clone()

	msSig = msSig.Add(msSig.Clone(), msSig)
	msPub = msPub.Add(msPub.Clone(), msPub)

	msSigByt, err = msSig.MarshalBinary()
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, msSigByt)
	assert.Nil(t, err)

	msSig = msSig.Sub(msSig, msSigOld)
	msPub = msPub.Sub(msPub, msPubOld)

	for i := 1; i < sigCount; i++ {
		msPub = msPub.Sub(msPub, pubList[i])
		msPub = msPub.Sub(msPub, pubList[i])
		msSig = msSig.Sub(msSig, sigList[i])
		msSig = msSig.Sub(msSig, sigList[i])

		msSigByt, err = msSig.MarshalBinary()
		assert.Nil(t, err)

		err = bls.Verify(suite, msPub, msg, msSigByt)
		assert.Nil(t, err)
	}

	pubByt, err := pubList[0].MarshalBinary()
	assert.Nil(t, err)
	msPubByt, err := msPub.MarshalBinary()
	assert.Nil(t, err)

	assert.Equal(t, pubByt, msPubByt)

	err = bls.Verify(suite, pubList[0], msg, msSigByt)
	assert.Nil(t, err)

	err = bls.Verify(suite, msPub, msg, sigbyt)
	assert.Nil(t, err)

}

var multiSigCount = 10

func TestBlsMerge(t *testing.T) {

	privs := make([]sig.Priv, multiSigCount)
	sigs := make([]sig.Sig, multiSigCount)
	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)
	toSign := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	var err error
	for _, newBid := range bitid.NewBitIDFuncs {
		for i := range privs {
			privs[i], err = NewBlspriv(newBid)
			assert.Nil(t, err)

			sigs[i], err = privs[i].(*Blspriv).Sign(toSign)
			assert.Nil(t, err)

			pub := privs[i].GetPub()
			pub.SetIndex(sig.PubKeyIndex(i))

			msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
			v, err := pub.VerifySig(msg, sigs[i])
			assert.Nil(t, err)
			assert.True(t, v)
		}

		mergeSig := sigs[0]
		mergePub := privs[0].GetPub()
		for i := 1; i < multiSigCount; i++ {
			newMergeSig, err := mergeSig.(sig.AllMultiSig).MergeSig(sigs[i].(sig.MultiSig))
			assert.Nil(t, err)
			mergeSig = newMergeSig.(sig.AllMultiSig)

			newMergePub, err := mergePub.(sig.MultiPub).MergePub(privs[i].GetPub().(sig.MultiPub))
			assert.Nil(t, err)
			mergePub = newMergePub.(sig.AllMultiPub)

			msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
			v, err := mergePub.(sig.AllMultiPub).VerifySig(msg, mergeSig.(sig.AllMultiSig))
			assert.Nil(t, err)
			assert.True(t, v)
		}

		subSig := mergeSig
		subPub := mergePub
		for i := 1; i < multiSigCount; i++ {
			newSubSig, err := subSig.(sig.AllMultiSig).SubSig(sigs[i].(sig.AllMultiSig))
			assert.Nil(t, err)
			subSig = newSubSig.(sig.AllMultiSig)

			newSubPub, err := subPub.(sig.MultiPub).SubMultiPub(privs[i].GetPub().(sig.MultiPub))
			assert.Nil(t, err)
			subPub = newSubPub.(sig.AllMultiPub)

			msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
			v, err := subPub.VerifySig(msg, subSig)
			assert.Nil(t, err)
			assert.True(t, v)
		}

		msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
		v, err := subPub.VerifySig(msg, sigs[0])
		assert.Nil(t, err)
		assert.True(t, v)

		msg = &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
		v, err = privs[0].GetPub().VerifySig(msg, subSig)
		assert.Nil(t, err)
		assert.True(t, v)
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var newBlsPrivFuncs []sig.NewPrivFunc

func init() {
	for _, nxt := range bitid.NewBitIDFuncs {
		bitIDFunc := nxt
		newBlsPrivFuncs = append(newBlsPrivFuncs, func() (sig.Priv, error) {
			return NewBlspriv(bitIDFunc)
		})
	}
}

func TestBlsPrintStats(t *testing.T) {
	t.Log("BLS stats")
	sig.RunFuncWithConfigSetting(func() { sig.SigTestPrintStats(newBlsPrivFuncs[0], t) },
		types.WithFalse, types.WithBothBool, types.WithTrue, types.WithFalse)
}

func TestBlsSharedSecret(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestComputeSharedSecret(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsEncode(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(nxt, t) },
			types.WithFalse, types.WithFalse, types.WithFalse, types.WithBothBool)
	}
}

func TestBlsFromBytes(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsSort(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsSign(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(nxt, types.NormalSignature, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithFalse)
	}
}

func TestBlsGetRand(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestRand(nxt, t) },
			types.WithFalse, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestBlsSerialize(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(nxt, types.NormalSignature, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsVRF(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

// func TestBlsTestMessageSerialize(t *testing.T) {
// 	RunFuncWithConfigSetting(func() { SigTestTestMessageSerialize(NewBlspriv, t) },
// 		WithBothBool, WithBothBool, WithBothBool, WithBothBool)
// }

func TestBlsMultiSignTestMsgSerialize(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsSignTestMsgSerialize(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
	}
}

func TestBlsSignMerge(t *testing.T) {
	for _, nxt := range newBlsPrivFuncs {
		sig.RunFuncWithConfigSetting(func() { sig.TestSigMerge(nxt, t) },
			types.WithBothBool, types.WithBothBool, types.WithTrue, types.WithBothBool)
	}
}
