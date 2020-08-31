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
	"fmt"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"sync"

	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/utils"
)

/////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type signedMsgState struct {
	sync.Mutex
	SigMap         map[sig.PubKeyID]*sig.SigItem // Map from pubkey to signed item
	singleSigCount int
	thrshSigCount  int
	thrshSig       *sig.SigItem
	coinVal        *types.BinVal
	msgHeader      *sig.MultipleSignedMessage
}

type unsignedMsgState struct {
	sync.Mutex
	SigMap         map[sig.PubKeyID]sig.Pub
	singleSigCount int
	msgHeader      *sig.UnsignedMessage
}

func newUnsignedMsgState(usm *sig.UnsignedMessage) (*unsignedMsgState, error) {
	return &unsignedMsgState{
		SigMap:    make(map[sig.PubKeyID]sig.Pub),
		msgHeader: usm.ShallowCopy().(*sig.UnsignedMessage),
	}, nil
}

func newSignedMsgState(sm *sig.MultipleSignedMessage) (*signedMsgState, error) {
	return &signedMsgState{
		msgHeader: sm.ShallowCopy().(*sig.MultipleSignedMessage),
		SigMap:    make(map[sig.PubKeyID]*sig.SigItem)}, nil
}

type msgIDcount struct {
	sync.Mutex
	pubKeyMap      map[sig.PubKeyID]bool
	thrshSigCounts []int
	singleSigCount int
}

/////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////

type signedMsgMap struct {
	msgMap         map[types.HashStr]*signedMsgState
	unsignedMsgMap map[types.HashStr]*unsignedMsgState
	sync.Mutex
	msgIDMap map[messages.MsgID]*msgIDcount

	// this is for keeping a unique set of items for different message sets
	totalSigCountMap map[types.HashStr]map[sig.PubKeyID]bool
}

// getTotalSigCount returns the number of unique signers for the set of msgHashes.
// Unique means if a signer signs multiple of msgHashes he will still only be counted once.
// Note that trackTotalSigCount must be called first (but only once) with the same
// set of msgsHashes.
func (sms *signedMsgMap) getTotalSigCount(msgHashes ...types.HashStr) (totalCount int, eachCount []int) {
	sms.Lock()
	defer sms.Unlock()

	if len(msgHashes) < 2 {
		panic("should be at least two messages")
	}
	eachCount = make([]int, len(msgHashes))
	// var maxCount int
	for i, nxtHash := range msgHashes {
		// First check if we have a threshold sig with the largest count
		if mItem, ok := sms.msgMap[nxtHash]; ok {
			if v := mItem.thrshSigCount; v > totalCount {
				totalCount = v
			}
			eachCount[i] = utils.Max(mItem.thrshSigCount, mItem.singleSigCount)
		}
		// Otherwise take the max individaul count
		if v := len(sms.totalSigCountMap[nxtHash]); v > totalCount {
			totalCount = v
		}
	}
	return
}

// trackTotalSigCount must be called before getTotalSigCount with the same set of hashes.
// This can be called for multiple sets of messages, but they all must be unique.
func (sms *signedMsgMap) trackTotalSigCount(msgHashes ...types.HashStr) {
	sms.Lock()
	defer sms.Unlock()

	if len(msgHashes) < 2 {
		panic("should be at least two messages")
	}
	newMap := make(map[sig.PubKeyID]bool)
	for _, nxt := range msgHashes {
		if _, ok := sms.totalSigCountMap[nxt]; ok {
			panic("must only be for unique messages")
		}
		sms.totalSigCountMap[nxt] = newMap
		// Add the messages we have already received
		var foundSigned bool
		if itm, ok := sms.msgMap[nxt]; ok {
			foundSigned = true
			for pubID, aSig := range itm.SigMap {
				// only keep non-threshold sigs
				if aSig.Pub.GetSigMemberNumber() == 1 {
					newMap[pubID] = true
				}
			}
		}

		if itm, ok := sms.unsignedMsgMap[nxt]; ok {
			if foundSigned {
				panic("should not use same msg type for both signed and unsigned messages")
			}
			for pubID := range itm.SigMap {
				newMap[pubID] = true
			}
		}

	}
}

