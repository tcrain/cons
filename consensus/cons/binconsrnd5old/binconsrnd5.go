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

/*
Implementation of signature based binary consensus algorithm.
*/
package binconsrnd5

import (
	"fmt"
	"github.com/tcrain/cons/consensus/coin"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

type BinConsRnd5 struct {
	cons.AbsConsItem
	coin          consinterface.CoinItemInterface
	Decided       int                  // -1 if a value has not yet been decided, 0 if 0 was decided, 1 if 1 was decided
	decidedRound  types.ConsensusRound // the round where the decision happened
	gotProposal   bool                 // for sanity checks
	IncludeProofs bool                 // true if messages should include signed values supporting the value in the message
	terminated    bool                 // set to true when consensus terminated
}

func (sc *BinConsRnd5) getMsgState() *MessageState { // TODO clean this up
	return sc.ConsItems.MsgState.(bincons1.BinConsMessageStateInterface).GetBinMsgState().(*MessageState)
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *BinConsRnd5) GetCommitProof() []messages.MsgHeader {
	var hdrType messages.HeaderID
	switch sc.decidedRound {
	case 0:
		panic(sc.decidedRound)
	case 1:
		hdrType = messages.HdrAuxProof
	default:
		hdrType = messages.HdrAuxBoth
	}
	return bincons1.GetBinConsCommitProof(hdrType, sc, sc.getMsgState(), true, sc.ConsItems.MC)
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *BinConsRnd5) GetBinState(localOnly bool) ([]byte, error) {
	msg, err := messages.CreateMsg(sc.PreHeaders)
	if err != nil {
		panic(err)
	}
	bs, err := sc.getMsgState().GetMsgState(sc.ConsItems.MC.MC.GetMyPriv(), localOnly, sc.GetBufferCount, sc.ConsItems.MC)
	if err != nil {
		return nil, err
	}
	_, err = messages.AppendHeader(msg, (messagetypes.ConsBinStateMessage)(bs))
	if err != nil {
		panic(err)
	}
	return msg.GetBytes(), nil
}

// GetProposeHeaderID returns the HeaderID messages.HdrBinPropose that will be input to GotProposal.
func (sc *BinConsRnd5) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrBinPropose
}

// GenerateNewItem creates a new bin cons item.
func (*BinConsRnd5) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &BinConsRnd5{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.IncludeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.gotProposal = false
	newItem.Decided = -1
	newItem.decidedRound = 0
	newItem.coin = coin.GenerateCoinIterfaceItem(gc.CoinType)

	return newItem
}

// Start allows GetProposalIndex to return true.
func (sc *BinConsRnd5) Start() {
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *BinConsRnd5) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *BinConsRnd5) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {
	sc.AbsGotProposal()
	bpm := hdr.(*messagetypes.BinProposeMessage)
	if bpm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	binMsgState := sc.getMsgState()

	binMsgState.Lock()
	// In case of recover, if we already sent a message then we don't want to send another
	roundStruct := binMsgState.getAuxRoundStruct(0, sc.ConsItems.MC)
	if roundStruct.sentAuxProof {
		binMsgState.Unlock()
		return nil
	}
	roundStruct.sentAuxProof = true
	binMsgState.Unlock()

	logging.Infof("Got local proposal for index %v bin val %v", sc.Index, bpm.BinVal)
	auxMsg := messagetypes.NewAuxProofMessage(sc.GeneralConfig.AllowSupportCoin)
	auxMsg.BinVal = bpm.BinVal
	auxMsg.Round = 0
	sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
		sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, sc.CommitProof...)

	return nil
}

