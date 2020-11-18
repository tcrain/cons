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
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	GenSetsFileName   = "gensets.json"
	GenSetsFolderName = "gensets"
)

func GetYIndex(value string) string {
	if v, ok := YIndexMap[value]; ok {
		return v
	}
	return value
}

var YIndexMap = map[string]string{"MsgsSent": "Messages Sent", "RoundDecide": "Decision Round", "BytesSent": "Bytes Sent"}

var MvCons4BcastTypeMap = map[string]string{"Normal": "", "Direct": "Direct", "Indices": "Sync"}
var SocMap = map[string]string{"NextRound": "nr", "SendProof": "sp"}
var RndCTMap = map[string]string{
	"BinConsRnd1":   "BC:s1",
	"BinConsRnd2":   "BC:ns1",
	"BinConsRnd3":   "BC:s2",
	"BinConsRnd4":   "BC:ns2",
	"BinConsRnd5":   "BC:s3",
	"BinConsRnd6":   "BC:ns3",
	"MvCons2":       "MC:3S",
	"MvBinConsRnd2": "MC:4S",
	"MvCons3":       "MC:2S",
	"RbBcast1":      "RB:2S",
	"RbBcast2":      "RB:3S",
}
var CoinMap = map[string]string{
	"KnownCoin":       "kc",
	"LocalCoin":       "lc",
	"StrongCoin1":     "sc",
	"StrongCoin1Echo": "sce",
	"StrongCoin2":     "pc",
	"StrongCoin2Echo": "pce",
}

var ByzMap = map[string]string{
	"BinaryBoth":       "B",
	"BinaryFlip":       "F",
	"HalfHalfFixedBin": "H",
	"HalfHalfNormal":   "HF",
	"Mute":             "M",
	"NonFaulty":        "N",
}

var RndMemberMap = map[string]string{
	"NotRandom":     "NR",
	"KnownPerCons":  "KPC",
	"VRFPerCons":    "VPC",
	"VRFPerMessage": "VPM",
}

var TrueFalseMap = map[string]string{
	"true":  "Y",
	"false": "N",
}

var ActTitle = "Time Per Decision (ms)"
var ConsTitle = "Time Per Consensus (ms)"
var MultiPlotTitles = map[string]int{ActTitle: 1, "BytesSent": 2, "MsgsSent": 3, "RoundDecide": 4}
var MultiPlotTitles2 = map[string]int{ActTitle: 1, ConsTitle: 2, "ConsBytesSent": 3, "ConsMsgsSent": 4}
var MultiPlotTitles3 = map[string]int{ActTitle: 1, ConsTitle: 2, "ConsBytesSent": 3, "Signed": 4}
var MultiPlotTitles4 = map[string]int{ActTitle: 1, "ConsMsgsSent": 2, "ConsBytesSent": 3, "Signed": 4}
var MultiPlotFile = "./scripts/graphscripts/multiplotfile.gp"
var MultiPlotFile2 = "./scripts/graphscripts/multiplotfile2.gp"
var MultiPlotFile3 = "./scripts/graphscripts/multiplotfile3.gp"
var MultiPlotFile4 = "./scripts/graphscripts/multiplotfile4.gp"
var AllMultiPlotTitles = []map[string]int{MultiPlotTitles, MultiPlotTitles2, MultiPlotTitles3, MultiPlotTitles4}
var AllMultipPlotFiles = []string{MultiPlotFile, MultiPlotFile2, MultiPlotFile3, MultiPlotFile4}
var SigMsgMap = map[string]string{"true": "y", "false": "n"}

var GenPerNodeByRndMemberType = GenSet{
	Name:             "RandomMemberType",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"},
		ExtraFields: []VaryField{
			{VaryField: "ConsType", Title: "CT:", NameMap: RndCTMap},
			{VaryField: "RndMemberType", Title: "RMT", NameMap: RndMemberMap},
		}}},
}

var GenPerNodeByConsCoin = GenSet{
	Name:             "NodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType", NameMap: RndCTMap},
		// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
		// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
		{VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
	}}},
}

