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
	"bytes"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/dual"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/auth/sig/qsafe"
	"github.com/tcrain/cons/consensus/auth/sig/sleep"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/statemachine/asset"
	"github.com/tcrain/cons/consensus/statemachine/currency"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channel/csnet"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/statemachine"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/storage"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

// TODO clean this stuff up
func SetTestConfigOptions(to *types.TestOptions, fistCall bool) {
	// set the defaults for the proposers
	//if to.BinConsPercentOnes == 0 {
	//	to.BinConsPercentOnes = config.DefaultPercentOnes
	//}
	if to.MvProposalSizeBytes == 0 {
		to.MvProposalSizeBytes = config.DefaultMvConsProposalBytes
	}
	if to.LocalRandMemberChange == 0 {
		to.LocalRandMemberChange = config.DefaultLocalRandMemberChange
	}
	if to.NumMsgProcessThreads == 0 {
		to.NumMsgProcessThreads = config.DefaultMsgProcesThreads
	}

	if fistCall {
		SetConfigOptions(*to)
	}
}

// SetConfigOptions must be called at the beginning of every test to set configuration options.
func SetConfigOptions(to types.TestOptions) {
	sig.SetUsePubIndex(to.UsePubIndex)
	sig.SetSleepValidate(to.SleepValidate)
	sig.SetUseMultisig(to.UseMultisig)
	sig.SetBlsMultiNew(to.BlsMultiNew)
}

func getStatsStringInternal(to types.TestOptions, msList []stats.MergedStats, saveToDisk bool,
	resultsFolder string, fileName string) string {

	var ret strings.Builder

	perProcMerged, mergedStats := stats.MergeStats(msList)
	// perProcNw, mergedNwStats := stats.MergeNwStats(int(to.MaxRounds), nwStatsList)
	ret.WriteString(fmt.Sprintf("\nMerged stats: %v, %v\n", mergedStats.String(), mergedStats.BasicNwStats.NwString()))
	ret.WriteString(fmt.Sprintf("\nMerged stats (per proc): %v, %v\n", perProcMerged.String(), perProcMerged.BasicNwStats.NwString()))

	if saveToDisk {
		for i, nxt := range []stats.MergedStats{perProcMerged, mergedStats} {
			resByt, err := json.MarshalIndent(nxt, "", "\t")
			if err != nil {
				panic(err)
			}
			// stats filename is stats_{perProc/merged}_{fileName}_{testID}_{nodes}_{constype}
			// results filename is results_{perProc/merged}_{fileName}_{testID}_{nodes}_{constype}
			var extraName string
			if i == 0 {
				extraName = "perProc"
			} else {
				extraName = "merged"
			}
			statFileName := fmt.Sprintf("stats_%v_%v_%v_%v_%d.txt", extraName, fileName, to.TestID, to.NumTotalProcs, to.ConsType)
			if err := ioutil.WriteFile(filepath.Join(resultsFolder, statFileName), resByt, os.ModePerm); err != nil {
				panic(err)
			}
		}
	}

	return ret.String()
}

// GetStatsString returns a human readable string of the statisitcs from a tests.
func GetStatsString(to types.TestOptions, statsList []stats.MergedStats, saveToDisk bool,
	resultsFolder string) string {
	numParticipants := to.NumTotalProcs - to.NumNonMembers
	var ret strings.Builder

	if to.NumNonMembers > 0 {
		ret.WriteString("Member statistics: \n")
		ret.WriteString(getStatsStringInternal(to, statsList[:numParticipants], saveToDisk, resultsFolder, "member"))

		ret.WriteString("Non-Member statistics: \n")
		ret.WriteString(getStatsStringInternal(to, statsList[numParticipants:], saveToDisk, resultsFolder, "nonMember"))
	}

	if to.NumFailProcs > 0 {
		ret.WriteString("Failure statistics: \n")
		ret.WriteString(getStatsStringInternal(to, statsList[:to.NumFailProcs], saveToDisk, resultsFolder, "failure"))

		if to.NumFailProcs < to.NumTotalProcs {
			ret.WriteString("Non-failure statistics: \n")
			ret.WriteString(getStatsStringInternal(to, statsList[to.NumFailProcs:numParticipants], saveToDisk, resultsFolder, "nonFailure"))
		}
	}

	if to.NumByz > 0 {
		ret.WriteString("Byzantine statistics: \n")
		ret.WriteString(getStatsStringInternal(to, statsList[:to.NumByz], saveToDisk, resultsFolder, "byz"))
	}

	ret.WriteString("Non-Byzantine statistics: \n")
	ret.WriteString(getStatsStringInternal(to, statsList[to.NumByz:], saveToDisk, resultsFolder, "nonByz"))

	ret.WriteString("Total statistics: \n")
	ret.WriteString(getStatsStringInternal(to, statsList, saveToDisk, resultsFolder, "total"))

	return ret.String()
}

const fixedSmSeed = 123456

var nonFixedSmSeed = time.Now().UTC().UnixNano()

// GenerateExtraParticipantRegisterState is called before the participant is registered,
// generating any additional test state.
func GenerateExtraParticipantRegisterState(to types.TestOptions, priv sig.Priv, pid int64) (ret []byte) {
	switch to.StateMachineType {
	case types.CurrencyTxProposer:
		priv = priv.GetBaseKey()
		initBlock := &currency.InitBlock{
			InitPubs:       []sig.Pub{priv.GetPub()},
			InitBalances:   []uint64{1000},
			InitUniqueName: currency.InitUniqueName}
		buf := bytes.NewBuffer(nil)
		_, err := initBlock.Encode(buf)
		if err != nil {
			panic(err)
		}
		ret = buf.Bytes()
	case types.DirectAssetProposer, types.ValueAssetProposer:
		priv = priv.GetBaseKey()
		var initialAssets []asset.AssetInterface
		switch to.StateMachineType {
		case types.DirectAssetProposer:
			initialAssets = []asset.AssetInterface{asset.CreateInitialDirectAsset(utils.Uint64ToBytes(uint64(pid)),
				asset.InitUniqueName)}
		case types.ValueAssetProposer:
			initialAssets = []asset.AssetInterface{asset.InternalNewValueAsset(1000, uint64(pid), asset.InitUniqueName)}
		}
		initBlock := asset.NewInitBlock([]sig.Pub{priv.GetPub()},
			initialAssets,
			string(asset.InitUniqueName))
		buf := bytes.NewBuffer(nil)
		_, err := initBlock.Encode(buf)
		if err != nil {
			panic(err)
		}
		ret = buf.Bytes()
	}
	return
}

// GenerateCausalStateMachine creates a state machine object given the by the test configuration.
func GenerateCausalStateMachine(to types.TestOptions, priv sig.Priv, proposer sig.Pub,
	extraParticipantInfo [][]byte, pid int64, testProc channelinterface.MainChannel,
	finishedChan chan channelinterface.ChannelCloseType, memberCheckerState consinterface.ConsStateInterface,
	generalConfig *generalconfig.GeneralConfig) (ret consinterface.CausalStateMachineInterface) {

	memberCount := to.NumTotalProcs - to.NumNonMembers

	switch to.StateMachineType {
	case types.CausalCounterProposer:
		ret = statemachine.NewCausalCounterProposerInfo(priv.GetPub(), proposer, to.GenRandBytes, config.InitRandBytes)
	case types.DirectAssetProposer:
		priv = priv.GetBaseKey()
		newAssetFunc := func() *asset.AssetTransfer {
			return asset.NewEmptyAssetTransfer(asset.NewDirectAsset, priv.NewSig, priv.GetPub().New)
		}

		ret = asset.NewAssetProposer(to.GenRandBytes, config.InitRandBytes, extraParticipantInfo, priv,
			asset.NewDirectAsset, newAssetFunc, asset.GenDirectAssetTransfer, asset.CheckDirectOutputFunc, asset.DirectSendToSelf,
			int(pid) < memberCount, memberCount, priv.GetPub().New, priv.NewSig)
	case types.ValueAssetProposer:
		priv = priv.GetBaseKey()
		newAssetFunc := func() *asset.AssetTransfer {
			return asset.NewEmptyAssetTransfer(asset.NewValueAsset, priv.NewSig, priv.GetPub().New)
		}
		ret = asset.NewAssetProposer(to.GenRandBytes, config.InitRandBytes, extraParticipantInfo, priv,
			asset.NewValueAsset, newAssetFunc, asset.GenValueAssetTransfer, asset.CheckValueOutputFunc, asset.ValueSendToSelf,
			int(pid) < memberCount, memberCount, priv.GetPub().New, priv.NewSig)

	default:
		panic(to.StateMachineType)
	}
	ret.Init(generalConfig, computeSMFinish(to), memberCheckerState, testProc, finishedChan)
	return ret
}

