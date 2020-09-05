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

/*
Implementation of signature based binary consensus algorithm.
*/
package bincons1

import (
	"fmt"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	// "sync/atomic"
	"math"
)

type BinCons1 struct {
	cons.AbsConsItem
	Decided          int                  // -1 if a value has not yet been decided, 0 if 0 was decided, 1 if 1 was decided
	decidedRound     types.ConsensusRound // the round where the decision happened
	SkipTimeoutRound types.ConsensusRound // will not wait for timeouts up to this round
	gotProposal      bool                 // for sanity checks
	// priv             sig.Priv             // the private key of the local node
	roundTimers   []channelinterface.TimerInterface // running timers for rounds of consensus, on timeout the algorithm takes some action if no progress made
	IncludeProofs bool                              // true if messages should include signed values supporting the value in the message
	terminated    bool
}

func (sc *BinCons1) getMsgState() *MessageState { // TODO clean this up
	return sc.ConsItems.MsgState.(BinConsMessageStateInterface).GetBinMsgState().(*MessageState)
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *BinCons1) GetBinState(localOnly bool) ([]byte, error) {
	msg, err := messages.CreateMsg(sc.PreHeaders)
	if err != nil {
		panic(err)
	}
	bs, err := sc.ConsItems.MsgState.GetMsgState(sc.ConsItems.MC.MC.GetMyPriv(), localOnly,
		sc.GetBufferCount, sc.ConsItems.MC)
	if err != nil {
		return nil, err
	}
	_, err = messages.AppendHeader(msg, (messagetypes.ConsBinStateMessage)(bs))
	if err != nil {
		panic(err)
	}
	return msg.GetBytes(), nil
}

// GetProposeHeaderID returns the HeaderID messages.HdrBinPropose that will be input to GotProposal.
func (sc *BinCons1) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrBinPropose
}

// GenerateNewItem creates a new bin cons item.
func (*BinCons1) GenerateNewItem(index types.ConsensusIndex,
	items *consinterface.ConsInterfaceItems, mainChannel channelinterface.MainChannel,
	prevItem consinterface.ConsItem, broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &BinCons1{AbsConsItem: newAbsItem}
	items.ConsItem = newItem
	newItem.IncludeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.Decided = -1
	newItem.gotProposal = false
	newItem.Decided = -1
	newItem.decidedRound = 0
	newItem.SkipTimeoutRound = 0
	newItem.stopTimers()

	return newItem
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *BinCons1) GetCommitProof() []messages.MsgHeader {
	return GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), true, sc.ConsItems.MC)
}

// Start allows GetProposalIndex to return true.
func (sc *BinCons1) Start() {
	sc.AbsConsItem.AbsStart()
	if sc.CheckMemberLocal() { // if the current node is a member then send an initial proposal
		sc.NeedsProposal = true
	}
}

// CanSkipMvTimeout returns true if the during the multivalue reduction the echo timeout can be skipped
func (sc *BinCons1) CanSkipMvTimeout() bool {
	return sc.SkipTimeoutRound > 1
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *BinCons1) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GetMVInitialRoundBroadcast returns the type of binary message that the multi-value reduction should broadcast for round 0.
func (sc *BinCons1) GetMVInitialRoundBroadcast(val types.BinVal) messages.InternalSignedMsgHeader {
	auxMsg := messagetypes.NewAuxProofMessage(sc.AllowSupportCoin)
	auxMsg.BinVal = val
	auxMsg.Round = 1
	return auxMsg
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *BinCons1) GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error {
	sc.AbsGotProposal()
	bpm := hdr.(*messagetypes.BinProposeMessage)
	if bpm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	binMsgState := sc.getMsgState()

	binMsgState.Lock()
	// In case of recover, if we already sent a message then we don't want to send another
	roundStruct := binMsgState.getAuxRoundStruct(0)
	if roundStruct.sentProposal {
		binMsgState.Unlock()
		return nil
	}
	binMsgState.Unlock()

	logging.Infof("Got local proposal for index %v bin val %v", sc.Index, bpm.BinVal)
	auxMsg := messagetypes.NewAuxProofMessage(false)
	auxMsg.BinVal = bpm.BinVal
	auxMsg.Round = 0
	sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
		sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, sc.CommitProof...)

	return nil
}

