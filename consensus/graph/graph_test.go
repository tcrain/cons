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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
	"testing"
	"time"
)

const (
	t   = 1
	n   = 4
	nmt = n - t
)

func TestGraphContains(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	initHashes := make([]types.HashBytes, n)
	for i, nxt := range g.tails {
		initHashes[i] = nxt[0].MyHash
	}

	addEvent(0, 1, 1, 0, g, nil, t)
	ev := EventInfo{
		LocalInfo: EventPointer{Index: 2,
			Hash: g.tails[0][0].MyHash},
		RemoteAncestors: []EventPointer{
			{ID: 1,
				Hash: initHashes[1]},
		},
	}
	// event should have all dependencies in the graph
	assert.Equal(t, 0, len(g.GetMissingDependencies(ev)))
	addEventInfo(ev, false, g, nil, t)
	assert.Equal(t, 0, len(g.GetMissingDependencies(ev)))

	ev2 := EventInfo{ // event is too far in future
		LocalInfo: EventPointer{Index: 4,
			Hash: g.tails[0][0].MyHash},
		RemoteAncestors: []EventPointer{
			{ID: 2,
				Hash: initHashes[1]},
		},
	}
	assert.Equal(t, 2, len(g.GetMissingDependencies(ev2)))

	ev3 := EventInfo{ // has a missing hash for the local ancestor
		LocalInfo: EventPointer{Index: 2,
			Hash: initHashes[0]},
		RemoteAncestors: []EventPointer{
			{ID: 1,
				Hash: initHashes[1]},
		},
	}
	assert.Equal(t, 1, len(g.GetMissingDependencies(ev3)))
}

func TestGraphAdd(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	initHashes := make([]types.HashBytes, n)
	for i, nxt := range g.tails {
		initHashes[i] = nxt[0].MyHash
	}

	addEvent(0, 1, 0, 0, g, types.ErrInvalidRemoteAncestor, t) // can't have same local and remote ID
	ev := EventInfo{
		LocalInfo: EventPointer{Index: 2,
			Hash: initHashes[0]},
		RemoteAncestors: []EventPointer{
			{ID: 1, // Index 1 does not exits
				Hash: initHashes[1]},
		},
	}
	addEventInfo(ev, false, g, types.ErrPrevIndexNotFound, t)
	ev = EventInfo{
		LocalInfo: EventPointer{
			Index: 0, // cannot add a new Event with Index 0
			Hash:  initHash,
		},
		RemoteAncestors: []EventPointer{{
			ID:   1,
			Hash: initHashes[1],
		}},
	}
	addEventInfo(ev, false, g, types.ErrInvalidIndex, t)
	addEvent(0, 1, 1, 0, g, nil, t)
	ev = EventInfo{
		LocalInfo: EventPointer{
			Index: 2,
			Hash:  initHashes[0], // invalid hash since this is the hash of Index 0 not 1
		},
		RemoteAncestors: []EventPointer{{
			ID:   1,
			Hash: initHashes[1],
		}},
	}
	addEventInfo(ev, false, g, types.ErrPrevIndexNotFound, t)
	addEvent(0, 2, 2, 0, g, nil, t)

	addEvent(3, 1, 0, 0, g, nil, t)
	ev = EventInfo{ // this is the same event as just inserted
		LocalInfo: EventPointer{
			ID:    3,
			Index: 1,
			Hash:  initHashes[3],
		},
		RemoteAncestors: []EventPointer{{
			Hash: initHashes[0],
		}},
	}
	addEventInfo(ev, false, g, types.ErrEventExists, t)
}

func TestGraphWitness(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	for idx := IndexType(1); idx < 5; idx++ {
		for i := IndexType(0); i < n; i++ {
			addEvent(i, idx, (i+1)%n, idx-1, g, nil, t)
		}
	}
	traverseGraph(g, func(ev *Event) {
		switch ev.LocalInfo.Index {
		case 0:
			assert.Equal(t, IndexType(0), ev.round)
			assert.NotNil(t, ev.WI)
		case 1, 2, 3:
			assert.Equal(t, IndexType(0), ev.round)
			assert.Nil(t, ev.WI)
		case 4:
			assert.Equal(t, IndexType(1), ev.round)
			assert.NotNil(t, ev.WI)
		}
	})
}

