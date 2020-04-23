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
	"sync"

	"github.com/tcrain/cons/consensus/channelinterface"
)

// UDPConnMap is a simple map of addresses to connection objects protected by a RW mutex
type UDPConnMap struct {
	connMap map[channelinterface.NetConInfo]*NetConnectionUDP
	mutex   sync.RWMutex
}

// NewUDPConnMap returns a new empty UDPConnMap
func NewUDPConnMap() *UDPConnMap {
	return &UDPConnMap{connMap: make(map[channelinterface.NetConInfo]*NetConnectionUDP)}
}

// Get returns the value in the map at addr, this is protected by a read mutex.
func (ucm *UDPConnMap) Get(addr channelinterface.NetConInfo) (ret *NetConnectionUDP) {
	ucm.mutex.RLock()
	ret = ucm.connMap[addr]
	ucm.mutex.RUnlock()
	return ret
}

// AddConnection adds the addresses to the mape with value connection. It is protected by a mutex.
func (ucm *UDPConnMap) AddConnection(addrs []channelinterface.NetConInfo, conn *NetConnectionUDP) {
	ucm.mutex.Lock()
	for _, addr := range addrs {
		ucm.connMap[addr] = conn
	}
	ucm.mutex.Unlock()
}

// RemoveConnection removes the addresses from the map. It is protected by a mutex.
func (ucm *UDPConnMap) RemoveConnection(addrs []channelinterface.NetConInfo) {
	ucm.mutex.Lock()
	for _, addr := range addrs {
		delete(ucm.connMap, addr)
	}
	ucm.mutex.Unlock()
}
