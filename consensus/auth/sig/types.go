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
	"github.com/tcrain/cons/consensus/types"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"time"
)

///////////////////////////////////////////////////////////
// Types
//////////////////////////////////////////////////////////

type SigStats struct {
	PubSize, SigSize, ThrshShareSize, VRFSize        int
	SigVerifyTime, SignTime                          time.Duration
	ShareVerifyTime, ShareGenTime                    time.Duration
	ShareCombineTime                                 time.Duration
	MultiCombineTime                                 time.Duration
	VRFGenTime, VRFVerifyTime                        time.Duration
	AllowsCoin, AllowsMulti, AllowsThresh, AllowsVRF bool
}

// ConsIDPub contains a consensus id and a public key.
type ConsIDPub struct {
	ID  types.ConsensusID
	Pub Pub
}

// PubKeyIndex is the index of the pub key in the sorted list of consensus participants
type PubKeyIndex int

// PubKeyBytes is the public key encoded as bytes
type PubKeyBytes []byte

// PubKeyStr is the public key endoced as a string
type PubKeyStr string

// PubKeyID is a string that uniquely identifies a public key for a certain consensus instance
// this may be an integer or the full public key, depending on how SetUsePubIndex was set
type PubKeyID string

// EdSuite is the group used by schnorr and threshold sigs (this can be changed)
var EdSuite = edwards25519.NewBlakeSHA256Ed25519()

// the group used by eddsa signature (this is fixed)
var EddsaGroup = new(edwards25519.Curve)

///////////////////////////////////////////////////////////
// Global variables
//////////////////////////////////////////////////////////

// The following settings should be changed in "../../generalconfig.go" and not here
var UsePubIndex = true

var UseMultisig = false

var SleepValidate = false

var BlsMultiNew = false

// SetUseMultisig set wheter to use multi-signatures or not
// If true will use multisig, must default to false
func SetUseMultisig(val bool) {
	UseMultisig = val
}

// GetUseMultisig returns the state set by SetUseMultisig
func GetUseMultisig() bool {
	return UseMultisig
}

// SetUsePubIndex sets how a node sends its identifier in messages.
// If true, the when sending a pub it only sends the index it is in the list of participants
// TODO change index when members are chagned, right now indecies are static are the start
func SetUsePubIndex(val bool) {
	UsePubIndex = val
}

// GetUsePubIndex returns the state set by SetUsePubIndex
func GetUsePubIndex() bool {
	return UsePubIndex
}

// SetSleepValidate sets whether or not nodes validate sigs (unsafe just for testing)
// If true then we dont validate sigs, just sleep and return true.
func SetSleepValidate(val bool) {
	SleepValidate = val
}

// DoSleepValidation is called instead of doing a validation when SleepValidate is set to true
func DoSleepValidation(d time.Duration) {

	time.Sleep(d)
}

// SetBlsMultiNew sets what type of multisigs to use, if SetUseMultisig was set to true
// If true we use multi-sigs from https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
func SetBlsMultiNew(val bool) {
	BlsMultiNew = val
}

func GetBlsMultiNew() bool {
	return BlsMultiNew
}
