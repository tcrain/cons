package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"io"
	"math/rand"
)

var AllSigStats = []sig.SigStats{ec.Stats, ed.EDDSAStats, ed.SchnorrStats, bls.Stats}

// rnd is for generating static keys for tests
var rnd = rand.New(rand.NewSource(1))

func GetAllSigStatsNewPriv(onlyVrf, onlyMulti bool) (ret []func() (sig.Priv, error)) {
	for _, nxt := range AllSigStats {
		if onlyVrf && !nxt.AllowsVRF {
			continue
		}
		if onlyMulti && !nxt.AllowsMulti {
			continue
		}
		var nxtFunc func() (sig.Priv, error)
		nxtFunc = func() (sig.Priv, error) {
			p, err := NewSleepPriv(&nxt, rnd)
			if err != nil {
				return nil, err
			}
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
func NewSleepPriv(stats *sig.SigStats, rnd io.Reader) (sig.Priv, error) {
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
	sleepPriv
}

// Sign signs a message and returns the signature.
func (p *MultiPriv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	s, err := p.sleepPriv.Sign(msg)
	if err != nil {
		return nil, err
	}
	return &multiSig{Sig: *s.(*Sig)}, nil
}

func (p *MultiPriv) ShallowCopy() sig.Priv {
	newP := *p
	newP.sleepPriv = p.sleepPriv.ShallowCopy().(sleepPriv)
	return &newP
}

func NewSleepMultiPriv(priv sig.Priv, stats *sig.SigStats) (sig.Priv, error) {
	priv.(sleepPriv).setPub(newMultiPub(priv.GetPub(), stats))
	return &MultiPriv{
		sleepPriv: priv.(sleepPriv),
	}, nil
}

func NewSleepVrfPriv(priv sig.Priv, stats *sig.SigStats) (sig.Priv, error) {
	p := &VrfPub{
		Pub:    priv.GetPub(),
		vRFPub: vRFPub{stats: stats},
	}
	priv.(sleepPriv).setPub(p)
	return &VrfPriv{
		sleepPriv: priv.(sleepPriv),
		vrfPriv: vrfPriv{
			stats: stats,
			pub:   p,
		},
	}, nil
}

type VrfPriv struct {
	sleepPriv
	vrfPriv
}

func (p *VrfPriv) ShallowCopy() sig.Priv {
	newP := *p
	newP.sleepPriv = p.sleepPriv.ShallowCopy().(sleepPriv)
	return &newP
}

type VrfPub struct {
	sig.Pub
	vRFPub
}

// New creates an empty sleep private key object
func (p *VrfPub) New() sig.Pub {
	return &VrfPub{Pub: p.Pub.New(),
		vRFPub: p.vRFPub,
	}
}

func (pub *VrfPub) ShallowCopy() sig.Pub {
	newPub := *pub
	newPub.Pub = pub.Pub.ShallowCopy()
	return &newPub
}

// FromPubBytes creates a public key object from the public key bytes
func (pub *VrfPub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p, err := pub.Pub.FromPubBytes(b)
	if err != nil {
		return p, err
	}
	return &VrfPub{
		Pub:    p,
		vRFPub: pub.vRFPub,
	}, nil
}

func NewSleepThrshPriv(n, t int, priv sig.Priv) (sig.Priv, error) {
	p := priv.(*Priv)
	return &Thrsh{
		n:         n,
		t:         t,
		idx:       p.index,
		Priv:      p,
		sharedPub: newSharedPub(p.stats, t),
	}, nil
}
