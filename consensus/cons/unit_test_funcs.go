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

package cons

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/channelinterface"
)

// ConsTestItems are used as input to certain unit tests.
type ConsTestItems struct {
	PrivKeys []sig.Priv // the private keys used in the test
	PubKeys  []sig.Pub  // the public keys used in the test
}

// MergeSigsForTest is used by certain unit tests to collect a set of singatures in a single message.
func MergeSigsForTest(items []*channelinterface.DeserializedItem) {
	var newItems []*sig.SigItem
	for _, deser := range items {
		newItems = append(newItems, deser.Header.(*sig.MultipleSignedMessage).GetSigItems()...)
	}
	items[0].Header.(*sig.MultipleSignedMessage).SetSigItems(newItems)
}
