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
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
	"math"
	"sort"
	"sync"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/utils"
)

type blsSigItem struct {
	pub      sig.AllMultiPub // *bls.Blspub
	sig      sig.AllMultiSig // *bls.Blssig
	proof    sig.VRFProof
	sigBytes *[]byte
}

type blsSigMsgState struct {
	cond           *sync.Cond
	msgHeader      *sig.MultipleSignedMessage
	sigs           map[int]*blsSigItem
	allSigs        bitid.NewBitIDInterface
	validatingSigs bitid.NewBitIDInterface
	hasNewSigs     int // 0 means we have new sigs not added to the signed count yet, a larger number means we computed a merged sig that had requested that amount (but may have had more or less signatures at the time)

	fullSigList      SortBlsItem
	nonDuplicateSigs map[blsSigItem]bool
	intFunc          bitid.FromIntFunc
	pool             bitid.BitIDPoolInterface
}

type blsSigMsgIDState struct {
	sync.Mutex
	allSigs bitid.NewBitIDInterface
}

////////////////////////////////////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////////////////////////////////////

// If localOnly is true then only proposal messages and signatures from the local node will be included.
func (smm *blsSigState) getAllMsgSigs(priv sig.Priv, localOnly bool,
	bufferCountFunc consinterface.BufferCountFunc, gc *generalconfig.GeneralConfig,
	mc *consinterface.MemCheckers) (ret []messages.MsgHeader) {
	smm.Lock()
	defer smm.Unlock()

	for _, sigms := range smm.msgMap {
		sm := sigms.msgHeader.ShallowCopy().(*sig.MultipleSignedMessage)
		var sigItems []*sig.SigItem

		minSigCount, _, _, err := bufferCountFunc(sm, gc, mc)
		if err != nil && err != types.ErrDontForwardMessage || minSigCount <= 0 {
			panic(fmt.Sprintf("%v, %T", err, sm))
		}

		sigms.cond.L.Lock()

		if localOnly && !messages.IsProposalHeader(mc.MC.GetIndex(), sigms.msgHeader.InternalSignedMsgHeader) {
			if sigItem, ok := sigms.sigs[int(priv.GetPub().GetIndex())]; ok {
				sigItems = append(sigItems, &sig.SigItem{Pub: sigItem.pub, Sig: sigItem.sig,
					SigBytes: *sigItem.sigBytes, VRFProof: sigItem.proof})
			}
		} else {

			if config.AllowMultiMerge {
				sigms.computeMergedSigsMultiple(minSigCount, mc)
			} else {
				sigms.computeMergedSigs(minSigCount, mc)
			}
			for sigItem := range sigms.nonDuplicateSigs {
				sigItems = append(sigItems, &sig.SigItem{Pub: sigItem.pub, Sig: sigItem.sig,
					SigBytes: *sigItem.sigBytes, VRFProof: sigItem.proof})
			}
		}
		sigms.cond.L.Unlock()
		if len(sigItems) == 0 {
			continue
		}
		sm.SetSigItems(sigItems)
		ret = append(ret, sm)
	}
	return
}

type blsSigState struct {
	sync.Mutex
	msgMap       map[types.HashStr]*blsSigMsgState
	msgIDMap     map[messages.MsgID]*blsSigMsgIDState
	numPubs      int
	index        types.ConsensusIndex
	intFunc      bitid.FromIntFunc
	newBitIDFunc bitid.NewBitIDFunc
	pool         bitid.BitIDPoolInterface
}

func newBlsSigMsgState(sm *sig.MultipleSignedMessage, intFunc bitid.FromIntFunc, idFunc bitid.NewBitIDFunc,
	pool bitid.BitIDPoolInterface) (*blsSigMsgState, error) {
	allSigs := intFunc(nil)
	validatingSigs := intFunc(nil)

	return &blsSigMsgState{
		msgHeader:      sm.ShallowCopy().(*sig.MultipleSignedMessage),
		cond:           sync.NewCond(&sync.Mutex{}),
		sigs:           make(map[int]*blsSigItem),
		allSigs:        allSigs,
		validatingSigs: validatingSigs,
		intFunc:        intFunc,
		// newBitIDFunc: idFunc,
		pool: pool, // bitid.NewBitIDPool(idFunc, false),
	}, nil
}

func newBlsSigState(index types.ConsensusIndex, intFunc bitid.FromIntFunc, idFunc bitid.NewBitIDFunc,
	prev *blsSigState) *blsSigState {
	if !sig.GetUsePubIndex() {
		panic("should only be used with pub indecies")
	}
	// pool := bitid.NewBitIDPool(idFunc, false)

	return &blsSigState{
		index:        index,
		newBitIDFunc: idFunc,
		pool:         prev.pool,
		intFunc:      intFunc,
		numPubs:      math.MaxInt32,
		msgMap:       make(map[types.HashStr]*blsSigMsgState),
		msgIDMap:     make(map[messages.MsgID]*blsSigMsgIDState)}
}

