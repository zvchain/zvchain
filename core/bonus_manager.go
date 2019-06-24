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

const (
	initialRewards     = 62 * common.TAS
	halveRewardsPeriod = 30000000
	halveRewardsTimes  = 3
	tokensOfMiners     = 3500000000 * common.TAS
	tokenLeftKey       = "TokenLeft"
)

const (
	initialDaemonNodeWeight = 629
	initialMinerNodeWeight  = 9228
	userNodeWeight          = 143
	totalNodeWeight         = 10000
	adjustWeight            = 114
	adjustWeightPeriod      = 10000000
)

const (
	castorRewardsWeight = 8
	packedRewardsWeight = 1
	verifyRewardsWeight = 1
	totalRewardsWeight  = 10
)

const (
	gasFeeCastorRewardsWeight = 9
	gasFeeVerifyRewardsWeight = 1
	gasFeeTotalRewardsWeight  = 10
)

var (
	userNodeAddress   = common.HexToAddress("0xe30c75b3fd8888f410ac38ec0a07d82dcc613053513855fb4dd6d75bc69e8139")
	daemonNodeAddress = common.HexToAddress("0xae1889182874d8dad3c3e033cde3229a3320755692e37cbe1caab687bf6a1122")
)

// RewardManager manage the reward transactions
type RewardManager struct {
	noRewards       bool
	noRewardsHeight uint64
	tokenLeft       uint64
	lock            sync.RWMutex
}

func newRewardManager() *RewardManager {
	manager := &RewardManager{}
	value := BlockChainImpl.LatestStateDB().GetData(common.RewardStorageAddress, tokenLeftKey)
	if value == nil {
		manager.tokenLeft = tokensOfMiners
	} else {
		manager.tokenLeft = common.ByteToUint64(value)
	}
	return manager
}

func (rm *RewardManager) blockHasRewardTransaction(blockHashByte []byte) bool {
	return BlockChainImpl.LatestStateDB().GetData(common.RewardStorageAddress, string(blockHashByte)) != nil
}

func (rm *RewardManager) GetRewardTransactionByBlockHash(blockHash []byte) *types.Transaction {
	transactionHash := BlockChainImpl.LatestStateDB().GetData(common.RewardStorageAddress, string(blockHash))
	if transactionHash == nil {
		return nil
	}
	transaction := BlockChainImpl.GetTransactionByHash(true, false, common.BytesToHash(transactionHash))
	return transaction
}

// GenerateReward generate the reward transaction for the group who just validate a block
func (rm *RewardManager) GenerateReward(targetIds []int32, blockHash common.Hash, groupID []byte, totalValue uint64) (*types.Reward, *types.Transaction, error) {
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
	transaction.Value = types.NewBigInt(totalValue / uint64(len(targetIds)))
	transaction.Type = types.TransactionTypeReward
	transaction.GasPrice = types.NewBigInt(0)
	transaction.Hash = transaction.GenHash()
	return &types.Reward{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupID: groupID, TotalValue: totalValue}, transaction, nil
}

// ParseRewardTransaction parse a bonus transaction and  returns the group id, targetIds, block hash and transcation value
func (rm *RewardManager) ParseRewardTransaction(transaction *types.Transaction) ([]byte, [][]byte, common.Hash, *types.BigInt, error) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupID := make([]byte, common.GroupIDLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupID); n != common.GroupIDLength {
		return nil, nil, common.Hash{}, nil, errors.New("ParseRewardTransaction Read GroupID Fail")
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
	blockHash := rm.parseRewardBlockHash(transaction)
	return groupID, ids, blockHash, transaction.Value, nil
}

func (rm *RewardManager) parseRewardBlockHash(tx *types.Transaction) common.Hash {
	return common.BytesToHash(tx.Data)
}

func (rm *RewardManager) contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.RewardStorageAddress, string(blockHash))
	if value != nil {
		return true
	}
	return false
}

func (rm *RewardManager) put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB) {
	accountdb.SetData(common.RewardStorageAddress, string(blockHash), transactionHash)
}

func (rm *RewardManager) blockRewards(height uint64) uint64 {
	if rm.noRewards && rm.noRewardsHeight <= height {
		return 0
	}
	return initialRewards >> (height / halveRewardsPeriod)
}

func (rm *RewardManager) userNodesRewards(height uint64) uint64 {
	rewards := rm.blockRewards(height)
	return rewards * userNodeWeight / totalNodeWeight
}

func (rm *RewardManager) daemonNodesRewards(height uint64) uint64 {
	rewards := rm.blockRewards(height)
	daemonNodeWeight := initialDaemonNodeWeight + height/adjustWeightPeriod*adjustWeight
	return rewards * daemonNodeWeight / totalNodeWeight
}

func (rm *RewardManager) minerNodesRewards(height uint64) uint64 {
	rewards := rm.blockRewards(height)
	if rewards == 0 {
		return 0
	}
	minerNodesWeight := initialMinerNodeWeight - height/adjustWeightPeriod*adjustWeight
	if minerNodesWeight > totalNodeWeight {
		return 0
	}
	return rewards * minerNodesWeight / totalNodeWeight
}

func (rm *RewardManager) reduceBlockRewards(height uint64) bool {
	if rm.noRewards {
		return false
	}
	rewards := rm.blockRewards(height)
	if rewards > rm.tokenLeft {
		rm.noRewards = true
		rm.noRewardsHeight = height
		return false
	}
	rm.tokenLeft -= rewards
	BlockChainImpl.LatestStateDB().SetData(common.RewardStorageAddress, tokenLeftKey, common.Uint64ToByte(rm.tokenLeft))
	return true
}

// CalculateCastorRewards Calculate castor's rewards in a block
func (rm *RewardManager) CalculateCastorRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * castorRewardsWeight / totalRewardsWeight
}

// CalculatePackedRewards Calculate castor's reword that packed a reward transaction
func (rm *RewardManager) CalculatePackedRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * packedRewardsWeight / totalRewardsWeight
}

// CalculateVerifyRewards Calculate verify-node's rewards in a block
func (rm *RewardManager) CalculateVerifyRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * verifyRewardsWeight / totalRewardsWeight
}

// CalculateGasFeeVerifyRewards Calculate verify-node's gas fee rewards
func (rm *RewardManager) CalculateGasFeeVerifyRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeVerifyRewardsWeight / gasFeeTotalRewardsWeight
}

// CalculateGasFeeCastorRewards Calculate castor's gas fee rewards
func (rm *RewardManager) CalculateGasFeeCastorRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeCastorRewardsWeight / gasFeeTotalRewardsWeight
}
