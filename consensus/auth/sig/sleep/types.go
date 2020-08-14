package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/dual"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/auth/sig/qsafe"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
)

var AllSigStats = []sig.SigStats{ec.Stats, ed.EDDSAStats, ed.SchnorrStats, bls.Stats}

func GetAllSigStatsNewPriv(onlyVrf, onlyMulti bool) (ret []func() (sig.Priv, error)) {
	for _, nxt := range AllSigStats {
		if onlyVrf && !nxt.AllowsVRF {
			continue
		}
		if onlyMulti && !nxt.AllowsMulti {
			continue
		}
		var nxtFunc func() (sig.Priv, error)
		var i sig.PubKeyIndex
		nxtFunc = func() (sig.Priv, error) {
			p, err := NewSleepPriv(&nxt, i)
			if err != nil {
				return nil, err
			}
			i++
			if nxt.AllowsVRF {
				p, err = NewSleepVrfPriv(p, &nxt)
				if err != nil {
					return nil, err
				}
			}
			if nxt.AllowsMulti && sig.UseMultisig {
				p, err = NewSleepMultiPriv(p, &nxt)
			}
			return p, err
		}
		ret = append(ret, nxtFunc)
	}
	return
}

// NewSleepPriv creates a new random sleep private key object
func NewSleepPriv(stats *sig.SigStats, i sig.PubKeyIndex) (sig.Priv, error) {
	rnd := rand.New(rand.NewSource(int64(i)))
	buff := make(sig.PubKeyBytes, stats.PubSize)
	if _, err := rnd.Read(buff); err != nil {
		return nil, err
	}
	p, err := (&Pub{stats: stats}).FromPubBytes(buff)
	if err != nil {
		return nil, err
	}
	return &Priv{
		stats: stats,
		pub:   p.(*Pub),
	}, nil
}

type MultiPriv struct {
	SleepPriv
}

// Returns key that is used for signing the sign type.
func (p *MultiPriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return p, nil
}

// Sign signs a message and returns the signature.
func (p *MultiPriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	s, err := p.SleepPriv.Sign(msg)
	if err != nil {
		return nil, err
	}
	return &multiSig{Sig: *s.(*Sig)}, nil
}

func (p *MultiPriv) ShallowCopy() sig.Priv {
	newP := *p
	newP.SleepPriv = p.SleepPriv.ShallowCopy().(SleepPriv)
	return &newP
}

func NewSleepMultiPriv(priv sig.Priv, stats *sig.SigStats) (sig.Priv, error) {
	priv.(SleepPriv).setPub(newMultiPub(priv.GetPub(), stats))
	return &MultiPriv{
		SleepPriv: priv.(SleepPriv),
	}, nil
}

func NewSleepVrfPriv(priv sig.Priv, stats *sig.SigStats) (sig.Priv, error) {
	p := &VrfPub{
		sleepPub: priv.GetPub().(sleepPub),
		vRFPub:   vRFPub{stats: stats},
	}
	priv.(SleepPriv).setPub(p)
	return &VrfPriv{
		SleepPriv: priv.(SleepPriv),
		vrfPriv: vrfPriv{
			stats: stats,
			pub:   p,
		},
	}, nil
}

type VrfPriv struct {
	SleepPriv
	vrfPriv
}

// Returns key that is used for signing the sign type.
func (p *VrfPriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return p, nil
}

func (p *VrfPriv) ShallowCopy() sig.Priv {
	newP := *p
	newP.SleepPriv = p.SleepPriv.ShallowCopy().(SleepPriv)
	return &newP
}

type VrfPub struct {
	sleepPub
	vRFPub
}

// DeserializeSig deserializes a public key and signature object from m, size is the number of bytes read
func (pub *VrfPub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	return pub.sleepPub.doDeserializeSig(&Sig{stats: pub.stats}, pub.New(), m, signType)
}

// New creates an empty sleep private key object
func (p *VrfPub) New() sig.Pub {
	return &VrfPub{sleepPub: p.sleepPub.New().(sleepPub),
		vRFPub: p.vRFPub,
	}
}

func (pub *VrfPub) ShallowCopy() sig.Pub {
	newPub := *pub
	newPub.sleepPub = pub.sleepPub.ShallowCopy().(sleepPub)
	return &newPub
}

