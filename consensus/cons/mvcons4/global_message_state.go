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

package mvcons4

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/graph"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sync"
)

type globalMessageState struct {
	mutex               sync.RWMutex
	messages            map[types.HashStr]*sig.MultipleSignedMessage // all messages received by their signed message hash
	messagesByEventHash map[types.HashStr]*sig.MultipleSignedMessage // all messages received by their event hash
	toProcess           []graph.EventInfo                            // slice of events which are missing dependencies
	toProcessMap        map[types.HashStr]*sig.MultipleSignedMessage // same an toProcess except in a map
	graph               *graph.Graph
	sendMessageOnIndex  [][]graph.IndexType // list of index messages we have received but not yet received all dependencies
	lastCreateIndices   []graph.IndexType   // indicies of the last created local message
	maxSentIndices      [][]graph.IndexType // Indices for each node of the maximum Indices of events sent
	myID                int                 // my consensus id
	gc                  *generalconfig.GeneralConfig
	maxDecidedIndex     types.ConsensusIndex // index of the most recent decision + 1
	decisions           [][][][]byte
	decidedRounds       [][]graph.IndexType
	proposals           [][]byte // slice of proposals
	latestStats         stats.StatsInterface
}

func initGlobalMessageState(nodeCount, nmt, participantCount int, myID int, initHash types.HashBytes,
	stats stats.StatsInterface, gc *generalconfig.GeneralConfig) *globalMessageState {

	maxSentIndices := make([][]graph.IndexType, participantCount)
	for i := 0; i < len(maxSentIndices); i++ {
		maxSentIndices[i] = make([]graph.IndexType, nodeCount)
	}
	return &globalMessageState{
		messages:            make(map[types.HashStr]*sig.MultipleSignedMessage),
		messagesByEventHash: make(map[types.HashStr]*sig.MultipleSignedMessage),
		toProcessMap:        make(map[types.HashStr]*sig.MultipleSignedMessage),
		// msgsByID: make([][]*graph.EventInfo, nodeCount),
		graph:           graph.InitGraph(nodeCount, nmt, initHash),
		myID:            myID,
		maxSentIndices:  maxSentIndices,
		gc:              gc,
		maxDecidedIndex: types.SingleComputeConsensusIDShort(1),
		decidedRounds:   [][]graph.IndexType{make([]graph.IndexType, nodeCount)},
		decisions:       [][][][]byte{nil},
		proposals:       [][]byte{nil},
		latestStats:     stats,
	}
}

func (gms *globalMessageState) addProposal(proposal []byte) {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	gms.proposals = append(gms.proposals, proposal)
}

func (gms *globalMessageState) getMsg(hsh types.HashStr) *sig.MultipleSignedMessage {
	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	return gms.messages[hsh]
}

// addCreateMessageOnIndices adds a set of Indices received from a remote node.
// Once satisfied, a sync should be created and sent to another node.
// If the Indices are older, then the node sends the missing events
func (gms *globalMessageState) addCreateMessageOnIndices(idx []graph.IndexType) (localHasNewer bool) {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	gms.sendMessageOnIndex = append(gms.sendMessageOnIndex, idx)

	evs, err := gms.graph.GetMoreRecent(idx)
	utils.PanicNonNil(err)
	localHasNewer = len(evs) > 0
	return
}

// checkCreateEvent checks if an sync should be created based on the Indices received from
// addCreateMessageOnIndices.
func (gms *globalMessageState) checkCreateEvent() int {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	var found int
	ids := gms.graph.GetIndices()
	for i, nxt := range gms.sendMessageOnIndex {
		ok := true
		for j, idx := range nxt {
			if ids[j] < idx {
				ok = false
				break
			}
		}
		if ok {
			found++
			gms.sendMessageOnIndex[i] = nil
		}
	}
	if found > 0 {
		var newIndices [][]graph.IndexType
		for _, nxt := range gms.sendMessageOnIndex {
			if nxt != nil {
				newIndices = append(newIndices, nxt)
			}
		}
		gms.sendMessageOnIndex = newIndices
	}
	return found
}

