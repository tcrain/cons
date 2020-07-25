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

package sig

import (
	"github.com/tcrain/cons/consensus/auth/bitid"
	// "fmt"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
)

/////////////////////////////////////////////////////////////////////////////////////////////////
//
/////////////////////////////////////////////////////////////////////////////////////////////////

// ThreshStateInterface is for storing state about threshold signatures.
// Methods should be concurrent safe.
type ThreshStateInterface interface {
	BasicThresholdInterface
	// VerifyPartialSig([]byte, Pub, Sig) error    // Verify a partial signature.
	CombinePartialSigs([]Sig) (*SigItem, error) // Create a threshold signatue for the partial signatures.
	PartialSign(msg SignedMessage) (Sig, error) // Sign a message using the local node's partial key.
}

type BasicThresholdInterface interface {
	ThresholdCountInterface
	// GetPartialPub() Pub // Get the partial public key for this node.
	GetSharedPub() Pub // Get the shared public key for the threshold.
}

type ThresholdCountInterface interface {
	GetT() int // Get the number of signatures needed for the threshold.
	GetN() int // Get the number of participants.
}

type CoinProof interface {
	Sig
}

type CoinProofPubInterface interface {
	ThresholdCountInterface
	CheckCoinProof(SignedMessage, CoinProof) error // Check if a coin proof is valid for a message.
	CombineProofs(myPriv Priv, items []*SigItem) (coinVal types.BinVal, err error)
	// DeserializeCoinProof(m *messages.Message) (coinProof CoinProof, size int, err error)
	NewCoinProof() CoinProof // returns an empty coin proof oject
}

type EncodeInterface interface {
	Encode(writer io.Writer) (n int, err error)
	Decode(reader io.Reader) (n int, err error)
}

// type VerifyFunc func(SignedMessage, Sig) (bool, error)

type SecondaryPriv interface {
	GetSecondaryPriv() Priv
}

type SecondaryPub interface {
	VerifySecondarySig(SignedMessage, Sig) (bool, error) // Verify a signature, returns (true, nil) if valid, (false, nil) if invalid, or (false, error) if there was an error verifying the signature
}

// VRFPub interface for public keys that support VRFs
type VRFPub interface {
	// NewVRFProof returns an empty VRFProof object
	NewVRFProof() VRFProof
	// ProofToHash checks the VRF for the message and returns the random bytes if valid
	ProofToHash(m SignedMessage, proof VRFProof) (index [32]byte, err error) // For validating VRFs
}

// MultiPub is for multisignatures.
type MultiPub interface {
	// MergePubPartial merges the pubs without updating the BitID identifiers.
	MergePubPartial(MultiPub)
	// DonePartialMerge should be called after merging keys with MergePubPartial to set the bitid.
	DonePartialMerge(bitid.BitIDInterface)
	// MergePub combines two BLS public key objects into a single one (doing all necessary steps)
	MergePub(MultiPub) (MultiPub, error)
	// GetBitID returns the BitID for this public key
	GetBitID() bitid.BitIDInterface
	// SubMultiPub removes the input pub from he pub and returns the new pub
	SubMultiPub(MultiPub) (MultiPub, error)
	// GenerateSerializedSig serialized the public key and the signature and returns the bytes
	GenerateSerializedSig(MultiSig) ([]byte, error)
}

type MultiSig interface {
	// SubSig removes sig2 from sig1, it assumes sig 1 already contains sig2
	SubSig(MultiSig) (MultiSig, error)
	// MergeBlsSig combines two signatures, it assumes the sigs are valid to be merged
	MergeSig(MultiSig) (MultiSig, error)
}

type AllMultiSig interface {
	MultiSig
	Sig
}

type AllMultiPub interface {
	MultiPub
	Pub
}

