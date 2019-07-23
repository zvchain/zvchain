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
	"github.com/zvchain/zvchain/common/secp256k1"
	"sync"

	"github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

const (
	maxPendingSize              = 40000
	maxQueueSize                = 10000
	rewardTxMaxSize             = 1000
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
	initTxSyncer(chain, pool)

	return pool
}

func (pool *txPool) tryAddTransaction(tx *types.Transaction) (bool, error) {
	if err := pool.RecoverAndValidateTx(tx); err != nil {
		Logger.Debugf("tryAddTransaction err %v, hash %v, sign %v", err.Error(), tx.Hash.Hex(), tx.HexSign())
		return false, err
	}
	b, err := pool.tryAdd(tx)
	if err != nil {
		Logger.Debugf("tryAdd tx fail: hash=%v, type=%v, err=%v", tx.Hash.Hex(), tx.Type, err)
	}
	return b, err
}

// AddTransaction try to add a transaction into the tool
func (pool *txPool) AddTransaction(tx *types.Transaction) (bool, error) {
	return pool.tryAddTransaction(tx)
}

// AddTransaction try to add a list of transactions into the tool
func (pool *txPool) AddTransactions(txs []*types.Transaction) (evilCount int) {
	if nil == txs || 0 == len(txs) {
		return
	}

	for _, tx := range txs {
		// this error can be ignored
		_, err := pool.tryAddTransaction(tx)
		if err != nil {
			if _, ok := evilErrorMap[err]; ok {
				evilCount++
			}
		}
	}
	return evilCount
}

// AddTransaction try to add a list of transactions into the tool asynchronously
func (pool *txPool) AsyncAddTxs(txs []*types.Transaction) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		if tx.Source != nil {
			continue
		}
		if tx.IsReward() {
			if pool.bonPool.get(tx.Hash) != nil {
				continue
			}
		} else {
			if pool.received.get(tx.Hash) != nil {
				continue
			}
		}
		if pool.asyncAdds.Contains(tx.Hash) {
			continue
		}
		if err := pool.RecoverAndValidateTx(tx); err == nil {
			pool.asyncAdds.Add(tx.Hash, tx)
		}
	}
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
	txs :=  pool.received.asSlice(maxPendingSize + maxQueueSize)
	for _, tx := range pool.received.queue {
		txs = append(txs, tx)
	}
	return txs
}

// TxNum returns the number of transactions in the pool
func (pool *txPool) TxNum() uint64 {
	return uint64(pool.received.Len() + pool.bonPool.len())
}

// PackForCast returns a list of transactions for casting a block
func (pool *txPool) PackForCast() []*types.Transaction {
	result := pool.packTx()
	return result
}

// RecoverAndValidateTx recovers the sender of the transaction and also validates the transaction
func (pool *txPool) RecoverAndValidateTx(tx *types.Transaction) error {
	return getValidator(tx)()
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
	if tx.Type == types.TransactionTypeReward {
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
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		accuSize += tx.Size()
		if accuSize <= txAccumulateSizeMaxPerBlock{
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

			txs = append(txs, tx)

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
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, txRaw := range txs {
		if txRaw.Type != types.TransactionTypeReward && txRaw.Source == nil {
			err := txRaw.RecoverSource()
			if err != nil {
				Logger.Errorf("backtopPool recover source fail:tx=%v", txRaw.Hash.Hex())
				continue
			}
		}
		// this error can be ignored
		_ = pool.add(txRaw)
	}
}

// GetRewardTxs returns all the reward transactions in the pool
func (pool *txPool) GetRewardTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		txs = append(txs, tx)
		return true
	})
	return txs
}

// ClearRewardTxs
func (pool *txPool) ClearRewardTxs() {
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		bhash := common.BytesToHash(tx.Data)
		// The reward transaction of the block already exists on the chain, or the block is not
		// on the chain, and the corresponding reward transaction needs to be deleted.
		reason := ""
		remove := false
		if pool.bonPool.hasReward(tx.Data) {
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