func (gms *globalMessageState) checkMsg(msg *sig.MultipleSignedMessage, mc *consinterface.MemCheckers) error {
	// First we make sure the id matches the signature
	pub := msg.SigItems[0].Pub
	str, err := pub.GetPubID()
	if err != nil {
		return err
	}
	pub = mc.MC.CheckMemberBytes(mc.MC.GetIndex(), str) // Check Normal member
	if pub == nil {
		return types.ErrNotMember
	}
	pidx := pub.GetIndex()
	ev := msg.GetBaseMsgHeader().(*messagetypes.EventMessage).Event
	if ev.LocalInfo.ID != graph.IndexType(pidx) {
		return types.ErrInvalidPubIndex
	}

	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	// check if we already received the message
	if prevMsg := gms.messages[msg.GetHashString()]; prevMsg != nil {
		return types.ErrAlreadyReceivedMessage
	}
	_, err = gms.graph.AddEvent(ev, true)
	switch err {
	case types.ErrPrevIndexNotFound: // we didn't find the previous index so we may have to add it later
		err = nil
	case nil:
	default:
		logging.Error("got invalid event ", ev, err)
		panic(1) /// TODO remove me
	}
	return err
}

func (gms *globalMessageState) storeMsg(msg *sig.MultipleSignedMessage, alreadyLocked bool) error {
	if !alreadyLocked {
		gms.mutex.Lock()
		defer gms.mutex.Unlock()
	}

	// check if we already received the message
	if gms.messages[msg.GetHashString()] != nil {
		return types.ErrAlreadyReceivedMessage
	}

	// add the message to the map
	ev := msg.GetBaseMsgHeader().(*messagetypes.EventMessage).Event
	evHash := msg.GetBaseMsgHeader().(*messagetypes.EventMessage).GetEventInfoHash()
	gms.messages[msg.GetHashString()] = msg
	gms.messagesByEventHash[types.HashStr(evHash)] = msg

	fromID := graph.IndexType(sig.GetSingleSupporter(msg).GetIndex())
	logging.Infof("got ev id %v, idx %v, from %v, me %v\n", ev.LocalInfo.ID, ev.LocalInfo.Index, fromID, gms.myID)

	_, err := gms.graph.AddEvent(ev, false)
	switch err {
	case nil: // ok
	case types.ErrPrevIndexNotFound:
		err = nil
		gms.toProcess = append(gms.toProcess, ev)
		gms.toProcessMap[types.HashStr(msg.GetBaseMsgHeader().(*messagetypes.EventMessage).GetEventInfoHash())] = msg
	default:
		panic(err)
	}
	// since the state changed, we check if we can add the other events
	var invalid []graph.EventInfo
	var added []*graph.Event
	added, gms.toProcess, invalid = gms.graph.AddEvents(gms.toProcess)
	if len(invalid) > 0 {
		logging.Error("got invalid events", invalid)
	}
	for _, nxt := range added {
		hsh := types.HashStr(nxt.MyHash)
		//prevMsg := gms.toProcessMap[hsh]
		delete(gms.toProcessMap, hsh)
		//gms.messages[prevMsg.GetHashString()] = prevMsg
		//gms.messagesByEventHash[hsh] = prevMsg
	}
	return err
}

func (gms *globalMessageState) checkCreateEventAll2Al(mc *consinterface.MemCheckers) (msgs []messages.MsgHeader) {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	if gms.graph.GetLargerIndexCount(graph.IndexType(gms.myID)) >= mc.MC.GetMemberCount()-mc.MC.GetFaultCount() {
		// create the new local event
		ev := gms.graph.CreateEventIndex(graph.IndexType(gms.myID), gms.getProposal(), false)
		// create and sign the message
		msg := messagetypes.NewEventMessage()
		msg.Event = ev.EventInfo
		msgHdr := gms.setupMsg(mc, msg)
		sm := msgHdr.(*sig.MultipleSignedMessage)
		// add the message to the map
		gms.messages[sm.GetHashString()] = sm
		gms.messagesByEventHash[types.HashStr(ev.MyHash)] = sm
		// gms.lastCreated = sm
		gms.lastCreateIndices = gms.graph.GetIndices()
		msgs = append(msgs, sm)
	}
	return
}

func (gms *globalMessageState) getRecoverReply(indices []graph.IndexType) []messages.MsgHeader {

	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	evs, err := gms.graph.GetMoreRecent(indices)
	utils.PanicNonNil(err)
	msgs := gms.eventsToMsgs(evs)
	return msgs
}

