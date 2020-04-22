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
package messagestate

import (
	"bytes"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

type partialMsgState struct {
	sync.Mutex
	sm          *sig.MultipleSignedMessage     // the signed message containing the partials (so we have the signature)
	partials    []*messagetypes.PartialMessage // the slice of partials
	hasCombined bool                           // is set to true when the message is successfuly reconstructed
}

func newPartialMsgState(sm *sig.MultipleSignedMessage) *partialMsgState {
	partial := sm.GetBaseMsgHeader().(*messagetypes.PartialMessage)
	partials := make([]*messagetypes.PartialMessage, len(partial.PartialMsgHashes))

	return &partialMsgState{
		sm:       sm,
		partials: partials}
}

type partialMsgMap struct {
	sync.Mutex
	partialMap map[sig.PubKeyID]map[types.HashStr]*partialMsgState
}

// getAllPartials returns all the partials that have not yet been combined.
// This will be used when the consensus state is stored or forwared to another node
// in case of failure.
func (smm *partialMsgMap) getAllPartials() (ret []messages.MsgHeader) {
	smm.Lock()
	defer smm.Unlock()

	for _, nxtMap := range smm.partialMap {
		for _, partial := range nxtMap {
			partial.Lock()

			if partial.hasCombined { // combined messages are included from the signedMsgState object
				partial.Unlock()
				continue
			}
			for _, nxt := range partial.partials {
				if nxt == nil {
					continue
				}
				// TODO send the partials as a single signed message so we dont have to include the signature
				// in each partial.
				spm := partial.sm.ShallowCopy().(*sig.MultipleSignedMessage)
				spm.InternalSignedMsgHeader = nxt
				ret = append(ret, spm)
			}
			partial.Unlock()
		}
	}
	return
}

func newPartialMsgMap() *partialMsgMap {
	return &partialMsgMap{partialMap: make(map[sig.PubKeyID]map[types.HashStr]*partialMsgState)}
}

// gotMsg checks if the same partial message has already been received.
// If it already has validated the same signature and this is a new partial then (true, nil) is returned.
// If it is a new partial and a new signature then (false, nil) is returned.
// Otherwise an error is returned.
// It assumes there is only one valid signer per hash, this must be insured
// by the implementation of the consensus message state. For example to ensure partial
// messages only come from the coordinator as in MvCons1.
func (smm *partialMsgMap) gotMsg(sm *sig.MultipleSignedMessage) (bool, error) {
	if len(sm.GetSigItems()) != 1 { // partials should only be signed once, this should be checked previously
		// return false, utils.ErrInvalidSig
		// panic because this should be checked by the consensus message state implementation
		panic("partials should be checked to only have 1 signature by the consensus message state implementation")
	}
	pubID, err := sm.GetSigItems()[0].Pub.GetPubID()
	if err != nil {
		return false, err
	}

	smm.Lock()
	defer smm.Unlock()

	var ok bool
	// check if we have received any messsages from the pub id
	var idMap map[types.HashStr]*partialMsgState
	if idMap, ok = smm.partialMap[pubID]; !ok {
		return false, nil // we dont have the message yet
	}

	// check if we have received this message
	var pms *partialMsgState
	if pms, ok = idMap[sm.GetHashString()]; !ok {
		return false, nil // we dont have the message yet
	}
	if pms.partials[sm.GetBaseMsgHeader().(*messagetypes.PartialMessage).PartialMsgIndex] != nil {
		return false, types.ErrDuplicatePartial // we already got this partial index for this message
	}

	// we have already validated the signature, so just check they are the same
	if bytes.Equal(pms.sm.GetSigItems()[0].SigBytes, sm.GetSigItems()[0].SigBytes) {
		return true, nil
	}
	return false, types.ErrInvalidSig
}

// storeMsg shold be called after gotMsg, and after the signatres are validated.
// It stores the new partial, and tries to reconstruct the partial, returning it if possible.
func (smm *partialMsgMap) storeMsg(hdrFunc consinterface.HeaderFunc,
	sm *sig.MultipleSignedMessage, gc *generalconfig.GeneralConfig, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error) {

	pubID, err := sm.GetSigItems()[0].Pub.GetPubID()
	if err != nil {
		panic("should have handled this earlier")
	}

	smm.Lock()
	var ok bool
	// check if we have received any messsages from the pub id
	var idMap map[types.HashStr]*partialMsgState
	if idMap, ok = smm.partialMap[pubID]; !ok {
		idMap = make(map[types.HashStr]*partialMsgState)
		smm.partialMap[pubID] = idMap
	}

	// check if we have received this partial before
	var pms *partialMsgState
	if pms, ok = idMap[sm.GetHashString()]; !ok {
		pms = newPartialMsgState(sm)
	}
	smm.Unlock()

	pms.Lock()
	defer pms.Unlock() // TODO better concurrency here?

	if pms.hasCombined { // if we already reconstructed the message we return an error so we dont process/forward the unneeded partial
		return nil, types.ErrAlreadyCombinedPartials
	}

	partial := sm.GetBaseMsgHeader().(*messagetypes.PartialMessage)
	if pms.partials[partial.PartialMsgIndex] != nil {
		logging.Info("Got duplicate partial")
		return nil, types.ErrDuplicatePartial
	}
	logging.Info("Got new partial index", partial.PartialMsgIndex)
	pms.partials[partial.PartialMsgIndex] = partial

	fullMsg, err := messagetypes.GenerateCombinedMessageBytes(pms.partials)
	if err == types.ErrNotEnoughPartials {
		// we dont have enoug messages to combine yet
		return nil, nil
	} else if err != nil {
		logging.Error("Error combining partials: ", err)
		// there is something wrong with some partial
		// we still forward this one since the error might be with another
		// if this fails when we have all partials then future calls will fail in gotMsg since we know the message cant be reconsturcted
		return nil, nil
	}

	// The reconstruction was fine, so we set has combined to true
	pms.hasCombined = true
	cm, err := messagetypes.GenerateCombinedMessage(partial, fullMsg, hdrFunc, gc, sm.GetSigItems()[0].Pub.New())
	if err != nil {
		logging.Error("Error generating combined message from partials: ", err)
		// So if the generation fails is means there was a problem with the message itself, so we won't try to combine again
		// and dont process the partial
		return nil, err
	}
	logging.Info("Successfully combined partials")
	// use the same signatures for the new msg
	scm := sm.ShallowCopy().(*sig.MultipleSignedMessage)
	scm.InternalSignedMsgHeader = cm
	return scm, nil
}
