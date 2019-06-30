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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/vm"
	"math/big"
	"sync"
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

const rewardVersion = 1

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
func (rm *RewardManager) GenerateReward(targetIds []int32, blockHash common.Hash, groupID []byte, totalValue uint64, packFee uint64) (*types.Reward, *types.Transaction, error) {
	buffer := &bytes.Buffer{}
	// Write version
	buffer.WriteByte(rewardVersion)

	// Write groupId
	buffer.Write(groupID)
	// pack fee
	buffer.Write(common.Uint64ToByte(packFee))

	if len(targetIds) == 0 {
		return nil, nil, errors.New("GenerateReward targetIds size 0")
	}

	// Write target indexes
	for _, idIdx := range targetIds {
		// Write the mem idx instead
		buffer.Write(common.UInt16ToByte(uint16(idIdx)))
	}

	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()

	transaction.Value = types.NewBigInt(totalValue / uint64(len(targetIds)))
	transaction.Type = types.TransactionTypeReward
	transaction.GasPrice = types.NewBigInt(0)
	transaction.GasLimit = types.NewBigInt(0)
	transaction.Hash = transaction.GenHash()
	return &types.Reward{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupID: groupID, TotalValue: totalValue}, transaction, nil
}

// ParseRewardTransaction parse a bonus transaction and  returns the group id, targetIds, block hash and transcation value
func (rm *RewardManager) ParseRewardTransaction(transaction *types.Transaction) (groupId []byte, targets [][]byte, blockHash common.Hash, packFee *big.Int, err error) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupID := make([]byte, common.GroupIDLength)
	version, e := reader.ReadByte()
	if e != nil {
		err = e
		return
	}
	if version != rewardVersion {
		err = fmt.Errorf("reward version error")
		return
	}
	if _, e := reader.Read(groupID); e != nil {
		err = fmt.Errorf("read group id error:%v", e)
		return
	}
	pf := make([]byte, 8)
	if _, e := reader.Read(pf); e != nil {
		err = fmt.Errorf("read pack fee error:%v", e)
		return
	}

	targetIdxs := make([]uint16, 0)
	idx := make([]byte, 2)

	for n, e := reader.Read(idx); n > 0 && e == nil; n, e = reader.Read(idx) {
		if e != nil {
			err = fmt.Errorf("read target idex error: %v", e)
			return
		}
		targetIdxs = append(targetIdxs, common.ByteToUInt16(idx))
	}

	group := GroupChainImpl.GetGroupByID(groupID)
	if group == nil {
		err = fmt.Errorf("group is nil, id=%v", common.ToHex(groupID))
		return
	}
	ids := make([][]byte, 0)

	for _, idx := range targetIdxs {
		if idx > uint16(len(group.Members)) {
			err = fmt.Errorf("target index exceed: group size %v, index %v", len(group.Members), idx)
			return
		}
		ids = append(ids, group.Members[idx])
	}

	blockHash = rm.parseRewardBlockHash(transaction)
	return groupID, ids, blockHash, new(big.Int).SetUint64(common.ByteToUint64(pf)), nil
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
	Logger.Debugf("tokenLeft:%v at %v", rm.tokenLeft, height)
	setRewardData(accountDB.AsAccountDBTS(), tokenLeftKey, common.Uint64ToByte(rm.tokenLeft))
	return true
}

// calculateCastorRewards Calculate castor's rewards in a block
func (rm *RewardManager) calculateCastorRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * castorRewardsWeight / totalRewardsWeight
}

// calculatePackedRewards Calculate castor's reword that packed a reward transaction
func (rm *RewardManager) calculatePackedRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * packedRewardsWeight / totalRewardsWeight
}

// calculateVerifyRewards Calculate verify-node's rewards in a block
func (rm *RewardManager) calculateVerifyRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * verifyRewardsWeight / totalRewardsWeight
}

// calculateGasFeeVerifyRewards Calculate verify-node's gas fee rewards
func (rm *RewardManager) calculateGasFeeVerifyRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeVerifyRewardsWeight / gasFeeTotalRewardsWeight
}

// calculateGasFeeCastorRewards Calculate castor's gas fee rewards
func (rm *RewardManager) calculateGasFeeCastorRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeCastorRewardsWeight / gasFeeTotalRewardsWeight
}

func (rm *RewardManager) CalculateCastRewardShare(height uint64, gasFee uint64) *types.CastRewardShare {
	return &types.CastRewardShare{
		ForBlockProposal:   rm.calculateCastorRewards(height),
		ForBlockVerify:     rm.calculateVerifyRewards(height),
		ForRewardTxPacking: rm.calculatePackedRewards(height),
		FeeForProposer:     rm.calculateGasFeeCastorRewards(gasFee),
		FeeForVerifier:     rm.calculateGasFeeVerifyRewards(gasFee),
	}
}
