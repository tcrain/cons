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

package bls

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func getBlsThresh() *BlsThrsh {
	index := sig.PubKeyIndex(0)
	return NewBlsThrsh(*thrshn, *thrsht, index, Thrshblsshare.PriScalars[index], Thrshblsshare.PubPoints[index], Thrshblsshare.SharedPub)
}

func TestBlsThreshPrintStats(t *testing.T) {
	t.Log("BLS Thresh stats")
	sig.SetUseMultisig(false)
	sig.SigTestPrintStats(GetBlsPartPrivFunc(), t)
}

func TestBlsPartShareVerify(t *testing.T) {
	blsThrsh := getBlsThresh()
	sig.RunFuncWithConfigSetting(func() {
		sig.TestPartThrsh(GetBlsPartPrivFunc(), types.NormalSignature, blsThrsh.GetT(), blsThrsh.GetN(), t)
	},
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestBlsPartialSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(GetBlsPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestBlsPartialSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(GetBlsPartPrivFunc(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestBlsPartialSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(GetBlsPartPrivFunc(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestBlsCoinProofSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(GetBlsPartPrivFunc(), types.CoinProof, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestBlsPartialVRF(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(GetBlsPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestBlsPartialMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(GetBlsPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestBlsPartialSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(GetBlsPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}
