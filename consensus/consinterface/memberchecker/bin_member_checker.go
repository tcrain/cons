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
package memberchecker

import (
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"sync/atomic"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/utils"
)

////////////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////////////

// BinRotateMemberChecker rotates consensus members based on the decided value.
// If absMemberChecker.AddPubKeys was called with len(keys) > membercount, then this member checker
// has the list of cons members for the next consensus instance
// rotate by one position (of the list of all participants) each time 0 is decided by consensus.
// Any other decided value keeps the members the same.
type BinRotateMemberChecker struct {
	absMemberChecker
	isReady uint32 // accessed atomically
}

func InitBinRotateMemberChecker(myPriv sig.Priv, gc *generalconfig.GeneralConfig) consinterface.MemberChecker {
	return &BinRotateMemberChecker{absMemberChecker: *initAbsMemberChecker(nil, false, 0,
		types.NonRandom, 0, myPriv, gc)}
}

// CheckRandRoundCoord is unsupported.
func (mc *BinRotateMemberChecker) CheckRandRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (randValue uint64, coordPub sig.Pub, err error) {

	panic("unsupported")
}

func (mc *BinRotateMemberChecker) AllowsChange() bool {
	return true
}

// GetRoundCoord returns the public key of the coordinator for round round.
// It is just the mod of the round in the sorted list of public keys.
func (mc *BinRotateMemberChecker) CheckRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {

	pos := utils.Abs((int(round)) % len(mc.sortedMemberPubs))

	coord := mc.sortedMemberPubs[pos]
	if coord == nil {
		panic("should not be nil")
	}
	if checkPub == nil {
		return coord, nil
	}
	return mc.checkCoordInternal(round, coord, checkPub)
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *BinRotateMemberChecker) New(newIndex types.ConsensusIndex) consinterface.MemberChecker {
	newMc := &BinRotateMemberChecker{}
	newMc.absMemberChecker = *mc.absMemberChecker.newAbsMc(newIndex)

	if newIndex.Index.IsInitIndex() {
		newMc.isReady = 1
	} else {
		newMc.isReady = 0
	}
	// if newIndex <= 1 {
	// 	store = 1
	// } else {
	// 	store = 0
	// }
	// atomic.StoreUint32(&mc.isReady, store)
	return newMc
}

// IsReady returns false until FinishUpdateState is called.
func (mc *BinRotateMemberChecker) IsReady() bool {
	return atomic.LoadUint32(&mc.isReady) > 0
}

// FinishUpdateState sets the member checker to ready.
func (mc *BinRotateMemberChecker) FinishUpdateState() {
	atomic.StoreUint32(&mc.isReady, 1)
}

// Update state computes the new members, in case 0 was decided, then they rotate by one from the list
// of all public keys (see BinRotateMemberChecker struct description).
// func (mc *BinRotateMemberChecker) UpdateState(prevDec []byte, prevSM consinterface.StateMachineInterface, prevMember MemberChecker) []sig.Pub {
func (mc *BinRotateMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface) (newMemberPubs,
	newAllPubs []sig.Pub) {

	if len(prevDec) == 0 {
		prevDec = []byte{0} // a nil decision is assumed as a 0
	}
	prev := prevMember.(*BinRotateMemberChecker)

	// First copy the previous state
	mc.copyPrevMemberState(&prev.absMemberChecker)

	var dec byte
	if len(prevDec) != 1 {
		// panic("must be bin decision")
		dec = 1
	} else {
		dec = prevDec[0]
	}
	// var ret []sig.Pub\
	var nxtMemberPubs, nxtOtherPubs sig.PubList
	switch dec {
	case 0:
		logging.Info("rotating membership because first byte of decided value was 0")
		// we change the members, take the one at the end of the list and put it at the head
		nxtAllPubs := append(prev.allPubs[len(prev.allPubs)-1:], prev.allPubs[:len(prev.allPubs)-1]...)
		nxtMemberPubs = nxtAllPubs[:len(prev.sortedMemberPubs)]
		nxtOtherPubs = nxtAllPubs[len(prev.sortedMemberPubs):]

	default:
		logging.Info("keeping same membership")
		// keep the same state as the prev
	}

	return mc.AbsGotDecision(fixedCoord, nxtMemberPubs, nxtOtherPubs, prevDec,
		randBytes, &prevMember.(*BinRotateMemberChecker).absMemberChecker)
}