func TestGraphVisible(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	addEvent(0, 1, 1, 0, g, nil, t)
	addEvent(1, 1, 0, 1, g, nil, t)

	tail0 := g.tails[0][0]
	tail1 := g.tails[1][0]

	traverseGraph(g, func(ev *Event) {
		if ev == tail0 {
			assert.True(t, g.hasPath(tail0, ev))
		} else if ev == tail1 {
			assert.True(t, g.hasPath(tail0, tail1))
		} else {
			p := g.hasPath(ev, tail0)
			switch ev.LocalInfo.ID {
			case 0, 1:
				assert.True(t, p)
			default:
				assert.False(t, p)
			}
			p = g.hasPath(ev, tail1)
			switch ev.LocalInfo.ID {
			case 0, 1:
				assert.True(t, p)
			default:
				assert.False(t, p)
			}
		}
	})
}

func TestGrraphGetMoreRecent(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	initIds := g.GetIndices()
	idx := makeAllStronglySee(g, 0, t)
	ev, ids, err := g.GetMoreRecent(g.GetIndices())
	assert.Nil(t, err)
	assert.Nil(t, ids)
	assert.Equal(t, 0, len(ev))
	ev, ids, err = g.GetMoreRecent(initIds)
	assert.Nil(t, err)
	assert.Nil(t, ids)
	assert.Equal(t, idx*n, IndexType(len(ev)))
}

func TestGraphGetDec(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var startIndex IndexType
	startIndex = makeAllStronglySee(g, startIndex, t)
	startIndex = makeAllStronglySee(g, startIndex, t)
	startIndex = makeAllStronglySee(g, startIndex, t)

	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)

	ev := g.getDecisionEventInfo(1)
	g2 := InitGraph(n, nmt, initHash)
	_, rem, inv := g2.AddEvents(ev)
	assert.Equal(t, 0, len(rem))
	assert.Equal(t, 0, len(inv))

	dec, _, _, _, _, _ = g2.GetDecision(1)
	assert.True(t, dec)
}

var initHash = types.GetHash([]byte(config.CsID))

func TestGraphEncode(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var startIndex IndexType
	startIndex = makeAllStronglySee(g, startIndex, t)
	startIndex = makeAllStronglySee(g, startIndex, t)
	startIndex = makeAllStronglySee(g, startIndex, t)

	node := g.tails[0][0].EventInfo
	b := bytes.NewBuffer(nil)
	n, err := node.Encode(b)
	assert.Nil(t, err)
	node1 := EventInfo{}
	n1, err := (&node1).Decode(b)
	assert.Nil(t, err)
	assert.Equal(t, n, n1)
	// assert.Equal(t, node, node1)
	assert.Equal(t, node, node1)

	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)

	ev := g.getDecisionEventInfo(1)
	g2 := InitGraph(n, nmt, initHash)
	_, rem, inv := g2.AddEvents(ev)
	assert.Equal(t, 0, len(rem))
	assert.Equal(t, 0, len(inv))

	dec, _, _, _, _, _ = g2.GetDecision(1)
	assert.True(t, dec)
}

func TestGraphStronglySees(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	roots := make([]*Event, n)
	for i, nxt := range g.tails {
		roots[i] = nxt[0]
	}

	for idx := IndexType(1); idx < 6; idx++ {
		for i := IndexType(0); i < n; i++ {
			addEvent(i, idx, (i+1)%n, idx-1, g, nil, t)
		}
	}
	traverseGraph(g, func(ev *Event) {
		for _, nxt := range roots {
			ss := g.stronglySees(nxt, ev)
			switch ev.LocalInfo.Index {
			case 5:
				assert.True(t, ss)
			case 4:
				if nxt.LocalInfo.ID == (ev.LocalInfo.ID+2)%n || nxt.LocalInfo.ID == (ev.LocalInfo.ID+3)%n || nxt.LocalInfo.ID == (ev.LocalInfo.ID+4)%n {
					assert.True(t, ss)
				} else {
					assert.False(t, ss)
				}
			case 3:
				if nxt.LocalInfo.ID == (ev.LocalInfo.ID+2)%n || nxt.LocalInfo.ID == (ev.LocalInfo.ID+3)%n {
					assert.True(t, ss)
				} else {
					assert.False(t, ss)
				}
			case 2:
				if nxt.LocalInfo.ID == (ev.LocalInfo.ID+2)%n {
					assert.True(t, ss)
				} else {
					assert.False(t, ss)
				}
			default:
				assert.False(t, ss)
			}
		}
	})
}

