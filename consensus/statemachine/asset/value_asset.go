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
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math/rand"
)

// ValueAsset is an asset that also contains a value.
type ValueAsset struct {
	BasicAsset
	Value uint64
}

func NewValueAsset() AssetInterface {
	return &ValueAsset{}
}

// GenInitialValueAssets generates value assets for the initial inputs,
// instanceID is the unique id of the system (e.g. config.CSID).
func GenInitialValueAssets(values []uint64, instanceID []byte) []AssetInterface {
	ret := make([]AssetInterface, len(values))
	for i, nxt := range values {
		ret[i] = InternalNewValueAsset(nxt, uint64(i), instanceID)
	}
	return ret
}

func InternalNewValueAsset(value uint64, pid uint64, instanceID []byte) *ValueAsset {
	id := types.GetHash(append(utils.Uint64ToBytes(pid), instanceID...))
	return &ValueAsset{
		BasicAsset: *NewBasicAsset(id),
		Value:      value,
	}
}

func GenValueAssetTransfer(priv sig.Priv, participants []sig.Pub, table *AccountTable) (tr *AssetTransfer) {
	pub := priv.GetPub()
	acc, err := table.GetAccount(pub)
	if err != nil {
		panic(err)
	}
	if len(acc.Assets) == 0 {
		return
	}
	// We consume all our assets
	// We send 1 to a random pub, and the change to ourselves

	inputs := acc.Assets
	var sendAmount uint64
	values := ""
	for _, nxt := range inputs {
		sendAmount += nxt.(*ValueAsset).Value
		values += fmt.Sprintf("%v ", nxt.(*ValueAsset).Value)
	}

	recipiants := make([]sig.Pub, 2)
	amounts := make([]uint64, 2)
	recipiants[0] = participants[rand.Intn(len(participants))]
	amounts[0] = 1
	recipiants[1] = priv.GetPub()
	amounts[1] = sendAmount - amounts[0]

	_, outputs := GenValueAssetOutput(acc, inputs, recipiants, amounts)

	logging.Infof("I have %v assets, vith values %v, %v, i send %v, %v\n", len(acc.Assets), sendAmount, values,
		outputs[0].(*ValueAsset).Value, outputs[1].(*ValueAsset).Value)

	tr, err = table.GenerateAssetTransfer(priv, inputs, outputs, recipiants, CheckValueOutputFunc)
	if err != nil {
		panic(err)
	}
	return
}

// genValueAssetOutput generates value asset outputs given the inputs
func GenValueAssetOutput(acc *AssetAccount, assets []AssetInterface,
	receivers []sig.Pub, amounts []uint64) (inputs []AssetInterface, outputs []AssetInterface) {

	inputs = assets

	if len(receivers) != len(amounts) {
		panic("invalid outputs")
	}

	// Check no overflow
	sendAmmounts := make([]uint64, len(assets))
	for i, nxt := range assets {
		sendAmmounts[i] = nxt.(*ValueAsset).Value
		if sendAmmounts[i] == 0 {
			panic("zero value input")
		}
	}
	if utils.CheckOverflow(sendAmmounts...) {
		panic("inputs would create overflow")
	}
	// Check enough input value
	totalInput := utils.SumUint64(sendAmmounts...)
	totalOutput := utils.SumUint64(amounts...)

	for _, v := range sendAmmounts {
		if v == 0 {
			panic("zero output")
		}
	}
	if totalInput < totalOutput {
		panic("not enough input")
	}

	// Take the hash of all the inputs
	buf := bytes.NewBuffer(nil)
	for _, nxt := range inputs {
		if _, err := buf.Write(nxt.GetID()); err != nil {
			panic(err)
		}
	}
	inputHash := types.GetHash(buf.Bytes())

	// The hash of the outputs is its index in the list of outputs appended to the hash of the inputs
	outputs = make([]AssetInterface, len(amounts))
	for i, v := range amounts {
		outputs[i] = InternalNewValueAsset(v, uint64(i), inputHash)
	}
	return
}

// CheckValueOutputFunc checks that the inputs and outputs are valid.
func CheckValueOutputFunc(account *AssetAccount, inputs []AssetInterface, outputs []AssetInterface) error {
	// Check no overflow
	sendAmmounts := make([]uint64, len(inputs))
	for i, nxt := range inputs {
		sendAmmounts[i] = nxt.(*ValueAsset).Value
	}
	if utils.CheckOverflow(sendAmmounts...) {
		return errOverflow
	}

	// Check output amounts
	outputAmmounts := make([]uint64, len(outputs))
	for i, nxt := range outputs {
		outputAmmounts[i] = nxt.(*ValueAsset).Value
		if outputAmmounts[i] == 0 {
			return errZeroOutput
		}
	}

	// Check enough input value
	totalInput := utils.SumUint64(sendAmmounts...)
	totalOutput := utils.SumUint64(outputAmmounts...)
	if totalInput < totalOutput {
		return errNotEnoughInputValue
	}

	// Take the hash of all the inputs
	buf := bytes.NewBuffer(nil)
	for _, nxt := range inputs {
		if _, err := buf.Write(nxt.GetID()); err != nil {
			panic(err)
		}
	}
	inputHash := types.GetHash(buf.Bytes())

	// The hash of the outputs is its index in the list of outputs appended to the hash of the inputs
	for i, nxt := range outputs {
		outputID := types.GetHash(append(utils.Uint64ToBytes(uint64(i)), inputHash...))
		if !bytes.Equal(nxt.GetID(), outputID) {
			return errInvalidOutputAsset
		}
	}
	return nil
}

func (va *ValueAsset) New() types.EncodeInterface {
	return &ValueAsset{
		BasicAsset: *va.BasicAsset.New().(*BasicAsset)}
}

// Encode the asset to the writer.
func (va *ValueAsset) Encode(writer io.Writer) (n int, err error) {
	if n, err = va.BasicAsset.Encode(writer); err != nil {
		return
	}

	var n1 int
	n1, err = utils.EncodeUint64(va.Value, writer)
	n += n1
	return
}

// Decode the asset from the reader.
func (ba *ValueAsset) Decode(reader io.Reader) (n int, err error) {
	if n, err = ba.BasicAsset.Decode(reader); err != nil {
		return
	}
	var n1 int
	ba.Value, n1, err = utils.ReadUint64(reader)
	n += n1
	return
}

// Marshal the asset.
func (ba *ValueAsset) MarshalBinary() (data []byte, err error) {
	return types.MarshalBinaryHelper(ba)
}

// Unmarshal the asset.
func (ba *ValueAsset) UnmarshalBinary(data []byte) error {
	return types.UnmarshalBinaryHelper(ba, data)
}