// NeedsConcurrent returns 1.
func (sc *BinConsRnd5) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// GetBinDecided returns -1 if not decided, or the decided value and the decided round.
func (sc *BinConsRnd5) GetBinDecided() (int, types.ConsensusRound) {
	return sc.Decided, sc.decidedRound
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
func (sc *BinConsRnd5) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (bool, bool) {

	_ = isLocal
	_ = senderChan

	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t
	// binMsgState := sc.getMsgState()

	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	var round types.ConsensusRound
	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.AuxProofMessage:
		round = w.Round
	case *messagetypes.AuxBothMessage:
		round = w.Round
		if round < 2 {
			panic("should have caught earlier")
		}
	default:
		var err error
		var retMsg messages.MsgHeader
		binMsgState := sc.getMsgState()
		round, retMsg, _, _, err = sc.coin.CheckCoinMessage(deser, isLocal,
			true, sc, binMsgState.coinState, binMsgState)
		if err != nil {
			panic(fmt.Sprint("got invalid message header", deser.HeaderType))
		}
		if retMsg != nil {
			sc.BroadcastCoin(retMsg, sc.MainChannel)
		}

	}
	// If we already decided then don't need to process this message
	if sc.terminated {
		logging.Infof("Got a msg for round %v, but already decided in round %v", round, sc.decidedRound)
		return false, false
	}

	// check if we can now advance rounds (note the message was already stored to the bincons state in BinConsRnd5MessageState.GotMsg
	for sc.CheckRound(nmt, t, round, sc.MainChannel) {
		round++
	}
	return true, true
}

// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
func (sc *BinConsRnd5) CanSkipMvTimeout() bool {
	panic("TODO")
	// return sc.getMsgState().getAuxRoundStruct(2, sc.ConsItems.MC).TotalBinBothMsgCount >
	// 	sc.ConsItems.MC.MC.GetMemberCount()-sc.ConsItems.MC.MC.GetFaultCount()
}

// SetInitialState does noting for this algorithm.
func (sc *BinConsRnd5) SetInitialState([]byte) {}

