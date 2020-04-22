// +build !windows

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

package qsafe

import (
	"github.com/open-quantum-safe/liboqs-go/oqs"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"time"
)

// QsafePub represents an quantum safe public key
type QsafePub struct {
	algDetails oqs.SignatureDetails
	pubBytes   sig.PubKeyBytes // The marshalled bytes of the public key
	pubString  sig.PubKeyStr   // Same as pubBytes, except a string
	index      sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
	pubID      sig.PubKeyID    // A unique string represeting this key for a consensus iteration (see PubKeyID type)
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *QsafePub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// NewVRFProof is not supported.
func (pub *QsafePub) NewVRFProof() sig.VRFProof {
	panic("VRF not supported")
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *QsafePub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

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

// FromPubBytes generates an public key object from the bytes of a public key
func (pub *QsafePub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p := pub.New().(*QsafePub)
	p.pubBytes = b
	p.pubString = sig.PubKeyStr(p.pubBytes)

	return p, nil
}

// PeekHeader returns nil.
func (*QsafePub) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {

	return
}

// GetMsgID returns the message id for an qsafe public key
func (pub *QsafePub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

// GetSigMemberNumber always returns one since qsafe signautres can only represent a single signer
func (pub *QsafePub) GetSigMemberNumber() int {
	return 1 // always 1 for qsafe
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (pub *QsafePub) SetIndex(index sig.PubKeyIndex) {
	pub.pubID = ""
	pub.index = index
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *QsafePub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.index
}

// New returns a blank qsafe public key object
func (pub *QsafePub) New() sig.Pub {
	return &QsafePub{algDetails: pub.algDetails}
}

// DeserializeSig takes a message and returns an qsafe public key object and signature as well as the number of bytes read
func (pub *QsafePub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
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

	ret.Sig = &QsafeSig{}
	l1, err = ret.Sig.Deserialize(m, types.NilIndexFuns)
	l += l1
	return ret, l, err
}

var qSafeVerifyTime = 88207 * time.Nanosecond // expected time to verify a qsafe sig on a Intel(R) Core(TM) i7-6820HQ CPU @ 2.70GHz

// VerifySig verifies that sig is a valid signature for msg by qsafe public key pub
func (pub *QsafePub) VerifySig(msg sig.SignedMessage, aSig sig.Sig) (bool, error) {
	switch v := aSig.(type) {
	case *QsafeSig:
		if sig.SleepValidate {
			sig.DoSleepValidation(qSafeVerifyTime)
			return true, nil
		}
		var s oqs.Signature
		s.Init(QsafeName, nil)
		vaild, err := s.Verify(msg.GetSignedMessage(), v.sigBytes, pub.pubBytes)
		s.Clean()
		return vaild, err
	default:
		return false, types.ErrInvalidSigType
	}
}

// ProofToHash is not supported.
func (pub *QsafePub) ProofToHash(m sig.SignedMessage, proof sig.VRFProof) (index [32]byte, err error) {
	panic("VRF not supported")
}

// GetRealPubBytes returns the pub key as bytes (same as GetPubBytes for keys)
func (pub *QsafePub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	return pub.GetPubBytes()
}

// GetPubBytes returns the pub key as bytes (same as GetRealPubBytes for keys)
func (pub *QsafePub) GetPubBytes() (sig.PubKeyBytes, error) {
	return pub.pubBytes, nil
}

// GetPubID returns the unique id for this pubkey (given some consensus instance), it could be the encoded bitid, or just the pub key
// depending on how SetUsePubIndex was set
func (pub *QsafePub) GetPubID() (sig.PubKeyID, error) {
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
func (pub *QsafePub) GetPubString() (sig.PubKeyStr, error) {
	var err error
	if pub.pubString == "" {
		pub.pubString = sig.PubKeyStr(pub.pubBytes)
	}
	return pub.pubString, err
}

func (pub *QsafePub) Encode(writer io.Writer) (n int, err error) {
	buff, err := pub.GetPubBytes()
	if err != nil {
		return 0, err
	}
	if len(buff) != pub.algDetails.LengthPublicKey {
		panic(len(buff))
	}
	return writer.Write(buff)
}
func (pub *QsafePub) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, pub.algDetails.LengthPublicKey)
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

// Serialize the pub key into the message, return the number of bytes written
func (pub *QsafePub) Serialize(m *messages.Message) (int, error) {
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

// GetBytes returns the bytes of the qsafe public key from the message
func (pub *QsafePub) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Deserialize updates the fields of the pub key object from m, and returns the number of bytes read
func (pub *QsafePub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
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

	return l, nil
}

// GetID returns the header id for pub objects
func (pub *QsafePub) GetID() messages.HeaderID {
	return messages.HdrQsafePub
}
