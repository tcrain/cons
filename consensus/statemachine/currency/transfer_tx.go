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

package currency

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/statemachine/transactionsm"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
	"sort"
)

// TransferTx represents a transaction sending currency from one group of public keys to another.
type TransferTx struct {
	Senders          sig.SortPub    // The keys sending the currency.
	Signatures       []sig.Sig      // The signatures of the senders (of the serialized transaction).
	SenderTxCounters []uint64       // The counter of how many transactions each sender has made.
	Recipiants       []sig.Pub      // The keys receiving the money.
	SendAmounts      []uint64       // The amount being sent from each sender.
	ReceiveAmounts   []uint64       // The amount being received at each receiver.
	SignedBytes      []byte         // The bytes to sign.
	NewPubFunc       func() sig.Pub // Used during deserialzation, should return an empty public key of the type being used.
	NewSigFunc       func() sig.Sig // Used during deserialization, should return an empty signature object of the key type being used.
	// FullBytes []byte
}

var ErrMustSignFirst = fmt.Errorf("must sign tx before encoding")
var ErrTooManyParticipants = fmt.Errorf("too many signatures")
var ErrTxTooLarge = fmt.Errorf("tx too large")
var ErrSendersNotSorted = fmt.Errorf("senders must be sorted")

const MaxParticipants = 100 // TODO
const MaxTxSize = 10000     // TODO

// Decode decodes the transaction into tx, returning the number of bytes read and any errors.
func (tx *TransferTx) Decode(reader io.Reader) (n int, err error) {
	var nxtN int
	var signedByteCount uint16
	signedByteCount, nxtN, err = utils.ReadUint16(reader)
	n += nxtN
	if err != nil {
		return
	}
	if signedByteCount > MaxTxSize {
		err = ErrTxTooLarge
		return
	}
	tx.SignedBytes = make([]byte, signedByteCount)
	nxtN, err = reader.Read(tx.SignedBytes)
	n += nxtN
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(tx.SignedBytes)
	var senderCount uint16
	senderCount, _, err = utils.ReadUint16(buf)
	if err != nil {
		return
	}
	if senderCount > MaxParticipants {
		err = ErrTooManyParticipants
		return
	}
	tx.SendAmounts = make([]uint64, int(senderCount))
	tx.Senders = make(sig.SortPub, int(senderCount))
	tx.SenderTxCounters = make([]uint64, int(senderCount))
	for i := 0; i < int(senderCount); i++ {
		tx.Senders[i] = tx.NewPubFunc().New()
		_, err = tx.Senders[i].Decode(buf)
		if err != nil {
			return
		}
		tx.SenderTxCounters[i], _, err = utils.ReadUint64(buf)
		if err != nil {
			return
		}
		tx.SendAmounts[i], _, err = utils.ReadUint64(buf)
		if err != nil {
			return
		}
	}
	if !sort.IsSorted(tx.Senders) {
		err = ErrSendersNotSorted
		return
	}

	var recipiantCount uint16
	recipiantCount, _, err = utils.ReadUint16(buf)
	if err != nil {
		return
	}
	if recipiantCount > MaxParticipants {
		err = ErrTooManyParticipants
		return
	}

	tx.ReceiveAmounts = make([]uint64, int(recipiantCount))
	tx.Recipiants = make([]sig.Pub, int(recipiantCount))
	for i := 0; i < int(recipiantCount); i++ {
		tx.Recipiants[i] = tx.NewPubFunc().New()
		_, err = tx.Recipiants[i].Decode(buf)
		if err != nil {
			return
		}
		tx.ReceiveAmounts[i], _, err = utils.ReadUint64(buf)
		if err != nil {
			return
		}
	}

	// signatures
	tx.Signatures = make([]sig.Sig, int(senderCount))
	for i := 0; i < int(senderCount); i++ {
		tx.Signatures[i] = tx.NewSigFunc()
		nxtN, err = tx.Signatures[i].Decode(reader)
		n += nxtN
		if err != nil {
			return
		}
	}
	return
}

func (tx *TransferTx) encodeSignedBytes() error {
	buf := bytes.NewBuffer(nil)
	_, err := utils.EncodeUint16(uint16(len(tx.Senders)), buf)
	if err != nil {
		return err
	}
	for i, pub := range tx.Senders {
		_, err = pub.Encode(buf)
		if err != nil {
			return err
		}
		_, err = utils.EncodeUint64(tx.SenderTxCounters[i], buf)
		if err != nil {
			return err
		}
		_, err = utils.EncodeUint64(tx.SendAmounts[i], buf)
		if err != nil {
			return err
		}
	}
	_, err = utils.EncodeUint16(uint16(len(tx.Recipiants)), buf)
	if err != nil {
		return err
	}
	for i, pub := range tx.Recipiants {
		_, err = pub.Encode(buf)
		if err != nil {
			return err
		}
		_, err = utils.EncodeUint64(tx.ReceiveAmounts[i], buf)
		if err != nil {
			return err
		}
	}
	tx.SignedBytes = buf.Bytes()
	return nil
}