// CheckRound checks for the given round if enough messages have been received to progress to the next round
// and return true if it can.
func (sc *BinConsRnd5) CheckRound(nmt int, t int, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) bool {

	binMsgState := sc.getMsgState()
	binMsgState.Lock()
	defer binMsgState.Unlock()

	if round == 0 {
		return true
	}
	roundStruct := sc.getMsgState().getAuxRoundStruct(round, sc.ConsItems.MC)
	if !roundStruct.gotCoin {
		if coinVals := binMsgState.coinState.GetCoins(round); len(coinVals) > 0 {
			roundStruct.gotCoin = true
			roundStruct.coinVals = coinVals

			if sc.AllowSupportCoin {
				if len(roundStruct.coinVals) > 1 {
					panic("cant support coin and have a weak coin")
				}
				// We need to add the supporters of coin to the counts for the next round
				auxMsg := messagetypes.NewAuxProofMessage(sc.AllowSupportCoin)
				auxMsg.BinVal = roundStruct.coinVals[0]
				auxMsg.Round = round + 1
				auxMsgCoin := messagetypes.NewAuxProofMessage(sc.AllowSupportCoin)
				auxMsgCoin.BinVal = types.Coin
				auxMsgCoin.Round = round + 1

				nxtRndStruct := binMsgState.getAuxRoundStruct(round+1, sc.ConsItems.MC)
				// track the count of these messages for unique signers
				binMsgState.Sms.TrackTotalSigCount(sc.ConsItems.MC, auxMsg, auxMsgCoin)
				nxtRndStruct.coinHeaders = []messages.InternalSignedMsgHeader{auxMsg, auxMsgCoin}

				// update the count for supporters of the coin in the next round
				nxtRndStruct.BinNumsAux[roundStruct.coinVals[0]], _ = binMsgState.Sms.GetTotalSigCount(sc.ConsItems.MC, auxMsg, auxMsgCoin)
			}
		}
	}

	if round > 0 {
		prvRoundStruct := sc.getMsgState().getAuxRoundStruct(round-1, sc.ConsItems.MC)
		if round == 1 {
			if sc.CheckMemberLocal() && !prvRoundStruct.sentAuxProof {
				return false
			}
		}
		if round > 1 {
			if sc.CheckMemberLocal() && (!prvRoundStruct.sentAuxProof ||
				!prvRoundStruct.sentAuxBoth || (!prvRoundStruct.sentCoin && !sc.GeneralConfig.AllowSupportCoin)) {

				return false
			}
		}
	}

	// Check if we have a valid aux proofValue
	auxProofValids, supportCoin := sc.getMsgState().getValidsAuxProof(round, sc.ConsItems.MC)
	if auxProofValids[1] {
		// check broadcast 1
		if sc.AllowSupportCoin && round > 1 {
			sc.checkBroadcastsAuxCoin(1, supportCoin, round, sc.getMsgState(), mainChannel)
		} else {
			sc.checkBroadcastsAux(1, supportCoin, round, sc.getMsgState(), mainChannel)
		}
	} else if auxProofValids[0] {
		// check broadcast 0
		if sc.AllowSupportCoin && round > 1 {
			sc.checkBroadcastsAuxCoin(0, supportCoin, round, sc.getMsgState(), mainChannel)
		} else {
			sc.checkBroadcastsAux(0, supportCoin, round, sc.getMsgState(), mainChannel)
		}
	} else if supportCoin && sc.AllowSupportCoin {
		if round <= 1 {
			panic("cannot support coin on round 0 or 1")
		}
		sc.checkBroadcastsAuxCoin(types.Coin, supportCoin, round, sc.getMsgState(), mainChannel)
	} else {
		return false
	}

	if round > 1 {
		// Check if we have a valid auxBoth value
		auxBothValids := sc.getMsgState().getValidsAuxBoth(round, sc.ConsItems.MC)
		logging.Info("bcast aux both", roundStruct.BinNumsAux, auxBothValids, round, sc.Index.Index)
		if auxBothValids[0] {
			// check broadcast 0
			sc.checkBroadcastsAuxBoth(0, round, sc.getMsgState(), mainChannel)
		} else if auxBothValids[1] {
			// check broadcast 1
			sc.checkBroadcastsAuxBoth(1, round, sc.getMsgState(), mainChannel)
		} else if auxBothValids[types.BothBin] {
			// check broadcast bothBin
			sc.checkBroadcastsAuxBoth(types.BothBin, round, sc.getMsgState(), mainChannel)
		} else {
			return false
		}
	}

	// check if we can decide
	shouldDecide := types.BinVal(2)
	if round == 1 {
		if roundStruct.BinNumsAux[1] >= nmt {
			shouldDecide = 1
		}
	} else {
		if roundStruct.BinNumsBoth[0] >= nmt {
			shouldDecide = 0
		} else if roundStruct.BinNumsBoth[1] >= nmt {
			shouldDecide = 1
		}
	}
	if shouldDecide < 2 {
		if sc.Decided >= 0 {
			if int(shouldDecide) != sc.Decided {
				panic(fmt.Errorf("decided different values, previous %v, new %v", sc.Decided, shouldDecide))
			}
		} else {
			sc.Decided = int(shouldDecide)
			sc.decidedRound = round
			logging.Infof("Decided proc %v bin round %v, binval %v, index %v", sc.TestIndex, round, shouldDecide, sc.Index)
			sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, shouldDecide == 0)
		}
	}

	// 4th Check if we can send a coin message
	// We can send the coin for the current round and the proposal for the next round if we have enough valid messages
	// if roundStruct.TotalBinBothMsgCount >= nmt {
	if sc.checkDone(round, nmt, t) {
		return false
	}
	switch round {
	case 1:
		if roundStruct.TotalBinAuxMsgCount < nmt {
			return false
		}
	default:
		if roundStruct.TotalBinBothMsgCount >= nmt {
			if sc.GeneralConfig.AllowSupportCoin {
				// we broadcast the coin and the aux in the next round together
			} else {
				sc.checkBroadcastsCoin(round, sc.getMsgState(), mainChannel)
			}
		} else { // not enough messages to go to the next round
			return false
		}
	}
	//} else {
	// sanity checks? TODO
	//}

	return true
}

