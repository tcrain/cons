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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/utils"
	"math"
	"strconv"
	"testing"
)

func TestDirectAsset(t *testing.T) {
	initAssets := make([]AssetInterface, 10)
	for i := 0; i < 10; i++ {
		initAssets[i] = CreateInitialDirectAsset([]byte(strconv.Itoa(i)), []byte(config.CsID))
	}

	inputs, outputs := GenDirectAssetOutput(nil, initAssets)

	err := CheckDirectOutputFunc(nil, inputs, outputs)
	assert.Nil(t, err)

	// check a bad hash
	oldID := outputs[0].(*DirectAsset).ID
	outputs[0].(*DirectAsset).ID[0] += 1
	err = CheckDirectOutputFunc(nil, inputs, outputs)
	assert.Equal(t, errInvalidOutputAsset, err)
	outputs[0].(*DirectAsset).ID = oldID

	// Check a bad output count
	err = CheckDirectOutputFunc(nil, inputs, outputs[1:])
	assert.Equal(t, errInvalidAssetCount, err)

	// check unique
	stringItems := make([]utils.Any, len(inputs)+len(outputs))
	for i, nxt := range append(inputs, outputs...) {
		stringItems[i] = string(nxt.GetID())
	}
	assert.True(t, utils.CheckUnique(stringItems...))

	// Do it again
	inputs, outputs = GenDirectAssetOutput(nil, outputs)

	err = CheckDirectOutputFunc(nil, inputs, outputs)
	assert.Nil(t, err)

	for _, nxt := range outputs {
		stringItems = append(stringItems, string(nxt.GetID()))
	}
	assert.True(t, utils.CheckUnique(stringItems...))

}

func TestValueAsset(t *testing.T) {
	values := utils.GenListUint64(1, 11)

	initAssets := GenInitialValueAssets(values, []byte(config.CsID))
	asInt := make([]AssetInterface, len(initAssets))
	for i, nxt := range initAssets {
		asInt[i] = nxt
	}

	receivers := make([]sig.Pub, len(initAssets))
	inputs, outputs := GenValueAssetOutput(nil, asInt, receivers, values)

	err := CheckValueOutputFunc(nil, inputs, outputs)
	assert.Nil(t, err)

	// check a zero value output
	oldVal := outputs[0].(*ValueAsset).Value
	outputs[0].(*ValueAsset).Value = 0
	err = CheckValueOutputFunc(nil, inputs, outputs)
	assert.Equal(t, errZeroOutput, err)
	outputs[0].(*ValueAsset).Value = oldVal

	// check a bad hash
	oldID := outputs[0].(*ValueAsset).ID
	outputs[0].(*ValueAsset).ID[0] += 1
	err = CheckValueOutputFunc(nil, inputs, outputs)
	assert.Equal(t, errInvalidOutputAsset, err)
	outputs[0].(*ValueAsset).ID = oldID

	// check overflow
	oldVal = inputs[0].(*ValueAsset).Value
	inputs[0].(*ValueAsset).Value = math.MaxUint64 - 1
	err = CheckValueOutputFunc(nil, inputs, outputs)
	assert.Equal(t, errOverflow, err)
	inputs[0].(*ValueAsset).Value = oldVal

	// makes so we dont have enough value to consume
	outputs[0].(*ValueAsset).Value += 1
	err = CheckValueOutputFunc(nil, inputs, outputs)
	assert.Equal(t, errNotEnoughInputValue, err)
	outputs[0].(*ValueAsset).Value -= 1

	// check unique
	stringItems := make([]utils.Any, len(inputs)+len(outputs))
	for i, nxt := range append(inputs, outputs...) {
		stringItems[i] = string(nxt.GetID())
	}
	assert.True(t, utils.CheckUnique(stringItems...))

	// Do it again
	inputs, outputs = GenValueAssetOutput(nil, outputs, receivers, values)

	err = CheckValueOutputFunc(nil, inputs, outputs)
	assert.Nil(t, err)

	for _, nxt := range outputs {
		stringItems = append(stringItems, string(nxt.GetID()))
	}
	assert.True(t, utils.CheckUnique(stringItems...))

}
