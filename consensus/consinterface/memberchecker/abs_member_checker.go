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
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

type pubAndID struct {
	sig.Pub
	sig.PubKeyID
}

// absMemberChecker implements part of the MemberChecker interface that can be reused by different
// implementations of member checkers.
type absMemberChecker struct {
	absRandMemberInterface
	config         *generalconfig.GeneralConfig
	myPriv         sig.Priv
	rndMemberCount int // if greater than 0, then this many members are chosen randomly
	rndMemberType  types.RndMemberType

	idx         types.ConsensusIndex
	stats       stats.StatsInterface
	gotDecision bool
	rotateCord  bool // if true then the cordinator will rotate with each consensus index if supported

	// These need to be recomputed on change
	fixedCoord       sig.Pub                  // if we have a fixed coordinator
	fixedCoordID     sig.PubKeyID             // the id of the fixed coordinator
	memberPubStrings map[sig.PubKeyID]sig.Pub // A map of the members.
	sortedMemberPubs sig.PubList              // only member keys
	otherPubs        sig.PubList              // non members
	allPubs          sig.PubList              // non-members appended to members
}

func (mc *absMemberChecker) copyPrevMemberState(prev *absMemberChecker) {
	mc.sortedMemberPubs = prev.sortedMemberPubs
	mc.otherPubs = prev.otherPubs
	mc.allPubs = prev.allPubs
	mc.memberPubStrings = prev.memberPubStrings
	mc.fixedCoord = prev.fixedCoord
	mc.fixedCoordID = prev.fixedCoordID
	mc.myPriv = prev.myPriv
}

// GetMyPriv returns the local node's private key.
func (mc *absMemberChecker) GetMyPriv() sig.Priv {
	return mc.myPriv
}

func (mc *absMemberChecker) GetStats() stats.StatsInterface {
	return mc.stats
}

