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
	"errors"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common/secp256k1"
	"github.com/zvchain/zvchain/network"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

const (
	maxPendingSize              = 40000
	maxQueueSize                = 10000
	rewardTxMaxSize             = 500
	txCountPerBlock             = 3000
	txAccumulateSizeMaxPerBlock = 1024 * 1024

	txMaxSize              = 64000   // Maximum size per transaction
	gasLimitPerTransaction = 500000  // the max gas limit for a transaction
	GasLimitPerBlock       = 2000000 // the max gas limit for a block
)

var evilErrorMap = map[error]struct{}{
	ErrHash:                          struct{}{},
	ErrSign:                          struct{}{},
	ErrDataSizeTooLong:               struct{}{},
	secp256k1.ErrInvalidMsgLen:       struct{}{},
	secp256k1.ErrRecoverFailed:       struct{}{},
	secp256k1.ErrInvalidSignatureLen: struct{}{},
	secp256k1.ErrInvalidRecoveryID:   struct{}{},
}

var (
	ErrNil             = errors.New("nil transaction")
	ErrHash            = errors.New("invalid transaction hash")
	ErrGasPrice        = errors.New("gas price is too low")
	ErrSign            = errors.New("sign error")
	ErrNonce           = errors.New("nonce error")
	ErrDataSizeTooLong = errors.New("data size too long")
)

type txPool struct {
	bonPool   *rewardPool
	received  *simpleContainer
	asyncAdds *lru.Cache // Asynchronously added, accelerates validated transaction
	// when add block on chain, does not participate in the broadcast

	receiptDb          *tasdb.PrefixedDatabase
	batch              tasdb.Batch
	chain              types.BlockChain
	gasPriceLowerBound *types.BigInt
	lock               sync.RWMutex
}

// newTransactionPool returns a new transaction tool object
func newTransactionPool(chain *FullBlockChain, receiptDb *tasdb.PrefixedDatabase) types.TransactionPool {
	pool := &txPool{
		receiptDb:          receiptDb,
		batch:              chain.batch,
		asyncAdds:          common.MustNewLRUCache(txCountPerBlock * maxReqBlockCount),
		chain:              chain,
		gasPriceLowerBound: types.NewBigInt(uint64(common.GlobalConf.GetInt("chain", "gasprice_lower_bound", 1))),
	}
	pool.received = newSimpleContainer(maxPendingSize, maxQueueSize, chain)
	pool.bonPool = newRewardPool(chain.rewardManager, rewardTxMaxSize)
	initTxSyncer(chain, pool, network.GetNetInstance())

	return pool
}

func (pool *txPool) tryAddTransaction(tx *types.Transaction) (ok bool, err error) {
	defer func() {
		if ok {
			if tx.IsReward() {
				Logger.Debugf("transaction added to pool: hash=%v, block=%v", tx.Hash.Hex(), parseRewardBlockHash(tx).Hex())
			} else {
				Logger.Debugf("transaction added to pool: hash=%v", tx.Hash.Hex())
			}
		}
		if err != nil {
			Logger.Debugf("tryAdd tx fail: hash=%v, type=%v, err=%v", tx.Hash.Hex(), tx.Type, err)
		}
	}()

	if tx.IsReward() {
		if pool.isRewardExists(tx) {
			err = fmt.Errorf("reward tx is exists: block=%v", parseRewardBlockHash(tx).Hex())
			return
		}
	} else {
		if exists, where := pool.IsTransactionExisted(tx.Hash); exists {
			err = fmt.Errorf("tx exists in %v, hash=%v", where, tx.Hash.Hex())
			return
		}
	}
	if err = pool.RecoverAndValidateTx(tx); err != nil {
		Logger.Debugf("tryAddTransaction err %v, hash %v, sign %v", err.Error(), tx.Hash.Hex(), tx.HexSign())
		return
	}
	ok, err = pool.tryAdd(tx)

	return
}

func (pool *txPool) isRewardExists(tx *types.Transaction) bool {
	hash := tx.Hash
	if pool.bonPool.contains(hash) {
		return true
	}
	if pool.asyncAdds.Contains(hash) {
		return true
	}

	if pool.hasReceipt(hash) {
		return true
	}
	// Checks if the reward of corresponding block is exists
	if pool.bonPool.hasReward(parseRewardBlockHash(tx).Bytes()) {
		return true
	}
	return false
}

// AddTransaction try to add a transaction into the tool
func (pool *txPool) AddTransaction(tx *types.Transaction) (bool, error) {
	return pool.tryAddTransaction(tx)
}

// AddTransaction try to add a list of transactions into the tool asynchronously
func (pool *txPool) AsyncAddTransaction(tx *types.Transaction) error {
	if tx.IsReward() {
		if pool.bonPool.get(tx.Hash) != nil {
			return nil
		}
	} else {
		if pool.received.get(tx.Hash) != nil {
			return nil
		}
	}
	if pool.asyncAdds.Contains(tx.Hash) {
		return nil
	}
	if err := pool.recoverAndBasicValidate(tx); err != nil {
		return err
	}
	pool.asyncAdds.Add(tx.Hash, tx)
	return nil
}

// GetTransaction trys to find a transaction from pool by hash and return it
func (pool *txPool) GetTransaction(reward bool, hash common.Hash) *types.Transaction {
	var tx = pool.bonPool.get(hash)
	if reward || tx != nil {
		return tx
	}
	tx = pool.received.get(hash)
	if tx != nil {
		return tx
	}
	if v, ok := pool.asyncAdds.Get(hash); ok {
		return v.(*types.Transaction)
	}
	return nil
}

