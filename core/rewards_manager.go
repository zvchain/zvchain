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
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"sync"
)

const (
	initialRewards     = 62 * common.ZVC         // the initial rewards of one block released
	halveRewardsPeriod = 30000000                // the period of halve block rewards
	halveRewardsTimes  = 3                       // the times of halve block rewards
	tokensOfMiners     = 3500000000 * common.ZVC // total amount of tokens belonging to miners
	noRewardsHeight    = 120000000
)

const (
	initialDaemonNodeWeight = 440      // initial daemon node weight of rewards
	initialMinerNodeWeight  = 6460     // initial miner node weight of rewards
	userNodeWeight          = 100      // initial user node weight of rewards
	totalNodeWeight         = 7000     // total weight of rewards
	adjustWeight            = 80       // weight of adjusted per period
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

// rewardManager manage the reward transactions
type rewardManager struct {
	lock sync.RWMutex
}

func NewRewardManager() *rewardManager {
	manager := &rewardManager{}
	return manager
}

func getRewardData(db types.AccountDBTS, key []byte) []byte {
	return db.GetDataSafe(rewardStoreAddr, key)
}

func setRewardData(db types.AccountDBTS, key, value []byte) {
	db.SetDataSafe(rewardStoreAddr, key, value)
}

func (rm *rewardManager) blockHasRewardTransaction(blockHashByte []byte) bool {
	accountDB, err := BlockChainImpl.LatestStateDB()
	if err != nil {
		log.DefaultLogger.Errorf("get lastdb failed,err = %v", err.Error())
		return false
	}
	return getRewardData(accountDB.AsAccountDBTS(), blockHashByte) != nil
}
func (rm *rewardManager) HasRewardedOfBlock(blockHash common.Hash, accountdb types.AccountDB) bool {
	value := getRewardData(accountdb.AsAccountDBTS(), blockHash.Bytes())
	return value != nil
}

func (rm *rewardManager) MarkBlockRewarded(blockHash common.Hash, transactionHash common.Hash, accountdb types.AccountDB) {
	setRewardData(accountdb.AsAccountDBTS(), blockHash.Bytes(), transactionHash.Bytes())
}

func (rm *rewardManager) GetRewardTransactionByBlockHash(blockHash common.Hash) *types.Transaction {
	accountDB, err := BlockChainImpl.LatestStateDB()
	if err != nil {
		log.DefaultLogger.Errorf("get lastdb failed,err = %v", err.Error())
		return nil
	}
	transactionHash := getRewardData(accountDB.AsAccountDBTS(), blockHash.Bytes())
	if transactionHash == nil {
		return nil
	}
	transaction := BlockChainImpl.GetTransactionByHash(true, false, common.BytesToHash(transactionHash))
	return transaction
}

// GenerateReward generate the reward transaction for the group who just validate a block
func (rm *rewardManager) GenerateReward(targetIds []int32, blockHash common.Hash, gSeed common.Hash, totalValue uint64, packFee uint64) (*types.Reward, *types.Transaction, error) {
	buffer := &bytes.Buffer{}
	// Write version
	buffer.WriteByte(rewardVersion)

	// Write groupId
	buffer.Write(gSeed.Bytes())
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
	return &types.Reward{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, Group: gSeed, TotalValue: totalValue}, transaction, nil
}

// ParseRewardTransaction parse a bonus transaction and  returns the group id, targetIds, block hash and transcation value
func (rm *rewardManager) ParseRewardTransaction(transaction *types.Transaction) (gSeed common.Hash, targets [][]byte, blockHash common.Hash, packFee *big.Int, err error) {
	reader := bytes.NewReader(transaction.ExtraData)
	gSeedBytes := make([]byte, common.HashLength)
	version, e := reader.ReadByte()
	if e != nil {
		err = e
		return
	}
	if version != rewardVersion {
		err = fmt.Errorf("reward version error")
		return
	}
	if _, e := reader.Read(gSeedBytes); e != nil {
		err = fmt.Errorf("read group seed error:%v", e)
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
		targetIdxs = append(targetIdxs, common.ByteToUInt16(idx))
	}

	gSeed = common.BytesToHash(gSeedBytes)
	group := GroupManagerImpl.GetGroupStoreReader().GetGroupBySeed(gSeed)
	if group == nil {
		err = fmt.Errorf("group is nil, gseed=%v", gSeed)
		return
	}
	ids := make([][]byte, 0)

	for _, idx := range targetIdxs {
		if idx > uint16(len(group.Members())) {
			err = fmt.Errorf("target index exceed: group size %v, index %v", len(group.Members()), idx)
			return
		}
		ids = append(ids, group.Members()[idx].ID())
	}

	blockHash = rm.parseRewardBlockHash(transaction)
	return gSeed, ids, blockHash, new(big.Int).SetUint64(common.ByteToUint64(pf)), nil
}

func (rm *rewardManager) parseRewardBlockHash(tx *types.Transaction) common.Hash {
	return common.BytesToHash(tx.Data)
}

func (rm *rewardManager) blockRewards(height uint64) uint64 {
	if height > noRewardsHeight {
		return 0
	}
	return initialRewards >> (height / halveRewardsPeriod)
}

func (rm *rewardManager) userNodesRewards(height uint64) uint64 {
	rewards := rm.blockRewards(height)
	if rewards == 0 {
		return 0
	}
	return rewards * userNodeWeight / totalNodeWeight
}

func (rm *rewardManager) daemonNodesRewards(height uint64) uint64 {
	rewards := rm.blockRewards(height)
	if rewards == 0 {
		return 0
	}
	daemonNodeWeight := initialDaemonNodeWeight + height/adjustWeightPeriod*adjustWeight
	return rewards * daemonNodeWeight / totalNodeWeight
}

func (rm *rewardManager) minerNodesRewards(height uint64) uint64 {
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

// CalculateCastorRewards Calculate castor's rewards in a block
func (rm *rewardManager) calculateCastorRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * castorRewardsWeight / totalRewardsWeight
}

// calculatePackedRewards Calculate castor's reword that packed a reward transaction
func (rm *rewardManager) calculatePackedRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * packedRewardsWeight / totalRewardsWeight
}

// calculateVerifyRewards Calculate verify-node's rewards in a block
func (rm *rewardManager) calculateVerifyRewards(height uint64) uint64 {
	minerNodesRewards := rm.minerNodesRewards(height)
	return minerNodesRewards * verifyRewardsWeight / totalRewardsWeight
}

// calculateGasFeeVerifyRewards Calculate verify-node's gas fee rewards
func (rm *rewardManager) calculateGasFeeVerifyRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeVerifyRewardsWeight / gasFeeTotalRewardsWeight
}

// calculateGasFeeCastorRewards Calculate castor's gas fee rewards
func (rm *rewardManager) calculateGasFeeCastorRewards(gasFee uint64) uint64 {
	return gasFee * gasFeeCastorRewardsWeight / gasFeeTotalRewardsWeight
}

func (rm *rewardManager) CalculateCastRewardShare(height uint64, gasFee uint64) *types.CastRewardShare {
	return &types.CastRewardShare{
		ForBlockProposal:   rm.calculateCastorRewards(height),
		ForBlockVerify:     rm.calculateVerifyRewards(height),
		ForRewardTxPacking: rm.calculatePackedRewards(height),
		FeeForProposer:     rm.calculateGasFeeCastorRewards(gasFee),
		FeeForVerifier:     rm.calculateGasFeeVerifyRewards(gasFee),
	}
}