// Pub is a public key object
type Pub interface {
	VerifySig(SignedMessage, Sig) (bool, error) // Verify a signature, returns (true, nil) if valid, (false, nil) if invalid, or (false, error) if there was an error verifying the signature
	// CheckSignature will should check what kind of message and signature to verify. It will call CheckCoinProof or VerfiySig
	// depending on the message type.
	CheckSignature(msg *MultipleSignedMessage, sig *SigItem) error
	GetPubBytes() (PubKeyBytes, error) // This is the key we use to sign/verify (might be an hash/operation performed on the original key)
	// GetRealPubBytes is the original unmodified public key. This is used when adding new public keys, from this we can generate all
	// the other values.
	GetRealPubBytes() (PubKeyBytes, error)
	GetPubString() (PubKeyStr, error) // This is the same as GetPubBytes, except returns a string, this is used to sort the keys
	GetSigMemberNumber() int          // This returns the number of members this sig counts for (can be > 1 for multisigs and threshold sigs)
	// GetPubID returns the id of the public key (see type definition for PubKeyID). IMPORTANT: this should only be used by
	// the member checker objects and when sending messages, since the id is based on the current set of member for a consensus index and can chage.
	// GetPubBytes, GetRealPubBytes, GetPubString should be used in other cases.
	// The member checker may change the PubID, after making a ShallowCopy, using AfterSortPubs to assign new IDs.
	GetPubID() (PubKeyID, error)
	SetIndex(PubKeyIndex)                  // This should be called after sorting all if the benchmark is using the index as the key id (see SetUsePubIndex)
	FromPubBytes(PubKeyBytes) (Pub, error) // This creates a public key object from the public key bytes
	ShallowCopy() Pub
	// GetIndex gets the index of the node represented by this key in the consensus participants
	GetIndex() PubKeyIndex

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////

	New() Pub                                                                                            // Creates a new public key object of the same type
	DeserializeSig(m *messages.Message, signType types.SignType) (sigItem *SigItem, size int, err error) // Deserializes a public key and signature object from m, size is the number of bytes read

	//////////////////////////////////////////////////////
	// Interfaces
	/////////////////////////////////////////////////////

	messages.MsgHeader
	EncodeInterface
}

// VRFPriv interface for private key VRF operations.
type VRFPriv interface {
	Evaluate(m SignedMessage) (index [32]byte, proof VRFProof) // For generating VRFs.
}

// Priv is a private key object
type Priv interface {
	ComputeSharedSecret(pub Pub) [32]byte // ComputeSharedSecret should perform Diffie-Hellman.
	GetPub() Pub                          // GetPub returns the public key associated with this private key
	// SetIndex should be called after sorting the pubs (see SetUsePubIndex) with the sorted pub and the new pub key.
	SetIndex(newPubIndex PubKeyIndex)
	GenerateSig(msg SignedMessage, vrf VRFProof, signType types.SignType) (*SigItem, error) // Signs a message and returns the SigItem object containing the signature
	GetPrivForSignType(signType types.SignType) (Priv, error)                               // Returns key that is used for signing the sign type.
	Sign(message SignedMessage) (Sig, error)                                                // Signs a message and returns the signature
	GetBaseKey() Priv                                                                       // Returns the normal version of the key (eg. for partial signature keys returns the normal key)
	ShallowCopy() Priv
	Clean() // Called when the priv key is no longer used

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////
	// New() Priv   // Creates a new private key object of the same type
	NewSig() Sig // Creates an empty sig object of the same type.
}

type VRFProof interface {
	EncodeInterface

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////
	New() VRFProof // Creates a VRFProof oject of the same type
}

// ThrshSig interface of a threshold signature
type ThrshSig interface {
	GetRand() types.BinVal // Get a random binary from the signature if supported.
}

type CorruptInterface interface {
	Corrupt() // Corrupts a signature for testing
}

// Sig is a signature object
type Sig interface {
	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////
	New() Sig // Creates a new sig object of the same type

	//////////////////////////////////////////////////////
	// Interfaces
	/////////////////////////////////////////////////////

	messages.MsgHeader
	EncodeInterface
}

// // MultiSigMsgHeader is a message that also includes signatures of the message
// type MultiSigMsgHeader interface {
// 	messages.MsgHeader
// 	GetSigItems() []*SigItem         // Returns a list of signatures of the message
// 	SetSigItems([]*SigItem)          // Sets the list of signatures of the message
// 	GetSignedHash() messages.HashBytes     // This returns the hash of the message (see issue #22)
// 	GetHashString() messages.HashStr // Same as GetSignedHash except returns the result as a string
// 	GetSignedMessage() []byte                  // Returns the message bytes
// 	ShallowCopy() MultiSigMsgHeader  // Make a shallow copy
// }

// CheckSingleSupporter checks if the message is either a Signed or Unsigned message
// and returns the supporter pub.
// It should be called after CheckSingleSupporter
func GetSingleSupporter(msg messages.MsgHeader) Pub {
	switch v := msg.(type) {
	case *MultipleSignedMessage:
		return v.GetSigItems()[0].Pub
	case *UnsignedMessage:
		return v.GetEncryptPubs()[0]
	default:
		panic(v)
	}
}

// CheckSingleSupporter checks if the message is either a Signed or Unsigned message
// and returns an error if it has a number of supporters not equal to 1.
func CheckSingleSupporter(msg messages.MsgHeader) error {
	switch v := msg.(type) {
	case *MultipleSignedMessage:
		if len(v.GetSigItems()) != 1 {
			return types.ErrInvalidSig
		}
		return nil
	case *UnsignedMessage:
		if len(v.GetEncryptPubs()) != 1 {
			return types.ErrInvalidSig
		}
		return nil
	}
	return types.ErrInvalidHeader
}
