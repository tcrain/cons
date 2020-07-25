package sleep

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"time"
)

type multiPub struct {
	sig.Pub
	stats *sig.SigStats
	bitID bitid.BitIDInterface
}

func newMultiPub(p sig.Pub, stats *sig.SigStats) *multiPub {
	ret := &multiPub{
		Pub:   p,
		stats: stats,
	}
	// ret.SetIndex(p.GetIndex())
	return ret
}

// New creates a new public key object of the same type
func (pub *multiPub) New() sig.Pub {
	return &multiPub{
		Pub:   pub.Pub.New(),
		stats: pub.stats,
	}
}

// FromPubBytes creates a public key object from the public key bytes
func (pub *multiPub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p, err := pub.Pub.FromPubBytes(b)
	if err != nil {
		panic(err)
	}
	return &multiPub{
		Pub:   p,
		stats: pub.stats,
	}, nil
}

// GetSigMemberNumber returns the number of nodes represented by this pub key
func (pub *multiPub) GetSigMemberNumber() int {
	return pub.bitID.GetNumItems()
}

// SetIndex sets the index of the node represented by this public key in the consensus participants
func (pub *multiPub) SetIndex(index sig.PubKeyIndex) {
	// pub.pubID = ""
	pub.Pub.SetIndex(index)
	var err error
	pub.bitID, err = bitid.CreateBitIDTypeFromInts([]int{int(index)})
	if err != nil {
		panic(err)
	}
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *multiPub) GetIndex() sig.PubKeyIndex {
	il := pub.bitID.GetItemList()
	if len(il) != 1 {
		panic("should only be used on the local key")
	}
	return sig.PubKeyIndex(il[0])
}

func (pub *multiPub) ShallowCopy() sig.Pub {
	newPub := *pub
	newPub.Pub = pub.ShallowCopy()
	return &newPub
}

// Sub*multiPub remove pub2 from pub1 and returns the resulting public key object
func (pub *multiPub) SubMultiPub(pub2 sig.MultiPub) (sig.MultiPub, error) {
	p1bid := pub.GetBitID()
	p2bid := pub2.GetBitID()
	if p1bid.GetNumItems() <= p2bid.GetNumItems() {
		return nil, types.ErrInvalidBitIDSub
	}
	if len(p1bid.CheckIntersection(p2bid)) != p2bid.GetNumItems() {
		return nil, types.ErrInvalidBitIDSub
	}
	pb1, err := pub.GetPubBytes()
	if err != nil {
		panic(err)
	}
	pb2, err := pub.GetPubBytes()
	if err != nil {
		panic(err)
	}

	newBytes, err := utils.XORbytes(pb1, pb2)
	if err != nil {
		panic(err)
	}
	newPub, err := pub.Pub.FromPubBytes(newBytes)
	if err != nil {
		panic(err)
	}

	time.Sleep(pub.stats.MultiCombineTime)
	return &multiPub{Pub: newPub, bitID: bitid.SubBitIDType(p1bid, p2bid), stats: pub.stats}, nil
}

// MergePubPartial only merges the pub itself, does not create the new bitid
func (pub *multiPub) MergePubPartial(pub2 sig.MultiPub) {
	time.Sleep(pub.stats.MultiCombineTime)
}

// GetPubID returns the id of the public key (see type definition for PubKeyID).
func (pub *multiPub) GetPubID() (sig.PubKeyID, error) {
	if pub.stats.AllowsMulti && sig.UseMultisig {
		if bid := pub.GetBitID(); bid != nil {
			return sig.PubKeyID(bid.GetStr()), nil
		}
		return "", types.ErrInvalidBitID
	}
	panic("should only be called for multisigs")
}

// DonePartialMerge should be called after merging keys with MergePubPartial to set the bitid
func (pub *multiPub) DonePartialMerge(bid bitid.BitIDInterface) {
	pub.bitID = bid
}

// GenerateSerializedSig serialized the public key and the signature and returns the bytes
func (pub *multiPub) GenerateSerializedSig(bsig sig.MultiSig) ([]byte, error) {
	// hash := header.GetSignedHash()
	si, err := sig.GenerateSigHelperFromSig(pub, bsig.(sig.Sig), nil, types.NormalSignature)
	if err != nil {
		return nil, err
	}
	return si.SigBytes, nil
}

