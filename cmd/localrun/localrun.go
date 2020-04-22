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
This package runs a test given a test configuration stored in a json file.
*/
package main

import (
	"flag"
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/consgen"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/rpcsetup"
	"github.com/tcrain/cons/consensus/types"
	"math/rand"
)

func main() {
	var optionsFile string
	var checkOnly bool

	flag.StringVar(&optionsFile, "o", "tofile.json", "Path to file containing list of options")
	flag.BoolVar(&checkOnly, "c", false, "Only check if the test option is valid then exit")
	flag.Parse()

	logging.Info("Loading test options from file: ", optionsFile)
	to, err := types.GetTestOptions(optionsFile)
	if err != nil {
		logging.Error(err)
		panic(err)
	}
	if checkOnly {
		return
	}

	logging.Infof("Loaded options: %s\n", to)

	if to.TestID == 0 {
		to.TestID = rand.Uint64()
		logging.Printf("Set test ID to %v\n", to.TestID)
	}

	cfg := consgen.GetConsConfig(to)
	if to, err = to.CheckValid(to.ConsType, cfg.GetIsMV()); err != nil {
		logging.Error("invalid config ", err)
		panic(err)
	}

	cons.RunConsType(consgen.GetConsItem(to), cfg.GetBroadcastFunc(to.ByzType), cfg, to, rpcsetup.TPanic{})
}
