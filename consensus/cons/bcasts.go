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

package cons

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// CheckIncludeEchoProofs checks what kind of collect broadcast (see types.CollectBroadcast)
// should be done for the message. This should be used for "commit" type messages, i.e.
// of the init->echo->commit message pattern.
// If gc.CollectBroadcast is types.EchoCommit or configIncludeProofs is true then proofs of validity
// should be included with the message.
// If the value returned in sendToCoord is non-nil the the message should only be sent to the coordinator
// (see types.Commit and types.EchoCommit).
func CheckIncludeEchoProofs(round types.ConsensusRound, ci *consinterface.ConsInterfaceItems,
	configIncludeProofs bool, gc *generalconfig.GeneralConfig) (includeProofs bool, sendToCoord sig.Pub) {
	// Include proofs if either:
	// (1) includeProofs is true
	if configIncludeProofs {
		includeProofs = true
	}
	// (2) EchoCommit is true, in this case if we are the coordinator then we will broadcast the
	// proof to all participants
	if gc.CollectBroadcast == types.EchoCommit {
		_, _, err := consinterface.CheckCoord(ci.MC.MC.GetMyPriv().GetPub(), ci.MC, round, nil)
		if err == nil { // If I am the coordinator, then I broadcast the message to all participants, with the proofs.
			includeProofs = true
			return
		}
	}
	// Check if we should send the commit to the next coord instead of all to all
	if gc.CollectBroadcast != types.Full {
		sendToCoord = GetNextCoordPubCollectBroadcast(round, ci, gc)
	}
	return
}

// GetCoordPubCollectBroadcast is used to check if a message should be broadcast to all nodes,
// or just to the next coordinator (see types.CollectBroadcast) for the CURRENT round/instance of consensus.
// It returns a the expected coordinator of round if gc.CollectBroadcast
// is types.EchoCommit, it returns nil otherwise.
func GetCoordPubCollectBroadcast(round types.ConsensusRound, ci *consinterface.ConsInterfaceItems,
	gc *generalconfig.GeneralConfig) sig.Pub {

	var cordPub sig.Pub
	if gc.CollectBroadcast == types.EchoCommit {
		var err error
		_, cordPub, err = consinterface.CheckCoord(nil, ci.MC, round, nil)
		if err != nil {
			logging.Error("could not calculate coordPub")
			cordPub = nil
		}
	}
	return cordPub
}

// GetNextCoordPubCollectBroadcast is used to check if a message should be broadcast to all nodes,
// or just to the next coordinator (see types.CollectBroadcast) for the NEXT round/instance of consensus.
// It returns a the expected coordinator of round if gc.CollectBroadcast
// is not types.Full (also see types.Commit, types.EchoCommit), it returns nil if gc.CollectBroadcast
// is full.
func GetNextCoordPubCollectBroadcast(round types.ConsensusRound, ci *consinterface.ConsInterfaceItems,
	gc *generalconfig.GeneralConfig) sig.Pub {

	var nxtCoordPub sig.Pub
	if gc.CollectBroadcast != types.Full {
		var err error
		if nxtCoordPub, err = ci.MC.MC.CheckEstimatedRoundCoordNextIndex(nil, round); err != nil {
			logging.Warning(err)
		}
	}
	return nxtCoordPub
}

