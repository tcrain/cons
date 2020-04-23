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
	"github.com/tcrain/cons/consensus/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

func extractCPUProfile(fileName string, fields map[string]string) (totalTime time.Duration, res map[string]time.Duration, err error) {
	var fl []string
	for _, v := range fields {
		fl = append(fl, v)
	}
	showCmd := fmt.Sprintf("-show=%v", strings.Join(fl, "|"))
	cmdString := []string{"go", "tool", "pprof", "-unit=ms", "-text", "-cum", showCmd, fileName}
	cmd := exec.Command(cmdString[0], cmdString[1:]...)
	// var out bytes.Buffer
	// cmd.Stdout = &out
	var errBuff bytes.Buffer
	cmd.Stderr = &errBuff
	var cmdOut []byte
	if cmdOut, err = cmd.Output(); err != nil {
		logging.Error(errBuff.String())
		return
	}

	res = make(map[string]time.Duration)

	scanner := bufio.NewScanner(bytes.NewBuffer(cmdOut))
	for scanner.Scan() {
		str := scanner.Text()
		// Duration: 2.87mins, Total samples = 170510ms (98.92%)
		if strings.HasPrefix(str, "Duration:") {
			// Get the total time
			if totalTime, err = time.ParseDuration(strings.Fields(str)[5]); err != nil {
				logging.Error(err)
				return
			}
		}
		if strings.Contains(str, "show=") {
			continue
		}
		for name, nxtField := range fields {
			var match bool
			if match, err = regexp.MatchString(nxtField, str); err != nil {
				logging.Error(err)
				return
			} else if match {
				// 134600ms 78.94% 78.94%   134600ms 78.94%  github.com/tcrain/cons/consensus/auth/sig/ec.(*Ecpub).ProofToHash
				var dur time.Duration
				if dur, err = time.ParseDuration(strings.Fields(str)[3]); err != nil {
					logging.Error(err)
					return
				}
				res[name] = dur
			}
		}
	}
	if err = scanner.Err(); err != nil {
		logging.Error(err)
		return
	}

	return
}

func writeProfileStatsFile(profileType string, folderPath string, fieldNames []VaryField, varyField VaryField, extraNames []VaryField,
	numExtraNames int, profileResult interface{}, ct types.ConsType, includeConsType bool) (fileName string, err error) {

	var out bytes.Buffer
	// extraName := genExtraNamesString(extraNames)
	consName, extraNameVal := genExtraNamesStringVal(ct, extraNames, profileResult, len(extraNames), varyField)

	endPath := fmt.Sprintf(
		"graph_profile_%v_%v_%v%v.txt", profileType, varyField.VaryField, consName, extraNameVal)

	fileName = filepath.Join(folderPath, endPath)

	consNameTitle, extraNameValTitle := genExtraNamesStringVal(ct, extraNames, profileResult, numExtraNames+1, varyField)

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// Write the title line
		out.WriteString(fmt.Sprintf("%v", ct))
		for _, nxt := range fieldNames {
			out.WriteString(fmt.Sprintf("\t%v", nxt.GetTitle()))
		}
		out.WriteString("\n")
	} else if err != nil {
		logging.Error(err)
		return "", err
	}
	// Write the values
	if includeConsType || len(extraNameValTitle) == 0 {
		out.WriteString(fmt.Sprintf("%v%v", consNameTitle, extraNameValTitle))
	} else {
		out.WriteString(fmt.Sprintf("%v", extraNameValTitle[1:]))
	}
	for _, nxt := range fieldNames {
		val := reflect.ValueOf(profileResult).FieldByName(nxt.VaryField).Interface()
		out.WriteString(fmt.Sprintf("\t%d", val))
	}
	out.WriteString("\n")

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		logging.Error(err)
		return "", err
	}
	defer func() { _ = file.Close() }()

	_, err = file.Write(out.Bytes())
	return
}
