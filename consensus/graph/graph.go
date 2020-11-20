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

package graph

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"sort"
)

// decType is set when the witness decides a value
type decType int

type IndexType int32

const (
	unknownDec decType = iota
	noDec
	yesDec
)

func (dt decType) String() string {
	switch dt {
	case unknownDec:
		return "unknownDec"
	case noDec:
		return "noDec"
	case yesDec:
		return "yesDec"
	}
	return fmt.Sprintf("decType:%d", dt)
}

type Graph struct {
	tails           [][]*Event                // last item(s) for each node
	undecided       [][]*Event                // earliest undecided item for each node
	nmt             int                       // n-t
	allWitnessMap   map[IndexType][][]*Event  // map of round to witnesses
	decidedRoundMap map[IndexType][]IndexType // map from round to the n indices of the events (largest index at each node) that were visible when this round first decided
	lastDecided     IndexType                 // last round decided
	eventCount      IndexType                 // total number of events
	lastAdded       *Event                    // the last event added
	seen            []*Event                  // used as temporary data during traversal
	known           [][]*Event                // each ID for the most event at each ID it knows from a decided event
	largestRoundGC  IndexType                 // the largest round hat has been garbage collected
	forks           []*forkInfo               // the forks that have been created so far
}

// containsEventPointer returns true if the event pointer is contained in the graph
func (g *Graph) ContainsEventPointer(ev EventPointer) bool {
	for _, nxt := range g.tails[ev.ID] {
		for nxt != nil && nxt.LocalInfo.Index > ev.Index {
			nxt = nxt.localAncestor
		}
		if nxt != nil && nxt.LocalInfo.Index == ev.Index && bytes.Equal(ev.Hash, nxt.MyHash) {
			return true
		}
	}
	return false
}

// getMissingDependencies returns the ancestors that of ev that are not in the graph.
func (g *Graph) GetMissingDependencies(ev EventInfo) (ret []EventPointer) {
	li := ev.LocalInfo
	li.Index-- // we decrement the index since we want the ancestor info
	if !g.ContainsEventPointer(li) {
		ret = append(ret, li)
	}
	for _, nxt := range ev.RemoteAncestors {
		if !g.ContainsEventPointer(nxt) {
			ret = append(ret, nxt)
		}
	}
	return
}

func (g *Graph) validate(ev EventInfo) error {
	if ev.LocalInfo.ID >= IndexType(len(g.tails)) {
		return types.ErrInvalidPubIndex
	}
	if ev.LocalInfo.Index == 0 {
		return types.ErrInvalidIndex
	}
	hl := types.GetHashLen()
	if len(ev.LocalInfo.Hash) != hl {
		return types.ErrInvalidHashSize
	}
	ids := make([]IndexType, 0, len(ev.RemoteAncestors))
	for _, nxt := range ev.RemoteAncestors {
		if nxt.ID == ev.LocalInfo.ID {
			return types.ErrInvalidRemoteAncestor
		}
		if len(nxt.Hash) != hl {
			return types.ErrInvalidHashSize
		}
		for _, nxtID := range ids {
			if nxt.ID == nxtID {
				return types.ErrDuplicateRemoteAncestors
			}
		}
		ids = append(ids, nxt.ID)
	}
	return nil
}

// InitGraph creates a Graph with n nodes each with an initial Event at Index 0
func InitGraph(nodeCount, nmt int, initHash types.HashBytes) *Graph {
	tails := make([][]*Event, nodeCount)
	// witnessMap := make(map[IndexType][]*Event)
	decidedRoundMap := make(map[IndexType][]IndexType)
	decidedRoundMap[0] = make([]IndexType, nodeCount)
	known := make([][]*Event, nodeCount)
	for i := 0; i < len(tails); i++ {
		known[i] = make([]*Event, nodeCount)
	}
	for i := 0; i < len(tails); i++ {
		evi := EventInfo{
			LocalInfo: EventPointer{
				ID:   IndexType(i),
				Hash: initHash},
			RemoteAncestors: []EventPointer{{Hash: initHash}},
		}
		b := bytes.NewBuffer(nil)
		_, err := evi.Encode(b)
		utils.PanicNonNil(err)
		ev := &Event{
			EventInfo: evi,
			WI: &witnessInfo{
				decided: yesDec,
			},
			MyHash: types.GetHash(b.Bytes()),
		}
		tails[i] = append(tails[i], ev)
		for j := 0; j < len(tails); j++ {
			known[j][i] = ev
		}
	}
	ret := &Graph{
		tails:           tails,
		nmt:             nmt,
		undecided:       make([][]*Event, nodeCount),
		allWitnessMap:   make(map[IndexType][][]*Event),
		decidedRoundMap: decidedRoundMap,
		eventCount:      IndexType(nodeCount),
		known:           known,
	}
	for _, nxt := range tails {
		ret.addWitness(nxt[0])
	}
	return ret
}

// GetLarger index count returns the number of IDs that have events with index as large or larger
// than the index of the event at ID localID (including the event at localID).
func (g *Graph) GetLargerIndexCount(localID IndexType) (count int) {
	prevLocal := g.tails[localID][0]
	myIdx := prevLocal.LocalInfo.Index

	for _, t := range g.tails {
		for _, nxt := range t {
			if nxt.LocalInfo.Index >= myIdx {
				count++
				break
			}
		}
	}
	return count
}

