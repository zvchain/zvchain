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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

// Duration definition related to miner operation
const (
	oneHourBlocks = 86400 / 3 / 24   // Blocks generated in one hour on average, used when status transforms from Frozen to Prepare
	oneDayBlocks  = 86400 / 3        // Blocks generated in one day on average
	twoDayBlocks  = 2 * oneDayBlocks // Blocks generated in two days on average, used when executes the miner refund
)

// mOperation define some functions on miner operation
// Used when executes the miner related transactions or stake operations from contract
// Different from the stateTransition, it doesn't take care of the gas and only focus on the operation
// In mostly case, some functions can be reused in stateTransition
type mOperation interface {
	Validate() error         // Validate the input args
	ParseTransaction() error // Parse the input transaction
	Operation() error        // Do the operation
}

type sysMinerOpType int8

// newOperation creates the mOperation instance base on msg type
func newOperation(db types.AccountDB, msg types.MinerOperationMessage, height uint64) mOperation {
	baseOp := newBaseOperation(db, msg, height)
	var operation mOperation

	switch msg.OpType() {
	case types.TransactionTypeStakeAdd:
		operation = &stakeAddOp{baseOperation: baseOp}
	case types.TransactionTypeMinerAbort:
		operation = &minerAbortOp{baseOperation: baseOp}
	case types.TransactionTypeStakeReduce:
		operation = &stakeReduceOp{baseOperation: baseOp}
	case types.TransactionTypeStakeRefund:
		operation = &stakeRefundOp{baseOperation: baseOp}
	default:
		operation = &unSupported{typ: msg.OpType()}
	}

	return operation
}

// stakeAddOp is for the stake add operation, miner can add stake for himself or others
type stakeAddOp struct {
	*baseOperation
	minerPks  *types.MinerPks
	value     uint64
	addSource common.Address
	addTarget common.Address
}

func (op *stakeAddOp) Validate() error {
	if len(op.msg.Payload()) == 0 {
		return fmt.Errorf("payload length error")
	}
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	if op.msg.Amount() == nil {
		return fmt.Errorf("amount is nil")
	}
	if !op.msg.Amount().IsUint64() {
		return fmt.Errorf("amount type not uint64")
	}
	return nil
}

func (op *stakeAddOp) ParseTransaction() error {
	m, err := types.DecodePayload(op.msg.Payload())
	if err != nil {
		return err
	}
	op.minerPks = m
	op.addSource = *op.msg.Operator()
	op.addTarget = *op.msg.OpTarget()

	// Only when stakes for self can specify the pks for security concern
	// Otherwise, bad guys can change somebody else's pks with low cost to make the victim cannot work correctly
	if len(m.Pk) > 0 && !bytes.Equal(op.addSource.Bytes(), op.addTarget.Bytes()) {
		return fmt.Errorf("cann't specify target pubkeys when stakes for others")
	}
	op.value = op.msg.Amount().Uint64()
	op.minerType = m.MType
	return nil
}

