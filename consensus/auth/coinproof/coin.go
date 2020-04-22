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
package coinproof

import (
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/proof/dleq"
	"io"
)

type CoinProof struct {
	suite     dleq.Suite
	newScalar func() kyber.Scalar
	newPoint  func() kyber.Point
	prf       *dleq.Proof
	xG, xH    kyber.Point
}

func EmptyCoinProof(suite dleq.Suite) *CoinProof {
	return &CoinProof{suite: suite, newScalar: suite.Scalar, newPoint: suite.Point}
}

func (cp *CoinProof) New() *CoinProof {
	return EmptyCoinProof(cp.suite)
}

func CreateCoinProof(suite dleq.Suite,
	sharedKey kyber.Point, msg []byte,
	priv kyber.Scalar) (ret *CoinProof, err error) {

	hasher := suite.Hash()
	hasher.Write(msg)
	hsh := hasher.Sum(nil)
	hSca := suite.Scalar().SetBytes(hsh)
	hPt := suite.Point().Mul(hSca, nil)

	ret = &CoinProof{suite: suite, newScalar: suite.Scalar, newPoint: suite.Point}
	ret.prf, ret.xG, ret.xH, err = dleq.NewDLEQProof(suite, sharedKey, hPt, priv)
	return
}

func (cp *CoinProof) GetShare() kyber.Point {
	return cp.xH
}

func (cp *CoinProof) Validate(localKey, sharedKey kyber.Point, msg []byte) error {
	hasher := cp.suite.Hash()
	hasher.Write(msg)
	hsh := hasher.Sum(nil)
	hSca := cp.suite.Scalar().SetBytes(hsh)
	hPt := cp.suite.Point().Mul(hSca, nil)

	// First check the local key
	check := cp.suite.Point().Mul(hSca, localKey)
	if !cp.xH.Equal(check) {
		return types.ErrInvalidSig
	}

	return cp.prf.Verify(cp.suite, sharedKey, hPt, cp.xG, cp.xH)
}

// GetID returns the header id for EDDSA pub objects
func (cp *CoinProof) GetID() messages.HeaderID {
	return messages.HdrCoinProof
}

// Serialize appends a serialized header to the message m, and returns the size of bytes written
func (cp *CoinProof) Serialize(m *messages.Message) (int, error) {
	return messages.SerializeHelper(cp.GetID(), cp, m)
}

// Deserialize deserialzes a header into the object, returning the number of bytes read
func (cp *CoinProof) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	return messages.DeserializeHelper(cp.GetID(), cp, m)
}

// GetMsgID returns the message id for a coin proof.
func (cp *CoinProof) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(cp.GetID())
}

// PeekHeader returns nil.
func (CoinProof) PeekHeaders(*messages.Message, types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	return
}

// GetBytes returns the bytes of the coin proof from the message.
func (cp *CoinProof) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

func (cp *CoinProof) Encode(writer io.Writer) (n int, err error) {

	var l1 int
	if l1, err = cp.prf.VH.MarshalTo(writer); err != nil {
		return
	}
	n += l1
	if l1, err = cp.prf.VG.MarshalTo(writer); err != nil {
		return
	}
	n += l1
	if l1, err = cp.prf.R.MarshalTo(writer); err != nil {
		return
	}
	n += l1
	if l1, err = cp.prf.C.MarshalTo(writer); err != nil {
		return
	}
	n += l1
	if l1, err = cp.xH.MarshalTo(writer); err != nil {
		return
	}
	n += l1
	if l1, err = cp.xG.MarshalTo(writer); err != nil {
		return
	}
	n += l1

	return
}

func (cp *CoinProof) Decode(reader io.Reader) (n int, err error) {
	var l1 int
	cp.prf = &dleq.Proof{}

	cp.prf.VH = cp.newPoint()
	if l1, err = cp.prf.VH.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	cp.prf.VG = cp.newPoint()
	if l1, err = cp.prf.VG.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	cp.prf.R = cp.newScalar()
	if l1, err = cp.prf.R.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	cp.prf.C = cp.newScalar()
	if l1, err = cp.prf.C.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	cp.xH = cp.newPoint()
	if l1, err = cp.xH.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	cp.xG = cp.newPoint()
	if l1, err = cp.xG.UnmarshalFrom(reader); err != nil {
		return
	}
	n += l1

	return
}