func newSignedMsgMap() *signedMsgMap {
	return &signedMsgMap{
		unsignedMsgMap:   make(map[types.HashStr]*unsignedMsgState),
		msgMap:           make(map[types.HashStr]*signedMsgState),
		totalSigCountMap: make(map[types.HashStr]map[sig.PubKeyID]bool),
		msgIDMap:         make(map[messages.MsgID]*msgIDcount)}
}

// getCoinVal returns the value for the random coin (if enough messages have been received), nil otherwise
func (sms *signedMsgMap) getCoinVal(hash types.HashStr) (ret types.BinVal, ready bool) {
	item, err := sms.getSignedMsgStateByHash(hash)
	if err == types.ErrMsgNotFound {
		return
	}
	if item == nil {
		panic("should not be nil")
	}
	item.Lock()
	defer item.Unlock()

	if item.coinVal != nil {
		return *item.coinVal, true
	}
	return
}

// GetThreshSig returns the threshold signature for the message ID (if supported).
func (sms *signedMsgMap) getThreshSig(hash types.HashStr) (*sig.SigItem, error) {
	item, err := sms.getSignedMsgStateByHash(hash)
	if err == types.ErrMsgNotFound {
		return nil, err
	}
	if item == nil {
		panic("should not be nil")
	}

	item.Lock()
	defer item.Unlock()

	if item.thrshSig == nil {
		return nil, types.ErrMsgNotFound
	}
	return item.thrshSig, nil
}

// GotMsg checks if the same message has already been received from the signers
// the message is updated so that it only contains the sigs not already seen
// an error is sent if no new signers.
func (sms *signedMsgMap) gotMsg(sm *sig.MultipleSignedMessage) error {
	item, _, err := sms.getSignedMsgState(sm)
	if err != nil {
		// panic(1)
		return err
	}
	var newItems []*sig.SigItem
	item.Lock()
	for _, si := range sm.GetSigItems() {
		str, err := si.Pub.GetPubID()
		if err != nil {
			panic(err)
		}

		if _, ok := item.SigMap[str]; !ok {
			newItems = append(newItems, si)
		}
	}
	/*if len(newItems) == 0 {
		var idsHave []uint32
		var idsGot []uint32
		for k, _ := range item.SigMap {
			idsHave = append(idsHave, generalconfig.Encoding.Uint32([]byte(k)))
		}
		for _, k := range sm.GetSigItems() {
			ii, _ := k.Pub.GetPubID()
			idsGot = append(idsGot, generalconfig.Encoding.Uint32([]byte(ii)))
		}
	} */
	item.Unlock()
	sm.SetSigItems(newItems)
	if len(newItems) == 0 {
		return types.ErrNoNewSigs
	}
	return nil
}

func (sms *signedMsgMap) gotUnsignedMsg(sm *sig.UnsignedMessage) error {
	item, _, err := sms.getUnsignedMsgState(sm)
	if err != nil {
		// panic(1)
		return err
	}
	var newItems []sig.Pub
	item.Lock()
	for _, pub := range sm.GetEncryptPubs() {
		str, err := pub.GetPubID()
		if err != nil {
			panic(err)
		}

		if _, ok := item.SigMap[str]; !ok {
			newItems = append(newItems, pub)
		}
	}
	/*if len(newItems) == 0 {
		var idsHave []uint32
		var idsGot []uint32
		for k, _ := range item.SigMap {
			idsHave = append(idsHave, generalconfig.Encoding.Uint32([]byte(k)))
		}
		for _, k := range sm.GetSigItems() {
			ii, _ := k.Pub.GetPubID()
			idsGot = append(idsGot, generalconfig.Encoding.Uint32([]byte(ii)))
		}
	} */
	item.Unlock()
	sm.SetEncryptPubs(newItems)
	if len(newItems) == 0 {
		return types.ErrNoNewSigs
	}
	return nil
}