func TestGraphDecided(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	for idx := IndexType(1); idx < 13; idx++ {
		for i := IndexType(0); i < n; i++ {
			addEvent(i, idx, (i+1)%n, idx-1, g, nil, t)
		}
	}
	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)
	traverseGraph(g, func(ev *Event) {
		switch ev.LocalInfo.Index {
		case 0, 4:
			assert.Equal(t, yesDec, ev.WI.decided)
		case 8, 12:
			assert.Equal(t, unknownDec, ev.WI.decided)
		default:
			assert.Nil(t, ev.WI)
		}
	})
}

func TestGraphDecideNo(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	for idx := IndexType(1); idx < 13; idx++ {
		for i := IndexType(0); i < n; i++ {
			addEvent(i, idx, (i+1)%(n-1), idx-1, g, nil, t)
		}
	}
	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)
	traverseGraph(g, func(ev *Event) {
		switch ev.LocalInfo.Index {
		case 0, 4:
			if ev.LocalInfo.ID == n-1 && ev.LocalInfo.Index == 4 {
				assert.Equal(t, noDec, ev.WI.decided)
			} else {
				assert.Equal(t, yesDec, ev.WI.decided)
			}
		case 8, 12:
			assert.Equal(t, unknownDec, ev.WI.decided)
		default:
			assert.Nil(t, ev.WI)
		}
	})
}

func TestGraphDecideRound2(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	// Generate the first set of witnesses
	idx := makeAllStronglySee(g, 0, t)

	// Generate the second set of witnesses, except witness round 1, ID 3 is only visible for 2 nodes
	idx++
	addEvent(0, idx, 1, idx-1, g, nil, t)
	addEvent(1, idx, 2, idx-1, g, nil, t)
	addEvent(2, idx, 0, idx-1, g, nil, t)
	addEvent(3, idx, 0, idx-1, g, nil, t)

	idx++
	addEvent(0, idx, 2, idx-1, g, nil, t)
	addEvent(1, idx, 0, idx-1, g, nil, t)
	addEvent(2, idx, 1, idx-1, g, nil, t)
	addEvent(3, idx, 0, idx-1, g, nil, t)

	idx++
	addEvent(0, idx, 1, idx-1, g, nil, t)
	addEvent(1, idx, 2, idx-1, g, nil, t)
	addEvent(2, idx, 3, idx-1, g, nil, t)
	addEvent(3, idx, 0, idx-1, g, nil, t)

	idx++
	addEvent(0, idx, 1, idx-1, g, nil, t)
	addEvent(1, idx, 0, idx-1, g, nil, t)
	addEvent(2, idx, 0, idx-1, g, nil, t)
	addEvent(3, idx, 0, idx-1, g, nil, t)

	// Generate 2 more rounds of witnesses
	// Witness round 1, ID 3 should not decide until the later round, while the others should decide in the first round
	idx = makeAllStronglySee(g, idx, t)
	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.False(t, dec)
	idx = makeAllStronglySee(g, idx, t)

	dec, _, _, _, _, _ = g.GetDecision(1)
	assert.True(t, dec)
}

func TestGraphDiff(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	assert.Equal(t, 0, len(g.computeDiff(g.tails[0][0])))

	addEvent(0, 1, 1, 0, g, nil, t)
	assert.Equal(t, 0, len(g.computeDiff(g.tails[0][0])))

	addEvent(1, 1, 0, 1, g, nil, t)
	assert.Equal(t, 0, len(g.computeDiff(g.tails[1][0])))
	assert.Equal(t, 1, len(g.computeDiff(g.tails[0][0])))

	addEvent(1, 2, 0, 1, g, nil, t)
	assert.Equal(t, 2, len(g.computeDiff(g.tails[0][0])))
	assert.Equal(t, 3, len(g.computeDiff(g.tails[2][0])))
	assert.Equal(t, 3, len(g.ComputeDiffID(2)))
	check := make([]IndexType, n)
	check[0] = 1
	assert.Equal(t, check, g.ComputeDiffIDIndex(0))
	assert.Equal(t, 2, len(g.ComputeDiffID(0)))

	addEvent(2, 1, 1, 1, g, nil, t)
	assert.Equal(t, 1, len(g.computeDiff(g.tails[2][0])))
	assert.Equal(t, 4, len(g.computeDiff(g.tails[3][0])))

	addEvent(3, 1, 2, 1, g, nil, t)
	assert.Equal(t, 1, len(g.computeDiff(g.tails[3][0])))

	addEvent(3, 2, 1, 2, g, nil, t)
	assert.Equal(t, 0, len(g.computeDiff(g.tails[3][0])))
}

