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

package statemachine

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/generalconfig"
	"io"
)

////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////

// AbsRandSMNotSupported should be included for state machines that do not support generating random bytes.
type AbsRandSMNotSupported int

func (spi AbsRandSMNotSupported) GetRand() (ret [32]byte) {
	return
}

////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////

// AbsRandSM can be included as part of the state machine if it supports generating random bytes.
type AbsRandSM struct {
	randBytes [32]byte
	useRand   bool
	gc        *generalconfig.GeneralConfig
}

// NewAbsRandSM allocates the internal object for a state machine that generates random bytes.
func NewAbsRandSM(initRandBytes [32]byte, useRand bool) AbsRandSM {
	return AbsRandSM{randBytes: initRandBytes,
		useRand: useRand}
}

// AbsRandInit should be called by the Init function of the state machine.
func (spi *AbsRandSM) AbsRandInit(gc *generalconfig.GeneralConfig) {
	spi.gc = gc
}

// GetRand returns the 32 random bytes for this state machine.
// It should only be called after RandHasDecided.
func (spi *AbsRandSM) GetRand() [32]byte {
	if !spi.useRand {
		panic("rand bytes not enabled")
	}
	return spi.randBytes
}

// RandGetProposal encodes random bytes using a VRF to the buffer.
// It uses the rand bytes input to NewAbsRandSM to generate the new bytes.
func (spi *AbsRandSM) RandGetProposal(buff io.Writer) {
	if spi.useRand {
		_, proof := spi.gc.Priv.(sig.VRFPriv).Evaluate(sig.BasicSignedMessage(spi.randBytes[:]))
		if _, err := proof.Encode(buff); err != nil {
			panic(err)
		}
	}
}

// RandCheckDecision should be called during CheckDecisions and return an error if the decided random bytes
// were not generated correctly. TODO actually verify the proof
func (spi *AbsRandSM) RandCheckDecision(buf io.Reader) (err error) {
	if spi.useRand {
		vrPrf := spi.gc.Priv.GetPub().(sig.VRFPub).NewVRFProof()
		_, err = vrPrf.Decode(buf)
		// The previous random is used as the input to the next TODO verify this is correct
		// _, err = proposer.ProofToHash(sig.BasicSignedMessage(spi.randBytes[:]), vrPrf)
	}
	return
}

// GetRndNumBytes returns the number of bytes that would be added from RandGetProposal.
func (spi *AbsRandSM) GetRndNumBytes() (n int) {
	if spi.useRand {
		return 32
	}
	return 0
}

// RandHasDecided should be called by the HasDecided function of the state machine
// with the encoded random bytes. If setBytes is true the state will be updated
// (false is just used for checking that the bytes were generated in a valid way).
// It returns an error if the bytes were not generated correctly.
func (spi *AbsRandSM) RandHasDecided(proposer sig.Pub, buf io.Reader, setBytes bool) (n int, err error) {
	if spi.useRand {
		vrPrf := spi.gc.Priv.GetPub().(sig.VRFPub).NewVRFProof()
		n, err = vrPrf.Decode(buf)
		if err != nil {
			return
		}
		// The previous random is used as the input to the next
		var rnd [32]byte
		rnd, err = proposer.(sig.VRFPub).ProofToHash(sig.BasicSignedMessage(spi.randBytes[:]), vrPrf)
		if err != nil {
			return
		}
		if setBytes {
			spi.randBytes = rnd
		}
	}
	return
}
