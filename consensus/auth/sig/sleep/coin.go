package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"time"
)

type coinPriv struct {
	SleepPriv
	coinPub *coin
}

func (p *coinPriv) GetPub() sig.Pub {
	return p.coinPub
}

func (p *coinPriv) ShallowCopy() sig.Priv {
	newP := *p
	newP.SleepPriv = p.SleepPriv.ShallowCopy().(SleepPriv)
	return &newP
}

// New creates an empty sleep private key object
func (p *coinPriv) New() sig.Priv {
	return &coinPriv{SleepPriv: p.New().(SleepPriv)}
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (p *coinPriv) SetIndex(index sig.PubKeyIndex) {
	p.SleepPriv.SetIndex(index)
	p.coinPub.SetIndex(index)
}

// Returns key that is used for signing the sign type.
func (p *coinPriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return p, nil
}

func NewCoinSleepPriv(n, t int, i sig.PubKeyIndex, stats *sig.SigStats) sig.Priv {
	cp, err := NewSleepPriv(stats, i)
	if err != nil {
		panic(err)
	}
	// p := cp.(*Priv)
	c := &coin{
		stats:     stats,
		n:         n,
		t:         t,
		Pub:       cp.GetPub(),
		sharedPub: newSharedPub(cp.GetPub().New, stats, t),
	}
	return &coinPriv{
		SleepPriv: cp.(SleepPriv),
		coinPub:   c,
	}
}

type coin struct {
	n, t int
	sig.Pub
	sharedPub *sharedPub
	stats     *sig.SigStats
}

// NewCoinProof returns an empty coin proof object
func (c *coin) NewCoinProof() sig.CoinProof {
	return &coinProof{
		stats: c.stats,
	}
}

func (c *coin) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p, err := c.Pub.FromPubBytes(b)
	if err != nil {
		return nil, err
	}
	return &coin{
		n:         c.n,
		t:         c.t,
		Pub:       p,
		sharedPub: c.sharedPub,
		stats:     c.stats,
	}, nil
}

func (c *coin) ShallowCopy() sig.Pub {
	newPub := *c
	newPub.Pub = c.Pub.ShallowCopy()
	return &newPub
}

func (c *coin) New() sig.Pub {
	return &coin{
		n:         c.n,
		t:         c.t,
		Pub:       c.Pub.New(),
		sharedPub: c.sharedPub,
		stats:     c.stats,
	}
}

func (c *coin) GetT() int {
	return c.t
}

func (c *coin) GetN() int {
	return c.n
}

//func (c *coin) GetPartialPub() sig.Pub {
//	return c.Pub
//}

func (c *coin) GetSharedPub() sig.Pub {
	return c.sharedPub
}

func (c *coin) CheckCoinProof(msg sig.SignedMessage, prf sig.CoinProof) error {
	time.Sleep(c.stats.ShareVerifyTime)
	return nil
}

// CombineProofs combines the given proofs and returns the resulting coin values.
// The proofs are expected to have already been validated by CheckcoinProof.
func (c *coin) CombineProofs(_ sig.Priv, items []*sig.SigItem) (coinVal types.BinVal, err error) {
	for i := 0; i < c.GetT(); i++ {
		time.Sleep(c.stats.ShareCombineTime)
	}
	// since all coin proofs are the same, just take the hash of any of them, and take the first byte mod 2
	hsh := types.GetHash(items[0].Sig.(*coinProof).buff)
	return types.BinVal(hsh[0] % 2), nil
}

func (c *coin) DeserializeCoinProof(m *messages.Message) (cp sig.CoinProof, size int, err error) {
	cp = &coinProof{stats: c.stats}
	//coinProof = coinproof.EmptyCoinProof(sig.EdSuite)
	size, err = cp.Deserialize(m, types.NilIndexFuns)
	return
}

func (c *coin) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.CoinProof && c.stats.AllowsCoin {
		var n int
		var l1 int
		var err error
		newPub := c.New()
		l1, err = newPub.Deserialize(m, types.NilIndexFuns)
		if err != nil {
			return nil, n, err
		}
		n += l1
		var coinProof sig.CoinProof
		if coinProof, l1, err = c.DeserializeCoinProof(m); err != nil {
			return nil, l1, err
		}
		n += l1
		return &sig.SigItem{Pub: newPub, Sig: coinProof}, n, nil
	}
	return c.Pub.DeserializeSig(m, signType)
}

type coinProof struct {
	stats *sig.SigStats
	buff  []byte
}

func (cp *coinProof) New() sig.Sig {
	return &coinProof{stats: cp.stats}
}

// GetMsgID returns the message id for a coin proof.
func (cp *coinProof) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(cp.GetID())
}

// PeekHeader returns nil.
func (coinProof) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// GetBytes returns the bytes of the coin proof from the message.
func (cp *coinProof) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cp *coinProof) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	return messages.DeserializeHelper(cp.GetID(), cp, m)
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cp *coinProof) Serialize(m *messages.Message) (int, error) {
	return messages.SerializeHelper(cp.GetID(), cp, m)
}

func (cp *coinProof) GetID() messages.HeaderID {
	return messages.HdrSleepCoin
}

func (cp *coinProof) Encode(writer io.Writer) (n int, err error) {
	n, err = writer.Write(cp.buff)
	if n != cp.stats.ThrshShareSize || err != nil {
		panic(n)
	}
	return
}

func (cp *coinProof) Decode(reader io.Reader) (n int, err error) {
	cp.buff = make([]byte, cp.stats.ThrshShareSize)
	return reader.Read(cp.buff)
}

func generateCoinProof(priv *Priv, header sig.SignedMessage) *coinProof {
	// coin proof is just the hash of the message, it is the same at all processes
	buff := make([]byte, priv.stats.ThrshShareSize)
	copy(buff, header.GetSignedHash())
	return &coinProof{
		stats: priv.stats,
		buff:  buff,
	}
}
