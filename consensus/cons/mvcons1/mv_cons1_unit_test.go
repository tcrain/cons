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

package mvcons1

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/stats"
	"testing"

	// "github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/testobjects"
	"github.com/tcrain/cons/consensus/types"
)

var mvConsTestProposal = []byte("This is a proposal")

var mvCons1UnitTestOptions = types.TestOptions{
	ConsType:      types.MvBinCons1Type,
	NumTotalProcs: config.ProcCount,
	IncludeProofs: false,
	StopOnCommit:  types.NextRound,
}

type mvCons1TestItems struct {
	cons.ConsTestItems
	bcons          *MvCons1
	msgState       *MessageState
	memberChecker  *consinterface.MemCheckers
	mainChannel    *testobjects.MockMainChannel
	forwardChecker consinterface.ForwardChecker
	gc             *generalconfig.GeneralConfig
}

var coordIndex int

func createMvConsTestItems(idx types.ConsensusIndex, to types.TestOptions) *mvCons1TestItems {
	sig.SetSleepValidate(false)
	privKeys, pubKeys := cons.MakeKeys(to)

	if to.RotateCord {
		coordIndex = 1
	}

	// sci := &MvCons1{}
	// sci.SetTestConfig(0, mvCons1UnitTestOptions)
	preHeader := make([]messages.MsgHeader, 1)
	preHeader[0] = messagetypes.NewConsMessage()
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs})
	gc := consinterface.CreateGeneralConfig(0, eis, stats.GetStatsObject(types.MvBinCons1Type, to.EncryptChannels),
		to, preHeader, privKeys[coordIndex])
	// sci.InitState(preHeader, privKeys[0], eis, nil)
	msgState := NewMvCons1MessageState(gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[0], gc).New(idx),
		SMC: memberchecker.NewNoSpecialMembers().New(idx)}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{})
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&MvCons1{}).GenerateNewItem(idx, consItems, mmc, nil,
		MvBinCons1Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*MvCons1)
	// sci.ResetState(idx, mmc, memberChecker, msgState, forwardChecker, nil)

	return &mvCons1TestItems{
		ConsTestItems: cons.ConsTestItems{
			PrivKeys: privKeys,
			PubKeys:  pubKeys},
		bcons:          sci,
		msgState:       msgState,
		memberChecker:  memberChecker,
		forwardChecker: forwardChecker,
		gc:             gc,
		mainChannel:    mmc}
}

func createMvSendItems(hdrType messages.HeaderID, idx types.ConsensusIndex, proposal []byte, bct cons.ConsTestItems, t *testing.T) []*channelinterface.DeserializedItem {
	ret := make([]*channelinterface.DeserializedItem, len(bct.PrivKeys))
	hash := types.GetHash(proposal)

	for i := range bct.PrivKeys {
		priv := bct.PrivKeys[i]
		var hdr, dser messages.InternalSignedMsgHeader
		switch hdrType {
		case messages.HdrMvInit:
			w := messagetypes.NewMvInitMessage()
			w.Proposal = proposal
			hdr = w
			dser = messagetypes.NewMvInitMessage()
		case messages.HdrMvEcho:
			w := messagetypes.NewMvEchoMessage()
			w.ProposalHash = hash
			hdr = w
			dser = messagetypes.NewMvEchoMessage()
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

		ret[i] = &channelinterface.DeserializedItem{
			Index:          idx,
			HeaderType:     hdrType,
			Header:         smd,
			IsDeserialized: true,
			Message:        encMsg}
	}
	return ret
}

// func TestMvCons1UnitMcFunc(t *testing.T) {
//	to := mvCons1UnitTestOptions
//	bct := createMvConsTestItems(1, to)
//	initItems := createMvSendItems(messages.HdrMvInit, 1, mvConsTestProposal, bct.consTestItems, t)

//	// valid
//	err := mvCons1MemberCheck(bct.memberChecker, initItems[0])
//	if err != nil {
//		t.Error(err)
//	}

//	// invalid sig
//	initItems[0].Header.(*messagetypes.MvInitMessage).SigItems[0].Sig.Corrupt()
//	err = mvCons1MemberCheck(bct.memberChecker, initItems[0])
//	if err == nil {
//		t.Error("Sig should be invalid")
//	}

//	echoItems := createMvSendItems(messages.HdrMvEcho, 1, mvConsTestProposal, bct.consTestItems, t)
//	mergeSigs(echoItems)
//	// all valid sigs
//	err = mvCons1MemberCheck(bct.memberChecker, echoItems[0])
//	if err != nil {
//		t.Error(err)
//	}
//	l := len(echoItems[0].Header.(*messagetypes.MvEchoMessage).SigItems)
//	if l != to.numTotalProcs {
//		t.Errorf("Had %v valid sigs, expected %v", l, to.numTotalProcs)
//	}

//	// 1 invalid sig
//	echoItems[1].Header.(*messagetypes.MvEchoMessage).SigItems[0].Sig.Corrupt()
//	err = mvCons1MemberCheck(bct.memberChecker, echoItems[1])
//	if err == nil {
//		t.Error("Sig should be invalid")
//	}
//	l = len(echoItems[1].Header.(*messagetypes.MvEchoMessage).SigItems)
//	if l != 0 {
//		t.Errorf("Had %v valid sigs, expected %v", l, 0)
//	}

//	// 1 invalid sig rest valid
//	err = mvCons1MemberCheck(bct.memberChecker, echoItems[0])
//	if err != nil {
//		t.Error(err)
//	}
//	l = len(echoItems[0].Header.(*messagetypes.MvEchoMessage).SigItems)
//	if l != to.numTotalProcs - 1 {
//		t.Errorf("Had %v valid sigs, expected %v", l, to.numTotalProcs - 1)
//	}

// }

func TestMvCons1UnitGotMsg(t *testing.T) {
	to := mvCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(idx, to)
	initItems := createMvSendItems(messages.HdrMvInit, idx, mvConsTestProposal, bct.ConsTestItems, t)

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
	bctInvalid := createMvConsTestItems(idx, to)
	echoItemsInvalid := createMvSendItems(messages.HdrMvEcho, idx, mvConsTestProposal, bctInvalid.ConsTestItems, t)
	cons.MergeSigsForTest(echoItemsInvalid)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, echoItemsInvalid[0], bct.gc, bct.memberChecker)
	if err != types.ErrNoValidSigs {
		t.Error(err)
	}

	echoItems := createMvSendItems(messages.HdrMvEcho, idx, mvConsTestProposal, bct.ConsTestItems, t)
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

func TestMvCons1UnitProcessMsg1(t *testing.T) {
	to := mvCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(idx, to)

	// proposal
	p := messagetypes.NewMvProposeMessage(idx, mvConsTestProposal)
	err := bct.bcons.GotProposal(p, bct.mainChannel)
	assert.Nil(t, err)
	testobjects.CheckInitMessage(bct.mainChannel, 1, 0, mvConsTestProposal, t)

	// Start with the proposal
	initItems := createMvSendItems(messages.HdrMvInit, idx, mvConsTestProposal, bct.ConsTestItems, t)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, initItems[coordIndex], bct.gc, bct.memberChecker)
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
	// we should send an echo
	testobjects.CheckEchoMessage(bct.mainChannel, 1, 0, types.GetHash(mvConsTestProposal), t)

	// Now the echo
	echoItems := createMvSendItems(messages.HdrMvEcho, idx, mvConsTestProposal, bct.ConsTestItems, t)
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
	// we should send an aux message
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, 1, t)

	// Now the BinCons - decide
	// round 1, all send 1
	auxProofItems := bincons1.CreateAuxProofItems(idx, 1, 1, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if !bct.bcons.HasDecided() {
		t.Error("should have decided")
	}
	_, decision, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, mvConsTestProposal) {
		t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
	}
	// we should have sent nothing
	testobjects.CheckNoSend(bct.mainChannel, t)

	// This should cause an invalid decision
	auxProofItems = bincons1.CreateAuxProofItems(idx, 1, 0, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Should have paniced")
			}
			t.Log(r)
		}()
		progress, forward = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	}()
}

