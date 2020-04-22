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
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	GenMergedGraphs  = false
	GenPerProcGraphs = true
)

type GraphType int

const (
	NormalType GraphType = iota
	PaperType
)

func (gt GraphType) String() string {
	switch gt {
	case NormalType:
		return fmt.Sprint("Normal")
	case PaperType:
		return fmt.Sprint("Paper")
	default:
		return fmt.Sprintf("Graph%d", gt)
	}
}

type GraphProperties struct {
	GraphType
	Name          string
	Width, Height int
}

var GraphTypes = []GraphProperties{{GraphType: NormalType, Width: 1600, Height: 1200},
	{GraphType: PaperType, Width: 400, Height: 400}}

var MultiPlotGraphTypes = []GraphProperties{{GraphType: NormalType, Width: 1600, Height: 1200},
	{GraphType: PaperType, Width: 800, Height: 600}}

func GenResults(folderPath string) error {
	toMap, err := LoadTestOptions(filepath.Join(folderPath, "testoptions"))
	if err != nil {
		logging.Error(err)
		return err
	}
	allResults, err := LoadResults(folderPath, toMap)
	if err != nil {
		logging.Error(err)
		return err
	}
	allCPUProfile, _, err := LoadProfiles(folderPath, toMap)
	if err != nil {
		logging.Error(err)
		return err
	}
	JoinResults(allResults, allCPUProfile)

	genSets, err := LoadGenSets(folderPath)
	if err != nil {
		logging.Error(err)
		return err
	}

	for _, nxt := range genSets {
		GenGraphByGenSet(folderPath, nxt, allResults)
	}

	return nil
}

func ComputePerToPerNodeCountProfile(results []CPUProfileResult) map[int]map[string]map[uint64][]CPUProfileResult {
	// make a map of per number of nodes
	nodesMap := make(map[int][]CPUProfileResult)
	for _, nxt := range results {
		nodesMap[nxt.NodeCount] = append(nodesMap[nxt.NodeCount], nxt)
	}

	ret := make(map[int]map[string]map[uint64][]CPUProfileResult)
	for i, nxt := range nodesMap {
		ret[i] = ComputePerToProfile(nxt)
	}
	return ret
}

func ComputePerToPerNodeCount(results []ResultFile) map[int]map[string][2]map[uint64][]ResultFile {
	// make a map of per number of nodes
	nodesMap := make(map[int][]ResultFile)
	for _, nxt := range results {
		nodesMap[nxt.NodeCount] = append(nodesMap[nxt.NodeCount], nxt)
	}

	ret := make(map[int]map[string][2]map[uint64][]ResultFile)
	for i, nxt := range nodesMap {
		ret[i] = ComputePerTo(nxt)
	}
	return ret
}

func ComputePerToProfile(results []CPUProfileResult) map[string]map[uint64][]CPUProfileResult {
	ret := make(map[string]map[uint64][]CPUProfileResult)
	for _, nxt := range results {
		testID := nxt.TestOptions.TestID
		nodeType := "total"
		resMap := ret[nodeType]
		if resMap == nil {
			resMap = make(map[uint64][]CPUProfileResult)
			ret[nodeType] = resMap
		}
		resMap[testID] = append(resMap[testID], nxt)
	}

	return ret
}

func ComputePerTo(results []ResultFile) map[string][2]map[uint64][]ResultFile {
	ret := make(map[string][2]map[uint64][]ResultFile)
	for _, nxt := range results {
		testID := nxt.TestOptions.TestID
		nodeType := nxt.NodeType
		perProc := nxt.PerProc
		var resMap [2]map[uint64][]ResultFile
		var ok bool
		if resMap, ok = ret[nodeType]; !ok {
			resMap[0] = make(map[uint64][]ResultFile)
			resMap[1] = make(map[uint64][]ResultFile)
			ret[nodeType] = resMap
		}
		if perProc {
			resMap[0][testID] = append(resMap[0][testID], nxt)
		} else {
			resMap[1][testID] = append(resMap[1][testID], nxt)
		}
	}

	return ret
}

