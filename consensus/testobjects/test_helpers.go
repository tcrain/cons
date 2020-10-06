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

package testobjects

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func CheckInitSupportMessage(mmc *MockMainChannel, index types.ConsensusInt, supportedIndex types.ConsensusInt,
	supportHash types.HashBytes, proposal []byte, t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		initMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.MvInitSupportMessage)
		if !bytes.Equal(initMsg.Proposal, proposal) || initMsg.SupportedIndex != supportedIndex || !bytes.Equal(initMsg.SupportedHash, supportHash) {
			t.Error("Sent invalid init msg: ", initMsg)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckInitMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	proposal []byte, t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		initMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.MvInitMessage)
		if !bytes.Equal(initMsg.Proposal, proposal) || initMsg.Round != round {
			t.Error("Sent invalid init msg: ", initMsg)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
		panic(1)
	}
}

func CheckEchoMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	hash types.HashBytes, t *testing.T) {
	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		echoMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.MvEchoHashMessage)
		if !bytes.Equal(echoMsg.ProposalHash, hash) || echoMsg.Round != round {
			t.Error("Sent invalid echo msg: ", echoMsg)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckCommitMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	hash types.HashBytes, t *testing.T) {
	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		commitMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.MvCommitMessage)
		if !bytes.Equal(commitMsg.ProposalHash, hash) || commitMsg.Round != round {
			t.Error("Sent invalid commit msg: ", commitMsg)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckNoSend(mmc *MockMainChannel, t *testing.T) {
	send := mmc.GetFirstHeader()
	if send != nil {
		t.Error("Expected to send nil, but sent message")
	}
}

func CheckAuxMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	binVal types.BinVal, t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		auxMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.AuxProofMessage)
		if auxMsg.Round != round || auxMsg.BinVal != binVal {
			t.Error("Sent invalid aux message: ", auxMsg)
			panic(1)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckBVMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	binVal types.BinVal, t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		switch auxMsg := sigMsg.GetBaseMsgHeader().(type) {
		case *messagetypes.BVMessage0:
			if auxMsg.Round != round || binVal != 0 {
				t.Error("Sent invalid aux message: ", auxMsg)
				panic(1)
			}
		case *messagetypes.BVMessage1:
			if auxMsg.Round != round || binVal != 1 {
				t.Error("Sent invalid aux message: ", auxMsg)
				panic(1)
			}
		default:
			t.Error("invalid internal header", auxMsg)
			panic(1)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckCoinMessage(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		auxMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.CoinMessage)
		if auxMsg.Round != round {
			t.Error("Sent invalid aux message: ", auxMsg)
			panic(1)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}

func CheckCoinMessage2(mmc *MockMainChannel, index types.ConsensusInt, round types.ConsensusRound,
	t *testing.T) {

	switch sigMsg := mmc.GetFirstHeader().(type) {
	case *sig.MultipleSignedMessage:
		if sigMsg.Index.Index != index {
			t.Error("Sent msg with invalid index", sigMsg.Index)
		}
		auxMsg := sigMsg.GetBaseMsgHeader().(*messagetypes.CoinMessage)
		if auxMsg.Round != round {
			t.Error("Sent invalid aux message: ", auxMsg)
			panic(1)
		}
	default:
		t.Errorf("Got invalid type message: %T", sigMsg)
	}
}
