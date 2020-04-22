# Causal and total ordering

The project supports two ways of ordering consensus instances,
total and causal. The main implemetation differences
come in the [statemachine](statemachines.md),
how consensus instances are identified, and in the
[cons_state.go](../consensus/cons/cons_state.go) vs
[causal_cons_state.go](../consensus/cons/causal_cons_state.go)
files which takes care of things like starting new consensus
instances, storing things to disk, recovery, etc..
Most consensus algorithms are compatible with both
ordering types, but certain test configurations may
only be compatible with one or the other, see
[testoptions.go](../consensus/types/testoptions.go)

## Total ordering
- Consensus instances are indentified by a uint64, where
the first instance starts with 1, and decisions happen
in incremental order at each integer.
- Note that in some cases consensus/statemachine computations
can take place in parallel (
when USECONCURRENT is > 0 in [testoptions.go](../consensus/types/testoptions.go),
or when using [MvCons3](consalgorithms.md). Decisions
still take place in a total order, but satemachine implementations
may become more complicated).
- In [cons_state.go](../consensus/cons/cons_state.go)
  - After a value is decided, a new consensus instance
  is allocated. When ready (i.e. if the local node
  is the current coordinator) the consensus instance then asks
  the state machine for the next proposal.
  - recovery/reliable channels:
  After a timeout where a node makes no progress,
  the node sends to it's neighbours its current index,
  if its neighbors have a larger or equal index, they send the state
  of the instances from that index.
  - after an instance decides, its state is stored to disk. Old instances
  are removed from memory given by KeepPast in [config.go](../config/config.go)

## Causal ordering
- Currently the main usage of causal ordering is allow the
use of non-terminating broadcasts (i.e. those that do not
solve consensus) to implement state machines that do not
require a total order. This may be expanded in the future
to thinks like sharding/partial replication.
- Consensus/broadcast instances are identified by a crypto hash.
This is computed by the hash of the consensus instances that it is
directly ordered after (as given by the state machine).
- In [causal_cons_state.go](../consensus/cons/causal_cons_state.go)
  - After decision this file does not directly allocate
  new consensus isntances.
  - Instead it is the responsibility of the state machine to
  tell this file when a new consensus instance should be allocated
  and how it should be ordered.
  - Recovery is done similarily to the total order, except is
  a little more complicated due to the more complex ordering.
  - Decided instances are stored to disk, but their ordering
  graph is kept in memory (TODO allow all to be stored to disk)
  