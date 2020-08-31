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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

// AbsRandLocalKnownMemberChecker uses a set of locally chosen member for the consensus.
type AbsRandLocalKnownMemberChecker struct {
	myPriv           sig.Priv
	myRand           *rand.Rand
	rndStats         stats.StatsInterface
	sortedMemberPubs sig.PubList
	// randMemberCount  int
	pubMap map[sig.PubKeyID]sig.Pub

	pubList []sig.Pub // randomly chosen members

	depthCount types.ConsensusInt // distance from inital consensus index

	perm                     []int
	index                    types.ConsensusIndex
	localRandChangeFrequency types.ConsensusInt // how often to change random members
	mainChannel              channelinterface.MainChannel
}

func initAbsRandLocalKnownMemberChecker(rnd *rand.Rand, priv sig.Priv,
	localRandChangeFrequency types.ConsensusInt, stats stats.StatsInterface) *AbsRandLocalKnownMemberChecker {

	ret := &AbsRandLocalKnownMemberChecker{}
	ret.rndStats = stats
	ret.myPriv = priv
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	ret.myRand = rnd
	ret.localRandChangeFrequency = localRandChangeFrequency
	return ret
}

func (arm *AbsRandLocalKnownMemberChecker) setMainChannel(mainChannel channelinterface.MainChannel) {
	arm.mainChannel = mainChannel
	arm.makeConnections()
}

func (arm *AbsRandLocalKnownMemberChecker) makeConnections() {
	// return
	if arm.mainChannel != nil {
		if errs := arm.mainChannel.MakeConnections(arm.pubList); len(errs) > 0 {
			logging.Error(errs)
		}
	}
}

// rndDoneNextUpdate calls mainChannel.RemoveConnections on the nodes used for this consensus instance.
func (arm *AbsRandLocalKnownMemberChecker) rndDoneNextUpdateState() error {
	if errs := arm.mainChannel.RemoveConnections(arm.pubList); len(errs) > 0 {
		logging.Error(errs)
		return fmt.Errorf("%v", errs)
	}
	return nil
}

func (arm *AbsRandLocalKnownMemberChecker) gotRand(_ [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, memberMap map[sig.PubKeyID]sig.Pub, prvMC absRandMemberInterface) {

	arm.myPriv = newPriv
	myPub := arm.myPriv.GetPub()
	myPid, err := myPub.GetPubID()
	if err != nil {
		panic(err)
	}

	if prvMC != nil {
		arm.depthCount = prvMC.(*AbsRandLocalKnownMemberChecker).depthCount + 1
	}

	if sortedMemberPubs == nil {
		arm.sortedMemberPubs = prvMC.(*AbsRandLocalKnownMemberChecker).sortedMemberPubs
	} else {
		arm.sortedMemberPubs = sortedMemberPubs
	}

	// keep the same random membership if sortedMemberPubs is nil, unless it is the first index,
	// or a change round
	if sortedMemberPubs == nil && (arm.index.Index.IsInitIndex() || arm.depthCount%arm.localRandChangeFrequency != 0) {
		prev := prvMC.(*AbsRandLocalKnownMemberChecker)
		arm.perm = prev.perm
		arm.pubMap = prev.pubMap
		arm.pubList = prev.pubList
		//if arm.index.Index.IsInitIndex() { // make connections on the first iteration
		// arm.makeConnections()
		//}
	} else { // new membership
		arm.perm = utils.GenRandPerm(participantNodeCount, len(arm.sortedMemberPubs), arm.myRand)
		arm.pubList = make([]sig.Pub, participantNodeCount)
		var gotMyPid bool
		var lastPid sig.PubKeyID
		for i, v := range arm.perm {
			p := arm.sortedMemberPubs[v]
			pid, err := p.GetPubID()
			if err != nil {
				panic(err)
			}
			if pid == myPid {
				gotMyPid = true
			}
			lastPid = pid
			arm.pubMap[pid] = p
			arm.pubList[i] = p
		}
		if !gotMyPid && memberMap[myPid] != nil { // we have to add ourselves as a member if we haven't already (and we are a normal member)
			delete(arm.pubMap, lastPid)
			arm.pubList[len(arm.pubList)-1] = myPub
			arm.pubMap[myPid] = myPub
		}
		// Here we are generating a new cons item, so in case of causal ordering it is where we have
		// received a valid proposal
	}
	arm.makeConnections()
}

func (arm *AbsRandLocalKnownMemberChecker) checkRandMember(msgID messages.MsgID, isProposalMsg bool, participantNodeCount,
	totalNodeCount int, pub sig.Pub) error {

	_, _, _ = msgID, participantNodeCount, totalNodeCount
	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	if isProposalMsg { // if it's proposal then all members are valid, not just the rand ones
		return nil
	}
	if _, ok := arm.pubMap[pid]; ok {
		return nil
	}

	// TODO if the msg is a proposal, then the message is a member as long as it is in sorted pub list
	// can use InternalSignedMsgHeader.NeedsSMValidation

	return types.ErrNotMember
}
func (arm *AbsRandLocalKnownMemberChecker) checkRandCoord(_, _ int,
	_ messages.MsgID, _ types.ConsensusRound, _ sig.Pub) (rndVal uint64, coord sig.Pub, err error) {

	panic("unused, the normal coordinator should be chosen")
}
func (arm *AbsRandLocalKnownMemberChecker) GotVrf(sig.Pub, messages.MsgID, sig.VRFProof) error {
	return nil
}
func (arm *AbsRandLocalKnownMemberChecker) getMyVRF(messages.MsgID) sig.VRFProof {
	return nil
}
func (arm *AbsRandLocalKnownMemberChecker) getRnd() (ret [32]byte) {
	return
}
func (arm *AbsRandLocalKnownMemberChecker) newRndMC(index types.ConsensusIndex,
	stats stats.StatsInterface) absRandMemberInterface {

	ret := &AbsRandLocalKnownMemberChecker{}
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	ret.rndStats = stats
	ret.mainChannel = arm.mainChannel
	ret.myRand = arm.myRand
	ret.myPriv = arm.myPriv
	ret.index = index
	ret.localRandChangeFrequency = arm.localRandChangeFrequency
	return ret
}