var GenPerNodeByCons = GenSet{
	Name:             "NodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType", NameMap: RndCTMap},
		// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
		// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
		// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
	}}},
}

var GenPerNodeByConsBuffFwd = GenSet{
	Name:             "NodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType", NameMap: RndCTMap},
		{VaryField: "BufferForwardType", Title: "BFT"},
		// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
		// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
	}}},
}

var GenPerNodeByConsMvCons4BcastType = GenSet{
	Name:             "NodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType", NameMap: RndCTMap},
		{VaryField: "MvCons4BcastType", Title: "nil", NameMap: MvCons4BcastTypeMap},
		// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
		// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
	}}},
}

var GenPerNodeByConsCB = GenSet{
	Name:             "NodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType", NameMap: RndCTMap},
		{VaryField: "CollectBroadcast", Title: "CB"},
		// {VaryField: "SigType", Title: "ST:"},
		// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
		// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
		// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
	}}},
}

var GenByNodeCount = GenSet{
	Name:             "ByNodeCount",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	SplitItems: []*SplitItem{{FieldName: "BinConsPercentOnes"}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByPercentOnes = GenSet{
	Name:             "ByPercentOnes",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "Percent_1_Proposals"},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByPercentOnesCoinType = GenSet{
	Name:             "ByPercentOnes",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "Percent_1_Proposals"},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			{VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByPercentOnesIncludeProofs = GenSet{
	Name:             "ByPercentOnes",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "Percent_1_Proposals"},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			{VaryField: "IncludeProofs", Title: "IP:", NameMap: TrueFalseMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByPercentOnesCombine = GenSet{
	Name:             "ByPercentOnes",
	BoolFilterFields: []FilterBoolField{PerProcFilter, TotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "Percent_1_Proposals"},
		ExtraFields: []VaryField{
			{VaryField: "ConsType", NameMap: RndCTMap},
			{VaryField: "AllowSupportCoin", Title: "CM:", NameMap: TrueFalseMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByCoinPresetsTypes = GenSet{
	Name:             "ByCoinPresets",
	BoolFilterFields: []FilterBoolField{PerProcFilter, NonByzFilter, NonTotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "UseFixedCoinPresets", Title: "Coin_Presets", NameMap: TrueFalseMap},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			// {VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	SplitItems: []*SplitItem{{FieldName: "BinConsPercentOnes"}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByByzTypes = GenSet{
	Name:             "ByByzTypes",
	BoolFilterFields: []FilterBoolField{PerProcFilter, NonByzFilter, NonTotalFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "ByzType", Title: "Fault_Type", NameMap: ByzMap},
		ExtraFields: []VaryField{
			// {VaryField: "AllowSupportCoin", Title: "_CombineMsgs"},
			{VaryField: "ConsType", NameMap: RndCTMap},
			// {VaryField: "StopOnCommit", Title: "SOC", NameMap: socMap},
			// {VaryField: "NoSignatures", Title: "SM:", NameMap: SigMsgMap},
			{VaryField: "CoinType", Title: "CI:", NameMap: CoinMap},
		}}},
	SplitItems: []*SplitItem{{FieldName: "BinConsPercentOnes"}},
	// SplitItems: []*SplitItem{{FieldName: "ConsType",
	//	NextSplitItem: &SplitItem{FieldName: "NodeCount",
	//		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenPerNodeByCoin = GenSet{
	Name:             "CoinType",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "CoinType"}, ExtraFields: []VaryField{
		{VaryField: "ConsType"}}}},
}

var GenByNodesSupportCoin = GenSet{
	Name: "NodesBySupportCoin",
	// FilterIntFields: []FilterIntField{{Name: "BinConsPercentOnes", Values: []int{0, 33, 66, 100}}},
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "AllowSupportCoin", Title: "_CombineMsgs"}}}},
	SplitItems: []*SplitItem{{FieldName: "ConsType"}},
}

