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
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
)

type absRandMemberInterface interface {
	// gotRand will be called when the consensus index has decided. If random bytes are enabled they will be included.
	// participantNodeCount is the number of pubs that should be selected from sortedMemberPubs to be chosen as members for
	// the consensus index.
	// If sortedMemberPubs is nil they they have not changed since the last consensus index.
	gotRand(rnd [32]byte, participantNodeCount int, myPriv sig.Priv, sortedMemberPubs sig.PubList,
		memberMap map[sig.PubKeyID]sig.Pub, prvMC absRandMemberInterface)
	// checkRandMember should return nil if pub should participate in this consensus given the inputs, or an error otherwise.
	checkRandMember(msgID messages.MsgID, isLocal, isProposalMsg bool, participantNodeCount, totalNodeCount int, pub sig.Pub) error
	// checkRandCoord checks if pub should participate in this consensus as a coordinator given the inputs.
	// If pub is nil then the coordinator pub should be returned as coord.
	// randVal is the random value given to the pub as coordinator, the lower the value the more it is supported as coordinator.
	// Otherwise an error is returned if pub is not a coordinator.
	checkRandCoord(participantNodeCount, totalNodeCount int, msgID messages.MsgID, round types.ConsensusRound,
		pub sig.Pub) (rndVal uint64, coord sig.Pub, err error)
	// GotVrf should be called when a VRFProof is received for the pub and message type.
	GotVrf(pub sig.Pub, isProposal bool, msgID messages.MsgID, proof sig.VRFProof) error
	// getMyVRF returns the VRFProof for the local node and message type for this consensus.
	getMyVRF(isProposal bool, id messages.MsgID) sig.VRFProof
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