func TestMissingDecision(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	for i := IndexType(0); i < 13; i++ {
		for j := IndexType(0); j < n-1; j++ {
			addEvent(j, i+1, (j+1)%(n-1), i, g, nil, t)
		}
	}
	// we know ID n-1 will decide No at round 1, since there are no witnesses for it visible yet
	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)
}

func addEvent(id, index, other, otherIndex IndexType, g *Graph, checkErr error, t *testing.T) {
	ev := EventInfo{
		LocalInfo: EventPointer{
			ID:    id,
			Index: index,
		},
		RemoteAncestors: []EventPointer{{
			ID:    other,
			Index: otherIndex,
		}},
	}
	addEventInfo(ev, true, g, checkErr, t)
}

func addEventInfo(ev EventInfo, setHash bool, g *Graph, checkErr error, t *testing.T) {
	if setHash {
		localAncestor, remoteAncestor := g.getAncestors(ev)
		if localAncestor != nil {
			ev.LocalInfo.Hash = localAncestor.MyHash
		}
		for i, nxt := range remoteAncestor {
			ev.RemoteAncestors[i].Hash = nxt.MyHash
		}
	}
	_, err := g.AddEvent(ev, false)
	assert.Equal(t, checkErr, err)
}

func traverseGraph(g *Graph, checkFunc func(*Event)) {
	checked := make(map[*Event]bool)
	for _, nxt := range g.tails {
		for _, ev := range nxt {
			recTraverseGraph(ev, g, checkFunc, checked)
		}
	}
}

func recTraverseGraph(ev *Event, g *Graph, checkFunc func(*Event), m map[*Event]bool) {
	if ev == nil {
		return
	}
	if _, ok := m[ev]; ok {
		return
	}
	m[ev] = true
	checkFunc(ev)
	recTraverseGraph(ev.localAncestor, g, checkFunc, m)
	for _, nxt := range ev.remoteAncestor {
		recTraverseGraph(nxt, g, checkFunc, m)
	}
}

const (
	rndSeed = 1
	numDec  = 100
)

func TestRandGen(t *testing.T) {
	r := rand.New(rand.NewSource(rndSeed))
	g := InitGraph(n, nmt, initHash)
	decisions := make([][][][]byte, numDec)
	eventCounts := make([]int, numDec)
	decisionCounts := make([]int, numDec)
	decisionRounds := make([][]IndexType, numDec)

	decTimes := make([]time.Time, numDec)

	var decidedIdx IndexType
	proposers := make([]IndexType, n)
	for i := range proposers {
		proposers[i] = IndexType(i)
	}
	proposerIdx := 0
	for decidedIdx < numDec {
		if proposerIdx%n == 0 {
			r.Shuffle(n, func(i, j int) { proposers[i], proposers[j] = proposers[j], proposers[i] })
		}
		nxtLocal := proposers[proposerIdx%n]
		proposerIdx++
		nxtRemote := nxtLocal
		for nxtRemote == nxtLocal {
			nxtRemote = IndexType(r.Intn(n))
		}
		g.CreateEvent(nxtLocal, nxtRemote, []byte{byte(nxtLocal)}, false)
		eventCounts[decidedIdx]++
		for true {
			decided, dec, decRound, _, _, _ := g.GetDecision(decidedIdx + 1)
			if decided {
				if decidedIdx >= IndexType(len(decisionCounts)) {
					decisionCounts = append(decisionCounts, 0)
					decisions = append(decisions, nil)
					decisionRounds = append(decisionRounds, nil)
					decTimes = append(decTimes, time.Time{})
				}
				decTimes[decidedIdx] = time.Now()
				decisionRounds[decidedIdx] = decRound
				decisions[decidedIdx] = dec
				for _, nxt := range dec {
					if len(nxt) > 0 {
						decisionCounts[decidedIdx]++
					}
				}
				decidedIdx++
			} else {
				break
			}
		}
	}
	t.Logf("Decisions: %v, events per decision %v, decision rounds %v", decisionCounts, eventCounts, decisionRounds)
	var sinceTimes []time.Duration
	prevTime := decTimes[0]
	for _, nxt := range decTimes[1:] {
		sinceTimes = append(sinceTimes, nxt.Sub(prevTime))
		prevTime = nxt
	}
	t.Log(sinceTimes)
}

