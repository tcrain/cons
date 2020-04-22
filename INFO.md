## Path of a message
- (Parallel) Passed from network to one of the message processing threads.
- (Parallel) Message processing threads:
  - Message is deserialized. Uses the cons function to check what
are valid messages to deserialize.
  - Message is passed to the message state. Calls got msg.
    - Default:
      - checks for duplicates
      - Checks membership.
      - Validates signature.
      - stores to state (for duplicate)
    - Overwritten for some cons:
      - Checks coordinator message is from coordinator.
      - Updates message types counts.
- Message is sent to main thread
  - Checks if the message type needs SM validation.
  - If yes:
    - SM is allocated if not already done.
    - Message is validated by the SM.
  - Message is processed by the cons.

## Total Order Proposals
- Cons decides, next cons is allocated.
  - (If local is leader:) Cons returns true from NeedsProposal.
     - Next SM is allocated from previous SM.
     - Tells SM a proposal is needed.
     - SM sends local message when proposal is ready.
     - Cons receives local message -> generates a proposal message.
  - Proposal message is received.
    - Follows path of message in previous section.
  
## How to do causal proposals?
- Differences:
  1. As soon as SM is created, proposal is ready.
    - No need to check with with cons if needed.
  - Want:
    - Multiple SM use the same cons instance.
- Cons instance needs multiple cons indecies?
- Proposal message needs to have multiple indecies.

## Some notes on profiling

- Profile regex
  - ``go tool pprof -text -cum -show='DeserializeMessage|VerifySig|GenerateSig$'  rpcnode cpuprof0.out``
  - ``go tool pprof -text -cum -show='DeserializeMessage|VerifySig|GenerateSig$|SimpleMessageState*.*GotMsg|CombinePartialSigs'  rpcnode cpuprof0.out``
- For memory subtract the finish from the start
 - ``go tool pprof -base=memprof_start0_proc0.out rpcnode memprof_finish0_proc0.out``

go tool pprof -text -cum -unit=bytes -show='MultipleSignedMe
ssage.*Sign' -base=profile_memstart_0_1_4_MvCons2.out profile_memend_0_1_4_MvCons2.out

- StopOnCommit means that the binary consensus will not go past
the round of decision.
  - In case of Byz procs this can mean some may not decide
  until they receive the no progress catchup messages
  - Unless IncludeProofs is true as in this case, the inital
  message of the next consesnus index will cointain the proof for
  decision of the previous round (similar to collect broadcast).