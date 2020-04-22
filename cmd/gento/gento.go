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
	"fmt"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/binconsrnd1"
	"github.com/tcrain/cons/consensus/cons/binconsrnd2"
	"github.com/tcrain/cons/consensus/cons/binconsrnd4"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/parse"
	"os"
	"path/filepath"
)

const (
	sleepval  = false
	fixedSeed = true
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

func GenBinNormalSimple() {
	// The test config for normal random test
	var binRndSig []types.ConsType
	percentOnes := []int{33}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes: types.WithTrue,
		SigTypes:           []types.SigType{types.TBLSDual},
		CoinTypes:          []types.CoinType{types.StrongCoin1Type},
	}, baseBinOptions)

	binRndSig = []types.ConsType{types.BinConsRnd1Type}

	folderName := "simplebin"

	ct := rndBinAll2All
	ct.StopOnCommit = types.SendProof
	ct.IncludeProofs = true
	ct.AllowSupportCoin = false
	ct.CoinType = types.StrongCoin1Type
	ct.SigType = types.TBLSDual
	ct.SleepValidate = true
	ct.UseFixedSeed = fixedSeed
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByNodeCount})

}

func GenBinNormalCoinSig(useCoinPresets bool, uniqueFolder bool) {
	// The test config for normal random test
	var binRndSig []types.ConsType
	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:  types.WithTrue,
		SigTypes:            []types.SigType{types.TBLSDual},
		CoinTypes:           []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets: []bool{useCoinPresets},
	}, baseBinOptions)

	binRndSig = []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}

	var folderName string
	if useCoinPresets {
		folderName = "s-coinpresets"
	} else {
		// binRndSig = []types.ConsType{types.BinConsRnd5Type, types.BinConsRnd5OldType}
		folderName = "s-coin"
	}
	if uniqueFolder {
		folderName = "s" + folderName
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

	if !useCoinPresets {
		/*		folderName = "s-coinpresets"
				if uniqueFolder {
					folderName = "s" + folderName
				}
				genTO(4, folderName, ct, []types.ConsType{types.BinConsRnd5Type}, []cons.ConfigOptions{binconsrnd5.Config{}},
					optsSig, percentOnes)*/
	}
}

func GenBinNormalCoinNoSig(useCoinPresets bool, uniqueFolder bool) {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type, types.BinConsRnd4Type, types.BinConsRnd6Type}

	percentOnes := []int{33, 50, 66}

	var folderName string
	if useCoinPresets {
		folderName = "s-coinpresets"
	} else {
		folderName = "s-coin"
	}
	if uniqueFolder {
		folderName = "n" + folderName
		genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnes})
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
	ct.UseFixedCoinPresets = useCoinPresets
	genTO(7, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
		optsNoSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnes})

	if !useCoinPresets {
		/*		folderName = "s-coinpresets"
				if uniqueFolder {
					folderName = "n" + folderName
				}
				genTO(10, folderName, ct, []types.ConsType{types.BinConsRnd6Type}, []cons.ConfigOptions{binconsrnd6.Config{}},
					optsNoSig, percentOnes)
		*/
	}

}

