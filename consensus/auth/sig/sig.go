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

package sig

import (
	"bytes"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

////////////////////////////////////////////////////////////
// Interfaces
////////////////////////////////////////////////////////////

// SigItem tracks a signature and public key object that should verify the signature
type SigItem struct {
	WasEncrypted bool     // Set to true if this was an encrypted message
	Pub          Pub      // The public key that will be used to verify the signature
	Sig          Sig      // The signature
	VRFProof     VRFProof // The VRF proof if enabled
	VRFID        uint64   // The first uint64 computed by the vrf
	SigBytes     []byte   // The bytes of the signature
	//CoinProof    CoinProof // Proof for the coin
}

//////////////////////////////////////////////////////////////////////////
// Sorting keys
/////////////////////////////////////////////////////////////////////////

type PubKeyStrList []PubKeyStr

/*// SortPubKeyStr is used to create a list of sorted pub key strings
type SortPubKeyStr []PubKeyStr

func (a SortPubKeyStr) Len() int      { return len(a) }
func (a SortPubKeyStr) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortPubKeyStr) Less(i, j int) bool {
	return a[i] < a[j]
}
*/

func CheckPubsEqual(a, b Pub) bool {
	pub1, err := a.GetPubString()
	if err != nil {
		panic(err)
	}
	pub2, err := b.GetPubString()
	if err != nil {
		panic(err)
	}
	return pub1 == pub2
}

func CheckPubsIDEqual(a, b Pub) bool {
	pub1, err := a.GetPubID()
	if err != nil {
		panic(err)
	}
	pub2, err := b.GetPubID()
	if err != nil {
		panic(err)
	}
	return pub1 == pub2
}

// SortPub is uses to create a sorted list of public keys
type SortPub []Pub

func (a SortPub) Len() int      { return len(a) }
func (a SortPub) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortPub) Less(i, j int) bool {
	// Sorting is done by comparing the result of GetPubString
	pub1, err := a[i].GetPubString()
	if err != nil {
		return true
	}
	pub2, err := a[j].GetPubString()
	if err != nil {
		return false
	}
	return pub1 < pub2
}

// SortPriv is private keys sorted by pub bytes
type SortPriv []Priv

func (a SortPriv) Len() int      { return len(a) }
func (a SortPriv) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortPriv) Less(i, j int) bool {
	pub1, err := a[i].GetPub().GetPubString()
	if err != nil {
		return true
	}
	pub2, err := a[j].GetPub().GetPubString()
	if err != nil {
		return false
	}
	return pub1 < pub2
}

func GetPubID(idx PubKeyIndex, pub Pub) PubKeyID {
	if UsePubIndex {
		buff := make([]byte, 4)
		config.Encoding.PutUint32(buff, uint32(idx))
		return PubKeyID(buff)
	}
	pid, err := pub.GetPubString()
	if err != nil {
		panic(err)
	}
	return PubKeyID(pid)
}

type PrivList []Priv
type PubList []Pub

// SetIndices should be called after sorting has completed to let the keys know their indecies
func (a PrivList) SetIndices() {
	for i, priv := range a {
		priv.SetIndex(PubKeyIndex(i))
	}
}

// AfterSortPubs should be called after sorting has completed to let the keys know their indecies
func AfterSortPubs(myPriv Priv, fixedCoord Pub, members PubList,
	otherPubs PubList) (newMyPriv Priv, coord Pub, newMembers, newOtherPubs []Pub, memberMap map[PubKeyID]Pub) {

	newMembers = make([]Pub, len(members), len(members)+1)
	newOtherPubs = make([]Pub, 0, len(otherPubs))

	var foundCord, coordMember bool // if we have a fixed coord we need to be sure it is a member
	if fixedCoord == nil {          // we don't have to check for a fixed coord as a member if nil
		foundCord = true
		coordMember = true
	}
	memberMap = make(map[PubKeyID]Pub, len(newMembers)+1)

	// We have to find our own pub key as well so we can update our own index
	myIndex := -1
	var myPstr PubKeyBytes
	var err error
	myPstr, err = myPriv.GetPub().GetRealPubBytes()
	if err != nil {
		panic(err)
	}
	var cordPStr PubKeyBytes
	if fixedCoord != nil {
		cordPStr, err = fixedCoord.GetRealPubBytes()
		if err != nil {
			panic(err)
		}
	}

	// Go through the members and compute their new indices
	for i, p := range members {
		var otherPstr PubKeyBytes
		otherPstr, err = p.GetRealPubBytes()
		if err != nil {
			panic(err)
		}
		// Make the new pub with the new index
		newPub := p.ShallowCopy()
		newPub.SetIndex(PubKeyIndex(i))
		pid, err := newPub.GetPubID()
		if err != nil {
			panic(err)
		}

		// Check coord pub
		if fixedCoord != nil && bytes.Equal(cordPStr, otherPstr) {
			if foundCord {
				panic("duplicate")
			}
			foundCord = true
			coordMember = true
			coord = newPub
		}

		// Add it to the list
		memberMap[pid] = newPub
		newMembers[i] = newPub

		// Check my pub
		if bytes.Equal(myPstr, otherPstr) {
			if myIndex != -1 {
				panic("duplicate keys")
			}
			myIndex = i
		}
	}
	// Remaining nodes
	var index int
	switch foundCord {
	case true:
		index = len(members)
	default:
		index = len(members) + 1 // we will add the coord at the end of the member list
	}
	for _, p := range otherPubs {
		var otherPstr PubKeyBytes
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

		// Check coord pub
		if fixedCoord != nil && bytes.Equal(cordPStr, otherPstr) {
			if foundCord {
				panic("duplicate")
			}
			foundCord = true
			// Make the new pub with the new index
			newPub := p.ShallowCopy()
			newPub.SetIndex(PubKeyIndex(len(members)))
			pid, err := newPub.GetPubID()
			if err != nil {
				panic(err)
			}
			memberMap[pid] = newPub
			coord = newPub
			newMembers = append(newMembers, coord)
		} else {
			// Make the new pub with the new index
			newPub := p.ShallowCopy()
			newPub.SetIndex(PubKeyIndex(index))
			_, err := newPub.GetPubID()
			if err != nil {
				panic(err)
			}
			newOtherPubs = append(newOtherPubs, newPub)
			index++
		}
	}

	// Set my new index
	newMyPriv = myPriv.ShallowCopy()
	newMyPriv.SetIndex(PubKeyIndex(myIndex))

	// Sanity checks
	var addCoord int
	if !coordMember {
		addCoord++
	}

	if len(newMembers) != len(members)+addCoord || len(memberMap) != len(members)+addCoord ||
		len(newOtherPubs) != len(otherPubs)-addCoord || len(members)+len(otherPubs) != len(newMembers)+len(newOtherPubs) {

		panic("didn't add all pubs")
	}
	if UsePubIndex {
		var i int
		for _, nxt := range newMembers {
			if PubKeyIndex(i) != nxt.GetIndex() {
				panic("out of order")
			}
			i++
		}
		for _, nxt := range newOtherPubs {
			if PubKeyIndex(i) != nxt.GetIndex() {
				panic("out of order")
			}
			i++
		}
	}
	if myIndex < 0 || !foundCord {
		panic("missing pub keys")
	}

	return
}

