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
package ed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDSSSharedMarshal(t *testing.T) {
	partSec := thrshdssshare.MemberScalars[0]
	partPub := thrshdssshare.MemberPoints[0].Clone().Mul(partSec, nil)

	dssm, err := thrshdssshare.PartialMarshal()
	assert.Nil(t, err)
	dss, err := dssm.PartialUnMartial()
	assert.Nil(t, err)

	newPartSec := dss.MemberScalars[0]
	assert.True(t, partSec.Equal(newPartSec))

	assert.True(t, partPub.Equal(dss.MemberPoints[0]))

}
