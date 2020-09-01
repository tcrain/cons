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

package types

import (
	"encoding/json"
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/logging"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// TestOptions are given as input to consensus tests to describe the test behaviour
type TestOptions struct {
	UseFixedSeed          bool                   // If true then use a fix seed to generate proposals/coin
	OrderingType          OrderingType           // Total order or causal order
	FailRounds            uint64                 // The round faulty processes will crash
	FailDuration          uint64                 // How long to sleep before restarting a failure in milliseconds
	MaxRounds             uint64                 // The number of consensus instances run by the test
	NumFailProcs          int                    // The number of faulty processes
	NumTotalProcs         int                    // The total number of processes
	NumNonMembers         int                    // How many of the processes are just listening/forwarding, but are not allowed to participate in consensus
	StorageType           StorageType            // The type of storage to use
	ClearDiskOnRestart    bool                   // If when faulty processes crash they should also lose their disk storage
	NetworkType           NetworkPropagationType // The network type to use (all to all or gossip)
	FanOut                int                    // How many neighbors a process has (when not using an all to all network)
	ConnectionType        NetworkProtocolType    // The connection type being used (UDP/TCP)
	ByzType               ByzType                // The behaviour of the byzantine processes
	NumByz                int                    // The number of byzantine processes
	CheckDecisions        bool                   // Check all processes have decided the same values at the end of the test
	MsgDropPercent        int                    // Percentage of messages to be artificially dropped by the network
	IncludeProofs         bool                   // Include signatures as part of messages that prove you are sending a valid message (see protocol description)
	SigType               SigType                // The type of signature to use
	UsePubIndex           bool                   // Identify processes just by their index in the list of pub keys (otherwise use the whole pub key as the id)
	SleepValidate         bool                   // If true we dont validate sigs, just sleep
	SleepCrypto           bool                   // If true all signature based crypto is done using sleeps
	MCType                MemberCheckerType      // if TestMemberCheckers is false, then test a specific type
	BufferForwarder       bool                   // Buffer several messages before forwarding them (in a gossip network)
	UseMultisig           bool                   // Use multi-signautres
	BlsMultiNew           bool                   // Use the new type of BLS multi-signatures (see https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html)
	MemCheckerBitIDType   BitIDType              // If using multi-sigs the type of bit ID to use in the member checker
	SigBitIDType          BitIDType              // If using multi-sigs the type of bit ID to use with the signatures
	StateMachineType      StateMachineType       // The application being implemented by the state machines
	PartialMessageType    PartialMessageType     // The type of partial messsages to use during broadcasts
	AllowConcurrent       uint64                 // Number of concurrent consensus indecies to allow to run.
	LocalRandMemberChange uint64                 // On consensus index mod this value, the local rand member checker will change.
	RotateCord            bool                   // If true then the coordinator will rotate each consensus index, if supported by the member checker.
	AllowSupportCoin      bool                   // True if AuxProofMessages can support the coin directly instead of a bin value.
	ConsType              ConsType               // The type of the consensus being used for the test.
	// UseFullBinaryState will (if true) keep the consensus state as the list of all valid messages received appended together,
	// if false stores only different messages with all the signatures at the end
	UseFullBinaryState    bool
	StorageBuffer         int                  // byte size of buffer for writing to disk
	IncludeCurrentSigs    bool                 // When forwarding a message (for non all-to-all networks) will include all sigs received so far
	CPUProfile            bool                 // If true will profile CPU usage
	MemProfile            bool                 // If true will profile mem usage
	NumMsgProcessThreads  int                  // Number of threads that will process messages
	MvProposalSizeBytes   int                  // Size of proposals when using MvCons1ProposalInfo to propose random bytes
	BinConsPercentOnes    int                  // When using binary consensus number of proposals that will be 1 vs 0 (randomly chosen) for testing
	CollectBroadcast      CollectBroadcastType // If true, when sending the commit message, will send it to the leader
	StopOnCommit          StopOnCommitType     // If true then the consensus will not execute rounds after deciding (the eventual message propagation will ensure termination)
	ByzStartIndex         uint64               // Index to start faulty behaviour
	TestID                uint64               // unique identifier for the test
	AdditionalP2PNetworks int                  // Generate additional P2P connection networks, used when sending the same message type multiple times when using buffer forwarder
	EncryptChannels       bool                 // True if network channels should be encrypted using nacl secretbox
	NoSignatures          bool                 // Use encrypted channels for message authentification instead of signatures
	CoinType              CoinType             // The type of coin being used
	UseFixedCoinPresets   bool                 // If true then will use predefined coins for the initial rounds of randomized consensus
	SharePubsRPC          bool                 // If true then during RPC tests nodes on the same machine will share public key objects for external nodes

	WarmUpInstances             int // Number of consensus instances to run before recording results
	KeepPast                    int // Number of previously decided consensus instances to keep in memory
	ForwardTimeout              int // milliseconds 	// for msg forwarder when you dont receive enough messages to foward a buffer automatically
	RandForwardTimeout          int // amount of time to randomly add to Forward timeout
	ProgressTimeout             int // milliseconds, if no progress in this time, let neighbors know
	MvConsTimeout               int // millseconds timeout when taking an action in the 3 step mv to bin reduction
	MvConsRequestRecoverTimeout int // millseconds timeout before requesting the full proposal after delivering the hash

	NodeChoiceVRFRelaxation int           // Additional chance to chose a node as a member when using VRF.
	CoordChoiceVRF          int           // Chance of each node being chosen as a coordinator when using VRF.
	GenRandBytes            bool          // If true the state machine shouldn generate random bytes each decision.
	RndMemberCount          int           // Only works if GenRandBytes is ture. This chooses RndMemberCount members to randomly decide which nodes will participate, if 0 random selection is not used.
	RndMemberType           RndMemberType // Type of random membeship selection, RndMemberCount must be > 0 for this.
	UseRandCoord            bool          // If true round coordinators will be chosen using VRFs note, that these are only calculated from within the existing random members
	// so the coordinator relaxation may need to be higher
}

// UsesVRFs returns true if this test configuration uses VRFs.
func (to TestOptions) UsesVRFs() bool {
	switch to.RndMemberType {
	case KnownPerCons, VRFPerCons, VRFPerMessage:
		return true
	}
	return to.UseRandCoord
}
func (to TestOptions) String() string {
	return fmt.Sprintf("{ConsType: %v, Rounds: %v, Fail round: %v, Total procs: %v, Nonmember procs: %v, Fail procs: %v, "+
		"\n\tConnection: %s, Msg Drop%%: %v, Network: %s, Nw fan out: %v, Storage type: %s, Clear disk on restart: %v,"+
		"\n\tInclude proofs: %v, Sig type: %s, Use multisig: %v, Use pub index: %v, Buffer Forwarder: %v,"+
		"\n\tState machine: %v, Allow concurrent: %v, Rotate cord: %v, Gen rand bytes: %v, Ordering: %v,"+
		"\n\tRand member type: %v, Rand coord: %v, Rand members %v, LocalRandMemberChange: %v, AllowSupportCoin: %v,"+
		"\n\tUseFullBinaryState %v, StorageBuffer %v, IncludeCurrentSigs %v, CPUProfile %v, MemProfile %v,"+
		"\n\tNumMsgProcessThreads %v, MvProposalSizeBytes %v, BinConsPercentOnes %v, CollectBroadcast: %v,"+
		"\n\tStopOnCommit: %v, FixedSeed: %v, EncryptChannels: %v, NoSignatures: %v, Byz procs: %v, ByzType: %s,"+
		"\n\tCoinType: %v, UseFixedCoinPresets: %v, Sleep Crypto: %v, Share Pubs: %v,  MCType: %v,"+
		"\n\tSig BitID: %v, MC BitID: %v, WarmUp: %v, KeepPast %v, FwdTimeout: %v, RndFwdTimeout: %v,"+
		"\n\t ProgressTimeout: %v, MVTimeout: %v, MVRecoverTimeout: %v, NodeVRFRelax: %v, CoordVRF: %v, TestID %v}",
		to.ConsType, to.MaxRounds, to.FailRounds, to.NumTotalProcs, to.NumNonMembers, to.NumFailProcs,
		to.ConnectionType, to.MsgDropPercent, to.NetworkType, to.FanOut, to.StorageType,
		to.ClearDiskOnRestart, to.IncludeProofs, to.SigType, to.UseMultisig, to.UsePubIndex, to.BufferForwarder,
		to.StateMachineType, to.AllowConcurrent, to.RotateCord, to.GenRandBytes, to.OrderingType, to.RndMemberType,
		to.UseRandCoord, to.RndMemberCount, to.LocalRandMemberChange, to.AllowSupportCoin,
		to.UseFullBinaryState, to.StorageBuffer, to.IncludeCurrentSigs, to.CPUProfile, to.MemProfile,
		to.NumMsgProcessThreads, to.MvProposalSizeBytes, to.BinConsPercentOnes, to.CollectBroadcast, to.StopOnCommit, to.UseFixedSeed,
		to.EncryptChannels, to.NoSignatures, to.NumByz, to.ByzType, to.CoinType, to.UseFixedCoinPresets,
		to.SleepCrypto, to.SharePubsRPC, to.MCType,
		to.SigBitIDType, to.MemCheckerBitIDType, to.WarmUpInstances, to.KeepPast, to.ForwardTimeout, to.RandForwardTimeout, to.ProgressTimeout,
		to.MvConsTimeout, to.MvConsRequestRecoverTimeout, to.NodeChoiceVRFRelaxation, to.CoordChoiceVRF, to.TestID)
}

func AllowsGetRandBytes(smType StateMachineType) bool {
	switch smType {
	case CounterProposer, BytesProposer, CounterTxProposer, CurrencyTxProposer:
		return true
	}
	return false
}

func makeStr(id string, val, newVal interface{}) string {
	return fmt.Sprintf("%v: %v -> %v, ", id, val, newVal)
}

// FieldDiff returns a list of the fields that differ between the test options.
func (to TestOptions) FieldDiff(other TestOptions) (ret []string) {
	toV := reflect.ValueOf(to)
	otherV := reflect.ValueOf(other)
	for i := 0; i < toV.NumField(); i++ {
		v1 := toV.Field(i).Interface()
		v2 := otherV.Field(i).Interface()
		if v1 != v2 {
			ret = append(ret, toV.Type().Field(i).Name)
		}
	}
	return ret
}

// StringDiff returns a human readable string of the differences between the test options.
func (to TestOptions) StringDiff(other TestOptions) string {
	b := strings.Builder{}
	b.WriteString("Config difference: ")
	toV := reflect.ValueOf(to)
	otherV := reflect.ValueOf(other)
	for i := 0; i < toV.NumField(); i++ {
		v1 := toV.Field(i).Interface()
		v2 := otherV.Field(i).Interface()
		if v1 != v2 {
			if _, err := b.WriteString(makeStr(toV.Type().Field(i).Name, v1, v2)); err != nil {
				panic(err)
			}
		}
	}
	return b.String()
}

func (to TestOptions) AllowsRandMembers(checkerType MemberCheckerType) bool {
	if !to.GenRandBytes && to.RndMemberType != LocalRandMember {
		return false
	}
	switch checkerType {
	case CurrentTrueMC, CurrencyMC:
		return true
	}
	return false
}

// AllowsOutOfOrderProposals should return true if proposals might not see the state of the previous commited when they are made.
func (to TestOptions) AllowsOutOfOrderProposals(consType ConsType) bool {
	switch to.StateMachineType {
	case CurrencyTxProposer:
		return false
	}
	switch consType {
	case BinCons1Type, BinConsRnd1Type:
		return false
	case MvBinCons1Type, MvBinConsRnd1Type, MvBinConsRnd2Type:
		if to.AllowConcurrent > 1 {
			return true
		}
		return false
	case MvCons2Type:
		if to.AllowConcurrent > 1 {
			return true
		}
		return false
	case MvCons3Type:
		return false
	case SimpleConsType:
		if to.AllowConcurrent > 1 {
			return true
		}
		return false
	case RbBcast1Type, RbBcast2Type:
		if to.AllowConcurrent > 1 {
			return true
		}
		return false
	default:
		panic("unknown cons type")
	}
}

func GetOrderForSM(smt StateMachineType) OrderingType {
	for _, nxt := range TotalOrderProposerTypes {
		if smt == nxt {
			return Total
		}
	}
	for _, nxt := range CausalProposerTypes {
		if smt == nxt {
			return Causal
		}
	}
	if smt == TestProposer {
		return Total
	}
	panic(smt)
}

// CheckValid returns an error if the generalconfig in invalid
func (to TestOptions) CheckValid(consType ConsType, isMv bool) (newTo TestOptions, err error) {
	newTo = to
	consProcs := to.NumTotalProcs - to.NumNonMembers // consensus participants

	if to.AllowSupportCoin {
		switch to.CoinType {
		case KnownCoinType, LocalCoinType, FlipCoinType:
			err = fmt.Errorf("cannot support coin when using known, flip, or local coin type")
			return
		}
		if consType == MvBinConsRnd1Type || consType == MvBinConsRnd2Type {
			err = fmt.Errorf("TODO allow support coin with reduction")
			return
		}
	}

	if to.RndMemberType != NonRandom {
		switch to.CoinType {
		case NoCoinType, KnownCoinType, FlipCoinType, LocalCoinType:
		default:
			err = fmt.Errorf("coin %v not supported with %v", to.CoinType, to.RndMemberType)
			return
		}
	}

	if to.CoinType != NoCoinType {
		switch UseTp1CoinThresh(to) {
		case true:
			switch to.CoinType {
			case StrongCoin2EchoType, StrongCoin1EchoType:
			// ok
			case StrongCoin2Type:
				// must be rndcons 4
				if to.ConsType != BinConsRnd4Type && to.ConsType != BinConsRnd6Type {
					err = fmt.Errorf("if using t+1 threshold coins and StrongCoin2, must use BinConsRnd4/6")
					return
				}
			case StrongCoin1Type:
				if to.ConsType != BinConsRnd4Type && to.ConsType != BinConsRnd6Type {
					err = fmt.Errorf("if using t+1 threshold coins and StrongCoin1, must use BinConsRnd4/6")
					return
				}
			}
		case false:
		}
	}

	if to.CoinType == StrongCoin1Type || to.CoinType == StrongCoin1EchoType {
		if to.SigType != TBLS && to.SigType != TBLSDual && to.SigType != CoinDual {
			err = fmt.Errorf("StrongCoin1Type must be used with TBLS or TBLSDual")
			return
		}
	}

	if to.CoinType == StrongCoin2Type || to.CoinType == StrongCoin2EchoType {
		if to.SigType != EDCOIN {
			err = fmt.Errorf("StrongCoin2Type must be used with EDCOIN")
			return
		}
	}

	if to.RndMemberType != NonRandom {
		if to.RndMemberCount > to.NumTotalProcs-to.NumNonMembers {
			err = fmt.Errorf("must have at least as many members as rand members, rnd members: %v, non members: %v, total mem: %v",
				to.RndMemberCount, to.NumTotalProcs, to.NumNonMembers)
			return
		}
	}

	if to.RndMemberType == VRFPerCons && to.UseRandCoord {
		err = fmt.Errorf("rand coord not supported with VRFPerCons")
		return
	}

	if to.NoSignatures {
		switch to.RndMemberType {
		case VRFPerCons, VRFPerMessage, KnownPerCons:
			err = fmt.Errorf("must use signatures with %v", to.RndMemberType)
			return
		}
		if to.CollectBroadcast != Full {
			err = fmt.Errorf("no signatures only support full broadcasts")
			return
		}
		if to.NetworkType != AllToAll && to.NetworkType != RequestForwarder {
			err = fmt.Errorf("must use either all to all network or request forward network with no signatures")
			return
		}
		if !to.EncryptChannels {
			err = fmt.Errorf("if not using signatures, must encrypt channels")
			return
		}
		if to.StopOnCommit == SendProof {
			err = fmt.Errorf("cannot send proofs if not using signatures")
			return
		}
		if to.IncludeProofs {
			err = fmt.Errorf("cannot use proofs if not using signatures")
			return
		}
	}

	if to.EncryptChannels {
		var found bool
		for _, nxt := range EncryptChannelsSigTypes {
			if nxt == to.SigType {
				found = true
				break
			}
		}
		if !found {
			err = fmt.Errorf("sig type %d does not support encrypted channels", to.SigType)
			return
		}
	}

	if !to.BufferForwarder && to.AdditionalP2PNetworks > 0 {
		err = fmt.Errorf("additional P2P networks not needed if buffer forwarder is being used")
		return
	}

	if to.BufferForwarder && !to.IncludeCurrentSigs {
		err = fmt.Errorf("if using buffer forwarder must include current signatures")
		return
	}

	if to.CollectBroadcast != Full && (to.RndMemberType != NonRandom || to.OrderingType == Causal) {
		err = fmt.Errorf("collect broadcast only supported for total order and non-random membership")
		return
	}

	if to.SigType == TBLS && (to.UseMultisig || to.BlsMultiNew || !to.UsePubIndex) {
		err = fmt.Errorf("For bls thrsh (TBLS) must not use multi sig or bls multi new and must use put index, useMultiSig: %v, BlsMultiNew: %v, UsePubIndex %v",
			to.UseMultisig, to.BlsMultiNew, to.UsePubIndex)
		return
	}

	if to.SigType == QSAFE && !config.AllowQsafe {
		err = fmt.Errorf("qsafe signatures disabled in config.go")
		return
	}

	if to.MCType == CurrencyMC && to.StateMachineType != CurrencyTxProposer {
		err = fmt.Errorf("if using CurrencySM, then must use SimpleCurrencyTxProposer state machie")
		return
	}

	if consProcs < 4 {
		err = fmt.Errorf("must have at least 4 consensus participants")
		return
	}
	if to.AllowConcurrent > 1 && to.MCType != TrueMC {
		err = fmt.Errorf("must use static membership for concurrent consensus")
		return
	}

	if to.UseMultisig && to.SigType != BLS {
		err = fmt.Errorf("multisig and sig type %v not valid", to.SigType)
		return
	}

	if to.OrderingType != GetOrderForSM(to.StateMachineType) {
		err = fmt.Errorf("sm type %v not valid for order type %v", to.OrderingType, to.StateMachineType)
		return
	}

	switch to.ConsType {
	case SimpleConsType:
		if to.StateMachineType != TestProposer {
			err = fmt.Errorf("must use SimpleProposalInfo with SimpleCons")
			return
		}
	default:
		if to.StateMachineType == TestProposer {
			err = fmt.Errorf("must use SimpleProposalInfo with SimpleCons")
			return
		}
		switch isMv {
		case true:
			if to.StateMachineType == BinaryProposer {
				err = fmt.Errorf("must not use binary proposer for multi value cons")
				return
			}
		case false:
			if to.StateMachineType != BinaryProposer {
				err = fmt.Errorf("must use binary proposer for non-multi value cons")
				return
			}
		}
	}

	switch to.SigType {
	case EDCOIN, TBLS, TBLSDual, CoinDual:
		if !to.UsePubIndex {
			err = fmt.Errorf("must use pub index with threshold signatures")
			return
		}
		if to.RndMemberType != NonRandom {
			err = fmt.Errorf("must use static membership for threhosld or coin signatures")
			return
		}
		switch to.MCType {
		case TrueMC, CurrentTrueMC:
		default:
			err = fmt.Errorf("must use static membership for threhosld or coin signatures")
			return
		}
	}

	switch consType {
	case BinConsRnd1Type, BinConsRnd2Type, BinConsRnd3Type, BinConsRnd4Type, BinConsRnd5Type,
		BinConsRnd6Type, MvBinConsRnd1Type:
		if !to.UsePubIndex {
			err = fmt.Errorf("must use pub index for BinConsRnd1Type/MvBinConsRnd1Type")
			return
		}
		switch to.SigType {
		case EDCOIN, TBLS, TBLSDual, CoinDual:
		default:
			err = fmt.Errorf("must use threshold or coin sigs for BinConsRnd1")
			return
		}
		switch to.RndMemberType {
		case NonRandom:
		default:
			err = fmt.Errorf("BinConsRnd1 does not support random membership")
			return
		}
	default:
		if to.UseFixedCoinPresets {
			err = fmt.Errorf("UseFixedCoinPresets not used for non-randomized consensus types")
			return
		}
	}
	if to.RndMemberCount > 0 && to.NumNonMembers > 0 {
		if consProcs < to.RndMemberCount {
			err = fmt.Errorf("must have at least as many normal members as random members")
			return
		}
		// panic(1)
		// return fmt.Errorf("currently not supported to have non-members and rand members") // TODO fix this.
	}
	if to.RndMemberType != NonRandom && to.RndMemberCount < 4 {
		err = fmt.Errorf("if using random membership selection, must have a RndMemberCout at least 4")
		return
	}
	if (consType == RbBcast1Type || consType == RbBcast2Type) && (to.RndMemberType == VRFPerCons ||
		to.RndMemberType == VRFPerMessage || to.UseRandCoord) {

		err = fmt.Errorf("cons type RbBcast does not support VRFPerCons or VRFPerMessage (because there will be differnt coordinators, and will block termination)")
		return
	}
	if to.NetworkType == RequestForwarder && (to.RndMemberType != LocalRandMember || to.BufferForwarder) {
		err = fmt.Errorf("request forwarder and local random mebership must be used together, and without BufferForwarder")
		return
	}
	if to.RndMemberType == LocalRandMember {
		if to.NetworkType != RequestForwarder {
			err = fmt.Errorf("localRandMember must be used with RequestForwarder network")
			return
		}
		//if to.RndMemberCount < to.FanOut {
		//	return fmt.Errorf("with request forwarder, must have ran out at least as large as rand member count")
		//}
	}
	if to.RndMemberType == NonRandom && to.RndMemberCount > 0 {
		err = fmt.Errorf("if > 0 RndMemberCount then must use a random member type")
		return
	}
	if !(to.GenRandBytes || to.RndMemberType == LocalRandMember) && to.RndMemberCount > 0 {
		err = fmt.Errorf("if ChooseRandMembers, then GenRandBytes must be true")
		return
	}
	if !to.GenRandBytes && to.UseRandCoord {
		err = fmt.Errorf("GenRandBytes must be true if using random coordinator")
	}
	if !to.AllowsRandMembers(to.MCType) && to.RndMemberCount > 0 {
		err = fmt.Errorf("member checker type %v does not support random membership", to.MCType)
		return
	}
	if to.NumFailProcs > 0 && to.ByzType != NonFaulty {
		err = fmt.Errorf("cannot have both byzantine and crash processes in the same test")
		return
	}
	if to.UseMultisig {
		switch to.RndMemberType {
		case NonRandom, KnownPerCons:
			// multisig allowed
		default:
			err = fmt.Errorf("multisig and random member selection not currently supported")
			return
		}
	}
	if to.NumNonMembers >= to.NumTotalProcs {
		err = fmt.Errorf("need to have some members")
		return
	}
	if to.ByzType != NonFaulty && to.NumByz == 0 {
		logging.Info("if byz type them must have at least 1 byz faulty process, updating the test config to use" +
			" 1/3 of the number of nodes")
		newTo.NumByz = GetOneThirdBottom(consProcs)
	}
	if (consProcs%3 == 0 && to.NumByz >= consProcs/3) || (consProcs%3 != 0 && to.NumByz > consProcs/3) {
		logging.Error("if byz type them must have less than 1/3 byz faulty process, updating the test config to use" +
			"1/3 of the number of nodes")
		newTo.NumByz = GetOneThirdBottom(consProcs)
	}
	if to.FanOut >= to.NumTotalProcs {
		err = fmt.Errorf("use all to all connection if network fan out is the same as the number of processes")
		return
	}
	if to.FanOut < 1 && (to.NetworkType == P2p || to.NetworkType == Random || to.NetworkType == RequestForwarder) {
		err = fmt.Errorf("Must have fan out of at least 1 for p2p network")
		return
	}
	if to.UseMultisig && !to.UsePubIndex {
		err = fmt.Errorf("Multisig must be used with use pub index")
		return
	}
	if consType == MvCons3Type && (config.KeepFuture < 5) {
		err = fmt.Errorf("when using MvCons3 generalconfig.KeepFuture must be at least 5")
		return
	}
	if consType == MvCons3Type && (to.AllowConcurrent != 0) {
		err = fmt.Errorf("when using MvCons3 AllowConcurrent must be 0")
		return
	}
	if consType == MvCons3Type && !(to.MCType == TrueMC) {
		err = fmt.Errorf("when using MvCons3 must use true memberchecker TrueMC")
		return
	}
	if consType == MvCons3Type && (to.RotateCord || to.RndMemberType != NonRandom || to.UseRandCoord) {
		err = fmt.Errorf("must set rotateCord to false and disable random membership when using MvCons3")
		return
	}
	if to.OrderingType == Causal {
		if to.RotateCord {
			err = fmt.Errorf("must set rotateCord to false when using causal order")
			return
		}
		if to.UseRandCoord {
			err = fmt.Errorf("UseRandCoord must be fale when using causal ordering")
			return
		}
		if to.ByzType != NonFaulty {
			err = fmt.Errorf("TODO") // TODO
			return
		}
	}
	if to.GenRandBytes {
		switch to.SigType {
		case TBLS, BLS, TBLSDual, EC: // OK
		default:
			err = fmt.Errorf("signatures type %v do not support generating random bytes", to.SigType)
			return
		}
	}

	return
}

// GetTestOptions generates a types.TestOptions object from a json formatted file.
func GetTestOptions(optionsPath string) (to TestOptions, err error) {
	var raw []byte
	raw, err = ioutil.ReadFile(filepath.Join(optionsPath))
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &to)
	return
}

// ToToDisk stores the test options to disk in folderPath, using TestID as the file name.
func TOToDisk(folderPath string, to TestOptions) error {
	toByt, err := json.MarshalIndent(to, "", "\t")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(folderPath, fmt.Sprintf("%v.json", to.TestID)),
		toByt, os.ModePerm); err != nil {

		return err
	}
	return nil
}

// ToToDisk stores the test options to disk in folderPath, using TestID as the file name.
func TOConsToDisk(folderPath string, to TestOptionsCons) error {
	toByt, err := json.MarshalIndent(to, "", "\t")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(folderPath, fmt.Sprintf("%v.json", to.TestID)),
		toByt, os.ModePerm); err != nil {

		return err
	}
	return nil
}

func GetOneThirdBottom(n int) int {
	switch n % 3 {
	case 0:
		return n/3 - 1
	default:
		return n / 3
	}
}
