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

package messagetypes

import (
	// "fmt"

	"bytes"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
)

var tstMsgIndex types.ConsensusInt = 12
var tstMsgIdxObj = types.ConsensusIndex{
	Index:         tstMsgIndex,
	FirstIndex:    tstMsgIndex,
	UnmarshalFunc: types.IntIndexFuns,
}

// helper function
func internalTestSignedMsgSerialize(newMsgFunc func(bool) messages.InternalSignedMsgHeader,
	checkEqualFunc func(messages.InternalSignedMsgHeader), idx types.ConsensusID, useAdditionalIndices bool, t *testing.T) {

	index := sig.TestKeyIndex
	thrshblsshare := bls.NewBlsShared(config.Thrshn, config.Thrsht)
	blsThresh := bls.NewBlsThrsh(config.Thrshn, config.Thrsht, index, thrshblsshare.PriScalars[index],
		thrshblsshare.PubPoints[index], thrshblsshare.SharedPub)
	privFunc := func() (sig.Priv, error) {
		index += 1
		return bls.NewBlsPartPriv(blsThresh)
	}

	priv, err := privFunc()
	assert.Nil(t, err)
	priv2, err := privFunc()
	assert.Nil(t, err)
	var sigItems []*sig.SigItem

	// first message
	pub := priv.GetPub()
	hdr := newMsgFunc(false)
	msg := signMsgAndSerialize(hdr, pub, priv, nil, idx, useAdditionalIndices, t)

	// second message
	// hdr.SetSigItems(nil)
	pub2 := priv2.GetPub()
	msg2 := signMsgAndSerialize(hdr, pub2, priv2, nil, idx, useAdditionalIndices, t)

	// Verify the first msg
	dser := sig.NewMultipleSignedMsg(types.ConsensusIndex{}, pub, newMsgFunc(true))
	unmarFunc := types.IntIndexFuns
	var mvHdrIdx types.ConsensusIndex
	if useAdditionalIndices {
		unmarFunc = types.HashIndexFuns
		mvHdrIdx, err = types.GenerateParentHash(tstMsgIdHash, mvMultiHashIdxs)
		assert.Nil(t, err)
	} else {
		mvHdrIdx, err = types.SingleComputeConsensusID(idx, nil)
		assert.Nil(t, err)
	}
	_, err = (dser).Deserialize(msg, unmarFunc)
	assert.Nil(t, err)
	assert.Equal(t, mvHdrIdx.Index, dser.Index.Index)
	checkEqualFunc(dser.InternalSignedMsgHeader)

	checkSigs(dser, priv, t)
	sigItems = append(sigItems, dser.GetSigItems()...)

	// Verify the second msg
	dser2 := sig.NewMultipleSignedMsg(types.ConsensusIndex{}, pub2, newMsgFunc(true))
	_, err = (dser2).Deserialize(msg2, unmarFunc)
	assert.Equal(t, mvHdrIdx.Index, dser2.Index.Index)
	assert.Nil(t, err)
	checkEqualFunc(dser2.InternalSignedMsgHeader)
	assert.Equal(t, dser.GetMsgID(), dser2.GetMsgID())

	checkSigs(dser2, priv2, t)
	sigItems = append(sigItems, dser2.GetSigItems()...)

	// Make a third msg with both sigs
	msg3 := signMsgAndSerialize(hdr, pub2, nil, sigItems, idx, useAdditionalIndices, t)

	// Verify the third msg
	dser3 := sig.NewMultipleSignedMsg(types.ConsensusIndex{}, pub2, newMsgFunc(true))
	_, err = (dser3).Deserialize(msg3, unmarFunc)
	assert.Equal(t, mvHdrIdx.Index, dser3.Index.Index)
	assert.Nil(t, err)
	checkEqualFunc(dser3.InternalSignedMsgHeader)
	assert.Equal(t, dser.GetMsgID(), dser2.GetMsgID())

	assert.Equal(t, len(dser3.GetSigItems()), 2)
	tmp := dser.GetSigItems()
	dser.SetSigItems(tmp[:1])
	checkSigs(dser, priv, t)
	dser.SetSigItems(tmp[1:])
	checkSigs(dser, priv2, t)
}

