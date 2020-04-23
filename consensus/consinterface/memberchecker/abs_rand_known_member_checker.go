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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

type absRoundKnownMemberChecker struct {
	pubMap           map[sig.PubKeyID]sig.Pub
	sortedMemberPubs sig.PubList
	perm             []int
	rnd              [32]byte
	rndStats         stats.StatsInterface
}

func initAbsRoundKnownMemberChecker(stats stats.StatsInterface) *absRoundKnownMemberChecker {
	ret := &absRoundKnownMemberChecker{rndStats: stats}
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	return ret
}

// rndDoneNextUpdate state does nothing here.
func (arm *absRoundKnownMemberChecker) rndDoneNextUpdateState() error {
	return nil
}

func (arm *absRoundKnownMemberChecker) setMainChannel(mainChannel channelinterface.MainChannel) {}

func (arm *absRoundKnownMemberChecker) gotRand(rnd [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, prvMC absRandMemberInterface) {

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

func (arm *absRoundKnownMemberChecker) checkRandMember(msgID messages.MsgID, isProposalMsg bool,
	participantNodeCount, totalNodeCount int, pub sig.Pub) error {
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
func (arm *absRoundKnownMemberChecker) GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error {
	return nil
}
func (arm *absRoundKnownMemberChecker) getMyVRF(id messages.MsgID) sig.VRFProof {
	return nil
}
func (arm *absRoundKnownMemberChecker) getRnd() [32]byte {
	return arm.rnd
}
func (arm *absRoundKnownMemberChecker) setRndStats(stats.StatsInterface) {

}
func (arm *absRoundKnownMemberChecker) newRndMC(index types.ConsensusIndex,
	stats stats.StatsInterface) absRandMemberInterface {

	ret := &absRoundKnownMemberChecker{rndStats: stats}
	ret.pubMap = make(map[sig.PubKeyID]sig.Pub)
	return ret
}
