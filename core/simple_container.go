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

	"github.com/Workiva/go-datastructures/slice/skip"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type simpleContainer struct {
	limit        int
	pendingLimit int
	queueLimit   int

	chain BlockChain

	sortedTxsByPrice *skip.SkipList
	pending          map[common.Address]*sortedTxsByNonce
	queue            map[common.Address]*sortedTxsByNonce

	allTxs map[common.Hash]*types.Transaction

	lock sync.RWMutex
}

type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type willPackedTxs []*types.Transaction

func (h willPackedTxs) Len() int           { return len(h) }
func (h willPackedTxs) Less(i, j int) bool { return h[i].GasPrice > h[j].GasPrice }
func (h willPackedTxs) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *willPackedTxs) Push(x interface{}) {
	*h = append(*h, x.(*types.Transaction))
}

func (h *willPackedTxs) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type sortedTxsByNonce struct {
	items   map[uint64]*types.Transaction
	indexes *nonceHeap
}

func newSimpleContainer(lp, lq int, chain BlockChain) *simpleContainer {
	c := &simpleContainer{
		lock:         sync.RWMutex{},
		limit:        lp + lq,
		pendingLimit: lp,
		queueLimit:   lq,
		chain:        chain,
		allTxs:       make(map[common.Hash]*types.Transaction),
		pending:      make(map[common.Address]*sortedTxsByNonce),
		queue:        make(map[common.Address]*sortedTxsByNonce),

		sortedTxsByPrice: skip.New(uint16(16)),
	}
	return c
}

func (c *simpleContainer) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.allTxs)
}

func (c *simpleContainer) contains(key common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.allTxs[key]
	return ok
}

func (c *simpleContainer) get(key common.Hash) *types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.allTxs[key]
}

func (c *simpleContainer) asSlice(limit int) []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	size := limit
	if len(c.allTxs) < size {
		size = len(c.allTxs)
	}
	txs := make([]*types.Transaction, size)
	for _, tx := range c.allTxs {
		txs = append(txs, tx)
	}
	return txs
}

// syncPending for sync available transactions in pending
func (c *simpleContainer) syncPending(f func(tx *types.Transaction) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, txs := range c.pending {
		for _, tx := range txs.items {
			if !f(tx) {
				return
			}
		}
	}
}

// eachForPack used to pack available transactions in pending, the transaction taken out
// whose nonce is continuous with the nonce in db, and they are sorted by gas price，the
// higher the cost, the more the front
func (c *simpleContainer) eachForPack(f func(tx *types.Transaction) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var txs willPackedTxs

	// currentNonce is the largest nonce of each address in the current willPackedTxs
	currentNonce := make(map[common.Address]uint64)

	for addr, sortedTxs := range c.pending {
		nonce := uint64((*sortedTxs.indexes)[0])
		currentNonce[addr] = nonce

		heap.Push(&txs, sortedTxs.items[nonce])
	}

	for txs.Len() > 0 {
		maxGasPriceTx := heap.Pop(&txs).(*types.Transaction)

		// Check if there is a transaction that meets the required nonce
		if tx, ok := c.pending[*maxGasPriceTx.Source].items[currentNonce[*maxGasPriceTx.Source]+1]; ok {
			heap.Push(&txs, tx)
			currentNonce[*tx.Source] = tx.Nonce
		}

		if !f(maxGasPriceTx) {
			break
		}
	}
}

func (c *simpleContainer) getMinGasTxs(count int) []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	iter := c.sortedTxsByPrice.IterAtPosition(0)
	for count > 0 {
		txs = append(txs, iter.Value().(*types.Transaction))
		count--
		iter.Next()
	}
	return txs
}

// getStateNonce fetches nonce from current state db
func (c *simpleContainer) getStateNonce(tx *types.Transaction) uint64 {
	return c.chain.LatestStateDB().GetNonce(*tx.Source)
}