// getCoinVal is not supported.
func (smm *blsSigState) getCoinVal(types.HashStr) (coinVal types.BinVal, ready bool) {
	panic("unsupported")
}

func (smm *blsSigState) gotUnsignedMsg(*sig.UnsignedMessage) error {
	panic("unsupported")
}
func (smm *blsSigState) storeUnsignedMsg(*sig.UnsignedMessage,
	*consinterface.MemCheckers) (int, int, error) {

	panic("unsupported")
}

// GetThreshSig is not supported
func (smm *blsSigState) getThreshSig(_ types.HashStr) (*sig.SigItem, error) {
	panic("unsupported")
}

// TODO trackTotalSigCount must be called before getTotalSigCount with the same set of hashes.
// This can be called for multiple sets of messages, but they all must be unique.
func (smm *blsSigState) trackTotalSigCount(_ ...types.HashStr) {
	panic("TODO")
}

// TODO getTotalSigCount returns the number of unique signers for the set of msgHashes.
// Unique means if a signer signs multiple of msgHashes he will still only be counted once.
// Note that trackTotalSigCount must be called first (but only once) with the same
// set of msgsHashes.
func (smm *blsSigState) getTotalSigCount(_ ...types.HashStr) (totalCount int, eachCount []int) {
	panic("todo")
}

func (smm *blsSigState) getSignedMsgStateByHash(hashStr types.HashStr) (*blsSigMsgState, error) {
	smm.Lock()
	defer smm.Unlock()

	item, ok := smm.msgMap[hashStr]
	if !ok {
		return nil, types.ErrMsgNotFound
	}
	return item, nil
}

func (smm *blsSigState) getSignedMsgStateByMsgID(msgID messages.MsgID) (*blsSigMsgIDState, error) {
	smm.Lock()
	defer smm.Unlock()

	idItem, ok := smm.msgIDMap[msgID]
	if !ok {
		return nil, types.ErrMsgNotFound
	}
	return idItem, nil
}

func (smm *blsSigState) getSignedMsgState(sm *sig.MultipleSignedMessage) (*blsSigMsgState, *blsSigMsgIDState, error) {
	if sm.Index.Index != smm.index.Index {
		panic(1)
	}
	hashStr := sm.GetHashString()
	smm.Lock()
	defer smm.Unlock()

	item, ok := smm.msgMap[hashStr]
	if !ok {
		var err error
		item, err = newBlsSigMsgState(sm, smm.intFunc, smm.newBitIDFunc, smm.pool)
		if err != nil {
			return nil, nil, err
		}
		smm.msgMap[hashStr] = item
	}
	msgID := sm.GetMsgID()
	idItem, ok := smm.msgIDMap[msgID]
	if !ok {
		allSigs := smm.newBitIDFunc()
		idItem = &blsSigMsgIDState{allSigs: allSigs}
		smm.msgIDMap[msgID] = idItem
	}
	return item, idItem, nil
}

type SortSigItem []*sig.SigItem

func (a SortSigItem) Len() int      { return len(a) }
func (a SortSigItem) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortSigItem) Less(i, j int) bool {
	_, _, _, iCount := a[i].Pub.(*bls.Blspub).GetBitID().GetBasicInfo()
	_, _, _, jCount := a[j].Pub.(*bls.Blspub).GetBitID().GetBasicInfo()
	return iCount < jCount
}

type SortBlsItem []*blsSigItem

func (a SortBlsItem) Len() int      { return len(a) }
func (a SortBlsItem) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortBlsItem) Less(i, j int) bool {
	_, _, _, iCount := a[i].pub.GetBitID().GetBasicInfo()
	_, _, _, jCount := a[j].pub.GetBitID().GetBasicInfo()
	return iCount < jCount
}

