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
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func makeFlipTable(inputFiles []string, folderPath, xLabel, title string) (err error) {
	var columns []Header
	rows := make([][3][][3]interface{}, 0)
	rowTitles := make([]string, 0)
	rowMinMax := make([]bool, 0)

	var isMinMax bool
	for _, nxtFile := range inputFiles {
		var buf []byte
		buf, err = ioutil.ReadFile(nxtFile)
		if err != nil {
			logging.Error(err)
			return
		}
		lines := strings.Split(string(buf), "\n")
		var currentMinMax bool
		if strings.Fields(lines[0])[2] != "None" {
			isMinMax = true
			currentMinMax = true
		}
		columns = append(columns, NewHeader(strings.Fields(lines[0])[1], false))
		for i, nxtLine := range lines[1:] {
			fields := strings.Fields(nxtLine)
			if len(fields) > 0 { // only the last field should be len 0
				if len(rows) <= i {
					var itm [3][][3]interface{}
					rows = append(rows, itm)
					rowTitles = append(rowTitles, fields[0])
					rowMinMax = append(rowMinMax, currentMinMax)
				}
				rows[i][0] = append(rows[i][0], [3]interface{}{0, fields[1], 0})
				rows[i][1] = append(rows[i][1], [3]interface{}{0, fields[2], 0})
				rows[i][2] = append(rows[i][2], [3]interface{}{0, fields[3], 0})
			}
		}
	}
	var f *os.File
	if f, err = openTableFile(folderPath, tableExtraString); err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	var tab *PrintTable
	if tab, _, err = InitTable(xLabel, columns, isMinMax, f); err != nil {
		return
	}
	for i, nxt := range rows {
		switch rowMinMax[i] {
		case true:
			if _, err = tab.AddMinMaxRow(rowTitles[i], nxt, 2); err != nil {
				return
			}
		case false:
			if _, err = tab.AddRow(rowTitles[i], nxt[1], 2); err != nil {
				return
			}
		}
	}
	if _, err = tab.Done(title); err != nil {
		return
	}
	return
}

func makeTable(inputFiles []string, folderPath, xLabel, title string) (err error) {

	// first get the column names
	var fields []Header
	var buf []byte
	buf, err = ioutil.ReadFile(inputFiles[0])
	if err != nil {
		logging.Error(err)
		return
	}
	lines := strings.Split(string(buf), "\n")
	var isMinMax bool
	if strings.Fields(lines[0])[2] != "None" {
		isMinMax = true
	}
	for _, nxtLine := range lines[1:] {
		flds := strings.Fields(nxtLine)
		if len(flds) > 0 {
			fields = append(fields, NewHeader(flds[0], isMinMax))
		}
	}
	var f *os.File
	if f, err = openTableFile(folderPath, ""); err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	var tab *PrintTable
	if tab, _, err = InitTable(xLabel, fields, false, f); err != nil {
		return
	}
	for _, nxtFile := range inputFiles {
		var buf []byte
		buf, err = ioutil.ReadFile(nxtFile)
		if err != nil {
			logging.Error(err)
			return
		}
		newLines := strings.Split(string(buf), "\n")
		var rows [][3]interface{}
		for _, nxtLine := range newLines[1:] {
			fields := strings.Fields(nxtLine)
			if len(fields) > 0 {
				rows = append(rows, [3]interface{}{fields[1], fields[2], fields[3]})
			}
		}
		if _, err = tab.AddRow(strings.Fields(newLines[0])[1], rows, 1); err != nil {
			return
		}
	}
	if _, err = tab.Done(title); err != nil {
		return
	}
	return
}

func writeTableHeaders(folderPath string) (err error) {
	if err = writeTableFileHeader(folderPath, ""); err != nil {
		logging.Error(err)
		return err
	}
	if err = writeTableFileHeader(folderPath, tableExtraString); err != nil {
		logging.Error(err)
		return err
	}
	return nil
}

func WriteLatexHeader(writer io.Writer) (n int, err error) {
	return writer.Write([]byte("\\documentclass{article}\n" +
		"\\usepackage[landscape]{geometry}\n" +
		"\\usepackage{multirow}\n" +
		"\\usepackage{graphicx}\n" +
		"\\begin{document}\n\n"))
}

func WriteLatexFooter(writer io.Writer) (n int, err error) {
	return writer.Write([]byte("\n\\end{document}\n"))
}

func buildTablePdfs(folderPath string) error {
	if err := runPDFLatex(folderPath, fmt.Sprintf(tableFileString, "")); err != nil {
		return err
	}
	if err := runPDFLatex(folderPath, fmt.Sprintf(tableFileString, tableExtraString)); err != nil {
		return err
	}
	return nil
}

func runPDFLatex(folderPath, fileName string) error {
	newCmd := []string{"pdflatex", fmt.Sprintf("-output-directory=%v", folderPath), filepath.Join(folderPath, fileName)}
	cmd := exec.Command(newCmd[0], newCmd[1:]...)
	logging.Print("Running command: ", newCmd)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		logging.Errorf("Error running command %s: %v, %v", cmd.Args, err, cmd.Stderr)
		return err
	}
	logging.Print("Ran command successfully", cmd.Args)
	return nil
}

var tableFileString = "tables%v.tex"
var tableExtraString = "flip"

func openTableFile(folderName, extra string) (*os.File, error) {
	tableFileName := filepath.Join(folderName, fmt.Sprintf(tableFileString, extra))
	return os.OpenFile(tableFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func writeTableFileHeader(folderName string, extra string) error {
	f, err := openTableFile(folderName, extra)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err = WriteLatexHeader(f); err != nil {
		return err
	}
	return nil
}

func writeTableFooters(folderPath string) (err error) {
	// write the table file footer
	if err = finishTableFile(folderPath, ""); err != nil {
		logging.Error(err)
		return err
	}
	if err = finishTableFile(folderPath, tableExtraString); err != nil {
		logging.Error(err)
		return err
	}
	return
}

func finishTableFile(folderName string, extra string) error {
	f, err := openTableFile(folderName, extra)
	if err != nil {
		return err
	}
	if _, err = WriteLatexFooter(f); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
