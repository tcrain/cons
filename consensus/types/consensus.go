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
	"bytes"
	"encoding"
	"fmt"
	"github.com/tcrain/cons/config"
	"io"
	"sort"
)

type LocalMessageType int

const (
	NonLocalMessage LocalMessageType = iota
	LocalMessage
	LoadedFromDiskMessage
)

type ConsensusIndex struct {
	Index             ConsensusID
	FirstIndex        ConsensusID
	AdditionalIndices []ConsensusID
	UnmarshalFunc     ConsensusIndexFuncs
}

func (ca ConsensusIndex) ShallowCopy() ConsensusIndex {
	ca.AdditionalIndices = append([]ConsensusID{}, ca.AdditionalIndices...)
	return ca
}

type ConsensusIndexFuncs struct {
	ConsensusIDUnMarshaler
	ComputeConsensusID
}

var HashIndexFuns = ConsensusIndexFuncs{
	ConsensusIDUnMarshaler: ConsensusHashUnmarshaler,
	ComputeConsensusID:     GenerateParentHash,
}
var IntIndexFuns = ConsensusIndexFuncs{
	ConsensusIDUnMarshaler: ConsensusIntUnmarshaler,
	ComputeConsensusID:     SingleComputeConsensusID,
}
var NilIndexFuns ConsensusIndexFuncs

// ConsensusIDUnMarshaler is used to unmarshal a consensus id from bytes data.
// It returns the id, and the number of bytes read, or an error.
type ConsensusIDUnMarshaler func(data []byte) (c ConsensusID, n int, err error)

type ComputeConsensusID func(idx ConsensusID, additionalIndices []ConsensusID) (ConsensusIndex, error)

// ConsensusID represents a unique identifier for the consensus index.
// It is either a uint64 or a hash.
type ConsensusID interface {
	encoding.BinaryMarshaler                 // To encode the id.
	Validate(isLocal LocalMessageType) error // Returns an error if the id is invalid. IsLocal should be true if the id comes from a local message.
	IsInitIndex() bool                       // Returns true if the index is the initial index.
}

// ConsensusInt is the index of consensus instance as a uint64.
type ConsensusInt uint64

func SingleComputeConsensusIDShort(idx ConsensusInt) ConsensusIndex {
	ret, err := SingleComputeConsensusID(idx, nil)
	if err != nil {
		panic(err)
	}
	return ret
}

// SingleComputeConsensusID is used for total order messages where only a single consensus id is used per instance.
func SingleComputeConsensusID(idx ConsensusID, additionalIndices []ConsensusID) (cid ConsensusIndex, err error) {
	if len(additionalIndices) != 0 {
		err = ErrTooManyAdditionalIndices
		return
	}
	return ConsensusIndex{
		Index:      idx,
		FirstIndex: idx,
		UnmarshalFunc: ConsensusIndexFuncs{
			ConsensusIDUnMarshaler: ConsensusIntUnmarshaler,
			ComputeConsensusID:     SingleComputeConsensusID,
		},
	}, nil
}

// GenerateParentHash is used for causal ordering to compute the parent hash.
func GenerateParentHash(idx ConsensusID, addIds []ConsensusID) (ConsensusIndex, error) {
	var combineHashes bytes.Buffer
	nxtIdx := idx
	for i := -1; i < len(addIds); i++ {
		// We combine the hashes to make the parent hash
		if _, err := combineHashes.Write([]byte(nxtIdx.(ConsensusHash))); err != nil {
			panic(err)
		}
		if i+1 < len(addIds) {
			nxtIdx = addIds[i+1]
		}
	}
	return ConsensusIndex{
		Index:             ConsensusHash(GetHash(combineHashes.Bytes())), // The combined hash:
		FirstIndex:        idx,
		AdditionalIndices: addIds,
		UnmarshalFunc: ConsensusIndexFuncs{
			ConsensusIDUnMarshaler: ConsensusHashUnmarshaler,
			ComputeConsensusID:     GenerateParentHash,
		},
	}, nil
}

// ConsensusIntUnmarshaler unmarshals a ConsensusInt.
func ConsensusIntUnmarshaler(data []byte) (c ConsensusID, n int, err error) {
	if len(data) < 8 {
		err = ErrNotEnoughBytes
		return
	}
	return ConsensusInt(config.Encoding.Uint64(data)), 8, nil
}

