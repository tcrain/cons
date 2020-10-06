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

package mvcons2

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/stats"
	"testing"

	// "github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/testobjects"
	"github.com/tcrain/cons/consensus/types"
)

var mvConsTestProposal = []byte("This is a proposal")
var mvConsOtherTestProposal = []byte("This is a different proposal")

var mvCons2UnitTestOptions = types.TestOptions{
	NumTotalProcs: config.ProcCount,
	IncludeProofs: false,
}

var coordIndex int

type mvCons1TestItems struct {
	cons.ConsTestItems
	bcons          *MvCons2
	msgState       *MessageState
	memberChecker  *consinterface.MemCheckers
	mainChannel    *testobjects.MockMainChannel
	forwardChecker consinterface.ForwardChecker
	gc             *generalconfig.GeneralConfig
}

func createMvConsTestItems(keyIndex int, privKeys []sig.Priv, pubKeys []sig.Pub, idx types.ConsensusIndex,
	to types.TestOptions) *mvCons1TestItems {
	sig.SetSleepValidate(false)
	if to.RotateCord {
		coordIndex = 1
	}

	preHeader := make([]messages.MsgHeader, 1)
	preHeader[0] = messagetypes.NewConsMessage()
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs})
	gc := consinterface.CreateGeneralConfig(keyIndex, eis, stats.GetStatsObject(types.MvCons2Type, to.EncryptChannels),
		to, preHeader, privKeys[keyIndex])
	msgState := NewMvCons2MessageState(gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[keyIndex], gc).New(idx),
		SMC: memberchecker.NewNoSpecialMembers().New(idx)}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{}, nil)
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&MvCons2{}).GenerateNewItem(idx, consItems, mmc, nil,
		MvCons2Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*MvCons2)

	return &mvCons1TestItems{
		ConsTestItems: cons.ConsTestItems{
			PrivKeys: privKeys,
			PubKeys:  pubKeys},
		bcons:          sci,
		msgState:       msgState,
		gc:             gc,
		memberChecker:  memberChecker,
		forwardChecker: forwardChecker,
		mainChannel:    mmc}
}

func createMvSendItems(hdrType messages.HeaderID, idx types.ConsensusIndex, round types.ConsensusRound, proposal []byte,
	bct cons.ConsTestItems, t *testing.T) []*deserialized.DeserializedItem {

	ret := make([]*deserialized.DeserializedItem, len(bct.PrivKeys))
	var hash types.HashBytes
	if proposal == nil {
		hash = types.GetZeroBytesHashLength()
	} else {
		hash = types.GetHash(proposal)
	}

	for i := range bct.PrivKeys {
		priv := bct.PrivKeys[i]
		var hdr, dser messages.InternalSignedMsgHeader
		switch hdrType {
		case messages.HdrMvInit:
			w := messagetypes.NewMvInitMessage()
			w.Proposal = proposal
			w.Round = round
			hdr = w
			dser = messagetypes.NewMvInitMessage()
		case messages.HdrMvEcho:
			w := messagetypes.NewMvEchoMessage()
			w.ProposalHash = hash
			w.Round = round
			hdr = w
			dser = messagetypes.NewMvEchoMessage()
		case messages.HdrMvCommit:
			w := messagetypes.NewMvCommitMessage()
			w.ProposalHash = hash
			w.Round = round
			hdr = w
			dser = messagetypes.NewMvCommitMessage()
		default:
			panic("invalid type")
		}
		sm := sig.NewMultipleSignedMsg(idx, priv.GetPub(), hdr)
		if _, err := sm.Serialize(messages.NewMessage(nil)); err != nil {
			panic(err)
		}
		mySig, err := priv.GenerateSig(sm, nil, sm.GetSignType())
		if err != nil {
			panic(err)
		}
		sm.SetSigItems([]*sig.SigItem{mySig})

		// serialize
		hdrs := make([]messages.MsgHeader, 1)
		hdrs[0] = sm
		msg := messages.InitMsgSetup(hdrs, t)
		encMsg := sig.NewUnencodedMsg(msg)

		// deserialize
		smd := sig.NewMultipleSignedMsg(types.ConsensusIndex{}, priv.GetPub(), dser)
		_, err = smd.Deserialize(msg, types.IntIndexFuns)
		if err != nil {
			t.Error(err)
		}

		ret[i] = &deserialized.DeserializedItem{
			Index:          idx,
			HeaderType:     hdrType,
			Header:         smd,
			IsDeserialized: true,
			Message:        encMsg}
	}
	return ret
}