func (c *simpleContainer) push(tx *types.Transaction) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if _, exist := c.allTxs[tx.Hash]; exist {
		return
	}

	// Compare nonce from DB
	if c.chain != nil {
		lastNonce := c.getStateNonce(tx)
		if tx.Nonce <= lastNonce || tx.Nonce > lastNonce+1000 {
			return
		}
	}

	if len(c.allTxs)-(c.pendingLimit+c.queueLimit) >= 0 {

		minGasTxs := c.getMinGasTxs(len(c.allTxs) - (c.pendingLimit + c.queueLimit) + 1)
		contrastTx := minGasTxs[len(minGasTxs)-1]

		if contrastTx.GasPrice >= tx.GasPrice {
			return
		}

		for _, minGasTx := range minGasTxs {
			if pending := c.pending[*minGasTx.Source]; pending != nil && pending.searchInSorted(minGasTx.Nonce) {
				willEnQueueTxs := pending.removeFromSorted(minGasTx)
				if willEnQueueTxs != nil {
					c.enQueue(willEnQueueTxs...)
				}
				return
			}

			if queue := c.queue[*minGasTx.Source]; queue != nil && queue.searchInSorted(minGasTx.Nonce) {
				queue.removeFromSorted(minGasTx)
				delete(c.allTxs, minGasTx.Hash)
				c.sortedTxsByPrice.Delete(minGasTx)
				return
			}
		}
	}

	if pending := c.pending[*tx.Source]; pending != nil && pending.searchInSorted(tx.Nonce) {

		oldTx := pending.items[tx.Nonce]
		if oldTx.GasPrice >= tx.GasPrice {
			return
		}
		c.replaceThroughSorted(oldTx, tx, pending)
		return
	}

	if queue := c.queue[*tx.Source]; queue != nil && queue.searchInSorted(tx.Nonce) {

		oldTx := queue.items[tx.Nonce]
		if oldTx.GasPrice >= tx.GasPrice {
			return
		}
		c.replaceThroughSorted(oldTx, tx, queue)
		return
	}

	c.enQueue(tx)

}

func (c *simpleContainer) enQueue(newTxs ...*types.Transaction) {

	for _, newTx := range newTxs {
		addr := *newTx.Source
		nonce := newTx.Nonce

		if c.queue[*newTx.Source] == nil {
			c.queue[addr] = newSortedTxsByNonce()
			heap.Init(c.queue[addr].indexes)
		}
		heap.Push(c.queue[addr].indexes, nonce)
		c.queue[addr].items[nonce] = newTx

		if c.allTxs[newTx.Hash] == nil {
			c.allTxs[newTx.Hash] = newTx
			c.sortedTxsByPrice.Insert(newTx)
		}
	}
}

func (c *simpleContainer) enPending(newTxs ...*types.Transaction) {

	for _, newTx := range newTxs {
		addr := *newTx.Source
		nonce := newTx.Nonce

		if c.pending[*newTx.Source] == nil {
			c.pending[addr] = newSortedTxsByNonce()
			heap.Init(c.pending[addr].indexes)
		}

		heap.Push(c.pending[addr].indexes, nonce)
		c.pending[addr].items[nonce] = newTx

		if c.allTxs[newTx.Hash] == nil {
			c.allTxs[newTx.Hash] = newTx
			c.sortedTxsByPrice.Insert(newTx)
		}
	}
}

func (s *sortedTxsByNonce) searchInSorted(nonce uint64) bool {
	if _, exist := s.items[nonce]; exist {
		return true
	}
	return false
}

func (c *simpleContainer) remove(key common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.allTxs[key] == nil {
		return
	}
	tx := c.allTxs[key]
	delete(c.allTxs, key)
	c.sortedTxsByPrice.Delete(tx)

	// Remove from pending
	for _, nonce := range *c.pending[*tx.Source].indexes {
		for j := 0; j < c.pending[*tx.Source].indexes.Len() && (*c.pending[*tx.Source].indexes)[j] == nonce; j++ {

			delete(c.pending[*tx.Source].items, nonce)
			heap.Remove(c.pending[*tx.Source].indexes, j)
		}
		break
	}

	if len(c.pending[*tx.Source].items) == 0 {
		delete(c.pending, *tx.Source)
	}
}

