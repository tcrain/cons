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

package bincons1

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/types"
)

type Config struct {
	cons.StandardBinConfig
}

// GetIncludeProofTypes returns the values for if the consensus supports including proofs or not or both.
func (Config) GetIncludeProofsTypes(gt cons.GetOptionType) []bool {
	switch gt {
	case cons.AllOptions:
		return types.WithBothBool
	case cons.MinOptions:
		return types.WithTrue
	default:
		panic(gt)
	}
}