// GenerateStateMachine creates a state machine object given the by the test configuration.
func GenerateStateMachine(to types.TestOptions, priv sig.Priv,
	extraParticipantInfo [][]byte, pid int64, testProc channelinterface.MainChannel,
	finishedChan chan channelinterface.ChannelCloseType, newPubFunc func() sig.Pub, newSigFunc func() sig.Sig,
	generalConfig *generalconfig.GeneralConfig) (ret consinterface.StateMachineInterface) {

	smSeed := nonFixedSmSeed
	if to.UseFixedSeed {
		smSeed = fixedSmSeed
	}

	switch to.StateMachineType {
	case types.BinaryProposer:
		numParticipants := to.NumTotalProcs - to.NumNonMembers
		if numParticipants == to.NumTotalProcs && false { // We use a random permutation to choose the number of ones exactly
			ret = statemachine.NewBinCons1ProposalInfo(to.BinConsPercentOnes, smSeed, numParticipants, to.NumTotalProcs)
		} else { // We choose one count at each process randomly
			ret = statemachine.NewBinCons1ProposalInfo(to.BinConsPercentOnes, smSeed+pid, numParticipants, to.NumTotalProcs)
		}
	case types.BytesProposer:
		ret = statemachine.NewMvCons1ProposalInfo(to.GenRandBytes, config.InitRandBytes, to.MvProposalSizeBytes, smSeed+pid)
	case types.CounterProposer:
		ret = statemachine.NewCounterProposalInfo(to.GenRandBytes, config.InitRandBytes)
	case types.TestProposer:
		ret = statemachine.NewSimpleProposalInfo()
	case types.CounterTxProposer:
		ret = statemachine.NewSimpleTxProposer(to.GenRandBytes, config.InitRandBytes, smSeed+pid)
	case types.CurrencyTxProposer:

		var proposerKey []sig.Priv // if we are not a participant then we dont propose transactions
		// if pid < int64(to.NumTotalProcs-to.NumNonMembers) {
		if priv != nil {
			proposerKey = []sig.Priv{priv}
		}
		// }
		ret = currency.NewSimpleTxProposer(to.GenRandBytes, config.InitRandBytes, extraParticipantInfo,
			proposerKey, newPubFunc, newSigFunc)
	default:
		panic(to.StateMachineType)
	}
	ret.Init(generalConfig, computeSMFinish(to), types.ConsensusInt(to.AllowConcurrent), testProc, finishedChan)
	return
}

// GenerateForwardChecker creates a forward checker object given by the test configuration.
func GenerateForwardChecker(t assert.TestingT, pub sig.Pub, pubKeys sig.PubList, to types.TestOptions,
	parRegs ...network.PregInterface) consinterface.ForwardChecker {

	switch to.NetworkType {
	case types.P2p, types.RequestForwarder:
		var pubCons []sig.PubList
		if to.BufferForwarder {
			pstr, err := pub.GetRealPubBytes()
			if err != nil {
				panic(err)
			}
			parInfoList, err := parRegs[0].GetParticipants(sig.PubKeyStr(pstr))
			assert.Nil(t, err)

			for _, nxt := range parInfoList {
				pubCons = append(pubCons, network.ParInfoListToPubList(nxt, pubKeys))
			}
		}
		return forwardchecker.NewP2PForwarder(to.NetworkType == types.RequestForwarder,
			to.FanOut, to.BufferForwarder, pubCons)
	case types.AllToAll:
		return forwardchecker.NewAllToAllForwarder()
	case types.Random:
		return forwardchecker.NewRandomForwarder(to.BufferForwarder, to.FanOut)
	default:
		panic("invalid nw type")
	}
}

// CheckSpecialKeys creates a special member checker for the given configuration.
// Threshold keys generate a ThrshSigMemChecker.
// Multisignature keys generate a MultiSigMemChecker.
// Otherwise a NoSpecialMembers object is returned.
func CheckSpecialKeys(to types.TestOptions, priv sig.Priv, pubs sig.PubList,
	_ consinterface.MemberChecker) (smc consinterface.SpecialPubMemberChecker) {

	switch to.SigType {
	//case types.EDCOIN:
	// smc = memberchecker.NewThrshSigMemChecker(priv.(*ed.PartialPriv).GetSharedPub())
	case types.EC, types.ED, types.QSAFE, types.SCHNORR, types.EDCOIN:
		smc = memberchecker.NewNoSpecialMembers()
	case types.BLS:
		if to.UseMultisig && to.UsePubIndex {
			smc = memberchecker.NewMultiSigMemChecker(pubs[:to.NumTotalProcs-to.NumNonMembers],
				to.MCType != types.TrueMC)
		} else {
			smc = memberchecker.NewNoSpecialMembers()
		}
	case types.TBLS:
		smc = memberchecker.NewThrshSigMemChecker([]sig.Pub{priv.(sig.ThreshStateInterface).GetSharedPub()})
	case types.TBLSDual:
		p1 := priv.(sig.ThreshStateInterface).GetSharedPub()
		p2 := priv.(sig.SecondaryPriv).GetSecondaryPriv().(sig.ThreshStateInterface).GetSharedPub()
		smc = memberchecker.NewThrshSigMemChecker([]sig.Pub{p1, p2})
	case types.CoinDual:
		p2 := priv.(sig.SecondaryPriv).GetSecondaryPriv().(sig.ThreshStateInterface).GetSharedPub()
		smc = memberchecker.NewThrshSigMemChecker([]sig.Pub{p2})
	default:
		panic("unknown priv type")
	}
	return
}

// GenRandKey returns 32 random bytes.
func GenRandKey() (ret [32]byte) {
	_, err := crand.Read(ret[:])
	if err != nil {
		panic(err)
	}
	return
}

// GenLocalRand returns a Rand object using key and config.CsID if rand member type is set to LocalRandMember.
func GenLocalRand(to types.TestOptions, key [32]byte) *rand.Rand {
	if to.RndMemberType != types.LocalRandMember {
		return nil
	}
	_, err := rand.Read(key[:])
	if err != nil {
		panic(err)
	}
	var nonce [16]byte
	_ = copy(nonce[:], config.CsID)

	src, err := utils.NewSeededCTRSource(key, nonce)
	if err != nil {
		panic(err)
	}
	return rand.New(src)
}