func (c *simpleContainer) replaceThroughSorted(old, newTx *types.Transaction, sortedMap *sortedTxsByNonce) {

	delete(c.allTxs, old.Hash)
	c.sortedTxsByPrice.Delete(old)

	sortedMap.items[old.Nonce] = newTx
	c.allTxs[newTx.Hash] = newTx
	c.sortedTxsByPrice.Insert(newTx)
}

func (c *simpleContainer) getPendingTxsLen() int {
	var pendingTxsLen int
	for _, v := range c.pending {
		pendingTxsLen += v.indexes.Len()
	}
	return pendingTxsLen
}

func (c *simpleContainer) getPendingMaxNonce(addr common.Address) (uint64, bool) {

	var maxNonce uint64
	if pending := c.pending[addr]; pending != nil {
		for nonce := range pending.items {
			if nonce > maxNonce {
				maxNonce = nonce
			}
		}
		return maxNonce, true
	}
	return maxNonce, false
}

// promoteQueueToPending used to promote available transactions to pending,
// these transactions must satisfy each nonce is continuous with largest nonce
// in the pending or the largest nonce in the db
func (c *simpleContainer) promoteQueueToPending() {
	c.lock.Lock()
	defer c.lock.Unlock()

	var accounts []common.Address
	for addr := range c.queue {
		accounts = append(accounts, addr)
	}

	ready := make([]*types.Transaction, 0)

	for _, addr := range accounts {
		if queue := c.queue[addr]; queue != nil && queue.indexes.Len() > 0 {

			next := (*queue.indexes)[0]

			// Find max nonce in pending
			if maxNonce, exist := c.getPendingMaxNonce(addr); exist {
				if next != maxNonce+1 {
					continue
				}
			} else {
				// Find the latest nonce in DB
				if c.chain != nil {
					lastNonce := c.getStateNonce(queue.items[next])
					if next != lastNonce+1 {
						continue
					}
				}
			}

			for ; queue != nil && queue.indexes.Len() > 0 && (*queue.indexes)[0] == next; next++ {
				ready = append(ready, queue.items[next])
				delete(queue.items, next)
				heap.Pop(queue.indexes)
			}
			continue
		}
	}

	c.enPending(ready...)
}

func newSortedTxsByNonce() *sortedTxsByNonce {
	s := &sortedTxsByNonce{
		items:   make(map[uint64]*types.Transaction),
		indexes: new(nonceHeap),
	}
	return s
}

func (s *sortedTxsByNonce) removeFromSorted(tx *types.Transaction) []*types.Transaction {

	nonce := tx.Nonce
	for i := 0; i < s.indexes.Len(); i++ {
		if (*s.indexes)[i] == nonce {
			heap.Remove(s.indexes, i)
			break
		}
	}

	delete(s.items, nonce)
	return s.filter(func(tx *types.Transaction) bool { return tx.Nonce > nonce })
}

func (s *sortedTxsByNonce) filter(compare func(*types.Transaction) bool) []*types.Transaction {
	var removed []*types.Transaction
	if len(s.items) == 0 && s.indexes.Len() == 0 {
		return nil
	}

	for nonce, tx := range s.items {
		if compare(tx) {
			removed = append(removed, tx)
			// Delete from items
			delete(s.items, nonce)
		}
	}
	// Update the heap
	if len(removed) > 0 {
		*s.indexes = make([]uint64, 0, len(s.items))
		for nonce := range s.items {
			*s.indexes = append(*s.indexes, nonce)
		}
		// Init the heap
		heap.Init(s.indexes)
	}
	return removed
}
