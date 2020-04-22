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
	"math"
	"math/rand"
	"sync"
)

type rndNodeInfo struct {
	// the initial random value from a node comes from the VRF
	values []uint64
	// if the consensus executes in rounds, then the random value for each node is computed deterministically
	// note this is only used for computing coordinators for each round
	rand *rand.Rand
}

type absRandMemberChecker struct {
	myIdx  types.ConsensusIndex
	myPriv sig.Priv
	myVrf  sig.VRFProof
	rnd    [32]byte // The random bytes

	randBasicMsg          sig.BasicSignedMessage // The index appended to the random bytes, this is what we use to compute the random per node
	randUint64            uint64
	vrfRand               map[sig.PubKeyID]*rndNodeInfo
	coordinatorRelaxation uint64
	rndLock               sync.RWMutex
	rndStats              stats.StatsInterface
}

func initAbsRandMemberChecker(priv sig.Priv, stats stats.StatsInterface) *absRandMemberChecker {
	return &absRandMemberChecker{
		rndStats:              stats,
		myPriv:                priv,
		coordinatorRelaxation: config.DefaultCoordinatorRelaxtion,
		vrfRand:               make(map[sig.PubKeyID]*rndNodeInfo)}
}

func (arm *absRandMemberChecker) newRndMC(idx types.ConsensusIndex, stats stats.StatsInterface) absRandMemberInterface {
	ret := initAbsRandMemberChecker(arm.myPriv, stats)
	ret.myIdx = idx
	return ret
}

// rndDoneNextUpdate state does nothing here.
func (arm *absRandMemberChecker) rndDoneNextUpdateState() error {
	return nil
}

func (arm *absRandMemberChecker) setRndStats(stats stats.StatsInterface) {
	arm.rndStats = stats
}

func (arm *absRandMemberChecker) getRnd() [32]byte {
	return arm.rnd
}

// getMyVRF returns the vrf proof for the local node.
func (arm *absRandMemberChecker) getMyVRF(msgID messages.MsgID) sig.VRFProof {
	if arm.myVrf == nil {
		panic("shouldnt be nil")
	}
	return arm.myVrf
}

// setCoordinatorRelaxation sets additional relaxation for choosing the coordinator.
// The reason is that there might not be a node with small enough rand value to be chosen at default.
func (arm *absRandMemberChecker) setCoordinatorRelaxation(percentage int) {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}
	arm.coordinatorRelaxation = uint64(percentage)
}

// gotRand should be called with the random bytes received from the state machine after deciding the previous
// consensus instance.
func (arm *absRandMemberChecker) gotRand(rnd [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, prvMC absRandMemberInterface) {

	arm.rnd = rnd
	arm.myPriv = newPriv

	// the rnd number is the first 8 bytes of the random string
	arm.randUint64 = config.Encoding.Uint64(rnd[:])

	// if myIdx is nil then this is the init member checker so we don't need the vrf
	if arm.myIdx.Index == nil { // TODO fix this
		return
	}

	// The random message is the consensus index appended to the random bytes
	m := messages.NewMsgBuffer()
	m.AddConsensusID(arm.myIdx.Index)
	m.AddBytes(rnd[:])

	arm.randBasicMsg = sig.BasicSignedMessage(m.GetRemainingBytes())

	// compute our own vrf
	_, prf := arm.myPriv.Evaluate(arm.randBasicMsg)
	if arm.rndStats != nil {
		arm.rndStats.CreatedVRF()
	}
	if err := arm.GotVrf(arm.myPriv.GetPub(), nil, prf); err != nil {
		panic(err)
	}
	arm.myVrf = prf
}

// checkRandMember uses the VRFs to determine if pub can participate in this consensus.
// participantNodeCount is the number of participants expected for this consensus.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandMemberChecker) checkRandMember(msgID messages.MsgID, isProposalMsg bool, participantNodeCount, totalNodeCount int, pub sig.Pub) error {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	// the nodes random bytes converted to a uint64
	arm.rndLock.RLock()
	rndInfo, ok := arm.vrfRand[pid]
	arm.rndLock.RUnlock()
	if !ok {
		return types.ErrNotReceivedVRFProof
	}
	rnd := rndInfo.values[0]
	// the threshold for the given number of nodes
	// thrsh := uint64((float64(participantNodeCount)/float64(totalNodeCount))*float64(math.MaxUint64))
	onePc := uint64(math.MaxUint64) / 100
	percentage := uint64(utils.Min((participantNodeCount*100)/totalNodeCount+config.DefaultNodeRelaxation, 100))
	thrsh := onePc * percentage
	if rnd <= thrsh {
		return nil
	}
	return types.ErrNotMember
}

func (arm *absRandMemberChecker) setMainChannel(mainChannel channelinterface.MainChannel) {}

// checkRandCoord uses the VRFs to determine if pub is a valid coordinator for this consensus.
// Note due to the random function, there can be 0 or multiple valid coordinators.
// participantNodeCount is the number of participants expected for this node.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandMemberChecker) checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID,
	round types.ConsensusRound, pub sig.Pub) (rndVal uint64, coord sig.Pub, err error) {

	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}

	// first check if we are a member, we need this since we rotate the cord following the first round
	if err = arm.checkRandMember(msgID, true, participantNodeCount, totalNodeCount, pub); err != nil { // Check if we are a member
		return 0, nil, err
	}

	// the nodes random bytes converted to a uint64
	arm.rndLock.Lock()
	defer arm.rndLock.Unlock()

	rndInfo, ok := arm.vrfRand[pid]
	if !ok {
		return 0, nil, types.ErrNotReceivedVRFProof
	}

	for i := len(rndInfo.values) - 1; types.ConsensusRound(i) < round; i++ {
		if rndInfo.rand == nil {
			rndInfo.rand = rand.New(rand.NewSource(int64(rndInfo.values[0])))
		}
		rndInfo.values = append(rndInfo.values, rndInfo.rand.Uint64())
	}

	// the threshold for the given number of nodes
	onePc := uint64(math.MaxUint64) / 100
	thrsh := uint64(float64(onePc) * float64(arm.coordinatorRelaxation) * (float64(100) / float64(totalNodeCount))) //arm.coordinatorRelaxation

	// thrsh := uint64((float64(1)/float64(totalNodeCount))*math.MaxUint64)
	if rndInfo.values[round] <= thrsh {
		return rndInfo.values[round], pub, nil
	}
	return 0, nil, types.ErrNotMember
}

// GotVrf should be called when a node's VRF proof is received for this consensus instance.
func (arm *absRandMemberChecker) GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}

	arm.rndLock.Lock()
	_, ok := arm.vrfRand[pid]
	arm.rndLock.Unlock()
	if !ok {
		rndByte, err := pub.ProofToHash(arm.randBasicMsg, proof)
		if arm.rndStats != nil {
			arm.rndStats.ValidatedVRF()
		}
		if err != nil {
			return err
		}
		seed := config.Encoding.Uint64(rndByte[:])
		rndInfo := &rndNodeInfo{values: []uint64{seed}}
		arm.rndLock.Lock()
		arm.vrfRand[pid] = rndInfo
		arm.rndLock.Unlock()
	}
	return nil
}
