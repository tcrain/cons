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
package binconsrnd6

import (
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/coin"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

type BinConsRnd6 struct {
	cons.AbsConsItem
	Decided         int                  // -1 if a value has not yet been decided, 0 if 0 was decided, 1 if 1 was decided
	decidedRound    types.ConsensusRound // the round where the decision happened
	gotProposal     bool                 // for sanity checks
	terminated      bool                 // set to true when consensus terminated
	terminatedRound types.ConsensusRound // round to stop at
	coin            consinterface.CoinItemInterface
}

func (sc *BinConsRnd6) getMsgState() *MessageState { // TODO clean this up
	return sc.ConsItems.MsgState.(bincons1.BinConsMessageStateInterface).GetBinMsgState().(*MessageState)
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *BinConsRnd6) GetCommitProof() []messages.MsgHeader {
	var hdr messages.HeaderID
	switch sc.decidedRound {
	case 1, 2:
		hdr = messages.HdrAuxStage0
	default:
		hdr = messages.HdrAuxStage1
	}
	return bincons1.GetBinConsCommitProof(hdr, sc, sc.getMsgState(), true, sc.ConsItems.MC)
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *BinConsRnd6) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *BinConsRnd6) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrBinPropose
}

// GenerateNewItem creates a new bin cons item.
func (*BinConsRnd6) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &BinConsRnd6{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.Decided = -1
	newItem.gotProposal = false
	newItem.Decided = -1
	newItem.decidedRound = 0
	newItem.coin = coin.GenerateCoinIterfaceItem(gc.CoinType)

	return newItem
}

// Start allows GetProposalIndex to return true.
func (sc *BinConsRnd6) Start() {
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *BinConsRnd6) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *BinConsRnd6) GotProposal(hdr messages.MsgHeader, _ channelinterface.MainChannel) error {
	sc.AbsGotProposal()
	bpm := hdr.(*messagetypes.BinProposeMessage)
	if bpm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	sms := sc.getMsgState()
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	roundStruct := sms.getAuxRoundStruct(1, sc.ConsItems.MC)
	roundStruct.sentInitialMessage = true

	logging.Infof("Got local proposal for index %v bin val %v", sc.Index, bpm.BinVal)
	if !sc.CheckMemberLocal() {
		panic("should not get proposal if not member")
	}

	bvInfo := roundStruct.supportBvInfoStage0[bpm.BinVal]
	if !bvInfo.sentMsg {
		bvInfo.sentMsg = true
		sc.broadcastMsg(messages.HdrBV0, bpm.BinVal, 1, 0)
	}
	return nil
}

// GetMVInitialRoundBroadcast returns the type of binary message that the multi-value reduction should broadcast for round 0.
func (sc *BinConsRnd6) GetMVInitialRoundBroadcast(val types.BinVal) messages.InternalSignedMsgHeader {
	panic("TODO")
}

