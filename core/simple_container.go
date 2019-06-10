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
	"container/heap"
	"sort"
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type simpleContainer struct {
	limit  int
	txs    types.PriorityTransactions
	txsMap map[common.Hash]*types.Transaction

	lock sync.RWMutex
}

func newSimpleContainer(l int) *simpleContainer {
	c := &simpleContainer{
		lock:   sync.RWMutex{},
		limit:  l,
		txsMap: map[common.Hash]*types.Transaction{},
		txs:    types.PriorityTransactions{},
	}
	heap.Init(&c.txs)
	return c
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txs.Len()
}

func (c *simpleContainer) sort() {
	c.lock.Lock()
	defer c.lock.Unlock()

	sort.Sort(c.txs)
}

func (c *simpleContainer) contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key] != nil
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.txsMap[key]
}

func (c *simpleContainer) asSlice(limit int) []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	size := limit
	if c.txs.Len() < size {
		size = c.txs.Len()
	}
	txs := make([]*types.Transaction, size)
	copy(txs, c.txs[:size])
	return txs
}

func (c *simpleContainer) push(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.txsMap[tx.Hash] != nil {
		return
	}

	if c.txs.Len() >= c.limit {
		for i, oldTx := range c.txs {
			if tx.GasPrice >= oldTx.GasPrice {
				delete(c.txsMap, oldTx.Hash)
				c.txs[i] = tx
				c.txsMap[tx.Hash] = tx
				heap.Fix(&c.txs, i)
				break
			}
		}
	} else {
		heap.Push(&c.txs, tx)
		c.txsMap[tx.Hash] = tx
	}
}

func (c *simpleContainer) remove(key common.Hash) {
	if !c.contains(key) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.txsMap[key] == nil {
		return
	}
	delete(c.txsMap, key)
	for i, tx := range c.txs {
		if tx.Hash == key {
			heap.Remove(&c.txs, i)
			break
		}
	}
}
