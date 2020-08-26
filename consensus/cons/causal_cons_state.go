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
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	config2 "github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"time"

	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/storage"
)

// ConsState runs and stores the details of the consensus, running the main loop.
// It runs all the generic code not specific to any consensus algorithm.
type CausalConsState struct {
	generalConfig *config2.GeneralConfig // General configuration input to all consensus intems.

	localIndexTime time.Time              // Keeps track of the last time the current consensus index made progress, in case of no progress after generalconfig.ProgressTimeout the we start recovery.
	initItem       consinterface.ConsItem // Consensus item used to generate all new consensus items.

	memberCheckerState *consinterface.CausalConsInterfaceState // The member checker state object.

	store           storage.StoreInterface       // Interface to the storage
	mainChannel     channelinterface.MainChannel // Interface to the network.
	isInStorageInit bool                         // Used during initialization, when recovering from disk this is set to true so we don't store messages we already have on disk.
	timeoutTime     time.Duration                // The timeout used after no proogress to start recovery.
}

// var timeoutTime = time.Millisecond * time.Duration(config.ProgressTimeout) // The timeout used after no proogress to start recovery.

// NetConsState initializes and returns a new consensus state object.
func NewCausalConsState(initIitem consinterface.ConsItem,
	generalConfig *config2.GeneralConfig,
	memberCheckerState *consinterface.CausalConsInterfaceState,
	store storage.StoreInterface,
	initStateMachine consinterface.CausalStateMachineInterface,
	mainChannel channelinterface.MainChannel) *CausalConsState {

	cs := &CausalConsState{}
	cs.mainChannel = mainChannel
	cs.localIndexTime = time.Now()
	cs.memberCheckerState = memberCheckerState
	cs.timeoutTime = time.Millisecond * time.Duration(generalConfig.ProgressTimeout)

	if initIitem.NeedsConcurrent() > 1 {
		panic("cannot have needs concurrent for causal")
	}

	// Initialize the consensus items, since we reuse them.
	cs.generalConfig = generalConfig
	cs.initItem = initIitem

	memberCheckerState.InitSM(initStateMachine)

	cs.store = store

	// Load from disk
	cs.isInStorageInit = true
	memberCheckerState.IsInStorageInit = true
	cs.mainChannel.StartInit()

	initStateMachine.StartInit(cs.memberCheckerState)

	var loadedCount int
	logging.Info("loading from disk")
	cs.store.Range(
		func(key interface{}, binstate, dec []byte) bool {

			msg := messages.NewMessage(binstate)
			_, err := msg.PopMsgSize()
			if err != nil {
				panic("other")
			}
			// deserialize the stored messages
			deserItems, errors := consinterface.UnwrapMessage(sig.FromMessage(msg),
				types.HashIndexFuns, types.LoadedFromDiskMessage,
				cs.initItem, memberCheckerState)
			if errors != nil {
				logging.Error("Error reading from disk: ", errors)
				// continue
			}
			for _, nxt := range deserItems {
				if nxt.Index.Index != types.ConsensusHash(key.(types.HashStr)) {
					panic(fmt.Sprint(nxt.Index, key))
				}
			}
			// reply the messages
			if len(deserItems) > 0 {
				_, errs := cs.ProcessMessage(&channelinterface.RcvMsg{CameFrom: 7, Msg: deserItems, SendRecvChan: nil, IsLocal: true})
				if len(errs) > 0 {
					logging.Error(errs)
				}
			}
			loadedCount++
			return true
		})
	logging.Info("loaded from disk until", loadedCount-1, cs.generalConfig.TestIndex)
	// done with initialization
	if itm := cs.memberCheckerState.GetLastDecidedItem(); itm != nil && loadedCount > 1 { // We have restarted from disk so we will stay in initialization mode until we can recover from others.
		cs.sendNoProgressMsg(itm.ConsItem.GetIndex(), itm, true) // update others on our current state
	}
	cs.finishInitialization()

	return cs
}

