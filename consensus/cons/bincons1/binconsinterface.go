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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/consinterface/messagestate"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

type getBinConsMsgStateInterface interface {
	GetBinConsMsgState() BinConsMessageStateInterface
}

// BinConsMessageStateInterface extends the messagestate.MessageState interface with some addition operations for the storage of messages specific to BinCons1.
// Note that operations can be called from multiple threads.
// The public methods are concurrency safe, the others should be synchronized as needed.
type BinConsMessageStateInterface interface {
	Lock()
	Unlock()

	// GetBinMsgState returns the base bin message state.
	GetBinMsgState() BinConsMessageStateInterface

	// GenerateProofs generates an auxProofMessage containing signatures supporting binVal and round.
	GenerateProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound, binVal types.BinVal, pub sig.Pub,
		mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error)
	// GetProofs generates an AuxProofMessage containing the signatures received for binVal in round round.
	GetProofs(headerID messages.HeaderID, sigCount int, round types.ConsensusRound, binVal types.BinVal, pub sig.Pub,
		mc *consinterface.MemCheckers) ([]*sig.MultipleSignedMessage, error)

	// See consinterface.MessageState.New
	New(idx types.ConsensusIndex) consinterface.MessageState
	// See as consinterface.MessageState.GotMsg
	GotMsg(hdrFunc consinterface.HeaderFunc,
		deser *deserialized.DeserializedItem, gc *generalconfig.GeneralConfig,
		mc *consinterface.MemCheckers) ([]*deserialized.DeserializedItem, error)

	// SetMv0Valid is called by the multivalue reduction MvCons1, when 0 becomes valid for round 1
	SetMv0Valid()
	// SetMv1Valid is called by the multivalue reduction MvCons1, when 1 becomes valid for round 1
	SetMv1Valid(mc *consinterface.MemCheckers)

	// sentProposal returns true if a proposal has been sent for round+1 (i.e. the following round)
	// if shouldSet is true, then sets to having sent the proposal for that round to true
	SentProposal(round types.ConsensusRound, shouldSet bool, mc *consinterface.MemCheckers) bool
	// SetSimpleMessageStateWrapper sets the simple message state object, this is used by the multivale reduction MvCons1 since they share the
	// same simple message state.
	SetSimpleMessageStateWrapper(sm *messagestate.SimpleMessageStateWrapper)
	// GetValidMessage count returns the number of signed AuxProofMessages received from different processes in round round.
	GetValidMessageCount(round types.ConsensusRound, mc *consinterface.MemCheckers) int
}

func GetBinConsCommitProof(headerID messages.HeaderID, sc cons.BinConsInterface,
	binMsgState BinConsMessageStateInterface, shouldLock bool,
	mc *consinterface.MemCheckers) []messages.MsgHeader {

	if !sc.HasDecided() {
		panic("should only get proof after decision")
	}
	dec, rnd := sc.GetBinDecided()

	var msg messages.MsgIDHeader
	switch headerID {
	case messages.HdrAuxProof:
		auxMsg := messagetypes.NewAuxProofMessage(false)
		auxMsg.BinVal = types.BinVal(dec)
		auxMsg.Round = rnd
		msg = auxMsg
	case messages.HdrAuxBoth:
		auxMsg := messagetypes.NewAuxBothMessage()
		auxMsg.BinVal = types.BinVal(dec)
		auxMsg.Round = rnd
		msg = auxMsg
	case messages.HdrAuxStage1:
		auxMsg := messagetypes.NewAuxStage1Message()
		auxMsg.BinVal = types.BinVal(dec)
		auxMsg.Round = rnd
		msg = auxMsg
	case messages.HdrAuxStage0:
		auxMsg := messagetypes.NewAuxStage0Message()
		auxMsg.BinVal = types.BinVal(dec)
		auxMsg.Round = rnd
		msg = auxMsg
	default:
		panic("invalid msg type")
	}

	var err error
	sigCount, _, _, err := sc.GetBufferCount(msg, sc.GetGeneralConfig(), mc)
	if err != nil {
		panic(err)
	}

	if shouldLock {
		binMsgState.Lock()
		defer binMsgState.Unlock()
	}

	prfMsg, err := binMsgState.GetProofs(headerID, sigCount, rnd, types.BinVal(dec),
		mc.MC.GetMyPriv().GetPub(), mc)
	if err != nil {
		panic(err)
	}
	ret := make([]messages.MsgHeader, len(prfMsg))
	for i, nxt := range prfMsg {
		ret[i] = nxt
	}
	return ret
}
