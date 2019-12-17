//   Copyright (C) 2019 ZVChain
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
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type OfflineTailor struct {
	dataSource string
	memSize    int
	outFile    string
	onlyVerify bool

	start        time.Time
	chain        *FullBlockChain
	out          io.Writer
	checkpoint   uint64
	usedNodes    map[common.Hash]struct{}
	incUsedNodes uint64
	usedSize     uint64

	accumulatePrunedNodes uint64
	accumulatePrunedSize  uint64

	lock sync.RWMutex
}

func NewOfflineTailor(dbDir string, mem int, cacheDir string, out string, onlyVerify bool) (*OfflineTailor, error) {
	config := &BlockChainConfig{
		dbfile:      dbDir,
		block:       "bh",
		blockHeight: "hi",
		state:       "st",
		reward:      "nu",
		tx:          "tx",
		receipt:     "rc",
		pruneMode:   false,
	}
	chain := &FullBlockChain{
		config:       config,
		init:         true,
		isAdjusting:  false,
		topRawBlocks: common.MustNewLRUCache(20),
	}

	options := &opt.Options{
		WriteBuffer: 128 * opt.MiB,
		Filter:      filter.NewBloomFilter(10),
	}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
	if err != nil {
		return nil, err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return nil, err
	}
	if onlyVerify {
		// Won't load cache data from file in only verify mode，
		// Just in case the problem that the node was erased is covered up
		chain.stateCache = account.NewDatabaseWithCache(chain.stateDb, false, mem, "")
	} else {
		chain.stateCache = account.NewDatabaseWithCache(chain.stateDb, false, mem, cacheDir)
	}
	chain.latestBlock = chain.loadCurrentBlock()

	cpChecker := newCpChecker(nil, chain)
	cp, err := cpChecker.calcCheckPointWithoutGroup(chain.Height())
	if err != nil {
		return nil, fmt.Errorf("fail to get latest checkpoint, err %v", err)
	}

	tailor := &OfflineTailor{
		dataSource: dbDir,
		memSize:    mem,
		chain:      chain,
		outFile:    out,
		usedNodes:  make(map[common.Hash]struct{}),
		checkpoint: cp,
		onlyVerify: onlyVerify,
	}
	if out == "" {
		tailor.out = os.Stdout
	} else {
		f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		tailor.out = f
	}

	return tailor, nil
}

func (t *OfflineTailor) info(format string, params ...interface{}) {
	t.out.Write([]byte(fmt.Sprintf(format, params...)))
	t.out.Write([]byte("\n"))
}

func (t *OfflineTailor) usedNodeStat() (size uint64, count int) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.usedSize, len(t.usedNodes)
}

func (t *OfflineTailor) resolveCallback(hash common.Hash, data []byte) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if _, ok := t.usedNodes[hash]; !ok {
		t.usedNodes[hash] = struct{}{}
		t.usedSize += uint64(len(data))
		t.incUsedNodes++
	}
}

func (t *OfflineTailor) nodeUsed(hash common.Hash) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if _, ok := t.usedNodes[hash]; !ok {
		return false
	}
	return true
}

func (t *OfflineTailor) collectUsedNodes() error {
	const noPruneBlock = TriesInMemory

	top := t.chain.Height()
	if top <= noPruneBlock {
		return fmt.Errorf("height less than %v, won't prune", noPruneBlock)
	}
	// Find the all heights to be collect nodes
	collectBlockHeights := make([]uint64, 0)
	cnt := uint64(0)
	for s := top; cnt < noPruneBlock; s-- {
		if t.chain.hasHeight(s) {
			collectBlockHeights = append(collectBlockHeights, s)
			if s <= t.checkpoint {
				cnt++
			}
		}
		if s == 0 {
			break
		}
	}
	if uint64(len(collectBlockHeights)) < noPruneBlock {
		return fmt.Errorf("real heights less than %v, won't prune", noPruneBlock)
	}
	t.info("all blocks need to collect using nodes: %v(%v-%v)", len(collectBlockHeights), collectBlockHeights[len(collectBlockHeights)-1], collectBlockHeights[0])
	begin := time.Now()
	for i := len(collectBlockHeights) - 1; i >= 0; i-- {
		h := collectBlockHeights[i]
		t.info("start collect block %v", h)
		b := time.Now()
		t.incUsedNodes = 0
		if _, err := t.chain.IntegrityVerify(h, nil, t.resolveCallback, false); err != nil {
			t.info("verify block %v fail, err %v", h, err)
			return err
		}
		s, c := t.usedNodeStat()
		t.info("collect %v finish, totalNodes %v, incNodes %v, totalSize %vMB, cost %v", h, c, t.incUsedNodes, float64(s)/1024/1024, time.Since(b).String())
	}
	s, c := t.usedNodeStat()
	t.info("collect nodes finished, cost %v, usedNode %v, size %vMB, start erasing...", time.Since(begin).String(), c, float64(s)/1024/1024)
	return nil
}

