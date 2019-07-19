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
	"sync"

	datacommon "github.com/Workiva/go-datastructures/common"
	"github.com/Workiva/go-datastructures/slice/skip"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type simpleContainer struct {
	txsMap     map[common.Hash]*types.Transaction
	chain      *FullBlockChain
	pending    *pendingContainer
	queue      map[common.Hash]*types.Transaction
	queueLimit int

	lock sync.RWMutex
}

type orderByNonceTx struct {
	item *types.Transaction
}

//Transactions with same nonce will be treat as equal, because only transactions same source will be insert to same list
func (tx *orderByNonceTx) Compare(e datacommon.Comparator) int {
	tx2 := e.(*orderByNonceTx)

	if tx.item.Hash == tx2.item.Hash {
		return 0
	}

	if tx.item.Nonce > tx2.item.Nonce {
		return 1
	}
	if tx.item.Nonce < tx2.item.Nonce {
		return -1
	}
	return 0
}

type priceHeap []*types.Transaction

func (h priceHeap) Len() int           { return len(h) }
func (h priceHeap) Less(i, j int) bool { return h[i].GasPrice.Cmp(h[j].GasPrice.Value()) > 0 }
func (h priceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *priceHeap) Push(x interface{}) {
	*h = append(*h, x.(*types.Transaction))
}

