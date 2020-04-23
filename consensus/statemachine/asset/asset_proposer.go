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
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/statemachine"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"math/rand"
	"time"
)

type assetGlobalState struct {
	assetTable *AccountTable // The account table of accounts and their assets.

	myProposalCount uint64
	// Keep track of how many transfers each pub key has done
	trafserCount map[sig.PubKeyStr]uint64
	// we finish the test when all pubs make lastProposal transfers
	finishedMap map[sig.PubKeyStr]bool

	isMember   bool // if this node is participating as a member
	numMembers int  // number of members sending transactions in this experiment

	myKey                sig.Priv                // the local nodes private key
	lastProposal         uint64                  // the number of transfers to perform per node
	initBlockBytes       [][]byte                // the initial state
	initPubs             []sig.Pub               // the initial pubs
	initialState         []byte                  // initBlockBytes appended together
	initAssetsIDs        []sig.ConsIDPub         // initial assets
	validFunc            CheckTransferOutputFunc // for checking if a transfer is valid
	newAssetTransferFunc func() *AssetTransfer   // for decoding asset transfers
	genAssetTransferFunc GenAssetTransferFunc    // for generating new proposals
	newPubFunc           func() sig.Pub          // for deciding public keys
	newSigFunc           func() sig.Sig          // for generating new signatures.

	startedRecordingStats bool // set to true once stat recording has started
}

// AssetProposer represents the statemachine object for a causally ordered state machine making
// asset transfers.
type AssetProposer struct {
	statemachine.AbsCausalStateMachine
	statemachine.AbsRandSM

	*assetGlobalState
}

// NewAssetProposer creates an empty AssetProposer object.
func NewAssetProposer(useRand bool, initRandBytes [32]byte,
	initBlocksBytes [][]byte, myKey sig.Priv,
	newAssetFunc func() AssetInterface,
	newAssetTransferFunc func() *AssetTransfer,
	genAssetTransferFunc GenAssetTransferFunc,
	validFunc CheckTransferOutputFunc, isMember bool, numMembers int,
	newPubFunc func() sig.Pub, newSigFunc func() sig.Sig) *AssetProposer {

	ret := &AssetProposer{
		AbsRandSM: statemachine.NewAbsRandSM(initRandBytes, useRand)}
	ret.assetGlobalState = &assetGlobalState{}
	ret.isMember = isMember
	ret.newAssetTransferFunc = newAssetTransferFunc
	ret.genAssetTransferFunc = genAssetTransferFunc
	ret.myKey = myKey
	ret.newPubFunc = newPubFunc
	ret.newSigFunc = newSigFunc
	ret.assetTable = NewAccountTable()
	ret.initBlockBytes = initBlocksBytes
	ret.numMembers = numMembers

	buf := bytes.NewBuffer(nil)
	for _, initBlockBytes := range initBlocksBytes {
		if len(initBlockBytes) == 0 { // empty block for non-participants
			continue
		}
		initBlock := InitBlock{NewPubFunc: newPubFunc,
			NewAssetFunc: newAssetFunc}
		tmpBuf := bytes.NewReader(initBlockBytes)
		if _, err := initBlock.Decode(tmpBuf); err != nil {
			panic(err)
		}

		_, err := initBlock.Encode(buf)
		if err != nil {
			panic(err)
		}

		var initPubsIDs []utils.Any
		for i, nxt := range initBlock.InitPubs {
			acc, err := ret.assetTable.GetAccount(nxt)
			if err != nil {
				panic(err)
			}
			// Add the asset to the pubs account
			acc.AddAsset(initBlock.InitAssets[i])
			pStr, err := nxt.GetPubString()
			if err != nil {
				panic(err)
			}
			// We only want to add the pubs once to the list of init pubs
			initPubsIDs = append(initPubsIDs, pStr)
			if !utils.CheckUnique(initPubsIDs...) {
				initPubsIDs = initPubsIDs[:len(initPubsIDs)-1]
			} else {
				ret.initPubs = append(ret.initPubs, nxt)
			}
			ret.initAssetsIDs = append(ret.initAssetsIDs,
				sig.ConsIDPub{ID: types.ConsensusHash(initBlock.InitAssets[i].GetID()),
					Pub: nxt})
		}
		if !utils.CheckUnique(initPubsIDs) {
			panic("should be unique") // sanity check
		}
	}

	var found bool
	for _, nxt := range ret.initPubs {
		if sig.CheckPubsEqual(ret.myKey.GetPub(), nxt) {
			found = true
		}
	}
	if !found {
		logging.Info("No assets")
	}

	ret.initialState = buf.Bytes()
	ret.trafserCount = make(map[sig.PubKeyStr]uint64, len(ret.initPubs))
	ret.finishedMap = make(map[sig.PubKeyStr]bool, len(ret.initPubs))

	ret.validFunc = validFunc

	return ret
}

