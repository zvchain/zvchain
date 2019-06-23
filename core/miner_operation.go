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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/vm"
	"math/big"
)

type mOperation interface {
	ParseTransaction() error
	Validate() error
	Operation() error
}

type baseOperation struct {
	accountdb vm.AccountDB
	tx        *types.Transaction
	height    uint64
}

func newBaseOperation(db vm.AccountDB, tx *types.Transaction, height uint64) *baseOperation {
	return &baseOperation{
		accountdb: db,
		tx:        tx,
		height:    height,
	}
}

func newOperation(db vm.AccountDB, tx *types.Transaction, height uint64) mOperation {
	baseOp := newBaseOperation(db, tx, height)
	var operation mOperation
	switch tx.Type {
	case types.TransactionTypeMinerApply:
		operation = &stakeAdder{baseOperation: baseOp}
	case types.TransactionTypeMinerAbort:
		operation = &minerAborter{baseOperation: baseOp}
	case types.TransactionTypeMinerCancelStake:
		operation = &stakeCanceler{baseOperation: baseOp}
	case types.TransactionTypeMinerRefund:
		operation = &stakeRefunder{baseOperation: baseOp}
	default:

	}
	return operation
}

type stakeAdder struct {
	*baseOperation
	miner     *types.Miner
	value     uint64
	addSource common.Address
	addTarget common.Address
}

func (op *stakeAdder) Validate() error {
	if op.tx.Data == nil {
		return fmt.Errorf("tx data is nil:%v", op.tx.Hash.Hex())
	}
	if op.tx.Target == nil {
		return fmt.Errorf("tx target is nil:%v", op.tx.Hash.Hex())
	}

	if !canTransfer(op.accountdb, *op.tx.Source, op.tx.Value.Value(), new(big.Int).SetUint64(0)) {
		return fmt.Errorf("balance not enough")
	}
	return nil
}

func (op *stakeAdder) ParseTransaction() error {
	m, err := tx2Miner(op.tx)
	if err != nil {
		return err
	}
	op.miner = m
	op.addSource = *op.tx.Source
	op.addTarget = *op.tx.Target
	op.value = op.tx.Value.Uint64()
	return nil
}

func (op *stakeAdder) Operation() error {
	miner := op.miner
	targetMiner, err := op.getMiner(op.addTarget, op.miner.Type)
	if err != nil {
		return err
	}

	// Already exists
	if targetMiner != nil {
		if targetMiner.Stake+miner.Stake < targetMiner.Stake {
			return fmt.Errorf("stake overflow:%v %v", targetMiner.Stake, miner.Stake)
		}
		miner = mergeMinerInfo(targetMiner, miner)
	} else {
		miner.ApplyHeight = op.height
	}
	if checkCanActivate(miner) {
		miner.Status = types.MinerStatusNormal

		// Add to pool so that the miner can start working
		op.addToPool(op.addTarget, op.miner.Type, miner.Stake)
	}
	// Save miner
	if err := op.setMiner(miner); err != nil {
		return err
	}

	targetDetailKey := getDetailKey(prefixStakeFrom, op.addSource, op.miner.Type, types.Staked)
	targetDetail, err := op.getDetail(op.addTarget, targetDetailKey)
	if err != nil {
		return err
	}

	if targetDetail != nil {
		if targetDetail.Value+op.value < targetDetail.Value {
			return fmt.Errorf("stake detail value overflow:%v %v", targetDetail.Value, op.value)
		}
		targetDetail.Value += op.value
	} else {
		targetDetail = &stakeDetail{
			Value: op.value,
		}
	}
	// Set detail of the targetMiner: who stakes for me
	targetDetail.Height = op.height
	if err := op.setDetail(op.addTarget, targetDetailKey, targetDetail); err != nil {
		return err
	}

	sourceDetailKey := getDetailKey(prefixStakeTo, op.addTarget, op.miner.Type, types.Staked)
	sourceDetail, err := op.getDetail(op.addSource, sourceDetailKey)
	if err != nil {
		return err
	}

	if sourceDetail != nil {
		if sourceDetail.Value+op.value < sourceDetail.Value {
			return fmt.Errorf("stake detail value overflow:%v %v", targetDetail.Value, op.value)
		}
		sourceDetail.Value += op.value
	} else {
		sourceDetail = &stakeDetail{
			Value: op.value,
		}
	}
	// Set detail of the staker: whom i stake for
	sourceDetail.Height = op.height
	if err := op.setDetail(op.addSource, sourceDetailKey, sourceDetail); err != nil {
		return err
	}

	op.accountdb.SubBalance(op.addSource, new(big.Int).SetUint64(op.value))

	return nil

}

