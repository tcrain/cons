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

package bls

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3/pairing"
	"golang.org/x/crypto/blake2b"
	"io"
	"sort"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/messages"
	"go.dedis.ch/kyber/v3"
)

// the EdSuite used by bls
var blssuite *pairing.SuiteBn256
var blsmarshalsize, blssigmarshalsize int

func init() {
	blssuite = pairing.NewSuiteBn256()
	blsmarshalsize = 128   //blssuite.PointLen()
	blssigmarshalsize = 64 //blssuite.ScalarLen()
}

type VRFProof []byte

func (prf VRFProof) Encode(writer io.Writer) (n int, err error) {
	return writer.Write(prf)
}
func (prf VRFProof) Decode(reader io.Reader) (n int, err error) {
	if len(prf) != blssigmarshalsize {
		panic("invalid alloc")
	}
	return reader.Read(prf)
}

func (prf VRFProof) New() sig.VRFProof {
	return make(VRFProof, blssigmarshalsize)
}

///////////////////////////////////////////////////////////////////////////////////////
// Operations on multi-sigs
///////////////////////////////////////////////////////////////////////////////////////

// SubBlsPub remove pub2 from pub1 and returns the resulting public key object
func (pub *Blspub) SubMultiPub(pub2 sig.MultiPub) (sig.MultiPub, error) {
	p1bid := pub.GetBitID()
	p2bid := pub2.GetBitID()

	nPbid, err := bitid.SubHelper(p1bid, p2bid, bitid.NonIntersectingAndEmpty, p1bid.New, nil)
	if err != nil {
		return nil, err
	}

	pb := pub.newPub.Clone().Sub(pub.newPub, pub2.(*Blspub).newPub)
	return &Blspub{
		newBidFunc: pub.newBidFunc,
		pub:        pb,
		newPub:     pb,
		bitID:      nPbid}, nil
}

// DonePartialMerge should be called after merging keys with MergePubPartial to set the bitid
func (pub *Blspub) DonePartialMerge(bid bitid.NewBitIDInterface) {
	pub.bitID = bid
}

// GenerateSerializedSig serialized the public key and the signature and returns the bytes
func (pub *Blspub) GenerateSerializedSig(bsig sig.MultiSig) ([]byte, error) {
	// hash := header.GetSignedHash()
	si, err := sig.GenerateSigHelperFromSig(pub, bsig.(*Blssig), nil, types.NormalSignature)
	if err != nil {
		return nil, err
	}
	return si.SigBytes, nil
}

// MergePubPartial only merges the pub itself, does not create the new bitid
func (pub *Blspub) MergePubPartial(pub2 sig.MultiPub) {
	pb := pub.newPub.Add(pub2.(*Blspub).newPub, pub.newPub)
	pub.pub = pb
	pub.newPub = pb
}

// MergeBlsPub combines two BLS public key objects into a single one
func (pub *Blspub) MergePub(pub2 sig.MultiPub) (sig.MultiPub, error) {
	p1bid := pub.GetBitID()
	p2bid := pub2.GetBitID()
	if p1bid == nil || p2bid == nil {
		return nil, types.ErrInvalidBitID
	}
	nBid, err := bitid.AddHelper(p1bid, p2bid, config.AllowMultiMerge, !config.AllowMultiMerge, p1bid.New, nil)
	if err != nil {
		return nil, err
	}

	pb := pub.newPub.Clone().Add(pub2.(*Blspub).newPub, pub.newPub)
	return &Blspub{
		newBidFunc: pub.newBidFunc,
		pub:        pb,
		newPub:     pb,
		bitID:      nBid}, nil
}

// GetBitID returns the bit id object representing the indecies of the nodes represented by the BLS public key oject
func (pub *Blspub) GetBitID() bitid.NewBitIDInterface {
	return pub.bitID
}

///////////////////////////////////////////////////////////////////////////////////////
// The below functions are for doing a merge in a loop (see func (msm *MultiSigMemChecker) CheckMemberLocalMsg(...))
///////////////////////////////////////////////////////////////////////////////////////

