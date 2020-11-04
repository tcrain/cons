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
package binconsrnd1

import (
	"fmt"
	"github.com/tcrain/cons/consensus/coin"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/deserialized"
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

type BinConsRnd1 struct {
	cons.AbsConsItem
	coin          consinterface.CoinItemInterface
	Decided       int                  // -1 if a value has not yet been decided, 0 if 0 was decided, 1 if 1 was decided
	decidedRound  types.ConsensusRound // the round where the decision happened
	gotProposal   bool                 // for sanity checks
	IncludeProofs bool                 // true if messages should include signed values supporting the value in the message
	terminated    bool                 // set to true when consensus terminated
}

func (sc *BinConsRnd1) getMsgState() *MessageState { // TODO clean this up
	return sc.ConsItems.MsgState.(bincons1.BinConsMessageStateInterface).GetBinMsgState().(*MessageState)
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *BinConsRnd1) GetCommitProof() []messages.MsgHeader {
	return bincons1.GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), true, sc.ConsItems.MC)
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *BinConsRnd1) GetBinState(localOnly bool) ([]byte, error) {
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
func (sc *BinConsRnd1) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrBinPropose
}

// GenerateNewItem creates a new bin cons item.
func (*BinConsRnd1) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &BinConsRnd1{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.IncludeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.gotProposal = false
	newItem.Decided = -1
	newItem.decidedRound = 0
	if !types.CheckStrongCoin(gc.CoinType) {
		panic("BinConsRnd1 needs a strong coin")
	}
	newItem.coin = coin.GenerateCoinIterfaceItem(gc.CoinType)

	return newItem
}

