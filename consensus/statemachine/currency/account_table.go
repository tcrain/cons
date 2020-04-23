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
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/copywritemap"
	"github.com/tcrain/cons/consensus/utils"
	"sort"
	"strings"
)

// AccountTable provides operations to track a set of accounts.
// It is not concurrency safe.
type AccountTable struct {
	table *copywritemap.CopyWriteMap // map[sig.PubKeyStr]*Account
	stats AccountStats
}

// NewAccountTable generates an empty account table.
func NewAccountTable() *AccountTable {
	return &AccountTable{table: copywritemap.NewCopyWriteMap()} // make(map[sig.PubKeyStr]*Account)}
}

func (at *AccountTable) Copy() *AccountTable {
	return &AccountTable{
		table: at.table.Copy(),
		stats: at.stats}
}

func (at *AccountTable) DoneKeep() {
	at.table.DoneKeep()
}

func (at *AccountTable) DoneClear() {
	at.table.DoneClear()
}

func (at *AccountTable) String() string {
	var b strings.Builder
	b.WriteString("----- Account table ------\n")
	var strItems []string
	at.table.Range(func(key, value interface{}) bool {
		acc := value.(Account)
		byt, err := acc.Pub.GetPubBytes()
		if err != nil {
			panic(err)
		}
		strItems = append(strItems, fmt.Sprintf("|  (%v) count: %v, bal: %v\n", byt[:5], acc.TxCounter, acc.Balance))
		return true
	})
	sort.Strings(strItems)
	for _, nxt := range strItems {
		b.WriteString(nxt)
	}
	b.WriteString("----- End account table -----\n")
	return b.String()
}

// GenerateTx generates a transaction given the inputs.
// An error is returned if the generated transaction would be invalid.
func (at *AccountTable) GenerateTx(senders sig.SortPriv, amounts []uint64,
	recipiants []sig.Pub, receiveAmounts []uint64) (tx *TransferTx, err error) {

	defer func() {
		if err == nil {
			at.stats.GenerateTx()
		} else {
			at.stats.FailedGenerateTx()
		}
	}()

	if !sort.IsSorted(senders) {
		err = ErrSendersNotSorted
		return
	}

	senderTxCounters := make([]uint64, len(senders))
	senderPubs := make(sig.SortPub, len(senders))
	sigs := make([]sig.Sig, len(senders))
	var acc Account
	for i, sender := range senders {
		pub := sender.GetPub()
		senderPubs[i] = pub
		acc, err = at.GetAccount(pub)
		if err != nil {
			return
		}
		senderTxCounters[i] = acc.TxCounter + 1
	}
	tx = &TransferTx{Senders: senderPubs,
		SenderTxCounters: senderTxCounters,
		SendAmounts:      amounts,
		Recipiants:       recipiants,
		ReceiveAmounts:   receiveAmounts,
		Signatures:       sigs}

	err = ValidateTransferTxFormat(tx)
	if err != nil {
		return
	}

	err = tx.encodeSignedBytes()
	if err != nil {
		return
	}
	var asig sig.Sig
	for i, priv := range senders {
		asig, err = priv.Sign(tx)
		if err != nil {
			return
		}
		sigs[i] = asig
	}

	err = at.ValidateTx(tx, false, nil)
	return
}

// Range performs the func f over the accounts, it stops if f returns false.
// This should not modify the accounts.
func (at *AccountTable) Range(f func(pub sig.PubKeyStr, acc Account) bool) {
	at.table.Range(func(key, value interface{}) bool {
		return f(key.(sig.PubKeyStr), value.(Account))
	})
}

// GetAccount info returns the transaction counter and balance of the account.
func (at *AccountTable) GetAccountInfo(pub sig.Pub) (txCounter, balance uint64, err error) {
	acc, err := at.GetAccount(pub)
	if err != nil {
		return
	}
	return acc.TxCounter, acc.Balance, nil
}

// GetAccount returns the account object for the given public key.
func (at *AccountTable) GetAccount(pub sig.Pub) (acc Account, err error) {
	var str sig.PubKeyStr
	str, err = pub.GetPubString()
	if err != nil {
		return
	}
	if acc, ok := at.table.Read(str); ok {
		return acc.(Account), nil
	}
	acc = NewAccount(pub)
	at.table.Write(str, acc)
	return acc, nil
}

