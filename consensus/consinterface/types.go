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

package consinterface

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

func ShouldWaitForRndCoord(memberType types.RndMemberType, gc *generalconfig.GeneralConfig) bool {
	if gc.UseRandCoord {
		return true
	}
	switch memberType {
	case types.KnownPerCons:
		return false
	case types.NonRandom:
		return false
	case types.LocalRandMember:
		return false
	default:
		return true
	}
}

// See BasicConsItem.GetBufferCount interface
type BufferCountFunc func(messages.MsgIDHeader, *generalconfig.GeneralConfig, *MemCheckers) (
	endThreshold int, maxPossible int, msgid messages.MsgID, err error)

// See BasicConsItem.GetHeader interface
type HeaderFunc func(sig.Pub, *generalconfig.GeneralConfig, messages.HeaderID) (messages.MsgHeader, error)
