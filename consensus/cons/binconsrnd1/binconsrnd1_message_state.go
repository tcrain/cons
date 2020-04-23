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

package binconsrnd1

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

// auxRandRoundStruct keeps stats about auxProof messages received for a single round
type auxRandRoundStruct struct {
	round            types.ConsensusRound // The round this struct represents
	BinNums          [2]int               // the number of signatures received for 0 an 1
	coinCount        int                  // number of signatures received supporting the coin
	TotalBinMsgCount int                  // the total number of signatures receieved for both 0 and 1 from unique processes (i.e. this can be less than BinNums[0]+BinNums[1])

	// Coin information for the current round
	checkedCoin bool // set to true if we have checked the coin at least once for decision
	// mustSupportNextRound is calculated when I send the coin
	// if I have n-t some value, then I must support that in the next round
	// Otherwise I must support the coin
	mustSupportNextRound types.BinVal

	// used if we can send coin instead of binval
	coinHeaders []messages.InternalSignedMsgHeader

	// Proposal information for the next round
	sentProposal bool // if a proposal has been sent for the NEXT round
}

/*type getBinConsMsgStateInterface interface {
	GetBinConsMsgState() BinConsMessageStateInterface
}
*/
/*// BinConsMessageStateInterface extends the messagestate.MessageState interface with some addition operations for the
// storage of messages specific to BinConsRnd1.
// Note that operations can be called from multiple threads.
// The public methods are concurrency safe, the others should be synchronized as needed.
type BinConsMessageStateInterface interface {
	consinterface.MessageState
	// GetProofs generates an AuxProofMessage containing the signatures received for binVal in round round.
	GetProofs(sigCount int, round types.ConsensusRound, binVal types.BinVal, pub sig.Pub, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error)
	// getAuxRoundStruct returns the auxRandRoundStruct for round round (making a new one if necessary)
	getAuxRoundStruct(round types.ConsensusRound) *auxRandRoundStruct
	// Compute the valid bin vals for round r.
	getValids(nmt int, t int, round types.ConsensusRound) [2]bool
	// generateProofs generates an auxProofMessage containing signatures supporting binVal and round.
	GenerateProofs(sigCount int, round types.ConsensusRound, binVal types.BinVal, pub sig.Pub,
		mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error)
	// Lock the object
	Lock()
	// Unlock the object
	Unlock()

	// SetSimpleMessageStateWrapper sets the simple message state object, this is used by the multivale reduction MvCons1 since they share the
	// same simple message state.
	SetSimpleMessageStateWrapper(sm *messagestate.SimpleMessageStateWrapper)
	// GetValidMessage count returns the number of signed AuxProofMessages received from different processes in round round.
	GetValidMessageCount(round types.ConsensusRound) int
}*/

// BinConsRnd1MessageState implements BinConsMessageStateInterface.
// It stores the messages of BinConsRnd1.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                                              // the simple message state is used to track the actual messages
	auxValues                               map[types.ConsensusRound]*auxRandRoundStruct // map from round index to auxRandRoundStruct
	index                                   types.ConsensusIndex                         // consensus index
	isMv                                    bool                                         // true if this is being used as part of mv cons
	mv0Valid                                bool                                         // for use with multivalued reduction MvCons1, set to true if 0 is valid for round 1
	mv1Valid                                bool                                         // for use with multivalued reduction MvCons1, set to true if 1 is valid for round 1
	gc                                      *generalconfig.GeneralConfig                 // configuration object
	coinState                               consinterface.CoinMessageStateInterface
	maxCoinPresetRound                      types.ConsensusRound
	mutex                                   sync.RWMutex
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

