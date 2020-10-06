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

package dual

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

type DualPub struct {
	sig.Pub
	sig.VRFPub
	pub2            sig.Pub
	useForCoin      types.SignType
	useForSecondary types.SignType
}

// New creates a new empty DualPub object.
func (dpp *DualPub) New() sig.Pub {
	ret := &DualPub{
		Pub:             dpp.Pub.New(),
		pub2:            dpp.pub2.New(),
		useForCoin:      dpp.useForCoin,
		useForSecondary: dpp.useForSecondary,
	}
	if vrfp, ok := ret.Pub.(sig.VRFPub); ok {
		ret.VRFPub = vrfp
	}
	return ret
}

func (pub *DualPub) SetPubs(pub1, pub2 sig.Pub) {
	pub.Pub = pub1
	pub.pub2 = pub2
	if vrfp, ok := pub.Pub.(sig.VRFPub); ok {
		pub.VRFPub = vrfp
	}
}

func (pub *DualPub) DeserializeSig(m *messages.Message, signType types.SignType) (*sig.SigItem, int, error) {
	if signType == types.NormalSignature ||
		(signType == types.CoinProof && pub.useForCoin == types.NormalSignature) ||
		(signType == types.SecondarySignature && pub.useForSecondary == types.NormalSignature) {

		return pub.Pub.DeserializeSig(m, signType)
	}
	return pub.pub2.DeserializeSig(m, signType)
}

// CheckSignature validates the signature with the public key.
// If the message type is a coin message, it is verified using CoinProof
func (pub *DualPub) CheckSignature(msg *sig.MultipleSignedMessage, sigItem *sig.SigItem) error {

	// Check if this is a coin proof or a signature
	signType := msg.GetSignType()
	// Validate the signature
	switch signType {
	case types.CoinProof:
		switch pub.useForCoin {
		case types.NormalSignature:
			return pub.Pub.CheckSignature(msg, sigItem)
		case types.SecondarySignature:
			return pub.pub2.CheckSignature(msg, sigItem)
		default:
			panic(pub.useForCoin)
		}
	case types.NormalSignature:
		return pub.Pub.CheckSignature(msg, sigItem)
	case types.SecondarySignature:
		switch pub.useForSecondary {
		case types.NormalSignature:
			return pub.Pub.CheckSignature(msg, sigItem)
		case types.SecondarySignature:
			return pub.pub2.CheckSignature(msg, sigItem)
		default:
			panic(pub.useForCoin)
		}
	default:
		panic(signType)
	}
}

// SetIndex sets the index of both the pubs
func (dpp *DualPub) SetIndex(idx sig.PubKeyIndex) {
	dpp.Pub.SetIndex(idx)
	dpp.pub2.SetIndex(idx)
}

// VerifySecondarySig calls VerifySig using the secondary public key.
func (dpp *DualPub) VerifySecondarySig(msg sig.SignedMessage, sign sig.Sig) (bool, error) {
	return dpp.pub2.VerifySig(msg, sign)
}

// ShallowCopy makes a copy of the object without following pointers.
func (dpp *DualPub) ShallowCopy() sig.Pub {
	ret := *dpp
	ret.Pub = dpp.Pub.ShallowCopy()
	ret.pub2 = dpp.pub2.ShallowCopy()
	return &ret
}

// GetRealPubBytes returns a buffer containing the results of GetRealPubBytes on both of the keys
func (dpp *DualPub) GetRealPubBytes() (ret sig.PubKeyBytes, err error) {
	buff := bytes.NewBuffer(nil)

	pb1, err := dpp.Pub.GetRealPubBytes()
	if err != nil {
		return nil, err
	}
	pb2, err := dpp.pub2.GetRealPubBytes()
	if err != nil {
		return nil, err
	}

	if _, err = utils.EncodeUvarint(uint64(len(pb1)), buff); err != nil {
		return
	}
	buff.Write(pb1)
	buff.Write(pb2)
	return buff.Bytes(), nil
}

// FromPubBytes creates a pub from the bytes created by GetRealPubBytes
func (dpp *DualPub) FromPubBytes(inBuff sig.PubKeyBytes) (sig.Pub, error) {
	ret := dpp.New().(*DualPub)
	buff := bytes.NewBuffer(inBuff)

	siz, _, err := utils.ReadUvarint(buff)
	if err != nil {
		return nil, err
	}

	p1, err := ret.Pub.FromPubBytes(buff.Next(int(siz)))
	if err != nil {
		return nil, err
	}
	p2, err := ret.pub2.FromPubBytes(buff.Bytes())
	if err != nil {
		return nil, err
	}
	ret.Pub = p1
	if vrfp, ok := ret.Pub.(sig.VRFPub); ok {
		ret.VRFPub = vrfp
	}
	ret.pub2 = p2
	return ret, nil
}
