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

package messagestate

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/messagetypes"
)

type SimpleMessageState struct {
	sync.RWMutex
	msgMap sigMsgMapInterface
	msgs   [][]byte // list of all messages, maybe remove this in the future since we do this in a more compressed way if generalconfig.UseFullBinaryState is false
	index  types.ConsensusIndex
	// myVrf         sig.VRFProof
	partialMsgMap *partialMsgMap
	gc            *generalconfig.GeneralConfig
}

// NewSimpleMessageState creates a new simple message state object for a single consensus index.
func NewSimpleMessageState(gc *generalconfig.GeneralConfig) *SimpleMessageState {
	var msgMap sigMsgMapInterface
	if gc.UseMultiSig {
		msgMap = &blsSigState{}
	} else {
		msgMap = &signedMsgMap{}
	}
	return &SimpleMessageState{
		gc:            gc,
		msgMap:        msgMap,
		partialMsgMap: newPartialMsgMap()}
}

// GetCoinVal returns the threshold coin value for the message ID (if supported).
func (sms *SimpleMessageState) GetCoinVal(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (coinVal types.BinVal, ready bool, err error) {

	err = sms.CheckGenThresholdCoin(hdr, threshold, mc)
	if err != nil {
		return
	}
	sm := sig.NewMultipleSignedMsg(sms.index, mc.MC.GetMyPriv().GetPub(), hdr)
	// First we serialize just to calculate the hash
	if _, err = sm.Serialize(messages.NewMessage(nil)); err != nil {
		return
	}
	coinVal, ready = sms.msgMap.getCoinVal(sm.GetHashString())
	return
}

// GetThreshSig returns the threshold signature for the message header (if supported).
func (sms *SimpleMessageState) GetThreshSig(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (*sig.SigItem, error) {

	err := sms.CheckGenThresholdSig(hdr, threshold, mc)
	if err != nil {
		return nil, err
	}
	sm := sig.NewMultipleSignedMsg(sms.index, mc.MC.GetMyPriv().GetPub(), hdr)
	// First we serialize just to calculate the hash
	if _, err := sm.Serialize(messages.NewMessage(nil)); err != nil {
		return nil, err
	}
	return sms.msgMap.getThreshSig(sm.GetHashString())

}

// GetSigCountMsgIDList returns list of received messages that have msgID for their MsgID and how many signatures have been received for each.
func (sms *SimpleMessageState) GetSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {
	return sms.msgMap.getSigCountMsgIDList(msgID)
}

// GetSigCountMsg returns the number of signatures received for this message.
func (sms *SimpleMessageState) GetSigCountMsg(sm types.HashStr) int {
	return sms.msgMap.getSigCountMsg(sm)
}

// TrackTotalSigCount must be called before getTotalSigCount with the same set of hashes.
// This can be called for multiple sets of messages, but they all must be unique.
func (sms *SimpleMessageState) TrackTotalSigCount(mc *consinterface.MemCheckers,
	hdrs ...messages.InternalSignedMsgHeader) {

	msgHashes, err := sms.hdrsToHashes(mc, hdrs...)
	if err != nil {
		panic(err)
	}
	sms.Lock()
	defer sms.Unlock()

	sms.msgMap.trackTotalSigCount(msgHashes...)
}

// hdrsToHashes converts the headers to hashes
func (sms *SimpleMessageState) hdrsToHashes(mc *consinterface.MemCheckers,
	hdrs ...messages.InternalSignedMsgHeader) ([]types.HashStr, error) {

	ret := make([]types.HashStr, len(hdrs))
	for i, hdr := range hdrs {
		sm := sig.NewMultipleSignedMsg(sms.index, mc.MC.GetMyPriv().GetPub(), hdr)
		// First we serialize just to calculate the hash
		if _, err := sm.Serialize(messages.NewMessage(nil)); err != nil {
			return nil, err
		}
		ret[i] = sm.GetHashString()
	}
	return ret, nil
}

// getTotalSigCount returns the number of unique signers for the set of msgHashes.
// Unique means if a signer signs multiple of msgHashes he will still only be counted once.
// Note that trackTotalSigCount must be called first (but only once) with the same
// set of msgsHashes.
func (sms *SimpleMessageState) GetTotalSigCount(mc *consinterface.MemCheckers,
	hdrs ...messages.InternalSignedMsgHeader) (totalCount int, eachCount []int) {

	msgHashes, err := sms.hdrsToHashes(mc, hdrs...)
	if err != nil {
		panic(err)
	}
	sms.Lock()
	defer sms.Unlock()
	return sms.msgMap.getTotalSigCount(msgHashes...)
}

// GetSigCountMsgHeader returns the number of signatures received for this message header.
func (sms *SimpleMessageState) GetSigCountMsgHeader(hdr messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (int, error) {

	sm := sig.NewMultipleSignedMsg(sms.index, mc.MC.GetMyPriv().GetPub(), hdr)
	// First we serialize just to calculate the hash
	if _, err := sm.Serialize(messages.NewMessage(nil)); err != nil {
		return 0, err
	}
	return sms.GetSigCountMsg(sm.GetHashString()), nil
}

// GetSigCountMsgID returns the number of sigs for this message's MsgID (see messages.MsgID).
func (sms *SimpleMessageState) GetSigCountMsgID(sm messages.MsgID) int {
	return sms.msgMap.getSigCountMsgID(sm)
}

// SetupSignedMessageDuplicates takes a list of headers that are assumed to have the same set of bytes to sign
// (i.e. the signed part are all the same though they may have different contents following the signed part,
// for example this is true with partial messages)
func (sms *SimpleMessageState) SetupSignedMessagesDuplicates(combined *messagetypes.CombinedMessage,
	hdrs []messages.InternalSignedMsgHeader, mc *consinterface.MemCheckers) (combinedSigned *sig.MultipleSignedMessage,
	partialsSigned []*sig.MultipleSignedMessage, err error) {

	if len(hdrs) < 1 {
		return nil, nil, types.ErrInvalidIndex
	}

	// Sign the message
	partialsSigned = make([]*sig.MultipleSignedMessage, len(hdrs))
	partialsSigned[0], err = sms.SetupSignedMessage(hdrs[0], true, 0, mc)
	if err != nil {
		return nil, nil, err
	}
	// Setup the combined signed message
	combinedSigned = partialsSigned[0].ShallowCopy().(*sig.MultipleSignedMessage)
	combinedSigned.InternalSignedMsgHeader = combined

	// Setup the partials
	for i := 0; i < len(hdrs); i++ {
		partialsSigned[i] = partialsSigned[0].ShallowCopy().(*sig.MultipleSignedMessage)
		partialsSigned[i].InternalSignedMsgHeader = hdrs[i]
	}
	return
}

// CheckGenThresholdSig will see if a threshold signature can be generated.
// If it is then it will be generated and stored in the message state.
// Note that SetupSignedMessage will generate a threshold signature automatically if possible.
func (sms *SimpleMessageState) CheckGenThresholdSig(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (err error) {

	if _, ok := mc.MC.GetMyPriv().(sig.ThreshStateInterface); !ok {
		err = types.ErrThresholdSigsNotSupported
		return
	}
	var count int
	if count, err = sms.GetSigCountMsgHeader(hdr, mc); err != nil {
		return err
	}
	if count >= threshold {
		_, err = sms.SetupSignedMessage(hdr, false, threshold, mc)
	} else {
		err = types.ErrNotEnoughSigs
	}
	return
}

// CheckGenThresholdCoin will see if a random coin value can be generated.
// If it is then it will be generated and stored in the message state.
// Note that SetupSignedMessage will generate a coin automatically if possible.
func (sms *SimpleMessageState) CheckGenThresholdCoin(hdr messages.InternalSignedMsgHeader,
	threshold int, mc *consinterface.MemCheckers) (err error) {

	if hdr.GetSignType() != types.CoinProof {
		err = types.ErrInvalidHeader
		return
	}

	myPriv := mc.MC.GetMyPriv()
	if _, ok := myPriv.GetPub().(sig.CoinProofPubInterface); !ok {
		err = types.ErrThresholdSigsNotSupported
		_ = myPriv.GetPub().(sig.CoinProofPubInterface)
		return
	}
	var count int
	if count, err = sms.GetSigCountMsgHeader(hdr, mc); err != nil {
		return err
	}
	if count >= threshold {
		_, err = sms.SetupSignedMessage(hdr, false, threshold, mc)
	} else {
		err = types.ErrNotEnoughSigs
	}
	return
}

// SetupSignedMessage takes a MultiSigMsgHeader, serializes the message appending signatures.
// If generateMySig is true, the serialized message will include the local nodes signature.
// It will generate a threshold signature if supported and enough signatures.
// Up to addOtherSigs number of signatures received so far for this message will additionally be appended.
// If alwaysSign is true signatures will be generated even if the configuration is set to not use signatures.
// If alwaysIncludeVRF is true then a VRF is added even if the configuration is set to not use signatures.
func (sms *SimpleMessageState) SetupSignedMessage(hdr messages.InternalSignedMsgHeader,
	generateMySig bool, addOthersSigsCount int, mc *consinterface.MemCheckers) (*sig.MultipleSignedMessage, error) {

	if !generateMySig && addOthersSigsCount < 1 {
		return nil, types.ErrNoValidSigs
	}

	priv := mc.MC.GetMyPriv()
	sm := sig.NewMultipleSignedMsg(sms.index, priv.GetPub(), hdr)

	// First we serialize just to calculate the hash
	if _, err := sm.Serialize(messages.NewMessage(nil)); err != nil {
		return nil, err
	}

	var myVrf sig.VRFProof
	// add the signatures
	myVrf = mc.MC.GetMyVRF(hdr.GetMsgID())
	_, err := sms.msgMap.setupSigs(sm, priv, generateMySig, myVrf, addOthersSigsCount, mc)
	return sm, err
}

func (sms *SimpleMessageState) SetupUnsignedMessage(hdr messages.InternalSignedMsgHeader,
	mc *consinterface.MemCheckers) (*sig.UnsignedMessage, error) {

	priv := mc.MC.GetMyPriv()
	sm := sig.NewUnsignedMessage(sms.index, priv.GetPub(), hdr)
	sm.SetEncryptPubs([]sig.Pub{priv.GetPub()})

	return sm, nil
}

// GetIndex returns the consensus index.
func (sms *SimpleMessageState) GetIndex() types.ConsensusIndex {
	return sms.index
}

// New creates a new empty SimpleMessageState object for the consensus index idx.
func (sms *SimpleMessageState) New(idx types.ConsensusIndex) consinterface.MessageState {
	return &SimpleMessageState{
		gc:            sms.gc,
		msgMap:        newSigMsgMap(sms.msgMap, idx),
		msgs:          make([][]byte, 0, 10),
		index:         idx,
		partialMsgMap: newPartialMsgMap()}
}

// GotMessage takes a deserialized message and the member checker for the current consensus index.
// If the message contains no new valid signatures then an error is returned.
// The value newTotalSigCount is the new number of signatures for the specific message, the value newMsgIDSigCount is the
// number of signatures for the MsgID of the message (see messages.MsgID).
// If the message is not a signed type message (not type *sig.MultipleSignedMessage then (0, 0, nil) is returned).
func (sms *SimpleMessageState) GotMsg(hdrFunc consinterface.HeaderFunc,
	deser *channelinterface.DeserializedItem, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) ([]*channelinterface.DeserializedItem, error) {

	if !deser.IsDeserialized {
		panic("should be deserialized")
	}
	if deser.Index.Index != sms.index.Index {
		panic(fmt.Sprintf("idx %v, %v", deser.Index, sms.index))
	}

	switch v := deser.Header.(type) {
	case *sig.MultipleSignedMessage: // check if it is a signed message

		if err := sms.checkIndex(v.Index); err != nil {
			return nil, err
		}

		var shouldNotValidate bool
		var err error
		if _, ok := v.GetBaseMsgHeader().(*messagetypes.PartialMessage); ok { // path for partial message
			shouldNotValidate, err = sms.partialMsgMap.gotMsg(v)
			if err != nil {
				return nil, err
			}
		} else { // path for normal message
			// check if there are any new signatures
			// this is done first so we don't validate signatures multiple times
			err = sms.msgMap.gotMsg(v)
			if err != nil {
				return nil, err
			}
		}

		var invalidSigs []*sig.SigItem
		if !shouldNotValidate {
			// only take the valid signatures
			allSigs := v.GetSigItems()
			validSigs := make([]*sig.SigItem, 0, len(allSigs))
			for _, sigItem := range allSigs {
				// Check which signatures are valid and which are invalid
				err := consinterface.CheckMember(mc, deser.Index, sigItem, v)
				if err == nil {
					validSigs = append(validSigs, sigItem)
				} else {
					logging.Info("Got message from non-member: ", err)
					invalidSigs = append(invalidSigs, sigItem)
				}
			}
			v.SetSigItems(validSigs)
		}

		if _, ok := v.GetBaseMsgHeader().(*messagetypes.PartialMessage); ok { // path for partial message
			cm, err := sms.partialMsgMap.storeMsg(hdrFunc, v, gc, mc)
			if err != nil {
				return nil, err
			}

			sms.addToMsgList(deser)

			ret := []*channelinterface.DeserializedItem{deser}
			if cm != nil { // we reconstructed a partial message

				// create a signed message from the new combined message
				m := messages.NewMessage(nil)
				_, err := cm.Serialize(m)
				if err != nil {
					panic(err)
				}
				cmdi := deser.CopyBasic()
				cmdi.HeaderType = cm.GetBaseMsgHeader().GetID()
				cmdi.Header = cm
				cmdi.Message = sig.EncodedMsg{Message: m}
				// now process it as a normal message
				newSM, err := sms.GotMsg(hdrFunc, cmdi, gc, mc)
				if err != nil { // if there was an error processing the full message then we just return the error
					return nil, err
				}
				// Otherwise we return both the partial and the combined
				ret = append(ret, newSM...)
			}
			return ret, nil

		} else { // path for normal message

			// store the message, v contains only valid signatures, invalidSigs are the remaining from the original message
			deser.NewTotalSigCount, deser.NewMsgIDSigCount, err = sms.msgMap.storeMsg(v, invalidSigs, mc)
			if err == nil {
				sms.addToMsgList(deser)

				return []*channelinterface.DeserializedItem{deser}, nil
			}
			return nil, err
		}

	case *sig.UnsignedMessage:

		allPubs := v.GetEncryptPubs()
		validPubs := make([]sig.Pub, 0, len(allPubs))
		for _, pub := range allPubs {
			pid, err := pub.GetPubID()
			if err != nil {
				panic(err)
			}
			if p := mc.MC.CheckMemberBytes(deser.Index, pid); p != nil {
				validPubs = append(validPubs, p)
			} else {
				logging.Info("Got message from non-member: ", err)
			}
		}
		v.SetEncryptPubs(validPubs)

		if deser.IsLocal != types.NonLocalMessage {
			// We should have already checked the pubs
			// v.SetEncryptPubs(append(v.GetEncryptPubs(), mc.MC.GetMyPriv().GetPub()))
		} else {

			// Check if we find the sender pub
			// TODO this should only be a single pub anyway, but since when loading from disk
			// all the pubs are loaded there may be multiple here
			// instead should only send a single pub when loaded from disk
			pb, err := deser.Message.Pub.GetRealPubBytes()
			if err != nil {
				panic(err)
			}
			var foundPub sig.Pub
			for _, nxt := range v.GetEncryptPubs() {
				op, err := nxt.GetRealPubBytes()
				if err != nil {
					panic(err)
				}
				if bytes.Equal(pb, op) {
					foundPub = nxt
					break
				}
			}
			if foundPub == nil {
				return nil, types.ErrInvalidPub
			}
			v.SetEncryptPubs([]sig.Pub{foundPub})
		}

		if err := sms.checkIndex(v.Index); err != nil {
			return nil, err
		}
		var err error
		// check if there are any new signatures
		// this is done first so we don't validate signatures multiple times
		err = sms.msgMap.gotUnsignedMsg(v)
		if err != nil {
			return nil, err
		}
		// store the message, v contains only valid signatures, invalidSigs are the remaining from the original message
		deser.NewTotalSigCount, deser.NewMsgIDSigCount, err = sms.msgMap.storeUnsignedMsg(v, mc)
		if err == nil {
			sms.addToMsgList(deser)

			return []*channelinterface.DeserializedItem{deser}, nil
		}
		return nil, err

	default:
		// We only track signed messages
		return []*channelinterface.DeserializedItem{deser}, nil
	}
}

func (sms *SimpleMessageState) checkIndex(idx types.ConsensusIndex) error {
	gc := sms.gc
	switch gc.Ordering {
	case types.Total:
		// Total order can only have a single index
		if len(idx.AdditionalIndices) != 0 {
			return types.ErrTooManyAdditionalIndices
		}
	case types.Causal:
		// ok
	default:
		panic(gc.Ordering)
	}
	return nil
}

func (sms *SimpleMessageState) addToMsgList(deser *channelinterface.DeserializedItem) {
	if sms.gc.UseFullBinaryState {
		sms.Lock()
		// for our message state, TODO remove messages state?
		sms.msgs = append(sms.msgs, deser.Message.GetBytes())
		sms.Unlock()
	}
}

// GetMsgState should return the serialized state of all the valid messages received for this consensus instance.
// This should be able to be processed by UnwrapMessage.
// If generalconfig.UseFullBinaryState is true it returns the received signed messages together in a list.
// Otherwise it returns each unique message with the list of signatures for each message behind it.
// If localOnly is true then only proposal messages and signatures from the local node will be included.
func (sms *SimpleMessageState) GetMsgState(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc,
	mc *consinterface.MemCheckers) ([]byte, error) {

	if sms.gc.UseFullBinaryState && !localOnly {
		sms.RLock()
		i := 0
		totalSize := 0
		for _, item := range sms.msgs {
			totalSize += len(item)
			i++
		}
		buff := make([]byte, totalSize)
		index := 0
		for _, item := range sms.msgs {
			copy(buff[index:], item)
			index += len(item)
		}
		sms.RUnlock()
		return buff, nil
	}
	// send both the partials and the normal messages
	// TODO howto handle partials
	hdrs := append(sms.msgMap.getAllMsgSigs(priv, localOnly, bufferCountFunc, sms.gc, mc), sms.partialMsgMap.getAllPartials()...)
	ret, err := messages.SerializeHeaders(hdrs)
	if err != nil {
		return nil, err
	}
	return ret.GetBytes(), nil
}
