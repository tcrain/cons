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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/testobjects"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"strconv"
	"testing"
)

const (
	initCount = 10
	endAfter  = 10
)

func TestDirectAssetProposer(t *testing.T) {
	initAssets := make([]AssetInterface, initCount)
	for i := range initAssets {
		initAssets[i] = CreateInitialDirectAsset([]byte(strconv.Itoa(i)), []byte(config.CsID))
	}

	testAssetProposer(initAssets, NewDirectAsset, GenDirectAssetTransfer, CheckDirectOutputFunc,
		DirectSendToSelf, t)
}

func TestValueAssetProposer(t *testing.T) {
	initAssetsValues := make([]uint64, initCount)
	for i := range initAssetsValues {
		initAssetsValues[i] = 10
	}
	initAssets := GenInitialValueAssets(initAssetsValues, []byte(config.CsID))

	testAssetProposer(initAssets, NewValueAsset, GenValueAssetTransfer, CheckValueOutputFunc,
		ValueSendToSelf, t)
}

func testAssetProposer(initAssets []AssetInterface, newAssetInterfaceFunc func() AssetInterface,
	genAssetTransferFunc GenAssetTransferFunc, validFunc CheckTransferOutputFunc,
	sendToSelfFunc SendToSelfAsset,
	t *testing.T) {

	privs := make([]sig.Priv, initCount)
	pubs := make([]sig.Pub, initCount)
	var err error
	for i := range privs {
		privs[i], err = ec.NewEcpriv()
		assert.Nil(t, err)
		pubs[i] = privs[i].GetPub()
	}

	initBlock := NewInitBlock(pubs, initAssets, string(InitUniqueName))
	buff := bytes.NewBuffer(nil)
	_, err = initBlock.Encode(buff)
	assert.Nil(t, err)

	newAssetFunc := func() *AssetTransfer {
		return NewEmptyAssetTransfer(newAssetInterfaceFunc, privs[0].NewSig, pubs[0].New)
	}

	var rnd [32]byte

	proposers := make([]*AssetProposer, initCount)
	mainChannels := make([]*testobjects.MockMainChannel, initCount)
	doneChans := make([]chan channelinterface.ChannelCloseType, initCount)

	for i, priv := range privs {
		proposers[i] = NewAssetProposer(false, rnd, [][]byte{buff.Bytes()}, priv, newAssetInterfaceFunc,
			newAssetFunc, genAssetTransferFunc, validFunc, sendToSelfFunc, true, len(privs), pubs[0].New, priv.NewSig)

		gc := &generalconfig.GeneralConfig{Stats: stats.GetStatsObject(types.MvCons2Type, false)}
		mainChannels[i] = &testobjects.MockMainChannel{}
		doneChans[i] = make(chan channelinterface.ChannelCloseType, 1)
		proposers[i].Init(gc, types.ConsensusInt(endAfter), nil, mainChannels[i], doneChans[i], false)

		proposers[i].StartInit(nil)

	}

	// Keep a tree of the decided values
	root := &utils.StringNode{}

	newProposers := make([]map[types.ParentConsensusHash]*AssetProposer, initCount)
	outputsToProposers := make([]map[types.ConsensusHash]*AssetProposer, initCount)
	for i := range newProposers {
		newProposers[i] = make(map[types.ParentConsensusHash]*AssetProposer)
		outputsToProposers[i] = make(map[types.ConsensusHash]*AssetProposer)

		newProposers[i][types.ParentConsensusHash(proposers[i].GetIndex().Index.(types.ConsensusHash))] = proposers[i]
		for _, nxtOut := range proposers[i].GetDependentItems() {
			outputsToProposers[i][nxtOut.ID.(types.ConsensusHash)] = proposers[i]
		}
	}

	prvNode := root
	for k := 0; k < endAfter+1; k++ {
		for i := range newProposers {
			proposals := mainChannels[i].GetProposals()
			assert.Equal(t, 1, len(proposals))

			p := proposals[0].Header.(*messagetypes.MvProposeMessage)
			if k == 2 { // decide nil once
				p.Proposal = nil
			}
			var updatedDecision []byte

			// Process our proposal at all other proposers
			for j, nxtProposer := range newProposers {
				parHash, err := types.GenerateParentHash(p.Index.FirstIndex, p.Index.AdditionalIndices)
				assert.Nil(t, err)
				parentSMs := []consinterface.CausalStateMachineInterface{outputsToProposers[j][p.Index.FirstIndex.(types.ConsensusHash)]}
				for _, nxtAdId := range p.Index.AdditionalIndices {
					parentSMs = append(parentSMs, outputsToProposers[j][nxtAdId.(types.ConsensusHash)])
				}
				newProposer := proposers[i].GenerateNewSM(append([]types.ConsensusID{p.Index.FirstIndex}, p.Index.AdditionalIndices...),
					parentSMs)

				if p.Proposal != nil {
					err = newProposer.ValidateProposal(pubs[i], p.Proposal)
					assert.Nil(t, err)
				}
				var outputs []sig.ConsIDPub
				outputs, updatedDecision = newProposer.HasDecided(pubs[i], p.Index, []sig.Pub{pubs[i]}, p.Proposal)
				for _, nxtOut := range outputs {
					outputsToProposers[j][nxtOut.ID.(types.ConsensusHash)] = newProposer.(*AssetProposer)
				}
				nxtProposer[types.ParentConsensusHash(parHash.Index.(types.ConsensusHash))] = newProposer.(*AssetProposer)
			}
			p.Proposal = updatedDecision
			nxtNode := &utils.StringNode{Value: string(p.Proposal)}
			prvNode.Children = append(prvNode.Children, nxtNode)
			prvNode = nxtNode
		}
	}
	// Be sure they all finished
	for _, nxt := range doneChans {
		_ = <-nxt
	}

	// Now check the decided values are valid
	checkProposer := NewAssetProposer(false, rnd, [][]byte{buff.Bytes()}, privs[0], newAssetInterfaceFunc,
		newAssetFunc, genAssetTransferFunc, validFunc, sendToSelfFunc, false, len(pubs), pubs[0].New, privs[0].NewSig)

	errors := checkProposer.CheckDecisions(root)
	for _, nxtErr := range errors {
		assert.Nil(t, nxtErr)
	}
}
