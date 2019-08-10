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
	"math"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

// Height of chain
func (chain *FullBlockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.QueryTopBlock().Height
}

// TotalQN of chain
func (chain *FullBlockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.QueryTopBlock().TotalQN
}

// GetTransactionByHash get a transaction by hash
func (chain *FullBlockChain) GetTransactionByHash(onlyReward, needSource bool, h common.Hash) *types.Transaction {
	tx := chain.transactionPool.GetTransaction(onlyReward, h)
	if tx == nil {
		chain.rwLock.RLock()
		defer chain.rwLock.RUnlock()
		rc := chain.transactionPool.GetReceipt(h)
		if rc != nil {
			tx = chain.queryBlockTransactionsOptional(int(rc.TxIndex), rc.Height, h)
			if tx != nil && needSource {
				tx.RecoverSource()
			}
		}
	}
	return tx
}

// GetTransactionPool return the transaction pool waiting for the block
func (chain *FullBlockChain) GetTransactionPool() types.TransactionPool {
	return chain.transactionPool
}

// IsAdjusting means whether need to adjust blockchain,
// which means there may be a fork
func (chain *FullBlockChain) IsAdjusting() bool {
	return chain.isAdjusting
}

// LatestAccountDB returns chain's last account database
func (chain *FullBlockChain) LatestAccountDB() (types.AccountDB, error) {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()
	lastBlockHeader := chain.QueryTopBlock()
	preRoot := common.BytesToHash(lastBlockHeader.StateTree.Bytes())
	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	return state, err
}

// QueryTopBlock returns the latest block header
func (chain *FullBlockChain) QueryTopBlock() *types.BlockHeader {
	//chain.rwLock.RLock()
	//defer chain.rwLock.RUnlock()

	return chain.getLatestBlock()
}

// HasBlock returns whether the chain has a block with specific hash
func (chain *FullBlockChain) HasBlock(hash common.Hash) bool {
	if b := chain.getTopBlockByHash(hash); b != nil {
		return true
	}
	return chain.hasBlock(hash)
}

// HasBlock returns whether the chain has a block with specific height
func (chain *FullBlockChain) HasHeight(height uint64) bool {
	return chain.hasHeight(height)
}

// QueryBlockHeaderByHeight returns the block header query by height,
// first query LRU, if there's not exist, then query db
func (chain *FullBlockChain) QueryBlockHeaderByHeight(height uint64) *types.BlockHeader {
	b := chain.getTopBlockByHeight(height)
	if b != nil {
		return b.Header
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	return chain.queryBlockHeaderByHeight(height)
}

// QueryBlockByHeight query the block by height
func (chain *FullBlockChain) QueryBlockByHeight(height uint64) *types.Block {
	b := chain.getTopBlockByHeight(height)
	if b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	bh := chain.queryBlockHeaderByHeight(height)
	if bh == nil {
		return nil
	}
	txs := chain.queryBlockTransactionsAll(bh.Hash)
	return &types.Block{
		Header:       bh,
		Transactions: txs,
	}
}

// QueryBlockHeaderByHash query block header according to the specified hash
func (chain *FullBlockChain) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	if b := chain.getTopBlockByHash(hash); b != nil {
		return b.Header
	}
	return chain.queryBlockHeaderByHash(hash)
}

// QueryBlockByHash query the block by block hash
func (chain *FullBlockChain) QueryBlockByHash(hash common.Hash) *types.Block {
	if b := chain.getTopBlockByHash(hash); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	return chain.queryBlockByHash(hash)
}

// QueryBlockHeaderCeil query first block header whose height >= height
func (chain *FullBlockChain) QueryBlockHeaderCeil(height uint64) *types.BlockHeader {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b.Header
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	hash := chain.queryBlockHashCeil(height)
	if hash == nil {
		return nil
	}
	return chain.queryBlockHeaderByHash(*hash)
}

// QueryBlockCeil query first block whose height >= height
func (chain *FullBlockChain) QueryBlockCeil(height uint64) *types.Block {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	hash := chain.queryBlockHashCeil(height)
	if hash == nil {
		return nil
	}
	return chain.queryBlockByHash(*hash)
}

// QueryBlockHeaderFloor query first block header whose height <= height
func (chain *FullBlockChain) QueryBlockHeaderFloor(height uint64) *types.BlockHeader {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b.Header
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	header := chain.queryBlockHeaderByHeightFloor(height)
	return header
}

// QueryBlockFloor query first block whose height <= height
func (chain *FullBlockChain) QueryBlockFloor(height uint64) *types.Block {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	header := chain.queryBlockHeaderByHeightFloor(height)
	if header == nil {
		return nil
	}

	txs := chain.queryBlockTransactionsAll(header.Hash)
	b := &types.Block{
		Header:       header,
		Transactions: txs,
	}
	return b
}

// QueryBlockBytesFloor query the block byte slice by height
func (chain *FullBlockChain) QueryBlockBytesFloor(height uint64) []byte {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	buf := bytes.NewBuffer([]byte{})
	blockHash, headerBytes := chain.queryBlockHeaderBytesFloor(height)
	if headerBytes == nil {
		return nil
	}
	buf.Write(headerBytes)

	body := chain.queryBlockBodyBytes(blockHash)
	if body != nil {
		buf.Write(body)
	}
	return buf.Bytes()
}

// GetBalance return the balance of specified address
func (chain *FullBlockChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

// GetBalance returns the nonce of specified address
func (chain *FullBlockChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(common.BytesToAddress(address.Bytes()))
}

// GetAccountDBByHash returns account database with specified block hash
func (chain *FullBlockChain) GetAccountDBByHash(hash common.Hash) (types.AccountDB, error) {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	header := chain.queryBlockHeaderByHash(hash)
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}

// AccountDBAt returns account database with specified block height
func (chain *FullBlockChain) AccountDBAt(height uint64) (types.AccountDB, error) {
	if height >= chain.Height() {
		return chain.LatestAccountDB()
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	h := height
	header := chain.queryBlockHeaderByHeightFloor(height)
	if header == nil {
		return nil, fmt.Errorf("no data at height %v-%v", h, height)
	}
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}

// BatchGetBlocksAfterHeight query blocks after the specified height
func (chain *FullBlockChain) BatchGetBlocksAfterHeight(height uint64, limit int) []*types.Block {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()
	return chain.batchGetBlocksAfterHeight(height, limit)
}

// CountBlocksInRange returns the count of block in a range of block height. the block with startHeight and endHeight
// will be included
func (chain *FullBlockChain) CountBlocksInRange(startHeight uint64, endHeight uint64) uint64 {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()
	return chain.countBlocksInRange(startHeight, endHeight)
}

func (chain *FullBlockChain) LatestCheckpoint() *types.BlockHeader {
	h := chain.cpChecker.latestCheckpoint()
	return chain.QueryBlockHeaderByHeight(h)
}