// Clone returns a new Blspub only containing the points (no bitid), should be called before merging the first set of keys with MergePubPartial
func (pub *Blspub) Clone() sig.MultiPub {
	return &Blspub{
		newBidFunc: pub.newBidFunc,
		pub:        pub.pub.Clone(),
		newPub:     pub.newPub.Clone()}
}

// NewVRFProof returns an empty VRFProof object
func (pub *Blspub) NewVRFProof() sig.VRFProof {
	return VRFProof(make([]byte, blssigmarshalsize))
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *Blspub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	signType := msg.GetSignType()
	if signType == types.CoinProof {
		return types.ErrCoinProofNotSupported
	}
	valid, err := pub.VerifySig(msg, sigItem.Sig)
	if err != nil {
		return err
	}
	if !valid {
		return types.ErrInvalidSig
	}
	return nil
}

// ProofToHash is for validating VRFs. For BLS we just use the first 32 bytes of the signature, TODO is this safe?
func (pub *Blspub) ProofToHash(m sig.SignedMessage, proof sig.VRFProof) (index [32]byte, err error) {
	bsig := &Blssig{}
	var n int
	n, err = bsig.Decode(bytes.NewBuffer(proof.(VRFProof)))
	if err != nil {
		return
	}
	if n != len(proof.(VRFProof)) {
		err = types.ErrInvalidMsgSize
		return
	}
	var valid bool
	valid, err = pub.doVerifySig(m, bsig)
	if err != nil {
		return
	}
	if !valid {
		err = types.ErrInvalidSig
		return
	}
	hf, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	hf.Write(proof.(VRFProof))
	hf.Sum(index[:0])
	return
}

/*// MergeBlsPubList combines a list of BLS public key objects into a single one
// Note this is not currently used, instead the MergePubPartial is used
func MergeBlsPubList(pubs ...*Blspub) (*Blspub, error) {
	if len(pubs) == 0 {
		return nil, types.ErrNilPub
	}
	if len(pubs) == 1 {
		return pubs[0], nil
	}
	pb := pubs[0].newPub.Clone()
	// bid := pubs[0].GetBitID()

	bids := make([]bitid.BitIDInterface, 0, len(pubs))
	bids = append(bids, pubs[0].GetBitID())
	for _, nxt := range pubs[1:] {
		// n1 := pb.Clone()
		pb = pb.Add(nxt.newPub, pb)
		bids = append(bids, nxt.GetBitID())
		// bid = MergeBitIDType(bid, nxt.GetBitID(), true)
	}

	return &Blspub{
		pub:    pb,
		newPub: pb,
		bitID:  bitid.MergeBitIDListType(true, bids...)}, nil
}
*/

///////////////////////////////////////////////////////////////////////////
// Public key
///////////////////////////////////////////////////////////////////////////