// MakeMemberChecker generates a member checker for the given configuration initialized with pubKeys.
func MakeMemberChecker(to types.TestOptions, randKey [32]byte, priv sig.Priv, pubKeys []sig.Pub,
	gc *generalconfig.GeneralConfig) (memberChecker consinterface.MemberChecker) {

	switch to.MCType {
	case types.TrueMC:
		memberChecker = memberchecker.InitTrueMemberChecker(to.RotateCord, priv, gc)
	case types.CurrentTrueMC:
		memberChecker = memberchecker.InitCurrentTrueMemberChecker(GenLocalRand(to, randKey), to.RotateCord,
			to.RndMemberCount, to.RndMemberType, types.ConsensusInt(to.LocalRandMemberChange), priv, gc)
	case types.BinRotateMC:
		memberChecker = memberchecker.InitBinRotateMemberChecker(priv, gc)
	case types.CurrencyMC:
		memberChecker = currency.InitCurrencyMemberChecker(GenLocalRand(to, randKey), to.RotateCord,
			to.RndMemberCount, to.RndMemberType, types.ConsensusInt(to.LocalRandMemberChange), priv, gc)
	default:
		panic("invalid mc type")
	}
	var fixedCoord sig.Pub
	switch to.OrderingType {
	case types.Causal:
		fixedCoord = pubKeys[0]
	}
	// The non-members are at the end of the list of pub keys
	numParticipants := to.NumTotalProcs - to.NumNonMembers
	memberChecker.AddPubKeys(fixedCoord, pubKeys[:numParticipants], pubKeys[numParticipants:], config.InitRandBytes)
	return
}

// MakePrivBlsS create a private threshold key given that a BlsShared object has already been created.
func MakePrivBlsS(to types.TestOptions, thrsh int, idx sig.PubKeyIndex, blsShared *bls.BlsShared) (p sig.Priv, err error) {

	blsThresh := bls.NewBlsThrsh(to.NumTotalProcs, thrsh,
		idx, blsShared.PriScalars[idx], blsShared.PubPoints[idx], blsShared.SharedPub)
	p, err = bls.NewBlsPartPriv(blsThresh)
	return
}

// MakeUnusedKey generates a private key for a test. The key should not be used directly,
// instead used an input to functions to that only use the new functions of the interfaces
// to generate new key objects.
func MakeUnusedKey(i sig.PubKeyIndex, to types.TestOptions) (p sig.Priv, err error) {
	intFunc, _ := bitid.GetBitIDFuncs(to.SigBitIDType)
	newBlsFunc := func() (sig.Priv, error) {
		return bls.NewBlspriv(intFunc)
	}
	switch to.SigType {
	case types.TBLS:
		p, err = bls.NewBlspriv(intFunc)
	case types.TBLSDual:
		coinType := types.NormalSignature
		if types.UseTp1CoinThresh(to) {
			coinType = types.SecondarySignature
		}
		p, err = dual.NewDualpriv(newBlsFunc, newBlsFunc, coinType, types.SecondarySignature)
	case types.CoinDual:
		p, err = dual.NewDualpriv(ec.NewEcpriv, newBlsFunc, types.SecondarySignature, types.NormalSignature)
	case types.EDCOIN:
		p, err = ed.NewSchnorrpriv() // we will use this to create the EDCOIN later
	default:
		p, err = MakeKey(i, to)
	}
	return
}

// MakeKey generates a private key for the given configuration.
func MakeKey(i sig.PubKeyIndex, to types.TestOptions) (p sig.Priv, err error) {
	switch to.SigType {
	case types.EC:
		if to.SleepCrypto {
			p, err = sleep.NewECPriv(i)
		} else {
			p, err = ec.NewEcpriv()
		}
	case types.BLS:
		intFunc, _ := bitid.GetBitIDFuncs(to.SigBitIDType)
		if to.SleepCrypto {
			p, err = sleep.NewBLSPriv(i, intFunc)
		} else {
			p, err = bls.NewBlspriv(intFunc)
		}
	case types.QSAFE:
		if to.SleepCrypto {
			p, err = sleep.NewQsafePriv(i)
		} else {
			p, err = qsafe.NewQsafePriv()
		}
	case types.ED:
		if to.SleepCrypto {
			p, err = sleep.NewEDPriv(i)
		} else {
			p, err = ed.NewEdpriv()
		}
	case types.SCHNORR:
		if to.SleepCrypto {
			p, err = sleep.NewSchnorrPriv(i)
		} else {
			p, err = ed.NewSchnorrpriv()
		}
	default:
		err = fmt.Errorf("invalid sig type")
	}
	return
}

func GetTBLSThresh(to types.TestOptions) int {
	var thrsh int
	if to.CoinType == types.NoCoinType {
		thrsh, _ = sig.GetDSSThresh(to)
	} else {
		thrsh = sig.GetCoinThresh(to)
	}
	return thrsh
}

// MakeKeys generates a list of private and public keys for the configuration.
// The slices are sorted by the public key.
func MakeKeys(to types.TestOptions) ([]sig.Priv, []sig.Pub) {
	// the number of participants participating in consensus
	// the remaining keys are the keys of replicas but not members.
	numMembers := to.NumTotalProcs - to.NumNonMembers
	privKeys := make(sig.PrivList, to.NumTotalProcs)
	pubKeys := make(sig.PubList, to.NumTotalProcs)

	var privFunc func(i sig.PubKeyIndex) sig.Priv
	switch to.SigType {
	case types.EC:
		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewECPriv(i)
				utils.PanicNonNil(err)
			} else {
				p, err = ec.NewEcpriv()
				utils.PanicNonNil(err)
			}
			return
		}
	case types.QSAFE:
		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewQsafePriv(i)
				utils.PanicNonNil(err)
			} else {
				p, err = qsafe.NewQsafePriv()
				utils.PanicNonNil(err)
			}
			return
		}
	case types.BLS:
		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			intFunc, _ := bitid.GetBitIDFuncs(to.SigBitIDType)
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewBLSPriv(i, intFunc)
				utils.PanicNonNil(err)
			} else {
				p, err = bls.NewBlspriv(intFunc)
				utils.PanicNonNil(err)
			}
			return
		}
	case types.ED:
		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewEDPriv(i)
				utils.PanicNonNil(err)
			} else {
				p, err = ed.NewEdpriv()
				utils.PanicNonNil(err)
			}
			return
		}
	case types.SCHNORR:
		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewSchnorrPriv(i)
				utils.PanicNonNil(err)
			} else {
				p, err = ed.NewSchnorrpriv()
				utils.PanicNonNil(err)
			}
			return
		}
	case types.TBLS:
		thrshn := numMembers
		thrsh := GetTBLSThresh(to)
		blsShared := bls.NewBlsShared(thrshn, thrsh)

		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewTBLSPriv(thrshn, thrsh, i)
				utils.PanicNonNil(err)
				return
			}
			if int(i) < thrshn {
				p, err = MakePrivBlsS(to, thrsh, sig.PubKeyIndex(i), blsShared)
				utils.PanicNonNil(err)
			} else {
				// nonMembers, just generate some keys, they won't be used as members
				blsThresh := bls.NewNonMemberBlsThrsh(thrshn, thrsh, sig.PubKeyIndex(i), blsShared.SharedPub)
				p, err = bls.NewBlsPartPriv(blsThresh)
				utils.PanicNonNil(err)
			}
			return
		}
	case types.TBLSDual:
		thrshn := numMembers
		blsSharedPrimary, blsSharedSecondary := GenTBLSDualDSSThresh(to)

		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewTBLSDualPriv(i, to)
				utils.PanicNonNil(err)
				return p
			}
			if int(i) < thrshn {
				p = GenTBLSDualThreshPriv(i, blsSharedPrimary, blsSharedSecondary, to)
			} else {
				// nonMembers, just generate some keys, they won't be used as members
				p = GenTBLSDualThreshPrivNonMember(i, blsSharedPrimary, blsSharedSecondary, to)
			}
			return
		}
	case types.CoinDual:
		thrshn := numMembers
		// coinType := types.NormalSignature
		thrsh := sig.GetCoinThresh(to)
		blsSharedSecondary := bls.NewBlsShared(thrshn, thrsh)

		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewCoinDualPriv(i, to)
				return
			}

			var primary, secondary sig.Priv
			if int(i) < thrshn {
				primary, err = ec.NewEcpriv()
				utils.PanicNonNil(err)
				secondary, err = MakePrivBlsS(to, thrsh, sig.PubKeyIndex(i), blsSharedSecondary)
				utils.PanicNonNil(err)
				p, err = dual.NewDualprivCustomThresh(primary, secondary, types.SecondarySignature,
					types.NormalSignature)
				utils.PanicNonNil(err)
			} else {
				// nonMembers, just generate some keys, they won't be used as members
				primary, err = ec.NewEcpriv()
				utils.PanicNonNil(err)
				blsThreshSecondary := bls.NewNonMemberBlsThrsh(thrshn, thrsh, sig.PubKeyIndex(i), blsSharedSecondary.SharedPub)
				secondary, err = bls.NewBlsPartPriv(blsThreshSecondary)
				utils.PanicNonNil(err)
				p, err = dual.NewDualprivCustomThresh(primary, secondary, types.SecondarySignature,
					types.NormalSignature)
				utils.PanicNonNil(err)
			}
			return
		}

	case types.EDCOIN:
		thrsh := sig.GetCoinThresh(to)
		dssShared := ed.NewCoinShared(to.NumTotalProcs, to.NumNonMembers, thrsh)

		privFunc = func(i sig.PubKeyIndex) (p sig.Priv) {
			var err error
			if to.SleepCrypto {
				p, err = sleep.NewEDCoinPriv(i, to)
				return
			}
			edThresh := ed.NewEdThresh(sig.PubKeyIndex(i), dssShared)
			p, err = ed.NewEdPartPriv(edThresh)
			utils.PanicNonNil(err)
			return
		}
	default:
		panic("invalid sig type")
	}

	for i := 0; i < to.NumTotalProcs; i++ {
		privKeys[i] = privFunc(sig.PubKeyIndex(i))
	}
	privKeys.SetIndices() // so we set the indicies
	for i := 0; i < to.NumTotalProcs; i++ {
		pubKeys[i] = privKeys[i].GetPub()
	}
	return privKeys, pubKeys
}