func (t *OfflineTailor) eraseBatch(batch tasdb.Batch, cnt, size uint64, start, thisRoundBegin time.Time) {
	writeBegin := time.Now()
	if err := batch.Write(); err != nil {
		t.info("write batch error %v", err)
		return
	}
	thisRoundCost := time.Since(thisRoundBegin)
	atomic.AddUint64(&t.accumulatePrunedNodes, cnt)
	atomic.AddUint64(&t.accumulatePrunedSize, size)

	rtSpeed := float64(size) / 1024 / 1024 / thisRoundCost.Seconds()
	totalCost := time.Since(start)

	t.info("erasing nodes %v, size %vMB, speed %.2fMB/s, realtimeSpeed %.2fMB/s, totalCost %v, writeCost %v", t.accumulatePrunedNodes, float64(t.accumulatePrunedSize/1024/1024), float64(t.accumulatePrunedSize)/1024/1024/totalCost.Seconds(), rtSpeed,
		time.Since(thisRoundBegin).String(), time.Since(writeBegin).String())

	batch.Reset()
	cnt = 0
	size = 0
	thisRoundBegin = time.Now()
}

func (t *OfflineTailor) eraseNodes() {
	iter := t.chain.stateDb.NewIterator()
	batch := t.chain.stateDb.NewBatch()
	cnt := uint64(0)
	size := uint64(0)
	begin := time.Now()
	start := begin
	for iter.Next() {
		keyHash := common.BytesToHash(iter.Key())
		if !t.nodeUsed(keyHash) {
			cnt++
			size += uint64(len(iter.Value()))
			batch.Delete(iter.Key())
		}
		if batch.ValueSize() > 50000 {
			t.eraseBatch(batch, cnt, size, start, begin)

			batch.Reset()
			cnt = 0
			size = 0
			begin = time.Now()
		}
	}
	if batch.ValueSize() > 0 {
		t.eraseBatch(batch, cnt, size, start, begin)
	}
	cost := time.Since(start)
	t.info("erasing finished, cost %v, prune nodes %v, size %vMB, speed %vMB/s", time.Since(start).String(), t.accumulatePrunedNodes, float64(t.accumulatePrunedSize/1024/1024), float64(t.accumulatePrunedSize)/1024/1024/cost.Seconds())
}

func (t *OfflineTailor) Pruning() {
	err := t.collectUsedNodes()
	t.chain.stateCache.TrieDB().SaveCache()
	if err == nil {
		t.eraseNodes()
		prefix := getBlockChainConfig().state
		t.info("start compaction...")
		begin := time.Now()
		t.chain.blocks.GetDB().CompactRange(util.Range{Start: append([]byte(prefix), 0x00), Limit: append([]byte(prefix), 0xff)})
		t.info("compaction finished, cost %v", time.Since(begin).String())
	}
}

func (t *OfflineTailor) Verify() error {
	const noPruneBlock = TriesInMemory

	// Find the all heights to be verified
	verifyBlockHeights := make([]uint64, 0)
	cnt := uint64(0)
	for s := t.chain.Height(); cnt < noPruneBlock; s-- {
		if t.chain.hasHeight(s) {
			verifyBlockHeights = append(verifyBlockHeights, s)
			if s <= t.checkpoint {
				cnt++
			}
		}
		if s == 0 {
			break
		}
	}

	t.info("all blocks need to verify: %v(%v-%v)", len(verifyBlockHeights), verifyBlockHeights[len(verifyBlockHeights)-1], verifyBlockHeights[0])
	begin := time.Now()
	for i := len(verifyBlockHeights) - 1; i >= 0; i-- {
		h := verifyBlockHeights[i]
		t.info("start verify block %v", h)
		b := time.Now()
		if _, err := t.chain.IntegrityVerify(h, func(stat *account.VerifyStat) {
			t.info("verify address %v at %v, balance %v, nonce %v, root %v, dataCount %v, dataSize %v, codeSize %v, cost %v", stat.Addr, h, stat.Account.Balance, stat.Account.Nonce, stat.Account.Root.Hex(), stat.DataCount, stat.DataSize, stat.CodeSize, stat.Cost.String())
		}, nil, false); err != nil {
			t.info("verify block %v fail, err %v", h, err)
			return err
		}
		t.info("verify %v finish, cost %v", h, time.Since(b).String())
	}
	t.info("verify nodes finished, cost %v", time.Since(begin).String())
	return nil
}
