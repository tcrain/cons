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
package messages

import (
	"fmt"
)

// Each message type is encoded in a 4 byte integer
const (
	HdrBcons            HeaderID = iota // A binary consensus message
	HdrBconsAux                         // A binary consensus aux message
	HdrTest                             // A test message
	HdrNetworkTest                      // A network test message
	HdrTestTimeout                      // A timeout test message
	HdrNoProgress                       // A message indicating a node has not made progress towards a decision after a timeout
	HdrConsMsg                          // A consensus message
	HdrBinState                         // A message containing multiple consensus messages
	HdrSimpleCons                       // A message for the simple consensus test protocol
	HdrPropose                          // A message containing a consensus proposal, used in the sime consensus and tests
	HdrBinPropose                       // A message containing a single binary value consensus proposal
	HdrMvPropose                        // A message containing a multi-value proposal
	HdrEcsig                            // A message containing an EC signature
	HdrEcpub                            // A message containing an EC public key
	HdrEcpriv                           // A message containing an EC private key
	HdrSignTest                         // A message for a signature test
	HdrAuxProof                         // A message containing proof supporting an aux binary value
	HdrAuxStage0                        // A message containing proof supporting an aux binary value
	HdrAuxStage1                        // A message containing proof supporting an aux binary value
	HdrAuxBoth                          // A message containing proof supporting an aux binary value or both binary values
	HdrBV0                              // A binary value message for 0
	HdrBV1                              // A binary message for 1
	HdrCoin                             // A message used to compute a random coin for a round for StrongCoin1.
	HdrCoinPre                          // A message sent before the coin to change a t+1 coin to an n-t one
	HdrCoinProof                        // A proof for a coin.
	HdrAuxProofTimeout                  // A round timeout in binary consensus
	HdrMvInitTimeout                    // An init message timeout in multivalue consensus
	HdrMvEchoTimeout                    // An echo message timeout in multivalue consensus
	HdrMvCommitTimeout                  // A commit message timeout in multivalue consensus
	HdrMvInit                           // A multivalue consensus init message
	HdrMvMultiInit                      // A multivalue consensus init message with multiple proposals
	HdrMvInitSupport                    // A multivalue consensus init message with pointers to a previous init message
	HdrMvEcho                           // A multivalue consensus echo message
	HdrMvCommit                         // A multivalue consensus commit message
	HdrMvRecoverTimeout                 // A multivalue consensus recovery timeout message
	HdrMvRequestRecover                 // A multivalue consensus message requesting the decided value
	HdrDSSPartSig                       // A partial threshold signature message
	// HdrDSSPartPub                       // A partial threshold public key message
	HdrEdsig      // An EDDSA signature message
	HdrEdpub      // An EDDSA public key message
	HdrSchnorrsig // A schnorr signature message
	HdrSchnorrpub // A schnorr public key message
	HdrBlssig     // A BLS signature message
	HdrBlspub     // A BLS public key message
	HdrEdPartPub  // A partial threshold signature
	HdrDualPub    // A dual pub message
	HdrPartialMsg // A message that is broken into several parts
	HdrHash       // A message that contains a hash
	HdrVrfProof   // A message that contains a VRF proof
	HdrQsafeSig   // A quantum safe signature
	HdrQsafePub   // A quantum safe public key
)

// String returns the header id name as a string
func (hi HeaderID) String() string {
	var msg string
	switch hi {
	case HdrBcons:
		msg = "HdrBcons"
	case HdrBconsAux:
		msg = "HdrBconsAux"
	case HdrTest:
		msg = "HdrTest"
	case HdrNetworkTest:
		msg = "HdrNetworkTest"
	case HdrTestTimeout:
		msg = "HdrTestTimeout"
	case HdrNoProgress:
		msg = "HdrNoProgress"
	case HdrDualPub:
		msg = "HdrDualPub"
	case HdrConsMsg:
		msg = "HdrConsMsg"
	case HdrBinState:
		msg = "HdrBinState"
	case HdrSimpleCons:
		msg = "HdrSimpleCons"
	case HdrPropose:
		msg = "HdrPropose"
	case HdrBV0:
		msg = "HdrBV0"
	case HdrBV1:
		msg = "HdrBV1"
	case HdrCoinProof:
		msg = "HdrCoinProof"
	case HdrCoin:
		msg = "HdrCoin"
	case HdrAuxBoth:
		msg = "HdrAuxBoth"
	case HdrBinPropose:
		msg = "HdrBinPropose"
	case HdrMvPropose:
		msg = "HdrMvPropose"
	case HdrEcsig:
		msg = "HdrEcsig"
	case HdrEcpub:
		msg = "HdrEcpub"
	case HdrEcpriv:
		msg = "HdrEcpriv"
	case HdrSignTest:
		msg = "HdrEcTest"
	case HdrAuxProof:
		msg = "HdrAuxProof"
	case HdrAuxStage0:
		msg = "HdrAuxStage0"
	case HdrAuxStage1:
		msg = "HdrAuxStage1"
	case HdrAuxProofTimeout:
		msg = "HdrAuxProofTimeout"
	case HdrMvInitTimeout:
		msg = "HdrMvInitTimeout"
	case HdrMvEchoTimeout:
		msg = "HdrMvEchoTimeout"
	case HdrMvCommitTimeout:
		msg = "HdrMvCommitTimeout"
	case HdrMvInit:
		msg = "HdrMvInit"
	case HdrMvInitSupport:
		msg = "HdrMvInitSupport"
	case HdrMvEcho:
		msg = "HdrMvEcho"
	case HdrMvCommit:
		msg = "HdrMvCommit"
	// case HdrMvRecover:
	// 	msg = "HdrMvRecover"
	case HdrMvRecoverTimeout:
		msg = "HdrMvRecoverTimeout"
	case HdrMvRequestRecover:
		msg = "HdrMvRequestRecover"
	// case HdrRequestIndex:
	// 	msg = "HdrRequestIndex"
	case HdrBlssig:
		msg = "HdrBlssig"
	case HdrBlspub:
		msg = "HdrBlspub"
	case HdrEdPartPub:
		msg = "HdrEdPartPub"
	case HdrPartialMsg:
		msg = "HdrPartialMsg"
	case HdrHash:
		msg = "HdrHashMsg"
	case HdrVrfProof:
		msg = "HdrVrfProof"
	default:
		msg = "UnknownHeaderID"
	}
	return fmt.Sprintf("%s:%d:", msg, hi)
}
