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
package binconsrnd2

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

// TODO add test where need to use binVal=coin

var binCons1UnitTestOptions = types.TestOptions{
	NumTotalProcs: config.ProcCount,
	IncludeProofs: false,
	SigType:       types.EDCOIN,
	StopOnCommit:  types.NextRound,
	CoinType:      types.StrongCoin2Type,
}

type binCons1TestItems struct {
	cons.ConsTestItems
	bcons          *BinConsRnd2
	msgState       *MessageState
	memberChecker  *consinterface.MemCheckers
	mainChannel    *testobjects.MockMainChannel
	forwardChecker consinterface.ForwardChecker
	gc             *generalconfig.GeneralConfig
}

func createBinConsTestItems(idx types.ConsensusIndex, to types.TestOptions) *binCons1TestItems {
	privKeys, pubKeys := cons.MakeKeys(to)

	// sci.SetTestConfig(0, binCons1UnitTestOptions)
	preHeader := make([]messages.MsgHeader, 0)
	// preHeader[0] = messagetypes.NewConsMessage()
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs})
	gc := consinterface.CreateGeneralConfig(0, eis, stats.GetStatsObject(
		types.BinConsRnd2Type, to.EncryptChannels), to, preHeader, privKeys[0])
	gc.AllowSupportCoin = true
	msgState := NewBinConsRnd2MessageState(false, gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[0], gc).New(idx),
		SMC: memberchecker.NewThrshSigMemChecker([]sig.Pub{privKeys[0].(sig.BasicThresholdInterface).GetSharedPub()})}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{})
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&BinConsRnd2{}).GenerateNewItem(idx, consItems, mmc, nil,
		Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*BinConsRnd2)

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

func TestBinConsRnd2UnitGotMsg(t *testing.T) {
	for _, supportCoin := range []bool{false, true} {
		idx := types.SingleComputeConsensusIDShort(1)
		to := binCons1UnitTestOptions
		bct := createBinConsTestItems(idx, to)
		auxProofItems := CreateAuxProofItems(idx, 1, supportCoin, 1, bct.ConsTestItems, t)
		cons.MergeSigsForTest(auxProofItems)

		// Test with some non-member sigs
		bctInvalid := createBinConsTestItems(idx, to)
		auxProofInvalidItems := CreateAuxProofItems(idx, 1, false, 1, bctInvalid.ConsTestItems, t)
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
}

func TestBinConsRnd2UnitProcessMsg1(t *testing.T) {

	var proposal types.BinVal
	for proposal = 0; proposal <= 1; proposal++ {
		var passedMultiRound, passedSingleRound bool
		for !passedMultiRound || !passedSingleRound {
			if runBinConsBasicTest(proposal, t) > 1 {
				passedMultiRound = true
			} else {
				passedSingleRound = true
			}
		}
	}
}

func runBinConsBasicTest(proposal types.BinVal, t *testing.T) types.ConsensusRound {
	to := binCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createBinConsTestItems(idx, to)

	notProposal := 1 - proposal

	// proposal
	p := messagetypes.NewBinProposeMessage(idx, proposal)
	bct.bcons.Start()
	err := bct.bcons.GotProposal(p, bct.mainChannel)
	assert.Nil(t, err)

	// loop until coin is proposal
	var round types.ConsensusRound
	round++
	for true {
		// should have sent BV message
		testobjects.CheckBVMessage(bct.mainChannel, 1, round, proposal, t)

		// round 0, all send proposal
		bvItems := CreateBVItems(idx, round, proposal, bct.ConsTestItems, t)
		cons.MergeSigsForTest(bvItems)
		_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, bvItems[0], bct.gc, bct.memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct.bcons.ProcessMessage(bvItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}

		if bct.bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		// we should have sent aux message
		testobjects.CheckAuxMessage(bct.mainChannel, 1, round, proposal, t)

		// all send aux message
		auxProofItems := CreateAuxProofItems(idx, round, false, proposal, bct.ConsTestItems, t)
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
			t.Error("should not have decided yet")
		}
		// we should have sent coin
		testobjects.CheckCoinMessage2(bct.mainChannel, 1, round, t)

		// create the coin
		coinItems := CreateCoinItems(idx, round, bct.ConsTestItems, t)
		cons.MergeSigsForTest(coinItems)
		_, err = bct.msgState.GotMsg(bct.bcons.GetHeader, coinItems[0], bct.gc, bct.memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward = bct.bcons.ProcessMessage(coinItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		roundStruct := bct.msgState.getAuxRoundStruct(round, bct.memberChecker)
		assert.True(t, roundStruct.gotCoin)
		assert.True(t, roundStruct.sentCoin)

		// We are done if the coin was proposal
		if roundStruct.coinVal == proposal {
			break
		}
		if round == 1 {
			assert.True(t, bct.bcons.sentBVR0[proposal])
		} else {
			// assert.True(t, roundStruct.sentProposal)
		}

		if bct.bcons.HasDecided() {
			t.Error("should not have decided before coin")
		}

		// we should have sent proposal for the next round
		round++
		// testobjects.CheckNoSend(bct.mainChannel, t)

	}

	if !bct.bcons.HasDecided() {
		t.Error("should have decided")
	}

	_, decision, _ := bct.bcons.GetDecision()
	if !bytes.Equal(decision, []byte{byte(proposal)}) {
		t.Errorf("should have decided proposal, but decided %v", decision)
	}
	// we should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)

	// Be sure not proposal is not valid
	n := bct.memberChecker.MC.GetMemberCount()
	f := bct.memberChecker.MC.GetFaultCount()
	valids := bct.msgState.getValids(n-f, f, round, bct.memberChecker)
	if valids[notProposal] {
		t.Error("not proposal should not be valid")
	}
	if !valids[proposal] {
		t.Error("proposal should not be valid")
	}

	// This should cause an invalid decision
	auxProofItems := CreateAuxProofItems(idx, round, false, notProposal, bct.ConsTestItems, t)
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
		_, _ = bct.bcons.ProcessMessage(auxProofItems[0], false, nil)
	}()
	// we should have not sent anything
	testobjects.CheckNoSend(bct.mainChannel, t)
	return round
}