// DoConsBroadcast broadcasts a message for the given inputs.
// If nxtCordPub is not nil (the next coordinator public key), the message is only sent to that node (see types.CollectBroadcast).
// If the msg is nil then just the proofs will be sent.
// If gc has partial messages enabled for the message type the message will be sent as pieces.
func DoConsBroadcast(nxtCoordPub sig.Pub, msg messages.InternalSignedMsgHeader, signMessage bool, proofMsgs []messages.MsgHeader,
	forwardFunc channelinterface.NewForwardFuncFilter, ci *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, gc *generalconfig.GeneralConfig) {

	// a nil mvMsg means we are the coordinator for this round, and we just need to send proofs
	if msg != nil && ci.ConsItem.ShouldCreatePartial(msg.GetID()) {
		err := PartialBroadcastFunc(gc.PartialMessageType, ci.ConsItem, msg, 0, ci.MsgState,
			forwardFunc, ci.MC, mainChannel)
		if err != nil {
			logging.Error(err)
		}
	} else {
		// Add any needed signatures
		var sms messages.MsgHeader
		var err error
		if msg != nil {
			if signMessage {
				sms, err = ci.MsgState.SetupSignedMessage(msg, true, 0, ci.MC)
				if err != nil {
					logging.Error(err)
				}
			} else {
				sms, err = ci.MsgState.SetupUnsignedMessage(msg, ci.MC)
				if err != nil {
					logging.Error(err)
				}
			}
		}
		if nxtCoordPub != nil { // Only broadcast to the next coorindator
			err = mainChannel.SendToPub(messages.AppendCopyMsgHeader(ci.ConsItem.GetPreHeader(), append(proofMsgs, sms)...), nxtCoordPub,
				ci.MC.MC.GetStats().IsRecordIndex())
			if err != nil { // TODO what to do with non all to all connections?
				logging.Warningf("Error sending message to next coord %v, index %v, will broadcast message instead", err,
					ci.ConsItem.GetIndex())
			} else {
				return
			}
		}
		var isProposal bool
		if msg != nil {
			isProposal = messages.IsProposalHeader(ci.ConsItem.GetIndex(), msg)
		}
		mainChannel.SendHeader(messages.AppendCopyMsgHeader(ci.ConsItem.GetPreHeader(), append(proofMsgs, sms)...),
			isProposal, true, forwardFunc,
			ci.MC.MC.GetStats().IsRecordIndex())
	}
}

// PartialBroadcastFunc is a helper function that broadcasts a partial message.
func PartialBroadcastFunc(partialType types.PartialMessageType, abi consinterface.ConsItem, mvMsg messages.InternalSignedMsgHeader,
	round types.ConsensusRound, messageState consinterface.MessageState, forwardFunc channelinterface.NewForwardFuncFilter,
	mc *consinterface.MemCheckers, mainChannel channelinterface.MainChannel) error {

	destinations := mainChannel.ComputeDestinations(forwardFunc)

	combined, partials, err := messagetypes.CreatePartial(mvMsg, round, len(destinations), partialType)
	if err != nil {
		return err
	}

	logging.Error(combined, partials)
	// setup the signatures for the partials
	combinedSigned, partialsSigned, err := messageState.SetupSignedMessagesDuplicates(combined, partials, mc)
	if err != nil {
		return err
	}

	logging.Error("hashes", combinedSigned.GetSignedHash(), partialsSigned[0].GetSignedHash())
	// Send the combined message to yourself
	msg, err := messages.CreateMsg(abi.GetPreHeader())
	if err != nil {
		return err
	}
	// Add any needed signatures for the combined
	// Then the new msg
	_, err = messages.AppendHeader(msg, combinedSigned)
	if err != nil {
		return err
	}
	// Send the message to yourself
	toSelf := []*channelinterface.DeserializedItem{{
		Index:          abi.GetIndex(),
		HeaderType:     combinedSigned.GetID(),
		Header:         combinedSigned,
		IsDeserialized: true,
		IsLocal:        types.LocalMessage,
		Message:        sig.FromMessage(msg)}}
	logging.Error("hashes2", combinedSigned.GetSignedHash(), partialsSigned[0].GetSignedHash())
	mainChannel.SendToSelf(toSelf, 0)

	for i, nxt := range partialsSigned {
		msg, err := messages.CreateMsg(abi.GetPreHeader())
		if err != nil {
			panic(err)
		}
		// Then the new msg
		_, err = messages.AppendHeader(msg, nxt)
		if err != nil {
			panic(err)
		}

		mainChannel.SendTo(msg.GetBytes(), destinations[i], mc.MC.GetStats().IsRecordIndex())
	}
	return nil
}
