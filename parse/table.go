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
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
	"strconv"
)

type Header struct {
	name   string
	minMax bool
}

func NewHeader(name string, minMax bool) Header {
	return Header{
		name:   name,
		minMax: minMax,
	}
}

type PrintTable struct {
	headers      []Header
	writer       io.Writer
	hasMinMaxRow bool
	n            int
}

func InitTable(leftHeader string, headers []Header, hasMinMaxRow bool, writer io.Writer) (ret *PrintTable, n int, err error) {
	defer func() {
		n = ret.n
	}()

	ret = &PrintTable{
		headers:      headers,
		writer:       writer,
		hasMinMaxRow: hasMinMaxRow,
	}
	var mStr string
	if hasMinMaxRow {
		mStr = "l"
	}
	var hasMinMax bool
	if err = ret.writeStr(fmt.Sprintf("\\begin{table}\n\\centering\n\\begin{tabular}{ l %v |", mStr)); err != nil {
		return
	}
	var str string
	for _, nxt := range headers {
		switch nxt.minMax {
		case true:
			str = " | c  c  c"
			hasMinMax = true
		case false:
			str = " | c"
		}
		if err = ret.writeStr(str); err != nil {
			return
		}
	}
	var lStr string
	switch hasMinMaxRow {
	case true:
		lStr = fmt.Sprintf("\\multicolumn{2}{c ||}{%v}", leftHeader)
	case false:
		lStr = leftHeader
	}

	if err = ret.writeStr(fmt.Sprintf(" }\n%v", lStr)); err != nil {
		return
	}

	for i, nxt := range headers {
		switch nxt.minMax {
		case true:
			var col string
			if i < len(headers)-1 {
				col = "|"
			}
			str = fmt.Sprintf(" & \\multicolumn{3}{c%v}{%v}", col, nxt.name)
		case false:
			switch hasMinMax {
			case true:
				str = fmt.Sprintf(" & \\multirow{3}{*}{%v}", nxt.name)
			case false:
				str = fmt.Sprintf(" & %v", nxt.name)
			}
		}
		if err = ret.writeStr(str); err != nil {
			return
		}
	}
	if err = ret.writeStr(" \\\\\n"); err != nil {
		return
	}
	if hasMinMax {
		for _, nxt := range headers {
			switch nxt.minMax {
			case true:
				str = " & min & avg & max"
			case false:
				str = " &"
			}
			if err = ret.writeStr(str); err != nil {
				return
			}
		}
		if err = ret.writeStr(" \\\\\n"); err != nil {
			return
		}
	}
	if err = ret.writeStr("\\hline\n"); err != nil {
		return
	}
	return
}

func (pt *PrintTable) writeStr(str string) error {
	n, err := pt.writer.Write([]byte(str))
	pt.n += n
	return err
}

func roundFloat(nxt interface{}, roundTo int) interface{} {
	var round float64
	switch roundTo {
	case 0:
		round = 1
	default:
		round = float64(utils.Exp(10, roundTo))
	}
	ret := nxt
	switch v := nxt.(type) {
	case string:
		if f, err := strconv.ParseFloat(v, 0); err == nil {
			ret = math.Round(f*round) / round
		}
	case float32:
		ret = math.Round(float64(v)*round) / round
	case float64:
		ret = math.Round(v*round) / round
	}
	return ret
}

func (pt *PrintTable) AddRow(title interface{}, values [][3]interface{}, roundTo int) (n int, err error) {
	defer func() {
		n = pt.n
	}()
	pt.n = 0

	if len(values) != len(pt.headers) {
		fmt.Println(title, values, pt.headers)
		panic("must have same number of rows as headers")
	}
	var mStr string
	if pt.hasMinMaxRow {
		mStr = fmt.Sprintf("%v &", title)
	} else {
		mStr = fmt.Sprintf("%v", title)
	}
	if err = pt.writeStr(mStr); err != nil {
		return
	}

	for i, nxt := range values {
		for j := 0; j < 3; j++ {
			nxt[j] = roundFloat(nxt[j], roundTo)
		}
		var str string
		switch pt.headers[i].minMax {
		case true:
			str = fmt.Sprintf(" & %v & %v & %v", nxt[0], nxt[1], nxt[2])
		case false:
			str = fmt.Sprintf(" & %v", nxt[1])
		}
		if err = pt.writeStr(str); err != nil {
			return
		}
	}
	if err = pt.writeStr(fmt.Sprintf(" \\\\ \n")); err != nil {
		return
	}
	return
}

func (pt *PrintTable) AddMinMaxRow(title interface{}, values [3][][3]interface{}, roundTo int) (n int, err error) {
	defer func() {
		n = pt.n
	}()
	pt.n = 0

	if len(values[0]) != len(pt.headers) {
		panic("must have same number of rows as headers")
	}
	if err = pt.writeStr(fmt.Sprintf("\\multirow{3}{*}{%v}", title)); err != nil {
		return
	}

	for k, idStr := range []string{"min", "avg", "max"} {
		if err = pt.writeStr(fmt.Sprintf(" & %v", idStr)); err != nil {
			return
		}
		for i, nxt := range values[k] {
			for j := 0; j < 3; j++ {
				nxt[j] = roundFloat(nxt[j], roundTo)
			}
			var str string
			switch pt.headers[i].minMax {
			case true:
				str = fmt.Sprintf(" & %v & %v & %v", nxt[0], nxt[1], nxt[2])
			case false:
				str = fmt.Sprintf(" & %v", nxt[1])
			}
			if err = pt.writeStr(str); err != nil {
				return
			}
		}
		if err = pt.writeStr(fmt.Sprintf(" \\\\ \n")); err != nil {
			return
		}
	}
	return
}

func (pt *PrintTable) Done(caption string) (n int, err error) {
	defer func() {
		n = pt.n
	}()
	n = 0
	if err = pt.writeStr(fmt.Sprintf("\\end{tabular}\n"+
		"\\caption{%v}\\label{tab:%v}\n"+
		"\\end{table}\n\n",
		caption, caption)); err != nil {

		return
	}
	return
}