func GenTBLSDualThreshPrivNonMember(i sig.PubKeyIndex, blsSharedPrimary, blsSharedSecondary *bls.BlsShared,
	to types.TestOptions) sig.Priv {

	coinType := types.NormalSignature
	if types.UseTp1CoinThresh(to) {
		coinType = types.SecondarySignature
	}
	thrshPrimary, thrshSecondary := sig.GetDSSThresh(to)
	thrshn := to.NumTotalProcs - to.NumNonMembers
	blsThreshPrimary := bls.NewNonMemberBlsThrsh(thrshn, thrshPrimary, i, blsSharedPrimary.SharedPub)
	blsThreshSecondary := bls.NewNonMemberBlsThrsh(thrshn, thrshSecondary, i, blsSharedSecondary.SharedPub)
	var primary, secondary sig.Priv
	var err error
	if primary, err = bls.NewBlsPartPriv(blsThreshPrimary); err != nil {
		panic(err)
	}
	if secondary, err = bls.NewBlsPartPriv(blsThreshSecondary); err != nil {
		panic(err)
	}
	var ret sig.Priv
	if ret, err = dual.NewDualprivCustomThresh(primary, secondary, coinType, types.SecondarySignature); err != nil {
		panic(err)
	}
	return ret
}

func GenTBLSDualThreshPriv(i sig.PubKeyIndex, blsSharedPrimary, blsSharedSecondary *bls.BlsShared,
	to types.TestOptions) sig.Priv {

	coinType := types.NormalSignature
	if types.UseTp1CoinThresh(to) {
		coinType = types.SecondarySignature
	}
	thrshPrimary, thrshSecondary := sig.GetDSSThresh(to)
	var err error
	var primary, secondary sig.Priv
	if primary, err = MakePrivBlsS(to, thrshPrimary, i, blsSharedPrimary); err != nil {
		panic(err)
	}
	if secondary, err = MakePrivBlsS(to, thrshSecondary, i, blsSharedSecondary); err != nil {
		panic(err)
	}
	var ret sig.Priv
	if ret, err = dual.NewDualprivCustomThresh(primary, secondary, coinType, types.SecondarySignature); err != nil {
		panic(err)
	}
	return ret
}

func GenTBLSDualDSSThresh(to types.TestOptions) (blsSharedPrimary, blsSharedSecondary *bls.BlsShared) {
	numMembers := to.NumTotalProcs - to.NumNonMembers
	thrshn := numMembers
	thrshPrimary, thrshSecondary := sig.GetDSSThresh(to)
	blsSharedPrimary = bls.NewBlsShared(thrshn, thrshPrimary)
	blsSharedSecondary = bls.NewBlsShared(thrshn, thrshSecondary)
	return
}

// computeLastIndex returns the largest consensus instance a process should participate in.
func computeLasIndex(to types.TestOptions, initItem consinterface.ConsItem) types.ConsensusInt {
	return types.ConsensusInt(to.MaxRounds) + initItem.NeedsConcurrent() - 1 + config.WarmUpInstances
}

// computeSMFinish returns the index a normal state machine should send on the done channel.
func computeSMFinish(to types.TestOptions) types.ConsensusInt {
	return types.ConsensusInt(to.MaxRounds) + config.WarmUpInstances
}

// computeFailRound returns the index a fail process should send on the done channel.
func computeFailRound(to types.TestOptions) types.ConsensusInt {
	return types.ConsensusInt(to.FailRounds) + config.WarmUpInstances - 1
}

func SetInitIndex(initSM consinterface.CausalStateMachineInterface) {
	// These items indicate our initial state
	initIdx := initSM.GetIndex()
	initParentHash := initIdx.Index.(types.ConsensusHash)

	types.InitIndexHash = initParentHash
}

func GenerateConsState(t assert.TestingT, to types.TestOptions, priv sig.Priv, randKey [32]byte, proposer sig.Pub,
	pubKeys sig.PubList, i int, initItem consinterface.ConsItem, ds storage.StoreInterface, stats stats.StatsInterface,
	retExtraParRegInfo [][]byte, testProc channelinterface.MainChannel, finishedChan chan channelinterface.ChannelCloseType,
	generalConfig *generalconfig.GeneralConfig, broadcastFunc consinterface.ByzBroadcastFunc,
	isFailure bool, parRegs ...network.PregInterface) (ret ConsStateInterface, memberCheckerState consinterface.ConsStateInterface) {

	// Let the stats know they can be used again
	stats.Restart()

	// Generate the state machine
	var causalSM consinterface.CausalStateMachineInterface
	var sm consinterface.StateMachineInterface
	switch to.OrderingType {
	case types.Total:
		sm = GenerateStateMachine(to, priv.GetBaseKey(), retExtraParRegInfo, int64(i), testProc, finishedChan,
			priv.GetBaseKey().GetPub().New, priv.GetBaseKey().NewSig, generalConfig)
		if isFailure {
			sm.FailAfter(computeFailRound(to))
		}
	case types.Causal:
		causalSM = GenerateCausalStateMachine(to, priv, proposer, retExtraParRegInfo, int64(i), testProc, finishedChan,
			memberCheckerState, generalConfig)
		if isFailure {
			causalSM.FailAfter(computeFailRound(to))
		}
	}

	// Generate the member checker
	memberChecker := MakeMemberChecker(to, randKey, priv, pubKeys, generalConfig)
	specialMemberChecker := CheckSpecialKeys(to, priv, pubKeys, memberChecker)
	// Generate the message state
	messageState := initItem.GenerateMessageState(generalConfig)
	// Generate the cons state
	forwardChecker := GenerateForwardChecker(t, priv.GetPub(), pubKeys, to, parRegs...)
	memberCheckerState = GenerateConsStateInterface(to, initItem, memberChecker, specialMemberChecker,
		messageState, forwardChecker, ds, broadcastFunc, generalConfig)

	switch to.RndMemberType { // TODO cleanup
	case types.LocalRandMember:
		testProc.(*csnet.DualNetMainChannel).SetMemberCheckerState(memberCheckerState)
	default:
		testProc.(*csnet.NetMainChannel).SetMemberCheckerState(memberCheckerState)
	}
	memberCheckerState.SetMainChannel(testProc)

	switch to.OrderingType {
	case types.Total:
		ret = NewConsState(initItem, generalConfig, memberCheckerState.(*consinterface.ConsInterfaceState),
			ds, computeLasIndex(to, initItem),
			types.ConsensusInt(to.AllowConcurrent), sm, testProc)
		return
	case types.Causal:
		ret = NewCausalConsState(initItem, generalConfig, memberCheckerState.(*consinterface.CausalConsInterfaceState), ds,
			causalSM, testProc)
		return
	default:
		panic(to.OrderingType)
	}

}

