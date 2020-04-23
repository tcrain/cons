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

package sig

import (
	"bytes"
	"crypto/rand"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/bits"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tcrain/cons/consensus/messages"
)

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var SignMsgSize = 26
var BLSSigInfoSize = 85
var EdSigInfoSize = 84

var EncryptMsgSize = 46
var EncryptOverhead = config.EncryptOverhead // 24
var Coin2ShareSize = 212

var EncryptTestMsg []byte
var SignTestMsg []byte

func init() {
	EncryptTestMsg = make([]byte, EncryptMsgSize)
	_, err := rand.Read(EncryptTestMsg)
	if err != nil {
		panic(err)
	}
	SignTestMsg = make([]byte, SignMsgSize)
	_, err = rand.Read(SignTestMsg)
	if err != nil {
		panic(err)
	}
}

// var Thrshn2, Thrsht2 = 10, 4
var TestKeyIndex = PubKeyIndex(0)
var TestIndex types.ConsensusInt = 10
var TestHash types.ConsensusHash = types.ConsensusHash(types.GetHash(utils.Uint64ToBytes(uint64(TestIndex))))
var AdditionalIndecies = []types.ConsensusID{
	types.ConsensusHash(types.GetHash(utils.Uint64ToBytes(uint64(TestIndex + 1)))),
	types.ConsensusHash(types.GetHash(utils.Uint64ToBytes(uint64(TestIndex + 2)))),
	types.ConsensusHash(types.GetHash(utils.Uint64ToBytes(uint64(TestIndex + 3)))),
}

func SigTestComputeSharedSecret(newPriv func() (Priv, error), t *testing.T) {
	priv1, err := newPriv()
	assert.Nil(t, err)
	priv2, err := newPriv()
	assert.Nil(t, err)

	assert.Equal(t, priv1.ComputeSharedSecret(priv2.GetPub()), priv2.ComputeSharedSecret(priv1.GetPub()))
}

func SigTestFromBytes(newPriv func() (Priv, error), t *testing.T) {
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()

	priv.SetIndex(TestKeyIndex)
	pub := priv.GetPub()
	pubBytes, err := pub.GetRealPubBytes()
	assert.Nil(t, err)
	pub2, err := pub.New().FromPubBytes(pubBytes)
	assert.Nil(t, err)
	pub2.SetIndex(TestKeyIndex)

	pubStr1, err := pub.GetPubString()
	assert.Nil(t, err)
	pubStr2, err := pub2.GetPubString()
	assert.Nil(t, err)
	assert.Equal(t, pubStr1, pubStr2)

	pubID1, err := pub.GetPubID()
	assert.Nil(t, err)
	pubID2, err := pub2.GetPubID()
	assert.Nil(t, err)
	assert.Equal(t, pubID1, pubID2)

	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)

	// toSign := sigMsg
	// if signHash {
	// 	toSign = hash
	// }

	msg := &MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	asig, err := priv.GenerateSig(msg, nil, types.NormalSignature)
	assert.Nil(t, err)

	v, err := pub.VerifySig(msg, asig.Sig)
	assert.Nil(t, err)
	assert.True(t, v)

	v, err = pub2.VerifySig(msg, asig.Sig)
	assert.Nil(t, err)
	assert.True(t, v)
}

func SigTestRand(newPriv func() (Priv, error), t *testing.T) {
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()
	pub := priv.GetPub()

	var binCount [2]int
	for i := 0; i < 100; i++ {
		msg := BasicSignedMessage("message to encode" + string(i))
		sig, err := priv.Sign(msg)
		assert.Nil(t, err)

		valid, err := pub.VerifySig(msg, sig)
		assert.True(t, valid)
		assert.Nil(t, err)

		binCount[sig.GetRand()]++
	}
	// This can fail because of randomness
	assert.True(t, binCount[0] > 30)
	assert.True(t, binCount[1] > 30)
}

