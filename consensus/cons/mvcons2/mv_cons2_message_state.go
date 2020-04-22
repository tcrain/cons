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
package mvcons2

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

type mvRoundStruct struct {
	supportedEchoHash   types.HashBytes // the hash of the echo message supported
	supportedCommitHash types.HashBytes // the hash of the commit message supported
	totalEchoMsgCount   int             // max number of a echo of messages we have received for this round
	totalCommitMsgCount int             // max number of a commit of messages we have received for this round
}

// MvCons2Message state stores the message of MvCons, including that of the binary consesnsus that it reduces to.
type MessageState struct {
	*messagestate.SimpleMessageStateWrapper                                        // the simple message state is used to track the actual messages
	index                                   types.ConsensusIndex                   // consensus index
	mvRoundStructs                          map[types.ConsensusRound]mvRoundStruct // information per round
	mutex                                   sync.RWMutex
}

// NewMvCons2MessageState generates a new MvCons2MessageStateObject.
func NewMvCons2MessageState(gc *generalconfig.GeneralConfig) *MessageState {
	ret := &MessageState{
		SimpleMessageStateWrapper: messagestate.InitSimpleMessageStateWrapper(gc)}
	return ret
}

// New inits and returns a new MvCons2MessageState object for the given consensus index.
func (sms *MessageState) New(idx types.ConsensusIndex) consinterface.MessageState {

	ret := &MessageState{
		index:                     idx,
		mvRoundStructs:            make(map[types.ConsensusRound]mvRoundStruct),
		SimpleMessageStateWrapper: sms.SimpleMessageStateWrapper.NewWrapper(idx)}
	return ret
}

// GetValidMessage count returns the number of signed echo and commit messages received from different processes in round round.
func (sms *MessageState) GetValidMessageCount(round types.ConsensusRound) (echoMsgCount, commitMsgCount int) {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()
	roundStruct := sms.mvRoundStructs[round]
	return roundStruct.totalEchoMsgCount, roundStruct.totalCommitMsgCount
}

// getSupportedEchoHash returns the echo hash supported for the round, or nil if there is none
func (sms *MessageState) getSupportedEchoHash(round types.ConsensusRound) types.HashBytes {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	return sms.mvRoundStructs[round].supportedEchoHash
}

// getSupportedCommitHash returns the echo hash supported for the round, or nil if there is none
func (sms *MessageState) getSupportedCommitHash(round types.ConsensusRound) types.HashBytes {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	return sms.mvRoundStructs[round].supportedCommitHash
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
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

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

			round := cons.GetMvMsgRound(nxt)

			w := nxt.Header.(*sig.MultipleSignedMessage)
			if len(w.SigItems) != 1 { // this should be checked above
				panic("should not have multiple sigs")
			}

			// Check it is from the valid round coord
			err = consinterface.CheckMemberCoord(mc, round, w.SigItems[0], w)
			if err != nil {
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
		round := cons.GetMvMsgRound(hdr[0])
		sms.mutex.Lock()
		roundStruct := sms.mvRoundStructs[round]

		// update the message count
		nmt := mc.MC.GetMemberCount() - mc.MC.GetFaultCount()
		if deser.HeaderType == messages.HdrMvEcho {
			if roundStruct.totalEchoMsgCount < deser.NewMsgIDSigCount {
				roundStruct.totalEchoMsgCount = deser.NewMsgIDSigCount
			}
			// check if we can support an echo hash (i.e. we got n-t of the same hash)
			if deser.NewTotalSigCount >= nmt {
				hashBytes := hdr[0].Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvEchoMessage).ProposalHash
				if roundStruct.supportedEchoHash != nil && !bytes.Equal(roundStruct.supportedEchoHash, hashBytes) {
					panic("got two hashes to support")
				}
				roundStruct.supportedEchoHash = hashBytes
			}

		} else if deser.HeaderType == messages.HdrMvCommit {
			if roundStruct.totalCommitMsgCount < deser.NewMsgIDSigCount {
				roundStruct.totalCommitMsgCount = deser.NewMsgIDSigCount
			}
			// check if we can support a commit hash (i.e. we got n-t of the same hash)
			if deser.NewTotalSigCount >= nmt {
				hashBytes := hdr[0].Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvCommitMessage).ProposalHash
				if roundStruct.supportedCommitHash != nil && !bytes.Equal(roundStruct.supportedCommitHash, hashBytes) {
					panic("got two hashes to support")
				}
				roundStruct.supportedCommitHash = hashBytes
			}
		}
		sms.mvRoundStructs[round] = roundStruct
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
func (sms *MessageState) GenerateProofs(sigCount int, round types.ConsensusRound, _ types.BinVal,
	pub sig.Pub, mc *consinterface.MemCheckers) (messages.MsgHeader, error) {

	return sms.checkMvProofs(round, sigCount, pub, mc, true, false)
}

// generateMvProofs generates an MvEcho message containing signature received supporting the proposalHash
func (sms *MessageState) checkMvProofs(round types.ConsensusRound, sigCount int, _ sig.Pub,
	mc *consinterface.MemCheckers, generateProofs, onlyCheckCommit bool) (prfMsg messages.MsgHeader, err error) {

	// if we have n-t echo messages supporting a value, then we use that
	// if we have n-t commit messages supporting 0, then we use that
	// we first try to get the echos because that is what the leader will send
	// if we don't have either then we should not have advanced to this round, because we either should have committed
	// or wait for more messages
	// next round echos can be:
	// (1) any value if n-t previous commit 0
	// (2) previous value if n-t echo supporting a previous value from round - 1

	var w messages.InternalSignedMsgHeader
	var count int

	if !onlyCheckCommit {
		// check echos
		sms.mutex.Lock()
		supportedEchoHash := sms.mvRoundStructs[round].supportedEchoHash
		sms.mutex.Unlock()
		if supportedEchoHash == nil || bytes.Equal(types.GetZeroBytesHashLength(), supportedEchoHash) {
			err = types.ErrNoEchoHash
		} else {
			echoMsg := messagetypes.NewMvEchoMessage()
			echoMsg.Round = round
			echoMsg.ProposalHash = supportedEchoHash
			count, err = sms.GetSigCountMsgHeader(echoMsg, mc)
			if count >= sigCount {
				w = echoMsg
			} else {
				err = types.ErrNoProofs
			}
		}
	}

	// check zero commits in case no echos
	if err != nil || onlyCheckCommit {
		commitMsg := messagetypes.NewMvCommitMessage()
		commitMsg.Round = round
		commitMsg.ProposalHash = types.GetZeroBytesHashLength()
		count, err = sms.GetSigCountMsgHeader(commitMsg, mc)
		if count >= sigCount {
			w = commitMsg
			err = nil
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
		var wid messages.HeaderID
		if w != nil {
			wid = w.GetID()
		}
		logging.Error("Not enough signatures for MV proofs: got, expected",
			prfMsgSig.GetSigCount(), sigCount, wid)
		err = types.ErrNotEnoughSigs
	}
	prfMsg = prfMsgSig
	return
}
