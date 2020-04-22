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
package statemachine

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/statemachine/transactionsm"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"sync/atomic"
	"time"
)

var poolSize = 100
var minProposeSize = 10
var maxProposeSize = 20

type sharedState struct {
	txPool    *transactionsm.TransactionPool
	txAlloc   transactionsm.TxAlloc
	txRetChan chan []transactionsm.TransactionInterface
	randSeed  int64
}

// SimpleTxProposer represents the statemachine object for the SimpleCons protocol, only this state machine can be used
// when testing this object.
type SimpleTxProposer struct {
	AbsStateMachine
	AbsRandSM
	myProposal       []transactionsm.TransactionInterface
	maxTxIdCommitted int // we only propose transactions with larger index than the most recent committed
	startedNext      int32
	prevItem         *SimpleTxProposer
	endChan          chan int // when the proposal thread has finished
	txThread         bool
	*sharedState
}

// NewSimpleTxProposer creates an empty SimpleTxProposer object.
func NewSimpleTxProposer(useRand bool, initRandBytes [32]byte, randSeed int64) *SimpleTxProposer {
	return &SimpleTxProposer{AbsRandSM: NewAbsRandSM(initRandBytes, useRand), endChan: make(chan int, 1),
		sharedState: &sharedState{randSeed: randSeed}}
}

func (spi *SimpleTxProposer) EndTest() {
	spi.absGeneralStateMachine.EndTest()
	spi.txPool.ClosePool()
}

func (spi *SimpleTxProposer) Collect() {
	spi.AbsStateMachine.Collect()
	spi.setDone()
	spi.txPool.Broadcast()
	if spi.txThread {
		<-spi.endChan
		close(spi.endChan)
	}
}

func (spi *SimpleTxProposer) StatsString(testDuration time.Duration) string {
	return fmt.Sprintf("Max id committed: %v, %v", spi.maxTxIdCommitted, spi.txPool.String())
}

// runTxProposeThread is started after decision and proposes txs for the next consensus instance.
// It is stopped when the next consensus instance has finished.
func (spi *SimpleTxProposer) runTxProposeThread() {
	spi.txThread = true
	go func() {
		rand := rand.New(rand.NewSource(spi.randSeed + int64(spi.index.Index.(types.ConsensusInt))))
		for atomic.LoadInt32(&spi.startedNext) == 0 && !spi.txPool.BlockIfFull() { // run until the pool is closed
			if err := spi.SubmitTransaction(spi.maxTxIdCommitted + rand.Intn(1000)); err != nil {
				logging.Info("Error during tx proposal: ", err)
			}
		}
		spi.endChan <- 1
	}()
}

// Init initalizes the simple proposal object state.
func (spi *SimpleTxProposer) Init(gc *generalconfig.GeneralConfig, lastProposal types.ConsensusInt, needsConcurrent types.ConsensusInt,
	mainChannel channelinterface.MainChannel, doneChan chan channelinterface.ChannelCloseType) {

	spi.AbsInit(gc, lastProposal, needsConcurrent, mainChannel, doneChan)
	spi.AbsRandSM.AbsRandInit(gc)

	// spi.proposalMap = make(map[types.ConsensusInt][]transactionsm.TransactionInterface)
	spi.txAlloc = func() transactionsm.TransactionInterface { return &transactionsm.TestTx{} }
	spi.txPool = transactionsm.AllocateTransactionPool(poolSize, minProposeSize, maxProposeSize)
	spi.txRetChan = make(chan []transactionsm.TransactionInterface, 1)
	spi.runTxProposeThread()
}

func (spi *SimpleTxProposer) txValidFunc(tx transactionsm.TransactionInterface) error {
	if tx.(*transactionsm.TestTx).Id <= int(spi.maxTxIdCommitted) {
		return types.ErrInvalidIndex
	}
	return nil
}

// GetInitialState returns []byte("initial state")
func (spi *SimpleTxProposer) GetInitialState() []byte {
	return []byte("initial state")
}