func GenBinNormalCoinByzNoSig(useCoinPresets bool) {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type, types.BinConsRnd4Type, types.BinConsRnd6Type}
	percentOnes := []int{33, 50, 66}

	tstIdx := uint64(50)
	for _, nxt := range types.AllByzTypes {

		var folderName string
		if useCoinPresets {
			folderName = "ns-coinbyzpresets"
		} else {
			folderName = "ns-coinbyz"
		}
		genGenSets(folderName, []parse.GenSet{parse.GenByByzTypes})
		optsNoSig := cons.ReplaceNilFields(cons.OptionStruct{
			IncludeProofsTypes: types.WithFalse,
			SigTypes:           []types.SigType{types.TBLSDual},
			CoinTypes:          []types.CoinType{types.StrongCoin1Type},
			ByzTypes:           []types.ByzType{nxt},
		}, baseBinOptions)
		ct := rndBinAll2All
		ct.ByzType = nxt
		ct.StopOnCommit = types.NextRound
		ct.IncludeProofs = false
		ct.EncryptChannels = true
		ct.NoSignatures = true
		ct.UseFixedSeed = fixedSeed
		ct.CoinType = types.StrongCoin1Type
		ct.SigType = types.TBLSDual
		ct.SleepValidate = sleepval
		ct.UseFixedCoinPresets = useCoinPresets
		genTO(tstIdx, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
			optsNoSig, percentOnes)
		tstIdx += 3
		if !useCoinPresets {
			/*			folderName = "ns-coinbyzpresets"
						genTO(tstIdx, folderName, ct, []types.ConsType{types.BinConsRnd6Type}, []cons.ConfigOptions{binconsrnd6.Config{}},
							optsNoSig, percentOnes)
			*/
		}
		tstIdx += 3
	}
}

func GenBinNormalCoinByzSig(useCoinPresets bool) {
	// The test config for normal random test
	var binRndSig []types.ConsType
	percentOnes := []int{33, 50, 66}

	tstIdx := uint64(1)
	for _, nxt := range types.AllByzTypes {
		optsSig := cons.ReplaceNilFields(cons.OptionStruct{
			IncludeProofsTypes: types.WithTrue,
			SigTypes:           []types.SigType{types.TBLSDual},
			CoinTypes:          []types.CoinType{types.StrongCoin1Type},
			ByzTypes:           []types.ByzType{nxt},
		}, baseBinOptions)

		binRndSig = []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}
		var folderName string
		if useCoinPresets {
			folderName = "s-coinbyzpresets"
		} else {
			folderName = "s-coinbyz"
		}

		ct := rndBinAll2All
		ct.ByzType = nxt
		ct.StopOnCommit = types.SendProof
		ct.IncludeProofs = true
		ct.AllowSupportCoin = false
		ct.CoinType = types.StrongCoin1Type
		ct.SigType = types.TBLSDual
		ct.SleepValidate = sleepval
		ct.UseFixedCoinPresets = useCoinPresets
		ct.UseFixedSeed = fixedSeed
		genTO(tstIdx, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
			optsSig, percentOnes)
		genGenSets(folderName, []parse.GenSet{parse.GenByByzTypes})
		tstIdx += 3
		if !useCoinPresets {
			/*			folderName = "s-coinbyzpresets"
						genTO(tstIdx, folderName, ct, []types.ConsType{types.BinConsRnd5Type}, []cons.ConfigOptions{binconsrnd5.Config{}},
							optsSig, percentOnes)
			*/
		}
		tstIdx += 3

	}
}

func GenBinNormalCoin2Sig(useCoinPresets bool) {
	// The test config for normal random test
	var binRndSig []types.ConsType

	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		UseFixedCoinPresets: []bool{useCoinPresets},
		IncludeProofsTypes:  types.WithBothBool,
		SigTypes:            []types.SigType{types.EDCOIN},
		CoinTypes:           []types.CoinType{types.StrongCoin2Type},
	}, baseBinOptions)

	binRndSig = []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}
	var folderName string
	if useCoinPresets {
		folderName = "s-coin2presets"
	} else {
		folderName = "s-coin2"
	}

	ct := rndBinAll2All
	ct.StopOnCommit = types.NextRound
	ct.AllowSupportCoin = false
	ct.CoinType = types.StrongCoin2Type
	ct.SigType = types.EDCOIN
	ct.SleepValidate = sleepval
	ct.UseFixedCoinPresets = useCoinPresets
	ct.UseFixedSeed = fixedSeed
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnesIncludeProofs})
	if !useCoinPresets {
		/*		folderName = "s-coin2presets"
				genTO(7, folderName, ct, []types.ConsType{types.BinConsRnd5Type}, []cons.ConfigOptions{binconsrnd5.Config{}},
					optsSig, percentOnes)
		*/
	}

}

