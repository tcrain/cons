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
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/storage"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sort"
	"sync"
	"time"
)

type CausalConsInterfaceState struct {
	initMemberChecker        MemberChecker               // New is called on this object to allocate all member checkers
	initSpecialMemberChecker SpecialPubMemberChecker     // New is called on this object to allocate all special member checkers
	initMessageState         MessageState                // New is called on this object to allocate all message state objects
	initForwardChecker       ForwardChecker              // New is called on this object to allocate all forward checkers
	initItem                 ConsItem                    // Consensus item used to generate all new consensus items.
	initSM                   CausalStateMachineInterface // Used to generate new state machines
	initHash                 types.ParentConsensusHash   // The initial parent state machine
	broadcastFunc            ByzBroadcastFunc

	// since we don't call DoneNextUpdateState on the init item, (because it has already decided), we call it
	// once the first index has decided
	calledInitDoneNext bool

	gc          *generalconfig.GeneralConfig
	mainChannel channelinterface.MainChannel

	consItemSorted SortedTimeConsInterfaceItems // Consensus items sorted by time last updated

	consItemsMap map[types.ParentConsensusHash]*ConsInterfaceItems         // Consensus items
	ProposalInfo map[types.ParentConsensusHash]CausalStateMachineInterface // Interface to the state machine.

	// Map from outputs of a consensus to their parent (inverse of parentToChildMap)
	childToParentMap map[types.ConsensusHash]types.ParentConsensusHash
	// Map from parent to their child outputs (inverse of childToParentMap)
	parentToChildMap map[types.ParentConsensusHash][]types.ConsensusHash

	// Map from output to owner
	outputToOwner map[types.ConsensusHash]sig.Pub

	// Map from idx to it the consensus items that consumed at least one of its outputs.
	// Note that we don't GC this since it is used for recovery, TODO find a way to keep this on disk
	parentToChildConsumers map[types.ParentConsensusHash][]types.ParentConsensusHash
	// Map from output to index of consumer
	// This is also used for recovery so is not GC'd // TODO find a way to store on disk
	outputToConsumer map[types.ConsensusHash]types.ParentConsensusHash

	// Map from consensus index to the child outputs that it is consuming.
	itemToConsumedOutputs map[types.ParentConsensusHash][]types.ConsensusHash

	// Set of proposals to be validated.
	// It is a map from a child index to a list of proposals.
	// Proposals that try to consume a non-existing child output will be added here.
	// When the child output is created after a decision these will be reprocessed.
	// TODO if a proposal arrives after the child item is consumed it will remain here forever, need to do GC
	ProposalsToValidate map[types.ConsensusHash][]*channelinterface.RcvMsg

	// Map from child outputs to boolean, set to true if we have sent an echo for the output.
	// Used so only 1 echo is sent per output.
	consumedEcho map[types.ConsensusHash]bool

	storage storage.StoreInterface // the storage

	mutex           sync.Mutex // concurrency control
	emptyPub        sig.Pub    // Will be used to create empty public key objects
	IsInStorageInit bool       // Used during initialization, when recovering from disk this is set to true so we don't send messages when we reply to values received.
	genRandBytes    bool       // Only works if GenRandBytes is ture. This uses NumTotalProcs and NumNonMembers to randomly decide which nodes will participate.

	// futureMessages is a map of messages by their consensus index that were too far in the future to be processed yet.
	// TODO should GC for invalid indices, now they are kept in case they are for an instance not yet generated
	futureMessages map[types.ConsensusHash][]*channelinterface.RcvMsg

	LastDecided types.ParentConsensusHash // TODO

}

func NewCausalConsInterfaceState(initItem ConsItem,
	initMemberChecker MemberChecker,
	initSpecialMemberChecker SpecialPubMemberChecker,
	initMessageState MessageState,
	initForwardChecker ForwardChecker,
	storage storage.StoreInterface,
	genRandBytes bool,
	emptyPub sig.Pub, broadcastFunc ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) *CausalConsInterfaceState {

	mcs := &CausalConsInterfaceState{}
	mcs.initItem = initItem
	mcs.broadcastFunc = broadcastFunc
	mcs.genRandBytes = genRandBytes
	// mcs.itemMap[1] = mcs.initItem.GenerateNewItem(1, memChecker, MsgState, FwdChecker, mainChannel, nil, cs.generalConfig)
	mcs.storage = storage
	mcs.initMemberChecker = initMemberChecker
	mcs.initSpecialMemberChecker = initSpecialMemberChecker
	mcs.initMessageState = initMessageState
	mcs.initForwardChecker = initForwardChecker
	mcs.emptyPub = emptyPub
	mcs.gc = gc

	mcs.futureMessages = make(map[types.ConsensusHash][]*channelinterface.RcvMsg)

	mcs.outputToOwner = make(map[types.ConsensusHash]sig.Pub)
	mcs.consItemsMap = make(map[types.ParentConsensusHash]*ConsInterfaceItems)         // Consensus items
	mcs.ProposalInfo = make(map[types.ParentConsensusHash]CausalStateMachineInterface) // Interface to the state machine.
	mcs.childToParentMap = make(map[types.ConsensusHash]types.ParentConsensusHash)     // Map from child to parent
	// GotProposal         map[types.ConsensusID]bool                       // set to true if received a proposal for the index
	mcs.ProposalsToValidate = make(map[types.ConsensusHash][]*channelinterface.RcvMsg) // set of proposals to be validated TODO GC this
	mcs.parentToChildMap = make(map[types.ParentConsensusHash][]types.ConsensusHash)   // map from parent to child
	mcs.consumedEcho = make(map[types.ConsensusHash]bool)                              // true if we have sent an echo for this proposal
	mcs.itemToConsumedOutputs = make(map[types.ParentConsensusHash][]types.ConsensusHash)
	mcs.outputToConsumer = make(map[types.ConsensusHash]types.ParentConsensusHash)

	// we keep a map from idx to its decided child idxs for recovery (that we don't GC), TODO find a way to keep this on disk
	mcs.parentToChildConsumers = make(map[types.ParentConsensusHash][]types.ParentConsensusHash)

	return mcs
}

