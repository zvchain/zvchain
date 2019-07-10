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
	"github.com/zvchain/zvchain/storage/vm"
	"math/big"
)

// Duration definition related to miner operation
const (
	oneDayBlocks = 86400 / 3        // Blocks generated in one day on average, used when status transforms from Frozen to Prepare
	twoDayBlocks = 2 * oneDayBlocks // Blocks generated in two days on average, used when executes the miner refund
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

// newOperation creates the mOperation instance base on msg type
func newOperation(db vm.AccountDB, msg vm.MinerOperationMessage, height uint64) mOperation {
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
	if !op.accountDB.CanTransfer(op.addSource, amount) {
		return fmt.Errorf("balance not enough")
	}
	// Sub the balance of source account
	op.accountDB.SubBalance(op.addSource, amount)

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

func (op *minerAbortOp) Validate() error {
	if len(op.msg.Payload()) != 1 {
		return fmt.Errorf("msg payload length error")
	}
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
	// Frozen miner must wait for 1day after frozen
	if miner.IsFrozen() && op.height <= miner.StatusUpdateHeight+oneDayBlocks {
		return fmt.Errorf("frozen miner can't abort less than 1days since frozen")
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
		if op.opVerifyRole() && GroupChainImpl.WhetherMemberInActiveGroup(op.cancelTarget.Bytes(), op.height) {
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
