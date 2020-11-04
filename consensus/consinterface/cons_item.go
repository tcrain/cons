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
The consinterface package defines the interfaces for implementations of the consensus objects.
It also contains the core functionalities that will be used by most consensus object, including
message tracking, membership checking, and message forwarding.
*/
package consinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/deserialized"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
)

type BasicConsItem interface {
	// GetHeader should a blank message header for the HeaderID, this object will be used to deserialize a message into itself (see consinterface.DeserializeMessage).
	GetHeader(emptyPub sig.Pub, generalConfig *generalconfig.GeneralConfig, headerID messages.HeaderID) (messages.MsgHeader, error)
	// GetBufferCount checks a MessageID and returns the thresholds for which it should be forwarded using the BufferForwarder (see forwardchecker.ForwardChecker interface).
	// endThreshold is the number of signatures needed to fully support this message (for example a leader message would be 1, and an echo message would be n-t),
	// maxPossible is the maximum different possible signatures valid for this message (for example a leader message would be 1, and an echo message would be n),
	// msgid is the MsgID of the message.
	// Forward thresholds are computed by nextThreshold*Forwarder.GetFanOut() (i.e. it grows exponentionally), until the message is forwarded after maxPossible has been reached.
	// If endThreshold < 1, then the message is not forwarded (i.e. we expect the original sender to handling the sending to all nodes).
	GetBufferCount(messages.MsgIDHeader, *generalconfig.GeneralConfig, *MemCheckers) (endThreshold int, maxPossible int, msgid messages.MsgID, err error) // How many of the same msg to buff before forwarding
}

