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

package binconsrnd1

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
	NumTotalProcs:       config.ProcCount,
	IncludeProofs:       false,
	SigType:             types.TBLS,
	StopOnCommit:        types.NextRound,
	CoinType:            types.StrongCoin1Type,
	UseFixedCoinPresets: false,
}

type binCons1TestItems struct {
	cons.ConsTestItems
	bcons          *BinConsRnd1
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
	gc := consinterface.CreateGeneralConfig(0, eis, stats.GetStatsObject(
		types.BinConsRnd1Type, to.EncryptChannels), to, preHeader, privKeys[0])
	gc.AllowSupportCoin = true
	msgState := NewBinConsRnd1MessageState(false, gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[0], gc).New(idx),
		SMC: memberchecker.NewThrshSigMemChecker([]sig.Pub{privKeys[0].(sig.BasicThresholdInterface).GetSharedPub()}).New(idx)}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{})
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&BinConsRnd1{}).GenerateNewItem(idx, consItems, mmc, nil,
		Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*BinConsRnd1)

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

func TestBinConsRnd1UnitGotMsg(t *testing.T) {
	sig.SetSleepValidate(false)
	for _, supportCoin := range []bool{false, true} {
		idx := types.SingleComputeConsensusIDShort(1)
		to := binCons1UnitTestOptions
		bct := createBinConsTestItems(idx, to)
		auxProofItems := CreateAuxProofItems(idx, 0, supportCoin, 1, bct.ConsTestItems, t)
		cons.MergeSigsForTest(auxProofItems)

		// Test with some non-member sigs
		bctInvalid := createBinConsTestItems(idx, to)
		auxProofInvalidItems := CreateAuxProofItems(idx, 0, false, 1, bctInvalid.ConsTestItems, t)
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

func TestBinConsRnd1UnitProcessMsg1(t *testing.T) {

	var proposal types.BinVal
	for _, supportCoin := range []bool{false, true} {
		for proposal = 0; proposal <= 1; proposal++ {
			var passedMultiRound, passedSingleRound bool
			for !passedMultiRound || !passedSingleRound {
				if runBinConsBasicTest(proposal, supportCoin, t) > 1 {
					passedMultiRound = true
				} else {
					passedSingleRound = true
				}
			}
		}
	}
}

func runBinConsBasicTest(proposal types.BinVal, supportCoin bool, t *testing.T) types.ConsensusRound {
	to := binCons1UnitTestOptions
	idx := types.SingleComputeConsensusIDShort(1)
	bct := createBinConsTestItems(idx, to)

	notProposal := 1 - proposal

	// proposal
	p := messagetypes.NewBinProposeMessage(idx, proposal)
	bct.bcons.Start()
	assert.Nil(t, bct.bcons.GotProposal(p, bct.mainChannel))
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 0, proposal, t)

	// round 0, all send proposal
	auxProofItems := CreateAuxProofItems(idx, 0, supportCoin, proposal, bct.ConsTestItems, t)
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
	testobjects.CheckAuxMessage(bct.mainChannel, 1, 1, proposal, t)

	// loop until coin is proposal
	var round types.ConsensusRound = 1
	for true {
		// all send 1
		auxProofItems = CreateAuxProofItems(idx, round, bct.gc.AllowSupportCoin, proposal, bct.ConsTestItems, t)
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
			t.Error("should not have decided before coin")
		}
		// We sould have sent the coin
		testobjects.CheckCoinMessage(bct.mainChannel, 1, round, t)
		// we should have not sent anything
		if !bct.gc.AllowSupportCoin {
			testobjects.CheckNoSend(bct.mainChannel, t)
		} else {
			testobjects.CheckAuxMessage(bct.mainChannel, 1, round+1, proposal, t)
		}

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
		coins := bct.msgState.coinState.GetCoins(round)
		roundStruct := bct.msgState.getAuxRoundStruct(round, bct.memberChecker)
		assert.Equal(t, 1, len(coins))

		// We are done if the coin was proposal
		if coins[0] == proposal {
			break
		}
		assert.True(t, roundStruct.sentProposal)

		if bct.bcons.HasDecided() {
			t.Error("should not have decided before coin")
		}

		// we should have sent proposal for the next round
		round++
		if !bct.gc.AllowSupportCoin {
			testobjects.CheckAuxMessage(bct.mainChannel, 1, round, proposal, t)
		} else {
			testobjects.CheckNoSend(bct.mainChannel, t)
		}
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
	auxProofItems = CreateAuxProofItems(idx, round, false, notProposal, bct.ConsTestItems, t)
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
	return round
}