// NewBinConsRnd1MessageState generates a new BinConsRnd1MessageState object.
func NewBinConsRnd1MessageState(isMv bool,
	gc *generalconfig.GeneralConfig) *MessageState {

	return &MessageState{
		coinState:                 coin.GenerateCoinMessageStateInterface(gc.CoinType, isMv, int64(gc.TestIndex), gc),
		gc:                        gc,
		isMv:                      isMv,
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
}

// New creates a new empty BinConsRnd1MessageState object for the consensus index idx.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	maxPresetRound, presets := coin.GetFixedCoinPresets(sms.gc.UseFixedCoinPresets, sms.isMv)

	return &MessageState{
		maxCoinPresetRound:        maxPresetRound,
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
	ars := sms.getAuxRoundStruct(round, mc)
	ret = ars.sentProposal
	if shouldSet {
		ars.sentProposal = shouldSet
	}
	sms.mutex.Unlock()
	return ret
}

// GetProofs generates an AuxProofMessage containing the signatures received for binVal in round round.
func (sms *MessageState) GetProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound, binVal types.BinVal,
	_ sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if headerID != messages.HdrAuxProof {
		panic("invalid header id")
	}

	var gotSigCount int
	var getCoin bool // Get proof for coin
	switch round {
	case 0, 1:
	default:
		// Check if we should use the actual value or the coin
		if sms.gc.AllowSupportCoin {
			coinVals := sms.coinState.GetCoins(round - 1)
			if len(coinVals) > 0 && coinVals[0] == binVal { // sanity check
				getCoin = true
			}
		}
	}

	var ret []*sig.MultipleSignedMessage
	for i := 0; i < 2; i++ {
		var supBinVal types.BinVal
		switch i {
		case 0:
			if getCoin {
				supBinVal = types.Coin
			} else {
				continue
			}
		case 1:
			supBinVal = binVal
		}
		w := messagetypes.NewAuxProofMessage(sms.gc.AllowSupportCoin)
		w.Round = round
		w.BinVal = supBinVal
		// Add the sigs
		signedMsg, err := sms.SetupSignedMessage(w, false, sigCount, mc)
		if err != nil {
			logging.Warning(err)
		}
		if signedMsg != nil {
			gotSigCount += signedMsg.GetSigCount()
			ret = append(ret, signedMsg)
		}
		if gotSigCount >= sigCount { // we have enough signatures
			break
		}
	}
	if sigCount > gotSigCount {
		panic("didn't get enough proofs")
	}
	return ret, nil
}

// GetValidMessage count returns the number of signed AuxProofMessages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount(round types.ConsensusRound, mc *consinterface.MemCheckers) int {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.getAuxRoundStruct(round, mc)
	return roundStruct.TotalBinMsgCount
}

