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
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"runtime/pprof"
	"testing"
)

func GetBaseSMType(consType types.ConsType, order types.OrderingType, consConfigs ConfigOptions,
	to types.TestOptions) types.StateMachineType {

	if order == types.Causal {
		return types.CausalCounterProposer
	}
	if consType == types.SimpleConsType {
		return types.TestProposer
	}
	if consConfigs.GetIsMV() {
		if to.AllowsOutOfOrderProposals(consType) {
			return types.CounterProposer
		}
		return types.CurrencyTxProposer
	}
	return types.BinaryProposer
}

// RunConsType runs a test for the given configuration and inputs.
func RunBasicTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}
	to.NumMsgProcessThreads = 1
	to.MaxRounds = config.MaxRounds
	to.NumTotalProcs = config.ProcCount
	if consConfigs.GetAllowsNonMembers() {
		to.NumNonMembers = config.NonMembers
	}
	to.ClearDiskOnRestart = false
	to.NetworkType = types.AllToAll
	to.CheckDecisions = true
	to.SleepValidate = config.TestSleepValidate
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)

	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	to.StorageType = types.Memstorage
	tconfig := ReplaceNilFields(OptionStruct{
		ByzTypes: []types.ByzType{types.NonFaulty},
	}, AllTestConfig)

	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

	// run a single test with UDP
	if to.EncryptChannels == false { // TODO UDP currently not supported encrypted channels
		udpTO := to
		udpTO.ConnectionType = types.UDP
		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, udpTO))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, []int{0}, t)
	}

	// Special test for currency state machine plus currency memberchecker.
	if to.OrderingType == types.Total && checkCurrencySM(consConfigs) {
		logging.Print("Running test with SimpleCurrencyTxProposer and Currency member checker")
		// for these tests we only want to run one test, so we just take the first config if none is set
		if len(toRun) == 0 {
			toRun = []int{0}
		}
		to.StateMachineType = types.CurrencyTxProposer
		to.MCType = types.CurrencyMC

		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, toRun, t)
	}

}

func RunByzTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {
	if config.AllowConcurrentTests {
		t.Parallel()
	}
	to.ByzType = types.HalfHalfFixedBin
	to.MaxRounds = config.MaxRounds
	to.NumTotalProcs = config.ProcCount
	to.NumNonMembers = config.NonMembers
	to.ClearDiskOnRestart = false
	to.NetworkType = types.AllToAll
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	// to.IncludeProofs = true
	to.SleepValidate = config.TestSleepValidate

	switch to.ConsType {
	case types.MvCons3Type, types.MvCons4Type:
	default:
		if consConfigs.GetIsMV() {
			to.RotateCord = true // So the tests dont take too long
		}
		// to.MCType = types.BinRotateMC
	}

	to.StorageType = types.Memstorage

	// numMembers is the number of participants in the consensus,
	// the fail processes will be taken from the members.
	// During test setup we make a list of all nodes where
	// the first numMembers in that list are the consensus participants.
	// The failures are chosen as the nodes at the head of this list.
	numMembers := to.NumTotalProcs - to.NumNonMembers
	to.NumByz = utils.GetOneThirdBottom(numMembers)

	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(ByzTest, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

}

func checkCurrencySM(consConfigs ConfigOptions) bool {
	for _, nxtSM := range consConfigs.GetStateMachineTypes(AllOptions) {
		switch nxtSM {
		case types.CurrencyTxProposer:
			for _, nxtMC := range consConfigs.GetMemberCheckerTypes(AllOptions) {
				switch nxtMC {
				case types.CurrencyMC:
					return true
				}
			}
		}
	}
	return false
}

func RunMemstoreTest(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}

	to.MaxRounds = config.MaxRounds
	to.NumTotalProcs = config.ProcCount
	to.NumNonMembers = config.NonMembers
	to.ClearDiskOnRestart = false
	to.StorageType = types.Memstorage
	to.NetworkType = types.AllToAll
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	to.SleepValidate = config.TestSleepValidate
	tconfig := ReplaceNilFields(OptionStruct{
		ByzTypes: []types.ByzType{types.NonFaulty},
	}, AllTestConfig)

	iter, err := NewTestOptIter(MinOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)

	runIterTests(initItem, consConfigs, iter, toRun, t)
}