// NeedsConcurrent returns 1.
func (sc *BinCons1) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// It processes AuxProofTimeout messages and AuxProof messages, moving through the rounds of consensus until a decision.
func (sc *BinCons1) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	_ *channelinterface.SendRecvChannel) (bool, bool) {

	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t
	binMsgState := sc.getMsgState()

	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	if deser.HeaderType == messages.HdrAuxProofTimeout && isLocal { // round timeout message
		if sc.Decided > -1 {
			return false, false
		}

		binMsgState.Lock()
		// Get the state for this round
		round := (types.ConsensusRound)(deser.Header.(messagetypes.AuxProofMessageTimeout))
		roundStruct := binMsgState.getAuxRoundStruct(round)
		// indicate that the timeout has passed
		roundStruct.timeoutState = cons.TimeoutPassed
		binMsgState.Unlock()

		// check if we can now advance rounds
		for sc.CheckRound(nmt, t, round, sc.MainChannel) {
			round++
		}

		return true, false
	}
	if deser.HeaderType == messages.HdrAuxProof { // aux proof message
		w := deser.Header.(*sig.MultipleSignedMessage).GetBaseMsgHeader().(*messagetypes.AuxProofMessage)
		round := w.Round

		// If we already decided then don't need to process this message
		// if sc.Decided > -1 && (round > sc.decidedRound+2 || sc.StopOnCommit) {
		if sc.terminated {
			logging.Infof("Got a msg for round %v, but already decided in round %v", round, sc.decidedRound)
			return false, false
		}

		// check if we can now advance rounds (note the message was already stored to the bincons state in BinCons1MessageState.GotMsg
		for sc.CheckRound(nmt, t, round, sc.MainChannel) {
			round++
		}
		return true, true
	}
	panic(fmt.Sprint("got invalid message header", deser.HeaderType))
}

// SetInitialState does noting for this algorithm.
func (sc *BinCons1) SetInitialState([]byte) {}

func (sc *BinCons1) checkDone(round types.ConsensusRound, nmt, t int) bool {
	_ = t
	if sc.terminated {
		return true
	}
	if sc.Decided < 0 {
		return false
	}
	if round < sc.decidedRound { // this message is from an earlier round
		return true
	}

	switch sc.StopOnCommit {
	case types.Immediate:
		sc.terminated = true
		return true
	case types.SendProof:
		// Send a proof of decision
		prfMsgs := GetBinConsCommitProof(messages.HdrAuxProof, sc, sc.getMsgState(), false, sc.ConsItems.MC)
		sc.BroadcastFunc(nil, sc.ConsItems, nil, true,
			sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, prfMsgs...)
		sc.terminated = true
		return true
	case types.NextRound:
		if round > sc.decidedRound+2 {
			return true
		}
		return false
	default:
		panic(sc.StopOnCommit)
	}
}

