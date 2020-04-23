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

package rpcsetup

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/types"
)

// PregClientWrapper wraps the ParRegClient object with the nework.PregInterface interface.
type PregClientWrapper struct {
	ParRegClientInterface
	id int
}

func NewPregClientWrapper(cli ParRegClientInterface, id int) *PregClientWrapper {
	return &PregClientWrapper{
		ParRegClientInterface: cli,
		id:                    id,
	}
}

func (pr *PregClientWrapper) GenBlsShared(idx, numThresh int) error {
	return pr.ParRegClientInterface.GenBlsShared(pr.id, idx, numThresh)
}
func (pr *PregClientWrapper) GetBlsShared(idx int) (index int, blsShared *bls.BlsSharedMarshal) {
	blssi, err := pr.ParRegClientInterface.GetBlsShared(pr.id, idx)
	if err != nil || blssi.BlsShared == nil {
		panic(types.ErrInvalidSharedThresh)
	}
	mar, err := blssi.BlsShared.PartialUnmarshal()
	if err != nil {
		panic(err)
	}
	ret, err := mar.PartialMarshal()
	if err != nil {
		panic(err)
	}
	return blssi.KeyIndex, &ret
}
func (pr *PregClientWrapper) GenDSSShared(numNonNumbers, numThresh int) error {
	return pr.ParRegClientInterface.GenDSSShared(pr.id, numNonNumbers, numThresh)
}
func (pr *PregClientWrapper) GetDSSShared() ed.DSSSharedMarshaled {
	ret, err := pr.ParRegClientInterface.GetDSSShared(pr.id)
	if err != nil {
		panic(err)
	}
	return *ret
}
func (pr *PregClientWrapper) RegisterParticipant(parInfo *network.ParticipantInfo) error {
	return pr.ParRegClientInterface.RegisterParticipant(pr.id, parInfo)
}
func (pr *PregClientWrapper) GetParticipants(pub sig.PubKeyStr) ([][]*network.ParticipantInfo, error) {
	return pr.ParRegClientInterface.GetParticipants(pr.id, pub)
}
func (pr *PregClientWrapper) GetAllParticipants() ([]*network.ParticipantInfo, error) {
	return pr.ParRegClientInterface.GetAllParticipants(pr.id)
}
