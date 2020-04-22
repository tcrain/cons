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
package utils

import (
	// "fmt"
	"net"

	"io/ioutil"
	"strings"
)

// ReadTCPIPFile reads a file with one ip address per line, checks if the
// IPs are valid, and returns them as a slice of strings.
// If includePort is true, then the port is not removed when the address is read.
func ReadTCPIPFile(filePath string, includePort bool) ([]string, error) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	preips := strings.Split(string(raw), "\n")
	var res []string

	for _, ip := range preips {
		if ip == "" {
			// return nil, fmt.Errorf("Invalid address: %v", ip)
			continue
		}
		_, err := net.ResolveTCPAddr("tcp", ip)
		if err != nil {
			return nil, err
		}
		if includePort {
			res = append(res, ip)
		} else {
			res = append(res, strings.Split(ip, ":")[0])
		}
	}
	return res, nil
}
