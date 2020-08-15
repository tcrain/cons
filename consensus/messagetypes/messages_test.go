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

package messagetypes

import (
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/messages"
)

//////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

func TestCreateMsg(t *testing.T) {
	var tstMsg testMessage

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = tstMsg
	hdrs[1] = tstMsg

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())

	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	if size != msg.Len() {
		t.Error("Got invalid size", size, msg.Len())
	}

	ht, err := msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	ht, err = msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)

}

func TestCreateBadMsg1(t *testing.T) {
	var tstMsg testMessage
	var badTstMsg badTestMessage1

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = tstMsg
	hdrs[1] = badTstMsg

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())
	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	if size != msg.Len() {
		t.Error("Got invalid size", size, msg.Len())
	}

	ht, err := msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	ht, err = msg.PeekHeaderTypeAt(4)
	if err != nil {
		t.Error(err)
	}
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	if err != ErrInvalidStringMsg {
		t.Error("Should have failed deserialization")
	}

}

func TestCreateBadMsg2(t *testing.T) {
	var tstMsg testMessage
	var badTstMsg badTestMessage2

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = tstMsg
	hdrs[1] = badTstMsg

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())
	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	if size != msg.Len() {
		t.Error("Got invalid size", size, msg.Len())
	}

	ht, err := msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	ht, err = msg.PeekHeaderTypeAt(4)
	if err != nil {
		t.Error(err)
	}
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	if err != types.ErrNotEnoughBytes {
		t.Error("Should have failed deserialization")
	}

}

func TestCreateBadMsg3(t *testing.T) {
	var tstMsg testMessage
	var badTstMsg badTestMessage3

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = tstMsg
	hdrs[1] = badTstMsg

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())
	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	if size != msg.Len() {
		t.Error("Got invalid size", size, msg.Len())
	}

	ht, err := msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	ht, err = msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	if err != types.ErrInvalidHash {
		t.Error("Should have failed hash check deserialization")
	}

}

func TestCreateTimeoutMsg(t *testing.T) {
	var tstMsg TestMessageTimeout

	hdrs := make([]messages.MsgHeader, 2)
	hdrs[0] = tstMsg
	hdrs[1] = tstMsg

	buff, err := messages.CreateMsg(hdrs)
	assert.Nil(t, err)

	msg := messages.NewMessage(buff.GetBytes())

	size, err := msg.PopMsgSize()
	assert.Nil(t, err)
	if size != msg.Len() {
		t.Error("Got invalid size", size, msg.Len())
	}

	ht, err := msg.PeekHeaderTypeAt(4)
	assert.Nil(t, err)
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)
	ht, err = msg.PeekHeaderTypeAt(4)
	if err != nil {
		t.Error(err)
	}
	if ht != tstMsg.GetID() {
		t.Error("Got invalid msg type", ht, tstMsg.GetID())
	}
	_, err = tstMsg.Deserialize(msg, types.IntIndexFuns)
	assert.Nil(t, err)

}