func GenerateConsStateInterface(to types.TestOptions, initItem consinterface.ConsItem,
	memberChecker consinterface.MemberChecker, specialMemberChecker consinterface.SpecialPubMemberChecker,
	messageState consinterface.MessageState, forwardChecker consinterface.ForwardChecker,
	storage storage.StoreInterface, broadcastFunc consinterface.ByzBroadcastFunc,
	generalConfig *generalconfig.GeneralConfig) (ret consinterface.ConsStateInterface) {

	switch to.OrderingType {
	case types.Total:
		ret = consinterface.NewConsInterfaceState(initItem, memberChecker,
			specialMemberChecker, messageState, forwardChecker, types.ConsensusInt(to.AllowConcurrent), to.GenRandBytes,
			generalConfig.Priv.GetPub().New(), broadcastFunc, generalConfig)
	case types.Causal:
		ret = consinterface.NewCausalConsInterfaceState(initItem, memberChecker,
			specialMemberChecker, messageState, forwardChecker, storage, to.GenRandBytes,
			generalConfig.Priv.GetPub().New(), broadcastFunc, generalConfig)
	default:
		panic(to.OrderingType)
	}
	return
}

func GenerateParticipantRegisters(to types.TestOptions) (ret []network.PregInterface) {
	// the network propagation configuration
	connType := network.NetworkPropagationSetup{
		AdditionalP2PNetworks: to.AdditionalP2PNetworks,
		NPT:                   to.NetworkType,
		FanOut:                to.FanOut}
	ret = append(ret, network.NewParReg(connType, to.NumTotalProcs)) // the participant register
	// request forwarder uses a second network for proposal messages
	if to.NetworkType == types.RequestForwarder {
		connType := network.NetworkPropagationSetup{
			NPT:    types.P2p,
			FanOut: to.FanOut}
		ret = append(ret, network.NewParReg(connType, to.NumTotalProcs)) // the participant register
	}
	return ret
}

func RegisterOtherNodes(t assert.TestingT, to types.TestOptions, testProc channelinterface.MainChannel,
	genPub func(sig.PubKeyBytes) sig.Pub, parRegs ...network.PregInterface) {

	allParticipants, err := parRegs[0].GetAllParticipants()
	if len(allParticipants) != to.NumTotalProcs {
		panic(fmt.Sprint(allParticipants, to.NumTotalProcs))
	}
	assert.Nil(t, err)
	for _, parInfo := range allParticipants {
		nodeInfo := channelinterface.NetNodeInfo{
			AddrList: parInfo.ConInfo,
			Pub:      genPub(sig.PubKeyBytes(parInfo.Pub))}
		testProc.AddExternalNode(nodeInfo)
	}

	switch to.NetworkType {
	case types.RequestForwarder:
		allParticipants, err := parRegs[1].GetAllParticipants()
		if len(allParticipants) != to.NumTotalProcs {
			panic(fmt.Sprint(allParticipants, to.NumTotalProcs))
		}
		assert.Nil(t, err)
		for _, parInfo := range allParticipants {
			nodeInfo := channelinterface.NetNodeInfo{
				AddrList: parInfo.ConInfo,
				Pub:      genPub(sig.PubKeyBytes(parInfo.Pub))}
			testProc.(*csnet.DualNetMainChannel).Secondary.AddExternalNode(nodeInfo)
		}
	}
}

func MakeConnections(t assert.TestingT, to types.TestOptions, testProc channelinterface.MainChannel,
	_ consinterface.ConsStateInterface, numCons []int,
	gc *generalconfig.GeneralConfig, pubKeys sig.PubList, parRegs ...network.PregInterface) {

	testProc.StartMsgProcessThreads()
	pstr, err := gc.Priv.GetPub().GetRealPubBytes()

	// main network
	parInfoList, err := parRegs[0].GetParticipants(sig.PubKeyStr(pstr))
	assert.Nil(t, err)
	var connectionPubs []sig.Pub
	for _, parInfo := range parInfoList {
		connectionPubs = append(connectionPubs, network.ParInfoListToPubList(parInfo, pubKeys)...)
	}
	connectionPubs = sig.RemoveDuplicatePubs(connectionPubs)

	switch to.NetworkType {
	case types.RequestForwarder:
		numCons[0] = to.RndMemberCount - 1 // -1 for yourself
		if len(parInfoList) != 0 {
			panic("connections should be made through the absRandLocal member checker")
		}
	default:
		numCons[0] = len(connectionPubs)
	}

	errs := testProc.MakeConnections(connectionPubs)
	if len(errs) > 0 {
		panic(errs)
	}

	// secondary network
	switch to.NetworkType {
	case types.RequestForwarder:
		parInfoList, err := parRegs[1].GetParticipants(sig.PubKeyStr(pstr))
		assert.Nil(t, err)
		var connectionPubs []sig.Pub
		for _, parInfo := range parInfoList {
			connectionPubs = append(connectionPubs, network.ParInfoListToPubList(parInfo, pubKeys)...)
		}
		connectionPubs = sig.RemoveDuplicatePubs(connectionPubs)

		numCons[1] = len(connectionPubs)
		errs := testProc.(*csnet.DualNetMainChannel).Secondary.MakeConnections(connectionPubs)
		if len(errs) > 0 {
			panic(errs)
		}
	}
}

func MakeMainChannel(to types.TestOptions, priv sig.Priv, initItem consinterface.ConsItem, stats stats.NwStatsInterface,
	_ *generalconfig.GeneralConfig, nodeConInfo []channelinterface.NetNodeInfo) (ret channelinterface.MainChannel) {

	bts := channelinterface.NewSimpleBehaviorTracker()
	switch to.ConnectionType {
	case types.TCP, types.UDP:
		switch to.NetworkType {
		case types.RequestForwarder:
			ret = csnet.NewDualNetMainChannel(priv, nodeConInfo, to.NumTotalProcs-1, to.ConnectionType,
				to.NumMsgProcessThreads, initItem, bts, to.EncryptChannels, to.MsgDropPercent, stats)
		default:
			ret, _ = csnet.NewNetMainChannel(priv, nodeConInfo[0], to.NumTotalProcs-1, to.ConnectionType,
				to.NumMsgProcessThreads, initItem, bts, to.EncryptChannels, to.MsgDropPercent, stats)
		}
	default:
		panic("Unknown connection type")
	}
	return
}

