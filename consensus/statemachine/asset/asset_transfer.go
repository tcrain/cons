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

package asset

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"sort"
)

type AssetTransfer struct {
	Sender    sig.Pub           // Who is sending the assets.
	Inputs    []types.HashBytes // The input assets.
	Outputs   []AssetInterface  // The output assets.
	Receivers []sig.Pub         // The receivers of the output assets.
	Signature sig.Sig           // The signature of the sender.

	// For encoding.
	NewAssetFunc func() AssetInterface
	NewSigFunc   func() sig.Sig
	NewPubFunc   func() sig.Pub
	SignedBytes  []byte // The bytes to sign.
}

func NewEmptyAssetTransfer(newAssetFunc func() AssetInterface, newSigFunc func() sig.Sig,
	newPubFunc func() sig.Pub) *AssetTransfer {
	return &AssetTransfer{
		NewAssetFunc: newAssetFunc,
		NewSigFunc:   newSigFunc,
		NewPubFunc:   newPubFunc,
	}
}

func NewAssetTransfer(sender sig.Pub, inputs []types.HashBytes, outputs []AssetInterface,
	receivers []sig.Pub) *AssetTransfer {

	return &AssetTransfer{
		Sender:    sender,
		Inputs:    inputs,
		Outputs:   outputs,
		Receivers: receivers,
	}
}

// New creates an empty asset transfer object.
func (at *AssetTransfer) New() types.EncodeInterface {
	return &AssetTransfer{
		Sender:       at.Sender.New(),
		NewSigFunc:   at.NewSigFunc,
		NewAssetFunc: at.NewAssetFunc,
	}
}

func CheckUnique(assets []AssetInterface) error {
	sorted := make(SortedAssets, len(assets))
	copy(sorted, assets)
	sort.Sort(sorted)

	for i := 1; i < len(sorted); i++ {
		if bytes.Equal(sorted[i-1].GetID(), sorted[i].GetID()) {
			return errNonUniqueAssets
		}
	}
	return nil
}

func (at *AssetTransfer) CheckFormat() error {
	if at.Sender == nil {
		return errNoSender
	}
	if _, err := at.Sender.GetPubString(); err != nil {
		return err
	}
	if len(at.Inputs) > config.MaxAdditionalIndices || len(at.Inputs) == 0 {
		return errInvalidAssetCount
	}
	if len(at.Outputs) > config.MaxAdditionalIndices {
		return errInvalidAssetCount
	}
	if config.SignCausalAssets {
		if at.Signature == nil {
			return errNoSignature
		}
	}

	if len(at.Outputs) != len(at.Receivers) {
		return errInvalidAssetCount
	}

	if err := CheckUnique(at.Outputs); err != nil {
		return err
	}

	// The inputs are checked for unique during validation
	return nil
}

// GetSignedBytes returns the bytes that should be signed by the senders.
func (tx *AssetTransfer) GetSignedMessage() []byte {
	if len(tx.SignedBytes) == 0 {
		panic(len(tx.SignedBytes))
	}
	return tx.SignedBytes
}

// GetSignedHash returns messages.GetHash(tx.GetSignedMessage())
func (tx *AssetTransfer) GetSignedHash() types.HashBytes {
	return types.GetHash(tx.SignedBytes)
}

//////////////////////////////////////// Encoding ///////////////////////////////////////////////////////

// encodeSignedBytes generates the bytes that will be signed
func (at *AssetTransfer) encodeSignedBytes() (err error) {
	var n int
	writer := bytes.NewBuffer(nil)

	// The sender key
	if n, err = at.Sender.Encode(writer); err != nil {
		return
	}

	if len(at.Inputs) > config.MaxAdditionalIndices || len(at.Inputs) == 0 {
		panic("encoding invalid asset transfer")
	}
	// The number of inputs
	var n1 int
	n1, err = utils.EncodeUvarint(uint64(len(at.Inputs)), writer)
	n += n1
	if err != nil {
		return
	}
	// The inputs
	for _, aid := range at.Inputs {
		n1, err = writer.Write(aid)
		n += n1
		if err != nil {
			return
		}
	}

	if len(at.Outputs) > config.MaxAdditionalIndices || len(at.Outputs) != len(at.Receivers) {
		panic("encoding invalid asset transfer")
	}
	// The number of outputs
	n1, err = utils.EncodeUvarint(uint64(len(at.Outputs)), writer)
	n += n1
	if err != nil {
		return
	}
	// The outputs
	for _, asset := range at.Outputs {
		n1, err = asset.Encode(writer)
		n += n1
		if err != nil {
			return
		}
	}
	for _, pub := range at.Receivers {
		n1, err = pub.Encode(writer)
		n += n1
		if err != nil {
			return
		}
	}
	at.SignedBytes = writer.Bytes()
	return
}

