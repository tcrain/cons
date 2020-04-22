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
	"fmt"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/types"
)

type absGeneralStateMachine struct {
	index         types.ConsensusIndex                   // the current consensus index
	mainChannel   channelinterface.MainChannel           // the network channel
	doneChan      chan channelinterface.ChannelCloseType // will put a message on this channel when the test is finished
	closed        *bool
	doneKeep      bool // set to true when DoneKeep is called
	doneClear     bool // set to true when DoneClear is called
	decided       bool
	GeneralConfig *generalconfig.GeneralConfig
}

// IsDone returns true is the test has finished for the state machine.
func (spi *absGeneralStateMachine) IsDone() bool {
	return *spi.closed
}

// FinishedLastRound returns true if the last test index has finished.
func (spi *absGeneralStateMachine) FinishedLastRound() bool {
	return *spi.closed
}

// GetIndex returns the current consensus index.
func (spi *absGeneralStateMachine) GetIndex() types.ConsensusIndex {
	return spi.index
}

// HasDecided returns true if the SM has decided.
func (spi *absGeneralStateMachine) GetDecided() bool {
	return spi.decided
}

// GetDone returns the done status of this SM.
func (spi *absGeneralStateMachine) GetDone() types.DoneType { // TODO cleanup
	if spi.doneKeep {
		return types.DoneKeep
	}
	if spi.doneClear {
		return types.DoneClear
	}
	return types.NotDone
}

func (spi *absGeneralStateMachine) AbsDoneKeep() {
	if spi.doneKeep {
		panic(fmt.Sprint("called done keep twice", spi.index))
	}
	if spi.doneClear {
		panic(fmt.Sprint("called done clear after done keep", spi.index))
	}
	spi.doneKeep = true
}

func (spi *absGeneralStateMachine) AbsDoneClear() {
	if spi.doneClear {
		panic(fmt.Sprint("called done clear twice", spi.index))
	}
	if spi.doneKeep {
		panic(fmt.Sprint("called done keep after done clear", spi.index))
	}
	spi.doneClear = true
}

// Collect is called when the item is about to be garbage collected and is used as a sanity check.
func (spi *absGeneralStateMachine) Collect() {
	if !spi.doneKeep && !spi.doneClear {
		panic("should have called done keep or done clear before garbage collection")
	}
}

// EndTest is called when the test is ending
func (spi *absGeneralStateMachine) EndTest() {
	if spi.GetDone() == types.NotDone {
		spi.AbsDoneClear()
	}
	return
}
