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

package binconsrnd5

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
	round                types.ConsensusRound // The round this struct represents
	BinNumsAux           [2]int               // the number of signatures received for 0 an 1
	BinNumsBoth          [3]int               // the number of signatures received for 0, 1, or bot
	coinCount            int                  // number of signatures received supporting the coin
	TotalBinAuxMsgCount  int                  // the total number of aux signatures receieved for both 0 and 1 from unique processes (i.e. this can be less than BinNums[0]+BinNums[1])
	TotalBinBothMsgCount int                  // the total number of auxBoth signatures receieved for both 0 and 1 from unique processes (i.e. this can be less than BinNums[0]+BinNums[1])

	// Coin information for the current round
	gotCoin  bool // set to true once we know the coin
	sentCoin bool // set to true once the coin broadcast has been sent
	// checkedCoin bool         // set to true if we have checked the coin at least once for decision
	coinVals []types.BinVal // The valid binary values of the coin (can be multiple values if using weak coin)
	// mustSupportNextRound is calculated when I send the coin
	// if I have n-t some value, then I must support that in the next round
	// Otherwise I must support the coin
	// mustSupportNextRound types.BinVal

	// used if we can send coin instead of binval
	coinHeaders []messages.InternalSignedMsgHeader

	// so we can track how many valid auxBoth messages we have received
	auxBothHeaders []messages.InternalSignedMsgHeader

	sentAuxProof bool // if an AuxProof has been sent for THIS round
	sentAuxBoth  bool // if an AuxBoth has been sent for THIS round
}

// BinConsRnd5MessageState implements BinConsMessageStateInterface.
// It stores the messages of BinConsRnd5.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                                              // the simple message state is used to track the actual messages
	auxValues                               map[types.ConsensusRound]*auxRandRoundStruct // map from round index to auxRandRoundStruct
	index                                   types.ConsensusIndex                         // consensus index
	isMv                                    bool                                         // true if this is being used as part of mv cons
	mv0Valid                                bool                                         // for use with multivalued reduction MvCons1, set to true if 0 is valid for round 1
	mv1Valid                                bool                                         // for use with multivalued reduction MvCons1, set to true if 1 is valid for round 1
	gc                                      *generalconfig.GeneralConfig                 // configuration object
	coinState                               consinterface.CoinMessageStateInterface
	maxPresetCoinRound                      types.ConsensusRound
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

// getValidsAuxProof returns what values are valid for aux proof messages in this round
func (sms *MessageState) getValidsAuxProof(round types.ConsensusRound, mc *consinterface.MemCheckers) (
	binsValid [2]bool, supportCoin bool) {

	t := mc.MC.GetFaultCount()
	nmt := mc.MC.GetMemberCount() - t

	switch round {
	case 0:
		return [2]bool{true, true}, false
	case 1:
		prvRoundStruct := sms.getAuxRoundStruct(round-1, mc)
		if prvRoundStruct.TotalBinAuxMsgCount < nmt { // be sure we have enough messages
			return [2]bool{false, false}, false
		}
		return [2]bool{prvRoundStruct.BinNumsAux[0] > t, prvRoundStruct.BinNumsAux[1] > t}, false
	case 2:
		round0Struct := sms.getAuxRoundStruct(0, mc)
		round1Struct := sms.getAuxRoundStruct(1, mc)
		if round1Struct.TotalBinAuxMsgCount < nmt { // be sure we have enough messages
			return [2]bool{false, false}, false
		}
		return [2]bool{round1Struct.BinNumsAux[0] >= nmt, round0Struct.BinNumsAux[1] > t}, false
	default:
		prvRoundStruct := sms.getAuxRoundStruct(round-1, mc)
		if prvRoundStruct.TotalBinBothMsgCount < nmt { // be sure we have enough (valid) messages
			return [2]bool{false, false}, false
		}

		// either we got n-t auxProof msgs for a value in the previous round
		ret := [2]bool{prvRoundStruct.BinNumsAux[0] >= nmt, prvRoundStruct.BinNumsAux[1] >= nmt}
		if ret == [2]bool{true, true} {
			panic("more than t faulty")
		}
		if ret[0] == true || ret[1] == true {
			return ret, false
		}

		// or we got n-t coin vals and we can support the coin
		var supportCoin bool
		if (sms.gc.AllowSupportCoin || (!sms.gc.AllowSupportCoin && prvRoundStruct.gotCoin)) &&
			prvRoundStruct.BinNumsBoth[2] >= nmt {

			supportCoin = true
			for _, coinVal := range prvRoundStruct.coinVals {
				ret[coinVal] = true
			}
		}
		return ret, supportCoin
	}
}

