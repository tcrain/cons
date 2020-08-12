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

package bincons1

import (
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

// auxRoundStruct keeps stats about auxProof messages received for a single round
type auxRoundStruct struct {
	BinNums          [2]int            // the number of signatures received for 0 an 1
	TotalBinMsgCount int               // the total number of signatures receieved for both 0 and 1 from unique processes (i.e. this can be less than BinNums[0]+BinNums[1])
	sentProposal     bool              // if a proposal has been sent for the NEXT round
	timeoutState     cons.TimeoutState // the timeout state for the round
}

// BinCons1MessageState implements BinConsMessageStateInterface.
// It stores the messages of BinCons1.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                                          // the simple message state is used to track the actual messages
	auxValues                               map[types.ConsensusRound]*auxRoundStruct // map from round index to auxRoundStruct
	index                                   types.ConsensusIndex                     // consensus index
	isMv                                    bool                                     // true if this is being used as part of mv cons
	mv0Valid                                bool                                     // for use with multivalued reduction MvCons1, set to true if 0 is valid for round 1
	mv1Valid                                bool                                     // for use with multivalued reduction MvCons1, set to true if 1 is valid for round 1
	mutex                                   sync.RWMutex
}

// SetSimpleMessageStateWrapper sets the simple message state object, this is used by the multivale reduction MvCons1 since they share the
// same simple message state.
func (sms *MessageState) SetSimpleMessageStateWrapper(sm *messagestate.SimpleMessageStateWrapper) {
	sms.SimpleMessageStateWrapper = sm
}

// GetBinMsgState returns the base bin message state.
func (sms *MessageState) GetBinMsgState() BinConsMessageStateInterface {
	return sms
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
func (sms *MessageState) SetMv1Valid(_ *consinterface.MemCheckers) {
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

// NewBinCons1MessageState generates a new BinCons1MessageState object.
func NewBinCons1MessageState(isMv bool,
	gc *generalconfig.GeneralConfig) *MessageState {

	return &MessageState{
		isMv:                      isMv,
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
}

// New creates a new empty BinCons1MessageState object for the consensus index idx.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	return &MessageState{
		auxValues:                 make(map[types.ConsensusRound]*auxRoundStruct),
		index:                     idx,
		isMv:                      sms.isMv,
		SimpleMessageStateWrapper: sms.SimpleMessageStateWrapper.NewWrapper(idx)}
}

// sentProposal returns true if a proposal has been sent for round+1 (i.e. the following round)
// if shouldSet is true, then sets to having sent the proposal for that round to true
func (sms *MessageState) SentProposal(round types.ConsensusRound, shouldSet bool, _ *consinterface.MemCheckers) bool {
	var ret bool
	sms.mutex.Lock()
	ars := sms.getAuxRoundStruct(round)
	ret = ars.sentProposal
	if shouldSet {
		ars.sentProposal = shouldSet
	}
	sms.mutex.Unlock()
	return ret
}

// GetProofs generates an AuxProofMessage continaining the signatures received for binVal in round round.
func (sms *MessageState) GetProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound,
	binVal types.BinVal, _ sig.Pub,
	mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if headerID != messages.HdrAuxProof {
		panic("invalid header id")
	}

	auxRoundStruct := sms.getAuxRoundStruct(round)
	if auxRoundStruct.BinNums[binVal] == 0 {
		return nil, types.ErrNoProofs
	}
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
func (sms *MessageState) GetValidMessageCount(round types.ConsensusRound, _ *consinterface.MemCheckers) int {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.getAuxRoundStruct(round)
	return roundStruct.TotalBinMsgCount
}

// getAuxRoundStruct returns the auxRoundStruct for round round (making a new one if necessary)
func (sms *MessageState) getAuxRoundStruct(round types.ConsensusRound) *auxRoundStruct {
	item := sms.auxValues[round]
	if item == nil {
		item = &auxRoundStruct{}
		sms.auxValues[round] = item
	}
	return item
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The message must be an AuxProofMessage, since that is the only valid message type for BinCons1.
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}
	w := deser.Header.(*sig.MultipleSignedMessage).GetBaseMsgHeader().(*messagetypes.AuxProofMessage) // only AuxProofMessages will be received

	// Check the membership/signatures/duplicates
	ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
	if err != nil {
		return nil, err
	}

	// Update the round struct with the new pubs
	// we don't mind if someone already updated these in the mean time
	sms.mutex.Lock()
	roundStruct := sms.getAuxRoundStruct(w.Round)

	// need to update the nubmer of sigs for that bin value, have to check since
	// someone with a large value could have updated it concurrently
	// because we call GotMsg from different threads
	if roundStruct.BinNums[w.BinVal] < deser.NewTotalSigCount {
		roundStruct.BinNums[w.BinVal] = deser.NewTotalSigCount
	}
	if roundStruct.TotalBinMsgCount < deser.NewMsgIDSigCount {
		roundStruct.TotalBinMsgCount = deser.NewMsgIDSigCount
	}

	sms.mutex.Unlock()
	return ret, nil
}

