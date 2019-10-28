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
	"bytes"
	"container/heap"
	"fmt"
	"sort"
	"sync"
	"time"

	datacommon "github.com/Workiva/go-datastructures/common"
	"github.com/Workiva/go-datastructures/slice/skip"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

const maxSyncCountPreSource = 50 // max count of tx with same source to sync to neighbour node

type simpleContainer struct {
	txsMap     map[common.Hash]*TransactionWithTime
	chain      *FullBlockChain
	pending    *pendingContainer
	queue      map[common.Hash]*types.Transaction
	queueLimit int
	txTimeout  time.Duration

	lock sync.RWMutex
}

type TransactionWithTime struct {
	item  *types.Transaction
	begin time.Time
}

func warpTransaction(tx *types.Transaction) *TransactionWithTime {
	return &TransactionWithTime{tx, time.Now()}
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
func (s *pendingContainer) push(tx *types.Transaction, stateNonce uint64) (added bool, evicted *types.Transaction, conflicted *types.Transaction) {
	var doInsertOrReplace = func() {
		newTxNode := newOrderByNonceTx(tx)
		existSource := s.waitingMap[*tx.Source].Get(newTxNode)[0]

		if existSource != nil {
			if existSource.(*orderByNonceTx).item.GasPrice.Cmp(tx.GasPrice.Value()) < 0 {
				//replace the existing one
				deleted := s.waitingMap[*tx.Source].Delete(existSource)
				if len(deleted) > 0 {
					Logger.Debugf("replace tx by high price: old=%v, new=%v", deleted[0].(*orderByNonceTx).item.Hash.Hex(), tx.Hash.Hex())
				}
				s.waitingMap[*tx.Source].Insert(newTxNode)
				evicted = existSource.(*orderByNonceTx).item
				conflicted = tx
			} else {
				evicted = tx
				conflicted = existSource.(*orderByNonceTx).item
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
			added = false
			return
		}
		bigNonceTx := skipGetLast(s.waitingMap[*tx.Source])
		if bigNonceTx != nil {
			bigNonce := bigNonceTx.(*orderByNonceTx).item.Nonce
			if tx.Nonce > bigNonce+1 {
				added = false
				return
			}

			doInsertOrReplace()
		}
	}

	//remove lowest price transaction if pending is full
	var lowPriceTx *types.Transaction
	if evicted == nil && s.size >= s.limit {
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
			evicted = lowPriceTx
			conflicted = tx
			Logger.Debugf("remove tx as pool is full: hash=%v, current pool size=%v", lowPriceTx.Hash.Hex(), s.size)
		}
	}
	added = true
	return
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
	countMap := make(map[common.Address]int)
	for _, txSkip := range s.waitingMap {
		for iter1 := txSkip.IterAtPosition(0); iter1.Next(); {
			tx := iter1.Value().(*orderByNonceTx).item
			count := countMap[*tx.Source]
			if count >= maxSyncCountPreSource {
				continue
			}
			count++
			countMap[*tx.Source] = count
			if !f(tx) {
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
	//timeOutDuration is the max time of a tx can keeped in tx pool, default value is 30 minutes
	timeOutDuration := common.GlobalConf.GetInt(configSec, "tx_timeout_duration", 60*30)
	timeout := time.Second * time.Duration(timeOutDuration)

	c := &simpleContainer{
		lock:       sync.RWMutex{},
		chain:      chain.(*FullBlockChain),
		txsMap:     make(map[common.Hash]*TransactionWithTime),
		pending:    newPendingContainer(pendingLimit),
		queue:      make(map[common.Hash]*types.Transaction),
		queueLimit: queueLimit,
		txTimeout:  timeout,
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			c.clearRoute()
		}
	}()

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
	warpedTx := c.txsMap[key]
	if warpedTx != nil {
		return warpedTx.item
	}
	return nil
}

func (c *simpleContainer) asSlice(limit int) []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

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
		err = ErrNonce
		Logger.Warnf("Tx nonce error! expect nonce:%d,real nonce:%d, source:%s ", stateNonce+1, tx.Nonce, tx.Source.AddrPrefixString())
		return
	}

	success, evicted, conflicted := c.pending.push(tx, stateNonce)
	if !success {
		evicted, conflicted, err = c.addToQueue(tx)
		if err != nil {
			return
		}
	}
	c.txsMap[tx.Hash] = warpTransaction(tx)
	if evicted != nil {
		Logger.Debugf("Tx %v replaced by %v as higher gas price when push()", evicted.Hash, tx.Hash)
		if evicted.Hash == tx.Hash {
			err = fmt.Errorf("existing a transaction with same nonce: %v", conflicted.Hash.Hex())
		}
		delete(c.txsMap, evicted.Hash)
	}
	return
}

func (c *simpleContainer) addToQueue(tx *types.Transaction) (evicted *types.Transaction, conflicted *types.Transaction, err error) {
	if len(c.queue) > c.queueLimit {
		err = fmt.Errorf("tx_pool's queue is full. current queue size: %d", len(c.queue))
		return
	}
	for _, old := range c.queue {
		if old.Nonce == tx.Nonce && bytes.Equal(old.Source.Bytes(), tx.Source.Bytes()) {
			if old.GasPrice.Cmp(tx.GasPrice.Value()) >= 0 {
				evicted = tx
				conflicted = old
				return
			} else {
				evicted = old
				conflicted = tx
				delete(c.queue, evicted.Hash)
			}
		}
	}
	c.queue[tx.Hash] = tx
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
	c.removeWithoutLock(tx.item)
}

func (c *simpleContainer) removeWithoutLock(tx *types.Transaction) {
	delete(c.txsMap, tx.Hash)
	c.pending.remove(tx)
	delete(c.queue, tx.Hash)
}

type nonceTxSlice []*types.Transaction

func (s nonceTxSlice) Len() int           { return len(s) }
func (s nonceTxSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s nonceTxSlice) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }

