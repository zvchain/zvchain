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

package types

import (
	"math/big"

	"github.com/zvchain/zvchain/common"
)

// BlockChain is a interface, encapsulates some methods for manipulating the blockchain
type BlockChain interface {
	ChainReader
	AccountRepository

	// CastBlock cast a block, current casters synchronization operation in the group
	CastBlock(height uint64, proveValue []byte, qn uint64, castor []byte, groupSeed common.Hash) *Block

	// AddBlockOnChain add a block on blockchain, there are five cases of return value：
	// 0, successfully add block on blockchain
	// -1, verification failed
	// 1, the block already exist on the blockchain, then we should discard it
	// 2, the same height block with a larger QN value on the chain, then we should discard it
	// 3, need adjust the blockchain, there will be a fork
	AddBlockOnChain(source string, b *Block) AddBlockResult

	// TotalQN of chain
	TotalQN() uint64

	// LatestStateDB returns chain's last account database
	LatestStateDB() (AccountDB, error)

	// QueryBlockByHash query the block by hash
	QueryBlockByHash(hash common.Hash) *Block

	// QueryBlockByHeight query the block by height
	QueryBlockByHeight(height uint64) *Block

	// QueryBlockCeil query first block whose height >= height
	QueryBlockCeil(height uint64) *Block

	// QueryBlockHeaderCeil query first block header whose height >= height
	QueryBlockHeaderCeil(height uint64) *BlockHeader

	// QueryBlockFloor query first block whose height <= height
	QueryBlockFloor(height uint64) *Block

	// QueryBlockHeaderFloor query first block header whose height <= height
	QueryBlockHeaderFloor(height uint64) *BlockHeader

	// BatchGetBlocksAfterHeight query blocks after the specified height
	BatchGetBlocksAfterHeight(height uint64, limit int) []*Block

	// GetTransactionByHash get a transaction by hash
	GetTransactionByHash(onlyReward, needSource bool, h common.Hash) *Transaction

	// GetTransactionPool return the transaction pool waiting for the block
	GetTransactionPool() TransactionPool

	// IsAdjusting means whether need to adjust blockchain, which means there may be a fork
	IsAdjusting() bool

	// Remove removes the block and blocks after it from the chain. Only used in a debug file, should be removed later
	Remove(block *Block) bool

	// Clear clear blockchain all data
	Clear() error

	// Close the open levelDb files
	Close()

	// GetRewardManager returns the reward manager
	GetRewardManager() RewardManager

	// GetAccountDBByHash returns account database with specified block hash
	GetAccountDBByHash(hash common.Hash) (AccountDB, error)

	// GetAccountDBByHeight returns account database with specified block height
	GetAccountDBByHeight(height uint64) (AccountDB, error)

	// GetConsensusHelper returns consensus helper reference
	GetConsensusHelper() ConsensusHelper

	// Version of chain Id
	Version() int

	// ResetTop reset the current top block with parameter bh
	ResetTop(bh *BlockHeader)

	// countBlocksInRange returns the count of block in a range of block height. the block with startHeight and endHeight
	// will be included
	CountBlocksInRange(startHeight uint64, endHeight uint64) uint64
}

type RewardManager interface {
	GetRewardTransactionByBlockHash(blockHash common.Hash) *Transaction
	GenerateReward(targetIds []int32, blockHash common.Hash, gSeed common.Hash, totalValue uint64, packFee uint64) (*Reward, *Transaction, error)
	ParseRewardTransaction(msg TxMessage) (gSeed common.Hash, targets [][]byte, blockHash common.Hash, packFee *big.Int, err error)
	CalculateCastRewardShare(height uint64, gasFee uint64) *CastRewardShare
	HasRewardedOfBlock(blockHash common.Hash, accountdb AccountDB) bool
	MarkBlockRewarded(blockHash common.Hash, transactionHash common.Hash, accountdb AccountDB)
}

// ExecutedTransaction contains the transaction and its receipt
type ExecutedTransaction struct {
	Receipt     *Receipt
	Transaction *Transaction
}

type TransactionPool interface {
	// PackForCast returns a list of transactions for casting a block
	PackForCast() []*Transaction

	// AddTransaction add new transaction to the transaction pool
	AddTransaction(tx *Transaction) (bool, error)

	// AddTransactions add new transactions to the transaction pool
	AddTransactions(txs []*Transaction) int

	// AsyncAddTxs rcv transactions broadcast from other nodes
	AsyncAddTxs(txs []*Transaction)

	// GetTransaction trys to find a transaction from pool by hash and return it
	GetTransaction(reward bool, hash common.Hash) *Transaction

	// GetReceipt returns the transaction's recipe by hash
	GetReceipt(hash common.Hash) *Receipt

	// GetReceived returns the received transactions in the pool with a limited size
	GetReceived() []*Transaction

	// GetAllTxs returns the all received transactions(including pending and queue) in the pool with a limited size
	GetAllTxs() []*Transaction

	// GetRewardTxs returns all the reward transactions in the pool
	GetRewardTxs() []*Transaction

	// TxNum returns the number of transactions in the pool
	TxNum() uint64

	// RemoveFromPool removes the transactions from pool by hash
	RemoveFromPool(txs []common.Hash)

	// BackToPool will put the transactions back to pool
	BackToPool(txs []*Transaction)

	// RecoverAndValidateTx recovers the sender of the transaction and also validates the transaction
	RecoverAndValidateTx(tx *Transaction) error

	SaveReceipts(blockHash common.Hash, receipts Receipts) error

	DeleteReceipts(txs []common.Hash) error

	//check transaction hash exist in local
	IsTransactionExisted(hash common.Hash) (exists bool, where int)
}

// GroupInfoI is a group management interface
type GroupInfoI interface {
}

// VMExecutor is a VM executor
type VMExecutor interface {
	Execute(statedb AccountDB, block *Block) (Receipts, *common.Hash, uint64, error)
}

// AccountRepository contains account query interface
type AccountRepository interface {
	// GetBalance return the balance of specified address
	GetBalance(address common.Address) *big.Int

	// GetBalance returns the nonce of specified address
	GetNonce(address common.Address) uint64
}

type Account interface {
	MinerSk() string
}