func GenerateMainChannel(t assert.TestingT, to types.TestOptions,
	initItem consinterface.ConsItem,
	gc *generalconfig.GeneralConfig,
	updateConInfo func(channelinterface.NetNodeInfo),
	stats stats.NwStatsInterface, extraParRegInfo []byte,
	parRegs ...network.PregInterface) (ret channelinterface.MainChannel, retConInfo []channelinterface.NetNodeInfo) {

	var nodeConInfo []channelinterface.NetNodeInfo
	switch to.ConnectionType {
	case types.TCP, types.UDP:
		switch to.NetworkType {
		case types.RequestForwarder:
			nodeConInfo = []channelinterface.NetNodeInfo{
				{AddrList: channelinterface.NewConInfoList(to.ConnectionType),
					Pub: gc.Priv.GetPub()},
				{AddrList: channelinterface.NewConInfoList(to.ConnectionType),
					Pub: gc.Priv.GetPub()},
			}
		default:
			conInfo := channelinterface.NewConInfoList(to.ConnectionType)
			nodeConInfo = []channelinterface.NetNodeInfo{{
				AddrList: conInfo,
				Pub:      gc.Priv.GetPub()}}
		}
	default:
		panic("Unknown connection type")
	}
	ret = MakeMainChannel(to, gc.Priv, initItem, stats, gc, nodeConInfo)

	// register the address
	// Register the ports with the participant register
	logging.Info("Registering participation")
	pstr, err := gc.Priv.GetPub().GetRealPubBytes()
	assert.Nil(t, err)

	conInfo := ret.GetLocalNodeConnectionInfo()
	if updateConInfo != nil {
		updateConInfo(conInfo)
	}
	retConInfo = append(retConInfo, conInfo)

	err = parRegs[0].RegisterParticipant(&network.ParticipantInfo{
		Pub:           sig.PubKeyStr(pstr),
		RegisterCount: gc.TestIndex,
		ConInfo:       conInfo.AddrList,
		ExtraInfo:     extraParRegInfo,
	})
	assert.Nil(t, err)

	switch to.NetworkType {
	// If we are using a request forwarder then we will have a second participant register for the proposal network
	case types.RequestForwarder:
		// Register the ports with the participant register

		conInfo := ret.(*csnet.DualNetMainChannel).Secondary.GetLocalNodeConnectionInfo()
		if updateConInfo != nil {
			updateConInfo(conInfo)
		}
		retConInfo = append(retConInfo, conInfo)

		logging.Info("Registering secondary address")
		err = parRegs[1].RegisterParticipant(&network.ParticipantInfo{
			Pub:           sig.PubKeyStr(pstr),
			RegisterCount: gc.TestIndex,
			ConInfo:       conInfo.AddrList,
			ExtraInfo:     extraParRegInfo,
		})
		assert.Nil(t, err)
	}

	return
}

func UpdateConnectionsAfterFailure(to types.TestOptions, gc *generalconfig.GeneralConfig,
	_ channelinterface.MainChannel,
	testProcs []channelinterface.MainChannel) {

	// Now update the other nodes with the new connection info
	conInfo := testProcs[gc.TestIndex].GetLocalNodeConnectionInfo()
	for j := 0; j < to.NumTotalProcs; j++ {
		if j == gc.TestIndex {
			continue
		}
		// go func(idx int) {testProcs[idx].AddExternalNode(conInfo)} (j)
		testProcs[j].AddExternalNode(conInfo)
	}
	switch to.NetworkType {
	case types.RequestForwarder:
		conInfo := testProcs[gc.TestIndex].(*csnet.DualNetMainChannel).Secondary.GetLocalNodeConnectionInfo()
		for j := 0; j < len(testProcs); j++ {
			if j != gc.TestIndex {
				testProcs[j].(*csnet.DualNetMainChannel).Secondary.AddExternalNode(conInfo)
			}
		}
	}

	/*	// Update the other nodes address of the failed node
		for j := 0; j < to.NumTotalProcs; j++ {
			if j == i {
				continue
			}
			switch to.NetworkType {
			case types.RequestForwarder: // Connections are made within the absRandLocal member checker
			default:
				errs := testProcs[j].MakeConnections([]sig.Pub{conInfo.Pub})
				if len(errs) > 0 {
					panic(errs)
				}
			}
		}
	*/

}

func WaitConnections(t assert.TestingT, to types.TestOptions, testProc channelinterface.MainChannel, numCons []int) {
	err := testProc.WaitUntilAtLeastSendCons(numCons[0])
	assert.Nil(t, err)
	switch to.NetworkType {
	case types.RequestForwarder:
		err := testProc.(*csnet.DualNetMainChannel).Secondary.WaitUntilAtLeastSendCons(numCons[1])
		assert.Nil(t, err)
	}
}

