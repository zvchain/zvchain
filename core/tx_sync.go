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
	txNofifyInterval    = 5
	txNotifyRoutine     = "ts_notify"
	tickerTxSyncTimeout = "sync_tx_timeout"
	txNotifyGap         = 60
	txMaxNotifyPerTime  = 50

	txReqRoutine  = "ts_req"
	txReqInterval = 5

	txPeerMaxLimit = 3000

	txValidteErrorLimit = 5
	txHitValidRate      = 0.5
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

type peerTxsHashes struct {
	txHashes   *lru.Cache
	sendHashes *lru.Cache
}

func newPeerTxsKeys() *peerTxsHashes {
	return &peerTxsHashes{
		txHashes:   common.MustNewLRUCache(txPeerMaxLimit),
		sendHashes: common.MustNewLRUCache(txPeerMaxLimit),
	}
}

func (ptk *peerTxsHashes) addTxHashes(hashs []common.Hash) {
	for _, k := range hashs {
		ptk.txHashes.Add(k, 1)
	}
}

func (ptk *peerTxsHashes) removeHashes(hashs []common.Hash) {
	for _, k := range hashs {
		ptk.txHashes.Remove(k)
	}
}

func (ptk *peerTxsHashes) reset() {
	ptk.txHashes = common.MustNewLRUCache(txPeerMaxLimit)
}

func (ptk *peerTxsHashes) resetSendHashes() {
	ptk.sendHashes = common.MustNewLRUCache(txPeerMaxLimit)
}

func (ptk *peerTxsHashes) checkReceivedHashesInHitRate(txs []*types.Transaction) bool {
	if ptk.sendHashes.Len() == 0 {
		return true
	}
	hasScaned := make(map[common.Hash]struct{})
	hitHashesLen := 0

	for _, tx := range txs {
		if _, ok := hasScaned[tx.Hash]; ok {
			continue
		}
		if ptk.sendHashes.Contains(tx.Hash) {
			hitHashesLen++
		}
		hasScaned[tx.Hash] = struct{}{}
	}

	rate := float64(hitHashesLen) / float64(ptk.sendHashes.Len())
	if rate < txHitValidRate {
		return false
	}
	return true
}

func (ptk *peerTxsHashes) addSendHash(txHash common.Hash) {
	ptk.sendHashes.ContainsOrAdd(txHash, 1)
}

func (ptk *peerTxsHashes) hasHash(k common.Hash) bool {
	return ptk.txHashes.Contains(k)
}

func (ptk *peerTxsHashes) forEach(f func(k common.Hash) bool) {
	for _, k := range ptk.txHashes.Keys() {
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
	ts.pool.ClearRewardTxs()
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
		ts.pool.received.eachForSync(func(tx *types.Transaction) bool {
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

func (ts *txSyncer) getOrAddCandidateKeys(id string) *peerTxsHashes {
	v, _ := ts.candidateKeys.Get(id)
	if v == nil {
		v = newPeerTxsKeys()
		ts.candidateKeys.Add(id, v)
	}
	return v.(*peerTxsHashes)
}

func (ts *txSyncer) onTxNotify(msg notify.Message)error {
	nm := notify.AsDefault(msg)
	if peerManagerImpl.getOrAddPeer(nm.Source()).isEvil() {
		ts.logger.Warnf("tx sync this source is is in evil...source is is %v\n", nm.Source())
		return fmt.Errorf("tx sync this source is is in evil...source is is %v\n", nm.Source())
	}
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
		if count > txMaxNotifyPerTime {
			ts.logger.Warnf("Rcv onTxNotify,but count exceeds limit")
			return fmt.Errorf("Rcv onTxNotify,but count exceeds limit")
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
	candidateKeys.addTxHashes(accepts)
	ts.logger.Debugf("Rcv txs notify from %v, size %v, accept %v, totalOfSource %v", nm.Source(), len(hashs), len(accepts), candidateKeys.txHashes.Len())
	return nil
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
		ptk.removeHashes(rms)
	}
	// Request transaction
	for _, v := range ts.candidateKeys.Keys() {
		ptk := ts.getOrAddCandidateKeys(v.(string))
		if ptk == nil {
			continue
		}
		if ptk.sendHashes.Len() > 0 {
			continue
		}
		rqs := make([]common.Hash, 0)
		ptk.forEach(func(k common.Hash) bool {
			if exist, _ := BlockChainImpl.GetTransactionPool().IsTransactionExisted(k); !exist {
				rqs = append(rqs, k)
				ptk.addSendHash(k)
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

	ts.chain.ticker.RegisterOneTimeRoutine(ts.syncTimeoutRoutineName(id), func() bool {
		return ts.syncTxComplete(id, true)
	}, syncNeightborTimeout)
}

func (ts *txSyncer) syncTxComplete(id string, timeout bool) bool {
	if timeout {
		peerManagerImpl.timeoutPeer(id)
		ts.logger.Warnf("sync txs from %v timeout", id)
	} else {
		peerManagerImpl.heardFromPeer(id)
	}
	candidateKeys := ts.getOrAddCandidateKeys(id)
	candidateKeys.resetSendHashes()
	ts.chain.ticker.RemoveRoutine(ts.syncTimeoutRoutineName(id))
	return true
}

func (ts *txSyncer) syncTimeoutRoutineName(id string) string {
	return tickerTxSyncTimeout + id
}

func (ts *txSyncer) onTxReq(msg notify.Message)error {
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
			return fmt.Errorf("Rcv tx req,but count exceeds limit")
		}
		count++
		hashs = append(hashs, common.BytesToHash(buf))
	}
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
		return  fmt.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
	}
	ts.logger.Debugf("Rcv tx req from %v, size %v,send transactions to %v size %v", nm.Source(), len(hashs),nm.Source(), len(txs))
	message := network.Message{Code: network.TxSyncResponse, Body: body}
	network.GetNetInstance().Send(nm.Source(), message)
	return nil
}

func (ts *txSyncer) onTxResponse(msg notify.Message)error {
	nm := notify.AsDefault(msg)
	if peerManagerImpl.getOrAddPeer(nm.Source()).isEvil() {
		ts.logger.Warnf("on tx response this source is is in evil...source is is %v\n", nm.Source())
		return fmt.Errorf("on tx response this source is is in evil...source is is %v\n", nm.Source())
	}

	defer func() {
		ts.syncTxComplete(nm.Source(), false)
	}()

	txs, e := types.UnMarshalTransactions(nm.Body())
	if e != nil {
		ts.logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return fmt.Errorf("Unmarshal got transactions error:%s", e.Error())
	}

	if len(txs) > txPeerMaxLimit {
		ts.logger.Errorf("rec tx too much,length is %v ,and from %s", len(txs), nm.Source())
		return fmt.Errorf("rec tx too much,length is %v ,and from %s", len(txs), nm.Source())
	}
	ts.logger.Debugf("Rcv txs from %v, size %v", nm.Source(), len(txs))
	evilCount := ts.pool.AddTransactions(txs)
	if evilCount > txValidteErrorLimit {
		ts.logger.Errorf("rec tx evil count over limit,count is %d", evilCount)
		peerManagerImpl.addEvilCount(nm.Source())
		return fmt.Errorf("rec tx evil count over limit,count is %d", evilCount)
	}

	peerManagerImpl.resetEvilCount(nm.Source())
	return nil
}
