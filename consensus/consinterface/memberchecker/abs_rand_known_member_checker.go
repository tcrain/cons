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
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

// absRoundKnowMemberChecker uses a single VRF (from the previous consensus) to determine the set of all members for the consensus.
type absRoundKnownMemberChecker struct {
	pubMap           map[sig.PubKeyID]sig.Pub
	sortedMemberPubs sig.PubList
	perm             []int
	rnd              [32]byte
	rndStats         stats.StatsInterface

	cidrand *absRandCoordByID
}

func (arm *absRoundKnownMemberChecker) GetRnd() [32]byte {
	return arm.rnd
}

func initAbsRoundKnownMemberChecker(priv sig.Priv, stats stats.StatsInterface,
	gc *generalconfig.GeneralConfig) *absRoundKnownMemberChecker {

	ret := &absRoundKnownMemberChecker{rndStats: stats}
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	if gc.UseRandCoord {
		// coordinators have their own VRFs, note that these are only calculated from within the existing random members
		// so the coordinator relaxation may need to be higher
		ret.cidrand = initAbsRandCoordByID(priv, stats, gc)
	}
	return ret
}

func (arm *absRoundKnownMemberChecker) newRndMC(idx types.ConsensusIndex,
	stats stats.StatsInterface) absRandMemberInterface {

	ret := &absRoundKnownMemberChecker{rndStats: stats}
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	if arm.cidrand != nil {
		ret.cidrand = arm.cidrand.newRndMC(idx, stats).(*absRandCoordByID)
	}
	return ret
}

// rndDoneNextUpdate state does nothing here.
func (arm *absRoundKnownMemberChecker) rndDoneNextUpdateState() error {
	return nil
}

func (arm *absRoundKnownMemberChecker) setMainChannel(channelinterface.MainChannel) {}

func (arm *absRoundKnownMemberChecker) gotRand(rnd [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, pubMap map[sig.PubKeyID]sig.Pub, prvMC absRandMemberInterface) {

	if arm.cidrand != nil {
		arm.cidrand.gotRand(rnd, participantNodeCount, newPriv, sortedMemberPubs, pubMap, prvMC)
	}
	arm.rnd = rnd
	rndGen := rand.New(rand.NewSource(int64(config.Encoding.Uint64(rnd[:]))))
	if len(sortedMemberPubs) == 0 {
		arm.sortedMemberPubs = prvMC.(*absRoundKnownMemberChecker).sortedMemberPubs
	} else {
		arm.sortedMemberPubs = sortedMemberPubs
	}
	arm.perm = utils.GenRandPerm(participantNodeCount, len(arm.sortedMemberPubs), rndGen)
	for _, v := range arm.perm {
		p := arm.sortedMemberPubs[v]
		pid, err := p.GetPubID()
		if err != nil {
			panic(err)
		}
		arm.pubMap[pid] = p
	}
}

func (arm *absRoundKnownMemberChecker) checkRandMember(msgID messages.MsgID, isLocal, isProposalMsg bool,
	participantNodeCount, totalNodeCount int, pub sig.Pub) error {

	_, _, _, _, _ = msgID, isLocal, isProposalMsg, participantNodeCount, totalNodeCount
	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	if _, ok := arm.pubMap[pid]; ok {
		return nil
	}
	return types.ErrNotMember
}
func (arm *absRoundKnownMemberChecker) checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID,
	round types.ConsensusRound, pub sig.Pub) (rndVal uint64, coord sig.Pub, err error) {

	if arm.cidrand != nil {
		return arm.cidrand.checkRandCoord(participantNodeCount, totalNodeCount, msgID, round, pub)
	}

	rndIdx := utils.Abs(int(round)) % len(arm.perm)
	coord = arm.sortedMemberPubs[arm.perm[rndIdx]]
	if pub == nil {
		return 0, coord, nil
	}
	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	coordPid, err := coord.GetPubID()
	if err != nil {
		panic(err)
	}
	if pid == coordPid {
		return 0, coord, nil
	}
	return 0, nil, types.ErrNotMember
}
func (arm *absRoundKnownMemberChecker) GotVrf(pub sig.Pub, isProposal bool, msgID messages.MsgID, vrf sig.VRFProof) error {
	if arm.cidrand != nil {
		return arm.cidrand.GotVrf(pub, isProposal, msgID, vrf)
	}
	return nil
}
func (arm *absRoundKnownMemberChecker) getMyVRF(isProposal bool, msgID messages.MsgID) sig.VRFProof {
	if arm.cidrand != nil {
		return arm.cidrand.getMyVRF(isProposal, msgID)
	}
	return nil
}
func (arm *absRoundKnownMemberChecker) getRnd() [32]byte {
	return arm.rnd
}