// FromPubBytes creates a public key object from the public key bytes
func (pub *VrfPub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p, err := pub.sleepPub.FromPubBytes(b)
	if err != nil {
		return p, err
	}
	return &VrfPub{
		sleepPub: p.(sleepPub),
		vRFPub:   pub.vRFPub,
	}, nil
}

func NewSleepThrshPriv(n, t int, priv sig.Priv, stats *sig.SigStats) (sig.Priv, error) {
	p := priv.(SleepPriv)
	ppub := &partialPub{
		sleepPub: p.GetPub().(sleepPub),
		stats:    stats,
	}
	p.setPub(ppub)
	return &Thrsh{
		n:         n,
		t:         t,
		idx:       p.GetPub().GetIndex(),
		SleepPriv: p.(SleepPriv),
		sharedPub: newSharedPub(p.GetPub().New, stats, t),
	}, nil
}

func NewECPriv(i sig.PubKeyIndex) (sig.Priv, error) {
	p, err := NewSleepPriv(&ec.Stats, i)
	utils.PanicNonNil(err)
	// return p, err
	return NewSleepVrfPriv(p, &ec.Stats)
}

func NewEDPriv(i sig.PubKeyIndex) (sig.Priv, error) {
	return NewSleepPriv(&ed.EDDSAStats, i)
}

func NewSchnorrPriv(i sig.PubKeyIndex) (sig.Priv, error) {
	return NewSleepPriv(&ed.SchnorrStats, i)
}

func NewBLSPriv(i sig.PubKeyIndex) (sig.Priv, error) {
	p, err := NewSleepPriv(&bls.Stats, i)
	utils.PanicNonNil(err)

	p, err = NewSleepVrfPriv(p, &bls.Stats)
	if sig.UseMultisig {
		p, err = NewSleepMultiPriv(p, &bls.Stats)
		utils.PanicNonNil(err)
	}
	return p, err
}

func NewTBLSPriv(n, t int, i sig.PubKeyIndex) (sig.Priv, error) {
	p, err := NewBLSPriv(i)
	utils.PanicNonNil(err)
	return NewSleepThrshPriv(n, t, p, &bls.Stats)
}

func NewTBLSDualPriv(i sig.PubKeyIndex, to types.TestOptions) (sig.Priv, error) {
	coinType := types.NormalSignature
	if types.UseTp1CoinThresh(to) {
		coinType = types.SecondarySignature
	}
	thrshPrimary, thrshSecondary := sig.GetDSSThresh(to)
	thrshn := to.NumTotalProcs - to.NumNonMembers

	var err error
	var primary, secondary sig.Priv
	primary, err = NewTBLSPriv(thrshn, thrshPrimary, i)
	utils.PanicNonNil(err)
	secondary, err = NewTBLSPriv(thrshn, thrshSecondary, i)
	utils.PanicNonNil(err)

	var ret sig.Priv
	ret, err = dual.NewDualprivCustomThresh(primary, secondary, coinType, types.SecondarySignature)
	utils.PanicNonNil(err)

	return ret, nil
}

func NewCoinDualPriv(i sig.PubKeyIndex, to types.TestOptions) (sig.Priv, error) {
	thrsht := sig.GetCoinThresh(to)
	thrshn := to.NumTotalProcs - to.NumNonMembers

	var err error
	var p sig.Priv
	var primary, secondary sig.Priv
	primary, err = ec.NewEcpriv()
	utils.PanicNonNil(err)
	secondary, err = NewTBLSPriv(thrshn, thrsht, i)
	utils.PanicNonNil(err)
	p, err = dual.NewDualprivCustomThresh(primary, secondary, types.SecondarySignature,
		types.NormalSignature)
	utils.PanicNonNil(err)
	return p, err
}

func NewEDCoinPriv(i sig.PubKeyIndex, to types.TestOptions) (sig.Priv, error) {
	thrsht := sig.GetCoinThresh(to)
	thrshn := to.NumTotalProcs - to.NumNonMembers

	p := NewCoinSleepPriv(thrshn, thrsht, i, &ed.SchnorrStats)
	return p, nil
}

func NewQsafePriv(i sig.PubKeyIndex) (sig.Priv, error) {
	return NewSleepPriv(&qsafe.Stats, i)
}
