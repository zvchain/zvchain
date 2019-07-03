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

	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
)

const (
	txNofifyInterval   = 5
	txNotifyRoutine    = "ts_notify"
	txNotifyGap        = 60 * time.Second
	txMaxNotifyPerTime = 50

	txReqRoutine  = "ts_req"
	txReqInterval = 5

	txMaxReceiveLimit = 1000
	txPeerMaxLimit    = 3000
)

type txSyncer struct {
	pool          *txPool
	chain         *FullBlockChain
	rctNotifiy    *lru.Cache
	ticker        *ticker.GlobalTicker
	candidateKeys *lru.Cache
	logger        taslog.Logger
}

var TxSyncer *txSyncer

type peerTxsHashs struct {
	txHashs *lru.Cache
}

func newPeerTxsKeys() *peerTxsHashs {
	return &peerTxsHashs{
		txHashs: common.MustNewLRUCache(txPeerMaxLimit),
	}
}

func (ptk *peerTxsHashs) addTxHashs(hashs []common.Hash) {
	for _, k := range hashs {
		ptk.txHashs.Add(k, 1)
	}
}

func (ptk *peerTxsHashs) removeHashs(hashs []common.Hash) {
	for _, k := range hashs {
		ptk.txHashs.Remove(k)
	}
}

func (ptk *peerTxsHashs) reset() {
	ptk.txHashs = common.MustNewLRUCache(txPeerMaxLimit)
}

func (ptk *peerTxsHashs) hasHash(k common.Hash) bool {
	return ptk.txHashs.Contains(k)
}

func (ptk *peerTxsHashs) forEach(f func(k common.Hash) bool) {
	for _, k := range ptk.txHashs.Keys() {
		if !f(k.(common.Hash)) {
			break
		}
	}
}

func initTxSyncer(chain *FullBlockChain, pool *txPool) {
	s := &txSyncer{
		rctNotifiy:    common.MustNewLRUCache(txPeerMaxLimit),
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

	notify.BUS.Subscribe(notify.TxSyncNotify, s.onTxNotify)
	notify.BUS.Subscribe(notify.TxSyncReq, s.onTxReq)
	notify.BUS.Subscribe(notify.TxSyncResponse, s.onTxResponse)
	TxSyncer = s
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
		// The reward transaction of the block already exists on the chain, or the block is not
		// on the chain, and the corresponding reward transaction needs to be deleted.
		reason := ""
		remove := false
		if ts.pool.bonPool.hasReward(tx.Data) {
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
			ts.logger.Debugf("remove from reward pool because %v: blockHash %v, size %v", reason, bhash.Hex(), rm)
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
		ts.pool.received.eachForPack(func(tx *types.Transaction) bool {
			if ts.checkTxCanBroadcast(tx.Hash) {
				txs = append(txs, tx)
				return len(txs) < txMaxNotifyPerTime
			}
			return true
		})
	}

	ts.sendTxHashs(txs)

	for _, tx := range txs {
		ts.rctNotifiy.Add(tx.Hash, time.Now())
	}
	return true
}

func (ts *txSyncer) sendTxHashs(txs []*types.Transaction) {
	if len(txs) > 0 {
		txHashs := make([]common.Hash, 0)

		for _, tx := range txs {
			txHashs = append(txHashs, tx.Hash)
		}

		bodyBuf := bytes.NewBuffer([]byte{})
		for _, k := range txHashs {
			bodyBuf.Write(k[:])
		}

		ts.logger.Debugf("notify transactions len:%d", len(txs))
		message := network.Message{Code: network.TxSyncNotify, Body: bodyBuf.Bytes()}

		netInstance := network.GetNetInstance()
		if netInstance != nil {
			go network.GetNetInstance().TransmitToNeighbor(message)
		}
	}
}

func (ts *txSyncer) getOrAddCandidateKeys(id string) *peerTxsHashs {
	v, _ := ts.candidateKeys.Get(id)
	if v == nil {
		v = newPeerTxsKeys()
		ts.candidateKeys.Add(id, v)
	}
	return v.(*peerTxsHashs)
}

func (ts *txSyncer) onTxNotify(msg notify.Message) {
	nm := notify.AsDefault(msg)
	reader := bytes.NewReader(nm.Body())
	var (
		hashs = make([]common.Hash, 0)
		buf   = make([]byte, len(common.Hash{}))
		count = 0
	)

	for {
		n, _ := reader.Read(buf)
		if n != len(common.Hash{}) {
			break
		}
		if count > txMaxReceiveLimit {
			ts.logger.Warnf("Rcv onTxNotify,but count exceeds limit")
			return
		}
		count++
		hashs = append(hashs, common.BytesToHash(buf))
	}
	candidateKeys := ts.getOrAddCandidateKeys(nm.Source())
	accepts := make([]common.Hash, 0)
	for _, k := range hashs {
		if exist, _ := ts.pool.IsTransactionExisted(k); !exist {
			accepts = append(accepts, k)
		}
	}
	candidateKeys.addTxHashs(accepts)
	ts.logger.Debugf("Rcv txs notify from %v, size %v, accept %v, totalOfSource %v", nm.Source(), len(hashs), len(accepts), candidateKeys.txHashs.Len())

}

func (ts *txSyncer) reqTxsRoutine() bool {
	if blockSync == nil || blockSync.isSyncing() {
		ts.logger.Debugf("block syncing, won't req txs")
		return false
	}
	ts.logger.Debugf("req txs routine, candidate size %v", ts.candidateKeys.Len())
	reqMap := make(map[common.Hash]byte)
	// Remove the same
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rms := make([]common.Hash, 0)
		ptk.forEach(func(k common.Hash) bool {
			if _, exist := reqMap[k]; exist {
				rms = append(rms, k)
			} else {
				reqMap[k] = 1
			}
			return true
		})
		ptk.removeHashs(rms)
	}
	// Request transaction
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		rqs := make([]common.Hash, 0)
		ptk.forEach(func(k common.Hash) bool {
			if exist, _ := BlockChainImpl.GetTransactionPool().IsTransactionExisted(k); !exist {
				rqs = append(rqs, k)
			}
			return true
		})
		ptk.reset()
		if len(rqs) > 0 {
			go ts.requestTxs(v.(string), &rqs)
		}
	}
	return true
}

func (ts *txSyncer) requestTxs(id string, hash *[]common.Hash) {
	ts.logger.Debugf("request txs from %v, size %v", id, len(*hash))

	bodyBuf := bytes.NewBuffer([]byte{})
	for _, k := range *hash {
		bodyBuf.Write(k[:])
	}

	message := network.Message{Code: network.TxSyncReq, Body: bodyBuf.Bytes()}

	network.GetNetInstance().Send(id, message)
}

func (ts *txSyncer) onTxReq(msg notify.Message) {
	nm := notify.AsDefault(msg)
	reader := bytes.NewReader(nm.Body())
	var (
		hashs = make([]common.Hash, 0)
		buf   = make([]byte, len(common.Hash{}))
		count = 0
	)
	for {
		n, _ := reader.Read(buf)
		if n != len(common.Hash{}) {
			break
		}
		if count > txPeerMaxLimit {
			ts.logger.Warnf("Rcv tx req,but count exceeds limit")
			return
		}
		hashs = append(hashs, common.BytesToHash(buf))
	}
	ts.logger.Debugf("Rcv tx req from %v, size %v", nm.Source(), len(hashs))

	txs := make([]*types.Transaction, 0)
	for _, txHash := range hashs {
		tx := BlockChainImpl.GetTransactionByHash(false, false, txHash)
		if tx != nil {
			txs = append(txs, tx)
		}
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