// CreateEventIndex creates an event at localID.
// All other ids that have the same index as (or larger than) the current localID index will be added
// as remote dependencies.
func (g *Graph) CreateEventIndex(localID IndexType, buff []byte, checkOnly bool) *Event {
	prevLocal := g.tails[localID][0]
	// myIdx := prevLocal.LocalInfo.Index

	var remoteEv []EventPointer
	for i, t := range g.tails {
		if i == int(localID) {
			continue
		}
		// use the tail with the maximum index
		nxtRemote := t[0].LocalInfo
		// use the hash of the event, not the previous hash
		nxtRemote.Hash = t[0].MyHash
		for _, nxt := range t[1:] {
			if nxt.LocalInfo.Index > nxtRemote.Index {
				nxtRemote = nxt.LocalInfo
				// use the hash of the event, not the previous hash
				nxtRemote.Hash = nxt.MyHash
			}
		}
		remoteEv = append(remoteEv, nxtRemote)
	}
	evInfo := EventInfo{
		LocalInfo: EventPointer{
			ID:    localID,
			Index: prevLocal.LocalInfo.Index + 1,
			Hash:  prevLocal.MyHash,
		},
		Buff:            buff,
		RemoteAncestors: remoteEv,
	}
	event, err := g.AddEvent(evInfo, checkOnly)
	utils.PanicNonNil(err)
	return event
}

// CreateEvent creates a new Event from the local node ID to the remote node ID with the Buff as its proposal
func (g *Graph) CreateEvent(localID, remoteID IndexType, buff []byte, checkOnly bool) *Event {
	prevLocal := g.tails[localID][0]
	prevRemote := g.tails[remoteID][0]
	for _, nxt := range g.tails[localID] {
		if nxt.LocalInfo.Index > prevLocal.LocalInfo.Index {
			prevLocal = nxt
		}
	}
	for _, nxt := range g.tails[remoteID] {
		if nxt.LocalInfo.Index > prevRemote.LocalInfo.Index {
			prevRemote = nxt
		}
	}
	evInfo := EventInfo{
		LocalInfo: EventPointer{
			ID:    localID,
			Index: prevLocal.LocalInfo.Index + 1,
			Hash:  prevLocal.MyHash,
		},
		Buff: buff,
		RemoteAncestors: []EventPointer{{
			ID:    remoteID,
			Index: prevRemote.LocalInfo.Index,
			Hash:  prevRemote.MyHash,
		}},
	}
	event, err := g.AddEvent(evInfo, checkOnly)
	utils.PanicNonNil(err)
	return event
}

func (g *Graph) checkFork(ev *Event, fork *forkInfo) bool {
	var forkEvsSeen []*Event
	g.recCheckFork(ev, fork, &forkEvsSeen)
	// unmark the traversed nodes
	g.resetSeen()
	if len(forkEvsSeen) > 1 { // we have seen the fork
		return true
	}
	return false
}

func (g *Graph) recCheckFork(ev *Event, fork *forkInfo, forkEvsSeen *[]*Event) {
	if ev == nil {
		return
	}
	if len(*forkEvsSeen) > 1 { // we have seen the fork
		return
	}
	if ev.seen != notSeen { // we have already checked this event
		return
	}

	// mark this event as traversed
	ev.seen = trueSeen
	g.seen = append(g.seen, ev)

	for _, nxt := range ev.seesForks { // check if the event is already know to have this fork
		if nxt == fork {
			*forkEvsSeen = fork.evs // we see this fork
			return
		}
	}
	var pastCount int              // how many of the fork events are before ev
	for _, nxt := range fork.evs { // check if this event is one from the fork
		if ev.round < nxt.round { // this event if before the fork
			pastCount++
		}
		if ev == nxt {
			*forkEvsSeen = append(*forkEvsSeen, ev)
		}
	}
	if pastCount == len(fork.evs) { // we have passed all events in the fork so we don't need to continue
		return
	}

	// traverse the ancestors
	g.recCheckFork(ev.localAncestor, fork, forkEvsSeen)
	for _, nxt := range ev.remoteAncestor {
		g.recCheckFork(nxt, fork, forkEvsSeen)
	}
}

