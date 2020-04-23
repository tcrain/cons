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

package csnet

import (
	"github.com/tcrain/cons/config"
	"sync"
)

type udpMsgPool struct {
	sendPacketPool sync.Pool
	newCount       int32
}

func newUdpMsgPool() *udpMsgPool {
	ret := &udpMsgPool{}
	ret.sendPacketPool.New = func() interface{} {
		return make([]byte, config.MaxTotalPacketSize)
	}
	return ret
}

func (nrc *udpMsgPool) getPacket() []byte {
	pkt := nrc.sendPacketPool.Get().([]byte)
	if len(pkt) != config.MaxTotalPacketSize {
		panic("bad packet")
	}
	return pkt
}

func (nrc *udpMsgPool) donePacket(packet []byte) {
	nrc.sendPacketPool.Put(packet[:config.MaxTotalPacketSize])
}