// NeedsConcurrent returns 1.
func (sc *BinConsRnd6) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// GetBinDecided returns -1 if not decided, or the decided value and the decided round.
func (sc *BinConsRnd6) GetBinDecided() (int, types.ConsensusRound) {
	return sc.Decided, sc.decidedRound
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
func (sc *BinConsRnd6) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	_ *channelinterface.SendRecvChannel) (bool, bool) {

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
	case *messagetypes.BVMessage0:
		round = w.Round
	case *messagetypes.BVMessage1:
		round = w.Round
	case *messagetypes.AuxProofMessage:
		round = w.Round
		if round < 3 {
			panic("should have caught earlier")
		}
	case *messagetypes.AuxStage0Message:
		round = w.Round
	case *messagetypes.AuxStage1Message:
		round = w.Round
		if round < 3 {
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
	if sc.HasDecided() {
		var stopExtra types.ConsensusRound = 1
		if sc.decidedRound == 1 { // if we decide on the first round we need to cointinue to round 3
			stopExtra++
		}
		if (sc.terminated && round > sc.terminatedRound) ||
			(sc.StopOnCommit == types.NextRound && sc.Decided > -1 && round > sc.decidedRound+stopExtra) {

			logging.Infof("Got a msg for round %v, but already decided in round %v", round, sc.decidedRound)
			return false, false
		}
	}
	// check if we can now advance rounds (note the message was already stored to the bincons state in BinConsRnd6MessageState.GotMsg
	for sc.CheckRound(nmt, t, round, sc.MainChannel) {
		round++
	}
	return true, true
}

// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
func (sc *BinConsRnd6) CanSkipMvTimeout() bool {
	panic("TODO")
}

// SetInitialState does noting for this algorithm.
func (sc *BinConsRnd6) SetInitialState([]byte) {}

// CheckRound checks for the given round if enough messages have been received to progress to the next round
// and return true if it can.
func (sc *BinConsRnd6) CheckRound(nmt int, t int, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) bool {

	binMsgState := sc.getMsgState()
	binMsgState.Lock()
	defer binMsgState.Unlock()

	// get the round state
	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)

	if !roundStruct.gotCoin {
		if coinVals := binMsgState.coinState.GetCoins(round); len(coinVals) > 0 {
			roundStruct.gotCoin = true
			roundStruct.coinVal = coinVals[0]
		}
	}

	if round > 1 && !binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC).sentInitialMessage {
		return false
	}

	// Check if we need to send a BV message for stage 0
	for binVal := types.BinVal(0); binVal < types.BothBin; binVal++ {
		bvInfo := &roundStruct.bvInfoStage0[binVal] // sc.getMsgState().getBVState(0, binVal, round, sc.ConsItems.MC)
		if !bvInfo.sentMsg && bvInfo.msgCount > t {
			// Broadcast BV
			bvInfo.sentMsg = true
			sc.broadcastMsg(messages.HdrBV0, binVal, round, 0)
		}
	}
	// Check if we need to send a BV message for stage 1
	for binVal := types.BinVal(0); binVal < types.BothBin; binVal++ {
		bvInfo := &roundStruct.supportBvInfoStage1[binVal] // sc.getMsgState().getBVState(1, binVal, round, sc.ConsItems.MC)
		if !bvInfo.sentMsg && bvInfo.msgCount > t {
			// Broadcast BV
			bvInfo.sentMsg = true
			sc.broadcastMsg(messages.HdrBV0, binVal, round, 1)
		}
	}

	// Check if we can send a stage 0 aux
	var validBVs []types.BinVal
	for _, binVal := range []types.BinVal{1, 0} { // we prefer 1 if we can
		// for binVal := types.BinVal(0); binVal < 2; binVal++ {
		bvInfo := sc.getMsgState().getBVState(0, binVal, round, sc.ConsItems.MC)
		if bvInfo != nil {
			if bvInfo.msgCount >= nmt {
				if !roundStruct.sentStage0Aux {
					// broadcast
					roundStruct.sentStage0Aux = true
					sc.broadcastMsg(messages.HdrAuxStage0, binVal, round, 0)
				}
			}
			if bvInfo.msgCount >= nmt {
				validBVs = append(validBVs, binVal)
			}
		}
	}

	// Special case
	if round < 3 {
		// Check for decision
		shouldDecide := types.Coin
		switch round {
		case 1:
			if roundStruct.auxStage0BinNums[1] >= nmt {
				shouldDecide = 1
			}
		case 2:
			if roundStruct.auxStage0BinNums[0] >= nmt {
				shouldDecide = 0
			}
		}
		if shouldDecide < 2 {
			if sc.Decided == -1 {
				sc.Decided = int(shouldDecide)
				logging.Infof("Decided proc %v bin round %v, binval %v, index %v",
					sc.TestIndex, round, sc.Decided, sc.Index)
				sc.decidedRound = round
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, sc.Decided == 0)
			}
			if sc.Decided != int(shouldDecide) {
				panic(fmt.Errorf("decided %v previously in round %v, now decided %v in round %v",
					sc.Decided, sc.decidedRound, shouldDecide, round))
			}
			if sc.StopOnCommit == types.Immediate {
				sc.terminated = true
				sc.terminatedRound = sc.decidedRound
			}
		}

		if sc.checkDone(round, nmt, t) {
			return false
		}

		// Check if we can send the first message for the next round
		nxtRoundStruct := binMsgState.getAuxRoundStruct(round+1, sc.ConsItems.MC)
		if !nxtRoundStruct.sentInitialMessage {
			var validStage0AuxCount int
			var validBins []types.BinVal
			for i := types.BinVal(0); i < 2; i++ {
				if roundStruct.supportBvInfoStage0[i].msgCount >= nmt {
					validBins = append(validBins, i)
				}
			}
			switch len(validBins) {
			case 1:
				validStage0AuxCount = roundStruct.auxStage0BinNums[validBins[0]]
			case 2:
				validStage0AuxCount = roundStruct.TotalAuxStage0MsgCount
			}
			if validStage0AuxCount >= nmt {
				// If we have the value of the coin, we can send aux stage 0 msg directly
				var supportCoin bool
				for _, nxt := range validBins {
					if nxt == roundStruct.coinVal && roundStruct.auxStage0BinNums[roundStruct.coinVal] > 0 { // need at least 1 coin msg
						supportCoin = true
					}
				}
				if supportCoin {
					// We broadcast the aux for the next round directly
					nxtRoundStruct.sentInitialMessage = true
					if !roundStruct.sentStage0Aux {
						nxtRoundStruct.sentStage0Aux = true
						// we broadcast for the next round
						sc.broadcastMsg(messages.HdrAuxStage0, roundStruct.coinVal, round+1, 0)
					}
				} else {
					// the value must have n-t messages
					var foundSupport bool
					var supportVal types.BinVal
					for _, nxt := range validBins {
						if roundStruct.auxStage0BinNums[nxt] >= nmt {
							foundSupport = true
							supportVal = nxt
						}
					}
					if foundSupport {
						nxtRoundStruct.sentInitialMessage = true
						if !nxtRoundStruct.bvInfoStage0[supportVal].sentMsg {
							nxtRoundStruct.bvInfoStage0[supportVal].sentMsg = true
							// we broadcast for the next round
							sc.broadcastMsg(messages.HdrBV0, supportVal, round+1, 0)
						}
					}
				}
			}
		}

		return true
	}

	// Check if we can send a middle aux
	if !roundStruct.sentAux && len(validBVs) > 0 && roundStruct.sentStage0Aux {
		// We either send the single value that is valid
		auxBinVal := validBVs[0]
		if len(validBVs) > 1 {
			// Or a message representing both 0 and 1
			auxBinVal = types.BothBin
		}
		// Broadcast aux
		roundStruct.sentAux = true
		sc.broadcastMsg(messages.HdrAuxProof, auxBinVal, round, 0)
	}

	// Check if we have received enough middle aux messages
	// First check if we should track both bin values
	// var validMiddleAuxCount int
	var stage1BV types.BinVal // The valid we will send for stage1 binary value
	var foundValidStage1BV bool

	switch len(validBVs) {
	case 0:
		// no valids
	case 1:
		if roundStruct.AuxBinNums[validBVs[0]] >= nmt {
			stage1BV = validBVs[0]
			foundValidStage1BV = true
		}
	case 2:
		if len(roundStruct.auxBinHeaders) == 0 { // we need to track these messages
			var hdrs []messages.InternalSignedMsgHeader
			for i := types.BinVal(0); i <= 2; i++ {
				nxt := messagetypes.NewAuxProofMessage(true)
				nxt.BinVal = i
				nxt.Round = round
				hdrs = append(hdrs, nxt)
			}
			roundStruct.auxBinHeaders = hdrs
			sms := sc.getMsgState().Sms
			sms.TrackTotalSigCount(sc.ConsItems.MC, hdrs...)
			binMsgState.updateAuxBinMsgCount(roundStruct, sc.ConsItems.MC)
			// roundStruct.TotalAuxBinMsgCount = sms.GetTotalSigCount(sc.ConsItems.MC, hdrs...)
		}
		if roundStruct.TotalAuxBinMsgCount >= nmt {
			stage1BV = types.BothBin
			foundValidStage1BV = true
		}

		/*		// first check if we have n-t of the same value
				// First check if we have both values we need to check the total valid for both values
				if !foundValidStage1BV {
					for nxtBv := types.BinVal(0); nxtBv < 2; nxtBv++ {
						if roundStruct.AuxBinNums[nxtBv] >= nmt {
							stage1BV = nxtBv
							foundValidStage1BV = true
							break
						}
					}
				}*/
	default:
		panic(len(validBVs))
	}

	// check if we have enough middle aux to send the nxt msg
	if !roundStruct.sentStage1BinValue && foundValidStage1BV && roundStruct.sentAux {
		roundStruct.sentStage1BinValue = true
		if stage1BV == types.BothBin {
			// we don't need to send the value since we can send the aux directly
		} else {
			// Broadcast stage1BV
			//roundStruct.sentStage1BinValue = true
			if !roundStruct.supportBvInfoStage1[stage1BV].sentMsg {
				roundStruct.supportBvInfoStage1[stage1BV].sentMsg = true
				sc.broadcastMsg(messages.HdrBV0, stage1BV, round, 1)
			}
		}
	}

	// Check if we can send a stage 1 aux
	if foundValidStage1BV && !roundStruct.sentStage1Aux && roundStruct.sentAux {
		// We can directly send bot
		if stage1BV == types.BothBin {
			// Broadcast stage 1 aux
			roundStruct.sentStage1Aux = true
			sc.broadcastMsg(messages.HdrAuxStage1, types.BothBin, round, 1)
		} else {
			// Check if we can send an aux
			for binVal := types.BinVal(0); binVal <= 1; binVal++ {
				if roundStruct.supportBvInfoStage1[binVal].msgCount >= nmt {
					// broadcast stage 1 aux for a binary value
					roundStruct.sentStage1Aux = true
					sc.broadcastMsg(messages.HdrAuxStage1, binVal, round, 1)
					break
				}
			}
		}
	}

	// Get the next round struct
	nxtRoundStruct := binMsgState.getAuxRoundStruct(round+1, sc.ConsItems.MC)

	// Get the valid binVals for stage1
	var validStage1 []types.BinVal
	for i := types.BinVal(0); i < types.BothBin; i++ { // check 1 or 0 valid
		if roundStruct.supportBvInfoStage1[i].msgCount >= nmt {
			validStage1 = append(validStage1, i)
			// update the pointers for the next round since we can support a binary value directly
			if nxtRoundStruct.supportBvInfoStage0[i] != nil { // sanity check
				if nxtRoundStruct.supportBvInfoStage0[i] != &roundStruct.supportBvInfoStage1[i] ||
					nxtRoundStruct.supportBvInfoStage0[1-i] != &nxtRoundStruct.bvInfoStage0[1-i] {
					panic("invalid bv info setup")
				}
			}
			nxtRoundStruct.supportBvInfoStage0[i] = &roundStruct.supportBvInfoStage1[i]
			nxtRoundStruct.supportBvInfoStage0[1-i] = &nxtRoundStruct.bvInfoStage0[1-i]
		}
	}
	if len(validStage1) > 1 {
		panic("should only have 1 single stage 1 valid value")
	}
	// If both 0 and 1 are valid bv stage 1 values, then bot is a valid bv value for stage 1
	if len(validBVs) > 1 {
		validStage1 = append(validStage1, types.BothBin)
	}
	// sanity check
	if len(validBVs) > 2 {
		panic("should not have 3 valid values")
	}
	if validStage1 != nil && len(validStage1) > 1 {
		if validStage1[0] == 0 && validStage1[1] == 1 {
			panic("more than t faulty")
		}
	}

	// Check for decision
	if validStage1 != nil && len(validStage1) >= 1 && validStage1[0] != types.BothBin {
		if roundStruct.auxStage1BinNums[validStage1[0]] >= nmt {
			// we can decide
			if sc.Decided == -1 {
				sc.Decided = int(validStage1[0])
				logging.Infof("Decided proc %v bin round %v, binval %v, index %v",
					sc.TestIndex, round, sc.Decided, sc.Index)
				sc.decidedRound = round
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, sc.Decided == 0)
			}
			if sc.Decided != int(validStage1[0]) {
				panic(fmt.Errorf("decided %v previously in round %v, now decided %v in round %v",
					sc.Decided, sc.decidedRound, validStage1[0], round))
			}
		}
	}

	if sc.checkDone(round, nmt, t) {
		return false
	}

	// compute the number of valid stage 1 aux messages received
	var validStage1AuxCount int
	switch len(validStage1) {
	case 0:
		// no valids
	case 1:
		validStage1AuxCount = roundStruct.auxStage1BinNums[validStage1[0]]
	case 2:
		// We need to track the total message count for the valid values
		if len(roundStruct.auxStage1BinHeaders) == 0 {
			var hdrs []messages.InternalSignedMsgHeader
			for _, nxt := range validStage1 {
				auxMsg := messagetypes.NewAuxStage1Message()
				auxMsg.Round = round
				auxMsg.BinVal = nxt
				hdrs = append(hdrs, auxMsg)
			}
			roundStruct.auxStage1BinHeaders = hdrs
			sms := sc.getMsgState().Sms
			sms.TrackTotalSigCount(sc.ConsItems.MC, hdrs...)
			// roundStruct.TotalAuxStage1MsgCount = sms.GetTotalSigCount(sc.ConsItems.MC, hdrs...)
			binMsgState.updateAuxStage1MsgCount(roundStruct, sc.ConsItems.MC)
		}
		validStage1AuxCount = roundStruct.TotalAuxStage1MsgCount
	default:
		panic(len(validStage1))
	}

	// Check if we need to send the coin
	if sc.Decided > -1 && round >= sc.decidedRound {
		// we don't need to send the coin in the round we decided
	} else { // We need to send the coin
		// We can send the coin for the current round and the proposal for the next round if we have enough valid messages
		if validStage1AuxCount >= nmt && !roundStruct.sentCoin {
			logging.Infof("bcast coin idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round, roundStruct, sc.TestIndex)
			roundStruct.sentCoin = true
			coinMsg := sc.coin.GenerateCoinMessage(round,
				true, sc, binMsgState.coinState, binMsgState)
			if coinMsg != nil {
				sc.BroadcastCoin(coinMsg, mainChannel)
			}
		}
	}

	// Check if we can send the first message for the next round
	if validStage1AuxCount >= nmt && !nxtRoundStruct.sentInitialMessage && roundStruct.sentStage1Aux {
		if validStage1 == nil || len(validStage1) < 1 {
			panic(validStage1)
		}
		if validStage1[0] != types.BothBin {
			// we broadcast the aux value directly
			nxtRoundStruct.sentInitialMessage = true
			if !nxtRoundStruct.sentStage0Aux {
				nxtRoundStruct.sentStage0Aux = true
				// Broadcast aux
				sc.broadcastMsg(messages.HdrAuxStage0, validStage1[0], round+1, 0)
			}
		} else {
			if len(validStage1) > 1 {
				panic("should not have two valid values")
			}
			if nxtRoundStruct.supportBvInfoStage0[0] != nil ||
				nxtRoundStruct.supportBvInfoStage0[1] != nil {

				panic("should not have set pointer otherwise should not support coin")
			}
			// we have to support the value of the coin
			if roundStruct.gotCoin {
				nxtRoundStruct.sentInitialMessage = true
				nxtRoundStruct.bvInfoStage0[roundStruct.coinVal].sentMsg = true
				// we broadcast for the next round
				sc.broadcastMsg(messages.HdrBV0, roundStruct.coinVal, round+1, 0)
			}
		}
	}

	return true
}

