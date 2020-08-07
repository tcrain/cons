package sleep

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"time"
)

type Pub struct {
	pubBytes  []byte
	pubString sig.PubKeyStr
	pubID     sig.PubKeyID
	index     sig.PubKeyIndex
	stats     *sig.SigStats
}

// New creates a new public key object of the same type
func (pub *Pub) New() sig.Pub {
	return &Pub{
		stats: pub.stats,
	}
}

// VerifySig verifies a signature, returns (true, nil) if valid, (false, nil) if invalid, or (false, error) if there was an error verifying the signature
func (pub *Pub) VerifySig(msg sig.SignedMessage, aSig sig.Sig) (bool, error) {
	time.Sleep(pub.stats.SigVerifyTime)
	return true, nil
}

// CheckSignature will should check what kind of message and signature to verify. It will call CheckCoinProof or VerfiySig
// depending on the message type.
func (pub *Pub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {
	// Check if this is a coin proof or a signature
	var err error
	signType := msg.GetSignType()
	if signType == types.CoinProof && (!pub.stats.AllowsCoin && !pub.stats.AllowsThresh) { // sanity check
		panic("coin proof not supported")
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

// GetPubBytes returns the key we use to sign/verify
func (pub *Pub) GetPubBytes() (sig.PubKeyBytes, error) {
	return pub.pubBytes, nil
}

// GetRealPubBytes is the same as GetPubBytes
func (pub *Pub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	return pub.pubBytes, nil
}

// GetPubString is the same as GetPubBytes, except returns a string, this is used to sort the keys
func (pub *Pub) GetPubString() (sig.PubKeyStr, error) {
	if pub.pubString == "" {
		pub.pubString = sig.PubKeyStr(pub.pubBytes)
	}
	return pub.pubString, nil
}

// GetSigMemberNumber returns the number of members this sig counts for (can be > 1 for multisigs and threshold sigs)
func (pub *Pub) GetSigMemberNumber() int {
	return 1
}

// GetPubID returns the id of the public key (see type definition for PubKeyID).
func (pub *Pub) GetPubID() (sig.PubKeyID, error) {
	if sig.UsePubIndex {
		if pub.pubID == "" {
			buff := make([]byte, 4)
			config.Encoding.PutUint32(buff, uint32(pub.index))
			pub.pubID = sig.PubKeyID(buff)
		}
		return pub.pubID, nil
	}
	pid, err := pub.GetPubString()
	return sig.PubKeyID(pid), err
}

// SetIndex should be called after sorting all if the benchmark is using the index as the key id (see SetUsePubIndex)
func (pub *Pub) SetIndex(index sig.PubKeyIndex) {
	pub.pubID = ""
	pub.index = index
}

// FromPubBytes creates a public key object from the public key bytes
func (pub *Pub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p := pub.New().(*Pub)
	p.pubBytes = b
	p.pubString = sig.PubKeyStr(p.pubBytes)
	return p, nil
}

func (pub *Pub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *Pub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndxe is false")
	}
	return pub.index
}

// DeserializeSig deserializes a public key and signature object from m, size is the number of bytes read
func (pub *Pub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	return pub.doDeserializeSig(&Sig{stats: pub.stats}, pub.New(), m, signType)
}

// doDeserializeSig deserializes a public key and signature object from m, size is the number of bytes read
func (pub *Pub) doDeserializeSig(theSig sig.Sig, newPub sig.Pub, m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.CoinProof && pub.stats.AllowsCoin {
		panic("should have handled in coin")
	}

	// var ret *sig.SigItem
	ret := &sig.SigItem{}
	var l, l1 int
	var err error
	if pub.stats.AllowsVRF {
		ret, l1, err = sig.DeserVRF(newPub, m)
		if err != nil {
			return nil, l, err
		}
		l += l1
	}
	ret.Pub = newPub
	l1, err = ret.Pub.Deserialize(m, types.NilIndexFuns)
	l += l1
	if err != nil {
		return nil, l, err
	}

	ret.Sig = theSig
	l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
	l += l1
	return ret, l, err
}

// PeekHeaders returns nil
func (pub *Pub) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (pub *Pub) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, pub.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the pub
	if sig.UseMultisig && pub.stats.AllowsMulti {
		panic("unsupported")
	} else if sig.UsePubIndex {
		(*messages.MsgBuffer)(m).AddUint32(uint32(pub.index))
		l += 4
	} else {
		buff, err := pub.GetPubBytes()
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

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (pub *Pub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(pub.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	// Now the pub
	if sig.UseMultisig && pub.stats.AllowsMulti {
		panic("unsupported")
	} else if sig.UsePubIndex {
		index, br, err := (*messages.MsgBuffer)(m).ReadUint32()
		if err != nil {
			return 0, err
		}
		l += br
		pub.index = sig.PubKeyIndex(index)
	} else {
		buff, err := (*messages.MsgBuffer)(m).ReadBytes(int(size) - l)
		if err != nil {
			return 0, err
		}
		l += len(buff)
		pub.pubBytes = buff
		pub.pubString = sig.PubKeyStr(buff)
	}
	if size != uint32(l) {
		return 0, types.ErrInvalidMsgSize
	}
	return l, nil
}

// GetBytes returns the bytes that make up the header
func (pub *Pub) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// GetID returns the header id for this header
func (pub *Pub) GetID() messages.HeaderID {
	return messages.HdrSleepPub
}

// GetMsgID returns the MsgID for this specific header (see MsgID definition for more details)
func (pub *Pub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

func (pub *Pub) Encode(writer io.Writer) (n int, err error) {
	buff, err := pub.GetPubBytes()
	if err != nil {
		return 0, err
	}
	if len(buff) != pub.stats.PubSize {
		panic(len(buff))
	}
	return writer.Write(buff)
}
func (pub *Pub) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, pub.stats.PubSize)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	if n != len(buf) {
		panic(n)
	}
	pub.pubBytes = buf
	pub.pubString = sig.PubKeyStr(buf)
	return
}
