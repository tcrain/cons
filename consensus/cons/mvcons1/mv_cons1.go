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
Implementation of multi-valued consensus reduction to BinCons1.
*/
package mvcons1

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/cons/binconsrnd1"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"sort"
)

// MvCons1 is a simple leader-based multi-value to binary consensus reduction using BinCons1.
// If the binary consensus terminates with 0 a null value is decided, otherwise the leaders proposal is decided.
// Assuming the leader is correct and the timeout configured correctly, termination happens in 3 message setps.
// (1) an init message broadcast from the leader
// (2) an all to all echo message containing the hash of the init
// (3) a aux proof message for round 1 of the binary consensus, allowing 1 to be decided in the binary consensus.
// If this does not happen, nodes start supporting 0 after a timeout and 0 may be decided.
// TODO generate the proofs from here for round 2, est 1 for bin cons (which would be n-t echos)
type MvCons1 struct {
	cons.AbsConsItem
	cons.AbsMVRecover
	binCons             cons.BinConsInterface                                // the binary consensus object
	decisionHash        types.HashStr                                        // the hash of the decided value
	decisionInitMsg     *messagetypes.MvInitMessage                          // the actual decided value (should be same as proposal if nonfaulty coord)
	proposerPub         sig.Pub                                              // the public key of the proposer of the decided message
	sentEcho            bool                                                 // true if an echo message was sent
	zeroHash            types.HashBytes                                      // if we don't receive an init message after the timeout we just send a zero hash to let others know we didn't receive anything
	validatedInitHashes map[types.HashStr]*channelinterface.DeserializedItem // hashes of init messages that have been validated by the state machine
	sortedInitHashes    cons.DeserSortVRF                                    // valid init messages sorted by VRFID, used when random member selection is enabled
	includeProofs       bool                                                 // true if the binary consensus should include proofs that messages are valid

	initTimer        channelinterface.TimerInterface // timer for the init message
	echoTimer        channelinterface.TimerInterface // timer for the echo messages
	echoTimeOutState cons.TimeoutState               // the timeout concerning the echo message
	initTimeoutState cons.TimeoutState               // the timeout concerning the init message
}

// GetBinState returns the entire state of the consensus as a string of bytes using MessageState.GetMsgState() as the list
// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning).
func (sc *MvCons1) GetBinState(localOnly bool) ([]byte, error) {
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
		return nil, err
	}
	return msg.GetBytes(), nil
}

// GetProposeHeaderID returns the HeaderID messages.HdrMvPropose that will be input to GotProposal.
func (sc *MvCons1) GetProposeHeaderID() messages.HeaderID {
	return messages.HdrMvPropose
}

// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
func (sc *MvCons1) GetCommitProof() []messages.MsgHeader {
	return sc.binCons.GetCommitProof()
}

// GenerateNewItem creates a new cons item.
func (*MvCons1) GenerateNewItem(index types.ConsensusIndex, items *consinterface.ConsInterfaceItems,
	mainChannel channelinterface.MainChannel, prevItem consinterface.ConsItem,
	broadcastFunc consinterface.ByzBroadcastFunc,
	gc *generalconfig.GeneralConfig) consinterface.ConsItem {

	newAbsItem := cons.GenerateAbsState(index, items, mainChannel, prevItem, broadcastFunc, gc)
	newItem := &MvCons1{AbsConsItem: newAbsItem}
	newItem.includeProofs = gc.Eis.(cons.ConsInitState).IncludeProofs
	newItem.zeroHash = types.GetZeroBytesHashLength()
	var binConsItem consinterface.ConsItem
	switch gc.ConsType {
	case types.MvBinCons1Type:
		binConsItem = &bincons1.BinCons1{}
	case types.MvBinConsRnd1Type:
		binConsItem = &binconsrnd1.BinConsRnd1{}
	default:
		panic(gc.ConsType)
	}
	newItem.binCons = binConsItem.GenerateNewItem(index, items, mainChannel,
		prevItem, broadcastFunc, gc).(cons.BinConsInterface)

	newItem.validatedInitHashes = make(map[types.HashStr]*channelinterface.DeserializedItem)
	newItem.InitAbsMVRecover(index, gc)
	items.ConsItem = newItem

	return newItem
}

