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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
)

type DualPriv struct {
	sig.Priv
	sig.ThreshStateInterface
	Priv2           sig.Priv
	pub             *DualPub
	useForCoin      types.SignType
	useForSecondary types.SignType
}

func NewDualprivCustomThresh(priv1, priv2 sig.Priv, useForCoin, useForSecondary types.SignType) (sig.Priv, error) {
	var ret DualPriv
	ret.Priv = priv1
	ret.Priv2 = priv2
	ret.useForCoin = useForCoin
	ret.useForSecondary = useForSecondary

	switch useForCoin { // Sanity check
	case types.NormalSignature, types.SecondarySignature:
	default:
		panic(useForCoin)
	}
	switch useForSecondary { // Sanity check
	case types.NormalSignature, types.SecondarySignature:
	default:
		panic(useForCoin)
	}

	if thrsh, ok := ret.Priv.(sig.ThreshStateInterface); ok {
		ret.ThreshStateInterface = thrsh
	}

	ret.pub = &DualPub{
		useForSecondary: useForSecondary,
		useForCoin:      useForCoin,
		Pub:             ret.Priv.GetPub(),
		pub2:            ret.Priv2.GetPub(),
	}
	return &ret, nil
}

func NewDualpriv(newPriv1 func() (sig.Priv, error), newPriv2 func() (sig.Priv, error),
	useForCoin, useForSecondary types.SignType) (sig.Priv, error) {
	priv1, err := newPriv1()
	if err != nil {
		return nil, err
	}
	priv2, err := newPriv2()
	if err != nil {
		return nil, err
	}
	return NewDualprivCustomThresh(priv1, priv2, useForCoin, useForSecondary)
}

func (priv *DualPriv) ShallowCopy() sig.Priv {
	ret := *priv
	return &ret
}

func (priv *DualPriv) SetIndex(idx sig.PubKeyIndex) {
	priv.Priv.SetIndex(idx)
	priv.Priv2.SetIndex(idx)
}

func (priv *DualPriv) GetPub() sig.Pub {
	return priv.pub
}

func (priv *DualPriv) GetSecondaryPriv() sig.Priv {
	return priv.Priv2
}

func (priv *DualPriv) ComputeSharedSecret(pub sig.Pub) [32]byte {
	return priv.Priv.ComputeSharedSecret(pub.(*DualPub).Pub)
}

// GenerateSig signs a message and returns the SigItem object containing the signature
func (priv *DualPriv) GenerateSig(header sig.SignedMessage, vrfProof sig.VRFProof,
	signType types.SignType) (*sig.SigItem, error) {

	if signType == types.NormalSignature ||
		(signType == types.CoinProof && priv.useForCoin == types.NormalSignature) ||
		(signType == types.SecondarySignature && priv.useForSecondary == types.NormalSignature) {

		return priv.Priv.GenerateSig(header, vrfProof, signType)
	}
	return priv.Priv2.GenerateSig(header, vrfProof, signType)
}

// Returns key that is used for signing the sign type.
func (priv *DualPriv) GetPrivForSignType(signType types.SignType) (sig.Priv, error) {
	if signType == types.NormalSignature ||
		(signType == types.CoinProof && priv.useForCoin == types.NormalSignature) ||
		(signType == types.SecondarySignature && priv.useForSecondary == types.NormalSignature) {

		return priv.Priv, nil
	}
	return priv.Priv2, nil
}
