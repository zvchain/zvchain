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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/vm"

	"math/big"

	"github.com/zvchain/zvchain/middleware/types"
)

// BlockChain is a interface, encapsulates some methods for manipulating the blockchain
type BlockChain interface {
	vm.ChainReader
	AccountRepository

	// CastBlock cast a block, current casters synchronization operation in the group
	CastBlock(height uint64, proveValue []byte, qn uint64, castor []byte, groupid []byte) *types.Block

	// AddBlockOnChain add a block on blockchain, there are five cases of return value：
	// 0, successfully add block on blockchain
	// -1, verification failed
	// 1, the block already exist on the blockchain, then we should discard it
	// 2, the same height block with a larger QN value on the chain, then we should discard it
	// 3, need adjust the blockchain, there will be a fork
	AddBlockOnChain(source string, b *types.Block) types.AddBlockResult

	// TotalQN of chain
	TotalQN() uint64

	// LatestStateDB returns chain's last account database
	LatestStateDB() *account.AccountDB

	// QueryBlockByHash query the block by hash
	QueryBlockByHash(hash common.Hash) *types.Block

	// QueryBlockByHeight query the block by height
	QueryBlockByHeight(height uint64) *types.Block

	// QueryBlockCeil query first block whose height >= height
	QueryBlockCeil(height uint64) *types.Block

	// QueryBlockHeaderCeil query first block header whose height >= height
	QueryBlockHeaderCeil(height uint64) *types.BlockHeader

	// QueryBlockFloor query first block whose height <= height
	QueryBlockFloor(height uint64) *types.Block

	// QueryBlockHeaderFloor query first block header whose height <= height
	QueryBlockHeaderFloor(height uint64) *types.BlockHeader

	// QueryBlockBytesFloor query the block byte slice by height
	QueryBlockBytesFloor(height uint64) []byte

	// BatchGetBlocksAfterHeight query blocks after the specified height
	BatchGetBlocksAfterHeight(height uint64, limit int) []*types.Block

	// GetTransactionByHash get a transaction by hash
	GetTransactionByHash(onlyReward, needSource bool, h common.Hash) *types.Transaction

	// GetTransactionPool return the transaction pool waiting for the block
	GetTransactionPool() TransactionPool

	// IsAdjusting means whether need to adjust blockchain, which means there may be a fork
	IsAdjusting() bool

	// Remove removes the block and blocks after it from the chain. Only used in a debug file, should be removed later
	Remove(block *types.Block) bool

	// Clear clear blockchain all data
	Clear() error

	// Close the open levelDb files
	Close()

	// GetRewardManager returns the reward manager
	GetRewardManager() *RewardManager

	// GetAccountDBByHash returns account database with specified block hash
	GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error)

	// GetAccountDBByHeight returns account database with specified block height
	GetAccountDBByHeight(height uint64) (vm.AccountDB, error)

	// GetConsensusHelper returns consensus helper reference
	GetConsensusHelper() types.ConsensusHelper

	// Version of chain Id
	Version() int

	// ResetTop reset the current top block with parameter bh
	ResetTop(bh *types.BlockHeader)
}

// ExecutedTransaction contains the transaction and its receipt
type ExecutedTransaction struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

type txSource int

const (
	txSync txSource = 1
)

type TransactionPool interface {
	// PackForCast returns a list of transactions for casting a block
	PackForCast() []*types.Transaction

	// AddTransaction add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	// AddTransactions add new transactions to the transaction pool
	AddTransactions(txs []*types.Transaction, from txSource)

	// AsyncAddTxs rcv transactions broadcast from other nodes
	AsyncAddTxs(txs []*types.Transaction)

	// GetTransaction trys to find a transaction from pool by hash and return it
	GetTransaction(reward bool, hash common.Hash) *types.Transaction

	// GetTransactionStatus returns the execute result status by hash
	GetTransactionStatus(hash common.Hash) (int, error)

	// GetReceipt returns the transaction's recipe by hash
	GetReceipt(hash common.Hash) *types.Receipt

	// GetReceived returns the received transactions in the pool with a limited size
	GetReceived() []*types.Transaction

	// GetRewardTxs returns all the reward transactions in the pool
	GetRewardTxs() []*types.Transaction

	// TxNum returns the number of transactions in the pool
	TxNum() uint64

	// RemoveFromPool removes the transactions from pool by hash
	RemoveFromPool(txs []common.Hash)

	// BackToPool will put the transactions back to pool
	BackToPool(txs []*types.Transaction)

	// RecoverAndValidateTx recovers the sender of the transaction and also validates the transaction
	RecoverAndValidateTx(tx *types.Transaction) error

	saveReceipts(blockHash common.Hash, receipts types.Receipts) error

	deleteReceipts(txs []common.Hash) error
}

// GroupInfoI is a group management interface
type GroupInfoI interface {
}

// VMExecutor is a VM executor
type VMExecutor interface {
	Execute(statedb *account.AccountDB, block *types.Block) (types.Receipts, *common.Hash, uint64, error)
}

// AccountRepository contains account query interface
type AccountRepository interface {
	// GetBalance return the balance of specified address
	GetBalance(address common.Address) *big.Int

	// GetBalance returns the nonce of specified address
	GetNonce(address common.Address) uint64
}
