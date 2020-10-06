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
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"sort"
	"sync"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/utils"
)

type ConsStateInterface interface {
	GetMemberChecker(cid types.ConsensusIndex) (*ConsInterfaceItems, error)
	// GetConsItem(idIdx types.ConsensusIndex) (ConsItem, error)
	GetNewPub() sig.Pub
	GetGeneralConfig() *generalconfig.GeneralConfig
	CheckGenMemberChecker(cid types.ConsensusIndex) (*ConsInterfaceItems, error)
	GetConsensusIndexFuncs() types.ConsensusIndexFuncs
	SetMainChannel(mainChannel channelinterface.MainChannel)
}

// ConsInterfaceState creates, keeps and deletes the member checkers, the message state object, and the forward checker
// objects for the current (generalconfig.KeepTotal) consensus instances.
// Objects for older consensus objects are garbage collected, and their state must be loaded from storage if they are needed.
type ConsInterfaceState struct {

	// StartedIndex       types.ConsensusInt                                // The maximum consensus index that has been started
	LocalIndex       types.ConsensusInt // The current consensus index.
	PredecisionIndex types.ConsensusInt // The maximum consensus index that has gotten a decided or possible decided value
	gcUpTo           types.ConsensusInt // the largest index garbage collected
	allowConcurrent  types.ConsensusInt // Number of concurrent consensus instances allowed to be run
	StartedIndex     types.ConsensusInt // The maximum consensus index that has been started

	firstItem types.ConsensusInt // The minimum consensus index not yet garbage collected.
	lastItem  types.ConsensusInt // The maximum consensus index allocated.
	// itemMap map[types.ConsensusInt]ConsItem      // Array of consensus items for different consensus instances.
	initItem ConsItem // Consensus item used to generate all new consensus items.

	gc          *generalconfig.GeneralConfig
	mainChannel channelinterface.MainChannel

	consItemsMap map[types.ConsensusInt]*ConsInterfaceItems
	//messageStates            []messagestate.MessageState     // the message states
	//forwardCheckers          [generalconfig.KeepTotal]forwardchecker.ForwardChecker // the forward checkers
	initMemberChecker        MemberChecker           // New is called on this object to allocate all member checkers
	initSpecialMemberChecker SpecialPubMemberChecker // New is called on this object to allocate all special member checkers
	initMessageState         MessageState            // New is called on this object to allocate all message state objects
	initForwardChecker       ForwardChecker          // New is called on this object to allocate all forard checkers
	initSM                   StateMachineInterface

	PreDecisions        map[types.ConsensusInt][]byte
	SupportIndex        map[types.ConsensusInt]types.ConsensusInt
	ProposalInfo        map[types.ConsensusInt]StateMachineInterface      // Interface to the state machine.
	GotProposal         map[types.ConsensusInt]bool                       // set to true if received a proposal for the index
	ProposalsToValidate map[types.ConsensusInt][]*channelinterface.RcvMsg // set of proposals to be validated

	lastMCInputIndex types.ConsensusInt // Largest index where a decision was last input into a member checker

	mutex         sync.Mutex // concurrency control
	emptyPub      sig.Pub    // Will be used to create empty public key objects
	broadcastFunc ByzBroadcastFunc

	lastNonNilDecs  []types.ConsensusInt // indecies of non-nil decisions (for garbage collection)
	collectedIndex  types.ConsensusInt   // garbage collected up to this index
	IsInStorageInit bool                 // Used during initialization, when recovering from disk this is set to true so we don't send messages when we reply to values received.
	genRandBytes    bool                 // Only works if GenRandBytes is ture. This uses NumTotalProcs and NumNonMembers to randomly decide which nodes will participate.

	SharedLock sync.Mutex // Shared with cons item
}

type ConsInterfaceItems struct {
	ConsItem   ConsItem // The consensus item.
	MC         *MemCheckers
	MsgState   MessageState
	FwdChecker ForwardChecker
	// LastProgress is used with causal, and is set to the time the index last made progress
	LastProgress time.Time
	// ProgressListIndex is the index of the item in the list sorted by LastProgress for causal.
	ProgressListIndex int
}

type SortedTimeConsInterfaceItems []*ConsInterfaceItems