// CheckRoundCoord returns the public key of the coordinator for round round.
// It is just the mod of the round in the sorted list of public keys.
func (mc *absMemberChecker) CheckRoundCoord(msgID messages.MsgID, checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {

	return getRoundCord(mc, mc.idx.Index, checkPub, round)
}

// CheckEstimatedRoundCoordNextIndex returns the estimated public key of the coordinator for round round for the following consensus index.
// It might not return the correct coordinator for the next index because the decision might change the membership
// for the next index. This function should not be used if it needs to know the next coordinator for certain.
// It returns an error if there is a fixed cordinator or random membership is enabled as it is unsupported in these cases.
func (mc *absMemberChecker) CheckEstimatedRoundCoordNextIndex(checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {

	if mc.fixedCoord != nil || mc.RandMemberType() != types.NonRandom {
		panic("unsupported")
	}

	return getRoundCord(mc, mc.idx.Index.(types.ConsensusInt)+1, checkPub, round)
}

func (mc *absMemberChecker) DoneNextUpdateState() error {
	if mc.absRandMemberInterface != nil {
		return mc.absRandMemberInterface.rndDoneNextUpdateState()
	}
	return nil
}

// CheckFixedCoord checks if checkPub the/a coordinator for round. If it is, err is returned as nil, otherwise an
// error is returned. If checkPub is nil, then it will return the fixed coordinator in coordPub.
// If there is no fixedCoordinator, then an error types.ErrNoFixedCoord is returned.
// Note this should only be called after the pub is verified to be a member with CheckMemberBytes
func (mc *absMemberChecker) CheckFixedCoord(pub sig.Pub) (coordPub sig.Pub, err error) {
	if mc.fixedCoord == nil {
		return nil, types.ErrNoFixedCoord
	}
	if pub == nil {
		return mc.fixedCoord, nil
	}
	return mc.checkCoordInternal(0, mc.fixedCoord, pub)
}

// SelectRandMembers returns true if the member checker is selecting random members.
func (mc *absMemberChecker) RandMemberType() types.RndMemberType {
	return mc.rndMemberType
}

func initAbsMemberChecker(localRand *rand.Rand, rotateCord bool, rndMemberCount int, rndMemberType types.RndMemberType,
	localRandChangeFrequency types.ConsensusInt, priv sig.Priv, config *generalconfig.GeneralConfig) *absMemberChecker {

	if priv == nil {
		panic("priv should not be nil")
	}

	stats := config.Stats

	ret := &absMemberChecker{rotateCord: rotateCord,
		config:         config,
		stats:          stats,
		rndMemberCount: rndMemberCount,
		rndMemberType:  rndMemberType,
		myPriv:         priv}
	switch rndMemberType {
	case types.VRFPerCons:
		ret.absRandMemberInterface = initAbsRandMemberChecker(priv, stats, config)
	case types.VRFPerMessage:
		ret.absRandMemberInterface = initAbsRandMemberCheckerByID(priv, stats, config)
	case types.KnownPerCons:
		ret.absRandMemberInterface = initAbsRoundKnownMemberChecker(stats)
	case types.LocalRandMember:
		ret.absRandMemberInterface = initAbsRandLocalKnownMemberChecker(localRand, priv,
			localRandChangeFrequency, stats)
	case types.NonRandom:
	// no abs rnd member checker
	default:
		panic(rndMemberType)
	}
	return ret
}

// GetNewPub returns an empty public key object.
func (mc *absMemberChecker) GetNewPub() sig.Pub {
	return mc.sortedMemberPubs[0].New()
}

// AbsGotDecision should be called at the end of UpdateState from the MemberChecker.
// The newAllPubs is a list of all new pubs in the system (note this list does not need to be
// sorted as it will be sorted here). If it is nil then the set of pubs must have
// not changed since the last consensus instance. These pubs will be sorted and returned to insure that
// their new ids are computed correctly.
// newMemberPubs pubs are the public keys that will now participate in consensus.
// newMemberPubs must be a equal to or a subset of newAllPubs.
// newMemberPubs should also be nil if they have not changed since the previous consensus index.
// myPubIndex is the index of the local nodes public key in the list, if it is negative then this nodes
// public key should not be in the  list (i.e. is not a member).
func (mc *absMemberChecker) AbsGotDecision(newFixedCoord sig.Pub, newMemberPubs, newOtherPubs sig.PubList,
	prevDec []byte, randBytes [32]byte, prevMember *absMemberChecker) (retMemberPubs, retAllPubs []sig.Pub) {

	if mc.gotDecision {
		panic("called got decision twice")
	}
	if !prevMember.GetIndex().Index.IsInitIndex() && !prevMember.gotDecision {
		panic("previous did not get decision")
	}
	// First copy the previous state
	// mc.copyPrevMemberState(prevMember)

	mc.gotDecision = true

	if (mc.fixedCoord == nil && newFixedCoord != nil) ||
		(mc.fixedCoord != nil && newFixedCoord == nil) {

		panic("cannot move between having a fixed coord an no fixed coord")
	}

	// if we have new pubs sort and store them
	if len(newMemberPubs) > 0 {
		mc.sortedMemberPubs = make(sig.PubList, len(newMemberPubs))
		copy(mc.sortedMemberPubs, newMemberPubs)
	}
	if len(newOtherPubs) > 0 {
		// make a copy // TODO is this needed?
		mc.otherPubs = make(sig.PubList, len(newOtherPubs))
		copy(mc.otherPubs, newOtherPubs)
		//mc.allPubs[0].SetIndex()
		// Sort all pubs because that is from where we compute IDs.
		// sort.Sort(mc.sortedAllPubs)
	}

	// Check if we changed the fixed coord
	if newFixedCoord != nil {
		if sig.CheckPubsEqual(mc.fixedCoord, newFixedCoord) {
			newFixedCoord = nil
		}
	}

	// If we only changed the coord then check if the coord is already a member, then we don't have to change the members
	if len(newOtherPubs) == 0 && len(newMemberPubs) == 0 && newFixedCoord != nil {
		for _, nxt := range mc.sortedMemberPubs {
			if sig.CheckPubsEqual(nxt, newFixedCoord) {
				mc.fixedCoord = nxt
				newFixedCoord = nil
				break
			}
		}
	}

	// Check if the membership has changed.
	if len(newOtherPubs) > 0 || len(newMemberPubs) > 0 || newFixedCoord != nil {

		if _, ok := (interface{})(mc.myPriv).(sig.ThreshStateInterface); ok {
			panic("threshold keys do not support membership change")
		}

		switch mc.myPriv.(type) {
		case *ed.PartialPriv, *bls.PartPriv: // TODO use general check
			panic("threshold keys do not support membership change")
		}

		// We have to recompute the ids for the pubs since we added new ones
		mc.myPriv, mc.fixedCoord, mc.sortedMemberPubs, mc.otherPubs, mc.memberPubStrings, mc.allPubs = sig.AfterSortPubs(
			mc.myPriv, mc.fixedCoord, mc.sortedMemberPubs, mc.otherPubs)

		if mc.fixedCoord != nil {
			var err error
			if mc.fixedCoordID, err = mc.fixedCoord.GetPubID(); err != nil {
				panic(err)
			}
		}
		// mc.allPubs = append(mc.sortedMemberPubs, mc.otherPubs...)

		// We are done choosing the static members, so set the return values
		// (we change them here since if membership has not changed then we dont modify them)
		retMemberPubs = mc.sortedMemberPubs
		retAllPubs = mc.allPubs
	}

	if mc.rndMemberCount > 0 {
		if len(prevDec) > 0 { // only sent the new random bytes if a non-nil decision
			mc.gotRand(randBytes, utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs)), mc.myPriv, retMemberPubs,
				prevMember.absRandMemberInterface)
		} else { // otherwise we keep the same random bytes
			mc.gotRand(prevMember.getRnd(), utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs)), mc.myPriv, retMemberPubs,
				prevMember.absRandMemberInterface)
		}
	}

	return
}

