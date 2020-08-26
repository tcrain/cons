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

package csnet

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"time"
)

var DeserFunc func(tci *TestConsItem, idx types.ConsensusIndex, msg *messages.Message, mc *consinterface.MemCheckers, ms consinterface.MessageState) ([]*channelinterface.DeserializedItem, error)

type TestConsItem struct {
	DeserFunc func(tci *TestConsItem, idx types.ConsensusIndex, msg *messages.Message, mc *consinterface.MemCheckers, ms consinterface.MessageState) ([]*channelinterface.DeserializedItem, error)
	Index     types.ConsensusIndex
}

func (sc *TestConsItem) ShouldCreatePartial(headerType messages.HeaderID) bool {
	return false
}
func (sc *TestConsItem) Broadcast(nextCoordPub sig.Pub, msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	additionalMsgs ...messages.MsgHeader) {
}
func (sc *TestConsItem) GetCommitProof() []messages.MsgHeader          { return nil }
func (sc *TestConsItem) SetCommitProof(prf []messages.MsgHeader)       {}
func (sc *TestConsItem) GetPrevCommitProof() []messages.MsgHeader      { return nil }
func (sc *TestConsItem) CheckMemberLocalMsg(msgID messages.MsgID) bool { return true }
func (sc *TestConsItem) AddPreHeader(header messages.MsgHeader)        {}
func (*TestConsItem) GenerateNewItem(index types.ConsensusIndex, consItems *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, bcastFunc consinterface.ByzBroadcastFunc, gc *generalconfig.GeneralConfig) consinterface.ConsItem {
	return &TestConsItem{DeserFunc: DeserFunc, Index: index}
}
func (sc *TestConsItem) CheckMemberLocal() bool                                   { return true }
func (sc *TestConsItem) GetGeneralConfig() *generalconfig.GeneralConfig           { return nil }
func (*TestConsItem) Collect()                                                    {}
func (sc *TestConsItem) GetConsInterfaceItems() *consinterface.ConsInterfaceItems { return nil }
func (sc *TestConsItem) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool) {
	return
}
func (sc *TestConsItem) Start()                {}
func (sc *TestConsItem) HasStarted() bool      { return true }
func (sc *TestConsItem) HasValidStarted() bool { return false }
func (sc *TestConsItem) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	return
}
func (sc *TestConsItem) SetInitialState([]byte) {}
func (sc *TestConsItem) NeedsConcurrent() types.ConsensusInt {
	return 1
}
func (*TestConsItem) GenerateMessageState(config *generalconfig.GeneralConfig) consinterface.MessageState {
	return nil
}
func (sc *TestConsItem) GetIndex() types.ConsensusIndex {
	return sc.Index
}
func (sc *TestConsItem) CanStartNext() bool {
	return false
}
func (sc *TestConsItem) SetTestConfig(testId int, options types.TestOptions) {
}
func (sc *TestConsItem) GetPreHeader() []messages.MsgHeader {
	return nil
}
func (sc *TestConsItem) ProcessMessage(item *channelinterface.DeserializedItem, isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (progress, shouldForward bool) {
	return false, false
}
func (sc *TestConsItem) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	return nil, nil, types.ConsensusIndex{}
}
func (sc *TestConsItem) ComputeDecidedValue(state []byte, decision []byte) []byte {
	return nil
}
func (sc *TestConsItem) HasDecided() bool {
	return false
}
func (sc *TestConsItem) GetBinState(localOnly bool) ([]byte, error) {
	return nil, nil
}
func (sc *TestConsItem) GetConsType() types.ConsType {
	panic("unused")
}
func (sc *TestConsItem) PrevHasBeenReset() {
	panic("unused")
}
func (sc *TestConsItem) GotProposal(messages.MsgHeader, channelinterface.MainChannel) error {
	return nil
}
func (sc *TestConsItem) GetProposeHeaderID() messages.HeaderID {
	return 0
}
func (sc *TestConsItem) InitState(preHeaders []messages.MsgHeader, priv sig.Priv, eis generalconfig.ExtraInitState,
	stats stats.StatsInterface) {
}
func (sc *TestConsItem) ResetState(index types.ConsensusIndex, memberChecker *consinterface.MemCheckers,
	messageState consinterface.MessageState, forwardChecker consinterface.ForwardChecker, prev consinterface.ConsItem) {
}
func (sc *TestConsItem) SetNextConsItem(next consinterface.ConsItem) {
	// panic("unused")
}
func (sc *TestConsItem) GetBufferCount(messages.MsgIDHeader, *generalconfig.GeneralConfig, *consinterface.MemCheckers) (
	endThreshold int, maxPossible int, msgid messages.MsgID, err error) {
	panic("not used")
}
func (*TestConsItem) GetHeader(emptyPub sig.Pub, generalConfig *generalconfig.GeneralConfig,
	headerID messages.HeaderID) (messages.MsgHeader, error) {
	switch headerID {
	case messages.HdrNetworkTest:
		return &messagetypes.NetworkTestMessage{}, nil
	default:
		return nil, types.ErrInvalidHeader
	}
}
func (sc *TestConsItem) DeserializeMessage(idx types.ConsensusIndex, msg *messages.Message,
	mc *consinterface.MemCheckers, ms consinterface.MessageState) ([]*channelinterface.DeserializedItem, error) {
	return sc.DeserFunc(sc, idx, msg, mc, ms)
}

