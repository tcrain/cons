package consinterface

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/dual"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/types"
	"sync"
)

type Shared struct {
	PubKeys   sig.PubList    // The sorted list of public keys
	DSSShared *ed.CoinShared // Information for threshold keys
	BlsShare  *bls.BlsShared // state of Shared generated keys when using threshold BLS
	BlsShare2 *bls.BlsShared // state of Shared generated keys when using threshold BLS

	didInitSetup, didSetup bool

	coord                    sig.Pub
	newMembers, newOtherPubs []sig.Pub
	memberMap                map[sig.PubKeyID]sig.Pub
	allPubs                  []sig.Pub
	staticNodeMaps           []map[sig.PubKeyStr]channelinterface.NetNodeInfo

	initMutex, mutex sync.Mutex
}

func (sh *Shared) GetStaticNodeMaps(t assert.TestingT, to types.TestOptions, allPubs []sig.Pub,
	parRegs ...network.PregInterface) []map[sig.PubKeyStr]channelinterface.NetNodeInfo {

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.staticNodeMaps == nil {
		genPub := func(pb sig.PubKeyBytes) sig.Pub {
			return network.GetPubFromList(pb, allPubs)
		}
		for _, parReg := range parRegs {
			allParticipants, err := parReg.GetAllParticipants()
			if len(allParticipants) != to.NumTotalProcs {
				panic(fmt.Sprint(allParticipants, to.NumTotalProcs))
			}
			assert.Nil(t, err)
			nxt := make(map[sig.PubKeyStr]channelinterface.NetNodeInfo)
			sh.staticNodeMaps = append(sh.staticNodeMaps, nxt)
			for _, parInfo := range allParticipants {
				nodeInfo := channelinterface.NetNodeInfo{
					AddrList: parInfo.ConInfo,
					Pub:      genPub(sig.PubKeyBytes(parInfo.Pub))}
				ps, err := nodeInfo.Pub.GetPubString()
				if err != nil {
					panic(err)
				}
				nxt[ps] = nodeInfo
			}
		}
	}

	return sh.staticNodeMaps
}

