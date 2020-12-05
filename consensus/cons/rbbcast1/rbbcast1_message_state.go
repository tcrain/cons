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

package rbbcast1

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"sync"
)

// RbBcast1Message state stores the message of RbBcast.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                      // the simple message state is used to track the actual messages
	index                                   types.ConsensusIndex // consensus index
	mutex                                   sync.RWMutex
	supportedEchoHash                       types.HashBytes // the hash of the echo message supported
	totalEchoMsgCount                       int             // max number of a echo of messages we have received for this round
}

// NewRbBcast1MessageState generates a new RbBcast1MessageStateObject.
func NewRbBcast1MessageState(gc *generalconfig.GeneralConfig) *MessageState {
	ret := &MessageState{
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
	return ret
}

// New inits and returns a new RbBcast1MessageState object for the given consensus index.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	ret := &MessageState{
		index:                     idx,
		SimpleMessageStateWrapper: sms.SimpleMessageStateWrapper.NewWrapper(idx)}
	return ret
}

// GetValidMessage count returns the number of signed echo and commit messages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount() (echoMsgCount int) {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	return sms.totalEchoMsgCount
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
// (1) HdrMvInit - the leaders proposal, here is ensures the message comes from the correct leader
// (2) HdrMvEcho - echo messages
// (4) HdrMvRequestRecover - unsigned message asking for the init message received for a given hash
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *deserialized.DeserializedItem, gc *generalconfig.GeneralConfig, mc *consinterface.MemCheckers) ([]*deserialized.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}

	switch deser.HeaderType {
	case messages.HdrMvInit, messages.HdrPartialMsg:

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

			w := nxt.Header.(*sig.MultipleSignedMessage)
			if len(w.SigItems) != 1 { // this should be checked above
				panic("should not have multiple sigs")
			}

			if cons.GetMvMsgRound(deser) != 0 {
				return nil, types.ErrInvalidRound
			}

			// Check it is from the valid round coord
			err = consinterface.CheckMemberCoord(mc, 0, w.SigItems[0], w)
			if err != nil {
				return nil, err
			}

			if nxt.HeaderType == messages.HdrMvInit {
				// handled in mv_cons2
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
		if cons.GetMvMsgRound(hdr[0]) != 0 {
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
		return []*deserialized.DeserializedItem{deser}, nil
	default:
		panic(fmt.Sprint("invalid msg type ", deser.HeaderType))
	}
}