func TestRandGenAll2All(t *testing.T) {
	r := rand.New(rand.NewSource(rndSeed))
	g := InitGraph(n, nmt, initHash)
	decisions := make([][][][]byte, numDec)
	eventCounts := make([]int, numDec)
	decisionCounts := make([]int, numDec)
	decisionRounds := make([][]IndexType, numDec)

	var decidedIdx IndexType
	proposers := make([]IndexType, n)
	for i := range proposers {
		proposers[i] = IndexType(i)
	}
	for decidedIdx < numDec {
		nxtLocal := IndexType(r.Intn(n))
		if g.GetLargerIndexCount(nxtLocal) >= nmt { // we can create an event at this id
			g.CreateEventIndex(nxtLocal, []byte{byte(nxtLocal)}, false)
		} else { // try another id to create an event at
			continue
		}
		eventCounts[decidedIdx]++
		for true {
			decided, dec, decRound, _, _, _ := g.GetDecision(decidedIdx + 1)
			if decided {
				if decidedIdx >= IndexType(len(decisionCounts)) {
					decisionCounts = append(decisionCounts, 0)
					decisions = append(decisions, nil)
					decisionRounds = append(decisionRounds, nil)
				}
				decisionRounds[decidedIdx] = decRound
				decisions[decidedIdx] = dec
				for _, nxt := range dec {
					if len(nxt) > 0 {
						decisionCounts[decidedIdx]++
					}
				}
				decidedIdx++
			} else {
				break
			}
		}
	}
	t.Logf("Decisions: %v, events per decision %v, decision rounds %v", decisionCounts, eventCounts, decisionRounds)
}

func TestAllStronglySee(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	initEvents := make([]*Event, len(g.tails))
	for i, nxt := range g.tails {
		initEvents[i] = nxt[0]
	}
	makeAllStronglySee(g, 0, t)
	for _, nxt := range g.tails {
		for _, front := range initEvents {
			assert.True(t, g.stronglySees(front, nxt[0]))
		}
	}
}

// makeAllStronglySee makes enough events so that all ids strongly see all other ids and have the same index, all ids
// must start with the same index.
// It returns the new index of all ids.
// Each event has a single remote ancestor.
func makeAllStronglySee(g *Graph, startIndex IndexType, t *testing.T) (newIndex IndexType) {
	// first all to all
	for i := IndexType(0); i < n; i++ {
		for j := IndexType(0); j < n-1; j++ {
			addEvent(i, startIndex+j+1, (i+j+1)%n, startIndex, g, nil, t)
		}
	}
	// Index 0 to the first half
	for i := IndexType(1); i < n/2; i++ {
		addEvent(0, startIndex+n+i-1, i, startIndex+n-1, g, nil, t)
	}
	// Index n-1 to the second half
	var j IndexType
	for i := IndexType(n) - 2; i >= n/2; i-- {
		addEvent(n-1, startIndex+n+j, i, startIndex+n-1, g, nil, t)
		j++
	}
	// Index 0 to Index n-1, should cause a strongly seeing
	addEvent(0, startIndex-1+n+n/2, n-1, startIndex-2+n+n/2, g, nil, t)
	// Index n-1 to 0, should cause a strongly seeing
	addEvent(n-1, startIndex-1+n+n/2, 0, startIndex-1+n+n/2, g, nil, t)
	// remaining indices to Index 0
	remoteIndex := startIndex - 1 + n + n/2
	for i := IndexType(1); i < n-1; i++ {
		// add until we all have the same number of events
		localIdx := startIndex + n
		for localIdx <= remoteIndex {
			addEvent(i, localIdx, 0, remoteIndex, g, nil, t)
			localIdx++
		}
	}
	return remoteIndex
}

// makeAllStronglySeeAll2All is the same as makeAllStronglySee, except each event has
// n - 1 remote ancestors.
func makeAllStronglySeeAll2All(g *Graph, startIndex IndexType, t *testing.T) (newIndex IndexType) {
	newIndex = startIndex + 2
	for k := 0; k < 2; k++ { // we need two sets of events to make them all strongly see
		prevEvents := make([][]EventPointer, len(g.tails)) // find the set of remote events
		for i := 0; i < len(g.tails); i++ {
			for j := 0; j < len(g.tails); j++ {
				if i == j {
					continue
				}
				prevEvents[i] = append(prevEvents[i], g.tails[j][0].LocalInfo)
			}
		}
		for i := 0; i < len(g.tails); i++ { // add the events
			ev := EventInfo{
				LocalInfo: EventPointer{
					ID:    IndexType(i),
					Index: g.tails[i][0].LocalInfo.Index + 1,
				},
				RemoteAncestors: prevEvents[i],
			}
			addEventInfo(ev, true, g, nil, t)
		}
	}
	return
}