func (smm *blsSigState) gotMsg(sm *sig.MultipleSignedMessage) error {
	if sm.Index.Index != smm.index.Index {
		panic(1)
	}
	item, _, err := smm.getSignedMsgState(sm)
	if err != nil {
		return err
	}

	// We sort so that the biggest sigs are first
	sigItems := sm.GetSigItems()
	sort.Sort((SortSigItem)(sigItems))

	item.cond.L.Lock()

	// n := atomic.AddInt32(&gotMsgCount, 1)

	var retry int
tryValidate:

	// TODO remove me?
	if config.StopMultiSigEarly {
		smm.Lock()
		_, _, allCount, _ := item.allSigs.GetBasicInfo()
		_, _, validatingCount, _ := item.validatingSigs.GetBasicInfo()
		if allCount > smm.numPubs || validatingCount > smm.numPubs {
			item.cond.L.Unlock()
			smm.Unlock()
			return types.ErrNoValidSigs
		}
		smm.Unlock()
	}

	var newItems []*sig.SigItem
	imValidating := item.pool.Get()

	for i, si := range sigItems {
		// str, err := si.Pub.GetPubID()
		// if err != nil {
		// 	panic(err)
		// }

		// we only keep the sig if it gives information for at least 1 new sig
		// TODO also keep sig if it contains more members
		pbid := si.Pub.(sig.MultiPub).GetBitID()
		newSigs := bitid.GetNewItemsHelper(item.allSigs, pbid, item.pool.Get)
		// TODO check this???

		if bitid.HasNewItemsHelper(imValidating, newSigs) {
			// check if someone is already validating these sigs
			// if len(item.validatingSigs.GetNewItems(pbid)) == 0 {
			item.pool.Done(newSigs)

			if bitid.HasIntersectionHelper(item.validatingSigs, pbid) {
				// conflicting validate, retry
				// item.cond.Signal()
				retry++
				item.cond.Wait()
				sigItems = append(newItems, sigItems[i:]...)
				item.pool.Done(imValidating)
				goto tryValidate
			}
			newItems = append(newItems, si)

			imValidating, err = bitid.AddHelper(imValidating, pbid, false, false, item.pool.Get, item.pool.Done)
			utils.PanicNonNil(err)
			// bitid.AddBitIDType(imValidating, pbid, false)
			// imValidating = sig.MergeBitIDType(imValidating, pbid, false)
		} else {
			// logging.Info("no new sigs", pbid.GetItemList(), item.allSigs.GetItemList(), imValidating.GetItemList())
		}
	}

	// Let the oters know what you are validating
	if len(newItems) > 0 {
		item.validatingSigs, err = bitid.AddHelper(item.validatingSigs, imValidating,
			false, false, item.pool.Get, item.pool.Done)
		utils.PanicNonNil(err)
		// bitid.AddBitIDType(item.validatingSigs, imValidating, false)
		// item.validatingSigs = sig.MergeBitIDType(item.validatingSigs, imValidating, false)
	}

	item.pool.Done(imValidating)
	item.cond.L.Unlock()

	sm.SetSigItems(newItems)
	if len(newItems) == 0 {
		return types.ErrNoValidSigs
	}
	return nil
}

