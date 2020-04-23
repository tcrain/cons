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

package mvcons3

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
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/testobjects"
	"github.com/tcrain/cons/consensus/types"
)

var mvConsTestProposal = []byte("This is a proposal")

// var mvConsOtherTestProposal = []byte("This is a different proposal")

var mvCons3UnitTestOptions = types.TestOptions{
	NumTotalProcs: config.ProcCount,
	IncludeProofs: false,
}

type mvCons1TestItems struct {
	cons.ConsTestItems
	bcons          *MvCons3
	msgState       *MessageState
	memberChecker  *consinterface.MemCheckers
	mainChannel    *testobjects.MockMainChannel
	forwardChecker consinterface.ForwardChecker
	gc             *generalconfig.GeneralConfig
}

func createMvConsTestItems(privKeys []sig.Priv, pubKeys []sig.Pub, privKeyIdx int, idx types.ConsensusIndex,
	to types.TestOptions) *mvCons1TestItems {

	preHeader := make([]messages.MsgHeader, 1)
	preHeader[0] = messagetypes.NewConsMessage()
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs})
	gc := consinterface.CreateGeneralConfig(privKeyIdx, eis,
		stats.GetStatsObject(types.MvCons3Type, to.EncryptChannels), to, preHeader, privKeys[privKeyIdx])
	msgState := NewMvCons3MessageState(gc).New(idx).(*MessageState)
	memberChecker := &consinterface.MemCheckers{
		MC:  memberchecker.InitTrueMemberChecker(false, privKeys[privKeyIdx], gc).New(idx),
		SMC: memberchecker.NewNoSpecialMembers().New(idx)}
	forwardChecker := forwardchecker.NewAllToAllForwarder().New(idx, memberChecker.MC.GetParticipants(), memberChecker.MC.GetAllPubs())
	memberChecker.MC.(*memberchecker.TrueMemberChecker).AddPubKeys(nil, pubKeys, nil, [32]byte{})
	mmc := &testobjects.MockMainChannel{}
	consItems := &consinterface.ConsInterfaceItems{
		MC:         memberChecker,
		MsgState:   msgState,
		FwdChecker: forwardChecker,
	}
	sci := (&MvCons3{}).GenerateNewItem(idx, consItems, mmc, nil,
		MvCons3Config{}.GetBroadcastFunc(types.NonFaulty), gc).(*MvCons3)
	sci.SetInitialState(initialState)
	return &mvCons1TestItems{
		ConsTestItems: cons.ConsTestItems{
			PrivKeys: privKeys,
			PubKeys:  pubKeys},
		bcons:          sci,
		msgState:       msgState,
		memberChecker:  memberChecker,
		gc:             gc,
		forwardChecker: forwardChecker,
		mainChannel:    mmc}
}

var initialState = []byte("some init state")
var initialHash = types.GetHash(initialState)

