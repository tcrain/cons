package graph

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
)

type Event struct {
	EventInfo
	MyHash              types.HashBytes
	localAncestor       *Event
	remoteAncestor      []*Event
	round               IndexType
	WI                  *witnessInfo // non-nil if this Event is a witness, nil otherwise
	known               []*Event     // first event at each id reachable from this node
	allAncestorsDecided bool         // set to true when all the ancestors witnesses of this event has decided
	seen                seen         // used during traversal
	seesForks           []*forkInfo
}

func (ev *Event) seesFork(id IndexType) bool {
	for _, nxt := range ev.seesForks {
		if nxt.id == id {
			return true
		}
	}
	return false
}

type forkInfo struct {
	id  IndexType
	evs []*Event
}

type seen byte

const (
	notSeen seen = iota
	falseSeen
	trueSeen
)

type witnessInfo struct {
	votes         [][]bool // the nodes that have path to this node from the follow round
	decided       decType
	decidedRound  IndexType // the round of the witness that caused this witness to decide
	stronglySeees []bool    // size n, true in Index i means the witness strongly sees the witness of round r-1 from node i
}

type EventInfo struct {
	LocalInfo       EventPointer   // Local id and index, plus hash of local ancestor
	RemoteAncestors []EventPointer // Remote ancestors
	Buff            []byte         // proposal
}

type EventPointer struct {
	ID, Index IndexType
	Hash      types.HashBytes
}

func (ac *EventPointer) Decode(reader io.Reader) (n int, err error) {
	var n1 int
	var v uint64
	if v, n1, err = utils.ReadUvarint(reader); err != nil {
		return
	}
	n += n1
	ac.ID = IndexType(v)

	if v, n1, err = utils.ReadUvarint(reader); err != nil {
		return
	}
	n += n1
	ac.Index = IndexType(v)

	if n1, ac.Hash, err = utils.ReadBytes(types.GetHashLen(), reader); err != nil {
		return
	}
	n += n1
	return
}

func (ac *EventPointer) Encode(writer io.Writer) (n int, err error) {
	var n1 int
	if n1, err = utils.EncodeUvarint(uint64(ac.ID), writer); err != nil {
		return
	}
	n += n1
	if n1, err = utils.EncodeUvarint(uint64(ac.Index), writer); err != nil {
		return
	}
	n += n1
	if len(ac.Hash) != types.GetHashLen() {
		panic(types.ErrInvalidHashSize)
	}
	if n1, err = writer.Write(ac.Hash); err != nil {
		return
	}
	n += n1
	return
}

func (ev *EventInfo) Encode(writer io.Writer) (n int, err error) {
	var n1 int
	if n1, err = ev.LocalInfo.Encode(writer); err != nil {
		return
	}
	n += n1

	if len(ev.RemoteAncestors) == 0 {
		err = types.ErrInvalidRemoteAncestorCount
		return
	}
	if n1, err = utils.EncodeUvarint(uint64(len(ev.RemoteAncestors)), writer); err != nil {
		return
	}
	n += n1
	for _, nxt := range ev.RemoteAncestors {
		if n1, err = nxt.Encode(writer); err != nil {
			return
		}
		n += n1
	}
	if n1, err = utils.EncodeUvarint(uint64(len(ev.Buff)), writer); err != nil {
		return
	}
	n += n1
	if n1, err = writer.Write(ev.Buff); err != nil {
		return
	}
	n += n1
	return
}

func (ev *EventInfo) Decode(reader io.Reader) (n int, err error) {
	var n1 int
	var v uint64
	if n1, err = (&ev.LocalInfo).Decode(reader); err != nil {
		return
	}
	n += n1
	if v, n1, err = utils.ReadUvarint(reader); err != nil {
		return
	}
	n += n1
	if v > config.MaxMsgSize || v == 0 {
		err = types.ErrInvalidRemoteAncestorCount
		return
	}
	ev.RemoteAncestors = make([]EventPointer, v)
	for i := 0; i < int(v); i++ {
		if n1, err = (&ev.RemoteAncestors[i]).Decode(reader); err != nil {
			return
		}
		n += n1
	}
	if v, n1, err = utils.ReadUvarint(reader); err != nil {
		return
	}
	n += n1
	if v > 0 {
		if n1, ev.Buff, err = utils.ReadBytes(int(v), reader); err != nil {
			return
		}
		n += n1
	}
	return
}
