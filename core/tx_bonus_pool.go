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
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type bonusPool struct {
	bm             *BonusManager
	pool           *lru.Cache // Is an LRU cache that stores the mapping of transaction hashes to transaction pointer
	blockHashIndex *lru.Cache // Is an LRU cache that stores the mapping of block hashes to slice of transaction pointer
}

func newBonusPool(pm *BonusManager, size int) *bonusPool {
	return &bonusPool{
		pool:           common.MustNewLRUCache(size),
		blockHashIndex: common.MustNewLRUCache(size),
		bm:             pm,
	}
}

func (bp *bonusPool) add(tx *types.Transaction) bool {
	if bp.pool.Contains(tx.Hash) {
		return false
	}
	bp.pool.Add(tx.Hash, tx)
	blockHash := bp.bm.parseBonusBlockHash(tx)

	var txs []*types.Transaction
	if v, ok := bp.blockHashIndex.Get(blockHash); ok {
		txs = v.([]*types.Transaction)
	} else {
		txs = make([]*types.Transaction, 0)
	}
	txs = append(txs, tx)
	bp.blockHashIndex.Add(blockHash, txs)
	return true
}

func (bp *bonusPool) remove(txHash common.Hash) {
	tx, _ := bp.pool.Get(txHash)
	if tx != nil {
		bp.pool.Remove(txHash)
		bhash := bp.bm.parseBonusBlockHash(tx.(*types.Transaction))
		bp.removeByBlockHash(bhash)
	}
}

func (bp *bonusPool) removeByBlockHash(blockHash common.Hash) int {
	txs, _ := bp.blockHashIndex.Get(blockHash)
	cnt := 0
	if txs != nil {
		for _, trans := range txs.([]*types.Transaction) {
			bp.pool.Remove(trans.Hash)
			cnt++
		}
		bp.blockHashIndex.Remove(blockHash)
	}
	return cnt
}

func (bp *bonusPool) get(hash common.Hash) *types.Transaction {
	if v, ok := bp.pool.Get(hash); ok {
		return v.(*types.Transaction)
	}
	return nil
}

func (bp *bonusPool) len() int {
	return bp.pool.Len()
}

func (bp *bonusPool) contains(hash common.Hash) bool {
	return bp.pool.Contains(hash)
}

func (bp *bonusPool) hasBonus(blockHashByte []byte) bool {
	return bp.bm.blockHasBonusTransaction(blockHashByte)
}

func (bp *bonusPool) forEach(f func(tx *types.Transaction) bool) {
	for _, k := range bp.pool.Keys() {
		v, _ := bp.pool.Peek(k)
		if v != nil {
			if !f(v.(*types.Transaction)) {
				break
			}
		}
	}
}