func (mc *absMemberChecker) checkCoordInternal(round types.ConsensusRound, cordPub, checkPub sig.Pub) (sig.Pub, error) {

	cordPubID, err := cordPub.GetPubID()
	if err != nil {
		panic(err)
	}
	sPub, err := checkPub.GetPubID()
	if err != nil {
		return nil, err
	}
	if sPub != cordPubID {
		return nil, types.ErrInvalidRoundCoord
	}
	return cordPub, nil
}

// newAbsMc creates a new abstract membercheck with the same internals as mc (but with the new index).
func (mc *absMemberChecker) newAbsMc(idx types.ConsensusIndex) *absMemberChecker {
	newStats := mc.stats.New(idx)
	newMc := &absMemberChecker{}
	if mc.rndMemberCount > 0 {
		newMc.absRandMemberInterface = mc.absRandMemberInterface.newRndMC(idx, newStats)
		if idx.Index.IsInitIndex() {
			// we do this here because we need to calculate the new values
			newMc.gotRand(mc.getRnd(), utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs)), mc.myPriv, nil,
				mc.absRandMemberInterface)
		}
	}
	newMc.stats = newStats
	newMc.fixedCoord = mc.fixedCoord
	newMc.fixedCoordID = mc.fixedCoordID
	newMc.rndMemberCount = mc.rndMemberCount
	newMc.rndMemberType = mc.rndMemberType
	newMc.memberPubStrings = mc.memberPubStrings
	newMc.sortedMemberPubs = mc.sortedMemberPubs
	newMc.idx = idx
	newMc.myPriv = mc.myPriv
	// newMc.rndStats = mc.rndStats
	newMc.allPubs = mc.allPubs
	newMc.rotateCord = mc.rotateCord
	newMc.otherPubs = mc.otherPubs
	return newMc
}

// SetMainChannel is called on the initial member checker to inform it of the network channel object
func (mc *absMemberChecker) SetMainChannel(mainChannel channelinterface.MainChannel) {
	if mc.absRandMemberInterface != nil {
		mc.absRandMemberInterface.setMainChannel(mainChannel)
	}
}

