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
package transactionsm

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/types"
	"testing"
	"time"
)

var poolSize = 100
var minProposeSize = 10
var maxProposeSize = 20

func TestPoolInsert(t *testing.T) {
	pool := AllocateTransactionPool(poolSize, minProposeSize, maxProposeSize)
	retChan := make(chan []TransactionInterface, 1)
	pool.GetProposal(types.ConsensusInt(1), retChan)

	var allTx []TransactionInterface
	for i := 0; i < minProposeSize; i++ {
		select {
		case <-retChan:
			assert.Failf(t, "Should not be available", "")
		default:
		}
		tx := &TestTx{i}
		allTx = append(allTx, tx)

		ins, errs := pool.SubmitTransaction(tx)
		assert.Nil(t, errs)
		assert.Equal(t, 1, len(ins))

		var inserted []TransactionInterface
		assert.Equal(t, append(inserted, tx), ins)
	}
	ret, ok := <-retChan
	assert.True(t, ok)
	assert.Equal(t, allTx, ret)
}

func TestPoolMultiple(t *testing.T) {
	pool := AllocateTransactionPool(poolSize, minProposeSize, maxProposeSize)

	var allTx []TransactionInterface
	// insert one less than the min proposal size
	var i int
	for i = 0; i < poolSize; i++ {
		tx := &TestTx{i}
		allTx = append(allTx, tx)
	}
	ins, errs := pool.SubmitTransaction(allTx...)
	assert.Nil(t, errs)
	assert.Equal(t, i, len(ins))
	assert.Equal(t, allTx, ins)

	retChan := make(chan []TransactionInterface, 1)
	for i := 0; i < poolSize; i += maxProposeSize {
		pool.GetProposal(types.ConsensusInt(i+1), retChan)
		ret, ok := <-retChan
		assert.True(t, ok)
		assert.Equal(t, allTx[i:i+maxProposeSize], ret)
	}
	select {
	case <-retChan:
		assert.Failf(t, "Should not be available", "")
	default:
	}
}

func TestPoolRemove(t *testing.T) {
	pool := AllocateTransactionPool(poolSize, minProposeSize, maxProposeSize)

	var allTx []TransactionInterface
	// insert one less than the min proposal size
	var i int
	for i = 0; i < poolSize; i++ {
		tx := &TestTx{i}
		allTx = append(allTx, tx)
	}
	ins, errs := pool.SubmitTransaction(allTx...)
	assert.Nil(t, errs)
	assert.Equal(t, i, len(ins))
	assert.Equal(t, allTx, ins)

	retChan := make(chan []TransactionInterface, 1)
	for i := 0; i < poolSize; i += maxProposeSize * 2 {
		pool.ValidatePool(func(tx TransactionInterface) error {
			if tx.(*TestTx).Id >= i {
				return nil
			}
			return fmt.Errorf("invalid tx id")
		})
		pool.GetProposal(types.ConsensusInt(i+1), retChan)
		ret, ok := <-retChan
		assert.True(t, ok)
		assert.Equal(t, allTx[i:i+maxProposeSize], ret)
	}
	select {
	case <-retChan:
		assert.Failf(t, "Should not be available", "")
	default:
	}
}

func TestPoolBlock(t *testing.T) {
	tx := &TestTx{1}
	pool := AllocateTransactionPool(1, 1, 10)
	ins, errs := pool.SubmitTransaction(tx)
	assert.Nil(t, errs)
	var inserted []TransactionInterface
	assert.Equal(t, append(inserted, tx), ins)

	doneChan := make(chan bool, 1)
	go func() {
		pool.BlockUntilNotFull()
		doneChan <- true
	}()
	time.Sleep(100 * time.Millisecond)

	select {
	case <-doneChan:
		assert.Failf(t, "Should not be avilable", "")
	default:
	}

	retChan := make(chan []TransactionInterface, 1)
	pool.GetProposal(types.ConsensusInt(1), retChan)
	ret, ok := <-retChan
	assert.True(t, ok)
	assert.Equal(t, []TransactionInterface{tx}, ret)

	_, ok = <-doneChan
	assert.True(t, ok)
}

func TestPoolConflict(t *testing.T) {
	tx := &TestTx{1}
	pool := AllocateTransactionPool(poolSize, 1, 10)
	ins, errs := pool.SubmitTransaction(tx)
	assert.Nil(t, errs)
	var inserted []TransactionInterface
	assert.Equal(t, append(inserted, tx), ins)

	ins, errs = pool.SubmitTransaction(tx)
	assert.Equal(t, []TransactionInterface(nil), ins)
	assert.Equal(t, []error{ErrTransactionAreadyInPool}, errs)
	retChan := make(chan []TransactionInterface, 1)
	pool.GetProposal(types.ConsensusInt(1), retChan)

	ret, ok := <-retChan
	assert.True(t, ok)
	assert.Equal(t, []TransactionInterface{tx}, ret)
}

func TestPoolNow(t *testing.T) {
	pool := AllocateTransactionPool(poolSize, minProposeSize, maxProposeSize)
	retChan := make(chan []TransactionInterface, 1)
	pool.GetProposal(types.ConsensusInt(1), retChan)

	var allTx []TransactionInterface
	// insert one less than the min proposal size
	var i int
	for i = 0; i < minProposeSize-1; i++ {
		tx := &TestTx{i}
		allTx = append(allTx, tx)
	}
	ins, errs := pool.SubmitTransaction(allTx...)
	assert.Nil(t, errs)
	assert.Equal(t, i, len(ins))

	assert.Equal(t, allTx, ins)

	select {
	case <-retChan:
		assert.Failf(t, "Should not be available", "")
	default:
	}
	pool.GetProposalNow(types.ConsensusInt(1))
	ret, ok := <-retChan
	assert.True(t, ok)
	assert.Equal(t, allTx, ret)
}

func TestPoolFull(t *testing.T) {
	tx := &TestTx{1}
	pool := AllocateTransactionPool(1, 1, 10)
	ins, errs := pool.SubmitTransaction(tx)
	assert.Nil(t, errs)
	var inserted []TransactionInterface
	assert.Equal(t, append(inserted, tx), ins)

	tx2 := &TestTx{2}
	ins, errs = pool.SubmitTransaction(tx2)
	assert.Equal(t, []TransactionInterface(nil), ins)
	assert.Equal(t, []error{ErrTransactionPoolFull}, errs)
	retChan := make(chan []TransactionInterface, 1)
	pool.GetProposal(types.ConsensusInt(1), retChan)

	ret, ok := <-retChan
	assert.True(t, ok)
	assert.Equal(t, []TransactionInterface{tx}, ret)
}

func TestTransactionListEncode(t *testing.T) {
	allocFunc := func() TransactionInterface { return &TestTx{} }
	txList := &TransactionList{TxAlloc: allocFunc}

	var allTx []TransactionInterface
	for i := 0; i < 100; i++ {
		nxt := &TestTx{i}
		allTx = append(allTx, nxt)
		txList.Items = append(txList.Items, nxt)
	}
	buf := bytes.NewBuffer(nil)
	n, err := txList.Encode(buf)
	assert.True(t, n > 0)
	assert.Nil(t, err)

	newTxList := &TransactionList{TxAlloc: allocFunc}
	nr, err := newTxList.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, n, nr)
	assert.Equal(t, txList.Items, newTxList.Items)
}
