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
package binconsrnd2

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

type BinConsRnd2 struct {
	cons.AbsConsItem
	coin            consinterface.CoinItemInterface
	Decided         int                  // -1 if a value has not yet been decided, 0 if 0 was decided, 1 if 1 was decided
	decidedRound    types.ConsensusRound // the round where the decision happened
	gotProposal     bool                 // for sanity checks
	terminated      bool                 // set to true when consensus terminated
	terminatedRound types.ConsensusRound

	sentBVR0 [2]bool // if we have sent BV messages in round 1
}

func (sc *BinConsRnd2) getMsgState() *MessageState { // TODO clean this up
	return sc.ConsItems.MsgState.(bincons1.BinConsMessageStateInterface).GetBinMsgState().(*MessageState)
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *BinConsRnd2) GetCommitProof() []messages.MsgHeader {
	return bincons1.GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), true, sc.ConsItems.MC)
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *BinConsRnd2) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *BinConsRnd2) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrBinPropose
}

// GenerateNewItem creates a new bin cons item.
func (*BinConsRnd2) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &BinConsRnd2{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.Decided = -1
	newItem.gotProposal = false
	newItem.Decided = -1
	newItem.decidedRound = 0
	newItem.coin = coin.GenerateCoinIterfaceItem(gc.CoinType)

	return newItem
}

// Start allows GetProposalIndex to return true.
func (sc *BinConsRnd2) Start() {
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *BinConsRnd2) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *BinConsRnd2) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {
	sc.AbsGotProposal()
	bpm := hdr.(*messagetypes.BinProposeMessage)
	if bpm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}

	logging.Infof("Got local proposal for index %v bin val %v", sc.Index, bpm.BinVal)
	if !sc.sentBVR0[bpm.BinVal] {
		sc.sentBVR0[bpm.BinVal] = true
		sc.getMsgState().Lock()
		roundStruct := sc.getMsgState().getAuxRoundStruct(1, sc.ConsItems.MC)
		if roundStruct.supportBvInfo[bpm.BinVal].echod == true {
			panic("should not be true")
		}
		roundStruct.supportBvInfo[bpm.BinVal].echod = true
		sc.getMsgState().Unlock()

		bvMsg := messagetypes.CreateBVMessage(bpm.BinVal, 1)
		sc.BroadcastFunc(nil, sc.ConsItems, bvMsg, !sc.NoSignatures,
			sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, sc.CommitProof...)
	}
	return nil
}

