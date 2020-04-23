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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"sync/atomic"
)

// InitCustomMemberChecker initiates a CustomMemberChecker object.
func InitCustomMemberChecker(localRand *rand.Rand, rotateCord bool, rndMemberCount int,
	rndMemberType types.RndMemberType, localRandChangeFrequency types.ConsensusInt, priv sig.Priv,
	gc *generalconfig.GeneralConfig) *CustomMemberChecker {

	return &CustomMemberChecker{absMemberChecker: *initAbsMemberChecker(localRand, rotateCord, rndMemberCount,
		rndMemberType, localRandChangeFrequency, priv, gc)}
}

// CustomMemberChecker is intended to extended to be used as a custom member checker associated
// with a state machine.
type CustomMemberChecker struct {
	absMemberChecker
	isReady uint32 // this is read and updated atomically as it is accessed from different threads
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *CustomMemberChecker) New(newIndex types.ConsensusIndex) *CustomMemberChecker {
	newMc := &CustomMemberChecker{}
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

// CheckRandRoundCoord should be called instead of CheckRoundCoord if random membership selection is enabled.
// If using VRFs then checkPub must not be nil.
// If checkPub is nil, then it will return the known coordinator in coordPub.
// If VRF is enabled randValue is the VRF random value for the inputs.
// Note this should be called after CheckRandMember for the same pub.
func (mc *CustomMemberChecker) CheckRandRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (randValue uint64, coordPub sig.Pub, err error) {

	if mc.RandMemberType() == types.LocalRandMember {
		coordPub, err = mc.CheckRoundCoord(msgID, checkPub, round)
		return
	}

	if checkPub == nil {
		return 0, nil, types.ErrNotMember
	}
	rndMemberCount := utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs))
	rndValue, checkPub, err := mc.checkRandCoord(rndMemberCount, len(mc.sortedMemberPubs), msgID, round, checkPub)
	if err != nil {
		return 0, nil, err
	}
	return rndValue, checkPub, nil
}

// IsReady returns false until FinishUpdateState is called.
func (mc *CustomMemberChecker) IsReady() bool {
	return atomic.LoadUint32(&mc.isReady) > 0
}

// UpdateState does nothing since the members do not change.
// func (mc *CustomMemberChecker) UpdateState(prevDec []byte, prevSM consinterface.StateMachineInterface, prevMember MemberChecker) []sig.Pub {
func (mc *CustomMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember *CustomMemberChecker, newMemberPubs, newAllPubs []sig.Pub) (sig.PubList, sig.PubList) {
	// if prevMember.(*CustomMemberChecker).idx+1 != mc.idx {
	// 	panic("out of oder member state update")
	// }
	// First copy the previous state
	mc.copyPrevMemberState(&prevMember.absMemberChecker)

	return mc.AbsGotDecision(fixedCoord, newMemberPubs, newAllPubs, prevDec,
		randBytes, &prevMember.absMemberChecker)
}

// FinishUpdateState sets the member checker to ready.
func (mc *CustomMemberChecker) FinishUpdateState() {
	atomic.StoreUint32(&mc.isReady, 1)
}
