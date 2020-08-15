package sleep

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math"
	"time"
)

const sharedIDBytes = 2 + 11
const sharedIDString = "sleepshared"

type sharedPub struct {
	sig.Pub
	memberCount int
	id          sig.PubKeyID
	stats       *sig.SigStats
}

func newSharedPub(newPubFunc func() sig.Pub, stats *sig.SigStats, memberCount int) *sharedPub {
	buff := bytes.NewBuffer(nil)
	if memberCount > math.MaxUint16 {
		panic("member count too large")
	}
	_, err := utils.EncodeUint16(uint16(memberCount), buff)
	if err != nil {
		panic(err)
	}
	if _, err = buff.WriteString(sharedIDString); err != nil {
		panic(err)
	}
	if buff.Len() != sharedIDBytes {
		panic("error encoding int")
	}
	return &sharedPub{
		Pub:         newPubFunc(),
		stats:       stats,
		memberCount: memberCount,
		id:          sig.PubKeyID(buff.Bytes()),
	}
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *sharedPub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	time.Sleep(pub.stats.SigVerifyTime)
	return nil
}

// GetPubID returns the id for this pubkey.
// Given that there is only one threshold pub per consensus it returns PubKeyID("blssharedpub").
func (pub *sharedPub) GetPubID() (sig.PubKeyID, error) {
	return pub.id, nil
}

// GetSigMemberNumber returns the number of nodes represented by this BLS pub key
func (pub *sharedPub) GetSigMemberNumber() int {
	return pub.memberCount
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *sharedPub) ShallowCopy() sig.Pub {
	newPub := *pub
	newPub.Pub = newPub.Pub.ShallowCopy()
	return &newPub
}

type Thrsh struct {
	n   int             // the number of participants
	t   int             // the number of signatures needed for the threshold
	idx sig.PubKeyIndex // the index of this node in the list of sorted pub keys

	SleepPriv            // the partial public key for this node
	sharedPub *sharedPub // the threshold public key
}

// PartialSign creates a signature on the message that can also be used to create a shared signature.
func (bt *Thrsh) PartialSign(msg sig.SignedMessage) (sig.Sig, error) {
	return bt.Sign(msg)
}

// Returns key that is used for signing the sign type.
func (bt *Thrsh) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return bt, nil
}

// Shallow copy makes a copy of the object without following pointers.
func (bt *Thrsh) ShallowCopy() sig.Priv {
	newBt := *bt
	newBt.SleepPriv = newBt.SleepPriv.ShallowCopy().(SleepPriv)
	return &newBt
}

// CombinePartialSigs generates a shared signature from the list of partial signatures.
func (bt *Thrsh) CombinePartialSigs(ps []sig.Sig) (*sig.SigItem, error) {
	if len(ps) < bt.t {
		return nil, types.ErrNotEnoughPartials
	}
	for i := 0; i < bt.t; i++ {
		time.Sleep(bt.sharedPub.stats.ShareCombineTime)
	}
	// since all sigs are the same we just take the bytes from any one of them
	si := &Sig{
		bytes: ps[0].(*Sig).bytes,
		stats: bt.sharedPub.stats,
	}
	m := messages.NewMessage(nil)
	// When using the  shared threshold key, we don't include the serialized public
	// key with the signature as there is only one threshold key
	// first we write a 0 to indicate no VRFProof
	err := (*messages.MsgBuffer)(m).WriteByte(0)
	if err != nil {
		return nil, err
	}
	_, err = si.Serialize(m)
	if err != nil {
		return nil, err
	}
	pid, err := bt.sharedPub.GetPubID()
	if err != nil {
		panic(err)
	}
	n, err := (*messages.MsgBuffer)(m).Write([]byte(pid))
	if err != nil {
		panic(err)
	}
	if n != sharedIDBytes {
		panic(n)
	}
	return &sig.SigItem{
		Pub:      bt.sharedPub,
		Sig:      si,
		SigBytes: m.GetBytes()}, nil
}

func (bt *Thrsh) GetT() int {
	return bt.t
}

func (bt *Thrsh) GetN() int {
	return bt.n
}

func (bt *Thrsh) GetPartialPub() sig.Pub {
	return bt.SleepPriv.GetPub()
}

func (bt *Thrsh) GetSharedPub() sig.Pub {
	return bt.sharedPub
}

/*func VerifyPartialSig(msg []byte, pubInt sig.Pub, sigInt sig.Sig) error {
	pp, b := pubInt.(*Pub)
	if !b {
		return types.ErrInvalidPub
	}
	ps, b := sigInt.(*Sig)
	if !b {
		return types.ErrInvalidSigType
	}
	ok, err := pp.VerifySig(sig.BasicSignedMessage(msg), ps)
	if err != nil {
		return err
	}
	if !ok {
		return types.ErrInvalidSig
	}
	return nil
}
*/

type partialPub struct {
	sleepPub
	stats *sig.SigStats
}

// DeserializeSig takes a message and returns a BLS public key object and partial signature object as well as the number of bytes read
func (pp *partialPub) DeserializeSig(m *messages.Message, _ types.SignType) (*sig.SigItem, int, error) {

	// var ret *sig.SigItem
	ret := &sig.SigItem{}
	var l, l1 int
	var err error
	newPub := pp.New()
	if pp.stats.AllowsVRF {
		ret, l1, err = sig.DeserVRF(newPub, m)
		if err != nil {
			return nil, l, err
		}
		l += l1
	}
	ht, err := m.PeekHeaderType()
	if err != nil {
		return nil, l, err
	}
	switch ht {
	case messages.HdrSleepSig: // this was signed by the shared threshold key
		// When using the shared threshold key, we don't include the serialized public
		// key with the signature as there is only one threshold key
		ret.Sig = &Sig{stats: pp.stats}
		l1, err := ret.Sig.Deserialize(m, types.NilIndexFuns)
		if err != nil {
			return ret, l, err
		}
		l += l1
		// The id is the number
		id, err := (*messages.MsgBuffer)(m).ReadBytes(sharedIDBytes)
		if err != nil {
			return ret, l, err
		}
		l += sharedIDBytes
		ret.Pub = &sharedPub{id: sig.PubKeyID(id), stats: pp.stats}
		return ret, l, err
	case messages.HdrSleepPub:
		ret.Pub = pp.New()
		l1, err := ret.Pub.Deserialize(m, types.NilIndexFuns)
		l += l1
		if err != nil {
			return nil, l, err
		}

		ret.Sig = &Sig{stats: pp.stats}
		l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
		l += l1
		return ret, l, err
	default:
		return nil, 0, types.ErrInvalidHeader
	}
}

// New creates an empty sleep private key object
func (p *partialPub) New() sig.Pub {
	return &partialPub{sleepPub: p.sleepPub.New().(sleepPub),
		stats: p.stats,
	}
}

func (pub *partialPub) ShallowCopy() sig.Pub {
	newPub := *pub
	newPub.sleepPub = pub.sleepPub.ShallowCopy().(sleepPub)
	return &newPub
}

// FromPubBytes creates a public key object from the public key bytes
func (pub *partialPub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p, err := pub.sleepPub.FromPubBytes(b)
	if err != nil {
		return p, err
	}
	return &partialPub{
		sleepPub: p.(sleepPub),
		stats:    pub.stats,
	}, nil
}