// AddEvent adds the Event to the Graph and returns an error if the Event in invalid.
// It returns the Event added.
func (g *Graph) AddEvent(ev EventInfo, checkOnly bool) (event *Event, err error) {
	// logging.Printf("add Event %+v", ev)
	if err := g.validate(ev); err != nil {
		return nil, err
	}
	// compute the hash of the event
	b := bytes.NewBuffer(nil)
	_, err = ev.Encode(b)
	utils.PanicNonNil(err)
	hsh := types.GetHash(b.Bytes())

	var localAncestor *Event
	myTails := g.tails[ev.LocalInfo.ID]
	// first check if we already have this event
	for _, nxt := range myTails {
		for nxt != nil && nxt.LocalInfo.Index > ev.LocalInfo.Index {
			nxt = nxt.localAncestor
		}
		if nxt != nil && nxt.LocalInfo.Index == ev.LocalInfo.Index && bytes.Equal(hsh, nxt.MyHash) {
			return nil, types.ErrEventExists
		}
	}
	// find the local ancestor
	var forkItem *Event // if we find a fork this will be the item at the adjacent index
	for _, nxt := range myTails {
		for nxt != nil && nxt.LocalInfo.Index >= ev.LocalInfo.Index {
			if nxt.LocalInfo.Index == ev.LocalInfo.Index { // we have a fork
				forkItem = nxt
			}
			nxt = nxt.localAncestor
		}
		if nxt == nil || nxt.LocalInfo.Index != ev.LocalInfo.Index-1 || !bytes.Equal(ev.LocalInfo.Hash, nxt.MyHash) {
			continue
		}
		localAncestor = nxt
		break
	}
	if localAncestor == nil {
		return nil, types.ErrPrevIndexNotFound
	}
	// find the remote ancestors
	remoteAncestors := make([]*Event, len(ev.RemoteAncestors))
	for i, nxtRma := range ev.RemoteAncestors {
		for _, nxt := range g.tails[nxtRma.ID] {
			for nxt != nil && nxt.LocalInfo.Index > nxtRma.Index {
				nxt = nxt.localAncestor
			}
			if nxt == nil || nxt.LocalInfo.Index != nxtRma.Index || !bytes.Equal(nxtRma.Hash, nxt.MyHash) {
				continue
			}
			remoteAncestors[i] = nxt
			break
		}
		if remoteAncestors[i] == nil {
			return nil, types.ErrPrevIndexNotFound
		}
	}

	if checkOnly {
		return nil, nil
	}
	// insert the event
	newEv := &Event{
		EventInfo:      ev,
		remoteAncestor: remoteAncestors,
		localAncestor:  localAncestor,
		MyHash:         hsh,
	}
	var found bool
	for i, nxt := range myTails {
		if nxt == localAncestor {
			myTails[i] = newEv
			found = true
		}
	}
	if !found && forkItem == nil { // sanity check
		panic("found a fork incorrectly")
	}
	if forkItem != nil { // we found a fork
		g.tails[ev.LocalInfo.ID] = append(g.tails[ev.LocalInfo.ID], newEv)
		// first see if this is part of an existing fork
		var gotForkItem bool
		for _, nxt := range g.forks {
			if nxt.id == ev.LocalInfo.ID && nxt.evs[0].LocalInfo.Index == ev.LocalInfo.Index {
				nxt.evs = append(nxt.evs, newEv)
				gotForkItem = true
				break
			}
		}
		if !gotForkItem {
			g.forks = append(g.forks, &forkInfo{
				id:  ev.LocalInfo.ID,
				evs: []*Event{forkItem, newEv},
			})
		}
	}
	g.computeWitness(newEv) // check if the new event is a witness
	g.eventCount++
	g.computeKnownItem(newEv) // update the list of visible items from this event

	// check if this event sees any forks
	for _, nxt := range g.forks {
		if g.checkFork(newEv, nxt) {
			newEv.seesForks = append(newEv.seesForks, nxt)
		}
	}

	return newEv, nil
}

// AddEvents adds a set of events to the Graph.
// added are the events that we successfully added
// remainder are the ones that were not able to be added because their ancestors have not yet been added.
// invalid are those that created errors when adding.
func (g *Graph) AddEvents(events []EventInfo) (added []*Event, remainder, invalid []EventInfo) {
	// first create a list for each node ID sorted by Index
	all := make([]SortEvent, len(g.tails))
	for _, nxt := range events {
		if nxt.LocalInfo.ID >= IndexType(len(all)) {
			invalid = append(invalid, nxt)
			continue
		}
		all[nxt.LocalInfo.ID] = append(all[nxt.LocalInfo.ID], nxt)
	}
	for _, nxt := range all {
		sort.Sort(nxt)
	}
	// loop through the events one at a time taking the oldest from each ID.
	// stop when all consumed or all produce an error
	var errorCount int
	for errorCount < len(g.tails) {
		errorCount = 0
		for i, nxt := range all {
			if len(nxt) == 0 {
				errorCount++
				continue
			}
			// if g.checkAdd(nxt[0]) == nil {
			ev, err := g.AddEvent(nxt[0], false)
			switch err {
			case types.ErrPrevIndexNotFound: // we are missing some events, check others then try again
				errorCount++
			case nil: // add successful
				added = append(added, ev)
				all[i] = nxt[1:]
			default: // invalid Event (don't increment error count since we will just try the next)
				invalid = append(invalid, nxt[0])
				all[i] = nxt[1:]
			}
		}
	}
	for _, nxt := range all {
		remainder = append(remainder, nxt...)
	}
	return
}

// GetIndices returns the current indices of each node id.
func (g *Graph) GetIndices() []IndexType {
	ret := make([]IndexType, len(g.tails))
	for i, nxt := range g.tails {
		for _, t := range nxt {
			if t.LocalInfo.Index > ret[i] {
				ret[i] = t.LocalInfo.Index
			}
		}
	}
	return ret
}

// getRounds returns the current round of each node id.
func (g *Graph) getRounds() []IndexType {
	ret := make([]IndexType, len(g.tails))
	for i, nxt := range g.tails {
		for _, t := range nxt {
			if t.round > ret[i] {
				ret[i] = t.round
			}
		}
	}
	return ret
}

