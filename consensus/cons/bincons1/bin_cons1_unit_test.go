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

package bincons1

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/stats"
	"testing"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/testobjects"
	"github.com/tcrain/cons/consensus/types"
)

var binCons1UnitTestOptions = types.TestOptions{
	NumTotalProcs: config.ProcCount,
	IncludeProofs: false,
	StopOnCommit:  types.NextRound,
}

type binCons1TestItems struct {
	cons.ConsTestItems
	bcons          *BinCons1
	msgState       *MessageState
	memberChecker  *consinterface.MemCheckers
	mainChannel    *testobjects.MockMainChannel
	forwardChecker consinterface.ForwardChecker
	gc             *generalconfig.GeneralConfig
}

func createBinConsTestItems(idx types.ConsensusIndex, to types.TestOptions) *binCons1TestItems {
	sig.SetSleepValidate(false)
	privKeys, pubKeys := cons.MakeKeys(to)

	// sci.SetTestConfig(0, binCons1UnitTestOptions)
	preHeader := make([]messages.MsgHeader, 0)
	// preHeader[0] = messagetypes.NewConsMessage()
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs})
	gc := consinterface.CreateGeneralConfig(0, eis,
		stats.GetStatsObject(types.BinCons1Type, to.EncryptChannels), to, preHeader, privKeys[0])
	msgState := NewBinCons1MessageState(false, gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[0], gc).New(idx),
		SMC: memberchecker.NewNoSpecialMembers().New(idx)}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{}, nil)
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&BinCons1{}).GenerateNewItem(idx, consItems, mmc, nil,
		Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*BinCons1)

	return &binCons1TestItems{
		ConsTestItems: cons.ConsTestItems{
			PrivKeys: privKeys,
			PubKeys:  pubKeys},
		bcons:          sci,
		msgState:       msgState,
		memberChecker:  memberChecker,
		mainChannel:    mmc,
		gc:             gc,
		forwardChecker: forwardChecker}
}

// func TestBinCons1UnitMcFunc(t *testing.T) {
// 	to := binCons1UnitTestOptions
// 	bct := createBinConsTestItems(1, to)
// 	auxProofItems := CreateAuxProofItems(1, 0, 1, bct.consTestItems, t)
// 	mergeSigs(auxProofItems)

// 	// all valid sigs
// 	err := binCons1MemberCheck(bct.memberChecker, auxProofItems[0])
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	l := len(auxProofItems[0].Header.(*messagetypes.AuxProofMessage).SigItems)
// 	if l != to.numTotalProcs {
// 		t.Errorf("Had %v valid sigs, expected %v", l, to.numTotalProcs)
// 	}

// 	// 1 invalid sig
// 	auxProofItems[1].Header.(*messagetypes.AuxProofMessage).SigItems[0].Sig.Corrupt()
// 	err = binCons1MemberCheck(bct.memberChecker, auxProofItems[1])
// 	if err == nil {
// 		t.Error("Sig should be invalid")
// 	}
// 	l = len(auxProofItems[1].Header.(*messagetypes.AuxProofMessage).SigItems)
// 	if l != 0 {
// 		t.Errorf("Had %v valid sigs, expected %v", l, 0)
// 	}

// 	// 1 invalid sig rest valid
// 	err = binCons1MemberCheck(bct.memberChecker, auxProofItems[0])
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	l = len(auxProofItems[0].Header.(*messagetypes.AuxProofMessage).SigItems)
// 	if l != to.numTotalProcs - 1 {
// 		t.Errorf("Had %v valid sigs, expected %v", l, to.numTotalProcs - 1)
// 	}
// }

