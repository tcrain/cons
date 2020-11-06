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

package gento

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/mvbinconsrnd2"
	"github.com/tcrain/cons/consensus/cons/mvcons2"
	"github.com/tcrain/cons/consensus/cons/mvcons3"
	"github.com/tcrain/cons/consensus/cons/rbbcast1"
	"github.com/tcrain/cons/consensus/cons/rbbcast2"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/parse"
)

func GenRandBenchTO() {
	// GenAll2AllSimple()
	GenAll2AllSimpleRand()
}

func GenAll2All() {
	GenAll2AllSimple()
	GenAll2AllCollectBcast()
}

func GenAll2AllSimple() {

	folderName := "all2all"
	genAll2AllSimple(folderName, false, 1)

	folderName = "all2all-sleep"
	genAll2AllSimple(folderName, true, 1)
}

func genAll2AllSimple(folderName string, sleepCrypto bool, nxtID uint64) uint64 {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:           []types.SigType{types.EC},
		CoinTypes:          []types.CoinType{types.NoCoinType},
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
		RotateCoordTypes:   types.WithTrue,
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons2Type}

	ct := mvAll2All
	ct.MaxRounds = 10
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = sleepCrypto
	ct.WarmUpInstances = 8
	ct.CPUProfile = false
	ct.KeepPast = 1
	ct.NumMsgProcessThreads = 3
	// set large timeouts since we don't have any faults for this test
	ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000
	ct.RotateCord = true
	ct.MCType = types.TrueMC

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)
	nxtID = genTO(nxtID, folderName, ct, []types.ConsType{types.RbBcast1Type}, []cons.ConfigOptions{rbbcast1.RbBcast1Config{}},
		optsSig, nil)

	ct.RotateCord = false
	nxtID = genTO(nxtID, folderName, ct, []types.ConsType{types.MvCons3Type}, []cons.ConfigOptions{mvcons3.MvCons3Config{}},
		optsSig, nil)

	ct.NoSignatures = true
	ct.EncryptChannels = true
	ct.RotateCord = true
	nxtID = genTO(nxtID, folderName, ct, []types.ConsType{types.RbBcast2Type}, []cons.ConfigOptions{rbbcast2.Config{}},
		optsSig, nil)

	consTypes = []types.ConsType{types.MvBinConsRnd2Type}
	ct.StopOnCommit = types.NextRound
	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		CoinTypes: []types.CoinType{types.FlipCoinType},
	}, optsSig)
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvbinconsrnd2.Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
	return nxtID
}

func GenAll2AllSimpleRand() {

	folderName := "all2all-rand"

	var nxtID uint64 = 1
	// nxtID = genAll2AllSimple(folderName, true, nxtID)

	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:  []types.SigType{types.EC},
		CoinTypes: []types.CoinType{types.NoCoinType},
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons2Type}

	ct := mvAll2All
	ct.MCType = types.CurrentTrueMC
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = true
	ct.WarmUpInstances = 8
	ct.CPUProfile = false
	// set large timeouts since we don't have any faults for this test
	// ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000

	// nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
	//	optsSig, nil)

	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		RandMemberCheckerTypes: []types.RndMemberType{types.VRFPerCons,
			types.KnownPerCons, types.VRFPerMessage},
	}, optsSig)
	ct.RndMemberCount = 10
	ct.GenRandBytes = true
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	ct.MCType = types.LaterMC
	nxtID = genTO(nxtID, folderName, ct, []types.ConsType{types.MvCons3Type}, []cons.ConfigOptions{mvcons3.MvCons3Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByRndMemberType})

}

func GenAll2AllCollectBcast() {

	folderName := "all2all-cbcast2s"
	genAll2AllCollectBcast2s(folderName, 1)

	folderName = "all2all-cbcast3s"
	genAll2AllCollectBcast3s(folderName, 1)

	folderName = "all2all-cbcast"
	genAll2AllCollectBcast(folderName, false, 1)

	folderName = "all2all-cbcast-sleep"
	genAll2AllCollectBcast(folderName, true, 1)
}

func genAll2AllCollectBcast2s(folderName string, nxtID uint64) uint64 {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:           []types.SigType{types.EC},
		CoinTypes:          []types.CoinType{types.NoCoinType},
		CollectBroadcast:   []types.CollectBroadcastType{types.Full, types.Commit},
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons3Type}

	ct := mvAll2All
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = true
	ct.WarmUpInstances = 8
	ct.CPUProfile = false
	// set large timeouts since we don't have any faults for this test
	ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000
	ct.MCType = types.TrueMC

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons3.MvCons3Config{}},
		optsSig, nil)

	ct.SigType = types.TBLS
	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		CollectBroadcast: []types.CollectBroadcastType{types.Full, types.Commit},
		SigTypes:         []types.SigType{types.TBLS},
	}, optsSig)
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons3.MvCons3Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByConsCB})
	return nxtID
}

func genAll2AllCollectBcast3s(folderName string, nxtID uint64) uint64 {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:           []types.SigType{types.EC},
		CoinTypes:          []types.CoinType{types.NoCoinType},
		CollectBroadcast:   types.AllCollectBroadcast,
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
		RotateCoordTypes:   types.WithTrue,
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons2Type}

	ct := mvAll2All
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = true
	ct.WarmUpInstances = 8
	ct.CPUProfile = false
	// set large timeouts since we don't have any faults for this test
	ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000
	ct.MCType = types.TrueMC
	ct.RotateCord = true

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	ct.SigType = types.TBLS
	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		CollectBroadcast: types.AllCollectBroadcast, // []types.CollectBroadcastType{types.EchoCommit, types.Commit},
		SigTypes:         []types.SigType{types.TBLS},
	}, optsSig)
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByConsCB})
	return nxtID
}

func genAll2AllCollectBcast(folderName string, sleepCrypto bool, nxtID uint64) uint64 {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:           []types.SigType{types.TBLS},
		CoinTypes:          []types.CoinType{types.NoCoinType},
		CollectBroadcast:   []types.CollectBroadcastType{types.Commit},
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
		RotateCoordTypes:   types.WithTrue,
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons3Type}

	ct := mvAll2All
	ct.CollectBroadcast = types.Commit
	ct.SigType = types.TBLS
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = sleepCrypto
	ct.WarmUpInstances = 8
	ct.CPUProfile = false
	// set large timeouts since we don't have any faults for this test
	ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000
	ct.ForwardTimeout = 100000
	ct.MCType = types.TrueMC

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons3.MvCons3Config{}},
		optsSig, nil)

	ct.RotateCord = true
	nxtID = genTO(nxtID, folderName, ct, []types.ConsType{types.RbBcast1Type}, []cons.ConfigOptions{rbbcast1.RbBcast1Config{}},
		optsSig, nil)

	ct.CollectBroadcast = types.EchoCommit
	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		CollectBroadcast: []types.CollectBroadcastType{types.EchoCommit}, // types.EchoCommit, types.Commit},
	}, optsSig)
	consTypes = []types.ConsType{types.MvCons2Type}

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
	return nxtID
}