// checkDone returns true if we don't need to process messages currently because we think everyone has decided.
func (sc *BinConsRnd5) checkDone(round types.ConsensusRound, nmt, t int) bool {
	_ = nmt
	if sc.terminated {
		return true
	}
	if sc.Decided < 0 {
		return false
	}
	if round < sc.decidedRound { // this message is from an earlier round
		return true
	}

	_ = t
	switch sc.StopOnCommit {
	case types.Immediate:
		sc.terminated = true
		return true
	case types.SendProof:
		// Send a proof of decision
		var hdrType messages.HeaderID
		switch sc.decidedRound {
		case 0:
			panic(round)
		case 1:
			hdrType = messages.HdrAuxProof
		default:
			hdrType = messages.HdrAuxBoth
		}
		prfMsgs := bincons1.GetBinConsCommitProof(hdrType, sc, sc.getMsgState(), false, sc.ConsItems.MC)
		sc.BroadcastFunc(nil, sc.ConsItems, nil, true,
			sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, prfMsgs...)
		sc.terminated = true
		return true
	case types.NextRound:
		// everyone will decide in the next round after decision round
		if round >= sc.decidedRound+1 {
			sc.terminated = true
			return true
		}
		return false
	default:
		panic(sc.StopOnCommit)
	}
}

func (sc *BinConsRnd5) beforeBcastSanityCheck(
	est types.BinVal,
	headerID messages.HeaderID,
	round types.ConsensusRound,
	roundStruct *auxRandRoundStruct,
	binMsgState *MessageState) {

	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t

	localMember := sc.CheckMemberLocal()
	if round == 1 {
		prevRoundState := binMsgState.getAuxRoundStruct(round-1, sc.ConsItems.MC)
		if prevRoundState == nil ||
			(localMember && !prevRoundState.sentAuxProof) {
			panic("invalid ordering")
		}
	}
	if round > 2 {
		// Sanity check
		prevRoundState := binMsgState.getAuxRoundStruct(round-1, sc.ConsItems.MC)
		if prevRoundState == nil ||
			(localMember && (!prevRoundState.sentAuxProof ||
				!prevRoundState.sentAuxBoth)) {
			panic("invalid ordering")
		}
		if sc.AllowSupportCoin && headerID == messages.HdrAuxProof {
			//if localMember && prevRoundState.sentCoin && round > 1 {
			//	panic("should not have sent coin")
			//}
			if round > binMsgState.maxPresetCoinRound+1 && localMember && ((!prevRoundState.sentCoin && roundStruct.sentAuxProof) ||
				(prevRoundState.sentCoin && !roundStruct.sentAuxProof)) {
				panic("invalid ordering 2")
			}
		} else {
			if !prevRoundState.gotCoin {
				if headerID == messages.HdrAuxBoth && est != types.BothBin {
					if prevRoundState.BinNumsBoth[est] < nmt && roundStruct.BinNumsAux[est] < nmt {
						panic("should have gotten coin")
					}
				} /*else {
					panic("should have gotten coin")
				}*/
			}
			if localMember && !prevRoundState.sentCoin {
				panic("should have sent coin")
			}
		}
	}

	switch headerID {
	case messages.HdrAuxBoth:
		if localMember && !roundStruct.sentAuxProof {
			panic("should send auxProof before auxBoth")
		}
	case messages.HdrCoin:
		if localMember && (!roundStruct.sentAuxProof || !roundStruct.sentAuxBoth) {
			panic("should send auxProof before auxBoth")
		}
	}
}

