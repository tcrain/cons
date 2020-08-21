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
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/copywritemap"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/statemachine"
	"github.com/tcrain/cons/consensus/statemachine/transactionsm"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var poolSize = 100

// var minProposeSize = 10
var maxProposeSize = 20

type rwOverride struct {
	mutex sync.RWMutex
}

func (rw *rwOverride) Lock() {
	rw.mutex.RLock()
}

func (rw *rwOverride) Unlock() {
	rw.mutex.RUnlock()
}

type currencyGlobalState struct {
	committedAccTable  *AccountTable      // The account table of committed transactions
	lastCommittedIndex types.ConsensusInt // The index of the most recent committed
	donePropose        []chan bool        // Closed when propose threads exit

	proposeThreadAccTable *AccountTable      // The account table used by the propose thread
	proposeThreadIndex    types.ConsensusInt // The account table index used by the propose thread
	tryPropose            uint32             // if > 0 then should try to propose
}

// SimpleCurrencyTxProposer represents the statemachine object for the SimpleCons protocol, only this state machine can be used
// when testing this object.
type SimpleCurrencyTxProposer struct {
	statemachine.AbsStateMachine
	statemachine.AbsRandSM
	myKeys         []sig.Priv
	txPool         *transactionsm.TransactionPool
	initBlockBytes [][]byte

	accTable    *AccountTable
	globalState *currencyGlobalState

	myProposals  []*TransferTx
	txRetChan    chan []transactionsm.TransactionInterface
	initPubs     []sig.Pub
	initialState []byte

	hashes         *copywritemap.CopyWriteMap // map[types.ConsensusInt]messages.HashBytes
	lastBlockIndex types.ConsensusInt

	validFunc func(transactionsm.TransactionInterface) error

	newPubFunc   func() sig.Pub
	newSigFunc   func() sig.Sig
	rand         *rand.Rand
	accountMutex *sync.RWMutex

	submitMutex *rwOverride
	submitCond  *sync.Cond
}

var randSeed int64 = 1234

// NewSimpleTxProposer creates an empty SimpleCurrencyTxProposer object.
func NewSimpleTxProposer(useRand bool, initRandBytes [32]byte,
	initBlocksBytes [][]byte, myKeys []sig.Priv,
	newPubFunc func() sig.Pub, newSigFunc func() sig.Sig) *SimpleCurrencyTxProposer {
	ret := &SimpleCurrencyTxProposer{
		AbsRandSM:  statemachine.NewAbsRandSM(initRandBytes, useRand),
		txPool:     transactionsm.AllocateTransactionPool(poolSize, len(myKeys), maxProposeSize),
		txRetChan:  make(chan []transactionsm.TransactionInterface, 1),
		hashes:     copywritemap.NewCopyWriteMap(), // make(map[types.ConsensusInt]messages.HashBytes),
		myKeys:     myKeys,
		accTable:   NewAccountTable(),
		newPubFunc: newPubFunc,
		newSigFunc: newSigFunc}

	ret.globalState = &currencyGlobalState{
		committedAccTable:     ret.accTable,
		proposeThreadAccTable: ret.accTable}
	ret.submitMutex = &rwOverride{}
	ret.accountMutex = &sync.RWMutex{}
	ret.submitCond = sync.NewCond(ret.submitMutex)
	ret.initBlockBytes = initBlocksBytes

	buf := bytes.NewBuffer(nil)
	for _, initBlockBytes := range initBlocksBytes {
		if len(initBlockBytes) == 0 {
			continue // non participant block
		}
		initBlock := InitBlock{NewPubFunc: newPubFunc}
		tmpBuf := bytes.NewReader(initBlockBytes)
		if _, err := initBlock.Decode(tmpBuf); err != nil {
			panic(err)
		}

		if _, err := initBlock.Encode(buf); err != nil {
			panic(err)
		}

		ret.initPubs = append(ret.initPubs, initBlock.InitPubs...)
		for i, pub := range initBlock.InitPubs {
			acc, err := ret.accTable.GetAccount(pub)
			if err != nil {
				panic(err)
			}
			acc.Balance = initBlock.InitBalances[i]
			ret.accTable.UpdateAccount(pub, acc)
		}
	}
	ret.initialState = buf.Bytes()
	ret.hashes.Write(types.ConsensusInt(0), types.GetHash(ret.initialState))

	ret.validFunc = func(tx transactionsm.TransactionInterface) error {
		ret.accountMutex.RLock()
		err := ret.globalState.proposeThreadAccTable.ValidateTx(tx.(*TransferTx), true, nil)
		ret.accountMutex.RUnlock()
		return err
	}

	return ret
}

