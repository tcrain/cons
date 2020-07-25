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

package ed

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/coinproof"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/sign/eddsa"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"io"
)

///////////////////////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////////////////////

var edpubmarshallsize = sig.EdSuite.PointLen()

// Edpub represents an EDDSA public key
type Edpub struct {
	pub          kyber.Point     // The point on the ecCurve
	pubBytes     sig.PubKeyBytes // The marshalled point
	pubString    sig.PubKeyStr   // The marshalled point (as a string)
	pubID        sig.PubKeyID    // A unique string represeting this key for a consensus iteration (see PubKeyID type)
	memberNumber int             // The number of signers that this key represents (greater than one incase it is a combined threshold signautre)
	useIndex     bool            // Whether or not to use the index as the identifier or the whole key (this will always be set from SetUseMultisig, except for combined threshold sigs for which this must be true
	pubEdtype    edtype          // The type of ed pub key
	index        sig.PubKeyIndex // The index of this key in the sorted list of public keys participating in a consensus
}

// Shallow copy makes a copy of the object without following pointers.
func (pub *Edpub) ShallowCopy() sig.Pub {
	newPub := *pub
	return &newPub
}

// NewEdThresholdPub creates a new threshold type EDDSA public key
// the only different for threshold pubs is that they are given index -1
func NewEdThresholdPub(pub kyber.Point, memberNumber int) (*Edpub, error) {
	ed := &Edpub{pub: pub, pubEdtype: schnorrType, useIndex: true, index: -1}
	ed.memberNumber = memberNumber
	_, err := ed.GetPubBytes()
	if err != nil {
		return ed, err
	}
	_, err = ed.GetPubID() // to populate values
	return ed, err
}

/*// NewSchnorrThresholdPub creates a new threshold type schnorr public key
// the only different for threshold pubs is that they are given index -1
func NewSchnorrThresholdPub(pub kyber.Point, memberNumber int) (*Edpub, error) {
	ed := &Edpub{pub: pub, pubEdtype: schnorrType, useIndex: true, index: -1}
	ed.memberNumber = memberNumber
	_, err := ed.GetPubBytes()
	if err != nil {
		return ed, err
	}
	_, err = ed.GetPubID() // to populate values
	return ed, err
}
*/
// NewEdpub creates a new a new EDDSA type public key
func NewEdpub(pub kyber.Point) (*Edpub, error) {
	ed := &Edpub{pub: pub, pubEdtype: eddsaType, useIndex: sig.UsePubIndex}
	ed.memberNumber = 1
	_, err := ed.GetPubBytes()
	return ed, err
}

// NewEdpub creates a new a new Schnorr type public key
func NewSchnorrpub(pub kyber.Point) (*Edpub, error) {
	ed := &Edpub{pub: pub, pubEdtype: schnorrType, useIndex: sig.UsePubIndex}
	ed.memberNumber = 1
	_, err := ed.GetPubBytes()
	return ed, err
}

// FromPubBytes generates an EDDSA public key object from the bytes of a public key
func (pub *Edpub) FromPubBytes(b sig.PubKeyBytes) (sig.Pub, error) {
	p := pub.New().(*Edpub)
	p.pubBytes = b
	_, err := p.getPub()
	p.pubString = sig.PubKeyStr(p.pubBytes)
	return p, err
}

// GetMsgID returns the message id for an EDDSA public key
func (pub *Edpub) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(pub.GetID())
}

// SetIndex sets the index of the node represented by this key in the consensus participants
func (pub *Edpub) SetIndex(index sig.PubKeyIndex) {
	pub.pubID = ""
	pub.index = index
}

// GetIndex gets the index of the node represented by this key in the consensus participants
func (pub *Edpub) GetIndex() sig.PubKeyIndex {
	if !sig.UsePubIndex {
		panic("invalid when UsePubIndex is false")
	}
	return pub.index
}

// GetSigMemberNumber returns the number of signers this signature represents
// this can be more than one as once combined, threshold signatures are EDDSA signatures
func (pub *Edpub) GetSigMemberNumber() int {
	if pub.memberNumber == 0 {
		return 1
	}
	return pub.memberNumber
}

// New returns a blank EDDSA public key object
func (pub *Edpub) New() sig.Pub {
	return &Edpub{pubEdtype: pub.pubEdtype, useIndex: sig.UsePubIndex}
}

func (pub *Edpub) DeserializeCoinProof(m *messages.Message) (coinProof *coinproof.CoinProof, size int, err error) {
	coinProof = coinproof.EmptyCoinProof(sig.EdSuite)
	size, err = coinProof.Decode((*messages.MsgBuffer)(m))
	return
}