// storeMsg shold be called after gotMsg, and after the signatres are validated.
// It stores the new valid signatures and returns the total number of signatures that the message has.
func (sms *signedMsgMap) storeMsg(sm *sig.MultipleSignedMessage, invalidSigs []*sig.SigItem,
	mc *consinterface.MemCheckers) (int, int, error) {

	_ = invalidSigs // TODO should track invalid signatures?
	_ = mc

	item, msgIDitem, err := sms.getSignedMsgState(sm)
	if err != nil {
		return 0, 0, err
	}
	sms.Lock() // TODO fix
	defer sms.Unlock()

	// msgID := sm.GetID()
	validSigs := sm.GetSigItems()
	if len(validSigs) == 0 {
		return 0, 0, types.ErrNoValidSigs
	}
	newValidSigs := make([]*sig.SigItem, 0, len(validSigs))
	item.Lock()
	msgIDitem.Lock()
	for _, si := range validSigs {
		str, err := si.Pub.GetPubID()
		if err != nil {
			panic(err)
		}
		// add it to the totalSigCount map (if we have created one)
		if tcm, ok := sms.totalSigCountMap[sm.GetHashString()]; ok {
			tcm[str] = true
		}

		// add to the per msg id count
		if !msgIDitem.pubKeyMap[str] {
			msgIDitem.pubKeyMap[str] = true
			if v := si.Pub.GetSigMemberNumber(); v > 1 {
				msgIDitem.thrshSigCounts = append(msgIDitem.thrshSigCounts, v)
			} else {
				msgIDitem.singleSigCount++
			}
		}
		// add to the per message count
		if _, ok := item.SigMap[str]; !ok {
			item.SigMap[str] = si
			if v := si.Pub.GetSigMemberNumber(); v > 1 {
				if item.thrshSig != nil {
					if item.thrshSigCount != v {
						panic(fmt.Sprintf("Got thrsh sig with count %v, but already had %v", item.thrshSigCount, v))
					}
				} else {
					item.thrshSigCount = v
					item.thrshSig = si
				}
			} else {
				item.singleSigCount++
			}
			newValidSigs = append(newValidSigs, si)
		}
	}
	ret := utils.MaxInt(item.singleSigCount, item.thrshSigCount)
	ret2 := utils.MaxInt(msgIDitem.singleSigCount, msgIDitem.thrshSigCounts...)
	item.Unlock()
	msgIDitem.Unlock()

	sm.SetSigItems(newValidSigs)
	if len(newValidSigs) == 0 {
		return 0, 0, types.ErrNoValidSigs
	}
	return ret, ret2, nil
}

func (sms *signedMsgMap) storeUnsignedMsg(sm *sig.UnsignedMessage,
	mc *consinterface.MemCheckers) (int, int, error) {

	_ = mc

	item, msgIDitem, err := sms.getUnsignedMsgState(sm)
	if err != nil {
		return 0, 0, err
	}
	sms.Lock() // TODO fix
	defer sms.Unlock()

	// msgID := sm.GetID()
	validPubs := sm.EncryptPubs
	if len(validPubs) == 0 {
		return 0, 0, types.ErrNoValidSigs
	}
	newValidPubs := make([]sig.Pub, 0, len(validPubs))
	item.Lock()
	msgIDitem.Lock()
	for _, pub := range validPubs {
		str, err := pub.GetPubID()
		if err != nil {
			panic(err)
		}
		// add it to the totalSigCount map (if we have created one)
		if tcm, ok := sms.totalSigCountMap[sm.GetHashString()]; ok {
			tcm[str] = true
		}

		// add to the per msg id count
		if !msgIDitem.pubKeyMap[str] {
			msgIDitem.pubKeyMap[str] = true
			msgIDitem.singleSigCount++
		}
		// add to the per message count
		if _, ok := item.SigMap[str]; !ok {
			item.SigMap[str] = pub
			item.singleSigCount++
			newValidPubs = append(newValidPubs, pub)
		}
	}
	ret := item.singleSigCount
	ret2 := msgIDitem.singleSigCount
	item.Unlock()
	msgIDitem.Unlock()

	sm.SetEncryptPubs(newValidPubs)
	if len(newValidPubs) == 0 {
		return 0, 0, types.ErrNoValidSigs
	}
	return ret, ret2, nil
}