// IsInitIndex returns if ci == 1
func (ci ConsensusInt) IsInitIndex() bool {
	return ci == 1
}

// Validate returns true if the index is > 0, if local it always return true.
func (ci ConsensusInt) Validate(isLocal LocalMessageType) error {
	if isLocal == LocalMessage {
		return nil
	}
	if ci > 0 {
		return nil
	}
	return ErrInvalidIndex
}

// MarshalBinary encodes the index using config.Encoding.PutUint64
func (ci ConsensusInt) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 8)
	config.Encoding.PutUint64(data, uint64(ci))
	return
}

// ParentConsensusHash represents multiple hashes (for causal).
// This is the ordered list of hashes that this consensus will consume.
// The order matters, i.e. each order is unique, but a consensus can only echo once for each consumption of an output.
// The first item in the list determines the member/forward checkers state.
// All the items in the list must be owned by the same public key.
type ParentConsensusHash ConsensusHash

// ConsensusHash represents a consensus instance as a hash string.
type ConsensusHash HashStr

// ConsensusHashUnmarshaler unmarshals the hash by checking bounds and casting as a string.
func ConsensusHashUnmarshaler(data []byte) (c ConsensusID, n int, err error) {
	hashLen := GetHashLen()
	if len(data) < hashLen {
		return nil, 0, ErrNotEnoughBytes
	}
	return ConsensusHash(data[:hashLen]), hashLen, nil
}

// IsInitIndex is TODO
var InitIndexHash ConsensusHash = "fix this"

func (cs ConsensusHash) IsInitIndex() bool {
	return cs == InitIndexHash
}

// Validate returns nil.
func (cs ConsensusHash) Validate(LocalMessageType) error {
	return nil
}

// MarshalBinary casts the hash as bytes.
func (cs ConsensusHash) MarshalBinary() (data []byte, err error) {
	return []byte(cs), nil
}

type SortConsIndex []ConsensusInt

func (a SortConsIndex) Len() int           { return len(a) }
func (a SortConsIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortConsIndex) Less(i, j int) bool { return a[i] < a[j] }
func (a SortConsIndex) Sort()              { sort.Sort(a) }

// ConsensusRound is the round of consensus for a given index
type ConsensusRound uint32

// OrderingType is the way consensus items are order.
type OrderingType int

const (
	Total  OrderingType = iota // Consensus items are ordered totally
	Causal                     // Consensus items are ordered causally.
)

var BothOrders = []OrderingType{Total, Causal}

// String returns the partial message type as a readable string.
func (pt OrderingType) String() string {
	switch pt {
	case Total:
		return "Total"
	case Causal:
		return "Causal"
	default:
		return fmt.Sprintf("OrderingType%d", pt)
	}
}

// PartialMessageType represents the type of partial messaging being used.
type PartialMessageType int

const (
	NoPartialMessages   PartialMessageType = iota // No partial messaging is used
	FullPartialMessages                           // Messages are broken up into even pieces based on the fan out size
)

// String returns the partial message type as a readable string.
func (pt PartialMessageType) String() string {
	switch pt {
	case NoPartialMessages:
		return "NoPartialMessages"
	case FullPartialMessages:
		return "FullPartialMessags"
	default:
		return fmt.Sprintf("PartialMsgType%d", pt)
	}
}

// ByzType describe what type of "Byzantine" (what a silly name) behaviour a node should exhibit (if any).
type ByzType int

const (
	// TODO more Byzantine types.
	NonFaulty        ByzType = iota // A node acts correctly.
	Mute                            // A node does not send any messages.
	BinaryBoth                      // A node sends messages for both 1 and 0 in binary consensus.
	BinaryFlip                      // A node sends the opposite of what it should in binary consensus.
	HalfHalfNormal                  // Send different values to half the nodes
	HalfHalfFixedBin                // Send different values to half the nodes, always using the fixed binary values
)

var AllByzTypes = []ByzType{NonFaulty, Mute, BinaryBoth, BinaryFlip, HalfHalfNormal, HalfHalfFixedBin}

// String returns the Byzantine type as a human readable string.
func (bt ByzType) String() string {
	switch bt {
	case NonFaulty:
		return "NonFaulty"
	case Mute:
		return "Mute"
	case BinaryBoth:
		return "BinaryBoth"
	case BinaryFlip:
		return "BinaryFlip"
	case HalfHalfNormal:
		return "HalfHalfNormal"
	case HalfHalfFixedBin:
		return "HalfHalfFixedBin"
	default:
		return fmt.Sprintf("ByzType%d", bt)
	}
}

