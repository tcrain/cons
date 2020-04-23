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

package consinterface

import (
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

type CoinItemInterface interface {
	BasicConsItem

	// CheckCoinMessage should be called from within ProcessMessage of the ConsensusItem that is using this coin.
	// It returns the round the coin corresponds to and true in first boolean position if made progress towards decision,
	// or false if already decided, and return true in second position if the message should be forwarded.
	// A message is returned if a message should be sent.
	// If the message is invalid an error is returned.
	CheckCoinMessage(deser *channelinterface.DeserializedItem, isLocal bool, alwaysGenerate bool, consItem ConsItem,
		coinMsgState CoinMessageStateInterface, msgState MessageState) (round types.ConsensusRound,
		ret messages.MsgHeader, progress, shouldForward bool, err error)

	GenerateCoinMessage(round types.ConsensusRound, alwaysGenerate bool, consItem ConsItem,
		coinMsgState CoinMessageStateInterface, msgState MessageState) (ret messages.MsgHeader)
}

type CoinMessageStateInterface interface {
	// CheckFinishedMessage checks if the message is for the coin and if the coin is already know.
	// If so true is returned, false otherwise.
	CheckFinishedMessage(deser *channelinterface.DeserializedItem) bool

	// GetCoinSignType returns what type of signature is used to sign coin messages.
	GetCoinSignType() types.SignType

	// GotMsg is called by the MessageState of the consensus using this coin.
	// deser is the deserialized message object.
	// It returns an error if the message is invalid.
	GotMsg(msgState MessageState, deser *channelinterface.DeserializedItem,
		gc *generalconfig.GeneralConfig, mc *MemCheckers) (types.ConsensusRound, error)

	// GetCoins returns the set of binary coin values that are currently valid.
	GetCoins(round types.ConsensusRound) []types.BinVal

	New(idx types.ConsensusIndex, presets []struct {
		Round types.ConsensusRound
		Val   types.BinVal
	}) CoinMessageStateInterface
}