// getSigCountMsgIDList returns the count for all messages that match the MsgID
func (sms *signedMsgMap) getSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {
	sms.Lock()
	defer sms.Unlock()

	ret := make([]consinterface.MsgIDCount, 0, 2)
	for _, item := range sms.msgMap {
		if item.msgHeader.GetMsgID() == msgID {
			item.Lock()
			ret = append(ret, consinterface.MsgIDCount{MsgHeader: item.msgHeader,
				Count: utils.MaxInt(item.singleSigCount, item.thrshSigCount)})
			item.Unlock()
		}
	}
	return ret
}

func (sms *signedMsgMap) getSigCountMsg(hash types.HashStr) int {
	item, err := sms.getSignedMsgStateByHash(hash)
	if err == types.ErrMsgNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}
	item.Lock()
	defer item.Unlock()

	return utils.MaxInt(item.singleSigCount, item.thrshSigCount)
}

func (sms *signedMsgMap) getSigCountMsgID(msgID messages.MsgID) int {
	item, err := sms.getSignedMsgStateByMsgID(msgID)
	if err == types.ErrMsgNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}
	item.Lock()
	defer item.Unlock()

	return utils.MaxInt(item.singleSigCount, item.thrshSigCounts...)
}

func (sms *signedMsgMap) getAllMsgSigs(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) (ret []messages.MsgHeader) {
	sms.Lock()
	defer sms.Unlock()

	for _, usm := range sms.unsignedMsgMap { // get unsigned messages
		usm.Lock()
		sm := usm.msgHeader.ShallowCopy().(*sig.UnsignedMessage)

		if localOnly {
			pb, err := mc.MC.GetMyPriv().GetPub().GetPubID()
			if err != nil {
				panic(err)
			}
			if pub := usm.SigMap[pb]; pub != nil {
				sm.SetEncryptPubs([]sig.Pub{pub})
				ret = append(ret, sm)
			}
		} else {
			var pubs []sig.Pub
			for _, pub := range usm.SigMap {
				pubs = append(pubs, pub)
			}
			sm.SetEncryptPubs(pubs)
			ret = append(ret, sm)
		}
		usm.Unlock()
	}

	for _, sigms := range sms.msgMap { // get signed messages
		sigms.Lock()
		sm := sigms.msgHeader.ShallowCopy().(*sig.MultipleSignedMessage)

		var minSigCount int
		var err error
		if bufferCountFunc != nil {
			minSigCount, _, _, err = bufferCountFunc(sm, gc, mc)
			if err != nil && err != types.ErrDontForwardMessage || minSigCount <= 0 {
				panic(fmt.Sprintf("%v, %v", err, sm.GetID()))
			}
		}

		var sigItems []*sig.SigItem
		// if local only and a non-proposal message then only get my signature
		if localOnly && !messages.IsProposalHeader(mc.MC.GetIndex(), sigms.msgHeader.InternalSignedMsgHeader) {
			myId, err := priv.GetPub().GetPubID()
			if err != nil {
				panic(err)
			}
			if nxt, ok := sigms.SigMap[myId]; ok {
				sigItems = append(sigItems, nxt)
			}
		} else {
			sigItems = joinSigs(sigms, minSigCount, priv, mc)
		}

		sigms.Unlock()
		if len(sigItems) == 0 {
			continue
		}
		sm.SetSigItems(sigItems)
		ret = append(ret, sm)
	}
	return
}

