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
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/util/random"
)

var sigMsg = sig.SignTestMsg

func initBench() {
	Thrshblsshare = NewBlsShared(*thrshn, *thrsht)
	Thrshblsshare2 = NewBlsShared(*thrshn2, *thrsht2)
}

// BenchmarkBlsCoinVerify measures the time to generate a coin, including verifing and combining shares
func BenchmarkBlsCoinVerify(b *testing.B) {
	initBench()
	privFunc := GetBlsPartPrivFunc()
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	privs := make([]sig.Priv, *thrshn)
	var err error
	sigItems := make([]*sig.SigItem, *thrshn)
	for i := range privs {
		privs[i], err = privFunc()
		assert.Nil(b, err)

		sigItems[i], err = privs[i].GenerateSig(msg, nil, types.NormalSignature)
		assert.Nil(b, err)
	}
	logging.Info("Size of BLS coin share: ", len(sigItems[0].SigBytes))

	var tPriv sig.ThreshStateInterface
	var sharedSig *sig.SigItem
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sigs []sig.Sig
		for j, nxt := range sigItems[:*thrsht] {
			ok, err := privs[j].GetPub().VerifySig(msg, nxt.Sig)
			assert.True(b, ok)
			assert.Nil(b, err)
			sigs = append(sigs, nxt.Sig)
		}
		tPriv = privs[0].(sig.ThreshStateInterface)
		sharedSig, err = tPriv.CombinePartialSigs(sigs)
		assert.Nil(b, err)
	}
	b.StopTimer()

	ok, err := tPriv.GetSharedPub().VerifySig(msg, sharedSig.Sig)
	assert.True(b, ok)
	assert.Nil(b, err)
}

// BenchmarkBlsCoinGen measures the time to generate a coin by combining shares (does not include verification)
func BenchmarkBlsCoinGen(b *testing.B) {
	initBench()
	privFunc := GetBlsPartPrivFunc()
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	privs := make([]sig.Priv, *thrshn)
	var err error
	sigItems := make([]*sig.SigItem, *thrshn)
	for i := range privs {
		privs[i], err = privFunc()
		assert.Nil(b, err)

		sigItems[i], err = privs[i].GenerateSig(msg, nil, types.NormalSignature)
		assert.Nil(b, err)
	}
	logging.Info("Size of BLS coin share: ", len(sigItems[0].SigBytes))

	var tPriv sig.ThreshStateInterface
	var sharedSig *sig.SigItem
	var sigs []sig.Sig
	for _, nxt := range sigItems[:*thrsht] {
		sigs = append(sigs, nxt.Sig)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tPriv = privs[0].(sig.ThreshStateInterface)
		sharedSig, err = tPriv.CombinePartialSigs(sigs)
		assert.Nil(b, err)
	}
	b.StopTimer()

	ok, err := tPriv.GetSharedPub().VerifySig(msg, sharedSig.Sig)
	assert.True(b, ok)
	assert.Nil(b, err)
}

func BenchmarkBLSCoinSign(b *testing.B) {
	initBench()
	privFunc := GetBlsPartPrivFunc()

	priv, err := privFunc()
	assert.Nil(b, err)

	sigMsg = sig.SignTestMsg
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
	var sigItem *sig.SigItem

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sigItem, err = priv.GenerateSig(msg, nil, types.NormalSignature)
		assert.Nil(b, err)
	}
	b.StopTimer()
	logging.Info("size of BLS coin share", len(sigItem.SigBytes))
}

func BenchmarkBLSShareVerify(b *testing.B) {
	initBench()
	privFunc := GetBlsPartPrivFunc()

	priv, err := privFunc()
	assert.Nil(b, err)

	sigMsg = sig.SignTestMsg
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}
	var sigItem *sig.SigItem

	sigItem, err = priv.GenerateSig(msg, nil, types.NormalSignature)
	assert.Nil(b, err)
	pub := priv.GetPub()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ok, err := pub.VerifySig(msg, sigItem.Sig)
		assert.True(b, ok)
		assert.Nil(b, err)
	}
	b.StopTimer()
	logging.Info("size of BLS coin share", len(sigItem.SigBytes))
}

