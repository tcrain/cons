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

type EncodeInterface interface {
	Encode(writer io.Writer) (n int, err error)
	Decode(reader io.Reader) (n int, err error)
}

// if two conflictobjects are equal then transactions conflict, otherwise they dont
type ConflictObject interface{}

type TransactionInterface interface {
	GetConflictObjects() []ConflictObject
	EncodeInterface
}

type TxAlloc func() TransactionInterface

type TransactionList struct {
	Items   []TransactionInterface
	TxAlloc TxAlloc
}

func (tx *TransactionList) Encode(writer io.Writer) (n int, err error) {
	n, err = utils.EncodeUint64(uint64(len(tx.Items)), writer)
	if err != nil {
		return
	}
	for _, nxt := range tx.Items {
		var nxtSize int
		nxtSize, err = nxt.Encode(writer)
		n += nxtSize
		if err != nil {
			return
		}
	}
	return
}

func (tx *TransactionList) Decode(reader io.Reader) (n int, err error) {
	var v uint64
	v, n, err = utils.ReadUint64(reader)
	for i := 0; i < int(v); i++ {
		nxt := tx.TxAlloc()
		var nxtSize int
		nxtSize, err = nxt.Decode(reader)
		n += nxtSize
		if err != nil {
			return
		}
		tx.Items = append(tx.Items, nxt)
	}
	return
}
