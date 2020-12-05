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

package parse

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestTable(t *testing.T) {
	h1 := NewHeader("nomax1", false)
	h2 := NewHeader("nomax2", false)
	h3 := NewHeader("nomax3", false)
	caption := "caption"
	round := 2

	str := &strings.Builder{}
	tab, _, err := InitTable("nomax", []Header{h1, h2, h3}, false, str)
	assert.Nil(t, err)
	values := [][3]interface{}{{1, 2.2222, 3}, {4.44444, 5, 6}, {7, 8, 9.99999}}
	_, err = tab.AddRow("r1", values, round)
	assert.Nil(t, err)
	_, err = tab.AddRow("r2", values, round)
	assert.Nil(t, err)
	_, err = tab.Done(caption)
	assert.Nil(t, err)

	h4 := NewHeader("max1", true)
	h5 := NewHeader("max2", true)
	h6 := NewHeader("max3", true)
	tab, _, err = InitTable("nomax", []Header{h4, h5, h6}, false, str)
	assert.Nil(t, err)
	_, err = tab.AddRow("r1", values, round)
	assert.Nil(t, err)
	_, err = tab.AddRow("r2", values, round)
	assert.Nil(t, err)
	_, err = tab.Done(caption)
	assert.Nil(t, err)

	tab, _, err = InitTable("both", []Header{h1, h4, h2}, false, str)
	assert.Nil(t, err)
	_, err = tab.AddRow("r1", values, round)
	assert.Nil(t, err)
	_, err = tab.AddRow("r2", values, round)
	assert.Nil(t, err)
	_, err = tab.Done(caption)
	assert.Nil(t, err)

	fmt.Println(str)

}

func TestTableMinMaxRow(t *testing.T) {
	h1 := NewHeader("nomax1", false)
	h2 := NewHeader("nomax2", false)
	h3 := NewHeader("nomax3", false)
	caption := "caption"
	round := 2

	str := &strings.Builder{}
	tab, _, err := InitTable("maxrow", []Header{h1, h2, h3}, true, str)
	assert.Nil(t, err)
	values := [3][][3]interface{}{{{1, 2.2222, 3}, {4.44444, 5, 6}, {7, 8, 9.99999}},
		{{1, 2.2222, 3}, {4.44444, 5, 6}, {7, 8, 9.99999}},
		{{1, 2.2222, 3}, {4.44444, 5, 6}, {7, 8, 9.99999}}}
	_, err = tab.AddMinMaxRow("r1", values, round)
	assert.Nil(t, err)
	_, err = tab.AddMinMaxRow("r2", values, round)
	assert.Nil(t, err)
	_, err = tab.Done(caption)
	assert.Nil(t, err)

	tab, _, err = InitTable("bothrow", []Header{h1, h2, h3}, true, str)
	assert.Nil(t, err)
	_, err = tab.AddRow("r1", values[0], round)
	assert.Nil(t, err)
	_, err = tab.AddMinMaxRow("r2", values, round)
	assert.Nil(t, err)
	_, err = tab.Done(caption)
	assert.Nil(t, err)

	fmt.Println(str)

}
