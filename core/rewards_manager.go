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
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/vm"
)

const (
	initialRewards     = 62 * common.TAS         // the initial rewards of one block released
	halveRewardsPeriod = 30000000                // the period of halve block rewards
	halveRewardsTimes  = 3                       // the times of halve block rewards
	tokensOfMiners     = 3500000000 * common.TAS // total amount of tokens belonging to miners
)

var tokenLeftKey = []byte("TokenLeft") // the key of token left stored in statedb

const (
	initialDaemonNodeWeight = 629      // initial daemon node weight of rewards
	initialMinerNodeWeight  = 9228     // initial miner node weight of rewards
	userNodeWeight          = 143      // initial user node weight of rewards
	totalNodeWeight         = 10000    // total weight of rewards
	adjustWeight            = 114      // weight of adjusted per period
	adjustWeightPeriod      = 10000000 // the period of adjusting weight
)

const (
	castorRewardsWeight = 8  // rewards weight of castor miner node
	packedRewardsWeight = 1  // rewards weight of miner packed reward-tx
	verifyRewardsWeight = 1  // rewards weight of verify miner node
	totalRewardsWeight  = 10 //total rewards weight
)

const (
	gasFeeCastorRewardsWeight = 9  // gas fee rewards weight of castor role
	gasFeeVerifyRewardsWeight = 1  // gas fee rewards weight of verify role
	gasFeeTotalRewardsWeight  = 10 // total rewards weight of gas fee
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
	return manager
}

func getRewardData(db vm.AccountDBTS, key []byte) []byte {
	return db.GetDataSafe(rewardStoreAddr, key)
}

func setRewardData(db vm.AccountDBTS, key, value []byte) {
	db.SetDataSafe(rewardStoreAddr, key, value)
}

func (rm *RewardManager) blockHasRewardTransaction(blockHashByte []byte) bool {
	return getRewardData(BlockChainImpl.LatestStateDB().AsAccountDBTS(), blockHashByte) != nil
}

func (rm *RewardManager) GetRewardTransactionByBlockHash(blockHash []byte) *types.Transaction {
	transactionHash := getRewardData(BlockChainImpl.LatestStateDB().AsAccountDBTS(), blockHash)
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
	transaction.GasLimit = types.NewBigInt(0)
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
	value := getRewardData(accountdb.AsAccountDBTS(), blockHash)
	if value != nil {
		return true
	}
	return false
}

func (rm *RewardManager) put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB) {
	setRewardData(accountdb.AsAccountDBTS(), blockHash, transactionHash)
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

func (rm *RewardManager) reduceBlockRewards(height uint64, accountDB *account.AccountDB) bool {
	if !rm.noRewards && rm.tokenLeft == 0 {
		value := getRewardData(BlockChainImpl.LatestStateDB().AsAccountDBTS(), tokenLeftKey)
		if value == nil {
			rm.tokenLeft = tokensOfMiners
		} else {
			rm.tokenLeft = common.ByteToUint64(value)
		}
	}
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
	setRewardData(accountDB.AsAccountDBTS(), tokenLeftKey, common.Uint64ToByte(rm.tokenLeft))
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
