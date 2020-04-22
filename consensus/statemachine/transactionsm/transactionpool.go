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
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sync"
)

type ValidateTxFunc func(TransactionInterface) error

type PoolStats struct {
	Inserted         uint64
	FailedInsert     uint64
	Proposed         uint64
	FailedValidation uint64
	Validated        uint64
}

func (ps PoolStats) String() string {
	return fmt.Sprintf("Proposed %v, Inserted %v, Failed inserted %v, ValidatedItem %v, Failed validation: %v",
		ps.Proposed, ps.Inserted, ps.FailedInsert, ps.Validated, ps.FailedValidation)
}

func (tp *TransactionPool) String() string {
	return tp.stats.String()
}

type TransactionPool struct {
	// Minimum number of transactions in a proposal
	minProposalSize int
	// Maximum number of transactions in a proposal
	maxProposalSize int
	// Maximum number of unsubmitted transactions
	maxPoolSize int
	// Transactions in the pool, but not submitted
	transactions []TransactionInterface
	// Transactions both in the pool and submitted
	transactionMap map[ConflictObject]TransactionInterface
	// current number of transactions in pool
	currentPoolSize int
	// Proposals that haven't been sent yet
	pendingProposals map[types.ConsensusID]chan []TransactionInterface
	// Sorted list of pending proposal indicies
	minPendingIndex types.SortConsIndex
	// When closed is set to true should unblock any waiters
	closed bool
	// Statistics
	stats PoolStats
	mutex sync.Mutex
	cond  *sync.Cond
}

func AllocateTransactionPool(maxPoolSize int, minProposalSize, maxProposalSize int) *TransactionPool {
	ret := &TransactionPool{
		minProposalSize:  minProposalSize,
		maxProposalSize:  maxProposalSize,
		maxPoolSize:      maxPoolSize,
		transactionMap:   make(map[ConflictObject]TransactionInterface),
		pendingProposals: make(map[types.ConsensusID]chan []TransactionInterface),
	}
	ret.cond = sync.NewCond(&ret.mutex)
	return ret
}

// Calls SubmitTransactionIfValid with a nil valid function.
func (tp *TransactionPool) SubmitTransaction(
	transList ...TransactionInterface) (inserted []TransactionInterface, errs []error) {

	return tp.SubmitTransactionIfValid(nil, transList...)
}

// SubmitTransaction is used to submit a transaction into the pool, it is safe to be called concurrently
// from multiple threads. ValidFunc is called after the pool checks for a conflicting transaction for each
// transaction, if validFunc is nil then all non-conflicting transactions are considered valid.
// It returns nil if the transaction was successfully inserted into the pool.
func (tp *TransactionPool) SubmitTransactionIfValid(validFunc ValidateTxFunc,
	transList ...TransactionInterface) (inserted []TransactionInterface, errs []error) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	for _, trans := range transList {
		if tp.currentPoolSize >= tp.maxPoolSize {
			errs = append(errs, ErrTransactionPoolFull)
			tp.stats.FailedInsert++
			return
		}
		cos := trans.GetConflictObjects()
		var err error
		for _, co := range cos {
			if _, ok := tp.transactionMap[co]; ok {
				err = ErrTransactionAreadyInPool
				tp.stats.FailedInsert++
				break
			}
		}

		if err == nil {
			var err error
			if validFunc != nil {
				err = validFunc(trans)
			}
			if err == nil {
				for _, co := range cos {
					tp.transactionMap[co] = trans
				}
				tp.transactions = append(tp.transactions, trans)

				tp.currentPoolSize++
				tp.stats.Inserted++
				inserted = append(inserted, trans)
			} else {
				tp.stats.FailedInsert++
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}
	// if we have enough transactions and a waiting proposal, then send it
	for tp.currentPoolSize >= tp.minProposalSize {
		if len(tp.pendingProposals) > 0 {
			idx := tp.minPendingIndex[0]
			logging.Info("Sending proposal for idx after insert", idx)
			tp.sendTx(idx, tp.pendingProposals[idx])
			delete(tp.pendingProposals, idx)
			tp.minPendingIndex = tp.minPendingIndex[1:]
		} else {
			break
		}
	}
	return
}

// GetProposal is called when the consensus index idx is ready for a proposal.
// Should only be called once per index.
func (tp *TransactionPool) GetProposal(idx types.ConsensusID,
	retChan chan []TransactionInterface) {

	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.currentPoolSize >= tp.minProposalSize || tp.closed {
		logging.Info("Sending proposal for idx", idx)
		tp.sendTx(idx, retChan)
	} else {
		logging.Info("Asked for proposal, but not enough tx (have %v, need %v), idx %v",
			tp.currentPoolSize, tp.minProposalSize, idx)
		tp.pendingProposals[idx] = retChan
		tp.minPendingIndex = append(tp.minPendingIndex, idx.(types.ConsensusInt))
		tp.minPendingIndex.Sort()
	}
}

// Validate pool will check the transactions in the pool against txFunc,
// any that return false will be removed from the pool.
func (tp *TransactionPool) ValidatePool(txFunc ValidateTxFunc) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	firstNonNil := len(tp.transactions)
	for i, tx := range tp.transactions {
		if tx == nil {
			continue
		}
		if txFunc(tx) != nil {
			tp.transactions[i] = nil
			tp.deleteTxFromMap(tx)
			tp.stats.FailedValidation++
			tp.currentPoolSize--
		} else {
			if i < firstNonNil {
				firstNonNil = i
			}
			tp.stats.Validated++
		}
	}
	tp.transactions = tp.transactions[firstNonNil:]
	tp.checkWakeUp()
}

