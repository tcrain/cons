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

package rpcsetup

import (
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"net/rpc"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/network"
)

type ParRegClientInterface interface {
	GenBlsShared(id, idx, numThresh int) error
	GetBlsShared(id, idx int) (*BlsSharedMarshalIndex, error)
	GenDSSShared(id, numNonMembers, numThresh int) error
	GetDSSShared(id int) (*ed.CoinSharedMarshaled, error)
	RegisterParticipant(id int, parInfo *network.ParticipantInfo) error
	GetParticipants(id int, pub sig.PubKeyStr) (pi [][]*network.ParticipantInfo, err error)
	GetAllParticipants(id int) (pi []*network.ParticipantInfo, err error)
	Close() (err error)
}

// ParRegClient acts as an interface to a ParticipantRegister through rpc.
type ParRegClient struct {
	cli *rpc.Client
}

// NewParRegClient creates a new client connection to the ParticipantRegister at address.
func NewParRegClient(address string) (pc *ParRegClient, err error) {
	pc = &ParRegClient{}
	pc.cli, err = rpc.DialHTTP("tcp", address)
	return
}

// NewParReg calls NewParReg on the server.
func (pc *ParRegClient) NewParReg(ps PregSetup) error {
	var reply None
	return pc.cli.Call("Preg.NewParReg", ps, &reply)
}

// Close closes the connection.
func (pc *ParRegClient) Close() (err error) {
	return pc.cli.Close()
}

// GenBlsShared calls ParticipantRegister.GenBlsShared on the server
func (pc *ParRegClient) GenBlsShared(id, idx, numThresh int) error {
	var reply None
	return pc.cli.Call("Preg.GenBlsShared",
		struct{ Id, Idx, NumThresh int }{id, idx, numThresh},
		&reply)
}

// GetBlsShared calls ParticipantRegister.GetBlsShared on the server
func (pc *ParRegClient) GetBlsShared(id, idx int) (*BlsSharedMarshalIndex, error) {
	var reply *BlsSharedMarshalIndex
	err := pc.cli.Call("Preg.GetBlsShared", struct{ Id, Idx int }{id, idx}, &reply)
	return reply, err
}

// GenDSSShared calls ParticipantRegister.GenDSSShared on the server
func (pc *ParRegClient) GenDSSShared(id, numNonMembers, numThresh int) error {
	var reply None
	return pc.cli.Call("Preg.GenDSSShared",
		struct{ Id, NumNonMembers, NumThresh int }{id, numNonMembers, numThresh},
		&reply)
}

// GetDSSShared calls ParticipantRegister.GetDSSShared on the server
func (pc *ParRegClient) GetDSSShared(id int) (*ed.CoinSharedMarshaled, error) {
	var reply *ed.CoinSharedMarshaled
	err := pc.cli.Call("Preg.GetDSSShared", id, &reply)
	return reply, err
}

// RegisterParticipant calls ParticipantRegister.RegisterParticipant on the server
func (pc *ParRegClient) RegisterParticipant(id int, parInfo *network.ParticipantInfo) error {
	var reply None
	return pc.cli.Call("Preg.RegisterParticipant",
		struct {
			Id      int
			ParInfo *network.ParticipantInfo
		}{id, parInfo},
		&reply)
}

// GetParticipants call ParticipantRegister.GetParticipants on the server
func (pc *ParRegClient) GetParticipants(id int, pub sig.PubKeyStr) (pi [][]*network.ParticipantInfo, err error) {
	err = pc.cli.Call("Preg.GetParticipants",
		struct {
			Id  int
			Pub sig.PubKeyStr
		}{id, pub}, &pi)
	return
}

// GetAllParticipants call ParRegClient.GetAllParticipants on the server
func (pc *ParRegClient) GetAllParticipants(id int) (pi []*network.ParticipantInfo, err error) {
	err = pc.cli.Call("Preg.GetAllParticipants", id, &pi)
	return
}
