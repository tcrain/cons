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

package mvcons4

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type Config struct {
	cons.StandardMvConfig
}

// GetSigTypes return the types of signatures supported by the consensus
func (Config) GetSigTypes(gt cons.GetOptionType) []types.SigType {
	switch gt {
	case cons.AllOptions:
		return types.BasicSigTypes
	case cons.MinOptions:
		return []types.SigType{types.EC}
	default:
		panic(gt)
	}
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
func (Config) GetOrderingTypes(cons.GetOptionType) []types.OrderingType {
	return []types.OrderingType{types.Total}
}

// GetMemberCheckerTypes returns the types of member checkers valid for the consensus.
func (Config) GetMemberCheckerTypes(cons.GetOptionType) []types.MemberCheckerType {
	return []types.MemberCheckerType{types.TrueMC}
}

// GetAllowConcurrentTypes returns the values for if the consensus supports running concurrent consensus instances
// when using total ordering or not or both.
// MvCons4 must run instances concurrently.
func (Config) GetAllowConcurrentTypes(cons.GetOptionType) []bool {
	return types.WithTrue
}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (Config) GetCollectBroadcast(cons.GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full}
}

// RequiresStaticMembership returns true if this consensus doesn't allow changing membership.
func (Config) RequiresStaticMembership() bool {
	return false
}

// GetUsePubIndex returns true.
func (Config) GetUsePubIndexTypes(cons.GetOptionType) []bool {
	return types.WithTrue
}

// GetIncludeProofTypes returns false.
func (Config) GetIncludeProofsTypes(cons.GetOptionType) []bool {
	return types.WithBothBool
}

// GetRandMemberCheckerTypes returns the non-random type.
func (Config) GetRandMemberCheckerTypes(cons.GetOptionType) []types.RndMemberType {
	return []types.RndMemberType{types.NonRandom}
}

// GetRotateCoordTypes returns false.
func (Config) GetRotateCoordTypes(cons.GetOptionType) []bool {
	return types.WithFalse
}