func ComputePerToAllTypesProfile(results []CPUProfileResult) (ret map[uint64][]CPUProfileResult) {
	ret = make(map[uint64][]CPUProfileResult)
	for _, nxt := range results {
		testID := nxt.TestOptions.TestID
		ret[testID] = append(ret[testID], nxt)
	}
	return ret
}

func ComputePerToAllTypes(results []ResultFile) (ret [2]map[uint64][]ResultFile) {
	ret[0] = make(map[uint64][]ResultFile)
	ret[1] = make(map[uint64][]ResultFile)
	for _, nxt := range results {
		testID := nxt.TestOptions.TestID
		perProc := nxt.PerProc
		if perProc {
			ret[0][testID] = append(ret[0][testID], nxt)
		} else {
			ret[1][testID] = append(ret[1][testID], nxt)
		}
	}
	return ret
}

func SortByNodeCount(results []ResultFile) map[types.ConsType][]ResultFile {
	ret := make(map[types.ConsType][]ResultFile)
	for _, nxt := range results {
		cid := nxt.TestOptions.ConsType
		ret[cid] = append(ret[cid], nxt)
	}
	for _, v := range ret {
		sort.Slice(v, func(i, j int) bool {
			return v[i].NodeCount < v[j].NodeCount
		})
	}
	return ret
}

func SortByNodeCountProfile(results []CPUProfileResult, ret map[types.ConsType][]CPUProfileResult) {
	for _, nxt := range results {
		cid := nxt.TestOptions.ConsType
		ret[cid] = append(ret[cid], nxt)
	}
	for _, v := range ret {
		sort.Slice(v, func(i, j int) bool {
			return v[i].NodeCount < v[j].NodeCount
		})
	}
}

// GetTestDiffs returns the fields that differ in the test options.
func GenTestDiffs(toSlice []types.TestOptions, includeNodeCount bool) []string {
	var ret []string
	for i, nxt := range toSlice {
		for _, nxtNxt := range toSlice[i:] {
			ret = append(ret, nxt.FieldDiff(nxtNxt)...)
		}
	}
	ret = utils.RemoveDuplicatesSortCountString(ret)
	_, ret = utils.RemoveFromSliceString("TestID", ret)
	if includeNodeCount {
		// Include node count as a difference
		ret = append(ret, "NodeCount")
	}
	return ret
}

func removeDifferentProcs(input []CPUProfileResult) (ret []CPUProfileResult) {
	items := make(map[struct {
		types.TestOptions
		int
	}]CPUProfileResult)
	for _, nxt := range input {
		items[struct {
			types.TestOptions
			int
		}{nxt.TestOptions, nxt.NodeCount}] = nxt
	}
	for _, nxt := range items {
		ret = append(ret, nxt)
	}
	return ret
}

func JoinResults(allResults []ResultFile, profileResults []CPUProfileResult) {
	for i, nxt := range allResults {
		for _, nxtCPU := range profileResults {
			if nxt.NodeCount == nxtCPU.NodeCount && nxt.TestOptions == nxtCPU.TestOptions {
				nxt.CPUProfileResults = append(nxt.CPUProfileResults, nxtCPU)
				allResults[i] = nxt
			}
		}
	}
}

