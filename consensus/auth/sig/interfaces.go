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
	"github.com/tcrain/cons/consensus/auth/coinproof"
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
	GetT() int          // Get the number of signatures needed for the threshold.
	GetN() int          // Get the number of participants.
	GetPartialPub() Pub // Get the partial public key for this node.
	GetSharedPub() Pub  // Get the shared public key for the threshold.
}

type CoinProofPubInterface interface {
	BasicThresholdInterface
	CheckCoinProof(SignedMessage, *coinproof.CoinProof) error // Check if a coin proof is valid for a message.
	CombineProofs(items []*SigItem) (coinVal types.BinVal, err error)
	DeserializeCoinProof(m *messages.Message) (coinProof *coinproof.CoinProof, size int, err error)
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
	SetIndex(PubKeyIndex)                                                    // This should be called after sorting all if the benchmark is using the index as the key id (see SetUsePubIndex)
	FromPubBytes(PubKeyBytes) (Pub, error)                                   // This creates a public key object from the public key bytes
	ProofToHash(m SignedMessage, proof VRFProof) (index [32]byte, err error) // For validating VRFs
	ShallowCopy() Pub
	// GetIndex gets the index of the node represented by this key in the consensus participants
	GetIndex() PubKeyIndex

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////

	New() Pub // Creates a new public key object of the same type
	// NewVRFProof returns an empty VRFProof object
	NewVRFProof() VRFProof
	DeserializeSig(m *messages.Message, signType types.SignType) (sigItem *SigItem, size int, err error) // Deserializes a public key and signature object from m, size is the number of bytes read

	//////////////////////////////////////////////////////
	// Interfaces
	/////////////////////////////////////////////////////

	messages.MsgHeader
	EncodeInterface
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
	Evaluate(m SignedMessage) (index [32]byte, proof VRFProof)                              // For generating VRFs.
	ShallowCopy() Priv
	Clean() // Called when the priv key is no longer used

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////
	New() Priv   // Creates a new private key object of the same type
	NewSig() Sig // Creates an empty sig object of the same type.
}

type VRFProof interface {
	EncodeInterface

	//////////////////////////////////////////////////////
	// Static methods
	/////////////////////////////////////////////////////
	New() VRFProof // Creates a VRFProof oject of the same type
}

// Sig is a signature object
type Sig interface {
	GetRand() types.BinVal // Get a random binary from the signature if supported.

	Corrupt() // Corrupts a signature for testing

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
