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
	"fmt"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/monitor"
	"sync/atomic"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

type newTopMessage struct {
	bh *types.BlockHeader
}

func (msg *newTopMessage) GetRaw() []byte {
	return nil
}

func (msg *newTopMessage) GetData() interface{} {
	return msg.bh
}

func (chain *FullBlockChain) saveBlockState(b *types.Block, state *account.AccountDB) error {
	root, err := state.Commit(true)
	if err != nil {
		return fmt.Errorf("state commit error:%s", err.Error())
	}

	triedb := chain.stateCache.TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		return fmt.Errorf("trie commit error:%s", err.Error())
	}
	return nil
}

func (chain *FullBlockChain) saveCurrentBlock(hash common.Hash) error {
	err := chain.blocks.AddKv(chain.batch, []byte(blockStatusKey), hash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (chain *FullBlockChain) updateLatestBlock(state *account.AccountDB, header *types.BlockHeader) {
	chain.latestStateDB = state
	chain.latestBlock = header

	Logger.Debugf("updateLatestBlock success,height=%v,root hash is %x", header.Height, header.StateTree)
	//taslog.Flush()
}

func (chain *FullBlockChain) saveBlockHeader(hash common.Hash, dataBytes []byte) error {
	return chain.blocks.AddKv(chain.batch, hash.Bytes(), dataBytes)
}

func (chain *FullBlockChain) saveBlockHeight(height uint64, dataBytes []byte) error {
	return chain.blockHeight.AddKv(chain.batch, common.UInt64ToByte(height), dataBytes)
}

func (chain *FullBlockChain) saveBlockTxs(blockHash common.Hash, dataBytes []byte) error {
	return chain.txDb.AddKv(chain.batch, blockHash.Bytes(), dataBytes)
}

// commitBlock persist a block in a batch
func (chain *FullBlockChain) commitBlock(block *types.Block, ps *executePostState) (ok bool, err error) {
	traceLog := monitor.NewPerformTraceLogger("commitBlock", block.Header.Hash, block.Header.Height)
	traceLog.SetParent("addBlockOnChain")
	defer traceLog.Log("")

	bh := block.Header

	var (
		headerBytes []byte
		bodyBytes   []byte
	)
	//b := time.Now()
	headerBytes, err = types.MarshalBlockHeader(bh)
	//ps.ts.AddStat("MarshalBlockHeader", time.Since(b))
	if err != nil {
		Logger.Errorf("Fail to json Marshal, error:%s", err.Error())
		return
	}

	//b = time.Now()
	bodyBytes, err = encodeBlockTransactions(block)
	//ps.ts.AddStat("encodeBlockTransactions", time.Since(b))
	if err != nil {
		Logger.Errorf("encode block transaction error:%v", err)
		return
	}

	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	defer chain.batch.Reset()

	// Commit state
	if err = chain.saveBlockState(block, ps.state); err != nil {
		return
	}
	// Save hash to block header key value pair
	if err = chain.saveBlockHeader(bh.Hash, headerBytes); err != nil {
		return
	}
	// Save height to block hash key value pair
	if err = chain.saveBlockHeight(bh.Height, bh.Hash.Bytes()); err != nil {
		return
	}
	// Save hash to transactions key value pair
	if err = chain.saveBlockTxs(bh.Hash, bodyBytes); err != nil {
		return
	}
	// Save hash to receipt key value pair
	if err = chain.transactionPool.SaveReceipts(bh.Hash, ps.receipts); err != nil {
		return
	}
	// Save current block
	if err = chain.saveCurrentBlock(bh.Hash); err != nil {
		return
	}
	// Batch write
	if err = chain.batch.Write(); err != nil {
		return
	}
	//ps.ts.AddStat("batch.Write", time.Since(b))

	chain.updateLatestBlock(ps.state, bh)

	rmTxLog := monitor.NewPerformTraceLogger("RemoveFromPool", block.Header.Hash, block.Header.Height)
	rmTxLog.SetParent("commitBlock")
	defer rmTxLog.Log("")

	// If the block is successfully submitted, the transaction
	// corresponding to the transaction pool should be deleted
	removeTxs := make([]common.Hash, 0)
	if len(ps.txs) > 0 {
		removeTxs = append(removeTxs, ps.txs.txsHashes()...)
	}
	// Remove eviction transactions from the transaction pool
	if ps.evictedTxs != nil {
		if len(ps.evictedTxs) > 0 {
			Logger.Infof("block commit remove evictedTxs: %v, block height: %d", ps.evictedTxs, bh.Height)
		}
		removeTxs = append(removeTxs, ps.evictedTxs...)
	}
	chain.transactionPool.RemoveFromPool(removeTxs)
	ok = true
	return
}

func (chain *FullBlockChain) resetTop(block *types.BlockHeader) error {
	if !chain.isAdjusting {
		chain.isAdjusting = true
		defer func() {
			chain.isAdjusting = false
		}()
	}

	// Add read and write locks, block reading at this time
	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	if nil == block {
		return fmt.Errorf("block is nil")
	}
	traceLog := monitor.NewPerformTraceLogger("resetTop", block.Hash, block.Height)
	traceLog.SetParent("addBlockOnChain")
	defer traceLog.Log("")
	if block.Hash == chain.latestBlock.Hash {
		return nil
	}
	Logger.Debugf("reset top hash:%s height:%d ", block.Hash.Hex(), block.Height)

	var err error

	defer chain.batch.Reset()

	curr := chain.getLatestBlock()
	recoverTxs := make([]*types.Transaction, 0)
	delReceipts := make([]common.Hash, 0)
	removeBlocks := make([]*types.BlockHeader, 0)

	for curr.Hash != block.Hash {
		// Delete the old block header
		if err = chain.saveBlockHeader(curr.Hash, nil); err != nil {
			return err
		}
		// Delete the old block height
		if err = chain.saveBlockHeight(curr.Height, nil); err != nil {
			return err
		}
		// Delete the old block's transactions
		if err = chain.saveBlockTxs(curr.Hash, nil); err != nil {
			return err
		}
		rawTxs := chain.queryBlockTransactionsAll(curr.Hash)
		for _, rawTx := range rawTxs {
			tHash := rawTx.GenHash()
			recoverTxs = append(recoverTxs, types.NewTransaction(rawTx, tHash))
			delReceipts = append(delReceipts, tHash)
		}

		chain.removeTopBlock(curr.Hash)
		removeBlocks = append(removeBlocks, curr)
		Logger.Debugf("remove block %v", curr.Hash.Hex())
		if curr.PreHash == block.Hash {
			break
		}
		curr = chain.queryBlockHeaderByHash(curr.PreHash)
	}
	// Delete receipts corresponding to the transactions in the discard block
	if err = chain.transactionPool.DeleteReceipts(delReceipts); err != nil {
		return err
	}
	// Reset the current block
	if err = chain.saveCurrentBlock(block.Hash); err != nil {
		return err
	}
	state, err := account.NewAccountDB(block.StateTree, chain.stateCache)
	if err != nil {
		return err
	}
	if err = chain.batch.Write(); err != nil {
		return err
	}
	chain.updateLatestBlock(state, block)

	chain.transactionPool.BackToPool(recoverTxs)
	log.ELKLogger.WithFields(logrus.Fields{
		"removedHeight": len(removeBlocks),
		"now":           time.TSInstance.Now().UTC(),
		"logType":       "resetTop",
		"version":       common.GzvVersion,
	}).Info("resetTop")
	for _, b := range removeBlocks {
		GroupManagerImpl.OnBlockRemove(b)
	}
	// invalidate latest cp cache
	chain.latestCP = atomic.Value{}

	// Notify reset top message
	notify.BUS.Publish(notify.NewTopBlock, &newTopMessage{bh: block})

	return nil
}

// removeOrphan remove the orphan block
func (chain *FullBlockChain) removeOrphan(block *types.Block) error {

	// Add read and write locks, block reading at this time
	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	if nil == block {
		return nil
	}
	hash := block.Header.Hash
	height := block.Header.Height
	Logger.Debugf("remove hash:%s height:%d ", hash.Hex(), height)

	var err error
	defer chain.batch.Reset()

	if err = chain.saveBlockHeader(hash, nil); err != nil {
		return err
	}
	if err = chain.saveBlockHeight(height, nil); err != nil {
		return err
	}
	if err = chain.saveBlockTxs(hash, nil); err != nil {
		return err
	}
	txs := chain.queryBlockTransactionsAll(hash)
	if txs != nil {
		txHashs := make([]common.Hash, len(txs))
		for i, tx := range txs {
			txHashs[i] = tx.GenHash()
		}
		if err = chain.transactionPool.DeleteReceipts(txHashs); err != nil {
			return err
		}
	}

	if err = chain.batch.Write(); err != nil {
		return err
	}
	chain.removeTopBlock(hash)
	return nil
}

func (chain *FullBlockChain) loadCurrentBlock() *types.BlockHeader {
	bs, err := chain.blocks.Get([]byte(blockStatusKey))
	if err != nil {
		return nil
	}
	hash := common.BytesToHash(bs)
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) hasBlock(hash common.Hash) bool {
	if ok, _ := chain.blocks.Has(hash.Bytes()); ok {
		return ok
	}
	return false
	//pre := gchain.queryBlockHeaderByHash(bh.PreHash)
	//return pre != nil
}

func (chain *FullBlockChain) hasHeight(h uint64) bool {
	if ok, _ := chain.blockHeight.Has(common.UInt64ToByte(h)); ok {
		return ok
	}
	return false
}

func (chain *FullBlockChain) queryBlockHash(height uint64) *common.Hash {
	result, _ := chain.blockHeight.Get(common.UInt64ToByte(height))
	if result != nil {
		hash := common.BytesToHash(result)
		return &hash
	}
	return nil
}

func (chain *FullBlockChain) queryBlockHashCeil(height uint64) *common.Hash {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(common.UInt64ToByte(height)) {
		hash := common.BytesToHash(iter.Value())
		return &hash
	}
	return nil
}

func (chain *FullBlockChain) queryBlockHeaderByHeightFloor(height uint64) *types.BlockHeader {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(common.UInt64ToByte(height)) {
		realHeight := common.ByteToUInt64(iter.Key())
		if realHeight == height {
			hash := common.BytesToHash(iter.Value())
			bh := chain.queryBlockHeaderByHash(hash)
			if bh == nil {
				Logger.Errorf("data error:height %v, hash %v", height, hash.Hex())
				return nil
			}
			if bh.Height != height {
				Logger.Errorf("key height not equal to value height:keyHeight=%v, valueHeight=%v", realHeight, bh.Height)
				return nil
			}
			return bh
		}
	}
	if iter.Prev() {
		hash := common.BytesToHash(iter.Value())
		return chain.queryBlockHeaderByHash(hash)
	}
	return nil
}

func (chain *FullBlockChain) queryBlockBodyBytes(hash common.Hash) []byte {
	bs, err := chain.txDb.Get(hash.Bytes())
	if err != nil {
		Logger.Errorf("get txDb err:%v, key:%v", err.Error(), hash.Hex())
		return nil
	}
	return bs
}

func (chain *FullBlockChain) queryBlockTransactionsAll(hash common.Hash) []*types.RawTransaction {
	bs := chain.queryBlockBodyBytes(hash)
	if bs == nil {
		return nil
	}
	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		Logger.Errorf("decode transactions err:%v, key:%v", err.Error(), hash.Hex())
		return nil
	}
	return txs
}

func (chain *FullBlockChain) batchGetBlocksAfterHeight(h uint64, limit int) []*types.Block {
	blocks := make([]*types.Block, 0)
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()

	// No higher block after the specified block height
	if !iter.Seek(common.UInt64ToByte(h)) {
		return blocks
	}
	cnt := 0
	for cnt < limit {
		hash := common.BytesToHash(iter.Value())
		b := chain.queryBlockByHash(hash)
		if b == nil {
			break
		}
		blocks = append(blocks, b)
		if !iter.Next() {
			break
		}
		cnt++
	}
	return blocks
}

// scanBlockHeightsInRange returns the heights of block in the given height range. the block with startHeight and endHeight
// will be included
func (chain *FullBlockChain) scanBlockHeightsInRange(startHeight uint64, endHeight uint64) []uint64 {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	// No higher block after the specified block height
	if !iter.Seek(common.UInt64ToByte(startHeight)) {
		return []uint64{}
	}

	hs := make([]uint64, 0)
	for {
		height := common.ByteToUInt64(iter.Key())
		if height > endHeight {
			break
		}
		hs = append(hs, height)
		if !iter.Next() {
			break
		}
	}
	return hs
}

// countBlocksInRange returns the count of blocks in the given height range. the block with startHeight and endHeight
// will be included
func (chain *FullBlockChain) countBlocksInRange(startHeight uint64, endHeight uint64) (count uint64) {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	// No higher block after the specified block height
	if !iter.Seek(common.UInt64ToByte(startHeight)) {
		return
	}
	for {
		height := common.ByteToUInt64(iter.Key())
		if height > endHeight {
			break
		}
		count++
		if !iter.Next() {
			break
		}
	}
	return
}

func (chain *FullBlockChain) queryBlockHeaderByHeight(height uint64) *types.BlockHeader {
	hash := chain.queryBlockHash(height)
	if hash != nil {
		return chain.queryBlockHeaderByHash(*hash)
	}
	return nil
}

func (chain *FullBlockChain) queryBlockByHash(hash common.Hash) *types.Block {
	bh := chain.queryBlockHeaderByHash(hash)
	if bh == nil {
		return nil
	}

	txs := chain.queryBlockTransactionsAll(hash)
	b := &types.Block{
		Header:       bh,
		Transactions: txs,
	}
	return b
}

func (chain *FullBlockChain) queryBlockHeaderBytes(hash common.Hash) []byte {
	result, _ := chain.blocks.Get(hash.Bytes())
	return result
}

func (chain *FullBlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	bs := chain.queryBlockHeaderBytes(hash)
	if bs != nil {
		block, err := types.UnMarshalBlockHeader(bs)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		return block
	}
	return nil
}

func (chain *FullBlockChain) addTopBlock(b *types.Block) {
	chain.topRawBlocks.Add(b.Header.Hash, b)
}

func (chain *FullBlockChain) removeTopBlock(hash common.Hash) {
	chain.topRawBlocks.Remove(hash)
}

func (chain *FullBlockChain) getTopBlockByHash(hash common.Hash) *types.Block {
	if v, ok := chain.topRawBlocks.Get(hash); ok {
		return v.(*types.Block)
	}
	return nil
}

func (chain *FullBlockChain) getTopBlockByHeight(height uint64) *types.Block {
	if chain.topRawBlocks.Len() == 0 {
		return nil
	}
	for _, k := range chain.topRawBlocks.Keys() {
		b := chain.getTopBlockByHash(k.(common.Hash))
		if b != nil && b.Header.Height == height {
			return b
		}
	}
	return nil
}

func (chain *FullBlockChain) queryBlockTransactionsOptional(txIdx int, height uint64) *types.RawTransaction {
	bh := chain.queryBlockHeaderByHeight(height)
	if bh == nil {
		return nil
	}
	bs, err := chain.txDb.Get(bh.Hash.Bytes())
	if err != nil {
		Logger.Errorf("queryBlockTransactionsOptional get txDb err:%v, key:%v", err.Error(), bh.Hash.Hex())
		return nil
	}
	tx, err := decodeTransaction(txIdx, bs)
	if tx != nil {
		return tx
	}
	return nil
}

// batchGetBlocksBetween query blocks of the height range [start, end)
func (chain *FullBlockChain) batchGetBlocksBetween(begin, end uint64) []*types.Block {
	blocks := make([]*types.Block, 0)
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()

	// No higher block after the specified block height
	if !iter.Seek(common.UInt64ToByte(begin)) {
		return blocks
	}
	for {
		height := common.ByteToUInt64(iter.Key())
		if height >= end {
			break
		}
		hash := common.BytesToHash(iter.Value())
		b := chain.queryBlockByHash(hash)
		if b == nil {
			break
		}

		blocks = append(blocks, b)
		if !iter.Next() {
			break
		}
	}
	return blocks
}
