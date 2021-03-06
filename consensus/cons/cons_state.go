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

package cons

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/storage"
	"github.com/tcrain/cons/consensus/utils"
)

// ConsState runs and stores the details of the consensus, running the main loop.
// It runs all the generic code not specific to any consensus algorithm.
type ConsState struct {
	generalConfig *generalconfig.GeneralConfig // General configuration input to all consensus intems.

	localIndexTime time.Time              // Keeps track of the last time the current consensus index made progress, in case of no progress after generalconfig.ProgressTimeout the we start recovery.
	initItem       consinterface.ConsItem // Consensus item used to generate all new consensus items.

	memberCheckerState     *consinterface.ConsInterfaceState                 // The member checker state object.
	futureMessages         map[types.ConsensusInt][]*channelinterface.RcvMsg // Map of messages by their consensus index that were too far in the future to be processed yet.
	futureMessagesMinIndex types.ConsensusInt                                // Max index of messages that have been processed
	store                  storage.StoreInterface                            // Interface to the storage
	mainChannel            channelinterface.MainChannel                      // Interface to the network.
	isInStorageInit        bool                                              // Used during initialization, when recovering from disk this is set to true so we don't store messages we already have on disk.
	allowConcurrent        types.ConsensusInt                                // Number of concurrent consensus instances allowed to be run
	maxInstances           types.ConsensusInt                                // Maximum number of consensus instances to run
	decidedLastIndex       bool                                              // set to true when decided up to maxInstances
	timeoutTime            time.Duration                                     // The timeout used after no proogress to start recovery.
	firstItem              consinterface.ConsItem
}

// NetConsState initializes and returns a new consensus state object.
func NewConsState(initIitem consinterface.ConsItem,
	generalConfig *generalconfig.GeneralConfig,
	memberCheckerState *consinterface.ConsInterfaceState,
	store storage.StoreInterface,
	maxInstances types.ConsensusInt,
	allowConcurrent types.ConsensusInt,
	initStateMachine consinterface.StateMachineInterface, mainChannel channelinterface.MainChannel) *ConsState {

	cs := &ConsState{}
	cs.maxInstances = maxInstances
	cs.allowConcurrent = allowConcurrent
	cs.mainChannel = mainChannel
	cs.localIndexTime = time.Now()
	cs.memberCheckerState = memberCheckerState
	cs.timeoutTime = time.Millisecond * time.Duration(generalConfig.ProgressTimeout)

	if initIitem.NeedsCompletionConcurrentProposals() > 1 && allowConcurrent > 0 {
		panic("cannot have both needs concurrent and allow concurrent")
	}

	// Initialize the consensus items, since we reuse them.
	cs.generalConfig = generalConfig
	cs.initItem = initIitem

	cs.futureMessagesMinIndex = 1

	firstIdx := types.SingleComputeConsensusIDShort(1)

	firstItem, err := cs.memberCheckerState.GetMemberChecker(firstIdx)
	if err != nil { // sanity check
		panic(err)
	}
	cs.firstItem = firstItem.ConsItem
	firstItem.ConsItem.SetInitialState(initStateMachine.GetInitialState(), store)
	firstItem.ConsItem.PrevHasBeenReset()

	cs.futureMessages = make(map[types.ConsensusInt][]*channelinterface.RcvMsg)
	cs.store = store

	// Load from disk
	cs.isInStorageInit = true
	memberCheckerState.IsInStorageInit = true
	cs.mainChannel.StartInit()

	// We make an initial proposal, but it will not be used if we have decided a value
	memberCheckerState.SetInitSM(initStateMachine)
	if prvIdx, ready := firstItem.ConsItem.GetProposalIndex(); ready {
		if prvIdx.Index.(types.ConsensusInt) != 0 {
			panic("should be 0")
		}
		cs.memberCheckerState.ProposalInfo[1].GetProposal()
		cs.memberCheckerState.GotProposal[1] = true
	}

	idx := types.ConsensusInt(0)
	logging.Info("loading from disk")
	for true {
		idx++
		// load the next consensus index from disk
		binstate, _, err := cs.store.Read(idx)
		if err != nil {
			logging.Errorf("Unable to read idx from disk %v, err %v", idx, err)
			break
		}
		if len(binstate) == 0 {
			break
		}
		msg := messages.NewMessage(binstate)
		_, err = msg.PopMsgSize()
		if err != nil {
			panic(err)
		}
		// deserialize the stored messages
		deserItems, errors := consinterface.UnwrapMessage(sig.FromMessage(msg),
			types.IntIndexFuns, types.LoadedFromDiskMessage,
			cs.initItem, memberCheckerState)
		if errors != nil {
			logging.Error("Error reading from disk: ", errors)
			// continue
		}
		for _, nxt := range deserItems {
			if nxt.Index.Index.(types.ConsensusInt) != idx {
				// panic(fmt.Sprint(nxt.Index, idx))
				logging.Infof("loaded message index %v from stored index of %v, this may be an error depending"+
					"on the consensus type (is not an error for MvCons4)", nxt.Index.Index.(types.ConsensusInt), idx)
			}
		}
		// reply the messages
		if len(deserItems) > 0 {
			_, errs := cs.ProcessMessage(&channelinterface.RcvMsg{CameFrom: 7, Msg: deserItems, SendRecvChan: nil, IsLocal: true})
			if len(errs) > 0 {
				panic(errs)
			}
		}
	}
	logging.Info("loaded from disk until", idx-1, cs.generalConfig.TestIndex)
	// done with initialization
	if idx > 1 { // We have restarted from disk so we will stay in initialization mode until we can recover from others.
		cs.sendNoProgressMsg() // update others on our current state
	}
	cs.finishInitialization()

	return cs
}