func (gms *globalMessageState) createEventFromIndices(createNewEvent bool, mc *consinterface.MemCheckers,
	myId, fromID sig.PubKeyIndex, destID graph.IndexType, indices []graph.IndexType) []messages.MsgHeader {

	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	var maxIndices []graph.IndexType
	if int(destID) >= mc.MC.GetMemberCount() {
		// use the max of those you have sent the node so far
		maxIndices = graph.MaxIndices(indices, gms.maxSentIndices[destID])
	} else {
		// First check if the node has already created more recent events
		evIds := gms.graph.ComputeDiffIDIndex(destID)
		// use the max of that and the given Indices, and those you have sent the node so far
		maxIndices = graph.MaxIndices(evIds, indices, gms.maxSentIndices[destID])
	}
	evs, err := gms.graph.GetMoreRecent(maxIndices)
	utils.PanicNonNil(err)

	ret := gms.createEvent(createNewEvent, gms.checkEqual(gms.graph.GetIndices(), gms.lastCreateIndices, true),
		mc, myId, fromID, evs)
	// update the sent Indices for the destination
	gms.maxSentIndices[destID] = gms.graph.GetIndices()

	return ret
}

func (gms *globalMessageState) createEventFromID(createEvent bool, mc *consinterface.MemCheckers,
	myId, otherID, destID sig.PubKeyIndex) []messages.MsgHeader {

	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	var maxIndices []graph.IndexType
	if int(destID) >= mc.MC.GetMemberCount() {
		maxIndices = gms.maxSentIndices[destID]
	} else {
		// compute the events that the destID might not know
		// check against what has been sent to that node so far
		maxIndices = graph.MaxIndices(gms.maxSentIndices[destID], gms.graph.ComputeDiffIDIndex(graph.IndexType(destID)))
	}
	evs, err := gms.graph.GetMoreRecent(maxIndices)
	utils.PanicNonNil(err)

	ret := gms.createEvent(createEvent, false, mc, myId, otherID, evs)
	// update the sent Indices for the destination
	gms.maxSentIndices[destID] = gms.graph.GetIndices()
	return ret
}

func (gms *globalMessageState) getStats(_ consinterface.MemberChecker) stats.StatsInterface {
	return gms.latestStats
}

func (gms *globalMessageState) setupMsg(mc *consinterface.MemCheckers,
	msg messages.InternalSignedMsgHeader) messages.MsgHeader {

	switch msg.GetID() {
	case messages.HdrIdx:
		if gms.gc.EncryptChannels {
			sm := sig.NewUnsignedMessage(gms.maxDecidedIndex, mc.MC.GetNewPub(), msg)
			sm.SetEncryptPubs([]sig.Pub{mc.MC.GetMyPriv().GetPub()})
			return sm
		}
	case messages.HdrEventInfo: // nothing extra to do
	default:
		panic(msg)
	}
	priv := mc.MC.GetMyPriv()
	sm := sig.NewMultipleSignedMsg(gms.maxDecidedIndex, priv.GetPub(), msg)
	// First we serialize just to calculate the hash
	_, err := sm.Serialize(messages.NewMessage(nil))
	utils.PanicNonNil(err)
	mySig, err := priv.GenerateSig(sm, nil, sm.InternalSignedMsgHeader.GetSignType())
	utils.PanicNonNil(err)
	gms.getStats(mc.MC).SignedItem()
	sm.SetSigItems([]*sig.SigItem{mySig})
	return sm
}

func (gms *globalMessageState) getProposal() (proposal []byte) {
	if len(gms.proposals) > 0 {
		proposal = gms.proposals[0]
		gms.proposals = gms.proposals[1:]
	}
	return
}

func (gms *globalMessageState) gotIndicesMsg(from graph.IndexType, msg *messagetypes.IndexMessage) {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	sentIds := gms.maxSentIndices[from]
	for i, nxt := range msg.Indices {
		if nxt > sentIds[i] {
			sentIds[i] = nxt
		}
	}
}

func (gms *globalMessageState) eventsToMsgs(otherEvents []*graph.Event) []messages.MsgHeader {

	retSize := len(otherEvents)
	ret := make([]messages.MsgHeader, retSize, retSize+1)
	for i, nxt := range otherEvents {
		msg, ok := gms.messagesByEventHash[types.HashStr(nxt.MyHash)]
		if !ok {
			panic("missing message")
		}
		ret[i] = msg
	}
	return ret
}