// GetMoreRecent takes a list of indices, one per node id and returns the events that follow.
// If any indices have been garbage collected and have events that are needed then the indices
// are returned as part of gcIdxs.
func (g *Graph) GetMoreRecent(indices []IndexType) (ret []*Event, gcIdxs []IndexType, err error) {
	if len(indices) != len(g.tails) {
		panic("sanity check")
	}
	ids := g.GetIndices() // here we keep the minimum index found for each id
	for i, nxt := range indices {
		for _, ev := range g.tails[i] {
			for ev != nil && ev.LocalInfo.Index > nxt {
				ret = append(ret, ev)
				if ids[i] > ev.LocalInfo.Index-1 {
					ids[i] = ev.LocalInfo.Index - 1
				}
				ev = ev.localAncestor
			}
		}
	}
	// reverse the slice so we send the older first
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}
	// now we find the decided indices that we were not able to include because they were garbage collected,
	// but have needed events
	// first check if we got all events we needed
	maxIds := g.GetIndices()
	var didntGetAll bool
	for i := range ids {
		if ids[i] != indices[i] && maxIds[i] != ids[i] {
			didntGetAll = true
		}
	}
	if !didntGetAll { // we got all events so we can return
		return
	}
	var prevDecIds, decIds []IndexType
	if g.lastDecided > 0 {
		prevDecIds = g.decidedRoundMap[g.lastDecided]
	}
	for i := g.lastDecided; i > 0; i-- {
		decIds = prevDecIds
		prevDecIds = g.decidedRoundMap[i-1]
		var idsHasIntersection bool // set to true if there is a intersection between the needed ids and the decided ids
		var decHasLarger bool       // set to true if an decided index is larger than than a needed index so we keep going
		for j, nxt := range decIds {
			if nxt > indices[j] && prevDecIds[j] <= ids[j] {
				idsHasIntersection = true
			}
			if indices[j] < nxt {
				decHasLarger = true
			}
		}
		if !decHasLarger {
			break
		}
		if idsHasIntersection {
			gcIdxs = append(gcIdxs, i)
		}
	}
	// reverse the slice so we send the older first
	for i, j := 0, len(decIds)-1; i < j; i, j = i+1, j-1 {
		decIds[i], decIds[j] = decIds[j], decIds[i]
	}

	return
}

// getAncestors returns the ancestors of the event.
func (g *Graph) getAncestors(ev EventInfo) (localPrev *Event, remotePrev []*Event) {
	for _, nxt := range g.tails[ev.LocalInfo.ID] {
		for nxt != nil && ev.LocalInfo.Index < nxt.LocalInfo.Index+1 {
			nxt = nxt.localAncestor
		}
		if nxt != nil && ev.LocalInfo.Index == nxt.LocalInfo.Index+1 { // sanity check
			localPrev = nxt
			break
		}
	}
	for _, nxtRemote := range ev.RemoteAncestors {
		for _, nxt := range g.tails[nxtRemote.ID] {
			for nxt != nil && nxtRemote.Index < nxt.LocalInfo.Index {
				nxt = nxt.localAncestor
			}
			if nxt != nil && nxtRemote.Index == nxt.LocalInfo.Index {
				remotePrev = append(remotePrev, nxt)
				break
			}
		}
	}
	return
}

type SortEvent []EventInfo

func (l SortEvent) Less(i, j int) bool {
	if l[i].LocalInfo.Index < l[j].LocalInfo.Index {
		return true
	}
	return false
}

func (l SortEvent) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l SortEvent) Len() int {
	return len(l)
}

// once ev is know to be a witness, add it to the map of witnesses
func (g *Graph) addWitness(ev *Event) {
	var events [][]*Event
	var ok bool
	if events, ok = g.allWitnessMap[ev.round]; !ok {
		events = make([][]*Event, len(g.tails))
		g.allWitnessMap[ev.round] = events
	}
	events[ev.LocalInfo.ID] = append(events[ev.LocalInfo.ID], ev)
}

func max(v ...IndexType) (ret IndexType) {
	for _, nxt := range v {
		if nxt > ret {
			ret = nxt
		}
	}
	return
}

func min(v ...IndexType) (ret IndexType) {
	if len(v) == 0 {
		return
	}
	ret = v[0]
	for _, nxt := range v[1:] {
		if nxt < ret {
			ret = nxt
		}
	}
	return
}

func (g *Graph) getAllWitnesses(round IndexType) [][]*Event {
	ret := g.allWitnessMap[round]
	if ret == nil {
		ret = make([][]*Event, len(g.tails))
		g.allWitnessMap[round] = ret
	}
	return ret
}

// computeWitness checks if the Event is a witness.
// A witness can strongly see > 2/3 of witnesses in previous round.
func (g *Graph) computeWitness(ev *Event) {
	// set witness and round
	round := ev.localAncestor.round // max(ev.localAncestor.round, ev.remoteAncestor.round) // ev.localAncestor.round
	for _, nxt := range ev.remoteAncestor {
		if nxt.round > round {
			round = nxt.round
		}
	}
	ev.round = round
	if ev.localAncestor.round < round { // ev.remoteAncestor.round {
		ev.WI = &witnessInfo{} // we are a witness if we are larger than our local ancestor round
	}
	// see which witnesses from the previous round are visible
	foundEach := g.allWitnessMap[round]
	stronglySees := make([]bool, len(g.tails))
	for i, found := range foundEach {
		for _, nxt := range found {
			if nxt != nil && g.stronglySees(nxt, ev) {
				stronglySees[i] = true
			}
		}
	}
	if utils.TrueCount(stronglySees) >= g.nmt { // if we strongly see nmt witnesses from the previous round, then we are a witness in the next round
		ev.round++
		ev.WI = &witnessInfo{stronglySeees: stronglySees}
	} else if ev.WI != nil { // if we are a witness, but not for the new round, compute the witnesses we see for the previous round
		foundEach := g.allWitnessMap[round-1]
		stronglySees := make([]bool, len(g.tails))
		for i, found := range foundEach {
			for _, nxt := range found {
				if nxt != nil && g.stronglySees(nxt, ev) {
					stronglySees[i] = true
				}
			}
		}
		ev.WI = &witnessInfo{stronglySeees: stronglySees}
	}
	if ev.WI != nil {
		logging.Infof("created witness at Index %v, round %v, ID %v, strongly sees %v", ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID, ev.WI.stronglySeees)
		g.addWitness(ev)
		g.checkDecided(ev) // since we added a new witness, we might have new decisions
	}
}

