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
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/tasdb"
	"github.com/zvchain/zvchain/taslog"
)

const (
	txNofifyInterval   = 5
	txNotifyRoutine    = "ts_notify"
	txNotifyGap        = 60 * time.Second
	txMaxNotifyPerTime = 50

	txReqRoutine  = "ts_req"
	txReqInterval = 5

	txIndexPersistRoutine  = "tx_index_persist"
	txIndexPersistInterval = 30

	txIndexPersistPerTime = 1000
)

type txSimpleIndexer struct {
	cache *lru.Cache
	db    *tasdb.PrefixedDatabase
}

func buildTxSimpleIndexer() *txSimpleIndexer {
	f := "d_txidx" + common.GlobalConf.GetString("instance", "index", "")
	options := &opt.Options{
		OpenFilesCacheCapacity:        100,
		BlockCacheCapacity:            16 * opt.MiB,
		WriteBuffer:                   32 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           4 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
		CompactionTotalSize:           16 * opt.MiB,
		BlockSize:                     1 * opt.MiB,
	}
	ds, err := tasdb.NewDataSource(f, options)
	if err != nil {
		Logger.Errorf("new datasource error:%v, file=%v", err, f)
		panic(fmt.Errorf("new data source error:file=%v, err=%v", f, err.Error()))
	}
	db, _ := ds.NewPrefixDatabase("tx")
	return &txSimpleIndexer{
		cache: common.MustNewLRUCache(10000),
		db:    db,
	}
}

func (indexer *txSimpleIndexer) close() {
	if indexer.db != nil {
		indexer.db.Close()
	}
}

func (indexer *txSimpleIndexer) cacheLen() int {
	return indexer.cache.Len()
}

func (indexer *txSimpleIndexer) add(tx *types.Transaction) {
	indexer.cache.Add(simpleTxKey(tx.Hash), tx.Hash)
}
func (indexer *txSimpleIndexer) remove(tx *types.Transaction) {
	indexer.cache.Remove(simpleTxKey(tx.Hash))
	indexer.db.Delete(common.UInt64ToByte(simpleTxKey(tx.Hash)))
}
func (indexer *txSimpleIndexer) get(k uint64) *types.Transaction {
	var txHash common.Hash
	var exist = false
	if v, ok := indexer.cache.Peek(k); ok {
		txHash = v.(common.Hash)
		exist = true
	} else {
		bs, err := indexer.db.Get(common.UInt64ToByte(k))
		if err == nil {
			txHash = common.BytesToHash(bs)
			exist = true
		}
	}
	if exist {
		return BlockChainImpl.GetTransactionByHash(false, false, txHash)
	}
	return nil
}

func (indexer *txSimpleIndexer) has(key uint64) bool {
	if indexer.cache.Contains(key) {
		return true
	}
	ok, _ := indexer.db.Has(common.UInt64ToByte(key))
	return ok
}

func (indexer *txSimpleIndexer) persistOldest() (int, error) {
	batch := indexer.db.NewBatch()
	cnt := 0
	removes := make([]uint64, 0)
	for _, k := range indexer.cache.Keys() {
		v, _ := indexer.cache.Peek(k)
		if v != nil {
			cnt++
			removes = append(removes, k.(uint64))
			batch.Put(common.UInt64ToByte(k.(uint64)), v.(common.Hash).Bytes())
		}
		if cnt >= txIndexPersistPerTime {
			break
		}
	}
	if err := batch.Write(); err != nil {
		return 0, err
	}
	for _, k := range removes {
		indexer.cache.Remove(k)
	}
	return len(removes), nil
}

type txSyncer struct {
	pool          *txPool
	chain         *FullBlockChain
	rctNotifiy    *lru.Cache
	indexer       *txSimpleIndexer
	ticker        *ticker.GlobalTicker
	candidateKeys *lru.Cache
	logger        taslog.Logger
}

var TxSyncer *txSyncer

type peerTxsKeys struct {
	lock   sync.RWMutex
	txKeys map[uint64]byte
}

func newPeerTxsKeys() *peerTxsKeys {
	return &peerTxsKeys{
		txKeys: make(map[uint64]byte),
	}
}

func (ptk *peerTxsKeys) addKeys(ks []uint64) {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	for _, k := range ks {
		ptk.txKeys[k] = 1
	}
}

func (ptk *peerTxsKeys) removeKeys(ks []uint64) {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	for _, k := range ks {
		delete(ptk.txKeys, k)
	}
}

func (ptk *peerTxsKeys) reset() {
	ptk.lock.Lock()
	defer ptk.lock.Unlock()
	ptk.txKeys = make(map[uint64]byte)
}

func (ptk *peerTxsKeys) hasKey(k uint64) bool {
	ptk.lock.RLock()
	defer ptk.lock.RUnlock()
	_, ok := ptk.txKeys[k]
	return ok
}

