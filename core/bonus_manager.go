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
	"errors"
	"sync"

	"github.com/zvchain/zvchain/common"

	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
)

// BonusManager manage the bonus transactions
type BonusManager struct {
	lock sync.RWMutex
}

func newBonusManager() *BonusManager {
	manager := &BonusManager{}
	return manager
}

func (bm *BonusManager) blockHasBonusTransaction(blockHashByte []byte) bool {
	return BlockChainImpl.LatestStateDB().GetData(common.BonusStorageAddress, string(blockHashByte)) != nil
}

func (bm *BonusManager) GetBonusTransactionByBlockHash(blockHash []byte) *types.Transaction {
	transactionHash := BlockChainImpl.LatestStateDB().GetData(common.BonusStorageAddress, string(blockHash))
	if transactionHash == nil {
		return nil
	}
	transaction := BlockChainImpl.GetTransactionByHash(true, false, common.BytesToHash(transactionHash))
	return transaction
}

// GenerateBonus generate the bonus transaction for the group who just validate a block
func (bm *BonusManager) GenerateBonus(targetIds []int32, blockHash common.Hash, groupID []byte, totalValue uint64) (*types.Bonus, *types.Transaction, error) {
	group := GroupChainImpl.getGroupByID(groupID)
	buffer := &bytes.Buffer{}
	buffer.Write(groupID)
	if len(targetIds) == 0 {
		return nil, nil, errors.New("GenerateBonus targetIds size 0")
	}
	for i := 0; i < len(targetIds); i++ {
		index := targetIds[i]
		buffer.Write(group.Members[index])
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()
	if len(buffer.Bytes())%common.AddressLength != 0 {
		return nil, nil, errors.New("GenerateBonus ExtraData Size Invalid")
	}
	transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeBonus
	transaction.GasPrice = common.MaxUint64
	transaction.Hash = transaction.GenHash()
	return &types.Bonus{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupID: groupID, TotalValue: totalValue}, transaction, nil
}

// ParseBonusTransaction parse a bonus transaction and  returns the group id, targetIds, block hash and transcation value
func (bm *BonusManager) ParseBonusTransaction(transaction *types.Transaction) ([]byte, [][]byte, common.Hash, uint64, error) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupID := make([]byte, common.GroupIDLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupID); n != common.GroupIDLength {
		return nil, nil, common.Hash{}, 0, errors.New("ParseBonusTransaction Read GroupID Fail")
	}
	ids := make([][]byte, 0)
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		if n != common.AddressLength {
			Logger.Debugf("ParseBonusTransaction Addr Size:%d Invalid", n)
			break
		}
		ids = append(ids, addr)
		addr = make([]byte, common.AddressLength)
	}
	blockHash := bm.parseBonusBlockHash(transaction)
	return groupID, ids, blockHash, transaction.Value, nil
}

func (bm *BonusManager) parseBonusBlockHash(tx *types.Transaction) common.Hash {
	return common.BytesToHash(tx.Data)
}

func (bm *BonusManager) contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.BonusStorageAddress, string(blockHash))
	if value != nil {
		return true
	}
	return false
}

func (bm *BonusManager) put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB) {
	accountdb.SetData(common.BonusStorageAddress, string(blockHash), transactionHash)
}