// CheckSignature validates the signature with the public key, it returns an error if a coin proof is included.
func (pub *Edpub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	signType := msg.GetSignType()
	if signType == types.CoinProof {
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

// DeserializeSig takes a message and returns an EDDSA public key object and signature as well as the number of bytes read
func (pub *Edpub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.CoinProof {
		panic("coin proof not supported")
	}
	var n int
	var l1 int
	var err error
	newPub := pub.New()
	l1, err = newPub.Deserialize(m, types.NilIndexFuns)
	if err != nil {
		return nil, n, err
	}
	n += l1

	asig := &Edsig{sigEdtype: pub.pubEdtype}
	l1, err = asig.Deserialize(m, types.NilIndexFuns)
	if err != nil {
		return nil, n, err
	}
	n += l1
	return &sig.SigItem{Pub: newPub, Sig: asig}, n, nil
}

// VerifySig verifies that sig is a valid signature for msg by EDDSA public key pub
func (pub *Edpub) VerifySig(msg sig.SignedMessage, asig sig.Sig) (bool, error) {
	switch v := asig.(type) {
	case *Edsig:
		point, err := pub.getPub()
		if err != nil {
			return false, err
		}
		if sig.SleepValidate {
			sig.DoSleepValidation(edPartialVerifyTime)
			return true, nil
		}
		switch pub.pubEdtype {
		case eddsaType:
			err = eddsa.Verify(point, msg.GetSignedMessage(), v.sig)
		case schnorrType:
			err = schnorr.Verify(sig.EdSuite, point, msg.GetSignedMessage(), v.sig)
		default:
			panic(pub.pubEdtype)
		}
		if err == nil {
			return true, nil
		}
		return false, err
	default:
		return false, types.ErrInvalidSigType
	}
}

func (pub *Edpub) getPub() (kyber.Point, error) {
	if pub.pub == nil {
		switch pub.pubEdtype {
		case eddsaType:
			pub.pub = sig.EddsaGroup.Point()
		case schnorrType:
			pub.pub = sig.EdSuite.Point()
		default:
			panic("invalid ed type")
		}
		err := pub.pub.UnmarshalBinary(pub.pubBytes)
		if err != nil {
			return nil, err
		}
	}
	return pub.pub, nil
}

// GetRealPubBytes returns the EDDSA pub key as bytes (same as GetPubBytes for EDDSA keys)
func (pub *Edpub) GetRealPubBytes() (sig.PubKeyBytes, error) {
	return pub.GetPubBytes()
}

// GetPubBytes returns the EDDSA pub key as bytes (same as GetRealPubBytes for ECDSA keys)
func (pub *Edpub) GetPubBytes() (sig.PubKeyBytes, error) {
	if pub.pubBytes == nil {
		var err error
		pub.pubBytes, err = pub.pub.MarshalBinary()
		if err != nil {
			return nil, err
		}
		pub.pubString = sig.PubKeyStr(pub.pubBytes)
	}
	return pub.pubBytes, nil
}

// GetPubID returns the unique id for this pubkey (given some consensus instance), it could be the encoded bitid, or just the pub key
// depending on how SetUsePubIndex was set
func (pub *Edpub) GetPubID() (sig.PubKeyID, error) {
	if pub.useIndex {
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
func (pub *Edpub) GetPubString() (sig.PubKeyStr, error) {
	if pub.pubBytes == nil {
		_, err := pub.GetPubBytes()
		if err != nil {
			return "", err
		}
	}
	return pub.pubString, nil
}

func (pub *Edpub) Encode(writer io.Writer) (n int, err error) {
	pb, err := pub.GetPubBytes()
	if err != nil {
		return 0, err
	}
	return writer.Write(pb)
}

func (pub *Edpub) Decode(reader io.Reader) (n int, err error) {
	buf := make([]byte, edpubmarshallsize)
	n, err = reader.Read(buf)
	if err != nil {
		return
	}
	pub.pubBytes = buf
	pub.pubString = sig.PubKeyStr(buf)
	pub.pub = nil
	return
}

// Serialize the pub key into the message, return the number of bytes written
func (pub *Edpub) Serialize(m *messages.Message) (int, error) {
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

// GetBytes returns the bytes of the EDDSA public key from the message
func (pub *Edpub) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// PeekHeader returns nil.
func (Edpub) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// Deserialize updates the fields of the pub key object from m, and returns the number of bytes read
func (pub *Edpub) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarshall function
	l, _, _, size, _, err := messages.ReadHeaderHead(pub.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return 0, err
	}

	if sig.UsePubIndex {
		index, br, err := (*messages.MsgBuffer)(m).ReadUint32()
		if err != nil {
			return 0, err
		}
		l += br
		pub.index = sig.PubKeyIndex(index)
	} else {
		// Now the pub
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

// GetID returns the header id for EDDSA pub objects
func (pub *Edpub) GetID() messages.HeaderID {
	switch pub.pubEdtype {
	case eddsaType:
		return messages.HdrEdpub
	case schnorrType:
		return messages.HdrSchnorrpub
	default:
		panic("invalid pub type")
	}
}