// Compute the valid bin vals for round r.
func (sms *MessageState) getValids(nmt int, t int, round types.ConsensusRound) [2]bool {
	mod := byte((round - 1) % 2)
	ret := [2]bool{}
	switch round {
	case 0:
		// in MV round 0 and 1 are handled by the multivalue code
		if !sms.isMv { // otherwise both 0 and 1 are valid for round 0
			ret[0] = true
			ret[1] = true
		}
	case 1:
		roundStructM1 := sms.getAuxRoundStruct(0)
		if !sms.isMv {
			// a value is valid as long as we received more than t messages supporting it in round 0
			ret[0] = roundStructM1.BinNums[0] > t
			ret[1] = roundStructM1.BinNums[1] > t
		} else {
			// special case when used in multivalue (set by SetMv1Valid() and SetMv0Valid())
			ret[0] = sms.mv0Valid
			ret[1] = sms.mv1Valid
		}
	case 2:
		// in round 2 we have a special case for 1
		if !sms.isMv {
			// we could have only decided 1 in round 1, so to support 1 in round 2 we need more than t signatures from round 0
			ret[1] = sms.getAuxRoundStruct(round - 2).BinNums[mod] > t
		} else {
			// special case when used in multivalue (set by SetMv1Valid())
			ret[1] = sms.mv1Valid
		}
		fallthrough // otherwise we use the default case
	default:
		if sms.getAuxRoundStruct(round - 2).BinNums[mod] >= nmt {
			// we could have decided notMod in round-2, so we need to prove this didn't happen if we want to support mod
			ret[mod] = true
		}
		if sms.getAuxRoundStruct(round - 1).BinNums[1-mod] >= nmt {
			// we could have decided mod in round-1, so we need to prove this didn't happen if we want to support notMod
			ret[1-mod] = true
		}
	}
	return ret
}

// generateProofs generates an auxProofMessage containing signatures supporting binVal and round.
func (sms *MessageState) GenerateProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound,
	binVal types.BinVal, pub sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if headerID != messages.HdrAuxProof {
		panic("invalid header type")
	}
	t := mc.MC.GetFaultCount()

	switch round {
	case 0:
		// no proofs needed for round 0
		return nil, types.ErrNoProofs
	case 1:
		// take the signatures from round 0 supporting binVal
		sm, err := sms.GetProofs(headerID, t+1, 0, binVal, pub, mc)
		if err != nil {
			return sm, err
		}
		if sm[0].GetSigCount() < mc.MC.GetFaultCount() {
			logging.Error("Not enough signatures for bin proofs: round, count, min:", round,
				sm[0].GetSigCount(), mc.MC.GetFaultCount())
			err = types.ErrNotEnoughSigs
		}
		return sm, err
	default:
		var sm []*sig.MultipleSignedMessage
		var err error
		mod := types.BinVal((round - 1) % 2) // mod is the only possible value decided in round - 1
		// we could not have decided notMod/binVal in the previous round, so we get proofs for mod from round-2 to show that notMod was not decided in that round
		if round == 2 && binVal == 1 { // special case for round 2
			sigCount = t + 1
		}
		if mod == binVal {
			sm, err = sms.GetProofs(headerID, sigCount, round-2, binVal, pub, mc)
		} else {
			// otherwise we could have decided notBinVal in the previous round, so we get proofs to show that this could have not happened
			sm, err = sms.GetProofs(headerID, sigCount, round-1, binVal, pub, mc)
		}
		if err != nil {
			return sm, err
		}
		var minCount int
		if round == 2 && binVal == 1 { // special case for round 2
			minCount = mc.MC.GetFaultCount()
		} else {
			// we should have n - t proofs
			minCount = mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
		}
		if sm[0].GetSigCount() < minCount {
			logging.Error("Not enough signatures for bin proofs: round, count, min:", round, sm[0].GetSigCount(), minCount)
			err = types.ErrNotEnoughSigs
		}
		return sm, err
	}
}