func (gms *globalMessageState) createEvent(createNewEvent bool,
	useLastEvent bool, mc *consinterface.MemCheckers,
	myId, otherID sig.PubKeyIndex, otherEvents []*graph.Event) []messages.MsgHeader {

	ret := gms.eventsToMsgs(otherEvents)
	if createNewEvent && !useLastEvent {
		// create our new local event
		ev := gms.graph.CreateEvent(graph.IndexType(myId), graph.IndexType(otherID), gms.getProposal(), false)
		// create and sign the message
		msg := messagetypes.NewEventMessage()
		msg.Event = ev.EventInfo
		msgHdr := gms.setupMsg(mc, msg)
		sm := msgHdr.(*sig.MultipleSignedMessage)
		// add the message to the map
		gms.messages[sm.GetHashString()] = sm
		gms.messagesByEventHash[types.HashStr(ev.MyHash)] = sm
		// gms.lastCreated = sm
		gms.lastCreateIndices = gms.graph.GetIndices()
		ret = append(ret, sm)
	}
	return ret
}

func (gms *globalMessageState) startedIdx(_ types.ConsensusIndex, mc consinterface.MemberChecker) {
	gms.latestStats = mc.GetStats()
}

func (gms *globalMessageState) getMyIndices() []graph.IndexType {

	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	return gms.graph.GetIndices()
}

func (gms *globalMessageState) getRecoverMsg() messages.MsgHeader {
	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	msg := messagetypes.NewIndexRecoverMsg(gms.maxDecidedIndex)
	msg.Indices = gms.graph.GetIndices()

	return msg
}

func (gms *globalMessageState) getMyIndicesMsg(isReply bool,
	mc *consinterface.MemCheckers) (sm messages.MsgHeader) {

	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	msg := messagetypes.NewIndexMessage()
	msg.Indices = gms.graph.GetIndices()
	msg.IsReply = isReply
	sm = gms.setupMsg(mc, msg)

	return sm
}

func (gms *globalMessageState) getDecision(index types.ConsensusIndex) (decision [][][]byte, decisionRounds []graph.IndexType) {
	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	idx := int(index.Index.(types.ConsensusInt))
	return gms.decisions[idx], gms.decidedRounds[idx]
}

func (gms *globalMessageState) canStartNext(index types.ConsensusIndex) bool {
	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	idx := index.Index.(types.ConsensusInt)
	if idx == 1 {
		return true
	}
	if len(gms.decisions) >= int(idx) {
		return true
	}
	return false
}

func (gms *globalMessageState) hasDecided(index types.ConsensusIndex, mc *consinterface.MemCheckers) bool {
	gms.mutex.Lock()
	defer gms.mutex.Unlock()

	idx := graph.IndexType(index.Index.(types.ConsensusInt))
	if len(gms.decisions) > int(idx) {
		return true
	} else if len(gms.decisions) < int(idx) {
		return false
	}

	dec, decs, _, decIdxs, decCount := gms.graph.GetDecision(idx)
	if dec && idx+1 > graph.IndexType(gms.maxDecidedIndex.Index.(types.ConsensusInt)) {
		gms.maxDecidedIndex = types.SingleComputeConsensusIDShort(types.ConsensusInt(idx) + 1)
	}
	if dec {
		if len(gms.decisions) != int(idx) {
			panic("out of order decision")
		}
		gms.decisions = append(gms.decisions, decs)
		gms.decidedRounds = append(gms.decidedRounds, decIdxs)

		// if we are not a member, we use the id with the largest value to calculate the round
		var checkID int
		var maxRnd graph.IndexType
		if gms.myID >= mc.MC.GetMemberCount() {
			for i, nxt := range decIdxs {
				if nxt > maxRnd {
					checkID = i
					maxRnd = nxt
				}
			}
		} else {
			checkID = gms.myID
		}
		logging.Info("dec round", decIdxs[checkID], gms.decidedRounds[idx-1][checkID], idx, checkID)
		mc.MC.GetStats().AddFinishRoundSet(types.ConsensusRound(decIdxs[checkID]-gms.decidedRounds[idx-1][checkID]),
			decCount)
	}

	return dec
}

func (gms *globalMessageState) getConsensusIndexMessages(index types.ConsensusIndex) []messages.MsgHeader {
	gms.mutex.RLock()
	defer gms.mutex.RUnlock()

	idx := graph.IndexType(index.Index.(types.ConsensusInt))
	events := gms.graph.GetDecisionEvents(idx)
	ret := make([]messages.MsgHeader, len(events))
	for i, nxt := range events {
		msg, ok := gms.messagesByEventHash[types.HashStr(nxt.MyHash)]
		if !ok {
			panic("missing message")
		}
		ret[i] = msg
	}
	return ret
}

func (gms *globalMessageState) checkEqual(a, b []graph.IndexType, skipMyID bool) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for i, nxt := range a {
		if skipMyID && i == gms.myID {
			continue
		}
		if nxt != b[i] {
			return false
		}
	}
	return true
}