// HasReceivedProposal returns true if the cons has received a valid proposal.
func (sc *MvCons1) HasValidStarted() bool {
	return len(sc.validatedInitHashes) > 0
}

// NeedsConcurrent returns 1.
func (sc *MvCons1) NeedsConcurrent() types.ConsensusInt {
	return 1
}

// SetInitialState does noting for this algorithm.
func (sc *MvCons1) SetInitialState([]byte) {}

// Start allows GetProposalIndex to return true.
func (sc *MvCons1) Start() {
	sc.AbsConsItem.AbsStart()

	if sc.CheckMemberLocal() {
		logging.Infof("Got local start multivalue index %v", sc.Index)
		// if memberChecker.CheckMemberBytes(sc.index, sc.Pub.GetPubString()) != nil {
		// Check if we are the coord, if so send coord message
		// Otherwise start timeout if we havent already received a proposal
		newMsg := messagetypes.NewMvInitMessage()
		_, _, err := consinterface.CheckCoord(sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC, 0, newMsg.GetMsgID())
		if err == nil {
			sc.NeedsProposal = true
			logging.Info("I am coordinator", sc.GeneralConfig.TestIndex, sc.Index)
		}
		// Start the init timer
		sc.initTimer = cons.StartInitTimer(0, sc.ConsItems, sc.MainChannel)
		sc.initTimeoutState = cons.TimeoutSent
	}
}

// GetProposalIndex returns sc.Index - 1.
// It returns false until start is called.
func (sc *MvCons1) GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool) {
	if sc.NeedsProposal {
		return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), true
	}
	return types.ConsensusIndex{}, false
}

// GotProposal takes the proposal, creates a round 0 AuxProofMessage and broadcasts it.
func (sc *MvCons1) GotProposal(hdr messages.MsgHeader,
	mainChannel channelinterface.MainChannel) error {

	pm := hdr.(*messagetypes.MvProposeMessage)
	if pm.Index.Index != sc.Index.Index {
		panic("Got bad index")
	}
	if sc.sentEcho {
		return nil
	}
	newMsg := messagetypes.NewMvInitMessage()
	newMsg.Proposal = pm.Proposal
	newMsg.ByzProposal = pm.ByzProposal

	sc.broadcastInit(newMsg, mainChannel)

	return nil
}

func (sc *MvCons1) checkSendEcho() {
	selectRandMembers := consinterface.ShouldWaitForRndCoord(sc.ConsItems.MC.MC.RandMemberType(), sc.GeneralConfig)
	if !sc.sentEcho && // we haven't sent an echo
		len(sc.sortedInitHashes) > 0 && // we have received an init and either (a) or (b)
		((selectRandMembers && sc.checkInitTimeoutPassed()) || // (a) we are selecting random members, and our init timeout ran out
			!selectRandMembers) { // (b) we are not selecting random members

		// send an echo message containing the hash of the proposal
		sort.Sort(sc.sortedInitHashes)
		// sc.sentEcho = true
		w := sc.sortedInitHashes[0].Header.(*sig.MultipleSignedMessage)
		proposalHash := types.GetHash(w.GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal)
		sc.broadcastEcho(proposalHash, sc.MainChannel)
	}

	// If we havent received a proposal yet, then we send a 0 msg for bin cons round 1.
	// (the first argument is 0 here because we the structure keeps track of having sent proposal for the round+1)
	if sc.checkInitTimeoutPassed() {
		msgState := sc.ConsItems.MsgState.(*MessageState)

		if !sc.sentEcho &&
			!msgState.BinConsMessageStateInterface.SentProposal(0, true, sc.ConsItems.MC) {
			// we broadcast an echo with a blank hash to let others know we support 0
			// this is so every gets at least n-t echo messages
			// TODO could also just count echo and round 1 aux proof 0's messages from unique senders
			sc.broadcastEcho(types.GetZeroBytesHashLength(), sc.MainChannel)
			logging.Infof("Supporting 0 for index %v, since no init recieved", sc.Index)

			auxMsg := sc.binCons.GetMVInitialRoundBroadcast(0)
			// TODO is this OK in rand member checker by ID, since we have different IDs for echo and aux???
			sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
				sc.ConsItems.FwdChecker.GetNewForwardListFunc(), sc.MainChannel, sc.GeneralConfig, nil)
			// cons.BroadcastBin(nil, sc.GeneralConfig.ByzType, sc.binCons, auxMsg, sc.MainChannel)
		}
	}
}

// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function processes the message and update the
// state of the consensus.
// For this consensus implementation messageState must be an instance of BinConsMessageStateInterface.
// It returns true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
// The following are the valid message types:
// messages.HdrMvInit is the leader proposal, once this is received an echo is sent containing the hash, and starts the echo timeoutout.
// messages.HdrMvInitTimeout the init timeout is started in GotProposal, if we don't receive a hash before the timeout, we support 0 in bin cons.
// messages.HdrMvEcho is the echo message, when these are received we run CheckEchoState.
// messages.HdrMvEchoTimeout once the echo timeout runs out, without receiving n-t equal echo messages, we support 0 in bin cons.
// messages.HdrMvRequestRecover a node terminated bin cons with 1, but didn't get the init message, so if we have it we send it.
// messages.HdrMvRecoverTimeout if a node terminated bin cons with 1, but didn't get the init mesage this timeout is started, once it runs out, we ask other nodes to send the init message.
// messages.HdrAuxProof messages are passed to the binray cons object.
func (sc *MvCons1) ProcessMessage(
	deser *channelinterface.DeserializedItem,
	isLocal bool,
	senderChan *channelinterface.SendRecvChannel) (bool, bool) {

	// msgState := sc.MessageState.(*MvCons1MessageState)

	if !deser.IsDeserialized {
		panic("should have deserialized message by now")
	}
	if deser.Index.Index != sc.Index.Index {
		panic("got wrong idx")
	}
	switch deser.HeaderType {
	case messages.HdrMvInitTimeout, messages.HdrMvEchoTimeout:
		if !isLocal {
			panic("should be local")
		}
		// Drop timeouts if already decided
		if dec, _ := sc.binCons.GetBinDecided(); dec > -1 {
			return false, false
		}
	}
	t := sc.ConsItems.MC.MC.GetFaultCount()
	nmt := sc.ConsItems.MC.MC.GetMemberCount() - t
	switch deser.HeaderType {
	case messages.HdrMvRecoverTimeout:
		if !isLocal {
			panic("should be local")
		}
		if sc.decisionInitMsg == nil {
			// we have not terminated binary consensus, but not received the init message after a timeout so we request it from neighbour nodes.
			sc.BroadcastRequestRecover(sc.PreHeaders, []byte(sc.decisionHash), sc.ConsItems.FwdChecker, sc.MainChannel,
				sc.ConsItems)
		}
		return false, false
	case messages.HdrMvInitTimeout:
		sc.initTimeoutState = cons.TimeoutPassed
		sc.checkSendEcho()
		return false, false
	case messages.HdrMvEchoTimeout:
		// we didnt receive enough echo messages after the init message.
		sc.echoTimeOutState = cons.TimeoutPassed
		sc.checkEchoState(t, nmt, sc.MainChannel)
		return true, false
	case messages.HdrMvInit:
		w := deser.Header.(*sig.MultipleSignedMessage)
		// sanity checks to ensure the init message comes from the coordinator
		_, _, err := consinterface.CheckCoord(w.SigItems[0].Pub, sc.ConsItems.MC, 0, w.GetMsgID())
		if err != nil {
			panic("should have handled this in GotMsg")
		}
		// Store it in a map of the proposals
		hashStr := types.HashStr(types.GetHash(deser.Header.(*sig.MultipleSignedMessage).InternalSignedMsgHeader.(*messagetypes.MvInitMessage).Proposal))
		if len(sc.validatedInitHashes) > 0 {
			logging.Infof("Recived duplicate proposals from coord at index %v", sc.Index)
		}
		sc.validatedInitHashes[hashStr] = deser
		var shouldForward bool
		sc.sortedInitHashes, shouldForward = cons.CheckForwardProposal(deser, hashStr, sc.decisionHash,
			sc.sortedInitHashes, sc.ConsItems)
		logging.Info("Got an mv init message of len", len(w.GetBaseMsgHeader().(*messagetypes.MvInitMessage).Proposal))

		sc.checkSendEcho()
		sc.checkEchoState(t, nmt, sc.MainChannel)
		// send any recovers that migt have requested this init msg
		sc.SendRecover(sc.validatedInitHashes, sc.InitHeaders, sc.ConsItems)
		return true, shouldForward
	case messages.HdrMvEcho:
		// check if we have enough echos to decide
		sc.checkEchoState(t, nmt, sc.MainChannel)
		return true, true
	case messages.HdrMvRequestRecover: // a node terminated bin cons, but didn't receive the init message
		sc.GotRequestRecover(sc.validatedInitHashes, deser, sc.InitHeaders, senderChan, sc.ConsItems)
		return false, false
	default: // This will be a bin cons message, so process it in the binary consensus
		_, forward := sc.binCons.ProcessMessage(deser, isLocal, senderChan)
		sc.checkEchoState(t, nmt, sc.MainChannel)
		// TODO don't always return true?
		return true, forward
	}
}