type minerAborter struct {
	*baseOperation
	typ  byte
	addr common.Address
}

func (op *minerAborter) ParseTransaction() error {
	op.typ = op.tx.Data[0]
	op.addr = *op.tx.Source
	return nil
}

func (op *minerAborter) Validate() error {
	if op.tx.Data == nil {
		return fmt.Errorf("tx data is nil")
	}
	if len(op.tx.Data) != 1 {
		return fmt.Errorf("tx data length should be 1")
	}
	return nil
}

func (op *minerAborter) Operation() error {
	miner, err := op.getMiner(op.addr, op.typ)
	if err != nil {
		return err
	}
	miner.AbortHeight = op.height
	miner.Status = types.MinerStatusAbort
	if err := op.setMiner(miner); err != nil {
		return err
	}
	op.removeFromPool(op.addr, op.typ, miner.Stake)
	return nil
}

type stakeCanceler struct {
	*baseOperation
	cancelTarget common.Address
	cancelSource common.Address
	value        uint64
	typ          byte
}

func (op *stakeCanceler) ParseTransaction() error {
	op.typ = op.tx.Data[0]
	op.cancelTarget = *op.tx.Target
	op.cancelSource = *op.tx.Source
	op.value = op.tx.Value.Uint64()
	return nil
}

func (op *stakeCanceler) Validate() error {
	if len(op.tx.Data) != 1 {
		return fmt.Errorf("tx data lenght should be 1")
	}
	return nil
}

func (op *stakeCanceler) Operation() error {
	miner, err := op.getMiner(op.cancelTarget, op.typ)
	if err != nil {
		return err
	}
	if miner == nil {
		return fmt.Errorf("miner info not found")
	}

	targetStakedDetailKey := getDetailKey(prefixStakeFrom, op.cancelSource, op.typ, types.Staked)
	targetStakedDetail, err := op.getDetail(op.cancelTarget, targetStakedDetailKey)
	if err != nil {
		return err
	}
	if targetStakedDetail == nil {
		return fmt.Errorf("no detail data")
	}

	if targetStakedDetail.Value > miner.Stake {
		panic(fmt.Errorf("detail stake more than total stake of the miner:%v %v %x", targetStakedDetail.Value, miner.Stake, miner.ID))
	}

	if targetStakedDetail.Value < op.value {
		return fmt.Errorf("detail stake less than cancel amount:%v %v", targetStakedDetail.Value, op.value)
	}

	// Decrease the stake from the Source
	// Removal will be taken if decreasing to zero
	targetStakedDetail.Value -= op.value
	targetStakedDetail.Height = op.height
	if targetStakedDetail.Value == 0 {
		op.removeDetail(op.cancelTarget, targetStakedDetailKey)
	} else {
		if err := op.setDetail(op.cancelTarget, targetStakedDetailKey, targetStakedDetail); err != nil {
			return err
		}
	}
	targetFrozenDetailKey := getDetailKey(prefixStakeFrom, op.cancelSource, op.typ, types.StakeFrozen)
	targetFrozenDetail, err := op.getDetail(op.cancelTarget, targetFrozenDetailKey)
	if err != nil {
		return err
	}
	if targetFrozenDetail == nil {
		targetFrozenDetail = &stakeDetail{
			Value: op.value,
		}
	} else {
		targetFrozenDetail.Value += op.value
	}
	targetFrozenDetail.Height = op.height
	// Update the frozen detail of target
	if err := op.setDetail(op.cancelTarget, targetFrozenDetailKey, targetFrozenDetail); err != nil {
		return err
	}

	sourceStakedDetailKey := getDetailKey(prefixStakeTo, op.cancelSource, op.typ, types.Staked)
	sourceStakedDetail, err := op.getDetail(op.cancelSource, sourceStakedDetailKey)
	if err != nil {
		return err
	}
	if sourceStakedDetail == nil {
		return fmt.Errorf("no detail data")
	}

	if sourceStakedDetail.Value > miner.Stake {
		panic(fmt.Errorf("detail stake more than total stake of the miner:%v %v %x", targetStakedDetail.Value, miner.Stake, miner.ID))
	}

	if sourceStakedDetail.Value < op.value {
		return fmt.Errorf("detail stake less than cancel amount:%v %v", targetStakedDetail.Value, op.value)
	}

	// Decrease the stake to the Target
	// Removal will be taken if decreasing to zero
	sourceStakedDetail.Value -= op.value
	sourceStakedDetail.Height = op.height
	if sourceStakedDetail.Value == 0 {
		op.removeDetail(op.cancelSource, sourceStakedDetailKey)
	} else {
		if err := op.setDetail(op.cancelSource, sourceStakedDetailKey, sourceStakedDetail); err != nil {
			return err
		}
	}
	sourceFrozenDetailKey := getDetailKey(prefixStakeTo, op.cancelTarget, op.typ, types.StakeFrozen)
	sourceFrozenDetail, err := op.getDetail(op.cancelSource, sourceFrozenDetailKey)
	if err != nil {
		return err
	}
	if sourceFrozenDetail == nil {
		sourceFrozenDetail = &stakeDetail{
			Value: op.value,
		}
	} else {
		sourceFrozenDetail.Value += op.value
	}
	sourceFrozenDetail.Height = op.height
	// Update the frozen detail of Source
	if err := op.setDetail(op.cancelSource, sourceFrozenDetailKey, sourceFrozenDetail); err != nil {
		return err
	}

	if miner.Stake < op.value {
		return fmt.Errorf("miner stake not enough:%v %v", miner.Stake, op.value)
	}
	// Update miner stake
	miner.Stake -= op.value
	if checkCanInActivate(miner) {
		if op.typ == types.MinerTypeVerify && miner.Status == types.MinerStatusNormal {
			if GroupChainImpl.WhetherMemberInActiveGroup(op.cancelTarget.Bytes(), op.height) {
				return fmt.Errorf("target miner in active groups, cannot abort")
			}
		}
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = op.height
		op.removeFromPool(op.cancelTarget, op.typ, miner.Stake)
	}
	if err := op.setMiner(miner); err != nil {
		return err
	}
	return nil

}

