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

package mvcons3

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type MvCons3Config struct {
	cons.StandardMvConfig
}

// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
func (MvCons3Config) RequiresStaticMembership() bool {
	return true
}

// GetSigTypes return the types of signatures supported by the consensus
func (MvCons3Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
	switch gt {
	case cons.AllOptions:
		return types.AllSigTypes
	case cons.MinOptions:
		return []types.SigType{types.TBLS}
	default:
		panic(gt)
	}
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
func (MvCons3Config) GetOrderingTypes(gt cons.GetOptionType) []types.OrderingType {
	return []types.OrderingType{types.Total}
}

// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
func (MvCons3Config) GetMemberCheckerTypes(gt cons.GetOptionType) []types.MemberCheckerType {
	return []types.MemberCheckerType{types.TrueMC}
}

// GetRandMemberCheckerTypes returns the types of random member checkers supported by the consensus
func (MvCons3Config) GetRandMemberCheckerTypes(gt cons.GetOptionType) []types.RndMemberType {
	return []types.RndMemberType{types.NonRandom}

}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
func (MvCons3Config) GetRotateCoordTypes(gt cons.GetOptionType) []bool {
	return types.WithFalse
}

// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
// when using total ordering or not or both.
func (MvCons3Config) GetAllowConcurrentTypes(gt cons.GetOptionType) []bool {
	return types.WithFalse
}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (MvCons3Config) GetCollectBroadcast(cons.GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full, types.Commit}
}
