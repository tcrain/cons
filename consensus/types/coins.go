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

package types

import "fmt"

type AllowsCoin int

const (
	NoCoin AllowsCoin = iota
	StrongCoin
	WeakCoin
)

func (ac AllowsCoin) String() string {
	switch ac {
	case NoCoin:
		return "NoCoin"
	case StrongCoin:
		return "StrongCoin"
	case WeakCoin:
		return "WeakCoin"
	default:
		return fmt.Sprintf("AllowsCoin%d", ac)
	}
}

type CoinType int

const (
	NoCoinType          CoinType = iota
	KnownCoinType                // predefined coins
	LocalCoinType                // each process chooses a random local coin
	StrongCoin1Type              // a strong coin implemented by an n-t (or t+1) threshold signature
	StrongCoin1EchoType          // StrongCoin1 with an extra message step (allows use of t+1 coin)
	StrongCoin2Type              // a strong coin implemented by an n-t (or t+1) threshold random coin (cachin'05)
	StrongCoin2EchoType          // StrongCoin2 with an extra message step (allows use of t+1 coin)
)

func CheckStrongCoin(coinType CoinType) bool {
	for _, nxt := range StrongCoins {
		if coinType == nxt {
			return true
		}
	}
	return false
}

var StrongCoins = []CoinType{StrongCoin1Type, StrongCoin1EchoType, StrongCoin2Type, StrongCoin2EchoType, KnownCoinType}
var WeakCoins = []CoinType{LocalCoinType}
var AllCoins = append(StrongCoins, WeakCoins...) // append([]CoinType{NoCoinType}, append(StrongCoins, WeakCoins...)...)

func (ct CoinType) String() string {
	switch ct {
	case NoCoinType:
		return "NoCoinType"
	case StrongCoin1Type:
		return "StrongCoin1"
	case StrongCoin1EchoType:
		return "StrongCoin1Echo"
	case StrongCoin2Type:
		return "StrongCoin2"
	case LocalCoinType:
		return "LocalCoin"
	case StrongCoin2EchoType:
		return "StrongCoin2Echo"
	case KnownCoinType:
		return "KnownCoin"
	default:
		return fmt.Sprintf("CoinType%d", ct)
	}
}
