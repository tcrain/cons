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

func TestSchnorrPrintStats(t *testing.T) {
	t.Log("Schnorr stats")
	sig.SigTestPrintStats(NewSchnorrpriv, t)
}

func TestSchnorrSharedSecret(t *testing.T) {
	sig.SigTestComputeSharedSecret(NewSchnorrpriv, t)
}

func TestSchnorrFromBytes(t *testing.T) {
	sig.SigTestFromBytes(NewSchnorrpriv, t)
}

func TestSchnorrSort(t *testing.T) {
	sig.SigTestSort(NewSchnorrpriv, t)
}

func TestSchnorrSign(t *testing.T) {
	sig.SigTestSign(NewSchnorrpriv, types.NormalSignature, t)
}

func TestSchnorrSerialize(t *testing.T) {
	sig.SigTestSerialize(NewSchnorrpriv, types.NormalSignature, t)
}

// func TestSchnorrTestMessageSerialize(t *testing.T) {
// 	SigTestTestMessageSerialize(NewSchnorrpriv, t)
// }

func TestSchnorrMultiSignTestMsgSerialize(t *testing.T) {
	sig.SigTestMultiSignTestMsgSerialize(NewSchnorrpriv, t)
}

func TestSchnorrSignTestMsgSerialize(t *testing.T) {
	sig.SigTestSignTestMsgSerialize(NewSchnorrpriv, t)
}