func (op *stakeAddOp) Operation() error {
	var add = false

	if op.addSource != op.addTarget && op.minerType == types.MinerTypeVerify {
		return fmt.Errorf("could not stake to other's verify node")
	}
	// Check balance
	amount := new(big.Int).SetUint64(op.value)
	if needTransfer(amount) {
		if !op.accountDB.CanTransfer(op.addSource, amount) {
			return fmt.Errorf("balance not enough")
		}
		// Sub the balance of source account
		op.accountDB.SubBalance(op.addSource, amount)
	}
	targetMiner, err := op.getMiner(op.addTarget)
	if err != nil {
		return err
	}

	// Already exists
	if targetMiner != nil {
		if targetMiner.IsFrozen() { // Frozen miner must abort first
			return fmt.Errorf("miner is frozen, cannot add stake")
		}
		if targetMiner.Stake+op.value < targetMiner.Stake {
			return fmt.Errorf("stake overflow:%v %v", targetMiner.Stake, op.value)
		}
		targetMiner.Stake += op.value
	} else {
		targetMiner = &types.Miner{
			ID:          op.addTarget.Bytes(),
			Stake:       op.value,
			ApplyHeight: op.height,
			Type:        op.minerType,
			Status:      types.MinerStatusPrepare,
		}
	}
	setPks(targetMiner, op.minerPks)
	// Check the upper bound of stake
	if !checkUpperBound(targetMiner, op.height) {
		return fmt.Errorf("stake more than upper bound:%v", targetMiner.Stake)
	}

	if targetMiner.IsActive() {
		// Update proposal total stake
		if op.opProposalRole() {
			op.addProposalTotalStake(op.value)
		}
	} else if checkCanActivate(targetMiner, op.height) { // Check if to active the miner
		targetMiner.UpdateStatus(types.MinerStatusActive, op.height)
		// Add to pool so that the miner can start working
		op.addToPool(op.addTarget, targetMiner.Stake)
		if op.opProposalRole() {
			add = true
		}
	}
	// Save miner
	if err := op.setMiner(targetMiner); err != nil {
		return err
	}

	// Set detail of the target account: who stakes from me
	detailKey := getDetailKey(op.addSource, op.minerType, types.Staked)
	detail, err := op.getDetail(op.addTarget, detailKey)
	if err != nil {
		return fmt.Errorf("get target detail error:%v", err)
	}

	if detail != nil {
		if detail.Value+op.value < detail.Value {
			return fmt.Errorf("stake detail value overflow:%v %v", detail.Value, op.value)
		}
		detail.Value += op.value
	} else {
		detail = &stakeDetail{
			Value: op.value,
		}
	}
	// Update height
	detail.Height = op.height
	if err := op.setDetail(op.addTarget, detailKey, detail); err != nil {
		return err
	}

	if add && MinerManagerImpl != nil {
		// Inform added proposer address to minerManager
		MinerManagerImpl.proposalAddCh <- op.addTarget
	}
	return nil

}

// minerAbortOp abort the miner, which can cause miner status transfer to Prepare
// and quit mining
type minerAbortOp struct {
	*baseOperation
	addr common.Address
}

func (op *minerAbortOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.addr = *op.msg.Operator()
	return nil
}

// this function will not be called, Because the validate is only valid on smart contract. And minerAbortOp command is
// not available on smart contract.
func (op *minerAbortOp) Validate() error {
	return nil
}

func (op *minerAbortOp) Operation() error {
	var remove = false
	miner, err := op.getMiner(op.addr)
	if err != nil {
		return err
	}
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	if miner.IsPrepare() {
		return fmt.Errorf("already in prepare status")
	}
	// Frozen miner must wait for 1 hour after frozen
	if miner.IsFrozen() && op.height <= miner.StatusUpdateHeight+oneHourBlocks {
		return fmt.Errorf("frozen miner can't abort less than 1 hour since frozen")
	}
	// Remove from pool if active
	if miner.IsActive() {
		op.removeFromPool(op.addr, miner.Stake)
		if op.opProposalRole() {
			remove = true
		}
	}

	// Update the miner status
	miner.UpdateStatus(types.MinerStatusPrepare, op.height)
	if err := op.setMiner(miner); err != nil {
		return err
	}
	if remove && MinerManagerImpl != nil {
		// Informs MinerManager the removal address
		MinerManagerImpl.proposalRemoveCh <- op.addr
	}

	return nil
}

// stakeReduceOp is for stake reduce operation
type stakeReduceOp struct {
	*baseOperation
	cancelTarget common.Address
	cancelSource common.Address
	value        uint64
}

func (op *stakeReduceOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.cancelTarget = *op.msg.OpTarget()
	op.cancelSource = *op.msg.Operator()
	op.value = op.msg.Amount().Uint64()
	return nil
}

func (op *stakeReduceOp) Validate() error {
	if len(op.msg.Payload()) != 1 {
		return fmt.Errorf("msg payload length error")
	}
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	if op.msg.Amount() == nil {
		return fmt.Errorf("amount is nil")
	}
	if !op.msg.Amount().IsUint64() {
		return fmt.Errorf("amount type not uint64")
	}
	return nil
}

func (op *stakeReduceOp) checkCanReduce(miner *types.Miner) error {
	if miner.IsFrozen() {
		return fmt.Errorf("frozen miner must abort first")
	} else if miner.IsActive() {
		if !checkLowerBound(miner, op.height) {
			return fmt.Errorf("active miner cann't reduce stake to below bound")
		}
	} else if miner.IsPrepare() {
		if op.opVerifyRole() && GroupManagerImpl.GetGroupStoreReader().MinerLiveGroupCount(op.cancelTarget, op.height) > 0 {
			return fmt.Errorf("miner still in active groups, cannot reduce stake")
		}
	} else {
		return fmt.Errorf("unkown miner roles %v", miner.Type)
	}
	return nil
}