func (smm *blsSigState) storeMsg(sm *sig.MultipleSignedMessage, invalidSigs []*sig.SigItem, mc *consinterface.MemCheckers) (int, int, error) {
	if sm.Index.Index != smm.index.Index {
		panic(1)
	}
	item, itemID, err := smm.getSignedMsgState(sm)
	if err != nil {
		// item must exist since we created it in gotmsg
		panic(err)
	}
	validSigs := sm.GetSigItems()
	if len(validSigs) == 0 && len(invalidSigs) == 0 {
		panic(sm)
	}
	newValidSigs := make([]*sig.SigItem, 0, len(validSigs))
	item.cond.L.Lock()
	itemID.Lock()

	if config.StopMultiSigEarly {
		smm.Lock()
		smm.numPubs = mc.MC.GetMemberCount() - mc.MC.GetFaultCount() + 1 // TODO what should this number be??
		smm.Unlock()
	}

	for _, si := range validSigs {
		bid := si.Pub.(sig.MultiPub).GetBitID()
		// indicate we are no longer validating this
		item.validatingSigs, err = bitid.SubHelper(item.validatingSigs, bid, bitid.NotSafe, item.pool.Get, item.pool.Done)
		utils.PanicNonNil(err)
		// item.validatingSigs = bitid.SubBitIDType(item.validatingSigs, bid)
		if bitid.HasNewItemsHelper(item.allSigs, bid) {
			// if item.allSigs.HasNewItems(bid) {
			item.hasNewSigs = 0
			// item.addSig(si.Pub.(*sig.Blspub), si.Sig.(*sig.Blssig), si.SigBytes, mc)

			// TODO DO this merge in single loop

			// update the state with the new sig
			newBid := bitid.GetNewItemsHelper(item.allSigs, bid, item.pool.Get)
			_, _, count, _ := newBid.GetBasicInfo()
			if count > 0 {
				// newBid := item.allSigs.GetNewItems(bid)
				//if newBid.GetNumItems() > 0 {
				item.allSigs, err = bitid.AddHelper(item.allSigs, newBid, false,
					false, item.pool.Get, item.pool.Done)
				utils.PanicNonNil(err)
				// bitid.AddBitIDType(item.allSigs, newBid, false)
				// item.allSigs = sig.MergeBitIDType(item.allSigs, newBid, false)
			}
			item.pool.Done(newBid)
			// update for the msg id
			newBid = bitid.GetNewItemsHelper(itemID.allSigs, bid, item.pool.Get)
			_, _, count, _ = newBid.GetBasicInfo()
			if count > 0 {
				// newBid = itemID.allSigs.GetNewItems(bid)
				// if newBid.GetNumItems() > 0 {
				itemID.allSigs, err = bitid.AddHelper(itemID.allSigs, newBid,
					false, false, item.pool.Get, item.pool.Done)
				utils.PanicNonNil(err)
				// bitid.AddBitIDType(itemID.allSigs, newBid, false)
				// itemID.allSigs = sig.MergeBitIDType(itemID.allSigs, newBid, false)
			}
			item.pool.Done(newBid)
			item.fullSigList = append(item.fullSigList, &blsSigItem{pub: si.Pub.(sig.AllMultiPub),
				sig: si.Sig.(sig.AllMultiSig), proof: si.VRFProof, sigBytes: &si.SigBytes})

			// add it to the list of valids
			newValidSigs = append(newValidSigs, si)
		}
	}
	sort.Sort(item.fullSigList)
	for _, invalid := range invalidSigs {
		// indicate we are no longer validating this
		item.validatingSigs, err = bitid.SubHelper(item.validatingSigs,
			invalid.Pub.(sig.MultiPub).GetBitID(), bitid.NonIntersecting, item.pool.Get, item.pool.Done)
		utils.PanicNonNil(err)
		// item.validatingSigs = bitid.SubBitIDType(item.validatingSigs, invalid.Pub.(sig.MultiPub).GetBitID())
	}
	_, _, ret, _ := item.allSigs.GetBasicInfo()
	_, _, ret2, _ := itemID.allSigs.GetBasicInfo()
	// ret := item.allSigs.GetNumItems()
	// ret2 := itemID.allSigs.GetNumItems()

	itemID.Unlock()
	item.cond.L.Unlock()
	// Let anyone know who was waiting to validate that you are done
	item.cond.Broadcast()
	sm.SetSigItems(newValidSigs)
	if len(newValidSigs) == 0 {
		return 0, 0, types.ErrNoValidSigs
	}
	return ret, ret2, nil
}

func (smm *blsSigState) getSigCountMsgIDList(msgID messages.MsgID) []consinterface.MsgIDCount {
	smm.Lock()
	defer smm.Unlock()

	ret := make([]consinterface.MsgIDCount, 0, 2)
	for _, item := range smm.msgMap {
		if item.msgHeader.GetMsgID() == msgID {
			item.cond.L.Lock()
			_, _, count, _ := item.allSigs.GetBasicInfo()
			ret = append(ret, consinterface.MsgIDCount{MsgHeader: item.msgHeader, Count: count})
			item.cond.L.Unlock()
		}
	}
	return ret
}

func (smm *blsSigState) getSigCountMsg(hash types.HashStr) int {
	item, err := smm.getSignedMsgStateByHash(hash)
	if err == types.ErrMsgNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}
	item.cond.L.Lock()
	_, _, ret, oth := item.allSigs.GetBasicInfo()
	if ret != oth {
		panic(fmt.Sprint(ret, oth))
	}
	// ret := item.allSigs.GetNumItems()
	item.cond.L.Unlock()
	return ret
}

func (smm *blsSigState) getSigCountMsgID(msgID messages.MsgID) int {
	item, err := smm.getSignedMsgStateByMsgID(msgID)
	if err == types.ErrMsgNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}
	item.Lock()
	_, _, ret, _ := item.allSigs.GetBasicInfo()
	item.Unlock()
	return ret
}

