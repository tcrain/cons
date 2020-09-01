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
This package runs a test given a test configuration stored in a json file.
*/
package main

import (
	"flag"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consgen"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/rpcsetup"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
)

var defaultTO = types.TestOptions{
	UseFixedSeed: false,
	OrderingType: types.Total,
	FailRounds:   0,
	FailDuration: 0,
	MaxRounds:    config.MaxRounds,
	NumFailProcs: 0,

	ConsType:                types.MvBinConsRnd2Type,
	NumTotalProcs:           config.ProcCount,
	NumNonMembers:           config.NonMembers,
	RndMemberCount:          config.ProcCount - 1,
	RndMemberType:           types.KnownPerCons,
	NetworkType:             types.AllToAll,
	UseRandCoord:            true,
	GenRandBytes:            true,
	LocalRandMemberChange:   0,
	NodeChoiceVRFRelaxation: 0,
	CoordChoiceVRF:          0,
	FanOut:                  3,
	CoinType:                types.FlipCoinType,

	StorageType:                 types.Diskstorage,
	ClearDiskOnRestart:          false,
	ConnectionType:              types.TCP,
	ByzType:                     types.NonFaulty,
	NumByz:                      0,
	CheckDecisions:              true,
	MsgDropPercent:              0,
	IncludeProofs:               false,
	SigType:                     types.BLS,
	UsePubIndex:                 true,
	SleepValidate:               true,
	SleepCrypto:                 true,
	MCType:                      types.CurrentTrueMC,
	BufferForwarder:             false,
	UseMultisig:                 false,
	BlsMultiNew:                 true,
	MemCheckerBitIDType:         types.BitIDSlice,
	SigBitIDType:                types.BitIDChoose,
	StateMachineType:            types.BytesProposer,
	PartialMessageType:          types.NoPartialMessages,
	AllowConcurrent:             0,
	RotateCord:                  false,
	AllowSupportCoin:            false,
	UseFullBinaryState:          false,
	StorageBuffer:               0,
	IncludeCurrentSigs:          false,
	CPUProfile:                  false,
	MemProfile:                  false,
	NumMsgProcessThreads:        2,
	MvProposalSizeBytes:         0,
	BinConsPercentOnes:          50,
	CollectBroadcast:            types.Full,
	StopOnCommit:                types.Immediate,
	ByzStartIndex:               0,
	TestID:                      0,
	AdditionalP2PNetworks:       0,
	EncryptChannels:             true,
	NoSignatures:                false,
	UseFixedCoinPresets:         false,
	SharePubsRPC:                false,
	WarmUpInstances:             0,
	KeepPast:                    0,
	ForwardTimeout:              0,
	RandForwardTimeout:          0,
	ProgressTimeout:             0,
	MvConsTimeout:               0,
	MvConsRequestRecoverTimeout: 0,
}

func main() {
	var optionsFile string
	var checkOnly bool

	flag.StringVar(&optionsFile, "o", "tofile.json", "Path to file containing list of options")
	flag.BoolVar(&checkOnly, "c", false, "Only check if the test option is valid then exit")
	flag.Parse()

	var oFlagSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "o" || f.Name == "c" {
			oFlagSet = true
		}
	})

	var to types.TestOptions
	var err error
	if oFlagSet {
		logging.Info("Loading test options from file: ", optionsFile)
		to, err = types.GetTestOptions(optionsFile)
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		if checkOnly {
			return
		}
	} else {
		logging.Info("Using default test options")
		to = defaultTO
	}

	logging.Infof("Loaded options: %s\n", to)

	if to.TestID == 0 {
		to.TestID = rand.Uint64()
		logging.Printf("Set test ID to %v\n", to.TestID)
	}

	cfg := consgen.GetConsConfig(to)
	if to, err = to.CheckValid(to.ConsType, cfg.GetIsMV()); err != nil {
		logging.Error("invalid config ", err)
		panic(err)
	}

	cons.RunConsType(consgen.GetConsItem(to), cfg.GetBroadcastFunc(to.ByzType), cfg, to, rpcsetup.TPanic{})
}
