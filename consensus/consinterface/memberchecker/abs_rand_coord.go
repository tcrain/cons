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
	"math"
	"sync"
)

// absRandCoordByID uses VRF to determine members for the consensus, the members are chosen
// for each message type.
type absRandCoordByID struct {
	myIdx  types.ConsensusIndex
	myPriv sig.Priv
	// myVrf                 sig.VRFProof
	rnd            [32]byte
	randBasicMsg   sig.BasicSignedMessage
	randUint64     uint64
	vrfRandByMsgID map[pubMsgID]prfRnd
	rndLock        sync.RWMutex
	rndStats       stats.StatsInterface
	gc             *generalconfig.GeneralConfig
}

func initAbsRandCoordByID(priv sig.Priv, stats stats.StatsInterface, gc *generalconfig.GeneralConfig) *absRandCoordByID {
	ret := &absRandCoordByID{
		rndStats:       stats,
		myPriv:         priv,
		gc:             gc,
		vrfRandByMsgID: make(map[pubMsgID]prfRnd)}
	return ret
}

func (arm *absRandCoordByID) setMainChannel(_ channelinterface.MainChannel) {}

func (arm *absRandCoordByID) newRndMC(idx types.ConsensusIndex, stats stats.StatsInterface) absRandMemberInterface {
	ret := &absRandCoordByID{
		rndStats:       stats,
		myPriv:         arm.myPriv,
		gc:             arm.gc,
		myIdx:          idx,
		vrfRandByMsgID: make(map[pubMsgID]prfRnd)}
	return ret
}

// rndDoneNextUpdate state does nothing here.
func (arm *absRandCoordByID) rndDoneNextUpdateState() error {
	return nil
}

func (arm *absRandCoordByID) getRnd() [32]byte {
	return arm.rnd
}

// gotRand should be called with the random bytes received from the state machine after deciding the previous
// consensus instance.
func (arm *absRandCoordByID) gotRand(rnd [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, _ map[sig.PubKeyID]sig.Pub, prvMC absRandMemberInterface) {

	_, _, _ = participantNodeCount, sortedMemberPubs, prvMC
	arm.myPriv = newPriv
	arm.rnd = rnd
	arm.randBasicMsg = rnd[:]
	arm.randUint64 = config.Encoding.Uint64(arm.randBasicMsg)
}

// getMyVRF returns the vrf proof for the local node.
func (arm *absRandCoordByID) getMyVRF(isProposal bool, msgID messages.MsgID) sig.VRFProof {
	if !isProposal {
		return nil
	}
	return arm.getMyVRFInternal(msgID)
}

func (arm *absRandCoordByID) getMyVRFInternal(msgID messages.MsgID) sig.VRFProof {
	// compute our own vrf
	mypid, err := arm.myPriv.GetPub().GetPubID()
	if err != nil {
		panic(err)
	}
	arm.rndLock.Lock()
	item, ok := arm.vrfRandByMsgID[pubMsgID{pub: mypid, msgID: msgID}]
	arm.rndLock.Unlock()
	if ok {
		return item.prf
	}

	newMsg := arm.computeNewMsg(msgID)
	_, prf := arm.myPriv.(sig.VRFPriv).Evaluate(newMsg)
	if arm.rndStats != nil {
		arm.rndStats.CreatedVRF()
	}
	// add it to our local map
	if err := arm.gotVrfInternal(arm.myPriv.GetPub(), msgID, prf); err != nil {
		panic(err)
	}
	return prf
}

// checkLocal is called to check if we are checking for local membership for the MsgID then we have computed the local VRF
func (arm *absRandCoordByID) checkLocal(isProposal bool, msgID messages.MsgID, pid sig.PubKeyID) {
	mypid, err := arm.myPriv.GetPub().GetPubID()
	if err != nil {
		panic(err)
	}
	if mypid == pid { // we are checking if the local node is a member, so we should first compute our own VRF is needed
		arm.rndLock.RLock()
		_, ok := arm.vrfRandByMsgID[pubMsgID{pub: mypid, msgID: msgID}]
		arm.rndLock.RUnlock()
		if !ok {
			arm.getMyVRF(isProposal, msgID)
		}
	}
}

// checkRandMember returns nil, since here we only determine the coordinator randomly.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandCoordByID) checkRandMember(msgID messages.MsgID, isLocal, isProposalMsg bool,
	participantNodeCount, totalNodeCount int, pub sig.Pub) error {

	_, _, _, _, _ = msgID, isLocal, participantNodeCount, totalNodeCount, pub
	if isProposalMsg {
		panic("should use method for coordinator")
	}
	return nil
}

// checkRandCoord uses the VRFs to determine if pub is a valid coordinator for this consensus.
// Note due to the random function, there can be 0 or multiple valid coordinators.
// participantNodeCount is the number of participants expected for this node.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandCoordByID) checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID,
	round types.ConsensusRound, pub sig.Pub) (rndValue uint64, coord sig.Pub, err error) {

	_, _, totalNodeCount = participantNodeCount, round, totalNodeCount
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	arm.checkLocal(true, msgID, pid) // check if this is a local message

	arm.rndLock.RLock()
	item, ok := arm.vrfRandByMsgID[pubMsgID{pub: pid, msgID: msgID}]
	arm.rndLock.RUnlock()
	if !ok {
		panic(1)
		return 0, nil, types.ErrNotReceivedVRFProof
	}

	// the threshold for the given number of nodes
	onePc := uint64(math.MaxUint64) / 100
	thrsh := uint64(float64(onePc) * float64(arm.gc.CoordChoiceVRF)) //* (float64(100) / float64(totalNodeCount))) //arm.coordinatorRelaxation

	// thrsh := uint64((float64(1)/float64(totalNodeCount))*math.MaxUint64)
	if item.rnd <= thrsh {
		return item.rnd, pub, nil
	}
	return 0, nil, types.ErrNotMember
}

// GotVrf should be called when a node's VRF proof is received for this consensus instance.
func (arm *absRandCoordByID) GotVrf(pub sig.Pub, isProposal bool, msgID messages.MsgID, proof sig.VRFProof) error {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	if !isProposal {
		return nil
	}
	return arm.gotVrfInternal(pub, msgID, proof)
}

func (arm *absRandCoordByID) gotVrfInternal(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error {

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}

	arm.rndLock.Lock()
	_, ok := arm.vrfRandByMsgID[pubMsgID{pub: pid, msgID: msgID}]
	arm.rndLock.Unlock()
	if !ok {
		newMsg := arm.computeNewMsg(msgID)
		rndByte, err := pub.(sig.VRFPub).ProofToHash(newMsg, proof)
		if arm.rndStats != nil {
			arm.rndStats.ValidatedVRF()
		}
		if err != nil {
			return err
		}
		rnd := config.Encoding.Uint64(rndByte[:])
		arm.rndLock.Lock()
		arm.vrfRandByMsgID[pubMsgID{pub: pid, msgID: msgID}] = prfRnd{prf: proof, rnd: rnd}
		arm.rndLock.Unlock()
	}
	return nil
}

func (arm *absRandCoordByID) computeNewMsg(msgID messages.MsgID) sig.BasicSignedMessage {
	// The message we use to compute the VRF is the MsgID (containing the index) plus the random bytes
	m := messages.NewMsgBuffer()
	m.AddBytes(msgID.ToBytes(arm.myIdx))
	m.AddBytes(arm.randBasicMsg)
	return m.GetRemainingBytes()
}