// NeedsConcurrent returns 1.
func (sc *BinConsRnd2) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// GetBinDecided returns -1 if not decided, or the decided value and the decided round.
func (sc *BinConsRnd2) GetBinDecided() (int, types.ConsensusRound) {
	return sc.Decided, sc.decidedRound
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
func (sc *BinConsRnd2) ProcessMessage(
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
	if sc.terminated && round > sc.terminatedRound {
		logging.Infof("Got a msg for round %v, but already decided in round %v, index %v", round, sc.decidedRound, sc.Index.Index)
		return false, false
	}

	// check if we can now advance rounds (note the message was already stored to the bincons state in BinConsRnd2MessageState.GotMsg
	for sc.CheckRound(nmt, t, round, sc.MainChannel) {
		round++
	}
	return true, true
}

// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
func (sc *BinConsRnd2) CanSkipMvTimeout() bool {
	msgState := sc.getMsgState()
	msgState.Lock()
	defer msgState.Unlock()

	return msgState.getAuxRoundStruct(2, sc.ConsItems.MC).TotalAuxBinMsgCount >
		sc.ConsItems.MC.MC.GetMemberCount()-sc.ConsItems.MC.MC.GetFaultCount()
}

// SetInitialState does noting for this algorithm.
func (sc *BinConsRnd2) SetInitialState([]byte) {}

// CheckRound checks for the given round if enough messages have been received to progress to the next round
// and return true if it can.
func (sc *BinConsRnd2) CheckRound(nmt int, t int, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) bool {

	binMsgState := sc.getMsgState()
	binMsgState.Lock()
	defer binMsgState.Unlock()

	// get the round state
	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)

	if coinVals := binMsgState.coinState.GetCoins(round); len(coinVals) > 0 {
		// roundStruct := sms.getAuxRoundStruct(round, sc.ConsItems.MC)
		roundStruct.gotCoin = true
		roundStruct.coinVal = coinVals[0]

		// update the round struct for the next round
		roundStructp1 := binMsgState.getAuxRoundStruct(round+1, sc.ConsItems.MC)
		roundStructp1.gotPrevCoin = true
		roundStructp1.prevCoinVal = roundStruct.coinVal

		nxtRoundStruct := roundStruct
		for nxtRoundStruct.supportBvInfo[0] != nil && nxtRoundStruct.gotCoin {
			rsp1 := binMsgState.getAuxRoundStruct(nxtRoundStruct.round+1, sc.ConsItems.MC)

			if !nxtRoundStruct.gotPrevCoin || !rsp1.gotPrevCoin { // sanity check
				panic("should have got prev coin")
			}
			rsp1.supportBvInfo[nxtRoundStruct.coinVal] = nxtRoundStruct.supportBvInfo[nxtRoundStruct.coinVal]
			rsp1.supportBvInfo[1-nxtRoundStruct.coinVal] = &rsp1.bvInfo[1-nxtRoundStruct.coinVal]
			rsp1.gotPrevCoin = true
			rsp1.prevCoinVal = nxtRoundStruct.coinVal

			nxtRoundStruct = rsp1
		}
	}

	for r := types.ConsensusRound(1); true; r++ {
		rs := sc.getMsgState().getAuxRoundStruct(r, sc.ConsItems.MC)
		if !rs.gotPrevCoin {
			break
		}
		for i, nxt := range rs.supportBvInfo {
			if nxt.round > 1 {
				prvRs := sc.getMsgState().getAuxRoundStruct(nxt.round, sc.ConsItems.MC)
				if prvRs.prevCoinVal == types.BinVal(i) {
					panic("bad setup")
				}
			}
		}
	}

	if round > 1 && !roundStruct.gotPrevCoin { // Be sure we know the coin of the previous round
		return false
	}

	// Compute the values still valid for this round (note this only relies on the values sent during the previous round)
	valids := binMsgState.getValids(nmt, t, round, sc.ConsItems.MC)

	// If we know the coin we can check for decision
	if roundStruct.gotCoin {
		coinVal := roundStruct.coinVal
		notCoin := 1 - coinVal

		// If we got enough messages for the mod and mod is valid for the round then we can decide
		if valids[coinVal] && roundStruct.AuxBinNums[coinVal] >= nmt {
			// Sanity checks
			if sc.Decided > -1 && types.BinVal(sc.Decided) != coinVal {
				panic("Bad cons 1")
			}
			if sc.Decided == -1 { // decide!
				logging.Infof("Decided bin round %v, binval %v, index %v", round, coinVal, sc.Index)
				sc.decidedRound = round
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, coinVal == 0)
			}
			if roundStruct.AuxBinNums[notCoin] >= nmt { // sanity check
				logging.Error("check", coinVal, round, valids[coinVal], roundStruct.AuxBinNums[coinVal],
					roundStruct.AuxBinNums[notCoin], nmt, sc.Index)
				panic("More than t faulty")
			}
			sc.Decided = int(coinVal)
			// Only send next round msg after deciding if necessary
			// TODO is other stopping mechanism better?
			//if roundStruct.AuxBinNums[notCoin] == 0 || !valids[notCoin] || sc.StopOnCommit == types.Immediate {
			//	return true
			//}
		}
	}
	// Check if we need to send BVBroadcast or Aux broadcast
	sc.checkBVAuxBroadcasts(nmt, t, round, roundStruct, sc.MainChannel)
	if sc.checkDone(round, nmt, t) {
		return false
	}

	// validCount is (at least) the total number of valid messages from distinct processes
	// this is different than roundStruct.TotalBinMsgCount because that is the total number of messages
	// received from different processes (even if their bin value isn't valid)
	// roundStruct.TotalBinMsgCount >= nmt is checked earlier and ensures we have received at least n-t
	// messages from distinct processes
	// so if just 1 or 0 is valid (but not both) then validCount >= nmt ensures that we have received n-t valid
	// messages from distinct processes
	// if both 1 and 0 are valid then roundStruct.TotalBinMsgCount >= nmt ensures that we have received n-t
	// valid messages from distinct processes
	var validCount int
	if valids[0] {
		validCount += roundStruct.AuxBinNums[0]
	}
	if valids[1] {
		validCount += roundStruct.AuxBinNums[1]
	}
	numMsgs := roundStruct.TotalAuxBinMsgCount
	// We can only perform an action if we have enough messages (n-t) from different processes
	if numMsgs < nmt {
		return false
	}
	// We can send the coin for the current round and the proposal for the next round if we have enough valid messages
	if validCount >= nmt {
		sc.checkCoinBroadcasts(nmt, t, round, roundStruct, binMsgState, mainChannel)
	}

	return true
}