// checkDone returns true if we don't need to process messages currently because we think everyone has decided.
func (sc *BinConsRnd6) checkDone(round types.ConsensusRound, nmt, t int) bool {
	_ = t
	if sc.terminated && round >= sc.terminatedRound {
		return true
	}
	if sc.Decided < 0 {
		return false
	}

	switch sc.StopOnCommit {
	case types.Immediate:
		sc.terminated = true
		sc.terminatedRound = sc.decidedRound
		return true
	case types.SendProof:
		if !sc.NoSignatures && !sc.terminated {
			// Send a proof of decision
			var hdr messages.HeaderID
			switch sc.decidedRound {
			case 1, 2:
				hdr = messages.HdrAuxStage0
			default:
				hdr = messages.HdrAuxStage1
			}

			prfMsgs := bincons1.GetBinConsCommitProof(hdr, sc, sc.getMsgState(), false, sc.ConsItems.MC)
			sc.BroadcastFunc(nil, sc.ConsItems, nil, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, prfMsgs...)
			sc.terminated = true
			sc.terminatedRound = sc.decidedRound
			return true
		}
		fallthrough
	case types.NextRound:
		// If in the decided round bot does not become valid in stage one then we do not need to go to the next round
		binMsgState := sc.getMsgState()
		if binMsgState.getBVState(0, 0, sc.decidedRound, sc.ConsItems.MC).msgCount < nmt ||
			binMsgState.getBVState(0, 1, sc.decidedRound, sc.ConsItems.MC).msgCount < nmt {

			return true
		}

		if sc.decidedRound == 1 {
			if round > sc.decidedRound+1 {
				return true
			}
		} else {
			// everyone decides 1 round after decision at most
			if round > sc.decidedRound {
				// sc.terminated = true
				return true
			}
		}
		return false
	default:
		panic(sc.StopOnCommit)
	}
}