func TestMvCons2UnitGotMsg(t *testing.T) {
	to := mvCons2UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	privKeys, pubKeys := cons.MakeKeys(to)
	bct := createMvConsTestItems(coordIndex, privKeys, pubKeys, idx, to)
	initItems := createMvSendItems(messages.HdrMvInit, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)

	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}

	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
	if err != types.ErrNoNewSigs {
		t.Error(err)
	}

	// Invalid coord
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex+1], bct.gc, bct.memberChecker)
	if err != types.ErrInvalidRoundCoord {
		t.Error(err)
	}

	// Non member
	privKeysInv, pubKeysInv := cons.MakeKeys(to)
	bctInvalid := createMvConsTestItems(0, privKeysInv, pubKeysInv, idx, to)
	echoItemsInvalid := createMvSendItems(messages.HdrMvEcho, idx, 0, mvConsTestProposal, bctInvalid.ConsTestItems, t)
	cons.MergeSigsForTest(echoItemsInvalid)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, echoItemsInvalid[0], bct.gc, bct.memberChecker)
	if err != types.ErrNoValidSigs {
		t.Error(err)
	}

	echoItems := createMvSendItems(messages.HdrMvEcho, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(echoItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, echoItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}

	// repeat message
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, echoItems[0], bct.gc, bct.memberChecker)
	if err != types.ErrNoNewSigs {
		t.Error(err)
	}
}

