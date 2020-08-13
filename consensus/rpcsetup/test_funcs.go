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
This package contains functions for running nodes over a network, who communicate using RPC for test setup. It also provided client interfaces to RPC functions.
See the cmd package for the running the actual binaries.
*/
package rpcsetup

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/dual"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/auth/sig/sleep"
	"github.com/tcrain/cons/consensus/consgen"
	"github.com/tcrain/cons/consensus/generalconfig"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/storage"
	"github.com/tcrain/cons/consensus/types"
)

// SingleConsSetup contains all the details for a single node to start running a test.
type SingleConsSetup struct {
	I      int                   // The index of the node in the test (note this is not used in consensus, consensus uses the index of the sorted pub key list)
	MyIP   string                // The ip address and port of the node
	To     types.TestOptions     // The test setup
	ParReg ParRegClientInterface // RPC connection to the participant register
	SCS    *SingleConsState      // Pointer to the actual state
	Mutex  *sync.RWMutex

	SetInitialConfig *bool
	SetInitialHash   *bool
}

// SingleConsState tracks the state of a single node when running an experiment over a network.
type SingleConsState struct {
	I                  int                            // The index of the node in the test (note this is not used in consensus, consensus uses the index of the sorted pub key list)
	MyIP               string                         // The ip address and port of the node
	PrivKey            sig.Priv                       // The private key
	RandKey            [32]byte                       // Key for random number generation
	PubKeys            sig.PubList                    // The sorted list of public keys
	DSSShared          *ed.CoinShared                 // Information for threshold keys
	To                 types.TestOptions              // The test setup
	TestProc           channelinterface.MainChannel   // The main channel
	NetNodeInfo        []channelinterface.NetNodeInfo // The local connection information
	StorageFileName    string                         // name of the storage file
	RetExtraParRegInfo [][]byte                       // Per process extra info (usually generated by the state machine, ex an init block)
	NumCons            []int                          // Number of connections to make

	FinishedChan     chan channelinterface.ChannelCloseType // Where we wait for cons to finish
	FailFinishedChan chan channelinterface.ChannelCloseType // Where we wait for failure cons to finish

	ConsState          cons.ConsStateInterface
	MemberCheckerState consinterface.ConsStateInterface
	BcastFunc          consinterface.ByzBroadcastFunc // Function to check if should act byzantine

	Sc    consinterface.ConsItem // The init consensus object
	Ds    storage.StoreInterface // The interface to the disk storage
	Stats stats.StatsInterface   // The statsistics tracking
	// NwStats              stats.NwStatsInterface                // Network statistics tracking
	ParReg     ParRegClientInterface        // RPC connection to the participant register
	ParRegsInt []network.PregInterface      // Interface to ParReg for using functions from cons package
	Gc         *generalconfig.GeneralConfig // the general config

	blsShare  *bls.BlsShared // state of shared generated keys when using threshold BLS
	blsShare2 *bls.BlsShared // state of shared generated keys when using threshold BLS
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////

func getAllPubKeys(scs *SingleConsState) error {
	// get all participants
	allPar, err := scs.ParReg.GetAllParticipants(0)
	if err != nil {
		return err
	}

	makeKeys := true
	switch scs.To.SigType {
	case types.TBLS, types.TBLSDual, types.EDCOIN, types.CoinDual: // threshold/coin keys are constructed below
		makeKeys = false
	}
	if scs.To.SleepCrypto { // For sleep crypto we use use the normal setup
		makeKeys = true
	}
	if makeKeys {
		for _, parInfo := range allPar {
			var pub sig.Pub
			pub, err = scs.PrivKey.GetPub().New().FromPubBytes(sig.PubKeyBytes(parInfo.Pub))
			if err != nil {
				return err
			}
			scs.PubKeys = append(scs.PubKeys, pub)
		}
	}
	if !scs.To.SleepCrypto {
		switch scs.To.SigType {
		case types.EDCOIN:
			dssThrsh := scs.DSSShared
			for i, nxtMem := range dssThrsh.MemberPoints {
				pub := ed.NewEdPartPub(sig.PubKeyIndex(i), nxtMem, ed.NewEdThresh(sig.PubKeyIndex(i), dssThrsh))
				scs.PubKeys = append(scs.PubKeys, pub)
			}
			for i, nxtNonMem := range dssThrsh.NonMemberPoints {
				numMembers := scs.To.NumTotalProcs - scs.To.NumNonMembers
				pub := ed.NewEdPartPub(sig.PubKeyIndex(i+numMembers),
					nxtNonMem, ed.NewEdThresh(sig.PubKeyIndex(i+numMembers), dssThrsh))
				scs.PubKeys = append(scs.PubKeys, pub)
			}
		case types.TBLS:
			// TBLS keys we construct directly from the BlsShared object
			for i := 0; i < scs.To.NumTotalProcs; i++ {
				scs.PubKeys = append(scs.PubKeys, bls.NewBlsPartPub(sig.PubKeyIndex(i), scs.blsShare.NumParticipants,
					scs.blsShare.NumThresh, scs.blsShare.PubPoints[i]))
			}
		case types.TBLSDual:
			priv := scs.PrivKey.(*dual.DualPriv)
			for i := 0; i < scs.To.NumTotalProcs; i++ {
				pub1 := bls.NewBlsPartPub(sig.PubKeyIndex(i), scs.blsShare.NumParticipants,
					scs.blsShare.NumThresh, scs.blsShare.PubPoints[i])
				pub2 := bls.NewBlsPartPub(sig.PubKeyIndex(i), scs.blsShare.NumParticipants,
					scs.blsShare.NumThresh, scs.blsShare2.PubPoints[i])
				dpub := priv.GetPub().New().(*dual.DualPub)
				dpub.SetPubs(pub1, pub2)
				scs.PubKeys = append(scs.PubKeys, dpub)
			}
		case types.CoinDual:
			priv := scs.PrivKey.(*dual.DualPriv)
			for i, parInfo := range allPar {
				// we get the ec pub from the bytes
				dp, err := priv.GetPub().New().FromPubBytes(sig.PubKeyBytes(parInfo.Pub))
				if err != nil {
					return err
				}
				// we gen the bls pub from the share
				blsPub := bls.NewBlsPartPub(sig.PubKeyIndex(i), scs.blsShare.NumParticipants,
					scs.blsShare.NumThresh, scs.blsShare.PubPoints[i])
				dpub := priv.GetPub().New().(*dual.DualPub)
				dpub.SetPubs(dp.(*dual.DualPub).Pub, blsPub)
				scs.PubKeys = append(scs.PubKeys, dpub)
			}
		}
	}
	for i, nxt := range scs.PubKeys {
		nxt.SetIndex(sig.PubKeyIndex(i))
	}
	return nil
}

func getDssShared(preg ParRegClientInterface) (*ed.CoinShared, error) {
	dssMarshaled, err := preg.GetDSSShared(0)
	if err != nil || dssMarshaled == nil {
		return nil, types.ErrInvalidSharedThresh
	}
	return dssMarshaled.PartialUnMartial()
}

func getBlsShared(idx int, preg ParRegClientInterface) (*bls.BlsShared, error) {
	blssi, err := preg.GetBlsShared(0, idx)
	if err != nil || blssi.BlsShared == nil {
		return nil, types.ErrInvalidSharedThresh
	}
	ret, err := blssi.BlsShared.PartialUnmarshal()
	return ret, err
}

func initialSetup(setup *SingleConsSetup) (err error) {
	setup.SCS = &SingleConsState{
		I:      setup.I,
		MyIP:   setup.MyIP,
		To:     setup.To,
		ParReg: setup.ParReg}
	scs := setup.SCS
	// TODO memory race here
	setup.Mutex.Lock()
	cons.SetTestConfigOptions(&scs.To, !*setup.SetInitialConfig)
	*setup.SetInitialConfig = true
	setup.Mutex.Unlock()
	scs.Stats = stats.GetStatsObject(scs.To.ConsType, scs.To.EncryptChannels)
	// scs.NwStats = &stats.BasicNwStats{}
	switch scs.To.SigType {
	case types.TBLS, types.TBLSDual, types.EDCOIN, types.CoinDual: // with TBLS the priv key is gererated in the shared state
	default:
		scs.PrivKey, err = cons.MakeKey(sig.PubKeyIndex(scs.I), scs.To)
		if err != nil {
			return
		}
	}

	sleepCrypto := setup.To.SleepCrypto
	switch setup.To.SigType {
	case types.EDCOIN:
		if sleepCrypto {
			scs.PrivKey, err = sleep.NewEDCoinPriv(sig.PubKeyIndex(setup.I), setup.To)
			if err != nil {
				return err
			}
		} else {
			// setup the threshold keys
			// we use a special path because we need to agree on the shared values
			dssShared, err := getDssShared(scs.ParReg)
			if err != nil {
				return err
			}
			edThresh := ed.NewEdThresh(sig.PubKeyIndex(scs.I), dssShared)
			scs.PrivKey, err = ed.NewEdPartPriv(edThresh)
			if err != nil {
				return err
			}
			scs.DSSShared = dssShared
		}
	case types.TBLS:
		numMembers := scs.To.NumTotalProcs - scs.To.NumNonMembers
		thrshn := numMembers
		thrsh := cons.GetTBLSThresh(scs.To)
		if sleepCrypto {
			scs.PrivKey, err = sleep.NewTBLSPriv(thrshn, thrsh, sig.PubKeyIndex(scs.I))
			if err != nil {
				return err
			}
		} else {
			// setup the threshold keys
			// we use a special path because we need to agree on the shared values
			// before creating the key
			// note that this is done in a centralized (unsafe) manner for efficiency
			// and we share all the private keys
			blss, err := getBlsShared(0, scs.ParReg)
			if err != nil {
				return err
			}

			if scs.I < numMembers {
				scs.PrivKey, err = cons.MakePrivBlsS(scs.To, thrsh, sig.PubKeyIndex(scs.I), blss)
				if err != nil {
					return err
				}
			} else {
				blsThresh := bls.NewNonMemberBlsThrsh(thrshn, thrsh, sig.PubKeyIndex(scs.I), blss.SharedPub)
				if scs.PrivKey, err = bls.NewBlsPartPriv(blsThresh); err != nil {
					return err
				}
			}
			scs.blsShare = blss
		}
	case types.TBLSDual:
		if sleepCrypto {
			scs.PrivKey, err = sleep.NewTBLSDualPriv(sig.PubKeyIndex(scs.I), scs.To)
			if err != nil {
				return err
			}
		} else {
			blss1, err := getBlsShared(0, scs.ParReg)
			if err != nil {
				return err
			}
			scs.blsShare = blss1
			blss2, err := getBlsShared(1, scs.ParReg)
			if err != nil {
				return err
			}
			scs.blsShare2 = blss2
			numMembers := scs.To.NumTotalProcs - scs.To.NumNonMembers
			if scs.I < numMembers {
				scs.PrivKey = cons.GenTBLSDualThreshPriv(sig.PubKeyIndex(scs.I), blss1, blss2, scs.To)
			} else {
				scs.PrivKey = cons.GenTBLSDualThreshPrivNonMember(sig.PubKeyIndex(scs.I), blss1, blss2, scs.To)
			}
		}
	case types.CoinDual:
		if sleepCrypto {
			scs.PrivKey, err = sleep.NewCoinDualPriv(sig.PubKeyIndex(scs.I), scs.To)
			if err != nil {
				return err
			}
		} else {
			numMembers := scs.To.NumTotalProcs - scs.To.NumNonMembers
			blss, err := getBlsShared(0, scs.ParReg)
			if err != nil {
				return err
			}
			scs.blsShare = blss
			var secondary sig.Priv
			thrsh := sig.GetCoinThresh(scs.To)
			if scs.I < numMembers {
				if secondary, err = cons.MakePrivBlsS(scs.To, thrsh, sig.PubKeyIndex(scs.I), blss); err != nil {
					panic(err)
				}
			} else {
				blsThreshSecondary := bls.NewNonMemberBlsThrsh(numMembers, thrsh, sig.PubKeyIndex(scs.I), blss.SharedPub)
				if secondary, err = bls.NewBlsPartPriv(blsThreshSecondary); err != nil {
					panic(err)
				}
			}
			var p1 sig.Priv
			p1, err = ec.NewEcpriv()
			if err != nil {
				return err
			}
			if scs.PrivKey, err = dual.NewDualprivCustomThresh(p1, secondary, types.SecondarySignature,
				types.NormalSignature); err != nil {
			}
		}
	}

	return
}

// RunCons sets up a consensus experiment for a single node given the input setup.
// Once setup is complete RunSingleConsType is called to run the experiment.
func RunCons(setup *SingleConsSetup) (err error) {
	err = initialSetup(setup)
	if err != nil {
		return
	}

	return setupSingleConsType(setup)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////

func runInitialSetup(setup *SingleConsSetup, t assert.TestingT) (err error) {
	scs := setup.SCS

	scs.PrivKey.SetIndex(sig.PubKeyIndex(scs.I))

	// Check if the generalconfig is valid
	consConfig := consgen.GetConsConfig(scs.To)
	scs.To, err = scs.To.CheckValid(scs.To.ConsType, consConfig.GetIsMV())
	if err != nil {
		return
	}
	scs.Sc = consgen.GetConsItem(scs.To)
	scs.BcastFunc = consConfig.GetBroadcastFunc(scs.To.ByzType)

	logging.Infof("Running cons %T at proc %v\n", scs.Sc, scs.I)

	randItem := rand.New(rand.NewSource(time.Now().UnixNano()))

	scs.StorageFileName = fmt.Sprint(randItem.Uint32())
	// stats := stats.GetStatsObject(scs.To.ConsType)
	scs.RandKey = cons.GenRandKey()

	// Only extra info for members
	// non-members will have nil extra info
	extraParRegInfo := cons.GenerateExtraParticipantRegisterState(scs.To, scs.PrivKey, int64(scs.I))
	// cons item
	preHeader := make([]messages.MsgHeader, 0)
	eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: scs.To.IncludeProofs,
		Id: uint32(scs.I), N: uint32(scs.To.NumTotalProcs)})

	// Remaining state
	scs.Gc = consinterface.CreateGeneralConfig(scs.I, eis, scs.Stats, scs.To,
		preHeader, scs.PrivKey)

	logging.Printf("Running cons %T with state machine %v\n", scs.Sc, scs.To.StateMachineType)

	// Open storage
	scs.Ds = cons.OpenTestStore(scs.To.StorageBuffer, scs.To.StorageType, cons.GetStoreType(scs.To),
		scs.I, scs.StorageFileName, true)

	// Partcipant registers
	scs.ParRegsInt = []network.PregInterface{NewPregClientWrapper(scs.ParReg, 0)}
	switch scs.To.NetworkType {
	case types.RequestForwarder:
		scs.ParRegsInt = append(scs.ParRegsInt, NewPregClientWrapper(scs.ParReg, 1))
	}

	// We want to use our external ip address for connections
	upConInfo := func(conInfo channelinterface.NetNodeInfo) {
		if scs.MyIP != "" {
			for i := range conInfo.AddrList {
				s := conInfo.AddrList[i].String()
				idx := strings.LastIndex(s, ":")
				if idx < 0 {
					panic("bad address")
				}
				conInfo.AddrList[i].Addr = scs.MyIP + s[idx:]
			}
		}
	}

	scs.TestProc, scs.NetNodeInfo = cons.GenerateMainChannel(t, scs.To, scs.Sc,
		scs.Gc, upConInfo, scs.Stats, extraParRegInfo, scs.ParRegsInt...)

	// get all the public keys
	logging.Info("Getting participant public keys")
	if err := getAllPubKeys(scs); err != nil {
		logging.Error(err)
		panic(err)
	}
	logging.Info("Done participant public keys")

	logging.Info("Adding connections")

	genPubFunc := func(pb sig.PubKeyBytes) sig.Pub {
		return network.GetPubFromList(pb, scs.PubKeys)
	}
	cons.RegisterOtherNodes(t, scs.To, scs.TestProc, genPubFunc, scs.ParRegsInt...)

	// Get extra participant info
	allParticipants, err := scs.ParRegsInt[0].GetAllParticipants()
	if len(allParticipants) != scs.To.NumTotalProcs {
		panic(fmt.Sprint(allParticipants, scs.To.NumTotalProcs))
	}
	scs.RetExtraParRegInfo = make([][]byte, len(allParticipants))
	for i, par := range allParticipants {
		scs.RetExtraParRegInfo[i] = par.ExtraInfo
	}

	// We need to compute the value of the initial hash // TODO clean this up
	if scs.To.OrderingType == types.Causal {
		initSM := cons.GenerateCausalStateMachine(scs.To, scs.PrivKey, scs.PrivKey.GetPub(),
			scs.RetExtraParRegInfo, int64(scs.I), scs.TestProc, nil, nil,
			scs.Gc)

		setup.Mutex.Lock()
		if !*setup.SetInitialHash {
			cons.SetInitIndex(initSM)
			*setup.SetInitialHash = true
		}
		setup.Mutex.Unlock()
	}

	// Will wait for procs on this chan to finish
	var finishChan chan channelinterface.ChannelCloseType
	scs.FinishedChan = make(chan channelinterface.ChannelCloseType, 1)
	finishChan = scs.FinishedChan
	if scs.I < scs.To.NumFailProcs {
		scs.FailFinishedChan = make(chan channelinterface.ChannelCloseType, 1)
		finishChan = scs.FailFinishedChan
	}

	// Generate the cons state
	scs.ConsState, scs.MemberCheckerState = cons.GenerateConsState(t, scs.To, scs.PrivKey, scs.RandKey, scs.PubKeys[0],
		scs.PubKeys, scs.I, scs.Sc, scs.Ds, scs.Stats, scs.RetExtraParRegInfo, scs.TestProc, finishChan, scs.Gc,
		scs.BcastFunc, scs.I < scs.To.NumFailProcs, scs.ParRegsInt...)

	return nil
}