// MemberCheckerType describes what type of member checker is being used.
type MemberCheckerType int

const (
	CurrentTrueMC MemberCheckerType = iota // Static membership, but membership checking only becomes available once the previous consensus instance terminates (for testing).
	TrueMC                                 // Static membership.
	LaterMC                                // Static membership, but may change, resulting in the undecided instances to be restarted (used for MvCons3)
	BinRotateMC                            // Rotates one index through the list of members each time 0 is decided.
	CurrencyMC                             // To be used with SimpleCurrencyTxProposer state machine
)

// String returns the human readable representation of the member chekcer type.
func (mc MemberCheckerType) String() string {
	switch mc {
	case TrueMC:
		return "TrueMC"
	case CurrentTrueMC:
		return "CurrentTrueMC"
	case BinRotateMC:
		return "BinRotateMC"
	case CurrencyMC:
		return "CurrencyMC"
	default:
		return fmt.Sprintf("MemChecker%d", mc)
	}
}

func UseTp1CoinThresh(to TestOptions) bool {
	switch to.CoinType {
	case StrongCoin2EchoType, StrongCoin1EchoType:
		return true
	}
	switch to.ConsType {
	case BinConsRnd4Type, BinConsRnd6Type:
		return true
	}
	return false
}

// AllMC is a list of all the possible member checker types.
var AllMC = []MemberCheckerType{TrueMC, CurrentTrueMC, LaterMC, BinRotateMC, CurrencyMC}

// ConsType tells what type of consensus is being used for the experiment.
type ConsType int

const (
	BinCons1Type      ConsType = iota // Binary consensus
	BinConsRnd1Type                   // Randomized binary consensus
	BinConsRnd2Type                   // Randomized binary consensus no signatures needed
	BinConsRnd3Type                   // Randomized binary consensus weak coin
	BinConsRnd4Type                   // Randomized binary consensus weak coin no signatures needed
	BinConsRnd5Type                   // Randomized binary consensus combination of BinConsRnd1 and BinConsRnd3
	BinConsRnd6Type                   // Randomized binary consensus combination of BinConsRnd2 and BinConsRnd4
	MvBinCons1Type                    // Multivalue reduction to binary consensus BinCons1
	MvBinConsRnd1Type                 // Multivalue reduction to binary consensus BinConsRnd1
	MvBinConsRnd2Type                 // Multivalue reduction to binary consensus BinConsRnd2
	MvCons2Type                       // Multivalue PBFT like
	MvCons3Type                       // Multivalue HotStuff like
	RbBcast1Type                      // Reliable broadcast like
	RbBcast2Type                      // Reliable broadcast
	SimpleConsType                    // Nodes broadcast their id and public key and wait to recieve this from all other nodes, just for testing.
	MockTestConsType                  // Test object
)

// String returns the consensus type in a human readable format.
func (ct ConsType) String() string {
	switch ct {
	case BinCons1Type:
		return "BinCons1"
	case BinConsRnd1Type:
		return "BinConsRnd1"
	case BinConsRnd2Type:
		return "BinConsRnd2"
	case BinConsRnd3Type:
		return "BinConsRnd3"
	case BinConsRnd4Type:
		return "BinConsRnd4"
	case BinConsRnd5Type:
		return "BinConsRnd5"
	case BinConsRnd6Type:
		return "BinConsRnd6"
	case MvBinCons1Type:
		return "MvBinCons1"
	case MvBinConsRnd1Type:
		return "MvBinConsRnd1"
	case MvBinConsRnd2Type:
		return "MvBinConsRnd2"
	case MvCons2Type:
		return "MvCons2"
	case MvCons3Type:
		return "MvCons3"
	case SimpleConsType:
		return "SimpleCons"
	case RbBcast1Type:
		return "RbBcast1"
	case RbBcast2Type:
		return "RbBcast2"
	case MockTestConsType:
		return "MockTestConsType"
	default:
		return fmt.Sprintf("ConsType%d", ct)
	}
}

type DoneType int // When a decided interface is kept or discarded.

const (
	NotDone   DoneType = iota // The consensus instance has not finished.
	DoneClear                 // The consensus instance predecision will be discarded.
	DoneKeep                  // The consensus instance predecision will be kept.
)