func SigTestVRF(newPriv func() (Priv, error), t *testing.T) {
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()
	pub := priv.GetPub()

	msg := BasicSignedMessage("message to encode")
	index, proof := priv.Evaluate(msg)

	indexCheck, err := pub.ProofToHash(msg, proof)
	assert.Nil(t, err)
	assert.Equal(t, index, indexCheck)

	msg2 := BasicSignedMessage("message to encode 2")
	index2, proof2 := priv.Evaluate(msg2)

	indexCheck2, err := pub.ProofToHash(msg2, proof2)
	assert.Nil(t, err)
	assert.Equal(t, index2, indexCheck2)

	var diffCount int
	for i, nxt := range index {
		xor := nxt ^ index2[i]
		diffCount += bits.OnesCount8(xor)
	}
	// TODO this assert may fail, it just checks that there is some sort of randomness being generated
	assert.True(t, diffCount > (8*32)/4, diffCount < (8*32*3)/4)
}

func SigTestSort(newPriv func() (Priv, error), t *testing.T) {
	size := 10
	privs := make(SortPriv, size)
	pubs := make(SortPub, size)
	for i := 0; i < size; i++ {
		priv, err := newPriv()
		assert.Nil(t, err)
		defer priv.Clean()

		privs[i] = priv
		pubs[i] = priv.GetPub()
	}
	sort.Sort(privs)
	sort.Sort(pubs)
	for i, priv := range privs {
		pb, err := priv.GetPub().GetPubBytes()
		assert.Nil(t, err)
		pbpb, err := pubs[i].GetPubBytes()
		assert.Nil(t, err)
		assert.Equal(t, pb, pbpb)
	}
}

func SigTestSign(newPriv func() (Priv, error), signType types.SignType, t *testing.T) {
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()

	priv.SetIndex(TestKeyIndex)

	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)

	// toSign := sigMsg
	// if signHash {
	// 	toSign = hash
	// }

	msg := &MultipleSignedMessage{Hash: hash, Msg: sigMsg}
	asig, err := priv.GenerateSig(msg, nil, signType)
	assert.Nil(t, err)

	asig2, err := priv.GenerateSig(msg, nil, signType)
	assert.Nil(t, err)

	pub := priv.GetPub()

	var verifyFunc func(SignedMessage, Sig) (bool, error)
	switch signType {
	case types.NormalSignature:
		verifyFunc = pub.VerifySig
	case types.SecondarySignature:
		verifyFunc = pub.(SecondaryPub).VerifySecondarySig
	default:
		panic(signType)
	}

	v, err := verifyFunc(msg, asig.Sig)
	assert.Nil(t, err)
	assert.True(t, v)

	asig.Sig.Corrupt()
	v, err = verifyFunc(msg, asig.Sig)
	// assert.Nil(t, err)
	assert.False(t, v)

	v, err = verifyFunc(msg, asig2.Sig)
	assert.Nil(t, err)
	assert.True(t, v)
}

func SigTestEncode(newPriv func() (Priv, error), t *testing.T) {
	UsePubIndex = false
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()
	priv.SetIndex(TestKeyIndex)

	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)
	mockMsg := &MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	asig, err := priv.GenerateSig(mockMsg, nil, types.NormalSignature)
	assert.Nil(t, err)

	buf := bytes.NewBuffer(nil)
	n1, err := asig.Sig.Encode(buf)
	assert.Nil(t, err)

	newsig := asig.Sig.New()
	n2, err := newsig.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, n1, n2)

	pub := priv.GetPub()
	buf = bytes.NewBuffer(nil)
	n1, err = pub.Encode(buf)
	assert.Nil(t, err)

	newpub := pub.New()
	n2, err = newpub.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, n1, n2)

	v, err := pub.VerifySig(mockMsg, asig.Sig)
	assert.Nil(t, err)
	assert.True(t, v)

	v, err = newpub.VerifySig(mockMsg, newsig)
	assert.Nil(t, err)
	assert.True(t, v)
}

