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
This package generates text files that can be used to generate graphs from benchmark results.
*/
package main

import (
	"flag"
	"github.com/tcrain/cons/parse"
	"path/filepath"
)

func main() {
	var folderPath string

	flag.StringVar(&folderPath, "o", "", "Path to folder containing benchmark results")
	flag.Parse()

	if err := parse.GenResults(filepath.Join(folderPath)); err != nil {
		panic(err)
	}

}