func TestMvCons2UnitProcessMsg1(t *testing.T) {
	to := mvCons2UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(coordIndex, privKeys, pubKeys, idx, to)

	// proposal
	p := messagetypes.NewMvProposeMessage(idx, mvConsTestProposal)
	bct.bcons.Start()
	assert.Nil(t, bct.bcons.GotProposal(p, bct.mainChannel))
	testobjects.CheckInitMessage(bct.mainChannel, 1, 0, mvConsTestProposal, t)

	// Start with the proposal
	initItems := createMvSendItems(messages.HdrMvInit, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward := bct.bcons.ProcessMessage(initItems[coordIndex], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}
	// should have sent an echo
	testobjects.CheckEchoMessage(bct.mainChannel, 1, 0, types.GetHash(mvConsTestProposal), t)

	// Now the echo
	echoItems := createMvSendItems(messages.HdrMvEcho, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(echoItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, echoItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(echoItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}
	// should have sent an commit
	testobjects.CheckCommitMessage(bct.mainChannel, 1, 0, types.GetHash(mvConsTestProposal), t)

	// Now the commit, should decide
	commitItems := createMvSendItems(messages.HdrMvCommit, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(commitItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, commitItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(commitItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if !bct.bcons.HasDecided() {
		t.Error("should have decided")
	}
	_, decision, _, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, mvConsTestProposal) {
		t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
	}
	// should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)

	// This should cause an invalid decision
	otherCommitItems := createMvSendItems(messages.HdrMvCommit, idx, 0, mvConsOtherTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(otherCommitItems)
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Should have paniced")
			}
			t.Log(r)
		}()
		_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, otherCommitItems[0], bct.gc, bct.memberChecker)
	}()
}

func TestMvCons2UnitProcessMsgNoProposal(t *testing.T) {
	to := mvCons2UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(coordIndex, privKeys, pubKeys, idx, to)

	// Skip with the proposal
	// Now the echo
	echoItems := createMvSendItems(messages.HdrMvEcho, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(echoItems)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, echoItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward := bct.bcons.ProcessMessage(echoItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}
	// should have sent commit message and echo message
	testobjects.CheckEchoMessage(bct.mainChannel, 1, 0, types.GetHash(mvConsTestProposal), t)
	testobjects.CheckCommitMessage(bct.mainChannel, 1, 0, types.GetHash(mvConsTestProposal), t)

	// Now the commit, should not decide because not init
	commitItems := createMvSendItems(messages.HdrMvCommit, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	cons.MergeSigsForTest(commitItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, commitItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(commitItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	// should not have sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)
	// TODO check recover timeout has started

	if bct.bcons.HasDecided() {
		t.Error("should not have decided")
	}

	// Send a fake proposal
	badProposal := append(mvConsTestProposal, []byte("bad")...)
	initItems := createMvSendItems(messages.HdrMvInit, idx, 0, badProposal, bct.ConsTestItems, t)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(initItems[coordIndex], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}

	// Send the correct proposal
	initItems = createMvSendItems(messages.HdrMvInit, idx, 0, mvConsTestProposal, bct.ConsTestItems, t)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
	if err != nil {

		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(initItems[coordIndex], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if !bct.bcons.HasDecided() {
		t.Error("should have decided")
	}
	_, decision, _, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, mvConsTestProposal) {
		t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
	}
}

func TestMvCons2UnitProcessMsg0(t *testing.T) {
	to := mvCons2UnitTestOptions
	var bct []*mvCons1TestItems
	privKeys, pubKeys := cons.MakeKeys(to)
	idx := types.SingleComputeConsensusIDShort(1)
	for i := coordIndex; i < coordIndex+2; i++ {
		bct = append(bct, createMvConsTestItems(i, privKeys, pubKeys, idx, to))
	}

	if bct == nil {
		panic(bct)
	}

	bct[1].bcons.Start()

	// No proposal
	// No echos
	// Just the echo timeout, this can happen when we didn't get enough equal echos before the timeout
	deser := &deserialized.DeserializedItem{
		Index:          idx,
		HeaderType:     messages.HdrMvEchoTimeout,
		Header:         (messagetypes.MvEchoMessageTimeout)(0),
		IsLocal:        types.LocalMessage,
		IsDeserialized: true}
	for i := 0; i < 2; i++ {
		progress, forward := bct[i].bcons.ProcessMessage(deser, true, nil)
		if !progress || forward {
			t.Error("should have valid message", progress, forward)
		}
		if bct[i].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckNoSend(bct[i].mainChannel, t)
	}

	for i := 0; i < 2; i++ {
		// commit timeout so we advance to round 1
		// Just the commit timeout
		deser = &deserialized.DeserializedItem{
			Index:          idx,
			HeaderType:     messages.HdrMvCommitTimeout,
			Header:         (messagetypes.MvCommitMessageTimeout)(0),
			IsLocal:        types.LocalMessage,
			IsDeserialized: true}
		progress, forward := bct[i].bcons.ProcessMessage(deser, true, nil)
		if !progress || forward {
			t.Error("should have valid message")
		}
		if bct[i].bcons.round != 0 {
			t.Error("should still  be on round 0")
		}
		if bct[i].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckCommitMessage(bct[i].mainChannel, 1, 0, types.GetZeroBytesHashLength(), t)
	}

	for i := 0; i < 2; i++ {
		// Now the commit - all send zero hash so we advance to next round
		commitItems := createMvSendItems(messages.HdrMvCommit, idx, 0, nil, bct[i].ConsTestItems, t)
		cons.MergeSigsForTest(commitItems)
		_, err := bct[i].msgState.GotMsg(bct[i].bcons.GetHeader, commitItems[0], bct[i].gc, bct[i].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct[i].bcons.ProcessMessage(commitItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i].bcons.HasDecided() {
			t.Error("should not have decided")
		}
		if bct[i].bcons.round != 1 {
			t.Error("should have reached round 1")
		}
		if i != 1 {
			testobjects.CheckNoSend(bct[i].mainChannel, t)
		} else {
			// proposal only at node 1
			p := messagetypes.NewMvProposeMessage(idx, mvConsTestProposal)
			assert.Nil(t, bct[i].bcons.GotProposal(p, bct[i].mainChannel))

			// first should be a commit message proving that we can send our own init message
			testobjects.CheckCommitMessage(bct[i].mainChannel, 1, 0, types.GetZeroBytesHashLength(), t)

			// now should be our own init message
			testobjects.CheckInitMessage(bct[i].mainChannel, 1, 1, mvConsTestProposal, t)
		}
	}

	for i := 0; i < 2; i++ {
		// round 1, send init, echo and commit so we decide
		// Start with the proposal
		initItems := createMvSendItems(messages.HdrMvInit, idx, 1, mvConsTestProposal, bct[i].ConsTestItems, t)
		_, err := bct[i].msgState.GotMsg(bct[i].bcons.GetHeader, initItems[coordIndex+1], bct[i].gc, bct[i].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct[i].bcons.ProcessMessage(initItems[coordIndex+1], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckEchoMessage(bct[i].mainChannel, 1, 1, types.GetHash(mvConsTestProposal), t)
	}

	for i := 0; i < 2; i++ {
		// Now the echo
		echoItems := createMvSendItems(messages.HdrMvEcho, idx, 1, mvConsTestProposal, bct[i].ConsTestItems, t)
		cons.MergeSigsForTest(echoItems)
		_, err := bct[i].msgState.GotMsg(bct[i].bcons.GetHeader, echoItems[0], bct[i].gc, bct[i].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct[i].bcons.ProcessMessage(echoItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckCommitMessage(bct[i].mainChannel, 1, 1, types.GetHash(mvConsTestProposal), t)
	}

	for i := 0; i < 2; i++ {
		// Now the commit, should decide
		commitItems := createMvSendItems(messages.HdrMvCommit, idx, 1, mvConsTestProposal, bct[i].ConsTestItems, t)
		cons.MergeSigsForTest(commitItems)
		_, err := bct[i].msgState.GotMsg(bct[i].bcons.GetHeader, commitItems[0], bct[i].gc, bct[i].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct[i].bcons.ProcessMessage(commitItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if !bct[i].bcons.HasDecided() {
			t.Error("should have decided")
		}
		_, decision, _, _ := bct[i].bcons.GetDecision()
		if !bytes.Equal(decision, mvConsTestProposal) {
			t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
		}
		testobjects.CheckNoSend(bct[i].mainChannel, t)
	}
}