func GenBinNormalCoin2NoSig(useCoinPresets bool) {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type, types.BinConsRnd4Type, types.BinConsRnd6Type}
	percentOnes := []int{33, 50, 66}

	var folderName string
	if useCoinPresets {
		folderName = "ns-coin2presets"
	} else {
		folderName = "ns-coin2"
	}
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnesIncludeProofs})

	optsNoSig := cons.ReplaceNilFields(cons.OptionStruct{
		UseFixedCoinPresets: []bool{useCoinPresets},
		IncludeProofsTypes:  types.WithFalse,
		SigTypes:            []types.SigType{types.EDCOIN},
		CoinTypes:           []types.CoinType{types.StrongCoin2Type},
	}, baseBinOptions)
	ct := rndBinAll2All
	ct.StopOnCommit = types.NextRound
	ct.IncludeProofs = false
	ct.EncryptChannels = true
	ct.NoSignatures = true
	ct.UseFixedSeed = fixedSeed
	ct.CoinType = types.StrongCoin2Type
	ct.SigType = types.EDCOIN
	ct.SleepValidate = sleepval
	ct.UseFixedCoinPresets = useCoinPresets
	genTO(13, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
		optsNoSig, percentOnes)
	if !useCoinPresets {
		/*		folderName = "ns-coin2presets"
				genTO(19, folderName, ct, []types.ConsType{types.BinConsRnd6Type}, []cons.ConfigOptions{binconsrnd6.Config{}},
					optsNoSig, percentOnes)
		*/
	}
}

func GenBinNormalCoin2EchoSig() {
	// The test config for normal random test
	binRndSig := []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}

	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:  []types.SigType{types.EDCOIN},
		CoinTypes: []types.CoinType{types.StrongCoin2Type, types.StrongCoin2EchoType},
	}, baseBinOptions)

	var folderName string
	folderName = "s-coin2echo"
	ct := rndBinAll2All
	ct.StopOnCommit = types.NextRound
	ct.AllowSupportCoin = false
	ct.IncludeProofs = false
	ct.CoinType = types.StrongCoin2Type
	ct.SigType = types.EDCOIN
	ct.SleepValidate = sleepval
	ct.UseFixedCoinPresets = false
	ct.UseFixedSeed = fixedSeed
	ct.EncryptChannels = true
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnesCoinType})
}

func GenBinNormalCoin2EchoNoSig() {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type}
	percentOnes := []int{33, 50, 66}

	var folderName string
	folderName = "ns-coin2echo"
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnesCoinType})

	optsNoSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:  []types.SigType{types.EDCOIN},
		CoinTypes: []types.CoinType{types.StrongCoin2Type, types.StrongCoin2EchoType},
	}, baseBinOptions)
	ct := rndBinAll2All
	ct.StopOnCommit = types.NextRound
	ct.IncludeProofs = false
	ct.EncryptChannels = true
	ct.NoSignatures = true
	ct.UseFixedSeed = fixedSeed
	ct.CoinType = types.StrongCoin2Type
	ct.SigType = types.EDCOIN
	ct.SleepValidate = sleepval
	ct.UseFixedCoinPresets = false
	genTO(7, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
		optsNoSig, percentOnes)

	optsNoSig = cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:  []types.SigType{types.EDCOIN},
		CoinTypes: []types.CoinType{types.StrongCoin2Type},
	}, baseBinOptions)
	ct.CoinType = types.StrongCoin2Type
	ct.SigType = types.EDCOIN
	ct.SleepValidate = sleepval
	ct.UseFixedCoinPresets = false
	genTO(13, folderName, ct, []types.ConsType{types.BinConsRnd4Type, types.BinConsRnd6Type}, []cons.ConfigOptions{binconsrnd4.Config{}},
		optsNoSig, percentOnes)

}

