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
package currency

import (
	"bytes"
	"fmt"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/merkle"
	"github.com/tcrain/cons/consensus/types"
	"github.com/tcrain/cons/consensus/utils"
	"io"
	"math"
)

var maxBlockTx = 1000
var ErrInvalidTxMerkleRoot = fmt.Errorf("invalid tx Merkle root")
var ErrTooManyTxInBlock = fmt.Errorf("too many transactions in block")

// var ErrTooFewTxInBlock = fmt.Errorf("must be at least one tx in block")

type TxBlock struct {
	Index        types.ConsensusInt
	TxMerkleRoot merkle.MerkleRoot
	Transactions []*TransferTx
	PrevHash     types.HashBytes

	NewPubFunc func() sig.Pub
	NewSigFunc func() sig.Sig
}

func (txl *TxBlock) Decode(reader io.Reader) (n int, err error) {
	var nxtN int
	var idx uint64
	idx, nxtN, err = utils.ReadUint64(reader)
	n += nxtN
	if err != nil {
		return
	}
	txl.Index = types.ConsensusInt(idx)

	txl.PrevHash = make(types.HashBytes, types.GetHashLen())
	nxtN, err = reader.Read(txl.PrevHash)
	n += nxtN
	if err != nil {
		return
	}

	var numTx uint16
	numTx, nxtN, err = utils.ReadUint16(reader)
	n += nxtN
	if err != nil {
		return
	}
	if int(numTx) > maxBlockTx {
		err = ErrTooManyTxInBlock
		return
	}
	txl.Transactions = make([]*TransferTx, int(numTx))
	getHashList := make([]merkle.GetHash, int(numTx))

	for i := 0; i < int(numTx); i++ {
		txl.Transactions[i] = &TransferTx{NewSigFunc: txl.NewSigFunc, NewPubFunc: txl.NewPubFunc}
		nxtN, err = txl.Transactions[i].Decode(reader)
		n += nxtN
		if err != nil {
			return
		}
		getHashList[i] = txl.Transactions[i]
	}

	if len(txl.Transactions) > 0 {
		txl.TxMerkleRoot = make(merkle.MerkleRoot, types.GetHashLen())
		nxtN, err = reader.Read(txl.TxMerkleRoot)
		n += nxtN
		if err != nil {
			return
		}
		if !bytes.Equal(merkle.ComputeMerkleRoot(
			getHashList, 2, types.GetNewHash), txl.TxMerkleRoot) {
			err = ErrInvalidTxMerkleRoot
			return
		}
	}
	return
}

func (txl *TxBlock) Encode(writer io.Writer) (n int, err error) {
	var nxtN int
	nxtN, err = utils.EncodeUint64(uint64(txl.Index), writer)
	n += nxtN
	if err != nil {
		return
	}
	if len(txl.PrevHash) != types.GetHashLen() {
		err = types.ErrInvalidHash
		return
	}
	nxtN, err = writer.Write(txl.PrevHash)
	n += nxtN
	if err != nil {
		return
	}
	hashItems := make([]merkle.GetHash, len(txl.Transactions))

	if len(txl.Transactions) > math.MaxUint16 || len(txl.Transactions) > maxBlockTx {
		err = ErrTooManyTxInBlock
		return
	}
	nxtN, err = utils.EncodeUint16(uint16(len(txl.Transactions)), writer)
	n += nxtN
	if err != nil {
		return
	}

	for i, nxt := range txl.Transactions {
		hashItems[i] = nxt
		nxtN, err = nxt.Encode(writer)
		n += nxtN
		if err != nil {
			return
		}
	}
	txl.TxMerkleRoot = merkle.ComputeMerkleRoot(hashItems, 2, types.GetNewHash)
	nxtN, err = writer.Write(txl.TxMerkleRoot)
	n += nxtN
	if err != nil {
		return
	}
	return
}
