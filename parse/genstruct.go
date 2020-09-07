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

/*
When adding new statistics, must add them to the StatsNames list in order for a graphs to be generated for that stat.
If the stat should be divided by the number of nodes for the values graphed, then the name must also be added to
DivStats.
*/

package parse

import (
	"fmt"
	"github.com/tcrain/cons/consensus/types"
	"path/filepath"
	"reflect"
)

// StatsNames is the list of statistics that will be used to create graphs
var StatsNames = []string{
	"Signed",
	"Validated",
	"VRFCreated",
	"VRFValidated",
	"ThrshCreated",
	"RoundDecide",
	"RoundParticipation",
	"DiskStorage",
	"DecidedNil",
	"MsgsSent",
	"BytesSent",
	"BufferForwardTimeouts",
	"ProposalCount",
	"MemberCount",
	"ProposalForwarded",
	"ForwardState",
	"ProgressTimeout",
}

// DivStats are the ones that need to be divided by the number of nodes to get the per node value.
var DivStats = map[string]bool{"RoundParticipation": true, "RoundDecide": true, "DiskStorage": true,
	"Signed": true, "ThrshCreated": true, "Validated": true, "VRFCreated": true, "VRFValidated": true,
	"CoinValidated": true, "CoinCreated": true, "ForwardState": true,
	"MsgsSent": true, "BytesSent": true, "MaxMsgsSent": true, "MaxBytesSent": true, "ProposalForwarded": true,
	"MinMsgsSent": true, "MinBytesSent": true}

// List of fields in the test options structure
var possibleVaryStringFields = []string{"NodeType"}
var possibleVaryIntFields = []string{"NodeCount"}
var possibleVaryBoolFields = []string{"PerProc"}

var stringVaryFieldsMap = map[string]bool{}
var intVaryFieldsMap = map[string]bool{}
var boolVaryFieldsMap = map[string]bool{}

func init() {
	to := types.TestOptions{}

	toV := reflect.ValueOf(to)
	for i := 0; i < toV.NumField(); i++ {
		switch t := toV.Field(i).Kind(); t {
		case reflect.String:
			possibleVaryStringFields = append(possibleVaryStringFields, toV.Type().Field(i).Name)
		case reflect.Bool:
			possibleVaryBoolFields = append(possibleVaryBoolFields, toV.Type().Field(i).Name)
		case reflect.Int, reflect.Uint64:
			possibleVaryIntFields = append(possibleVaryIntFields, toV.Type().Field(i).Name)
		default:
			panic(fmt.Errorf("unsupproted type %v in test options", t))
		}
	}
	for _, nxt := range possibleVaryIntFields {
		intVaryFieldsMap[nxt] = true
	}
	for _, nxt := range possibleVaryBoolFields {
		boolVaryFieldsMap[nxt] = true
	}
	for _, nxt := range possibleVaryStringFields {
		stringVaryFieldsMap[nxt] = true
	}
}

// NodeCount int
// PerProc   bool
// NodeType  string

// SplitByField returns a map where the keys are the values of fieldName from allResults, and the values are maps from
// the cons types to the result files that correspond the the value of the fieldName.
func SplitByField(allResults []ResultFile, fieldName string) map[interface{}][]ResultFile {

	ret := make(map[interface{}][]ResultFile)

	items := make(map[interface{}]bool)
	for _, nxt := range allResults {
		items[reflect.ValueOf(nxt).FieldByName(fieldName).Interface()] = true
	}

	if stringVaryFieldsMap[fieldName] {
		for nxt := range items {
			ret[nxt] = FilterResultsFiles(allResults,
				[]FilterStringField{{
					Name:   fieldName,
					Values: []string{nxt.(string)}},
				}, nil, nil)
		}
	}

	if intVaryFieldsMap[fieldName] {
		for nxt := range items {
			var val int
			switch v := nxt.(type) {
			case int:
				val = v
			case uint64:
				val = int(v)
			case types.ConsType:
				val = int(v)
			default:
				panic(v)
			}
			ret[nxt] = FilterResultsFiles(allResults, nil,
				[]FilterIntField{{
					Name:   fieldName,
					Values: []int{val}},
				}, nil)
		}
	}

	if boolVaryFieldsMap[fieldName] {
		for nxt := range items {
			ret[nxt] = FilterResultsFiles(allResults, nil, nil,
				[]FilterBoolField{{
					Name:   fieldName,
					Values: []bool{nxt.(bool)},
				}})
		}
	}

	return ret
}