func GenBinNormalCoinCombineSig() {
	// The test config for normal random test
	binRndSig := []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}

	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:    types.WithTrue,
		SigTypes:              []types.SigType{types.TBLSDual},
		CoinTypes:             []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets:   types.WithFalse,
		AllowSupportCoinTypes: types.WithBothBool,
	}, baseBinOptions)

	var folderName string
	folderName = "s-coincombine"
	ct := rndBinAll2All
	ct.StopOnCommit = types.SendProof
	ct.IncludeProofs = true
	ct.AllowSupportCoin = false
	ct.CoinType = types.StrongCoin1Type
	ct.SigType = types.TBLSDual
	ct.SleepValidate = sleepval
	ct.UseFixedSeed = fixedSeed
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByPercentOnesCombine})
}

func GenBinNormalCoinSigScale(fixedCoinPresets bool) {
	// The test config for normal random test
	var binRndSig []types.ConsType

	percentOnes := []int{33, 50, 66}
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:  types.WithTrue,
		SigTypes:            []types.SigType{types.TBLSDual},
		CoinTypes:           []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets: []bool{fixedCoinPresets},
	}, baseBinOptions)

	binRndSig = []types.ConsType{types.BinConsRnd1Type, types.BinConsRnd3Type, types.BinConsRnd5Type}
	var folderName string
	if fixedCoinPresets {
		folderName = "s-coinscalepresets"
	} else {
		folderName = "s-coinscale"
	}
	ct := rndBinAll2All
	ct.StopOnCommit = types.SendProof
	ct.IncludeProofs = true
	ct.AllowSupportCoin = false
	ct.CoinType = types.StrongCoin1Type
	ct.SigType = types.TBLSDual
	ct.SleepValidate = sleepval
	ct.UseFixedSeed = fixedSeed
	ct.UseFixedCoinPresets = fixedCoinPresets
	genTO(1, folderName, ct, binRndSig, []cons.ConfigOptions{binconsrnd1.Config{}},
		optsSig, percentOnes)
	genGenSets(folderName, []parse.GenSet{parse.GenByNodeCount})
	if !fixedCoinPresets {
		/*		folderName = "s-coinscalepresets"
				genTO(4, folderName, ct, []types.ConsType{types.BinConsRnd5Type}, []cons.ConfigOptions{binconsrnd5.Config{}},
					optsSig, percentOnes)
		*/
	}

}

