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
package cons

import (
	"fmt"
	"github.com/tcrain/cons/consensus/types"
)

type GetOptionType int

const (
	AllOptions GetOptionType = iota
	MinOptions
)

// OptionStruct represents a set of options represented by the functions of ConfigOptions.
type OptionStruct struct {
	//IsMv bool
	//RequiresStaticMembership bool

	StopOnCommitTypes      []types.StopOnCommitType
	OrderingTypes          []types.OrderingType
	ByzTypes               []types.ByzType
	StateMachineTypes      []types.StateMachineType
	SigTypes               []types.SigType
	UsePubIndexTypes       []bool
	IncludeProofsTypes     []bool
	MemberCheckerTypes     []types.MemberCheckerType
	RandMemberCheckerTypes []types.RndMemberType
	RotateCoordTypes       []bool
	AllowSupportCoinTypes  []bool
	AllowConcurrentTypes   []types.ConsensusInt
	CollectBroadcast       []types.CollectBroadcastType
	EncryptChannelsTypes   []bool
	AllowNoSignaturesTypes []bool
	CoinTypes              []types.CoinType
	UseFixedCoinPresets    []bool
}

// ReplaceNilFields replaces the nil fields of os with those of other.
func ReplaceNilFields(os, other OptionStruct) OptionStruct {
	if len(os.ByzTypes) == 0 {
		os.ByzTypes = other.ByzTypes
	}
	if len(os.AllowNoSignaturesTypes) == 0 {
		os.AllowNoSignaturesTypes = other.AllowNoSignaturesTypes
	}
	if len(os.StopOnCommitTypes) == 0 {
		os.StopOnCommitTypes = other.StopOnCommitTypes
	}
	if len(os.UseFixedCoinPresets) == 0 {
		os.UseFixedCoinPresets = other.UseFixedCoinPresets
	}
	if len(os.UsePubIndexTypes) == 0 {
		os.UsePubIndexTypes = other.UsePubIndexTypes
	}
	if len(os.StateMachineTypes) == 0 {
		os.StateMachineTypes = other.StateMachineTypes
	}
	if len(os.MemberCheckerTypes) == 0 {
		os.MemberCheckerTypes = other.MemberCheckerTypes
	}
	if len(os.IncludeProofsTypes) == 0 {
		os.IncludeProofsTypes = other.IncludeProofsTypes
	}
	if len(os.AllowSupportCoinTypes) == 0 {
		os.AllowSupportCoinTypes = other.AllowSupportCoinTypes
	}
	if len(os.AllowConcurrentTypes) == 0 {
		os.AllowConcurrentTypes = other.AllowConcurrentTypes
	}
	if len(os.RandMemberCheckerTypes) == 0 {
		os.RandMemberCheckerTypes = other.RandMemberCheckerTypes
	}
	if len(os.RotateCoordTypes) == 0 {
		os.RotateCoordTypes = other.RotateCoordTypes
	}
	if len(os.SigTypes) == 0 {
		os.SigTypes = other.SigTypes
	}
	if len(os.OrderingTypes) == 0 {
		os.OrderingTypes = other.OrderingTypes
	}
	if len(os.CollectBroadcast) == 0 {
		os.CollectBroadcast = other.CollectBroadcast
	}
	if len(os.EncryptChannelsTypes) == 0 {
		os.EncryptChannelsTypes = other.EncryptChannelsTypes
	}
	if len(os.CoinTypes) == 0 {
		os.CoinTypes = other.CoinTypes
	}
	return os
}

// ComputeMaxConfigs returns the number of possible combinations of the fields of os.
func (os OptionStruct) ComputeMaxConfigs() int {
	return len(os.OrderingTypes) *
		len(os.StopOnCommitTypes) *
		len(os.ByzTypes) *
		len(os.UsePubIndexTypes) *
		len(os.StateMachineTypes) *
		len(os.SigTypes) *
		len(os.RotateCoordTypes) *
		len(os.RandMemberCheckerTypes) *
		len(os.AllowConcurrentTypes) *
		len(os.AllowSupportCoinTypes) *
		len(os.IncludeProofsTypes) *
		len(os.MemberCheckerTypes) *
		len(os.CollectBroadcast) *
		len(os.EncryptChannelsTypes) *
		len(os.AllowNoSignaturesTypes) *
		len(os.CoinTypes) *
		len(os.UseFixedCoinPresets)
}

var noValidConfigs = fmt.Errorf("no valid configs")

// checkConflict returns an error if the TestOptions are not valid for the ConfigOptions.
func checkConflict(opt GetOptionType, consConfig ConfigOptions, to types.TestOptions) (err error) {
	if err = checkIntersection(opt, consConfig, to); err != nil {
		return
	}
	_, err = to.CheckValid(to.ConsType, consConfig.GetIsMV())
	return
}

// checkIntersection returns an error if there is no intersection between the TestOptions and the ConfigOptions.
func checkIntersection(opt GetOptionType, consConfig ConfigOptions, to types.TestOptions) error {
	var ok bool
	for _, nxt := range consConfig.GetOrderingTypes(opt) {
		if to.OrderingType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetStateMachineTypes(opt) {
		if to.StateMachineType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetStopOnCommitTypes(opt) {
		if to.StopOnCommit == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetAllowNoSignatures(opt) {
		if to.NoSignatures == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetCoinTypes(opt) {
		if to.CoinType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetSigTypes(opt) {
		if to.SigType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetUsePubIndexTypes(opt) {
		if to.UsePubIndex == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetIncludeProofsTypes(opt) {
		if to.IncludeProofs == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetMemberCheckerTypes(opt) {
		if to.MCType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetRandMemberCheckerTypes(opt) {
		if to.RndMemberType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetRotateCoordTypes(opt) {
		if to.RotateCord == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetAllowSupportCoinTypes(opt) {
		if to.AllowSupportCoin == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetByzTypes(opt) {
		if to.ByzType == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetCollectBroadcast(opt) {
		if to.CollectBroadcast == nxt {
			ok = true
		}
	}
	if !ok {
		return noValidConfigs
	}

	ok = false
	for _, nxt := range consConfig.GetAllowConcurrentTypes(opt) {
		switch nxt {
		case true:
			if to.AllowConcurrent >= 1 {
				ok = true
			}
		case false:
			if to.AllowConcurrent < 1 {
				ok = true
			}
		}
	}
	if !ok {
		return noValidConfigs
	}
	return nil
}