func (spi *SimpleCurrencyTxProposer) StatsString(testDuration time.Duration) string {
	return fmt.Sprintf("\n%v\nAccount stats: %v\nPool stats: %v\n%v", spi.accTable.stats.StatsString(testDuration),
		spi.accTable.stats.String(), spi.txPool.String(), spi.accTable.String())
}

func (spi *SimpleCurrencyTxProposer) runTxProposeThread(idx int, privKey sig.Priv) chan bool {
	if privKey == nil {
		return nil
	}
	doneChan := make(chan bool, 1)
	rnd := rand.New(rand.NewSource(randSeed + int64(idx)))
	go func() {
		spi.submitMutex.Lock()
		var tx *TransferTx
		for true {

			// our transactions are just sending 1 value to a random pub from the initial list
			sender := []sig.Priv{privKey}
			amount := []uint64{1}
			recAmount := []uint64{1}
			pkidx := rnd.Intn(len(spi.initPubs))
			recipiant := []sig.Pub{spi.initPubs[pkidx]}

			atomic.StoreUint32(&spi.globalState.tryPropose, 0)

			var err error
			spi.accountMutex.RLock()
			if tx == nil || spi.globalState.proposeThreadAccTable.ValidateTx(tx, false, nil) != nil {
				tx, err = spi.globalState.proposeThreadAccTable.GenerateTx(sender, amount, recipiant, recAmount)
			}
			spi.accountMutex.RUnlock()

			if err == nil {
				err = spi.SubmitTransaction(tx)
			} else {
				logging.Error("error creating tx", err, spi.globalState.proposeThreadIndex, spi.GeneralConfig.TestIndex)
			}

			if spi.txPool.IsClosed() {
				close(doneChan)
				return
			}

			if err != nil && atomic.LoadUint32(&spi.globalState.tryPropose) == 0 {
				spi.submitCond.Wait()
				// time.Sleep(10*time.Millisecond)
			}
		}
		spi.submitMutex.Unlock()

	}()

	return doneChan
}

// Init initalizes the simple proposal object state.
func (spi *SimpleCurrencyTxProposer) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
	spi.AbsRandSM.AbsRandInit(gc)
	for i, priv := range spi.myKeys {
		if c := spi.runTxProposeThread(gc.TestIndex+i, priv); c != nil {
			spi.globalState.donePropose = append(spi.globalState.donePropose, c)
		}
	}
	// spi.runTxProposeThread()
}