func (ptk *peerTxsKeys) forEach(f func(k uint64) bool) {
	ptk.lock.RLock()
	defer ptk.lock.RUnlock()
	for k := range ptk.txKeys {
		if !f(k) {
			break
		}
	}
}

func initTxSyncer(chain *FullBlockChain, pool *txPool) {
	s := &txSyncer{
		rctNotifiy:    common.MustNewLRUCache(1000),
		indexer:       buildTxSimpleIndexer(),
		pool:          pool,
		ticker:        ticker.NewGlobalTicker("tx_syncer"),
		candidateKeys: common.MustNewLRUCache(100),
		chain:         chain,
		logger:        taslog.GetLoggerByIndex(taslog.TxSyncLogConfig, common.GlobalConf.GetString("instance", "index", "")),
	}
	s.ticker.RegisterPeriodicRoutine(txNotifyRoutine, s.notifyTxs, txNofifyInterval)
	s.ticker.StartTickerRoutine(txNotifyRoutine, false)

	s.ticker.RegisterPeriodicRoutine(txReqRoutine, s.reqTxsRoutine, txReqInterval)
	s.ticker.StartTickerRoutine(txReqRoutine, false)

	s.ticker.RegisterPeriodicRoutine(txIndexPersistRoutine, s.persistIndexRoutine, txIndexPersistInterval)
	s.ticker.StartTickerRoutine(txIndexPersistRoutine, false)

	notify.BUS.Subscribe(notify.TxSyncNotify, s.onTxNotify)
	notify.BUS.Subscribe(notify.TxSyncReq, s.onTxReq)
	notify.BUS.Subscribe(notify.TxSyncResponse, s.onTxResponse)
	TxSyncer = s
}

func simpleTxKey(hash common.Hash) uint64 {
	return hash.Big().Uint64()
}

func (ts *txSyncer) add(tx *types.Transaction) {
	if ts.indexer.cacheLen() >= 10000 {
		ts.indexer.persistOldest()
	}
	ts.indexer.add(tx)
}

func (ts *txSyncer) persistIndexRoutine() bool {
	cnt, err := ts.indexer.persistOldest()
	ts.logger.Infof("persist tx index cache total %v, persist %v, %v", ts.indexer.cacheLen(), cnt, err)
	return true
}

func (ts *txSyncer) clearJob() {
	for _, k := range ts.rctNotifiy.Keys() {
		t, ok := ts.rctNotifiy.Get(k)
		if ok {
			if time.Since(t.(time.Time)).Seconds() > float64(txNotifyGap) {
				ts.rctNotifiy.Remove(k)
			}
		}
	}
	ts.pool.bonPool.forEach(func(tx *types.Transaction) bool {
		bhash := common.BytesToHash(tx.Data)
		// The bonus transaction of the block already exists on the chain, or the block is not
		// on the chain, and the corresponding bonus transaction needs to be deleted.
		reason := ""
		remove := false
		if ts.pool.bonPool.hasBonus(tx.Data) {
			remove = true
			reason = "tx exist"
		} else if !ts.chain.hasBlock(bhash) {
			// The block is not on the chain. It may be that this height has passed, or it maybe
			// the height of the future. It cannot be distinguished here.
			remove = true
			reason = "block not exist"
		}

		if remove {
			rm := ts.pool.bonPool.removeByBlockHash(bhash)
			ts.indexer.remove(tx)
			ts.logger.Debugf("remove from bonus pool because %v: blockHash %v, size %v", reason, bhash.Hex(), rm)
		}
		return true
	})
}

func (ts *txSyncer) checkTxCanBroadcast(txHash common.Hash) bool {
	if t, ok := ts.rctNotifiy.Get(txHash); !ok || time.Since(t.(time.Time)).Seconds() > float64(txNotifyGap) {
		return true
	}
	return false
}

func (ts *txSyncer) notifyTxs() bool {
	ts.clearJob()

	txs := make([]*types.Transaction, 0)
	ts.pool.bonPool.forEach(func(tx *types.Transaction) bool {
		if ts.checkTxCanBroadcast(tx.Hash) {
			txs = append(txs, tx)
			return len(txs) < txMaxNotifyPerTime
		}
		return true
	})

	if len(txs) < txMaxNotifyPerTime {
		for _, tx := range ts.pool.received.asSlice(maxPendingSize+ maxQueueSize) {
			if ts.checkTxCanBroadcast(tx.Hash) {
				txs = append(txs, tx)
				if len(txs) >= txMaxNotifyPerTime {
					break
				}
			}
		}
	}

	ts.sendSimpleTxKeys(txs)

	for _, tx := range txs {
		ts.rctNotifiy.Add(tx.Hash, time.Now())
		ts.indexer.add(tx)
	}

	return true
}