func TestAllStronglySeeAll2All(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	initEvents := make([]*Event, len(g.tails))
	for i, nxt := range g.tails {
		initEvents[i] = nxt[0]
	}
	makeAllStronglySeeAll2All(g, 0, t)
	for _, nxt := range g.tails {
		for _, front := range initEvents {
			assert.True(t, g.stronglySees(front, nxt[0]))
		}
	}
}

func TestGraphGetDecAll2All(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var startIndex IndexType
	startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
	startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
	startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
	logging.Error(startIndex, g.tails[0][0].WI, g.tails[0][0].round)

	dec, _, _, _, _, _ := g.GetDecision(1)
	assert.True(t, dec)

	evs := g.getDecisionEventInfo(1)
	g2 := InitGraph(n, nmt, initHash)
	_, rem, inv := g2.AddEvents(evs)
	assert.Equal(t, 0, len(rem))
	assert.Equal(t, 0, len(inv))

	dec, _, _, _, _, _ = g2.GetDecision(1)
	assert.True(t, dec)
}

func TestMaxIndices(t *testing.T) {
	a := []IndexType{0, 1, 2, 3, 4, 5, 6}
	b := []IndexType{1, 1, 3, 2, 100, 0, 7}
	c := []IndexType{1, 1, 1, 1, 1, 1, 1}

	assert.Equal(t, []IndexType{1, 1, 3, 3, 100, 5, 7}, MaxIndices(a, b, c))
}

func TestGraphGC(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var startIndex IndexType
	ids := [][]IndexType{g.GetIndices()}
	for i := 0; i < 4; i++ { // enough to decide index 2
		startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
		ids = append(ids, g.GetIndices())
	}
	var dec bool
	dec, _, _, _, _, _ = g.GetDecision(1)
	assert.True(t, dec)
	dec, _, _, _, _, _ = g.GetDecision(2)
	assert.True(t, dec)
	dec, _, _, _, _, _ = g.GetDecision(3)
	assert.False(t, dec)

	for i, nxt := range ids {
		evs, idxs, err := g.GetMoreRecent(nxt)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(idxs))
		if i < len(ids)-1 {
			assert.True(t, len(evs) > 0)
		}
	}
	// gc the first decision
	g.CheckGCIndex(1, func(*Event) bool { return true })
	evs, idxs, err := g.GetMoreRecent(ids[0])
	assert.Nil(t, err)
	assert.Equal(t, []IndexType{1}, idxs)
	assert.True(t, len(evs) > 0)
	evs, idxs, err = g.GetMoreRecent(ids[1])
	assert.Nil(t, err)
	assert.Equal(t, 0, len(idxs))
	assert.True(t, len(evs) > 0)

	// decide again so we can gc the 2nd decision
	startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
	dec, _, _, _, _, _ = g.GetDecision(3)
	assert.True(t, dec)
	// gc the second decision
	g.CheckGCIndex(2, func(*Event) bool { return true })

	evs, idxs, err = g.GetMoreRecent(ids[0])
	assert.Nil(t, err)
	assert.Equal(t, []IndexType{1}, idxs)
	assert.True(t, len(evs) > 0)
	evs, idxs, err = g.GetMoreRecent(ids[1])
	assert.Nil(t, err)
	assert.Equal(t, []IndexType{1}, idxs)
	assert.True(t, len(evs) > 0)
	evs, idxs, err = g.GetMoreRecent(ids[2])
	assert.Nil(t, err)
	assert.Equal(t, 0, len(idxs))
	assert.True(t, len(evs) > 0)

	// decide again so we can gc the thrid decision
	startIndex = makeAllStronglySeeAll2All(g, startIndex, t)
	dec, _, _, _, _, _ = g.GetDecision(4)
	assert.True(t, dec)
	// gc the third decision
	g.CheckGCIndex(3, func(*Event) bool { return true })

	for i := 0; i < 3; i++ {
		evs, idxs, err = g.GetMoreRecent(ids[i])
		assert.Nil(t, err)
		assert.Equal(t, []IndexType{2, 1}, idxs)
		assert.True(t, len(evs) > 0)
	}
	evs, idxs, err = g.GetMoreRecent(ids[3])
	assert.Nil(t, err)
	assert.Equal(t, 0, len(idxs))
	assert.True(t, len(evs) > 0)

	nxtIds := g.GetIndices()
	nxtIds[0]++
	evs, idxs, err = g.GetMoreRecent(nxtIds)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(idxs))
	assert.True(t, len(evs) == 0)

	nxtIds[1]--
	evs, idxs, err = g.GetMoreRecent(nxtIds)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(idxs))
	assert.True(t, len(evs) == 1)
}

