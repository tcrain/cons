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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

func BroadcastHalfHalfFixedBin(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	BroadcastHalfHalf(true, nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel,
		gc, additionalMsgs...)
}

func BroadcastHalfHalfNormal(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	BroadcastHalfHalf(false, nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel,
		gc, additionalMsgs...)
}

// BroadcastHalfHalf broadcast different messages to the nodes.
func BroadcastHalfHalf(fixedBin bool, nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	if !generalconfig.CheckFaulty(ci.ConsItem.GetIndex(), gc) {
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		return
	}

	frontHalf, backHalf := ci.FwdChecker.GetHalfHalfForwardListFunc()
	switch auxMsg := msg.(type) {
	case *messagetypes.AuxProofMessage:
		if fixedBin {
			auxMsg.BinVal = 0
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			auxMsg.BinVal = 1 - auxMsg.BinVal
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.AuxBothMessage:
		if fixedBin {
			auxMsg.BinVal = 0
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			auxMsg.BinVal = 1 - auxMsg.BinVal
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.AuxStage0Message:
		if fixedBin {
			auxMsg.BinVal = 0
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		auxMsg.BinVal = 1 - auxMsg.BinVal
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.AuxStage1Message:
		if fixedBin {
			auxMsg.BinVal = 0
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			auxMsg.BinVal = 1 - auxMsg.BinVal
		}
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.BVMessage0:
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		msg = messagetypes.CreateBVMessage(1, auxMsg.Round)
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.BVMessage1:
		var binVal types.BinVal = 1
		if fixedBin {
			binVal = 0
		}
		msg = messagetypes.CreateBVMessage(binVal, auxMsg.Round)
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, frontHalf, mainChannel, additionalMsgs...)
		// Send the opposite bin val
		msg = messagetypes.CreateBVMessage(1-binVal, auxMsg.Round)
		ci.ConsItem.Broadcast(nextCoordPub, msg, signMessage, backHalf, mainChannel, additionalMsgs...)
	case *messagetypes.MvInitMessage, *messagetypes.MvInitSupportMessage:
		// Broadcast normally to first half
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, frontHalf, mainChannel, gc, additionalMsgs...)
		// Broadcast with byzantine proposal to back half
		switch auxMsg := auxMsg.(type) {
		case *messagetypes.MvInitMessage:
			auxMsg.Proposal, auxMsg.ByzProposal = auxMsg.ByzProposal, auxMsg.Proposal
		case *messagetypes.MvInitSupportMessage:
			auxMsg.Proposal, auxMsg.ByzProposal = auxMsg.ByzProposal, auxMsg.Proposal
		default:
			panic(auxMsg)
		}
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, backHalf, mainChannel, gc, additionalMsgs...)
		switch auxMsg := auxMsg.(type) {
		case *messagetypes.MvInitMessage:
			auxMsg.Proposal, auxMsg.ByzProposal = auxMsg.ByzProposal, auxMsg.Proposal
		case *messagetypes.MvInitSupportMessage:
			auxMsg.Proposal, auxMsg.ByzProposal = auxMsg.ByzProposal, auxMsg.Proposal
		default:
			panic(auxMsg)
		}
	default:
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	}
}

// Broadcast an aux proof message, but flip the binary value.
func BroadcastBinFlip(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	if !generalconfig.CheckFaulty(ci.ConsItem.GetIndex(), gc) {
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		return
	}

	switch auxMsg := msg.(type) {
	case *messagetypes.AuxBothMessage:
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			// auxMsg.BinVal = 1 - auxMsg.BinVal
			auxMsg.BinVal = types.Coin
		}
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxStage1Message:
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			// auxMsg.BinVal = 1 - auxMsg.BinVal
			auxMsg.BinVal = types.Coin
		}
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxStage0Message:
		// Send the opposite bin val
		auxMsg.BinVal = 1 - auxMsg.BinVal
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxProofMessage:
		// Send the opposite bin val
		if auxMsg.BinVal == types.Coin {
			auxMsg.BinVal = 0 // TODO what should be done here?
		} else {
			if auxMsg.AllowCoin {
				auxMsg.BinVal = types.Coin
			} else {
				auxMsg.BinVal = 1 - auxMsg.BinVal
			}
		}
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	// ci.ConsItem.Broadcast(nextCoordPub, auxMsg, forwardFunc, mainChannel, additionalMsgs...)
	case *messagetypes.BVMessage0:
		// Send the opposite bin val
		msg = messagetypes.CreateBVMessage(1, auxMsg.Round)
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.BVMessage1:
		// Send the opposite bin val
		msg = messagetypes.CreateBVMessage(0, auxMsg.Round)
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	default:
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	}
}

// Broadcast an aux proof message, but dont actually send it.
func BroadcastMute(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	if !generalconfig.CheckFaulty(ci.ConsItem.GetIndex(), gc) {
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		return
	}

	// Send nothing
}

// Broadcast two aux proof messages, one for 0 and one for 1.
func BroadcastBinBoth(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	if !generalconfig.CheckFaulty(ci.ConsItem.GetIndex(), gc) {
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		return
	}

	switch auxMsg := msg.(type) {
	case *messagetypes.AuxStage0Message:
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		// Send the opposite bin val
		// auxMsg.SetSigItems(nil)
		BroadcastBinFlip(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxStage1Message:
		auxMsg.BinVal = types.Coin
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		auxMsg.BinVal = 0
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		auxMsg.BinVal = 1
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxBothMessage:
		auxMsg.BinVal = types.Coin
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		auxMsg.BinVal = 0
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		auxMsg.BinVal = 1
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	case *messagetypes.AuxProofMessage:
		if auxMsg.AllowCoin { // Broadcast all 3 options
			auxMsg.BinVal = types.Coin
			consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
			auxMsg.BinVal = 0
			consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
			auxMsg.BinVal = 1
			consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		} else {
			// Send the normal bin val
			consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
			// Send the opposite bin val
			// auxMsg.SetSigItems(nil)
			BroadcastBinFlip(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		}
	case *messagetypes.BVMessage0, *messagetypes.BVMessage1:
		// Send the normal bin val
		consinterface.NormalBroadcast(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		// Send the opposite bin val
		// auxMsg.SetSigItems(nil)
		BroadcastBinFlip(nextCoordPub, ci, auxMsg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	default:
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	}
}

// BroadcastMuteExceptInit only broadcasts init messages
func BroadcastMuteExceptInit(nextCoordPub sig.Pub,
	ci *consinterface.ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	forwardFunc channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader) {

	if !generalconfig.CheckFaulty(ci.ConsItem.GetIndex(), gc) {
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
		return
	}

	switch msg.GetID() {
	case messages.HdrMvInit, messages.HdrMvInitSupport:
		consinterface.NormalBroadcast(nextCoordPub, ci, msg, signMessage, forwardFunc, mainChannel, gc, additionalMsgs...)
	default: // Do nothing
	}
}
