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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
)

// SimpleMessageStateWrapper is just a wrapper that allows the bin cons and mv cons message state objects
// to reuse these methods directly without reimplementing them.
type SimpleMessageStateWrapper struct {
	Sms SimpleMessageState
}

// InitSimpleMessageStateWrapper initializes the wrapper.
func InitSimpleMessageStateWrapper(gc *generalconfig.GeneralConfig) *SimpleMessageStateWrapper {
	return &SimpleMessageStateWrapper{*NewSimpleMessageState(gc)}
}

func (sms *SimpleMessageStateWrapper) GetSigCountMsg(hashStr types.HashStr) int {
	return sms.Sms.GetSigCountMsg(hashStr)
}

// GetThreshSig returns the threshold signature for the message ID (if supported).
func (sms *SimpleMessageStateWrapper) GetThreshSig(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (*sig.SigItem, error) {

	return sms.Sms.GetThreshSig(hdr, threshold, mc)
}

func (sms *SimpleMessageStateWrapper) GetSigCountMsgHeader(header messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (int, error) {

	return sms.Sms.GetSigCountMsgHeader(header, mc)
}

func (sms *SimpleMessageStateWrapper) GetSigCountMsgID(msgID messages.MsgID) int {
	return sms.Sms.GetSigCountMsgID(msgID)
}

func (sms *SimpleMessageStateWrapper) GetSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {
	return sms.Sms.GetSigCountMsgIDList(msgID)
}

func (sms *SimpleMessageStateWrapper) SetupSignedMessagesDuplicates(combined *messagetypes.CombinedMessage, hdrs []messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (combinedSigned *sig.MultipleSignedMessage, partialsSigned []*sig.MultipleSignedMessage, err error) {

	return sms.Sms.SetupSignedMessagesDuplicates(combined, hdrs, mc)
}

func (sms *SimpleMessageStateWrapper) SetupSignedMessage(hdr messages.InternalSignedMsgHeader,
	generateMySig bool, addOthersSigsCount int, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error) {

	return sms.Sms.SetupSignedMessage(hdr, generateMySig, addOthersSigsCount, mc)
}

func (sms *SimpleMessageStateWrapper) GetIndex() types.ConsensusIndex {
	return sms.Sms.index
}

func (sms *SimpleMessageStateWrapper) GetMsgState(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc,
	mc *consinterface.MemCheckers) ([]byte, error) {
	return sms.Sms.GetMsgState(priv, localOnly, bufferCountFunc, mc)
}

func (sms *SimpleMessageStateWrapper) SetupUnsignedMessage(hdr messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (*sig.UnsignedMessage, error) {

	return sms.Sms.SetupUnsignedMessage(hdr, mc)
}

// NewWrapper creates a new SimpleMessageStateWrapper by calling the New(idx) method on the SimpleMessageState object.
func (sms *SimpleMessageStateWrapper) NewWrapper(firstIdx types.ConsensusIndex) *SimpleMessageStateWrapper {
	return &SimpleMessageStateWrapper{*sms.Sms.New(firstIdx).(*SimpleMessageState)}
}

// GetCoinVal returns the threshold coin value for the message ID (if supported).
func (sms *SimpleMessageStateWrapper) GetCoinVal(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (coinVal types.BinVal, ready bool, err error) {

	return sms.Sms.GetCoinVal(hdr, threshold, mc)
}