// GetConflictObjects returns the pub keys.
func (tx *TransferTx) GetConflictObjects() []transactionsm.ConflictObject {
	ret := make([]transactionsm.ConflictObject, len(tx.Senders))
	for i, pub := range tx.Senders {
		str, err := pub.GetPubString() // TODO how should do this?
		if err != nil {
			panic("should have handled this earlier")
		}
		ret[i] = str
	}
	return ret
}

// Encode encodes the transaction into writer, it returns the number of bytes written and any errors.
func (tx *TransferTx) Encode(writer io.Writer) (n int, err error) {
	if tx.SignedBytes == nil {
		err = ErrMustSignFirst
		return
	}
	var nxtN int
	if len(tx.SignedBytes) > math.MaxUint16 {
		panic(1)
	}
	nxtN, err = utils.EncodeUint16(uint16(len(tx.SignedBytes)), writer)
	n += nxtN
	if err != nil {
		return
	}

	nxtN, err = writer.Write(tx.SignedBytes)
	n += nxtN
	if err != nil {
		return
	}

	// Signatures
	for _, sig := range tx.Signatures {
		nxtN, err = sig.Encode(writer)
		n += nxtN
		if err != nil {
			return
		}
	}
	return
}

// GetSignedBytes returns the bytes that should be signed by the senders.
func (tx *TransferTx) GetSignedMessage() []byte {
	return tx.SignedBytes
}

// GetSignedHash returns messages.GetHash(tx.GetSignedMessage())
func (tx *TransferTx) GetSignedHash() types.HashBytes {
	return types.GetHash(tx.SignedBytes)
}

var ErrIntOverflow = fmt.Errorf("integer overflow")
var ErrNotEnoughSend = fmt.Errorf("receive amount exceeds send ammount")
var ErrNoSend = fmt.Errorf("must send a at least 1 value")
var ErrNoSender = fmt.Errorf("must have at least one sender")
var ErrInvalidSenderCount = fmt.Errorf("senders must match send array")
var ErrInvalidReceiverCount = fmt.Errorf("receivers must match receivers array")
var ErrNotEnoughBalance = fmt.Errorf("account does not have enough balance")
var ErrInvalidTxCounters = fmt.Errorf("tx counters array much match senders array")
var ErrInvalidTxCounter = fmt.Errorf("tx counter does not match account counter")
var ErrInvalidSignatureCount = fmt.Errorf("signature array must match signer array")
var ErrSignatureInvalid = fmt.Errorf("signature does not validate")
var ErrNonUniqueSenders = fmt.Errorf("senders not unique")
var ErrNonUniqueReceivers = fmt.Errorf("receivers not unique")

// ValidateTransferTxFormat returns an error if the decoded transaction format contains any errors.
func ValidateTransferTxFormat(tx *TransferTx) error {
	if len(tx.Senders) == 0 {
		return ErrNoSender
	}
	if !sort.IsSorted(tx.Senders) {
		return ErrSendersNotSorted
	}
	if len(tx.Senders) > MaxParticipants {
		return ErrTooManyParticipants
	}
	if len(tx.Recipiants) > MaxParticipants {
		return ErrTooManyParticipants
	}
	if len(tx.SenderTxCounters) != len(tx.Senders) {
		return ErrInvalidTxCounters
	}
	if len(tx.Senders) != len(tx.SendAmounts) {
		return ErrInvalidSenderCount
	}
	if len(tx.Senders) != len(tx.Signatures) {
		return ErrInvalidSignatureCount
	}
	if len(tx.Recipiants) != len(tx.ReceiveAmounts) {
		return ErrInvalidReceiverCount
	}
	if utils.CheckOverflow(tx.SendAmounts...) || utils.CheckOverflow(tx.ReceiveAmounts...) {
		return ErrIntOverflow
	}
	sendSum := utils.SumUint64(tx.SendAmounts...)
	if sendSum == 0 {
		return ErrNoSend
	}
	if sendSum < utils.SumUint64(tx.ReceiveAmounts...) {
		return ErrNotEnoughSend
	}

	sndStrs := make([]utils.Any, len(tx.Senders))
	var err error
	for i, nxt := range tx.Senders {
		sndStrs[i], err = nxt.GetPubString()
		if err != nil {
			return err
		}
	}
	if !utils.CheckUnique(sndStrs...) {
		return ErrNonUniqueSenders
	}

	rcvStrs := make([]utils.Any, len(tx.Recipiants))
	for i, nxt := range tx.Recipiants {
		rcvStrs[i], err = nxt.GetPubString()
		if err != nil {
			return err
		}
	}
	if !utils.CheckUnique(rcvStrs...) {
		return ErrNonUniqueReceivers
	}
	return nil
}