// RunConsType runs a test for the given configuration and inputs.
func RunConsType(initItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	options ConfigOptions,
	to types.TestOptions,
	t assert.TestingT) {

	if t == nil {
		panic(t)
	}

	if config.AllowConcurrentTests && to.TestID == 0 {
		to.TestID = rand.Uint64()
		logging.Printf("Set test ID to %v\n", to.TestID)
	}

	// Set test options
	SetTestConfigOptions(&to, true)

	// Check if the generalconfig is valid
	var err error
	to, err = to.CheckValid(to.ConsType, options.GetIsMV())
	if err != nil {
		panic(err)
	}

	logging.Printf("\nRunning test config:\n%s\n", to)
	// t.Logf("\nRunning generalconfig: %s\n", to)
	// make the private keys
	privKeys, pubKeys := MakeKeys(to)
	// randItem := rand.New(rand.NewSource(time.Now().UnixNano()))

	statsList := make([]stats.StatsInterface, to.NumTotalProcs)
	randKeys := make([][32]byte, to.NumTotalProcs)
	fileNames := make([]string, to.NumTotalProcs)
	for i := 0; i < to.NumTotalProcs; i++ {
		fileNames[i] = fmt.Sprint(to.TestID)
		statsList[i] = stats.GetStatsObject(initItem.GetConsType(), to.EncryptChannels)
		randKeys[i] = GenRandKey()
	}
	// Only extra info for members
	// non-members will have nil extra info
	extraParRegInfo := make([][]byte, to.NumTotalProcs)
	retExtraParRegInfo := make([][]byte, to.NumTotalProcs)
	for i := 0; i < to.NumTotalProcs; i++ { //-to.NumNonMembers; i++ {
		extraParRegInfo[i] = GenerateExtraParticipantRegisterState(to, privKeys[i], int64(i))
	}
	generalConfigs := make([]*generalconfig.GeneralConfig, to.NumTotalProcs)

	for i := 0; i < to.NumTotalProcs; i++ {
		// cons item
		preHeader := make([]messages.MsgHeader, 0)
		eis := (generalconfig.ExtraInitState)(ConsInitState{IncludeProofs: to.IncludeProofs,
			Id: uint32(i), N: uint32(to.NumTotalProcs)})

		// Remaining state
		generalConfigs[i] = consinterface.CreateGeneralConfig(i, eis, statsList[i], to,
			preHeader, privKeys[i])
	}

	fmt.Printf("Running cons %T with state machine %v\n", initItem, to.StateMachineType)

	// Storage objects.
	ds := make([]storage.StoreInterface, to.NumTotalProcs)
	// Open storage
	for i := 0; i < to.NumTotalProcs; i++ {
		ds[i] = OpenTestStore(to.StorageBuffer, to.StorageType, GetStoreType(to), i, fileNames[i], true)
	}

	testProcs := make([]channelinterface.MainChannel, to.NumTotalProcs) // the network channels
	parRegs := GenerateParticipantRegisters(to)

	logging.Info("Opening network ports")
	// Create the network ports at each node.
	for i := 0; i < to.NumTotalProcs; i++ {
		testProcs[i], _ = GenerateMainChannel(t, to, initItem,
			generalConfigs[i], nil, statsList[i], extraParRegInfo[i], parRegs...)
	}

	logging.Info("Adding connections")

	// For each node add the connection information to the nodes it will connect to.
	getPubFunc := func(pb sig.PubKeyBytes) sig.Pub {
		return network.GetPubFromList(pb, pubKeys)
	}
	switch to.ConnectionType {
	case types.TCP, types.UDP:
		// First we let each node know all the other node's addresses
		for i := 0; i < to.NumTotalProcs; i++ {
			RegisterOtherNodes(t, to, testProcs[i], getPubFunc, parRegs...)
		}
	default:
		panic("invalid connection type")
	}

	// Get extra participant info
	allParticipants, err := parRegs[0].GetAllParticipants()
	if len(allParticipants) != to.NumTotalProcs {
		panic(fmt.Sprint(allParticipants, to.NumTotalProcs))
	}
	for i, par := range allParticipants {
		retExtraParRegInfo[i] = par.ExtraInfo
	}

	// We need to compute the value of the initial hash // TODO clean this up
	if to.OrderingType == types.Causal {
		initSM := GenerateCausalStateMachine(to, privKeys[0], pubKeys[0], retExtraParRegInfo, int64(0),
			testProcs[0], nil, nil, generalConfigs[0])
		SetInitIndex(initSM)
	}

	// Will wait for procs on this chan to finish
	finishedChan := make(chan channelinterface.ChannelCloseType, to.NumTotalProcs)

	// Will wait for failed procs on this chan to finish
	finishedFailChan := make(chan channelinterface.ChannelCloseType, to.NumFailProcs)

	// Consensus state objects
	consStates := make([]ConsStateInterface, to.NumTotalProcs)
	memberCheckerStates := make([]consinterface.ConsStateInterface, to.NumTotalProcs) // the member cheker state objects

	// Start the processes that will fail in the middle of the test, if any (as given by the options)
	logging.Info("Starting faulty processes")
	for i := 0; i < to.NumFailProcs; i++ {
		consStates[i], memberCheckerStates[i] = GenerateConsState(t, to, privKeys[i], randKeys[i], pubKeys[0], pubKeys,
			i, initItem, ds[i], statsList[i], retExtraParRegInfo, testProcs[i], finishedFailChan, generalConfigs[i],
			broadcastFunc, true, parRegs...)
	}

	// Start the remaining processes that do not fail.
	logging.Info("Starting normal processes")
	for i := to.NumFailProcs; i < to.NumTotalProcs; i++ {
		consStates[i], memberCheckerStates[i] = GenerateConsState(t, to, privKeys[i], randKeys[i], pubKeys[0], pubKeys,
			i, initItem, ds[i], statsList[i], retExtraParRegInfo, testProcs[i], finishedChan, generalConfigs[i],
			broadcastFunc, false, parRegs...)
	}

	// Connect the nodes to eachother
	numCons := make([][]int, to.NumTotalProcs)
	for i := 0; i < to.NumTotalProcs; i++ {
		switch to.NetworkType {
		case types.RequestForwarder:
			numCons[i] = make([]int, 2)
		default:
			numCons[i] = make([]int, 1)
		}
	}
	for i := 0; i < to.NumTotalProcs; i++ {
		MakeConnections(t, to, testProcs[i], memberCheckerStates[i],
			numCons[i], generalConfigs[i], pubKeys, parRegs...)
	}
	// Wait until all connected
	for i := 0; i < to.NumTotalProcs; i++ {
		WaitConnections(t, to, testProcs[i], numCons[i])
	}

	// Start the procs that will fail
	failFinishID := make(chan int, to.NumFailProcs)
	for i := 0; i < to.NumFailProcs; i++ {
		// Run the main loop
		go func(i int) {
			RunMainLoop(consStates[i], testProcs[i])
			logging.Info("finished index", i)
			time.Sleep(time.Duration(to.FailDuration) * time.Millisecond)
			failFinishID <- i
		}(i)
	}

	// Start the normal procs
	var wgNormal sync.WaitGroup
	for i := to.NumFailProcs; i < to.NumTotalProcs; i++ {
		// Run the main loop
		go func(i int) {
			RunMainLoop(consStates[i], testProcs[i])
			wgNormal.Done()
		}(i)
		wgNormal.Add(1)
	}

	logging.Info("Waiting for failures to finish")
	// wait for the failures to finish
	for i := 0; i < to.NumFailProcs; i++ {
		<-finishedFailChan
	}
	logging.Info("Shutting down failures connections")
	// Tell them to shut down
	for i := 0; i < to.NumFailProcs; i++ {
		testProcs[i].Close()
	}

	// Restart the failures
	logging.Info("Restarting failures")
	var wgFailNew sync.WaitGroup
	for l := 0; l < to.NumFailProcs; l++ {
		i := <-failFinishID
		// close the disk store
		err = ds[i].Close()
		assert.Nil(t, err)

		// Open the storage for the restarted nodes
		ds[i] = OpenTestStore(to.StorageBuffer, to.StorageType, GetStoreType(to), i, fileNames[i], to.ClearDiskOnRestart)
		// Clear the storage for any decided values past the fail round
		UpdateStorageAfterFail(to, ds[i])

		logging.Info("Restarting failure", i)
		switch to.ConnectionType {
		// Setup the network for the restarted processes
		case types.UDP, types.TCP:
		default:
			panic("invalid connection type")
		}

		oldTestProc := testProcs[i]
		testProcs[i], _ = GenerateMainChannel(t, to, initItem, generalConfigs[i], nil, statsList[i],
			extraParRegInfo[i], parRegs...)

		// Register the addresses of the other nodes
		RegisterOtherNodes(t, to, testProcs[i], getPubFunc, parRegs...)
		// For all the other processes update their connection information with the new addresses.
		UpdateConnectionsAfterFailure(to, generalConfigs[i], oldTestProc, testProcs)

		consStates[i], memberCheckerStates[i] = GenerateConsState(t, to, privKeys[i], randKeys[i], pubKeys[0], pubKeys,
			i, initItem, ds[i], statsList[i], retExtraParRegInfo, testProcs[i], finishedChan, generalConfigs[i],
			broadcastFunc, false, parRegs...)

		// make the connections to the other nodes
		MakeConnections(t, to, testProcs[i], memberCheckerStates[i],
			numCons[i], generalConfigs[i], pubKeys, parRegs...)

		logging.Info("Done starting failure", i)

		// Restart the failed processes main loop
		go func(k int) {
			RunMainLoop(consStates[k], testProcs[k])
			wgFailNew.Done()
		}(i)
		wgFailNew.Add(1)
	}
	logging.Info("ALL failures restarted!")

	logging.Info("Waiting for all to finish")
	// Wait for all to finish
	for i := 0; i < to.NumTotalProcs; i++ {
		id := <-finishedChan
		logging.Info("thread id finished: ", id)
		// testProcs[id].Close()
	}
	// Tell them to shut down
	var closeWG sync.WaitGroup
	logging.Info("Shutting down connections")
	for i := 0; i < to.NumTotalProcs; i++ {
		closeWG.Add(1)
		go func(idx int) {
			testProcs[idx].Close()
			logging.Info("thread closed id", idx)
			closeWG.Done()
		}(i)
	}
	// Wait for all to finish shutting down
	closeWG.Wait()
	wgNormal.Wait()
	wgFailNew.Wait()

	// Check the decisions
	if to.CheckDecisions {

		logging.Info("Verifying decisions")
		switch to.OrderingType {
		case types.Total:
			var decs [][][]byte
			for i := 0; i < to.NumTotalProcs; i++ {
				decs = append(decs, GetDecisions(to, initItem, ds[0]))
			}
			VerifyDecisions(to, retExtraParRegInfo, initItem.GetConsType(), privKeys[0], generalConfigs[0], decs)
		case types.Causal:
			var roots []*utils.StringNode
			var orderedDecisions [][]byte
			for i := 0; i < to.NumTotalProcs; i++ {
				nxtRoot, decs := GetCausalDecisions(memberCheckerStates[i])
				orderedDecisions = decs // TODO clean this up (only need to gen once)
				roots = append(roots, nxtRoot)
			}
			VerifyCausalDecisions(to, retExtraParRegInfo, privKeys[0], generalConfigs[0],
				roots, orderedDecisions)
		default:
			panic(to.OrderingType)
		}
	}

	logging.Info("Closing disk storage")
	for i := 0; i < to.NumTotalProcs; i++ {
		err := ds[i].Close()
		if err != nil {
			logging.Error(err)
		}
		err = ds[i].DeleteFile()
		if err != nil {
			logging.Error(err)
		}
	}

	if config.PrintStats {
		msList := make([]stats.MergedStats, len(statsList))
		for i, nxt := range statsList {
			msList[i] = nxt.MergeLocalStats(int(types.ComputeNumRounds(to)))
			sString := fmt.Sprintf("Stats for proc %v: %v, %v", i, msList[i].String(), msList[i].NwString())
			logging.Infof(sString)

		}
		logging.Print(GetStatsString(to, msList, false, ""))
		logging.Print(consStates[to.NumTotalProcs-1].SMStatsString(msList[0].FinishTime.Sub(msList[0].StartTime))) // TODO merge these stats also???
	}

	for _, nxt := range privKeys {
		nxt.Clean()
	}
}