// ConsItem represents an implementation of a consensus algorithm.
// Certain methods must be implemnted as static methods.
// Each iteration of consensus will use a single instance of this object.
// When the system is done with a consensus iteration, it will call ResetState(index) on the object, the object
// should the reinitialize its as the object will be reused for a new consensus iteration.
// There are generalconfig.KeepTotal instaces of these objects created throughout the life of the application, and they are just
// reused by calling ResetState(index).
type ConsItem interface {
	BasicConsItem
	GetGeneralConfig() *generalconfig.GeneralConfig
	// ProcessMessage is called on every message once it has been checked that it is a valid message (using the static method ConsItem.DerserializeMessage), that it comes from a member
	// of the consensus and that it is not a duplicate message (using the MemberChecker and MessageState objects). This function should process the message and update the
	// state of the consensus.
	// It should return true in first position if made progress towards decision, or false if already decided, and return true in second position if the message should be forwarded.
	ProcessMessage(item *deserialized.DeserializedItem, isLocal bool,
		senderChan *channelinterface.SendRecvChannel) (progress, shouldForward bool)
	// HasDecided should return true if this consensus item has reached a decision.
	HasDecided() bool
	// GetBinState should return the entire state of the consensus as a string of bytes using the MessageState object (normally this would just be MessageState.GetMsgState() as the list
	// of all messages, with a messagetypes.ConsBinStateMessage header appended to the beginning), this value will be stored to disk and used to send the state to other nodes.
	// If localOnly is true then only messages signed by the local node should be included.
	GetBinState(localOnly bool) ([]byte, error)
	// PrintState()
	// GetDecision should return the decided value of the consensus. It should only be called after HasDecided returns true.
	// Proposer it the node that proposed the decision, prvIdx is the index of the decision that preceeded this decision
	// (normally this is the current index - 1), futureFixed is the first larger index that this decision does not depend
	// on (normally this is the current index + 1). prvIdx and futureFixed are different for MvCons3 as that
	// consensus piggybacks consensus instances on top of eachother.
	GetDecision() (proposer sig.Pub, decision []byte, prvIdx, futureFixed types.ConsensusIndex)
	// GetPreHeader returns the serialized header that is attached to all messages sent by this consensus item.
	// It is normally nil.
	GetPreHeader() []messages.MsgHeader
	// AddPreHeader appends a header that will be attached to all messages sent by this consensus item.
	AddPreHeader(messages.MsgHeader)
	// Start is called exactly once for each consensus instance, after start is called, the consensus may return
	// true from GetProposalInfo as soon as it is ready for a proposal from the application.
	// Note that messages may be sent to the consensus before Start is called, they should be processed as necessary.
	// Finished last index is true once the last index has decided (some consensus instances may start after the last index
	// has decided to ensure all items eventually decide)
	Start(finishedLastIndex bool)
	// HasStarted returns true if start has been called
	HasStarted() bool
	// GetProposalIndex returns the previous consensus index from which this index will use (must be at least 1
	// smaller than the current index). The value is used only if ready is true. Ready should be false until
	// this consensus index needs a proposal (this cannot return true before start is called).
	GetProposalIndex() (prevIdx types.ConsensusIndex, ready bool)
	// GotProposal is called at most once for each consensus index after start by the application as the local input to the consensus. The hdr value will be used as the input to the consensus.
	GotProposal(hdr messages.MsgHeader, mainChannel channelinterface.MainChannel) error
	// InitState is called once before running the first consensus iteration. The input values are expected to remain the same across all consensus iterations.
	// GenerateInitState(preHeaders []messages.MsgHeader, priv sig.Priv, eis ExtraInitState, stats stats.StatsInterface)
	// ResetState should reset any stored state for the current consensus index, and prepare for the new consensus index given by the input.
	GenerateNewItem(index types.ConsensusIndex, consItems *ConsInterfaceItems, mainChannel channelinterface.MainChannel,
		prevItem ConsItem, broadcastFunc ByzBroadcastFunc, gc *generalconfig.GeneralConfig) ConsItem
	// SetNextConsItem gives a pointer to the next consensus item at the next consensus instance, it is called when the next instance is created
	SetNextConsItem(next ConsItem)
	// CanStartNext should return true if it is safe to start the next consensus instance (if parallel instances are enabled),
	// in most consensus algorithms this will always return true. In MvCons3 this will return true once this consensus instance
	// know what instance it follows (i.e. the accepted init message points to), or when a timer has run out.
	CanStartNext() bool
	// GetNextInfo will be called after CanStartNext returns true.
	// It is used to get information about how the state machine for this instance will be generated.
	// prevIdx should be the index that this consensus index will follow (normally this is just idx - 1).
	// preDecision is either nil or the value that will be decided if a non-nil value is decided.
	// hasInfo returns true if the values for proposer and preDecision are ready.
	// If false is returned then the next is started, but the current instance has no state machine created.
	// This function is mainly used for MvCons3 since the order of state machines depends on depends on the execution
	// of the consensus instances.
	// For other consensus algorithms prevIdx should return (idx - 1) and hasInfo should be false unless
	// AllowConcurrent is enabled.
	GetNextInfo() (prevIdx types.ConsensusIndex, proposer sig.Pub, preDecision []byte, hasInfo bool)
	// SetInitialState should be called on the initial consensus index (1) to set the initial state.
	SetInitialState([]byte)
	GetIndex() types.ConsensusIndex // GetIndex returns the consensus index of the item.
	// HasValidStarted returns true if this cons item has processed a valid proposal, or if it know other nodes
	// have (i.e. if the binary reduction has terminated)
	HasValidStarted() bool
	// GetCustomRecoverMsg is called when there is no progress after a timeout. If the standard
	// recovery message is used (messagetypes.NoProgressMessage) then the recovery is handled by the
	// consensus state objects.
	// Otherwise the returned message is sent to the peers and the consensus must handle
	// the recovery itself.
	// If createEmpty is true then the message will be used to deserialize a received message.
	GetCustomRecoverMsg(createEmpty bool) messages.MsgHeader
	// GetRecoverMsgType returns the HeaderID of the recovery messages used by this consensus.
	// If messages.HdrNoProgress is returned then the default recover is used
	GetRecoverMsgType() messages.HeaderID
	// ProcessCustomRecoveryMessage is called when a valid custom recover message equal of the type
	// returned by GetCustomRecoverMessage (that is not messagetypes.NoProgressMessage).
	// The consensus should then perform the appropriate recovery.
	// Note that the message is not checked with member checkers/signatures, etc.
	ProcessCustomRecoveryMessage(item *deserialized.DeserializedItem,
		senderChan *channelinterface.SendRecvChannel)
	// GetCommitProof returns a signed message header that counts at the commit message for this consensus.
	GetCommitProof() []messages.MsgHeader
	// SetCommitProof takes the value returned from GetCommitProof of the previous consensus instance once it has decided.
	// The consensus can then use this as needed.
	SetCommitProof(prf []messages.MsgHeader)
	// GetPrevCommitProof returns a signed message header that counts at the commit message for the previous consensus.
	// This should only be called after DoneKeep has been called on this instance.
	// cordPub is the expected public key of the coordinator of the current round (used for collect broadcast)
	GetPrevCommitProof() (cordPub sig.Pub, proof []messages.MsgHeader)
	// Broadcast a value.
	// If nextCoordPub is nil the message will only be sent to that node, otherwise it will be sent
	// as normal (nextCoordPub is used when CollectBroadcast is true in test options).
	Broadcast(nextCoordPub sig.Pub, msg messages.InternalSignedMsgHeader,
		signMessage bool,
		forwardFunc channelinterface.NewForwardFuncFilter,
		mainChannel channelinterface.MainChannel,
		additionalMsgs ...messages.MsgHeader)
	// CheckMemberLocalMsg checks if the local node is a member of the consensus for this message type
	CheckMemberLocalMsg(hdr messages.InternalSignedMsgHeader) bool
	// CheckMemberLocal checks if the node is a member of the consensus.
	CheckMemberLocal() bool
	// GetConsInterfaceItems returns the ConsInterfaceItems for this consesnsus instance.
	GetConsInterfaceItems() *ConsInterfaceItems

	// PrevHasBeenReset is called when the previous consensus index has been reset to a new index
	PrevHasBeenReset()
	// SetTestConfig is run before tests to set any options specific to the consensus.
	// Test id should be the index of the node in the test.
	// SetTestConfig(testId int, options types.TestOptions)
	// GetConsType returns the type of consensus this instance implements.
	GetConsType() types.ConsType
	// Collect is called when the item is about to be garbage collected.
	Collect()
	// ForwardOldIndices should return true if messages from old consensus indices should be forwarded
	// even after decision.
	ForwardOldIndices() bool

	///////////////////////////////////////////////////////////////////////////
	// Static methods
	///////////////////////////////////////////////////////////////////////////

	// NeedsConcurrent returns the number of concurrent instances the consensus needs to run correctly (this is usually 1).
	NeedsConcurrent() types.ConsensusInt
	// ComputeDecidedValue is a static method that should return the decided value, state comes from ConsItem.GetBinState, decision comes from ConsItem.GetDecision.
	ComputeDecidedValue(state []byte, decision []byte) []byte
	// GetProposeHeaderID returns the HeaderID for the message type that will be input to GotProposal.
	GetProposeHeaderID() messages.HeaderID
	// GenerateMessageState generates a new message state object given the inputs.
	GenerateMessageState(gc *generalconfig.GeneralConfig) MessageState
	// ShouldCreatePartial returns true if the message type should be sent as a partial message
	ShouldCreatePartial(headerType messages.HeaderID) bool
}

