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

package memberchecker

import (
	"github.com/stretchr/testify/assert"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/bls"
	"github.com/tcrain/cons/consensus/auth/sig/ec"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messagetypes"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

var testConfig = &generalconfig.GeneralConfig{NodeChoiceVRFRelaxation: config.DefaultNodeRelaxation,
	CoordChoiceVRF: config.DefaultCoordinatorRelaxtion}

func newAbsRnd(priv sig.Priv) absRandMemberInterface {
	return initAbsRandMemberChecker(priv, nil, testConfig)
}
func newAbsRndByID(priv sig.Priv) absRandMemberInterface {
	return initAbsRandMemberCheckerByID(priv, nil, testConfig)
}

func TestAbsRandMemberCheckerEC(t *testing.T) {
	intTestAbsRandMemberChecker(ec.NewEcpriv, newAbsRnd, t)
	intTestAbsRandMemberChecker(ec.NewEcpriv, newAbsRndByID, t)
}

func TestAbsRandMemberCheckerBLS(t *testing.T) {
	blsFunc := func() (sig.Priv, error) {
		return bls.NewBlspriv(bitid.NewMultiBitIDFromInts)
	}
	intTestAbsRandMemberChecker(blsFunc, newAbsRnd, t)
	intTestAbsRandMemberChecker(blsFunc, newAbsRndByID, t)
}

func TestAbsRandKnownMemberChecker(t *testing.T) {
	intAbsRandKnownMemberCheckerTest(ec.NewEcpriv, t)
}

func intAbsRandKnownMemberCheckerTest(newPrivFunc func() (sig.Priv, error), t *testing.T) {
	numKeys := 100
	privs := make(sig.PrivList, numKeys)
	var err error
	for i := 0; i < numKeys; i++ {
		privs[i], err = newPrivFunc()
		assert.Nil(t, err)
	}
	// sort.Sort(privs)
	privs.SetIndices()

	initMsg := sig.BasicSignedMessage("some message")
	initRnd, _ := privs[0].(sig.VRFPriv).Evaluate(initMsg)
	a := make([]absRandMemberInterface, len(privs))

	pubs := make(sig.PubList, len(privs))
	for i, p := range privs {
		pubs[i] = p.GetPub()
	}

	count := len(privs) / 2
	for i := range privs {
		a[i] = initAbsRoundKnownMemberChecker(nil)
		a[i].gotRand(initRnd, len(privs)/2, privs[0], pubs, nil)
	}

	var memCount int
	for _, p := range pubs {
		if a[0].checkRandMember(nil, false, count, len(privs), p) == nil {
			memCount++
		}
	}
	assert.Equal(t, count, memCount)

	_, coordR0, err := a[0].checkRandCoord(count, len(privs), nil, 0, nil)
	assert.Nil(t, err)

	_, _, err = a[0].checkRandCoord(count, len(privs), nil, 0, coordR0)
	assert.Nil(t, err)

	_, coordR1, err := a[0].checkRandCoord(count, len(privs), nil, 1, nil)
	assert.Nil(t, err)

	_, _, err = a[0].checkRandCoord(count, len(privs), nil, 1, coordR1)
	assert.Nil(t, err)

	assert.NotEqual(t, coordR0, coordR1)
}

func intTestAbsRandMemberChecker(newPrivFunc func() (sig.Priv, error), newMCFunc func(sig.Priv) absRandMemberInterface, t *testing.T) {
	numKeys := 100
	privs := make(sig.PrivList, numKeys)
	var err error
	for i := 0; i < numKeys; i++ {
		privs[i], err = newPrivFunc()
		assert.Nil(t, err)
	}
	// sort.Sort(privs)
	privs.SetIndices()

	initMsg := sig.BasicSignedMessage("some message")
	initRnd, _ := privs[0].(sig.VRFPriv).Evaluate(initMsg)
	a := make([]absRandMemberInterface, len(privs))
	for i, priv := range privs {
		a[i] = newMCFunc(priv).newRndMC(types.SingleComputeConsensusIDShort(types.ConsensusInt(0)), nil)
		a[i].gotRand(initRnd, 0, priv, nil, nil)
	}
	count := len(privs)

	somemsg := messagetypes.NewAuxProofMessage(false)
	somemsg.Round = 1
	msgID := somemsg.GetMsgID()

	for i, priv := range privs {
		if i != 0 { // since we already added the VRF of the local key when gotRand was called, we only check this for other keys
			assert.NotNil(t, a[0].checkRandMember(msgID, false, count, count, priv.GetPub()))
		}

		prf := a[i].getMyVRF(msgID)
		err := a[0].GotVrf(priv.GetPub(), msgID, prf)
		assert.Nil(t, err)

		assert.Nil(t, a[0].checkRandMember(msgID, false, count, count, priv.GetPub()))
	}

	var memberCount int
	for _, priv := range privs {
		if err := a[0].checkRandMember(msgID, false, count/2, count, priv.GetPub()); err == nil {
			memberCount++
		}
	}
	tenPc := int(float64(count) * .2)
	// TODO these may fail b/c of random
	assert.True(t, memberCount <= count/2+2*tenPc)
	assert.True(t, memberCount >= count/2-2*tenPc)

	var coordCount int
	for i := 0; i < 10; i++ {
		for j, priv := range privs {
			somemsg := messagetypes.NewAuxProofMessage(false)
			somemsg.Round = types.ConsensusRound(i)
			msgID := somemsg.GetMsgID()
			prf := a[j].getMyVRF(msgID)
			err := a[0].GotVrf(priv.GetPub(), msgID, prf)
			assert.Nil(t, err)

			if _, _, err := a[0].checkRandCoord(count/2, count, msgID, types.ConsensusRound(i), priv.GetPub()); err == nil {
				coordCount++
			}
		}
	}
	// TODO these may fail b/c of random
	// be sure that after 10 rounds we have at least one coordinator
	assert.True(t, coordCount > 0)
	// be sure we didnt get too many coordinators
	fifteenPc := int(float64(count) * .15)
	assert.True(t, coordCount < testConfig.CoordChoiceVRF*fifteenPc)
}
