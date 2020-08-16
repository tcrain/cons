package rpcsetup

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/dual"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/types"
	"sync"
)

type Shared struct {
	PubKeys   sig.PubList    // The sorted list of public keys
	DSSShared *ed.CoinShared // Information for threshold keys
	BlsShare  *bls.BlsShared // state of Shared generated keys when using threshold BLS
	BlsShare2 *bls.BlsShared // state of Shared generated keys when using threshold BLS

	didInitSetup, didSetup bool

	initMutex, mutex sync.Mutex
}

func (sh *Shared) initialKeySetup(to types.TestOptions, parReg ParRegClientInterface) error {
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

func (sh *Shared) getAllPubKeys(to types.TestOptions, privKey sig.Priv, parReg ParRegClientInterface,
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