// finishInitialization is called once the consensus is ready to start sending it's own messages
func (cs *ConsState) finishInitialization() {
	// we can now start new storage
	cs.isInStorageInit = false
	cs.memberCheckerState.IsInStorageInit = false
	cs.mainChannel.EndInit()
	// cs.checkProgress(cs.memberCheckerState.LocalIndex)
}

func (cs *ConsState) Start() {
	if cs.memberCheckerState.LocalIndex == cs.firstItem.GetIndex().Index {
		cs.firstItem.Start(false)
	}
	cs.checkProgress(cs.memberCheckerState.LocalIndex)
}

// ProcessLocalMessage should be called when a message sent from the local node is ready to be processed.
func (cs *ConsState) ProcessLocalMessage(rcvMsg *channelinterface.RcvMsg) {
	var unprocessed []*deserialized.DeserializedItem

	// Loop through the messages.
	// Only messages of type cs.items[0].GetProposeHeaderID() are processed here, as they are only value when coming from the local node.
	// Other messages are then just processed in the normal path cs.ProcessMessage.
	// New message types can be added here as needed.
	for _, deser := range rcvMsg.Msg {
		// only propose msgs for now
		logging.Info("Processing local message type: ", deser.HeaderType, deser.Index)
		if deser.HeaderType == cs.initItem.GetProposeHeaderID() {
			if deser.Index.Index.(types.ConsensusInt) < cs.memberCheckerState.LocalIndex {
				// This can happen if we decide before the proposal is processed
				logging.Warningf("Got old proposal index %v, expected %v", deser.Index, cs.memberCheckerState.LocalIndex)
				//continue
			}

			// Inform the consensus that a proposal has been received from the local state machine.
			ci, err := cs.memberCheckerState.GetMemberChecker(deser.Index)
			if err != nil {
				logging.Info(err)
				continue
			}
			err = ci.ConsItem.GotProposal(deser.Header, cs.mainChannel)
			if err != nil {
				panic(err)
			}
			// Do any background tasks.
			cs.checkProgress(deser.Index.Index)
		} else {
			unprocessed = append(unprocessed, deser)
		}
	}
	if len(unprocessed) > 0 {
		// remaining messages are processed using cs.ProcessMessage
		rcvMsg.Msg = unprocessed
		returnMsg, err := cs.ProcessMessage(rcvMsg)
		if returnMsg != nil {
			logging.Warning("Got a return message from a local message", unprocessed[0].HeaderType)
		}
		if err != nil {
			logging.Warning("Got an error from a local msg: ", err)
		}
	}
}