// checkDone returns true if we don't need to process messages currently because we think everyone has decided.
func (sc *BinConsRnd2) checkDone(rnd types.ConsensusRound, nmt, t int) bool {
	if sc.terminated && rnd >= sc.terminatedRound {
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
			prfMsgs := bincons1.GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), false, sc.ConsItems.MC)
			sc.BroadcastFunc(nil, sc.ConsItems, nil, !sc.NoSignatures,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, prfMsgs...)

			sc.terminated = true
			sc.terminatedRound = sc.decidedRound
		}
		fallthrough
	case types.NextRound:
		// If for the decided round, we have no not coin messages, then we think we are done (at least until we get more messages)
		// because everyone we know would have decided in this round.
		binMsgState := sc.getMsgState()
		decidedRoundStruct := binMsgState.getAuxRoundStruct(sc.decidedRound, sc.ConsItems.MC)
		valids, _ := binMsgState.getMostRecentValids(nmt, t, sc.decidedRound, sc.ConsItems.MC)
		notCoin := 1 - decidedRoundStruct.coinVal
		if decidedRoundStruct.AuxBinNums[notCoin] == 0 || !valids[notCoin] {
			return true
		}

		// If we got the decided value as a coin in the round after the decided round then everyone is done.
		for rnd > sc.decidedRound {
			roundStruct := binMsgState.getAuxRoundStruct(rnd, sc.ConsItems.MC)
			if roundStruct.gotCoin {
				if roundStruct.coinVal == decidedRoundStruct.coinVal {
					logging.Infof("terminated index %v, decided round %v, terminated round %v, coin vals %v",
						sc.Index.Index, sc.decidedRound, rnd, roundStruct.coinVal)
					sc.terminated = true
					sc.terminatedRound = rnd
					return false
				}
				rnd++
			} else {
				break
			}
		}
		return false
	default:
		panic(sc.StopOnCommit)
	}
}

// check if need to broadcast a BV message
func (sc *BinConsRnd2) checkBVAuxBroadcasts(nmt int, t int, round types.ConsensusRound,
	roundStruct *auxRandRoundStruct,
	mainChannel channelinterface.MainChannel) {

	if round == 1 { // For round 1 we can echo both values
		for _, i := range []types.BinVal{1, 0} { // we prefer 1 if we can
			if sc.sentBVR0[i] { // we already sent a message for this bin value
				continue
			}
			if !roundStruct.bvInfo[i].echod && roundStruct.bvInfo[i].msgCount > t { // we need to echo the message
				// Store the values
				roundStruct.bvInfo[i].echod = true
				sc.sentBVR0[i] = true
				// Broadcast the message for the next round
				bvMsg := messagetypes.CreateBVMessage(i, round)
				// Set to true before checking if we are a member, since check member will always
				// give the same result for this round
				sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)
				if sc.CheckMemberLocalMsg(bvMsg.GetMsgID()) {
					sc.BroadcastFunc(nil, sc.ConsItems, bvMsg, !sc.NoSignatures,
						sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig)
				}
			}
		}
	} else if roundStruct.supportBvInfo[0] != nil {
		if !roundStruct.gotPrevCoin {
			panic("should have gotten coinVal")
		}
		coinVal := roundStruct.prevCoinVal
		if roundStruct.bvInfo[coinVal].msgCount > t { // sanity check
			panic("more than t faulty")
		}

		for _, i := range []types.BinVal{1, 0} { // we prefer 1 if we can

			if !roundStruct.supportBvInfo[i].echod && roundStruct.supportBvInfo[i].msgCount > t { // we need to echo the message
				// Store the values
				roundStruct.supportBvInfo[i].echod = true
				// Broadcast the message for the next round
				auxMsg := messagetypes.CreateBVMessage(i, roundStruct.supportBvInfo[i].round)
				// Set to true before checking if we are a member, since check member will always
				// give the same result for this round
				sc.ConsItems.MC.MC.GetStats().AddParticipationRound(roundStruct.supportBvInfo[i].round)
				if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
					sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, !sc.NoSignatures,
						sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig)
					// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfs...)
				}
			}
		}
	}

	if !roundStruct.sentAux && roundStruct.checkedCoin && roundStruct.gotPrevCoin {
		if !roundStruct.gotPrevCoin {
			panic("should have gotten coin")
		}
		var shouldSend bool
		var sendVal types.BinVal
		if roundStruct.supportBvInfo[roundStruct.prevCoinVal].msgCount >= nmt || roundStruct.mustSupportCoin {
			sendVal = roundStruct.prevCoinVal
			shouldSend = true
		} else if roundStruct.supportBvInfo[1-roundStruct.prevCoinVal].msgCount >= nmt {
			sendVal = 1 - roundStruct.prevCoinVal
			shouldSend = true
		}
		if shouldSend {
			roundStruct.sentAux = true
			auxMsg := messagetypes.NewAuxProofMessage(false)
			auxMsg.BinVal = sendVal
			auxMsg.Round = round
			// send the aux
			sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round)
			if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
				sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, !sc.NoSignatures,
					sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig)
			}
		}
	}

}