func (op *stakeReduceOp) Operation() error {
	miner, err := op.getMiner(op.cancelTarget)
	if err != nil {
		return err
	}
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	if miner.Stake < op.value {
		return fmt.Errorf("miner stake not enough:%v %v", miner.Stake, op.value)
	}
	originStake := miner.Stake
	// Update miner stake
	miner.Stake -= op.value

	// Check if can do the reduce operation
	if err := op.checkCanReduce(miner); err != nil {
		return err
	}

	// Sub the corresponding total stake of the proposals
	if miner.IsActive() && op.opProposalRole() {
		op.subProposalTotalStake(op.value)
	}
	if err := op.setMiner(miner); err != nil {
		return err
	}

	// Get Target account detail: staked-detail of who stakes for me
	stakedDetailKey := getDetailKey(op.cancelSource, op.minerType, types.Staked)
	stakedDetail, err := op.getDetail(op.cancelTarget, stakedDetailKey)
	if err != nil {
		return err
	}
	if stakedDetail == nil {
		return fmt.Errorf("target account has no staked detail data")
	}

	// Must not happened
	if stakedDetail.Value > originStake {
		panic(fmt.Errorf("detail stake more than total stake of the miner:%v %v %x", stakedDetail.Value, originStake, miner.ID))
	}

	if stakedDetail.Value < op.value {
		return fmt.Errorf("detail stake less than cancel amount:%v %v", stakedDetail.Value, op.value)
	}

	// Decrease the stake of the staked-detail
	// Removal will be taken if decreasing to zero
	stakedDetail.Value -= op.value
	stakedDetail.Height = op.height
	if stakedDetail.Value == 0 {
		op.removeDetail(op.cancelTarget, stakedDetailKey)
	} else {
		if err := op.setDetail(op.cancelTarget, stakedDetailKey, stakedDetail); err != nil {
			return err
		}
	}
	// Get Target account detail: frozen-detail of who stake for me
	frozenDetailKey := getDetailKey(op.cancelSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := op.getDetail(op.cancelTarget, frozenDetailKey)
	if err != nil {
		return err
	}
	if frozenDetail == nil {
		frozenDetail = &stakeDetail{
			Value: op.value,
		}
	} else {
		// Accumulate the frozen value
		frozenDetail.Value += op.value
	}
	frozenDetail.Height = op.height
	// Update the frozen detail of target
	if err := op.setDetail(op.cancelTarget, frozenDetailKey, frozenDetail); err != nil {
		return err
	}
	return nil
}

// stakeRefundOp is for stake refund operation, it only happens after stake-reduce ops
type stakeRefundOp struct {
	*baseOperation
	refundTarget common.Address
	refundSource common.Address
}

func (op *stakeRefundOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.refundTarget = *op.msg.OpTarget()
	op.refundSource = *op.msg.Operator()
	return nil
}

func (op *stakeRefundOp) Validate() error {
	if len(op.msg.Payload()) != 1 {
		return fmt.Errorf("msg payload length error")
	}
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	return nil
}

func (op *stakeRefundOp) Operation() error {
	// Get the detail in target account: frozen-detail of the source
	frozenDetailKey := getDetailKey(op.refundSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := op.getDetail(op.refundTarget, frozenDetailKey)
	if err != nil {
		return err
	}
	if frozenDetail == nil {
		return fmt.Errorf("target has no frozen detail")
	}
	// Check reduce-height
	if op.height <= frozenDetail.Height+twoDayBlocks {
		return fmt.Errorf("refund cann't happen util 2days after last reduce")
	}

	// Remove frozen data
	op.removeDetail(op.refundTarget, frozenDetailKey)

	// Restore the balance
	op.accountDB.AddBalance(op.refundSource, new(big.Int).SetUint64(frozenDetail.Value))
	return nil

}

// minerFreezeOp freeze the miner, which can cause miner status transfer to Frozen
// and quit mining.
// It was called by the group-create routine when the miner didn't participate in the process completely
type minerFreezeOp struct {
	*baseOperation
	addr common.Address
}

func (op *minerFreezeOp) ParseTransaction() error {
	return nil
}

func (op *minerFreezeOp) Validate() error {
	return nil
}

func (op *minerFreezeOp) Operation() error {
	if !op.opVerifyRole() {
		return fmt.Errorf("not operates a verifier:%v", op.addr.Hex())
	}
	miner, err := op.getMiner(op.addr)
	if err != nil {
		return err
	}
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	if miner.IsFrozen() {
		return fmt.Errorf("already in forzen status")
	}
	if !miner.IsVerifyRole() {
		return fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID))
	}

	// Remove from pool if active
	if miner.IsActive() {
		op.removeFromPool(op.addr, miner.Stake)
	}

	// Update the miner status
	miner.UpdateStatus(types.MinerStatusFrozen, op.height)
	if err := op.setMiner(miner); err != nil {
		return err
	}

	return nil
}