func (smm *blsSigState) setupSigs(sm *sig.MultipleSignedMessage, priv sig.Priv, generateMySig bool,
	myVrf sig.VRFProof, addOtherSigsCount int, mc *consinterface.MemCheckers) (bool, error) {

	if sm.Index.Index != smm.index.Index {
		panic(1)
	}
	sigms, _, err := smm.getSignedMsgState(sm)
	if err != nil {
		return false, err
	}
	var sigItems []*sig.SigItem
	if generateMySig {
		mc.MC.GetStats().SignedItem()
		mySig, err := priv.GenerateSig(sm, myVrf, sm.GetSignType())
		if err != nil {
			return false, err
		}
		sigItems = append(sigItems, mySig)
		// blsitem := sigms.addSig(mySig.Pub.(*sig.Blspub), mySig.Sig.(*sig.Blssig), mySig.SigBytes, mc)
		// sigItems = append(sigItems, &sig.SigItem{blsitem.pub, blsitem.sig, *blsitem.sigBytes})
	}
	maxCount := smm.getSigCountMsg(sm.GetHashString())
	sigms.cond.L.Lock()

	if addOtherSigsCount > 0 { //|| generalconfig.IncludeCurrentSigs {
		// sigItems = make([]*sig.SigItem, len(sigms.nonDuplicateSigs))
		if config.AllowMultiMerge {
			sigms.computeMergedSigsMultiple(addOtherSigsCount, mc)
		} else {
			sigms.computeMergedSigs(addOtherSigsCount, mc)
		}
		var i int
		for sigItem := range sigms.nonDuplicateSigs {

			sigItems = append(sigItems, &sig.SigItem{Pub: sigItem.pub, Sig: sigItem.sig,
				VRFProof: sigItem.proof, SigBytes: *sigItem.sigBytes})
			i++
		}
	}

	if len(sigItems) == 0 {
		panic(fmt.Sprint(sm, sm.GetBaseMsgHeader().GetID(), sm.Index, smm.index))
	}
	sm.SetSigItems(sigItems)

	sigCount := sm.GetSigCount()

	if addOtherSigsCount > 0 {
		if sigCount < maxCount && sigCount < addOtherSigsCount {
			panic(fmt.Sprint(sigCount, maxCount, addOtherSigsCount)) // sanity check
		}
	}
	if sigCount < addOtherSigsCount {
		err = types.ErrNotEnoughSigs
	}
	sigms.cond.L.Unlock()

	return generateMySig, err
}

func (ss *blsSigMsgState) checkSubSig(item *blsSigItem) {
	// TODO can this remove too many?
	var didSub bool
	for true {
		_, dups := utils.GetDuplicates(item.pub.GetBitID().GetItemList())
		if len(dups) == 0 {
			break
		}
		dupBid := ss.intFunc(dups)

		for _, nxt := range ss.fullSigList {
			_, _, nxtCount, _ := nxt.pub.GetBitID().GetBasicInfo()
			if bitid.HasIntersectionCountHelper(dupBid, nxt.pub.GetBitID()) == nxtCount {
				// if len(dupBid.CheckIntersection(nxt.pub.GetBitID())) == nxt.pub.GetBitID().GetNumItems() {
				// panic(1)
				// we can sub
				newPub, err := item.pub.SubMultiPub(nxt.pub)
				if err != nil {
					panic(err)
				}
				newSig, err := item.sig.SubSig(nxt.sig)
				if err != nil {
					panic(err)
				}
				item.pub = newPub.(sig.AllMultiPub)
				item.sig = newSig.(sig.AllMultiSig)
				didSub = true
				// we did a sub so we loop again from the beginning
				break
			}
		}
		// we didn't do any subs this loop so we are done
		break
	}
	if didSub {
		sigBuff, err := item.pub.GenerateSerializedSig(item.sig)
		if err != nil {
			panic(err)
		}
		item.sigBytes = &sigBuff
	}
}

