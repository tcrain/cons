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
	"encoding/json"
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

type ResultFile struct {
	CPUProfileResults []CPUProfileResult
	FileName          string
	stats.MergedStats
	NodeCount int
	PerProc   bool
	ByzNode   bool
	Total     bool
	NodeType  string
	types.TestOptions
}

// LoadResults load the MergedStats results for the given test options from the folder path.
func LoadResults(folderPath string, toMap map[uint64]types.TestOptions) (ret []ResultFile, err error) {

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		logging.Error(err)
		return nil, err
	}

	// stats filename is stats_{perProc/merged}_{fileName}_{testID}_{nodes}_{constype}
	// results filename is results_{perProc/merged}_{fileName}_{testID}_{nodes}_{constype}
	for _, nxt := range files {
		if nxt.IsDir() {
			continue
		}
		if !strings.HasPrefix(nxt.Name(), "stats") {
			continue
		}

		hdrName, perProc, perProcString, nodeType, testID, numNodes, consType, err := readItem(nxt.Name())
		if err != nil {
			logging.Error(err)
			return nil, err
		}
		if hdrName != "stats" {
			panic(nxt.Name())
		}
		statsName := fmt.Sprintf("stats_%v_%v_%v_%v_%d.txt", perProcString, nodeType, testID, numNodes, consType)
		resultsName := fmt.Sprintf("results_%v_%v_%v_%v_%d.txt", perProcString, nodeType, testID, numNodes, consType)

		mr, err := GetMergedResults(filepath.Join(folderPath, statsName))
		if err != nil {
			logging.Error(err)
			return nil, err
		}
		to, ok := toMap[testID]
		if !ok {
			err = fmt.Errorf("testID test options not found: %v", testID)
			logging.Error(err)
			return nil, err
		}
		to.ConsType = types.ConsType(consType)

		var byzNode, Total bool
		switch nodeType {
		case "total":
			Total = true
		case "byz":
			byzNode = true
		case "nonByz":
			byzNode = false
		default:
			panic(nodeType)
		}

		ret = append(ret, ResultFile{
			FileName:    filepath.Join(folderPath, resultsName),
			ByzNode:     byzNode,
			Total:       Total,
			PerProc:     perProc,
			NodeType:    nodeType,
			MergedStats: mr,
			NodeCount:   numNodes,
			TestOptions: to})
	}
	return
}

// readItem parses a results file name
func readItem(fileName string) (hdrName string, perProc bool, perProcString, nodeType string, testID uint64, numNodes, consType int, err error) {
	fileName = strings.TrimSuffix(fileName, ".txt")
	pieces := strings.Split(fileName, "_")
	if len(pieces) != 6 {
		err = fmt.Errorf("invalid result file name: %v", fileName)
		logging.Error(err)
		return
	}
	hdrName = pieces[0]
	perProcString = pieces[1]
	switch pieces[1] {
	case "perProc":
		perProc = true
	case "merged":
	default:
		err = fmt.Errorf("invalid perProc string: %v", pieces[1])
		logging.Error(err)
		return
	}
	nodeType = pieces[2]
	testID, err = strconv.ParseUint(pieces[3], 10, 64)
	if err != nil {
		logging.Error(err)
		return
	}
	numNodes, err = strconv.Atoi(pieces[4])
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

func LoadGenSets(folderPath string) (ret []GenSet, err error) {
	var raw []byte
	raw, err = ioutil.ReadFile(filepath.Join(folderPath, GenSetsFolderName, GenSetsFileName))
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &ret)
	return
}

// LoadTestOptions loads all the test options files in the folder path, and stores
// them in a map where TestOptions.TestID is used as the key.
func LoadTestOptions(folderPath string) (map[uint64]types.TestOptions, error) {
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		logging.Error(err)
		return nil, err
	}
	results := make(map[uint64]types.TestOptions)
	for _, nxt := range files {
		if !nxt.IsDir() && strings.HasSuffix(nxt.Name(), ".json") {
			to, err := types.GetTestOptions(filepath.Join(folderPath, nxt.Name()))
			if err != nil {
				logging.Error(err)
				return nil, err
			}
			results[to.TestID] = to
		}
	}
	return results, nil
}

// GetMergedResults generates a stats.MergedStats object from a json formatted file.
func GetMergedResults(filePath string) (mergedStats stats.MergedStats, err error) {
	var raw []byte
	raw, err = ioutil.ReadFile(filePath)
	if err != nil {
		logging.Error(err)
		return
	}
	err = json.Unmarshal(raw, &mergedStats)
	return
}