func RunMsgDropTest(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}

	to.MsgDropPercent = 10
	to.MaxRounds = config.MaxRounds
	to.NumNonMembers = config.NonMembers
	to.NumTotalProcs = config.ProcCount
	to.ClearDiskOnRestart = false
	to.NetworkType = types.AllToAll
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	//to.StateMachineType = types.CounterProposer
	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	to.SleepValidate = config.TestSleepValidate
	to.StorageType = types.Diskstorage
	tconfig := ReplaceNilFields(OptionStruct{
		ByzTypes: []types.ByzType{types.NonFaulty},
	}, AllTestConfig)

	logging.Print("Running msg drop test with all to all network")
	iter, err := NewTestOptIter(MinOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

	if !config.RunAllTests {
		logging.Print("Exiting test early because config.RunAllTests is false")
		return
	}

	to.NetworkType = types.P2p
	to.FanOut = config.FanOut
	logging.Print("Running msg drop test with peer to peer network")
	iter, err = NewTestOptIter(MinOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)
}

func RunRandMCTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, runLocalRand bool, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}

	to.MaxRounds = config.MaxRounds
	to.GenRandBytes = true
	to.ClearDiskOnRestart = false
	to.MCType = types.CurrentTrueMC
	to.RndMemberType = types.KnownPerCons
	to.NetworkType = types.AllToAll
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	to.SleepValidate = config.TestSleepValidate
	to.SigType = types.BLS

	// to.StorageType = types.Memstorage
	to.NumTotalProcs = 30
	to.NumNonMembers = config.NonMembers
	// We choose rand members as all other nodes, since on recover we may get a different set of other nodes
	to.RndMemberCount = to.NumTotalProcs - to.NumNonMembers
	to.FanOut = 6
	to.CoordChoiceVRF = 30 // for the test we allow more coordinators so we decide nil less often
	if to.ConsType == types.MvCons3Type {
		to.MCType = types.LaterMC
	}

	logging.Print("Running with VRF type random member selection")
	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

	if to.OrderingType == types.Total && to.ConsType != types.RbBcast1Type && to.ConsType != types.RbBcast2Type {
		// we dont run random coordinator with rbbcasts since they must have only a single broadcaster
		// we allow RbBcast and random with causal ordering because we use a fixed coordinator
		logging.Print("Running with VRF type random member selection and random coordinator")
		cTo := to
		cTo.UseRandCoord = true
		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(BasicTestConfigs, cTo))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, []int{0}, t)
	}

	if runLocalRand {
		// for these tests we only want to run one test, so we just take the first config if none is set
		if len(toRun) == 0 {
			toRun = []int{0}
		}

		to.LocalRandMemberChange = 5
		to.EncryptChannels = true
		to.NoSignatures = true

		logging.Print("Running with local random member selection")
		to.NetworkType = types.RequestForwarder
		to.RndMemberType = types.LocalRandMember
		to.GenRandBytes = false
		tconfig := ReplaceNilFields(OptionStruct{
			ByzTypes: []types.ByzType{types.NonFaulty},
		}, AllTestConfig)

		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, toRun, t)

		if to.ConsType != types.RbBcast1Type && to.ConsType != types.RbBcast2Type { // TODO fix local rand member recover after failure for RBBCast (different proposals)

			if to.OrderingType == types.Total {
				logging.Print("Running with local random member selection and random coord")
				cTo := to
				cTo.GenRandBytes = true
				cTo.UseRandCoord = true
				iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, cTo))
				assert.Nil(t, err)
				runIterTests(initItem, consConfigs, iter, toRun, t)
			}

			numMembers := to.RndMemberCount
			// restart from disk enough remain live to continue
			logging.Print("Running with local random member selection and less than 1/3 fail and clear disk (just a single test)")
			to.NumFailProcs = utils.GetOneThirdBottom(numMembers)
			to.FailRounds = config.MaxRounds / 2
			to.ClearDiskOnRestart = true
			iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
			assert.Nil(t, err)
			runIterTests(initItem, consConfigs, iter, toRun, t)

			// restart from disk enough remain live to continue
			logging.Print("Running  with local random member selection and less than 1/3 fail and recover from disk (just a single test)")
			to.NumFailProcs = utils.GetOneThirdBottom(numMembers)
			to.FailRounds = config.MaxRounds / 2
			to.ClearDiskOnRestart = false
			iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
			assert.Nil(t, err)
			runIterTests(initItem, consConfigs, iter, toRun, t)
		}
	}
}