// getAuxRoundStruct returns the auxRandRoundStruct for round round (making a new one if necessary)
func (sms *MessageState) getAuxRoundStruct(round types.ConsensusRound,
	mc *consinterface.MemCheckers) *auxRandRoundStruct {

	item := sms.auxValues[round]
	if item == nil {
		item = &auxRandRoundStruct{round: round}
		if round == 0 {
			if sms.isMv {
				item.checkedCoin = true
			}
		} else if round == 1 && sms.isMv { // when using mv reduction, first round coin value is always 1 so we can decide 1 right away

			if sms.gc.AllowSupportCoin {
				auxMsg := messagetypes.NewAuxProofMessage(sms.gc.AllowSupportCoin)
				auxMsg.BinVal = 1
				auxMsg.Round = 2
				auxMsgCoin := messagetypes.NewAuxProofMessage(sms.gc.AllowSupportCoin)
				auxMsgCoin.BinVal = types.Coin
				auxMsgCoin.Round = 2

				nxtRndStruct := sms.getAuxRoundStruct(2, mc)
				// track the count of these messages for unique signers
				sms.Sms.TrackTotalSigCount(mc, auxMsg, auxMsgCoin)
				nxtRndStruct.coinHeaders = []messages.InternalSignedMsgHeader{auxMsg, auxMsgCoin}
			}
		}
		sms.auxValues[round] = item
	}
	return item
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The message must be an AuxProofMessage, since that is the only valid message type for BinConsRnd1.
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
	defer sms.mutex.Unlock()

	switch w := deser.Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(type) {
	case *messagetypes.AuxProofMessage:
		roundStruct := sms.getAuxRoundStruct(w.Round, mc)

		if roundStruct.TotalBinMsgCount < deser.NewMsgIDSigCount {
			roundStruct.TotalBinMsgCount = deser.NewMsgIDSigCount
		}
		binVal := w.BinVal
		switch binVal {
		case types.Coin:
			// we got a message supporting the coin
			if !gc.AllowSupportCoin || w.Round <= 1 {
				panic("should have caught earlier")
			}
			if roundStruct.coinCount < deser.NewTotalSigCount {
				roundStruct.coinCount = deser.NewTotalSigCount
			}

			prevCoins := sms.coinState.GetCoins(w.Round - 1)
			if len(prevCoins) == 0 {
				break
			}
			binVal = prevCoins[0]
			fallthrough
		default:
			// need to update the nubmer of sigs for that bin value, have to check since
			// someone with a large value could have updated it concurrently
			// because we call GotMsg from different threads
			if roundStruct.BinNums[binVal] < deser.NewTotalSigCount {
				roundStruct.BinNums[binVal] = deser.NewTotalSigCount
			}

			if deser.NewMsgIDSigCount == 0 {
				panic(1)
			}
			sms.checkSupportCoin(w.Round, gc, mc)
		}
	default:
		round, err := sms.coinState.GotMsg(sms, deser, gc, mc)
		if err != nil {
			return nil, err
		}
		sms.checkSupportCoin(round+1, gc, mc)
	}

	return ret, nil
}

func (sms *MessageState) checkSupportCoin(round types.ConsensusRound, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) {

	// If we allow coin support, then we must calculate for both coin support types messages
	if gc.AllowSupportCoin && round > 1 {
		prevCoins := sms.coinState.GetCoins(round - 1)
		roundStruct := sms.getAuxRoundStruct(round, mc)
		if len(prevCoins) > 0 {
			if len(roundStruct.coinHeaders) == 0 {
				for _, nxt := range append(prevCoins, types.Coin) {
					msg := messagetypes.NewAuxProofMessage(gc.AllowSupportCoin)
					msg.Round = round
					msg.BinVal = nxt
					roundStruct.coinHeaders = append(roundStruct.coinHeaders, msg)
				}
				sms.Sms.TrackTotalSigCount(mc, roundStruct.coinHeaders...)
			}
			nmt := mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
			// update the count for supporters of the coin
			count, _ := sms.Sms.GetTotalSigCount(mc, roundStruct.coinHeaders...)
			if roundStruct.BinNums[prevCoins[0]] >= nmt && roundStruct.BinNums[prevCoins[0]] > count {
				panic("support for value and coin should always be larger that support for just value")
			}
			roundStruct.BinNums[prevCoins[0]] = count
		}
	}
}

// Compute the valid bin vals for round r.
func (sms *MessageState) getValids(nmt int, t int, round types.ConsensusRound,
	mc *consinterface.MemCheckers) [2]bool {

	ret, _, _ := sms.getMostRecentValids(nmt, t, round, mc)
	return ret
}