func internalTestUnsignedMsgSerialize(newMsgFunc func(bool) messages.InternalSignedMsgHeader,
	checkEqualFunc func(messages.InternalSignedMsgHeader), idx types.ConsensusID, useAdditionalIndices bool, t *testing.T) {

	privFunc := ec.NewEcpriv

	priv, err := privFunc()
	assert.Nil(t, err)
	priv2, err := privFunc()
	assert.Nil(t, err)

	// first message
	pub := priv.GetPub()
	hdr := newMsgFunc(false)
	msg := unsignMsgAndSerialize(hdr, pub, nil, idx, useAdditionalIndices, t)

	// second message
	// hdr.SetSigItems(nil)
	pub2 := priv2.GetPub()
	msg2 := unsignMsgAndSerialize(hdr, pub2, nil, idx, useAdditionalIndices, t)

	pubs := []sig.Pub{pub, pub2}

	// Verify the first msg
	dser := sig.NewUnsignedMessage(types.ConsensusIndex{}, pub, newMsgFunc(true))
	unmarFunc := types.IntIndexFuns
	var mvHdrIdx types.ConsensusIndex
	if useAdditionalIndices {
		unmarFunc = types.HashIndexFuns
		mvHdrIdx, err = types.GenerateParentHash(tstMsgIdHash, mvMultiHashIdxs)
		assert.Nil(t, err)
	} else {
		mvHdrIdx, err = types.SingleComputeConsensusID(idx, nil)
		assert.Nil(t, err)
	}
	_, err = (dser).Deserialize(msg, unmarFunc)
	assert.Nil(t, err)
	assert.Equal(t, mvHdrIdx.Index, dser.Index.Index)
	assert.Equal(t, 0, len(dser.GetEncryptPubs()))
	checkEqualFunc(dser.InternalSignedMsgHeader)

	// Verify the second msg
	dser2 := sig.NewUnsignedMessage(types.ConsensusIndex{}, pub2, newMsgFunc(true))
	_, err = (dser2).Deserialize(msg2, unmarFunc)
	assert.Equal(t, mvHdrIdx.Index, dser2.Index.Index)
	assert.Nil(t, err)
	checkEqualFunc(dser2.InternalSignedMsgHeader)
	assert.Equal(t, dser.GetMsgID(), dser2.GetMsgID())
	assert.Equal(t, 0, len(dser2.GetEncryptPubs()))

	// Make a third msg with both sigs
	msg3 := unsignMsgAndSerialize(hdr, pub2, pubs, idx, useAdditionalIndices, t)

	// Verify the third msg
	dser3 := sig.NewUnsignedMessage(types.ConsensusIndex{}, pub2, newMsgFunc(true))
	_, err = (dser3).Deserialize(msg3, unmarFunc)
	assert.Equal(t, mvHdrIdx.Index, dser3.Index.Index)
	assert.Nil(t, err)
	checkEqualFunc(dser3.InternalSignedMsgHeader)
	assert.Equal(t, dser.GetMsgID(), dser2.GetMsgID())

	assert.Equal(t, 2, len(dser3.GetEncryptPubs()))
}

var rndMsg = sig.BasicSignedMessage("some message")

