// +build !windows

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

package qsafe

import (
	"github.com/open-quantum-safe/liboqs-go/oqs"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

// TODO shared secret gen
/*func TestQsafeSharedSecret(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() {sig.SigTestComputeSharedSecret(NewQsafePriv, t)},
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}
*/

func TestQsafeEncode(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(NewQsafePriv, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestQsafeFromBytes(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(NewQsafePriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestQsafeSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(NewQsafePriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestQsafeSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewQsafePriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestQsafeSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewQsafePriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(NewQsafePriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(NewQsafePriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

var sigMsg = []byte("sign this message")

// BenchmarkVerifyQsafe tests the cost to verify a Qsafe signature
func BenchmarkVerifyQsafe(b *testing.B) {

	var priv oqs.Signature
	priv.Init(QsafeName, nil)
	pub, err := priv.GenerateKeyPair()
	assert.Nil(b, err)
	buff, err := priv.Sign(sigMsg)
	assert.Nil(b, err)

	b.Log("Qsafe sig length", len(buff))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sig oqs.Signature
		sig.Init(QsafeName, nil)
		v, err := sig.Verify(sigMsg, buff, pub)
		assert.Nil(b, err)
		assert.True(b, v)
		sig.Clean()
	}
}

// BenchmarkSignQsafe is the tests how long it takes to sign a message.
func BenchmarkSignQsafe(b *testing.B) {

	var priv oqs.Signature
	priv.Init(QsafeName, nil)
	_, err := priv.GenerateKeyPair()
	assert.Nil(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		priv.Sign(sigMsg)
	}
}