func MakeOutput(folderPath string, varyField VaryField, extraNames []VaryField, cpuFieldNames []VaryField,
	items map[types.ConsType][]ResultFile,
	profileItems map[types.ConsType][]CPUProfileResult, includeConsType bool) error {

	// st := stats.MergedStats{}
	// stV := reflect.ValueOf(st)
	// var fNames []string
	// for i := 0; i < stV.NumField(); i++ {
	//	fNames = append(fNames, stV.Type().Field(i).Name)
	// }

	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		logging.Error(err)
		return err
	}

	// make a profile map
	profileItems = make(map[types.ConsType][]CPUProfileResult)
	for ct, nxt := range items {
		for _, nxt := range nxt {
			profileItems[ct] = append(profileItems[ct], nxt.CPUProfileResults...)
		}
	}

	// make profile graphs
	// First a map for all profiles
	resultsMapProfile := make(map[struct{ title, xIndex string }][]string)
	for ct, res := range profileItems {
		nxtTitle := "CPUTimeSpent"
		extraNames := append([]VaryField{{VaryField: "NodeIndex"}}, append([]VaryField{varyField}, extraNames...)...)
		varyField := VaryField{VaryField: "NodeIndex"}
		for _, nxtRes := range res {
			if nxtName, err := writeProfileStatsFile(
				"cpu", folderPath, cpuFieldNames, varyField, extraNames, 2,
				nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			} else {
				resultsMapProfile[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}] = append(
					resultsMapProfile[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}], nxtName)
			}
		}
	}
	// We only want one profile (TODO merge instead)
	for k, val := range profileItems {
		profileItems[k] = removeDifferentProcs(val)
	}
	for ct, res := range profileItems {
		nxtTitle := "CPUTimeSpent"
		for _, nxtRes := range res {
			if nxtName, err := writeProfileStatsFile(
				"cpu", folderPath, cpuFieldNames, varyField, append([]VaryField{varyField}, extraNames...),
				1, nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			} else {
				resultsMapProfile[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}] = append(
					resultsMapProfile[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}], nxtName)
			}
		}
	}

	resultsMap := make(map[struct{ title, xIndex string }][]string)
	var err error
	for ct, res := range items {
		for _, nxtRes := range res {
			for _, nxt := range StatsNames {
				minStr := "Min" + nxt
				maxStr := "Max" + nxt
				var fileName string
				if fileName, err = makeStatsFile(folderPath, nxt, maxStr, minStr, varyField, extraNames,
					nxtRes, ct, includeConsType); err != nil {

					logging.Error(err)
					return err
				}
				resultsMap[struct{ title, xIndex string }{nxt, varyField.GetTitle()}] = append(
					resultsMap[struct{ title, xIndex string }{nxt, varyField.GetTitle()}], fileName)
			}
			ms := nxtRes.MergedStats

			// Output the times
			nxtTitle := "AllTime"
			totalTime := float64(ms.FinishTime.Sub(ms.StartTime)) / float64(time.Millisecond)
			var fileName string
			if fileName, err = intMakeStatsFile(folderPath, nxtTitle, "None", "None",
				varyField, extraNames, totalTime, 0, 0, nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			}
			resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}] = append(
				resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}], fileName)

			// TimeDivCons
			nxtTitle = "TotalTimeDivNumCons"
			printTitle := ActTitle
			totalTimeDivNumCons := float64(ms.FinishTime.Sub(ms.StartTime)) / float64(ms.RecordCount) / float64(time.Millisecond)
			if fileName, err = intMakeStatsFile(folderPath, nxtTitle, "None", "None",
				varyField, extraNames, totalTimeDivNumCons, 0, 0, nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			}
			resultsMap[struct{ title, xIndex string }{printTitle, varyField.GetTitle()}] = append(
				resultsMap[struct{ title, xIndex string }{printTitle, varyField.GetTitle()}], fileName)

			// TotalConsTime
			nxtTitle = "TotalConsTime"
			consTime := float64(ms.ConsTime) / float64(time.Millisecond)
			if fileName, err = intMakeStatsFile(folderPath, nxtTitle, "None", "None",
				varyField, extraNames, consTime, 0, 0, nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			}
			resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}] = append(
				resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}], fileName)

			// TimePerCons
			nxtTitle = "TimePerCons"
			timePerCons := float64(ms.ConsTime) / float64(ms.RecordCount) / float64(time.Millisecond)
			minConsTime := float64(ms.MinConsTime) / float64(time.Millisecond)
			maxConsTime := float64(ms.MaxConsTime) / float64(time.Millisecond)
			if fileName, err = intMakeStatsFile(folderPath, nxtTitle, "MaxTimePerCons", "MinTimePerCons",
				varyField, extraNames, timePerCons, maxConsTime, minConsTime, nxtRes, ct, includeConsType); err != nil {

				logging.Error(err)
				return err
			}
			resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}] = append(
				resultsMap[struct{ title, xIndex string }{nxtTitle, varyField.GetTitle()}], fileName)

			// Graph by each cons
			nxtTitle = "TimeBetweenDecisions"
			for i, nxt := range ms.SinceTimes {
				if fileName, err = writeLineStatsFile(folderPath, nxtTitle, "None", "None",
					VaryField{VaryField: "ConsIndex"}, i, append([]VaryField{varyField}, extraNames...),
					float64(nxt)/float64(time.Millisecond),
					0, 0, nxtRes, ct, includeConsType, 2); err != nil {

					logging.Error(err)
					return err
				}
			}
			resultsMap[struct{ title, xIndex string }{nxtTitle, "ConsIndex"}] = append(
				resultsMap[struct{ title, xIndex string }{nxtTitle, "ConsIndex"}], fileName)

			// Graph by each cons
			nxtTitle = "TimeEachCons"
			for i, nxt := range ms.ConsTimes {
				if fileName, err = writeLineStatsFile(folderPath, nxtTitle, "None", "None",
					VaryField{VaryField: "ConsIndex"}, i, append([]VaryField{varyField}, extraNames...),
					float64(nxt)/float64(time.Millisecond),
					0, 0, nxtRes, ct, includeConsType, 2); err != nil {

					logging.Error(err)
					return err
				}
			}
			resultsMap[struct{ title, xIndex string }{nxtTitle, "ConsIndex"}] = append(
				resultsMap[struct{ title, xIndex string }{nxtTitle, "ConsIndex"}], fileName)

		}
	}
	// Generate the graphs
	var multiPlotStrings [][]string
	for title, fileNames := range resultsMap {
		// remove duplicate fileNames
		newFileNames := utils.RemoveDuplicatesSortCountString(fileNames)
		// make the graph
		var nxtMultiPlotString []string
		if nxtMultiPlotString, err = GenGraph(title.title, title.xIndex, title.title, folderPath, extraNames,
			newFileNames, GraphTypes, MultiPlotTitles[title.title]); err != nil {
			logging.Error(err)
			return err
		}
		if len(nxtMultiPlotString) > 0 {
			multiPlotStrings = append(multiPlotStrings, nxtMultiPlotString)
		}
	}
	// Generate the multiplot strings
	if len(multiPlotStrings) > 0 {
		var joinedMultiPlotStrings []string
		for i := range multiPlotStrings[0] {
			var nxtString []string
			for j := range multiPlotStrings {
				nxtString = append(nxtString, multiPlotStrings[j][i])
			}
			joinedMultiPlotStrings = append(joinedMultiPlotStrings, strings.Join(nxtString, "; "))
		}

		//for _, nxt := range joinedMultiPlotStrings {
		//	fmt.Println("\n\n!!!!!!!!!!!!!!!!!!11\n\n", nxt, "\n\n!!!!!!!!!!!!!!!!!!11\n\n")
		// }
		if err := GenMultiPlot(folderPath, MultiPlotGraphTypes, joinedMultiPlotStrings); err != nil {
			logging.Error(err)
			return err
		}
	}

	// Generate the profile graphs
	for title, fileNames := range resultsMapProfile {
		// remove duplicate fileNames
		newFileNames := utils.RemoveDuplicatesSortCountString(fileNames)
		// make the graph
		if err := GenProfileGraph(title.title, title.xIndex, title.title, len(cpuFieldNames)+1,
			folderPath, extraNames, newFileNames, GraphTypes); err != nil {

			logging.Error(err)
			return err
		}
	}

	return nil
}