func (sms *MessageState) getMostRecentValids(nmt, t int, round types.ConsensusRound,
	mc *consinterface.MemCheckers) (valids [2]bool,
	validRounds [2]types.ConsensusRound, items [2]*auxRandRoundStruct) {

	var got0, got1 bool

	for !got0 || !got1 {
		switch round {
		case 0: // round 0 all values are valid
			// in MV round 0 and 1 are handled by the multivalue code
			if !sms.isMv { // otherwise both 0 and 1 are valid for round 0
				if !got0 {
					got0 = true
					valids[0] = true
					validRounds[0] = round
				}
				if !got1 {
					got1 = true
					valids[1] = true
					validRounds[1] = round
				}
			}
		case 1: // round 1 we need t from round 0
			nxtStruct := sms.getAuxRoundStruct(round-1, mc)
			if len(sms.coinState.GetCoins(round-1)) == 0 {
				panic("should have coin")
			}
			if !sms.isMv {
				if !got0 {
					got0 = true
					if nxtStruct.BinNums[0] > t {
						valids[0] = true
						items[0] = nxtStruct
						validRounds[0] = round
					}
				}
				if !got1 {
					got1 = true
					if nxtStruct.BinNums[1] > t {
						valids[1] = true
						items[1] = nxtStruct
						validRounds[1] = round
					}
				}
			} else {
				// special case when used in multivalue (set by SetMv1Valid() and SetMv0Valid())
				if !got0 && sms.mv0Valid {
					valids[0] = true
					items[0] = nxtStruct
					validRounds[0] = round
				}
				if !got1 && sms.mv1Valid {
					valids[1] = true
					items[1] = nxtStruct
					validRounds[1] = round
				}
				got1 = true
				got0 = true
			}
		default: // Check the coin
			if round <= 1 {
				panic(round)
			}
			nxtStruct := sms.getAuxRoundStruct(round-1, mc)
			coins := sms.coinState.GetCoins(round - 1)
			if len(coins) == 0 {
				panic("should have coin")
			}
			switch coins[0] {
			case 0: // For 1 to be valid we need n-t 1 otw 0 could have been decided
				if !got1 {
					got1 = true
					if nxtStruct.BinNums[1] >= nmt {
						valids[1] = true
						items[1] = nxtStruct
						validRounds[1] = round
					}
				}
			case 1: // For 0 to be valid we need n-t 0, otw 1 could have been decided
				if !got0 {
					got0 = true
					if nxtStruct.BinNums[0] >= nmt {
						valids[0] = true
						items[0] = nxtStruct
						validRounds[0] = round
					}
				}
			default:
				panic(coins)
			}
		}
		round -= 1
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
		return nil, nil
	}
	// When using mv and round = 1 we dont need proofs for 0 since it is made valid on a timeout.
	if round == 1 && binVal == 0 && sms.mv0Valid {
		return nil, nil
	}

	t := mc.MC.GetFaultCount()
	nmt := mc.MC.GetMemberCount() - t
	// var ret []*sig.MultipleSignedMessage

	valids, validRounds, _ := sms.getMostRecentValids(nmt, t, round, mc)
	if !valids[binVal] {
		panic("tried to get invalid bin val")
	}
	switch validRounds[binVal] {
	case 0:
		// no proofs needed for round 0
		return nil, nil
	case 1:
		// take the signatures from round 0 supporting binVal
		// sm, err := sms.GetProofs(headerID, sigCount, 0, binVal, pub, mc)
		sm, err := sms.GetProofs(headerID, mc.MC.GetFaultCount()+1, 0, binVal, pub, mc)
		if err != nil {
			return sm, err
		}
		if sm[0].GetSigCount() < mc.MC.GetFaultCount() {
			logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
				mc.MC.GetFaultCount())
			err = types.ErrNotEnoughSigs
			panic(err)
		}
		return sm, err
	default:
		sm, err := sms.GetProofs(headerID, sigCount, validRounds[binVal]-1, binVal, pub, mc)
		if err != nil {
			return sm, err
		}
		if len(sm) > 1 { // sanity check
			// not enough proofs, so we need coin messages
			prvCoins := sms.coinState.GetCoins(validRounds[binVal] - 2)
			if len(prvCoins) == 0 || prvCoins[0] != binVal { // sanity check
				panic("shouldn't reach")
			}
		}
		var sigCount int
		for _, nxt := range sm {
			sigCount += nxt.GetSigCount()
		}
		if sigCount < nmt {
			logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sigCount, nmt)
			err = types.ErrNotEnoughSigs
			panic(err) // TODO remove?
		}
		return sm, nil
	}
}