func mapToNonceTxSlice(txMap map[common.Hash]*types.Transaction) *nonceTxSlice {
	sl := make(nonceTxSlice, 0, len(txMap))
	for _, v := range txMap {
		sl = append(sl, v)
	}

	sort.Sort(sl)
	return &sl
}

// promoteQueueToPending tris to move the transactions to the pending list for casting and syncing if possible
func (c *simpleContainer) promoteQueueToPending() {
	c.lock.Lock()
	defer c.lock.Unlock()
	nonceCache := make(map[common.Address]uint64)
	sl := mapToNonceTxSlice(c.queue)
	for _, tx := range *sl {
		stateNonce := c.getNonceWithCache(nonceCache, tx)
		if tx.Nonce <= stateNonce {
			Logger.Debugf("Tx %v removed from pool as same nonce tx existing in the chain", tx.Hash)
			delete(c.txsMap, tx.Hash)
			delete(c.queue, tx.Hash)
			continue
		}
		success, evicted, _ := c.pending.push(tx, stateNonce)
		if evicted != nil {
			Logger.Debugf("Tx %v replaced by %v as higher gas price when promoteQueueToPending", evicted.Hash, tx.Hash)
			delete(c.txsMap, evicted.Hash)
			delete(c.queue, evicted.Hash)
		}
		if success {
			delete(c.queue, tx.Hash)
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

func (c *simpleContainer) evictPending() {
	nonceCache := make(map[common.Address]uint64)
	txs := c.asSlice(maxPendingSize)
	for _, tx := range txs {
		stateNonce := c.getNonceWithCache(nonceCache, tx)
		if tx.Nonce <= stateNonce {
			Logger.Debugf("Tx %v evicted from pending, chain nonce is %d and tx nonce is %d", tx.Hash, stateNonce, tx.Nonce)
			c.remove(tx.Hash)
		}
	}
}

func (c *simpleContainer) evictTimeout() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, tx := range c.txsMap {
		if time.Since(tx.begin) > c.txTimeout {
			Logger.Debugf("Tx %v evicted as timeout, tx entered to pool on %v", tx.item.Hash, tx.begin)
			c.removeWithoutLock(tx.item)
		}
	}
}

func (c *simpleContainer) clearRoute() {
	start := time.Now()
	c.evictPending()
	c.evictTimeout()
	Logger.Debugf("clearRoute tasks %f seconds", time.Since(start).Seconds())
}