// checkEchoState checks if we have received enough echo messages to perform an action.
func (sc *MvCons1) checkEchoState(t, nmt int, mainChannel channelinterface.MainChannel) {
	msgState := sc.ConsItems.MsgState.(*MessageState)

	if dec, _ := sc.binCons.GetBinDecided(); dec == 0 {
		// if we decided 0 then we can return
		return
	}
	msgState.mutex.Lock()
	defer msgState.mutex.Unlock()

	// we need at least n-t echo message from different nodes, or n-t bin cons message from round 1 to take an action
	if msgState.Sms.GetSigCountMsgID(messagetypes.MvMsgID{HdrID: messages.HdrMvEcho}) >= nmt || sc.checkEnoughBin(nmt, mainChannel) {
		// we got n-t echo messages so set the echo timeout to passed
		// sc.echoTimeOutState = cons.TimeoutPassed

		// go through each of the different recieved echo message
		for _, msgCount := range msgState.Sms.GetSigCountMsgIDList(messagetypes.MvMsgID{HdrID: messages.HdrMvEcho}) {
			// check if we got n-t of the same echo hash that is not the zero hash (the zero hash means no init was received)
			echohdr := msgCount.MsgHeader.GetBaseMsgHeader().(*messagetypes.MvEchoMessage)
			if msgCount.Count >= nmt && !bytes.Equal(echohdr.ProposalHash, sc.zeroHash) {
				// Now know this hash is the only non-nil value that could be decided, so we support it for a decision
				// Let the bin cons know 1 is valid
				msgState.BinConsMessageStateInterface.SetMv1Valid(sc.ConsItems.MC)

				// we got n-t echo messages so set the echo timeout to passed
				sc.echoTimeOutState = cons.TimeoutPassed

				proposalHash := types.HashStr(echohdr.ProposalHash)
				msgState.setEchoHash(echohdr.ProposalHash) // let the msg state know we have an echo hash to support

				if !msgState.BinConsMessageStateInterface.SentProposal(0, true, sc.ConsItems.MC) {
					auxMsg := sc.binCons.GetMVInitialRoundBroadcast(1)
					if sc.CheckMemberLocalMsg(auxMsg) { // Check if we are a member for this type of message
						// if we didn't already support 0 for binary consesnsus for round 1, then we can support 1
						// (the first argument is 0 here because we the structure keeps track of having sent proposal for the round+1)

						// Broadcast our bin msg for round 1
						logging.Infof("Got n-t messages for hash %v, will support 1 in bincons for round 1", proposalHash)
						var prfMsgs []*sig.MultipleSignedMessage
						// Check what type of broadcast we should do
						includeProofs, nxtCoordPub := cons.CheckIncludeEchoProofs(0, sc.ConsItems,
							sc.includeProofs, sc.GeneralConfig)
						if includeProofs {
							sigCount, _, _, err := sc.GetBufferCount(echohdr, sc.GeneralConfig, sc.ConsItems.MC)
							if err != nil {
								panic(err)
							}
							prfMsgs, err = msgState.generateMvProofs(sigCount, sc.ConsItems.MC.MC.GetMyPriv().GetPub(), sc.ConsItems.MC)
							if err != nil {
								panic(fmt.Sprint(sigCount, err))
							}
						}
						prfs := make([]messages.MsgHeader, len(prfMsgs))
						for i, nxt := range prfMsgs {
							prfs[i] = nxt
						}
						sc.BroadcastFunc(nxtCoordPub, sc.ConsItems, auxMsg, true,
							sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, prfs...)
						// cons.BroadcastBin(nxtCoordPub, sc.ByzType, sc.binCons, auxMsg, mainChannel, prfs...)
					}
				}
				if sc.decisionInitMsg != nil {
					// sanity check
					if sc.decisionHash != proposalHash {
						panic("Got different decisions")
					}
				} else {
					// sanity check
					if len(proposalHash) != types.GetHashLen() {
						panic(fmt.Sprintf("Got invalid hash len %v, expected %v", len(proposalHash), types.GetHashLen()))
					}

					sc.decisionHash = proposalHash
					// check if we have received the init message for this hash
					if v := sc.validatedInitHashes[sc.decisionHash]; v != nil {
						// v := msgState.initMessageByHash[sc.decisionHash]
						sc.decisionInitMsg = v.Header.(*sig.MultipleSignedMessage).GetBaseMsgHeader().(*messagetypes.MvInitMessage)
						sc.proposerPub = v.Header.(*sig.MultipleSignedMessage).SigItems[0].Pub
						logging.Infof("Got mv delivery at index %v", sc.Index)
						sc.stopTimers()
					} else {
						sc.StartRecoverTimeout(sc.Index, mainChannel)
					}
				}
			}
		}
		// Start echo timeout since we have n-t messages
		sc.sendEchoTimeout(mainChannel)
		if sc.checkEchoTimeoutPassed() {
			// check if we have not supported any value for round 1
			// (the first argument is 0 here because we the structure keeps track of having sent proposal for the round+1)
			if !msgState.BinConsMessageStateInterface.SentProposal(0, true, sc.ConsItems.MC) {
				auxMsg := sc.binCons.GetMVInitialRoundBroadcast(0)
				if sc.CheckMemberLocalMsg(auxMsg) { // check if we are a member for this type of message
					// We support 0 since we have not gotten a proposal
					logging.Infof("Supporting 0 for index %v, since not enough echos received", sc.Index)
					sc.BroadcastFunc(nil, sc.ConsItems, auxMsg, true,
						sc.ConsItems.FwdChecker.GetNewForwardListFunc(), mainChannel, sc.GeneralConfig, nil)
					// cons.BroadcastBin(nil, sc.ByzType, sc.binCons, auxMsg, mainChannel)
				}
			}
			// Let the bincons know that 0 is also valid after the timeout
			msgState.BinConsMessageStateInterface.SetMv0Valid()
		}
		// Check if bincons has made progress
		if dec, _ := sc.binCons.GetBinDecided(); dec == -1 {

			var round types.ConsensusRound = 1
			for sc.binCons.CheckRound(nmt, t, round, mainChannel) {
				round++
			}
		}
		if dec, _ := sc.binCons.GetBinDecided(); dec > -1 {
			sc.stopTimers()
		}
	}
}