func GenBinNormalCoinScaleNoSig(fixedCoinPresets bool) {
	// The test config for normal random test
	binRndNoSig := []types.ConsType{types.BinConsRnd2Type, types.BinConsRnd4Type, types.BinConsRnd6Type}

	percentOnes := []int{33, 50, 66}

	var folderName string
	if fixedCoinPresets {
		folderName = "ns-coinscalepresets"
	} else {
		folderName = "ns-coinscale"
	}
	genGenSets(folderName, []parse.GenSet{parse.GenByNodeCount})

	optsNoSig := cons.ReplaceNilFields(cons.OptionStruct{
		IncludeProofsTypes:  types.WithFalse,
		SigTypes:            []types.SigType{types.TBLSDual},
		CoinTypes:           []types.CoinType{types.StrongCoin1Type},
		UseFixedCoinPresets: []bool{fixedCoinPresets},
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
	ct.UseFixedCoinPresets = fixedCoinPresets
	genTO(7, folderName, ct, binRndNoSig, []cons.ConfigOptions{binconsrnd2.Config{}},
		optsNoSig, percentOnes)
	if !fixedCoinPresets {
		/*		folderName = "s-coinscalepresets"
				genTO(10, folderName, ct, []types.ConsType{types.BinConsRnd6Type}, []cons.ConfigOptions{binconsrnd6.Config{}},
					optsNoSig, percentOnes)
		*/
	}

}

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

	GenBinNormalSimple()

	// GenBinNormalCoinSig(true, false)
	GenBinNormalCoinSig(true, false)
	GenBinNormalCoinNoSig(true, true)
	// GenBinNormalCoinNoSig(true, false)

	GenBinNormalCoinSig(false, false)
	GenBinNormalCoinNoSig(false, true)

	GenBinNormalCoinByzNoSig(true)
	GenBinNormalCoinByzSig(true)

	GenBinNormalCoinByzNoSig(false)
	GenBinNormalCoinByzSig(false)

	GenBinNormalCoin2Sig(true)
	GenBinNormalCoin2NoSig(true)
	GenBinNormalCoin2Sig(false)
	GenBinNormalCoin2NoSig(false)

	GenBinNormalCoin2EchoSig()
	GenBinNormalCoin2EchoNoSig()

	GenBinNormalCoinCombineSig()

	GenBinNormalCoinSigScale(true)
	GenBinNormalCoinScaleNoSig(true)
	GenBinNormalCoinSigScale(false)
	GenBinNormalCoinScaleNoSig(false)

}

var baseBinOptions = cons.OptionStruct{
	OrderingTypes:          []types.OrderingType{types.Total},
	ByzTypes:               []types.ByzType{types.NonFaulty},
	StateMachineTypes:      []types.StateMachineType{types.BinaryProposer},
	SigTypes:               []types.SigType{types.TBLS},
	UsePubIndexTypes:       types.WithTrue,
	IncludeProofsTypes:     types.WithFalse,
	MemberCheckerTypes:     []types.MemberCheckerType{types.CurrentTrueMC},
	RandMemberCheckerTypes: []types.RndMemberType{types.NonRandom},
	RotateCoordTypes:       types.WithFalse,
	AllowSupportCoinTypes:  types.WithFalse,
	AllowConcurrentTypes:   []types.ConsensusInt{0},
	CollectBroadcast:       []types.CollectBroadcastType{types.Full},
}

var baseMVOptions = cons.OptionStruct{
	OrderingTypes:          []types.OrderingType{types.Total},
	ByzTypes:               []types.ByzType{types.NonFaulty},
	StateMachineTypes:      []types.StateMachineType{types.CounterProposer},
	SigTypes:               []types.SigType{types.EC},
	UsePubIndexTypes:       types.WithTrue,
	IncludeProofsTypes:     types.WithFalse,
	MemberCheckerTypes:     []types.MemberCheckerType{types.CurrentTrueMC},
	RandMemberCheckerTypes: []types.RndMemberType{types.NonRandom},
	RotateCoordTypes:       types.WithFalse,
	AllowSupportCoinTypes:  types.WithFalse,
	AllowConcurrentTypes:   []types.ConsensusInt{0},
	CollectBroadcast:       []types.CollectBroadcastType{types.Full},
}

var rndBinAll2All = types.TestOptions{
	StopOnCommit:         types.SendProof,
	CPUProfile:           false,
	ConsType:             types.BinConsRnd1Type,
	NumTotalProcs:        10,
	OrderingType:         types.Total,
	MaxRounds:            100,
	StorageType:          types.Diskstorage,
	NetworkType:          types.AllToAll,
	ConnectionType:       types.TCP,
	CheckDecisions:       true,
	IncludeProofs:        false,
	SigType:              types.TBLS,
	UsePubIndex:          true,
	MCType:               types.CurrentTrueMC,
	StateMachineType:     types.BinaryProposer,
	RotateCord:           false,
	CoinType:             types.StrongCoin1Type,
	NumMsgProcessThreads: 20,
	UseFixedSeed:         fixedSeed,
}

var mvAll2All = types.TestOptions{
	CPUProfile:       true,
	ConsType:         types.MvBinCons1Type,
	NumTotalProcs:    10,
	OrderingType:     types.Total,
	MaxRounds:        10,
	StorageType:      types.Diskstorage,
	NetworkType:      types.AllToAll,
	ConnectionType:   types.TCP,
	CheckDecisions:   true,
	IncludeProofs:    false,
	SigType:          types.EC,
	UsePubIndex:      true,
	MCType:           types.CurrentTrueMC,
	StateMachineType: types.CounterProposer,
	RotateCord:       false,
	UseFixedSeed:     fixedSeed,
}

var mvBuffForward = types.TestOptions{
	ConsType:              types.MvBinCons1Type,
	NumTotalProcs:         10,
	OrderingType:          types.Total,
	MaxRounds:             10,
	StorageType:           types.Diskstorage,
	NetworkType:           types.P2p,
	ConnectionType:        types.TCP,
	CheckDecisions:        true,
	IncludeProofs:         false,
	IncludeCurrentSigs:    true,
	SigType:               types.BLS,
	BlsMultiNew:           true,
	UseMultisig:           true,
	SleepValidate:         false,
	UsePubIndex:           true,
	MCType:                types.CurrentTrueMC,
	StateMachineType:      types.CounterProposer,
	BufferForwarder:       true,
	FanOut:                6,
	RotateCord:            false,
	AdditionalP2PNetworks: 2,
	UseFixedSeed:          fixedSeed,
}

var mvP2p = mvAll2All

var mvAll2AllTBLS = mvAll2All

func init() {
	mvP2p.NetworkType = types.P2p
	mvP2p.FanOut = 4

	mvAll2AllTBLS.SigType = types.TBLS
	mvAll2AllTBLS.MCType = types.TrueMC
}

func genTestSigOptions(baseOptions cons.OptionStruct) cons.OptionStruct {
	return cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes: types.AllSigTypes,
	}, baseOptions)
}