var GenByNodesSupportCoinProofs = GenSet{
	Name: "NodesBySupportCoin",
	// FilterIntFields: []FilterIntField{{Name: "BinConsPercentOnes", Values: []int{0, 33, 66, 100}}},
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "AllowSupportCoin", Title: "_CombineMsgs"}, {VaryField: "IncludeProofs", Title: "_Proofs"}}}},
	SplitItems: []*SplitItem{{FieldName: "ConsType", NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}},
}

var GenByNodesPercentOnes = GenSet{
	Name:             "PercentOnesByNodes",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "%_1"},
		ExtraFields: []VaryField{{VaryField: "NodeCount"}}}},
	SplitItems: []*SplitItem{{FieldName: "ConsType", NextSplitItem: &SplitItem{FieldName: "AllowSupportCoin",
		NextSplitItem: &SplitItem{FieldName: "IncludeProofs"}}}},
}

var GenByPercentOnesInclueProofs = GenSet{
	Name:             "ByPercentOnesIncludeProofs",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "BinConsPercentOnes", Title: "%_1"},
		ExtraFields: []VaryField{{VaryField: "IncludeProofs"}},
	}},
	SplitItems: []*SplitItem{{FieldName: "ConsType",
		NextSplitItem: &SplitItem{FieldName: "NodeCount",
			NextSplitItem: &SplitItem{FieldName: "AllowSupportCoin"}}}},
}

var GenPerToNodeByCons = GenSet{
	Name:             "PerTO",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "NodeCount"}, ExtraFields: []VaryField{
		{VaryField: "ConsType"}}}},
	SplitItems: []*SplitItem{{FieldName: "TestID"}},
}

var GenPerRndMemberTypeByCons = GenSet{
	Name:             "RndMemberType",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "RndMemberType"}, ExtraFields: []VaryField{
		{VaryField: "NodeCount"}, {VaryField: "ConsType"}}}},
}

var GenPerTOByCons = GenSet{
	Name:             "TO",
	BoolFilterFields: []FilterBoolField{PerProcFilter},
	GenItems: []GenItem{{VaryField: VaryField{VaryField: "TestID"}, ExtraFields: []VaryField{
		{VaryField: "NodeCount"}, {VaryField: "ConsType"}}}},
}

var PerProcFilter = FilterBoolField{Name: "PerProc", Values: []bool{true}}
var ByzNodeFilter = FilterBoolField{Name: "ByzNode", Values: []bool{true}}
var TotalFilter = FilterBoolField{Name: "Total", Values: []bool{true}}
var NonTotalFilter = FilterBoolField{Name: "Total", Values: []bool{false}}
var NonByzFilter = FilterBoolField{Name: "ByzNode", Values: []bool{false}}

// GenSetToDisk stores the items to disk in folderPath, using const GenSetsFileName as the file name.
func GenSetToDisk(folderPath string, items []GenSet) error {
	itemsByt, err := json.MarshalIndent(items, "", "\t")
	if err != nil {
		return err
	}
	path := filepath.Join(folderPath, GenSetsFolderName)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(path, GenSetsFileName), itemsByt, os.ModePerm); err != nil {
		return err
	}
	return nil
}

type GenSet struct {
	Name string

	FilterStringFields []FilterStringField
	FilterIntFields    []FilterIntField
	BoolFilterFields   []FilterBoolField

	SplitItems []*SplitItem

	GenItems []GenItem
}

type SplitItem struct {
	FieldName     string
	NextSplitItem *SplitItem
}

type FilterBoolField struct {
	Name   string
	Values []bool
}

type FilterStringField struct {
	Name   string
	Values []string
}

type FilterIntField struct {
	Name   string
	Values []int
}

type GenItem struct {
	VaryField   VaryField
	ExtraFields []VaryField
}

type VaryField struct {
	Title     string
	VaryField string
	NameMap   map[string]string
}

func (vf VaryField) GetValue(value interface{}) interface{} {
	if vf.NameMap != nil {
		if out, ok := vf.NameMap[fmt.Sprintf("%v", value)]; ok {
			return out
		}
	}
	return value
}

func (vf VaryField) GetTitle() string {
	if vf.Title == "nil" {
		return ""
	}
	if vf.Title != "" {
		return vf.Title
	}
	return vf.VaryField
}