// Len is the number of elements in the collection.
func (si SortedTimeConsInterfaceItems) Len() int {
	return len(si)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (si SortedTimeConsInterfaceItems) Less(i, j int) bool {
	return si[i].LastProgress.Before(si[j].LastProgress)
}

// Swap swaps the elements with indexes i and j.
func (si SortedTimeConsInterfaceItems) Swap(i, j int) {
	si[i], si[j] = si[j], si[i]
}

// Init updates the fields of the member checker state
func NewConsInterfaceState(initItem ConsItem,
	initMemberChecker MemberChecker,
	initSpecialMemberChecker SpecialPubMemberChecker,
	initMessageState MessageState,
	initForwardChecker ForwardChecker,
	allowConcurrent types.ConsensusInt,
	genRandBytes bool,
	emptyPub sig.Pub, broadcastFunc ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) *ConsInterfaceState {

	mcs := &ConsInterfaceState{}
	mcs.initItem = initItem
	mcs.broadcastFunc = broadcastFunc
	mcs.allowConcurrent = allowConcurrent
	mcs.genRandBytes = genRandBytes
	mcs.consItemsMap = make(map[types.ConsensusInt]*ConsInterfaceItems)
	// mcs.itemMap[1] = mcs.initItem.GenerateNewItem(1, memChecker, MsgState, FwdChecker, mainChannel, nil, cs.generalConfig)
	mcs.LocalIndex = 1
	mcs.PredecisionIndex = 1
	mcs.StartedIndex = 1
	mcs.firstItem = 1
	mcs.lastItem = 1
	mcs.gcUpTo = 1
	mcs.lastMCInputIndex = 1

	mcs.initMemberChecker = initMemberChecker
	mcs.initSpecialMemberChecker = initSpecialMemberChecker
	mcs.initMessageState = initMessageState
	mcs.initForwardChecker = initForwardChecker
	mcs.emptyPub = emptyPub
	mcs.gc = gc

	mcs.ProposalInfo = make(map[types.ConsensusInt]StateMachineInterface)
	mcs.PreDecisions = make(map[types.ConsensusInt][]byte)
	mcs.SupportIndex = make(map[types.ConsensusInt]types.ConsensusInt)
	mcs.GotProposal = make(map[types.ConsensusInt]bool)
	mcs.ProposalsToValidate = make(map[types.ConsensusInt][]*channelinterface.RcvMsg)

	return mcs
}

func (mcs *ConsInterfaceState) SetInitSM(initStateMachine StateMachineInterface) {
	mcs.initSM = initStateMachine
	mcs.ProposalInfo[0] = initStateMachine
	mcs.ProposalInfo[1] = initStateMachine.StartIndex(1)
	initStateMachine.DoneKeep()
}

// GetGeneralConfig returns the GeneralConfig object.
func (mcs *ConsInterfaceState) GetGeneralConfig() *generalconfig.GeneralConfig {
	return mcs.gc
}

func (mcs *ConsInterfaceState) IncrementStartedIndex() {
	mcs.mutex.Lock()
	mcs.StartedIndex++
	mcs.mutex.Unlock()
}

// GetConsIDUnmarFunc returns the function used to unmarshal consensus IDs.
func (mcs *ConsInterfaceState) GetConsensusIndexFuncs() types.ConsensusIndexFuncs {
	return types.IntIndexFuns
}

// CheckValidateProposal checks if item is a proposal that needs to be validated by the SM.
// If it is not true, nil is returned.
// It it is, but the state machine is not ready to validate it then false, nil is returned.
// (This object will take care of reprocessing the message in this case.)
// If it is ready then true is returned.
// If there is an error processing the message then an error is returned.
// This should only be called from the main message thread.
func (mcs *ConsInterfaceState) CheckValidateProposal(item *deserialized.DeserializedItem,
	sendRecvChan *channelinterface.SendRecvChannel, isLocal bool) (readyToProcess bool, err error) {

	if msm, ok := item.Header.(*sig.MultipleSignedMessage); ok { // is this a signed message?

		// TODO ("check unsigend messages")

		supportIdx, proposal, err := msm.NeedsSMValidation(item.Index, 0)
		if err != nil {
			return false, err
		}
		if proposal != nil { // does this proposal need validation?
			idx := item.Index.Index.(types.ConsensusInt)
			supIdx := supportIdx.Index.(types.ConsensusInt)
			if supIdx >= idx { // invalid support index
				return true, types.ErrInvalidIndex
			}

			if mcs.allowConcurrent > 0 { // if we are using concurrency then our support index is just the smaller value
				supIdx = utils.MaxConsensusIndex(mcs.LocalIndex-1,
					types.ConsensusInt(utils.SubOrZero(uint64(idx), uint64(mcs.allowConcurrent))))
			}
			pi, ok := mcs.ProposalInfo[supIdx]
			if supIdx != 0 && (!ok || !pi.GetDecided()) { // pi.GetDone() == types.NotDone { // the support index state machine is not ready to validate
				logging.Warningf("need to validate proposal later support idx %v, idx %v, decided %v", supIdx, item.Index, mcs.LocalIndex)
				mcs.ProposalsToValidate[supIdx] = append(mcs.ProposalsToValidate[supIdx],
					&channelinterface.RcvMsg{SendRecvChan: sendRecvChan, IsLocal: isLocal,
						Msg: []*deserialized.DeserializedItem{item}})
				return false, nil
			}
			switch pi.GetDone() {
			case types.DoneKeep, types.NotDone: // Validate the proposal
				return true, pi.ValidateProposal(msm.SigItems[0].Pub, proposal)
			case types.DoneClear: // Proposal for a consensus index that decided nil
				logging.Errorf("Could not validate proposal as its support index %v had DoneClear called on it", supIdx)
				return false, nil
			default:
				panic("should not reach")
			}
		}
	}
	return true, nil
}

// SetMainChannel must be called before the object is used.
func (mcs *ConsInterfaceState) SetMainChannel(mainChannel channelinterface.MainChannel) {
	if mcs.mainChannel != nil {
		panic("shold only call this once")
	}
	if mainChannel == nil {
		panic("shouldn't be nil")
	}

	mcs.mainChannel = mainChannel
	mcs.initMemberChecker.SetMainChannel(mainChannel)

	// we need to allocate the first item
	parIndex, err := types.SingleComputeConsensusID(types.ConsensusInt(1), nil)
	if err != nil {
		panic(err)
	}

	newItem := &ConsInterfaceItems{}
	// create the new member checkers
	memCheck := mcs.initMemberChecker.New(parIndex)
	specialMemCheck := mcs.initSpecialMemberChecker.New(parIndex)
	newItem.MC = &MemCheckers{memCheck, specialMemCheck}

	// create the new message state
	newItem.MsgState = mcs.initMessageState.New(parIndex)

	// create the new forwarder
	newItem.FwdChecker = mcs.initForwardChecker.New(parIndex, newItem.MC.MC.GetParticipants(), newItem.MC.MC.GetAllPubs())

	newItem.ConsItem = mcs.initItem.GenerateNewItem(parIndex, newItem, mcs.mainChannel, nil,
		mcs.broadcastFunc, mcs.gc)

	mcs.consItemsMap[parIndex.Index.(types.ConsensusInt)] = newItem
}

// GetNewPub returns an empty public key object.
func (mcs *ConsInterfaceState) GetNewPub() sig.Pub {
	return mcs.emptyPub.New()
}

/*// Reset sets the fields to an initial state (this should be called before starting consensus).
func (mcs *ConsInterfaceState) Reset() {
	mcs.mutex.Lock()

	mcs.LocalIndex = 1
	// mcs.startedIndex = 1
	mcs.firstItem = 1
	mcs.lastItem = 1
	mcs.consItemsMap = make(map[types.ConsensusInt]*ConsInterfaceItems)

	mcs.mutex.Unlock()
}*/

/*func (mcs *ConsInterfaceState) GetConsItem(idIdx types.ConsensusIndex) (ConsItem, error) {
	if mcs.mainChannel == nil {
		panic("must set main channel")
	}
	idx := idIdx.Index.(types.ConsensusInt)

	if idx < types.ConsensusInt(utils.SubOrZero(uint64(mcs.LocalIndex), uint64(mcs.gc.KeepPast))) {
		return nil, types.ErrIndexTooOld
	}

	// mcs.mutex.RLock()
	// defer mcs.mutex.RUnlock()
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	items := mcs.getIndex(idx, true)
	items.MC.MC.CheckIndex(items.ConsItem.GetIndex())
	return items.ConsItem, nil
}
*/
func (mcs *ConsInterfaceState) CheckGenMemberChecker(cid types.ConsensusIndex) (*ConsInterfaceItems, error) {

	// nothing to do here, items are generated in GetMemberChecker
	return mcs.GetMemberChecker(cid)
}

// GetMemberChecker returns the member checkers, messages state, and forward checker for the given index.
// If useLock is true, then accesses are protected by a lock.
func (mcs *ConsInterfaceState) GetMemberChecker(cid types.ConsensusIndex) (*ConsInterfaceItems, error) {
	if mcs.mainChannel == nil {
		panic("must set main channel")
	}
	idx := cid.Index.(types.ConsensusInt)

	// mcs.mutex.RLock()
	// defer mcs.mutex.RUnlock()
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	if idx < types.ConsensusInt(utils.SubOrZero(uint64(mcs.LocalIndex), uint64(mcs.gc.KeepPast))) {
		return nil, types.ErrIndexTooOld
	}
	if idx > mcs.StartedIndex+config.DropFuture {
		return nil, types.ErrIndexTooNew
	}
	if idx > mcs.StartedIndex+config.KeepFuture {
		return nil, types.ErrParentNotFound
	}

	items := mcs.getIndex(idx, true)
	if items.MC == nil || items.MsgState == nil {
		panic("shouldn't be nil")
	}
	if items.MC.MC.IsReady() && items.FwdChecker == nil {
		panic("shouldn't be nil")
	}
	if items.MC.MC.GetIndex().Index != idx {
		panic(fmt.Sprintf("got wrong index %v, expected %v", items.MC.MC.GetIndex(), idx))
	}
	items.MC.MC.CheckIndex(items.ConsItem.GetIndex())

	return items, nil
}

// DoneIndex is called with consensus has finished at idx, binstate is the value decided, which is passed
// to the member checker's UpdateState method.
// It allocates the objects for new consensus instances and garbage collects old ones as necessary.
// It is concurrent safe with GetMemberChecker.
// This should only be called from the main thread.
// It returns true if the last consensus index has finished.
func (mcs *ConsInterfaceState) DoneIndex(nextIdxID, supportIndex, futureDependentIndex types.ConsensusID,
	proposer sig.Pub, dec []byte) (finishedLastRound bool) {

	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	nextIdx := nextIdxID.(types.ConsensusInt)
	if nextIdx != mcs.LocalIndex {
		panic("Out of order updates")
	}

	// If this value decided came from a different index then previously suggested, we reset it
	if len(dec) != 0 {
		if oldSupport, ok := mcs.SupportIndex[nextIdx]; ok && oldSupport != supportIndex { // sanity check
			panic(fmt.Sprint("must not change support index from initial value ", nextIdx, oldSupport, supportIndex))
			// pi.DoneClear()
			// pi = cs.proposalInfo[supportIndex].StartIndex(nextIdx)
			// cs.supportIndex[nextIdx] = supportIndex
			// cs.proposalInfo[nextIdx] = pi
		}
	}

	// can we allocate a new SM from one already called DoneClear?
	// no because we always get our most recent support when getting a message

	// Allocate the SM for the decided index if necessary
	pi := mcs.ProposalInfo[nextIdx]
	var gotInput bool
	// Inform the proposal object of decision
	if pi != nil {
		if mcs.allowConcurrent > 0 { // when running concurrent we always do the SM 1 at a time at decision
			pi.DoneClear()
			pi = nil
		} else if mcs.initItem.NeedsConcurrent() > 1 && len(dec) == 0 { // if we need concurrent and decide nil, then this instance wont be used
			pi.DoneClear()
		} else if preDec, ok := mcs.PreDecisions[nextIdx]; ok {
			// if we have a preDecision and it is the same as decision the we have already called pi.HasDecided
			if !bytes.Equal(dec, preDec) {
				if len(dec) != 0 && len(preDec) != 0 {
					panic("must decide the same as the preDecision if non-nil decision")
				}
				// since the decision and predecision are different then we wont use this index
				pi.DoneClear()
				pi = nil
			} else if len(preDec) > 0 {
				gotInput = true // we had an actual non-nil predecision
			}
		}
	}

	var randBytes [32]byte // Random bytes from the state machine if enabled

	// if we decided a non-nil value
	// or if we are running concurrent instances (we keep all instances if we are concurrent)
	// then we keep the SM
	if len(dec) > 0 || // non-nil decision
		mcs.allowConcurrent > 0 || // concurrency allowed
		(mcs.allowConcurrent == 0 && mcs.initItem.NeedsConcurrent() == 1) { // no concurrency or need concurrent

		if pi == nil {
			pi = mcs.allocSM(supportIndex.(types.ConsensusInt), nextIdx)
		}
		if !gotInput {
			pi.HasDecided(proposer, nextIdx, dec)
			mcs.reprocessProposals(nextIdx)
		}
		if mcs.genRandBytes {
			randBytes = pi.GetRand()
		}
		pi.DoneKeep()
	}

	// Update the member checker
	mcs.updateMC(nextIdx, futureDependentIndex, dec, randBytes)

	// Update the state index
	mcs.LocalIndex++

	// garbage collection
	if len(dec) > 0 { // non nil decision // TODO should do this differently for memory managment?

		var gcUntil types.ConsensusInt
		mcs.lastNonNilDecs = append(mcs.lastNonNilDecs, nextIdx)
		if len(mcs.lastNonNilDecs) > mcs.gc.KeepPast {
			mcs.lastNonNilDecs = mcs.lastNonNilDecs[1:]
			gcUntil = types.ConsensusInt(utils.SubOrZero(uint64(mcs.lastNonNilDecs[0]), 1))
		}

		for v := mcs.gcUpTo; v < gcUntil; v++ {
			// if v := types.ConsensusInt(utils.SubOrZero(uint64(mcs.LocalIndex), config.KeepPast+1)); v > 0 {
			logging.Info("Garbage collection of cons instance", v)

			mcs.clearIndex(v)
			mcs.consItemsMap[v+1].ConsItem.PrevHasBeenReset()
			mcs.gcUpTo = v
		}
	}
	if pi == nil {
		return false
	}
	return pi.FinishedLastRound()
}

func (mcs *ConsInterfaceState) allocSM(supportIndex, nextIdx types.ConsensusInt) StateMachineInterface {
	prvSM := mcs.ProposalInfo[supportIndex]
	if !prvSM.GetDecided() && mcs.initItem.NeedsConcurrent() > 1 {
		panic("idx")
	}
	pi := prvSM.StartIndex(nextIdx)
	mcs.SupportIndex[nextIdx] = supportIndex
	mcs.ProposalInfo[nextIdx] = pi
	logging.Info("gen sm from support", supportIndex, "my index", nextIdx)
	return pi
}

func (mcs *ConsInterfaceState) clearIndex(v types.ConsensusInt) {
	if ci, ok := mcs.consItemsMap[v]; ok {
		if err := ci.MC.MC.DoneNextUpdateState(); err != nil {
			panic(err)
		}
		ci.ConsItem.Collect()
	}
	delete(mcs.consItemsMap, v)
	if item, ok := mcs.ProposalInfo[v]; ok {
		item.Collect() // sanity check
	}

	delete(mcs.ProposalInfo, v)
	delete(mcs.PreDecisions, v)
	delete(mcs.SupportIndex, v)
	delete(mcs.GotProposal, v)
}

func (mcs *ConsInterfaceState) reprocessProposals(nextIdx types.ConsensusInt) {
	lst := mcs.ProposalsToValidate[nextIdx]
	if len(lst) > 0 {
		for _, nxt := range lst {
			mcs.mainChannel.ReprocessMessage(nxt)
		}
		delete(mcs.ProposalsToValidate, nextIdx)
	}
}

func (mcs *ConsInterfaceState) updateMC(idx types.ConsensusInt, futureDependentID types.ConsensusID, dec []byte, randBytes [32]byte) {
	var futureDependentIndex types.ConsensusInt
	if futureDependentID == nil {
	} else {
		futureDependentIndex = futureDependentID.(types.ConsensusInt)
		if mcs.lastMCInputIndex > futureDependentIndex { // || (mcs.allowConcurrent == 0 && futureDependentIndex <= mcs.StartedIndex) {
			panic("member checkers decided out of order")
		}
		mcs.lastMCInputIndex = futureDependentIndex
		if futureDependentIndex <= idx {
			panic("future dependent index must come after this index")
		}
	}
	prevItems := mcs.consItemsMap[idx]
	if prevItems == nil {
		panic("shouldn't be nil")
	}
	prevMemberChecker := prevItems.MC
	// mcs.lastMCInputIndex = futureDependentIndex

	nextConsItem := mcs.getIndex(idx+1, true)
	nextMemberChecker := nextConsItem.MC

	// inform member checker of the next consensus index that the previous has decided
	newPubs, newAllPubs, changedMembers := nextMemberChecker.MC.UpdateState(nil, dec, randBytes, prevMemberChecker.MC,
		mcs.ProposalInfo[idx], futureDependentID)
	if newPubs != nil && nextMemberChecker.MC.IsReady() {
		panic("Member checker should not be ready if the members change until, FinishUpdateStateIsCalled")
	}
	// inform the special member checker as well
	nextMemberChecker.SMC.UpdateState(newPubs, randBytes, prevMemberChecker.SMC)

	// Generate the forward checker if needed.
	if nextConsItem.FwdChecker == nil {
		nextConsItem.FwdChecker = mcs.initForwardChecker.New(nextConsItem.ConsItem.GetIndex(), newPubs, newAllPubs)
	} else if newPubs != nil || newAllPubs != nil {
		panic("created fwd checker before we knew the participants")
	}

	// call finish update state on the member checker
	nextMemberChecker.MC.FinishUpdateState()
	if !nextMemberChecker.MC.IsReady() { // sanity check
		panic("should know members by now")
	}

	if futureDependentID == nil { // no changes so we don't have to check for invalid future indices
		if changedMembers {
			panic("should not have changed members if we had no future dependent index")
		}
		return
	}

	// If we changed members, then we have to check for all future indices that have started to reset them
	// we start clearing no earlier then 1 index after the member checker that just processed the input // TODO clean this up
	// (because we cant clear the index we just created, or any index before futureDependedIndex as the current
	// decided index needs that not to change)
	startClear := utils.MaxConsensusIndex(idx+2, futureDependentIndex)
	var mustClear bool
	clearedFrom := mcs.lastItem + 1
	prev := mcs.consItemsMap[startClear-1]
	for i := startClear; i <= mcs.lastItem; i++ {
		item := mcs.consItemsMap[i]
		if changedMembers && item.MC.MC.IsReady() { // we only have to clear the item if it has checked membership already
			mustClear = true
		}
		if !mustClear {
			prev = item
			continue
		}
		if clearedFrom > i {
			clearedFrom = i
		}
		prev.ConsItem.SetNextConsItem(nil)
		// If we started the index, then we have to reset it
		if i <= mcs.StartedIndex {
			if pi, ok := mcs.ProposalInfo[i]; ok {
				pi.DoneClear()
				pi.Collect()
				delete(mcs.ProposalInfo, i)
			}
		}
		// If we have already processes messages, then we reprocess them with the new member checker
		binstate, err := item.ConsItem.GetBinState(false)
		utils.PanicNonNil(err)
		if len(binstate) > 0 {
			mcs.mainChannel.ReprocessMessageBytes(binstate)
		}
		err = item.MC.MC.Invalidated()
		utils.PanicNonNil(err)
		mcs.clearIndex(i)
		prev = item
	}
	if clearedFrom <= mcs.StartedIndex { // update the started index for items removed
		var maxStarted types.ConsensusInt
		for nxt := range mcs.ProposalInfo {
			if maxStarted < nxt {
				maxStarted = nxt
			}
		}
		mcs.StartedIndex = maxStarted
	}
	mcs.lastItem = clearedFrom - 1 // we have removed all later items

	var items sort.IntSlice
	for k := range mcs.consItemsMap {
		items = append(items, int(k))
	}
	items.Sort()
	pr := items[0]
	for _, nxt := range items[1:] {
		if nxt != pr+1 {
			panic(1)
		}
		pr = nxt
	}
	if mcs.lastItem != types.ConsensusInt(items[len(items)-1]) {
		panic(1)
	}
}

func (mcs *ConsInterfaceState) getIndex(endidx types.ConsensusInt, alreadyWriteLocked bool) *ConsInterfaceItems {

	item := mcs.consItemsMap[endidx]
	if item == nil { // allocate any missing objects
		if !alreadyWriteLocked {
			panic(1)
		}

		item = mcs.consItemsMap[endidx]
		if item == nil {

			mcs.initSM.CheckStartStatsRecording(endidx)

			if endidx <= mcs.lastItem { // sanity check
				panic(fmt.Sprint("should be allocated", endidx, mcs.lastItem, mcs.gcUpTo))
			}
			prevItem := mcs.consItemsMap[mcs.lastItem]
			if prevItem == nil { // sanity check
				panic(fmt.Sprint("should be allocated", mcs.lastItem))
			}
			for nxtIdx := mcs.lastItem + 1; nxtIdx <= endidx; nxtIdx++ {
				if mcs.consItemsMap[nxtIdx] != nil {
					panic("should not have been allocated")
				}

				// we need to allocate this item
				newItem := &ConsInterfaceItems{}
				// create the new member checkers
				//runtime.SetFinalizer(newItem, func(item *ConsInterfaceItems) {
				//})

				parIndex, err := types.SingleComputeConsensusID(nxtIdx, nil)
				if err != nil {
					panic(err)
				}

				memCheck := mcs.initMemberChecker.New(parIndex)
				specialMemCheck := mcs.initSpecialMemberChecker.New(parIndex)
				newItem.MC = &MemCheckers{memCheck, specialMemCheck}

				// create the new message state
				newItem.MsgState = mcs.initMessageState.New(parIndex)

				// create the new forwarder
				if memCheck.IsReady() {
					// if memCheck.AllowsChange() {
					//	panic("should not be ready if allows change")
					//}
					newItem.FwdChecker = mcs.initForwardChecker.New(parIndex, memCheck.GetParticipants(), memCheck.GetAllPubs())
				}

				mcs.SharedLock.Lock()
				newItem.ConsItem = mcs.initItem.GenerateNewItem(parIndex, newItem, mcs.mainChannel, prevItem.ConsItem,
					mcs.broadcastFunc, mcs.gc)
				prevItem.ConsItem.SetNextConsItem(newItem.ConsItem)
				mcs.SharedLock.Unlock()

				mcs.consItemsMap[parIndex.Index.(types.ConsensusInt)] = newItem
				prevItem = newItem
			}
			mcs.lastItem = endidx
			item = prevItem
		} else if endidx > mcs.lastItem { // sanity check
			panic("shouldn't yet be allocated")
		}
		endID, err := types.SingleComputeConsensusID(endidx, nil)
		if err != nil {
			panic(err)
		}
		if !item.MC.MC.CheckIndex(endID) || item.ConsItem.GetIndex().Index != endidx {
			panic("error creating item")
		}
	}
	return item
}

// CheckPreDecision will check if index has a possible decided value ready and allocates
// any necessary objects.
// This should only be called from the main thread.
func (mcs *ConsInterfaceState) CheckPreDecision() {
	if mcs.IsInStorageInit { // predecisions not needed during init since each init value has been decided
		return
	}

	for idx := mcs.LocalIndex;
	// idx <= cs.startedIndex; idx++ {
	// idx < cs.startedIndex || idx == cs.memberCheckerState.LocalIndex; idx++ {
	idx <= mcs.StartedIndex; idx++ {

		idxItem, err := types.SingleComputeConsensusID(idx, nil)
		if err != nil {
			panic(err)
		}
		nxtItem, err := mcs.GetMemberChecker(idxItem)
		if err != nil {
			panic(err)
		}
		if _, ok := mcs.PreDecisions[idx]; ok { // skip this index if we already got the predecision
			continue
		}

		// Get information about what the previous started index can decide
		if prvIdx, proposer, preDecision, ready := nxtItem.ConsItem.GetNextInfo(); !ready {
			// the predecision is not ready so we go to the next
			continue
		} else { // we are ready for another pre-decision

			// update the pre-decision index
			// cs.memberCheckerState.PredecisionIndex++
			if len(preDecision) == 0 && prvIdx.Index != idx-1 {
				panic(
					fmt.Sprintf("should not have nil pre decision that doesn't support the previous index, expected %v, got %v",
						idx, prvIdx))
			}
			if mcs.allowConcurrent > 0 && prvIdx.Index != idx-1 {
				panic(
					fmt.Sprintf("should pre decision that doesn't support the previous index with concurrency, expected %v, got %v",
						idx, prvIdx))
			}

			// See if you need to allocate the SM for the previous started index
			nextSM := mcs.ProposalInfo[idx]
			if nextSM == nil {
				prvSM := mcs.ProposalInfo[prvIdx.Index.(types.ConsensusInt)]
				if prvSM == nil {
					continue
					// panic(fmt.Sprint("nil prvSM ", prvIdx, idx, cs.memberCheckerState.LocalIndex))
				}
				if prvIdx.Index.(types.ConsensusInt) != 0 && !prvSM.GetDecided() {
					// The prvSM might has passed due to a timeout, allowing this instance to start
					// But then was able to decide later, so is supported, but may have not received the init message
					// So we might have to skil this instance
					continue
					/*					privItem, err := cs.memberCheckerState.GetConsItem(prvIdx)
										if err != nil {
											panic(err)
										}
										if !privItem.CanStartNext() { // Be sure the previous item no longer allows to start the next
											// This can only happened if previously it passed a timeout allowing it to start next
											// but then it decided anyway, but did not yet receive the init message so now has to wait
											continue
										}
										panic(fmt.Sprint(prvSM, prvIdx, idx, cs.memberCheckerState.LocalIndex, cs.startedIndex, cs.generalConfig.TestIndex))*/
				}
				nextSM = prvSM.StartIndex(idx)
				// Get the proposal for the SM if needed
				if prvIdx2, ready := nxtItem.ConsItem.GetProposalIndex(); ready && !mcs.GotProposal[idx] {
					mcs.GotProposal[idx] = true
					if prvIdx.Index != prvIdx2.Index {
						logging.Error("got different support indecies")
					} else {
						nextSM.GetProposal()
						logging.Info("get proposal 1", mcs.gc.TestIndex, idx)
					}
				}
				logging.Info("gen sm from support", prvIdx, "my index", idx)
				mcs.ProposalInfo[idx] = nextSM
				mcs.SupportIndex[idx] = prvIdx.Index.(types.ConsensusInt)
			}
			// Input the preDecision to the previous SM
			mcs.SupportIndex[idx] = prvIdx.Index.(types.ConsensusInt)
			mcs.PreDecisions[idx] = preDecision
			if len(preDecision) > 0 {
				nextSM.HasDecided(proposer, idx, preDecision)
				mcs.reprocessProposals(idx)
			}
		}
	}
}

// CheckProposalNeeded will check if item needs a proposal, and will request one if needed from the SM.
// This should only be called from the main thread.
func (mcs *ConsInterfaceState) CheckProposalNeeded(nextIdx types.ConsensusInt) {
	if nextIdx > mcs.StartedIndex { // if we haven't started this index yet then exit
		return
	}

	idxItem, err := types.SingleComputeConsensusID(nextIdx, nil)
	if err != nil {
		panic(err)
	}
	nextItem, err := mcs.GetMemberChecker(idxItem)
	if err != nil {
		panic(err)
	}

	if preIdx, ready := nextItem.ConsItem.GetProposalIndex(); ready && !mcs.GotProposal[nextIdx] {
		var pi StateMachineInterface
		if pi = mcs.ProposalInfo[nextIdx]; pi == nil {
			prvSM := mcs.ProposalInfo[preIdx.Index.(types.ConsensusInt)]
			if prvSM == nil {
				// TODO better check, normally this should not happen only when restart from disk
				logging.Warningf("nil prvSM, support %v, idx %v\n", preIdx, nextIdx)
				return
			}
			if preIdx.Index.(types.ConsensusInt) != 0 && !prvSM.GetDecided() && mcs.allowConcurrent == 0 {
				panic(fmt.Sprintf("not decided prvSM, support %v, idx %v\n", preIdx, nextIdx))
			}
			if prvSM.GetDone() == types.DoneClear {
				logging.Errorf("got invalid init: previous pointer decided nil %v, current idx %v", preIdx, nextIdx)
				return
			}
			pi = prvSM.StartIndex(nextIdx)
			logging.Info("gen sm from support", preIdx, "my index", nextIdx)
			if mcs.allowConcurrent > 0 { // if using concurrency then we keep the SM, since we recreate them all at decision
				mcs.ProposalInfo[nextIdx] = pi
				mcs.SupportIndex[nextIdx] = preIdx.Index.(types.ConsensusInt)
			}
		} else if mcs.SupportIndex[nextIdx] != preIdx.Index {
			panic("support index should not change")
		}

		pi.GetProposal()
		mcs.GotProposal[nextIdx] = true
	}
}

// Collect is called when the process is terminating.
func (mcs *ConsInterfaceState) Collect() {
	for _, nxt := range mcs.consItemsMap {
		nxt.ConsItem.Collect()
	}
	for _, nxt := range mcs.ProposalInfo {
		if nxt.GetDone() == types.NotDone {
			nxt.DoneClear()
		}
		nxt.EndTest()
		nxt.Collect()
	}
	logging.Info("done collect", mcs.gc.TestIndex)
}
