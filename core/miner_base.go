//   Copyright (C) 2019 ZVChain
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
	"fmt"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
	"math/big"
)

var (
	prefixMiner               = []byte("minfo")
	prefixDetail              = []byte("dt")
	prefixPoolProposal        = []byte("p")
	prefixPoolVerifier        = []byte("v")
	keyPoolProposalTotalStake = []byte("totalstake")
)

const (
	MinMinerStake             = 500 * common.TAS // minimal token of miner can stake
	MaxMinerStakeAdjustPeriod = 5000000          // maximal token of miner can stake
	initialMinerNodesAmount   = 200              // The number initial of miner nodes envisioned
	MoreMinerNodesPerHalfYear = 12               // The number of increasing nodes per half year
	initialTokenReleased      = 500000000        // The initial amount of tokens released
	tokenReleasedPerHalfYear  = 400000000        //  The amount of tokens released per half year
	stakeAdjustTimes          = 24               // stake adjust times
)

// minimumStake shows miner can stake the min value
func minimumStake() uint64 {
	return MinMinerStake
}

// maximumStake shows miner can stake the max value
func maximumStake(height uint64) uint64 {
	period := height / MaxMinerStakeAdjustPeriod
	if period > stakeAdjustTimes {
		period = stakeAdjustTimes
	}
	nodeAmount := initialMinerNodesAmount + period*MoreMinerNodesPerHalfYear
	return tokenReleased(height) / nodeAmount * common.TAS
}

func tokenReleased(height uint64) uint64 {
	adjustTimes := height / MaxMinerStakeAdjustPeriod
	if adjustTimes > stakeAdjustTimes {
		adjustTimes = stakeAdjustTimes
	}

	var released uint64 = initialTokenReleased
	for i := uint64(0); i < adjustTimes; i++ {
		halveTimes := i * MaxMinerStakeAdjustPeriod / halveRewardsPeriod
		if halveTimes > halveRewardsTimes {
			halveTimes = halveRewardsTimes
		}
		released += tokenReleasedPerHalfYear >> halveTimes
	}
	return released
}

// Special account address
// Need to access by AccountDBTS for concurrent situations
var (
	minerPoolAddr   = common.BigToAddress(big.NewInt(1)) // The Address storing total stakes of each roles and addresses of all active nodes
	rewardStoreAddr = common.BigToAddress(big.NewInt(2)) // The Address storing the block hash corresponding to the reward transaction
)

type stakeDetail struct {
	Value  uint64 // Stake operation amount
	Height uint64 // Operation height
}

func getDetailKey(address common.Address, typ types.MinerType, status types.StakeStatus) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(prefixDetail)
	buf.Write(address.Bytes())
	buf.WriteByte(byte(typ))
	buf.WriteByte(byte(status))
	return buf.Bytes()
}