// SetTestConfig is run before tests to set any options specific to the consensus.
// It should be run before InitAbsState
func CreateGeneralConfig(testId int, eis generalconfig.ExtraInitState,
	statsInt stats.StatsInterface, to types.TestOptions, initHeaders []messages.MsgHeader,
	priv sig.Priv) *generalconfig.GeneralConfig {

	gc := &generalconfig.GeneralConfig{}
	gc.NoSignatures = to.NoSignatures
	gc.EncryptChannels = to.EncryptChannels
	gc.UseFixedSeed = to.UseFixedSeed
	gc.ByzStartIndex = to.ByzStartIndex
	gc.StopOnCommit = to.StopOnCommit
	gc.IncludeProofs = to.IncludeProofs
	gc.CollectBroadcast = to.CollectBroadcast
	gc.UseMultiSig = to.UseMultisig
	gc.AllowSupportCoin = to.AllowSupportCoin
	gc.Stats = statsInt
	gc.Eis = eis
	gc.NetworkType = to.NetworkType
	gc.TestIndex = testId
	if testId < to.NumByz {
		gc.IsByz = true
	}
	gc.ConsType = to.ConsType
	gc.PartialMessageType = to.PartialMessageType
	gc.SetTestConfig = true
	gc.InitHeaders = initHeaders
	gc.Priv = priv
	gc.Ordering = to.OrderingType
	gc.CPUProfile = to.CPUProfile
	gc.MemProfile = to.MemProfile
	gc.TestID = to.TestID
	gc.UseFullBinaryState = to.UseFullBinaryState
	gc.IncludeCurrentSigs = to.IncludeCurrentSigs
	gc.CoinType = to.CoinType
	gc.UseTp1CoinThresh = types.UseTp1CoinThresh(to)
	gc.UseFixedCoinPresets = to.UseFixedCoinPresets
	gc.MemCheckerBitIDType = to.MemCheckerBitIDType
	gc.SigBitIDType = to.SigBitIDType
	gc.WarmUpInstances = to.WarmUpInstances
	gc.KeepPast = to.KeepPast
	gc.ForwardTimeout = to.ForwardTimeout
	gc.ProgressTimeout = to.ProgressTimeout
	gc.MvConsTimeout = to.MvConsTimeout
	gc.MvConsVRFTimeout = to.MvConsVRFTimeout
	gc.MvConsRequestRecoverTimeout = to.MvConsRequestRecoverTimeout
	gc.NodeChoiceVRFRelaxation = to.NodeChoiceVRFRelaxation
	gc.CoordChoiceVRF = to.CoordChoiceVRF
	gc.RandForwardTimeout = to.RandForwardTimeout
	gc.UseRandCoord = to.UseRandCoord
	gc.BufferForwardType = to.BufferForwardType
	gc.MvCons4BcastType = to.MvCons4BcastType

	return gc
}