/*// findWitnesses is a recursive call that finds the
func (g *Graph) findWitnesses(r int, ev *Event, found []*Event) {
	if ev == nil || ev.round < r {
		return
	}
	if ev.WI != nil && ev.round == r{
		found[ev.ID] = ev
	}
	g.findWitnesses(r, ev.remoteAncestor, found)
	g.findWitnesses(r, ev.localAncestor, found)
}
*/

// checkDecided checks if the Event has caused any previous events to decide
func (g *Graph) checkDecided(ev *Event) {
	var valDecided bool
	g.recCheckDecided(ev, &valDecided)
	g.resetSeen()
	if valDecided {
		for dec, _, _, _, _, _ := g.GetDecision(g.lastDecided + 1); dec; dec, _, _, _, _, _ = g.GetDecision(g.lastDecided + 1) {
			if dec {
				g.lastDecided++
				decIdx := make([]IndexType, len(g.tails))
				for i, t := range g.tails { // take the (largest) index of the tails as the decided index
					for _, nxt := range t {
						if nxt.LocalInfo.Index > decIdx[i] {
							decIdx[i] = nxt.LocalInfo.Index
						}
					}
				}
				g.decidedRoundMap[g.lastDecided] = decIdx
			}
		}
	}
}

/*func (g *Graph) addMap(ev *Event, m map[int][]*Event) {
	var events []*Event
	var ok bool
	if events, ok = m[ev.round]; !ok {
		events = make([]*Event, len(g.tails))
		m[ev.round] = events
	}
	events[ev.ID] = ev
}
*/

// GetWitnesses returns the set of witnesses for the given index.
func (g *Graph) GetWitnesses(idx IndexType) [][]*Event {
	return g.getAllWitnesses(idx)
}

// recCheckDecided is a recursive call that performs a depth first traversal, checking if witnesses can decide,
// starting from the oldest, a witness can only decide if all the witnesses it can see have decided.
func (g *Graph) recCheckDecided(ev *Event, newDecided *bool) (ret bool) {
	if ev.allAncestorsDecided {
		return true
	}
	switch ev.seen {
	case notSeen:
	case trueSeen:
		return true
	case falseSeen:
		return false
	}
	defer func() {
		if ret {
			ev.seen = trueSeen
		} else {
			ev.seen = falseSeen
		}
		g.seen = append(g.seen, ev)
	}()

	if ev.WI != nil && ev.WI.decided != unknownDec { // if we have decided then all of our ancestors have decided, so return
		ret = true
		return
	}
	// we check all ancestors before returning so we generate the info during the traversal
	localDec := g.recCheckDecided(ev.localAncestor, newDecided)
	remoteDec := true
	for _, nxt := range ev.remoteAncestor {
		if nxtRemoteDec := g.recCheckDecided(nxt, newDecided); !nxtRemoteDec {
			remoteDec = false
		}
	}
	if !localDec || !remoteDec { // if either of our ancestors have not decided then exit since we should not decide yet
		ret = false
		return
	}
	if ev.WI == nil { // only witnesses decide
		ret = true
		ev.allAncestorsDecided = true
		return
	}
	g.computeDecided(ev)              // check if this witness can decide
	ret = ev.WI.decided != unknownDec // return true if we decided, false otherwise
	if ret {
		ev.allAncestorsDecided = true
		*newDecided = true
	}
	return
}

