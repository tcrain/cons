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

func GenAll2AllSimple() {
	var consTypes []types.ConsType
	optsSig := cons.ReplaceNilFields(cons.OptionStruct{
		SigTypes:  []types.SigType{types.EC},
		CoinTypes: []types.CoinType{types.NoCoinType},
	}, baseMVOptions)

	consTypes = []types.ConsType{types.MvCons2Type}

	folderName := "all2all"

	ct := mvAll2All
	ct.StopOnCommit = types.Immediate
	ct.IncludeProofs = false
	ct.SleepCrypto = true
	ct.WarmUpInstances = 1
	ct.CPUProfile = false
	ct.MvConsTimeout = 200

	nxtID := genTO(1, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	optsSig = cons.ReplaceNilFields(cons.OptionStruct{
		RandMemberCheckerTypes: []types.RndMemberType{types.VRFPerCons,
			types.KnownPerCons, types.VRFPerMessage},
	}, optsSig)
	ct.RndMemberCount = 10
	ct.GenRandBytes = true
	nxtID = genTO(nxtID, folderName, ct, consTypes, []cons.ConfigOptions{mvcons2.MvCons2Config{}},
		optsSig, nil)

	genGenSets(folderName, []parse.GenSet{parse.GenPerNodeByRndMemberType})

}
