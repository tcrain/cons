# Consensus and broadcast algorithms

This project focuses on algorithms that do not rely on synchrony
for correctness, and tolerate 1/3 of Byznatine faults.
Note the implementations may not follow exactly the papers
they are presented in.

## Multi-value consensus
Partial synchrony is needed for termination for these algorithms.
- [MvCons2](../consensus/cons/mvcons2) - PBFT/Tendermit like consensus algoirhtm.
A consensus instance consists of an inital broadcast by a predefined
coordinator, followed by all to all echo message, and an all to all commit message
resulting in decision. If a decision is not made after a timeout, the coordinator
is rotated, and decision is tried again. Messages from the previous
round must be used to ensure all non-faulty processes decided the same value.
Consensus messages must be signed.
- [MvCons3](../consensus/cons/mvcons3) - HotStuff based consensus. Rotating
coordinator based consensus consisting of coodrinator broadcast and
an echo where multiple consensus instances are "piggybacked" together.
Allows more frequent decisions, but higher latency between proposal and decision.
Designed to be used with multi or threshold signatures and CollectBroadcast: Commit
in test options.
Note that due to consensus instances being run in parallel,
state machines implementations can become more complicated
(see [statemachines](statemachines.md)).
**Note that MvCons3 currently does not support state machine membership
changing, except when using randomly selected membership.**
- [MvCons4](../consensus/cons/mvcons4) - Hash graph based consensus.
All members propose a value during each consensus step, and multiple
consensus instances are "piggybacked" together (i.e. this is similar
to MvCons3, except all members make a proposal instead of having
a leader).  During execution, nodes create events that contain hashes of
the events that they have seen of events from external nodes, creating
a directed acyclic graph. There are multiple ways messages can be broadcast
using the MvCons4BroadcastType option, the possible values are:
  - Normal: Messages are broadcast similar to other consensus algorithms.
  I.e a node creates an event at an index in the graph once it has seen
  events at the previous index from n-t distinct nodes. This then is broadcast
  through whatever network type is selected. 
  - Direct: When a node receives a new event from an external node, it creates
  a new local event, then sends it to a randomly chosen node. It keeps track
  of the events the receving node has seen so far based on local information
  and sends any missing dependencies that the random receiver may need.
  Each node is performing this operation concurrently (i.e at any point
  in time n of these operations should be in progress).
  - Sync: A node selects a random node, sending a vector clock of the
  indicies it has seen at each node. The receiver node creates a new local
  evnet, sending it back to the node, along with any dependines more advanced
  than those given by the vector clock. Compared to Direct, this should use
  less trafic, but creates an extra message step for every event creation.
  Each node is performing this operation concurrently (i.e at any point
  in time n of these operations should be in progress).

## Binary consensus

#### Asynchronous binary consensus
These algorithms use a common coin to terminate
after an expected constant number of rounds.
**Note that these algorithms do not currently support membership changes
as the use threshold cryptography (TODO implement distributed key gen).
(If using a predictable coin then membership changes can be allowed,
but the algorithms no longer guarantee constant time termination)**

- [BinConsRnd1](../consensus/cons/bincoinsrnd1) - Signature based, can
decide the value of the coin in each round (https://arxiv.org/abs/2002.04393). 
- [BinConsRnd2](../consensus/cons/bincoinsrnd2) - Signature based,
can decided 0 or 1 in each round (https://eprint.iacr.org/2000/034.pdf).
- [BinConsRnd3](../consensus/cons/bincoinsrnd3) - Signature based, combination
of BinConsRnd1 and BinConsRnd2 (https://arxiv.org/abs/2004.09547).
- [BinConsRnd4](../consensus/cons/bincoinsrnd4) - Not signature based, can
                                                  decide the value of the coin in each round (https://arxiv.org/abs/2002.08765). 
- [BinConsRnd5](../consensus/cons/bincoinsrnd5) - Not signature based,
can decided 0 or 1 in each round. Supports weak coins (https://arxiv.org/abs/2002.08765).
- [BinConsRnd6](../consensus/cons/bincoinsrnd6) - Not signature based,
combination of BinConsRnd4 and BinConsRnd5 (https://arxiv.org/abs/2004.09547).

#### Partial synchronous binary consensus

- [BinCons1](../consensus/cons/bincoins1) - Partial synchronous binary
consensus using signatures. (https://arxiv.org/abs/2001.07867)


## Binary to multi-value consensus reductions
Note these are simple reductions that allow ``nil`` to be decided,
TODO: implemnt more interesting reductions.
- [MvCons1](../consensus/cons/mvcons1) - Reduction to BinCons1, allows decision
in 3 message steps.
- [MvConsRnd1](../consensus/cons/mvcons1) - Reduction to BinConsRnd1, allows decision
                                            in 3 message steps.

## Reliable broadcast
Classical Bracha reliable broadcast (https://core.ac.uk/download/pdf/82523202.pdf)
- [RbBcast1](../consensus/cons/rbbcast1) - Signature based version of reliable broadcast,
allows delivery after 2 message steps in the case of a non-faulty broadcaster.
- [RbBcast2](../consensus/cons/rbbcast2) - Non-signature based, delivery after 3 message steps.
