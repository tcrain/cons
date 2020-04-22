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
package dual

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func NewDualPrivThrsh() func() (sig.Priv, error) {
	func1 := bls.GetBlsPartPrivFunc()
	func2 := bls.GetBlsPartPrivFunc2()

	return func() (sig.Priv, error) {
		p1, err := func1()
		if err != nil {
			return nil, err
		}
		p2, err := func2()
		if err != nil {
			return nil, err
		}
		return NewDualprivCustomThresh(p1, p2, types.SecondarySignature, types.SecondarySignature)
	}
}

func TestDualPartShareVerify(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.TestPartThrsh(NewDualPrivThrsh(), types.NormalSignature, config.Thrsht, config.Thrshn, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestDualPartSecondaryShareVerify(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() {
		sig.TestPartThrsh(NewDualPrivThrsh(), types.SecondarySignature, config.Thrsht2, config.Thrshn2, t)
	},
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestDualPartialSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(NewDualPrivThrsh(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualPartialSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewDualPrivThrsh(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestDualPartialSecondarySign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewDualPrivThrsh(), types.SecondarySignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestDualPartialSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewDualPrivThrsh(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualPartialSecondarySerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewDualPrivThrsh(), types.SecondarySignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualPartialVRF(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(NewDualPrivThrsh(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualPartialMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(NewDualPrivThrsh(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualPartialSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(NewDualPrivThrsh(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}