var ErrMustSignFirst = fmt.Errorf("must sign tx before encoding")

// Encode the asset transfer to the writer.
func (at *AssetTransfer) Encode(writer io.Writer) (n int, err error) {
	if config.SignCausalAssets {
		if at.SignedBytes == nil {
			err = ErrMustSignFirst
			return
		}
	}

	// be sure we have encoded the signed part
	if at.SignedBytes == nil {
		if err = at.encodeSignedBytes(); err != nil {
			return
		}
	}

	var nxtN int
	// First the size of the signed part
	nxtN, err = utils.EncodeUvarint(uint64(len(at.SignedBytes)), writer)
	n += nxtN
	if err != nil {
		return
	}

	// The main message
	nxtN, err = writer.Write(at.SignedBytes)
	n += nxtN
	if err != nil {
		return
	}

	if config.SignCausalAssets {
		// The signature
		nxtN, err = at.Signature.Encode(writer)
		n += nxtN
	}

	if n > MaxAssetSize {
		panic("fix me")
	}

	return
}

// Decode the asset transfer from the writer.
func (at *AssetTransfer) Decode(r io.Reader) (n int, err error) {
	var nxtN int
	var signedByteCount uint64
	signedByteCount, nxtN, err = utils.ReadUvarint(r)
	n += nxtN
	if err != nil {
		return
	}
	if signedByteCount > MaxAssetSize { // TODO
		err = errAssetTooLarge
		return
	}
	at.SignedBytes = make([]byte, signedByteCount)
	nxtN, err = r.Read(at.SignedBytes)
	n += nxtN
	if err != nil {
		return
	}

	reader := bytes.NewReader(at.SignedBytes)
	// First the sender pub
	at.Sender = at.NewPubFunc()
	if n, err = at.Sender.Decode(reader); err != nil {
		return
	}

	// The number of inputs
	v, n1, err := utils.ReadUvarint(reader)
	n += n1
	if err != nil {
		return
	}
	inputCount := int(v)
	switch {
	case inputCount <= 0, inputCount > config.MaxAdditionalIndices:
		err = errInvalidAssetCount
		return
	}
	// The inputs
	at.Inputs = make([]types.HashBytes, inputCount)
	for i := range at.Inputs {
		at.Inputs[i], n1, err = types.DecodeHash(reader)
		n += n1
		if err != nil {
			return
		}
	}

	// The number of outputs.
	v, n1, err = utils.ReadUvarint(reader)
	n += n1
	if err != nil {
		return
	}
	assetCount := int(v)
	switch {
	case assetCount <= 0, assetCount > config.MaxAdditionalIndices:
		err = errInvalidAssetCount
		return
	}
	// The outputs.
	at.Outputs = make([]AssetInterface, assetCount)
	for i := range at.Outputs {
		at.Outputs[i] = at.NewAssetFunc()
		n1, err = at.Outputs[i].Decode(reader)
		n += n1
		if err != nil {
			return
		}
	}
	// The recipiants
	at.Receivers = make([]sig.Pub, assetCount)
	for i := range at.Receivers {
		at.Receivers[i] = at.NewPubFunc()
		n1, err = at.Receivers[i].Decode(reader)
		n += n1
		if err != nil {
			return
		}
	}

	if config.SignCausalAssets {
		// The signature
		at.Signature = at.NewSigFunc()
		n1, err = at.Signature.Decode(r)
		n += n1
	}

	return
}

// Marshal the asset transfer.
func (ba *AssetTransfer) MarshalBinary() (data []byte, err error) {
	return types.MarshalBinaryHelper(ba)
}

// Unmarshal the asset transfer.
func (ba *AssetTransfer) UnmarshalBinary(data []byte) error {
	return types.UnmarshalBinaryHelper(ba, data)
}