// GetInitialState returns []byte("initial state")
func (spi *SimpleCurrencyTxProposer) GetInitialState() []byte {
	return spi.initialState
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *SimpleCurrencyTxProposer) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	n := spi.GetRndNumBytes()

	var validBlock bool
	var txBlock *TxBlock
	var err error
	validBlock, txBlock, err = spi.deserializeBlock(bytes.NewReader(originProposal[n:]))
	if err != nil {
		panic(err)
	}
	if !validBlock {
		panic(types.ErrInvalidProposal)
	}

	// We send 1 extra value on the first tx
	tx := txBlock.Transactions[0]
	for i := range tx.SendAmounts {
		tx.SendAmounts[i] += 1
	}

	// Get the sender so we can sign the modified tx
	senders := make([]sig.Priv, len(tx.Senders))
	for i, snd := range tx.Senders {
		for _, nxt := range spi.myKeys {
			if sig.CheckPubsEqual(snd, nxt.GetPub()) {
				senders[i] = nxt
			}
		}
	}

	// Generate the new tx
	spi.accountMutex.Lock()
	tx, err = spi.accTable.GenerateTx(senders, tx.SendAmounts, tx.Recipiants, tx.ReceiveAmounts)
	spi.accountMutex.Unlock()
	if err != nil {
		panic(err) // TODO don't panic here since this can happen if not enough money, leave it now for testing
	}
	txBlock.Transactions[0] = tx

	// Create the new proposal
	buff := bytes.NewBuffer(nil)
	// Copy the rand bytes
	if _, err := buff.Write(originProposal[:n]); err != nil {
		panic(err)
	}
	// DoEncode the txs
	if _, err := txBlock.Encode(buff); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *SimpleCurrencyTxProposer) ValidateProposal(proposer sig.Pub, dec []byte) (err error) {
	buf := bytes.NewReader(dec)
	_, err = spi.RandHasDecided(proposer, buf, false)
	if err != nil {
		logging.Error("invalid mv vrf proof", err)
		return err
	}
	var validBlock bool
	var txBlock *TxBlock
	validBlock, txBlock, err = spi.deserializeBlock(buf)
	if err != nil {
		return err
	}
	if !validBlock {
		return types.ErrInvalidProposal
	}
	// Validate the transactions
	spi.accountMutex.Lock()
	logging.Infof("block size %v index %v\n", len(txBlock.Transactions), spi.GetIndex())
	for _, nxtTx := range txBlock.Transactions {
		if err = spi.accTable.ValidateTx(nxtTx, true, nil); err != nil {
			logging.Warning("Invalid tx invalidation", err, spi.GetIndex().Index, spi.GeneralConfig.TestIndex)
			break
		}
	}
	spi.accountMutex.Unlock()
	return
}

func (spi *SimpleCurrencyTxProposer) deserializeBlock(buf *bytes.Reader) (validBlock bool,
	txBlock *TxBlock, err error) {

	// Deserialze the transactions

	remainLen := buf.Len()
	validBlock = true
	txBlock = &TxBlock{NewPubFunc: spi.newPubFunc, NewSigFunc: spi.newSigFunc}
	n, err := txBlock.Decode(buf)
	if err != nil {
		logging.Error("Error decoding decided transaction list: ", err)
		validBlock = false
	}
	if n != remainLen {
		logging.Errorf("Only partially decoded transaction list %v of %v bytes", n, remainLen)
		validBlock = false
	}

	if txBlock.Index != spi.lastBlockIndex+1 {
		logging.Errorf("Got out of order block, expected index: %v, got %v", txBlock.Index, spi.lastBlockIndex+1)
		validBlock = false
	} else {
		ph, _ := spi.hashes.Read(spi.lastBlockIndex)
		if !bytes.Equal(ph.(types.HashBytes), txBlock.PrevHash) {
			logging.Errorf("Got invalid previous hash of block")
			validBlock = false
		}
	}

	return
}

// HasDecided is called after the index nxt has decided.
func (spi *SimpleCurrencyTxProposer) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, dec []byte) {
	spi.AbsHasDecided(nxt, dec)

	// Deserialze the transactions
	var validBlock bool
	var txBlock *TxBlock
	if len(dec) != 0 {
		buf := bytes.NewReader(dec)
		var err error
		_, err = spi.RandHasDecided(proposer, buf, true)
		if err != nil {
			logging.Error("invalid mv vrf proof", err)
		}
		validBlock, txBlock, _ = spi.deserializeBlock(buf)
	}

	if validBlock {
		if txBlock == nil { // sanity check
			panic("shouldn't be nil")
		}
		spi.lastBlockIndex += 1
		spi.hashes.Write(spi.lastBlockIndex, types.GetHash(dec))

		// Validate and consume the transactions
		spi.accountMutex.Lock()
		logging.Infof("block size %v index %v\n", len(txBlock.Transactions), nxt)
		for _, nxtTx := range txBlock.Transactions {
			if err := spi.accTable.ValidateTx(nxtTx, true, nil); err == nil {
				spi.accTable.ConsumeTx(nxtTx, nil)
			} else {
				logging.Warning(err)
			}
		}
		spi.accountMutex.Unlock()

		spi.accountMutex.Lock()
		// Update the propose threads pointers
		spi.globalState.proposeThreadAccTable = spi.accTable
		spi.globalState.proposeThreadIndex = spi.GetIndex().Index.(types.ConsensusInt)

		spi.accountMutex.Unlock()

		// remove any invalid transactions from the pool
		spi.validateAndRestartPropose()
	}

	// reinsert any valid transactions
	spi.accountMutex.RLock()
	var toReinsert []transactionsm.TransactionInterface
	for _, nxtTx := range spi.myProposals {
		if nxtTx == nil && spi.accTable.ValidateTx(nxtTx, false, nil) == nil {
			toReinsert = append(toReinsert, nxtTx)
		}
	}
	spi.accountMutex.RUnlock()

	if len(toReinsert) > 0 {
		spi.txPool.SubmitTransaction(toReinsert...)
	}

	// wake any waiters
	spi.wakeProposeThread()
}