// computeMergedSigsMultiple will merge sigs into 1 valid sig so that all received addresses are covered
// (allows a single sig to be merge multiple times)
func (ss *blsSigMsgState) computeMergedSigsMultiple(maxSigCount int, mc *consinterface.MemCheckers) {
	// if we have no sigs then just return
	_, _, allCount, _ := ss.allSigs.GetBasicInfo()
	if allCount == 0 || ss.hasNewSigs >= maxSigCount {
		return
	}
	ss.hasNewSigs = maxSigCount
	// Loop until we cover all sigs
	soFar := ss.pool.Get()
	var sigItemList []*blsSigItem
	var toCheck bitid.NewBitIDInterface
	var err error
	if config.StartWithNonConflictingMultiSig {
		toCheck, err = bitid.SubHelper(ss.allSigs, soFar, bitid.NotSafe, ss.pool.Get, nil)
		utils.PanicNonNil(err)
		// toCheck = bitid.SubBitIDType(ss.allSigs, soFar) // to check is all sigs
		item := ss.computeNonConflitingSig(toCheck, mc)
		sigItemList = append(sigItemList, item)
		soFar, err = bitid.AddHelper(soFar, item.pub.GetBitID(), false, false, ss.pool.Get, ss.pool.Done)
		utils.PanicNonNil(err)
		// bitid.AddBitIDType(soFar, item.pub.GetBitID(), false)
		// soFar = sig.MergeBitIDType(soFar, item.pub.GetBitID(), false)
		ss.pool.Done(toCheck)
	}

	var toCheckCount, soFarCount int
	toCheck, toCheckCount, soFarCount = ss.getToCheck(soFar)
	for toCheckCount > 0 && soFarCount < maxSigCount {
		// for toCheck = bitid.SubBitIDType(ss.allSigs, soFar); toCheck.GetNumItems() > 0 && soFar.GetNumItems() < maxSigCount; toCheck = bitid.SubBitIDType(ss.allSigs, soFar) {

		// find the sig that covers the most
		var item *blsSigItem
		var itemIntersect int
		for _, nxt := range ss.fullSigList {
			intersect := bitid.HasIntersectionCountHelper(toCheck, nxt.pub.GetBitID())
			// intersect := len(toCheck.CheckIntersection(nxt.pub.GetBitID()))
			if intersect > itemIntersect {
				item = nxt
				itemIntersect = intersect
			}
		}
		if item == nil {
			panic("should not be nil")
		}
		sigItemList = append(sigItemList, item)
		soFar, err = bitid.AddHelper(soFar, item.pub.GetBitID(), false, false, ss.pool.Get, ss.pool.Done)
		utils.PanicNonNil(err)
		// bitid.AddBitIDType(soFar, item.pub.GetBitID(), false)
		// soFar = sig.MergeBitIDType(soFar, item.pub.GetBitID(), false)
		ss.pool.Done(toCheck)
		toCheck, toCheckCount, soFarCount = ss.getToCheck(soFar)
	}
	ss.pool.Done(toCheck)
	ss.pool.Done(soFar)

	// Make the merged sig
	newItem := ss.createMergedSig(sigItemList, mc)

	// _, dups := utils.GetDuplicates(newItem.pub.GetBitID().GetItemList())
	// n := atomic.AddInt32(&dupNum, int32(len(dups)))
	// c := atomic.AddInt32(&dupCount, 1)

	if config.AllowSubMultSig {
		ss.checkSubSig(newItem)
	}

	ss.updateMergedSigState([]*blsSigItem{newItem})
}

func (ss *blsSigMsgState) getToCheck(soFar bitid.NewBitIDInterface) (
	bitid.NewBitIDInterface, int, int) {

	toCheck, err := bitid.SubHelper(ss.allSigs, soFar, bitid.NotSafe, ss.pool.Get, nil)
	utils.PanicNonNil(err)
	_, _, toCheckCount, _ := toCheck.GetBasicInfo()
	_, _, _, soFarCount := soFar.GetBasicInfo()
	return toCheck, toCheckCount, soFarCount
}

func (ss *blsSigMsgState) computeNonConflitingSig(toCheck bitid.NewBitIDInterface, mc *consinterface.MemCheckers) *blsSigItem {
	// Find the group of mergeable sigs that covers the most of the remaining bits (toCheck)
	toMerge := ss.pool.Get()
	found := true
	var internalItemList []*blsSigItem // list of mergeable sigs
	for found {

		var item *blsSigItem
		var itemIntersect int
		found = false
		for _, nxt := range ss.fullSigList {
			if !bitid.HasIntersectionHelper(toMerge, nxt.pub.GetBitID()) {
				// if !toMerge.HasIntersection(nxt.pub.GetBitID()) {
				intersect := bitid.HasIntersectionCountHelper(toCheck, nxt.pub.GetBitID())
				// intersect := len(toCheck.CheckIntersection(nxt.pub.GetBitID()))
				if intersect > itemIntersect {
					item = nxt
					itemIntersect = intersect
				}
			}
		}
		if item != nil {
			internalItemList = append(internalItemList, item)
			var err error
			toMerge, err = bitid.AddHelper(toMerge, item.pub.GetBitID(), false,
				false, ss.pool.Get, ss.pool.Done)
			utils.PanicNonNil(err)
			// bitid.AddBitIDType(toMerge, item.pub.GetBitID(), false)
			// toMerge = sig.MergeBitIDType(toMerge, item.pub.GetBitID(), false)
			found = true
		}
	}
	ss.pool.Done(toMerge)

	// Add it to the list of items found so far
	newItem := ss.createMergedSig(internalItemList, mc)
	return newItem
}

