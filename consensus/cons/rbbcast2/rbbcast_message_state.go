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

package rbbcast2

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
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
	supportedCommitHash                     types.HashBytes // hash of the supported commit message
	supportedCommitCount                    int             // number of messages we have received for the supported commit
}

// NewRbBcast2MessageState generates a new RbBcast2MessageStateObject.
func NewRbBcast2MessageState(gc *generalconfig.GeneralConfig) *MessageState {
	ret := &MessageState{
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
	return ret
}

// New inits and returns a new RbBcast2MessageState object for the given consensus index.
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

// getSupportedCommitHash returns the commit hash supported for the round, or nil if there is none, and the number of
// messages received supporting it.
func (sms *MessageState) getSupportedCommitHash() (hash types.HashBytes, msgCount int) {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	return sms.supportedCommitHash, sms.supportedCommitCount
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// The valid message types are
// (1) HdrMvInit - the leaders proposal, here is ensures the message comes from the correct leader
// (2) HdrMvEcho - echo messages
// (3) HdrMvCommit - commit messages
// (4) HdrMvRequestRecover - unsigned message asking for the init message received for a given hash
func (sms *MessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *deserialized.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*deserialized.DeserializedItem, error) {

	if !deser.IsDeserialized {
		// Only track deserialzed messages
		panic("should be deserialized")
	}

	switch deser.HeaderType {
	case messages.HdrMvInit, messages.HdrPartialMsg:

		// An MvInit message should only be signed once
		if err := sig.CheckSingleSupporter(deser.Header); err != nil {
			return nil, err
		}
		// Check the membership/signatures/duplicates
		ret, err := sms.Sms.GotMsg(hdrFunc, deser, gc, mc)
		if err != nil {
			return nil, err
		}

		// If the message resulted in a combination then we will have the combined message and the partial message here
		for _, nxt := range ret {

			if err := consinterface.CheckMemberCoordHdr(mc, 0, deser.Header); err != nil {
				return nil, err
			}

			if nxt.HeaderType == messages.HdrMvInit {
				// handled in mv_cons2
			}
		}
		return ret, nil
	case messages.HdrMvEcho, messages.HdrMvCommit:
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

		t := mc.MC.GetFaultCount()
		nmt := mc.MC.GetMemberCount() - t
		if deser.HeaderType == messages.HdrMvEcho {
			// update the message count
			if sms.totalEchoMsgCount < deser.NewMsgIDSigCount {
				sms.totalEchoMsgCount = deser.NewMsgIDSigCount
			}
			// check if we can support an echo hash (i.e. we got n-t of the same hash)
			if deser.NewTotalSigCount >= nmt {
				hashBytes := hdr[0].Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.MvEchoMessage).ProposalHash
				if sms.supportedEchoHash != nil && !bytes.Equal(sms.supportedEchoHash, hashBytes) {
					panic("got two hashes to support")
				}
				sms.supportedEchoHash = hashBytes
			}
		} else {
			// Once we have more than t commit messages we can support the value
			if deser.NewTotalSigCount > t {
				hashBytes := hdr[0].Header.(messages.InternalSignedMsgHeader).GetBaseMsgHeader().(*messagetypes.MvCommitMessage).ProposalHash

				if sms.supportedCommitHash != nil && !bytes.Equal(sms.supportedCommitHash, hashBytes) {
					panic("got two hashes to support")
				}
				if sms.supportedEchoHash != nil && !bytes.Equal(sms.supportedEchoHash, hashBytes) {
					panic("got conflicting echo and commit hashes")
				}
				sms.supportedCommitHash = hashBytes
				if sms.supportedCommitCount < deser.NewTotalSigCount {
					sms.supportedCommitCount = deser.NewTotalSigCount
				}
			}
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

// Generate proofs returns a signed message with signatures supporting the input values,
// it can be a MvEchoMessage.
func (sms *MessageState) GenerateProofs(sigCount int, _ types.ConsensusRound, _ types.BinVal,
	_ sig.Pub, mc *consinterface.MemCheckers) (messages.MsgHeader, error) {

	// if we have n-t echo messages supporting a value, then we use that
	var w messages.InternalSignedMsgHeader
	var count int
	var err error
	// check echos
	sms.mutex.Lock()
	supportedEchoHash := sms.supportedEchoHash
	sms.mutex.Unlock()
	if len(supportedEchoHash) == 0 {
		return nil, types.ErrNoEchoHash
	} else {
		echoMsg := messagetypes.NewMvEchoMessage()
		echoMsg.ProposalHash = supportedEchoHash
		count, err = sms.GetSigCountMsgHeader(echoMsg, mc)
		if count >= sigCount {
			w = echoMsg
		} else {
			return nil, types.ErrNoProofs
		}
	}

	// Add sigs
	prfMsgSig, err := sms.SetupSignedMessage(w, false, sigCount, mc)
	if err != nil {
		logging.Error(err)
		return nil, err
	}
	if prfMsgSig.GetSigCount() < sigCount {
		logging.Error("Not enough signatures for MV proofs: got, expected", prfMsgSig.GetSigCount(), sigCount, w.GetID())
		err = types.ErrNotEnoughSigs
	}
	return prfMsgSig, err
}
