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
	"math/big"
	"sync"

	"github.com/zvchain/zvchain/common"

	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
)

const initialRewards = 62 * common.TAS
const halveRewardsPeriod = 30000000
const initialDaemonNodeRatio = 629
const initialMinerNodeRatio = 9228
const adjustRadio = 114
const adjustRadioPeriod = 10000000
const ratioMultiple = 10000

// RewardManager manage the reward transactions
type RewardManager struct {
	lock sync.RWMutex
}

func newRewardManager() *RewardManager {
	manager := &RewardManager{}
	return manager
}

func (bm *RewardManager) blockHasRewardTransaction(blockHashByte []byte) bool {
	return BlockChainImpl.LatestStateDB().GetData(common.RewardStorageAddress, string(blockHashByte)) != nil
}

func (bm *RewardManager) GetRewardTransactionByBlockHash(blockHash []byte) *types.Transaction {
	transactionHash := BlockChainImpl.LatestStateDB().GetData(common.RewardStorageAddress, string(blockHash))
	if transactionHash == nil {
		return nil
	}
	transaction := BlockChainImpl.GetTransactionByHash(true, false, common.BytesToHash(transactionHash))
	return transaction
}

// GenerateReward generate the reward transaction for the group who just validate a block
func (bm *RewardManager) GenerateReward(targetIds []int32, blockHash common.Hash, groupID []byte, totalValue uint64) (*types.Reward, *types.Transaction, error) {
	group := GroupChainImpl.getGroupByID(groupID)
	buffer := &bytes.Buffer{}
	buffer.Write(groupID)
	if len(targetIds) == 0 {
		return nil, nil, errors.New("GenerateReward targetIds size 0")
	}
	for i := 0; i < len(targetIds); i++ {
		index := targetIds[i]
		buffer.Write(group.Members[index])
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()
	if len(buffer.Bytes())%common.AddressLength != 0 {
		return nil, nil, errors.New("GenerateReward ExtraData Size Invalid")
	}
	transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeReward
	transaction.GasPrice = common.MaxUint64
	transaction.Hash = transaction.GenHash()
	return &types.Reward{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupID: groupID, TotalValue: totalValue}, transaction, nil
}

// ParseRewardTransaction parse a reward transaction and  returns the group id, targetIds, block hash and transcation value
func (bm *RewardManager) ParseRewardTransaction(transaction *types.Transaction) ([]byte, [][]byte, common.Hash, uint64, error) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupID := make([]byte, common.GroupIDLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupID); n != common.GroupIDLength {
		return nil, nil, common.Hash{}, 0, errors.New("ParseRewardTransaction Read GroupID Fail")
	}
	ids := make([][]byte, 0)
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		if n != common.AddressLength {
			Logger.Debugf("ParseRewardTransaction Addr Size:%d Invalid", n)
			break
		}
		ids = append(ids, addr)
		addr = make([]byte, common.AddressLength)
	}
	blockHash := bm.parseRewardBlockHash(transaction)
	return groupID, ids, blockHash, transaction.Value, nil
}

func (bm *RewardManager) parseRewardBlockHash(tx *types.Transaction) common.Hash {
	return common.BytesToHash(tx.Data)
}

func (bm *RewardManager) contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.RewardStorageAddress, string(blockHash))
	if value != nil {
		return true
	}
	return false
}

func (bm *RewardManager) put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB) {
	accountdb.SetData(common.RewardStorageAddress, string(blockHash), transactionHash)
}

func (bm *RewardManager) blockRewards(height uint64) uint64 {
	return initialRewards >> (height / halveRewardsPeriod)
}

func (bm *RewardManager) NodesRewards(height uint64) (daemonNodesRewards, minerNodesRewards, userNodesRewards *big.Int) {
	rewards := big.NewInt(0).SetUint64(bm.blockRewards(height))
	if rewards.Uint64() == 0 {
		return
	}
	daemonNodesRewards = big.NewInt(rewards.Int64())
	minerNodesRewards = big.NewInt(rewards.Int64())
	userNodesRewards = big.NewInt(rewards.Int64())
	adjust := height / adjustRadioPeriod * adjustRadio
	daemonNodeRatio := initialDaemonNodeRatio + adjust
	minerNodesRatio := initialMinerNodeRatio - adjust
	daemonNodesRewards = daemonNodesRewards.Mul(daemonNodesRewards, big.NewInt(0).SetUint64(daemonNodeRatio)).Div(daemonNodesRewards, big.NewInt(ratioMultiple))
	minerNodesRewards = minerNodesRewards.Mul(minerNodesRewards, big.NewInt(0).SetUint64(minerNodesRatio)).Div(minerNodesRewards, big.NewInt(ratioMultiple))
	userNodesRewards = userNodesRewards.Sub(userNodesRewards, daemonNodesRewards).Sub(userNodesRewards, minerNodesRewards)
	return
}