// computeDecided checks if the witness can decide
func (g *Graph) computeDecided(ev *Event) {
	initW := g.getAllWitnesses(ev.round + 1)
	votes := make([]bool, len(g.tails))
	var witnessCount int
	for i, wts := range initW {
		if len(wts) > 0 {
			witnessCount++
		}
		for _, nxt := range wts {
			if nxt == nil {
				continue
			}
			if g.hasPath(ev, nxt) {
				votes[i] = true
			}
		}
	}
	if witnessCount < g.nmt { // not enough witnesses
		return
	}
	prevVotes := votes
	for r := ev.round + 2; true; r++ {
		votes := make([]bool, len(g.tails))
		// w := g.getValidWitness(r)
		allWitness := g.getAllWitnesses(r)
		prevW := g.getAllWitnesses(r - 1)
		var witnessCount int

		// check each witness if it has enough votes to decide
		for i, nxtW := range allWitness {
			if len(nxtW) > 0 {
				witnessCount++
			}
			for _, witness := range nxtW {
				var yesVotes, noVotes int
				// collect the votes of the previous round from the ones we strongly see
				for j, prevWitness := range prevW {
					if len(prevWitness) == 0 {
						continue
					}
					if witness.WI.stronglySeees[j] {
						// if g.stronglySees(prevWitness, witness) {
						if prevVotes[j] {
							yesVotes++
						} else {
							noVotes++
						}
					}
				}
				// check we have enough votes
				if yesVotes+noVotes >= g.nmt {
					if yesVotes+noVotes > len(g.tails) {
						panic("more votes than nodes")
					}
					// update the known for the garbage collection
					if ev.known != nil {
						g.known[ev.LocalInfo.ID] = ev.known
					}
					// set my vote TODO add coin round
					votes[i] = yesVotes >= noVotes
					if yesVotes >= g.nmt { // decide yes
						if ev.WI.decided == noDec {
							panic("decided different values")
						}
						ev.WI.decidedRound = witness.round
						ev.WI.decided = yesDec
						logging.Infof("from Event Index %v, round %v, ID %v from Event"+
							" decided at Index %v, round %v, ID %v", witness.LocalInfo.Index, witness.round, witness.LocalInfo.ID,
							ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID)
						return
					}
					if noVotes >= g.nmt {
						if ev.WI.decided == yesDec {
							panic("decided different values")
						}
						ev.WI.decidedRound = witness.round
						ev.WI.decided = noDec
						logging.Infof("from Event Index %v, round %v, ID %v from Event"+
							"decided at Index %v, round %v, ID %v", witness.LocalInfo.Index, witness.round, witness.LocalInfo.ID,
							ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID)
						return
					}
				}
			}
		}
		if witnessCount < g.nmt { // not enough witnesses to decide, exit
			return
		}
		prevVotes = votes
	}
}

// GetDecision returns true if the round has decided.
// It returns the set of n decided values (a voted no decision will have a nil value),
// and the rounds that caused each of the n events to decide.
// decCount is the number of nodes that decided yes for the index.
func (g *Graph) GetDecision(round IndexType) (hasDecided bool, decision [][][]byte, decisionRounds []IndexType,
	decisionIndex []IndexType, decCount int, decisionBools []bool) {

	allW := g.getAllWitnesses(round)
	var witnessCount int
	for _, w := range allW {
		if len(w) > 0 {
			witnessCount++ // we have at least one witness at this ID
		}
		for _, nxt := range w {
			if nxt == nil {
				// no witness for this index, if we find another witness that has decided then we know this witness
				// will decide false as it will get a majority of no votes
				continue
			}
			if nxt.WI.decided == unknownDec { // a witness still is undecided so we have not yet decided
				return
			}
		}
	}
	if witnessCount == 0 { // we found no witnesses
		return
	}
	if witnessCount < g.nmt { // sanity check
		panic("should have at least n-t witnesses in a decision round")
	}
	hasDecided = true
	// we had decided witnesses and no not yet decided ones - so we have decided
	// some ids may not have witnesses yet, but those will decide nil, since all will have majority no votes when/if the witness is created
	decision = make([][][]byte, len(g.tails))
	decisionRounds = make([]IndexType, len(g.tails))
	decisionIndex = make([]IndexType, len(g.tails))
	decisionBools = make([]bool, len(g.tails))
	for i, w := range allW {
		var gotYesDec bool // we should only get 1 Yes decision at most per ID
		for _, nxt := range w {
			if nxt != nil {
				decisionRounds[i] = nxt.WI.decidedRound
				decisionIndex[i] = nxt.LocalInfo.Index
				switch nxt.WI.decided {
				case yesDec:
					if gotYesDec { // sanity check
						panic("decided multiple witnesses for the same ID")
					}
					gotYesDec = true
					decisionBools[i] = true
					decCount++
					if len(nxt.Buff) > 0 { // Add the decision of the first event
						decision[i] = append(decision[i], nxt.Buff)
					}
					nxt = nxt.localAncestor
					// Add any until the previous decision
					for nxt != nil {
						if nxt.WI != nil {
							if nxt.WI.decided == unknownDec {
								panic("should be decided")
							}
							break
						}
						if len(nxt.Buff) > 0 {
							decision[i] = append(decision[i], nxt.Buff)
						}
						nxt = nxt.localAncestor
					}
				case noDec:
				default:
					panic("should have decided")
				}
			}
		}
	}
	// reverse the decisions per node since we want the earliest first
	for _, nxt := range decision {
		for i, j := 0, len(nxt)-1; i < j; i, j = i+1, j-1 {
			nxt[i], nxt[j] = nxt[j], nxt[i]
		}
	}
	return
}

// performAtIndex calls the function toPerform on each event with index idx or smaller.
func (g *Graph) performAtIndex(id int, idx IndexType, toPerform func(*Event) bool) {
	for _, nxt := range g.tails[id] {
		for nxt != nil {
			nxtNxt := nxt.localAncestor // copy before function because function might nil this value
			if nxt.LocalInfo.Index <= idx {
				if !toPerform(nxt) {
					break
				}
			}
			nxt = nxtNxt
		}
	}
}

