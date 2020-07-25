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

func newSharedPub(stats *sig.SigStats, memberCount int) *sharedPub {
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
		Pub:         &Pub{stats: stats},
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

	*Priv                // the partial public key for this node
	sharedPub *sharedPub // the threshold public key
}

// PartialSign creates a signature on the message that can also be used to create a shared signature.
func (bt *Thrsh) PartialSign(msg sig.SignedMessage) (sig.Sig, error) {
	return bt.Sign(msg)
}

// CombinePartialSigs generates a shared signature from the list of partial signatures.
func (bt *Thrsh) CombinePartialSigs(ps []sig.Sig) (*sig.SigItem, error) {
	if len(ps) < bt.t {
		return nil, types.ErrNotEnoughPartials
	}
	for i := 0; i < bt.t; i++ {
		time.Sleep(bt.stats.ShareCombineTime)
	}
	// since all sigs are the same we just take the bytes from any one of them
	si := &Sig{
		bytes: ps[0].(*Sig).bytes,
		stats: bt.stats,
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
	return bt.pub
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