// CheckRound checks for the given round if enough messages have been received to progress to the next round.
func (sc *BinCons1) CheckRound(nmt int, t int, round types.ConsensusRound,
	mainChannel channelinterface.MainChannel) bool {

	binMsgState := sc.getMsgState()
	binMsgState.Lock()
	defer binMsgState.Unlock()

	// get the round state
	roundStruct := binMsgState.getAuxRoundStruct(round)

	numMsgs := roundStruct.TotalBinMsgCount // roundStruct.BinNums[0] + roundStruct.BinNums[1]
	// We can skip timeouts on older rounds if a non-fault process has made it to this round
	if numMsgs > t {
		sc.SkipTimeoutRound = round
	}
	// We can only perform an action if we have enough messages (n-t)
	if numMsgs < nmt {
		return false
	}
	var mod types.BinVal // mod is the value we can decide for this round (if we have enough messages)
	// roud 0 is a special case where we cant decided, and we prioritize 1 for the next round
	if round == 0 {
		mod = 1
	} else {
		mod = types.BinVal(round % 2)
	}
	notMod := 1 - mod
	validsR := binMsgState.getValids(nmt, t, round)
	// If we got enough messages for the mod and mod is valid for the round then we can decide
	if round > 0 && validsR[mod] && roundStruct.BinNums[mod] >= nmt {
		// Sanity checks
		if sc.Decided > -1 && types.BinVal(sc.Decided) != mod {
			panic("Bad cons 1")
		}
		if sc.Decided == -1 { // decide!
			logging.Infof("Decided bin round %v, binval %v, index %v", round, mod, sc.Index)
			// Once we decide we dont need timers
			sc.stopTimers()
			sc.SkipTimeoutRound = math.MaxUint32
			sc.decidedRound = round
			sc.ConsItems.MC.MC.GetStats().AddFinishRound(round, mod == 0)
		}
		if roundStruct.BinNums[notMod] >= nmt { // sanity check
			logging.Error("check", mod, round, validsR[mod], roundStruct.BinNums[mod], roundStruct.BinNums[notMod], nmt, sc.Index)
			panic("More than t faulty")
		}
		sc.Decided = int(mod)
		// Only send next round msg after deciding if necessary
		// TODO is other stopping mechanism better?
		if sc.StopOnCommit == types.Immediate {
			sc.terminated = true
			return true
		}
		if roundStruct.BinNums[notMod] == 0 || !validsR[notMod] || sc.StopOnCommit == types.Immediate {
			return true
		}
	}
	if sc.checkDone(round, nmt, t) {
		return true
	}
	if !roundStruct.sentProposal && sc.CheckMemberLocal() { // if we havent sent a message for the following round and we are a member of this consensus then enter
		if round > types.ConsensusRound(t) && sc.SkipTimeoutRound <= round { // if we haven't started a timeout for this round, then set one up
			if roundStruct.timeoutState == cons.TimeoutNotSent {
				roundStruct.timeoutState = cons.TimeoutSent
				deser := []*channelinterface.DeserializedItem{
					{
						Index:          sc.Index,
						HeaderType:     messages.HdrAuxProofTimeout,
						Header:         (messagetypes.AuxProofMessageTimeout)(round),
						IsLocal:        types.LocalMessage,
						IsDeserialized: true}}
				sc.roundTimers = append(sc.roundTimers, mainChannel.SendToSelf(deser, cons.GetTimeout(round, t)))
				return true
			} else if roundStruct.timeoutState == cons.TimeoutSent {
				return true
			}
		}

		// validCount is (at least) the total number of valid messages from distinct processes
		// this is different than roundStruct.TotalBinMsgCount because that is the total number of messages
		// received from different processes (even if their bin value isn't valid)
		// roundStruct.TotalBinMsgCount >= nmt is checked earlier and ensures we have received at least n-t
		// messages from distinct processes
		// so if just 1 or 0 is valid (but not both) then validCount >= nmt ensures that we have received n-t valid
		// messages from distinct processes
		// if both 1 and 0 are valid then roundStruct.TotalBinMsgCount >= nmt ensures that we have received n-t
		// valid messages from distinct processes
		var validCount int
		if validsR[0] {
			validCount += roundStruct.BinNums[0]
		}
		if validsR[1] {
			validCount += roundStruct.BinNums[1]
		}
		// We can send the proposal for the next round if we have enough valid messages
		if validCount >= nmt {
			validsRp1 := binMsgState.getValids(nmt, t, round+1)
			var est types.BinVal // the estimate for the next round
			// the estimate is chosen based on preference
			// (1) notMod is chosen if notMod is valid and we have received n-t notmod signatures
			// (2) otherwise mod is chosen if mod is valid and we have received at least 1 mod signature
			// (3) otherwise notMod is chosen
			if validsR[notMod] && roundStruct.BinNums[notMod] >= nmt && validsRp1[notMod] { // preference (1)
				est = notMod
			} else if validsR[mod] && roundStruct.BinNums[mod] > 0 && validsRp1[mod] { // preference (2)
				est = mod
			} else { // preference (3)
				est = notMod
				// sanity checks
				if !validsR[notMod] {
					panic("should be valid")
				}
				if !validsRp1[notMod] {
					panic(fmt.Sprintf("should be valid next round %v, %v, %v, %v, %v, %v, %v %v, %v",
						est, mod, validsR, validsRp1, round, roundStruct.BinNums[0], roundStruct.BinNums[1], t, nmt))
				}
				if roundStruct.BinNums[notMod] == 0 {
					panic("should have messages")
				}
			}
			// sanity check
			if !validsRp1[mod] && !validsRp1[notMod] {
				panic(fmt.Sprint("neither valid", round+1))
			}
			// Broadcast the message for the next round
			auxMsg := messagetypes.NewAuxProofMessage(false)
			var prfMsgs []*sig.MultipleSignedMessage
			auxMsg.BinVal = est
			auxMsg.Round = round + 1
			sc.ConsItems.MC.MC.GetStats().AddParticipationRound(round + 1)
			roundStruct.sentProposal = true // Set to true before checking if we are a member, since check member will always
			// give the same result for this round
			if sc.CheckMemberLocalMsg(auxMsg) {
				if sc.IncludeProofs {
					// collect signatures to support your choice
					var err error
					sigCount, _, _, err := sc.GetBufferCount(auxMsg, sc.GeneralConfig, sc.ConsItems.MC)
					if err != nil {
						panic(err)
					}
					prfMsgs, err = binMsgState.GenerateProofs(messages.HdrAuxProof, sigCount, round+1, est,
						sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
					if err != nil {
						// if MvCons and round == 2, est == 1, then we could not have decided 0, so we dont need proofs
						if !(round+1 == 2 && est == 1 && binMsgState.mv1Valid) {
							logging.Error(err, "est:", est, "round:", round+1, "cons idx:", sc.Index)
							panic("should have proofs")
						}
						prfMsgs = nil
					}
				}
				prfs := make([]messages.MsgHeader, len(prfMsgs))
				for i, nxt := range prfMsgs {
					prfs[i] = nxt
				}
				sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
					sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfs...)
				// cons.BroadcastBin(nil, sc.ByzType, sc, auxMsg, mainChannel, prfs...)
			}
		}
	} else {
		// sanity checks? TODO
	}

	return true
}

