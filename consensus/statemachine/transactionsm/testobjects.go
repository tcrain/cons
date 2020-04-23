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

package transactionsm

import (
	"github.com/tcrain/cons/consensus/utils"
	"io"
)

type TestTx struct {
	Id int
}

func (tx *TestTx) GetConflictObjects() []ConflictObject {
	return []ConflictObject{tx.Id}
}

func (tx *TestTx) Encode(writer io.Writer) (n int, err error) {
	return utils.EncodeUint64(uint64(tx.Id), writer)
}
func (tx *TestTx) Decode(reader io.Reader) (n int, err error) {
	var txid uint64
	txid, n, err = utils.ReadUint64(reader)
	tx.Id = int(txid)
	return
}