// checkBroadcastsAuxCoin if need to broadcast messages when AllowSupportCoin=true.
// This broadcasts the aux message for round round and if round > 2, the coin message for round-1.
func (sc *BinConsRnd5) checkBroadcastsAuxCoin(est types.BinVal, supportCoin bool,
	round types.ConsensusRound,
	binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
	sc.beforeBcastSanityCheck(est, messages.HdrAuxProof, round, roundStruct, binMsgState)
	if !roundStruct.sentAuxProof && sc.CheckMemberLocal() {

		// Support whatever the coin will be
		if supportCoin {
			est = types.Coin
		}

		// Store the values
		roundStruct.sentAuxProof = true
		binMsgState.getAuxRoundStruct(round-1, sc.ConsItems.MC).sentCoin = true
		logging.Infof("bcast idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round, est, roundStruct, sc.TestIndex)

		// Broadcast the message for the round
		auxMsg := messagetypes.NewAuxProofMessage(sc.GeneralConfig.AllowSupportCoin)
		var prfMsgs []messages.MsgHeader
		auxMsg.BinVal = est
		auxMsg.Round = round
		sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

		// Set to true before checking if we are a member, since check member will always
		// give the same result for this round
		if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
			if round > 1 && !(round == 2 && sc.getMsgState().isMv) { // don't use a coin for round 1 of multi value reduction

				if round > binMsgState.maxPresetCoinRound {
					var coinMsg messages.MsgHeader
					//if !roundStruct.sentCoin {
					logging.Infof("bcast coin idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round-1, roundStruct, sc.TestIndex)
					//roundStruct.sentCoin = true
					coinMsg = sc.coin.GenerateCoinMessage(round-1,
						true, sc, binMsgState.coinState, binMsgState)
					//	}
					if coinMsg != nil {
						prfMsgs = append(prfMsgs, coinMsg)
					}
				}
			}
			prfMsgs = append(prfMsgs, sc.generateProofsInternal(auxMsg, supportCoin, est, round)...)
			sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfMsgs...)
			// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfMsgs...)
		}
	}
	return true
}

// checkBroadcastsAux checks if an aux message needs to be broadcast when AllowSupportCoin=false.
func (sc *BinConsRnd5) checkBroadcastsAux(est types.BinVal, supportCoin bool, round types.ConsensusRound,
	binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	if round == 1 && supportCoin {
		panic("cannot support coin on round 1")
	}

	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
	sc.beforeBcastSanityCheck(est, messages.HdrAuxProof, round, roundStruct, binMsgState)

	if !roundStruct.sentAuxProof && sc.CheckMemberLocal() {
		// Store the values

		roundStruct.sentAuxProof = true
		logging.Infof("bcast idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round, est, roundStruct, sc.TestIndex)

		// Broadcast the message for the round
		auxMsg := messagetypes.NewAuxProofMessage(sc.GeneralConfig.AllowSupportCoin)
		var prfMsgs []messages.MsgHeader
		auxMsg.BinVal = est
		auxMsg.Round = round
		sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

		// Set to true before checking if we are a member, since check member will always
		// give the same result for this round
		if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
			prfMsgs = append(prfMsgs, sc.generateProofsInternal(auxMsg, supportCoin, est, round)...)
			sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfMsgs...)
		}
	}
	return true
}

// checkBroadcastsAux checksBoth if an auxBoth message needs to be broadcast when AllowSupportCoin=false.
func (sc *BinConsRnd5) checkBroadcastsAuxBoth(est types.BinVal, round types.ConsensusRound,
	binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	if round < 2 {
		panic("should only send aux both after round 1")
	}

	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
	sc.beforeBcastSanityCheck(est, messages.HdrAuxBoth, round, roundStruct, binMsgState)

	if !roundStruct.sentAuxBoth && sc.CheckMemberLocal() {
		// Store the values
		roundStruct.sentAuxBoth = true
		logging.Infof("bcast aux both idx %v, round %v, est %v, %+v, %v",
			sc.Index.Index, round, est, roundStruct, sc.TestIndex)

		// Broadcast the message for the round
		auxMsg := messagetypes.NewAuxBothMessage()
		var prfMsgs []messages.MsgHeader
		auxMsg.BinVal = est
		auxMsg.Round = round
		sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)

		// Set to true before checking if we are a member, since check member will always
		// give the same result for this round
		if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
			prfMsgs = append(prfMsgs, sc.generateProofsInternal(auxMsg, false, est, round)...)
			sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfMsgs...)
		}
	}
	return true
}