// check if need to coin or messages for the next round
func (sc *BinConsRnd2) checkCoinBroadcasts(nmt int, t int, round types.ConsensusRound,
	roundStruct *auxRandRoundStruct, binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) {

	// if we haven't sent a coin message for the round then we can send it
	nxtRoundStruct := binMsgState.getAuxRoundStruct(round+1, sc.ConsItems.MC)
	if !nxtRoundStruct.checkedCoin {
		nxtRoundStruct.checkedCoin = true
		// compute what value we will support next round
		validsR := binMsgState.getValids(nmt, t, round, sc.ConsItems.MC)
		if validsR[0] && roundStruct.AuxBinNums[0] >= nmt { // got n-t 0
			nxtRoundStruct.mustSupportCoin = false
		} else if validsR[1] && roundStruct.AuxBinNums[1] >= nmt { // got n-t 1
			nxtRoundStruct.mustSupportCoin = false
		} else if validsR[1] && validsR[0] && roundStruct.AuxBinNums[0] > 0 &&
			roundStruct.AuxBinNums[1] > 0 { // got mix of 1 and 0, support the coin

			nxtRoundStruct.mustSupportCoin = true
		} else {
			panic("should not reach")
		}

		if !roundStruct.sentCoin {
			roundStruct.sentCoin = true
			coinMsg := sc.coin.GenerateCoinMessage(round,
				true, sc, binMsgState.coinState, binMsgState)
			if coinMsg != nil {
				sc.BroadcastCoin(coinMsg, mainChannel)
			}
		}
	}
	// if we got the coin and we haven't sent a message for the following round
	// and we are a member of this consensus then send a proposal
	if roundStruct.gotCoin && sc.CheckMemberLocal() {

		var est types.BinVal // the estimate for the next round

		// Sanity check
		//if prevRoundState == nil || !prevRoundState.gotCoin ||
		//	(!prevRoundState.sentProposal && sc.CheckMemberLocal()) ||
		//	(!prevRoundState.sentCoin && sc.CheckMemberLocal()) {
		//	panic("invalid ordering")
		//}

		// coinVal values
		coinVal := roundStruct.coinVal

		if nxtRoundStruct.mustSupportCoin {
			est = coinVal
		} else {
			// valids, _ := binMsgState.getMostRecentValids(nmt, t, round, sc.ConsItems.MC)
			if roundStruct.AuxBinNums[0] >= nmt {
				est = 0
			} else if roundStruct.AuxBinNums[1] >= nmt {
				est = 1
			} else {
				panic("should be a valid value")
			}
		}
		// We can send the proposal for the next round
		if !nxtRoundStruct.sentProposal {
			nxtRoundStruct.sentProposal = true
			if est != coinVal {
				// Store the values
				if !nxtRoundStruct.supportBvInfo[est].echod {
					nxtRoundStruct.supportBvInfo[est].echod = true
					// Broadcast the BV message for the next round
					bvMsg := messagetypes.CreateBVMessage(est, round+1)
					// Set to true before checking if we are a member, since check member will always
					// give the same result for this round
					sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round + 1)
					if sc.CheckMemberLocalMsg(bvMsg.GetMsgID()) {
						sc.BroadcastFunc(nil, sc.ConsItems, bvMsg, !sc.NoSignatures,
							sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig)
						// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfs...)
					}
				}
			} else { // we broadcast the aux for the next round directly
				// Store the values
				if !nxtRoundStruct.sentAux {
					nxtRoundStruct.sentAux = true
					// Broadcast the BV message for the next round
					auxMsg := messagetypes.NewAuxProofMessage(false)
					auxMsg.BinVal = est
					auxMsg.Round = round + 1
					// Set to true before checking if we are a member, since check member will always
					// give the same result for this round
					if sc.CheckMemberLocalMsg(auxMsg.GetMsgID()) {
						sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, !sc.NoSignatures,
							sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig)
						// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfs...)
					}
				}
			}
		}
	}

}

