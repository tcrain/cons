- More interesting reductions from binary to multi-value cons.
- More interesting state machines.
- More interesting on disk store - probably want multi-versioned LSM
with in memory cache, but something more streamlined than level or rocks.
- Snapshotting.
- Slow synchronous bakup path, allowing tolerating n/2 faults and
to ensure eventual consistency.
- Penalties for malicious behaviour.
- Add fuzzing for intpus and messages.
- For random forwarder make an option to not change randomly every time,
to allow for some stabilization of the network, but still have the random properties
- P2P propgation layer for for example updates to the addresses of participants,
or transaction propagation before proposal.
- Byz type that sends different init messages to half and half.
- Allow use of shorter identifiers in transaction proposers, now the full public key is used in the encoded transaction.

- Byz type that tries to make the items go extra rounds

- keep stats on number of timeouts in buffer forwarder
- fix bench results with long names (and repeating ones, i.e.
should only create a new config for each test difference, not each
different in each item)
- figure out good function for threshold counts in broadcaster


- make firewall rules
- choose graphs to make

- when to start the next consensus (eg timeout for leader message)
  - use sm to say, either start immediately, or start when sm is ready
  
- encryption over udp