type stakeRefunder struct {
	*baseOperation
	typ          byte
	refundTarget common.Address
	refundSource common.Address
}

func (op *stakeRefunder) ParseTransaction() error {
	op.typ = op.tx.Data[0]
	op.refundTarget = *op.tx.Target
	op.refundSource = *op.tx.Source
	return nil
}

func (op *stakeRefunder) Validate() error {
	if len(op.tx.Data) != 1 {
		return fmt.Errorf("tx data lenght should be 1")
	}
	return nil
}

func (op *stakeRefunder) Operation() error {
	targetMiner, err := op.getMiner(op.refundTarget, op.typ)
	if err != nil {
		return err
	}
	targetFrozenDetailKey := getDetailKey(prefixStakeFrom, op.refundSource, op.typ, types.StakeFrozen)
	targetFrozenDetail, err := op.getDetail(op.refundTarget, targetFrozenDetailKey)
	if err != nil {
		return err
	}
	if targetFrozenDetail == nil {
		return fmt.Errorf("target no frozen detail")
	}
	sourceFrozenDetailKey := getDetailKey(prefixStakeTo, op.refundTarget, op.typ, types.StakeFrozen)
	sourceFrozenDetail, err := op.getDetail(op.refundSource, sourceFrozenDetailKey)
	if err != nil {
		return err
	}
	if targetFrozenDetail == nil {
		return fmt.Errorf("source no frozen detail")
	}
	if targetFrozenDetail.Value != sourceFrozenDetail.Value {
		return fmt.Errorf("source frozen value not equal to target:%v %v", sourceFrozenDetail.Value, targetFrozenDetail.Value)
	}
	if op.typ == types.MinerTypeProposal {
		if !(op.height > targetFrozenDetail.Height+10 || (targetMiner.Status == types.MinerStatusAbort && op.height > targetMiner.AbortHeight+10)) {
			return fmt.Errorf("refund must happen after 10 blocks height since canceled")
		}
	} else { // todo what about verifier

	}

	op.removeDetail(op.refundTarget, targetFrozenDetailKey)
	op.removeDetail(op.refundSource, sourceFrozenDetailKey)

	op.accountdb.AddBalance(op.refundSource, new(big.Int).SetUint64(targetFrozenDetail.Value))
	return nil

}
