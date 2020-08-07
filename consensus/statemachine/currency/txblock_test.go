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
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func TestTxBlock(t *testing.T) {
	newPrivFunc := ec.NewEcpriv
	newSigFunc := (&ec.Ecsig{}).New

	numAccounts := 100
	var initBalance uint64 = 100
	var err error
	accountTable, privs := generateAccountTable(newPrivFunc, numAccounts, initBalance, t)

	var sendAmmount uint64 = 10
	recPrivs := make([]sig.Priv, numAccounts)

	txList := make([]*TransferTx, numAccounts)
	for i := 0; i < numAccounts; i++ {
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
		txList[i] = tx
	}
	txBlock := &TxBlock{
		PrevHash:     types.GetZeroBytesHashLength(),
		Transactions: txList}

	buf := bytes.NewBuffer(nil)
	var n1, n2 int
	n1, err = txBlock.Encode(buf)
	assert.Nil(t, err)

	txBlock2 := &TxBlock{
		NewPubFunc: privs[0].GetPub().New,
		NewSigFunc: newSigFunc,
	}

	n2, err = txBlock2.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, n1, n2)
	assert.Equal(t, txBlock.TxMerkleRoot, txBlock2.TxMerkleRoot)
	assert.Equal(t, txBlock.Index, txBlock2.Index)
	assert.Equal(t, txBlock.PrevHash, txBlock2.PrevHash)
	assert.Equal(t, len(txBlock.Transactions), len(txBlock2.Transactions))

	for i, tx2 := range txBlock2.Transactions {
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
}

func TestInitBlock(t *testing.T) {
	newPrivFunc := ec.NewEcpriv
	priv, err := newPrivFunc()
	assert.Nil(t, err)
	newPubFunc := priv.GetPub().New

	blockSize := 100
	var initAmmount uint64 = 10

	ib := InitBlock{
		InitUniqueName: InitUniqueName,
		InitBalances:   make([]uint64, blockSize),
		InitPubs:       make([]sig.Pub, blockSize)}
	for i := 0; i < blockSize; i++ {
		priv, err := newPrivFunc()
		assert.Nil(t, err)
		ib.InitPubs[i] = priv.GetPub()
		ib.InitBalances[i] = initAmmount
	}

	buf := bytes.NewBuffer(nil)
	n1, err := ib.Encode(buf)
	assert.Nil(t, err)

	ib2 := InitBlock{NewPubFunc: newPubFunc}
	n2, err := ib2.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, n1, n2)

	assert.Equal(t, ib.InitUniqueName, ib2.InitUniqueName)
	assert.Equal(t, ib.InitBalances, ib2.InitBalances)

	for i, pub := range ib.InitPubs {
		ps1, err := pub.GetPubString()
		assert.Nil(t, err)
		ps2, err := ib2.InitPubs[i].GetPubString()
		assert.Nil(t, err)
		assert.Equal(t, ps1, ps2)
	}
}