func parseDetailKey(key []byte) (common.Address, types.MinerType, types.StakeStatus) {
	reader := bytes.NewReader(key)

	detail := make([]byte, len(prefixDetail))
	n, err := reader.Read(detail)
	if err != nil || n != len(prefixDetail) {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	addrBytes := make([]byte, len(common.Address{}))
	n, err = reader.Read(addrBytes)
	if err != nil || n != len(addrBytes) {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	mtByte, err := reader.ReadByte()
	if err != nil {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	stByte, err := reader.ReadByte()
	if err != nil {
		panic(fmt.Errorf("parse detail key error:%v", err))
	}
	return common.BytesToAddress(addrBytes), types.MinerType(mtByte), types.StakeStatus(stByte)
}

func setPks(miner *types.Miner, pks *types.MinerPks) *types.Miner {
	if len(pks.Pk) > 0 {
		miner.PublicKey = pks.Pk
	}
	if len(pks.VrfPk) > 0 {
		miner.VrfPublicKey = pks.VrfPk
	}

	return miner
}

// checkCanActivate if status can be set to types.MinerStatusActive
func checkCanActivate(miner *types.Miner, height uint64) bool {
	// pks not completed
	if !miner.PksCompleted() {
		return false
	}
	// If the stake up to the lower bound, then activate the miner
	return checkLowerBound(miner, height)
}

func checkUpperBound(miner *types.Miner, height uint64) bool {
	return miner.Stake <= maximumStake(height)
}

func checkLowerBound(miner *types.Miner, height uint64) bool {
	return miner.Stake >= minimumStake()
}

func getMinerKey(typ types.MinerType) []byte {
	buf := bytes.NewBuffer(prefixMiner)
	buf.WriteByte(byte(typ))
	return buf.Bytes()
}

func getPoolKey(prefix []byte, address common.Address) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func getMiner(db vm.AccountDB, address common.Address, mType types.MinerType) (*types.Miner, error) {
	data := db.GetData(address, getMinerKey(mType))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		err := msgpack.Unmarshal(data, &miner)
		if err != nil {
			return nil, err
		}
		return &miner, nil
	}
	return nil, nil
}

func parseDetail(value []byte) (*stakeDetail, error) {
	var detail stakeDetail
	err := msgpack.Unmarshal(value, &detail)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func getDetail(db vm.AccountDB, address common.Address, detailKey []byte) (*stakeDetail, error) {
	data := db.GetData(address, detailKey)
	if data != nil && len(data) > 0 {
		return parseDetail(data)
	}
	return nil, nil
}

func getProposalTotalStake(db vm.AccountDBTS) uint64 {
	totalStakeBytes := db.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
	totalStake := uint64(0)
	if len(totalStakeBytes) > 0 {
		totalStake = common.ByteToUInt64(totalStakeBytes)
	}
	return totalStake
}

type baseOperation struct {
	minerType types.MinerType
	accountDB vm.AccountDB
	minerPool vm.AccountDBTS
	msg       vm.MinerOperationMessage
	height    uint64
}

func newBaseOperation(db vm.AccountDB, msg vm.MinerOperationMessage, height uint64) *baseOperation {
	return &baseOperation{
		accountDB: db,
		minerPool: db.AsAccountDBTS(),
		msg:       msg,
		height:    height,
	}
}

func (op *baseOperation) opProposalRole() bool {
	return types.IsProposalRole(op.minerType)
}
func (op *baseOperation) opVerifyRole() bool {
	return types.IsVerifyRole(op.minerType)
}

func (op *baseOperation) addToPool(address common.Address, addStake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		op.addProposalTotalStake(addStake)
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.SetDataSafe(minerPoolAddr, key, []byte{1})
}

func (op *baseOperation) addProposalTotalStake(addStake uint64) {
	totalStake := getProposalTotalStake(op.minerPool)
	// Must not happen
	if addStake+totalStake < totalStake {
		panic(fmt.Errorf("total stake overflow:%v %v", addStake, totalStake))
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(addStake+totalStake))
}

func (op *baseOperation) subProposalTotalStake(subStake uint64) {
	totalStake := getProposalTotalStake(op.minerPool)
	// Must not happen
	if totalStake < subStake {
		panic("total stake less than sub stake")
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-subStake))
}

func (op *baseOperation) removeFromPool(address common.Address, stake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		totalStakeBytes := op.minerPool.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		if totalStake < stake {
			panic(fmt.Errorf("totalStake less than stake: %v %v", totalStake, stake))
		}
		op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-stake))
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.RemoveDataSafe(minerPoolAddr, key)
}

func (op *baseOperation) getDetail(address common.Address, detailKey []byte) (*stakeDetail, error) {
	return getDetail(op.accountDB, address, detailKey)
}

func (op *baseOperation) setDetail(address common.Address, detailKey []byte, sd *stakeDetail) error {
	bs, err := msgpack.Marshal(sd)
	if err != nil {
		return err
	}
	op.accountDB.SetData(address, detailKey, bs)
	return nil
}

func (op *baseOperation) removeDetail(address common.Address, detailKey []byte) {
	op.accountDB.RemoveData(address, detailKey)
}

func (op *baseOperation) getMiner(address common.Address) (*types.Miner, error) {
	return getMiner(op.accountDB, address, op.minerType)
}

func (op *baseOperation) setMiner(miner *types.Miner) error {
	bs, err := msgpack.Marshal(miner)
	if err != nil {
		return err
	}
	op.accountDB.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)
	return nil
}
