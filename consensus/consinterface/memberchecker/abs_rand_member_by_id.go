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
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math"
)

type pubMsgID struct {
	pub   sig.PubKeyID
	msgID messages.MsgID
}

type prfRnd struct {
	prf sig.VRFProof
	rnd uint64
}

// absRandMemberCheckerByID uses VRF to determine members for the consensus, the members are chosen
// for each message type.
type absRandMemberCheckerByID struct {
	*absRandCoordByID
}

func initAbsRandMemberCheckerByID(priv sig.Priv, stats stats.StatsInterface,
	gc *generalconfig.GeneralConfig) *absRandMemberCheckerByID {

	return &absRandMemberCheckerByID{
		absRandCoordByID: initAbsRandCoordByID(priv, stats, gc),
	}
}

func (arm *absRandMemberCheckerByID) newRndMC(idx types.ConsensusIndex, stats stats.StatsInterface) absRandMemberInterface {
	return &absRandMemberCheckerByID{
		absRandCoordByID: arm.absRandCoordByID.newRndMC(idx, stats).(*absRandCoordByID),
	}
}

// getMyVRF returns the vrf proof for the local node.
func (arm *absRandMemberCheckerByID) getMyVRF(isProposal bool, msgID messages.MsgID) sig.VRFProof {
	_ = isProposal
	return arm.getMyVRFInternal(msgID)
}

// checkLocal is called to check if we are checking for local membership for the MsgID then we have computed the local VRF
func (arm *absRandMemberCheckerByID) checkLocal(isProposal bool, msgID messages.MsgID, pid sig.PubKeyID) {
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

// checkRandMember uses the VRFs to determine if pub can participate in this consensus.
// participantNodeCount is the number of participants expected for this consensus.
// totalNodeCount is the total number of nodes in the system.
// It returns nil if the node can participate for this message, otherwise an error.
func (arm *absRandMemberCheckerByID) checkRandMember(msgID messages.MsgID, isLocal, isProposalMsg bool,
	participantNodeCount, totalNodeCount int, pub sig.Pub) error {

	_ = isLocal
	if arm.randBasicMsg == nil {
		panic("should not call this until after gotRand has been called")
	}

	pid, err := pub.GetPubID()
	if err != nil {
		panic(err)
	}
	arm.checkLocal(isProposalMsg, msgID, pid) // check if this is a local message

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
	percentage := uint64(utils.Min((participantNodeCount*100)/totalNodeCount+arm.gc.NodeChoiceVRFRelaxation, 100))
	thrsh := onePc * percentage
	if item.rnd <= thrsh {
		return nil
	}
	return types.ErrNotMember
}

// GotVrf should be called when a node's VRF proof is received for this consensus instance.
func (arm *absRandMemberCheckerByID) GotVrf(pub sig.Pub, isProposal bool, msgID messages.MsgID, proof sig.VRFProof) error {

	_ = isProposal
	return arm.gotVrfInternal(pub, msgID, proof)
}