func joinSigs(sigms *signedMsgState, addOthersSigsCount int, priv sig.Priv, mc *consinterface.MemCheckers) (sigItems []*sig.SigItem) {
	// var sigItems []*sig.SigItem
	if addOthersSigsCount > 0 { //|| generalconfig.IncludeCurrentSigs {
		totalCount := 0

		signType := sigms.msgHeader.GetBaseMsgHeader().GetSignType()
		var err error
		if priv, err = priv.GetPrivForSignType(signType); err != nil {
			panic(err)
		}

		// Check if this is a threshold signature, or a coin threshold message
		if thrsh, ok := priv.(sig.ThreshStateInterface); ok { // We use a threshold signature
			// add threshold sigs first
			if tSig := sigms.thrshSig; tSig != nil {
				sigItems = append(sigItems, tSig)
				totalCount += tSig.Pub.GetSigMemberNumber()
			} else {
				// if we don't have a threshold sig already, see if we can create one
				t := thrsh.GetT()
				if len(sigms.SigMap) >= t && t >= addOthersSigsCount { // only generate a threshold sig if it will cover all we need
					sigList := make([]sig.Sig, t)
					i := 0
					for _, asig := range sigms.SigMap {
						sigList[i] = asig.Sig
						i++
						if i >= t {
							break
						}
					}
					tSig, err := thrsh.CombinePartialSigs(sigList)
					if err != nil {
						panic(err)
					}
					logging.Infof("Generating threshold signature, threshold %v, msg id %v",
						tSig.Pub.GetSigMemberNumber(), sigms.msgHeader.GetID())
					mc.MC.GetStats().CombinedThresholdSig()
					sigms.thrshSig = tSig
					sigms.thrshSigCount = tSig.Pub.GetSigMemberNumber()
					sigItems = append(sigItems, tSig)
					totalCount += tSig.Pub.GetSigMemberNumber()
				}
			}
		} else if thrsh, ok := priv.GetPub().(sig.CoinProofPubInterface); ok && signType == types.CoinProof { // We use a coin proof
			// thrsh := priv.GetPub().(sig.CoinProofPubInterface)
			t := thrsh.GetT()
			if sigms.coinVal == nil && len(sigms.SigMap) >= t {
				var coinSigs []*sig.SigItem
				for _, nxt := range sigms.SigMap {
					coinSigs = append(coinSigs, nxt)
				}
				if sigms.coinVal == nil {
					coinVal, err := thrsh.CombineProofs(priv, coinSigs)
					if err != nil {
						panic(err)
					}
					sigms.coinVal = &coinVal
					mc.MC.GetStats().ComputedCoin()
				}
			}
		}

		// if the threshold signautre was not enough to satisfy the signautre count
		if totalCount < addOthersSigsCount {
			totalCount = 0
			sigItems = append(make([]*sig.SigItem, 0, len(sigms.SigMap)+1), sigItems...)

			// add my signature first
			pid, err := mc.MC.GetMyPriv().GetPub().GetPubID()
			utils.PanicNonNil(err)
			if itm, ok := sigms.SigMap[pid]; ok {
				sigItems = append(sigItems, itm)
				totalCount++
			}
			// add normal sigs as needed
			for nxtPid, asig := range sigms.SigMap {
				if nxtPid == pid {
					continue
				}
				if totalCount >= addOthersSigsCount {
					break
				}
				sigItems = append(sigItems, asig)
				totalCount++
			}
		}
		// sigItems = sigms.Priv.ComputeSigs(sigms.SigMap)
	}
	return
}

