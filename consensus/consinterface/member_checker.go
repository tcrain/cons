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

package consinterface

import (
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
)

// MemCheckers holds the member checker and special member checker for a consensus instance.
type MemCheckers struct {
	MC  MemberChecker           // the member checker
	SMC SpecialPubMemberChecker // the special member checker
}

// A MemberChecker is created for each consensus instance.
// It is responsible for tracking who can participate in consensus by having a list of their public keys.
// Once consensus is completed for the previous index, this member check receives the decision through UpdateState to see
// if its members should change, this depends on the implementation.
// Once UpdateState is called, IsReady should always return true.
// IsReady may want to return true before this, if for example the consensus uses a fixed set of members, this will
// allow received messages to be processed earlier.
// MemberChecker needs to be concurrent safe for the IsReady method.
// CheckMemberBytes will be called by different threads after IsReady is true,
// if they are not read only then they will need some sync.
// If CheckMemberBytes return nil (a member is not found), then SpecialPubMemberChecker.CheckMemberLocalMsg
// will be called.
// An initial member checker is input to ConsState. Each member checker will then be created by calling new on this
// initial member checker. The member checker will not know about state changes since the initial state until
// UpdateState is called on it.
type MemberChecker interface {
	New(id types.ConsensusIndex) MemberChecker // New generates a new member checker for the index, this is called on the inital member checker each time.
	IsReady() bool                             // IsReady should return true if the member check knows all the members. It needs to be concurrent safe.

	AllowsChange() bool // AllowsChange returns true if the member checker allows changing members, used as a sanity check.

	// CheckMemberBytes checks the the pub key is a member and returns corresponding pub key object, nil if it is not a member
	CheckMemberBytes(types.ConsensusIndex, sig.PubKeyID) sig.Pub
	// CheckMemberBytes checks the the bytes using GetPubString and returns corresponding pub key object, nil if it is not a member
	// CheckPubString(types.ConsensusIndex, sig.PubKeyID) sig.Pub
	// CheckRandMember can be called after CheckMemberBytes is successful when ChooseRandomMember is enabled.
	// MsgID is the id of the message being checked.
	// IsProposal is true if msgID corresponds to a proposal message for the current consensus type.
	CheckRandMember(pub sig.Pub, msgID messages.MsgID, isProposalMsg bool) error
	// SelectRandMembers returns true if the member checker is selecting random members.
	RandMemberType() types.RndMemberType

	// UpdateState is called when the previous consensus instance decides, the input is the decision plus the previous member checker.
	// Calling this should not change what IsReady returns, only after FinishUpdateState should IsReady always return true.
	// The reason is that after UpdateState is called, and before FinishUpdateState is called, SpecialMemberChecker.Update state is called.
	// Update state returns two lists of pubs.
	// fixedCoord is non-nil if we have a fixed coordinator for this instance.
	// newAllPubs are the list of all pubs in the system.
	// newMemberPubs pubs are the public keys that will now participate in consensus.
	// newMemberPubs must be a equal to or a subset of newAllPubs.
	// If membership is not changed since the previous consensus instance then nil is returned for these items.
	// Proof is the vrf proof of the current node if random membership is enabled.
	// The return values are nil if the membership has not changed.
	UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
		prevMember MemberChecker, prevSM GeneralStateMachineInterface) (newMemberPubs, newAllPubs []sig.Pub)
	// DoneNextUpdateState is called when the member checker will no longer be used as input to generate a new member checker
	// as the prevMemberChecker to update state.
	DoneNextUpdateState() error

	// GotVrf should be called when a node's VRF proof is received for this consensus instance.
	// If the VRF was valid it returns the uint64 that represents the vrf // TODO is it sufficient to just use the first 8 bytes of the rand?
	GotVrf(pub sig.Pub, msgID messages.MsgID, proof sig.VRFProof) error
	// GetMyVRF returns the vrf proof for the local node.
	GetMyVRF(id messages.MsgID) sig.VRFProof

	// GetNewPub returns an empty public key object.
	GetNewPub() sig.Pub
	GetIndex() types.ConsensusIndex
	FinishUpdateState()                         // FinishUpdate state is called after UpdateState. After this call, IsReady should always return true.
	CheckIndex(index types.ConsensusIndex) bool // CheckIndex should return true if the index is the same as the one used in New, it is for testing.
	GetMemberCount() int                        // GetMemberCount returns the number of consensus members.
	GetFaultCount() int                         // GetFaultCount returns the number of possilbe faultly nodes that the consensus can handle.
	GetParticipants() sig.PubList               // GetParticipant return the pub key of the participants.
	GetAllPubs() sig.PubList                    // GetAllPub returns all pubs in the system
	GetMyPriv() sig.Priv                        // GetMyPriv returns the local nodes private key
	// CheckRoundCoord checks if checkPub the/a coordinator for round. If it is, err is returned as nil, otherwise an
	// error is returned. If checkPub is nil, then it will return the known coordinator in coordPub.
	// Note this should only be called after the pub is verified to be a member with CheckMemberBytes
	CheckRoundCoord(msgID messages.MsgID, checkPub sig.Pub, round types.ConsensusRound) (coordPub sig.Pub, err error)
	// CheckEstimatedRoundCoordNextIndex returns the estimated public key of the coordinator for round round for the following consensus index.
	// It might not return the correct coordinator for the next index because the decision might change the membership
	// for the next index. This function should not be used if it needs to know the next coordinator for certain.
	CheckEstimatedRoundCoordNextIndex(checkPub sig.Pub,
		round types.ConsensusRound) (coordPub sig.Pub, err error)
	// CheckRandRoundCoord should be called instead of CheckRoundCoord if random membership selection is enabled.
	// If using VRFs then checkPub must not be nil.
	// If checkPub is nil, then it will return the known coordinator in coordPub.
	// If VRF is enabled randValue is the VRF random value for the inputs.
	// Note this should be called after CheckRandMember for the same pub.
	CheckRandRoundCoord(msgID messages.MsgID, checkPub sig.Pub, round types.ConsensusRound) (randValue uint64,
		coordPub sig.Pub, err error)

	// CheckFixedCoord checks if checkPub the/a coordinator for round. If it is, err is returned as nil, otherwise an
	// error is returned. If checkPub is nil, then it will return the fixed coordinator in coordPub.
	// If there is no fixedCoordinator, then an error types.ErrNoFixedCoord is returned.
	// Note this should only be called after the pub is verified to be a member with CheckMemberBytes
	CheckFixedCoord(pub sig.Pub) (coordPub sig.Pub, err error)
	GetStats() stats.StatsInterface

	// SetStats(stats.StatsInterface) // SetStats sets the stats object that will be used to keep stats, TODO clean this up
	Validated(types.SignType) // ValidatedItem is called for stats to let it know a signature was validated. TODO cleanup.

	// AddPubKeys is used for adding the public keys to the very inital member checker, and should not
	// be called on later member checkers, as they should change pub keys through the call to UpdateState.
	// priv is the local nodes private key.
	// fixedCoord is non-nil if we have a fixed coordinator for this instance.
	// allPubs are the list of all pubs in the system.
	// memberPubs pubs are the public keys that will now participate in consensus.
	// memberPubs must be equal to or a subset of newAllPubs.
	AddPubKeys(fixedCoord sig.Pub, memberPubKeys, allPubKeys sig.PubList, initRandBytes [32]byte)

	// SetMainChannel is called on the initial member checker to inform it of the network channel object
	SetMainChannel(mainChannel channelinterface.MainChannel)
}

