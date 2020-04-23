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
	"net/rpc"
)

// RPCNodeClient creates a connection to a node running a RunningCons rpc server,
// (see cmd.RunninCons)
type RPCNodeClient struct {
	cli *rpc.Client
}

// NewRPCNodeClient creates a client connection to the node at address running a RunningCons rpc server.
func NewRPCNodeClient(address string) (rnc *RPCNodeClient, err error) {
	rnc = &RPCNodeClient{}
	rnc.cli, err = rpc.DialHTTP("tcp", address)
	return
}

// GetDecisions calls RunningCons.GetDecisions for index i at the remote node.
func (rnc *RPCNodeClient) GetDecisions(i int) (dec [][]byte, err error) {
	err = rnc.cli.Call("RunningCons.GetDecisions", i, &dec)
	return
}

// GetDecisions calls RunningCons.GetDecisions for index i at the remote node.
func (rnc *RPCNodeClient) GetCausalDecisions(i int) (dec CausalDecisions, err error) {
	err = rnc.cli.Call("RunningCons.GetCausalDecisions", i, &dec)
	return
}

// GetResults calls RunningCons.GetResults for index i at the remote node.
func (rnc *RPCNodeClient) GetResults(i int) (res RpcResults, err error) {
	err = rnc.cli.Call("RunningCons.GetResults", i, &res)
	return
}

// AllStart calls RunningCons.AllStart at the remote node.
func (rnc *RPCNodeClient) AllStart() (err error) {
	var in, res None
	err = rnc.cli.Call("RunningCons.AllStart", in, &res)
	return
}

// AllFinished calls RunningCons.AllFinished at the remote node.
func (rnc *RPCNodeClient) AllFinished() (err error) {
	var in, res None
	err = rnc.cli.Call("RunningCons.AllFinished", in, &res)
	return
}

// Reset calls RunningCons.Reset at the remote node.
func (rnc *RPCNodeClient) Reset() (err error) {
	var in, res None
	err = rnc.cli.Call("RunningCons.Reset", in, &res)
	return
}

// Close closes the connection.
func (rnc *RPCNodeClient) Close() (err error) {
	return rnc.cli.Close()
}

// NewMvCons1Running calls RunningCons.NewMvConsRunning at the remote node.
func (rnc *RPCNodeClient) NewConsRunning(nra NewRunningArgs) (err error) {
	var res None
	err = rnc.cli.Call("RunningCons.NewConsRunning", nra, &res)
	return
}

// Exit calls RunningCons.Exit at the remote node.
func (rnc *RPCNodeClient) Exit() (err error) {
	var in, res None
	err = rnc.cli.Call("RunningCons.Exit", in, &res)
	return
}
