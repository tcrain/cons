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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"sort"
	"testing"
)

func TestTx(t *testing.T) {
	txTest(ec.NewEcpriv, (&ec.Ecsig{}).New, t)
}

func TestTxBLS(t *testing.T) {
	txTest(bls.NewBlspriv, (&bls.Blssig{}).New, t)
}

func generateAccountTable(newPrivFunc func() (sig.Priv, error), numAccounts int,
	initBalance uint64, t *testing.T) (accountTable *AccountTable, privs sig.SortPriv) {

	accountTable = NewAccountTable()
	privs = make(sig.SortPriv, numAccounts)
	var err error

	for i := 0; i < numAccounts; i++ {
		privs[i], err = newPrivFunc()
		assert.Nil(t, err)
	}
	sort.Sort(privs)
	assert.True(t, sort.IsSorted(privs))
	for i := 0; i < numAccounts; i++ {
		// Add an account with money
		pub := privs[i].GetPub()
		acc, err := accountTable.GetAccount(pub)
		assert.Nil(t, err)
		acc.Balance = initBalance
		accountTable.UpdateAccount(pub, acc)
	}
	return
}

func txTest(newPrivFunc func() (sig.Priv, error), newSigFunc func() sig.Sig, t *testing.T) {

	numAccounts := 100
	var initBalance uint64 = 100
	var err error
	accountTable, privs := generateAccountTable(newPrivFunc, numAccounts, initBalance, t)

	var sendAmmount uint64 = 10
	recPrivs := make([]sig.Priv, numAccounts)
	for i := 0; i < 100; i++ {
		// Make a tx
		recPrivs[i], err = newPrivFunc()
		assert.Nil(t, err)
		tx, err := accountTable.GenerateTx([]sig.Priv{privs[i]}, []uint64{sendAmmount},
			[]sig.Pub{recPrivs[i].GetPub()}, []uint64{sendAmmount})
		assert.Nil(t, err)

		err = ValidateTransferTxFormat(tx)
		assert.Nil(t, err)

		err = accountTable.ValidateTx(tx, true, nil)
		assert.Nil(t, err)

		buf := bytes.NewBuffer(nil)
		n1, err := tx.Encode(buf)
		assert.Nil(t, err)

		tx2 := &TransferTx{NewPubFunc: privs[i].GetPub().New, NewSigFunc: newSigFunc}
		n2, err := tx2.Decode(buf)
		assert.Nil(t, err)
		assert.Equal(t, n1, n2)

		err = ValidateTransferTxFormat(tx2)
		assert.Nil(t, err)

		err = accountTable.ValidateTx(tx2, true, nil)
		assert.Nil(t, err)

		accountTable.ConsumeTx(tx2, nil)

		acc, err := accountTable.GetAccount(privs[i].GetPub())
		assert.Nil(t, err)
		assert.Equal(t, acc.Balance, initBalance-sendAmmount)

		acc, err = accountTable.GetAccount(recPrivs[i].GetPub())
		assert.Nil(t, err)
		assert.Equal(t, acc.Balance, sendAmmount)
	}

	for i := 0; i < 1; i++ {
		// Make a tx
		tx, err := accountTable.GenerateTx([]sig.Priv{privs[i]}, []uint64{sendAmmount},
			[]sig.Pub{recPrivs[i].GetPub()}, []uint64{sendAmmount})
		assert.Nil(t, err)

		err = ValidateTransferTxFormat(tx)
		assert.Nil(t, err)

		// be sure it doesn't validate with a wrong signature
		tx.Signatures[0].(sig.CorruptInterface).Corrupt()
		err = accountTable.ValidateTx(tx, true, nil)
		assert.Equal(t, ErrSignatureInvalid, err)
	}

	// tx with multiple signatures
	txSize := numAccounts / 2
	for i := 0; i < numAccounts; i += txSize {
		sndAmts := make([]uint64, txSize)
		rcvAmts := make([]uint64, txSize)
		rcvPubs := make([]sig.Pub, txSize)
		for j := 0; j < txSize; j++ {
			sndAmts[j] = sendAmmount
			rcvAmts[j] = sendAmmount
			rcvPubs[j] = recPrivs[j+i].GetPub()
		}
		tx, err := accountTable.GenerateTx(privs[i:i+txSize], sndAmts,
			rcvPubs, rcvAmts)
		assert.Nil(t, err)

		err = ValidateTransferTxFormat(tx)
		assert.Nil(t, err)
		err = accountTable.ValidateTx(tx, true, nil)
		assert.Nil(t, err)

		accountTable.ConsumeTx(tx, nil)

		for j := i; j < i+txSize; j++ {
			acc, err := accountTable.GetAccount(privs[j].GetPub())
			assert.Nil(t, err)
			assert.Equal(t, acc.Balance, initBalance-2*sendAmmount)
		}
	}

	for i := 0; i < 1; i++ {
		// Make a tx with too much balance
		acc, err := accountTable.GetAccount(privs[i].GetPub())
		assert.Nil(t, err)

		// not enough balance
		_, err = accountTable.GenerateTx([]sig.Priv{privs[i]}, []uint64{acc.Balance + 1},
			[]sig.Pub{recPrivs[i].GetPub()}, []uint64{acc.Balance + 1})
		assert.Equal(t, ErrNotEnoughBalance, err)

		tx, err := accountTable.GenerateTx([]sig.Priv{privs[i]}, []uint64{acc.Balance},
			[]sig.Pub{recPrivs[i].GetPub()}, []uint64{acc.Balance})
		assert.Nil(t, err)
		tx.SendAmounts[0] += 1

		err = ValidateTransferTxFormat(tx)
		assert.Nil(t, err)
		err = accountTable.ValidateTx(tx, true, nil)
		assert.Equal(t, ErrNotEnoughBalance, err)

		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("Should have paniced")
				}
				t.Log(r)
			}()
			accountTable.ConsumeTx(tx, nil)
		}()
	}

}
