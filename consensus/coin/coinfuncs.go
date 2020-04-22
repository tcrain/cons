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
package coin

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/coin/knowncoin"
	"github.com/tcrain/cons/consensus/coin/localcoin"
	"github.com/tcrain/cons/consensus/coin/strongcoin1"
	"github.com/tcrain/cons/consensus/coin/strongcoin1echo"
	"github.com/tcrain/cons/consensus/coin/strongcoin2"
	"github.com/tcrain/cons/consensus/coin/strongcoin2echo"
	"github.com/tcrain/cons/consensus/consinterface"
	"github.com/tcrain/cons/consensus/generalconfig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

func GetFixedCoinPresets(useFixedCoinPresets bool, isMv bool) (maxPresetRound types.ConsensusRound,
	presets []struct {
		Round types.ConsensusRound
		Val   types.BinVal
	}) {

	presets = []struct {
		Round types.ConsensusRound
		Val   types.BinVal
	}{{0, 1}}
	if useFixedCoinPresets {
		presets = append(presets, []struct {
			Round types.ConsensusRound
			Val   types.BinVal
		}{{1, 1}, {2, 0}}...)
	}
	if isMv {
		presets = append(presets, struct {
			Round types.ConsensusRound
			Val   types.BinVal
		}{1, 1})
	}
	for _, nxt := range presets {
		if nxt.Round > maxPresetRound {
			maxPresetRound = nxt.Round
		}
	}
	return
}

func GenerateCoinMessageStateInterface(coinType types.CoinType, isMV bool, pid int64,
	gc *generalconfig.GeneralConfig) consinterface.CoinMessageStateInterface {

	switch coinType {
	case types.KnownCoinType:
		return knowncoin.NewKnownCoinMsgState(isMV, gc)
	case types.StrongCoin1Type:
		return strongcoin1.NewStrongCoin1MsgState(isMV, gc)
	case types.StrongCoin1EchoType:
		return strongcoin1echo.NewStrongCoin1EchoMsgState(isMV, gc)
	case types.StrongCoin2Type:
		return strongcoin2.NewStrongCoin2MsgState(isMV, gc)
	case types.StrongCoin2EchoType:
		return strongcoin2echo.NewStrongCoin2EchoMsgState(isMV, gc)
	case types.LocalCoinType:
		return localcoin.NewLocalCoinMsgState(isMV, config.LocalCoinSeed+pid, gc)
	default:
		panic(coinType)
	}
}

func GenerateCoinIterfaceItem(coinType types.CoinType) consinterface.CoinItemInterface {
	switch coinType {
	case types.KnownCoinType:
		return knowncoin.NewKnownCoin()
	case types.StrongCoin1Type:
		return strongcoin1.NewStrongCoin1()
	case types.StrongCoin1EchoType:
		return strongcoin1echo.NewStrongCoin1Echo()
	case types.StrongCoin2Type:
		return strongcoin2.NewStrongCoin2()
	case types.StrongCoin2EchoType:
		return strongcoin2echo.NewStrongCoin2Echo()
	case types.LocalCoinType:
		return localcoin.NewLocalCoin()
	default:
		panic(coinType)
	}
}

func GetCoinHeader(emptyPub sig.Pub, gc *generalconfig.GeneralConfig, headerType messages.HeaderID) (messages.MsgHeader, error) {
	switch gc.CoinType {
	case types.KnownCoinType:
		return knowncoin.KnownCoin{}.GetHeader(emptyPub, gc, headerType)
	case types.StrongCoin1Type:
		return strongcoin1.StrongCoin1{}.GetHeader(emptyPub, gc, headerType)
	case types.StrongCoin1EchoType:
		return strongcoin1echo.StrongCoin1Echo{}.GetHeader(emptyPub, gc, headerType)
	case types.StrongCoin2Type:
		return strongcoin2.StrongCoin2{}.GetHeader(emptyPub, gc, headerType)
	case types.StrongCoin2EchoType:
		return strongcoin2echo.StrongCoin2Echo{}.GetHeader(emptyPub, gc, headerType)
	case types.LocalCoinType:
		return localcoin.LocalCoin{}.GetHeader(emptyPub, gc, headerType)
	}
	return nil, types.ErrInvalidHeader
}

func GetBufferCount(hdr messages.MsgIDHeader, gc *generalconfig.GeneralConfig,
	memberChecker *consinterface.MemCheckers) (int, int, messages.MsgID, error) {

	switch gc.CoinType {
	case types.KnownCoinType:
		return knowncoin.KnownCoin{}.GetBufferCount(hdr, gc, memberChecker)
	case types.StrongCoin1Type:
		return strongcoin1.StrongCoin1{}.GetBufferCount(hdr, gc, memberChecker)
	case types.StrongCoin1EchoType:
		return strongcoin1echo.StrongCoin1Echo{}.GetBufferCount(hdr, gc, memberChecker)
	case types.StrongCoin2Type:
		return strongcoin2.StrongCoin2{}.GetBufferCount(hdr, gc, memberChecker)
	case types.StrongCoin2EchoType:
		return strongcoin2echo.StrongCoin2Echo{}.GetBufferCount(hdr, gc, memberChecker)
	case types.LocalCoinType:
		return localcoin.LocalCoin{}.GetBufferCount(hdr, gc, memberChecker)
	}
	return 0, 0, nil, types.ErrInvalidHeader
}