func (sh *Shared) InitialKeySetup(to types.TestOptions, parReg network.ParRegClientInterface) error {
	sh.initMutex.Lock()
	defer sh.initMutex.Unlock()

	if sh.didInitSetup {
		return nil
	}
	sh.didInitSetup = true
	sleepCrypto := to.SleepCrypto

	var err error
	switch to.SigType {
	case types.EDCOIN:
		if sleepCrypto {
		} else {
			// setup the threshold keys
			// we use a special path because we need to agree on the Shared values
			sh.DSSShared, err = getDssShared(parReg)
			if err != nil {
				return err
			}
		}
	case types.TBLS:
		if sleepCrypto {
		} else {
			// setup the threshold keys
			// we use a special path because we need to agree on the Shared values
			// before creating the key
			// note that this is done in a centralized (unsafe) manner for efficiency
			// and we share all the private keys
			sh.BlsShare, err = getBlsShared(0, parReg)
			if err != nil {
				return err
			}
		}
	case types.TBLSDual:
		if sleepCrypto {
		} else {
			sh.BlsShare, err = getBlsShared(0, parReg)
			if err != nil {
				return err
			}
			sh.BlsShare2, err = getBlsShared(1, parReg)
			if err != nil {
				return err
			}
		}
	case types.CoinDual:
		if sleepCrypto {
		} else {
			sh.BlsShare, err = getBlsShared(0, parReg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (sh *Shared) AfterSortPubs(myPriv sig.Priv, fixedCoord sig.Pub, members sig.PubList,
	otherPubs sig.PubList) (newMyPriv sig.Priv, coord sig.Pub, newMembers,
	newOtherPubs []sig.Pub, memberMap map[sig.PubKeyID]sig.Pub, allPubs []sig.Pub) {

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.newOtherPubs == nil {
		_, sh.coord, sh.newMembers, sh.newOtherPubs, sh.memberMap, sh.allPubs = sig.AfterSortPubs(myPriv,
			fixedCoord, members, otherPubs)
	}
	coord, newMembers, otherPubs, memberMap, allPubs = sh.coord, sh.newMembers, sh.newOtherPubs, sh.memberMap, sh.allPubs

	myIndex := -1
	var myPstr sig.PubKeyBytes
	var err error
	myPstr, err = myPriv.GetPub().GetRealPubBytes()
	if err != nil {
		panic(err)
	}

	// Go through the members and compute their new indices
	for i, p := range sh.newMembers {
		var otherPstr sig.PubKeyBytes
		otherPstr, err = p.GetRealPubBytes()
		if err != nil {
			panic(err)
		}

		// Check my pub
		if bytes.Equal(myPstr, otherPstr) {
			if myIndex != -1 {
				panic("duplicate keys")
			}
			myIndex = i
		}
	}
	// Remaining nodes
	index := len(members)
	for _, p := range otherPubs {
		var otherPstr sig.PubKeyBytes
		otherPstr, err = p.GetRealPubBytes()
		if err != nil {
			panic(err)
		}
		// Check my pub
		if bytes.Equal(myPstr, otherPstr) {
			if myIndex != -1 {
				panic("duplicate keys")
			}
			myIndex = index
		}
		index++
	}
	if myIndex == -1 {
		panic("didnt find my pub")
	}

	// Set my new index
	newMyPriv = myPriv.ShallowCopy()
	newMyPriv.SetIndex(sig.PubKeyIndex(myIndex))

	return
}

func (sh *Shared) GetAllPubKeys(to types.TestOptions, privKey sig.Priv, parReg network.ParRegClientInterface,
) error {

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.didSetup {
		return nil
	}
	sh.didSetup = true

	// get all participants
	allPar, err := parReg.GetAllParticipants(0)
	if err != nil {
		return err
	}

	makeKeys := true
	switch to.SigType {
	case types.TBLS, types.TBLSDual, types.EDCOIN, types.CoinDual: // threshold/coin keys are constructed below
		makeKeys = false
	}
	if to.SleepCrypto { // For sleep crypto we use use the normal setup
		makeKeys = true
	}
	if makeKeys {
		for _, parInfo := range allPar {
			var pub sig.Pub
			pub, err = privKey.GetPub().New().FromPubBytes(sig.PubKeyBytes(parInfo.Pub))
			if err != nil {
				return err
			}
			sh.PubKeys = append(sh.PubKeys, pub)
		}
	}
	if !to.SleepCrypto {
		switch to.SigType {
		case types.EDCOIN:
			for i, nxtMem := range sh.DSSShared.MemberPoints {
				pub := ed.NewEdPartPub(sig.PubKeyIndex(i), nxtMem, ed.NewEdThresh(sig.PubKeyIndex(i), sh.DSSShared))
				sh.PubKeys = append(sh.PubKeys, pub)
			}
			for i, nxtNonMem := range sh.DSSShared.NonMemberPoints {
				numMembers := to.NumTotalProcs - to.NumNonMembers
				pub := ed.NewEdPartPub(sig.PubKeyIndex(i+numMembers),
					nxtNonMem, ed.NewEdThresh(sig.PubKeyIndex(i+numMembers), sh.DSSShared))
				sh.PubKeys = append(sh.PubKeys, pub)
			}
		case types.TBLS:
			// TBLS keys we construct directly from the BlsShared object
			for i := 0; i < to.NumTotalProcs; i++ {
				sh.PubKeys = append(sh.PubKeys, bls.NewBlsPartPub(sig.PubKeyIndex(i), sh.BlsShare.NumParticipants,
					sh.BlsShare.NumThresh, sh.BlsShare.PubPoints[i]))
			}
		case types.TBLSDual:
			priv := privKey.(*dual.DualPriv)
			for i := 0; i < to.NumTotalProcs; i++ {
				pub1 := bls.NewBlsPartPub(sig.PubKeyIndex(i), sh.BlsShare.NumParticipants,
					sh.BlsShare.NumThresh, sh.BlsShare.PubPoints[i])
				pub2 := bls.NewBlsPartPub(sig.PubKeyIndex(i), sh.BlsShare.NumParticipants,
					sh.BlsShare.NumThresh, sh.BlsShare2.PubPoints[i])
				dpub := priv.GetPub().New().(*dual.DualPub)
				dpub.SetPubs(pub1, pub2)
				sh.PubKeys = append(sh.PubKeys, dpub)
			}
		case types.CoinDual:
			priv := privKey.(*dual.DualPriv)
			for i, parInfo := range allPar {
				// we get the ec pub from the bytes
				dp, err := priv.GetPub().New().FromPubBytes(sig.PubKeyBytes(parInfo.Pub))
				if err != nil {
					return err
				}
				// we gen the bls pub from the share
				blsPub := bls.NewBlsPartPub(sig.PubKeyIndex(i), sh.BlsShare.NumParticipants,
					sh.BlsShare.NumThresh, sh.BlsShare.PubPoints[i])
				dpub := priv.GetPub().New().(*dual.DualPub)
				dpub.SetPubs(dp.(*dual.DualPub).Pub, blsPub)
				sh.PubKeys = append(sh.PubKeys, dpub)
			}
		}
	}
	for i, nxt := range sh.PubKeys {
		nxt.SetIndex(sig.PubKeyIndex(i))
	}
	return nil
}

func getDssShared(preg network.ParRegClientInterface) (*ed.CoinShared, error) {
	dssMarshaled, err := preg.GetDSSShared(0)
	if err != nil || dssMarshaled == nil {
		return nil, types.ErrInvalidSharedThresh
	}
	return dssMarshaled.PartialUnMartial()
}

func getBlsShared(idx int, preg network.ParRegClientInterface) (*bls.BlsShared, error) {
	blssi, err := preg.GetBlsShared(0, idx)
	if err != nil || blssi.BlsShared == nil {
		return nil, types.ErrInvalidSharedThresh
	}
	ret, err := blssi.BlsShared.PartialUnmarshal()
	return ret, err
}
