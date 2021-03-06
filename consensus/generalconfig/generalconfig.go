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

package generalconfig

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
)

type GeneralConfig struct {
	UseFixedSeed       bool                     // If true then use a fix seed to generate proposals/coin
	InitHeaders        []messages.MsgHeader     // These are headers that will be appended at the beginning of all consensus messages for any consensus instance.
	PartialMessageType types.PartialMessageType // if true then mv init messsages will be sent as partial messages
	SetTestConfig      bool                     // for santiy check
	Priv               sig.Priv                 // The local nodes private key // TODO remove this since we should only use the keys that we can receive from the member checker
	Stats              stats.StatsInterface     // performances statistics
	TestIndex          int                      // process i
	Eis                ExtraInitState
	NetworkType        types.NetworkPropagationType // The network type to use (all to all or gossip)
	AllowConcurrent    types.ConsensusInt           // Number of concurrent consensus instances allowed to be run
	Ordering           types.OrderingType
	AllowSupportCoin   bool           // True if AuxProofMessages can support the coin directly instead of a bin value.
	ConsType           types.ConsType // The type of consensus being used for the test.
	CPUProfile         bool           // Profile CPU usage
	MemProfile         bool           // Profile Memory allocation
	TestID             uint64         // Unique test ID
	// UseFullBinaryState will (if true) keep the consensus state as the list of all valid messages received appended together,
	// if false stores only different messages with all the signatures at the end
	UseFullBinaryState          bool
	IncludeCurrentSigs          bool                       // When forwarding a message (for non all-to-all networks) will incude all sigs received so far
	CollectBroadcast            types.CollectBroadcastType // If true, when sending the commit message, will send it to the leader
	IncludeProofs               bool                       // Include signatures as part of messages that prove you are sending a valid message (see protocol description)
	StopOnCommit                types.StopOnCommitType     // If true then the consensus will not execute rounds after deciding (the eventual message propagation will ensure termination)
	ByzStartIndex               uint64                     // Index to start faulty behaviour
	IsByz                       bool                       // True if the node is faulty
	NoSignatures                bool                       // Use encrypted channels instead of signatures
	EncryptChannels             bool                       // If the channels are encrypted
	CoinType                    types.CoinType             // The type of coin being used
	UseTp1CoinThresh            bool                       // if true need t+1 signatures for a threshold signature, false otherwise
	UseFixedCoinPresets         bool                       // If true then will use predefined coins for the initial rounds of randomized consensus
	UseMultiSig                 bool                       // True if multisignatures are enabled.
	MemCheckerBitIDType         types.BitIDType            // If using multi-sigs the type of bit ID to use in the member checker
	SigBitIDType                types.BitIDType            // If using multi-sigs the type of bit ID to use with the signatures
	WarmUpInstances             int                        // Number of consensus instances to run before recording results
	KeepPast                    int                        // Number of previously decided consensus instances to keep in memory
	ForwardTimeout              int                        // milliseconds 	// for msg forwarder when you dont receive enough messages to foward a buffer automatically
	RandForwardTimeout          int                        // amount of time to randomly add to Forward timeout
	ProgressTimeout             int                        // milliseconds, if no progress in this time, let neighbors know
	MvConsTimeout               int                        // millseconds timeout when taking an action in the MV consensus algorithms
	MvConsVRFTimeout            int                        // millseconds timeout for waiting for a proposal when VRFs are enabled (only used by MVCons3)
	MvConsRequestRecoverTimeout int                        // millseconds timeout before requesting the full proposal after delivering the hash
	NodeChoiceVRFRelaxation     int                        // Additional chance to chose a node as a member when using VRF.
	CoordChoiceVRF              int                        // Chance of each node being chosen as a coordinator when using VRF.
	UseRandCoord                bool                       // If true round coordinators will be chosen using VRFs
	BufferForwardType           types.BufferForwardType    // Type of buffer forwarder to use
	MvCons4BcastType            types.MvCons4BcastType     // the type of message broadcast used by MvCons4
	MCType                      types.MemberCheckerType    // if TestMemberCheckers is false, then test a specific type
}

type ExtraInitState interface {
}

// CheckFaulty returns true if the node should act faulty
func CheckFaulty(idx types.ConsensusIndex, gc *GeneralConfig) bool {
	if !gc.IsByz {
		return false
	}
	switch idx := idx.Index.(type) {
	case types.ConsensusInt:
		if idx > types.ConsensusInt(gc.ByzStartIndex)+types.ConsensusInt(gc.WarmUpInstances) {
			return true
		}
		return false
	case types.ConsensusHash:
		return true // TODO always true?
	default:
		panic(idx)
	}
}
