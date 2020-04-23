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

package rbbcast1

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/types"
)

type RbBcast1Config struct {
	cons.StandardMvConfig
}

// GetBroadcastFunc returns the broadcast function for the given byzantine type
func (rb RbBcast1Config) GetBroadcastFunc(bt types.ByzType) consinterface.ByzBroadcastFunc {
	switch bt {
	case types.Mute: // We have to broadcast or will not terminate
		return cons.BroadcastMuteExceptInit
	case types.HalfHalfFixedBin, types.HalfHalfNormal:
		panic("halfhalf byz type not supported")
	default:
		return rb.StandardMvConfig.GetBroadcastFunc(bt)
	}
}

// GetOrderingTypes returns the types of ordering supported by the consensus.
//func (RbBcast1Config) GetOrderingTypes(gt types.GetOptionType) []types.OrderingType {
//	return []types.OrderingType{types.Causal}
//}
// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (RbBcast1Config) GetIncludeProofsTypes(gt cons.GetOptionType) []bool {
	return types.WithFalse
}

// GetUseMultiSigTypes() []bool
// GetRotateCoordTypes returns the values for if the consensus supports rotating coordinator or not or both.
//func (RbBcast1Config) GetRotateCoordTypes(gt cons.GetOptionType) []bool {
//	return types.WithFalse
//}

// GetCollectBroadcast returns the values for if the consensus supports broadcasting the commit message
// directly to the leader.
func (RbBcast1Config) GetCollectBroadcast(cons.GetOptionType) []types.CollectBroadcastType {
	return []types.CollectBroadcastType{types.Full, types.Commit}
}

// GetByzTypes returns the fault types to test.
func (RbBcast1Config) GetByzTypes(optionType cons.GetOptionType) []types.ByzType {
	return []types.ByzType{types.NonFaulty, types.Mute}
}
