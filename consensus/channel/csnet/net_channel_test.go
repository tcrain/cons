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
package csnet

import (
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/stats"
	"testing"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/forwardchecker"
	"github.com/tcrain/cons/consensus/consinterface/memberchecker"
	"github.com/tcrain/cons/consensus/types"
)

// TestNetChannel checks tcp and udp connections are able to send messages to eachother
func TestNetChannel(t *testing.T) {
	testNetChannelNetwork(types.TCP, true, t)
	t.Log("Completed test with tcp")
	// testNetChannelNetwork(types.UDP, false, t)
	t.Log("Completed test with udp")
}

func testNetChannelNetwork(network types.NetworkProtocolType, encryptChannels bool, t *testing.T) {
	testProcs := make([]channelinterface.MainChannel, config.ProcCount)
	connInfos := make([]channelinterface.NetNodeInfo, config.ProcCount)
	connStatus := make([]*ConnStatus, config.ProcCount)

	memberCheckerState := make([]*consinterface.ConsInterfaceState, config.ProcCount)
	privs := make([]sig.Priv, config.ProcCount)
	for i := 0; i < config.ProcCount; i++ {
		connStatus[i] = NewConnStatus(network)
		priv, err := ec.NewEcpriv()
		if err != nil {
			panic(err)
		}
		privs[i] = priv
		connInfos[i] = channelinterface.NetNodeInfo{AddrList: channelinterface.NewConInfoList(network),
			Pub: priv.GetPub()}

		memberCheckerState[i] = consinterface.NewConsInterfaceState(&TestConsItem{},
			&TestMemberChecker{},
			&memberchecker.NoSpecialMembers{},
			&TestMessageState{},
			forwardchecker.NewAllToAllForwarder(),
			0, false, priv.GetPub().New(),
			consinterface.NormalBroadcast, nil)
		memberCheckerState[i].SetInitSM(TestSM{})
	}

	DeserFunc = DeserializeMessage // set the deserialize test function
	for i := 0; i < config.ProcCount; i++ {
		bt := channelinterface.NewSimpleBehaviorTracker()
		tp, ci := NewNetMainChannel(privs[i], connInfos[i], config.ProcCount-1, network,
			config.DefaultMsgProcesThreads, &TestConsItem{}, bt,
			encryptChannels, 0, &stats.BasicNwStats{})
		testProcs[i] = tp
		connInfos[i].AddrList = ci
		tp.SetMemberCheckerState(memberCheckerState[i])
		memberCheckerState[i].SetMainChannel(tp)
	}

	for i := 0; i < config.ProcCount; i++ {
		// for _, conInfo := range connInfos {
		// testProcs[i].AddExternalNode(conInfo)
		// }
		for j := 0; j < config.ProcCount; j++ {
			if j == i {
				continue
			}
			testProcs[i].AddExternalNode(connInfos[j])
			err := testProcs[i].MakeConnections([]sig.Pub{connInfos[j].Pub})
			assert.Nil(t, err)
			testProcs[i].StartMsgProcessThreads()
		}
	}
	for i := 0; i < config.ProcCount; i++ {
		err := testProcs[i].WaitUntilAtLeastSendCons(config.ProcCount - 1)
		assert.Nil(t, err)
	}

	channelinterface.RunChannelTest(testProcs, config.MaxBenchRounds-1, t)
}
