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
package localcoin

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// LocalCoin represents a local random coin.
type LocalCoin struct {
}

func NewLocalCoin() consinterface.CoinItemInterface {
	return &LocalCoin{}
}

func (sc1 *LocalCoin) GenerateCoinMessage(types.ConsensusRound, bool, consinterface.ConsItem,
	consinterface.CoinMessageStateInterface, consinterface.MessageState) (ret messages.MsgHeader) {

	return
}

// CheckCoinMessage should be called from within ProcessMessage of the ConsensusItem that is using this coin.
// It returns the round the coin corresponds to and true in first boolean position if made progress towards decision,
// or false if already decided, and return true in second position if the message should be forwarded.
// If the message is invalid an error is returned.
func (*LocalCoin) CheckCoinMessage(*channelinterface.DeserializedItem,
	bool, bool, consinterface.ConsItem, consinterface.CoinMessageStateInterface,
	consinterface.MessageState) (round types.ConsensusRound, ret messages.MsgHeader,
	progress, shouldForward bool, err error) {

	err = types.ErrInvalidHeader
	return
}

// GetHeader should a blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
func (LocalCoin) GetHeader(sig.Pub, *generalconfig.GeneralConfig,
	messages.HeaderID) (messages.MsgHeader, error) {

	return nil, types.ErrInvalidHeader
}

// GetBufferCount returns the thresholds for messages of type coin.
// The thresholds are n-t.
func (LocalCoin) GetBufferCount(messages.MsgIDHeader, *generalconfig.GeneralConfig,
	*consinterface.MemCheckers) (endThreshold int,
	maxPossible int, msgid messages.MsgID, err error) { // How many of the same msg to buff before forwarding

	return 0, 0, nil, types.ErrInvalidHeader
}