func (tp *TransactionPool) deleteTxFromMap(tx TransactionInterface) {
	cos := tx.GetConflictObjects()
	for _, co := range cos {
		delete(tp.transactionMap, co)
	}
}

func (tp *TransactionPool) checkWakeUp() {
	if tp.currentPoolSize < tp.maxPoolSize {
		tp.cond.Broadcast()
	}
}

func (tp *TransactionPool) IsClosed() bool {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	return tp.closed
}

func (tp *TransactionPool) ClosePool() {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	tp.closed = true
	tp.cond.Broadcast()

	for idx, retChan := range tp.pendingProposals {
		logging.Info("Sending immediate proposal for idx %v", idx)
		tp.sendTx(idx, retChan)
		delete(tp.pendingProposals, idx)
	}
}

func (tp *TransactionPool) Broadcast() {
	tp.cond.Broadcast()
}

func (tp *TransactionPool) BlockIfFull() bool {
	tp.mutex.Lock()
	if tp.currentPoolSize >= tp.maxPoolSize && !tp.closed {
		tp.cond.Wait()
	}
	closed := tp.closed
	tp.mutex.Unlock()
	return closed
}

func (tp *TransactionPool) BlockUntilNotFull() bool {
	tp.mutex.Lock()
	for tp.currentPoolSize >= tp.maxPoolSize && !tp.closed {
		tp.cond.Wait()
	}
	closed := tp.closed
	tp.mutex.Unlock()
	return closed
}

func (tp *TransactionPool) sendTx(idx types.ConsensusID, retChan chan []TransactionInterface) {
	// txCount := utils.Min(tp.currentPoolSize, tp.maxProposalSize)
	proposal := make([]TransactionInterface, 0, utils.Min(tp.currentPoolSize, tp.maxProposalSize))

	var i int
	var tx TransactionInterface
	for i = 0; i < len(tp.transactions); i++ {
		if len(proposal) == tp.maxProposalSize {
			break
		}
		tx = tp.transactions[i]
		if tx == nil {
			continue
		}
		proposal = append(proposal, tx)
		tp.deleteTxFromMap(tx)
	}
	tp.transactions = tp.transactions[i:]
	tp.currentPoolSize -= len(proposal)
	tp.stats.Proposed += uint64(len(proposal))
	logging.Info("Sending proposal size %v for idx %v", len(proposal), idx)
	// let any waiters know that there is room in the pool
	tp.cond.Broadcast()
	logging.Info("proposal size", len(proposal))
	retChan <- proposal
}

// GetProposalNow immediately returns a proposal for index idx even if less than minProposalSize
// transactions are ready. Must be called at most once, after GetProposal for the same index.
func (tp *TransactionPool) GetProposalNow(idx types.ConsensusID) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if retChan, ok := tp.pendingProposals[idx]; ok {
		logging.Info("Sending immediate proposal for idx %v", idx)
		tp.sendTx(idx, retChan)
		delete(tp.pendingProposals, idx)
	} else {
		logging.Info("Called send immediate, but already sent", idx)
	}
}