func createMvSendItems(hdrType messages.HeaderID, idx types.ConsensusIndex, proposal []byte,
	supportHeaderSigMsg *sig.MultipleSignedMessage, bct cons.ConsTestItems, t *testing.T) []*channelinterface.DeserializedItem {

	var supportHash types.HashBytes
	var supportIndex types.ConsensusInt
	if supportHeaderSigMsg != nil {
		// sanity check
		_ = supportHeaderSigMsg.InternalSignedMsgHeader.(*messagetypes.MvInitSupportMessage)
		supportHash = supportHeaderSigMsg.Hash
		supportIndex = supportHeaderSigMsg.Index.Index.(types.ConsensusInt)
	} else {
		supportHash = initialHash
		supportIndex = 0
	}

	ret := make([]*channelinterface.DeserializedItem, len(bct.PrivKeys))

	for i := range bct.PrivKeys {
		priv := bct.PrivKeys[i]
		var hdr, dser messages.InternalSignedMsgHeader
		switch hdrType {
		case messages.HdrMvInitSupport:
			w := messagetypes.NewMvInitSupportMessage()
			w.Proposal = proposal
			w.SupportedIndex = supportIndex
			w.SupportedHash = supportHash
			hdr = w
			dser = messagetypes.NewMvInitSupportMessage()
		case messages.HdrMvEcho:
			w := messagetypes.NewMvEchoMessage()
			w.ProposalHash = supportHash
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

func createTestItems(privKeys []sig.Priv, pubKeys []sig.Pub, privKeyIdx int, to types.TestOptions) [10]*mvCons1TestItems {
	var bct [10]*mvCons1TestItems
	var prev *MvCons3
	for i := types.ConsensusInt(1); i <= 5; i++ {
		nxt := createMvConsTestItems(privKeys, pubKeys, privKeyIdx, types.SingleComputeConsensusIDShort(i), to)
		if prev != nil {
			prev.SetNextConsItem(nxt.bcons)
			nxt.bcons.prevItem = prev
		} else {
			nxt.bcons.PrevHasBeenReset()
		}
		prev = nxt.bcons
		bct[int(i)-1] = nxt
	}
	return bct
}

func TestMvCons3UnitGotMsg(t *testing.T) {
	to := mvCons3UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)
	bct := createTestItems(privKeys, pubKeys, 0, to)
	idx := types.SingleComputeConsensusIDShort(1)
	initItems := createMvSendItems(messages.HdrMvInitSupport, idx, mvConsTestProposal,
		nil, bct[0].ConsTestItems, t)

	_, err := bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, initItems[0], bct[0].gc, bct[0].memberChecker)
	if err != nil {
		t.Error(err)
	}

	_, err = bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, initItems[0], bct[0].gc, bct[0].memberChecker)
	if err != types.ErrNoNewSigs {
		t.Error(err)
	}

	// Invalid coord
	_, err = bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, initItems[1], bct[0].gc, bct[0].memberChecker)
	if err != types.ErrInvalidRoundCoord {
		t.Error(err)
	}

	// Non member
	privKeys2, pubKeys2 := cons.MakeKeys(to)
	bctInvalid := createMvConsTestItems(privKeys2, pubKeys2, 0, idx, to)
	echoItemsInvalid := createMvSendItems(messages.HdrMvEcho, idx, mvConsTestProposal,
		nil, bctInvalid.ConsTestItems, t)
	cons.MergeSigsForTest(echoItemsInvalid)
	_, err = bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, echoItemsInvalid[0], bct[0].gc, bct[0].memberChecker)
	if err != types.ErrNoValidSigs {
		t.Error(err)
	}

	// valid echo
	echoItems := createMvSendItems(messages.HdrMvEcho, idx, nil,
		initItems[0].Header.(*sig.MultipleSignedMessage), bct[0].ConsTestItems, t)
	cons.MergeSigsForTest(echoItems)
	_, err = bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, echoItems[0], bct[0].gc, bct[0].memberChecker)
	if err != nil {
		t.Error(err)
	}

	// repeat message
	_, err = bct[0].msgState.GotMsg(bct[0].bcons.GetHeader, echoItems[0], bct[0].gc, bct[0].memberChecker)
	if err != types.ErrNoNewSigs {
		t.Error(err)
	}
}