// HasReceivedProposal panics because BonCons has no proposals.
func (sc *BinCons1) HasValidStarted() bool {
	panic("unused")
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// Here the only message type is messages.HdrAuxProof, it returns n-t, n for the thresholds.
func (*BinCons1) GetBufferCount(hdr messages.MsgIDHeader, _ *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch hdr.GetID() {
	case messages.HdrAuxProof:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	default:
		return 0, 0, nil, types.ErrInvalidHeader
	}
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// Here only messages.HdrAuxProof are valid headerIDs.
func (*BinCons1) GetHeader(emptyPub sig.Pub, _ *generalconfig.GeneralConfig, headerType messages.HeaderID) (
	messages.MsgHeader, error) {

	switch headerType {
	case messages.HdrAuxProof:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewAuxProofMessage(false)), nil
	default:
		return nil, types.ErrInvalidHeader
	}
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *BinCons1) HasDecided() bool {
	return sc.Decided > -1
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *BinCons1) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *BinCons1) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil, nil,
		sc.GeneralConfig.AllowConcurrent > 0
}

// GetDecision returns the binary value decided as a single byte slice.
func (sc *BinCons1) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	if sc.Decided == -1 {
		panic("should have decided")
	}
	return nil, []byte{byte(sc.Decided)}, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
}

// stopTimers stops any current running round timers.
func (sc *BinCons1) stopTimers() {
	for _, t := range sc.roundTimers {
		t.Stop()
	}
	sc.roundTimers = nil
}

// GetConsType returns the type of consensus this instance implements.
func (sc *BinCons1) GetConsType() types.ConsType {
	return types.BinCons1Type
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *BinCons1) PrevHasBeenReset() {
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *BinCons1) SetNextConsItem(_ consinterface.ConsItem) {
}

// ShouldCreatePartial returns true if the message type should be sent as a partial message
func (sc *BinCons1) ShouldCreatePartial(_ messages.HeaderID) bool {
	return false
}

/////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////

// GetBinDecided returns -1 if not decided, or the decided value and the decided round
func (sc *BinCons1) GetBinDecided() (int, types.ConsensusRound) {
	return sc.Decided, sc.decidedRound
}

// GenerateMessageState generates a new message state object given the inputs.
func (*BinCons1) GenerateMessageState(gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewBinCons1MessageState(false, gc)
}

// Collect is called when the item is being garbage collected.
func (sc *BinCons1) Collect() {
	sc.AbsConsItem.Collect()
	sc.stopTimers()
}