// Start allows GetProposalIndex to return true.
func (sc *BinConsRnd1) Start(finishedLastRound bool) {
	_ = finishedLastRound
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// GetMVInitialRoundBroadcast returns the type of binary message that the multi-value reduction should broadcast for round 0.
func (sc *BinConsRnd1) GetMVInitialRoundBroadcast(val types.BinVal) messages.InternalSignedMsgHeader {
	auxMsg := messagetypes.NewAuxProofMessage(sc.AllowSupportCoin)
	auxMsg.BinVal = val
	auxMsg.Round = 1
	return auxMsg
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *BinConsRnd1) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *BinConsRnd1) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {
	sc.AbsGotProposal()
	bpm := hdr.(*messagetypes.BinProposeMessage)
	if bpm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	binMsgState := sc.getMsgState()

	binMsgState.Lock()
	// In case of recover, if we already sent a message then we don't want to send another
	roundStruct := binMsgState.getAuxRoundStruct(0, sc.ConsItems.MC)
	if roundStruct.sentProposal {
		binMsgState.Unlock()
		return nil
	}
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
func (sc *BinConsRnd1) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// GetBinDecided returns -1 if not decided, or the decided value and the decided round.
func (sc *BinConsRnd1) GetBinDecided() (int, types.ConsensusRound) {
	return sc.Decided, sc.decidedRound
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
func (sc *BinConsRnd1) ProcessMessage(
	deser *deserialized.DeserializedItem,
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
	// If we already decided then don't need to process this message
	if sc.terminated {
		logging.Infof("Got a msg, but already decided in round %v", sc.decidedRound)
		return false, false
	}

	var round types.ConsensusRound
	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
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

	// check if we can now advance rounds (note the message was already stored to the bincons state in BinConsRnd1MessageState.GotMsg
	for sc.CheckRound(nmt, t, round, sc.MainChannel) {
		round++
	}
	return true, true
}

// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
func (sc *BinConsRnd1) CanSkipMvTimeout() bool {
	msgState := sc.getMsgState()
	msgState.Lock()
	defer msgState.Unlock()

	return msgState.getAuxRoundStruct(2, sc.ConsItems.MC).TotalBinMsgCount >=
		sc.ConsItems.MC.MC.GetMemberCount()-sc.ConsItems.MC.MC.GetFaultCount()
}

// SetInitialState does noting for this algorithm.
func (sc *BinConsRnd1) SetInitialState([]byte) {}

// CheckRound checks for the given round if enough messages have been received to progress to the next round
// and return true if it can.
func (sc *BinConsRnd1) CheckRound(nmt int, t int, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) bool {

	binMsgState := sc.getMsgState()
	binMsgState.Lock()
	defer binMsgState.Unlock()

	// binMsgState.checkSupportCoin(round, sc.GeneralConfig, sc.ConsItems.MC)

	// get the round state
	roundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)

	numMsgs := roundStruct.TotalBinMsgCount // roundStruct.BinNums[0] + roundStruct.BinNums[1]
	// We can only perform an action if we have enough messages (n-t)
	if numMsgs < nmt {
		return false
	}

	var prevRoundState *auxRandRoundStruct
	for r := types.ConsensusRound(1); r <= round; r++ {
		prevRoundState = binMsgState.getAuxRoundStruct(r-1, sc.ConsItems.MC)
		// We cannot continue if we don't know the coin for the previous round, or if we haven't sent a proposal
		if (sc.CheckMemberLocal() && !prevRoundState.sentProposal) || len(binMsgState.coinState.GetCoins(r-1)) == 0 {
			return false
		}
	}

	// Compute the values still valid for this round (note this only relies on the values sent during the previous round)
	valids, _, _ := binMsgState.getMostRecentValids(nmt, t, round, sc.ConsItems.MC)

	// If we know the coin we can check for decision
	coinVals := binMsgState.coinState.GetCoins(round)
	if round > 0 && len(coinVals) > 0 {
		coinVal := coinVals[0]
		notCoin := 1 - coinVal

		// If we got enough messages for the mod and mod is valid for the round then we can decide
		if round > 0 && valids[coinVal] && roundStruct.BinNums[coinVal] >= nmt {
			// Sanity checks
			if sc.Decided > -1 && types.BinVal(sc.Decided) != coinVal {
				panic("Bad cons 1")
			}
			if sc.Decided == -1 { // decide!
				logging.Infof("Decided proc %v bin round %v, binval %v, index %v", sc.TestIndex, round, coinVal, sc.Index)
				sc.decidedRound = round
				sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, coinVal == 0)
			}
			if roundStruct.BinNums[notCoin] >= nmt { // sanity check
				logging.Error("check", coinVal, round, valids[coinVal], roundStruct.BinNums[coinVal],
					roundStruct.BinNums[notCoin], nmt, sc.Index)
				panic("More than t faulty")
			}
			sc.Decided = int(coinVal)
			sc.SetDecided()
			// Only send next round msg after deciding if necessary
			// TODO is other stopping mechanism better?
			if sc.StopOnCommit == types.Immediate {
				sc.terminated = true
			}
			if sc.StopOnCommit != types.SendProof && (roundStruct.BinNums[notCoin] == 0 || !valids[notCoin] || sc.StopOnCommit == types.Immediate) {

				return true
			}
		}
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
		validCount += roundStruct.BinNums[0]
	}
	if valids[1] {
		validCount += roundStruct.BinNums[1]
	}
	// We can send the coin for the current round and the proposal for the next round if we have enough valid messages
	if validCount >= nmt {
		if sc.checkDone(round, nmt, t) {
			return true
		}
		if sc.GeneralConfig.AllowSupportCoin {
			sc.checkBroadcastsAuxCoin(nmt, t, round, roundStruct, prevRoundState, binMsgState, mainChannel)
		} else {
			sc.checkBroadcasts(nmt, t, round, roundStruct, prevRoundState, binMsgState, mainChannel)
		}
	} else {
		return false
		// sanity checks? TODO
	}

	return true
}

