package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"time"
)

// VRFPub for public keys that support VRFs
type vRFPub struct {
	stats *sig.SigStats
}

// NewVRFProof returns an empty VRFProof object
func (vrfPub *vRFPub) NewVRFProof() sig.VRFProof {
	return (&VRFProof{size: vrfPub.stats.VRFSize}).New()
}

// ProofToHash checks the VRF for the message and returns the random bytes if valid
func (vrfPub *vRFPub) ProofToHash(_ sig.SignedMessage, proof sig.VRFProof) (index [32]byte, err error) {
	copy(index[:], proof.(*VRFProof).buff)
	time.Sleep(vrfPub.stats.VRFVerifyTime)
	return
}

type vrfPriv struct {
	stats *sig.SigStats
	pub   sig.Pub
}

// Evaluate generates a vrf. The index and thee first 32 bytes are the same.
// They are the hash of the key index and the message.
func (p *vrfPriv) Evaluate(m sig.SignedMessage) (index [32]byte, proof sig.VRFProof) {
	h := types.GetNewHash()
	id, err := p.pub.GetPubID()
	if err != nil {
		panic(err)
	}
	h.Write([]byte(id))
	h.Write(m.GetSignedMessage())
	copy(index[:], h.Sum(nil))
	prf := &VRFProof{size: p.stats.VRFSize}
	prf.buff = make([]byte, prf.size)
	copy(prf.buff, index[:])
	proof = prf

	time.Sleep(p.stats.VRFGenTime)
	return
}

type VRFProof struct {
	buff []byte
	size int
}

func (prf *VRFProof) Encode(writer io.Writer) (n int, err error) {
	return writer.Write(prf.buff)
}
func (prf *VRFProof) Decode(reader io.Reader) (n int, err error) {
	if len(prf.buff) != prf.size {
		panic("invalid alloc")
	}
	return reader.Read(prf.buff)
}

func (prf *VRFProof) New() sig.VRFProof {
	return &VRFProof{
		buff: make([]byte, prf.size),
		size: prf.size,
	}
}
