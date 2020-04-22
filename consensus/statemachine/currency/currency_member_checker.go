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
package currency

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"sort"
)

func InitCurrencyMemberChecker(localRand *rand.Rand, rotateCord bool, rndMemberCount int,
	rndMemberType types.RndMemberType, localRandChangeFrequency types.ConsensusInt, priv sig.Priv,
	gc *generalconfig.GeneralConfig) consinterface.MemberChecker {

	ret := &CurrencyMemberChecker{CustomMemberChecker: *memberchecker.InitCustomMemberChecker(localRand,
		rotateCord, rndMemberCount, rndMemberType, localRandChangeFrequency, priv, gc)}

	return ret
}

// AllowsChange returns true if the member checker allows changing members, used as a sanity check.
func (mc *CurrencyMemberChecker) AllowsChange() bool {
	return true
}

// CurrencyMemberChecker is to be used with SimpleCurrencyTxProposer, it chooses the top memberCount
// members based on who owns the most currency.
type CurrencyMemberChecker struct {
	memberchecker.CustomMemberChecker
}

// New generates a new member checker for the index, this is called on the inital member checker each time.
func (mc *CurrencyMemberChecker) New(newIndex types.ConsensusIndex) consinterface.MemberChecker {
	ret := &CurrencyMemberChecker{CustomMemberChecker: *mc.CustomMemberChecker.New(newIndex)}
	return ret
}

// UpdateState chooses the pub keys with the most currency as the next members (keeping the same number of members
// as the previous member checker)
func (mc *CurrencyMemberChecker) UpdateState(fixedCoord sig.Pub, prevDec []byte, randBytes [32]byte,
	prevMember consinterface.MemberChecker, prevSM consinterface.GeneralStateMachineInterface) (newMemberPubs,
	newOtherPubs []sig.Pub) {

	if !prevSM.GetDecided() {
		panic("should have updated the SM first")
	}

	prvCurSM := prevSM.(*SimpleCurrencyTxProposer)

	// Get the balances of all the accounts
	allPubCurrency := make([]struct {
		uint64
		sig.Pub
	}, 0, len(prevMember.GetAllPubs()))
	prvCurSM.accTable.Range(func(pub sig.PubKeyStr, acc Account) bool {
		allPubCurrency = append(allPubCurrency, struct {
			uint64
			sig.Pub
		}{acc.Balance, acc.Pub})
		return true
	})

	// Sort them by their value
	sort.Slice(allPubCurrency, func(i, j int) bool {
		// We use > here because we want the largest value at the beginning of the list
		if allPubCurrency[i].uint64 == allPubCurrency[j].uint64 {
			var p1, p2 sig.PubKeyStr
			var err error
			if p1, err = allPubCurrency[i].GetPubString(); err != nil {
				panic(err)
			}
			if p2, err = allPubCurrency[j].GetPubString(); err != nil {
				panic(err)
			}
			return p1 > p2
		}
		return allPubCurrency[i].uint64 > allPubCurrency[j].uint64
	})

	// Copy this to the new List
	newAllPubs := make(sig.PubList, len(allPubCurrency))
	for i, nxt := range allPubCurrency {
		newAllPubs[i] = nxt.Pub
	}
	memberCount := prevMember.GetMemberCount()
	// Members are the ones with the most value
	newMemberPubs = newAllPubs[:memberCount]
	// Others are the ones with less value
	newOtherPubs = newAllPubs[memberCount:]

	mc.CustomMemberChecker.UpdateState(fixedCoord, prevDec, randBytes,
		&prevMember.(*CurrencyMemberChecker).CustomMemberChecker, newMemberPubs, newOtherPubs)

	return
}

// AddPubKeys adds the pub keys as members to the consensus.
// Note that for the first consensus we use the pubs directly as given.
// In following rounds we take the top len(memberPubKeys) with the most currency as the members.
func (mc *CurrencyMemberChecker) AddPubKeys(fixedCoord sig.Pub, memberPubKeys, otherPubs sig.PubList, initRandBytes [32]byte) {
	mc.CustomMemberChecker.AddPubKeys(fixedCoord, memberPubKeys, otherPubs, initRandBytes)
}