// checkDone returns true if we don't need to process messages currently because we think everyone has decided.
func (sc *BinConsRnd1) checkDone(round types.ConsensusRound, nmt, t int) bool {
	if sc.terminated {
		return true
	}
	if sc.Decided < 0 {
		return false
	}
	if round < sc.decidedRound { // this message is from an earlier round
		return true
	}

	switch sc.StopOnCommit {
	case types.Immediate:
		sc.terminated = true
		return true
	case types.SendProof:
		if sc.AllowSupportCoin && sc.IncludeProofs {
			// in this case the proof was already contained in the aux message sent with the coin
			sc.terminated = true
			return true
		} else {
			prfMsgs := bincons1.GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), false, sc.ConsItems.MC)
			/*		// Send a proof of decision
					binMsgState := sc.getMsgState()
					nxtPrfs, err := binMsgState.GenerateProofs(messages.HdrAuxProof, nmt, sc.decidedRound, types.BinVal(sc.Decided),
						sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
					if err != nil {
						logging.Error(err, "est:", sc.Decided, "round:", sc.decidedRound, "cons idx:", sc.Index)
						panic("should have proofs")
					}
					prfMsgs := make([]messages.MsgHeader, len(nxtPrfs))
					for i, nxt := range nxtPrfs {
						prfMsgs[i] = nxt
					}*/
			sc.BroadcastFunc(nil, sc.ConsItems, nil, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, prfMsgs...)
		}
		sc.terminated = true
		return true
	case types.NextRound:
		// If for the decided round, we have no not coin messages, then we think we are done (at least until we get more messages)
		// because everyone we know would have decided in this round.
		round := sc.decidedRound
		binMsgState := sc.getMsgState()
		decidedRoundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
		valids, _, _ := binMsgState.getMostRecentValids(nmt, t, round, sc.ConsItems.MC)
		decidedCoinVal := binMsgState.coinState.GetCoins(round)[0]
		notDecidedCoinVal := 1 - decidedCoinVal
		if decidedRoundStruct.BinNums[notDecidedCoinVal] == 0 || !valids[notDecidedCoinVal] {
			return true
		}

		// If we got the decided value as a coin in the round after the decided round then everyone is done.
		round++
		// nxtRoundStruct := binMsgState.getAuxRoundStruct(round, sc.ConsItems.MC)
		coinVals := []types.BinVal{decidedCoinVal}
		nxtCoinVal := binMsgState.coinState.GetCoins(round)
		for len(nxtCoinVal) > 0 {
			coinVals = append(coinVals, nxtCoinVal[0])
			if nxtCoinVal[0] == decidedCoinVal {
				logging.Infof("terminated index %v, decided round %v, terminated round %v, coin vals %v",
					sc.Index.Index, sc.decidedRound, round, coinVals)
				sc.terminated = true
				return true
			}
			round++
			nxtCoinVal = binMsgState.coinState.GetCoins(round)
		}
		return false
	default:
		panic(sc.StopOnCommit)
	}
}