// HasReceivedProposal panics because BonCons has no proposals.
func (sc *BinConsRnd2) HasReceivedProposal() bool {
	panic("unused")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrAuxProof, it returns n-t, n for the thresholds.
func (sc *BinConsRnd2) GetBufferCount(hdr messages.MsgIDHeader,
	gc *generalconfig.GeneralConfig, memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	// TODO should have a different threshold for hdrBV, since it can be t+1?
	case messages.HdrAuxProof, messages.HdrBV0, messages.HdrBV1:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return coin.GetBufferCount(hdr, gc, memberChecker)
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrAuxProof are valid headerIDs.
func (sc *BinConsRnd2) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig,
	headerType messages.HeaderID) (messages.MsgHeader, error) {

	var hdr messages.InternalSignedMsgHeader
	switch headerType {
	case messages.HdrBV0:
		hdr = messagetypes.NewBVMessage0()
	case messages.HdrBV1:
		hdr = messagetypes.NewBVMessage1()
	case messages.HdrAuxProof:
		hdr = messagetypes.NewAuxProofMessage(gc.AllowSupportCoin)
	default:
		return coin.GetCoinHeader(emptyPub, gc, headerType)
	}
	if gc.NoSignatures {
		return sig.NewUnsignedMessage(types.ConsensusIndex{}, emptyPub,
			hdr), nil
	}
	return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
		hdr), nil
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *BinConsRnd2) HasDecided() bool {
	return sc.Decided > -1
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *BinConsRnd2) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *BinConsRnd2) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil, nil, true
}

// GetDecision returns the binary value decided as a single byte slice.
func (sc *BinConsRnd2) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	if sc.Decided == -1 {
		panic("should have decided")
	}
	return nil, []byte{byte(sc.Decided)}, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
}

// GetConsType returns the type of consensus this instance implements.
func (sc *BinConsRnd2) GetConsType() types.ConsType {
	return types.BinConsRnd2Type
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *BinConsRnd2) PrevHasBeenReset() {
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *BinConsRnd2) SetNextConsItem(_ consinterface.ConsItem) {
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *BinConsRnd2) ShouldCreatePartial(_ messages.HeaderID) bool {
	return false
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// Broadcast a coin msg.
func (sc *BinConsRnd2) BroadcastCoin(coinMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	mainChannel.SendHeader(messages.AppendCopyMsgHeader(sc.PreHeaders, coinMsg),
		messages.IsProposalHeader(sc.Index, coinMsg.(messages.InternalSignedMsgHeader).GetBaseMsgHeader()),
		true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		sc.ConsItems.MC.MC.GetStats().IsRecordIndex())
}

/*// Broadcast an aux proof message.
func (sc *BinConsRnd2) Broadcast(nxtCoordPub sig.Pub, auxMsg messages.InternalSignedMsgHeader,
	forwardFunc channelinterface.NewForwardFuncFilter, mainChannel channelinterface.MainChannel,
	additionalMsgs ...messages.MsgHeader) {

	cons.DoConsBroadcast(nxtCoordPub, auxMsg, additionalMsgs, forwardFunc,
		sc.ConsItems, mainChannel, sc.GeneralConfig)
}
*/
// GenerateMessageState generates a new message state object given the inputs.
func (*BinConsRnd2) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewBinConsRnd2MessageState(false, gc)
}

// Collect is called when the item is being garbage collected.
func (sc *BinConsRnd2) Collect() {
}
