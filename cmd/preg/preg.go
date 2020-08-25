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
This package contains code for running a Preg object that represents a participant register object that is accessable through RPC.
*/
package main

import (
	"flag"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"net"
	"net/http"
	"net/rpc"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/rpcsetup"
)

const maxParRegs = 10

// Preg represents a participant register object that is accessable through RPC.
type Preg struct {
	parregs  []*network.ParticipantRegister // the participant register
	doneChan chan int                       // used to indicate when it shuts down
}

// NewParReg generates a new Preg object.
func NewParReg() *Preg {
	return &Preg{doneChan: make(chan int, 1), parregs: make([]*network.ParticipantRegister, maxParRegs)}
}

// NewParReg resets the state of the participat register.
func (preg *Preg) NewParReg(ps rpcsetup.PregSetup, reply *rpcsetup.None) error {
	if ps.ID >= maxParRegs {
		return fmt.Errorf("invalid par reg id")
	}
	preg.parregs[ps.ID] = network.NewParReg(ps.ConnType, ps.NodeCount)
	return nil
}

func (preg *Preg) checkID(id int) error {
	if id >= len(preg.parregs) || preg.parregs[id] == nil {
		return NilPreg
	}
	return nil
}

// GenDSSShared generates a shared threshold key object for the given threshold.
// This is not safe since it is centralized.
func (preg *Preg) GenDSSShared(input struct{ Id, numNonMembers, NumThresh int }, reply *rpcsetup.None) error {
	if err := preg.checkID(input.Id); err != nil {
		return err
	}
	return preg.parregs[input.Id].GenDSSShared(input.numNonMembers, input.NumThresh)
}

// GenBlsShared generates a shared BLS threshold key object for the given threshold.
// This is not safe since it is centralized.
func (preg *Preg) GenBlsShared(input struct{ Id, Idx, NumThresh int }, reply *rpcsetup.None) error {
	if err := preg.checkID(input.Id); err != nil {
		return err
	}
	return preg.parregs[input.Id].GenBlsShared(input.Idx, input.NumThresh)
}

// GetBlsShared returns the BlsSharedMarshal object generated by GenBlsShared plus the index of the
// call to GetBlsShared.
func (preg *Preg) GetBlsShared(input struct{ Id, Idx int }, reply **network.BlsSharedMarshalIndex) error {
	if err := preg.checkID(input.Id); err != nil {
		return err
	}
	keyIdx, bss := preg.parregs[input.Id].GetBlsShared(input.Idx)
	*reply = &network.BlsSharedMarshalIndex{KeyIndex: keyIdx, Idx: input.Idx, BlsShared: bss}
	return nil
}

// GetDSSShared returns the CoinSharedMarshaled object generated by GenDSSShared.
func (preg *Preg) GetDSSShared(id int, reply **ed.CoinSharedMarshaled) error {
	if err := preg.checkID(id); err != nil {
		return err
	}
	ret := preg.parregs[id].GetDSSShared()
	*reply = &ret
	return nil
}

var NilPreg = fmt.Errorf("Must call NewParReg first")

// RegisterParticipant calls network.ParticipantRegister.RegisterParticipant
func (preg *Preg) RegisterParticipant(input struct {
	Id      int
	ParInfo *network.ParticipantInfo
},
	reply *rpcsetup.None) error {

	if err := preg.checkID(input.Id); err != nil {
		return err
	}
	return preg.parregs[input.Id].RegisterParticipant(input.ParInfo)
}

// GetParticipants calls network.ParticipantRegister.GetParticipants
func (preg *Preg) GetParticipants(input struct {
	Id  int
	Pub sig.PubKeyStr
},
	reply *[][]*network.ParticipantInfo) error {

	if err := preg.checkID(input.Id); err != nil {
		return err
	}
	var err error
	*reply, err = preg.parregs[input.Id].GetParticipants(input.Pub)
	return err
}

// GetAllParticipants calls network.ParticipantRegister.GetAllParticipants
func (preg *Preg) GetAllParticipants(id int, reply *[]*network.ParticipantInfo) error {
	if err := preg.checkID(id); err != nil {
		return err
	}
	var err error
	*reply, err = preg.parregs[id].GetAllParticipants()
	return err
}

// Exit shuts down the participant register
func (preg *Preg) Exit(none rpcsetup.None, reply *rpcsetup.None) error {
	preg.doneChan <- 1
	return nil
}

func main() {
	var port int
	flag.IntVar(&port, "p", config.ParRegRPCPort, "tcp port to run rpc server")
	flag.Parse()

	fmt.Printf("Running participat register on port %v\n", port)

	pr := NewParReg()
	err := rpc.Register(pr)
	if err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		panic(err)
	}
	go func() {
		err := http.Serve(l, nil)
		if err != nil {
			panic(err)
		}
	}()
	<-pr.doneChan
}