// Init is called to initialize the object, lastProposal is the number of consensus instances to run, after which a message should be sent
// on doneChan telling the consensus to shut down.
func (da *AssetProposer) Init(gc *generalconfig.GeneralConfig, endAfter types.ConsensusInt,
	memberCheckerState consinterface.ConsStateInterface, mainChannel channelinterface.MainChannel,
	doneChan chan channelinterface.ChannelCloseType) {

	da.lastProposal = uint64(endAfter)
	da.AbsRandSM.AbsRandInit(gc)

	myIndex, err := types.GenerateParentHash(da.GetInitialFirstIndex(), nil)
	if err != nil {
		panic(err)
	}

	da.AbsInit(myIndex, gc,
		da.initAssetsIDs, mainChannel, doneChan)

}

// GenerateNewSM is called on this init SM to generate a new SM given the items to be consumed.
// It should just generate the item, it should not change the state of any of the parentSMs.
// This will be called on the initial CausalStateMachineInterface passed to the system
func (da *AssetProposer) GenerateNewSM(consumedIndices []types.ConsensusID,
	parentSMs []consinterface.CausalStateMachineInterface) consinterface.CausalStateMachineInterface {

	if len(parentSMs) == 0 {
		panic("should only have a parent state machine")
	}
	var parDeps [][]sig.ConsIDPub
	for _, parent := range parentSMs {
		if !parent.GetDecided() {
			panic("should be decided")
		}
		parDeps = append(parDeps, parent.GetDependentItems())
	}
	if len(parDeps) == 0 {
		panic("should have parent dependencies")
	}

	// sanity check TODO remove
	for i, nxt := range consumedIndices {
		var found bool
		for _, nxtPar := range parDeps[i] {
			if bytes.Equal(types.HashBytes(nxt.(types.ConsensusHash)), types.HashBytes(nxtPar.ID.(types.ConsensusHash))) {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprint("parent doesn't have consumed child", i))
		}
	}

	ret := &AssetProposer{}
	*ret = *parentSMs[0].(*AssetProposer)
	var startRecordingStats bool
	if !ret.startedRecordingStats {
		var txCount uint64
		for _, nxt := range ret.trafserCount {
			txCount += nxt
			if txCount > (config.WarmUpInstances-1)*uint64(ret.numMembers) {
				startRecordingStats = true
				ret.startedRecordingStats = true
				break
			}
		}
	}
	ret.AbsStartIndex(consumedIndices, parentSMs, startRecordingStats)
	logging.Infof("generating new direct asset causal proposer")
	return ret
}

// HasEchoed should return true if this SM has echoed a proposal
// HasEchoed() bool

// GetDependentItems returns a  list of items dependent from this SM.
// This list must be the same as the list returned from HasDecided
func (da *AssetProposer) GetDependentItems() []sig.ConsIDPub {
	return da.AbsGetDependentItems()
}