// HasDecided is called after the index nxt has decided.
func (spi *SimpleTxProposer) HasDecided(proposer sig.Pub, nxt types.ConsensusInt, dec []byte) {
	spi.AbsHasDecided(nxt, dec)
	spi.prevItem.setDone()

	if len(dec) != 0 {
		buf := bytes.NewReader(dec)
		var err error
		_, err = spi.RandHasDecided(proposer, buf, true)
		if err != nil {
			logging.Error("invalid mv vrf proof", err)
			// TODO should return here?
		}
		remainLen := buf.Len()

		// Deserialze the transactions
		txList := transactionsm.TransactionList{TxAlloc: spi.txAlloc}
		n, err := txList.Decode(buf)
		if err != nil {
			logging.Error("Error decoding decided transaction list: ", err)
		}
		if n != remainLen {
			logging.Errorf("Only partially decoded transaction list %v of %v bytes", n, len(dec))
		}

		// Store the max committed id
		for _, nxtTx := range txList.Items {
			if int(nxtTx.(*transactionsm.TestTx).Id) > spi.maxTxIdCommitted {
				spi.maxTxIdCommitted = int(nxtTx.(*transactionsm.TestTx).Id)
			}
		}

		// remove any invalid transactions from the pool
		spi.txPool.ValidatePool(spi.txValidFunc)
	}
	// reinsert any valid transactions
	var toReinsert []transactionsm.TransactionInterface
	for _, nxtTx := range spi.myProposal {
		if err := spi.txValidFunc(nxtTx); err == nil {
			toReinsert = append(toReinsert, nxtTx)
		}
	}
	if len(toReinsert) > 0 {
		spi.txPool.SubmitTransaction(toReinsert...)
	}
	// if !spi.IsDone() {
	spi.runTxProposeThread() // We start the propose thread for the next consensus instance
	// }
}