// BenchmarkBLSBatchVerify measures the time it takes to verify a batch of signatures on differnt messages
func BenchmarkBLSBatchVerify(b *testing.B) {
	batchSize := 100
	suite := bn256.NewSuite()
	var pubs []kyber.Point
	var msgs [][]byte
	var sigs [][]byte

	for i := 0; i < batchSize; i++ {
		msg := []byte(fmt.Sprintf("TestMsg %v", i))
		suite := bn256.NewSuite()
		private, public := bls.NewKeyPair(suite, random.New())
		sig, err := bls.Sign(suite, private, msg)
		assert.Nil(b, err)
		pubs = append(pubs, public)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aggregatedSig, err := bls.AggregateSignatures(suite, sigs...)
		assert.Nil(b, err)
		err = bls.BatchVerify(suite, pubs, msgs, aggregatedSig)
		assert.Nil(b, err)
	}
}

// BenchmarkBLSAggVerify measures the time it takes to verify an aggregate signature on the same msg
func BenchmarkBLSAggVerify(b *testing.B) {
	aggSize := 200
	suite := bn256.NewSuite()
	private, publicAgg := bls.NewKeyPair(suite, random.New())
	msg := []byte("test msg")
	sig, err := bls.Sign(suite, private, msg)
	assert.Nil(b, err)
	aggSig := suite.G1().Point()
	err = aggSig.UnmarshalBinary(sig)
	assert.Nil(b, err)

	for i := 1; i < aggSize; i++ {
		private, public := bls.NewKeyPair(suite, random.New())
		publicAgg = publicAgg.Add(publicAgg, public)
		sig, err := bls.Sign(suite, private, msg)
		assert.Nil(b, err)
		nxtSig := suite.G1().Point()
		err = nxtSig.UnmarshalBinary(sig)
		assert.Nil(b, err)
		aggSig = aggSig.Add(aggSig, nxtSig)
	}

	sig, err = aggSig.MarshalBinary()
	assert.Nil(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = bls.Verify(suite, publicAgg, msg, sig)
		assert.Nil(b, err)
	}
}

// BenchmarkBLSAggAndVerify measures the time it takes to aggregate and verify a set of signatures on the same msg
func BenchmarkBLSAggAndVerify(b *testing.B) {
	aggSize := 200
	suite := bn256.NewSuite()
	msg := []byte("test msg")
	var pubs []kyber.Point
	var sigs [][]byte

	for i := 0; i < aggSize; i++ {
		private, public := bls.NewKeyPair(suite, random.New())
		sig, err := bls.Sign(suite, private, msg)
		assert.Nil(b, err)
		pubs = append(pubs, public)
		sigs = append(sigs, sig)
	}

	publicAgg := suite.G2().Point()
	aggSig := suite.G1().Point()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < aggSize; j++ {
			publicAgg = publicAgg.Add(publicAgg, pubs[j])
			nxtSig := suite.G1().Point()
			err := nxtSig.UnmarshalBinary(sigs[j])
			assert.Nil(b, err)
			aggSig = aggSig.Add(aggSig, nxtSig)
		}
		sig, err := aggSig.MarshalBinary()
		assert.Nil(b, err)
		err = bls.Verify(suite, publicAgg, msg, sig)
		assert.Nil(b, err)
	}
}

func BenchmarkVerifyBLS(b *testing.B) {
	suite := bn256.NewSuite()
	private, public := bls.NewKeyPair(suite, random.New())
	sig, err := bls.Sign(suite, private, sigMsg)
	assert.Nil(b, err)
	b.Log("BLS sig length", len(sig))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = bls.Verify(suite, public, sigMsg, sig)
		assert.Nil(b, err)
	}
}

func BenchmarkSignBLS(b *testing.B) {
	suite := bn256.NewSuite()
	private, _ := bls.NewKeyPair(suite, random.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bls.Sign(suite, private, sigMsg)
		assert.Nil(b, err)
	}
}

func BenchmarkCombinePubBLS(b *testing.B) {
	suite := bn256.NewSuite()
	_, public1 := bls.NewKeyPair(suite, random.New())
	_, public2 := bls.NewKeyPair(suite, random.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		public1.Clone().Add(public1, public2)
	}
}