// AddPubKeys is used for adding the public keys to the very inital member checker, and should not
// be called on later member checkers, as they should change pub keys through the call to UpdateState.
// allPubs are the list of all pubs in the system.
// memberPubs pubs are the public keys that will now participate in consensus.
// memberPubs must be equal to or a subset of newAllPubs.
// If shared is non-nil then the local nodes on the machine will share the same initial member objects
// (to save memory for experiments that run many nodes on the same machine).
func (mc *absMemberChecker) AddPubKeys(fixedCoord sig.Pub, memberPubKeys, otherPubs sig.PubList, initRandBytes [32]byte,
	shared *consinterface.Shared) {

	if shared == nil {
		mc.sortedMemberPubs = make(sig.PubList, len(memberPubKeys))
		copy(mc.sortedMemberPubs, memberPubKeys)
		mc.otherPubs = make(sig.PubList, len(otherPubs))
		copy(mc.otherPubs, otherPubs)
		mc.fixedCoord = fixedCoord

		mc.myPriv, mc.fixedCoord, mc.sortedMemberPubs, mc.otherPubs, mc.memberPubStrings, mc.allPubs = sig.AfterSortPubs(
			mc.myPriv, mc.fixedCoord, mc.sortedMemberPubs, mc.otherPubs)
	} else {
		mc.myPriv, mc.fixedCoord, mc.sortedMemberPubs, mc.otherPubs, mc.memberPubStrings, mc.allPubs = shared.AfterSortPubs(
			mc.myPriv, fixedCoord, memberPubKeys, otherPubs)
	}

	if mc.fixedCoord != nil {
		var err error
		if mc.fixedCoordID, err = mc.fixedCoord.GetPubID(); err != nil {
			panic(err)
		}
	}
	// mc.allPubs = append(mc.sortedMemberPubs, mc.otherPubs...)

	if mc.rndMemberCount > 0 {
		mc.gotRand(initRandBytes, utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs)),
			mc.myPriv, mc.sortedMemberPubs, nil)
	}
}

func (mc *absMemberChecker) GetMyVRF(id messages.MsgID) sig.VRFProof {
	if mc.rndMemberCount > 0 {
		return mc.getMyVRF(id)
	}
	return nil
}

// GetMemberCount returns the number of consensus members.
func (mc *absMemberChecker) GetMemberCount() int {
	if mc.rndMemberCount > 0 {
		return utils.Min(len(mc.sortedMemberPubs), mc.rndMemberCount)
	}
	return len(mc.sortedMemberPubs)
}

// GetParticipant return the pub key of the participants.
func (mc *absMemberChecker) GetParticipants() sig.PubList {
	return mc.sortedMemberPubs
}

// GetAllPub returns all pubs in the system
func (mc *absMemberChecker) GetAllPubs() sig.PubList {
	return mc.allPubs
}

// GetParticipantCount returns the total number of nodes in the system
func (mc *absMemberChecker) GetParticipantCount() int {
	return len(mc.sortedMemberPubs) + len(mc.otherPubs)
}

// GetFaultCount returns the number of possilbe faultly nodes that the consensus can handle.
func (mc *absMemberChecker) GetFaultCount() int {
	// TODO dont do this on the fly
	return utils.GetOneThirdBottom(mc.GetMemberCount())
}

func (mc *absMemberChecker) GetIndex() types.ConsensusIndex {
	return mc.idx
}

// CheckIndex should return true if the index is the same as the one used in New, it is for testing.
func (mc *absMemberChecker) CheckIndex(idx types.ConsensusIndex) bool {
	if mc.idx.Index != idx.Index {
		return false
	}
	return true
}

// ValidatedItem is called for stats to let it know a signature was validated. TODO cleanup.
func (mc *absMemberChecker) Validated(signType types.SignType) {
	if mc.stats != nil {
		switch signType {
		case types.CoinProof:
			mc.stats.ValidatedCoinShare()
		default:
			mc.stats.ValidatedItem()
		}
	}
}

// CheckMemberBytes checks the the pub key is a member and returns corresponding pub key object
func (mc *absMemberChecker) CheckMemberBytes(idx types.ConsensusIndex, pubBytes sig.PubKeyID) sig.Pub {
	if idx.Index != mc.idx.Index {
		panic("wrong member checker index")
	}
	var ret sig.Pub
	ret = mc.memberPubStrings[pubBytes]
	if ret != nil {
		return ret
	}
	return nil
}

// CheckRandMember can be called after CheckMemberBytes is successful when ChooseRandomMember is enabled.
// If ChooseRandomMember is false it return nil,
// otherwise it uses the random bytes and the VRF to decide if this pud is a random member.
func (mc *absMemberChecker) CheckRandMember(pub sig.Pub, msgID messages.MsgID, isProposalMsg bool) error {
	if mc.rndMemberCount == 0 {
		return nil
	}
	return mc.checkRandMember(msgID, isProposalMsg, utils.Min(mc.rndMemberCount, len(mc.sortedMemberPubs)), len(mc.sortedMemberPubs), pub)
}
