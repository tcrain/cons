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
This package runs a benchmark over the network given a test configuration stored in a json file.
There must be a Preg (participant register) process running an accessable through RPC.
Also each node that will be running the test must have a rpcnode.RunningCons process running and accessable through RPC.
*/
package main

import (
	"flag"
	"fmt"
	"github.com/tcrain/cons/consensus/consgen"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	// "time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/rpcsetup"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

func InitiateParticipantRegisters(to types.TestOptions, cli *rpcsetup.ParRegClient) error {
	// the network propagation configuration
	connType := network.NetworkPropagationSetup{
		AdditionalP2PNetworks: to.AdditionalP2PNetworks,
		NPT:                   to.NetworkType,
		FanOut:                to.FanOut}
	if err := cli.NewParReg(rpcsetup.PregSetup{ID: 0, ConnType: connType, NodeCount: to.NumTotalProcs}); err != nil {
		return err
	}

	// request forwarder uses a second network for proposal messages
	if to.NetworkType == types.RequestForwarder {
		connType := network.NetworkPropagationSetup{
			NPT:    types.P2p,
			FanOut: to.FanOut}
		if err := cli.NewParReg(rpcsetup.PregSetup{ID: 1, ConnType: connType, NodeCount: to.NumTotalProcs}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var ipFile string
	var parRegIP string
	var optionsFile string
	var resultsFolder string

	flag.StringVar(&ipFile, "f", "ipfile", "Path to file containing list of rpc node addresses, 1 per line")
	flag.StringVar(&parRegIP, "p", fmt.Sprintf("127.0.0.1:%v", config.ParRegRPCPort),
		"Address of participant register")
	flag.StringVar(&optionsFile, "o", "tofile.json", "Path to file containing list of options")
	flag.StringVar(&resultsFolder, "r", "./benchresults/"+time.Now().Format(time.RFC3339), "Path to folder where results should be stored")
	flag.Parse()

	logging.Printf("Storing results in folder: %v\n", resultsFolder)
	toFolder := filepath.Join(resultsFolder, "testoptions")
	if err := os.MkdirAll(toFolder, os.ModePerm); err != nil {
		panic(err)
	}

	logging.Print("Loading test options from file: ", optionsFile)
	to, err := types.GetTestOptions(optionsFile)
	if err != nil {
		logging.Error(err)
		panic(err)
	}
	logging.Infof("Loaded options: %s\n", to)

	if to.TestID == 0 {
		to.TestID = rand.Uint64()
		logging.Printf("Set test ID to %v\n", to.TestID)
	}

	if to, err = to.CheckValid(to.ConsType, consgen.GetConsConfig(to).GetIsMV()); err != nil {
		logging.Error("invalid config ", err)
		panic(err)
	}

	logging.Info("Reading ip file: ", ipFile)
	ips, err := utils.ReadTCPIPFile(ipFile, true)
	if err != nil {
		logging.Error(err)
		panic(err)
	}
	logging.Print("Loaded ips: ", ips)

	logging.Infof("Connecting to participant register at: %v", parRegIP)
	var preg *rpcsetup.ParRegClient
	preg, err = rpcsetup.NewParRegClient(parRegIP)
	if err != nil {
		logging.Error(err)
		panic(err)
	}

	logging.Infof("Initiating participant registers")
	err = InitiateParticipantRegisters(to, preg)
	if err != nil {
		panic(err)
	}
	logging.Print("Connected and initiated participant register")

	switch to.SigType {
	case types.EDCOIN:
		err = preg.GenDSSShared(0, to.NumNonMembers, cons.GetCoinThresh(to))
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		logging.Print("Generated shared EDCOIN at participant register")
	case types.TBLS:
		err = preg.GenBlsShared(0, 0, cons.GetTBLSThresh(to))
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		logging.Print("Generated shared BLS at participant register")
	case types.CoinDual:
		err = preg.GenBlsShared(0, 0, cons.GetCoinThresh(to))
		if err != nil {
			logging.Error(err)
			panic(err)
		}
	case types.TBLSDual:
		thrshPrimary, thrshSecondary := cons.GetDSSThresh(to)
		err = preg.GenBlsShared(0, 0, thrshPrimary)
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		err = preg.GenBlsShared(0, 1, thrshSecondary)
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		logging.Print("Generated shared BLS at participant register")
	}

	rpcServers := make([]*rpcsetup.RPCNodeClient, len(ips))
	var mutex sync.RWMutex
	var wg sync.WaitGroup
	for i, ip := range ips {
		wg.Add(1)
		go func(ip string, idx int) {
			logging.Infof("Connecting to rpc node at: %v to reset\n", ip)
			var err error
			ser, err := rpcsetup.NewRPCNodeClient(ip)
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			mutex.Lock()
			rpcServers[idx] = ser
			mutex.Unlock()

			mutex.RLock()
			err = rpcServers[idx].Reset()
			mutex.RUnlock()
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			logging.Infof("Performed reset at rpc node at: %v\n", ip)
			wg.Done()
		}(ip, i)
	}
	wg.Wait()

	// setup each participant in the consensus
	// note that depnding on the generalconfig, each physical node may have multiple consensus participants
	var maxServer int
	for i := 0; i < to.NumTotalProcs; i++ {
		wg.Add(1)
		go func(idx int) {
			mod := idx % len(ips)
			logging.Info("Calling cons setup at index: ", idx)
			nra := rpcsetup.NewRunningArgs{I: idx,
				To:            to,
				ParRegAddress: parRegIP}

			err := rpcServers[mod].NewConsRunning(nra)
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			logging.Info("Done cons setup at index: ", idx)

			wg.Done()
		}(i)
		maxServer = i
	}
	wg.Wait()

	maxServer = utils.Min(len(ips), maxServer+1)
	// time.Sleep(1 * time.Second)

	// start eh consensus at each node
	logging.Printf("\nRunning general config: \n%s\n", to)
	// var wg sync.WaitGroup
	for i := 0; i < maxServer; i++ {
		wg.Add(1)
		go func(idx int) {
			logging.Infof("Starting cons node index %v\n", idx)
			err = rpcServers[idx].AllStart()
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			logging.Infof("Done cons node index %v\n", idx)
			wg.Done()
		}(i)
	}
	wg.Wait()
	logging.Print("Done consensus at all nodes")

	for i := 0; i < maxServer; i++ {
		wg.Add(1)
		go func(idx int) {
			logging.Infof("Stopping cons node index %v\n", idx)
			err = rpcServers[idx].AllFinished()
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			logging.Infof("Done stopping cons node index %v\n", idx)
			wg.Done()
		}(i)
	}
	wg.Wait()

	var sm consinterface.StateMachineInterface
	if to.CheckDecisions {

		// check the decided values are valid for the state machine
		pi, err := preg.GetAllParticipants(0)
		if err != nil {
			logging.Error("error getting participant info ", err)
			panic(err)
		}
		epi := make([][]byte, len(pi))
		for i, nxt := range pi {
			epi[i] = nxt.ExtraInfo
		}
		priv, err := cons.MakeKey(to)
		if err != nil {
			logging.Error("error generating private key", err)
			panic(err)
		}

		// Generate a state machine to validate the decisions
		preHeader := make([]messages.MsgHeader, 0)
		eis := (generalconfig.ExtraInitState)(cons.ConsInitState{IncludeProofs: to.IncludeProofs,
			Id: uint32(0), N: uint32(to.NumTotalProcs)})
		generalConfig := consinterface.CreateGeneralConfig(0, eis, nil, to, preHeader, priv)

		switch to.OrderingType {
		case types.Total:
			// check the decide values are all equal
			decs := make([][][]byte, to.NumTotalProcs)
			for i := 0; i < to.NumTotalProcs; i++ {
				wg.Add(1)
				go func(idx int) {
					logging.Infof("Getting decisions cons node index %v\n", idx)
					mod := idx % len(ips)
					res, err := rpcServers[mod].GetDecisions(idx)
					if err != nil {
						logging.Error(err)
						panic(err)
					}
					decs[idx] = res
					logging.Infof("Done getting decisions cons node index %v\n", idx)
					wg.Done()
				}(i)
			}
			wg.Wait()

			sm = cons.VerifyDecisions(to, epi, to.ConsType, priv, generalConfig, decs)
		case types.Causal:
			// check the decide values are all equal
			decs := make([]rpcsetup.CausalDecisions, to.NumTotalProcs)
			for i := 0; i < to.NumTotalProcs; i++ {
				wg.Add(1)
				go func(idx int) {
					logging.Infof("Getting decisions cons node index %v\n", idx)
					mod := idx % len(ips)
					res, err := rpcServers[mod].GetCausalDecisions(idx)
					if err != nil {
						logging.Error(err)
						panic(err)
					}
					decs[idx] = res
					logging.Infof("Done getting decisions cons node index %v\n", idx)
					wg.Done()
				}(i)
			}
			wg.Wait()
			roots := make([]*utils.StringNode, len(decs))
			for i, nxt := range decs {
				roots[i] = nxt.Root
			}
			cons.VerifyCausalDecisions(to, epi, priv, generalConfig,
				roots, decs[0].OrderedDecisions)
		}
	}
	results := make([]rpcsetup.RpcResults, to.NumTotalProcs)
	for i := 0; i < to.NumTotalProcs; i++ {
		wg.Add(1)
		go func(idx int) {
			logging.Infof("Getting results cons node index %v\n", idx)
			mod := idx % len(ips)
			res, err := rpcServers[mod].GetResults(idx)
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			results[idx] = res
			logging.Infof("Done getting results cons node index %v\n", idx)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// get the statistics.
	statsList := make([]stats.MergedStats, to.NumTotalProcs)
	for i := 0; i < to.NumTotalProcs; i++ {
		statsList[i] = *results[i].Stats
	}
	// save the profile results
	profFolder := filepath.Join(resultsFolder, "profile")
	if err = os.MkdirAll(profFolder, os.ModePerm); err != nil {
		panic(err)
	}
	for i, nxt := range statsList {
		// profile filename is profile_{cpu/memstart/memend}_{procIndex}_{testID}_{nodes}_{constype}
		if to.CPUProfile {
			cpuPrfName := fmt.Sprintf("profile_cpu_%v_%v_%v_%d.out", i, to.TestID, to.NumTotalProcs, to.ConsType)
			if err = ioutil.WriteFile(filepath.Join(profFolder, cpuPrfName), nxt.CpuProfile, os.ModePerm); err != nil {
				logging.Error(err)
				panic(err)
			}
		}
		if to.MemProfile {
			memPrfName := fmt.Sprintf("profile_memstart_%v_%v_%v_%d.out", i, to.TestID, to.NumTotalProcs, to.ConsType)
			if err = ioutil.WriteFile(filepath.Join(profFolder, memPrfName), nxt.StartMemProfile, os.ModePerm); err != nil {
				logging.Error(err)
				panic(err)
			}

			memPrfName = fmt.Sprintf("profile_memend_%v_%v_%v_%d.out", i, to.TestID, to.NumTotalProcs, to.ConsType)
			if err = ioutil.WriteFile(filepath.Join(profFolder, memPrfName), nxt.EndMemProfile, os.ModePerm); err != nil {
				logging.Error(err)
				panic(err)
			}
		}
	}

	logging.Printf(cons.GetStatsString(to, statsList, true, resultsFolder))
	if sm != nil {
		totalTime := statsList[0].FinishTime.Sub(statsList[0].StartTime)
		logging.Print(sm.StatsString(totalTime)) // TODO merge these stats also???
	}

	// Store the test options files
	if err := types.TOToDisk(toFolder, to); err != nil {
		panic(err)
	}

	// Reset the servers
	for i, ip := range ips {
		wg.Add(1)
		go func(ip string, idx int) {
			err = rpcServers[idx].Reset()
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			logging.Infof("Performed reset at rpc node at: %v\n", ip)
			if err = rpcServers[idx].Close(); err != nil {
				logging.Error("Error closing rpc connection: ", err)
			}
			wg.Done()
		}(ip, i)
	}
	// Close the connection to the participant register
	if err = preg.Close(); err != nil {
		logging.Error("Error closing participant register connection: ", err)
	}

	wg.Wait()

}