func SigTestSerialize(newPriv func() (Priv, error), signType types.SignType, t *testing.T) {
	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()

	priv.SetIndex(TestKeyIndex)

	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)
	mockMsg := &MultipleSignedMessage{Hash: hash, Msg: sigMsg}
	// toSign := sigMsg
	// if signHash {
	// 	toSign = hash
	// }

	asig, err := priv.GenerateSig(mockMsg, nil, signType)
	assert.Nil(t, err)

	pub := priv.GetPub()

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = pub
	if signType == types.CoinProof {
		hdrs[1] = asig.CoinProof
	} else {
		hdrs[1] = asig.Sig
	}

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())

	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	assert.Equal(t, size, msg.Len())

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, pub.GetID())

	pub2 := pub.New()
	_, err = pub2.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)

	if UsePubIndex || UseMultisig {
		id, err := pub2.GetPubID()
		assert.Nil(t, err)
		oldid, err := pub.GetPubID()
		assert.Nil(t, err)
		assert.Equal(t, id, oldid)
		pub2 = pub
	}

	assert.Nil(t, err)
	pubString, err := pub.GetPubString()
	assert.Nil(t, err)
	pubString2, err := pub2.GetPubString()
	assert.Nil(t, err)
	assert.Equal(t, pubString, pubString2)

	pubBytes, err := pub.GetPubBytes()
	assert.Nil(t, err)
	pubBytes2, err := pub2.GetPubBytes()
	assert.Nil(t, err)
	assert.Equal(t, pubBytes, pubBytes2)

	ht, err = msg.PeekHeaderType()
	assert.Nil(t, err)
	if signType == types.CoinProof {
		assert.Equal(t, asig.CoinProof.GetID(), ht)
	} else {
		assert.Equal(t, asig.Sig.GetID(), ht)
	}

	if signType != types.CoinProof {
		asig2 := asig.Sig.New()
		_, err = asig2.Deserialize(msg, types.IntIndexFuns)
		assert.Nil(t, err)

		asig3, err := priv.GenerateSig(mockMsg, nil, signType)
		assert.Nil(t, err)

		// if !UsePubIndex {
		var verifyFunc func(SignedMessage, Sig) (bool, error)
		switch signType {
		case types.NormalSignature:
			verifyFunc = pub2.VerifySig
		case types.SecondarySignature:
			verifyFunc = pub2.(SecondaryPub).VerifySecondarySig
		default:
			panic(signType)
		}

		v, err := verifyFunc(mockMsg, asig.Sig)
		assert.Nil(t, err)
		assert.True(t, v)

		v, err = verifyFunc(mockMsg, asig2)
		assert.Nil(t, err)
		assert.True(t, v)

		v, err = verifyFunc(mockMsg, asig3.Sig)
		assert.Nil(t, err)
		assert.True(t, v)
		//}
	} else {
		coinProof2 := asig.CoinProof.New()
		_, err = coinProof2.Deserialize(msg, types.IntIndexFuns)
		assert.Nil(t, err)

		coinProof3, err := priv.GenerateSig(mockMsg, nil, signType)
		assert.Nil(t, err)

		err = pub2.(CoinProofPubInterface).CheckCoinProof(mockMsg, asig.CoinProof)
		assert.Nil(t, err)

		err = pub2.(CoinProofPubInterface).CheckCoinProof(mockMsg, coinProof2)
		assert.Nil(t, err)

		err = pub2.(CoinProofPubInterface).CheckCoinProof(mockMsg, coinProof3.CoinProof)
		assert.Nil(t, err)

	}
}

// func SigTestTestMessageSerialize(newPriv func() (Priv, error), t *testing.T) {
// 	var ID uint32 = 10
// 	keyIndex := PubKeyIndex(13)

// 	priv, err := newPriv()
// 	priv.SetIndex(keyIndex)
// 	assert.Nil(t, err)
// 	hdr := &SigTestMessage{}
// 	hdr.Priv = priv
// 	hdr.Pub = priv.GetPub()
// 	hdr.ID = ID

// 	hdrs := make([]messages.MsgHeader, 1)
// 	hdrs[0] = hdr

// 	msg := messages.InitMsgSetup(hdrs, t)

// 	ht, err := msg.PeekHeaderType()
// 	assert.Nil(t, err)
// 	assert.Equal(t, ht, hdr.GetID())

// 	dser := &SigTestMessage{}
// 	dser.Pub = priv.GetPub().New()

// 	_, err = (dser).Deserialize(msg)
// 	assert.Nil(t, err)
// 	assert.Equal(t, dser.ID, ID)