// checkEnoughBin checks if we have received nmt aux proof messages for round 1 from different processes
func (sc *MvCons1) checkEnoughBin(nmt int, _ channelinterface.MainChannel) bool {
	// in case enough bin round 1 messages were sent, we have to start the echo timeout
	return sc.ConsItems.MsgState.(*MessageState).BinConsMessageStateInterface.GetValidMessageCount(
		1, sc.ConsItems.MC) >= nmt
}

// checkEchoTimeoutPassed returns true if the echo timeout has already expired, or bincons is past round 1
func (sc *MvCons1) checkEchoTimeoutPassed() bool {
	if sc.echoTimeOutState == cons.TimeoutPassed || sc.binCons.CanSkipMvTimeout() {
		// init timeout can be passed since it is before echo
		sc.initTimeoutState = cons.TimeoutPassed
		return true
	}
	return false
}

// checkInitTimeoutPassed returns true if the init timeout has already expired, or bincons is past round 1
func (sc *MvCons1) checkInitTimeoutPassed() bool {
	sc.checkEchoTimeoutPassed()
	if sc.initTimeoutState == cons.TimeoutPassed {
		return true
	}
	return false
}

// sendEchoTimeout starts the echo timeout
func (sc *MvCons1) sendEchoTimeout(mainChannel channelinterface.MainChannel) {
	if sc.echoTimeOutState == cons.TimeoutNotSent && !sc.HasDecided() && !sc.binCons.CanSkipMvTimeout() {
		sc.echoTimeOutState = cons.TimeoutSent
		sc.echoTimer = cons.StartEchoTimer(0, sc.ConsItems, mainChannel)
	}
}

// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
// The messages are:
// (1) HdrMvInit returns 0, 0 if generalconfig.MvBroadcastInitForBufferForwarder is true (meaning don't forward the message)
//     otherwise returns 1, 1 (meaning forward the message right away)
// (2) HdrMvEcho returns n-t, n for the thresholds.
// (3) otherwise bincons.GetBufferCount is checked
func (sc *MvCons1) GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig, memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {
	switch hdr.GetID() {
	case messages.HdrMvInit:
		if config.MvBroadcastInitForBufferForwarder { // This is an all to all broadcast
			return 1, 1, nil, types.ErrDontForwardMessage
		}
		return 1, 1, hdr.GetMsgID(), nil // otherwise we propagate it through gossip
	case messages.HdrMvEcho:
		memCount := memberChecker.MC.GetMemberCount()
		return memCount - memberChecker.MC.GetFaultCount(), memCount, hdr.GetMsgID(), nil
	case messages.HdrPartialMsg:
		if sc.PartialMessageType == types.NoPartialMessages {
			panic("should not have partials")
		}
		return 1, 1, hdr.GetMsgID(), nil
	}
	return sc.binCons.GetBufferCount(hdr, gc, memberChecker)
}

// GetHeader return blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
// The valid headers are HdrMvInit, HdrMvEcho, HdrMvRequestRecover, or any from the binary consensus.
func (*MvCons1) GetHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch headerType {
	case messages.HdrMvInit:
		var internalMsg messages.InternalSignedMsgHeader
		internalMsg = messagetypes.NewMvInitMessage()
		if gc.PartialMessageType != types.NoPartialMessages {
			// if partials are used then MvInit must always construct into combined messages
			// TODO add test where a combined message is sent with a different internal header type
			internalMsg = messagetypes.NewCombinedMessage(internalMsg)
		}
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, internalMsg), nil
	case messages.HdrMvEcho:
		return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewMvEchoMessage()), nil
	case messages.HdrMvRequestRecover:
		return messagetypes.NewMvRequestRecoverMessage(), nil
	case messages.HdrPartialMsg:
		if gc.PartialMessageType != types.NoPartialMessages {
			return sig.NewMultipleSignedMsg(types.ConsensusIndex{}, emptyPub, messagetypes.NewPartialMessage()), nil
		}
	}
	switch gc.ConsType {
	case types.MvBinCons1Type:
		return (&bincons1.BinCons1{}).GetHeader(emptyPub, gc, headerType)
	case types.MvBinConsRnd1Type:
		return (&binconsrnd1.BinConsRnd1{}).GetHeader(emptyPub, gc, headerType)
	default:
		panic(gc.ConsType)
	}

}

func (sc *MvCons1) ShouldCreatePartial(headerType messages.HeaderID) bool {
	if sc.PartialMessageType != types.NoPartialMessages && headerType == messages.HdrMvInit {
		return true
	}
	return false
}

// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled)
func (sc *MvCons1) CanStartNext() bool {
	return true
}

// GetNextInfo will be called after CanStartNext returns true.
// It returns sc.Index - 1, nil.
// If false is returned then the next is started, but the current instance has no state machine created.
func (sc *MvCons1) GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasNextInfo bool) {
	return types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1), nil,
		nil, sc.GeneralConfig.AllowConcurrent > 0
}

// HasDecided should return true if this consensus item has reached a decision.
func (sc *MvCons1) HasDecided() bool {
	switch dec, _ := sc.binCons.GetBinDecided(); dec {
	case -1:
		return false
	case 0:
		return true
	case 1:
		if sc.decisionInitMsg != nil {
			return true
		}
		// bincons has decided, but we haven't yet received the proposal
		return false
	default:
		panic("invalid decided")
	}
}

