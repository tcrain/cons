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

package binconsrnd1

import (
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/types"
	"testing"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

// CreateAuxProofItems is used during unit tests to create a list of signed message supporting a binvalue and round.
func CreateAuxProofItems(idx types.ConsensusIndex, round types.ConsensusRound, allowSupportCoin bool, binVal types.BinVal, bct cons.ConsTestItems, t *testing.T) []*deserialized.DeserializedItem {
	ret := make([]*deserialized.DeserializedItem, len(bct.PrivKeys))

	for i := range bct.PrivKeys {
		priv := bct.PrivKeys[i]
		inHdr := messagetypes.NewAuxProofMessage(allowSupportCoin)
		inHdr.Round = round
		inHdr.BinVal = binVal

		hdr := sig.NewMultipleSignedMsg(idx, priv.GetPub(), inHdr)
		if _, err := hdr.Serialize(messages.NewMessage(nil)); err != nil {
			panic(err)
		}
		mySig, err := priv.GenerateSig(hdr, nil, hdr.GetSignType())
		if err != nil {
			panic(err)
		}
		hdr.SetSigItems([]*sig.SigItem{mySig})
		hdrs := make([]messages.MsgHeader, 1)
		hdrs[0] = hdr
		msg := messages.InitMsgSetup(hdrs, t)
		encMsg := sig.NewUnencodedMsg(msg)

		dser := sig.NewMultipleSignedMsg(idx, priv.GetPub(), messagetypes.NewAuxProofMessage(allowSupportCoin))
		_, err = (dser).Deserialize(msg, types.IntIndexFuns)
		if err != nil {
			t.Error(err)
		}

		ret[i] = &deserialized.DeserializedItem{
			Index:          idx,
			HeaderType:     messages.HdrAuxProof,
			Header:         dser,
			IsDeserialized: true,
			Message:        encMsg}
	}
	return ret
}

// CreateCoinItems is used during unit tests to create a list of signed messages for a coin.
func CreateCoinItems(idx types.ConsensusIndex, round types.ConsensusRound,
	bct cons.ConsTestItems, t *testing.T) []*deserialized.DeserializedItem {

	ret := make([]*deserialized.DeserializedItem, len(bct.PrivKeys))

	for i := range bct.PrivKeys {
		priv := bct.PrivKeys[i]
		inHdr := messagetypes.NewCoinMessage()
		inHdr.Round = round

		hdr := sig.NewMultipleSignedMsg(idx, priv.GetPub(), inHdr)
		if _, err := hdr.Serialize(messages.NewMessage(nil)); err != nil {
			panic(err)
		}
		mySig, err := priv.GenerateSig(hdr, nil, hdr.GetSignType())
		if err != nil {
			panic(err)
		}
		hdr.SetSigItems([]*sig.SigItem{mySig})
		hdrs := make([]messages.MsgHeader, 1)
		hdrs[0] = hdr
		msg := messages.InitMsgSetup(hdrs, t)
		encMsg := sig.NewUnencodedMsg(msg)

		dser := sig.NewMultipleSignedMsg(idx, priv.GetPub(), messagetypes.NewCoinMessage())
		_, err = (dser).Deserialize(msg, types.IntIndexFuns)
		if err != nil {
			t.Error(err)
		}

		ret[i] = &deserialized.DeserializedItem{
			Index:          idx,
			HeaderType:     hdr.GetID(),
			Header:         dser,
			IsDeserialized: true,
			Message:        encMsg}
	}
	return ret
}
