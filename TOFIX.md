- Cleanup of non-signed messages structure.
- When pubs are changed, update their IDs, plus my pub's ID
  - done => but now should look into making a more efficient, since we have to do linear operations each time the pub list is changed
- NOTE: random member slection does not support multisignature, because we need the VRFs for each signature.
TODO is there a way to do this?
  - should clean this up as well since we have different generate sig function after merge
- Cleanup function definitions for inputs taking functions.
- More efficient no progress messages, especially when using many nodes
over all to all connection
- Way to add membership changing for MVCons3
- Cleanup the state transition and creation for member the abs member
checkers.
Currently it works as follows:
    - Their state is created from the initial member checker.
    - Then if they want to take the previous member checkers state,
this must  be done by the top level object.
    - Then if keys
change since the previous member checker, the AbsGotDecision
is called with non-nil values.
- Use uvarint everywhere for serialization.
- Remove places where byz nodes can cause inf memory usage
  - For example in the mv cons can have inf proposals, instead should only keep the
  one you echoed, plus the lowest value one?
  - NW connections?
  - Bin cons rounds
  - Hashes for parent indices when using causal
    - Including futureMessage of the causal cons state object.
- Prevent flooding using NoProgressMessages
- **Restart from disk on failure when using LocalRandMember + causal ordering
may not work because the restart ordering might not be the same which can mean we run
a differnt SM and can cause prolems, for example the RBBcast to not terminate since we send 2 differnt proposals.
This also relates to the following point.**
- On restart add some time to recover from other nodes before directly going into starting consensus.
  - Also on that, store proposals to disk so on recover we don't send a different proposal.
  - For now tests with causal ordering and clear disk are not supported since they can create non-termination after sending confliting proposals
  - (it should not terminate because that indicated a faulty owner of the asset, but this doesn't work for the benchmark where we terminate after enough had make proposals)
  - Currently there is also a problem where the aux message of round 1 of the binary consensus
  is non-deterministic, because it depends on the order received.
- Currently causal sends hashes of non-consumed items no progress timeout.
This is inefficient since there can be many of these, should do this in a more efficient way (hash tree?)
(see TODO in causalconsinterfacestate.go).
- Object marshalling should use
  - encoding.BinaryMarshaler interface
    - MarshalBinary() (data []byte, err error)
  - encoding.BinaryUnmarshaler interface
    - UnmarshalBinary(data []byte) error
- **!!!! Check/cleanup the locking in the msg state objects**
**- When using RequestForwarder (LocalRand) and someone asks for an
index from disk, the entire state is returned, allowing the receiver
to compute your membership (should only send your local signature),
this is already done for instances still in memory.**
- Some consensus may need a proposal immediately vs getting a proposal when ready.
  - Currently there is a GetProposal function only
  - If a consensus does not make a proposal within a timeout nodes assume it is faulty and will proposal nil for that round
  - Another option could be to only start the timeout when there is something ready for proposal.
- Golang on Ubuntu on Windows sometimes crashes when printing strings of raw public keys,
(I think this is a bug), but anyway should be sure I am not printing/logging any random strings
  - This was in print AccountTable, to fix I print the pub keys as bytes instead of string
- When connections are added internally in NetMainChannel, for example in
SendToPub, then we don't garbage collect them because we don't know
how long they will last for.
- For local rand, all outputs of a decision will have the same members,
maybe needs to be different

- Make MVCons2 work for causal (MVcons1 should be fine,
mvCons3 not supported, mvCons2 needs to be able to decide nil)
- Update scripts
- Causal ordering - collect broadcast and include proofs missing
in causal consensus state.
  - also missing byz index check in check faulty broadcast
- When in init from disk dont:
  - make new signatures
  - validate signatures
  - count stats
  
- Random network with TCP connects all nodes together.

- Setting the sleep time, esp the initial sleep time for UDP connections.
- UDP connections in general.

- Remove use of globals. They breaks parallel testing, and is bad design.

- During experiments a firewall rule is added that opens all tcp and udp ports.

- Concurrency - In SM check validation based on concurrency enabled, and in commit.

- A signed message can have duplicates of the same signature and they will all be validated.
  - Should remove duplicates instead
  
- When saving to disk all the pubs for each unsigned message are tracked,
on sending this for recover, the full message is sent, but this should only be
the local pub and not all others, this also creates the identity problem in rand local network type
  - also see simpleMessageState.GotMsg
  
- **IMPORTANT:** Currently old decided instances are garbage collected, but should oly
garbage collect ones that have terminated if not using signatures.
(If using signatures can always GC immediatly because have proof of decision).
If using no-signatures can GC after the next instance decides but need
an echo message to ensure the slow ones get enough messages.

- **IMPORTANT:** - unsigned leaser messages need to be checked for validated by SM
  - see consinterfacestate.go line 178, // TODO ("check unsigend messages")