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
	"fmt"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/parse"
	"os"
	"path/filepath"
)

const (
	sleepval  = false
	fixedSeed = true
)

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
	MemberCheckerTypes:     []types.MemberCheckerType{types.TrueMC},
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
	BufferForwardType:     types.ThresholdBufferForward,
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