// SpecialPubMemberChecker is used for keys that do not follow the rules of 1 key = 1 participant.
// For example multisignatures or threshold signatures.
// It should keep track of these keys, CheckMemberLocalMsg will only be called once MemberChecker.IsReady returns true.
// CheckMemberLocalMsg is called after MemberChecker.CheckMemberBytes in case a normal member is not found.
type SpecialPubMemberChecker interface {
	New(id types.ConsensusIndex) SpecialPubMemberChecker                                     // New generates a new member checker for the index, this is called on the inital special member checker each time.
	UpdateState(newPubs sig.PubList, randBytes [32]byte, prevMember SpecialPubMemberChecker) // UpdateState is called after MemberCheck.Update state. New pubs should be nil if they didn't change from the previous iteration.
	// CheckMemberLocalMsg checks if pub is a special member and returns new pub that has all the values filled.
	// This is called after MemberChecker.CheckMemberBytes in case a normal member is not found.
	// This sould be safe to be called concurrently.
	// TODO use a special pub type?
	// TODO this should be called with just they bytes like MemberChecker.CheckMemberBytes
	CheckMember(types.ConsensusIndex, sig.Pub) (sig.Pub, error)
}

// CheckCoord checks if pub is a coordinator, for the round and MsgID, returning an error if it is not.
// If pub is nil, then it returns the coordinator pub for the inputs.
// It will fail if pub is nil and random VRF membership is set.
// randValue is the randomValue from the VRF for the inputs.
// It should be used by the consensus implementations.
func CheckCoord(pub sig.Pub, mc *MemCheckers, rnd types.ConsensusRound,
	msgID messages.MsgID) (randValue uint64, coordPub sig.Pub, err error) {

	coordPub, err = mc.MC.CheckFixedCoord(pub)
	switch err {
	case types.ErrNoFixedCoord:
		// continue
	default:
		return
	}

	if mc.MC.RandMemberType() != types.NonRandom { // if we are using random
		return mc.MC.CheckRandRoundCoord(msgID, pub, rnd) // Check if we are the random coord
	} else { // Normal path
		coordPub, err = mc.MC.CheckRoundCoord(msgID, pub, rnd)
		return
	}
}