// CheckGCIndex will check for garbage collection of events happening at index index and before.
func (g *Graph) CheckGCIndex(index IndexType, gcFunc func(*Event) bool) {
	// First find the minimum set of indices that are know by all nodes
	// we cannot GC earlier than this as we may receive an event that depends on one of them
	// known := make([]IndexType, len(g.tails))
	known := g.GetIndices()
	rounds := g.getRounds()
	for i := 0; i < len(g.tails); i++ {
		for _, nxt := range g.known {
			if nxt[i].LocalInfo.Index < known[i] {
				known[i] = nxt[i].LocalInfo.Index
				rounds[i] = nxt[i].round
			}
		}
	}
	dec := g.decidedRoundMap[index]
	for i, nxt := range dec {
		g.performAtIndex(i, minIdx(nxt, known[i]), func(ev *Event) bool {
			ret := gcFunc(ev)
			ev.remoteAncestor = nil
			ev.localAncestor = nil
			return ret
		})
	}
	// find the minimum round of the known nodes, we can gc the witness info upto 1 round before this
	minRound := min(rounds...)
	if minRound > 0 {
		minRound--
	}
	if minRound > index { // we cannot GC past the input index
		minRound = index
	}
	for minRound > g.largestRoundGC {
		g.largestRoundGC++
		delete(g.allWitnessMap, g.largestRoundGC)
	}
}

// hasPath returns true if there is a path from from (earlier round) to to (later round).
func (g *Graph) hasPath(from, to *Event) bool {
	if to == nil {
		return false
	}
	if to.seesFork(from.LocalInfo.ID) {
		return false // if we see a fork at the creator of the event then we vote no
	}
	if from == to {
		return true
	}
	if from.round > to.round {
		return false
	}
	if to.known != nil {
		/*		known := to.known[from.LocalInfo.ID]
				if known == nil {
					return false
				}
				if known.LocalInfo.Index >= from.LocalInfo.Index {
					return true
				}
				return false
		*/
	}
	for _, nxt := range to.remoteAncestor {
		if g.hasPath(from, nxt) {
			return true
		}
	}
	return g.hasPath(from, to.localAncestor)
}

// stronglySeesItem returns true if there is a (set of) path(s) from from (earlier round) to to (later round) that goes through
// events created by > 2/3 of the nodes.
func (g *Graph) stronglySeesItem(from, to *Event) bool {
	return false
}

// stronglySees returns true if there is a (set of) path(s) from from (earlier round) to to (later round) that goes through
// events created by > 2/3 of the nodes.
func (g *Graph) stronglySees(from, to *Event) bool {
	if to.seesFork(from.LocalInfo.ID) {
		return false
	}
	sees := make([]bool, len(g.tails))
	g.recStronglySees(from, to, sees)
	g.resetSeen()
	val := utils.TrueCount(sees) >= g.nmt
	return val
}

func (g *Graph) resetSeen() {
	for _, nxt := range g.seen {
		nxt.seen = notSeen
	}
	g.seen = g.seen[:0] // reset the seen list to 0 (but don't gc since we will reuse it)
}

// recStronglySees is the recursive call of stronglySees
// passed map is true if the event has a path to the node, false if not
func (g *Graph) recStronglySees(from, to *Event, sees []bool) bool {
	if to == nil {
		return false
	}
	switch to.seen {
	case trueSeen:
		return true
	case falseSeen:
		return false
	}
	if from.round > to.round { // this event is too old
		return false
	}
	if from == to {
		sees[to.LocalInfo.ID] = true // since the Event can see itself
		return true
	}
	// if v, ok := passed[to]; ok { // we have already observed this event
	//	return v
	//}
	// see if we can reach the end from any of our ancestors
	var remote bool
	for _, nxt := range to.remoteAncestor {
		if g.recStronglySees(from, nxt, sees) {
			remote = true
		}
	}
	local := g.recStronglySees(from, to.localAncestor, sees)
	g.seen = append(g.seen, to)
	if remote || local {
		// passed[to] = true
		to.seen = trueSeen
		sees[to.LocalInfo.ID] = true
		return true
	}
	// passed[to] = false
	to.seen = falseSeen
	return false
}

// GetIndices returns the indices of the events.
func GetIndices(ev []*Event) []IndexType {
	ret := make([]IndexType, len(ev))
	for i, nxt := range ev {
		ret[i] = nxt.LocalInfo.Index
	}
	return ret
}

// GetIndicesID returns the indices of the events per id
func GetIndicesID(ev []*Event, n int) [][]int {
	ret := make([][]int, n)
	for _, nxt := range ev {
		ret[nxt.LocalInfo.ID] = append(ret[nxt.LocalInfo.ID], int(nxt.LocalInfo.Index))
	}
	for _, nxt := range ret {
		sort.Ints(nxt)
	}
	return ret
}

func minIdx(a, b IndexType) IndexType {
	if a < b {
		return a
	}
	return b
}

// MaxIndices returns the max at each index of the inputs.
func MaxIndices(items ...[]IndexType) []IndexType {
	if len(items) == 0 {
		return nil
	}
	ret := make([]IndexType, len(items[0]))
	for i := 0; i < len(ret); i++ {
		for _, nxt := range items {
			if ret[i] < nxt[i] {
				ret[i] = nxt[i]
			}
		}
	}
	return ret
}

// ComputeDiffID returns the events to which there is no path from the most recent Event of ID from.
// (this is used to send values to a different node when creating a new Event)
func (g *Graph) ComputeDiffID(id IndexType) (ret []*Event) {
	ev := g.tails[id][0]
	for _, nxt := range g.tails[id][1:] {
		if nxt.LocalInfo.Index > ev.LocalInfo.Index {
			ev = nxt
		}
	}
	return g.computeDiff(ev)
}