// HasDecided is called each time a consensus decision takes place, given the index and the decided vaule.
// Proposer is the public key of the node that proposed the decision.
// It returns a list of causally dependent StateMachines that result from the decisions.
func (da *AssetProposer) HasDecided(proposer sig.Pub, index types.ConsensusIndex, decision []byte) []sig.ConsIDPub {
	buf := bytes.NewReader(decision)
	var err error
	_, err = da.RandHasDecided(proposer, buf, true)
	if err != nil {
		logging.Error("invalid mv vrf proof", err)
		panic(err)
	}

	// decode the transfer
	tr := da.newAssetTransferFunc()
	if _, err = tr.Decode(buf); err != nil {
		panic(err)
	}
	// it should already be validated
	if err = da.assetTable.ValidateTransfer(tr, da.validFunc, false); err != nil { // sanity check TODO remove me
		panic(err)
	}
	// consume the transfer
	da.assetTable.ConsumeTransfer(tr)

	// get the outputs
	outputs := make([]sig.ConsIDPub, len(tr.Outputs))
	for i, nxt := range tr.Outputs {
		outputs[i].ID = types.ConsensusHash(nxt.GetID())
		outputs[i].Pub = tr.Receivers[i]
	}

	pStr, err := tr.Sender.GetPubString()
	if err != nil {
		panic(err)
	}
	da.trafserCount[pStr]++
	if da.trafserCount[pStr] == da.lastProposal {
		da.finishedMap[pStr] = true
	}
	end := len(da.finishedMap) == da.numMembers

	da.AbsHasDecided(proposer, index, decision, outputs, end)

	// make the next proposal
	da.getProposal(proposer)

	return outputs
}

// CheckDecisions is for testing and will be called at the end of the test with
// a causally ordered tree of all the decided values, it should then check if the
// decided values are valid.
func (da *AssetProposer) CheckDecisions(root *utils.StringNode) (errors []error) {
	// Do causally ordered search
	// errors = da.checkNode(root, errors)
	if root.Value != "" {
		panic("root shouldn't have decision")
	}
	return da.recCheckDecisions(root, errors)
}

func (da *AssetProposer) recCheckDecisions(nxt *utils.StringNode, errors []error) []error {

	for _, nxt := range nxt.Children {
		errors = da.checkNode(nxt, errors)
	}
	for _, nxt := range nxt.Children {
		errors = da.recCheckDecisions(nxt, errors)
	}
	return errors
}

func (da *AssetProposer) checkNode(nxt *utils.StringNode, errors []error) []error {
	dec := []byte(nxt.Value)

	buf := bytes.NewReader(dec)
	// decode the transfer
	tr := da.newAssetTransferFunc()
	if _, err := tr.Decode(buf); err != nil {
		errors = append(errors, err)
	} else {
		// it should already be validated
		if err := da.assetTable.ValidateTransfer(tr, da.validFunc, false); err != nil {
			errors = append(errors, err)
		} else {
			// consume the transfer
			da.assetTable.ConsumeTransfer(tr)
		}
	}
	return errors
}

// FailAfter is for testing, once index is reached, it should send a message on doneChan (the input from Init), telling the consensus to shutdown.
func (da *AssetProposer) FailAfter(index types.ConsensusInt) {
	da.lastProposal = uint64(index)
}

// GetInitialFirstIndex returns the hash of some unique string for the first instance of the state machine.
func (da *AssetProposer) GetInitialFirstIndex() types.ConsensusID {
	return da.AbsGetInitialFirstIndex(da.initialState)
}

// StartInit is called on the init state machine to start the program.
func (da *AssetProposer) StartInit(memberCheckerState consinterface.ConsStateInterface) {
	da.getProposal(da.myKey.GetPub())
}