// check if need to broadcast messages when AllowSupportCoin=true
func (sc *BinConsRnd1) checkBroadcastsAuxCoin(nmt int, t int, round types.ConsensusRound,
	roundStruct, prevRoundState *auxRandRoundStruct, binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	if !roundStruct.checkedCoin {
		roundStruct.checkedCoin = true
		var est types.BinVal // the estimate for the next round
		if round == 0 {      // 0 round
			// prefer 1
			if roundStruct.BinNums[1] > t {
				est = 1
			} else {
				if roundStruct.BinNums[0] <= t {
					panic("should have more than t 0")
				}
				est = 0
			}
		} else { // all other rounds

			// Sanity check
			if prevRoundState == nil || len(binMsgState.coinState.GetCoins(prevRoundState.round)) == 0 ||
				(sc.CheckMemberLocal() && !prevRoundState.sentProposal) {
				// (sc.CheckMemberLocal() && !prevRoundState.sentCoin) {

				panic("invalid ordering")
			}
			// compute what value we will support next round
			validsR, _, _ := binMsgState.getMostRecentValids(nmt, t, round, sc.ConsItems.MC)
			if validsR[1] && roundStruct.BinNums[1] >= nmt { // got n-t 1
				roundStruct.mustSupportNextRound = 1
			} else if validsR[1] && validsR[0] && roundStruct.BinNums[0] > 0 && roundStruct.BinNums[1] > 0 { // got mix of 1 and 0, support the coin
				roundStruct.mustSupportNextRound = types.Coin
			} else if validsR[0] && roundStruct.BinNums[0] >= nmt { // got n-t 0
				roundStruct.mustSupportNextRound = 0
			} else {
				panic("should not reach")
			}
			est = roundStruct.mustSupportNextRound
		}

		if !roundStruct.sentProposal && sc.CheckMemberLocal() {
			// Store the values
			roundStruct.sentProposal = true

			logging.Infof("bcast idx %v, round %v, est %v, %+v, %v", sc.Index.Index, round+1, est, roundStruct, sc.TestIndex)

			// Broadcast the message for the next round
			auxMsg := messagetypes.NewAuxProofMessage(sc.GeneralConfig.AllowSupportCoin)
			var prfMsgs []messages.MsgHeader
			auxMsg.BinVal = est
			auxMsg.Round = round + 1
			sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round + 1)

			// Set to true before checking if we are a member, since check member will always
			// give the same result for this round
			if sc.CheckMemberLocalMsg(auxMsg) {
				// add the coin
				if round > binMsgState.maxCoinPresetRound {
					coinMsg := sc.coin.GenerateCoinMessage(round,
						true, sc, binMsgState.coinState, binMsgState)
					if coinMsg == nil && ((round > 0 && !sc.getMsgState().isMv) || round > 1) {
						panic("must broadcast coin and aux message together")
					}
					prfMsgs = append(prfMsgs, coinMsg)
				}
				if sc.IncludeProofs {
					// collect signatures to support your choice
					sigCount, _, _, err := sc.GetBufferCount(auxMsg, sc.GeneralConfig, sc.ConsItems.MC)
					if err != nil {
						panic(err)
					}
					var supportEsts []types.BinVal
					// we generate proofs for the previous rounds since we don't know the value of the coin for this round yet
					// if we are supporting the coin, then we generate proofs for both values but from the previous rounds
					if est == types.Coin {
						supportEsts = []types.BinVal{0, 1}
					} else {
						supportEsts = []types.BinVal{est}
					}
					// Normally the proof round would be for the same as the aux (round + 1)
					// But since we don't know the coin for this round yet, we look for proofs from the previous round
					prfRound := round
					for _, est := range supportEsts {
						nxtPrfs, err := binMsgState.GenerateProofs(messages.HdrAuxProof, sigCount, prfRound, est,
							sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
						if err != nil {
							logging.Error(err, "est:", est, "round:", prfRound, "cons idx:", sc.Index)
							panic("should have proofs")
						} else {
							for _, nxt := range nxtPrfs {
								prfMsgs = append(prfMsgs, nxt)
							}
						}
					}
				}
				sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
					sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfMsgs...)
				// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfMsgs...)
			}
		}
	}
	return true
}

