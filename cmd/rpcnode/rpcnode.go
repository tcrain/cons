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
This package contains the code for running consensus processes through a process running an RPC interface.
*/
package main

import (
	"flag"
	"fmt"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"net"
	"net/http"
	"net/rpc"
	"runtime"
	"sync"

	// _ "net/http/pprof"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/rpcsetup"
)

func init() {
	// go func() {
	// 	logging.Info(http.ListenAndServe("localhost:6060", nil))
	// }()
}

// NewRunningCons generates a new RunningCons object.
func NewRunningCons(myIP string) *RunningCons {
	return &RunningCons{
		running:  make(map[int]*rpcsetup.SingleConsSetup),
		doneChan: make(chan int, 1),
		myIP:     myIP}
}

// RunningCons sotres multiple consensus processes accessable through an RPC interface.
type RunningCons struct {
	mutex            sync.RWMutex
	running          map[int]*rpcsetup.SingleConsSetup
	doneChan         chan int
	myIP             string
	setInitialConfig bool
	setInitialHash   bool
	shared           *rpcsetup.Shared
}

// Reset clears all running consensus object.
func (rc *RunningCons) Reset(none rpcsetup.None, res *rpcsetup.None) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	for _, scs := range rc.running {
		rpcsetup.Reset(scs.SCS)
	}
	rc.shared = nil
	rc.setInitialHash = false
	rc.setInitialConfig = false

	rc.running = make(map[int]*rpcsetup.SingleConsSetup)

	// perform a garbage collection
	runtime.GC()

	logging.Info("Num goroutines after reset: %v", runtime.NumGoroutine())

	return nil
}

func (rc *RunningCons) GetCausalDecisions(i int, causalDec *rpcsetup.CausalDecisions) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	var scs *rpcsetup.SingleConsSetup
	var ok bool
	if scs, ok = rc.running[i]; !ok {
		return fmt.Errorf("Missing index %v", i)
	}

	causalDec.Root, causalDec.OrderedDecisions = cons.GetCausalDecisions(scs.SCS.MemberCheckerState)
	return nil
}

// GetDecisions returns the decided values of the consensus processes.
func (rc *RunningCons) GetDecisions(i int, dec *[][]byte) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	var scs *rpcsetup.SingleConsSetup
	var ok bool
	if scs, ok = rc.running[i]; !ok {
		return fmt.Errorf("Missing index %v", i)
	}
	var err error
	*dec, err = rpcsetup.GetDecisions(scs.SCS)
	if err != nil {
		return err
	}
	return nil
}

// GetResults returns the statistics of the consensus processes.
func (rc *RunningCons) GetResults(i int, res *rpcsetup.RpcResults) error {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	var scs *rpcsetup.SingleConsSetup
	var ok bool
	if scs, ok = rc.running[i]; !ok {
		return fmt.Errorf("Missing index %v", i)
	}
	stats := scs.SCS.Stats.MergeLocalStats(int(types.ComputeNumRounds(scs.To)))
	res.Stats = &stats
	// res.NwStats = scs.SCS.Stats.(*stats.BasicNwStats)
	return nil
}

// AllStart starts all the consensus processes tracked by this physical node.
func (rc *RunningCons) AllStart(in rpcsetup.None, none *rpcsetup.None) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Recovered panic: %v", r)
		}
	}()

	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	var wg sync.WaitGroup
	for _, scs := range rc.running {
		wg.Add(1)
		go func(scs *rpcsetup.SingleConsSetup) {
			rpcsetup.RunSingleConsType(scs)
			wg.Done()
		}(scs)
	}
	wg.Wait()
	return
}

// AllFinished is called after all consensus processes have completed running on this physical node to shutdown the processes.
func (rc *RunningCons) AllFinished(in rpcsetup.None, none *rpcsetup.None) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	for _, scs := range rc.running {
		rpcsetup.AllFinished(scs.SCS)
	}
	return nil
}

// NewMvConsRunning initalizes a multivalue consensus process at this node.
func (rc *RunningCons) NewConsRunning(nra rpcsetup.NewRunningArgs, none *rpcsetup.None) error {
	parReg, err := rpcsetup.NewParRegClient(nra.ParRegAddress)
	if err != nil {
		return err
	}

	rc.mutex.Lock()

	if _, ok := rc.running[nra.I]; ok {
		rc.mutex.Unlock()
		return fmt.Errorf("Already registered index %v", nra.I)
	}

	var shared *rpcsetup.Shared
	if nra.To.SharePubsRPC {
		if rc.shared == nil {
			rc.shared = &rpcsetup.Shared{}
		}
		shared = rc.shared
	} else {
		shared = &rpcsetup.Shared{}
	}

	scs := &rpcsetup.SingleConsSetup{
		I:                nra.I,
		To:               nra.To,
		ParReg:           parReg,
		MyIP:             rc.myIP,
		Mutex:            &rc.mutex,
		SetInitialConfig: &rc.setInitialConfig,
		SetInitialHash:   &rc.setInitialHash,
		Shared:           shared,
	}
	rc.running[nra.I] = scs
	rc.mutex.Unlock()

	err = rpcsetup.RunCons(scs)
	if err != nil {
		return err
	}

	return nil
}

// Exit shuts down the RunningCons process.
func (rc *RunningCons) Exit(none rpcsetup.None, reply *rpcsetup.None) error {
	rc.doneChan <- 1
	return nil
}

func main() {
	var port int
	var myIP string
	flag.IntVar(&port, "p", config.RPCNodePort, "tcp port to run rpc server")
	flag.StringVar(&myIP, "i", "", "ip of server visible to other nodes")
	flag.Parse()

	fmt.Printf("Running rpc consensus node setup on port %v\n", port)

	rc := NewRunningCons(myIP)
	err := rpc.Register(rc)
	if err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		panic(err)
	}

	go func() {
		err = http.Serve(l, nil)
		if err != nil {
			panic(err)
		}
	}()

	<-rc.doneChan
}