func TestMvCons3UnitDecide(t *testing.T) {
	to := mvCons3UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)

	// the first proposer
	bct := createTestItems(privKeys, pubKeys, 0, to)
	// the proposer for the second index
	bct2 := createTestItems(privKeys, pubKeys, 1, to)

	var prevSupport *sig.MultipleSignedMessage

	for i := types.ConsensusInt(1); i <= 4; i++ {

		// proposal
		idx := types.SingleComputeConsensusIDShort(i)
		p := messagetypes.NewMvProposeMessage(idx, mvConsTestProposal)
		bct[i-1].bcons.Start()
		if i == 1 {
			if err := bct[i-1].bcons.GotProposal(p, bct[i-1].mainChannel); err != nil {
				panic(err)
			}
			testobjects.CheckInitSupportMessage(bct[i-1].mainChannel, i, i-1, initialHash, mvConsTestProposal, t)
		} else {
			testobjects.CheckNoSend(bct[i-1].mainChannel, t)
		}
		bct2[i-1].bcons.Start()
		if i == 2 {
			if err := bct2[i-1].bcons.GotProposal(p, bct2[i-1].mainChannel); err != nil {
				panic(err)
			}
			// first we have the proof to support our init
			if prevSupport == nil {
				panic("should have been set")
			}
			testobjects.CheckEchoMessage(bct2[i-1].mainChannel, i-1, 0, prevSupport.Hash, t)
			// next the init message
			testobjects.CheckInitSupportMessage(bct2[i-1].mainChannel, i, i-1, prevSupport.Hash, mvConsTestProposal, t)
		} else {
			testobjects.CheckNoSend(bct2[i-1].mainChannel, t)
		}

		// be sure the timeouts are correct
		for idx := types.ConsensusInt(1); idx <= 4; idx++ {
			var ts cons.TimeoutState
			if idx <= i {
				ts = cons.TimeoutPassed
			} else {
				ts = cons.TimeoutNotSent
			}
			assert.Equal(t, ts, bct[idx-1].bcons.prevTimeoutState, bct2[idx-1].bcons.prevTimeoutState)
		}

		// test with an invalid hash
		invalidPrev := sig.NewMultipleSignedMsg(types.SingleComputeConsensusIDShort(i-1),
			pubKeys[0].New(), messagetypes.NewMvInitSupportMessage())
		invalidPrev.Hash = types.GetHash([]byte("bad hash"))
		initInvalidItems := createMvSendItems(messages.HdrMvInitSupport, types.SingleComputeConsensusIDShort(i),
			mvConsTestProposal, invalidPrev, bct[i-1].ConsTestItems, t)
		_, err := bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, initInvalidItems[i-1], bct[i-1].gc, bct[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward := bct[i-1].bcons.ProcessMessage(initInvalidItems[i-1], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckNoSend(bct[i-1].mainChannel, t)

		// test with a valid hash
		initItems := createMvSendItems(messages.HdrMvInitSupport, types.SingleComputeConsensusIDShort(i),
			mvConsTestProposal, prevSupport, bct[i-1].ConsTestItems, t)
		_, err = bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, initItems[i-1], bct[i-1].gc, bct[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		prevSupport = initItems[i-1].Header.(*sig.MultipleSignedMessage)
		progress, forward = bct[i-1].bcons.ProcessMessage(initItems[i-1], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckEchoMessage(bct[i-1].mainChannel, i, 0, prevSupport.Hash, t)

		_, err = bct2[i-1].msgState.GotMsg(bct2[i-1].bcons.GetHeader, initItems[i-1], bct2[i-1].gc, bct2[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward = bct2[i-1].bcons.ProcessMessage(initItems[i-1], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct2[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckEchoMessage(bct2[i-1].mainChannel, i, 0, prevSupport.Hash, t)

		// send the echo
		echoItems := createMvSendItems(messages.HdrMvEcho, types.SingleComputeConsensusIDShort(i),
			nil, prevSupport, bct[i-1].ConsTestItems, t)
		cons.MergeSigsForTest(echoItems)
		_, err = bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, echoItems[0], bct[i-1].gc, bct[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward = bct[i-1].bcons.ProcessMessage(echoItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckNoSend(bct[i-1].mainChannel, t)

		_, err = bct2[i-1].msgState.GotMsg(bct2[i-1].bcons.GetHeader, echoItems[0], bct2[i-1].gc, bct2[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward = bct2[i-1].bcons.ProcessMessage(echoItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct2[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		testobjects.CheckNoSend(bct2[i-1].mainChannel, t)

		for idx := types.ConsensusInt(1); idx <= i; idx++ {
			assert.True(t, bct[idx-1].bcons.CanStartNext(), bct2[idx-1].bcons.CanStartNext())
			if i == 4 && idx == 1 {
				assert.True(t, bct[idx-1].bcons.HasDecided(), bct2[idx-1].bcons.HasDecided())
				_, dec1, _ := bct[idx-1].bcons.GetDecision()
				_, dec2, _ := bct2[idx-1].bcons.GetDecision()
				assert.Equal(t, mvConsTestProposal, dec1, dec2)
			} else {
				assert.False(t, bct[idx-1].bcons.HasDecided(), bct2[idx-1].bcons.HasDecided())
			}
		}
	}
}

// TODO test invalid support init hash
// Decide but skip an instance
func TestMvCons3UnitDecideMissing(t *testing.T) {
	to := mvCons3UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)
	bct := createTestItems(privKeys, pubKeys, 0, to)

	// proposal
	p := messagetypes.NewMvProposeMessage(types.SingleComputeConsensusIDShort(1), mvConsTestProposal)
	bct[0].bcons.Start()
	err := bct[0].bcons.GotProposal(p, bct[0].mainChannel)
	assert.Nil(t, err)
	testobjects.CheckInitSupportMessage(bct[0].mainChannel, 1, 0, initialHash, mvConsTestProposal, t)

	var prevSupport *sig.MultipleSignedMessage

	for i := types.ConsensusInt(1); i <= 5; i++ {

		// be sure the timeouts are correct
		for idx := types.ConsensusInt(1); idx <= 4; idx++ {
			var ts cons.TimeoutState
			if idx <= i {
				ts = cons.TimeoutPassed
			} else {
				ts = cons.TimeoutNotSent
			}
			assert.Equal(t, ts, bct[idx-1].bcons.prevTimeoutState)
		}

		if i == 2 {
			// No init
			// Just the init timeout
			deser := &channelinterface.DeserializedItem{
				Index:          types.SingleComputeConsensusIDShort(i),
				HeaderType:     messages.HdrMvInitTimeout,
				Header:         nil,
				IsLocal:        types.LocalMessage,
				IsDeserialized: true}
			progress, forward := bct[i-1].bcons.ProcessMessage(deser, true, nil)
			if !progress || forward {
				t.Error("should have valid message")
			}
			testobjects.CheckNoSend(bct[i-1].mainChannel, t)
		} else {
			/*			if i > 1 {
							p := messagetypes.NewMvProposeMessage(i, mvConsTestProposal)
							bct[i-1].bcons.GotProposal(p, bct[i-1].mainChannel)
							testobjects.CheckInitSupportMessage(bct[i-1].mainChannel, i-1, prevSupport.Index, prevSupport.Hash, mvConsTestProposal, t)
						}
			*/
			initItems := createMvSendItems(messages.HdrMvInitSupport, types.SingleComputeConsensusIDShort(i),
				mvConsTestProposal, prevSupport, bct[i-1].ConsTestItems, t)

			_, err := bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, initItems[i-1], bct[i-1].gc, bct[i-1].memberChecker)
			if err != nil {
				t.Error(err)
			}
			prevSupport = initItems[i-1].Header.(*sig.MultipleSignedMessage)

			progress, forward := bct[i-1].bcons.ProcessMessage(initItems[i-1], false, nil)
			if !progress || !forward {
				t.Error("should have valid message")
			}
			if bct[i-1].bcons.HasDecided() {
				t.Error("should not have decided yet")
			}
			testobjects.CheckEchoMessage(bct[i-1].mainChannel, i, 0, prevSupport.Hash, t)

			echoItems := createMvSendItems(messages.HdrMvEcho, types.SingleComputeConsensusIDShort(i),
				nil, prevSupport, bct[i-1].ConsTestItems, t)
			cons.MergeSigsForTest(echoItems)
			_, err = bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, echoItems[0], bct[i-1].gc, bct[i-1].memberChecker)
			if err != nil {
				t.Error(err)
			}
			progress, forward = bct[i-1].bcons.ProcessMessage(echoItems[0], false, nil)
			if !progress || !forward {
				t.Error("should have valid message")
			}
			if bct[i-1].bcons.HasDecided() {
				t.Error("should not have decided yet")
			}
			testobjects.CheckNoSend(bct[i-1].mainChannel, t)
		}

		for idx := types.ConsensusInt(1); idx <= i; idx++ {
			if idx != 2 {
				assert.True(t, bct[idx-1].bcons.CanStartNext())
			}
			if i == 5 && idx == 1 {
				if !bct[idx-1].bcons.HasDecided() {
					t.Error("should have decided by now")
				}
				_, decision, _ := bct[idx-1].bcons.GetDecision()
				if !bytes.Equal(decision, mvConsTestProposal) {
					t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
				}
			} else if bct[idx-1].bcons.HasDecided() {
				t.Error("should not have decided yet")
			}
		}
		if i < 5 {
			bct[i].bcons.Start()
		}
	}
}

// Decide but send an init skipping one
func TestMvCons3UnitDecideSkip(t *testing.T) {
	to := mvCons3UnitTestOptions
	privKeys, pubKeys := cons.MakeKeys(to)
	bct := createTestItems(privKeys, pubKeys, 0, to)

	// proposal
	p := messagetypes.NewMvProposeMessage(types.SingleComputeConsensusIDShort(1), mvConsTestProposal)
	bct[0].bcons.Start()
	err := bct[0].bcons.GotProposal(p, bct[0].mainChannel)
	assert.Nil(t, err)
	testobjects.CheckInitSupportMessage(bct[0].mainChannel, 1, 0, initialHash, mvConsTestProposal, t)

	var prevSupport *sig.MultipleSignedMessage

	for i := types.ConsensusInt(1); i <= 5; i++ {

		// be sure the timeouts are correct
		for idx := types.ConsensusInt(1); idx <= 4; idx++ {
			var ts cons.TimeoutState
			if idx <= i {
				ts = cons.TimeoutPassed
			} else {
				ts = cons.TimeoutNotSent
			}
			assert.Equal(t, ts, bct[idx-1].bcons.prevTimeoutState)
		}

		var initItems []*channelinterface.DeserializedItem
		if i == 2 {
			// second init will point back to initial
			initItems = createMvSendItems(messages.HdrMvInitSupport, types.SingleComputeConsensusIDShort(i), mvConsTestProposal, nil, bct[i-1].ConsTestItems, t)
		} else {
			initItems = createMvSendItems(messages.HdrMvInitSupport, types.SingleComputeConsensusIDShort(i), mvConsTestProposal, prevSupport, bct[i-1].ConsTestItems, t)
		}

		_, err := bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, initItems[i-1], bct[i-1].gc, bct[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		prevSupport = initItems[i-1].Header.(*sig.MultipleSignedMessage)

		progress, forward := bct[i-1].bcons.ProcessMessage(initItems[i-1], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}
		if i == 2 { // we should not send an echo since the init supports and older index than we have already supported
			testobjects.CheckNoSend(bct[i-1].mainChannel, t)
		} else {
			testobjects.CheckEchoMessage(bct[i-1].mainChannel, i, 0, prevSupport.Hash, t)
		}

		echoItems := createMvSendItems(messages.HdrMvEcho, types.SingleComputeConsensusIDShort(i), nil, prevSupport, bct[i-1].ConsTestItems, t)
		cons.MergeSigsForTest(echoItems)
		_, err = bct[i-1].msgState.GotMsg(bct[i-1].bcons.GetHeader, echoItems[0], bct[i-1].gc, bct[i-1].memberChecker)
		if err != nil {
			t.Error(err)
		}
		progress, forward = bct[i-1].bcons.ProcessMessage(echoItems[0], false, nil)
		if !progress || !forward {
			t.Error("should have valid message")
		}
		if bct[i-1].bcons.HasDecided() {
			t.Error("should not have decided yet")
		}

		for idx := types.ConsensusInt(1); idx <= i; idx++ {
			assert.True(t, bct[idx-1].bcons.CanStartNext())
			if i == 5 && (idx == 2 || idx == 1) {
				if !bct[idx-1].bcons.HasDecided() {
					t.Error("should have decided by now")
				}
				_, decision, _ := bct[idx-1].bcons.GetDecision()
				if idx == 1 {
					if len(decision) != 0 {
						t.Errorf("should have decided nil, but decided %v", decision)
					}
				} else {
					if !bytes.Equal(decision, mvConsTestProposal) {
						t.Errorf("should have decided %v, but decided %v", mvConsTestProposal, decision)
					}
				}
			} else if bct[idx-1].bcons.HasDecided() {
				t.Error("should not have decided yet", idx, i)
			}
		}
	}
}
