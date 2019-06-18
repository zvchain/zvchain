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
	maxTxPoolSize               = 50000
	bonusTxMaxSize              = 1000
	txCountPerBlock             = 3000
	txAccumulateSizeMaxPerBlock = 1024 * 1024
	gasLimitMax                 = 500000

	// Maximum size per transaction
	txMaxSize = 64000
)

var (
	ErrNil  = errors.New("nil transaction")
	ErrHash = errors.New("invalid transaction hash")
)

type txPool struct {
	bonPool   *bonusPool
	received  *simpleContainer
	asyncAdds *lru.Cache // Asynchronously added, accelerates validated transaction
	// when add block on chain, does not participate in the broadcast

	receiptDb          *tasdb.PrefixedDatabase
	batch              tasdb.Batch
	chain              BlockChain
	gasPriceLowerBound uint64
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
		gasPriceLowerBound: uint64(common.GlobalConf.GetInt("chain", "gasprice_lower_bound", 1)),
	}
	pool.received = newSimpleContainer(maxTxPoolSize)
	pool.bonPool = newBonusPool(chain.bonusManager, bonusTxMaxSize)
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
		pool.tryAddTransaction(tx, from)
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
		if tx.Type == types.TransactionTypeBonus {
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
			TxSyncer.add(tx)
		}
	}
}

// GetTransaction trys to find a transaction from pool by hash and return it
func (pool *txPool) GetTransaction(bonus bool, hash common.Hash) *types.Transaction {
	var tx = pool.bonPool.get(hash)
	if bonus || tx != nil {
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
	return pool.received.asSlice(maxTxPoolSize)
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
	if tx.Type == types.TransactionTypeBonus {
		if ok, err := BlockChainImpl.GetConsensusHelper().VerifyBonusTransaction(tx); !ok {
			return err
		}
	} else {
		if tx.Type == types.TransactionTypeTransfer || tx.Type == types.TransactionTypeContractCall{
			if tx.Target == nil{
				return fmt.Errorf("target is nil")
			}
		}
		if tx.GasPrice == 0 {
			return fmt.Errorf("illegal tx gasPrice")
		}
		if tx.GasLimit > gasLimitMax {
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

		//check nonce
		stateNonce := pool.chain.LatestStateDB().GetNonce(src)
		if !IsTestTransaction(tx) && (tx.Nonce <= stateNonce || tx.Nonce > stateNonce+1000) {
			return fmt.Errorf("nonce error:%v %v", tx.Nonce, stateNonce)
		}
		//
		//if !pk.Verify(msg, sign) {
		//	return fmt.Errorf("verify sign fail, hash=%v", tx.Hash.Hex())
		//}
	}

	return nil
}

func (pool *txPool) tryAdd(tx *types.Transaction) (bool, error) {
	if tx == nil {
		return false, ErrNil
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if exist, where := pool.isTransactionExisted(tx); exist {
		return false, fmt.Errorf("tx exist in %v", where)
	}

	pool.add(tx)

	return true, nil
}

func (pool *txPool) add(tx *types.Transaction) bool {
	if tx.Type == types.TransactionTypeBonus {
		pool.bonPool.add(tx)
	} else {
		pool.received.push(tx)
	}
	TxSyncer.add(tx)

	return true
}

func (pool *txPool) remove(txHash common.Hash) {
	pool.bonPool.remove(txHash)
	pool.received.remove(txHash)
	pool.asyncAdds.Remove(txHash)
}

func (pool *txPool) isTransactionExisted(tx *types.Transaction) (exists bool, where int) {
	if tx.Type == types.TransactionTypeBonus {
		if pool.bonPool.contains(tx.Hash) {
			return true, 1
		}
	} else {
		if pool.received.contains(tx.Hash) {
			return true, 1
		}
	}
	if pool.asyncAdds.Contains(tx.Hash) {
		return true, 2
	}

	if pool.hasReceipt(tx.Hash) {
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
	if len(txs) < txAccumulateSizeMaxPerBlock {
		for _, tx := range pool.received.asSlice(10000) {
			//gas price too low
			if tx.GasPrice < pool.gasPriceLowerBound {
				continue
			}
			txs = append(txs, tx)
			accuSize += tx.Size()
			if accuSize >= txAccumulateSizeMaxPerBlock {
				break
			}
		}
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
}

// BackToPool will put the transactions back to pool
func (pool *txPool) BackToPool(txs []*types.Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, txRaw := range txs {
		if txRaw.Type != types.TransactionTypeBonus && txRaw.Source == nil {
			err := txRaw.RecoverSource()
			if err != nil {
				Logger.Errorf("backtopPool recover source fail:tx=%v", txRaw.Hash.Hex())
				continue
			}
		}
		pool.add(txRaw)
	}
}

// GetBonusTxs returns all the bonus transactions in the pool
func (pool *txPool) GetBonusTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0)
	pool.bonPool.forEach(func(tx *types.Transaction) bool {
		txs = append(txs, tx)
		return true
	})
	return txs
}