func (at *AccountTable) UpdateAccount(pub sig.Pub, account Account) {
	str, err := pub.GetPubString()
	if err != nil {
		panic(err)
	}
	at.table.Write(str, account)
}

// ConsumeTx consumes the transaction, sending any leftover sent money to leftOverPub.
// It should only be called after a transaction has validated correctly.
func (at *AccountTable) ConsumeTx(tx *TransferTx, leftOverPub sig.Pub) {
	at.stats.TxConsumedCount++
	for i, pub := range tx.Senders {
		acc, err := at.GetAccount(pub)
		if err != nil {
			panic(err)
		}
		acc.TxCounter += 1
		if acc.TxCounter != tx.SenderTxCounters[i] {
			panic(tx.SenderTxCounters[i])
		}
		if acc.Balance < tx.SendAmounts[i] {
			panic(tx.SendAmounts[i])
		}
		acc.Balance -= tx.SendAmounts[i]
		at.stats.MoneyTransfered += tx.SendAmounts[i]
		at.UpdateAccount(pub, acc)
	}
	for i, pub := range tx.Recipiants {
		acc, err := at.GetAccount(pub)
		if err != nil {
			panic(err)
		}
		if utils.CheckOverflow(acc.Balance, tx.ReceiveAmounts[i]) {
			panic(ErrIntOverflow)
		}
		acc.Balance += tx.ReceiveAmounts[i]
		at.UpdateAccount(pub, acc)
	}
	totalSent := utils.SumUint64(tx.SendAmounts...)
	totalReceived := utils.SumUint64(tx.ReceiveAmounts...)
	if totalSent < totalReceived {
		panic(ErrNotEnoughSend)
	}

	remain := totalSent - totalReceived
	if remain > 0 && leftOverPub != nil {
		leftOverAcc, err := at.GetAccount(leftOverPub)
		if err != nil {
			panic(err)
		}
		if utils.CheckOverflow(leftOverAcc.Balance, remain) {
			panic(ErrIntOverflow)
		}
		leftOverAcc.Balance += remain
		at.UpdateAccount(leftOverPub, leftOverAcc)
	}
}

// ValidateTx returns an error if the transaction is not valid. If validate sig is false then
// the transaction's signatures will not be validated.
// Unspent sent value would be sent to leftOverPub.
func (at *AccountTable) ValidateTx(tx *TransferTx, validateSig bool, leftOverPub sig.Pub) (err error) {
	defer func() {
		if err == nil {
			at.stats.PassedValidation()
		} else {
			at.stats.FailedValidation()
		}
	}()

	if err = ValidateTransferTxFormat(tx); err != nil {
		return
	}
	for i, pub := range tx.Senders {
		var acc Account
		acc, err = at.GetAccount(pub)
		if err != nil {
			return
		}
		if acc.TxCounter+1 != tx.SenderTxCounters[i] {
			err = ErrInvalidTxCounter
			return
		}
		if acc.Balance < tx.SendAmounts[i] {
			err = ErrNotEnoughBalance
			return
		}
	}
	if validateSig {
		for i, pub := range tx.Senders {
			var valid bool
			valid, err = pub.VerifySig(tx, tx.Signatures[i])
			if !valid {
				err = ErrSignatureInvalid
				return
			}
			if err != nil {
				return
			}
		}
	}
	totalSent := utils.SumUint64(tx.SendAmounts...)
	totalReceived := utils.SumUint64(tx.ReceiveAmounts...)
	if totalSent < totalReceived {
		panic("opps!!, shouldnt be here")
	}

	remain := totalSent - totalReceived
	if remain > 0 && leftOverPub != nil {
		var leftOverAcc Account
		leftOverAcc, err = at.GetAccount(leftOverPub)
		if err != nil {
			return
		}
		if utils.CheckOverflow(leftOverAcc.Balance, remain) {
			err = ErrIntOverflow
			return
		}
	}
	return
}