// computeMergedSigs will merge the sigs into a minimum set of sigs (using a greedy approach) so that all received addresses are covered
// (does not allow a single sig to be merged multiple times)
// TODO make this concurrent with GotMsg and StoreMsg?
func (ss *blsSigMsgState) computeMergedSigs(maxSigCount int, mc *consinterface.MemCheckers) {
	_, _, count, _ := ss.allSigs.GetBasicInfo()
	if count == 0 || ss.hasNewSigs >= maxSigCount {
		return
	}
	ss.hasNewSigs = maxSigCount
	// Loop until we cover all sigs
	soFar := ss.pool.Get()
	var sigItemList []*blsSigItem
	toCheck, toCheckCount, soFarCount := ss.getToCheck(soFar)
	for toCheckCount > 0 && soFarCount < maxSigCount {
		// for toCheck := bitid.SubBitIDType(ss.allSigs, soFar); toCheck.GetNumItems() > 0 && soFar.GetNumItems() < maxSigCount; toCheck = bitid.SubBitIDType(ss.allSigs, soFar) {

		// // Find the group of mergeable sigs that covers the most of the remaining bits (toCheck)
		// toMerge, err := sig.CreateBitIDTypeFromInts(nil)
		// if err != nil {
		// 	panic(err)
		// }
		// found := true
		// var internalItemList []*BlsSigItem // list of mergeable sigs
		// for found {
		// 	var item *BlsSigItem
		// 	var itemIntersect int
		// 	found = false
		// 	for _, nxt := range ss.fullSigList {
		// 		if len(toMerge.CheckIntersection(nxt.pub.GetBitID())) == 0 {
		// 			intersect := len(toCheck.CheckIntersection(nxt.pub.GetBitID()))
		// 			if intersect > itemIntersect {
		// 				item = nxt
		// 				itemIntersect = intersect
		// 			}
		// 		}
		// 	}
		// 	if item != nil {
		// 		internalItemList = append(internalItemList, item)
		// 		toMerge = sig.MergeBitIDType(toMerge, item.pub.GetBitID(), false)
		// 		found = true
		// 	}
		// }

		// // Add it to the list of items found so far
		// newItem := ss.createMergedSig(internalItemList, mc)
		newItem := ss.computeNonConflitingSig(toCheck, mc)

		sigItemList = append(sigItemList, newItem)
		var err error
		soFar, err = bitid.AddHelper(soFar, newItem.pub.GetBitID(), false, false, ss.pool.Get, ss.pool.Done)
		utils.PanicNonNil(err)
		// bitid.AddBitIDType(soFar, newItem.pub.GetBitID(), false)
		// soFar = sig.MergeBitIDType(soFar, newItem.pub.GetBitID(), false)
		ss.pool.Done(toCheck)
		toCheck, toCheckCount, soFarCount = ss.getToCheck(soFar)
	}
	ss.pool.Done(toCheck)
	ss.pool.Done(soFar)
	ss.updateMergedSigState(sigItemList)
}

func (ss *blsSigMsgState) createMergedSig(internalItemList []*blsSigItem, mc *consinterface.MemCheckers) *blsSigItem {

	_ = mc

	// Create the merged sig from this group
	comSig := internalItemList[0].sig
	comSigBuff := *internalItemList[0].sigBytes
	comPub := internalItemList[0].pub
	var err error

	// TODO merge at once1!!!

	for _, nxtSigItem := range internalItemList[1:] {
		newComSig, err := comSig.MergeSig(nxtSigItem.sig)
		if err != nil {
			panic(err)
		}
		comSig = newComSig.(sig.AllMultiSig)
		// smc := mc.GetSpecialMemberChecker().(*MultiSigMemChecker)

		// newPub := smc.getPub(sig.PubKeyID(sig.MergeBitIDType(comPub.GetBitID(), nxtSigItem.pub.GetBitID(), true).GetStr()))
		//if newPub == nil {
		newComPub, err := comPub.MergePub(nxtSigItem.pub)
		if err != nil {
			panic(err)
		}
		comPub = newComPub.(sig.AllMultiPub)
		// So we generate the bitID
		// _, err := newPub.InformState(nil)
		// if err != nil {
		// 	panic(err)
		// }
		// smc.addPub(newPub)
		//}
		// comPub = newPub
	}
	// Create the sig bytes if needed
	if len(internalItemList) > 1 {
		comSigBuff, err = comPub.GenerateSerializedSig(comSig)
		if err != nil {
			panic(err)
		}
	}

	newItem := &blsSigItem{pub: comPub, sig: comSig, sigBytes: &comSigBuff} // no VRFProof for merged signature
	// If we did a merge then we created a new sigs and should add it to our list
	if len(internalItemList) > 1 {
		ss.fullSigList = append(ss.fullSigList, newItem)
	}
	return newItem
}

