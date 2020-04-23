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

// DeserSortVRF must only contain sig.MultiSignedMessages with a single signature.
// They will be sorted by the signature's VRFID.
// This is used by cons who need to decide which leader message to echo when their timeout has run out
// when random member selection is enabled.
type DeserSortVRF []*channelinterface.DeserializedItem

func (a DeserSortVRF) Len() int      { return len(a) }
func (a DeserSortVRF) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a DeserSortVRF) Less(i, j int) bool {
	return a[i].Header.(*sig.MultipleSignedMessage).SigItems[0].VRFID < a[j].Header.(*sig.MultipleSignedMessage).SigItems[0].VRFID
}
