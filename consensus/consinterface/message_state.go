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

package consinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// MsgIDCount is used for the return value of MessageState.GetSigCountMsgIDList.
// It contains a header and the number of signatures received so far for that MsgID.
type MsgIDCount struct {
	MsgHeader *sig.MultipleSignedMessage
	Count     int // Count is the number of signatures received for MsgHeader.
}

// MessageState tracks the signed messages received for each consensus instance.
type MessageState interface {
	// GotMessage takes a deserialized message and the member checker for the current consensus index.
	// If the message contains no new valid signatures then an error is returned.
	// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
	// number of signatures for the MsgID of the message (see messages.MsgID).
	// If the message is not a signed type message (not type *sig.MultipleSignedMessage then (0, 0, nil) is returned).
	GotMsg(HeaderFunc,
		*deserialized.DeserializedItem, *generalconfig.GeneralConfig, *MemCheckers) ([]*deserialized.DeserializedItem, error)
	// GetMsgState should return the serialized state of all the valid messages received for this consensus instance.
	// This should be able to be processed by UnwrapMessage.
	// If generalconfig.UseFullBinaryState is true it returns the received signed messages together in a list.
	// Otherwise it returns each unique message with the list of signatures for each message behind it.
	// bufferContFunc is the same as consinterface.ConsItem.GetBufferCount(), the returned value endThreshold will
	// determine how many of each signature is added for each message type.
	// If localOnly is true then only proposal messages and signatures from the local node will be included.
	GetMsgState(priv sig.Priv, localOnly bool,
		bufferCountFunc BufferCountFunc,
		mc *MemCheckers) ([]byte, error)
	// New creates a new empty MessageState object for the consensus index idx.
	New(idx types.ConsensusIndex) MessageState
	// GetIndex returns the consensus index.
	GetIndex() types.ConsensusIndex
	// SetupSigned message takes a messages.InternalSignedMsgHeader, serializes the message appending signatures
	// creating and returning a new *sig.MultipleSignedMessage.
	// If generateMySig is true, the serialized message will include the local nodes signature.
	// Up to addOtherSigs number of signatures received so far for this message will additionally be appended.
	SetupSignedMessage(hdr messages.InternalSignedMsgHeader, generateMySig bool, addOthersSigsCount int, mc *MemCheckers) (*sig.MultipleSignedMessage, error)
	SetupUnsignedMessage(hdr messages.InternalSignedMsgHeader,
		mc *MemCheckers) (*sig.UnsignedMessage, error)
	// SetupSignedMessageDuplicates takes a list of headers that are assumed to have the same set of bytes to sign
	// (i.e. the signed parts are all the same though they may have different contents following the signed part,
	// for example this is true with partial messages)
	SetupSignedMessagesDuplicates(combined *messagetypes.CombinedMessage, hdrs []messages.InternalSignedMsgHeader,
		mc *MemCheckers) (combinedSigned *sig.MultipleSignedMessage, partialsSigned []*sig.MultipleSignedMessage, err error)
	GetSigCountMsg(types.HashStr) int                                                           // GetSigCountMsg returns the number of signatures received for this message.
	GetSigCountMsgHeader(header messages.InternalSignedMsgHeader, mc *MemCheckers) (int, error) // GetSigCountMsgHeader returns the number of signatures received for this header.
	GetSigCountMsgID(messages.MsgID) int                                                        // GetSigCountMsgID returns the number of sigs for this message's MsgID (see messages.MsgID).
	// GetSigCountMsgIDList returns list of received messages that have msgID for their MsgID and how many signatures have been received for each.
	GetSigCountMsgIDList(msgID messages.MsgID) []MsgIDCount
	// GetThreshSig returns the threshold signature for the message ID (if supported).
	GetThreshSig(hdr messages.InternalSignedMsgHeader, threshold int, mc *MemCheckers) (*sig.SigItem, error)
	// GetCoinVal returns the threshold coin value for the message ID (if supported).
	GetCoinVal(hdr messages.InternalSignedMsgHeader, threshold int, mc *MemCheckers) (coinVal types.BinVal, ready bool, err error)
}
