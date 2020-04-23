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
This package contains a command that loads a json TestOptionsCons file and outs the cons ids.
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	var optionsFile string

	flag.StringVar(&optionsFile, "o", "tofile.json", "Path to file containing list of options")
	flag.Parse()

	optionsFile = filepath.Join(optionsFile)
	logging.Print("Loading test options from file: ", optionsFile)

	var raw []byte
	var err error
	raw, err = ioutil.ReadFile(filepath.Join(optionsFile))
	if err != nil {
		panic(err)
	}
	var to types.TestOptionsCons
	err = json.Unmarshal(raw, &to)
	if err != nil {
		panic(err)
	}

	var items []string
	for _, nxt := range to.ConsTypes {
		items = append(items, fmt.Sprintf("%d ", nxt))
	}
	fmt.Println(strings.Join(items, " "))

	return
}
