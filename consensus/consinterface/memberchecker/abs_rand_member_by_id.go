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
	"sync"
)

type absRandMemberInterface interface {
	// gotRand will be called when the consensus index has decided. If random bytes are enabled they will be included.
	// participantNodeCount is the number of pubs that should be selected from sortedMemberPubs to be chosen as members for
	// the consensus index.
	// If sortedMemberPubs is nil they they have not changed since the last consensus index.
	gotRand(rnd [32]byte, participantNodeCount int, myPriv sig.Priv, sortedMemberPubs sig.PubList, prvMC absRandMemberInterface)
	// checkRandMember should return nil if pub should participate in this consensus given the inputs, or an error otherwise.
	checkRandMember(msgID messages.MsgID, isProposalMsg bool, participantNodeCount, totalNodeCount int, pub sig.Pub) error
	// checkRandCoord checks if pub should participate in this consensus as a coordinator given the inputs.
	// If pub is nil then the coordinator pub should be returned as coord.
	// randVal is the random value given to the pub as coordinator, the lower the value the more it is supported as coordinator.
	// Otherwise an error is returned if pub is not a coordinator.
	checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID, round types.ConsensusRound,
		pub sig.Pub) (rndVal uint64, coord sig.Pub, err error)
	// GotVrf should be called when a VRFProof is received for the pub and message type.
	GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error
	// getMyVRF returns the VRFProof for the local node and message type for this consensus.
	getMyVRF(id messages.MsgID) sig.VRFProof
	// getRnd returns the random value given by gotRnd.
	getRnd() [32]byte
	// newRndMC is called to generate a new random member checker for the given index. It should be called on the
	// initial random member checker.
	newRndMC(index types.ConsensusIndex, stats stats.StatsInterface) absRandMemberInterface
	// setMainChannel is called once during initialization to set the main channel object.
	setMainChannel(mainChannel channelinterface.MainChannel)
	// rndDoneNextUpdate state should be called by MemberChecker.DoneNextUpdateState
	rndDoneNextUpdateState() error
}

type pubMsgID struct {
	pub   sig.PubKeyID
	msgID messages.MsgID
}

type prfRnd struct {
	prf sig.VRFProof
	rnd uint64
}

type absRandMemberCheckerByID struct {
	myIdx  types.ConsensusIndex
	myPriv sig.Priv
	// myVrf                 sig.VRFProof
	rnd                   [32]byte
	randBasicMsg          sig.BasicSignedMessage
	randUint64            uint64
	vrfRandByMsgID        map[pubMsgID]prfRnd
	coordinatorRelaxation uint64
	rndLock               sync.RWMutex
	rndStats              stats.StatsInterface
}

func initAbsRandMemberCheckerByID(priv sig.Priv, stats stats.StatsInterface) *absRandMemberCheckerByID {
	return &absRandMemberCheckerByID{
		rndStats:              stats,
		myPriv:                priv,
		coordinatorRelaxation: config.DefaultCoordinatorRelaxtion,
		vrfRandByMsgID:        make(map[pubMsgID]prfRnd)}
}

func (arm *absRandMemberCheckerByID) setMainChannel(mainChannel channelinterface.MainChannel) {}

func (arm *absRandMemberCheckerByID) newRndMC(idx types.ConsensusIndex, stats stats.StatsInterface) absRandMemberInterface {
	ret := initAbsRandMemberCheckerByID(arm.myPriv, stats)
	ret.myIdx = idx
	return ret
}

// rndDoneNextUpdate state does nothing here.
func (arm *absRandMemberCheckerByID) rndDoneNextUpdateState() error {
	return nil
}

// setCoordinatorRelaxation sets additional relaxation for choosing the coordinator.
// The reason is that there might not be a node with small enough rand value to be chosen at default.
func (arm *absRandMemberCheckerByID) setCoordinatorRelaxation(percentage int) {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}
	arm.coordinatorRelaxation = uint64(percentage)
}

func (arm *absRandMemberCheckerByID) getRnd() [32]byte {
	return arm.rnd
}