func (spi *SimpleCurrencyTxProposer) wakeProposeThread() {
	atomic.StoreUint32(&spi.globalState.tryPropose, 1)
	spi.submitCond.Broadcast()
}

// Submits a transaction to the state machine.
func (spi *SimpleCurrencyTxProposer) SubmitTransaction(tx *TransferTx) error {
	if _, err := spi.txPool.SubmitTransactionIfValid(spi.validFunc, tx); err != nil {
		return err[0]
	}
	return nil
}

// var allocFunc = func() transactionsm.TransactionInterface { return &transactionsm.TestTx{} }

// StartIndex is called when a consensus index is ready for a proposal.
// StartIndex is called when the previous consensus index has finished.
func (spi *SimpleCurrencyTxProposer) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &SimpleCurrencyTxProposer{}
	*ret = *spi
	ret.hashes = ret.hashes.Copy()
	ret.accountMutex.Lock()
	ret.accTable = ret.accTable.Copy()
	ret.accountMutex.Unlock()
	ret.myProposals = nil

	ret.AbsStartIndex(nxt)
	return ret
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *SimpleCurrencyTxProposer) DoneClear() {
	spi.AbsDoneClear()
	spi.accountMutex.Lock()

	// In case the proposals are using this account table, then set it to the previously committed index
	if spi.globalState.proposeThreadAccTable == spi.accTable {
		spi.globalState.proposeThreadAccTable = spi.globalState.committedAccTable
		spi.globalState.proposeThreadIndex = spi.globalState.lastCommittedIndex

		// validate and restart tx propose pool
		spi.accountMutex.Unlock()
		spi.validateAndRestartPropose()
		spi.accountMutex.Lock()
	}

	spi.accTable.DoneClear()
	spi.hashes.DoneClear()
	spi.accountMutex.Unlock()
}

func (spi *SimpleCurrencyTxProposer) validateAndRestartPropose() {
	spi.accountMutex.RLock() // The pool is concurrency safe so we only need to lock the account table for reads
	// remove any invalid transactions from the pool
	spi.txPool.ValidatePool(func(tx transactionsm.TransactionInterface) error {
		return spi.accTable.ValidateTx(tx.(*TransferTx), false, nil)
	})
	spi.accountMutex.RUnlock()

	// wake any waiters
	spi.wakeProposeThread()
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *SimpleCurrencyTxProposer) DoneKeep() {
	spi.AbsDoneKeep()
	spi.accountMutex.Lock()

	if spi.globalState.proposeThreadIndex > spi.GetIndex().Index.(types.ConsensusInt) ||
		spi.globalState.proposeThreadAccTable == spi.accTable {
		// do nothing
	} else {
		spi.globalState.proposeThreadAccTable = spi.accTable
		spi.globalState.proposeThreadIndex = spi.GetIndex().Index.(types.ConsensusInt)

		// validate and restart tx propose pool
		spi.accountMutex.Unlock()
		spi.validateAndRestartPropose()
		spi.accountMutex.Lock()
	}
	spi.globalState.committedAccTable = spi.accTable
	spi.globalState.lastCommittedIndex = spi.GetIndex().Index.(types.ConsensusInt)

	spi.accTable.DoneKeep()
	spi.hashes.DoneKeep()
	spi.accountMutex.Unlock()
}

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *SimpleCurrencyTxProposer) GetProposal() {

	// In case the proposals are using this account table, then set it to the previously committed index
	if spi.globalState.proposeThreadAccTable != spi.accTable {
		// validate and restart tx propose pool
		spi.accountMutex.Lock()
		spi.globalState.proposeThreadAccTable = spi.accTable
		spi.globalState.proposeThreadIndex = spi.GetIndex().Index.(types.ConsensusInt)
		spi.accountMutex.Unlock()

		spi.validateAndRestartPropose()
	}

	// Get the proposal
	spi.txPool.GetProposal(spi.GetIndex().Index, spi.txRetChan)
	// spi.txPool.GetProposalNow(spi.GetIndex())

	logging.Info("get proposal", spi.GetIndex())
	var proposeTx []transactionsm.TransactionInterface
	select {
	case proposeTx = <-spi.txRetChan:
	case <-time.After(100 * time.Second):
		panic(1) // TODO remove this, just for testing
	}
	logging.Info("got proposal", spi.GetIndex())

	txs := make([]*TransferTx, len(proposeTx))
	for i, tx := range proposeTx {
		txs[i] = tx.(*TransferTx)
	}
	spi.myProposals = txs

	// DoEncode the proposal
	ph, _ := spi.hashes.Read(spi.lastBlockIndex)
	txBlock := &TxBlock{Index: spi.lastBlockIndex + 1,
		PrevHash:     ph.(types.HashBytes),
		Transactions: txs,
	}

	writer := bytes.NewBuffer(nil)
	spi.RandGetProposal(writer)

	_, err := txBlock.Encode(writer)
	if err != nil {
		panic(err)
	}

	w := messagetypes.NewMvProposeMessage(spi.GetIndex(), writer.Bytes())
	if generalconfig.CheckFaulty(spi.GetIndex(), spi.GeneralConfig) {
		logging.Info("get byzantine proposal", spi.GetIndex(), spi.GeneralConfig.TestIndex)
		w.ByzProposal = spi.GetByzProposal(w.Proposal, spi.GeneralConfig)
	}
	spi.AbsGetProposal(w)
}