type minerPenaltyOp struct {
	*baseOperation
	targets []common.Address
	rewards []common.Address
	value   uint64
}

func (op *minerPenaltyOp) ParseTransaction() error {
	return nil
}

func (op *minerPenaltyOp) Validate() error {
	return nil
}

func (op *minerPenaltyOp) Operation() error {
	if !op.opVerifyRole() {
		return fmt.Errorf("not operates verifiers")
	}
	// Firstly, frozen the targets
	for _, addr := range op.targets {
		miner, err := op.getMiner(addr)
		if err != nil {
			return err
		}
		if miner == nil {
			return fmt.Errorf("no miner info")
		}
		if !miner.IsVerifyRole() {
			return fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID))
		}

		// Remove from pool if active
		if miner.IsActive() {
			op.removeFromPool(addr, miner.Stake)
		}
		// Must not happen
		if miner.Stake < op.value {
			panic(fmt.Errorf("stake less than punish value:%v %v of %v", miner.Stake, op.value, addr.Hex()))
		}

		// Sub total stake and update the miner status
		miner.Stake -= op.value
		miner.UpdateStatus(types.MinerStatusFrozen, op.height)
		if err := op.setMiner(miner); err != nil {
			return err
		}
		// Add punishment detail
		punishmentKey := getDetailKey(punishmentDetailAddr, op.minerType, types.StakePunishment)
		punishmentDetail, err := op.getDetail(addr, punishmentKey)
		if err != nil {
			return err
		}
		if punishmentDetail == nil {
			punishmentDetail = &stakeDetail{
				Value: op.value,
			}
		} else {
			// Accumulate the punish value
			punishmentDetail.Value += op.value
		}
		punishmentDetail.Height = op.height
		// Update the punishment detail of target
		if err := op.setDetail(addr, punishmentKey, punishmentDetail); err != nil {
			return err
		}

		// Sub the stake detail
		normalStakeKey := getDetailKey(addr, op.minerType, types.Staked)
		normalDetail, err := op.getDetail(addr, normalStakeKey)
		if err != nil {
			return err
		}
		// Must not happen
		if normalDetail == nil {
			panic(fmt.Errorf("penalty can't find detail of the target:%v", addr.Hex()))
		}
		if normalDetail.Value > op.value {
			normalDetail.Value -= op.value
			normalDetail.Height = op.height
			if err := op.setDetail(addr, normalStakeKey, normalDetail); err != nil {
				return err
			}
		} else {
			remain := op.value - normalDetail.Value
			normalDetail.Value = 0
			op.removeDetail(addr, normalStakeKey)

			// Need to sub frozen stake detail if remain > 0
			if remain > 0 {
				frozenKey := getDetailKey(addr, op.minerType, types.StakeFrozen)
				frozenDetail, err := op.getDetail(addr, frozenKey)
				if err != nil {
					return err
				}
				if frozenDetail == nil {
					panic(fmt.Errorf("penalty can't find frozen detail of target:%v", addr.Hex()))
				}
				if frozenDetail.Value < remain {
					panic(fmt.Errorf("frozen detail value less than remain punish value %v %v %v", frozenDetail.Value, remain, addr.Hex()))
				}
				frozenDetail.Value -= remain
				frozenDetail.Height = op.height
				if err := op.setDetail(addr, frozenKey, frozenDetail); err != nil {
					return err
				}
			}
		}
	}

	// Finally, add the penalty stake to the balance of rewards
	if len(op.rewards) > 0 {
		addEach := new(big.Int).SetUint64(op.value * uint64(len(op.targets)) / uint64(len(op.rewards)))
		for _, addr := range op.rewards {
			op.accountDB.AddBalance(addr, addEach)
		}
	}

	return nil
}
