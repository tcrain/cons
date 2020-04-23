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

package consgen

import (
	"github.com/tcrain/cons/consensus/cons"
	"github.com/tcrain/cons/consensus/cons/bincons1"
	"github.com/tcrain/cons/consensus/cons/binconsrnd1"
	"github.com/tcrain/cons/consensus/cons/binconsrnd2"
	"github.com/tcrain/cons/consensus/cons/binconsrnd3"
	"github.com/tcrain/cons/consensus/cons/binconsrnd4"
	"github.com/tcrain/cons/consensus/cons/binconsrnd5"
	binconsrnd5old "github.com/tcrain/cons/consensus/cons/binconsrnd5old"
	"github.com/tcrain/cons/consensus/cons/binconsrnd6"
	"github.com/tcrain/cons/consensus/cons/mvcons1"
	"github.com/tcrain/cons/consensus/cons/mvcons2"
	"github.com/tcrain/cons/consensus/cons/mvcons3"
	"github.com/tcrain/cons/consensus/cons/rbbcast1"
	"github.com/tcrain/cons/consensus/cons/rbbcast2"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
)

func GetConsItem(to types.TestOptions) (consItem consinterface.ConsItem) {
	switch to.ConsType {
	case types.BinCons1Type:
		consItem = &bincons1.BinCons1{}
	case types.BinConsRnd1Type:
		consItem = &binconsrnd1.BinConsRnd1{}
	case types.BinConsRnd2Type:
		consItem = &binconsrnd2.BinConsRnd2{}
	case types.BinConsRnd3Type:
		consItem = &binconsrnd3.BinConsRnd3{}
	case types.BinConsRnd4Type:
		consItem = &binconsrnd4.BinConsRnd4{}
	case types.BinConsRnd5Type:
		consItem = &binconsrnd5.BinConsRnd5{}
	case types.BinConsRnd5OldType:
		consItem = &binconsrnd5old.BinConsRnd5{}
	case types.BinConsRnd6Type:
		consItem = &binconsrnd6.BinConsRnd6{}
	case types.MvBinCons1Type:
		consItem = &mvcons1.MvCons1{}
	case types.MvBinConsRnd1Type:
		consItem = &mvcons1.MvCons1{}
	case types.MvCons2Type:
		consItem = &mvcons2.MvCons2{}
	case types.MvCons3Type:
		consItem = &mvcons3.MvCons3{}
	case types.RbBcast1Type:
		consItem = &rbbcast1.RbBcast1{}
	case types.RbBcast2Type:
		consItem = &rbbcast2.RbBcast2{}
	default:
		logging.Error("invalid cons type", to.ConsType)
		panic(to.ConsType)
	}
	return
}

func GetConsConfig(to types.TestOptions) (consConfig cons.ConfigOptions) {
	switch to.ConsType {
	case types.BinCons1Type:
		consConfig = bincons1.Config{}
	case types.BinConsRnd1Type:
		consConfig = binconsrnd1.Config{}
	case types.BinConsRnd2Type:
		consConfig = binconsrnd2.Config{}
	case types.BinConsRnd3Type:
		consConfig = binconsrnd3.Config{}
	case types.BinConsRnd4Type:
		consConfig = binconsrnd4.Config{}
	case types.BinConsRnd5Type:
		consConfig = binconsrnd5.Config{}
	case types.BinConsRnd5OldType:
		consConfig = binconsrnd5old.Config{}
	case types.BinConsRnd6Type:
		consConfig = binconsrnd6.Config{}
	case types.MvBinCons1Type:
		consConfig = mvcons1.MvBinCons1Config{}
	case types.MvBinConsRnd1Type:
		consConfig = mvcons1.MvBinConsRnd1Config{}
	case types.MvCons2Type:
		consConfig = mvcons2.MvCons2Config{}
	case types.MvCons3Type:
		consConfig = mvcons3.MvCons3Config{}
	case types.RbBcast1Type:
		consConfig = rbbcast1.RbBcast1Config{}
	case types.RbBcast2Type:
		consConfig = rbbcast2.RbBcast2Config{}
	default:
		logging.Error("invalid cons type", to.ConsType)
		panic(to.ConsType)
	}
	return
}
