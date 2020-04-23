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

package cons

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// AbsConsItem implements some of the methods from the consinterface.ConsItem interface
// and can be used as an abstract class for a ConsItem implementation.
type AbsConsItem struct {
	*generalconfig.GeneralConfig
	Index         types.ConsensusIndex // The index of this consensus
	PreHeaders    []messages.MsgHeader // Headers to be appended to the beginning of all consensus messages for this specific consensus instance.
	isMember      int                  // 0 if it is not yet known if this node is a member of this consensus, 1 if this node is not a member, 0 if this node is a member
	CommitProof   []messages.MsgHeader // Proof of committal from last consensus round
	ConsItems     *consinterface.ConsInterfaceItems
	PrevItem      consinterface.ConsItem
	NextItem      consinterface.ConsItem
	Started       bool
	NeedsProposal bool
	BroadcastFunc consinterface.ByzBroadcastFunc
	GotProposal   bool
	MainChannel   channelinterface.MainChannel
}

// HasStarted returns true if Start has ben called
func (sc *AbsConsItem) HasStarted() bool {
	return sc.Started
}

// Broadcast a message.
// If nextCoordPub is nil the message will only be sent to that node, otherwise it will be sent
// as normal (nextCoordPub is used when CollectBroadcast is true in test options).
func (sc *AbsConsItem) Broadcast(nxtCoordPub sig.Pub, auxMsg messages.InternalSignedMsgHeader,
	signMessage bool, forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel, additionalMsgs ...messages.MsgHeader) {

	DoConsBroadcast(nxtCoordPub, auxMsg, signMessage, additionalMsgs, forwardFunc,
		sc.ConsItems, mainChannel, sc.GeneralConfig)
}

func (sc *AbsConsItem) GetGeneralConfig() *generalconfig.GeneralConfig {
	return sc.GeneralConfig
}

// SetCommitProof takes the value returned from GetCommitProof of the previous consensus instance once it has decided.
// The consensus can then use this as needed.
func (sc *AbsConsItem) SetCommitProof(prf []messages.MsgHeader) {
	if len(prf) > 0 {
		sc.CommitProof = prf
	}
}

// GetPrevCommitProof returns a signed message header that counts at the commit message for the previous consensus.
// This should only be called after DoneKeep has been called on this instance.
func (sc *AbsConsItem) GetPrevCommitProof() []messages.MsgHeader {
	return sc.CommitProof
}

// Start should be called once the consensus instance has started.
func (sc *AbsConsItem) AbsStart() {
	sc.Started = true
	sc.ConsItems.MC.MC.GetStats().AddStartTime()
	logging.Infof("Starting consensus index %v", sc.Index)
}

func (sc *AbsConsItem) AbsGotProposal() {
	if !sc.NeedsProposal {
		panic("should only get proposal after it was needed")
	}
	if sc.GotProposal {
		panic("got multiple proposals for same cons")
	}
	if !sc.CheckMemberLocal() {
		panic("non member should not get proposal")
	}
	sc.GotProposal = true
}

// GetIndex returns the consensus index of the item.
func (sc *AbsConsItem) GetIndex() types.ConsensusIndex {
	return sc.Index
}

// GetPreHeader returns the header that is attached to all messages sent by this consensus item.
func (sc *AbsConsItem) GetPreHeader() []messages.MsgHeader {
	return sc.PreHeaders
}

// AddPreHeader appends a header that will be attached to all messages sent by this consensus item.
func (sc *AbsConsItem) AddPreHeader(header messages.MsgHeader) {
	sc.PreHeaders = append(sc.PreHeaders, header)
}

// GetConsInterfaceItems returns the ConsInterfaceItems for this consesnsus instance.
func (sc *AbsConsItem) GetConsInterfaceItems() *consinterface.ConsInterfaceItems {
	return sc.ConsItems
}

// GenerateAbsState should be called within consinterface.GenerateNewIem, it sets up the inital headers.
func GenerateAbsState(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) AbsConsItem {

	aci := AbsConsItem{}
	aci.MainChannel = mainChannel
	aci.GeneralConfig = gc
	if !aci.SetTestConfig {
		panic("should set test generalconfig before calling init")
	}
	aci.Index = index
	aci.BroadcastFunc = broadcastFunc
	aci.PreHeaders = make([]messages.MsgHeader, len(aci.InitHeaders))
	copy(aci.PreHeaders, aci.InitHeaders)
	aci.isMember = 0
	aci.ConsItems = items
	aci.PrevItem = prevItem
	// _, ok := aci.Index.Index.(types.ConsensusHash)
	// if ok || aci.Index.Index.(types.ConsensusInt) > config.WarmUpInstances {
	// 	aci.ConsItems.MC.MC.GetStats().StartRecording(aci.GeneralConfig.CPUProfile,
	//		aci.GeneralConfig.MemProfile, aci.GeneralConfig.TestIndex, aci.GeneralConfig.TestID)
	//}

	return aci
}

// ComputeDecidedValue returns decision.
func (sc *AbsConsItem) ComputeDecidedValue(state []byte, decision []byte) []byte {
	return decision
}

// CheckMemberLocal checks if the node is a member of the consensus.
func (sc *AbsConsItem) CheckMemberLocal() bool {
	switch sc.isMember {
	case 0: // 0 means we need to check the member checker for membership
		if consinterface.CheckMemberLocal(sc.ConsItems.MC) {
			logging.Info("I AM a member", sc.GeneralConfig.TestIndex, sc.Index)
			sc.isMember = 2
		} else {
			logging.Info("I am NOT a member", sc.GeneralConfig.TestIndex, sc.Index)
			sc.isMember = 1
		}
		return sc.CheckMemberLocal()
	case 1: // is not a member
		return false
	case 2: // is a member
		return true
	default:
		panic("invalid member check")
	}
}

// CheckMemberLocalMsg checks if the local node is a member of the consensus for this message type
func (sc *AbsConsItem) CheckMemberLocalMsg(msgID messages.MsgID) bool {
	if sc.CheckMemberLocal() {
		// is proposal message is true since we always send the message when creating it locally
		return consinterface.CheckRandMember(sc.ConsItems.MC, sc.ConsItems.MC.MC.GetMyPriv().GetPub(),
			true, msgID) == nil
		// return aci.ConsItems.MC.MC.CheckRandMember(aci.Priv.GetPub(), msgID, true) == nil
	}
	return false
}