func makeStatsFile(folderPath, nxt, maxStr, minStr string, varyField VaryField, extraNames []VaryField, nxtRes ResultFile,
	ct types.ConsType, includeConsType bool) (fileName string, err error) {

	stV := reflect.ValueOf(nxtRes.MergedStats)
	var val, max, min interface{}
	val = stV.FieldByName(nxt).Interface().(uint64)
	if stats.DivStats[nxt] {
		val = float64(val.(uint64)) / float64(nxtRes.MergedStats.RecordCount)
	}
	max = stV.FieldByName(maxStr).Interface().(uint64)
	if stats.DivStats[maxStr] {
		max = float64(max.(uint64)) / float64(nxtRes.MergedStats.RecordCount)
	}
	min = stV.FieldByName(minStr).Interface().(uint64)
	if stats.DivStats[minStr] {
		min = float64(min.(uint64)) / float64(nxtRes.MergedStats.RecordCount)
	}

	return intMakeStatsFile(folderPath, nxt, maxStr, minStr, varyField, extraNames, val, max, min, nxtRes, ct, includeConsType)
}

func intMakeStatsFile(folderPath, nxt, maxStr, minStr string, varyField VaryField, extraNames []VaryField,
	val, max, min interface{}, nxtRes ResultFile, ct types.ConsType, includeConsType bool) (fileName string, err error) {

	varyValue := reflect.ValueOf(nxtRes).FieldByName(varyField.VaryField).Interface()
	// results filename is graph_{varyField}_{extraNameVal}_{constype}.txt
	// nxtRes.NodeCount
	return writeLineStatsFile(folderPath, nxt, maxStr, minStr, varyField, varyValue, extraNames, val,
		max, min, nxtRes, ct, includeConsType, len(extraNames))
}