type TestMessageState struct {
	index types.ConsensusIndex
}

func (tms *TestMessageState) GotMsg(_ consinterface.HeaderFunc,
	dsi *channelinterface.DeserializedItem, _ *generalconfig.GeneralConfig, _ *consinterface.MemCheckers) (
	[]*channelinterface.DeserializedItem, error) {
	return []*channelinterface.DeserializedItem{dsi}, nil
}

func (tms *TestMessageState) GetMsgState(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc,
	mc *consinterface.MemCheckers) ([]byte, error) {
	return nil, nil
}

func (tms *TestMessageState) AddMyVRFProof(proof sig.VRFProof) {}

func (tms *TestMessageState) New(idx types.ConsensusIndex) consinterface.MessageState {
	return &TestMessageState{idx}
}
func (tms *TestMessageState) SetupUnsignedMessage(hdr messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (*sig.UnsignedMessage, error) {
	return nil, nil
}
func (tms *TestMessageState) GetCoinVal(hdr messages.InternalSignedMsgHeader, threshold int, mc *consinterface.MemCheckers) (
	coinVal types.BinVal, ready bool, err error) {
	return
}
func (tms *TestMessageState) GetThreshSig(hdr messages.InternalSignedMsgHeader, threshold int, mc *consinterface.MemCheckers) (*sig.SigItem, error) {
	return nil, nil
}
func (tms *TestMessageState) GetIndex() types.ConsensusIndex {
	return tms.index
}
func (tms *TestMessageState) SetupSignedMessagesDuplicates(combined *messagetypes.CombinedMessage, hdrs []messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (combinedSigned *sig.MultipleSignedMessage, partialsSigned []*sig.MultipleSignedMessage, err error) {

	return
}
func (tms *TestMessageState) SetupSignedMessage(sm messages.InternalSignedMsgHeader,
	generateMySig bool, addOthersSigsCount int, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error) {

	return nil, nil
}
func (tms *TestMessageState) GetSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {
	panic("unused")
}
func (tms *TestMessageState) GetSigCountMsg(types.HashStr) int {
	panic("unused")
}
func (tms *TestMessageState) GetSigCountMsgID(messages.MsgID) int {
	panic("unused")
}
func (tms *TestMessageState) GetSigCountMsgHeader(header messages.InternalSignedMsgHeader, mc *consinterface.MemCheckers) (int, error) {
	panic("unused")
}

type TestMemberChecker struct {
	index types.ConsensusIndex
}

func (mc *TestMemberChecker) DoneNextUpdateState() error { return nil }

func (mc *TestMemberChecker) GetMyPriv() sig.Priv {
	return nil
}

func (mc *TestMemberChecker) SetMainChannel(channel channelinterface.MainChannel) {}
func (mc *TestMemberChecker) AllowsChange() bool {
	return false
}
func (mc *TestMemberChecker) RandMemberType() types.RndMemberType {
	return types.NonRandom
}
func (mc *TestMemberChecker) GetParticipants() sig.PubList {
	return nil
}
func (mc *TestMemberChecker) GetAllPubs() sig.PubList {
	return nil
}
func (mc *TestMemberChecker) GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error {
	panic("unused")
}
func (mc *TestMemberChecker) GetMyVRF(id messages.MsgID) sig.VRFProof { return nil }

func (mc *TestMemberChecker) CheckRandMember(pub sig.Pub, msgID messages.MsgID, isProposalMsg bool) error {
	return nil
}
func (mc *TestMemberChecker) SelectRandMembers() bool { return false }

func (mc *TestMemberChecker) GetNewPub() sig.Pub {
	return nil
}
func (mc *TestMemberChecker) SetStats(stats stats.StatsInterface) {
}
func (mc *TestMemberChecker) CheckRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {
	panic("unused")
}
func (mc *TestMemberChecker) GetIndex() types.ConsensusIndex {
	return mc.index
}
func (mc *TestMemberChecker) AddPubKeys(fixedCoord sig.Pub, memberPubKeys, allPubKeys sig.PubList, initRandBytes [32]byte, sh *consinterface.Shared) {
}
func (mc *TestMemberChecker) GetParticipantCount() int {
	return 0
}
func (mc *TestMemberChecker) CheckRandRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (randValue uint64, coordPub sig.Pub, err error) {
	return
}
func (mc *TestMemberChecker) New(idx types.ConsensusIndex) consinterface.MemberChecker {
	return &TestMemberChecker{idx}
}
func (mc *TestMemberChecker) CheckEstimatedRoundCoordNextIndex(checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {
	return
}
func (mc *TestMemberChecker) GetStats() stats.StatsInterface { return nil }
func (mc *TestMemberChecker) CheckFixedCoord(sig.Pub) (sig.Pub, error) {
	panic("unused")
}
func (mc *TestMemberChecker) Validated(types.SignType) {
}
func (mc *TestMemberChecker) IsReady() bool {
	return true
}
func (mc *TestMemberChecker) GetMemberCount() int {
	panic("shouldnt be called")
}
func (mc *TestMemberChecker) GetFaultCount() int {
	panic("shouldnt be called")
}
func (mc *TestMemberChecker) CheckMemberBytes(types.ConsensusIndex, sig.PubKeyID) sig.Pub {
	panic("shouldnt be called")
}
func (mc *TestMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface) (newMemberPubs, newAllPubs []sig.Pub) {

	panic("unused")
}
func (mc *TestMemberChecker) FinishUpdateState() {
}
func (mc *TestMemberChecker) CheckIndex(index types.ConsensusIndex) bool {
	return true
}

func DeserializeMessage(tci *TestConsItem, idx types.ConsensusIndex, msg *messages.Message,
	mc *consinterface.MemCheckers, ms consinterface.MessageState) ([]*channelinterface.DeserializedItem, error) {

	tstMsg := &messagetypes.NetworkTestMessage{}
	var tstMsgTimeout messagetypes.TestMessageTimeout
	ht, err := msg.PeekHeaderType()
	if err != nil {
		panic(err)
		return nil, err
	}
	if ht == tstMsgTimeout.GetID() {
		_, err = tstMsgTimeout.Deserialize(msg, types.IntIndexFuns)
		if err != nil {
			panic(err)
			return nil, err
		}
		return []*channelinterface.DeserializedItem{
			{
				Index:          idx,
				HeaderType:     ht,
				Header:         tstMsgTimeout,
				IsDeserialized: true},
		}, nil
	} else if ht == tstMsg.GetID() {
		_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
		if err != nil {
			panic(err)
			return nil, err
		}
		return []*channelinterface.DeserializedItem{
			{
				Index:          idx,
				HeaderType:     ht,
				Header:         tstMsg,
				IsDeserialized: true},
		}, nil
	}
	panic(ht)
	return nil, types.ErrInvalidHeader
}

type TestSM struct{}

func (TestSM) GetProposal()                                                            {}
func (TestSM) FinishedLastRound() bool                                                 { return false }
func (TestSM) HasDecided(proposer sig.Pub, index types.ConsensusInt, decision []byte)  {}
func (TestSM) StartIndex(index types.ConsensusInt) consinterface.StateMachineInterface { return nil }
func (TestSM) FailAfter(index types.ConsensusInt)                                      {}
func (TestSM) Init(generalConfig *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {
}
func (TestSM) GetByzProposal(originProposal []byte, gc *generalconfig.GeneralConfig) (byzProposal []byte) {
	return
}
func (TestSM) GetDone() types.DoneType                                    { return types.NotDone }
func (TestSM) CheckDecisions([][]byte) (outOfOrderErrors, errors []error) { return }
func (TestSM) CheckStartStatsRecording(index types.ConsensusInt)          {}
func (TestSM) GetIndex() (ret types.ConsensusIndex)                       { return }
func (TestSM) ValidateProposal(proposer sig.Pub, proposal []byte) error   { return nil }
func (TestSM) GetInitialState() []byte                                    { return nil }
func (TestSM) StatsString(testDuration time.Duration) string              { return "" }
func (TestSM) GetDecided() bool                                           { return false }
func (TestSM) GetRand() (ret [32]byte)                                    { return }
func (TestSM) DoneClear()                                                 {}
func (TestSM) DoneKeep()                                                  {}
func (TestSM) Collect()                                                   {}
func (TestSM) EndTest()                                                   {}
