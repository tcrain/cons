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
package binconsrnd2

import (
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/coin"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

// bvInfo keeps info on a BVBroadcast
type bvInfo struct {
	// bVal types.BinVal // the binary value
	msgCount int // the number of messages received for this binary value
	round    types.ConsensusRound
	echod    bool
}

// auxRandRoundStruct keeps stats about auxProof messages received for a single round
type auxRandRoundStruct struct {
	round               types.ConsensusRound // The round this struct represents
	AuxBinNums          [2]int               // the number of signatures received for 0 an 1
	coinCount           int                  // number of signatures received supporting the coin
	TotalAuxBinMsgCount int                  // the total number of signatures receieved for both 0 and 1 from unique processes (i.e. this can be less than BinNums[0]+BinNums[1])

	bvInfo [2]bvInfo // bvInfo for the current round

	// Coin info for the previous round
	gotPrevCoin   bool
	prevCoinVal   types.BinVal
	supportBvInfo [2]*bvInfo // pointer to the previous instance for coin of the previous round

	// Coin information for the current round
	gotCoin     bool         // set to true once we know the coin
	sentCoin    bool         // set to true once the coin broadcast has been sent
	checkedCoin bool         // set to true if we have checked the coin at least once for decision
	coinVal     types.BinVal // The binary value of the coin

	// must support coin is calculated by the previous round
	// it is set to true if less than n-t aux messages supporting the same value are received before
	// broadcasting the coin
	// if true then the aux message for the round must be the coin value of the previous round
	mustSupportCoin bool

	// Proposal information for the round
	sentProposal bool // if a proposal has been sent for THIS round

	sentAux bool // if an AUX message has been sent for THIS round
}

// BinConsRnd2MessageState implements BinConsMessageStateInterface.
// It stores the messages of BinConsRnd2.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper // the simple message state is used to track the actual messages
	// initialBv [2]bvInfo // initial binary value broadcasts

	auxValues map[types.ConsensusRound]*auxRandRoundStruct // map from round index to auxRandRoundStruct
	index     types.ConsensusIndex                         // consensus index
	isMv      bool                                         // true if this is being used as part of mv cons
	mv0Valid  bool                                         // for use with multivalued reduction MvCons1, set to true if 0 is valid for round 1
	mv1Valid  bool                                         // for use with multivalued reduction MvCons1, set to true if 1 is valid for round 1
	gc        *generalconfig.GeneralConfig                 // configuration object
	coinState consinterface.CoinMessageStateInterface
	mutex     sync.RWMutex
}

// SetSimpleMessageStateWrapper sets the simple message state object, this is used by the multivale reduction MvCons1 since they share the
// same simple message state.
func (sms *MessageState) SetSimpleMessageStateWrapper(sm *messagestate.SimpleMessageStateWrapper) {
	sms.SimpleMessageStateWrapper = sm
}

// SetMv0Valid is called by the multivalue reduction MvCons1, when 0 becomes valid for round 1
func (sms *MessageState) SetMv0Valid() {
	// dont need locks because only accessed in main thread
	// sms.mutex.Lock()
	if !sms.mv0Valid {
		logging.Info("Setting 0 valid for mv cons for index", sms.index)
		sms.mv0Valid = true
	}
	// sms.mutex.Unlock()
}

// SetMv1Valid is called by the multivalue reduction MvCons1, when 1 becomes valid for round 1
func (sms *MessageState) SetMv1Valid() {
	// dont need locks because only accessed in main thread
	// sms.mutex.Lock()
	if !sms.mv1Valid {
		logging.Info("Setting 1 valid for mv cons for index", sms.index)
		sms.mv1Valid = true
	}
	// sms.mutex.Unlock()
}

// Lock the object
func (sms *MessageState) Lock() {
	sms.mutex.Lock()
}

// Unlock the object
func (sms *MessageState) Unlock() {
	sms.mutex.Unlock()
}

