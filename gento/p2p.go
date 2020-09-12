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

package gento

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/mvcons2"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/parse"
)

func GenP2PTO() {
	genP2PTest("p2p-buffwd", true, 1)
}

func genP2PTest(folderName string, sleepCrypto bool, nxtID uint64) uint64 {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:           []types.SigType{types.EC},
		CoinTypes:          []types.CoinType{types.NoCoinType},
		MemberCheckerTypes: []types.MemberCheckerType{types.TrueMC},
		RotateCoordTypes:   types.WithTrue,
	}, baseMVOptions)
	consTypes = []types.ConsType{types.MvCons2Type}

	ct := mvAll2All
	ct.MaxRounds = 10
	ct.NetworkType = types.P2p
	ct.FanOut = 8
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = sleepCrypto
	ct.WarmUpInstances = 5
	ct.CPUProfile = false
	ct.KeepPast = 1
	ct.NumMsgProcessThreads = 3
	// set large timeouts since we don't have any faults for this test
	ct.MvConsTimeout = 100000
	ct.MvConsRequestRecoverTimeout = 100000
	ct.ProgressTimeout = 100000
	ct.ForwardTimeout = 100
	ct.RotateCord = true
	ct.MCType = types.TrueMC

	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	ct.BufferForwardType = types.FixedBufferForward
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByConsBuffFwd})
	return nxtID
}
