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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func NewDualPrivBls() (sig.Priv, error) {
	return NewDualpriv(bls.NewBlspriv, bls.NewBlspriv, types.NormalSignature, types.SecondarySignature)
}

func TestDualSharedSecret(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestComputeSharedSecret(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualEncode(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(NewDualPrivBls, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestDualFromBytes(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewDualPrivBls, types.NormalSignature, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithFalse)
}

func TestDualSecondarySign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewDualPrivBls, types.SecondarySignature, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithFalse)
}

func TestDualGetRand(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestRand(NewDualPrivBls, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestDualSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewDualPrivBls, types.NormalSignature, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualSecondary(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewDualPrivBls, types.SecondarySignature, t) },
		types.WithTrue, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualVRF(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

// func TestDualTestMessageSerialize(t *testing.T) {
// 	RunFuncWithConfigSetting(func() { SigTestTestMessageSerialize(NewDualPrivBls, t) },
// 		WithBothBool, WithBothBool, WithBothBool, WithBothBool)
// }

func TestDualMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestDualSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(NewDualPrivBls, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}