// ComputeDiffID returns the index of the events to which there is no path from the most recent Event of ID from.
// (this is used to send values to a different node when creating a new Event)
func (g *Graph) ComputeDiffIDIndex(id IndexType) (ret []IndexType) {
	ev := g.tails[id][0]
	for _, nxt := range g.tails[id][1:] {
		if nxt.LocalInfo.Index > ev.LocalInfo.Index {
			ev = nxt
		}
	}
	// known := make([]*Event, len(g.tails))
	// g.computeKnownRec([]*Event{ev}, known, 0, make(map[*Event]bool))
	known := g.computeKnownItem(ev)
	ret = make([]IndexType, len(g.tails))
	for i, nxt := range known {
		if nxt != nil {
			ret[i] = nxt.LocalInfo.Index
		}
	}
	return
}

// computeDiff returns the events to which there is no path from Event from.
// (this is used to send values to a different node when creating a new Event)
func (g *Graph) computeDiff(from *Event) (ret []*Event) {
	// known := make([]*Event, len(g.tails))
	// g.computeKnownRec([]*Event{from}, known, 0, make(map[*Event]bool))
	known := g.computeKnownItem(from)
	found := make(map[*Event]bool)
	for i, t := range g.tails {
		if IndexType(i) == from.LocalInfo.ID {
			continue
		}
		for _, t := range t {
			for t != nil && t.LocalInfo.Index > 0 && !found[t] && (known[i] == nil || t.LocalInfo.Index > known[i].LocalInfo.Index) {
				found[t] = true
				ret = append(ret, t)
				t = t.localAncestor
			}
		}
	}
	return ret
}

func (g *Graph) computeKnownItem(ev *Event) (ret []*Event) {
	if ev.known == nil {
		ev.known = g.computeKnown(ev, ev.remoteAncestor...)
	}
	return ev.known
}

func checkEv(ev *Event, ret []*Event) {
	if ev == nil {
		return
	}
	i := ev.LocalInfo.ID
	if ret[i] == nil {
		ret[i] = ev
	} else if ev.LocalInfo.Index > ret[i].LocalInfo.Index {
		ret[i] = ev
	}
}

func (g *Graph) computeKnown(ev1 *Event, evs ...*Event) (ret []*Event) {
	ret = make([]*Event, len(g.tails))
	checkEv(ev1, ret)
	if ev1.localAncestor != nil {
		for _, nxt := range ev1.localAncestor.known {
			checkEv(nxt, ret)
		}
	}
	for _, ev := range evs {
		checkEv(ev, ret)
		for _, nxt := range ev.known {
			checkEv(nxt, ret)
		}
	}
	return
}

// GetDecisionEvents returns the set of events that caused round r to decide that follow those that caused
// round r+1 to decide.
// If called on a round not yet decided it returns the events after the most recent decision.
func (g *Graph) GetDecisionEvents(r IndexType) (ret []*Event, err error) {
	var from, to []IndexType
	var ok bool
	if from, ok = g.decidedRoundMap[r-1]; !ok {
		from = g.decidedRoundMap[g.lastDecided]
	}
	if to, ok = g.decidedRoundMap[r]; !ok {
		to = g.GetIndices()
	}
	if r == 1 {
		from = make([]IndexType, len(g.tails))
	}

	for i, tails := range g.tails {
		for _, nxt := range tails {
			for nxt.LocalInfo.Index > from[i] {
				if nxt.LocalInfo.Index > 0 && (to == nil || nxt.LocalInfo.Index <= to[i]) {
					ret = append(ret, nxt)
				}
				nxt = nxt.localAncestor
				if nxt == nil { // this index has already been gc'd
					err = types.ErrEventAlreadyGC
					break
				}
			}
		}
	}
	return
}

func (g *Graph) getDecisionEventInfo(r IndexType) (ret []EventInfo) {
	evs, err := g.GetDecisionEvents(r)
	if err != nil {
		panic(err)
	}
	ret = make([]EventInfo, len(evs))
	for i, nxt := range evs {
		ret[i] = nxt.EventInfo
	}
	return
}

/***********************************************************************
Old Functions
*************************************************************************/

// computeKnownRec performs a breath first search to find the closest ancestor to ev at each Index
func (g *Graph) computeKnownRec(toCheck []*Event, ret []*Event, count int, found map[*Event]bool) {
	if len(toCheck) == 0 { // || count == len(g.tails) {
		return
	}
	nxt := toCheck[0]
	toCheck = toCheck[1:]
	if found[nxt] {
		g.computeKnownRec(toCheck, ret, count, found)
		return
	}
	found[nxt] = true
	prev := ret[nxt.LocalInfo.ID]
	if prev == nil || prev.LocalInfo.Index < nxt.LocalInfo.Index {
		count++
		ret[nxt.LocalInfo.ID] = nxt
	}
	if nxt.LocalInfo.Index > 0 {
		toCheck = append(append(toCheck, nxt.localAncestor), nxt.remoteAncestor...)
	}
	g.computeKnownRec(toCheck, ret, count, found)
}

// hasPath returns true if there is a path from from (earlier round) to to (later round).
func (g *Graph) hasPathOld(from, to *Event) bool {
	if to == nil {
		return false
	}
	if from == to {
		return true
	}
	if from.round > to.round {
		return false
	}
	for _, nxt := range to.remoteAncestor {
		if g.hasPath(from, nxt) {
			return true
		}
	}
	return g.hasPath(from, to.localAncestor)
}