// 	if UsePubIndex {
// 		id, err := dser.Pub.GetPubID()
// 		assert.Nil(t, err)
// 		oldid, err := priv.GetPub().GetPubID()
// 		assert.Nil(t, err)
// 		assert.Equal(t, id, oldid)
// 		dser.Pub, err = priv.GetPub().InformState(priv)
// 		assert.Nil(t, err)
// 	}
// 	valid, err := dser.Pub.VerifySig(&MockMultipleSignedMessage{MultipleSignedMessage{Hash: dser.Hash, Msg: dser.Msg}}, dser.Sig)
// 	assert.Nil(t, err)
// 	assert.True(t, valid)

// 	if UsePubIndex {
// 		id, err := dser.Pub2.GetPubID()
// 		assert.Nil(t, err)
// 		oldid, err := priv.GetPub().GetPubID()
// 		assert.Nil(t, err)
// 		assert.Equal(t, id, oldid)
// 		dser.Pub2, err = priv.GetPub().InformState(priv)
// 		assert.Nil(t, err)
// 	}
// 	valid, err = dser.Pub2.VerifySig(&MockMultipleSignedMessage{MultipleSignedMessage{Hash: dser.Hash2, Msg: dser.Msg2}}, dser.Sig2)
// 	assert.Nil(t, err)
// 	assert.True(t, valid)
// }

// Helper function
func signMsgAndSerialize(useAdditionalIndices bool, hdr messages.InternalSignedMsgHeader,
	sigItems []*SigItem, priv Priv, t *testing.T) *messages.Message {

	var sm *MultipleSignedMessage
	var idx types.ConsensusIndex
	var err error
	if useAdditionalIndices {
		idx, err = types.GenerateParentHash(TestHash, AdditionalIndecies)
	} else {
		idx, err = types.SingleComputeConsensusID(TestIndex, nil)
	}
	assert.Nil(t, err)

	if priv != nil {
		sm = NewMultipleSignedMsg(idx, priv.GetPub(), hdr)
		_, err := sm.Serialize(messages.NewMessage(nil))
		// assert.Equal(t, utils.ErrNilPriv, err)
		assert.Nil(t, err)
		mySig, err := priv.GenerateSig(sm, nil, types.NormalSignature)
		assert.Nil(t, err)
		sm.SetSigItems([]*SigItem{mySig})
	} else {
		sm = NewMultipleSignedMsg(idx, sigItems[0].Pub.New(), hdr)
	}
	sm.SetSigItems(append(sm.GetSigItems(), sigItems...))
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = sm
	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	return msg
}