// CheckSerVRF is a helper function used by implementations of GenerateSig
func CheckSerVRF(vrfProof VRFProof, m *messages.Message) error {
	// if we start with 1 then a VRFProof is included
	// 0 means no VRFProof
	if vrfProof != nil {
		(*messages.MsgBuffer)(m).AddByte(1)
		_, err := vrfProof.Encode((*messages.MsgBuffer)(m))
		if err != nil {
			return err
		}
	} else {
		(*messages.MsgBuffer)(m).AddByte(0)
	}
	return nil
}

// DeserVRF is a helper function used by implementations of DeserializeSig
func DeserVRF(pub Pub, m *messages.Message) (*SigItem, int, error) {
	var l int
	byt, err := (*messages.MsgBuffer)(m).ReadByte()
	l += 1
	if err != nil {
		return nil, l, err
	}
	ret := &SigItem{}
	switch byt {
	case 0: // no VRF proof
	case 1: // VRF proof included
		ret.VRFProof = pub.(VRFPub).NewVRFProof()
		l1, err := ret.VRFProof.Decode((*messages.MsgBuffer)(m))
		l += l1
		if err != nil {
			return nil, l, err
		}
	default: // should be 1 or 0
		return nil, l, types.ErrInvalidFormat
	}
	return ret, l, nil
}

// GenerateSigHelper is a helper function for implementations of GenerateSig
func GenerateSigHelper(priv Priv, header SignedMessage, allowsVRF bool, vrfProof VRFProof, signType types.SignType) (*SigItem, error) {
	if signType == types.CoinProof {
		panic(types.ErrCoinProofNotSupported)
	}
	m := messages.NewMessage(nil)
	if allowsVRF {
		if err := CheckSerVRF(vrfProof, m); err != nil {
			return nil, err
		}
	}
	_, err := priv.GetPub().Serialize(m) // priv.SerializePub(m)
	if err != nil {
		return nil, err
	}
	si, err := priv.Sign(header)
	if err != nil {
		return nil, err
	}
	_, err = si.Serialize(m)
	if err != nil {
		return nil, err
	}

	return &SigItem{
		VRFProof: vrfProof,
		Pub:      priv.GetPub(),
		Sig:      si,
		SigBytes: m.GetBytes()}, nil
}

// GenerateSigHelper is a helper function for implementations of GenerateSig
func GenerateSigHelperFromSig(pub Pub, sig Sig, vrfProof VRFProof, signType types.SignType) (*SigItem, error) {
	m := messages.NewMessage(nil)
	if err := CheckSerVRF(vrfProof, m); err != nil {
		return nil, err
	}
	_, err := pub.Serialize(m) // priv.SerializePub(m)
	if err != nil {
		return nil, err
	}
	_, err = sig.Serialize(m)
	if err != nil {
		return nil, err
	}

	return &SigItem{
		VRFProof: vrfProof,
		Pub:      pub,
		Sig:      sig,
		SigBytes: m.GetBytes()}, nil
}

func RemoveDuplicatePubs(pubs []Pub) (ret []Pub) {
	pubMap := make(map[PubKeyStr]Pub, len(pubs))
	for _, nxt := range pubs {
		pStr, err := nxt.GetPubString()
		if err != nil {
			panic(err)
		}
		pubMap[pStr] = nxt
	}
	ret = make([]Pub, 0, len(pubMap))
	for _, nxt := range pubMap {
		ret = append(ret, nxt)
	}
	return
}

func VerifySignature() {

}