// CheckMemberCoordHdr checks if the message comes from the coordinator, returning an error if not.
// It sets the random VRF for this sigItem if enabled.
// It is called within the message state checks.
func CheckMemberCoordHdr(mc *MemCheckers, rnd types.ConsensusRound, msg messages.MsgHeader) error {
	switch v := msg.(type) {
	case *sig.MultipleSignedMessage:
		sigItems := v.GetSigItems()
		if len(sigItems) != 1 {
			return types.ErrInvalidSig
		}
		return CheckMemberCoord(mc, rnd, sigItems[0], v)
	case *sig.UnsignedMessage:
		pubs := v.GetEncryptPubs()
		if len(pubs) != 1 {
			return types.ErrInvalidSig
		}
		_, _, err := CheckCoord(pubs[0], mc, rnd, msg.GetMsgID())
		if err != nil {
			logging.Errorf("Got proposal from invalid cord at index %v", mc.MC.GetIndex())
			return err
		}
		return nil
	}
	return types.ErrInvalidHeader
}

// CheckMemberCoord checks if the sigItem is the coordinator returning an error if it is not.
// It sets the random VRF for this sigItem if enabled.
// It is called within the message state checks.
func CheckMemberCoord(mc *MemCheckers, rnd types.ConsensusRound,
	sigItem *sig.SigItem, msg *sig.MultipleSignedMessage) error {

	randVal, _, err := CheckCoord(sigItem.Pub, mc, rnd, msg.GetMsgID())
	if err != nil {
		logging.Errorf("Got proposal from invalid cord at index %v", mc.MC.GetIndex())
		return err
	}
	sigItem.VRFID = randVal
	return nil
}

// CheckMemberLocal returns true if the local node is a member of the consensus.
func CheckMemberLocal(mc *MemCheckers) bool {
	// Check if we are the fixed coord
	myPub := mc.MC.GetMyPriv().GetPub()
	if _, err := mc.MC.CheckFixedCoord(myPub); err == nil {
		return true
	}
	// Check if we are a normal member
	str, err := myPub.GetPubID()
	if err != nil {
		panic("bad local pub")
	}

	return mc.MC.CheckMemberBytes(mc.MC.GetIndex(), str) != nil
}

// CheckRandMember checks if the public key is a member for the given consensus instance and
// message type. It should only be called after CheckMember.
// Note that CheckMember already calls this function internally to if random membership is supported.
// Random membership means that out of the known members only a certain set will be chosen randomly
// for this specific consensus/message pair.
func CheckRandMember(mc *MemCheckers, pub sig.Pub, isProposalMsg bool, msgID messages.MsgID) error {
	// Check if we are a member based on random selection
	if err := mc.MC.CheckRandMember(pub, msgID, isProposalMsg); err != nil {
		// If we are not a random member we still may be the fixed coord, so check
		switch _, fixErr := mc.MC.CheckFixedCoord(pub); fixErr {
		case types.ErrNoFixedCoord: // no fixed coord
			return err
		case nil: // We are the fixed coord
		default: // neither the fixed coord or a random member
			return err
		}
	}
	// we are a member
	return nil
}

// CheckMemberLocalMsg checks if the pub is a member for the index, and validates the signature with the pub,
// and returns a new pub object that has all the values filled.
func CheckMember(mc *MemCheckers, idx types.ConsensusIndex, sigItem *sig.SigItem, msg *sig.MultipleSignedMessage) error {
	if !mc.MC.CheckIndex(idx) {
		panic(fmt.Sprintf("wrong member checker index %v", idx))
	}
	msgID := msg.GetMsgID()
	prePub := sigItem.Pub
	str, err := prePub.GetPubID()
	if err != nil {
		return err
	}

	sigItem.Pub = mc.MC.CheckMemberBytes(idx, str)
	if sigItem.Pub == nil { // We are not a normal member, check for special member
		sigItem.Pub, err = mc.SMC.CheckMember(idx, prePub)
		if err != nil { // Not found as a normal or special member
			sigItem.Pub = prePub
			return types.ErrNotMember
		}
	} else { // We are a normal member, check random member if needed
		if sigItem.VRFProof != nil { // Add the VRF proof  to the random member checker
			err = mc.MC.GotVrf(sigItem.Pub, msgID, sigItem.VRFProof)
			if err != nil {
				return err
			}
		}
		_, valMsg, err := msg.GetBaseMsgHeader().NeedsSMValidation(msg.Index, 0)
		if err != nil {
			panic(err) // TODO panic here or return err?
			return err
		}
		// Check if we are a member based on random selection
		if err := CheckRandMember(mc, sigItem.Pub, valMsg != nil, msgID); err != nil {
			return err
		}
	}
	mc.MC.Validated(msg.GetSignType())
	return sigItem.Pub.CheckSignature(msg, sigItem)
}
