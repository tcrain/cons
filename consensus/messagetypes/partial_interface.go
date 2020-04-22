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
package messagetypes

import (
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// CreatePartial creates a list of partial messages and the corresponding combined message given a header, the number of pieces to
// break the message into and the partial message type.
func CreatePartial(hdr messages.InternalSignedMsgHeader, round types.ConsensusRound, pieces int,
	partialType types.PartialMessageType) (combined *CombinedMessage, partials []messages.InternalSignedMsgHeader, err error) {

	if pieces < 1 {
		return nil, nil, types.ErrNotEnoughPartials
	}
	switch partialType {
	case types.FullPartialMessages:
		logging.Errorf("%+v, %+v\n", hdr, pieces)
		partials, err = GeneratePartialMessageDirect(hdr, round, pieces)
		if err != nil {
			return
		}
	case types.NoPartialMessages:
		return nil, nil, types.ErrInvalidHeader
	default:
		panic("partial type must  be caught")
	}
	combined = NewCombinedMessageFromPartial(partials[0].(*PartialMessage), hdr)
	return
}
