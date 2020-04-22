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
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var sigAndVrfFields = map[string]string{"SignatureValidation": "VerifySig",
	"SignatureCreation": "GenerateSig", "VRFValidation": "auth/sig/.*ProofToHash",
	"VRFCreation": "auth/sig/.*Evaluate", "CombinePartials": "auth/sig/.*CombinePartialSigs"}

type CPUResult struct {
	AllTime             uint64
	OtherOperations     uint64
	VRFCreation         uint64
	VRFValidation       uint64
	SignatureValidation uint64
	SignatureCreation   uint64
	CombinePartials     uint64
}

var VRFCPUFieldNames = []string{
	"SignatureCreation",
	"VRFCreation",
	"SignatureValidation",
	"VRFValidation",
	"CombinePartials",
	"OtherOperations",
}

var ThrshCPUFieldNames = []string{
	"SignatureCreation",
	"SignatureValidation",
	"CombinePartials",
	"OtherOperations",
}

var SigCPUFieldNames = []string{
	"SignatureCreation",
	"SignatureValidation",
	"OtherOperations",
}

func GetCPUFieldNamesResultFile(results []ResultFile) []VaryField {
	items := make(map[string]bool)
	for _, nxt := range results {
		for _, nxt := range GetCPUFieldNames(nxt.TestOptions) {
			items[nxt] = true
		}
	}
	ret := make([]VaryField, 0, len(items))
	for nxt := range items {
		ret = append(ret, VaryField{VaryField: nxt})
	}
	return ret
}

func GetCPUFieldNames(to types.TestOptions) []string {
	switch to.UsesVRFs() {
	case true:
		return VRFCPUFieldNames
	}
	switch to.SigType {
	case types.EDCOIN, types.TBLS:
		return ThrshCPUFieldNames
	default:
		return SigCPUFieldNames
	}
}

type CPUProfileResult struct {
	CPUResult
	CPUFileName string
	NodeIndex   int
	NodeCount   int
	types.TestOptions
}

type MemProfileResult struct {
	MemStartFileName string
	MemEndFileName   string
	NodeIndex        int
	NodeCount        int
	types.TestOptions
}

func LoadProfiles(folderPath string, toMap map[uint64]types.TestOptions) (
	retCPU []CPUProfileResult, retMem []MemProfileResult, err error) {

	folderPath = filepath.Join(folderPath, "profile")
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		logging.Error(err)
		return nil, nil, err
	}

	// profile filename is profile_{cpu/memstart/memend}_{procIndex}_{testID}_{nodes}_{constype}
	for _, nxt := range files {
		if nxt.IsDir() {
			continue
		}
		if !strings.HasPrefix(nxt.Name(), "profile_") {
			continue
		}

		profileType, nodeIndex, testID, nodeCount, consType, err := readProfileName(nxt.Name())
		if err != nil {
			logging.Error(err)
			return nil, nil, err
		}
		switch profileType {
		case "cpu":
			// Check if you have mem profiles
			cpuProfileName := filepath.Join(folderPath,
				fmt.Sprintf("profile_cpu_%v_%v_%v_%v.out", nodeIndex, testID, nodeCount, consType))

			// Check if nill profile
			if st, err := os.Stat(cpuProfileName); err != nil {
				logging.Error(err)
				return nil, nil, err
			} else if st.Size() == 0 {
				logging.Error("Empty CPU profile, ", cpuProfileName)
				continue
			}

			to, ok := toMap[testID]
			if !ok {
				err = fmt.Errorf("testID %v test options not found", testID)
				logging.Error(err)
				return nil, nil, err
			}
			to.ConsType = types.ConsType(consType)

			fieldsMap := make(map[string]string)

			for _, nxtField := range GetCPUFieldNames(to) {
				if nxtField == "OtherOperations" {
					continue
				}
				fieldsMap[nxtField] = sigAndVrfFields[nxtField]
			}
			totalTime, cpuRes, err := extractCPUProfile(cpuProfileName, fieldsMap)
			if err != nil {
				return nil, nil, err
			}

			var summedTime time.Duration
			for _, nxt := range cpuRes {
				summedTime += nxt
			}
			remainTime := totalTime - summedTime

			retCPU = append(retCPU, CPUProfileResult{
				CPUResult: CPUResult{
					AllTime:             uint64(totalTime / time.Millisecond),
					OtherOperations:     uint64(remainTime / time.Millisecond),
					VRFCreation:         uint64(cpuRes["VRFCreation"] / time.Millisecond),
					SignatureCreation:   uint64(cpuRes["SignatureCreation"] / time.Millisecond),
					VRFValidation:       uint64(cpuRes["VRFValidation"] / time.Millisecond),
					SignatureValidation: uint64(cpuRes["SignatureValidation"] / time.Millisecond),
					CombinePartials:     uint64(cpuRes["CombinePartials"] / time.Millisecond),
				},
				CPUFileName: cpuProfileName,
				NodeCount:   nodeCount,
				NodeIndex:   nodeIndex,
				TestOptions: to,
			})
		case "memstart":
			memStartProfileName := filepath.Join(folderPath,
				fmt.Sprintf("profile_memstart_%v_%v_%v_%v.out", nodeIndex, testID, nodeCount, consType))
			memEndProfileName := filepath.Join(folderPath,
				fmt.Sprintf("profile_memend_%v_%v_%v_%v.out", nodeIndex, testID, nodeCount, consType))
			if _, err := os.Stat(memEndProfileName); err != nil {
				err = fmt.Errorf("error opening memory end profile: %v", err)
				logging.Error(err)
				return nil, nil, err
			}
			to, ok := toMap[testID]
			if !ok {
				err = fmt.Errorf("testID test options not found: %v", testID)
				logging.Error(err)
				return nil, nil, err
			}
			to.ConsType = types.ConsType(consType)

			retMem = append(retMem, MemProfileResult{
				MemEndFileName:   memEndProfileName,
				MemStartFileName: memStartProfileName,
				NodeCount:        nodeCount,
				NodeIndex:        nodeIndex,
				TestOptions:      to,
			})
		}
	}
	return
}

func readProfileName(fileName string) (profileType string, procIndex int,
	testID uint64, nodeCount int, consType int, err error) {

	// profile filename is profile_{cpu/memstart/memend}_{procIndex}_{testID}_{nodes}_{constype}
	fileName = strings.TrimSuffix(fileName, ".out")
	pieces := strings.Split(fileName, "_")
	if len(pieces) != 6 {
		err = fmt.Errorf("invalid result file name: %v", fileName)
		logging.Error(err)
		return
	}
	profileType = pieces[1]
	procIndex, err = strconv.Atoi(pieces[2])
	if err != nil {
		logging.Error(err)
		return
	}
	testID, err = strconv.ParseUint(pieces[3], 10, 64)
	if err != nil {
		logging.Error(err)
		return
	}
	nodeCount, err = strconv.Atoi(pieces[4])
	if err != nil {
		logging.Error(err)
		return
	}
	consType, err = strconv.Atoi(pieces[5])
	if err != nil {
		logging.Error(err)
		return
	}
	return

}