func (cs *ConsState) sendNoProgressMsg() {
	var hdr messages.MsgHeader

	// Need to send progress requests
	i := cs.memberCheckerState.LocalIndex
	idx := types.SingleComputeConsensusIDShort(i)
	logging.Info("Sending no progress message for index", i, cs.generalConfig.TestIndex)

	cs.localIndexTime = time.Now()
	idxItem, err := cs.memberCheckerState.GetMemberChecker(idx)
	if err != nil {
		panic("local index error")
	}
	idxItem.MC.MC.GetStats().AddProgressTimeout()

	hdr = idxItem.ConsItem.GetCustomRecoverMsg(false)
	// if hdr = idxItem.ConsItem.GetCustomRecoverMsg(false); hdr != nil { // custom recovery message
	// } else { // just use the normal recovery message
	//	hdr = messagetypes.NewNoProgressMessage(idx, false, cs.generalConfig.TestIndex)
	// }

	mm, err := messages.CreateMsgSingle(hdr)
	if err != nil {
		panic(err)
	}
	cs.mainChannel.SendAlways(mm.GetBytes(), false, idxItem.FwdChecker.GetNoProgressForwardFunc(), true, nil)
}

// ProcessMessage should be called when a message sent from an external node is ready to be processed.
// It returns a list of messages to be sent back to the sender of the original message (if any).
func (cs *ConsState) ProcessMessage(rcvMsg *channelinterface.RcvMsg) (returnMsg [][]byte, returnErrs []error) {

	defer func() {
		// After we have finished processing the messages, we do the following
		// (1) send any messages waiting to be forwarded
		// (2) if we have not made progress after a timeout, start a recovery

		// we only execute the loop for the current index if the item only needs 1 concurrent instance
		var runLoopOnce bool
		if cs.initItem.NeedsCompletionConcurrentProposals() <= 1 && !cs.initItem.ForwardOldIndices() {
			runLoopOnce = true
		}

		var forwarded bool
		var i types.ConsensusInt
		if cs.initItem.ForwardOldIndices() { // start from the oldest remaining index
			i = cs.memberCheckerState.GetOldestMemberCheckerIdx()
		} else {
			i = cs.memberCheckerState.LocalIndex // start from the most recent undecided index
		}
		for ; i <= cs.memberCheckerState.StartedIndex; i++ {
			// forward any messages waiting
			if sendForward(types.SingleComputeConsensusIDShort(i), cs.memberCheckerState, cs.mainChannel) {
				forwarded = true
			}
			if forwarded && runLoopOnce {
				break
			}
		}
		if forwarded {
			// Since we forwarded a message we consider that we have made progress so we return without sending a no progress message
			// TODO should always do this?
			return
		}
		// If we haven't made progress after a timeout, inform our neighbors
		if time.Since(cs.localIndexTime) > cs.timeoutTime && !cs.isInStorageInit { //&& cs.memberCheckerState.LocalIndex <= cs.maxInstances {
			cs.sendNoProgressMsg()
		}
	}()

	msg := rcvMsg.Msg

	if msg == nil { // if the message is nil it was becase of a timeout in the recv loop, so we just run the defered function.
		return nil, nil
	}

	// Process each message one by one
	for _, deser := range msg {

		ht := deser.HeaderType
		endidx := deser.Index.Index.(types.ConsensusInt)
		myIndex := cs.memberCheckerState.LocalIndex

		// if endidx > cs.maxInstances {
		//	continue
		// }

		// Check if it is a no change message
		if cs.initItem.GetRecoverMsgType() == messages.HdrNoProgress && ht == messages.HdrNoProgress {
			// The external node hasn't made any progress, so if we are more advanced than it,
			// then we send it what we have
			// var returnMessages [][]byte
			// For each index that it is missing we send back the messages we have received for that index
			stopAt := utils.MinConsensusIndex(endidx+config.MaxRecovers, myIndex+cs.initItem.NeedsCompletionConcurrentProposals(),
				cs.memberCheckerState.StartedIndex+1)
			logging.Infof("Got a header no change for idx %v, my idx is %v, started index is %v, will send until %v, my test id %v, sender test id is %v",
				endidx, myIndex, cs.memberCheckerState.StartedIndex, stopAt, cs.generalConfig.TestIndex, deser.Header.(*messagetypes.NoProgressMessage).TestID)

			// ("Got a header no change for idx %v, my idx is %v, started index is %v, will send until %v\n", endidx, myIndex, cs.startedIndex, stopAt)

			// conv, _ := consinterface.ConvertIndex(cs.localIndex, cs.localIndex, cs.itemsIndex)
			// cs.items[conv].PrintState()

			idxItem, err := cs.memberCheckerState.GetMemberChecker(types.SingleComputeConsensusIDShort(myIndex))
			utils.PanicNonNil(err)
			stats := idxItem.MC.MC.GetStats()
			for i := endidx; i < stopAt; i++ {
				// First cons starts at 1, so can skip 0
				if i == 0 {
					continue
				}
				stats.AddForwardState()
				if i < myIndex {
					// Recover from disk
					binstate, _, err := cs.store.Read(i)
					if err != nil {
						panic(err)
					}
					returnMsg = append(returnMsg, binstate)
				} else if i >= myIndex && i <= myIndex+config.KeepFuture {
					idx := types.SingleComputeConsensusIDShort(i)
					idxItem, err := cs.memberCheckerState.GetMemberChecker(idx)
					if err != nil {
						panic(err)
					}
					if idxItem.MsgState.GetIndex().Index.(types.ConsensusInt) != i {
						panic("invalid index")
					}
					bs, err := idxItem.ConsItem.GetBinState(cs.generalConfig.NetworkType == types.RequestForwarder)
					if err != nil {
						logging.Error(err)
					} else {
						returnMsg = append(returnMsg, bs)
					}
				} else {
					// append nothing
				}
			}
			// return returnMessages, nil
			continue
		} else if cs.initItem.GetRecoverMsgType() == ht { // custom recovery message type
			item, err := cs.memberCheckerState.GetMemberChecker(types.SingleComputeConsensusIDShort(cs.memberCheckerState.LocalIndex))
			utils.PanicNonNil(err)
			item.ConsItem.ProcessCustomRecoveryMessage(deser, rcvMsg.SendRecvChan)
			continue
		}

		// Process the message
		if endidx < types.ConsensusInt(utils.SubOrZero(uint64(myIndex), uint64(cs.generalConfig.KeepPast))) { // An old message
			// otherwise the message was too old to be processed, so we just drop it
			returnErrs = append(returnErrs, types.ErrIndexTooOld)
			continue
		} else if endidx > cs.memberCheckerState.StartedIndex+config.KeepFuture { // A message for a future index
			logging.Infof("Got a message for future idx %v, my started idx is %v",
				endidx, cs.memberCheckerState.StartedIndex)
			if endidx > cs.memberCheckerState.StartedIndex+config.DropFuture {
				// The message is too far in the future to be processed so we just drop it
				logging.Info("Drop future")
				returnErrs = append(returnErrs, types.ErrIndexTooNew)
				continue
			}
			// We store the message to be processed later
			cs.futureMessages[endidx] = append(cs.futureMessages[endidx],
				&channelinterface.RcvMsg{CameFrom: 8, Msg: []*deserialized.DeserializedItem{deser}, SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
		} else { // A message for one of the recent consensus indecies
			endIdxItem := types.SingleComputeConsensusIDShort(endidx)
			idxItem, err := cs.memberCheckerState.GetMemberChecker(endIdxItem)
			if err != nil {
				panic(err)
			}

			if !deser.IsDeserialized { // The message is not deserialized
				if idxItem.MC.MC.IsReady() { // If the member checker is ready we can retry deserializing the message
					cs.mainChannel.ReprocessMessage(&channelinterface.RcvMsg{CameFrom: 9, Msg: []*deserialized.DeserializedItem{deser}, SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
				} else {
					if endidx == myIndex {
						panic("should know members by now")
					}
					// We cannot process the message yet, so we store it to be processed later
					cs.futureMessages[endidx] = append(cs.futureMessages[endidx],
						&channelinterface.RcvMsg{CameFrom: 10, Msg: []*deserialized.DeserializedItem{deser}, SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
				}
			} else {
				if deser.MC != idxItem.MC { // we have changed the member checker since we made the item
					// this can happen when using a cons type that piggybacks messages from earlier consensus indices which
					// may cause the members to change (i.e. MvCons3)
					logging.Info("changed the member checker since deserializing this item")
					continue
				}
				var shouldForward, progress bool
				if deser.HeaderType == messages.HdrPartialMsg { // Partial messages are not processed by the consensus, but we do want to forward them
					shouldForward = true
					progress = false
				} else { // Normal messages are processed by the consensus
					// The message is deserialized so we send it to the consensus to be processed
					readyToProcess, err := cs.memberCheckerState.CheckValidateProposal(
						deser, rcvMsg.SendRecvChan, rcvMsg.IsLocal)
					if err != nil {
						returnErrs = append(returnErrs, err)
						continue
					}
					if !readyToProcess {
						continue // the MemberCheckerState object is responsible for reprocessing messages so we just continue.
					}
					// start the stats if needed (this is also called when we call Start on the cons item,
					// but we call it here because we can process messages before we call start)
					idxItem.MC.MC.GetStats().AddStartTime()
					cs.memberCheckerState.SharedLock.Lock()
					progress, shouldForward = idxItem.ConsItem.ProcessMessage(deser, rcvMsg.IsLocal, rcvMsg.SendRecvChan)
					cs.memberCheckerState.SharedLock.Unlock()
				}
				// add the message to the forward checker queue
				if !rcvMsg.IsLocal { // Dont forward local messages, since they are broadcast within the cons
					checkForward(rcvMsg.SendRecvChan, deser, shouldForward, cs.memberCheckerState, cs.mainChannel)
				}
				if progress {
					// If we made progess we might have to advance the consensus index
					cs.checkProgress(endidx)
				}
			}
		}
	}
	return //nil, nil
}

// checkProgress is called after a message is processed that made progress in the consensus.
// It upadates any state in case decisions were made.
func (cs *ConsState) checkProgress(cidx types.ConsensusID) {
	idx := cidx.(types.ConsensusInt)

	var finishedLastRound bool // set to true if the last round decided
	// only need to check for progress if the idx is at least as big as the current index
	if idx >= cs.memberCheckerState.LocalIndex {
		// Check if the current index has a pre-decision ready
		cs.memberCheckerState.CheckPreDecision()

		// Check if the current index needs a proposal
		cs.memberCheckerState.CheckProposalNeeded(idx)
	}

	// start from the current index and check if can advance the state
	item, err := cs.memberCheckerState.GetMemberChecker(types.SingleComputeConsensusIDShort(cs.memberCheckerState.LocalIndex))
	utils.PanicNonNil(err)
	nextIdx := cs.memberCheckerState.LocalIndex
	nextIdxItem := types.SingleComputeConsensusIDShort(nextIdx)
	nextItem := item
	cs.memberCheckerState.CheckProposalNeeded(nextIdx)
	// Loop on any decided instances
	for nextItem.ConsItem.HasDecided() {
		// memC, _, fwdChecker, err := cs.memberCheckerState.GetMemberChecker(nextIdxItem)
		// if err != nil { // sanity check
		//	panic(fmt.Sprint(err, nextIdx, cs.memberCheckerState.LocalIndex))
		//}

		if nextIdx >= cs.maxInstances {
			cs.decidedLastIndex = true
		}

		bs, err := nextItem.ConsItem.GetBinState(false)
		if err != nil {
			panic(err)
		}
		proposer, dec, supportIndex, futureDependentIndex := nextItem.ConsItem.GetDecision()
		_, preProposer, preDec, nextReady := nextItem.ConsItem.GetNextInfo()
		if len(preDec) > 0 && !nextReady {
			panic("should be ready after decision")
		}
		// If both dec and preDec are non nil then they must be equal
		if len(preDec) > 0 && len(dec) > 0 {
			if !bytes.Equal(dec, preDec) {
				panic("preDec and dec should be equal after commit")
			}
			if proposer != nil && preProposer != nil {
				str, err := proposer.GetPubString()
				if err != nil {
					panic(err)
				}
				pstr, err := preProposer.GetPubString()
				if err != nil {
					panic(err)
				}
				if str != pstr {
					panic("got different proposers")
				}
			}
		}

		// Forward any additional items
		nextItem.FwdChecker.ConsDecided(nextItem.MC.MC.GetStats())
		sendForward(types.SingleComputeConsensusIDShort(nextIdx), cs.memberCheckerState, cs.mainChannel)

		// Store the decided value
		// Update the storage if we are not initializing
		if !cs.isInStorageInit {
			err := cs.store.Write(nextIdx, bs, dec)
			if err != nil {
				panic(err)
			}
			nextItem.MC.MC.GetStats().DiskStore(len(bs) + len(dec))
		}

		// Update the member checker
		finishedLastRound = cs.memberCheckerState.DoneIndex(nextIdx, supportIndex.Index, futureDependentIndex.Index, proposer, dec)
		logging.Infof("Decided cons state index %v", nextIdx)

		// Update the state index
		nextIdx++
		nextIdxItem = types.SingleComputeConsensusIDShort(nextIdx)

		newNextItem, err := cs.memberCheckerState.GetMemberChecker(nextIdxItem)
		if err != nil {
			panic(err)
		}
		if !newNextItem.MC.MC.IsReady() {
			panic("mc should be ready")
		}

		// Give the next instance the CommitProof from the instance
		var prf []messages.MsgHeader
		if cs.generalConfig.CollectBroadcast != types.Full || cs.generalConfig.IncludeProofs {
			prf = nextItem.ConsItem.GetCommitProof()
			if len(prf) > 0 {
				newNextItem.ConsItem.SetCommitProof(prf)
			}
		}

		// we are done with the previous instance
		nextItem = newNextItem

		// Start the next cons instance if ready
		if cs.memberCheckerState.StartedIndex < cs.memberCheckerState.LocalIndex {
			if cs.memberCheckerState.StartedIndex != cs.memberCheckerState.LocalIndex-1 {
				panic("bad order of starting consensus")
			}
			//cs.memberCheckerState.
			cs.memberCheckerState.IncrementStartedIndex()
			logging.Info("Starting cons index ", cs.memberCheckerState.StartedIndex)
			if cs.memberCheckerState.StartedIndex != nextIdx {
				panic(1)
			}
			//pi := cs.proposalInfo[prvIdx].StartIndex(cs.startedIndex)
			//pi.GetProposal()
			//cs.proposalInfo[cs.startedIndex] = pi
			startIdx, err := cs.memberCheckerState.GetMemberChecker(
				types.SingleComputeConsensusIDShort(cs.memberCheckerState.LocalIndex))
			if err != nil {
				panic(err)
			}
			startIdx.ConsItem.Start(finishedLastRound)
		}

		// If this was the last round then we broadcast the commit proof so all can finish (since we won't
		// broadcast an init for the next round).
		if finishedLastRound &&
			(cs.generalConfig.CollectBroadcast != types.Full ||
				(cs.generalConfig.IncludeProofs && cs.generalConfig.StopOnCommit == types.Immediate)) {

			nxtCoordPub, commitProof := nextItem.ConsItem.GetPrevCommitProof()
			if cs.generalConfig.CollectBroadcast == types.Full { // we always broadcast
				nxtCoordPub = nil
			}
			// only if we are the expected coordinator for the next round
			if nxtCoordPub == nil || sig.CheckPubsEqual(nextItem.MC.MC.GetMyPriv().GetPub(), nxtCoordPub) {
				cs.mainChannel.SendHeader(messages.AppendCopyMsgHeader(nextItem.ConsItem.GetPreHeader(),
					commitProof...), false, false, nextItem.FwdChecker.GetNewForwardListFunc(),
					true, nil)
			}
		}

		// Check if the index needs a proposal
		cs.memberCheckerState.CheckProposalNeeded(nextIdx)

		// (2) Update the state of the next item
		// Process any messages in case membership was not ready from before
		if val, ok := cs.futureMessages[nextIdx]; ok {
			for _, rcvMsg := range val {
				cs.mainChannel.ReprocessMessage(rcvMsg)
			}
			delete(cs.futureMessages, nextIdx)
		}
		if cs.futureMessagesMinIndex <= nextIdx {
			cs.futureMessagesMinIndex = nextIdx + 1
		}
	}

	// now start concurrent instances if enabled, or if the cons needs concurrent instances
	if !cs.isInStorageInit && (cs.initItem.NeedsCompletionConcurrentProposals() > 1 || cs.allowConcurrent > 0) {

		// for cs.startedIndex < cs.memberCheckerState.LocalIndex ||

		// Continue until up to KeepFuture
		// Or until the max concurrent instances allowed
		for cs.memberCheckerState.StartedIndex < cs.memberCheckerState.LocalIndex+config.KeepFuture &&
			(cs.memberCheckerState.StartedIndex < cs.memberCheckerState.LocalIndex+cs.allowConcurrent-1 ||
				cs.initItem.NeedsCompletionConcurrentProposals() > 1) {

			// if the current started index is not ready to start the next then we dont continue
			nxtItem, err := cs.memberCheckerState.GetMemberChecker(
				types.SingleComputeConsensusIDShort(cs.memberCheckerState.StartedIndex))
			if err != nil {
				panic(err)
			}
			if !nxtItem.ConsItem.CanStartNext() { // !cs.itemMap[cs.startedIndex].CanStartNext() {
				break
			}
			// The previous started index is ready to start the next index, so
			// as long as the member checker of the next index is ready then we can start
			//mc, _, _, err := cs.memberCheckerState.GetMemberChecker(
			//	types.SingleComputeConsensusIDShort(cs.memberCheckerState.StartedIndex + 1))
			//if err != nil {
			//	panic(err)
			//}
			if !nxtItem.MC.MC.IsReady() {
				break
			}

			// Get information about what the previous started index can decide
			cs.memberCheckerState.CheckPreDecision()
			// start the next index
			cs.memberCheckerState.IncrementStartedIndex()
			logging.Info("Starting index ", cs.memberCheckerState.StartedIndex)
			startItem, err := cs.memberCheckerState.GetMemberChecker(
				types.SingleComputeConsensusIDShort(cs.memberCheckerState.StartedIndex))
			if err != nil {
				panic(err)
			}
			cs.memberCheckerState.SharedLock.Lock()
			startItem.ConsItem.Start(finishedLastRound)
			cs.memberCheckerState.SharedLock.Unlock()

			// See the new index needs a proposal
			cs.memberCheckerState.CheckPreDecision()
			cs.memberCheckerState.CheckProposalNeeded(cs.memberCheckerState.StartedIndex)
		}
	}

	for ; cs.futureMessagesMinIndex <= cs.memberCheckerState.StartedIndex+config.KeepFuture; cs.futureMessagesMinIndex++ {

		if val, ok := cs.futureMessages[cs.futureMessagesMinIndex]; ok {
			futureItem, err := cs.memberCheckerState.GetMemberChecker(
				types.SingleComputeConsensusIDShort(cs.futureMessagesMinIndex))
			if err != nil {
				panic(err)
			}
			// Process any future messages for that item
			if !futureItem.MC.MC.IsReady() {
				break
			}
			for _, rcvMsg := range val {
				cs.mainChannel.ReprocessMessage(rcvMsg)
			}
			delete(cs.futureMessages, cs.futureMessagesMinIndex)
		}
	}

	// if idx <= cs.localIndex {
	if idx < cs.memberCheckerState.LocalIndex+cs.initItem.NeedsCompletionConcurrentProposals() { // TODO how should this work for MvCons3?
		// we have made some sort of progress so update time
		cs.localIndexTime = time.Now()
	}
}

// SMStatsString prints the statistics of the state machine.
func (cs *ConsState) SMStatsString(testDuration time.Duration) string {
	return cs.memberCheckerState.ProposalInfo[cs.memberCheckerState.LocalIndex-1].StatsString(testDuration)
}

// SMStats returns the stats object of the state machine.
func (cs *ConsState) SMStats() consinterface.SMStats {
	return cs.memberCheckerState.ProposalInfo[cs.memberCheckerState.LocalIndex-1].GetSMStats()
}

// Collect is called when the process is terminating.
func (cs *ConsState) Collect() {
	cs.memberCheckerState.Collect()
}