// Helper function
func signMsgAndSerialize(hdr messages.InternalSignedMsgHeader, pub sig.Pub, priv sig.Priv,
	sigitems []*sig.SigItem, firstIdx types.ConsensusID, useAdditionalIndices bool, t *testing.T) *messages.Message {

	var addIds []types.ConsensusID
	if useAdditionalIndices {
		addIds = mvMultiHashIdxs
	}
	unmarFunc := types.IntIndexFuns
	if useAdditionalIndices {
		unmarFunc = types.HashIndexFuns
	}
	idx, err := unmarFunc.ComputeConsensusID(firstIdx, addIds)
	assert.Nil(t, err)

	msm := sig.NewMultipleSignedMsg(idx, pub, hdr)
	if priv != nil {
		_, err := msm.Serialize(messages.NewMessage(nil))
		assert.Nil(t, err)
		// Add a VRF for fun
		_, vrfProof := priv.(sig.VRFPriv).Evaluate(rndMsg)
		mySig, err := priv.GenerateSig(msm, vrfProof, msm.GetSignType())
		assert.Nil(t, err)
		msm.SetSigItems([]*sig.SigItem{mySig})
	}
	msm.SetSigItems(append(msm.GetSigItems(), sigitems...))
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = msm
	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	return msg
}

func unsignMsgAndSerialize(hdr messages.InternalSignedMsgHeader, pub sig.Pub,
	pubs []sig.Pub, firstIdx types.ConsensusID, useAdditionalIndices bool, t *testing.T) *messages.Message {

	var addIds []types.ConsensusID
	if useAdditionalIndices {
		addIds = mvMultiHashIdxs
	}
	unmarFunc := types.IntIndexFuns
	if useAdditionalIndices {
		unmarFunc = types.HashIndexFuns
	}
	idx, err := unmarFunc.ComputeConsensusID(firstIdx, addIds)
	assert.Nil(t, err)

	msm := sig.NewUnsignedMessage(idx, pub, hdr)
	msm.SetEncryptPubs(pubs)
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = msm
	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	return msg
}

// Helper function
func checkSigs(dser *sig.MultipleSignedMessage, priv sig.Priv, t *testing.T) {
	var sigItems []*sig.SigItem
	for _, sigItem := range dser.GetSigItems() {
		if sig.GetUsePubIndex() {
			id, err := sigItem.Pub.GetPubID()
			assert.Nil(t, err)
			oldid, err := priv.GetPub().GetPubID()
			assert.Nil(t, err)
			assert.Equal(t, id, oldid)
			sigItem.Pub = priv.GetPub()
		}
		valid, err := sigItem.Pub.VerifySig(dser, sigItem.Sig)
		assert.Nil(t, err)
		assert.True(t, valid)
		// Check the VRF
		_, err = sigItem.Pub.(sig.VRFPub).ProofToHash(rndMsg, sigItem.VRFProof)
		assert.Nil(t, err)
		sigItems = append(sigItems, sigItem)
	}
	dser.SetSigItems(sigItems)
}

func TestBinStateSerialize(t *testing.T) {

	hdr := (ConsBinStateMessage)([]byte("test"))
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	_, err = hdr.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
}

func TestConsMsgSerialize(t *testing.T) {

	hdr := &ConsMessage{}
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	_, err = hdr.Deserialize(msg, types.IntIndexFuns)
	assert.Equal(t, nil, hdr.GetIndex().Index)
	assert.Nil(t, err)
}

func TestNoProgressMsgSerialize(t *testing.T) {

	hdr := &NoProgressMessage{0, true, basicMessage{tstMsgIdxObj}}
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	idx, err := hdr.PeekHeaders(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, tstMsgIndex, idx.Index)
	assert.Equal(t, nil, idx.FirstIndex)
	assert.Equal(t, []types.ConsensusID(nil), idx.AdditionalIndices)

	_, err = hdr.Deserialize(msg, types.IntIndexFuns)
	assert.Equal(t, tstMsgIndex, hdr.GetIndex().Index)
	assert.True(t, hdr.IsUnconsumedOutput)
	assert.Nil(t, err)
}