// Blspub represent a BLS public key object
type Blspub struct {
	pub         kyber.Point             // public key as a point on the ecCurve
	newPub      kyber.Point             // if using new multi sigs, is pub * hpk, otherwise is just the public key
	hpk         kyber.Scalar            // if using new multi sigs is the hash of the public key, otherwise nil
	pubBytes    sig.PubKeyBytes         // the serialized public key
	newPubBytes sig.PubKeyBytes         // if using new multi sigs, is pub * hpk, otherwise is just the public key
	pubString   sig.PubKeyStr           // should be the string representation of newPubBytes
	bitID       bitid.NewBitIDInterface // Array of bits inidcating which indecies of the members this public key represents (i.e. can be multiple if using multi-sigs)
	// pubID       PubKeyID
	// pubID should only be used when sig.UseMultiSig= false
	pubID      sig.PubKeyID // A unique string represeting this key for a consensus iteration (see PubKeyID type)
	newBidFunc bitid.FromIntFunc
	idx        sig.PubKeyIndex // only used is multi-sig is disabled
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *Blspub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// GetMsgID returns the msg id for BLS pub
func (pub *Blspub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

// SetIndex sets the index of the node represented by this public key in the consensus participants
func (pub *Blspub) SetIndex(index sig.PubKeyIndex) {
	pub.pubID = ""
	if sig.UseMultisig {
		pub.bitID = pub.newBidFunc(sort.IntSlice{int(index)})
	} else {
		pub.idx = index
	}
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *Blspub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	if sig.UseMultisig {
		il := pub.bitID.GetItemList()
		if len(il) != 1 {
			panic("should only be used on the local key")
		}
		return sig.PubKeyIndex(il[0])
	}
	return pub.idx
}

// GetSigMemberNumber returns the number of nodes represented by this BLS pub key
func (pub *Blspub) GetSigMemberNumber() int {
	if !sig.UseMultisig {
		return 1
	}
	if sig.UsePubIndex {
		_, _, _, count := pub.GetBitID().GetBasicInfo()
		return count
	}
	return 1
}

// FromPubBytes creates a BLS pub key object from the bytes of a public key
func (pub *Blspub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p := pub.New().(*Blspub)
	p.pubBytes = b
	_, err := p.getPub()
	// p.pubString = sig.PubKeyStr(p.pubBytes)
	return p, err
}

// New generates an empty Blspub object
func (pub *Blspub) New() sig.Pub {
	return &Blspub{
		newBidFunc: pub.newBidFunc,
	}
}

// DeserializeSig takes a message and returns a BLS public key object and signature as well as the number of bytes read
func (pub *Blspub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.CoinProof {
		return nil, 0, types.ErrCoinProofNotSupported
	}
	ret, l, err := sig.DeserVRF(pub, m)
	if err != nil {
		return nil, l, err
	}
	ret.Pub = pub.New()
	l1, err := ret.Pub.Deserialize(m, types.NilIndexFuns)
	l += l1
	if err != nil {
		return nil, l, err
	}

	ret.Sig = &Blssig{}
	l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
	l += l1
	return ret, l, err
}

var blsVerifyTime = 3077948 * time.Nanosecond // expected time to verify a bls sig on a Intel(R) Core(TM) i7-6820HQ CPU @ 2.70GHz (Amazon C5 instances)

// VerifySig verifies that sig is a valid signature for msg by BLS public key pub
func (pub *Blspub) VerifySig(msg sig.SignedMessage, asig sig.Sig) (bool, error) {
	return pub.doVerifySig(msg, asig)
}

func (pub *Blspub) doVerifySig(msg sig.SignedMessage, asig sig.Sig) (bool, error) {
	switch v := asig.(type) {
	case *Blssig:
		p, err := pub.getPub()
		if err != nil {
			return false, err
		}
		if sig.SleepValidate {
			sig.DoSleepValidation(blsVerifyTime)
			return true, nil
		}
		if err := VerifyBLS(blssuite, p, msg.GetSignedMessage(), v.sig); err != nil {
			// fmt.Sprintf("%v, %v", pub.GetBitID().GetItemList(), p)
			return false, err
		}
		return true, nil
	default:
		return false, types.ErrInvalidSigType
	}
}

func (pub *Blspub) getPub() (kyber.Point, error) {
	if pub.pub == nil {
		pubBytes, err := pub.GetRealPubBytes()
		if err != nil {
			return nil, err
		}
		if sig.BlsMultiNew {
			p, newPub, hpk, err := blsmsFromBytes(pubBytes)
			if err != nil {
				return nil, err
			}
			pub.pub = p
			pub.newPub = newPub
			pub.hpk = hpk
		} else {
			p := blssuite.G2().Point()
			if err := p.UnmarshalBinary(pubBytes); err != nil {
				return nil, err
			}
			pub.pub = p
			pub.newPub = p
		}
	}
	return pub.newPub, nil
}

// GetPubString is the same as GetPubBytes, except returns a string
func (pub *Blspub) GetPubString() (sig.PubKeyStr, error) {
	if pub.newPubBytes == nil {
		_, err := pub.GetPubBytes()
		if err != nil {
			return "", err
		}
	}
	return pub.pubString, nil
}

// GetPubBytes returns the BLS pub key as bytes, if using new multi sigs, is pub * hpk, otherwise is just the public key
func (pub *Blspub) GetPubBytes() (sig.PubKeyBytes, error) {
	if pub.newPubBytes == nil {
		var err error
		if pub.pub == nil && pub.pubBytes != nil {
			_, err = pub.getPub()
			if err != nil {
				return nil, err
			}
		}
		pub.newPubBytes, err = pub.newPub.MarshalBinary()
		if err != nil {
			return nil, err
		}
		pub.pubString = sig.PubKeyStr(pub.newPubBytes)
	}
	return pub.newPubBytes, nil
}

// GetPubBytes returns the BLS pub key as bytes
func (pub *Blspub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	if pub.pubBytes == nil {
		if pub.pub == nil {
			return nil, types.ErrNoPubBytes
		}
		var err error
		pub.pubBytes, err = pub.pub.MarshalBinary()
		if err != nil {
			return nil, err
		}
	}
	return pub.pubBytes, nil
}

// GetPubID returns the unique id for this pubkey (given some consensus instance), it could be the encoded bitid, or just the pub key
// depending on how SetUsePubIndex was set
func (pub *Blspub) GetPubID() (sig.PubKeyID, error) {
	if sig.UseMultisig {
		if bid := pub.GetBitID(); bid != nil {
			return sig.PubKeyID(bid.GetStr()), nil
		}
		return "", types.ErrInvalidBitID
	}
	if sig.UsePubIndex {
		if pub.pubID == "" {
			buff := make([]byte, 4)
			config.Encoding.PutUint32(buff, uint32(pub.idx))
			pub.pubID = sig.PubKeyID(buff)
		}
		return pub.pubID, nil
	}
	pid, err := pub.GetPubString()
	return sig.PubKeyID(pid), err
}

func (pub *Blspub) Encode(writer io.Writer) (n int, err error) {
	pb, err := pub.GetRealPubBytes()
	if err != nil {
		return 0, err
	}
	if len(pb) != blsmarshalsize {
		return 0, types.ErrInvalidMsgSize
	}
	return writer.Write(pb)
}

func (pub *Blspub) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, blsmarshalsize)
	n, err = reader.Read(buf)
	if err != nil || n != blsmarshalsize {
		panic(err)
		return
	}
	pub.pubBytes = buf
	// pub.pubString = sig.PubKeyStr(buf)
	pub.pub = nil
	return
}