func (spi *SimpleTxProposer) setDone() {
	atomic.StoreInt32(&spi.startedNext, 1)
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (spi *SimpleTxProposer) DoneClear() {
	spi.AbsDoneClear()
	spi.prevItem.setDone()
	spi.setDone()
	spi.prevItem = nil
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (spi *SimpleTxProposer) DoneKeep() {
	spi.AbsDoneKeep()
	if spi.index.Index.(types.ConsensusInt) != 0 {
		spi.prevItem.setDone()
	}
	spi.prevItem = nil
}

func (spi *SimpleTxProposer) SubmitTransaction(txID int) error {
	_, err := spi.txPool.SubmitTransaction(&transactionsm.TestTx{txID})
	if err != nil {
		return err[0]
	}
	return nil
}

var allocFunc = func() transactionsm.TransactionInterface { return &transactionsm.TestTx{} }

// GetProposal is called when a consensus index is ready for a proposal.
// It should send the proposal for the consensus index by calling mainChannel.HasProposal().
func (spi *SimpleTxProposer) GetProposal() {
	// Get the proposal

	// can optionally validate the pool, to remove any transactions from slow proposers
	spi.txPool.ValidatePool(spi.txValidFunc)

	spi.txPool.GetProposal(spi.index.Index, spi.txRetChan)
	// We can call GetProposalNow for the consensus to get a proposal immediately, but it may return an empty proposal.
	// In this benchmark there is always a thread running making proposals, so it is not needed.
	// spi.txPool.GetProposalNow(spi.index.Index)

	proposeTx := <-spi.txRetChan

	// We only propose the valid transactions, we might have some invalid transactions from slow proposers
	validTx := make([]transactionsm.TransactionInterface, 0, len(proposeTx))
	for _, nxt := range proposeTx {
		if err := spi.txValidFunc(nxt); err == nil {
			validTx = append(validTx, nxt)
		}
	}

	writer := bytes.NewBuffer(nil)
	spi.RandGetProposal(writer)

	// Encode the proposal
	txList := transactionsm.TransactionList{validTx, allocFunc}
	_, err := txList.Encode(writer)
	if err != nil {
		panic(err)
	}
	spi.myProposal = validTx

	w := messagetypes.NewMvProposeMessage(spi.index, writer.Bytes())
	if generalconfig.CheckFaulty(spi.GetIndex(), spi.GeneralConfig) {
		logging.Info("get byzantine proposal", spi.GetIndex(), spi.GeneralConfig.TestIndex)
		w.ByzProposal = spi.GetByzProposal(w.Proposal, spi.GeneralConfig)
	}
	spi.AbsGetProposal(w)
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (spi *SimpleTxProposer) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	n := spi.GetRndNumBytes()
	buf := bytes.NewReader(originProposal[n:])
	remainLen := buf.Len()

	// Deserialze the transactions
	txList := transactionsm.TransactionList{TxAlloc: spi.txAlloc}
	nTx, err := txList.Decode(buf)
	if err != nil {
		panic(err)
	}
	if nTx != remainLen {
		panic(fmt.Errorf("only partially decoded transaction list %v of %v bytes", n, len(originProposal)))
	}

	// We remove a tx from the original proposal for the new one
	txList.Items = txList.Items[1:]

	// Create the new proposal
	buff := bytes.NewBuffer(nil)
	// Copy the rand bytes
	if _, err := buff.Write(originProposal[:n]); err != nil {
		panic(err)
	}
	// Encode the txs
	if _, err := txList.Encode(buff); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// ValidateProposal should return true if the input proposal is valid.
func (spi *SimpleTxProposer) ValidateProposal(proposer sig.Pub, dec []byte) error {
	buf := bytes.NewReader(dec)
	_, err := spi.RandHasDecided(proposer, buf, false)
	if err != nil {
		return err
	}
	remainLen := buf.Len()

	// Deserialze the transactions
	txList := transactionsm.TransactionList{TxAlloc: spi.txAlloc}
	n, err := txList.Decode(buf)
	if err != nil {
		return fmt.Errorf("error decoding decided transaction list: %v", err)
	}
	if n != remainLen {
		return fmt.Errorf("only partially decoded transaction list %v of %v bytes", n, len(dec))
	}
	for _, tx := range txList.Items {
		if err := spi.txValidFunc(tx); err != nil {
			logging.Warning("Got an invalid proposal", err, tx.(*transactionsm.TestTx).Id, spi.maxTxIdCommitted)
			return err
		}
	}

	return nil
}

// StartIndex is called when the previous consensus index has finished.
func (spi *SimpleTxProposer) StartIndex(nxt types.ConsensusInt) consinterface.StateMachineInterface {
	ret := &SimpleTxProposer{
		AbsStateMachine:  spi.AbsStateMachine,
		AbsRandSM:        spi.AbsRandSM,
		maxTxIdCommitted: spi.maxTxIdCommitted,
		sharedState:      spi.sharedState,
		endChan:          make(chan int, 1),
		prevItem:         spi}

	ret.AbsStartIndex(nxt)
	return ret
}

// CheckDecisions verifies that the decided values were valid.
func (spi *SimpleTxProposer) CheckDecisions(decs [][]byte) (outOforderErrors, errs []error) {
	var maxTotalId int
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

		var maxLocalId int
		// Deserialze the transactions
		txList := transactionsm.TransactionList{TxAlloc: spi.txAlloc}
		n, err := txList.Decode(buf)
		if err != nil {
			errs = append(errs, fmt.Errorf("error decoding decided transaction list: %v, idx %v", err, i+1))
		}
		if n != remainLen {
			errs = append(errs, fmt.Errorf("only partially decoded transaction list %v of %v bytes, idx %v", n, len(dec), i+1))
		}
		for _, tx := range txList.Items {
			if tx.(*transactionsm.TestTx).Id < maxTotalId {
				outOforderErrors = append(outOforderErrors,
					fmt.Errorf("decided a smaller tx id %v than the previous max %v, idx %v",
						tx.(*transactionsm.TestTx).Id, maxTotalId, i+1))
			}
			maxLocalId = utils.Max(maxLocalId, tx.(*transactionsm.TestTx).Id)
		}
		maxTotalId = utils.Max(maxTotalId, maxLocalId)
	}
	return
}