func GetStoreType(to types.TestOptions) (storeType storage.ConsensusIDType) {
	switch to.OrderingType {
	case types.Causal:
		storeType = storage.HashKey
	case types.Total:
		storeType = storage.ConsensusIndexKey
	default:
		panic(to.OrderingType)
	}
	return
}

func UpdateStorageAfterFail(to types.TestOptions, ds storage.StoreInterface) {
	// TODO should do this differently?
	if to.OrderingType == types.Total {
		for j := computeFailRound(to) + 1; true; j++ {
			s, _, err := ds.Read(j)
			if len(s) == 0 || err != nil {
				break
			}
			err = ds.Write(j, nil, nil)
			if err != nil {
				panic(err)
			}
		}
	}
}

// GetDecisions returns the list of decided values for a given consensus item and storage.
func GetDecisions(to types.TestOptions, consItem consinterface.ConsItem, ds storage.StoreInterface) (res [][]byte) {
	for j := types.ConsensusInt(1); j <= types.ConsensusInt(to.MaxRounds)+config.WarmUpInstances; j++ {
		st, dec, err := ds.Read(j)
		if err != nil {
			panic(err)
		}
		res = append(res, consItem.ComputeDecidedValue(st, dec))
	}
	return
}

func VerifyDecisions(to types.TestOptions, retExtraParRegInfo [][]byte, consType types.ConsType,
	privKey sig.Priv, generalConfig *generalconfig.GeneralConfig, decs [][][]byte) consinterface.StateMachineInterface {

	sm := GenerateStateMachine(to, nil, retExtraParRegInfo,
		0, nil, nil, privKey.GetBaseKey().GetPub().New,
		privKey.GetBaseKey().NewSig, generalConfig)
	outOfOrderErrs, errs := CheckDecisions(decs, sm, to)
	if len(errs) > 0 {
		panic(fmt.Sprint(errs, outOfOrderErrs))
	}
	if len(outOfOrderErrs) > 0 {
		logging.Errorf("Got some out of order errors: %v", outOfOrderErrs)
		if !to.AllowsOutOfOrderProposals(consType) {
			fmt.Println(decs)
			panic(outOfOrderErrs)
		}
	}
	return sm
}

func VerifyCausalDecisions(to types.TestOptions, retExtraParRegInfo [][]byte,
	privKey sig.Priv, generalConfig *generalconfig.GeneralConfig, roots []*utils.StringNode,
	orderedDecisions [][]byte) {

	errs := CheckCausalDecisions(roots, orderedDecisions, GenerateCausalStateMachine(to, privKey, privKey.GetPub(),
		retExtraParRegInfo, 0, nil, nil, nil, generalConfig), to)
	if len(errs) > 0 {
		panic(fmt.Sprint(errs))
	}

}

func GetCausalDecisions(state consinterface.ConsStateInterface) (
	rootNode *utils.StringNode, orderedDecisions [][]byte) {

	return state.(*consinterface.CausalConsInterfaceState).GetCausalDecisions()
}

func CheckCausalDecisions(decidedRoots []*utils.StringNode, orderedDecisions [][]byte, pi consinterface.CausalStateMachineInterface,
	to types.TestOptions) (errors []error) {

	// We need the tree here because nodes can decide in different orders.
	if err := recCheckEqual(decidedRoots, to); err != nil {
		return []error{err}
	}

	// TODO clean this up, we can just process the ordered decisions directly
	orderedRoot := &utils.StringNode{}
	nxtNode := orderedRoot
	for _, nxt := range orderedDecisions {
		newNode := &utils.StringNode{Value: string(nxt)}
		nxtNode.Children = append(nxtNode.Children, newNode)
		nxtNode = newNode
	}

	return pi.CheckDecisions(orderedRoot)
}

func recCheckEqual(nodes []*utils.StringNode, to types.TestOptions) error {
	if len(nodes) != to.NumTotalProcs {
		return fmt.Errorf("not enough decisions")
	}
	for _, nxt := range nodes {
		if nodes[0].Value != nxt.Value {
			return fmt.Errorf("decided different values: %v, %v", nodes[0].Value, nxt.Value)
		}
	}
	for i := range nodes[0].Children {
		var nxtNodes []*utils.StringNode
		for _, nxt := range nodes {
			nxtNodes = append(nxtNodes, nxt.Children[i])
		}
		if err := recCheckEqual(nxtNodes, to); err != nil {
			return err
		}
	}
	return nil
}

// CheckDecisions checks if the decided values are valid.
// (1) Checks that all processes decided the same value.
// (2) Checks with the state machine that the decided values are valid.
func CheckDecisions(decidedValues [][][]byte, pi consinterface.StateMachineInterface, to types.TestOptions) (outOforderErrors, errors []error) {
	if len(decidedValues) < 2 {
		panic("not enough processes")
	}
	var decs [][]byte
	for j := types.ConsensusInt(0); j < types.ConsensusInt(to.MaxRounds)+config.WarmUpInstances; j++ {
		initial := decidedValues[0][j]
		decs = append(decs, initial)
		logging.Infof("Decided cons %v, %v", j, initial)
		for i := 1; i < to.NumTotalProcs; i++ {
			next := decidedValues[i][j]
			if !bytes.Equal(initial, next) {
				return nil, []error{fmt.Errorf(
					"got unequal values %v, %v, for procs %v, %v, cons instance %v", initial, next, 0, i, j)}
			}
		}
	}
	return pi.CheckDecisions(decs)
}

// OpenTestStore opens the storage object given the test configuration, this could be an in-memory store or an on-disk store.
func OpenTestStore(bufferSize int, st types.StorageType, idType storage.ConsensusIDType, i int, fileName string, shouldClear bool) storage.StoreInterface {
	switch st {
	case types.Diskstorage:
		return openDiskStore(bufferSize, idType, i, fileName, shouldClear)
	case types.Memstorage:
		return storage.NewConsMemStore(idType)
	default:
		panic("invalid storage type")
	}
}

// openDiskStore opens a disk storage, if shouldClear is true, then all key/value pairs are deleted.
func openDiskStore(bufferSize int, idType storage.ConsensusIDType, i int, name string, shouldClear bool) *storage.ConsDiskstore {
	//openType := os.O_RDWR|os.O_CREATE
	folderPath := filepath.Join("teststorage")
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		panic(err)
	}
	fullpath := filepath.Join(folderPath, fmt.Sprintf("%v_test%v.store", name, i))
	ds, err := storage.NewConsDiskStore(fullpath, idType, bufferSize)
	if err != nil {
		panic(err)
	}
	if shouldClear {
		err = ds.Clear()
		if err != nil {
			panic(err)
		}
	}
	return ds
}
