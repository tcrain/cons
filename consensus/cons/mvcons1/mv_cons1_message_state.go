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
package mvcons1

import (
	"bytes"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/binconsrnd1"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

// MvCons1Message state stores the message of MvCons, including that of the binary consesnsus that it reduces to.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                      // the simple message state is used to track the actual messages
	bincons1.BinConsMessageStateInterface                        // the binary consensus messages
	index                                   types.ConsensusIndex // consensus index
	supportedEchoHash                       types.HashBytes      // the hash of the echo message supported
	mutex                                   sync.RWMutex
}

// NewMvCons1MessageState generates a new MvCons1MessageStateObject.
func NewMvCons1MessageState(gc *generalconfig.GeneralConfig) *MessageState {

	var binMsgState bincons1.BinConsMessageStateInterface
	switch gc.ConsType {
	case types.MvBinCons1Type:
		binMsgState = bincons1.NewBinCons1MessageState(true, gc)
	case types.MvBinConsRnd1Type:
		binMsgState = binconsrnd1.NewBinConsRnd1MessageState(true, gc)
	default:
		panic(gc.ConsType)
	}
	ret := &MessageState{
		BinConsMessageStateInterface: binMsgState,
		SimpleMessageStateWrapper:    messagestate.InitSimpleMessageStateWrapper(gc)}
	// Use the same simple message state
	ret.BinConsMessageStateInterface.SetSimpleMessageStateWrapper(ret.SimpleMessageStateWrapper)
	return ret
}

// New inits and returns a new MvCons1MessageState object for the given consensus index.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	ret := &MessageState{
		BinConsMessageStateInterface: sms.BinConsMessageStateInterface.New(idx).(bincons1.BinConsMessageStateInterface),
		index:                        idx,
		SimpleMessageStateWrapper:    sms.SimpleMessageStateWrapper.NewWrapper(idx)}
	// Use the same simple message state
	ret.BinConsMessageStateInterface.SetSimpleMessageStateWrapper(ret.SimpleMessageStateWrapper)

	return ret
}

func (sms *MessageState) GetBinConsMsgState() bincons1.BinConsMessageStateInterface {
	return sms.BinConsMessageStateInterface
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The valid message types are
// (1) HdrMvInit - the leaders proposal, here is ensures the message comes from the correct leader
// (2) HdrMvEcho - echo messages
// (3) HdrMvRequestRecover - unsigned message asking for the init message received for a given hash
// (4) -- Any messages to the binary consensus
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}
	switch deser.HeaderType {
	case messages.HdrMvInit, messages.HdrPartialMsg: // Init or partial messages must only come from the coordinator

		// only have round 0 messages for mvcons1
		round := cons.GetMvMsgRound(deser)
		if round != 0 {
			logging.Info("Received a non-zero round mv init message, round:", round)
			return nil, types.ErrInvalidRound
		}

		w := deser.Header.(*sig.MultipleSignedMessage)
		// An MvInit message should only be signed once
		if len(w.SigItems) != 1 {
			logging.Errorf("Got proposal with multiple signatures at index %v", sms.index)
			return nil, types.ErrInvalidSig
		}

		// Check the membership/signatures/duplicates
		ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
		if err != nil {
			return nil, err
		}

		// Check it is from the valid round coord
		err = consinterface.CheckMemberCoord(mc, 0, w.SigItems[0], w)
		if err != nil {
			return nil, err
		}

		// If the message resulted in a combination then we will have the combined message and the partial message here
		for _, nxt := range ret {
			round := cons.GetMvMsgRound(nxt)
			if round != 0 {
				logging.Info("Received a non-zero round mv init message, round:", round)
				return nil, types.ErrInvalidRound
			}
			if nxt.HeaderType == messages.HdrMvInit {
				// nothing to do
			}
		}
		return ret, nil
	case messages.HdrMvEcho:
		round := deser.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvEchoMessage).Round
		if round != 0 {
			logging.Info("Received a non-zero round mv echo message, round:", round)
			return nil, types.ErrInvalidRound
		}

		// Check the membership/signatures/duplicates
		ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
		return ret, err
	case messages.HdrMvRequestRecover:
		// we handle this in cons
		// Nothing to do since it is just asking for a message
		return []*channelinterface.DeserializedItem{deser}, nil
	default:
		// check if it is a bin cons message
		return sms.BinConsMessageStateInterface.GotMsg(hdrFunc, deser, gc, mc)
		// panic(fmt.Sprint("invalid msg type ", deser.HeaderType))
	}
}

// Generate proofs returns a signed message with signatures supporting the input values,
// it can be a MvEchoMessage if round == 2 && binVal == 1 otherwise it is a AuxProof message
func (sms *MessageState) GenerateProofs(hdrID messages.HeaderID, sigCount int, round types.ConsensusRound, binVal types.BinVal,
	pub sig.Pub, mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if round == 2 && binVal == 1 { // in this case we support using the echo hash
		return sms.generateMvProofs(sigCount, pub, mc)
	}
	return sms.BinConsMessageStateInterface.GenerateProofs(hdrID, sigCount, round, binVal, pub, mc)
}

func (sms *MessageState) setEchoHash(proposalHash types.HashBytes) {
	if sms.supportedEchoHash != nil && !bytes.Equal(sms.supportedEchoHash, proposalHash) { // sanity check
		panic("tried to support different echo hashes")
	}
	sms.supportedEchoHash = proposalHash
}

// generateMvProofs generates an MvEcho message containing signature received supporting the proposalHash
func (sms *MessageState) generateMvProofs(sigCount int, _ sig.Pub,
	mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error) {

	if sms.supportedEchoHash == nil {
		panic("tried to get proofs for an echo message before sending the hash")
	}
	w := messagetypes.NewMvEchoMessage()
	w.ProposalHash = sms.supportedEchoHash
	// Add sigs
	msm, err := sms.Sms.SetupSignedMessage(w, false, sigCount, mc)
	if err != nil {
		logging.Error(err)
		return []*sig.MultipleSignedMessage{msm}, err
	}
	nmt := mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
	if msm.GetSigCount() < nmt {
		logging.Error("Not enough signatures for MV proofs: got, expected", msm.GetSigCount(), nmt)
		err = types.ErrNotEnoughSigs
	}
	return []*sig.MultipleSignedMessage{msm}, err
}