func TestSimpleConsMsgSerialize(t *testing.T) {

	priv, err := ec.NewEcpriv()
	assert.Nil(t, err)

	pub := priv.GetPub()
	hdr := NewSimpleConsMessage(pub)
	hdr.MyPub = pub
	msg := signMsgAndSerialize(hdr, pub, priv, nil, tstMsgIndex, false, t)

	dser := sig.NewMultipleSignedMsg(types.ConsensusIndex{}, pub, NewSimpleConsMessage(pub))
	_, err = (dser).Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, tstMsgIndex, dser.GetIndex().Index)

	checkSigs(dser, priv, t)
}

func TestPartialMsgSerialize(t *testing.T) {
	fullMsg := []byte("This is a msg to send")
	var round types.ConsensusRound = 10
	split := bytes.Split(fullMsg, []byte(" "))
	tmpHdr := GeneratePartialMessages(fullMsg, round, split)[0].(*PartialMessage)
	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		var hdr messages.InternalSignedMsgHeader
		if createEmpty {
			hdr = NewPartialMessage()
		} else {
			hdr = GeneratePartialMessages(fullMsg, round, split)[0]
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*PartialMessage)
		assert.Equal(t, tmpHdr.FullMsgHash, hdr.FullMsgHash)
		assert.Equal(t, tmpHdr.PartialMsgHashes, hdr.PartialMsgHashes)
		assert.Equal(t, tmpHdr.PartialMsg, split[0])
		assert.Equal(t, tmpHdr.Round, hdr.Round)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestAuxProofMsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= types.Coin; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewAuxProofMessage(binVal == types.Coin)
			if !createEmpty {
				hdr.Round = round
				hdr.BinVal = binVal
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*AuxProofMessage)
			assert.Equal(t, round, hdr.Round)
			assert.Equal(t, binVal, hdr.BinVal)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestAuxStage0MsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= 1; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewAuxStage0Message()
			if !createEmpty {
				hdr.Round = round
				hdr.BinVal = binVal
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*AuxStage0Message)
			assert.Equal(t, round, hdr.Round)
			assert.Equal(t, binVal, hdr.BinVal)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestAuxStage1MsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= 1; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewAuxStage1Message()
			if !createEmpty {
				hdr.Round = round
				hdr.BinVal = binVal
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*AuxStage1Message)
			assert.Equal(t, round, hdr.Round)
			assert.Equal(t, binVal, hdr.BinVal)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestAuxBothMsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= types.Coin; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewAuxBothMessage()
			if !createEmpty {
				hdr.Round = round
				hdr.BinVal = binVal
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*AuxBothMessage)
			assert.Equal(t, round, hdr.Round)
			assert.Equal(t, binVal, hdr.BinVal)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestBVMsg0Serialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= 1; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewBVMessage0()
			if !createEmpty {
				hdr.Round = round
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*BVMessage0)
			assert.Equal(t, round, hdr.Round)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestBVMsg1Serialize(t *testing.T) {
	var round types.ConsensusRound = 22
	var binVal types.BinVal
	for ; binVal <= 1; binVal++ {
		createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
			hdr := NewBVMessage1()
			if !createEmpty {
				hdr.Round = round
			}
			return hdr
		}
		checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
			hdr := msg.GetBaseMsgHeader().(*BVMessage1)
			assert.Equal(t, round, hdr.Round)
		}
		internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
		internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	}
}

func TestCoinMsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 23
	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewCoinMessage()
		if !createEmpty {
			hdr.Round = round
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*CoinMessage)
		assert.Equal(t, round, hdr.Round)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestCoinPreMsgSerialize(t *testing.T) {
	var round types.ConsensusRound = 23
	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewCoinPreMessage()
		if !createEmpty {
			hdr.Round = round
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*CoinPreMessage)
		assert.Equal(t, round, hdr.Round)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestNetworkTestMsgSerialize(t *testing.T) {
	var id uint32 = 13
	hdr := &NetworkTestMessage{id, types.ConsensusInt(10)}
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	dser := &NetworkTestMessage{}
	_, err = dser.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, dser.ID, id)
	assert.Equal(t, dser.GetIndex(), types.ConsensusInt(10))
}