func (dt DoneType) String() string {
	switch dt {
	case NotDone:
		return "NotDone"
	case DoneClear:
		return "DoneClear"
	case DoneKeep:
		return "DoneKeep"
	default:
		return fmt.Sprintf("DoneType%d", dt)
	}
}

var AllCollectBroadcast = []CollectBroadcastType{Full, Commit, EchoCommit}

type StopOnCommitType int

const (
	Immediate StopOnCommitType = iota // Terminate as soon as a value is decided.
	SendProof                         // Terminate after sending a proof of termination.
	NextRound                         // Terminate in the round where which all must have deiced.
)

func (sc StopOnCommitType) String() string {
	switch sc {
	case Immediate:
		return fmt.Sprintf("Immediate")
	case SendProof:
		return fmt.Sprintf("SendProof")
	case NextRound:
		return fmt.Sprintf("NextRound")
	default:
		return fmt.Sprintf("StopOnCommit%d", sc)
	}
}

var AllStopOnCommitTypes = []StopOnCommitType{Immediate, NextRound, SendProof}

type CollectBroadcastType int

const (
	Full       CollectBroadcastType = iota // Normal broadcast.
	Commit                                 // Send commit to next coordinator instead of all to all broadcast.
	EchoCommit                             // Send echo and commit to coordinator instead of all to all broadcast.
)

func (cbt CollectBroadcastType) String() string {
	switch cbt {
	case Full:
		return "Full"
	case Commit:
		return "Commit"
	case EchoCommit:
		return "EchoCommit"
	default:
		return fmt.Sprintf("CollectBroadcastType%d", cbt)
	}
}

type RndMemberType int

const (
	NonRandom       RndMemberType = iota // Members are decided non randomly.
	VRFPerCons                           // Members are selected based on their VRF for each consensus.
	VRFPerMessage                        // Members are selected based on their VRF for each message type.
	KnownPerCons                         // Members are selected randomly, but known publically.
	LocalRandMember                      // Members are chosen locally at random
)

var AllRandMemberTypes = []RndMemberType{NonRandom, VRFPerCons, VRFPerMessage, KnownPerCons, LocalRandMember}

func (rmt RndMemberType) String() string {
	switch rmt {
	case NonRandom:
		return "NotRandom"
	case VRFPerCons:
		return "VRFPerCons"
	case VRFPerMessage:
		return "VRFPerMessage"
	case KnownPerCons:
		return "KnownPerCons"
	case LocalRandMember:
		return "LocalRandMember"
	default:
		return fmt.Sprintf("RndMemberType%d", rmt)
	}
}

// BinVal is a binary value stored as a byte, any value not 0 other than 2 is considered 1
// 2 means Coin which is used in the randomized consensus.
type BinVal byte

const (
	Coin    BinVal = 2
	BothBin BinVal = 2
)

// HashBytes is a hash as a byte slice
type HashBytes []byte

// GetSignedHash is a helper function
func (hb HashBytes) GetSignedHash() HashBytes {
	return hb
}

type SortedHashBytes []HashBytes

func (a SortedHashBytes) Len() int           { return len(a) }
func (a SortedHashBytes) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortedHashBytes) Less(i, j int) bool { return bytes.Compare(a[i], a[j]) < 0 }

// DecodeHash reads a hash from the reader, returning a error if it could not
// read a valid hash size.
func DecodeHash(reader io.Reader) (hb HashBytes, n int, err error) {
	hb = make(HashBytes, GetHashLen())
	if n, err = reader.Read(hb); err != nil {
		return
	}
	if n != GetHashLen() {
		err = ErrNotEnoughBytes
	}
	return
}

// Encode the hash into the writer.
func (hb HashBytes) Encode(writer io.Writer) (n int, err error) {
	if err = hb.Validate(); err != nil {
		return
	}
	return writer.Write(hb)
}

// Validate returns an error if the hash is the wrong length.
func (hb HashBytes) Validate() error {
	if len(hb) != GetHashLen() {
		return ErrInvalidHashSize
	}
	return nil
}

// HashStr is a hash a a string
type HashStr string

// MarshalBinary encodes returns the encoded hash.
func (v HashStr) MarshalBinary() (data []byte, err error) {
	if len(v) != GetHashLen() {
		return nil, ErrInvalidHashSize
	}
	return []byte(v), nil
}

// IsMember is set when the node knows if it is a member or not
type IsMember int

const (
	PossibleMember IsMember = iota
	NonMemberNode
	MemberNode
)
