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
Membercheckers are responsible for tracking who can participate in consensus by having a list of their public keys.
Membership can change on each iteration of consenus depding on the implementation of the application and memberchecker.
*/
package memberchecker

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
)

func getRoundCord(mc *absMemberChecker, idx types.ConsensusID, checkPub sig.Pub,
	round types.ConsensusRound) (coordPub sig.Pub, err error) {

	coord := mc.sortedMemberPubs[getRoundIdx(mc.rotateCord, round, idx, len(mc.sortedMemberPubs))]
	if coord == nil {
		panic("should not be nil")
	}
	if checkPub == nil {
		return coord, nil
	}
	return mc.checkCoordInternal(round, coord, checkPub)
}

func getRoundIdx(rotateCord bool, round types.ConsensusRound, idx types.ConsensusID, pubCount int) int {
	roundIdx := int(round)
	if rotateCord {
		// TODO check this
		roundIdx += int(idx.(types.ConsensusInt))
	}
	return utils.Abs(roundIdx % pubCount)
}
