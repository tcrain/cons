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

/* MessageState tracks the signed messages received for each consensus instance,
and does things like reject duplicates and check message thresholds.
SignedItem messages are those that implement sig.MultiSigMsgHeader.
*/
package messagestate

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

///////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////

type sigMsgMapInterface interface {
	// gotMsg will be called by MessageState.GotMsg.
	// gotMsg checks if the same message has already been received from the signers
	// the message is updated so that it only contains the sigs not already seen
	// an error is sent if no new signers.
	gotMsg(sm *sig.MultipleSignedMessage) error
	gotUnsignedMsg(sm *sig.UnsignedMessage) error
	storeUnsignedMsg(sm *sig.UnsignedMessage,
		mc *consinterface.MemCheckers) (int, int, error)
	// storeMsg shold be called after gotMsg, and after the signatres are validated.
	// It stores the new valid signatures and returns the total number of signatures that the message has.
	// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
	// number of signatures for the MsgID of the message (see messages.MsgID).
	// sm should only contain valid signatures, invalidSigs should contain the remaining sigs that were included in sm during the call to gotMsg.
	storeMsg(sm *sig.MultipleSignedMessage, invalidSigs []*sig.SigItem, mc *consinterface.MemCheckers) (newTotalSigCount, newMsgIDSigCount int, err error)
	// setupSigs is called by MessageState.SetupSignedMessage.
	// It adds the signatures to the message as described in MessageState.SetupSignedMessage.
	// If generateSig is true is generates the local signature, and add the local VRFproof if non-nil
	setupSigs(sm *sig.MultipleSignedMessage, priv sig.Priv, generateMySig bool, myVrfProof sig.VRFProof, addOthersSigsCount int, mc *consinterface.MemCheckers) (bool, error)
	getSigCountMsg(hash types.HashStr) int  // getSigCountMsg returns the number of signatures received for this message.
	getSigCountMsgID(sm messages.MsgID) int // getSigCountMsgID returns the number of sigs for this message's MsgID (see messages.MsgID).
	// getSigCountMsgIDList returns list of received messages that have msgID for their MsgID and how many signatures have been received for each.
	getSigCountMsgIDList(messages.MsgID) []consinterface.MsgIDCount
	// getAllMsgsSigs returns a list of all received MsgHeaders with received signatures attached.
	// bufferContFunc is the same as consinterface.ConsItem.GetBufferCount(), the returned value endThreshold will
	// determine how many of each signature is added for each message type.
	// If localOnly is true then only proposal messages and signatures from the local node will be included.
	getAllMsgSigs(priv sig.Priv, localOnly bool,
		bufferCountFunc consinterface.BufferCountFunc, gc *generalconfig.GeneralConfig,
		mc *consinterface.MemCheckers) (ret []messages.MsgHeader)
	// getThreshSig returns the threshold signature for the message ID (if supported).
	getThreshSig(hash types.HashStr) (*sig.SigItem, error)
	// trackTotalSigCount must be called before getTotalSigCount with the same set of hashes.
	// This can be called for multiple sets of messages, but they all must be unique.
	trackTotalSigCount(msgHashes ...types.HashStr)
	// getCoinVal returns the value for the random coin and true if it is valid, false otherwise
	getCoinVal(hash types.HashStr) (coinVal types.BinVal, ready bool)
	// getTotalSigCount returns the number of unique signers for the set of msgHashes.
	// Unique means if a signer signs multiple of msgHashes he will still only be counted once.
	// Note that trackTotalSigCount must be called first (but only once) with the same
	// set of msgsHashes.
	getTotalSigCount(msgHashes ...types.HashStr) (totalCount int, eachCount []int)
}

// a speical sigMsgMap is used for BLS sigs when multisignatures is enabled,
// otherwise just the simple version is used
func newSigMsgMap(prev sigMsgMapInterface, index types.ConsensusIndex) sigMsgMapInterface {
	if !sig.GetUsePubIndex() || !sig.GetUseMultisig() {
		return newSignedMsgMap()
	}
	switch v := prev.(type) {
	case *blsSigState:
		// If we use BLS signatures, and multi-signatures, and pub key indecides then we use the special BLS sig map
		// that allow for merging signatures
		return newBlsSigState(index)
	case *signedMsgMap:
		return newSignedMsgMap()
	default:
		panic(v)
	}
}
