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

package ec

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"

	// "sort"
	// "crypto/ecdsa"
	// "crypto/elliptic"
	// "crypto/rand"
	// "math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tcrain/cons/consensus/messages"
)

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestECPrintStats(t *testing.T) {
	t.Log("EC stats")
	sig.SigTestPrintStats(NewEcpriv, t)
}

func TestEcSharedSecret(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestComputeSharedSecret(NewEcpriv, t) },
		types.WithBothBool, types.WithBothBool, types.WithBothBool, types.WithBothBool)
}

func TestEcEncode(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(NewEcpriv, t) },
		types.WithFalse, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEcFromBytes(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(NewEcpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEcSort(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(NewEcpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEcSign(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(NewEcpriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
}

func TestEcSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(NewEcpriv, types.NormalSignature, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEcVRF(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(NewEcpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

// func TestEcTestMessageSerialize(t *testing.T) {
// 	RunFuncWithConfigSetting(func() { SigTestTestMessageSerialize(NewEcpriv, t) },
// 		WithBothBool, WithFalse, WithFalse, WithBothBool)
// }

func TestMultiSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(NewEcpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestSignTestMsgSerialize(t *testing.T) {
	sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(NewEcpriv, t) },
		types.WithBothBool, types.WithFalse, types.WithFalse, types.WithBothBool)
}

func TestEcSerializePriv(t *testing.T) {
	priv, err := NewEcpriv()
	assert.Nil(t, err)

	hash := types.GetHash([]byte("sign this message"))
	signMsg := &sig.MultipleSignedMessage{Hash: hash}

	asig, err := priv.(*Ecpriv).Sign(signMsg)
	assert.Nil(t, err)

	pub := priv.GetPub()

	hdrs := make([]messages.MsgHeader, 3)
	hdrs[0] = priv.(*Ecpriv)
	hdrs[1] = pub
	hdrs[2] = asig

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())

	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	assert.Equal(t, size, msg.Len())

	ht, err := msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, priv.(*Ecpriv).GetID())

	priv2 := &Ecpriv{}
	_, err = priv2.Deserialize(msg, types.NilIndexFuns)
	assert.Nil(t, err)

	ht, err = msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, ht, pub.GetID())

	pub2 := priv.GetPub().New()
	_, err = pub2.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)

	if sig.UsePubIndex {
		id, err := pub2.GetPubID()
		assert.Nil(t, err)
		oldid, err := pub.GetPubID()
		assert.Nil(t, err)
		assert.Equal(t, id, oldid)
		pub2 = priv.GetPub()
	}

	pubString, err := pub.GetPubString()
	assert.Nil(t, err)
	pubString2, err := pub2.GetPubString()
	assert.Nil(t, err)
	assert.Equal(t, pubString, pubString2)

	pubBytes, err := pub.GetPubBytes()
	assert.Nil(t, err)
	pubBytes2, err := pub2.GetPubBytes()
	assert.Nil(t, err)
	assert.Equal(t, pubBytes, pubBytes2)

	ht, err = msg.PeekHeaderType()
	assert.Nil(t, err)
	assert.Equal(t, asig.GetID(), ht)

	sig2 := &Ecsig{}
	_, err = sig2.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)

	sig3, err := priv2.Sign(signMsg)
	assert.Nil(t, err)

	v, err := pub2.VerifySig(&sig.MultipleSignedMessage{Hash: hash}, asig)
	assert.Nil(t, err)
	assert.True(t, v)

	v, err = pub2.VerifySig(&sig.MultipleSignedMessage{Hash: hash}, sig2)
	assert.Nil(t, err)
	assert.True(t, v)

	v, err = pub2.VerifySig(&sig.MultipleSignedMessage{Hash: hash}, sig3)
	assert.Nil(t, err)
	assert.True(t, v)
}