// Serialize the pub key into the message, return the number of bytes written
func (pub *Blspub) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, pub.GetID(), 0, (*messages.MsgBuffer)(m))

	if sig.UseMultisig {
		// the bit id
		bid := pub.GetBitID()
		if bid == nil {
			return 0, types.ErrInvalidBitID
		}
		l1, err := bid.Encode((*messages.MsgBuffer)(m))
		l += l1
		if err != nil {
			return l, err
		}
	} else if sig.UsePubIndex {
		(*messages.MsgBuffer)(m).AddUint32(uint32(pub.idx))
		l += 4
	} else {
		// now the pub
		buff, err := pub.GetRealPubBytes()
		if err != nil {
			return 0, err
		}
		(*messages.MsgBuffer)(m).AddBytes(buff)
		l += len(buff)
	}

	// now update the size
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}
	return l, nil
}

// GetBytes returns the bytes of the BLS public key from the message
func (pub *Blspub) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// PeekHeader returns nil.
func (Blspub) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// Deserialize updates the fields of the BLS pub key object from m, and returns the number of bytes read
func (pub *Blspub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(pub.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	if sig.UseMultisig {
		// the bit id
		pub.bitID = pub.newBidFunc(nil)
		br, err := pub.bitID.Decode((*messages.MsgBuffer)(m))
		l += br
		if err != nil {
			return l, err
		}
	} else if sig.UsePubIndex {
		index, br, err := (*messages.MsgBuffer)(m).ReadUint32()
		if err != nil {
			return 0, err
		}
		l += br
		pub.idx = sig.PubKeyIndex(index)
	} else {
		// Now the pub
		buff, err := (*messages.MsgBuffer)(m).ReadBytes(int(size) - l)
		if err != nil {
			return 0, err
		}
		l += len(buff)
		pub.pubBytes = buff
		// pub.pubString = sig.PubKeyStr(buff)
	}
	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}
	pub.pub = nil
	return l, nil
}

// GetID returns the header id for BLS pub objects
func (pub *Blspub) GetID() messages.HeaderID {
	return messages.HdrBlspub
}
