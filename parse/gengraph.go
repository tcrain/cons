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
	"bufio"
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/utils"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const gpPath = "./scripts/graphscripts/plotfile.gp"
const gpMultiPath = "./scripts/graphscripts/multiplotfile.gp"
const gpProfilePath = "./scripts/graphscripts/plotprofile.gp"

func lessFunc(iItems, jItems []string) bool {
	if len(iItems) != len(jItems) {
		return len(iItems) < len(jItems)
	}
	for k := range iItems {
		iIntStart := len(iItems[k]) - len(strings.TrimLeftFunc(iItems[k], func(r rune) bool {
			return unicode.IsNumber(r)
		}))
		jIntStart := len(jItems[k]) - len(strings.TrimLeftFunc(jItems[k], func(r rune) bool {
			return unicode.IsNumber(r)
		}))
		var iInt, jInt int
		var err error
		if iIntStart > 0 {
			if iInt, err = strconv.Atoi(iItems[k][:iIntStart]); err != nil {
				logging.Error(err)
				panic(err)
			}
		}
		if jIntStart > 0 {
			if jInt, err = strconv.Atoi(jItems[k][:jIntStart]); err != nil {
				logging.Error(err)
				panic(err)
			}
		}
		if iInt != jInt {
			return iInt < jInt
		}
		iStr := iItems[k][iIntStart:]
		jStr := jItems[k][jIntStart:]
		if iStr != jStr {
			return iStr < jStr
		}
	}
	return false
}

func GenGraph(title, xLabel, yLabel, folderPath string, extraNames []VaryField, inputFiles []string,
	properties []GraphProperties, isMultiPlot int) (multiPlotStrings []string, err error) {

	yLabel = GetYIndex(yLabel)
	// Sort the file names
	sort.Slice(inputFiles, func(i, j int) bool {
		_, iFile := filepath.Split(inputFiles[i])
		_, jFile := filepath.Split(inputFiles[j])
		iItems := strings.Split(iFile, "_")
		jItems := strings.Split(jFile, "_")
		return lessFunc(iItems, jItems)
	})

	// Get all the columns
	var columns []string
	for _, nxt := range inputFiles {
		file, err := os.Open(nxt) //, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			logging.Error(err)
			return nil, err
		}

		scn := bufio.NewScanner(file)
		for scn.Scan() {
			columns = append(columns, strings.Fields(scn.Text())[0])
		}
		_ = file.Close()
	}
	columns = utils.RemoveDuplicatesSortCountString(columns)

	// Sort the lines of the files
	for _, nxt := range inputFiles {
		file, err := os.Open(nxt) //, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			logging.Error(err)
			return nil, err
		}

		var lines []string
		scn := bufio.NewScanner(file)
		for scn.Scan() {
			lines = append(lines, scn.Text())
		}
		_ = file.Close()

		// Sort the lines
		if len(lines) < 1 {
			panic("empty file")
		}
		afterLines := lines[1:]

		// add any missing header column lines // TODO why do we need this
		columnMap := make(map[string]bool)
		for _, nxt := range lines {
			columnMap[strings.Fields(nxt)[0]] = true
		}
		for _, nxt := range columns {
			if _, ok := columnMap[nxt]; !ok {
				// TODO why is this here?
				afterLines = append(afterLines, nxt+"\t0\t0\t0")
			}
		}

		// sort the lines
		sort.Slice(afterLines, func(i, j int) bool {
			iItems := strings.Fields(afterLines[i])[:1]
			jItems := strings.Fields(afterLines[j])[:1]
			return lessFunc(iItems, jItems)
		})

		// Store them back to the file
		lines = append(lines[:1], afterLines...)
		buf := bytes.NewBuffer(nil)
		for _, nxtLine := range lines {
			if _, err := buf.WriteString(nxtLine); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString("\n"); err != nil {
				panic(err)
			}
		}
		if err := ioutil.WriteFile(nxt, buf.Bytes(), os.ModePerm); err != nil {
			logging.Error(err)
			return nil, err
		}
	}

	extraName := genExtraNamesString(extraNames, true)

	for _, nxtP := range properties {
		arg := fmt.Sprintf("tit='%v'; ylab='%v'; xlab='%v'; filenames='%v'; outputfile='%v'; width='%v'; height='%v'",
			title, yLabel, xLabel, strings.Join(inputFiles, " "),
			filepath.Join(folderPath, fmt.Sprintf("%v%v_%v%v.png", nxtP, xLabel, yLabel, extraName)),
			nxtP.Width, nxtP.Height)

		if isMultiPlot > 0 {
			mps := fmt.Sprintf("tit%v='%v'; ylab%v='%v'; xlab%v='%v'; filenames%v='%v'",
				isMultiPlot, title, isMultiPlot, yLabel, isMultiPlot, xLabel, isMultiPlot, strings.Join(inputFiles, " "))
			multiPlotStrings = append(multiPlotStrings, mps)
		}

		// Write the command to a file so we can rerun it
		prtCmd := fmt.Sprintf("gnuplot -e \"%v\" %v", arg, gpPath)
		cmdFileName := filepath.Join(folderPath, fmt.Sprintf("%v%v_%v%v.sh", nxtP, xLabel, yLabel, extraName))
		if err := ioutil.WriteFile(cmdFileName, []byte(prtCmd), os.ModePerm); err != nil {
			return nil, err
		}

		newCmd := []string{"gnuplot", "-e", arg, gpPath}
		cmd := exec.Command(newCmd[0], newCmd[1:]...)
		logging.Print("Running command: ", newCmd)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			logging.Errorf("Error running command %s: %v, %v", cmd.Args, err, cmd.Stderr)
			return nil, err
		}
		logging.Print("Ran command successfully", cmd.Args)
	}
	return
}

