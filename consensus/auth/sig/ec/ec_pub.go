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

package ec

import (
	// "fmt"
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/tcrain/cons/consensus/auth/coinproof"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"time"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/messages"
)

////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////

// the ecCurve used for ECDSA sigs
var ecCurve = elliptic.P256()
var ecPointsEncodedLen = 1 + 2*((ecCurve.Params().BitSize+7)>>3)
var ecVRFlen = 64 + 65

///////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////

type VRFProof []byte

func (prf VRFProof) Encode(writer io.Writer) (n int, err error) {
	return writer.Write(prf)
}
func (prf VRFProof) Decode(reader io.Reader) (n int, err error) {
	if len(prf) != ecVRFlen {
		panic("invalid alloc")
	}
	return reader.Read(prf)
}

func (prf VRFProof) New() sig.VRFProof {
	return make(VRFProof, ecVRFlen)
}

// Ecpub represents an ECDSA public key
type Ecpub struct {
	pub       *p256.PublicKey // The public key
	pubBytes  sig.PubKeyBytes // The marshalled bytes of the public key
	pubString sig.PubKeyStr   // Same as pubBytes, except a string
	index     sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
	pubID     sig.PubKeyID    // A unique string represeting this key for a consensus iteration (see PubKeyID type)
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *Ecpub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *Ecpub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	signType := msg.GetSignType()
	if signType == types.CoinProof || sigItem.CoinProof != nil {
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

// NewVRFProof returns an empty VRFProof object, and is not supported for ed.
func (pub *Ecpub) NewVRFProof() sig.VRFProof {
	return VRFProof(make([]byte, ecVRFlen))
}

func (*Ecpub) DeserializeCoinProof(*messages.Message) (coinProof *coinproof.CoinProof, size int, err error) {
	panic("unsupported")
}

// FromPubBytes generates an ECDSA public key object from the bytes of a public key
func (pub *Ecpub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p := pub.New().(*Ecpub)
	p.pubBytes = b
	p.pubString = sig.PubKeyStr(p.pubBytes)
	_, err := p.getPub()
	return p, err
}

// PeekHeader returns nil.
func (*Ecpub) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	return
}

// GetMsgID returns the message id for an ECDSA public key
func (pub *Ecpub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

// GetSigMemberNumber always returns one since ECDSA signautres can only represent a single signer
func (pub *Ecpub) GetSigMemberNumber() int {
	return 1 // always 1 for ec
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (pub *Ecpub) SetIndex(index sig.PubKeyIndex) {
	pub.pubID = ""
	pub.index = index
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *Ecpub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndxe is false")
	}
	return pub.index
}

// New returns a blank ECDSA public key object
func (pub *Ecpub) New() sig.Pub {
	return &Ecpub{}
}

// DeserializeSig takes a message and returns an ECDSA public key object and signature as well as the number of bytes read
func (pub *Ecpub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
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

	ret.Sig = &Ecsig{}
	l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
	l += l1
	return ret, l, err
}

var ecVerfiyTime = 82688 * time.Nanosecond // expected time to verify a ecdsa sig on a Intel(R) Core(TM) i7-6820HQ CPU @ 2.70GHz

// VerifySig verifies that sig is a valid signature for msg by ECDSA public key pub
func (pub *Ecpub) VerifySig(msg sig.SignedMessage, aSig sig.Sig) (bool, error) {
	switch v := aSig.(type) {
	case *Ecsig:
		if sig.SleepValidate {
			sig.DoSleepValidation(ecVerfiyTime)
			return true, nil
		}
		p, err := pub.getPub()
		if err != nil {
			return false, err
		}
		return ecdsa.Verify(p, msg.GetSignedHash(), v.R, v.S), nil
	default:
		return false, types.ErrInvalidSigType
	}
}

func (pub *Ecpub) getPub() (*ecdsa.PublicKey, error) {
	var err error
	if pub.pub == nil {
		x, y := elliptic.Unmarshal(ecCurve, pub.pubBytes)
		if x == nil {
			return nil, types.ErrInvalidPub
		}
		pub.pub = &p256.PublicKey{PublicKey: &ecdsa.PublicKey{}}
		pub.pub.X = x
		pub.pub.Y = y
		pub.pub.Curve = ecCurve
		_, err = p256.NewVRFVerifier(pub.pub.PublicKey)
	}
	return pub.pub.PublicKey, err
}

// ProofToHash is for validating VRFs.
func (pub *Ecpub) ProofToHash(m sig.SignedMessage, proof sig.VRFProof) (index [32]byte, err error) {
	return pub.pub.ProofToHash(m.GetSignedMessage(), proof.(VRFProof))
}

// GetRealPubBytes returns the ECDSA pub key as bytes (same as GetPubBytes for ECDSA keys)
func (pub *Ecpub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	return pub.GetPubBytes()
}

// GetPubBytes returns the ECDSA pub key as bytes (same as GetRealPubBytes for ECDSA keys)
func (pub *Ecpub) GetPubBytes() (sig.PubKeyBytes, error) {
	if pub.pubBytes == nil {
		pub.pubBytes = elliptic.Marshal(ecCurve, pub.pub.X, pub.pub.Y)
		pub.pubString = sig.PubKeyStr(pub.pubBytes)
	}
	return pub.pubBytes, nil
}

// GetPubID returns the unique id for this pubkey (given some consensus instance), it could be the encoded bitid, or just the pub key
// depending on how SetUsePubIndex was set
func (pub *Ecpub) GetPubID() (sig.PubKeyID, error) {
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

// GetPubString is the same as GetPubBytes, except returns a string
func (pub *Ecpub) GetPubString() (sig.PubKeyStr, error) {
	var err error
	if pub.pubBytes == nil {
		_, err = pub.GetPubBytes()
	}
	return pub.pubString, err
}

func (pub *Ecpub) Encode(writer io.Writer) (n int, err error) {
	buff, err := pub.GetPubBytes()
	if err != nil {
		return 0, err
	}
	if len(buff) != ecPointsEncodedLen {
		panic(len(buff))
	}
	return writer.Write(buff)
}
func (pub *Ecpub) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, ecPointsEncodedLen)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	if n != len(buf) {
		panic(n)
	}
	pub.pubBytes = buf
	pub.pubString = sig.PubKeyStr(buf)
	_, err = pub.getPub()
	return
}

// Serialize the pub key into the message, return the number of bytes written
func (pub *Ecpub) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, pub.GetID(), 0, (*messages.MsgBuffer)(m))

	// now the pub
	if sig.UsePubIndex {
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

// GetBytes returns the bytes of the ECDSA public key from the message
func (pub *Ecpub) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Deserialize updates the fields of the pub key object from m, and returns the number of bytes read
func (pub *Ecpub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(pub.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	// Now the pub
	if sig.UsePubIndex {
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

	pub.pub = nil
	return l, nil
}

// GetID returns the header id for ECDSA pub objects
func (pub *Ecpub) GetID() messages.HeaderID {
	return messages.HdrEcpub
}
