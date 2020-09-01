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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"sync/atomic"

	"github.com/tcrain/cons/consensus/auth/sig"
)

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// TrueMemberChecker has a fixed list of member from the beginning.
type TrueMemberChecker struct {
	absMemberChecker
}

func InitTrueMemberChecker(rotateCord bool, myPriv sig.Priv, gc *generalconfig.GeneralConfig) consinterface.MemberChecker {
	return &TrueMemberChecker{absMemberChecker: *initAbsMemberChecker(nil, rotateCord,
		0, types.NonRandom, 0, myPriv, gc)}
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *TrueMemberChecker) New(idx types.ConsensusIndex) consinterface.MemberChecker {
	newMc := &TrueMemberChecker{}
	newMc.absMemberChecker = *mc.absMemberChecker.newAbsMc(idx)
	return newMc
}

// AllowsChange returns true if the member checker allows changing members, used as a sanity check.
func (mc *TrueMemberChecker) AllowsChange() bool {
	return false
}

// CheckRandRoundCoord is unsupported.
func (mc *TrueMemberChecker) CheckRandRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (randValue uint64, coordPub sig.Pub, err error) {

	_, _, _ = msgID, checkPub, round
	panic("unsupported")
}

// IsReady always returns true.
func (mc *TrueMemberChecker) IsReady() bool {
	return true
}

// UpdateState does nothing since the members do not change.
// func (mc *TrueMemberChecker) UpdateState(prevDec []byte, prevSM consinterface.StateMachineInterface, prevMember MemberChecker) []sig.Pub {
func (mc *TrueMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface) (newMemberPubs,
	newAllPubs []sig.Pub) {

	if len(prevDec) > 0 && !prevSM.GetDecided() {
		panic("should have updated the SM first")
	}
	// nothing
	// if prevMember.(*TrueMemberChecker).idx+1 != mc.idx {
	//	panic("out of oder member state update")
	// }

	return mc.AbsGotDecision(fixedCoord, newMemberPubs, newAllPubs,
		prevDec, randBytes, &prevMember.(*TrueMemberChecker).absMemberChecker)
}

// FinishUpdateState does nothing since the members do not change.
func (mc *TrueMemberChecker) FinishUpdateState() {
}

////////////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////////////

func InitCurrentTrueMemberChecker(localRand *rand.Rand, rotateCord bool, rndMemberCount int,
	rndMemberType types.RndMemberType, localRandChangeFrequency types.ConsensusInt, priv sig.Priv,
	gc *generalconfig.GeneralConfig) consinterface.MemberChecker {

	return &CurrentTrueMemberChecker{absMemberChecker: *initAbsMemberChecker(localRand, rotateCord, rndMemberCount,
		rndMemberType, localRandChangeFrequency, priv, gc)}
}

// AllowsChange returns true if the member checker allows changing members, used as a sanity check.
func (mc *CurrentTrueMemberChecker) AllowsChange() bool {
	return mc.RandMemberType() != types.NonRandom
}

// CurrentTrueMemberChecker is the same as TrueMemberChecker, except IsReady returns true
// only once FinishUpdateState has been called.
// It is just for testing
type CurrentTrueMemberChecker struct {
	absMemberChecker
	isReady uint32 // this is read and updated atomically as it is accessed from different threads
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *CurrentTrueMemberChecker) New(newIndex types.ConsensusIndex) consinterface.MemberChecker {
	newMc := &CurrentTrueMemberChecker{}
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
func (mc *CurrentTrueMemberChecker) IsReady() bool {
	return atomic.LoadUint32(&mc.isReady) > 0
}

// UpdateState does nothing since the members do not change.
// func (mc *CurrentTrueMemberChecker) UpdateState(prevDec []byte, prevSM consinterface.StateMachineInterface, prevMember MemberChecker) []sig.Pub {
func (mc *CurrentTrueMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface) (newMemberPubs,
	newAllPubs []sig.Pub) {

	if !prevSM.GetDecided() {
		panic("should have updated the SM first")
	}

	// if prevMember.(*CurrentTrueMemberChecker).idx+1 != mc.idx {
	// 	panic("out of oder member state update")
	// }
	// First copy the previous state
	mc.copyPrevMemberState(&prevMember.(*CurrentTrueMemberChecker).absMemberChecker)

	return mc.AbsGotDecision(fixedCoord, newMemberPubs, newAllPubs, prevDec,
		randBytes, &prevMember.(*CurrentTrueMemberChecker).absMemberChecker)
}

// FinishUpdateState sets the member checker to ready.
func (mc *CurrentTrueMemberChecker) FinishUpdateState() {
	atomic.StoreUint32(&mc.isReady, 1)
}