func TestBinCons1UnitGotMsg(t *testing.T) {
	idx := types.SingleComputeConsensusIDShort(1)
	to := binCons1UnitTestOptions
	bct := createBinConsTestItems(idx, to)
	auxProofItems := CreateAuxProofItems(idx, 0, 1, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)

	// Test with some non-member sigs
	bctInvalid := createBinConsTestItems(idx, to)
	auxProofInvalidItems := CreateAuxProofItems(idx, 0, 1, bctInvalid.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofInvalidItems)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofInvalidItems[0], bct.gc, bct.memberChecker)
	if err != types.ErrNoValidSigs {
		t.Error(err)
	}

	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}

	// repeat message
	_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != types.ErrNoNewSigs {
		t.Error(err)
	}
}

func TestBinCons1UnitProcessMsg1(t *testing.T) {
	to := binCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createBinConsTestItems(idx, to)

	// proposal 1
	p := messagetypes.NewBinProposeMessage(idx, 1)
	bct.bcons.Start(false)
	assert.Nil(t, bct.bcons.GotProposal(p, bct.mainChannel))
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 0, 1, t)

	// round 0, all send 1
	auxProofItems := CreateAuxProofItems(idx, 0, 1, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward := bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}

	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}
	// we should have sent 1
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, 1, t)

	// round 1, all send 1
	auxProofItems = CreateAuxProofItems(idx, 1, 1, bct.ConsTestItems, t)
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
	_, decision, _, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, []byte{1}) {
		t.Errorf("should have decided 1, but decided %v", decision)
	}
	// we should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)

	// Be sure 0 is not valid
	n := bct.memberChecker.MC.GetMemberCount()
	f := bct.memberChecker.MC.GetFaultCount()
	valids := bct.msgState.getValids(n-f, f, 1)
	if valids[0] {
		t.Error("0 should not be valid")
	}
	if !valids[1] {
		t.Error("1 should be valid")
	}

	// This should cause an invalid decision
	auxProofItems = CreateAuxProofItems(idx, 1, 0, bct.ConsTestItems, t)
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
	// we should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)
}

func TestBinCons1UnitProcessMsg0(t *testing.T) {
	to := binCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createBinConsTestItems(idx, to)

	// proposal 0
	bct.bcons.Start(false)
	p := messagetypes.NewBinProposeMessage(idx, 0)
	assert.Nil(t, bct.bcons.GotProposal(p, bct.mainChannel))
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 0, 0, t)

	// round 0, all send 0
	auxProofItems := CreateAuxProofItems(idx, 0, 0, bct.ConsTestItems, t)
	cons.MergeSigsForTest(auxProofItems)
	_, err := bct.msgState.GotMsg(bct.bcons.GetHeader, auxProofItems[0], bct.gc, bct.memberChecker)
	if err != nil {
		t.Error(err)
	}
	progress, forward := bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	if !progress || !forward {
		t.Error("should have valid message")
	}
	// we should have sent 0
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, 0, t)

	if bct.bcons.HasDecided() {
		t.Error("should not have decided yet")
	}

	// round 1, all send 0
	auxProofItems = CreateAuxProofItems(idx, 1, 0, bct.ConsTestItems, t)
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
	// we should have sent 0
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 2, 0, t)

	// Be sure 1 is not valid
	n := bct.memberChecker.MC.GetMemberCount()
	f := bct.memberChecker.MC.GetFaultCount()
	valids := bct.msgState.getValids(n-f, f, 1)
	if !valids[0] {
		t.Error("0 should be valid")
	}
	if valids[1] {
		t.Error("1 should not be valid")
	}

	// round 2, all send 0
	auxProofItems = CreateAuxProofItems(idx, 2, 0, bct.ConsTestItems, t)
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
	_, decision, _, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, []byte{0}) {
		t.Errorf("should have decided 0, but decided %v", decision)
	}
	// we should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)

	// Be sure 1 is not valid
	valids = bct.msgState.getValids(n-f, f, 2)
	if !valids[0] {
		t.Error("0 should be valid")
	}
	if valids[1] {
		t.Error("1 should not be valid")
	}

	// This should cause an invalid decision
	auxProofItems = CreateAuxProofItems(idx, 2, 1, bct.ConsTestItems, t)
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

// TODO add timer round test