// NewBinConsRnd2MessageState generates a new BinConsRnd2MessageState object.
func NewBinConsRnd2MessageState(isMv bool,
	gc *generalconfig.GeneralConfig) *MessageState {

	return &MessageState{
		coinState:                 coin.GenerateCoinMessageStateInterface(gc.CoinType, isMv, int64(gc.TestIndex), gc),
		gc:                        gc,
		isMv:                      isMv,
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
}

// New creates a new empty BinConsRnd2MessageState object for the consensus index idx.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	_, presets := coin.GetFixedCoinPresets(sms.gc.UseFixedCoinPresets, sms.isMv)

	return &MessageState{
		coinState:                 sms.coinState.New(idx, presets),
		auxValues:                 make(map[types.ConsensusRound]*auxRandRoundStruct),
		index:                     idx,
		gc:                        sms.gc,
		isMv:                      sms.isMv,
		SimpleMessageStateWrapper: sms.SimpleMessageStateWrapper.NewWrapper(idx)}
}

// GetBinMsgState returns the base bin message state.
func (sms *MessageState) GetBinMsgState() bincons1.BinConsMessageStateInterface {
	return sms
}

// sentProposal returns true if a proposal has been sent for round+1 (i.e. the following round)
// if shouldSet is true, then sets to having sent the proposal for that round to true
func (sms *MessageState) SentProposal(round types.ConsensusRound,
	shouldSet bool, mc *consinterface.MemCheckers) bool {

	var ret bool
	sms.mutex.Lock()
	panic("TODO mv reduction")
	//ars := sms.getAuxRoundStruct(round, mc)
	//ret = ars.sentProposal
	if shouldSet {
		// ars.sentProposal = shouldSet
	}
	sms.mutex.Unlock()
	return ret
}

// GetProofs generates an AuxProofMessage continaining the signatures received for binVal in round round.
func (sms *MessageState) GetProofs(headerID messages.HeaderID, sigCount int,
	round types.ConsensusRound, binVal types.BinVal,
	_ sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if headerID != messages.HdrAuxProof {
		panic("invalid header type")
	}

	// t := mc.MC.GetFaultCount()
	// nmt := mc.MC.GetMemberCount() - t
	w := messagetypes.NewAuxProofMessage(false)
	w.Round = round
	w.BinVal = binVal
	// Add the sigs
	signedMsg, err := sms.SetupSignedMessage(w, false, sigCount, mc)
	if err != nil {
		logging.Error(err)
		panic(err)
	}
	return []*sig.MultipleSignedMessage{signedMsg}, nil
}

// GetValidMessage count returns the number of signed AuxProofMessages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount(round types.ConsensusRound, mc *consinterface.MemCheckers) int {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.getAuxRoundStruct(round, mc)
	return roundStruct.TotalAuxBinMsgCount
}