// FilterResultsFiles returns a map of the results from allResults that match any value for either of the string filters
// or int filters.
func FilterResultsFiles(allResults []ResultFile, stringFilters []FilterStringField,
	intFilters []FilterIntField, boolFilters []FilterBoolField) []ResultFile {

	var ret []ResultFile
loopNextResult:
	for _, nxt := range allResults {
		nxtV := reflect.ValueOf(nxt)
		for _, fil := range stringFilters {
			val := nxtV.FieldByName(fil.Name).Interface().(string)
			for _, nxtVal := range fil.Values {
				if val == nxtVal {
					ret = append(ret, nxt)
					continue loopNextResult
				}
			}
		}
		if len(boolFilters) > 0 {
			foundAllbool := true
			for _, fil := range boolFilters {
				val := nxtV.FieldByName(fil.Name).Interface().(bool)
				var found bool
				for _, nxtVal := range fil.Values {
					if val == nxtVal {
						found = true
						// ret = append(ret, nxt)
						// continue loopNextResult
					}
				}
				if !found {
					foundAllbool = false
				}
			}
			if foundAllbool {
				ret = append(ret, nxt)
				continue loopNextResult
			}
		}
		for _, fil := range intFilters {
			var val int
			switch v := nxtV.FieldByName(fil.Name).Interface().(type) {
			case int:
				val = v
			case uint64:
				val = int(v)
			case types.ConsType:
				val = int(v)
			default:
				panic(v)
			}
			for _, nxtVal := range fil.Values {
				if val == nxtVal {
					ret = append(ret, nxt)
					continue loopNextResult
				}
			}
		}
	}

	return ret
}

func GenGraphByGenSet(folderPath string, genSet GenSet, allResults []ResultFile) {
	// 1st filter items
	// 2nd recursively go through split items, at each tail create a graph set using the GenItems
	filteredResults := FilterResultsFiles(allResults, genSet.FilterStringFields, genSet.FilterIntFields, genSet.BoolFilterFields)

	if len(genSet.SplitItems) == 0 {
		genSet.SplitItems = []*SplitItem{nil}
	}
	for _, nxt := range genSet.SplitItems {
		recGen(filteredResults, genSet.GenItems, folderPath, genSet.Name, nxt, true)
	}

}

func recGen(resultFiles []ResultFile, genItems []GenItem, folderPath, endPath string,
	splitItem *SplitItem, includeConsType bool) {

	if splitItem == nil {
		// generate
		// MakeOutput(folderPath, varyField string, extraNames, cpuFieldNames []string, items map[types.ConsType][]ResultFile,
		//	profileItems map[types.ConsType][]CPUProfileResult, includeConsType bool)
		for _, nxt := range genItems {
			if err := MakeOutput(filepath.Join(folderPath, endPath), nxt.VaryField, nxt.ExtraFields,
				GetCPUFieldNamesResultFile(resultFiles), SortByNodeCount(resultFiles),
				nil, includeConsType); err != nil {
				panic(err)
			}
		}
		return
	}
	for nxtVal, nxtItems := range SplitByField(resultFiles, splitItem.FieldName) {
		if splitItem.FieldName == "ConsType" {
			includeConsType = false
		}
		nxtPath := fmt.Sprintf("%v_%v%v", endPath, nxtVal, splitItem.FieldName)
		recGen(nxtItems, genItems, folderPath, nxtPath, splitItem.NextSplitItem, includeConsType)
	}
}
