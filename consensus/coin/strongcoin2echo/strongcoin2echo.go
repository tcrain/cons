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

package strongcoin2echo

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// StrongCoin2Echo represents a strong coin implemented by an n-t threshold signautre.
type StrongCoin2Echo struct {
}

func NewStrongCoin2Echo() consinterface.CoinItemInterface {
	return &StrongCoin2Echo{}
}

func (sc1 *StrongCoin2Echo) GenerateCoinMessage(round types.ConsensusRound, alwaysGenerate bool, consItem consinterface.ConsItem,
	coinMsgState consinterface.CoinMessageStateInterface, msgState consinterface.MessageState) (ret messages.MsgHeader) {

	cms := coinMsgState.(*MsgState)
	cms.mutex.Lock()
	crs := cms.getRoundStruct(round)
	if crs.sentPreCoin {
		cms.mutex.Unlock()
		return
	}
	crs.sentPreCoin = true
	cms.mutex.Unlock()

	if consItem.CheckMemberLocal() {
		// Start with the pre coin message
		coinMsg := messagetypes.NewCoinPreMessage()
		coinMsg.Round = round

		// Set to true before checking if we are a member, since check member will always
		// give the same result for this round
		if consItem.CheckMemberLocalMsg(coinMsg) || alwaysGenerate {
			// compute the coin
			// Add any needed signatures
			var err error
			if consItem.GetGeneralConfig().EncryptChannels {
				ret, err = msgState.SetupUnsignedMessage(coinMsg, consItem.GetConsInterfaceItems().MC)
			} else {
				ret, err = msgState.SetupSignedMessage(coinMsg, true,
					0, consItem.GetConsInterfaceItems().MC)
				if err != nil {
					logging.Error(err)
					panic(err)
				}
			}
		}
	}

	return
}

// CheckCoinMessage should be called from within ProcessMessage of the ConsensusItem that is using this coin.
// It returns the round the coin corresponds to and true in first boolean position if made progress towards decision,
// or false if already decided, and return true in second position if the message should be forwarded.
// If the message is invalid an error is returned.
func (sc1 *StrongCoin2Echo) CheckCoinMessage(deser *deserialized.DeserializedItem,
	isLocal bool, alwaysGenerate bool, consItem consinterface.ConsItem, coinMsgState consinterface.CoinMessageStateInterface,
	msgState consinterface.MessageState) (round types.ConsensusRound, ret messages.MsgHeader,
	progress, shouldForward bool, err error) {

	_ = isLocal
	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.CoinPreMessage:
		round = w.Round
		progress, shouldForward = true, true
		// Check if we have enough messages to send the second part of the coin
		cms := coinMsgState.(*MsgState)
		cms.mutex.Lock()
		crs := cms.getRoundStruct(round)
		if crs.sentCoin || !crs.shouldSendCoin {
			cms.mutex.Unlock()
			return
		}
		crs.sentCoin = true
		cms.mutex.Unlock()

		if consItem.CheckMemberLocal() {
			coinMsg := messagetypes.NewCoinMessage()
			coinMsg.Round = round

			// Set to true before checking if we are a member, since check member will always
			// give the same result for this round
			if consItem.CheckMemberLocalMsg(coinMsg) || alwaysGenerate {
				// compute the coin
				// Add any needed signatures
				var err error
				coinSigMsg, err := msgState.SetupSignedMessage(coinMsg, true,
					0, consItem.GetConsInterfaceItems().MC)
				if err != nil {
					logging.Error(err)
					panic(err)
				}
				ret = coinSigMsg
			}
		}
	case *messagetypes.CoinMessage:
		round = w.Round
		progress, shouldForward = true, true
	default:
		logging.Info("got invalid coin message header", deser.HeaderType)
		err = types.ErrInvalidHeader
	}
	return
}

// GetHeader should a blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
func (StrongCoin2Echo) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig,
	headerID messages.HeaderID) (messages.MsgHeader, error) {

	switch headerID {
	case messages.HdrCoin:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewCoinMessage()), nil
	case messages.HdrCoinPre:
		if gc.EncryptChannels {
			return sig.NewUnsignedMessage(types.ConsensusIndex{}, emptyPub, messagetypes.NewCoinPreMessage()), nil
		}
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewCoinPreMessage()), nil
	default:
		return nil, types.ErrInvalidHeader
	}
}

// GetBufferCount returns the thresholds for messages of type coin.
// The thresholds are n-t.
func (StrongCoin2Echo) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (endThreshold int,
	maxPossible int, msgid messages.MsgID, err error) { // How many of the same msg to buff before forwarding

	switch hdr.GetID() {
	// TODO should have a different threshold for hdrBV, since it can be t+1?
	case messages.HdrCoinPre:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	case messages.HdrCoin:

		if gc.UseTp1CoinThresh {
			tp1 := memberChecker.MC.GetFaultCount() + 1
			return tp1, tp1, hdr.GetMsgID(), nil
		}
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return 0, 0, nil, types.ErrInvalidHeader
	}
}