var mvinitproposal = []byte("a proposal is written here")
var mvinitround types.ConsensusRound = 10
var mvinitsupport = []byte("another proposal")
var mvinitsupportindex types.ConsensusInt = 9

func createMvInitMsg(createEmpty bool) messages.InternalSignedMsgHeader {
	hdr := NewMvInitMessage()
	if !createEmpty {
		hdr.Proposal = mvinitproposal
		hdr.Round = mvinitround
	}
	return hdr
}

func checkMvInitMsgFunc(t *testing.T) func(messages.InternalSignedMsgHeader) {
	return func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().GetBaseMsgHeader().(*MvInitMessage)
		assert.Equal(t, mvinitproposal, hdr.Proposal)
		assert.Equal(t, mvinitround, hdr.Round)
		idx, p, err := hdr.NeedsSMValidation(tstMsgIdxObj, 0)
		assert.Nil(t, err)
		assert.Equal(t, tstMsgIndex-1, idx.Index)
		assert.Equal(t, mvinitproposal, p)
	}
}

func TestMvInitMessageSerialize(t *testing.T) {
	internalTestSignedMsgSerialize(createMvInitMsg, checkMvInitMsgFunc(t), tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMvInitMsg, checkMvInitMsgFunc(t), tstMsgIndex, false, t)
}

func TestMvInitSupportMsgSerialize(t *testing.T) {
	proposalHashSupport := types.GetHash(mvinitsupport)

	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewMvInitSupportMessage()
		if !createEmpty {
			hdr.Proposal = mvinitproposal
			hdr.SupportedIndex = mvinitsupportindex
			hdr.SupportedHash = proposalHashSupport
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*MvInitSupportMessage)
		assert.Equal(t, mvinitproposal, hdr.Proposal)
		assert.Equal(t, mvinitsupportindex, hdr.SupportedIndex)
		assert.Equal(t, proposalHashSupport, hdr.SupportedHash)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

var tstMsgIdHash = types.ConsensusHash(types.GetHash([]byte("0")))

var mvMultiHashIdxs = []types.ConsensusID{
	types.ConsensusHash(types.GetHash([]byte("1"))),
	types.ConsensusHash(types.GetHash([]byte("2"))),
	types.ConsensusHash(types.GetHash([]byte("3")))}
var mvMultiProposals = [][]byte{[]byte("1"), []byte("2"), []byte("3")}

func TestMvMultiInitSupportMsgSerialize(t *testing.T) {
	mvHdrIdx, err := types.GenerateParentHash(tstMsgIdHash, mvMultiHashIdxs)
	assert.Nil(t, err)

	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewMvMultiInitMessage()
		if !createEmpty {
			hdr.Proposals = mvMultiProposals
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*MvMultiInitMessage)
		assert.Equal(t, mvMultiProposals, hdr.Proposals)
		for i, p := range mvMultiProposals {
			idx, pro, err := msg.NeedsSMValidation(mvHdrIdx, i)
			assert.Nil(t, err)
			assert.Equal(t, p, pro)
			assert.Equal(t, mvHdrIdx.Index, idx.Index)
		}
		_, _, err := msg.NeedsSMValidation(mvHdrIdx, len(mvMultiProposals))
		assert.Equal(t, err, types.ErrInvalidIndex)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIdHash, true, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestCombinedMessageSerailze(t *testing.T) {
	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		internal := createMvInitMsg(createEmpty)
		if createEmpty {
			return NewCombinedMessage(internal)
		} else {
			// create the internal message
			// break it into partials
			partialsInt, err := GeneratePartialMessageDirect(createMvInitMsg(false), mvinitround, 2)
			assert.Nil(t, err)

			// reconstruct the partials
			partials := make([]*PartialMessage, len(partialsInt))
			for i, nxt := range partialsInt {
				partials[i] = nxt.(*PartialMessage)
			}
			fullMsg, err := GenerateCombinedMessageBytes(partials)
			assert.Nil(t, err)

			hdrFunc := func(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, hid messages.HeaderID) (messages.MsgHeader, error) {
				if hid == messages.HdrMvInit {
					return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, createMvInitMsg(true)), nil
				}
				return nil, types.ErrInvalidHeader
			}

			combinedv1 := NewCombinedMessageFromPartial(partials[0], createMvInitMsg(false))

			combined, err := GenerateCombinedMessage(partials[0], fullMsg, hdrFunc,
				nil, &ec.Ecpub{})
			assert.Nil(t, err)
			assert.Equal(t, combinedv1, combined)

			return combined
		}
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMvInitMsgFunc(t), tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMvInitMsgFunc(t), tstMsgIndex, false, t)
}