// getAuxRoundStruct returns the auxRandRoundStruct for round round (making a new one if necessary)
func (sms *MessageState) getAuxRoundStruct(round types.ConsensusRound,
	_ *consinterface.MemCheckers) *auxRandRoundStruct {

	item := sms.auxValues[round]
	if item == nil {
		item = &auxRandRoundStruct{round: round}
		item.bvInfo[0].round = round
		item.bvInfo[1].round = round
		if round == 1 {
			item.gotPrevCoin = true
			item.supportBvInfo[0] = &item.bvInfo[0]
			item.supportBvInfo[1] = &item.bvInfo[1]
			item.checkedCoin = true
			item.prevCoinVal = 1
		} else if round == 1 && sms.isMv { // when using mv reduction, first round coin value is always 1 so we can decide 1 right away
			item.gotCoin = true
			item.sentCoin = true
			item.prevCoinVal = 1
			item.checkedCoin = true
			item.coinVal = 1
			// TODO what about bv info?
			panic("TODO")
		}
		sms.auxValues[round] = item
	}
	return item
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The message must be an AuxProofMessage, since that is the only valid message type for BinConsRnd2.
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}

	if sms.coinState.CheckFinishedMessage(deser) {
		return nil, types.ErrCoinAlreadyProcessed
	}

	// Check the membership/signatures/duplicates
	ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
	if err != nil {
		return nil, err
	}

	// Update the round struct with the new pubs
	// we don't mind if someone already updated these in the mean time
	sms.mutex.Lock()

	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.BVMessage0, *messagetypes.BVMessage1:
		binVal, round, stage := messagetypes.GetBVMessageInfo(w)
		if stage != 0 {
			return nil, types.ErrInvalidStage
		}
		roundStruct := sms.getAuxRoundStruct(round, mc)

		if roundStruct.bvInfo[binVal].msgCount < deser.NewTotalSigCount {
			roundStruct.bvInfo[binVal].msgCount = deser.NewTotalSigCount
		}
	case *messagetypes.AuxProofMessage:
		if w.Round == 0 {
			return nil, types.ErrInvalidRound
		}
		roundStruct := sms.getAuxRoundStruct(w.Round, mc)

		binVal := w.BinVal
		// need to update the nubmer of sigs for that bin value, have to check since
		// someone with a large value could have updated it concurrently
		// because we call GotMsg from different threads
		if roundStruct.AuxBinNums[binVal] < deser.NewTotalSigCount {
			roundStruct.AuxBinNums[binVal] = deser.NewTotalSigCount
		}
		if roundStruct.TotalAuxBinMsgCount < deser.NewMsgIDSigCount {
			roundStruct.TotalAuxBinMsgCount = deser.NewMsgIDSigCount
		}

		if deser.NewMsgIDSigCount == 0 {
			panic(1)
		}
	default:
		if _, err := sms.coinState.GotMsg(sms, deser, gc, mc); err != nil {
			return nil, err
		}

	}

	sms.mutex.Unlock()
	return ret, nil
}

// Compute the valid bin vals for round r.
func (sms *MessageState) getValids(nmt int, t int, round types.ConsensusRound,
	mc *consinterface.MemCheckers) [2]bool {

	ret, _ := sms.getMostRecentValids(nmt, t, round, mc)
	return ret
}

func (sms *MessageState) getMostRecentValids(nmt, t int, round types.ConsensusRound,
	mc *consinterface.MemCheckers) (valids [2]bool, validRounds [2]types.ConsensusRound) {

	_ = t
	if round == 0 && sms.isMv {
		// false
	} else if round == 1 && sms.isMv {
		// special case when used in multivalue (set by SetMv1Valid() and SetMv0Valid())
		valids[0], valids[1] = sms.mv0Valid, sms.mv1Valid
	} else {
		ars := sms.getAuxRoundStruct(round, mc)
		if ars.supportBvInfo[0] != nil {
			if !ars.gotPrevCoin {
				panic("should have gotten coin")
			}
			for i := 0; i < 2; i++ {
				valids[i] = ars.supportBvInfo[i].msgCount >= nmt
				validRounds[i] = ars.supportBvInfo[i].round
			}
		}
	}
	return
}

// generateProofs generates an auxProofMessage containing signatures supporting binVal and round.
func (sms *MessageState) GenerateProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound,
	binVal types.BinVal, pub sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if headerID != messages.HdrAuxProof {
		panic("invalid header id")
	}

	// if MvCons and round == 2, est == 1, then we could not have decided 0, so we dont need proofs
	// (the proofs are the echo messages from mv-cons)
	if round <= 2 && binVal == 1 && sms.mv1Valid {
		panic("TODO")
		return nil, nil
	}
	// When using mv and round = 1 we dont need proofs for 0 since it is made valid on a timeout.
	if round == 1 && binVal == 0 && sms.mv0Valid {
		panic("TODO")
		return nil, nil
	}

	t := mc.MC.GetFaultCount()
	nmt := mc.MC.GetMemberCount() - t
	// var ret []*sig.MultipleSignedMessage

	sm, err := sms.GetProofs(messages.HdrAuxProof, sigCount, round, binVal, pub, mc)
	if err != nil {
		return sm, err
	}
	if len(sm) > 1 { // sanity check
		panic("shouldn't reach")
	}
	if sm[0].GetSigCount() < nmt {
		logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sigCount, nmt)
		err = types.ErrNotEnoughSigs
		panic(err) // TODO remove?
	}
	return sm, nil
}