// CheckDecisions verifies that the decided values were valid.
func (spi *SimpleCurrencyTxProposer) CheckDecisions(decs [][]byte) (outOforderErrors, errs []error) {
	var prevBlockIdx types.ConsensusInt
	ph, _ := spi.hashes.Read(types.ConsensusInt(0))
	prevHash := ph.(types.HashBytes)

	// Allocate a blank account table
	var a [32]byte
	newSpi := NewSimpleTxProposer(false, a, spi.initBlockBytes, spi.myKeys,
		spi.newPubFunc, spi.newSigFunc)
	accTable := newSpi.accTable

	for i, dec := range decs {
		if len(dec) == 0 {
			continue
		}
		buf := bytes.NewReader(dec)
		err := spi.RandCheckDecision(buf)
		if err != nil {
			errs = append(errs, err)
		}
		remainLen := buf.Len()

		// Deserialze the transactions
		txBlock := TxBlock{NewPubFunc: spi.newPubFunc, NewSigFunc: spi.newSigFunc}
		n, err := txBlock.Decode(buf)
		if err != nil {
			errs = append(errs, fmt.Errorf("error decoding decided transaction list: %v, idx %v", err, i))
		}
		if n != remainLen {
			errs = append(errs, fmt.Errorf("only partially decoded transaction list %v of %v bytes, idx %v", n, len(dec), i))
		}
		if txBlock.Index != prevBlockIdx+1 {
			errs = append(errs, fmt.Errorf("got out of order block, expected index: %v, got %v", txBlock.Index, spi.lastBlockIndex+1))
		}
		if !bytes.Equal(prevHash, txBlock.PrevHash) {
			errs = append(errs, fmt.Errorf("got invalid prev hash at index %v", i))
		}
		prevHash = types.GetHash(dec)
		prevBlockIdx = txBlock.Index

		for _, tx := range txBlock.Transactions {
			if err := accTable.ValidateTx(tx, true, nil); err != nil {
				outOforderErrors = append(outOforderErrors, err)
			} else {
				accTable.ConsumeTx(tx, nil)
			}
		}
	}

	return
}

func (spi *SimpleCurrencyTxProposer) EndTest() {
	spi.AbsStateMachine.EndTest()
	spi.txPool.ClosePool()
	spi.submitCond.Broadcast()
	// this returns when closed
	for _, nxt := range spi.globalState.donePropose {
		_ = <-nxt
	}
}