// getValidsAuxBoth returns what values are valid for aux both messages in this round
func (sms *MessageState) getValidsAuxBoth(round types.ConsensusRound, mc *consinterface.MemCheckers) [3]bool {
	t := mc.MC.GetFaultCount()
	nmt := mc.MC.GetMemberCount() - t
	roundStruct := sms.getAuxRoundStruct(round, mc)
	prvRoundStruct := sms.getAuxRoundStruct(round-1, mc)

	switch round {
	case 0, 1:
		panic("aux both not supported in round 0 or 1")
	case 2:
		// either n-t for each bin value from aux in round 2
		// or got n-t 0 in round 1, and t+1 1 in round 0
		round1Struct := prvRoundStruct
		round0Struct := sms.getAuxRoundStruct(0, mc)
		return [3]bool{roundStruct.BinNumsAux[0] >= nmt, roundStruct.BinNumsAux[1] >= nmt,
			round1Struct.gotCoin && round1Struct.BinNumsAux[0] >= nmt && round0Struct.BinNumsAux[1] > t &&
				roundStruct.TotalBinAuxMsgCount >= nmt}
	default:
		// either n-t for each bin value from aux in the current round
		// or n-t binboth messages supporting BinBoth in the previous round
		return [3]bool{roundStruct.BinNumsAux[0] >= nmt, roundStruct.BinNumsAux[1] >= nmt,
			prvRoundStruct.gotCoin && prvRoundStruct.BinNumsBoth[2] >= nmt && roundStruct.TotalBinAuxMsgCount >= nmt}
	}
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

// NewBinConsRnd5MessageState generates a new BinConsRnd5MessageState object.
func NewBinConsRnd5MessageState(isMv bool,
	gc *generalconfig.GeneralConfig) *MessageState {

	if gc.AllowSupportCoin { // TODO this is ok, but add a panic if weak coin
		// panic(1)
	}

	return &MessageState{
		coinState:                 coin.GenerateCoinMessageStateInterface(gc.CoinType, isMv, int64(gc.TestIndex), gc),
		gc:                        gc,
		isMv:                      isMv,
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
}

// New creates a new empty BinConsRnd5MessageState object for the consensus index idx.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	_, presets := coin.GetFixedCoinPresets(sms.gc.UseFixedCoinPresets, sms.isMv)

	// we must fix round 0 and 1 to 1, following rounds we can use other presets
	var maxPresetRound types.ConsensusRound = 1
	myPresets := []struct {
		Round types.ConsensusRound
		Val   types.BinVal
	}{{0, 1}, {1, 1}}
	for _, nxt := range presets {
		if nxt.Round >= 1 {
			nxt.Round++
			maxPresetRound = nxt.Round
			myPresets = append(myPresets, nxt)
		}
	}

	return &MessageState{
		maxPresetCoinRound:        maxPresetRound,
		coinState:                 sms.coinState.New(idx, myPresets),
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
	ret = ars.sentAuxProof
	if shouldSet {
		ars.sentAuxProof = shouldSet
	}
	sms.mutex.Unlock()
	return ret
}

// GetProofs generates an AuxProofMessage continaining the signatures received for binVal in round round.
// The caller should already know what proofs it needs.
// If the proof is for HdrAuxBoth then binVal must equal BothBin for which there are n-t messages received in the round
// If the proof is for HdrAuxProof then the binval must be a binary value for which there are n-t messages received in the round
func (sms *MessageState) GetProofs(headerID messages.HeaderID, sigCount int,
	round types.ConsensusRound, binVal types.BinVal,
	pub sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	_ = pub
	switch headerID {
	case messages.HdrAuxBoth:
		if binVal != types.BothBin {
			// panic("must call for auxBoth message with binBoth bin value")
		}
		w := messagetypes.NewAuxBothMessage()
		w.Round = round
		w.BinVal = binVal

		// Add the sigs
		signedMsg, err := sms.SetupSignedMessage(w, false, sigCount, mc)
		if err != nil {
			logging.Error(err)
			panic(err)
		}
		return []*sig.MultipleSignedMessage{signedMsg}, nil
	case messages.HdrAuxProof: // This is a message that will prove a HdrAuxBoth message is valid in the same round
		var getCoin bool   // Get proof for coin
		var getNormal bool // Get proof for binVal
		switch round {
		case 0, 1, 2:
			getNormal = true
		default:
			t := mc.MC.GetFaultCount()
			nmt := mc.MC.GetMemberCount() - t
			// Check if we should use the actual value or the coin
			var coinCount int
			prvStruct := sms.getAuxRoundStruct(round-1, mc)
			if prvStruct.gotCoin && prvStruct.coinVals[0] == binVal { // sanity check
				coinCount = sms.getAuxRoundStruct(round, mc).coinCount
			}
			if coinCount < nmt {
				getNormal = true
			}
			if coinCount > 0 {
				getCoin = true
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
				if getNormal {
					supBinVal = binVal
				} else {
					continue
				}
			}
			w := messagetypes.NewAuxProofMessage(sms.gc.AllowSupportCoin)
			w.Round = round
			w.BinVal = supBinVal

			// Add the sigs
			signedMsg, err := sms.SetupSignedMessage(w, false, sigCount, mc)
			if err != nil {
				logging.Error(err)
				panic(err)
			}
			ret = append(ret, signedMsg)
		}
		return ret, nil
	default:
		panic(headerID)
	}
}

// GetValidMessageCount returns the number of signed AuxProofMessages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount(round types.ConsensusRound, mc *consinterface.MemCheckers) int {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.getAuxRoundStruct(round, mc)
	return roundStruct.TotalBinAuxMsgCount
}

// GetValidAuxMessageCount returns the number of signed AuxProofMessages received from different processes in round round.
func (sms *MessageState) GetValidBothMessageCount(round types.ConsensusRound, mc *consinterface.MemCheckers) int {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.getAuxRoundStruct(round, mc)
	return roundStruct.TotalBinBothMsgCount
}

// getAuxRoundStruct returns the auxRandRoundStruct for round round (making a new one if necessary)
func (sms *MessageState) getAuxRoundStruct(round types.ConsensusRound,
	mc *consinterface.MemCheckers) *auxRandRoundStruct {

	item := sms.auxValues[round]
	if item == nil {
		item = &auxRandRoundStruct{round: round}
		if sms.maxPresetCoinRound >= round {
			item.sentCoin = true
			item.gotCoin = true
		}
		if round == 0 || round == 1 {
			item.gotCoin = true
			item.sentCoin = true
			item.sentAuxBoth = true
			if sms.isMv {
				// item.checkedCoin = true
			}
		} else if round == 1 && sms.isMv { // when using mv reduction, first round coin value is always 1 so we can decide 1 right away
			item.gotCoin = true
			item.sentCoin = true
			item.coinVals = append(item.coinVals, 1)

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
// The message must be an AuxProofMessage, since that is the only valid message type for BinConsRnd5.
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig, mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

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
		if roundStruct.TotalBinAuxMsgCount < deser.NewMsgIDSigCount {
			roundStruct.TotalBinAuxMsgCount = deser.NewMsgIDSigCount
		}
		binVal := w.BinVal
		switch binVal {
		case types.Coin:
			// we got a message supporting the coin
			if !gc.AllowSupportCoin || w.Round <= sms.maxPresetCoinRound {
				return nil, types.ErrCoinProofNotSupported
			}
			if roundStruct.coinCount < deser.NewTotalSigCount {
				roundStruct.coinCount = deser.NewTotalSigCount
			}
			prevRoundStruct := sms.getAuxRoundStruct(w.Round-1, mc)
			if !prevRoundStruct.gotCoin {
				break
			}
			if len(prevRoundStruct.coinVals) > 1 {
				panic("cant support coin and have a weak coin")
			}
			binVal = prevRoundStruct.coinVals[0]
			fallthrough
		default:
			// need to update the nubmer of sigs for that bin value, have to check since
			// someone with a large value could have updated it concurrently
			// because we call GotMsg from different threads
			if roundStruct.BinNumsAux[binVal] < deser.NewTotalSigCount {
				roundStruct.BinNumsAux[binVal] = deser.NewTotalSigCount
			}

			// If we got n-t of a single value then we must set that as a valid value for the auxBoth messages
			nmt := mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
			foundVal := -1
			for i := 0; i < 2; i++ {
				if roundStruct.BinNumsAux[i] >= nmt {
					foundVal = i
				}
			}
			if w.Round > 1 && len(roundStruct.auxBothHeaders) == 0 && foundVal >= 0 {
				hdr1 := messagetypes.NewAuxBothMessage()
				hdr1.Round = w.Round
				hdr1.BinVal = types.BinVal(foundVal)
				hdr2 := messagetypes.NewAuxBothMessage()
				hdr2.Round = w.Round
				hdr2.BinVal = types.BothBin
				roundStruct.auxBothHeaders = []messages.InternalSignedMsgHeader{hdr1, hdr2}
				sms.Sms.TrackTotalSigCount(mc, roundStruct.auxBothHeaders...)
				sms.updateBothMsgCount(roundStruct, mc)
				/*				count := sms.Sms.GetTotalSigCount(mc, roundStruct.auxBothHeaders...)
								if roundStruct.TotalBinBothMsgCount >= nmt &&
									roundStruct.TotalBinBothMsgCount > count {

									panic("should not have more messages for a single value")
								}
								roundStruct.TotalBinBothMsgCount = count
				*/
			}

			// If we allow coin support, then we must calculate for both coin support types messages
			if gc.AllowSupportCoin && w.Round > sms.maxPresetCoinRound+1 {
				prevRoundStruct := sms.getAuxRoundStruct(w.Round-1, mc)
				if len(prevRoundStruct.coinVals) > 1 {
					panic("cant support coin and have a weak coin")
				}
				if prevRoundStruct.gotCoin {
					// update the count for supporters of the coin
					count, _ := sms.Sms.GetTotalSigCount(mc, roundStruct.coinHeaders...)
					if roundStruct.BinNumsAux[prevRoundStruct.coinVals[0]] >= nmt &&
						roundStruct.BinNumsAux[prevRoundStruct.coinVals[0]] > count {

						panic("support for value and coin should always be larger that support for just value")
					}
					roundStruct.BinNumsAux[prevRoundStruct.coinVals[0]] = count
				}
			}

			if deser.NewMsgIDSigCount == 0 {
				panic(1)
			}
		}
	case *messagetypes.AuxBothMessage:
		if w.Round < 2 {
			return nil, types.ErrInvalidHeader
		}

		roundStruct := sms.getAuxRoundStruct(w.Round, mc)

		binVal := w.BinVal
		if roundStruct.BinNumsBoth[binVal] < deser.NewTotalSigCount {
			roundStruct.BinNumsBoth[binVal] = deser.NewTotalSigCount
		}
		// If we know what values are valid then we get the total msg count for those valid values
		if len(roundStruct.auxBothHeaders) > 0 {
			sms.updateBothMsgCount(roundStruct, mc)
			// roundStruct.TotalBinBothMsgCount = sms.Sms.GetTotalSigCount(mc, roundStruct.auxBothHeaders...)
		} else if binVal == types.BothBin { // Otherwise we just get the count for the values that support BinBoth
			if roundStruct.TotalBinBothMsgCount < deser.NewTotalSigCount {
				roundStruct.TotalBinBothMsgCount = deser.NewTotalSigCount
			}
		}

		if deser.NewMsgIDSigCount == 0 {
			panic(1)
		}

	default:
		if _, err := sms.coinState.GotMsg(sms, deser, gc, mc); err != nil {
			return nil, err
		}
	}
	// sms.mutex.Unlock()
	return ret, nil
}

func (sms *MessageState) updateBothMsgCount(roundStruct *auxRandRoundStruct, mc *consinterface.MemCheckers) {
	var eachCount []int
	roundStruct.TotalBinBothMsgCount, eachCount = sms.Sms.GetTotalSigCount(mc, roundStruct.auxBothHeaders...)
	for i, nxt := range roundStruct.auxBothHeaders {
		if roundStruct.BinNumsBoth[nxt.(*messagetypes.AuxBothMessage).BinVal] < eachCount[i] {
			roundStruct.BinNumsBoth[nxt.(*messagetypes.AuxBothMessage).BinVal] = eachCount[i]
		}
	}
}

// generateProofs generates an auxProofMessage containing signatures supporting binVal and round.
func (sms *MessageState) GenerateProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound,
	binVal types.BinVal, pub sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

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

	// proofs =>
	// For aux message ->
	//    Round 1: t+1 Aux round 0 messages
	//    Round 2: (i) n-t auxProof round 0 messages for 0 (ii) t+1 auxProof round 1 messages for 1
	//    Round r > 2: (i) n-t auxBoth messages for bot from previous round -> This also needs to be a valid coin value
	//                 (ii) n-t auxProof messages for v from previous round
	// For both message ->
	//    To support bin: n-t auxProof messages from same round
	//    To support bot:
	//                Round 1: t+1 auxProof messages for both 0 and 1
	//                Round 1: n-t auxBoth messages for bot from previous round
	switch round {
	case 0:
		if headerID != messages.HdrAuxProof {
			panic(headerID)
		}
		return nil, nil
	case 1:
		if headerID != messages.HdrAuxProof {
			panic(headerID)
		}
		if binVal > 1 {
			panic(binVal)
		}
		// sm, err := sms.GetProofs(messages.HdrAuxProof, sigCount, 0, binVal, pub, mc)
		sm, err := sms.GetProofs(messages.HdrAuxProof, t+1, 0, binVal, pub, mc)
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
	case 2:
		switch headerID {
		case messages.HdrAuxProof:
			if binVal > 1 {
				panic(binVal)
			}
			var count int
			var prfRound types.ConsensusRound
			switch binVal {
			case 0: // we need n-t 0 messages from round 1
				prfRound = 1
				count = nmt
			case 1: // we need t+1 messages from round 0
				prfRound = 0
				count = t + 1
			default:
				panic(binVal)
			}
			// sm, err := sms.GetProofs(messages.HdrAuxProof, sigCount, 0, binVal, pub, mc)
			sm, err := sms.GetProofs(messages.HdrAuxProof, count, prfRound, binVal, pub, mc)
			if err != nil {
				return sm, err
			}
			if sm[0].GetSigCount() < count {
				logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
					count)
				err = types.ErrNotEnoughSigs
				panic(err)
			}
			return sm, err
		case messages.HdrAuxBoth:
			if binVal <= 1 {
				// need n-t round r auxProof messages
				sm, err := sms.GetProofs(messages.HdrAuxProof, sigCount, round, binVal, pub, mc)
				if err != nil {
					return sm, err
				}
				if sm[0].GetSigCount() < nmt {
					logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
						mc.MC.GetFaultCount())
					err = types.ErrNotEnoughSigs
					panic(err)
				}
				return sm, err
			} else {
				// We need n-t round 1 messages for 0
				sm, err := sms.GetProofs(messages.HdrAuxProof, nmt, 1, 0, pub, mc)
				if err != nil {
					return sm, err
				}
				if sm[0].GetSigCount() < nmt {
					logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
						mc.MC.GetFaultCount())
					err = types.ErrNotEnoughSigs
					panic(err)
				}
				// We also need t+1 round 0 messages for t
				sm2, err := sms.GetProofs(messages.HdrAuxProof, t+1, 0, 1, pub, mc)
				if err != nil {
					return sm2, err
				}
				if sm2[0].GetSigCount() <= mc.MC.GetFaultCount() {
					logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm2[0].GetSigCount(),
						mc.MC.GetFaultCount())
					err = types.ErrNotEnoughSigs
					panic(err)
				}
				return append(sm, sm2...), err
			}
		default:
			panic(headerID)
		}
	default: // round > 2
		if (binVal > 1 && headerID == messages.HdrAuxProof) ||
			(binVal > 1 && headerID == messages.HdrAuxBoth) { // n-t messages auxBoth from previous round supporting bot
			sm, err := sms.GetProofs(messages.HdrAuxBoth, sigCount, round-1, binVal, pub, mc)
			if err != nil {
				return sm, err
			}
			if sm[0].GetSigCount() < nmt {
				logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
					mc.MC.GetFaultCount())
				err = types.ErrNotEnoughSigs
				panic(err)
			}
			if headerID == messages.HdrAuxProof {
				// TODO also get support for coin if weak coin here
			}
			return sm, err
		} else if (binVal <= 1 && headerID == messages.HdrAuxProof) || // n-t messages auxProof from PREVIOUS round supporting bin val
			(binVal <= 1 && headerID == messages.HdrAuxBoth) { // n-t messages auxProof from SAME round supporting bin val

			prfRound := round
			if headerID == messages.HdrAuxProof {
				prfRound--
			}
			sm, err := sms.GetProofs(messages.HdrAuxProof, sigCount, prfRound, binVal, pub, mc)
			if err != nil {
				return sm, err
			}
			var sumSigs int
			for _, nxt := range sm {
				sumSigs += nxt.GetSigCount()
			}
			if sumSigs < nmt {
				logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(),
					mc.MC.GetFaultCount())
				err = types.ErrNotEnoughSigs
				panic(err)
			}
			return sm, err
		}
		panic("should not reach")
	}
}
