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

package mvcons3

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
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

// MvCons3Message state stores the message of MvCons
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                      // the simple message state is used to track the actual messages
	index                                   types.ConsensusIndex // consensus index
	priv                                    sig.Priv             // local node private key
	supportedEchoHash                       types.HashBytes      // the hash of the echo message supported
	totalEchoMsgCount                       int                  // max number of a echo of messages we have received for this round
	prev                                    *MessageState        // message state for the previous index
	mutex                                   sync.RWMutex
}

// NewMvCons3MessageState generates a new MvCons3MessageStateObject.
func NewMvCons3MessageState(gc *generalconfig.GeneralConfig) *MessageState {
	ret := &MessageState{
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
	return ret
}

// New inits and returns a new MvCons3MessageState object for the given consensus index.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	ret := &MessageState{
		index:                     idx,
		SimpleMessageStateWrapper: sms.SimpleMessageStateWrapper.NewWrapper(idx)}
	return ret
}

// GetValidMessage count returns the number of signed echo and commit messages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount() (echoMsgCount int) {
	sms.mutex.RLock()
	defer sms.mutex.RUnlock()

	return sms.totalEchoMsgCount
}

// getMostRecentSupport returns the index and hash of the largest index with a valid supported (n-t echos) init message
func (sms *MessageState) getMostRecentSupport() (index types.ConsensusInt, hash types.HashBytes) {
	sms.mutex.Lock()
	supportedEchoHash := sms.supportedEchoHash
	prev := sms.prev
	sms.mutex.Unlock()

	if supportedEchoHash != nil {
		return sms.index.Index.(types.ConsensusInt), supportedEchoHash
	}
	return prev.getMostRecentSupport()
}

// getSupportedEchoHash returns the echo hash supported for the round, or nil if there is none
func (sms *MessageState) getSupportedEchoHash() types.HashBytes {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	return sms.supportedEchoHash
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The valid message types are
// (1) HdrMvInitSupport - the leaders proposal, here is ensures the message comes from the correct leader
// (2) HdrMvEcho - echo messages
// (4) HdrMvRequestRecover - unsigned message asking for the init message received for a given hash
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}

	switch deser.HeaderType {
	case messages.HdrMvInitSupport, messages.HdrPartialMsg:

		// An MvInit message should only be signed once
		w := deser.Header.(*sig.MultipleSignedMessage)
		if len(w.SigItems) != 1 {
			return nil, types.ErrInvalidSig
		}

		// Check the membership/signatures/duplicates
		ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
		if err != nil {
			return nil, err
		}

		// If the message resulted in a combination then we will have the combined message and the partial message here
		for _, nxt := range ret {
			round := types.ConsensusRound(sms.index.Index.(types.ConsensusInt) - 1) // TODO how to do index instead of round properly?

			// Check it is from the valid round coord
			w := nxt.Header.(*sig.MultipleSignedMessage)
			if len(w.SigItems) != 1 { // sanity check
				panic("should not have multiple sigs")
			}

			// Check it is from the valid round coord
			err = consinterface.CheckMemberCoord(mc, round, w.SigItems[0], w)
			if err != nil {
				return nil, err
			}

			if nxt.HeaderType == messages.HdrMvInitSupport {
				// taken care of in MvCons3
			}
		}
		return ret, nil
	case messages.HdrMvEcho:
		// Check the membership/signatures/duplicates
		hdr, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
		if err != nil {
			return hdr, err
		}
		if len(hdr) != 1 {
			panic("should not create a new msg")
		}
		round := cons.GetMvMsgRound(hdr[0])
		if round != 0 {
			logging.Info("Received a non-zero round mv init message, round:", round)
			return nil, types.ErrInvalidRound
		}
		sms.mutex.Lock()

		// update the message count
		nmt := mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
		if sms.totalEchoMsgCount < deser.NewMsgIDSigCount {
			sms.totalEchoMsgCount = deser.NewMsgIDSigCount
		}
		// check if we can support an echo hash (i.e. we got n-t of the same hash)
		if deser.NewTotalSigCount >= nmt {
			hashBytes := hdr[0].Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvEchoMessage).ProposalHash
			if sms.supportedEchoHash != nil && !bytes.Equal(sms.supportedEchoHash, hashBytes) {
				panic("got two hashes to support")
			}
			sms.supportedEchoHash = hashBytes
		}
		sms.mutex.Unlock()
		return hdr, err
	case messages.HdrMvRequestRecover:
		// we handle this in cons
		// Nothing to do since it is just asking for a message
		return []*channelinterface.DeserializedItem{deser}, nil
	default:
		panic(fmt.Sprint("invalid msg type ", deser.HeaderType))
	}
}

// Generate proofs returns a signed message with signatures supporting the input values,
// it can be a MvEchoMessage.
func (sms *MessageState) GenerateProofs(sigCount int, _ types.ConsensusRound, _ types.BinVal,
	_ sig.Pub, mc *consinterface.MemCheckers) (messages.MsgHeader, error) {

	return sms.checkMvProofs(sigCount, mc, true)
}

// generateMvProofs generates an MvEcho message containing signature received supporting the proposalHash
func (sms *MessageState) checkMvProofs(sigCount int,
	mc *consinterface.MemCheckers, generateProofs bool) (prfMsg messages.MsgHeader, err error) {

	var w messages.InternalSignedMsgHeader
	var count int

	// check echos
	sms.mutex.Lock()
	supportedEchoHash := sms.supportedEchoHash
	sms.mutex.Unlock()
	if len(supportedEchoHash) == 0 || bytes.Equal(types.GetZeroBytesHashLength(), supportedEchoHash) {
		err = types.ErrNoEchoHash
	} else {
		echoMsg := messagetypes.NewMvEchoMessage()
		echoMsg.ProposalHash = supportedEchoHash
		count, err = sms.GetSigCountMsgHeader(echoMsg, mc)
		if count >= sigCount {
			w = echoMsg
		} else {
			err = types.ErrNoProofs
		}
	}

	if !generateProofs || err != nil { // return if we have an error or we are not supposed to generate proofs
		return
	}

	// Add sigs
	prfMsgSig, err := sms.SetupSignedMessage(w, false, sigCount, mc)
	if err != nil {
		logging.Error(err)
		return
	}
	if prfMsgSig.GetSigCount() < sigCount {
		logging.Error("Not enough signatures for MV proofs: got, expected", prfMsgSig.GetSigCount(), sigCount)
		err = types.ErrNotEnoughSigs
	}
	prfMsg = prfMsgSig
	return
}