// setupSingConsType finished the setup started by RunBinCons1 or RunMvCons1
func setupSingleConsType(setup *SingleConsSetup) (err error) {

	t := TPanic{}
	if err = runInitialSetup(setup, t); err != nil {
		return err
	}

	scs := setup.SCS

	// Connect the nodes to eachother
	for i := 0; i < scs.To.NumTotalProcs; i++ {
		switch scs.To.NetworkType {
		case types.RequestForwarder:
			scs.NumCons = make([]int, 2)
		default:
			scs.NumCons = make([]int, 1)
		}
	}
	cons.MakeConnections(t, scs.To, scs.TestProc, scs.MemberCheckerState,
		scs.NumCons, scs.Gc, scs.PubKeys, scs.ParRegsInt...)

	// Wait until all connected
	cons.WaitConnections(t, scs.To, scs.TestProc, scs.NumCons)

	return
}

// RunSingleConsType is called after the setup of RunBinCons1 or RunMvCons1 has completed on all nodes.
func RunSingleConsType(setup *SingleConsSetup) {
	scs := setup.SCS

	go func() { // Run the cons
		cons.RunMainLoop(scs.ConsState, scs.TestProc)
	}()

	if scs.I >= scs.To.NumFailProcs { // we are not a failure
		// Wait to finish
		<-scs.FinishedChan // wait to finish
		return
	}

	// We are a failure so we will restart after we have failed
	t := TPanic{}
	// Wait for the failure to finish
	<-scs.FailFinishedChan
	// We are a failure so we restart
	scs.TestProc.Close()
	err := scs.Ds.Close()
	assert.Nil(t, err)

	// Restart after the given timeout
	time.Sleep(time.Duration(scs.To.FailDuration) * time.Millisecond)

	// Open the storage for the restarted nodes
	scs.Ds = cons.OpenTestStore(scs.To.StorageBuffer, scs.To.StorageType, cons.GetStoreType(scs.To), scs.I,
		scs.StorageFileName, scs.To.ClearDiskOnRestart)
	cons.UpdateStorageAfterFail(scs.To, scs.Ds)

	scs.TestProc = cons.MakeMainChannel(scs.To, scs.PrivKey, scs.Sc, scs.Stats, scs.Gc, scs.NetNodeInfo)

	genPubFunc := func(pb sig.PubKeyBytes) sig.Pub {
		return network.GetPubFromList(pb, scs.PubKeys)
	}
	// Register the addresses of the other nodes
	cons.RegisterOtherNodes(t, scs.To, scs.TestProc, genPubFunc, scs.ParRegsInt...)

	// Generate the ConsState
	scs.ConsState, scs.MemberCheckerState = cons.GenerateConsState(t, scs.To, scs.PrivKey, scs.RandKey, scs.PubKeys[0],
		scs.PubKeys, scs.I, scs.Sc, scs.Ds, scs.Stats, scs.RetExtraParRegInfo, scs.TestProc, scs.FinishedChan, scs.Gc,
		scs.BcastFunc, false, scs.ParRegsInt...)

	// Make Connections
	cons.MakeConnections(t, scs.To, scs.TestProc, scs.MemberCheckerState,
		scs.NumCons, scs.Gc, scs.PubKeys, scs.ParRegsInt...)

	go func() { // Run the cons
		cons.RunMainLoop(scs.ConsState, scs.TestProc)
	}()

	<-scs.FinishedChan // wait to finish
}

// AllFinished is called after RunSingleConsType has completed running on all nodes to shutdown the node.
func AllFinished(scs *SingleConsState) {
	scs.TestProc.Close()

	//RunConsType(sciList, messageState, sc, spi, scRestart, spiRestart, privKeys, memberCheckers, nwStatsList, to, t)
	sString := fmt.Sprintf("Stats for proc %v: %v, %v", scs.I, (scs.Stats).String(), (scs.Stats).NwString())
	logging.Infof(sString)
}

// Reset is called after AllFinished to do remaining cleanup
func Reset(scs *SingleConsState) {

	scs.PrivKey.Clean()

	err := scs.Ds.Close()
	if err != nil {
		panic(err)
	}

	// Close the connection to the participant register
	if err = scs.ParReg.Close(); err != nil {
		logging.Error("Error closing participant register connection: ", err)
	}
}

// GetDecisions returns the decided values at the node. It should be called after RunSingleConsType has finished,
// and before AllFinished has been called.
func GetDecisions(scs *SingleConsState) ([][]byte, error) {

	return cons.GetDecisions(scs.To, scs.Sc, scs.Ds), nil
}
