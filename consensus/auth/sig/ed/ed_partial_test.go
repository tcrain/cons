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
	"flag"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
	// "sort"
	// "github.com/tcrain/cons/consensus/messages"
)

var thrshdssshare *CoinShared
var thrshn = flag.Int("thrshn", config.Thrshn, "benchmark threshold signature n")
var thrsht = flag.Int("thrsht", config.Thrsht, "benchmark threshold signature t")

func init() {
	thrshdssshare = NewCoinShared(*thrshn, 0, *thrsht)
}

func GetEdPartPrivFunc() func() (sig.Priv, error) {
	index := sig.TestKeyIndex
	return func() (sig.Priv, error) {
		edThresh := NewEdThresh(index, thrshdssshare)
		index += 1
		return NewEdPartPriv(edThresh)
	}
}

func TestEdPartPrintStats(t *testing.T) {
	t.Log("ED threshold stats")
	sig.SigTestPrintStats(GetEdPartPrivFunc(), t)
}

func TestEdPartSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(GetEdPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdPartSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(GetEdPartPrivFunc(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestEdPartSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(GetEdPartPrivFunc(), types.NormalSignature, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdCoinProofSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(GetEdPartPrivFunc(), types.CoinProof, t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

// func TestEdPartTestMessageSerialize(t *testing.T) {
// 	RunFuncWithConfigSetting(func() { SigTestTestMessageSerialize(GetEdPartPrivFunc(), t) },
// 		WithBothBool, WithFalse, WithFalse, WithBothBool)
// }

func TestEdPartMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(GetEdPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEdPartSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(GetEdPartPrivFunc(), t) },
		types.WithTrue, types.WithFalse, types.WithFalse, types.WithBothBool)
}