func TestMvCons1UnitProcessMsgNoProposal(t *testing.T) {
	to := mvCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(idx, to)

	// Skip with the proposal
	// Now the echo
	echoItems := createMvSendItems(messages.HdrMvEcho, idx, mvConsTestProposal, bct.ConsTestItems, t)
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
	// we should send an aux message
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, 1, t)

	// Now the BinCons - decide
	// round 1, all send 1
	auxProofItems := bincons1.CreateAuxProofItems(idx, 1, 1, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided")
	}
	// TODO check we started timeout for recover request
	// should have sent nothing
	testobjects.CheckNoSend(bct.mainChannel, t)

	// Send a fake proposal
	badProposal := append(mvConsTestProposal, []byte("bad")...)
	initItems := createMvSendItems(messages.HdrMvInit, idx, badProposal, bct.ConsTestItems, t)
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
	initItems = createMvSendItems(messages.HdrMvInit, idx, mvConsTestProposal, bct.ConsTestItems, t)
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
	_, decision, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, mvConsTestProposal) {
		t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
	}
}

func TestMvCons1UnitProcessMsg0(t *testing.T) {
	to := mvCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createMvConsTestItems(idx, to)

	// No proposal
	// Just the echo timeout
	deser := &channelinterface.DeserializedItem{
		Index:          idx,
		HeaderType:     messages.HdrMvEchoTimeout,
		IsLocal:        types.LocalMessage,
		IsDeserialized: true}
	progress, forward := bct.bcons.ProcessMessage(deser, true, nil)
	if !progress || forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}

	// Now the BinCons - decide 0
	// round 1, all send 0
	auxProofItems := bincons1.CreateAuxProofItems(idx, 1, 0, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if bct.bcons.HasDecided() {
		t.Error("should not have decided")
	}
	// we should have all sent 0
	// first 0 for round 1 since we didn't send the echo message
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, 0, t)
	// now 0 for round 2 since we got n-t 0 echo messages for round 1
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 2, 0, t)

	// round 2, all send 0
	auxProofItems = bincons1.CreateAuxProofItems(idx, 2, 0, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	if !bct.bcons.HasDecided() {
		t.Error("should have decided")
	}
	// should have not sent
	testobjects.CheckNoSend(bct.mainChannel, t)

	_, decision, _ := bct.bcons.GetDecision()
	if len(decision) != 0 {
		t.Errorf("should have decided len 0, but decided %v", decision)
	}
}
