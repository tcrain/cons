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

package types

import (
	"fmt"
)

var ErrDuplicateRoundCoord = fmt.Errorf("dupliate round coordinator message")
var ErrInvalidRoundCoord = fmt.Errorf("invalid round coordinator")
var ErrAlreadyReceivedMessage = fmt.Errorf("already received message")
var ErrInvalidHeader = fmt.Errorf("got an invalid msg header")
var ErrInvalidIndex = fmt.Errorf("index not in range")
var ErrIndexTooOld = fmt.Errorf("index too old")
var ErrIndexTooNew = fmt.Errorf("index too far in future")
var ErrDeserialize = fmt.Errorf("error deserialize")
var ErrClosingTime = fmt.Errorf("time to close")
var ErrTimeout = fmt.Errorf("timeout")
var ErrNoValidSigs = fmt.Errorf("no valid sigs")
var ErrNoNewSigs = fmt.Errorf("no new sigs")
var ErrInvalidBinStateMsg = fmt.Errorf("invalid bin state msg")
var ErrDontForwardMessage = fmt.Errorf("message should not be forwarded")
var ErrTooManyAdditionalIndices = fmt.Errorf("message has too many additional indices")

// member checker
var ErrNotMember = fmt.Errorf("not a member")
var ErrInvalidSig = fmt.Errorf("invalid sig")
var ErrWrongMemberChecker = fmt.Errorf("wrong member checker")
var ErrMemCheckerNotReady = fmt.Errorf("memberchecker not ready")
var ErrNoFixedCoord = fmt.Errorf("no fixed coord")
var ErrCoinProofNotSupported = fmt.Errorf("coin proofs not supported by signature type")
var ErrCoinAlreadyProcessed = fmt.Errorf("coin already processed for this round")

// when using random membership
var ErrNotReceivedVRFProof = fmt.Errorf("have not received node's VRF proof")

// used by auth/sig
var ErrInvalidSigType = fmt.Errorf("invalid sig type")
var ErrInvalidHashSize = fmt.Errorf("invalid input size")
var ErrNilPriv = fmt.Errorf("nil priv key")
var ErrNotEnoughSigs = fmt.Errorf("not enough sigs")
var ErrThresholdSigsNotSupported = fmt.Errorf("threshold signatures not supported")
var ErrInvalidPub = fmt.Errorf("inavlid pub")
var ErrMsgNotFound = fmt.Errorf("msg not found")
var ErrNilPub = fmt.Errorf("nil pub key")
var ErrInvalidPubIndex = fmt.Errorf("invalid pub index")
var ErrNotEnoughPartials = fmt.Errorf("not enough partial signatures")
var ErrInvalidSessionID = fmt.Errorf("invalid session id")
var ErrInvalidSharedThresh = fmt.Errorf("invalid shared threshold setup object")

// used by bls sig
var ErrInvalidBitID = fmt.Errorf("invalid BitID")
var ErrIntersectingBitIDs = fmt.Errorf("must not have intersecting bitIDs to merge sigs")
var ErrInvalidBitIDSub = fmt.Errorf("invalid BitID subtraction")
var ErrUnsortedBitID = fmt.Errorf("bitID must be sorted")
var ErrTooLargeBitID = fmt.Errorf("bitID too large")
var ErrNoPubBytes = fmt.Errorf("missing pub bytes")
var ErrNoItems = fmt.Errorf("no more items to iterate")
var ErrInvalidBitIDEncoding = fmt.Errorf("invalid bitID encoding")

// channel
var ErrInvalidFormat = fmt.Errorf("string in invalid format")
var ErrInvalidMsgSize = fmt.Errorf("invalid msg size")

// csnet
var ErrConnAlreadyExists = fmt.Errorf("connection already exists")
var ErrConnDoesntExist = fmt.Errorf("connection doesn't exist")
var ErrNoRemovedCons = fmt.Errorf("no removed connections")
var ErrNotEnoughConnections = fmt.Errorf("not enough connections")

// msg
var ErrWrongMessageType = fmt.Errorf("tried to deserialize wrong message type")
var ErrNotEnoughBytes = fmt.Errorf("not enough bytes to read")
var ErrDecryptionFailed = fmt.Errorf("decryption failed")
var ErrInvalidStage = fmt.Errorf("invalid message stage")
var ErrNilMsg = fmt.Errorf("nil msg")

// var ErrInvalidMsgSize = fmt.Errorf("Invalid msg size")

// ParticipantRegister
var ErrPubNotFound = fmt.Errorf("should register connection info before requesting")

// causal
var ErrAlreadyConsumedEcho = fmt.Errorf("already consumed echo")
var ErrConsAlreadyDecided = fmt.Errorf("consensus index already decided")
var ErrParentNotFound = fmt.Errorf("parent index not found")
var ErrNotOwner = fmt.Errorf("inputs owned by different pubs")

// bincons1
var ErrNoProofs = fmt.Errorf("no proofs for round")

// mvcons1
var ErrInvalidRound = fmt.Errorf("invalid round")
var ErrNoEchoHash = fmt.Errorf("tried to get proofs for an echo message before sending the hash")
var ErrNilProposal = fmt.Errorf("proposal must not be nil")
var ErrInvalidProposal = fmt.Errorf("invalid proposal")

// partial Message
var ErrHashIdxOutOfRange = fmt.Errorf("partial message hash index out of range")
var ErrInvalidHash = fmt.Errorf("invalid Hash")
var ErrIncorrectNumPartials = fmt.Errorf("incorrect number of partials")
var ErrInvalidPartialIndecies = fmt.Errorf("invalid partial msg indecies to reconstruct")
var ErrDuplicatePartial = fmt.Errorf("duplicate partial")
var ErrAlreadyCombinedPartials = fmt.Errorf("has already combined partials")

// storage
var ErrInvalidKeySize = fmt.Errorf("invalid key size")