func (h *priceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type pendingContainer struct {
	limit int
	size  int

	waitingMap map[common.Address]*skip.SkipList //*orderByNonceTx. Map of transactions group by source for waiting
}

// push the transaction into the pending list. tx which returns false will push to the queue
func (s *pendingContainer) push(tx *types.Transaction, stateNonce uint64) bool {
	var doInsertOrReplace = func() {
		newTxNode := newOrderByNonceTx(tx)
		existSource := s.waitingMap[*tx.Source].Get(newTxNode)[0]

		if existSource != nil {
			if existSource.(*orderByNonceTx).item.GasPrice.Cmp(tx.GasPrice.Value()) < 0 {
				//replace the existing one
				s.waitingMap[*tx.Source].Delete(existSource)
				s.waitingMap[*tx.Source].Insert(newTxNode)
			}
		} else {
			s.size++
			s.waitingMap[*tx.Source].Insert(newTxNode)
		}
	}

	if tx.Nonce == stateNonce+1 {
		if s.waitingMap[*tx.Source] == nil {
			s.waitingMap[*tx.Source] = skip.New(uint16(16))
		}

		doInsertOrReplace()
	} else {
		if s.waitingMap[*tx.Source] == nil {
			return false
		}
		bigNonceTx := skipGetLast(s.waitingMap[*tx.Source])
		if bigNonceTx != nil {
			bigNonce := bigNonceTx.(*orderByNonceTx).item.Nonce
			if tx.Nonce > bigNonce+1 {
				return false
			}

			doInsertOrReplace()
		}
	}

	//remove lowest price transaction if pending is full
	var lowPriceTx *types.Transaction
	if s.size >= s.limit {
		for _, sourcedMap := range s.waitingMap {
			lastTx := skipGetLast(sourcedMap).(*orderByNonceTx).item
			if lowPriceTx == nil {
				lowPriceTx = lastTx
			}
			if lowPriceTx.GasPrice.Cmp(lastTx.GasPrice.Value()) > 0 {
				lowPriceTx = lastTx
			}
		}
		if lowPriceTx != nil {
			s.remove(lowPriceTx)
		}
	}

	return true
}

func (s *pendingContainer) peek(f func(tx *types.Transaction) bool) {
	if s.size == 0 {
		return
	}
	packingList := new(priceHeap)
	heap.Init(packingList)

	noncePositionMap := make(map[common.Address]uint64)
	for _, list := range s.waitingMap {
		heap.Push(packingList, list.ByPosition(0).(*orderByNonceTx).item)
	}
	for {
		if packingList.Len() == 0 {
			return
		}
		tx := heap.Pop(packingList).(*types.Transaction)
		if !f(tx) {
			return
		}
		next := noncePositionMap[*tx.Source] + 1
		if s.waitingMap[*tx.Source] != nil && s.waitingMap[*tx.Source].Len() > next {
			nextTx := s.waitingMap[*tx.Source].ByPosition(next).(*orderByNonceTx)
			noncePositionMap[*tx.Source] = next
			heap.Push(packingList, nextTx.item)
		}
	}
}

func (s *pendingContainer) asSlice(limit int) []*types.Transaction {
	slice := make([]*types.Transaction, 0, s.size)
	count := 0
	for _, txSkip := range s.waitingMap {
		for iter1 := txSkip.IterAtPosition(0); iter1.Next(); {
			slice = append(slice, iter1.Value().(*orderByNonceTx).item)
			count++
			if count >= limit {
				break
			}
		}
	}
	return slice
}

// will break when f(tx) returns false
func (s *pendingContainer) eachForSync(f func(tx *types.Transaction) bool) {
	for _, txSkip := range s.waitingMap {
		for iter1 := txSkip.IterAtPosition(0); iter1.Next(); {
			if !f(iter1.Value().(*orderByNonceTx).item) {
				return
			}
		}
	}
}

func (s *pendingContainer) remove(tx *types.Transaction) {
	if s.waitingMap[*tx.Source] != nil {
		deleted := s.waitingMap[*tx.Source].Delete(newOrderByNonceTx(tx))
		s.size = s.size - len(deleted)
		if s.waitingMap[*tx.Source].Len() == 0 {
			delete(s.waitingMap, *tx.Source)
		}
	}
}

func newOrderByNonceTx(tx *types.Transaction) *orderByNonceTx {
	s := &orderByNonceTx{
		item: tx,
	}
	return s
}

func newPendingContainer(limit int) *pendingContainer {
	s := &pendingContainer{
		limit:      limit,
		size:       0,
		waitingMap: make(map[common.Address]*skip.SkipList),
	}
	return s
}

func newSimpleContainer(pendingLimit int, queueLimit int, chain types.BlockChain) *simpleContainer {
	c := &simpleContainer{
		lock:       sync.RWMutex{},
		chain:      chain.(*FullBlockChain),
		txsMap:     make(map[common.Hash]*types.Transaction),
		pending:    newPendingContainer(pendingLimit),
		queue:      make(map[common.Hash]*types.Transaction),
		queueLimit: queueLimit,
	}
	return c
}

func (c *simpleContainer) Len() int {
	return c.pending.size + len(c.queue)
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

// asSlice only working for rpc now, does not need the lock
func (c *simpleContainer) asSlice(limit int) []*types.Transaction {
	//c.lock.RLock()
	//defer c.lock.RUnlock()

	size := limit
	if c.pending.size < size {
		size = c.pending.size
	}
	txs := c.pending.asSlice(size)
	return txs
}

func (c *simpleContainer) eachForPack(f func(tx *types.Transaction) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.pending.peek(f)
}

func (c *simpleContainer) eachForSync(f func(tx *types.Transaction) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.pending.eachForSync(f)
}

// push try to push transaction to pool. if error return means the transaction is discarded and the error can be ignored
func (c *simpleContainer) push(tx *types.Transaction) (err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.txsMap[tx.Hash] != nil {
		return
	}
	stateNonce := c.getStateNonce(tx)
	if tx.Nonce <= stateNonce || tx.Nonce > stateNonce+1000 {
		err = Logger.Warnf("Tx nonce error! expect nonce:%d,real nonce:%d ", stateNonce+1, tx.Nonce)
		return
	}

	success := c.pending.push(tx, stateNonce)
	if !success {
		if len(c.queue) > c.queueLimit {
			return
		}
		c.queue[tx.Hash] = tx
	}
	c.txsMap[tx.Hash] = tx
	return
}

func (c *simpleContainer) remove(key common.Hash) {
	if !c.contains(key) {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	tx := c.txsMap[key]
	if tx == nil {
		return
	}

	delete(c.txsMap, key)
	c.pending.remove(tx)
	delete(c.queue, tx.Hash)
}

// promoteQueueToPending tris to move the transactions to the pending list for casting and syncing if possible
func (c *simpleContainer) promoteQueueToPending() {
	c.lock.Lock()
	defer c.lock.Unlock()
	nonceCache := make(map[common.Address]uint64)
	for hash, tx := range c.queue {
		//TODO: queue should order by nonce
		stateNonce := c.getNonceWithCache(nonceCache, tx)
		success := c.pending.push(tx, stateNonce)
		if success {
			delete(c.queue, hash)
		}
	}
}

func (c *simpleContainer) getNonceWithCache(cache map[common.Address]uint64, tx *types.Transaction) uint64 {
	if cache[*tx.Source] != 0 {
		return cache[*tx.Source]
	}
	nonce := c.chain.latestStateDB.GetNonce(*tx.Source)
	cache[*tx.Source] = nonce
	return nonce
}

// getStateNonce fetches nonce from current state db
func (c *simpleContainer) getStateNonce(tx *types.Transaction) uint64 {
	return c.chain.latestStateDB.GetNonce(*tx.Source)
}

func skipGetLast(skip *skip.SkipList) datacommon.Comparator {
	if skip.Len() == 0 {
		return nil
	}
	return skip.ByPosition(skip.Len() - 1)
}
