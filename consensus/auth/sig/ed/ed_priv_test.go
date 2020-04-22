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
package ed

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
	// "sort"
	// "github.com/tcrain/cons/consensus/messages"
)

func TestEdSharedSecret(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestComputeSharedSecret(NewEdpriv, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestEdEncode(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(NewEdpriv, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdFromBytes(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(NewEdpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(NewEdpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewEdpriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestEdGetRand(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestRand(NewEdpriv, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestEdSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewEdpriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

// func TestEdTestMessageSerialize(t *testing.T) {
// 	RunFuncWithConfigSetting(func() { SigTestTestMessageSerialize(NewEdpriv, t) },
// 		WithBothBool, WithFalse, WithFalse, WithBothBool)
// }

func TestEdMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(NewEdpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(NewEdpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}