// GetDecision returns the decided value as a byte slice,
// or nil if a nil value was decided
func (sc *MvCons1) GetDecision() (sig.Pub, []byte, types.ConsensusIndex) {
	switch dec, _ := sc.binCons.GetBinDecided(); dec {
	case 0:
		// we decided nil
		return nil, nil, types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
	case 1:
		if sc.decisionInitMsg != nil {
			return sc.proposerPub, sc.decisionInitMsg.Proposal,
				types.SingleComputeConsensusIDShort(sc.Index.Index.(types.ConsensusInt) - 1)
		}
	}
	panic("should have decided")
}

// GetConsType returns the type of consensus this instance implements.
func (sc *MvCons1) GetConsType() types.ConsType {
	return types.MvBinCons1Type
}

// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
func (sc *MvCons1) SetNextConsItem(consinterface.ConsItem) {
}

// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
func (sc *MvCons1) PrevHasBeenReset() {
}

// stopTimers is used to stop running timers
func (sc *MvCons1) stopTimers() {
	if sc.initTimer != nil {
		sc.initTimer.Stop()
		sc.initTimer = nil
	}
	if sc.echoTimer != nil {
		sc.echoTimer.Stop()
		sc.echoTimer = nil
	}
	sc.StopRecoverTimeout()
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// broadcastInit broadcasts an int message
func (sc *MvCons1) broadcastInit(newMsg *messagetypes.MvInitMessage, mainChannel channelinterface.MainChannel) {
	sc.ConsItems.MC.MC.GetStats().BroadcastProposal()
	var forwardFunc channelinterface.NewForwardFuncFilter
	if config.MvBroadcastInitForBufferForwarder { // we change who we broadcast to depending on the configuration
		forwardFunc = channelinterface.ForwardAllPub // we broadcast the init message to all nodes directly
	} else {
		forwardFunc = sc.ConsItems.FwdChecker.GetNewForwardListFunc() // we propoagte the init message using gossip
	}
	// sc.CommitProof is the proof from the previous consensus instance (if to.CollectBroadcast is true)
	sc.BroadcastFunc(nil, sc.ConsItems, newMsg, true,
		forwardFunc, mainChannel, sc.GeneralConfig, sc.CommitProof...)
	// BroadcastMv(nil, sc.ByzType, sc, forwardFunc, newMsg, sc.CommitProof, mainChannel)
}

// broadcastEcho broadcasts an echo message
func (sc *MvCons1) broadcastEcho(proposalHash []byte, mainChannel channelinterface.MainChannel) {

	sc.sentEcho = true
	newMsg := messagetypes.NewMvEchoMessage()
	newMsg.ProposalHash = proposalHash
	if sc.CheckMemberLocalMsg(newMsg) { // Check if we are a member for this message type
		logging.Info("bcast echo", sc.GeneralConfig.TestIndex, sc.Index, sc.ConsItems.MC.MC.GetMemberCount())

		// cordPub := cons.GetCoordPubCollectBroadcastEcho(0, sc.ConsItems, sc.GeneralConfig)
		var cordPub sig.Pub // TODO collect broadcast
		sc.BroadcastFunc(cordPub, sc.ConsItems, newMsg, true, sc.ConsItems.FwdChecker.GetNewForwardListFunc(),
			mainChannel, sc.GeneralConfig, nil)
		// BroadcastMv(cordPub, sc.ByzType, sc, sc.ConsItems.FwdChecker.GetNewForwardListFunc(), newMsg, nil, mainChannel)
	} else {
		logging.Info("dont bcast echo", sc.GeneralConfig.TestIndex, sc.Index, sc.ConsItems.MC.MC.GetMemberCount())
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//
//////////////////////////////////////////////////////////////////////////////////////////

// GenerateMessageState generates a new message state object given the inputs.
func (*MvCons1) GenerateMessageState(
	gc *generalconfig.GeneralConfig) consinterface.MessageState {

	return NewMvCons1MessageState(gc)
}

// Collect is called when the item is being garbage collected.
func (sc *MvCons1) Collect() {
	sc.AbsConsItem.Collect()
	sc.binCons.Collect()
	sc.stopTimers()
	sc.StopRecoverTimeout()
}