// GetReceived returns the received transactions in the pool with a limited size
func (pool *txPool) GetReceived() []*types.Transaction {
	return pool.received.asSlice(maxPendingSize + maxQueueSize)
}

// GetAllTxs returns the all received transactions(including pending and queue) in the pool with a limited size
func (pool *txPool) GetAllTxs() []*types.Transaction {
	txs := pool.received.asSlice(maxPendingSize + maxQueueSize)
	for _, tx := range pool.received.queue {
		txs = append(txs, tx)
	}
	return txs
}

// TxNum returns the number of transactions in the pool
func (pool *txPool) TxNum() uint64 {
	return uint64(pool.received.Len() + pool.bonPool.len())
}

// TxQueueNum returns the number of transactions in the queue
func (pool *txPool) TxQueueNum() uint64 {
	return uint64(len(pool.received.queue))
}

// PackForCast returns a list of transactions for casting a block
func (pool *txPool) PackForCast() []*types.Transaction {
	result := pool.packTx()
	return result
}

// RecoverAndValidateTx recovers the sender of the transaction and also validates the transaction, including state validate
func (pool *txPool) RecoverAndValidateTx(tx *types.Transaction) error {
	return getValidator(tx, true)()
}

// recoverAndBasicValidate recovers the sender of the transaction and also validates the transaction, excluding state validate
func (pool *txPool) recoverAndBasicValidate(tx *types.Transaction) error {
	return getValidator(tx, false)()
}

func (pool *txPool) tryAdd(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if exist, where := pool.IsTransactionExisted(tx.Hash); exist {
		return false, fmt.Errorf("tx exist in %v", where)
	}

	err := pool.add(tx)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (pool *txPool) add(tx *types.Transaction) (err error) {
	if tx.IsReward() {
		pool.bonPool.add(tx)
	} else {
		err = pool.received.push(tx)
	}
	return err
}

func (pool *txPool) remove(txHash common.Hash) {
	pool.bonPool.remove(txHash)
	pool.received.remove(txHash)
	pool.asyncAdds.Remove(txHash)
}

func (pool *txPool) IsTransactionExisted(hash common.Hash) (exists bool, where int) {
	if pool.bonPool.contains(hash) {
		return true, 1
	}

	if pool.received.contains(hash) {
		return true, 1
	}

	if pool.asyncAdds.Contains(hash) {
		return true, 2
	}

	if pool.hasReceipt(hash) {
		return true, 3
	}
	return false, -1
}

func (pool *txPool) packTx() []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	accuSize := 0
	pool.bonPool.forEachByBlock(func(bhash common.Hash, rewardTxs []*types.Transaction) bool {
		tx := rewardTxs[0]
		accuSize += tx.Size()
		if accuSize <= txAccumulateSizeMaxPerBlock {
			txs = append(txs, tx)
			return true
		}
		return false
	})

	if accuSize < txAccumulateSizeMaxPerBlock {
		pool.received.eachForPack(func(tx *types.Transaction) bool {
			// gas price too low
			if tx.GasPrice.Cmp(pool.gasPriceLowerBound.Value()) < 0 {
				return true
			}

			// ignore the vm call
			if IgnoreVmCall {
				if tx.Type == types.TransactionTypeContractCreate || tx.Type == types.TransactionTypeContractCall {
					return true
				}
			}

			accuSize = accuSize + tx.Size()
			if accuSize <= txAccumulateSizeMaxPerBlock {
				txs = append(txs, tx)
				return true
			}
			return false
		})
	}
	return txs
}

// RemoveFromPool removes the transactions from pool by hash
func (pool *txPool) RemoveFromPool(txs []common.Hash) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, tx := range txs {
		pool.remove(tx)
	}
	go pool.received.promoteQueueToPending()
}

// BackToPool will put the transactions back to pool
func (pool *txPool) BackToPool(txs []*types.Transaction) {
	for _, tx := range txs {
		// this error can be ignored
		pool.tryAdd(tx)
	}
}

// GetRewardTxs returns all the reward transactions in the pool
func (pool *txPool) GetRewardTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	pool.bonPool.forEachByBlock(func(bhash common.Hash, tx []*types.Transaction) bool {
		txs = append(txs, tx...)
		return true
	})
	return txs
}

// ClearRewardTxs
func (pool *txPool) ClearRewardTxs() {
	pool.bonPool.forEachByBlock(func(bhash common.Hash, txs []*types.Transaction) bool {
		// The reward transaction of the block already exists on the chain, or the block is not
		// on the chain, and the corresponding reward transaction needs to be deleted.
		reason := ""
		remove := false
		if pool.bonPool.hasReward(bhash.Bytes()) {
			remove = true
			reason = "tx exist"
		} else if !pool.chain.HasBlock(bhash) {
			// The block is not on the chain. It may be that this height has passed, or it maybe
			// the height of the future. It cannot be distinguished here.
			remove = true
			reason = "block not exist"
		}

		if remove {
			rm := pool.bonPool.removeByBlockHash(bhash)
			Logger.Debugf("remove from reward pool because %v: blockHash %v, size %v", reason, bhash.Hex(), rm)
		}
		return true
	})
}
