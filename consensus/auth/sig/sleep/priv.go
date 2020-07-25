package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

type sleepPriv interface {
	setPub(sig.Pub)
	sig.Priv
}

// Priv represents the sleep private key object
type Priv struct {
	pub   sig.Pub
	stats *sig.SigStats
	// rand *rand.Rand
	index sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
}

func (p *Priv) setPub(pub sig.Pub) {
	p.pub = pub
}

// Clean does nothing
func (p *Priv) Clean() {
}

// ComputeSharedSecret returns the hash of Diffie-Hellman.
func (p *Priv) ComputeSharedSecret(pub sig.Pub) (ret [32]byte) {
	return
}

// Shallow copy makes a copy of the object without following pointers.
func (p *Priv) ShallowCopy() sig.Priv {
	newPriv := *p
	newPriv.pub = newPriv.pub.ShallowCopy()
	return &newPriv
}

// NewSig returns an empty sig object of the same type.
func (p *Priv) NewSig() sig.Sig {
	return &Sig{stats: p.stats}
}

// GetBaseKey returns the same key.
func (p *Priv) GetBaseKey() sig.Priv {
	return p
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (p *Priv) SetIndex(index sig.PubKeyIndex) {
	p.index = index
	p.pub.SetIndex(index)
}

// GetPub returns the coreesponding sleep public key object
func (p *Priv) GetPub() sig.Pub {
	return p.pub
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (p *Priv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof, signType types.SignType) (*sig.SigItem, error) {
	if signType == types.CoinProof {
		if !p.stats.AllowsCoin {
			panic("config does not support coin")
		}
		if vrfProof != nil {
			panic("vrf not supported by with coin")
		}
		var err error
		m := messages.NewMessage(nil)
		_, err = p.GetPub().Serialize(m) // priv.SerializePub(m)
		if err != nil {
			return nil, err
		}
		coinProof := generateCoinProof(p, header)
		if _, err = coinProof.Serialize(m); err != nil {
			return nil, err
		}
		return &sig.SigItem{
			Pub:      p.GetPub(),
			Sig:      coinProof,
			SigBytes: m.GetBytes()}, nil
	}
	return sig.GenerateSigHelper(p, header, vrfProof, signType)
}

// Returns key that is used for signing the sign type.
func (p *Priv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	return p, nil
}

// Sign signs a message and returns the signature.
func (p *Priv) Sign(msg sig.SignedMessage) (sig.Sig, error) {
	hash := msg.GetSignedHash()
	if len(hash) != 32 {
		return nil, types.ErrInvalidHashSize
	}
	// we just use the hash of the message as the signature
	byt := make([]byte, p.stats.SigSize)
	copy(byt, msg.GetSignedHash())
	return &Sig{
		bytes: byt,
		stats: p.stats,
	}, nil
}