// checkBroadcastsCoin checks if a coin message needs to be broadcast when AllowSupportCoin=false.
func (sc *BinConsRnd5) checkBroadcastsCoin(round types.ConsensusRound,
	binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	if round < 2 {
		panic("should only send coin after round 1")
	}

	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
	sc.beforeBcastSanityCheck(types.Coin, messages.HdrCoin, round, roundStruct, binMsgState)

	if !roundStruct.sentCoin {
		logging.Infof("bcast coin idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round, roundStruct, sc.TestIndex)
		roundStruct.sentCoin = true
		coinMsg := sc.coin.GenerateCoinMessage(round,
			true, sc, binMsgState.coinState, binMsgState)
		if coinMsg != nil {
			sc.BroadcastCoin(coinMsg, mainChannel)
		}
	}

	return true
}

// generateProofsInternal returns the proofs needed to broadcast the message.
func (sc *BinConsRnd5) generateProofsInternal(hdr messages.MsgIDHeader, supportCoin bool, est types.BinVal,
	round types.ConsensusRound) (prfMsgs []messages.MsgHeader) {

	if sc.IncludeProofs {
		// collect signatures to support your choice
		sigCount, _, _, err := sc.GetBufferCount(hdr, sc.GeneralConfig, sc.ConsItems.MC)
		if err != nil {
			panic(err)
		}
		// Generate proofs
		if hdr.GetID() == messages.HdrAuxProof && supportCoin {
			est = types.BothBin
		}
		nxtPrfs, err := sc.getMsgState().GenerateProofs(hdr.GetID(), sigCount, round, est,
			sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
		if err != nil {
			logging.Error(err, "est:", est, "round:", round, "cons idx:", sc.Index)
			panic("should have proofs")
		} else {
			for _, nxt := range nxtPrfs {
				prfMsgs = append(prfMsgs, nxt)
			}
		}
	}
	return
}

// HasReceivedProposal panics because BonCons has no proposals.
func (sc *BinConsRnd5) HasReceivedProposal() bool {
	panic("unused")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrAuxProof, it returns n-t, n for the thresholds.
func (sc *BinConsRnd5) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig, memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {
	switch hdr.GetID() {
	case messages.HdrAuxProof, messages.HdrAuxBoth:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return coin.GetBufferCount(hdr, gc, memberChecker)
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrAuxProof are valid headerIDs.
func (sc *BinConsRnd5) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrAuxProof:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
			messagetypes.NewAuxProofMessage(gc.AllowSupportCoin)), nil
	case messages.HdrAuxBoth:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
			messagetypes.NewAuxBothMessage()), nil
	default:
		return coin.GetCoinHeader(emptyPub, gc, headerType)
	}
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *BinConsRnd5) HasDecided() bool {
	return sc.Decided > -1
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *BinConsRnd5) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *BinConsRnd5) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil, nil, true
}

// GetDecision returns the binary value decided as a single byte slice.
func (sc *BinConsRnd5) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	if sc.Decided == -1 {
		panic("should have decided")
	}
	return nil, []byte{byte(sc.Decided)}, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
}

// GetConsType returns the type of consensus this instance implements.
func (sc *BinConsRnd5) GetConsType() types.ConsType {
	return types.BinConsRnd5OldType
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *BinConsRnd5) PrevHasBeenReset() {
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *BinConsRnd5) SetNextConsItem(next consinterface.ConsItem) {
	_ = next
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *BinConsRnd5) ShouldCreatePartial(headerType messages.HeaderID) bool {
	_ = headerType
	return false
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// Broadcast a coin msg.
func (sc *BinConsRnd5) BroadcastCoin(coinMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	mainChannel.SendHeader(messages.AppendCopyMsgHeader(sc.PreHeaders, coinMsg),
		messages.IsProposalHeader(sc.Index, coinMsg.(messages.InternalSignedMsgHeader).GetBaseMsgHeader()),
		true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		sc.ConsItems.MC.MC.GetStats().IsRecordIndex())
}

// GenerateMessageState generates a new message state object given the inputs.
func (*BinConsRnd5) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewBinConsRnd5MessageState(false, gc)
}

// Collect is called when the item is being garbage collected.
func (sc *BinConsRnd5) Collect() {
}