func (mcs *CausalConsInterfaceState) AddFutureMessage(rcvMsg *channelinterface.RcvMsg) {
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	for _, nxtMsg := range rcvMsg.Msg {
		nxtIdx := nxtMsg.Index.FirstIndex.(types.ConsensusHash)
		var found bool
		for i := -1; i < len(nxtMsg.Index.AdditionalIndices); i++ {
			if _, ok := mcs.childToParentMap[nxtIdx]; !ok {
				mcs.futureMessages[nxtIdx] = append(mcs.futureMessages[nxtIdx], rcvMsg)
				found = true
				break
			}
			if i+1 < len(nxtMsg.Index.AdditionalIndices) {
				nxtIdx = nxtMsg.Index.AdditionalIndices[i+1].(types.ConsensusHash)
			}
		}
		if !found {
			mcs.mainChannel.ReprocessMessage(
				&channelinterface.RcvMsg{CameFrom: 34, Msg: []*channelinterface.DeserializedItem{nxtMsg},
					SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
		}
	}
}

// GetGeneralConfig returns the GeneralConfig object.
func (mcs *CausalConsInterfaceState) GetGeneralConfig() *generalconfig.GeneralConfig {
	return mcs.gc
}

func (mcs *CausalConsInterfaceState) checkMsgHashes(item *channelinterface.DeserializedItem) error {
	if mcs.IsInStorageInit {
		return nil // in loading from disk so ok
	}
	switch item.Header.(type) {
	case *sig.MultipleSignedMessage, *sig.UnsignedMessage:
		// check if we already decided this index
		if mcs.storage.Contains(item.Index.Index.(types.ConsensusHash)) {
			return types.ErrConsAlreadyDecided
		}
	}
	return nil
}

func (mcs *CausalConsInterfaceState) UpdateProgressTime(index types.ConsensusIndex) {
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	hash := types.ParentConsensusHash(index.Index.(types.ConsensusHash))
	item := mcs.consItemsMap[hash]
	mcs.moveToEnd(item.ProgressListIndex, item.ProgressListIndex+1)
	// TODO remove check
	if !sort.IsSorted(mcs.consItemSorted) {
		panic("should be sorted")
	}
}

// moveToEnd is called for the items from startIdx until endIdx to ve moved to the end of consItemSorted,
// and have their times updated
func (mcs *CausalConsInterfaceState) moveToEnd(startIdx, endIdx int) {
	// exit if nothing to do
	if startIdx >= endIdx {
		return
	}

	now := time.Now()

	startLen := len(mcs.consItemSorted)

	// compute the items to be moved to the end
	var newEnd []*ConsInterfaceItems
	newEnd = append(newEnd, mcs.consItemSorted[startIdx:endIdx]...)
	size := len(newEnd)
	// move the other items forward
	for i := endIdx; i < len(mcs.consItemSorted); i++ {
		mcs.consItemSorted[i-size] = mcs.consItemSorted[i]
		mcs.consItemSorted[i].ProgressListIndex -= size
	}
	// put the new end items at the end
	for i, nxt := range newEnd {
		idx := len(mcs.consItemSorted) - i - 1
		mcs.consItemSorted[idx] = nxt
		nxt.LastProgress = now
		nxt.ProgressListIndex = idx
	}

	if startLen != len(mcs.consItemSorted) {
		panic("error sort") // sanity
	}
}

// GetComputeConsensusIDFunc returns types.GenerateParentHash.
func (mcs *CausalConsInterfaceState) GetConsensusIndexFuncs() types.ConsensusIndexFuncs {
	return types.HashIndexFuns
}

// CheckValidateProposal checks if item is a proposal that needs to be validated by the SM.
// If it is not true, nil is returned.
// It it is, but the state machine is not ready to validate it then false, nil is returned.
// (This object will take care of reprocessing the message in this case.)
// If it is ready then true is returned.
// If there is an error processing the message then an error is returned.
// It validates the sent value with each of the hashes given by the signed message in indices and AdditionalIndices
func (mcs *CausalConsInterfaceState) CheckValidateProposal(item *channelinterface.DeserializedItem,
	sendRecvChan *channelinterface.SendRecvChannel, isLocal bool) (readyToProcess bool, err error) {

	if msm, ok := item.Header.(*sig.MultipleSignedMessage); ok { // is this a signed message?

		mcs.mutex.Lock()
		defer mcs.mutex.Unlock()

		// does this proposal need validation?
		_, p, err := msm.NeedsSMValidation(item.Index, 0)
		if err != nil {
			return false, err
		}

		if p != nil { // We aneed to validate this proposal

			// check if this instance was already decided
			if err := mcs.checkMsgHashes(item); err != nil {
				return false, err
			}

			// Go through each index and validate
			nxtIdx := msm.Index.FirstIndex
			for i := -1; i < len(msm.Index.AdditionalIndices); i++ {
				_, proposal, err := msm.NeedsSMValidation(item.Index, i+1)
				if err != nil {
					return false, err
				}
				var ok bool
				var parIdx types.ParentConsensusHash
				if parIdx, ok = mcs.childToParentMap[nxtIdx.(types.ConsensusHash)]; ok {
					// parIdx := curSM.GetParentIndex()
					parSM := mcs.ProposalInfo[parIdx]
					if ok = parSM.GetDecided(); ok {
						if err = parSM.ValidateProposal(msm.SigItems[0].Pub, proposal); err != nil {
							return false, err
						}
					}
				}
				if !ok {
					// The proposal can be validated when the parent index is ready
					// TODO could send invalid hashes for this so may want to clean in someway, or only keep ones you know about
					// so then if a valid one is dropped it will be resent later.
					// TODO here could also append already decided hashes? No because we need to check parent first with the disk storage
					logging.Warningf("need to validate proposal later, idx %v", nxtIdx)
					mcs.ProposalsToValidate[nxtIdx.(types.ConsensusHash)] =
						append(mcs.ProposalsToValidate[nxtIdx.(types.ConsensusHash)],
							&channelinterface.RcvMsg{SendRecvChan: sendRecvChan, IsLocal: isLocal,
								Msg: []*channelinterface.DeserializedItem{item}})
					return false, nil
				}
				if i+1 < len(msm.Index.AdditionalIndices) {
					nxtIdx = msm.Index.AdditionalIndices[i+1]
				}
			}

			// Generate the combined hash, so we can use it to allocate the SM
			if mcs.getIndex(types.ParentConsensusHash(msm.Index.Index.(types.ConsensusHash))) == nil {
				panic("should be allocated")
			}

			// This will allocate the SM, this is done so that messages from this consensus index can be processed

			// if err := mcs.genIndex(types.ParentConsensusHash(parHash.(types.ConsensusHash)),
			//	msm.Index.(types.ConsensusHash), msm.AdditionalIndices); err != nil {

			//	panic(err)
			// }
		}
	}
	return true, nil
}

// CheckSendEcho should be called when an echo is sent.
// If none of the indices have sent an echo before then nil is returned.
// Otherwise an error is returned.
func (mcs *CausalConsInterfaceState) CheckSendEcho(msm *sig.MultipleSignedMessage) error {

	panic("depricated")
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	// check if we already consumed
	nxtIdx := msm.Index.FirstIndex
	for i := -1; i < len(msm.Index.AdditionalIndices); i++ {
		if mcs.consumedEcho[nxtIdx.(types.ConsensusHash)] {
			return types.ErrAlreadyConsumedEcho
		}
		if i+1 < len(msm.Index.AdditionalIndices) {
			nxtIdx = msm.Index.AdditionalIndices[i+1]
		}
	}
	// consume
	nxtIdx = msm.Index.FirstIndex
	for i := -1; i < len(msm.Index.AdditionalIndices); i++ {
		mcs.consumedEcho[nxtIdx.(types.ConsensusHash)] = true
		if i+1 < len(msm.Index.AdditionalIndices) {
			nxtIdx = msm.Index.AdditionalIndices[i+1]
		}
	}
	return nil
}

// SetMainChannel must be called before the object is used.
func (mcs *CausalConsInterfaceState) SetMainChannel(mainChannel channelinterface.MainChannel) {
	if mcs.mainChannel != nil {
		panic("shold only call this once")
	}
	if mainChannel == nil {
		panic("shouldn't be nil")
	}

	mcs.mainChannel = mainChannel
	mcs.initMemberChecker.SetMainChannel(mainChannel)
}

func (mcs *CausalConsInterfaceState) InitSM(initSM CausalStateMachineInterface) {

	// These items indicate our initial state
	initFirstIndex := initSM.GetInitialFirstIndex()
	initIdx := initSM.GetIndex()
	initParentHash := initIdx.Index.(types.ConsensusHash)
	initParentIdx := types.ParentConsensusHash(initParentHash)
	initDependentItems := initSM.GetDependentItems()

	mcs.initHash = initParentIdx

	// sanity check
	if initFirstIndex != initIdx.FirstIndex {
		panic("should be equal")
	}

	// sanity check
	if chckIdx, err := types.GenerateParentHash(initFirstIndex, nil); err != nil || chckIdx.Index != initParentHash {
		panic(err)
	}

	if len(initDependentItems) == 0 {
		panic("must start with some resources")
	}

	initDepHashes := make([]types.ConsensusHash, len(initDependentItems))
	for i, nxt := range initDependentItems {
		initDepHashes[i] = nxt.ID.(types.ConsensusHash)
		mcs.childToParentMap[initDepHashes[i]] = initParentIdx
		mcs.outputToOwner[initDepHashes[i]] = nxt.Pub
	}
	mcs.ProposalInfo[initParentIdx] = initSM
	mcs.parentToChildMap[initParentIdx] = initDepHashes

	mcs.initSM = initSM
	mcs.initSM.DoneKeep()

	// we need to allocate the first item
	if types.InitIndexHash != initParentHash { //TODO fix this
		panic("diff init hashes")
	}

	newItem := &ConsInterfaceItems{}
	// create the new member checkers
	memCheck := mcs.initMemberChecker.New(initIdx)
	specialMemCheck := mcs.initSpecialMemberChecker.New(initIdx)
	newItem.MC = &MemCheckers{memCheck, specialMemCheck}

	// create the new message state
	newItem.MsgState = mcs.initMessageState.New(initIdx)

	// create the new forwarder
	newItem.FwdChecker = mcs.initForwardChecker.New(initIdx, newItem.MC.MC.GetParticipants(), newItem.MC.MC.GetAllPubs())

	// create the cons item
	newItem.ConsItem = mcs.initItem.GenerateNewItem(initIdx, newItem, mcs.mainChannel, nil,
		mcs.broadcastFunc, mcs.gc)

	newItem.LastProgress = time.Now()
	mcs.consItemSorted = append(mcs.consItemSorted, newItem)
	mcs.consItemsMap[initParentIdx] = newItem

	newItem.ConsItem.Start()
}

// GetNewPub returns an empty public key object.
func (mcs *CausalConsInterfaceState) GetNewPub() sig.Pub {
	return mcs.emptyPub.New()
}

// getConsInterfaceItems returns the last decided consInterfaceItems.
func (mcs *CausalConsInterfaceState) GetLastDecidedItem() *ConsInterfaceItems {
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	return mcs.getIndex(mcs.LastDecided)
}

func (mcs *CausalConsInterfaceState) GetConsItem(idx types.ConsensusIndex) (ConsItem, error) {
	if mcs.mainChannel == nil {
		panic("must set main channel")
	}
	// mcs.mutex.RLock()
	// defer mcs.mutex.RUnlock()
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	items := mcs.getIndex(types.ParentConsensusHash(idx.Index.(types.ConsensusHash)))
	items.MC.MC.CheckIndex(idx)
	return items.ConsItem, nil
}

func (mcs *CausalConsInterfaceState) CheckGenMemberChecker(idx types.ConsensusIndex) (
	*MemCheckers, MessageState, ForwardChecker, error) {

	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	return mcs.genIndex(idx)
}

// GetMemberChecker returns the member checkers, messages state, and forward checker for the given index.
// If useLock is true, then accesses are protected by a lock.
func (mcs *CausalConsInterfaceState) GetMemberChecker(cid types.ConsensusIndex) (*MemCheckers,
	MessageState, ForwardChecker, error) {

	if mcs.mainChannel == nil {
		panic("must set main channel")
	}
	idx := cid.Index.(types.ConsensusHash)

	// mcs.mutex.RLock()
	// defer mcs.mutex.RUnlock()
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	items := mcs.getIndex(types.ParentConsensusHash(idx))
	if items != nil {
		if items.MC == nil || items.MsgState == nil {
			panic("shouldn't be nil")
		}
		if items.MC.MC.IsReady() && items.FwdChecker == nil {
			panic("shouldn't be nil")
		}
		if items.MC.MC.GetIndex().Index != idx {
			panic(fmt.Sprintf("got wrong index %v, expected %v", items.MC.MC.GetIndex(), idx))
		}
		return items.MC, items.MsgState, items.FwdChecker, nil
	}
	// mcs.genIndex()
	return nil, nil, nil, types.ErrInvalidIndex
}

func (mcs *CausalConsInterfaceState) recGenChildInd(itm types.ParentConsensusHash,
	ret []types.ParentConsensusHash, maxCount int, soFar map[types.ParentConsensusHash]bool) []types.ParentConsensusHash {

	//if len(ret) >= maxCount { // TODO fix this as could mean we never send the ones we need to progress?
	//	return ret
	//}

	children := mcs.parentToChildConsumers[itm]
	/*	if !ok { // this item has not yet decided, or has not children
		ci := mcs.consItemsMap[itm]
		if ci.ConsItem.HasDecided() {
			if len(mcs.ProposalInfo[itm].GetDependentItems()) != 0 {
				panic("should have no outputs")
			}
		}
	}*/
	for _, nxt := range children {
		if !soFar[nxt] {
			ret = append(ret, nxt)
			soFar[nxt] = true
			ret = mcs.recGenChildInd(nxt, ret, maxCount, soFar)
		}
	}
	return ret
}

// GetChildIndices returns the decided children that consumed an output from this item.
func (mcs *CausalConsInterfaceState) GenChildIndices(cid types.ConsensusIndex, isUnconsumedOutput bool,
	maxCount int) (ret [][]byte, err error) {

	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	var idx types.ParentConsensusHash
	if !isUnconsumedOutput { // the index is an undecided consensus index
		idx = types.ParentConsensusHash(cid.Index.(types.ConsensusHash))
		// Check if we have the item
		if _, ok := mcs.consItemsMap[idx]; !ok && idx != mcs.initHash {
			if !mcs.storage.Contains(types.ConsensusHash(idx)) { // we do not know this index
				return nil, types.ErrInvalidIndex
			}
		}
	} else { // the index is an unconsumed output at the sender node
		// we get the consumer
		var ok bool
		if idx, ok = mcs.outputToConsumer[cid.Index.(types.ConsensusHash)]; !ok {
			return nil, types.ErrInvalidIndex
		}
	}

	var items []types.ParentConsensusHash
	if idx != mcs.initHash {
		items = append(items, idx)
	}

	items = mcs.recGenChildInd(idx, items, maxCount, make(map[types.ParentConsensusHash]bool))
	if len(items) > 50 {
		a := make(map[types.ParentConsensusHash]bool)
		for _, nxt := range items {
			a[nxt] = true
		}
	}
	// items = items[:utils.Min(len(items), maxCount)] // TODO fix this as we might never send the ones we need to decide
	for _, nxt := range items {
		// first check the disk
		binstate, _, err := mcs.storage.Read(types.ConsensusHash(nxt))
		if err != nil {
			panic(err)
		}
		if len(binstate) == 0 { // must not yet be decided, so take from memory
			binstate, err = mcs.consItemsMap[nxt].ConsItem.GetBinState(mcs.gc.NetworkType == types.RequestForwarder)
			if err != nil {
				panic(err) // TODO should Panic here?
			}
		}
		if len(binstate) == 0 {
			panic("should have bin state")
		}
		ret = append(ret, binstate)
	}
	return
}

// DoneIndex is called with consensus has finished at idx, binstate is the value decided, which is passed
// to the member checker's UpdateState method.
// It allocates the objects for new consensus instances and garbage collects old ones as necessary.
// It is concurrent safe with GetMemberChecker.
func (mcs *CausalConsInterfaceState) DoneIndex(cid types.ConsensusIndex, proposer sig.Pub, dec []byte) {

	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	decidedIndex := types.ParentConsensusHash(cid.Index.(types.ConsensusHash))

	mcs.LastDecided = decidedIndex // TODO

	// Let the member checker know we are done generating new items from it since this index decided
	// by calling DoneNextUpdateState
	// (since we don't call DoneNextUpdateState on the init item, (because it has already decided), we call it
	// once the first index has decided)
	if !mcs.calledInitDoneNext {
		mcs.calledInitDoneNext = true
		if err := mcs.initMemberChecker.DoneNextUpdateState(); err != nil {
			panic(err)
		}
	}
	consItem := mcs.consItemsMap[decidedIndex]
	if err := consItem.MC.MC.DoneNextUpdateState(); err != nil {
		panic(err)
	}

	// Let the SM know we have decided
	pi := mcs.ProposalInfo[decidedIndex]
	cItems := pi.HasDecided(proposer, cid, dec)
	pi.DoneKeep()

	// See if we have output any children
	if len(cItems) > 0 {
		// make a copy because we want the new list type (for clarity)
		childItems := make([]types.ConsensusHash, len(cItems))
		for i, newIdx := range cItems {
			childItems[i] = newIdx.ID.(types.ConsensusHash)
			// update the maps for the outputs
			mcs.childToParentMap[childItems[i]] = decidedIndex
			mcs.outputToOwner[childItems[i]] = newIdx.Pub
			// reprocess any proposals depending on the child items that we received before it was created
			mcs.reprocessProposals(childItems[i])
		}
		// update the map with he child items
		mcs.parentToChildMap[decidedIndex] = childItems
	} else { // no children so we GC directly
		// TODO safe to GC?
		mcs.parentGcIdx(decidedIndex)
	}

	// TODO do we care about denpent hashes here
	// Should only care in validate when they are consumed?
	// Then we only call decide on the proposal item that we created

	// Take the indecies that this index has consumed
	consumedIndices := mcs.itemToConsumedOutputs[decidedIndex]
	if len(consumedIndices) == 0 {
		panic("didn't find consumed indices")
	}
	for _, nextIdx := range consumedIndices {
		var ok bool
		var parentHash types.ParentConsensusHash
		if parentHash, ok = mcs.childToParentMap[nextIdx]; !ok {
			panic("didn't find parent hash")
		}

		pi := mcs.ProposalInfo[parentHash]
		if pi == nil {
			panic("could not find proposer")
		}

		// update the map to who consumed this index  (TODO find a way to do this on disk (this is used for recovery))
		mcs.outputToConsumer[nextIdx] = decidedIndex
		// add it to the parent's list of consumed items (TODO find a way to do this on disk (this is used for recovery))
		mcs.addChildConsumerToParent(parentHash, decidedIndex)

		// do garbage collection

		// remove the consumed index from the parent's to owner map
		delete(mcs.outputToOwner, nextIdx)

		// remove from the echo list since we are done with this item
		delete(mcs.consumedEcho, nextIdx)
		// remove the pointer from the consumed index to the parent since it is consumed
		delete(mcs.childToParentMap, nextIdx)
		// remove the decided item from the items supported at the parent
		parChildItems := mcs.parentToChildMap[parentHash]
		newParChildItems := parChildItems
		for i, nxtChi := range parChildItems {
			if nxtChi == nextIdx {
				newParChildItems[i] = newParChildItems[len(newParChildItems)-1]
				newParChildItems = newParChildItems[:len(newParChildItems)-1]
				break
			}
		}
		if len(parChildItems)-1 != len(newParChildItems) {
			panic("didn't find child item")
		}
		mcs.parentToChildMap[parentHash] = newParChildItems
		// If all outputs from the parent are consumed then we can GC the parent
		if len(newParChildItems) == 0 {
			// TODO safe to GC?
			mcs.parentGcIdx(parentHash)
		}
	}
	// gc the consumed map since
	delete(mcs.itemToConsumedOutputs, decidedIndex)
}

func (mcs *CausalConsInterfaceState) addChildConsumerToParent(parentHash, consumerHash types.ParentConsensusHash) {
	// add it to the parent's list of decided items (TODO find a way to do this on disk)
	parChildList := mcs.parentToChildConsumers[parentHash]
	var found bool
	for _, nxtChi := range parChildList {
		if nxtChi == consumerHash {
			found = true
			break
		}
	}
	if !found {
		mcs.parentToChildConsumers[parentHash] = append(parChildList, consumerHash)
	}
}

func (mcs *CausalConsInterfaceState) parentGcIdx(nxtIdx types.ParentConsensusHash) {
	if itm, ok := mcs.consItemsMap[nxtIdx]; ok {
		itm.ConsItem.Collect()

		mcs.ProposalInfo[nxtIdx].Collect()

		delete(mcs.consItemsMap, nxtIdx)
		if mcs.consItemSorted[itm.ProgressListIndex] != itm {
			panic("list not setup correctly")
		}
		mcs.moveToEnd(itm.ProgressListIndex, itm.ProgressListIndex+1)
		// Remove the item
		mcs.consItemSorted = mcs.consItemSorted[:len(mcs.consItemSorted)-1]

		// TODO remove check
		if !sort.IsSorted(mcs.consItemSorted) {
			panic("should be sorted")
		}
	}
	delete(mcs.ProposalInfo, nxtIdx)
	// delete(mcs.GotProposal, supportIndex)
	delete(mcs.parentToChildMap, nxtIdx)
}

func (mcs *CausalConsInterfaceState) reprocessProposals(nextIdx types.ConsensusHash) {
	lst := mcs.ProposalsToValidate[nextIdx]
	if len(lst) > 0 {
		for _, nxt := range lst {
			mcs.mainChannel.ReprocessMessage(nxt)
		}
		delete(mcs.ProposalsToValidate, nextIdx)
	}
	lst = mcs.futureMessages[nextIdx]
	if len(lst) > 0 {
		for _, nxt := range lst {
			mcs.mainChannel.ReprocessMessage(nxt)
		}
		delete(mcs.futureMessages, nextIdx)
	}
}

func (mcs *CausalConsInterfaceState) getIndex(itemIndex types.ParentConsensusHash) (item *ConsInterfaceItems) {
	if item = mcs.consItemsMap[itemIndex]; item != nil {
		// TODO is there a check that needs to happen here?
		return item
	}
	return nil
}

func (mcs *CausalConsInterfaceState) genIndex(idx types.ConsensusIndex) (*MemCheckers,
	MessageState, ForwardChecker, error) {

	itemIndex := types.ParentConsensusHash(idx.Index.(types.ConsensusHash))
	firstIndex := idx.FirstIndex.(types.ConsensusHash)
	additionalIndices := idx.AdditionalIndices

	tmp, err := types.GenerateParentHash(firstIndex, additionalIndices)
	if err != nil {
		panic(err)
	}
	if types.ParentConsensusHash(tmp.Index.(types.ConsensusHash)) != itemIndex {
		panic("invalid hash") // sanity check
	}

	if item := mcs.consItemsMap[itemIndex]; item != nil {
		// TODO is there a check that needs to happen here?
		// panic("cons item already allocated")
		return item.MC, item.MsgState, item.FwdChecker, nil
	}

	var ok bool
	var parIdx types.ParentConsensusHash
	if parIdx, ok = mcs.childToParentMap[firstIndex]; !ok {
		// panic("didn't find parent nxtIdx")
		return nil, nil, nil, types.ErrParentNotFound
	}
	var parItem *ConsInterfaceItems
	if parItem, ok = mcs.consItemsMap[parIdx]; !ok {
		panic("didn't find parent item")
	}

	// Generate the new state machine
	// get the owner
	var owner sig.Pub
	if owner, ok = mcs.outputToOwner[firstIndex]; !ok {
		panic("missing owner")
	}
	parSM := mcs.ProposalInfo[parIdx]
	smItems := make([]CausalStateMachineInterface, len(additionalIndices)+1)
	smItems[0] = parSM
	consumedIDs := make([]types.ConsensusID, len(additionalIndices)+1)
	consumedIDs[0] = firstIndex
	consumedHashes := make([]types.ConsensusHash, len(additionalIndices)+1) // same as consumedIDs, ust a different list type
	consumedHashes[0] = firstIndex

	for i, nxtIdx := range additionalIndices {

		consumedIDs[i+1] = nxtIdx
		consumedHashes[i+1] = nxtIdx.(types.ConsensusHash)
		var pi types.ParentConsensusHash
		var ok bool
		if pi, ok = mcs.childToParentMap[nxtIdx.(types.ConsensusHash)]; !ok {
			return nil, nil, nil, types.ErrParentNotFound
			// panic("didn't find parent nxtIdx")
		}
		// check the owners are all the same
		if !sig.CheckPubsEqual(owner, mcs.outputToOwner[nxtIdx.(types.ConsensusHash)]) {
			return nil, nil, nil, types.ErrNotOwner
		}
		if _, ok = mcs.consItemsMap[pi]; !ok {
			panic("didn't find parent item") // sanity check
		}
		var nxtSM CausalStateMachineInterface
		if nxtSM, ok = mcs.ProposalInfo[pi]; !ok {
			panic("didn't find SM") // sanity check
		}
		nxtSM.CheckOK()
		smItems[i+1] = nxtSM
	}

	// Tell the parent which ones is is consuming
	// TODO GC this in case it is consumed by another one
	// add it to the parent's list of decided items (TODO find a way to do this on disk)
	for _, nxt := range consumedHashes {
		pidx, ok := mcs.childToParentMap[nxt]
		if !ok {
			panic("should have parent")
		}
		mcs.addChildConsumerToParent(pidx, itemIndex)
	}

	// update the map telling what it is consuming
	mcs.itemToConsumedOutputs[itemIndex] = consumedHashes

	// Store the new SM
	mcs.ProposalInfo[itemIndex] = mcs.initSM.GenerateNewSM(consumedIDs, smItems)

	item := &ConsInterfaceItems{}
	// create the new member checkers

	memCheck := mcs.initMemberChecker.New(idx)
	specialMemCheck := mcs.initSpecialMemberChecker.New(idx)
	item.MC = &MemCheckers{memCheck, specialMemCheck}

	// create the member checker
	// first get rand bytes
	var randBytes [32]byte
	if mcs.genRandBytes {
		randBytes = mcs.ProposalInfo[parIdx].GetRand()
	}
	// inform member checker of the next consensus index that the previous has decided
	// if are created from the initial item, we use the inital state as the decided value

	var dec []byte
	if parIdx == mcs.initHash {
		dec = mcs.initSM.GetInitialState()
	} else {
		_, dec, _ = parItem.ConsItem.GetDecision()
	}
	newPubs, _ := item.MC.MC.UpdateState(owner, dec, randBytes, parItem.MC.MC, parSM)
	if newPubs != nil && item.MC.MC.IsReady() {
		panic("Member checker should not be ready if the members change until, FinishUpdateStateIsCalled")
	}
	// inform the special member checker as well
	item.MC.SMC.UpdateState(newPubs, randBytes, parItem.MC.SMC)

	// Generate the forward checker.
	item.FwdChecker = mcs.initForwardChecker.New(
		idx, memCheck.GetParticipants(), memCheck.GetAllPubs())

	// call finish update state on the member checker
	item.MC.MC.FinishUpdateState()
	if !item.MC.MC.IsReady() { // sanity check
		panic("should know members by now")
	}

	// create the new message state
	item.MsgState = mcs.initMessageState.New(idx)

	item.ConsItem = mcs.initItem.GenerateNewItem(idx, item, mcs.mainChannel, parItem.ConsItem,
		mcs.broadcastFunc, mcs.gc)
	// item.ConsItem.Start()

	// prevItem.ConsItem.SetNextConsItem(newItem.ConsItem)

	// store the item
	mcs.consItemsMap[itemIndex] = item
	item.LastProgress = time.Now()
	mcs.consItemSorted = append(mcs.consItemSorted, item)
	item.ProgressListIndex = len(mcs.consItemSorted) - 1

	// TODO remove check
	if !sort.IsSorted(mcs.consItemSorted) {
		panic("should be sorted")
	}

	// return item
	return item.MC, item.MsgState, item.FwdChecker, nil
}

// getUnconsumedOutputs returns the unconsumed outputs for the given consensus item.
func (mcs *CausalConsInterfaceState) GetUnconsumedOutputs(idx types.ParentConsensusHash) []types.ConsensusHash {
	return mcs.parentToChildMap[idx]
}

// GetTimeoutItems returns two lists of ConsInterfaceItems, each item in the lists
// has not had its LastProgress time updated for more than the config.ProgressTimeout.
// The first list contains items that have been decided, but not yet garbage collected (i.e. they still have
// unconsumed output items).
// The second contains are items that have received a valid proposal, but not yet decided (and have passed a timeout).
// The third list is the same as the second list except contains items even if they have not passed the timeout
func (mcs *CausalConsInterfaceState) GetTimeoutItems() (decidedTimeoutItems,
	startedTimeoutItems, startedItems []*ConsInterfaceItems) {

	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	now := time.Now()

	// consItemSorted has the smallest times first (i.e. the earliest)
	var startTimeoutIdx, endTimeoutIndex int
	for _, nxt := range mcs.consItemSorted {
		if now.Sub(nxt.LastProgress) >= config.ProgressTimeout*time.Millisecond {
			endTimeoutIndex++
			if nxt.ConsItem.HasDecided() {
				decidedTimeoutItems = append(decidedTimeoutItems, nxt)
				continue
			}
			if nxt.ConsItem.HasReceivedProposal() ||
				types.ParentConsensusHash(nxt.ConsItem.GetIndex().Index.(types.ConsensusHash)) == mcs.initHash {

				startedTimeoutItems = append(startedTimeoutItems, nxt)
			}
		}
		if !nxt.ConsItem.HasDecided() {
			startedItems = append(startedItems, nxt)
		}
	}
	mcs.moveToEnd(startTimeoutIdx, endTimeoutIndex)
	return
}

/*// checkProposalNeeded will check if item needs a proposal, and will request one if needed from the SM
func (cs *CausalConsInterfaceState) CheckProposalNeeded(nextIdx types.ConsensusID) {
	nextItem, err := cs.GetConsItem(nextIdx)
	if err != nil {
		panic(err)
	}

	if _, ready := nextItem.GetProposalIndex(); ready && !cs.GotProposal[nextIdx] {
		pi := cs.ProposalInfo[nextIdx]
		pi.GetProposal()
		cs.GotProposal[nextIdx] = true
	}
}
*/
// GetCausalDecisions returns the decided values in causal order tree.
func (mcs *CausalConsInterfaceState) GetCausalDecisions() (root *utils.StringNode, orderedDecisions [][]byte) {
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	mcs.storage.Range(
		func(key interface{}, binstate, dec []byte) bool {
			orderedDecisions = append(orderedDecisions, dec)
			return true
		})

	createdMap := make(map[types.ParentConsensusHash]bool)
	return mcs.recTraverse(types.ParentConsensusHash(mcs.initSM.GetIndex().Index.(types.ConsensusHash)), createdMap), orderedDecisions
}

func (mcs *CausalConsInterfaceState) recTraverse(idx types.ParentConsensusHash,
	createdMap map[types.ParentConsensusHash]bool) (node *utils.StringNode) {

	st, dec, err := mcs.storage.Read(types.ConsensusHash(idx))
	if err != nil {
		panic(err)
	}
	node = &utils.StringNode{Value: string(mcs.initItem.ComputeDecidedValue(st, dec))}
	items := mcs.parentToChildConsumers[idx]
	for _, nxt := range items {
		if _, ok := createdMap[nxt]; !ok {
			node.Children = append(node.Children, mcs.recTraverse(nxt, createdMap))
		}
	}
	sort.Sort(node.Children)
	return
}

// Collect is called when the item is being garbage collected.
func (mcs *CausalConsInterfaceState) Collect() {
	mcs.mutex.Lock()
	defer mcs.mutex.Unlock()

	for _, nxt := range mcs.consItemsMap {
		nxt.ConsItem.Collect()
	}
	for _, nxt := range mcs.ProposalInfo {
		nxt.EndTest()
		nxt.Collect()
	}
}