// RunConsType runs a test for the given configuration and inputs.
func RunMultiSigTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}

	to.MaxRounds = config.MaxRounds
	to.NumNonMembers = config.NonMembers
	to.NumTotalProcs = config.ProcCount
	to.ClearDiskOnRestart = false
	to.NetworkType = types.P2p
	to.FanOut = config.FanOut
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	to.ConnectionType = types.TCP
	to.UseMultisig = true
	to.UsePubIndex = true
	to.SigType = types.BLS
	to.BlsMultiNew = true
	to.ConsType = consType
	to.SleepValidate = config.TestSleepValidate

	to.StorageType = types.Memstorage

	logging.Print("Running with multisigs")
	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

	var knownPerCons bool
	for _, nxt := range consConfigs.GetRandMemberCheckerTypes(AllOptions) {
		if nxt == types.KnownPerCons {
			knownPerCons = true
		}
	}
	if knownPerCons {
		logging.Print("Running with multisigs and randomized members")
		tmpTo := to
		tmpTo.NumTotalProcs = 10
		tmpTo.RndMemberType = types.KnownPerCons
		if tmpTo.ConsType == types.MvCons3Type {
			tmpTo.MCType = types.LaterMC
		} else {
			tmpTo.MCType = types.CurrentTrueMC
		}
		tmpTo.RndMemberCount = tmpTo.NumTotalProcs - tmpTo.NumNonMembers - 1
		tmpTo.GenRandBytes = true
		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, tmpTo))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, toRun, t)
	}
	logging.Print("Running with multisigs and buffer forwarder")
	to.BufferForwardType = types.ThresholdBufferForward
	to.IncludeCurrentSigs = true
	to.AdditionalP2PNetworks = 2
	iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
	assert.Nil(t, err)

	runIterTests(initItem, consConfigs, iter, toRun, t)

	numMembers := to.NumTotalProcs - to.NumNonMembers
	// restart from disk enough remain live to continue
	logging.Print("Running with multisigs and less than 1/3 fail and recover from disk")
	to.NumFailProcs = utils.GetOneThirdBottom(numMembers)
	to.FailRounds = config.MaxRounds / 2
	to.BufferForwardType = types.NoBufferForward
	to.IncludeCurrentSigs = false
	to.AdditionalP2PNetworks = 0

	iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(SingleSMTest, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, []int{0}, t)
}

// RunConsType runs a test for the given configuration and inputs.
func RunP2pNwTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	_ = toRun
	if config.AllowConcurrentTests {
		t.Parallel()
	}

	to.MaxRounds = config.MaxRounds
	to.NumNonMembers = config.NonMembers
	to.NumTotalProcs = 10 // config.ProcCount
	to.ClearDiskOnRestart = false
	to.NetworkType = types.P2p
	to.FanOut = config.FanOut
	to.CheckDecisions = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)

	to.ConnectionType = types.TCP
	to.UsePubIndex = true
	to.ConsType = consType
	to.SleepValidate = config.TestSleepValidate
	to.StorageType = types.Memstorage
	tconfig := ReplaceNilFields(OptionStruct{
		ByzTypes: []types.ByzType{types.NonFaulty},
	}, AllTestConfig)

	logging.Print("Running with static P2P network")
	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, []int{0}, t)

	logging.Print("Running with random P2P network")
	to.NetworkType = types.Random
	iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, []int{0}, t)

	logging.Print("Running with static P2P network and fixed buffer forward")
	to.NetworkType = types.P2p
	to.BufferForwardType = types.FixedBufferForward
	to.ForwardTimeout = 10
	to.ProgressTimeout = 100
	iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, []int{0}, t)
}