///////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////

// ProcessConsMessageList takes a list of one or more serialzed consensus messages, then breaks them into individual ones
// and processes them. Only the staic methods of ConsItem should be used.
// This is a static method, and can be called concurrently.
func ProcessConsMessageList(isLocal types.LocalMessageType, unmarFunc types.ConsensusIndexFuncs,
	ci ConsItem, msg sig.EncodedMsg,
	memberCheckerState ConsStateInterface) ([]*deserialized.DeserializedItem, []error) {

	var errors []error
	var items []*deserialized.DeserializedItem
	// Break the message into seperate ones
	for true {
		ht, err := msg.PeekHeaderType()
		if err != nil {
			break
		}
		hdrType, err := ci.GetHeader(memberCheckerState.GetNewPub(), memberCheckerState.GetGeneralConfig(), ht)
		if err != nil {
			logging.Infof("Got unexpected message type: %v, expected %v, err: %v", ht, hdrType, err)
			errors = append(errors, err)
			break
		}
		// w := hdrType.CreateCopy()
		nxt, err := hdrType.GetBytes(msg.Message)
		if err != nil {
			logging.Error(err)
			errors = append(errors, err)
			break
		}
		if len(nxt) == 0 {
			err = types.ErrNilMsg
			errors = append(errors, err)
			break
		}
		nxtMsg := msg.NewEncodedMsgCopy(nxt)

		deser, err := FinishUnwrapMessage(isLocal, hdrType, nxtMsg,
			unmarFunc, ci, memberCheckerState)
		if err != nil {
			// if err != types.ErrAlreadyReceivedMessage {
			//	panic(1)
			// }
			logging.Info(err)
			errors = append(errors, err)
			continue
		}
		items = append(items, deser...)

	}
	if len(items) == 0 {
		return nil, errors
	}
	return items, errors
}