func (sc *BinConsRnd6) broadcastMsg(msgType messages.HeaderID, binVal types.BinVal, round types.ConsensusRound, stage byte) {
	var msg messages.InternalSignedMsgHeader

	switch msgType {
	case messages.HdrBV0, messages.HdrBV1:
		msg = messagetypes.CreateBVMessageStage(binVal, round, stage)
	case messages.HdrAuxProof:
		auxMsg := messagetypes.NewAuxProofMessage(true)
		auxMsg.BinVal = binVal
		auxMsg.Round = round
		msg = auxMsg
	case messages.HdrAuxStage0:
		auxMsg := messagetypes.NewAuxStage0Message()
		auxMsg.BinVal = binVal
		auxMsg.Round = round
		msg = auxMsg
	case messages.HdrAuxStage1:
		auxMsg := messagetypes.NewAuxStage1Message()
		auxMsg.BinVal = binVal
		auxMsg.Round = round
		msg = auxMsg
	default:
		panic(msgType)
	}
	sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)
	if sc.CheckMemberLocalMsg(msg) {
		sc.BroadcastFunc(nil, sc.ConsItems, msg, !sc.NoSignatures,
			sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig)

	}
}

// HasReceivedProposal panics because BonCons has no proposals.
func (sc *BinConsRnd6) HasValidStarted() bool {
	panic("unused")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrAuxProof, it returns n-t, n for the thresholds.
func (sc *BinConsRnd6) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig, memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {
	switch hdr.GetID() {
	// TODO should have a different threshold for hdrBV, since it can be t+1?
	case messages.HdrAuxProof, messages.HdrBV0, messages.HdrBV1,
		messages.HdrAuxStage1, messages.HdrAuxStage0:

		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return coin.GetBufferCount(hdr, gc, memberChecker)
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrAuxProof are valid headerIDs.
func (sc *BinConsRnd6) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	var msg messages.InternalSignedMsgHeader
	switch headerType {
	case messages.HdrBV0:
		msg = messagetypes.NewBVMessage0()
	case messages.HdrBV1:
		msg = messagetypes.NewBVMessage1()
	case messages.HdrAuxProof:
		msg = messagetypes.NewAuxProofMessage(true)
	case messages.HdrAuxStage0:
		msg = messagetypes.NewAuxStage0Message()
	case messages.HdrAuxStage1:
		msg = messagetypes.NewAuxStage1Message()
	default:
		return coin.GetCoinHeader(emptyPub, gc, headerType)
	}
	if gc.NoSignatures {
		return sig.NewUnsignedMessage(types.ConsensusIndex{}, emptyPub, msg), nil
	}
	return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
		msg), nil
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *BinConsRnd6) HasDecided() bool {
	return sc.Decided > -1
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *BinConsRnd6) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *BinConsRnd6) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil, nil,
		sc.GeneralConfig.AllowConcurrent > 0
}

// GetDecision returns the binary value decided as a single byte slice.
func (sc *BinConsRnd6) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	if sc.Decided == -1 {
		panic("should have decided")
	}
	return nil, []byte{byte(sc.Decided)}, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
}

// GetConsType returns the type of consensus this instance implements.
func (sc *BinConsRnd6) GetConsType() types.ConsType {
	return types.BinConsRnd6Type
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *BinConsRnd6) PrevHasBeenReset() {
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *BinConsRnd6) SetNextConsItem(consinterface.ConsItem) {
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *BinConsRnd6) ShouldCreatePartial(messages.HeaderID) bool {
	return false
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// Broadcast a coin msg.
func (sc *BinConsRnd6) BroadcastCoin(coinMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	mainChannel.SendHeader(messages.AppendCopyMsgHeader(sc.PreHeaders, coinMsg),
		messages.IsProposalHeader(sc.Index, coinMsg.(messages.InternalSignedMsgHeader).GetBaseMsgHeader()),
		true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		sc.ConsItems.MC.MC.GetStats().IsRecordIndex())
}

// GenerateMessageState generates a new message state object given the inputs.
func (*BinConsRnd6) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewBinConsRnd6MessageState(false, gc)
}