func (ss *blsSigMsgState) updateMergedSigState(sigItemList []*blsSigItem) {

	//TODO this is an expensive loop

	// sanity check
	if config.AllowMultiMerge && len(sigItemList) != 1 {
		panic(len(sigItemList))
	}

	// Update the local state with the new merged sigs
	sigIdx := make(map[int]bool)
	for _, newSigItem := range sigItemList {
		// Update the added sigs
		// newLen := newSigItem.pub.GetBitID().GetNumItems()
		bid := newSigItem.pub.GetBitID()
		iter := bid.NewIterator()
		for i, err := iter.NextID(); err == nil; i, err = iter.NextID() {
			sigIdx[i] = true
			// for _, i := range ss.allSigs.GetItemList() {

			// should we replace the old signature with the new one?
			// always in the case of generalconfig.AllowMultiMerge, because it covers all signautres
			// TODO, otherwise when generalconfig.AllowMultiMerge is false can there be a better option??
			// if generalconfig.AllowMultiMerge || !ok || item.pub.GetBitID().GetNumItems() <= newLen {
			// item, ok := ss.sigs[i]
			ss.sigs[i] = newSigItem
			// item = &newSigItem
			// }
		}
		iter.Done()
	}

	// Update the map of non-duplicate sigs
	nonDuplicateSigs := make(map[blsSigItem]bool, len(ss.nonDuplicateSigs))
	// for _, i := range ss.allSigs.GetItemList() {
	for i := range sigIdx {
		item, ok := ss.sigs[i]
		if !ok {
			panic("should be found")
		}
		nonDuplicateSigs[*item] = true
	}
	// sanity check
	if config.AllowMultiMerge && len(nonDuplicateSigs) != 1 {
		panic(len(nonDuplicateSigs))
	}
	// nonDuplicateSigs[newSigItem] = true
	ss.nonDuplicateSigs = nonDuplicateSigs
}

// old way of combining sigs, every time a sig is received
// new way is the computeMergedSigs, computed each time a message is sent
// func (ss *BlsSigMsgState) addSig(pub *sig.Blspub, bsig *sig.Blssig, sigBytes []byte, mc MemberChecker) BlsSigItem {

//	var maxCombine *BlsSigItem
//	var maxCombineCount int

//	for _, nxt := range ss.sigs {
//	// for _, nxt := range ss.fullSigList {
//		// we only merge with non-conflicting
//		// TODO compute this better
//		if len(nxt.pub.GetBitID().CheckIntersection(pub.GetBitID())) == 0 {
//			combineCount := nxt.pub.GetBitID().GetNumItems()
//			if combineCount > maxCombineCount {
//				maxCombineCount = combineCount
//				maxCombine = nxt
//			}
//		}
//	}

//	var newSigItem BlsSigItem
//	if maxCombineCount > 0 {
//		comSig, err := sig.MergeBlsSig(bsig, maxCombine.sig)
//		if err != nil {
//			panic(err)
//		}
//		smc := mc.GetSpecialMemberChecker().(*MultiSigMemChecker)
//		comPub := smc.getPub(sig.MergeBitID(pub.GetBitID(), maxCombine.pub.GetBitID()).Str)
//		if comPub == nil {
//			comPub, err = sig.MergeBlsPub(pub, maxCombine.pub)
//			if err != nil {
//				panic(err)
//			}
//			// So we generate the bitID
//			_, err := comPub.InformState(nil)
//			if err != nil {
//				panic(err)
//			}
//			smc.addPub(comPub)
//		}
//		m, err := comPub.GenerateSerializedSig(comSig)
//		if err != nil {
//			panic(err)
//		}
//		newSigItem = BlsSigItem{comPub, comSig, &m}
//	} else {
//		newSigItem = BlsSigItem{pub, bsig, &sigBytes}
//	}
//	ss.allSigs = sig.MergeBitID(ss.allSigs, newSigItem.pub.GetBitID())

//	newLen := newSigItem.pub.GetBitID().GetNumItems()

//	// Update the added sigs
//	for _, i := range newSigItem.pub.GetBitID().GetItemList() {
//		// for _, i := range ss.allSigs.GetItemList() {
//		item, ok := ss.sigs[i]
//		if !ok || item.pub.GetBitID().GetNumItems() < newLen {
//			ss.sigs[i] = &newSigItem
//			// item = &newSigItem
//		}
//	}

//	nonDuplicateSigs := make(map[BlsSigItem]bool, len(ss.nonDuplicateSigs))
//	for _, i := range ss.allSigs.GetItemList() {
//		item, ok := ss.sigs[i]
//		if !ok {
//			panic("should be found")
//		}
//		nonDuplicateSigs[*item] = true
//	}
//	// nonDuplicateSigs[newSigItem] = true
//	ss.nonDuplicateSigs = nonDuplicateSigs

//	return newSigItem
// }
