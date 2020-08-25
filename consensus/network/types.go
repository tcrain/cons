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

/*
This package contains utilities for setting up a network of consensus nodes.
*/
package network

import (
	"bytes"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/types"
)

// NetworkPropagationSetup describes how the messages will be propagated for an experiment.
type NetworkPropagationSetup struct {
	NPT                   types.NetworkPropagationType // The type of network propagation used.
	FanOut                int                          // The number of nodes a node sends/fowards its messages to (only used for P2p or Random)
	AdditionalP2PNetworks int                          // Generate additional P2P connection networks
}

// ParticipantInfo describes a participant that will take place in an experiment.
type ParticipantInfo struct {
	RegisterCount int                           // The index of the pub key
	Pub           sig.PubKeyStr                 // The public key of the participant
	ConInfo       []channelinterface.NetConInfo // The addresses that the participant is listening on.
	ExtraInfo     []byte                        // Any extra information to register
}

// SortPub is uses to create a sorted list of public keys
type SortParInfo []*ParticipantInfo

func (a SortParInfo) Len() int      { return len(a) }
func (a SortParInfo) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortParInfo) Less(i, j int) bool {
	// Sorting is done by comparing the result of the Public key string
	return a[i].Pub < a[j].Pub
}

func ParInfoListToPubList(pr []*ParticipantInfo, pubKeys sig.PubList) []sig.Pub {
	ret := make([]sig.Pub, len(pr))
	for i, nxt := range pr {
		ret[i] = GetPubFromList(sig.PubKeyBytes(nxt.Pub), pubKeys)
	}
	return ret
}

func GetPubFromList(pBytes sig.PubKeyBytes, pubs []sig.Pub) sig.Pub {
	for _, p := range pubs {
		b, err := p.GetRealPubBytes()
		if err != nil {
			panic(err)
		}
		if bytes.Equal(pBytes, b) {
			return p
		}
	}
	panic("could not find pub")
}

type ParRegClientInterface interface {
	GenBlsShared(id, idx, numThresh int) error
	GetBlsShared(id, idx int) (*BlsSharedMarshalIndex, error)
	GenDSSShared(id, numNonMembers, numThresh int) error
	GetDSSShared(id int) (*ed.CoinSharedMarshaled, error)
	RegisterParticipant(id int, parInfo *ParticipantInfo) error
	GetParticipants(id int, pub sig.PubKeyStr) (pi [][]*ParticipantInfo, err error)
	GetAllParticipants(id int) (pi []*ParticipantInfo, err error)
	Close() (err error)
}

// BlsSharedMarshalIndex is used as the reply to the rpc call to ParticipantRegister.GetBlsShared.
type BlsSharedMarshalIndex struct {
	KeyIndex  int
	Idx       int
	BlsShared *bls.BlsSharedMarshal
}
