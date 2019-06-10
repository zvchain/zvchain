//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

func (pool *txPool) saveReceipt(txHash common.Hash, dataBytes []byte) error {
	return pool.receiptDb.AddKv(pool.batch, txHash.Bytes(), dataBytes)
}

func (pool *txPool) saveReceipts(bhash common.Hash, receipts types.Receipts) error {
	if nil == receipts || 0 == len(receipts) {
		return nil
	}
	for _, receipt := range receipts {
		executedTxBytes, err := msgpack.Marshal(receipt)
		if nil != err {
			return err
		}
		if err := pool.saveReceipt(receipt.TxHash, executedTxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (pool *txPool) deleteReceipts(txs []common.Hash) error {
	if nil == txs || 0 == len(txs) {
		return nil
	}
	var err error
	for _, tx := range txs {
		err = pool.saveReceipt(tx, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTransactionStatus returns the execute result status by hash
func (pool *txPool) GetTransactionStatus(hash common.Hash) (uint, error) {
	executedTx := pool.loadReceipt(hash)
	if executedTx == nil {
		return 0, ErrNil
	}
	return executedTx.Status, nil
}

func (pool *txPool) loadReceipt(hash common.Hash) *types.Receipt {
	txBytes, _ := pool.receiptDb.Get(hash.Bytes())
	if txBytes == nil {
		return nil
	}

	var rs types.Receipt
	err := msgpack.Unmarshal(txBytes, &rs)
	if err != nil {
		return nil
	}
	return &rs
}

func (pool *txPool) hasReceipt(hash common.Hash) bool {
	ok, _ := pool.receiptDb.Has(hash.Bytes())
	return ok
}

// GetReceipt returns the transaction's recipe by hash
func (pool *txPool) GetReceipt(hash common.Hash) *types.Receipt {
	rs := pool.loadReceipt(hash)
	if rs == nil {
		return nil
	}
	return rs
}