// UnwrapMessage is called each time a message is received from the network or from a local send.
// It can be called by many different threads concurrently, it is responsible for de-serializing a message, checking if it is valid,
// checking if it from a valid consensus member and is a new message. If all those requirements are satisfied, the message is de-serialized
// and sent to be processed by the main consensus loop.
// Only the static methods of ConsItem should be used.
// This is a static method, and can be called concurrently.
func UnwrapMessage(msg sig.EncodedMsg,
	unmarFunc types.ConsensusIndexFuncs, isLocal types.LocalMessageType,
	consItem ConsItem, memberCheckerState ConsStateInterface) ([]*deserialized.DeserializedItem, []error) {

	ht, err := msg.PeekHeaderType()
	if err != nil {
		logging.Info("Got a bad message", err)
		return nil, []error{err}
	}
	var ret []*deserialized.DeserializedItem
	switch ht { // Check for general headers not made by the consensus, but by the outer layer
	case messages.HdrNoProgress:
		// A no progress message means that a node has not made any progress after a timeout.
		// If we have any more advanced state then we should send this information to that process.
		for len(msg.GetBytes()) > 0 {
			w := messagetypes.NewNoProgressMessage(types.ConsensusIndex{}, false, 0)
			_, err := w.Deserialize(msg.Message, unmarFunc)
			if err != nil {
				return ret, []error{err}
			}
			msg.Message.GetBytes()
			ret = append(ret, &deserialized.DeserializedItem{
				Index:          w.GetIndex(),
				HeaderType:     ht,
				Header:         w,
				IsDeserialized: true})
		}
		return ret, nil
	case consItem.GetRecoverMsgType():
		for len(msg.GetBytes()) > 0 {
			w := consItem.GetCustomRecoverMsg(true)
			idx, err := w.PeekHeaders(msg.Message, unmarFunc)
			if err != nil {
				return ret, []error{err}
			}
			_, err = w.Deserialize(msg.Message, unmarFunc)
			if err != nil {
				return ret, []error{err}
			}
			msg.Message.GetBytes()
			ret = append(ret, &deserialized.DeserializedItem{
				Index:          idx,
				HeaderType:     ht,
				Header:         w,
				IsDeserialized: true})
		}
		return ret, nil
	}
	// Check if it is a binstate msg
	if ht == messages.HdrBinState {
		// ConsBinStateMessages are just a list of messages, so remove the header and process it.
		// These messages are used during recovery and contain the current state of a node for all messages (see messagetypes.ConsBinStateMessage).
		// Remove the header
		_, err := messagetypes.ConsBinStateMessage{}.Deserialize(msg.Message, unmarFunc)
		if err != nil {
			return nil, []error{err}
		}
	}
	// Now it is just a list of one or more consensus messages
	return ProcessConsMessageList(isLocal, unmarFunc, consItem, msg, memberCheckerState)
}

// FinishUnwrapMessage is called after UnwrapMessage and ProcessConsMessageList.
// If the consensus is ready then it does the actual deserialization and returns the list of de-serialized messages.
// If the consensus is not ready to deserialize the message (for example if it does not know who the members of the consensus are
// then the IsDeserialzed flag of the return value is set to false.
// This is a static method, and can be called concurrently.
func FinishUnwrapMessage(isLocal types.LocalMessageType, w messages.MsgHeader, msg sig.EncodedMsg, unmarFunc types.ConsensusIndexFuncs,
	consItem ConsItem, memberCheckerState ConsStateInterface) (
	[]*deserialized.DeserializedItem, error) {

	ht, err := msg.PeekHeaderType()
	if err != nil {
		logging.Info("Got a bad message", err)
		return nil, err
	}
	if ht != w.GetID() {
		panic("sanity fail")
	}
	idx, err := w.PeekHeaders(msg.Message, unmarFunc)
	if err != nil {
		return nil, err
	}

	// First cons starts at 1
	if err := idx.Index.Validate(isLocal); err != nil {
		return nil, err
	}
	// Check if you can generate the member checker
	idxItem, err := memberCheckerState.CheckGenMemberChecker(idx)
	if err != nil {
		if err == types.ErrParentNotFound {
			// we will try to process it later
			logging.Infof("index not found for message, will process later %v", idx.Index)
			return []*deserialized.DeserializedItem{
				{
					IsLocal:        isLocal,
					Index:          idx,
					HeaderType:     ht,
					Header:         w,
					IsDeserialized: false,
					Message:        msg}}, nil
		}
		logging.Info("error getting member checker for index %v, err %v", idx.Index, err)
		return nil, err // TODO should keep message in certain cases?
	}

	if idxItem.MC.MC.IsReady() {
		// deserialize the actual message
		// return ConsItem.DeserializeMessage(idx, msg, MC, ms)
		return DeserializeMessage(isLocal, idx, consItem, w, msg, unmarFunc, idxItem.MC, idxItem.MsgState,
			memberCheckerState.GetGeneralConfig())
	}
	// not ready so we try later
	logging.Infof("member checker for index %v not ready, will try later", idx.Index)
	return []*deserialized.DeserializedItem{
		{
			IsLocal:        isLocal,
			Index:          idx,
			HeaderType:     ht,
			Header:         w,
			IsDeserialized: false,
			Message:        msg}}, nil
}