// check if need to broadcast messages when AllowSupportCoin=false
func (sc *BinConsRnd1) checkBroadcasts(nmt int, t int, round types.ConsensusRound,
	roundStruct, prevRoundState *auxRandRoundStruct, binMsgState *MessageState,
	mainChannel channelinterface.MainChannel) bool {

	// if we haven't sent a coin message for the following round then we can send it
	if round > 0 && !roundStruct.checkedCoin {
		roundStruct.checkedCoin = true
		// compute what value we will support next round
		validsR, _, _ := binMsgState.getMostRecentValids(nmt, t, round, sc.ConsItems.MC)
		if validsR[1] && roundStruct.BinNums[1] >= nmt { // got n-t 1
			roundStruct.mustSupportNextRound = 1
		} else if validsR[1] && validsR[0] && roundStruct.BinNums[0] > 0 &&
			roundStruct.BinNums[1] > 0 { // got mix of 1 and 0, support the coin
			roundStruct.mustSupportNextRound = types.Coin
		} else if validsR[0] && roundStruct.BinNums[0] >= nmt { // got n-t 0
			roundStruct.mustSupportNextRound = 0
		} else {
			panic("should not reach")
		}

		coinMsg := sc.coin.GenerateCoinMessage(round,
			true, sc, binMsgState.coinState, binMsgState)
		if coinMsg != nil {
			sc.BroadcastCoin(coinMsg, mainChannel)
		}

	}
	// if we got the coin and we haven't sent a message for the following round
	// and we are a member of this consensus then send a proposal

	coins := sc.getMsgState().coinState.GetCoins(round)
	if (round == 0 || len(coins) > 0) && !roundStruct.sentProposal && sc.CheckMemberLocal() {

		var est types.BinVal // the estimate for the next round
		if round == 0 {      // 0 round
			// just take the one with the most votes
			if roundStruct.BinNums[1] >= roundStruct.BinNums[0] {
				est = 1
			}
		} else { // all other rounds

			// Sanity check
			if prevRoundState == nil || len(sc.getMsgState().coinState.GetCoins(prevRoundState.round)) == 0 ||
				(!prevRoundState.sentProposal && sc.CheckMemberLocal()) {
				panic("invalid ordering")
			}

			// coinVal values
			coinVal := coins[0]
			switch roundStruct.mustSupportNextRound {
			case 0:
				est = 0
			case 1:
				est = 1
			case types.Coin:
				est = coinVal
			}
		}
		// Store the values
		roundStruct.sentProposal = true

		// Broadcast the message for the next round
		auxMsg := messagetypes.NewAuxProofMessage(sc.GeneralConfig.AllowSupportCoin)
		var prfMsg []*sig.MultipleSignedMessage
		auxMsg.BinVal = est
		auxMsg.Round = round + 1
		// Set to true before checking if we are a member, since check member will always
		// give the same result for this round
		if sc.CheckMemberLocalMsg(auxMsg) {
			if sc.IncludeProofs {
				// collect signatures to support your choice
				var err error
				sigCount, _, _, err := sc.GetBufferCount(auxMsg, sc.GeneralConfig, sc.ConsItems.MC)
				if err != nil {
					panic(err)
				}
				prfMsg, err = binMsgState.GenerateProofs(messages.HdrAuxProof, sigCount, round+1, est,
					sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
				if err != nil {
					logging.Error(err, "est:", est, "round:", round+1, "cons idx:", sc.Index)
					panic("should have proofs")
					// prfMsg = nil
				}
			}
			prfs := make([]messages.MsgHeader, len(prfMsg))
			for i, nxt := range prfMsg {
				prfs[i] = nxt
			}
			sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfs...)
			// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfs...)
		}
	}

	return true
}

// HasReceivedProposal panics because BonCons has no proposals.
func (sc *BinConsRnd1) HasValidStarted() bool {
	panic("unused")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrAuxProof, it returns n-t, n for the thresholds.
func (*BinConsRnd1) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrAuxProof:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return coin.GetBufferCount(hdr, gc, memberChecker)
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrAuxProof are valid headerIDs.
func (sc *BinConsRnd1) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrAuxProof:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub,
			messagetypes.NewAuxProofMessage(gc.AllowSupportCoin)), nil
	default:
		return coin.GetCoinHeader(emptyPub, gc, headerType)
	}
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *BinConsRnd1) HasDecided() bool {
	return sc.Decided > -1
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *BinConsRnd1) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *BinConsRnd1) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil, nil,
		sc.GeneralConfig.AllowConcurrent > 0
}

// GetDecision returns the binary value decided as a single byte slice.
func (sc *BinConsRnd1) GetDecision() (sig.Pub, []byte, types.ConsensusIndex, types.ConsensusIndex) {
	if sc.Decided == -1 {
		panic("should have decided")
	}
	return nil, []byte{byte(sc.Decided)}, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1),
		types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) + 1)
}

// GetConsType returns the type of consensus this instance implements.
func (sc *BinConsRnd1) GetConsType() types.ConsType {
	return types.BinConsRnd1Type
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *BinConsRnd1) PrevHasBeenReset() {
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *BinConsRnd1) SetNextConsItem(_ consinterface.ConsItem) {
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *BinConsRnd1) ShouldCreatePartial(_ messages.HeaderID) bool {
	return false
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// Broadcast a coin msg.
func (sc *BinConsRnd1) BroadcastCoin(coinMsg messages.MsgHeader,
	mainChannel channelinterface.MainChannel) {

	sts := sc.ConsItems.MC.MC.GetStats()
	mainChannel.SendHeader(messages.AppendCopyMsgHeader(sc.PreHeaders, coinMsg),
		messages.IsProposalHeader(sc.Index, coinMsg.(messages.InternalSignedMsgHeader).GetBaseMsgHeader()),
		true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
		sts.IsRecordIndex(), sts)
}

// GenerateMessageState generates a new message state object given the inputs.
func (*BinConsRnd1) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewBinConsRnd1MessageState(false, gc)
}
