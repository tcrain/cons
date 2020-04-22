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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/eddsa"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/kyber/v3/util/key"
)

var sigMsg = sig.SignTestMsg

func initBench() {
	thrshdssshare = NewDSSShared(*thrshn, 0, *thrsht)
}

// BenchmarkEdCoinVerify measures the time to generate a coin, including verifying shares
func BenchmarkEdCoinVerify(b *testing.B) {
	initBench()
	privFunc := GetEdPartPrivFunc()
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	privs := make([]sig.Priv, *thrshn)
	var err error
	sigItems := make([]*sig.SigItem, *thrshn)
	for i := range privs {
		privs[i], err = privFunc()
		assert.Nil(b, err)

		sigItems[i], err = privs[i].GenerateSig(msg, nil, types.CoinProof)
		assert.Nil(b, err)
	}
	logging.Info("Size of ED coin share: ", len(sigItems[0].SigBytes), " threshold ", *thrshn, *thrsht)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, nxt := range sigItems[:*thrsht] {
			err = privs[j].GetPub().(sig.CoinProofPubInterface).CheckCoinProof(msg, nxt.CoinProof)
			assert.Nil(b, err)
		}
		_, err = privs[0].GetPub().(sig.CoinProofPubInterface).CombineProofs(sigItems)
		assert.Nil(b, err)
	}

}

// BenchmarkEdCoinGen measures the time to generate a coin, NOT including verifying shares
func BenchmarkEdCoinGen(b *testing.B) {
	initBench()
	privFunc := GetEdPartPrivFunc()
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	privs := make([]sig.Priv, *thrshn)
	var err error
	sigItems := make([]*sig.SigItem, *thrshn)
	for i := range privs {
		privs[i], err = privFunc()
		assert.Nil(b, err)

		sigItems[i], err = privs[i].GenerateSig(msg, nil, types.CoinProof)
		assert.Nil(b, err)
	}
	logging.Info("Size of ED coin share: ", len(sigItems[0].SigBytes), " threshold ", *thrshn, *thrsht)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = privs[0].GetPub().(sig.CoinProofPubInterface).CombineProofs(sigItems)
		assert.Nil(b, err)
	}

}

// BenchmarkEdCoinSign measures the time to generate a singe con=in share
func BenchmarkEdCoinSign(b *testing.B) {
	initBench()
	privFunc := GetEdPartPrivFunc()

	priv, err := privFunc()
	assert.Nil(b, err)

	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := priv.GenerateSig(msg, nil, types.CoinProof)
		assert.Nil(b, err)
	}
}

// BenchmarkCoinShares measures the time it takes to verify a single coin share
func BenchmarkCoinShares(b *testing.B) {
	initBench()

	privFunc := GetEdPartPrivFunc()
	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	privs, err := privFunc()
	assert.Nil(b, err)

	sigItems, err := privs.GenerateSig(msg, nil, types.CoinProof)
	assert.Nil(b, err)
	logging.Info("Size of ED coin share: ", len(sigItems.SigBytes))
	pub := privs.GetPub().(sig.CoinProofPubInterface)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = pub.CheckCoinProof(msg, sigItems.CoinProof)
		assert.Nil(b, err)
	}
}

// BenchmarkVerifyShares tests the cost to make a schnorr signature
func BenchmarkSignShares(b *testing.B) {
	initBench()

	privFunc := GetEdPartPrivFunc()

	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	p, err := privFunc()
	assert.Nil(b, err)

	var si *sig.SigItem
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		si, err = p.GenerateSig(msg, nil, types.NormalSignature)
		assert.Nil(b, err)
	}
	b.StopTimer()

	logging.Info("ed sig size", len(si.SigBytes))
}

// BenchmarkVerifyShares tests the cost to verify a schnorr signature
func BenchmarkVerifyShares(b *testing.B) {
	initBench()

	privFunc := GetEdPartPrivFunc()

	hash := types.GetHash(sigMsg)
	msg := &sig.MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	p, err := privFunc()
	assert.Nil(b, err)

	si, err := p.GenerateSig(msg, nil, types.NormalSignature)
	assert.Nil(b, err)

	logging.Info("ed sig size", len(si.SigBytes))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		valid, err := p.GetPub().VerifySig(msg, si.Sig)
		assert.Nil(b, err)
		assert.True(b, valid)

	}
}

func BenchmarkVerifyED(b *testing.B) {
	suite := edwards25519.NewBlakeSHA256Ed25519()
	pair := key.NewKeyPair(suite)
	private := pair.Private
	public := pair.Public
	sig, err := schnorr.Sign(suite, private, sigMsg)
	assert.Nil(b, err)
	b.Log("ED sig length", len(sig))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = eddsa.Verify(public, sigMsg, sig)
		assert.Nil(b, err)
	}
}
func BenchmarkSignED(b *testing.B) {
	suite := edwards25519.NewBlakeSHA256Ed25519()
	pair := key.NewKeyPair(suite)
	private := pair.Private

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := schnorr.Sign(suite, private, sigMsg)
		assert.Nil(b, err)
	}
}
