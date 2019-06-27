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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

const (
	maxPendingSize              = 40000
	maxQueueSize                = 10000
	rewardTxMaxSize             = 1000
	txCountPerBlock             = 3000
	txAccumulateSizeMaxPerBlock = 1024 * 1024

	txMaxSize = 64000
	// Maximum size per transaction
)

// gasLimitMax expresses the max gasLimit of a transaction
var gasLimitMax = new(types.BigInt).SetUint64(500000)

var (
	ErrNil      = errors.New("nil transaction")
	ErrHash     = errors.New("invalid transaction hash")
	ErrGasPrice = errors.New("gas price is too low")
)

type txPool struct {
	bonPool   *rewardPool
	received  *simpleContainer
	asyncAdds *lru.Cache // Asynchronously added, accelerates validated transaction
	// when add block on chain, does not participate in the broadcast

	receiptDb          *tasdb.PrefixedDatabase
	batch              tasdb.Batch
	chain              BlockChain
	gasPriceLowerBound *types.BigInt
	lock               sync.RWMutex
}

type txPoolAddMessage struct {
	txs   []*types.Transaction
	txSrc txSource
}

func (m *txPoolAddMessage) GetRaw() []byte {
	panic("implement me")
}

func (m *txPoolAddMessage) GetData() interface{} {
	panic("implement me")
}

// newTransactionPool returns a new transaction tool object
func newTransactionPool(chain *FullBlockChain, receiptDb *tasdb.PrefixedDatabase) TransactionPool {
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

func (pool *txPool) tryAddTransaction(tx *types.Transaction, from txSource) (bool, error) {
	if err := pool.RecoverAndValidateTx(tx); err != nil {
		Logger.Debugf("tryAddTransaction err %v, from %v, hash %v, sign %v", err.Error(), from, tx.Hash.Hex(), tx.HexSign())
		return false, err
	}
	b, err := pool.tryAdd(tx)
	if err != nil {
		Logger.Debugf("tryAdd tx fail: from %v, hash=%v, type=%v, err=%v", from, tx.Hash.Hex(), tx.Type, err)
	}
	return b, err
}

// AddTransaction try to add a transaction into the tool
func (pool *txPool) AddTransaction(tx *types.Transaction) (bool, error) {
	return pool.tryAddTransaction(tx, 0)
}

// AddTransaction try to add a list of transactions into the tool
func (pool *txPool) AddTransactions(txs []*types.Transaction, from txSource) {
	if nil == txs || 0 == len(txs) {
		return
	}
	for _, tx := range txs {
		// this error can be ignored
		_, _ = pool.tryAddTransaction(tx, from)
	}
	notify.BUS.Publish(notify.TxPoolAddTxs, &txPoolAddMessage{txs: txs, txSrc: from})
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
		if tx.Type == types.TransactionTypeReward {
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
	height := BlockChainImpl.Height()
	if !tx.IsReward() && !validGasPrice(&tx.GasPrice.Int, height) {
		return ErrGasPrice
	}
	if !tx.Hash.IsValid() {
		return ErrHash
	}
	size := 0
	if tx.Data != nil {
		size += len(tx.Data)
	}
	if tx.ExtraData != nil {
		size += len(tx.ExtraData)
	}
	if size > txMaxSize {
		return fmt.Errorf("tx size(%v) should not larger than %v", size, txMaxSize)
	}

	if tx.Hash != tx.GenHash() {
		return fmt.Errorf("tx hash error")
	}

	if tx.Sign == nil {
		return fmt.Errorf("tx sign nil")
	}

	var source *common.Address
	if tx.IsReward() {
		if ok, err := BlockChainImpl.GetConsensusHelper().VerifyRewardTransaction(tx); !ok {
			return err
		}
	} else {
		if err := tx.BoundCheck(); err != nil {
			return err
		}
		if tx.GasLimit.Cmp(gasLimitMax) > 0 {
			return fmt.Errorf("gasLimit too  big! max gas limit is 500000 Ra")
		}

		var sign = common.BytesToSign(tx.Sign)
		if sign == nil {
			return fmt.Errorf("BytesToSign fail, sign=%v", tx.Sign)
		}
		msg := tx.Hash.Bytes()
		pk, err := sign.RecoverPubkey(msg)
		if err != nil {
			return err
		}
		src := pk.GetAddress()
		source = &src
		tx.Source = source
	}

	return nil
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
		if tx.GasPrice.Cmp(pool.gasPriceLowerBound.Value()) > 0 {
			err = pool.received.push(tx)
		}
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
		txs = append(txs, tx)
		accuSize += tx.Size()
		return accuSize < txAccumulateSizeMaxPerBlock
	})

	if accuSize < txAccumulateSizeMaxPerBlock {
		pool.received.eachForPack(func(tx *types.Transaction) bool {
			// gas price too low
			if tx.GasPrice.Cmp(pool.gasPriceLowerBound.Value()) < 0 {
				return true
			}
			txs = append(txs, tx)
			accuSize = accuSize + tx.Size()
			return accuSize < txAccumulateSizeMaxPerBlock
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
