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

package channelinterface

import (
	"github.com/tcrain/cons/consensus/types"
	"net"
	"strings"
)

// NewConInfo stores network addresses
type NetConInfo struct {
	Addr string // The address, ip or ipv6
	Nw   string // The netowrk type, tcp of udp
}

// Network returns nci.Nw
func (nci NetConInfo) Network() string {
	return nci.Nw
}

// String returns Nci.Addr
func (nci NetConInfo) String() string {
	return nci.Addr
}

// UpdateNw takes creates a new NetConInfo object using
// an existing NetConInfo object and an address, it takes
// the ip from nci, and the rest of the information from addr
func UpdateNw(nci NetConInfo, addr net.Addr) NetConInfo {
	s := addr.String()
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		panic("bad address")
	}
	newPart := s[idx:]

	idx = strings.LastIndex(nci.Addr, ":")
	if idx < 0 {
		panic("bad address")
	}

	return NetConInfo{
		Addr: nci.Addr[:idx] + newPart,
		Nw:   nci.Nw,
	}
}

// NetConInfoFromAddr creates a new NewConInfo object from
// a net.Addr object
func NetConInfoFromAddr(addr net.Addr) NetConInfo {
	return NetConInfo{
		Addr: addr.String(),
		Nw:   addr.Network()}
}

// NetConInfoFromString creates a new NetConInfo object from a string
// for example tcp://127.0.0.1:1234
func NetConInfoFromString(str string) (NetConInfo, error) {
	idx := strings.Index(str, "://")
	if idx < 0 {
		return NetConInfo{}, types.ErrInvalidFormat
	}
	return NetConInfo{
		Addr: str[:idx],
		Nw:   str[idx:],
	}, nil
}
