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
package strongcoin1

import (
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"sync"
)

type roundInfo struct {
	gotCoin  bool
	coinVal  types.BinVal
	sentCoin bool // set to true once the coin broadcast has been sent
}

// StrongCoin1MsgState represents the state of messages implementing a strong coin
// through the use of threshold signatures.
type MsgState struct {
	mutex sync.Mutex

	roundStructs  map[types.ConsensusRound]*roundInfo
	fixedCoinRand *rand.Rand // Used if we are using a fixed coin values for experiment repeatability
	fixedCoinUint uint64     // Used if we are using a fixed coin values for experiment repeatability
}

// NewStrongCoin1MsgState generates a new StrongCoin1MsgState object.
func NewStrongCoin1MsgState(isMv bool,
	gc *generalconfig.GeneralConfig) *MsgState {

	_ = isMv // TODO
	var fixedRand *rand.Rand
	var fixedCoinUint uint64
	if gc.UseFixedSeed {
		fixedRand = rand.New(rand.NewSource(config.CoinSeed))
		fixedCoinUint = fixedRand.Uint64()
	}

	return &MsgState{
		fixedCoinRand: fixedRand,
		fixedCoinUint: fixedCoinUint}
}

// New creates a new empty StrongCoin1MsgState object for the consensus index idx.
func (sms *MsgState) New(_ types.ConsensusIndex, presets []struct {
	Round types.ConsensusRound
	Val   types.BinVal
}) consinterface.CoinMessageStateInterface {

	var fixedCoinUint uint64
	if sms.fixedCoinRand != nil {
		fixedCoinUint = sms.fixedCoinRand.Uint64()
	}
	ret := &MsgState{
		fixedCoinRand: sms.fixedCoinRand,
		fixedCoinUint: fixedCoinUint,
		roundStructs:  make(map[types.ConsensusRound]*roundInfo)}

	for _, nxt := range presets {
		ri := ret.getRoundStruct(nxt.Round)
		ri.coinVal = nxt.Val
		ri.gotCoin = true
		ri.sentCoin = true
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

	roundInfo := sms.getRoundStruct(round)
	if roundInfo.gotCoin {
		return []types.BinVal{roundInfo.coinVal}
	}
	return nil
}

func (sms *MsgState) getRoundStruct(round types.ConsensusRound) (ret *roundInfo) {
	if ret = sms.roundStructs[round]; ret == nil {
		ret = &roundInfo{}
		sms.roundStructs[round] = ret
	}
	return
}

// CheckFinishedMessage checks if the message is for the coin and if the coin is already known.
// If so true is returned, false otherwise.
func (sms *MsgState) CheckFinishedMessage(deser *channelinterface.DeserializedItem) bool {

	if deser.HeaderType == messages.HdrCoin {
		round := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.CoinMessage).Round
		if len(sms.GetCoins(round)) > 0 {
			return true
		}
	}
	return false
}

// GotMsg processes messages of type CoinMessage. Once an n-t threshold of these messages have been received
// from different processes the value of the coin is revealed.
func (sms *MsgState) GotMsg(msgState consinterface.MessageState, deser *channelinterface.DeserializedItem,
	gc *generalconfig.GeneralConfig, mc *consinterface.MemCheckers) (types.ConsensusRound, error) {

	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	var round types.ConsensusRound
	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.CoinMessage:
		round = w.Round
		roundStruct := sms.getRoundStruct(w.Round)
		if !roundStruct.gotCoin {
		checkCoin:
			var coinThrsh int
			if gc.UseTp1CoinThresh {
				coinThrsh = mc.MC.GetFaultCount() + 1
			} else {
				coinThrsh = mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
			}
			item, err := msgState.GetThreshSig(w, coinThrsh, mc)
			switch err {
			case nil:
				// sanity check
				sigCount, err := msgState.GetSigCountMsgHeader(w, mc)
				if err != nil {
					panic(err)
				}
				if sigCount < coinThrsh {
					panic(fmt.Sprintf("got %v sigs, expected at least %v", sigCount, coinThrsh))
				}
				switch gc.UseFixedSeed {
				case true: // we use a predetermined coin so benchmarks have the same results
					roundStruct.coinVal = utils.GetBitAt(uint(w.Round)%64, sms.fixedCoinUint)
				default:
					roundStruct.coinVal = item.Sig.GetRand()
				}
				logging.Infof("id %v, index %v, round %v, coin %v\n", gc.TestIndex, msgState.GetIndex().Index,
					w.Round, roundStruct.coinVal)
				roundStruct.gotCoin = true

			case types.ErrMsgNotFound, types.ErrNotEnoughSigs: // not enough signatures
				// sanity check
				sigCount, err := msgState.GetSigCountMsgHeader(w, mc)
				if err != nil {
					panic(err)
				}
				if sigCount >= coinThrsh {
					goto checkCoin
				}
			default:
				panic(err)
			}
		}
	}
	return round, nil
}
