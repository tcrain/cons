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
	witnessMap      map[IndexType][]*Event    // map of round to witnesses
	decidedRoundMap map[IndexType][]IndexType // map from round to the n indices of the events that were visible when this round first decided
	lastDecided     IndexType                 // last round decided
	eventCount      IndexType                 // total number of events
	lastAdded       *Event                    // the last event added
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
	witnessMap := make(map[IndexType][]*Event)
	decidedRoundMap := make(map[IndexType][]IndexType)
	decidedRoundMap[0] = make([]IndexType, nodeCount)
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
		ev := Event{
			EventInfo: evi,
			wi: &witnessInfo{
				decided: yesDec,
			},
			MyHash: types.GetHash(b.Bytes()),
		}
		tails[i] = append(tails[i], &ev)
	}
	ret := &Graph{
		tails:           tails,
		nmt:             nmt,
		undecided:       make([][]*Event, nodeCount),
		witnessMap:      witnessMap,
		decidedRoundMap: decidedRoundMap,
		eventCount:      IndexType(nodeCount),
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
	myIdx := prevLocal.LocalInfo.Index

	var remoteEv []EventPointer
	for i, t := range g.tails {
		if i == int(localID) {
			continue
		}
		for _, nxt := range t {
			if nxt.LocalInfo.Index >= myIdx {
				// use the hash of the event, not the previous hash
				nxtRemote := nxt.LocalInfo
				nxtRemote.Hash = nxt.MyHash
				remoteEv = append(remoteEv, nxtRemote)
				break
			}
		}
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
	for _, nxt := range myTails {
		for nxt != nil && nxt.LocalInfo.Index >= ev.LocalInfo.Index {
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
	if !found {
		g.tails[ev.LocalInfo.ID] = append(g.tails[ev.LocalInfo.ID], newEv)
	}
	g.computeWitness(newEv)
	g.eventCount++
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

// GetMoreRecent takes a list of indices, one per node id and returns the events that follow.
func (g *Graph) GetMoreRecent(indices []IndexType) (ret []*Event, err error) {
	if len(indices) != len(g.tails) {
		err = types.ErrInvalidIndex
		return
	}
	for i, nxt := range indices {
		for _, ev := range g.tails[i] {
			for ev.LocalInfo.Index > nxt {
				ret = append(ret, ev)
				ev = ev.localAncestor
			}
		}
	}
	// reverse the slice so we send the older first
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
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
	var events []*Event
	var ok bool
	if events, ok = g.witnessMap[ev.round]; !ok {
		events = make([]*Event, len(g.tails))
		g.witnessMap[ev.round] = events
	}
	if events[ev.LocalInfo.ID] != nil {
		panic("tried to add multiple witnesses")
	}
	events[ev.LocalInfo.ID] = ev
}

func max(v ...IndexType) (ret IndexType) {
	for _, nxt := range v {
		if nxt > ret {
			ret = nxt
		}
	}
	return
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
		ev.wi = &witnessInfo{} // we are a witness if we are larger than our local ancestor round
	}
	// see which witnesses from the previous round are visible
	found := g.witnessMap[round]
	stronglySees := make([]bool, len(g.tails))
	for i, nxt := range found {
		if nxt != nil && g.stronglySees(nxt, ev) {
			stronglySees[i] = true
		}
	}
	if utils.TrueCount(stronglySees) >= g.nmt { // if we strongly see nmt witnesses from the previous round, then we are a witness in the next round
		ev.round++
		ev.wi = &witnessInfo{stronglySeees: stronglySees}
	} else if ev.wi != nil { // if we are a witness, but not for the new round, compute the witnesses we see for the previous round
		found := g.witnessMap[round-1]
		stronglySees := make([]bool, len(g.tails))
		for i, nxt := range found {
			if nxt != nil && g.stronglySees(nxt, ev) {
				stronglySees[i] = true
			}
		}
		ev.wi = &witnessInfo{stronglySeees: stronglySees}
	}
	if ev.wi != nil {
		logging.Infof("created witness at Index %v, round %v, ID %v, strongly sees %v", ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID, ev.wi.stronglySeees)
		g.addWitness(ev)
		g.checkDecided(ev) // since we added a new witness, we might have new decisions
	}
}

/*// findWitnesses is a recursive call that finds the
func (g *Graph) findWitnesses(r int, ev *Event, found []*Event) {
	if ev == nil || ev.round < r {
		return
	}
	if ev.wi != nil && ev.round == r{
		found[ev.ID] = ev
	}
	g.findWitnesses(r, ev.remoteAncestor, found)
	g.findWitnesses(r, ev.localAncestor, found)
}
*/

// checkDecided checks if the Event has caused any previous events to decide
func (g *Graph) checkDecided(ev *Event) {
	var valDecided bool
	g.recCheckDecided(ev, make(map[*Event]bool), &valDecided)
	if valDecided {
		for dec, _, _, _, _ := g.GetDecision(g.lastDecided + 1); dec; dec, _, _, _, _ = g.GetDecision(g.lastDecided + 1) {
			if dec {
				g.lastDecided++
				decIdx := make([]IndexType, len(g.tails))
				for i, t := range g.tails { // take the index of the tails as the decided index
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

// recCheckDecided is a recursive call that performs a depth first traversal, checking if witnesses can decide,
// starting from the oldest, a witness can only decide if all the witnesses it can see have decided.
func (g *Graph) recCheckDecided(ev *Event, checked map[*Event]bool, newDecided *bool) (ret bool) {
	if val, ok := checked[ev]; ok {
		return val
	}
	defer func() {
		checked[ev] = ret
	}()

	if ev.wi != nil && ev.wi.decided != unknownDec { // if we have decided then all of our ancestors have decided, so return
		ret = true
		return
	}
	// we check all ancestors before returning so we generate the info during the traversal
	localDec := g.recCheckDecided(ev.localAncestor, checked, newDecided)
	remoteDec := true
	for _, nxt := range ev.remoteAncestor {
		if nxtRemoteDec := g.recCheckDecided(nxt, checked, newDecided); !nxtRemoteDec {
			remoteDec = false
		}
	}
	if !localDec || !remoteDec { // if either of our ancestors have not decided then exit since we should not decide yet
		ret = false
		return
	}
	if ev.wi == nil { // only witnesses decide
		ret = true
		return
	}
	g.computeDecided(ev)              // check if this witness can decide
	ret = ev.wi.decided != unknownDec // return true if we decided, false otherwise
	if ret {
		*newDecided = true
	}
	return
}

// computeDecided checks if the witness can decide
func (g *Graph) computeDecided(ev *Event) {
	w := g.witnessMap[ev.round+1]
	votes := make([]bool, len(g.tails))
	if len(w) < g.nmt { // not enough witnesses
		return
	}
	for i, nxt := range w {
		if nxt == nil {
			continue
		}
		votes[i] = g.hasPath(ev, nxt)
	}
	prevVotes := votes
	for r := ev.round + 2; true; r++ {
		votes := make([]bool, len(g.tails))
		w := g.witnessMap[r]
		prevW := g.witnessMap[r-1]
		// if we don't have enough witnesses in the previous round then we cannot decide, so exit
		if len(prevW) < g.nmt {
			return
		}
		// check each witness if it has enough votes to decide
		for i, witness := range w {
			if witness == nil {
				continue
			}
			var yesVotes, noVotes int
			// collect the votes of the previous round from the ones we strongly see
			for j, prevWitness := range prevW {
				if prevWitness == nil {
					continue
				}
				if witness.wi.stronglySeees[j] {
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
				// set my vote TODO add coin round
				votes[i] = yesVotes >= noVotes
				if yesVotes >= g.nmt { // decide yes
					if ev.wi.decided == noDec {
						panic("decided different values")
					}
					ev.wi.decidedRound = witness.round
					ev.wi.decided = yesDec
					logging.Infof("from Event Index %v, round %v, ID %v from Event"+
						" decided %v at Index %v, round %v, ID %v", witness.LocalInfo.Index, witness.round, witness.LocalInfo.ID,
						ev.wi.decided, ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID)
					return
				}
				if noVotes >= g.nmt {
					if ev.wi.decided == yesDec {
						panic("decided different values")
					}
					ev.wi.decidedRound = witness.round
					ev.wi.decided = noDec
					logging.Infof("from Event Index %v, round %v, ID %v from Event"+
						"decided %v at Index %v, round %v, ID %v", witness.LocalInfo.Index, witness.round, witness.LocalInfo.ID,
						ev.wi.decided, ev.LocalInfo.Index, ev.round, ev.LocalInfo.ID)
					return
				}
			}
		}
		prevVotes = votes
	}
}

// GetDecision returns true if the round has decided.
// It returns the set of n decided values (a voted no decision will have a nil value),
// and the rounds that caused each of the n events to decide.
// decCount is the number of nodes that decided yes for the index.
func (g *Graph) GetDecision(round IndexType) (hasDecided bool, decision [][][]byte, decisionRounds []IndexType,
	decisionIndex []IndexType, decCount int) {

	w := g.witnessMap[round]
	var witnessCount int
	for _, nxt := range w {
		if nxt == nil {
			continue
		}
		witnessCount++
		if nxt.wi.decided == unknownDec { // a witness still is undecided
			return
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
	for i, nxt := range w {
		if nxt != nil {
			decisionRounds[i] = nxt.wi.decidedRound
			decisionIndex[i] = nxt.LocalInfo.Index
			switch nxt.wi.decided {
			case yesDec:
				decCount++
				if len(nxt.Buff) > 0 { // Add the decision of the first event
					decision[i] = append(decision[i], nxt.Buff)
				}
				nxt = nxt.localAncestor
				// Add any until the previous decision
				for nxt != nil {
					if nxt.wi != nil {
						if nxt.wi.decided == unknownDec {
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
	// reverse the decisions per node since we want the earliest first
	for _, nxt := range decision {
		for i, j := 0, len(nxt)-1; i < j; i, j = i+1, j-1 {
			nxt[i], nxt[j] = nxt[j], nxt[i]
		}
	}
	return
}

// hasPath returns true if there is a path from from (earlier round) to to (later round).
func (g *Graph) hasPath(from, to *Event) bool {
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

// stronglySees returns true if there is a (set of) path(s) from from (earlier round) to to (later round) that goes through
// events created by > 2/3 of the nodes.
func (g *Graph) stronglySees(from, to *Event) bool {
	sees := make([]bool, len(g.tails))
	g.recStronglySees(from, to, sees, make(map[*Event]bool))
	val := utils.TrueCount(sees) >= g.nmt
	return val
}

// recStronglySees is the recursive call of stronglySees
func (g *Graph) recStronglySees(from, to *Event, sees []bool, passed map[*Event]bool) bool {
	if to == nil {
		return false
	}
	if from.round > to.round { // this event is too old
		return false
	}
	if passed[to] { // we have already observed this event
		return true
	}
	if from == to {
		sees[to.LocalInfo.ID] = true // since the Event can see itself
		return true
	}
	// se if we can reach the end from any of our ancestors
	var remote bool
	for _, nxt := range to.remoteAncestor {
		if g.recStronglySees(from, nxt, sees, passed) {
			remote = true
		}
	}
	local := g.recStronglySees(from, to.localAncestor, sees, passed)
	if remote || local {
		passed[to] = true
		sees[to.LocalInfo.ID] = true
		return true
	}
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
	known := make([]*Event, len(g.tails))
	g.computeKnownRec([]*Event{ev}, known, 0, make(map[*Event]bool))
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
	known := make([]*Event, len(g.tails))
	g.computeKnownRec([]*Event{from}, known, 0, make(map[*Event]bool))
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

// GetDecisionEvents returns the set of events that caused round r to decide that follow those that caused
// round r+1 to decide.
// If called on a round not yet decided it returns the events after the most recent decision.
func (g *Graph) GetDecisionEvents(r IndexType) (ret []*Event) {

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
			}
		}
	}
	return ret
}

func (g *Graph) getDecisionEventInfo(r IndexType) (ret []EventInfo) {
	evs := g.GetDecisionEvents(r)
	ret = make([]EventInfo, len(evs))
	for i, nxt := range evs {
		ret[i] = nxt.EventInfo
	}
	return
}