// finishInitialization is called once the consensus is ready to start sending it's own messages
func (cs *CausalConsState) finishInitialization() {
	// we can now start new storage
	cs.isInStorageInit = false
	cs.memberCheckerState.IsInStorageInit = false
	cs.mainChannel.EndInit()
	// cs.checkProgress(cs.memberCheckerState.LocalIndex)
}

// ProcessLocalMessage should be called when a message sent from the local node is ready to be processed.
func (cs *CausalConsState) ProcessLocalMessage(rcvMsg *channelinterface.RcvMsg) {
	var unprocessed []*channelinterface.DeserializedItem

	// Loop through the messages.
	// Only messages of type cs.items[0].GetProposeHeaderID() are processed here, as they are only value when coming from the local node.
	// Other messages are then just processed in the normal path cs.ProcessMessage.
	// New message types can be added here as needed.
	for _, deser := range rcvMsg.Msg {
		// only propose msgs for now
		logging.Info("Processing local message type: ", deser.HeaderType, deser.Index)
		if deser.HeaderType == cs.initItem.GetProposeHeaderID() {

			// Compute the new consensus index
			mvHdr := deser.Header.(*messagetypes.MvProposeMessage)

			// If we have already decided this index then we drop the msg
			if cs.store.Contains(mvHdr.Index.Index) {
				logging.Warningf("Got old proposal index %v, last decided %v", deser.Index, cs.memberCheckerState.LastDecided)
				// panic(mvHdr.Index.Index) // TODO remove
				continue
			}

			// Generate the index if it hasn't been done already
			_, _, _, err := cs.memberCheckerState.CheckGenMemberChecker(mvHdr.Index)
			if err != nil {
				panic(err)
			}

			// Inform the consensus that a proposal has been received from the local state machine.
			ci, err := cs.memberCheckerState.GetConsItem(mvHdr.Index)
			if err != nil {
				panic(err)
				// logging.Info(err)
				// continue
			}
			if !ci.HasStarted() {
				ci.Start()
			}
			err = ci.GotProposal(deser.Header, cs.mainChannel)
			if err != nil {
				panic(err)
			}
			// Do any background tasks.
			cs.checkProgress(mvHdr.Index)
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
		for _, nxtErr := range err {
			switch nxtErr {
			case nil:
				break
			case types.ErrInvalidIndex, types.ErrIndexTooOld:
				logging.Warning("Got an error from a local msg: ", err)
			default:
				panic(err)
			}
		}
	}
}

func (cs *CausalConsState) sendNoProgressMsg(idx types.ConsensusIndex, item *consinterface.ConsInterfaceItems,
	hasDecided bool) {

	// update time
	cs.localIndexTime = time.Now()

	var hdrs []messages.MsgHeader

	// Need to send progress requests
	// idx := item.ConsItem.GetIndex()

	if !hasDecided {
		// i := types.ConsensusHash(cs.memberCheckerState.LastDecided)
		logging.Info("Sending no progress message for index", idx, cs.generalConfig.TestIndex)
		hdrs = append(hdrs, messagetypes.NewNoProgressMessage(idx, hasDecided, cs.generalConfig.TestIndex))
	} else {
		// If it has decided then we send the unconsumed indices for that decided index
		for _, nxt := range cs.memberCheckerState.GetUnconsumedOutputs(types.ParentConsensusHash(idx.Index.(types.ConsensusHash))) {
			hdrs = append(hdrs, messagetypes.NewNoProgressMessage(
				types.ConsensusIndex{Index: nxt}, hasDecided, cs.generalConfig.TestIndex))
		}
	}

	mm, err := messages.CreateMsg(hdrs)
	if err != nil {
		panic(err)
	}
	cs.mainChannel.SendAlways(mm.GetBytes(), false, item.FwdChecker.GetNoProgressForwardFunc(), true)

	if !hasDecided {
		bs, err := item.ConsItem.GetBinState(cs.generalConfig.NetworkType == types.RequestForwarder)
		if err != nil {
			panic(err) // TODO should Panic here?
		}
		cs.mainChannel.SendAlways(bs, false, item.FwdChecker.GetNoProgressForwardFunc(), true)
	}
}

// ProcessMessage should be called when a message sent from an external node is ready to be processed.
// It returns a list of messages to be sent back to the sender of the original message (if any).
func (cs *CausalConsState) ProcessMessage(rcvMsg *channelinterface.RcvMsg) (returnMsg [][]byte, returnErrs []error) {
	defer func() {

		// After processing the message, if the timeout has passed do some actions
		if time.Since(cs.localIndexTime) > cs.timeoutTime && !cs.isInStorageInit {
			// -- on send no progress
			// we have a set of non-gc'd items
			// (non-gc'd means they have non-consumed children)
			// for each we send a no-progress
			// if it has not decided but has a valid proposal then we send the state (in addition to the no-progress)
			// (this is because these won't be computed as children so the behind node wouldn't get this state)

			// -- on receive no progress
			// if recipient has that or more recent, then must send that plus any children

			// After we have finished processing the messages, we do the following
			// (1) send any messages waiting to be forwarded
			// (2) if we have not made progress after a timeout, start a recovery
			decidedTimeoutItems, startedTimeoutItems, startedItems := cs.memberCheckerState.GetTimeoutItems()
			// Check forwarding for all started items
			for _, nxt := range startedItems {
				if sendForward(nxt.ConsItem.GetIndex(), cs.memberCheckerState, cs.mainChannel) {
					// anything to do here?
				}
			}
			// start recovery for no-progress items
			for _, nxt := range startedTimeoutItems {
				cs.sendNoProgressMsg(nxt.ConsItem.GetIndex(), nxt, false)
			}
			for _, nxt := range decidedTimeoutItems {
				// TODO for the decided items send in a more efficient way (maybe hash tree?)
				cs.sendNoProgressMsg(nxt.ConsItem.GetIndex(), nxt, true)
			}
			if initIdx, itm := cs.memberCheckerState.GetInitItem(); itm != nil { // In case we have any initial unconsumed indices
				cs.sendNoProgressMsg(initIdx, itm, true)
			}
		}

	}()

	msg := rcvMsg.Msg

	if msg == nil { // if the message is nil it was becase of a timeout in the recv loop, so we just run the defered function.
		return nil, nil
	}

	// Process each message one by one
	for _, deser := range msg {

		ht := deser.HeaderType

		// Check if it is a no change message
		switch ht {
		case messages.HdrNoProgress: // TODO

			// HdrNoProgress means decided but not yet garbage collected
			// We check if we have this hash either live or decided, and send the value

			var binStates [][]byte
			var err error
			if binStates, err = cs.memberCheckerState.GenChildIndices(deser.Index,
				deser.Header.(*messagetypes.NoProgressMessage).IsUnconsumedOutput, config.MaxRecovers); err != nil {

				// don't include the index because it might just be a future index we have not see yet so not an error
				switch err {
				case types.ErrParentNotFound, types.ErrInvalidIndex:
				default:
					returnErrs = append(returnErrs, err)
				}
				continue
			}
			if len(binStates) > 0 {
				returnMsg = append(returnMsg, binStates...)
			}

			// The external node hasn't made any progress, so if we are more advanced than it,
			// then we send it what we have
			// var returnMessages [][]byte
			// For each index that it is missing we send back the messages we have received for that index

			continue
		}

		// this is the parent index
		parIdx := types.ParentConsensusHash(deser.Index.Index.(types.ConsensusHash))

		// if we already decided this index then we skip the message
		if !cs.isInStorageInit && cs.store.Contains(types.ConsensusHash(parIdx)) {
			logging.Warning("got a message from an already decided index", parIdx)
			returnErrs = append(returnErrs, types.ErrIndexTooOld)
			continue
		}

		// If the memberchecker doesn't exist then we drop the message
		mc, _, _, err := cs.memberCheckerState.GetMemberChecker(deser.Index)
		if err != nil {
			if deser.IsDeserialized {
				panic("should find mc here") // sanity check
			}
			logging.Info("got a message, but did not find its member checker for index ", err, parIdx)
			cs.memberCheckerState.AddFutureMessage(
				&channelinterface.RcvMsg{CameFrom: 10, Msg: []*channelinterface.DeserializedItem{deser},
					SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
			continue
		}
		item, err := cs.memberCheckerState.GetConsItem(deser.Index)
		if err != nil {
			panic(err)
		}
		if !item.HasStarted() {
			item.Start()
		}
		if !deser.IsDeserialized { // The message is not deserialized
			if mc.MC.IsReady() { // If the member checker is ready we can retry deserializing the message
				cs.mainChannel.ReprocessMessage(&channelinterface.RcvMsg{CameFrom: 9, Msg: []*channelinterface.DeserializedItem{deser}, SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
			} else {
				// We cannot process the message yet, so we store it to be processed later
				cs.memberCheckerState.AddFutureMessage(
					&channelinterface.RcvMsg{CameFrom: 10, Msg: []*channelinterface.DeserializedItem{deser},
						SendRecvChan: rcvMsg.SendRecvChan, IsLocal: rcvMsg.IsLocal})
			}
			continue
		}
		// The message is deserialized
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
			progress, shouldForward = item.ProcessMessage(deser, rcvMsg.IsLocal, rcvMsg.SendRecvChan)
		}
		// add the message to the forward checker queue
		checkForward(rcvMsg.SendRecvChan, deser, shouldForward, item, cs.memberCheckerState, cs.mainChannel)
		if progress {
			// If we made progress we might have to advance the consensus index
			cs.checkProgress(deser.Index)
		}

		// check if we should do any forwarding for this cons item
		sendForward(deser.Index, cs.memberCheckerState, cs.mainChannel)
	}

	return //nil, nil
}

// checkProgress is called after a message is processed that made progress in the consensus.
// It upadates any state in case decisions were made.
func (cs *CausalConsState) checkProgress(idx types.ConsensusIndex) {
	// if we already stored the item then we decided already so can skip
	if !cs.isInStorageInit {
		if cs.store.Contains(idx.Index.(types.ConsensusHash)) {
			return
		}
	} else if cs.memberCheckerState.LastDecided == types.ParentConsensusHash(idx.Index.(types.ConsensusHash)) {
		// in init if we decided this value already then skip
		return
	}

	memC, _, _, err := cs.memberCheckerState.GetMemberChecker(idx)
	if err != nil { // sanity check
		panic(fmt.Sprint(err, idx, cs.memberCheckerState.LastDecided))
	}
	nextItem, err := cs.memberCheckerState.GetConsItem(idx)
	if err != nil {
		panic(err) // sanity check
	}

	// We have made progress on this item
	cs.memberCheckerState.UpdateProgressTime(idx)

	if !nextItem.HasDecided() {
		return
	}

	bs, err := nextItem.GetBinState(false)
	if err != nil {
		panic(err)
	}
	proposer, dec, _ := nextItem.GetDecision()

	// Update the member checker
	updatedDec := cs.memberCheckerState.DoneIndex(idx, proposer, dec)
	// Store the decided value
	// Update the storage if we are not initializing
	if !cs.isInStorageInit {

		err := cs.store.Write(idx.Index.(types.ConsensusHash), bs, updatedDec)
		if err != nil {
			panic(err)
		}
		memC.MC.GetStats().DiskStore(len(bs) + len(dec))
	}

	logging.Infof("Decided cons state index %v", idx)

}

// GetCausalDecisions returns the decided values in causal order tree.
func (cs *CausalConsState) GetCausalDecisions() (root *utils.StringNode, orderedDecisions [][]byte) {
	return cs.memberCheckerState.GetCausalDecisions()
}

// SMStatsString returns the statistics of the state machine.
func (cs *CausalConsState) SMStatsString(testDuration time.Duration) string {
	// TODO this wil get nil pointer
	//panic("fix me")
	return cs.memberCheckerState.ProposalInfo[cs.memberCheckerState.LastDecided].StatsString(testDuration)
}

// Collect is called when the item is being garbage collected.
func (sc *CausalConsState) Collect() {
	sc.memberCheckerState.Collect()
}
