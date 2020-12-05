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
	"fmt"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

// None is used as input to an RPC call that takes no arguments.
// It is used so the functions follow the needed go RPC interface.
type None int

// PregSetup is used as the agrument to create a new ParticipantRegister through rpc
type PregSetup struct {
	ID        int                             // identifier of the par reg
	ConnType  network.NetworkPropagationSetup // The network propagation type
	NodeCount int                             // The number of participants in the consensus
}

// NewRunningArgs is used as the argument to create a new consensus participant for an experiment.
type NewRunningArgs struct {
	I             int               // The index of the node in the test (note this is not used in consensus, consensus uses the index of the sorted pub key list)
	To            types.TestOptions // The options of the test
	ParRegAddress string            // The address of the participant register
}

// CausalDecisions represents the decided values of a causally ordered test.
type CausalDecisions struct {
	Root             *utils.StringNode
	OrderedDecisions [][]byte
}

// RpcResults is used to transport the results of an experiment in a struct as the result of a rpc call.
// TODO this is hardcoded to binary/simple nw stats, should make more general.
type RpcResults struct {
	Stats *stats.MergedStats // Binary consensus statistics
	// NwStats  *stats.BasicNwStats  // Network statistics
	SMStats consinterface.SMStats
}

type TPanic struct{}

func (tp TPanic) Errorf(format string, args ...interface{}) {
	logging.Errorf(format, args...)
	panic(fmt.Errorf(format, args...))
}
