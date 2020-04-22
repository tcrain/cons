# Scaling mechanisms
Various configurations are implemented to allow scaling.
The idea is to make them general so they can be applied
to all (or most of) the consensus algorithms.

## For small to mid size networks
The idea of this is to use all-to-one and one-to-all broadcasts
to replace all-to-all broadcasts. This will be used in combination
with threshold or multi signatures to reduce the network and
cryptography costs.
See (https://arxiv.org/abs/1803.05069) and (https://arxiv.org/abs/1804.01626)

This setting is found in [testoptions.go](../consensus/types/testoptions.go)
for setting **CollectBroadcast**
- Full - This is the normal all to all broadcast at each step
- Commit - The commit message of consensus will be sent to the next coordinator instead of to all other nodes.
For 2 step consensus like MvCons3 (HotStuff) this will mean there are no all to all broadcasts.
For 3 step consensus (MvCons1, MvCons2) this means the middle step will remain an all to all step.  
- EchoCommit - This is for 3 step consensus algorithms (MvCons1, MvCons2), here the middle (echo)
message will be broadcast to the coordinator who will combine the signatures and broadcast them.
This adds an additional message step (so they become 4 step algorithms), but they
no longer have any all to all broadcasts.


## For small to large networks
The idea here is to use multi-signatures and a peer to peer overlay network where
the fan out is a small constant.

The initial proposal is broadcast throughout the network.
The messages that are normally performed as all to all broadcasts (echo/commit messages)
are first sent to the direct neighbours. Once a threshold of messages are collected
the signatures are merged, then sent to a different set of neighbors.
The idea is that at each step the number of signatures merged into the multisignature
will grow exponentially, thus meaning nodes will only need to perform logaritmic
number of signature validations/broadcasts.

This is similar to the techniques of (https://www.usenix.org/system/files/conference/usenixsecurity16/sec16_paper_kokoris-kogias.pdf),
except there a tree construction is used, which is more efficient, but maybe less
fault tolerant.

For this sort of tests see [testoptions.go](../consensus/types/testoptions.go)
options
- BufferForwarder
- IncludeCurrentSigs
- AdditionalP2PNetworks
- NetworkType
- FanOut
- UseMultisig

## For large+ networks
Here techniques are used to choose the consensus member randomly.
The members should be unpredictable and only be member for a short time
based on the expected fault model.
There are two points to consider when using such models:
- While they still maybe considered asynchronous by defining specific fault models,
they are inherently adding an extra layer of synchrony as the few selected
members have to be changed before they can be corrupted.
- They will tolerate less faults than the underlying consensus algorithms
allow (i.e. to ensure with high probability the random selection has less that 1/3 faults).

#### VRF member selection
Here the members are selected randomly based on the output of a verifiable
random function computed from the previous output of a previous verifialbe
random function, see (https://arxiv.org/abs/1607.01341).
Note that this slightly complicates the consensus algoirhtms as there is no fixed
leader, so nodes who suspect they might be the leader broadcast a proposal,
then nodes must wait for a timeout and choose the leader as the one with the best
score they have received so far.

There are 3 main ways on how frequently the VRF is computed, from
[testoptions.go](../consensus/types/testoptions.go)
- VRFPerMessage - each node generates a VRF for each message broadcast and sees if it is a member, this has the most frequent membership change
- VRFPerCons - each node generates a VRF once per consensus instance and sees if it is a member.
- KnownPerCons - here the members are generated from a single VRF coming from the leader of the previous consensus instance,
this is the least secure, but allows consensus algorithms to be used as normal. 

Some additional [testoptions.go](../consensus/types/testoptions.go) that can be used together:
- NetworkType
- FanOut
- RndMemberCount 

#### Local random membership selection
Here the initial proposal is signed, and is propagated as normal
throughout the network.
For the following messages (echo/broadcast) each node locally
chooses a random set of members from which it connects
(subscribes) to. So each node has a (possilby unique) set
of members known only to itself. Again the members should
only remain for a short period as it is a small set, which
is why being locally known helps. Currently the local set of members
can be changed per consensus/broadcast instance, by setting
LocalRandMemberChange in [testoptions.go](../consensus/types/testoptions.go).
Ideally this would change more frequently (see issue #3).

Given that the set of members is only known locally,
consensus algorihtms that require signatures are not valid here
as they use proofs of validity that depend on an exact set of nodes.
Most of the algoirhtms that do not require signature should work though
as they rely on echo/deliver thresholds that are more natural in this model.
This is based on (https://arxiv.org/pdf/1812.10844.pdf)

Note that this model is also designed with causal ordering in mind,
see [ordering](ordering.md) and [statemachines](statemachines.md).

Settings for [testoptions.go](../consensus/types/testoptions.go):
- RndMemberCount
- LocalRandMemberChange
- NetworkType = types.RequestForwarder
- RndMemberType = types.LocalRandMember
