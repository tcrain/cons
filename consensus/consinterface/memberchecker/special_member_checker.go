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
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/consensus/auth/sig"
)

/////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////

// NoSpecialMembers implements SpecialPubMemberChecker, but contains no special public keys.
type NoSpecialMembers struct {
	// absMemberChecker
}

// NewNoSpecialMembers generates an empty NoSpecialMembers object.
func NewNoSpecialMembers() *NoSpecialMembers {
	return &NoSpecialMembers{}
}

// New generates a new member checker for the index, this is called on the inital special member checker each time.
func (tsm *NoSpecialMembers) New(idx types.ConsensusIndex) consinterface.SpecialPubMemberChecker {
	newMC := &NoSpecialMembers{}
	// newMC.absMemberChecker = *tsm.absMemberChecker.newAbsMc(idx)
	return newMC
}

// UpdateState is called after MemberCheck.Update state.
// Here it does nothing.
func (tsm *NoSpecialMembers) UpdateState(newKeys sig.PubList, randBytes [32]byte,
	prevMember consinterface.SpecialPubMemberChecker) {
}

// CheckMemberLocalMsg checks if pub is a special member, here is always returns an error since there are no speical pubs.
func (tsm *NoSpecialMembers) CheckMember(idx types.ConsensusIndex, pub sig.Pub) (sig.Pub, error) {
	return nil, types.ErrNotMember
}

/////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////

// NoSpecialMembers implements SpecialPubMemberChecker, it allows for multi-signature public keys
// using BLS signatures. Keys will be generated using the BitID of the merged public key being requested
// in CheckMemberLocalMsg by merging the normal public key, using their indecies.
type MultiSigMemChecker struct {
	// absMemberChecker
	allowsChange  bool
	idx           types.ConsensusIndex
	originPubList sig.PubList // the normal public keys, we use these to generate the merged public keys
}

// NewMultSigMemChecker generates a new MultiSigMemChecker object.
func NewMultiSigMemChecker(pubs sig.PubList, allowsChange bool) *MultiSigMemChecker {
	if !sig.GetUsePubIndex() {
		panic("should only be used with pub indecies")
	}
	return &MultiSigMemChecker{originPubList: pubs, allowsChange: allowsChange}
}

// New generates a new member checker for the index, this is called on the inital special member checker each time.
func (msm *MultiSigMemChecker) New(idx types.ConsensusIndex) consinterface.SpecialPubMemberChecker {
	newMC := &MultiSigMemChecker{idx: idx, originPubList: msm.originPubList, allowsChange: msm.allowsChange}
	// newMC.absMemberChecker = *msm.absMemberChecker.newAbsMc(idx)
	return newMC
}

// UpdateState is called after MemberCheck.Update state.
// If there are new keys, they will be used to generate the multi-signature public keys.
func (msm *MultiSigMemChecker) UpdateState(newKeys sig.PubList, randBytes [32]byte,
	prevMember consinterface.SpecialPubMemberChecker) {

	if !msm.allowsChange {
		if newKeys != nil {
			panic("should not have change membership")
		}
		return
	}

	// If we have new keys we no longer use our generated multisigs
	if newKeys != nil {
		// TODO keep the multisigs that are still valid for the new keys
		msm.originPubList = newKeys
	} else {
		msm.originPubList = prevMember.(*MultiSigMemChecker).originPubList
	}
}

// CheckMemberLocalMsg checks if pub is a special member and returns new pub that has all the values filled.
// If pub has a valid BitID then CheckMemberLocalMsg will always return a merged public key by merging the
// indices from teh BitID using the normal public keys.
func (msm *MultiSigMemChecker) CheckMember(idx types.ConsensusIndex, pub sig.Pub) (sig.Pub, error) {
	if idx.Index != msm.idx.Index {
		panic(fmt.Sprintf("wrong member checker index %v, %v", idx, msm.idx))
	}

	// TODO keep a cache of constructed pubs, so we don't have to create the full new ones each time
	// We have to construct the new pub
	blsPub := pub.(sig.MultiPub)
	bid := blsPub.GetBitID()
	iter := bid.NewIterator()
	i, err := iter.NextID()
	if i < 0 || i >= len(msm.originPubList) || err != nil {
		// The bitid contains an invalid index
		return nil, types.ErrInvalidBitID
	}
	mrgPub := msm.originPubList[i].(sig.MultiPub).Clone()
	// Go through the pub keys and merge them one by one into the new key
	for i, err := iter.NextID(); err == nil; i, err = iter.NextID() {
		if i < 0 || i >= len(msm.originPubList) {
			// The bitid contains an invalid index
			return nil, types.ErrInvalidBitID
		}
		mrgPub.MergePubPartial(msm.originPubList[i].(sig.MultiPub))
	}
	iter.Done()
	// finish the merge
	mrgPub.DonePartialMerge(bid)
	return mrgPub.(sig.Pub), nil
}

/////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////

// ThrshSigMemChecker implements SpecialPubMemberChecker, it allows for using a threshold public key.
// It does not allow the public keys to change over consensus instances as that would require computing
// a new threshold key, something that is not currently supported.
type ThrshSigMemChecker struct {
	// absMemberChecker
	idx  types.ConsensusIndex
	pubs []sig.Pub // the threshold public keys
}

// NewThrshSigMemberChecker generates a new ThrshSigMemChecker object
func NewThrshSigMemChecker(pubs []sig.Pub) *ThrshSigMemChecker {
	return &ThrshSigMemChecker{pubs: pubs}
}

// New generates a new member checker for the index, this is called on the inital special member checker each time.
func (tsm *ThrshSigMemChecker) New(idx types.ConsensusIndex) consinterface.SpecialPubMemberChecker {
	newMC := &ThrshSigMemChecker{idx: idx, pubs: tsm.pubs}
	// newMC.absMemberChecker = *tsm.absMemberChecker.newAbsMc(idx)
	return newMC
}

// UpdateState is called after MemberCheck.Update state.
// It panics if newKeys != nil, as changing keys is not supported.
func (tsm *ThrshSigMemChecker) UpdateState(newKeys sig.PubList, randBytes [32]byte,
	prevMember consinterface.SpecialPubMemberChecker) {

	if newKeys != nil {
		panic("thrsh sig member change not supported")
	}
}

// CheckMemberLocalMsg checks if pub is the threshold key and returns new pub that has all the values filled.
func (tsm *ThrshSigMemChecker) CheckMember(idx types.ConsensusIndex, pub sig.Pub) (sig.Pub, error) {
	if idx.Index != tsm.idx.Index {
		panic(fmt.Sprintf("wrong member checker index %v, %v", idx, tsm.idx))
	}
	str, err := pub.GetPubID()
	if err != nil {
		return nil, err
	}
	for _, tsmPub := range tsm.pubs {
		myPubID, err := tsmPub.GetPubID()
		if err != nil {
			panic(err)
		}
		if str == myPubID {
			return tsmPub, nil
		}
	}
	return nil, types.ErrNotMember
}