func (ts *txSyncer) sendSimpleTxKeys(txs []*types.Transaction) {
	if len(txs) > 0 {
		txKeys := make([]uint64, 0)

		for _, tx := range txs {
			txKeys = append(txKeys, simpleTxKey(tx.Hash))
		}

		bodyBuf := bytes.NewBuffer([]byte{})
		for _, k := range txKeys {
			bodyBuf.Write(common.UInt64ToByte(k))
		}

		ts.logger.Debugf("notify transactions len:%d", len(txs))
		message := network.Message{Code: network.TxSyncNotify, Body: bodyBuf.Bytes()}

		netInstance := network.GetNetInstance()
		if netInstance != nil {
			go network.GetNetInstance().TransmitToNeighbor(message)
		}
	}
}

func (ts *txSyncer) getOrAddCandidateKeys(id string) *peerTxsKeys {
	v, _ := ts.candidateKeys.Get(id)
	if v == nil {
		v = newPeerTxsKeys()
		ts.candidateKeys.Add(id, v)
	}
	return v.(*peerTxsKeys)
}

func (ts *txSyncer) onTxNotify(msg notify.Message) {
	nm := notify.AsDefault(msg)
	reader := bytes.NewReader(nm.Body())

	keys := make([]uint64, 0)
	buf := make([]byte, 8)
	for {
		n, _ := reader.Read(buf)
		if n != 8 {
			break
		}
		keys = append(keys, common.ByteToUInt64(buf))
	}

	candidateKeys := ts.getOrAddCandidateKeys(nm.Source())

	accepts := make([]uint64, 0)

	for _, k := range keys {
		if !ts.indexer.has(k) {
			accepts = append(accepts, k)
		}
	}
	candidateKeys.addKeys(accepts)
	ts.logger.Debugf("Rcv txs notify from %v, size %v, accept %v, totalOfSource %v", nm.Source(), len(keys), len(accepts), len(candidateKeys.txKeys))

}

func (ts *txSyncer) reqTxsRoutine() bool {
	if blockSync == nil || blockSync.isSyncing() {
		ts.logger.Debugf("block syncing, won't req txs")
		return false
	}
	ts.logger.Debugf("req txs routine, candidate size %v", ts.candidateKeys.Len())
	reqMap := make(map[uint64]byte)
	// Remove the same
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rms := make([]uint64, 0)
		ptk.forEach(func(k uint64) bool {
			if _, exist := reqMap[k]; exist {
				rms = append(rms, k)
			} else {
				reqMap[k] = 1
			}
			return true
		})
		ptk.removeKeys(rms)
	}
	// Request transaction
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rqs := make([]uint64, 0)
		ptk.forEach(func(k uint64) bool {
			if !ts.indexer.has(k) {
				rqs = append(rqs, k)
			}
			return true
		})
		ptk.reset()
		if len(rqs) > 0 {
			go ts.requestTxs(v.(string), rqs)
		}
	}
	return true
}

func (ts *txSyncer) requestTxs(id string, keys []uint64) {
	ts.logger.Debugf("request txs from %v, size %v", id, len(keys))

	bodyBuf := bytes.NewBuffer([]byte{})
	for _, k := range keys {
		bodyBuf.Write(common.UInt64ToByte(k))
	}

	message := network.Message{Code: network.TxSyncReq, Body: bodyBuf.Bytes()}

	network.GetNetInstance().Send(id, message)
}

func (ts *txSyncer) onTxReq(msg notify.Message) {
	nm := notify.AsDefault(msg)
	reader := bytes.NewReader(nm.Body())
	keys := make([]uint64, 0)
	buf := make([]byte, 8)
	for {
		n, _ := reader.Read(buf)
		if n != 8 {
			break
		}
		keys = append(keys, common.ByteToUInt64(buf))
	}

	ts.logger.Debugf("Rcv tx req from %v, size %v", nm.Source(), len(keys))

	txs := make([]*types.Transaction, 0)
	for _, k := range keys {
		tx := ts.indexer.get(k)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	if len(txs) == 0 {
		return
	}
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		ts.logger.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	ts.logger.Debugf("send transactions to %v size %v", nm.Source(), len(txs))
	message := network.Message{Code: network.TxSyncResponse, Body: body}
	network.GetNetInstance().Send(nm.Source(), message)
}

func (ts *txSyncer) Close() {
	if ts.indexer != nil {
		ts.indexer.close()
	}
}

func (ts *txSyncer) onTxResponse(msg notify.Message) {
	nm := notify.AsDefault(msg)
	txs, e := types.UnMarshalTransactions(nm.Body())
	if e != nil {
		ts.logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return
	}

	ts.logger.Debugf("Rcv txs from %v, size %v", nm.Source(), len(txs))
	ts.pool.AddTransactions(txs, txSync)
}