func SigTestMultiSignTestMsgSerialize(newPriv func() (Priv, error), t *testing.T) {
	var round uint32 = 22
	var binVal types.BinVal = 1

	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()
	priv.SetIndex(TestKeyIndex)
	priv2, err := newPriv()
	assert.Nil(t, err)
	defer priv2.Clean()

	priv2.SetIndex(TestKeyIndex + 1)
	var sigItems []*SigItem

	// sign with the first sig
	hdr := NewMultiSignTestMsg(priv.GetPub())
	hdr.Round = round
	hdr.BinVal = binVal
	msg := signMsgAndSerialize(false, hdr, nil, priv, t)

	// sign with the second sig
	msg2 := signMsgAndSerialize(false, hdr, nil, priv2, t)

	// Verify the first msg
	dser := NewMultiSignTestMsg(priv.GetPub())
	sm := NewMultipleSignedMsg(types.ConsensusIndex{}, priv.GetPub(), dser)
	_, err = (sm).Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, round, dser.Round)
	assert.Equal(t, binVal, dser.BinVal)
	assert.Equal(t, TestIndex, sm.Index.Index)
	assert.Equal(t, []types.ConsensusID(nil), sm.Index.AdditionalIndices)

	for _, sigItem := range sm.GetSigItems() {
		if UsePubIndex || UseMultisig {
			id, err := sigItem.Pub.GetPubID()
			assert.Nil(t, err)
			oldid, err := priv.GetPub().GetPubID()
			assert.Nil(t, err)
			assert.Equal(t, id, oldid)
			sigItem.Pub = priv.GetPub()
		}

		valid, err := sigItem.Pub.VerifySig(sm, sigItem.Sig)
		assert.Nil(t, err)
		assert.True(t, valid)

		sigItems = append(sigItems, sigItem)
	}

	// Verify the second msg
	dser2 := NewMultiSignTestMsg(priv2.GetPub())
	sm2 := NewMultipleSignedMsg(types.ConsensusIndex{}, priv2.GetPub(), dser2)
	_, err = (sm2).Deserialize(msg2, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, round, dser2.Round)
	assert.Equal(t, binVal, dser2.BinVal)
	assert.Equal(t, TestIndex, sm2.Index.Index)
	assert.Equal(t, []types.ConsensusID(nil), sm2.Index.AdditionalIndices)

	for _, sigItem := range sm2.GetSigItems() {
		if UsePubIndex || UseMultisig {
			id, err := sigItem.Pub.GetPubID()
			assert.Nil(t, err)
			oldid, err := priv2.GetPub().GetPubID()
			assert.Nil(t, err)
			assert.Equal(t, oldid, id)
			sigItem.Pub = priv2.GetPub()
		}

		valid, err := sigItem.Pub.VerifySig(sm, sigItem.Sig)
		assert.Nil(t, err)
		assert.True(t, valid)
		sigItems = append(sigItems, sigItem)
	}

	// Make a third msg with both sigs
	msg3 := signMsgAndSerialize(false, hdr, sigItems, nil, t)

	// Verify the third msg
	dser3 := NewMultiSignTestMsg(priv2.GetPub())
	sm3 := NewMultipleSignedMsg(types.ConsensusIndex{}, priv2.GetPub(), dser3)
	_, err = sm3.Deserialize(msg3, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, round, dser3.Round)
	assert.Equal(t, binVal, dser3.BinVal)
	assert.Equal(t, TestIndex, sm3.Index.Index)
	assert.Equal(t, []types.ConsensusID(nil), sm3.Index.AdditionalIndices)

	var i int
	var sigItem *SigItem
	for i, sigItem = range sm3.GetSigItems() {
		if UsePubIndex || UseMultisig {
			id, err := sigItem.Pub.GetPubID()
			assert.Nil(t, err)
			oldid1, err := priv.GetPub().GetPubID()
			assert.Nil(t, err)
			oldid2, err := priv2.GetPub().GetPubID()
			assert.Nil(t, err)

			var lpriv Priv
			if oldid1 == id {
				lpriv = priv
			} else if oldid2 == id {
				lpriv = priv2
			} else {
				t.Error("invalid id")
			}
			sigItem.Pub = lpriv.GetPub()
		}

		valid, err := sigItem.Pub.VerifySig(sm, sigItem.Sig)
		assert.Nil(t, err)
		assert.True(t, valid)
		sigItems = append(sigItems, sigItem)
	}
	assert.Equal(t, i, 1)
}