// Merge*multiPub combines two BLS public key objects into a single one
func (pub *multiPub) MergePub(pub2 sig.MultiPub) (sig.MultiPub, error) {
	p1bid := pub.GetBitID()
	p2bid := pub2.GetBitID()
	if p1bid == nil || p2bid == nil {
		return nil, types.ErrInvalidBitID
	}
	if !config.AllowMultiMerge && len(p1bid.CheckIntersection(p2bid)) > 0 {
		return nil, types.ErrIntersectingBitIDs
	}

	pb1, err := pub.GetPubBytes()
	if err != nil {
		panic(err)
	}
	pb2, err := pub.GetPubBytes()
	if err != nil {
		panic(err)
	}

	newBytes, err := utils.XORbytes(pb1, pb2)
	if err != nil {
		panic(err)
	}
	newPub, err := pub.Pub.FromPubBytes(newBytes)
	if err != nil {
		panic(err)
	}

	return &multiPub{
		stats: pub.stats,
		Pub:   newPub,
		bitID: bitid.MergeBitIDType(p1bid, p2bid, config.AllowMultiMerge)}, nil
}

// GetBitID returns the bit id object representing the indecies of the nodes represented by the BLS public key oject
func (pub *multiPub) GetBitID() bitid.BitIDInterface {
	return pub.bitID
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (pub *multiPub) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, pub.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the pub
	if sig.UseMultisig && pub.stats.AllowsMulti {
		// the bit id
		bid := pub.GetBitID()
		if bid == nil {
			return 0, types.ErrInvalidBitID
		}
		bidEncoding := bid.Encode()
		l1, _ := (*messages.MsgBuffer)(m).AddUint32(uint32(len(bidEncoding)))
		l += l1
		(*messages.MsgBuffer)(m).AddBytes(bidEncoding)
		l += len(bidEncoding)
		// now update the size
		err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
		if err != nil {
			return 0, err
		}
		return l, nil
	}
	return pub.Pub.Serialize(m)
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (pub *multiPub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc

	// Now the pub
	if sig.UseMultisig && pub.stats.AllowsMulti {
		l, _, _, size, _, err := messages.ReadHeaderHead(pub.GetID(), nil, (*messages.MsgBuffer)(m))
		if err != nil {
			return 0, err
		}
		// the bit id
		bitIDlen, br, err := (*messages.MsgBuffer)(m).ReadUint32()
		if err != nil {
			return 0, err
		}
		l += br
		bitID, err := (*messages.MsgBuffer)(m).ReadBytes(int(bitIDlen))
		if err != nil {
			return 0, err
		}
		l += len(bitID)
		pub.bitID, err = bitid.DecodeBitIDType(bitID)
		if err != nil {
			return 0, err
		}
		if size != uint32(l) {
			return 0, types.ErrInvalidMsgSize
		}
		return l, nil
	}
	return pub.Pub.Deserialize(m, unmarFunc)
}

type multiSig struct {
	Sig
}

func (s *multiSig) New() sig.Sig {
	return &multiSig{Sig: *s.Sig.New().(*Sig)}
}

// MergeSig combines two signatures, it assumes the sigs are valid to be merged
func (sig1 *multiSig) MergeSig(sig2 sig.MultiSig) (sig.MultiSig, error) {
	newBytes, err := utils.XORbytes(sig1.Sig.bytes, sig2.(*multiSig).bytes)
	if err != nil {
		panic(err)
	}
	newSig := sig1.New().(*multiSig)
	newSig.bytes = newBytes
	return &multiSig{Sig: newSig.Sig}, nil
}

// SubSig removes sig2 from sig1, it assumes sig 1 already contains sig2
func (sig1 *multiSig) SubSig(sig2 sig.MultiSig) (sig.MultiSig, error) {
	newBytes, err := utils.XORbytes(sig1.Sig.bytes, sig2.(*multiSig).bytes)
	if err != nil {
		panic(err)
	}
	newSig := sig1.New().(*multiSig)
	newSig.bytes = newBytes
	return &multiSig{Sig: newSig.Sig}, nil
}