// RunConsType runs a test for the given configuration and inputs.
func RunFailureTests(to types.TestOptions, consType types.ConsType, initItem consinterface.ConsItem,
	consConfigs ConfigOptions, toRun []int, t *testing.T) {

	if config.AllowConcurrentTests {
		t.Parallel()
	}
	to.FailRounds = config.MaxRounds / 2
	to.MaxRounds = config.MaxRounds
	to.NumTotalProcs = config.ProcCount
	to.NumNonMembers = config.NonMembers
	to.ClearDiskOnRestart = false
	to.NetworkType = types.AllToAll
	to.ConnectionType = types.TCP
	to.CheckDecisions = true
	to.UsePubIndex = true
	to.StateMachineType = GetBaseSMType(consType, to.OrderingType, consConfigs, to)
	to.ConsType = consType
	to.StorageType = types.Diskstorage
	to.SleepValidate = config.TestSleepValidate
	tconfig := ReplaceNilFields(OptionStruct{
		ByzTypes: []types.ByzType{types.NonFaulty},
	}, AllTestConfig)

	// numMembers is the number of participants in the consensus,
	// the fail processes will be taken from the members.
	// During test setup we make a list of all nodes where
	// the first numMembers in that list are the consensus participants.
	// The failures are chosen as the nodes at the head of this list.
	numMembers := to.NumTotalProcs - to.NumNonMembers

	// restart from disk enough remain live to continue
	logging.Print("Less than 1/3 fail and recover from disk")
	to.NumFailProcs = utils.GetOneThirdBottom(numMembers)
	iter, err := NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
	assert.Nil(t, err)
	runIterTests(initItem, consConfigs, iter, toRun, t)

	if !config.RunAllTests {
		logging.Print("Exiting test early because config.RunAllTests is false")
		return
	}

	// clear disk enough fail that progress is stopped
	if to.ConsType != types.MvCons3Type { // TODO fix
		logging.Print("All fail and recover from disk")
		to.ClearDiskOnRestart = false
		to.NumFailProcs = to.NumTotalProcs
		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, toRun, t)
	}

	// clear disk: the test below may not work because as processes are restarting they may send new values
	// as they recover // TODO make a way to wait before making proposals after recovering
	// TODO for now only test with total order because with causal sending different proposals can hault termination
	if to.OrderingType == types.Total {
		// restart from disk enough remain live to continue
		logging.Print("Less than 1/3 fail and clear their disk")
		to.ClearDiskOnRestart = true
		to.NumFailProcs = utils.GetOneThirdBottom(numMembers)
		iter, err = NewTestOptIter(AllOptions, consConfigs, NewSingleIter(tconfig, to))
		assert.Nil(t, err)
		runIterTests(initItem, consConfigs, iter, toRun, t)

		// logging.Print("Half fail and clear their disk")
		//to.ClearDiskOnRestart = true
		//to.NumFailProcs = to.NumTotalProcs / 2
		//iter, err = types.NewTestOptIter(types.AllOptions, consConfigs, types.NewSingleIter(types.AllTestConfig, to))
		//assert.Nil(t, err)
		//runIterTests(initItem, consConfigs, iter, toRun, t)
	}
}

// runIterTests runs the tests from the iterator
func runIterTests(initItem consinterface.ConsItem, consConfigs ConfigOptions,
	iter *TestOptIter, toRun []int, t testing.TB) {

	var i int
	prv, hasNxt := iter.Next()
	if len(toRun) == 0 || utils.ContainsInt(toRun, 0) {
		logging.Printf("\nRunning test #%v\n", i)
		_, err := prv.CheckValid(prv.ConsType, consConfigs.GetIsMV())
		assert.Nil(t, err)

		runConsDebug(initItem, consConfigs.GetBroadcastFunc(prv.ByzType), consConfigs, prv, t)
	}

	for hasNxt {
		var nxt types.TestOptions
		nxt, hasNxt = iter.Next()
		i++

		if len(toRun) == 0 || utils.ContainsInt(toRun, i) {
			logging.Printf("Running test #%v\n", i)
			logging.Print("Changing config: ", prv.StringDiff(nxt))
			_, err := nxt.CheckValid(nxt.ConsType, consConfigs.GetIsMV())
			assert.Nil(t, err)
			runConsDebug(initItem, consConfigs.GetBroadcastFunc(nxt.ByzType), consConfigs, nxt, t)
		}
		prv = nxt
	}

}

func runConsDebug(initItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	options ConfigOptions,
	to types.TestOptions,
	t assert.TestingT) {

	sv := sig.SleepValidate
	sig.SetSleepValidate(config.TestSleepValidate)

	labels := pprof.Labels("consFunc", to.ConsType.String())
	pprof.Do(context.Background(), labels, func(_ context.Context) {
		RunConsType(initItem, broadcastFunc, options, to, t)
	})
	sig.SetSleepValidate(sv)
}
