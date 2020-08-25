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
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/ed"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/network"
	"github.com/tcrain/cons/consensus/types"
	"sync"
	"testing"
)

func TestRpcSetup(t *testing.T) {
	to := types.TestOptions{
		NumTotalProcs:   4,
		SigType:         0,
		UsePubIndex:     true,
		ConsType:        0,
		EncryptChannels: false,
		NoSignatures:    false,
		CoinType:        0,
	}
	parRegs := &PregTestInterface{pregs: cons.GenerateParticipantRegisters(to)}
	running := make(map[int]*SingleConsSetup)
	mutex := &sync.RWMutex{}
	var wg sync.WaitGroup

	var sic, sih bool
	for i := 0; i < to.NumTotalProcs; i++ {
		scs := &SingleConsSetup{
			I:                i,
			To:               to,
			ParReg:           parRegs,
			MyIP:             "127.0.0.1",
			SetInitialConfig: &sic,
			SetInitialHash:   &sih,
			Mutex:            mutex,
			Shared:           &consinterface.Shared{},
		}
		running[i] = scs

		wg.Add(1)
		go func(scs *SingleConsSetup) {
			err := initialSetup(scs)
			assert.Nil(t, err)
			err = runInitialSetup(scs, t)
			assert.Nil(t, err)
			wg.Done()
		}(scs)
	}
	wg.Wait()
}

type PregTestInterface struct {
	pregs []network.PregInterface
}

func (p *PregTestInterface) GenBlsShared(id, idx, numThresh int) error {
	return p.pregs[id].GenBlsShared(idx, numThresh)
}
func (p *PregTestInterface) GetBlsShared(id, idx int) (*network.BlsSharedMarshalIndex, error) {
	idx, b := p.pregs[id].GetBlsShared(idx)
	return &network.BlsSharedMarshalIndex{
		KeyIndex:  id,
		Idx:       idx,
		BlsShared: b,
	}, nil
}
func (p *PregTestInterface) GenDSSShared(id, numNonMembers, numThresh int) error {
	return p.pregs[id].GenDSSShared(numNonMembers, numThresh)
}
func (p *PregTestInterface) GetDSSShared(id int) (*ed.CoinSharedMarshaled, error) {
	ret := p.pregs[id].GetDSSShared()
	return &ret, nil
}
func (p *PregTestInterface) RegisterParticipant(id int, parInfo *network.ParticipantInfo) error {
	return p.pregs[id].RegisterParticipant(parInfo)
}
func (p *PregTestInterface) GetParticipants(id int, pub sig.PubKeyStr) (pi [][]*network.ParticipantInfo, err error) {
	return p.pregs[id].GetParticipants(pub)
}
func (p *PregTestInterface) GetAllParticipants(id int) (pi []*network.ParticipantInfo, err error) {
	return p.pregs[id].GetAllParticipants()
}
func (p *PregTestInterface) Close() error {
	return nil
}