func GenMultiPlot(folderPath string, properties []GraphProperties, args []string) error {
	for i, nxtP := range properties {

		arg := args[i]
		arg = fmt.Sprintf("%v; outputfile='%v'; width='%v'; height='%v'",
			arg,
			filepath.Join(folderPath, fmt.Sprintf("%v_multiplot.png", nxtP)),
			nxtP.Width, nxtP.Height)

		// Write the command to a file so we can rerun it
		prtCmd := fmt.Sprintf("gnuplot -e \"%v\" %v", arg, gpMultiPath)
		cmdFileName := filepath.Join(folderPath, fmt.Sprintf("%v_multiplot.sh", nxtP))
		if err := ioutil.WriteFile(cmdFileName, []byte(prtCmd), os.ModePerm); err != nil {
			return err
		}

		newCmd := []string{"gnuplot", "-e", arg, gpMultiPath}
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
	}
	return nil

}

func GenProfileGraph(title, xLabel, yLabel string, numFields int,
	folderPath string, extraNames []VaryField, inputFiles []string, properties []GraphProperties) error {

	yLabel = GetYIndex(yLabel)
	// Sort the file names
	sort.Slice(inputFiles, func(i, j int) bool {
		_, iFile := filepath.Split(inputFiles[i])
		_, jFile := filepath.Split(inputFiles[j])
		iItems := strings.Split(iFile, "_")
		jItems := strings.Split(jFile, "_")
		return lessFunc(iItems, jItems)
	})

	extraName := genExtraNamesString(extraNames, true)

	// We need to put all in a single file since no nest for in gnuplot
	var output bytes.Buffer
	buf, err := ioutil.ReadFile(inputFiles[0])
	if err != nil {
		logging.Error(err)
		return err
	}
	if _, err := output.Write(buf); err != nil {
		return err
	}
	for _, nxtFile := range inputFiles[1:] {
		buf, err := ioutil.ReadFile(nxtFile)
		if err != nil {
			logging.Error(err)
			return err
		}
		buffer := bytes.NewBuffer(buf)
		if _, err := buffer.ReadBytes('\n'); err != nil {
			return err
		}
		if _, err := output.Write(buffer.Bytes()); err != nil {
			return err
		}
	}
	outName := filepath.Join(folderPath, fmt.Sprintf("%v_%v%v-prof.txt", xLabel, yLabel, extraName))
	if err := ioutil.WriteFile(outName, output.Bytes(), os.ModePerm); err != nil {
		return err
	}

	for _, nxtP := range properties {
		arg := fmt.Sprintf("numCol='%v'; tit='%v'; ylab='%v'; xlab='%v'; filenames='%v'; outputfile='%v'; width='%v'; height='%v'",
			numFields, title, yLabel, xLabel, outName,
			filepath.Join(folderPath, fmt.Sprintf("%v%v_%v%v.png", nxtP, xLabel, yLabel, extraName)),
			nxtP.Width, nxtP.Height)

		// Write the command to a file so we can rerun it
		prtCmd := fmt.Sprintf("gnuplot -e \"%v\" %v", arg, gpPath)
		cmdFileName := filepath.Join(folderPath, fmt.Sprintf("%v%v_%v%v.sh", nxtP, xLabel, yLabel, extraName))
		if err := ioutil.WriteFile(cmdFileName, []byte(prtCmd), os.ModePerm); err != nil {
			return err
		}

		newCmd := []string{"gnuplot", "-e", arg, gpProfilePath}
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
	}
	return nil
}