func writeLineStatsFile(folderPath, nxt, maxStr, minStr string, varyField VaryField, varyValue interface{}, extraNames []VaryField,
	val, max, min interface{}, nxtRes ResultFile, ct types.ConsType, includeConsType bool, numTitles int) (fileName string, err error) {

	var out bytes.Buffer
	// extraName := genExtraNamesString(extraNames)
	consName, extraNameVal := genExtraNamesStringVal(ct, extraNames, nxtRes, len(extraNames), varyField)

	endPath := fmt.Sprintf(
		"graph_%v_%v_%v%v.txt", nxt, varyField.VaryField, consName, extraNameVal)

	fileName = filepath.Join(folderPath, endPath)

	consNameTitle, extraNameValTitle := genExtraNamesStringVal(ct, extraNames, nxtRes, numTitles, varyField)

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		if includeConsType || len(extraNameValTitle) == 0 {
			out.WriteString(fmt.Sprintf("# %v%v\t%v\t%v\t%v\n", consNameTitle, extraNameValTitle, minStr, nxt, maxStr))
		} else {
			out.WriteString(fmt.Sprintf("# %v\t%v\t%v\t%v\n", extraNameValTitle[1:], minStr, nxt, maxStr))
		}
	} else if err != nil {
		logging.Error(err)
		return "", err
	}
	varyValueStr := varyField.GetValue(varyValue)
	out.WriteString(fmt.Sprintf("%v\t%v\t%v\t%v\t\n", varyValueStr, min, val, max))

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		logging.Error(err)
		return "", err
	}
	defer func() { _ = file.Close() }()

	_, err = file.Write(out.Bytes())
	return
}

func genExtraNamesStringVal(ct types.ConsType, extraNames []VaryField, res interface{}, numValues int,
	frontVal VaryField) (consName, restName string) {

	if len(extraNames) == 0 { // || (len(extraNames) == 1 && extraNames[0].VaryField == "ConsType") {
		return "", ""
	}
	var ret []string
	for _, nxt := range extraNames {
		if nxt.VaryField == "ConsType" {
			consName = fmt.Sprintf("%v", nxt.GetValue(ct))
			continue
		}
		val := reflect.ValueOf(res).FieldByName(nxt.VaryField).Interface()
		nxtStr := fmt.Sprintf("%v%v", nxt.GetTitle(), nxt.GetValue(val))
		if nxt.VaryField == frontVal.VaryField {
			ret = append([]string{nxtStr}, ret...)
		} else {
			ret = append(ret, nxtStr)
		}
	}
	ret = ret[:utils.Min(numValues, len(ret))]
	if len(ret) > 0 {
		restName = "," + strings.Join(ret, ",")
	}
	return
}

func genExtraNamesString(extraNames []VaryField, singleValue bool) string {
	var ret strings.Builder
	for _, nxt := range extraNames {
		if nxt.VaryField == "ConsType" {
			continue
		}
		ret.WriteString(fmt.Sprintf("_%v", nxt.VaryField))
		if singleValue {
			break
		}
	}
	return ret.String()
}