func SigTestSignTestMsgSerialize(newPriv func() (Priv, error), t *testing.T) {
	var round uint32 = 22
	var binVal types.BinVal = 1

	priv, err := newPriv()
	assert.Nil(t, err)
	defer priv.Clean()

	priv.SetIndex(TestKeyIndex)
	hdr := NewMultiSignTestMsg(priv.GetPub())
	hdr.Round = round
	hdr.BinVal = binVal

	// test without the additional indices
	msg := signMsgAndSerialize(false, hdr, nil, priv, t)
	dser := NewMultiSignTestMsg(priv.GetPub())
	sm := NewMultipleSignedMsg(types.ConsensusIndex{}, priv.GetPub(), dser)
	_, err = sm.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, round, dser.Round)
	assert.Equal(t, binVal, dser.BinVal)
	assert.Equal(t, TestIndex, sm.Index.Index)
	assert.Equal(t, []types.ConsensusID(nil), sm.Index.AdditionalIndices)

	if UsePubIndex || UseMultisig {
		id, err := sm.SigItems[0].Pub.GetPubID()
		assert.Nil(t, err)
		oldid, err := priv.GetPub().GetPubID()
		assert.Nil(t, err)
		assert.Equal(t, id, oldid)
		sm.SigItems[0].Pub = priv.GetPub()
	}

	valid, err := sm.SigItems[0].Pub.VerifySig(sm, sm.SigItems[0].Sig)
	assert.Nil(t, err)
	assert.True(t, valid)

	// test with the additional indices
	msg = signMsgAndSerialize(true, hdr, nil, priv, t)
	dser = NewMultiSignTestMsg(priv.GetPub())
	sm = NewMultipleSignedMsg(types.ConsensusIndex{}, priv.GetPub(), dser)
	_, err = sm.Deserialize(msg, types.HashIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, round, dser.Round)
	assert.Equal(t, binVal, dser.BinVal)
	tsthsh, err := types.GenerateParentHash(TestHash, AdditionalIndecies)
	assert.Nil(t, err)
	assert.Equal(t, tsthsh.Index, sm.Index.Index)
	assert.Equal(t, AdditionalIndecies, sm.Index.AdditionalIndices)

	if UsePubIndex || UseMultisig {
		id, err := sm.SigItems[0].Pub.GetPubID()
		assert.Nil(t, err)
		oldid, err := priv.GetPub().GetPubID()
		assert.Nil(t, err)
		assert.Equal(t, id, oldid)
		sm.SigItems[0].Pub = priv.GetPub()
	}

	valid, err = sm.SigItems[0].Pub.VerifySig(sm, sm.SigItems[0].Sig)
	assert.Nil(t, err)
	assert.True(t, valid)

}

func TestPartThrsh(privFunc func() (Priv, error), signType types.SignType, tval, n int, t *testing.T) {
	var privs []Priv
	var sigs []Sig

	sigMsg := []byte("sign this message")
	hash := types.GetHash(sigMsg)
	msg := &MultipleSignedMessage{Hash: hash, Msg: sigMsg}

	tm1 := tval - 1
	for i := 0; i < n; i++ {
		p, err := privFunc()
		assert.Nil(t, err)
		privs = append(privs, p)

		si, err := p.GenerateSig(msg, nil, signType)
		assert.Nil(t, err)
		sigs = append(sigs, si.Sig)

		var verifyFunc func(SignedMessage, Sig) (bool, error)
		var thrsh ThreshStateInterface
		switch signType {
		case types.NormalSignature:
			verifyFunc = p.GetPub().VerifySig
			thrsh = p.(ThreshStateInterface)
		case types.SecondarySignature:
			if p2, ok := p.GetPub().(SecondaryPub); ok {
				verifyFunc = p2.VerifySecondarySig
				thrsh = p.(SecondaryPriv).GetSecondaryPriv().(ThreshStateInterface)
			} else {
				verifyFunc = p.GetPub().VerifySig
				thrsh = p.(ThreshStateInterface)
			}
		default:
			panic(signType)
		}

		valid, err := verifyFunc(msg, si.Sig)
		assert.Nil(t, err)
		assert.True(t, valid)

		thrshSig, err := thrsh.CombinePartialSigs(sigs)
		if i < tm1 {
			assert.NotNil(t, err)
			continue
		} else {
			assert.Nil(t, err)

			valid, err := thrsh.GetSharedPub().VerifySig(msg, thrshSig.Sig)
			assert.Nil(t, err)
			assert.True(t, valid)

			thrshSig.Sig.Corrupt()
			valid, err = thrsh.GetSharedPub().VerifySig(msg, thrshSig.Sig)
			assert.NotNil(t, err)
			assert.False(t, valid)
		}
	}

}

func RunFuncWithConfigSetting(toRun func(), usePubIndex types.BoolSetting, useMultisig types.BoolSetting,
	blsMultiNew types.BoolSetting, sleepValidate types.BoolSetting) {

	for _, upi := range usePubIndex {
		SetUsePubIndex(upi)
		for _, ums := range useMultisig {
			SetUseMultisig(ums)
			for _, bmn := range blsMultiNew {
				SetBlsMultiNew(bmn)
				for _, sv := range sleepValidate {
					SetSleepValidate(sv)
					toRun()
				}
			}
		}
	}
}