// DeserializeMessage deserializes a message for the consensus type.
// This is a static method, and can be called concurrently.
func DeserializeMessage(isLocal types.LocalMessageType, idx types.ConsensusIndex, ci ConsItem, w messages.MsgHeader, msg sig.EncodedMsg,
	unmarFunc types.ConsensusIndexFuncs, mc *MemCheckers,
	ms MessageState, gc *generalconfig.GeneralConfig) ([]*deserialized.DeserializedItem, error) {

	encMsg := msg.ShallowCopy()
	var err error
	switch v := w.(type) {
	case *sig.MultipleSignedMessage:
		_, err = v.DeserializeEncoded(msg, unmarFunc)
	default:
		_, err = w.Deserialize(msg.Message, unmarFunc)
	}
	if err != nil {
		logging.Infof("deserialization error %v at index %v", err, idx.Index)
		return nil, err
	}
	logging.Info("Got a message of type:", w.GetID(), idx.Index)

	di := &deserialized.DeserializedItem{
		IsLocal:        isLocal,
		Index:          idx,
		HeaderType:     w.GetID(),
		Header:         w,
		IsDeserialized: true,
		MC:             mc,
		Message:        encMsg}
	return internalCheckDeserializedMessage(di, ci, mc, ms, gc)
}

// CheckLocalDeserializedMessage can be called after successful derserialization, it checks with the member checker and messages state
// if the message is signed by a member and is not a duplicate.
func CheckLocalDeserializedMessage(di *deserialized.DeserializedItem, ci ConsItem, memberCheckerState ConsStateInterface) ([]*deserialized.DeserializedItem, error) {
	if di.IsLocal != types.LocalMessage {
		panic("should only be called with local messages")
	}
	idxItem, err := memberCheckerState.GetMemberChecker(di.Index)
	switch err {
	// case types.ErrIndexTooOld:
	// We have already finished and gc'd this cons instance, so drop the message
	//	return nil, err
	case nil: //&&
		switch di.Header.(type) {
		case *sig.MultipleSignedMessage, *sig.UnsignedMessage:
			if idxItem.MC.MC.IsReady() {
				return internalCheckDeserializedMessage(di, ci, idxItem.MC, idxItem.MsgState, memberCheckerState.GetGeneralConfig())
			}
			return nil, types.ErrMemCheckerNotReady
		default:
			// if it is not a signed message we dont keep state about it and just process it directly
			return []*deserialized.DeserializedItem{di}, nil
		}
	}
	return nil, err
}
func internalCheckDeserializedMessage(di *deserialized.DeserializedItem, ci ConsItem, mc *MemCheckers,
	ms MessageState, gc *generalconfig.GeneralConfig) ([]*deserialized.DeserializedItem, error) {

	// Check if you received this message already
	ret, err := ms.GotMsg(ci.GetHeader, di, gc, mc)
	if err != nil {
		logging.Info("Error with deserialized message: ", err, di.HeaderType, di.Index)
		return nil, err
	}
	return ret, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////////////

type ByzBroadcastFunc func(nextCoordPub sig.Pub,
	ci *ConsInterfaceItems,
	msg messages.InternalSignedMsgHeader,
	signMessage bool,
	filter channelinterface.NewForwardFuncFilter,
	mainChannel channelinterface.MainChannel,
	gc *generalconfig.GeneralConfig,
	additionalMsgs ...messages.MsgHeader)