func genTestCollectBroadcast(baseOptions cons.OptionStruct) cons.OptionStruct {
	return cons.ReplaceNilFields(cons.OptionStruct{
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
		SigTypes:           []types.SigType{types.TBLS},
		CollectBroadcast:   types.AllCollectBroadcast,
	}, baseOptions)
}

func genGenSets(folderName string, items []parse.GenSet) {
	folderPath := filepath.Join("testconfigs", folderName)
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		panic(err)
	}
	if err := parse.GenSetToDisk(folderPath, items); err != nil {
		panic(err)
	}
}

func genTO(startID uint64, folderName string, baseTO types.TestOptions, consTypes []types.ConsType,
	consConfigs []cons.ConfigOptions, options cons.OptionStruct, percentOnes []int) uint64 {

	if startID == 0 {
		panic("start id must be at least 1")
	}

	toMap := make(map[types.TestOptions][]types.ConsType)
	for _, nxtConfig := range consConfigs {
		iter, err := cons.NewTestOptIter(cons.AllOptions, nxtConfig, cons.NewSingleIter(options, baseTO))
		if err != nil {
			panic(err)
		}
		nxt, hasNxt := iter.Next()
		for ; hasNxt; nxt, hasNxt = iter.Next() {
			toMap[nxt] = consTypes
		}
		toMap[nxt] = consTypes
	}
	if len(percentOnes) > 0 {
		newToMap := make(map[types.TestOptions][]types.ConsType)
		for nxtTO := range toMap {
			for _, nxtPo := range percentOnes {
				nxtTO.BinConsPercentOnes = nxtPo
				newToMap[nxtTO] = consTypes
			}
		}
		toMap = newToMap
	}
	folderPath := filepath.Join("testconfigs", folderName)
	fmt.Println("\nGen test options folder", folderPath)
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		panic(err)
	}

	var i = startID
	var prv types.TestOptionsCons
	for nxt, cts := range toMap {
		nxt.TestID = i
		fmt.Println("Config change:", prv.StringDiff(nxt))
		prv = types.TestOptionsCons{
			TestOptions: nxt,
			ConsTypes:   cts,
		}
		if err := types.TOConsToDisk(folderPath, prv); err != nil {
			panic(err)
		}
		i++
	}
	return i
}