func (da *AssetProposer) getProposal(lastProposer sig.Pub) {
	if !da.isMember {
		return
	}
	if da.myProposalCount < da.lastProposal {

		// for the experiment, we only make a proposal after we have completed our previous
		if !sig.CheckPubsEqual(da.myKey.GetPub(), lastProposer) {
			return
		}

		da.myProposalCount++
		if tr := da.genAssetTransferFunc(da.myKey, da.initPubs, da.assetTable); tr != nil {

			buff := bytes.NewBuffer(nil)
			da.RandGetProposal(buff)

			// use the index
			byts, err := tr.MarshalBinary()
			if err != nil {
				panic(err)
			}

			buff.Write(byts)

			addIds := make([]types.ConsensusID, len(tr.Inputs[1:]))
			for i, nxt := range tr.Inputs[1:] {
				addIds[i] = types.ConsensusHash(nxt)
			}

			parIdx, err := types.GenerateParentHash(types.ConsensusHash(tr.Inputs[0]), addIds)
			if err != nil {
				panic(err)
			}
			w := messagetypes.NewMvProposeMessage(parIdx, buff.Bytes())
			if generalconfig.CheckFaulty(da.GetIndex(), da.GeneralConfig) {
				logging.Info("get byzantine proposal", da.GetIndex(), da.GeneralConfig.TestIndex)
				w.ByzProposal = da.GetByzProposal(w.Proposal, da.GeneralConfig)
			}
			da.AbsGetProposal(w)
		} else {
			logging.Info("no assets to propose")
		}
	} else {
		// panic(da.myProposalCount)
	}
}

// GetByzProposal should generate a byzantine proposal based on the configuration
func (da *AssetProposer) GetByzProposal(originProposal []byte,
	gc *generalconfig.GeneralConfig) (byzProposal []byte) {

	n := da.GetRndNumBytes()
	buf := bytes.NewReader(originProposal[n:])

	tr := da.newAssetTransferFunc()
	if _, err := tr.Decode(buf); err != nil {
		panic(err)
	}
	inputs := make([]AssetInterface, len(tr.Inputs))
	for i, nxt := range tr.Inputs {
		inputs[i] = NewBasicAsset(nxt)
	}

	// we change the recipient
	tr.Receivers[0] = da.initPubs[rand.Intn(da.numMembers)]

	// Generate the new tx
	tx, err := da.assetTable.GenerateAssetTransfer(da.myKey, inputs, tr.Outputs, tr.Receivers, da.validFunc)
	if err != nil {
		panic(err)
	}

	// Create the new proposal
	buff := bytes.NewBuffer(nil)
	// Copy the rand bytes
	if _, err := buff.Write(originProposal[:n]); err != nil {
		panic(err)
	}
	// Encode the txs
	if _, err := tx.Encode(buff); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// ValidateProposal should return nil if the input proposal is valid, otherwise an error.
// Proposer is the public key of the node that proposed the value.
// It is called on the parent state machine instance of the proposal after it has decided.
func (da *AssetProposer) ValidateProposal(proposer sig.Pub, proposal []byte) error {
	buf := bytes.NewReader(proposal)
	_, err := da.RandHasDecided(proposer, buf, false)
	if err != nil {
		return err
	}

	tr := da.newAssetTransferFunc()
	if _, err = tr.Decode(buf); err != nil {
		return err
	}

	// check the sender pub key is equal to the proposer
	if !sig.CheckPubsEqual(proposer, tr.Sender) {
		return types.ErrInvalidRoundCoord
	}

	if err = da.assetTable.ValidateTransfer(tr, da.validFunc, config.SignCausalAssets); err != nil {
		return err
	}

	return nil
}

// GetInitialState returns the initial state of the program.
func (da *AssetProposer) GetInitialState() []byte {
	return da.initialState
}

// StatsString returns statistics for the state machine.
func (da *AssetProposer) StatsString(testDuration time.Duration) string {
	return ""
}

// DoneClear should be called if the instance of the state machine will no longer be used (it should perform any cleanup).
func (da *AssetProposer) DoneClear() {
	da.AbsDoneClear()
	// nothing to do
}

// DoneKeep should be called if the instance of the state machine will be kept.
func (da *AssetProposer) DoneKeep() {
	da.AbsDoneKeep()
	// nothing to do
}
