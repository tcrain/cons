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

package knowncoin

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"sync"
)

// KnownCoinMsgState represents the state of messages implementing a strong coin
// through the use of threshold signatures.
type MsgState struct {
	mutex sync.Mutex

	coinVal       map[types.ConsensusRound]types.BinVal
	fixedCoinRand *rand.Rand // Used if we are using a fixed coin values for experiment repeatability
	fixedCoinUint uint64     // Used if we are using a fixed coin values for experiment repeatability
}

// NewKnownCoinMsgState generates a new KnownCoinMsgState object.
func NewKnownCoinMsgState(isMv bool,
	_ *generalconfig.GeneralConfig) *MsgState {

	_ = isMv // TODO
	var fixedRand *rand.Rand
	var fixedCoinUint uint64
	fixedRand = rand.New(rand.NewSource(config.CoinSeed))
	fixedCoinUint = fixedRand.Uint64()

	return &MsgState{
		fixedCoinRand: fixedRand,
		fixedCoinUint: fixedCoinUint}
}

// New creates a new empty KnownCoinMsgState object for the consensus index idx.
func (sms *MsgState) New(_ types.ConsensusIndex, presets []struct {
	Round types.ConsensusRound
	Val   types.BinVal
}) consinterface.CoinMessageStateInterface {

	var fixedCoinUint uint64
	fixedCoinUint = sms.fixedCoinRand.Uint64()
	ret := &MsgState{
		fixedCoinRand: sms.fixedCoinRand,
		fixedCoinUint: fixedCoinUint,
		coinVal:       make(map[types.ConsensusRound]types.BinVal)}

	for _, nxt := range presets {
		ret.coinVal[nxt.Round] = nxt.Val
	}
	return ret
}

// GetCoinSignType returns what type of signature is used to sign coin messages
func (sms *MsgState) GetCoinSignType() types.SignType {
	return types.NormalSignature
}

// GetCoins returns the set of binary coin values that are currently valid.
func (sms *MsgState) GetCoins(round types.ConsensusRound) []types.BinVal {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	return []types.BinVal{sms.getCoin(round)}
}

func (sms *MsgState) getCoin(round types.ConsensusRound) types.BinVal {
	if val, ok := sms.coinVal[round]; ok {
		return val
	}
	val := utils.GetBitAt(uint(round)%64, sms.fixedCoinUint)
	sms.coinVal[round] = val
	return val
}

// CheckFinishedMessage checks if the message is for the coin and if the coin is already known.
// If so true is returned, false otherwise.
func (sms *MsgState) CheckFinishedMessage(deser *channelinterface.DeserializedItem) bool {

	return false
}

// GotMsg processes messages of type CoinMessage. Once an n-t threshold of these messages have been received
// from different processes the value of the coin is revealed.
func (sms *MsgState) GotMsg(consinterface.MessageState, *channelinterface.DeserializedItem,
	*generalconfig.GeneralConfig, *consinterface.MemCheckers) (types.ConsensusRound, error) {

	return 0, types.ErrInvalidMsgSize
}