func (sms *signedMsgMap) setupSigs(sm *sig.MultipleSignedMessage, priv sig.Priv, generateMySig bool,
	myVrf sig.VRFProof, addOthersSigsCount int, mc *consinterface.MemCheckers) (bool, error) {

	sigms, _, err := sms.getSignedMsgState(sm)
	if err != nil {
		return false, err
	}
	// var sigItems []*sig.SigItem
	sms.Lock()
	sigms.Lock()
	sigItems := joinSigs(sigms, addOthersSigsCount, priv, mc)
	sigms.Unlock()
	sms.Unlock()

	if generateMySig {
		mc.MC.GetStats().SignedItem()
		mySig, err := priv.GenerateSig(sm, myVrf, sm.InternalSignedMsgHeader.GetSignType())
		if err != nil {
			return false, err
		}
		sigItems = append(sigItems, mySig)
	}

	if len(sigItems) == 0 {
		// panic(fmt.Sprint(sm, sm.GetID(), sm.Index))
		return false, types.ErrNotEnoughSigs
	}
	sm.SetSigItems(sigItems)
	return generateMySig, nil
}

func (sms *signedMsgMap) getSignedMsgState(sm *sig.MultipleSignedMessage) (*signedMsgState, *msgIDcount, error) {
	hashStr := sm.GetHashString()
	sms.Lock()
	defer sms.Unlock()

	item, ok := sms.msgMap[hashStr]
	if !ok {
		if sms.unsignedMsgMap[hashStr] != nil {
			panic("should not have both signed and unsiged versions of same message")
		}
		var err error
		item, err = newSignedMsgState(sm)
		if err != nil {
			return nil, nil, err
		}
		sms.msgMap[hashStr] = item
	}
	msgID := sm.GetMsgID()
	idItem, ok := sms.msgIDMap[msgID]
	if !ok {
		idItem = &msgIDcount{pubKeyMap: make(map[sig.PubKeyID]bool)}
		sms.msgIDMap[msgID] = idItem
	}
	return item, idItem, nil
}

func (sms *signedMsgMap) getUnsignedMsgState(sm *sig.UnsignedMessage) (*unsignedMsgState, *msgIDcount, error) {
	hashStr := sm.GetHashString()
	sms.Lock()
	defer sms.Unlock()

	item, ok := sms.unsignedMsgMap[hashStr]
	if !ok {
		// sanity check
		if sms.msgMap[hashStr] != nil {
			panic("should not have both signed and unsiged versions of same message")
		}

		var err error
		item, err = newUnsignedMsgState(sm)
		if err != nil {
			return nil, nil, err
		}
		sms.unsignedMsgMap[hashStr] = item
	}
	msgID := sm.GetMsgID()
	idItem, ok := sms.msgIDMap[msgID]
	if !ok {
		idItem = &msgIDcount{pubKeyMap: make(map[sig.PubKeyID]bool)}
		sms.msgIDMap[msgID] = idItem
	}
	return item, idItem, nil
}

func (sms *signedMsgMap) getSignedMsgStateByHash(hashStr types.HashStr) (*signedMsgState, error) {
	sms.Lock()
	defer sms.Unlock()

	item, ok := sms.msgMap[hashStr]
	if !ok {
		return nil, types.ErrMsgNotFound
	}
	return item, nil
}

func (sms *signedMsgMap) getUnsignedMsgStateByHash(hashStr types.HashStr) (*unsignedMsgState, error) {
	sms.Lock()
	defer sms.Unlock()

	item, ok := sms.unsignedMsgMap[hashStr]
	if !ok {
		return nil, types.ErrMsgNotFound
	}
	return item, nil
}

func (sms *signedMsgMap) getSignedMsgStateByMsgID(msgID messages.MsgID) (*msgIDcount, error) {
	sms.Lock()
	defer sms.Unlock()

	idItem, ok := sms.msgIDMap[msgID]
	if !ok {
		return nil, types.ErrMsgNotFound
	}
	return idItem, nil
}

/////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////