func TestMvEchoMsgSerialize(t *testing.T) {
	var proposal = []byte("a proposal is written here")
	var round types.ConsensusRound = 10
	proposalHash := types.GetHash(proposal)

	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewMvEchoMessage()
		if !createEmpty {
			hdr.ProposalHash = proposalHash
			hdr.Round = round
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*MvEchoMessage)
		assert.Equal(t, proposalHash, hdr.ProposalHash)
		assert.Equal(t, round, hdr.Round)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestHashMsgSerialize(t *testing.T) {
	var proposal = []byte("a proposal is written here")
	proposalHash := types.GetHash(proposal)

	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewHashMessage()
		if !createEmpty {
			hdr.TheHash = proposalHash
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*HashMessage)
		assert.Equal(t, proposalHash, hdr.TheHash)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestMvCommitMsgSerialize(t *testing.T) {
	var proposal = []byte("a proposal is written here")
	var round types.ConsensusRound = 10
	proposalHash := types.GetHash(proposal)

	createMsgFunc := func(createEmpty bool) messages.InternalSignedMsgHeader {
		hdr := NewMvCommitMessage()
		if !createEmpty {
			hdr.ProposalHash = proposalHash
			hdr.Round = round
		}
		return hdr
	}
	checkMsgFunc := func(msg messages.InternalSignedMsgHeader) {
		hdr := msg.GetBaseMsgHeader().(*MvCommitMessage)
		assert.Equal(t, proposalHash, hdr.ProposalHash)
		assert.Equal(t, round, hdr.Round)
	}
	internalTestSignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
	internalTestUnsignedMsgSerialize(createMsgFunc, checkMsgFunc, tstMsgIndex, false, t)
}

func TestMvRequestRecoverMessageSerialize(t *testing.T) {
	var proposal = []byte("a proposal is written here")

	proposalHash := types.GetHash(proposal)
	hdr := &MvRequestRecoverMessage{}
	hdr.ProposalHash = proposalHash
	hdr.Index = tstMsgIdxObj

	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	dser := &MvRequestRecoverMessage{}

	_, err = (dser).Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, tstMsgIndex, dser.GetIndex().Index)
	assert.Equal(t, dser.ProposalHash, proposalHash)
}

func TestVRFProofMsgSerialize(t *testing.T) {
	bm := sig.BasicSignedMessage("some random message")
	priv, err := ec.NewEcpriv()
	if err != nil {
		panic(err)
	}
	index, proof := priv.(sig.VRFPriv).Evaluate(bm)

	hdr := NewVRFProofMessage(tstMsgIdxObj, proof)
	hdrs := make([]messages.MsgHeader, 1)
	hdrs[0] = hdr

	msg := messages.InitMsgSetup(hdrs, t)

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, hdr.GetID())

	newHdr := &VRFProofMessage{VRFProof: ec.VRFProof{}.New()}
	_, err = newHdr.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	assert.Equal(t, tstMsgIndex, newHdr.GetIndex().Index)

	newIdx, err := priv.GetPub().(sig.VRFPub).ProofToHash(bm, newHdr.VRFProof)
	assert.Nil(t, err)
	assert.Equal(t, index, newIdx)
}