func TestGraphFork(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var idx IndexType = 1
	// create a fork at id 0, index 1
	addEvent(0, idx, 1, idx-1, g, nil, t)
	addEvent(0, idx, 2, idx-1, g, nil, t)
	// we should have a fork
	assert.Equal(t, 1, len(g.forks))

	// add two events at ID 1, seeing each fork, be sure the 2nd sees the fork
	ev1 := EventInfo{
		LocalInfo: EventPointer{
			ID:    1,
			Index: 1,
			Hash:  g.tails[1][0].MyHash,
		},
		RemoteAncestors: []EventPointer{{
			ID:    0,
			Index: 1,
			Hash:  g.tails[0][0].MyHash,
		}},
	}
	addEventInfo(ev1, false, g, nil, t)
	assert.Equal(t, 0, len(g.tails[1][0].seesForks))

	ev2 := EventInfo{
		LocalInfo: EventPointer{
			ID:    1,
			Index: 2,
			Hash:  g.tails[1][0].MyHash,
		},
		RemoteAncestors: []EventPointer{{
			ID:    0,
			Index: 1,
			Hash:  g.tails[0][1].MyHash,
		}},
	}
	addEventInfo(ev2, false, g, nil, t)
	assert.Equal(t, 1, len(g.tails[1][0].seesForks))

	// add an event that sees this event, it should see a fork
	addEvent(2, 1, 1, 2, g, nil, t)
	assert.Equal(t, 1, len(g.tails[2][0].seesForks))
}

func TestGraphDecideFork(t *testing.T) {
	g := InitGraph(n, nmt, initHash)
	var idx IndexType = 1
	// create a fork at id 0, index 1
	addEvent(0, idx, 1, idx-1, g, nil, t)
	addEvent(0, idx, 2, idx-1, g, nil, t)
	// we should have a fork
	assert.Equal(t, 1, len(g.forks))
	// have some events see the first and others the second
	for i := IndexType(1); i < n; i++ {
		ev := EventInfo{
			LocalInfo: EventPointer{
				ID:    i,
				Index: 1,
				Hash:  g.tails[i][0].MyHash,
			},
			RemoteAncestors: []EventPointer{{
				ID:    0,
				Index: 1,
				Hash:  g.tails[0][i%2].MyHash, // have half point to the different events of the fork
			}},
		}
		addEventInfo(ev, false, g, nil, t)
	}
	// make enough events to decide
	idx = makeAllStronglySee(g, idx, t)
	idx = makeAllStronglySee(g, idx, t)

	dec, _, _, _, _, bools := g.GetDecision(1)
	assert.True(t, dec)
	assert.False(t, bools[0])       // the 0 index should decide false because of the fork
	for _, nxt := range bools[1:] { // the other indices should decide true
		assert.True(t, nxt)
	}

}

func TestGetMissingEvents(t *testing.T) {
	g := InitGraph(n, nmt, initHash)

	addEvent(0, 1, 1, 0, g, nil, t)
	addEvent(1, 1, 0, 1, g, nil, t)
	addEvent(0, 2, 1, 1, g, nil, t)

	e1 := g.tails[0][0]
	ids := g.GetIndices()

	maxIDs, evs := g.GetMissingEvents(e1, ids)
	assert.Equal(t, 0, len(evs))

	checkIDs := make([]IndexType, len(g.tails))
	checkIDs[0] = 2
	checkIDs[1] = 1
	assert.Equal(t, checkIDs, maxIDs)

	ids[0] = 1
	ids[1] = 0
	maxIDs, evs = g.GetMissingEvents(e1, ids)
	assert.Equal(t, 2, len(evs))
	assert.Equal(t, checkIDs, maxIDs)

}
