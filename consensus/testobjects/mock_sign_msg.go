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

package testobjects

import (
	"github.com/tcrain/cons/consensus/types"
)

type MockSignMsg []byte

func (mc MockSignMsg) GetSignedHash() types.HashBytes {
	return types.GetHash(mc)
}

func (mc MockSignMsg) GetSignedMessage() []byte {
	return mc
}
