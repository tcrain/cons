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

/*
This package outputs test options files to disk based on the variables.
*/
package main

import (
	"flag"
	"github.com/tcrain/cons/consensus/utils"
	"github.com/tcrain/cons/gento"
)

/*func GenBufferForward() {
	// buffer forward
	folderName := "mvbufforward"
	mvc := []types.ConsType{types.MvCons2Type}
	nxtID := genTO(1, folderName, mvBuffForward, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}}, baseMVOptions, nil)
	genTO(nxtID, "mvbufforward", mvP2p, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}}, baseMVOptions, nil)
	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
}

func GenRandVRF() {
	// rand VRF
	folderName := "mvvrf"
	mvc := []types.ConsType{types.MvCons2Type}
	to := mvAll2All
	to.CPUProfile = true
	to.SleepValidate = sleepval
	to.SigType = types.BLS
	to.NumMsgProcessThreads = 5
	nxtID := genTO(1, folderName, to, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}},
		cons.ReplaceNilFields(cons.OptionStruct{SigTypes: []types.SigType{types.BLS}}, baseMVOptions), nil)
	to.GenRandBytes = true
	to.RndMemberCount = 10
	to.RndMemberType = types.KnownPerCons
	genTO(nxtID, "mvvrf", to, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}},
		cons.ReplaceNilFields(cons.OptionStruct{
			SigTypes:               []types.SigType{types.BLS},
			RandMemberCheckerTypes: []types.RndMemberType{types.VRFPerMessage, types.VRFPerCons, types.KnownPerCons}}, baseMVOptions), nil)
	genSet := parse.GenPerNodeByCons
	genSet.GenItems = []parse.GenItem{{VaryField: parse.VaryField{VaryField: "NodeCount"},
		ExtraFields: []parse.VaryField{{VaryField: "ConsType"}, {VaryField: "RndMemberType"}}}}
	genGenSets(folderName, []parse.GenSet{genSet})
}

func GenMvAll2All() {
	// normal MV
	folderName := "mvall2all"
	mvc := []types.ConsType{types.MvCons2Type}
	genTO(1, folderName, mvAll2All, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}}, baseMVOptions, nil)
	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
}

func GenP2PMv() {
	// p2p mv
	mvc := []types.ConsType{types.MvCons2Type}
	folderName := "mvp2p"
	genTO(1, folderName, mvP2p, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}},
		cons.ReplaceNilFields(cons.OptionStruct{
			SigTypes: []types.SigType{types.EC, types.TBLS},
		}, baseMVOptions), nil)
	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
}

func GenTBLSMv() {
	// normal TBLS mv
	mvc := []types.ConsType{types.MvCons2Type}
	folderName := "mvall2allTBLS"
	genTO(1, folderName, mvAll2AllTBLS, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}},
		cons.ReplaceNilFields(cons.OptionStruct{
			MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
			SigTypes:           []types.SigType{types.TBLS},
		}, baseMVOptions), nil)
	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByCons})
}

func GenCollectBroadcastMV() {
	// mv collect broadcast
	folderName := "mvall2allcb"
	mvc := []types.ConsType{types.MvCons2Type}
	genTO(1, folderName, mvAll2AllTBLS, mvc, []cons.ConfigOptions{cons.StandardMvConfig{}},
		genTestCollectBroadcast(baseMVOptions), nil)
	genSet := parse.GenPerNodeByCons
	genSet.GenItems = []parse.GenItem{{VaryField: parse.VaryField{VaryField: "NodeCount"},
		ExtraFields: []parse.VaryField{{VaryField: "ConsType"}, {VaryField: "CollectBroadcast"}}}}
	genGenSets(folderName, []parse.GenSet{genSet})
}

func GenBinNormalCoinSig5(useCoinPresets bool) {
	// The test config for normal random test
	binRndSig := []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}

	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:  types.WithTrue,
		SigTypes:            []types.SigType{types.TBLSDual},
		CoinTypes:           []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets: []bool{useCoinPresets},
	}, baseBinOptions)

	var folderName string
	if useCoinPresets {
		folderName = "s-coin5presets"
	} else {
		folderName = "s-coin5"
	}
	ct := rndBinAll2All
	ct.StopOnCommit = types.SendProof
	ct.IncludeProofs = true
	ct.AllowSupportCoin = false
	ct.CoinType = types.StrongCoin1Type
	ct.SigType = types.TBLSDual
	ct.SleepValidate = sleepval
	ct.UseFixedSeed = fixedSeed
	ct.UseFixedCoinPresets = useCoinPresets
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnes})
}

func GenBinNormalCoinNoSigSingle(useCoinPresets bool) {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type, types.BinConsRnd4Type}

	percentOnes := []int{0, 33, 50, 66, 100}

	var folderName string
	if useCoinPresets {
		folderName = "ns-coinpresets"
	} else {
		folderName = "ns-coin"
	}

	optsNoSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:  types.WithFalse,
		SigTypes:            []types.SigType{types.TBLSDual},
		CoinTypes:           []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets: []bool{useCoinPresets},
	}, baseBinOptions)
	ct := rndBinAll2All
	ct.StopOnCommit = types.NextRound
	ct.IncludeProofs = false
	ct.EncryptChannels = true
	ct.NoSignatures = true
	ct.UseFixedSeed = fixedSeed
	ct.CoinType = types.StrongCoin1Type
	ct.SigType = types.TBLSDual
	ct.SleepValidate = sleepval
	ct.MaxRounds = 1000
	ct.UseFixedCoinPresets = useCoinPresets
	genTO(4, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
		optsNoSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnes})
}


*/

func main() {
	/*GenBufferForward()
	GenRandVRF()
	GenMvAll2All()
	GenP2PMv()
	GenTBLSMv()
	GenCollectBroadcastMV()
	GenBinBoth()
	GenBinDualCoin()*/
	//GenBinNormalCoinSigOther(true)
	// GenBinNormalCoinNoSigOther(true)

	/*		GenBinNormalCoinNoSigSingle(true)
			GenBinNormalCoinNoSigSingle(false)

			GenBinNormalCoinSig5(true)
			GenBinNormalCoinSig5(false)

	GenBinNormalCoinSigScaleOnce(false)
	GenBinNormalCoinSigScaleOnce(true)
	*/

	var getType int
	flag.IntVar(&getType, "i", -1, "set of test options to generate, -1 means generate all")
	flag.Parse()

	genTOFuncs := []func(){
		gento.GenCoinTO,
		gento.GenRandBenchTO,
		gento.GenAll2All,
		gento.GenP2PTO,
	}

	var genIdx []int
	switch getType {
	case -1:
		genIdx = utils.GenList(len(genTOFuncs))
	default:
		genIdx = append(genIdx, getType)
	}
	for _, nxt := range genIdx {
		genTOFuncs[nxt]()
	}
}