// gotRand should be called with the random bytes received from the state machine after deciding the previous
// consensus instance.
func (arm *absRandMemberCheckerByID) gotRand(rnd [32]byte, participantNodeCount int, newPriv sig.Priv,
	sortedMemberPubs sig.PubList, prvMC absRandMemberInterface) {

	arm.myPriv = newPriv
	arm.rnd = rnd
	arm.randBasicMsg = sig.BasicSignedMessage(rnd[:])
	arm.randUint64 = config.Encoding.Uint64(arm.randBasicMsg)
}

// getMyVRF returns the vrf proof for the local node.
func (arm *absRandMemberCheckerByID) getMyVRF(msgID messages.MsgID) sig.VRFProof {
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
	if err := arm.GotVrf(arm.myPriv.GetPub(), msgID, prf); err != nil {
		panic(err)
	}
	return prf
}

// checkLocal is called to check if we are checking for local membership for the MsgID then we have computed the local VRF
func (arm *absRandMemberCheckerByID) checkLocal(msgID messages.MsgID, pid sig.PubKeyID) {
	mypid, err := arm.myPriv.GetPub().GetPubID()
	if err != nil {
		panic(err)
	}
	if mypid == pid { // we are checking if the local node is a member, so we should first compute our own VRF is needed
		arm.rndLock.RLock()
		_, ok := arm.vrfRandByMsgID[pubMsgID{pub: mypid, msgID: msgID}]
		arm.rndLock.RUnlock()
		if !ok {
			arm.getMyVRF(msgID)
		}
	}
}

// checkRandMember uses the VRFs to determine if pub can participate in this consensus.
// participantNodeCount is the number of participants expected for this consensus.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandMemberCheckerByID) checkRandMember(msgID messages.MsgID, isProposalMsg bool, participantNodeCount, totalNodeCount int, pub sig.Pub) error {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	arm.checkLocal(msgID, pid) // check if this is a local message

	// the nodes random bytes converted to a uint64
	arm.rndLock.RLock()
	item, ok := arm.vrfRandByMsgID[pubMsgID{pub: pid, msgID: msgID}]
	arm.rndLock.RUnlock()
	if !ok {
		return types.ErrNotReceivedVRFProof
	}
	// the threshold for the given number of nodes
	// thrsh := uint64((float64(participantNodeCount)/float64(totalNodeCount))*float64(math.MaxUint64))
	onePc := uint64(math.MaxUint64) / 100
	percentage := uint64(utils.Min((participantNodeCount*100)/totalNodeCount+config.DefaultNodeRelaxation, 100))
	thrsh := onePc * percentage
	if item.rnd <= thrsh {
		return nil
	}
	return types.ErrNotMember
}

// checkRandCoord uses the VRFs to determine if pub is a valid coordinator for this consensus.
// Note due to the random function, there can be 0 or multiple valid coordinators.
// participantNodeCount is the number of participants expected for this node.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandMemberCheckerByID) checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID,
	round types.ConsensusRound, pub sig.Pub) (rndValue uint64, coord sig.Pub, err error) {

	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	arm.checkLocal(msgID, pid) // check if this is a local message

	arm.rndLock.RLock()
	item, ok := arm.vrfRandByMsgID[pubMsgID{pub: pid, msgID: msgID}]
	arm.rndLock.RUnlock()
	if !ok {
		panic(1)
		return 0, nil, types.ErrNotReceivedVRFProof
	}

	// the threshold for the given number of nodes
	onePc := uint64(math.MaxUint64) / 100
	thrsh := uint64(float64(onePc) * float64(arm.coordinatorRelaxation) * (float64(100) / float64(totalNodeCount))) //arm.coordinatorRelaxation

	// thrsh := uint64((float64(1)/float64(totalNodeCount))*math.MaxUint64)
	if item.rnd <= thrsh {
		return item.rnd, pub, nil
	}
	return 0, nil, types.ErrNotMember
}

// GotVrf should be called when a node's VRF proof is received for this consensus instance.
func (arm *absRandMemberCheckerByID) GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error {
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

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

func (arm *absRandMemberCheckerByID) computeNewMsg(msgID messages.MsgID) sig.BasicSignedMessage {
	// The message we use to compute the VRF is the MsgID (containing the index) plus the random bytes
	m := messages.NewMsgBuffer()
	m.AddBytes(msgID.ToBytes(arm.myIdx))
	m.AddBytes(arm.randBasicMsg)
	return m.GetRemainingBytes()
}
