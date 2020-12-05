package memberchecker

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
)

func InitLaterMemberChecker(localRand *rand.Rand, rotateCord bool, rndMemberCount int,
	rndMemberType types.RndMemberType, localRandChangeFrequency types.ConsensusInt, priv sig.Priv,
	gc *generalconfig.GeneralConfig) consinterface.MemberChecker {

	ret := &LaterMemberChecker{
		absMemberChecker: *initAbsMemberChecker(localRand, rotateCord, rndMemberCount,
			rndMemberType, localRandChangeFrequency, priv, gc)}
	ret.gs = &globalLaterState{}
	return ret
}

// AllowsChange returns true if the member checker allows changing members, used as a sanity check.
func (mc *LaterMemberChecker) AllowsChange() bool {
	return mc.RandMemberType() != types.NonRandom
}

// LaterMemberChecker is used for consensus algorithms that piggyback consensus messages
// on later consensus instances, so that when a decision happens the members are changed
// not on the index that was decided, but on a future one. As returned by GetDecision()
// from the consinterface.ConsItem interface.
// The consensus instance starts with IsReady=true.
// When UpdateState is called, the previous consensus index member checker that received
// a non nil decision is input.
type LaterMemberChecker struct {
	gs *globalLaterState
	absMemberChecker
	myIndex uint64 // consensus index as uint64
}

type globalLaterState struct {
	decItems []decStats
	// minIdx uint64 // minimum index int items
	// maxFixedFuture uint64 // max consensus index where members will not change
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *LaterMemberChecker) New(newIndex types.ConsensusIndex) consinterface.MemberChecker {
	newMc := &LaterMemberChecker{gs: mc.gs}
	newMc.absMemberChecker = *mc.absMemberChecker.newAbsMc(newIndex)
	newMc.myIndex = uint64(newIndex.Index.(types.ConsensusInt))

	if newIndex.Index.IsInitIndex() {
		if len(newMc.gs.decItems) > 0 {
			panic("shoud create the initial index first")
		}
		newMc.gs.decItems = []decStats{{
			fixedCoord: nil,
			prevDec:    nil,
			mc:         newMc,
		}}
	} else {
		// find the item that we use to generate the member checker
		var decItem decStats
		for _, nxt := range mc.gs.decItems {
			if nxt.genUntil > newMc.myIndex {
				break
			}
			decItem = nxt
		}
		var newMemberPubs, newAllPubs []sig.Pub
		newMc.copyPrevMemberState(&decItem.mc.absMemberChecker)
		newMemberPubs, newAllPubs, _ = newMc.AbsGotDecision(decItem.fixedCoord, nil, nil,
			decItem.prevDec, decItem.randBytes, &decItem.mc.absMemberChecker)
		if newMemberPubs != nil || newAllPubs != nil {
			panic("non-random members should not change, TODO allow this")
		}
	}
	return newMc
}

// IsReady returns true
func (mc *LaterMemberChecker) IsReady() bool {
	return true
}

type decStats struct {
	fixedCoord sig.Pub
	prevDec    []byte
	randBytes  [32]byte
	mc         *LaterMemberChecker
	genUntil   uint64 // everything with higher index must use this set to generate their mc
}

// UpdateState updates the random member checker state.
func (mc *LaterMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface,
	futureFixed types.ConsensusID) (newMemberPubs,
	newAllPubs []sig.Pub, changedMembers bool) {

	_ = prevMember
	if len(prevDec) > 0 {
		if !prevSM.GetDecided() {
			panic("should have updated the SM first")
		}
	}

	if len(prevDec) != 0 { // We only update the state if the decision was non-nil
		// Remove any decStatus items no longer needed
		newSli := make([]decStats, 0, len(mc.gs.decItems))
		for i := len(mc.gs.decItems) - 1; i >= 0; i-- {
			newSli = append(newSli, mc.gs.decItems[i])
			if mc.gs.decItems[i].genUntil <= mc.myIndex {
				break
			}
		}
		// reverse the slice (since we created it backwards)
		for i, j := 0, len(newSli)-1; i < j; i, j = i+1, j-1 {
			newSli[i], newSli[j] = newSli[j], newSli[i]
		}
		mc.gs.decItems = append(newSli,
			decStats{
				fixedCoord: fixedCoord,
				prevDec:    prevDec,
				randBytes:  randBytes,
				mc:         mc,
				genUntil:   uint64(futureFixed.(types.ConsensusInt)),
			})
		return nil, nil, mc.absMemberChecker.rndMemberCount > 0
	}
	return nil, nil, false
}

// FinishUpdateState does nothing.
func (mc *LaterMemberChecker) FinishUpdateState() {
}
