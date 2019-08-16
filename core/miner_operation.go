//   Copyright (C) 2019 ZVChain
//
//   This program is free software: you can redistribute it and/or modify/

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
	onBlockSeconds = 3
	oneHourBlocks  = 86400 / onBlockSeconds / 24 // Blocks generated in one hour on average, used when status transforms from Frozen to Prepare
	oneDayBlocks   = 86400 / onBlockSeconds      // Blocks generated in one day on average
	twoDayBlocks   = 2 * oneDayBlocks            // Blocks generated in two days on average, used when executes the miner refund
	stakeBuffer    = 15 * oneDayBlocks
	cancelFundGuardBuffer    = 7 * oneDayBlocks
)

// mOperation define some functions on miner operation
// Used when executes the miner related transactions or stake operations from contract
// Different from the stateTransition, it doesn't take care of the gas and only focus on the operation
// In mostly case, some functions can be reused in stateTransition
type mOperation interface {
	ParseTransaction() error // Parse the input transaction
	Transition() *result     // Do the operation
}

// newOperation creates the mOperation instance base on msg type
func newOperation(db types.AccountDB, msg types.TxMessage, height uint64) mOperation {
	base := newTransitionContext(db, msg, nil, height)
	var operation mOperation

	switch msg.OpType() {
	case types.TransactionTypeStakeAdd:
		operation = &stakeAddOp{transitionContext: base}
	case types.TransactionTypeMinerAbort:
		operation = &minerAbortOp{transitionContext: base}
	case types.TransactionTypeStakeReduce:
		operation = &stakeReduceOp{transitionContext: base}
	case types.TransactionTypeStakeRefund:
		operation = &stakeRefundOp{transitionContext: base}
	case types.TransactionTypeApplyGuardMiner:
		operation = &applyGuardMinerOp{transitionContext: base}
	case types.TransactionTypeVoteMinerPool:
		operation = &voteMinerPoolOp{transitionContext: base}
	case types.TransactionTypeCancelGuard:
		operation = &cancelGuardOp{transitionContext: base}
	default:
		operation = &unSupported{typ: msg.OpType()}
	}

	return operation
}

type voteMinerPoolOp struct {
	*transitionContext
	source     common.Address
	targetAddr common.Address
}

func (op *voteMinerPoolOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target must be not nil")
	}
	op.source = *op.msg.Operator()
	op.targetAddr = *op.msg.OpTarget()
	return nil
}

func (op *voteMinerPoolOp) Transition() *result {
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB,op.targetAddr,types.MinerTypeProposal)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err = baseOp.processVote(op, targetMiner,baseOp.ticketsFullFunc())
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

type applyGuardMinerOp struct {
	*transitionContext
	targetAddr common.Address
}

func (op *applyGuardMinerOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	op.targetAddr = *op.msg.Operator()
	return nil
}

func (op *applyGuardMinerOp) Transition() *result {
	ret := newResult()
	miner, err := getMiner(op.accountDB,op.targetAddr,types.MinerTypeProposal)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	if miner == nil {
		ret.setError(fmt.Errorf("no miner info"), types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, miner)
	err = baseOp.processApplyGuard(op, miner,baseOp.becomeFullGuardNodeFunc())
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// if guard is invalid,or change vote, then do this op
type reduceTicketsOp struct {
	*transitionContext
	target common.Address
	source    common.Address
}

func newReduceTicketsOp(db types.AccountDB, targetAddress common.Address, source common.Address, height uint64) *reduceTicketsOp {
	base := newTransitionContext(db, nil, nil, height)
	return &reduceTicketsOp{
		transitionContext: base,
		target:         targetAddress,
		source:            source,
	}
}

func (op *reduceTicketsOp) ParseTransaction() error {
	return nil
}

func (op *reduceTicketsOp) Transition() *result {
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB,op.target,types.MinerTypeProposal)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err = baseOp.processReduceTicket(op, targetMiner,baseOp.afterTicketReduceFunc())
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// stakeAddOp is for the stake add operation, miner can add stake for himself3 or others
type cancelGuardOp struct {
	*transitionContext
	cancelTarget common.Address
}

func (op *cancelGuardOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target can not be nil")
	}
	op.cancelTarget = *op.msg.OpTarget()
	return nil
}

func (op *cancelGuardOp) Transition() *result {
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB,op.cancelTarget,types.MinerTypeProposal)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err = baseOp.processCancelGuard(op, targetMiner)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// stakeAddOp is for the stake add operation, miner can add stake for himself or others
type stakeAddOp struct {
	*transitionContext
	minerPks  *types.MinerPks
	value     uint64
	addSource common.Address
	addTarget common.Address
	minerType types.MinerType
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

func (op *stakeAddOp) Transition() *result {
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB, op.addTarget, op.minerType)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType, targetMiner)
	err = baseOp.checkStakeAdd(op, targetMiner)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	err = baseOp.processStakeAdd(op, targetMiner, baseOp.checkUpperBound)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// minerAbortOp abort the miner, which can cause miner status transfer to Prepare
// and quit mining
type minerAbortOp struct {
	*transitionContext
	addr      common.Address
	minerType types.MinerType
}

func (op *minerAbortOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.addr = *op.msg.Operator()
	return nil
}

func (op *minerAbortOp) Transition() *result {
	ret := newResult()
	miner, err := getMiner(op.accountDB,op.addr,op.minerType)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType, miner)
	err = baseOp.processMinerAbort(op, miner)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// stakeReduceOp is for stake reduce operation
type stakeReduceOp struct {
	*transitionContext
	cancelTarget common.Address
	cancelSource common.Address
	value        uint64
	minerType    types.MinerType
}

func (op *stakeReduceOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.cancelTarget = *op.msg.OpTarget()
	op.cancelSource = *op.msg.Operator()
	op.value = op.msg.Amount().Uint64()
	return nil
}


func (op *stakeReduceOp) Transition() *result {
	ret := newResult()
	miner, err := getMiner(op.accountDB,op.cancelTarget,op.minerType)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType, miner)
	err = baseOp.processStakeReduce(op, miner)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	return ret
}

// stakeRefundOp is for stake refund operation, it only happens after stake-reduce ops
type stakeRefundOp struct {
	*transitionContext
	refundTarget common.Address
	refundSource common.Address
	minerType  types.MinerType
}

func (op *stakeRefundOp) ParseTransaction() error {
	op.minerType = types.MinerType(op.msg.Payload()[0])
	op.refundTarget = *op.msg.OpTarget()
	op.refundSource = *op.msg.Operator()
	return nil
}

func (op *stakeRefundOp) Transition() *result {
	ret := newResult()
	// Get the detail in target account: frozen-detail of the source
	frozenDetailKey := getDetailKey(op.refundSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := getDetail(op.accountDB,op.refundTarget, frozenDetailKey)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	if frozenDetail == nil {
		ret.setError(fmt.Errorf("target has no frozen detail"), types.RSFail)
		return ret
	}
	// Check reduce-height
	if op.height <= frozenDetail.Height+twoDayBlocks {
		ret.setError(fmt.Errorf("refund cann't happen util 2days after last reduce"), types.RSFail)
		return ret
	}

	// Remove frozen data
	removeDetail(op.accountDB,op.refundTarget, frozenDetailKey)

	// Restore the balance
	op.accountDB.AddBalance(op.refundSource, new(big.Int).SetUint64(frozenDetail.Value))
	return ret

}

// minerFreezeOp freeze the miner, which can cause miner status transfer to Frozen
// and quit mining.
// It was called by the group-create routine when the miner didn't participate in the process completely
type minerFreezeOp struct {
	*transitionContext
	addr common.Address
}

func (op *minerFreezeOp) ParseTransaction() error {
	return nil
}

func (op *minerFreezeOp) Transition() *result {
	ret := newResult()
	miner, err := getMiner(op.accountDB,op.addr,types.MinerTypeVerify)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	if miner == nil {
		ret.setError(fmt.Errorf("no miner info"), types.RSFail)
		return ret
	}
	if miner.IsFrozen() {
		ret.setError(fmt.Errorf("already in forzen status"), types.RSFail)
		return ret
	}
	if !miner.IsVerifyRole() {
		ret.setError(fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID)), types.RSFail)
		return ret
	}

	// Remove from pool if active
	if miner.IsActive() {
		removeFromPool(op.accountDB,types.MinerTypeVerify,op.addr, miner.Stake)
	}

	// Update the miner status
	miner.UpdateStatus(types.MinerStatusFrozen, op.height)
	if err := setMiner(op.accountDB,miner); err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}

	return ret
}

type minerPenaltyOp struct {
	*transitionContext
	targets []common.Address
	rewards []common.Address
	value   uint64
}

func (op *minerPenaltyOp) ParseTransaction() error {
	return nil
}

func (op *minerPenaltyOp) Transition() *result {
	ret := newResult()
	// Firstly, frozen the targets
	for _, addr := range op.targets {
		miner, err := getMiner(op.accountDB,addr,types.MinerTypeVerify)
		if err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}
		if miner == nil {
			ret.setError(fmt.Errorf("no miner info"), types.RSFail)
			return ret
		}
		if !miner.IsVerifyRole() {
			ret.setError(fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID)), types.RSFail)
			return ret
		}

		// Remove from pool if active
		if miner.IsActive() {
			removeFromPool(op.accountDB,types.MinerTypeVerify,addr, miner.Stake)
		}
		// Must not happen
		if miner.Stake < op.value {
			panic(fmt.Errorf("stake less than punish value:%v %v of %v", miner.Stake, op.value, addr.AddrPrefixString()))
		}

		// Sub total stake and update the miner status
		miner.Stake -= op.value
		miner.UpdateStatus(types.MinerStatusFrozen, op.height)
		if err := setMiner(op.accountDB,miner); err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}
		// Add punishment detail
		punishmentKey := getDetailKey(common.PunishmentDetailAddr, types.MinerTypeVerify, types.StakePunishment)
		punishmentDetail, err := getDetail(op.accountDB,addr, punishmentKey)
		if err != nil {
			ret.setError(err, types.RSFail)
			return ret
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
		if err := setDetail(op.accountDB,addr, punishmentKey, punishmentDetail); err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}

		// Sub the stake detail
		normalStakeKey := getDetailKey(addr, types.MinerTypeVerify, types.Staked)
		normalDetail, err := getDetail(op.accountDB,addr, normalStakeKey)
		if err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}
		// Must not happen
		if normalDetail == nil {
			panic(fmt.Errorf("penalty can't find detail of the target:%v", addr.AddrPrefixString()))
		}
		if normalDetail.Value > op.value {
			normalDetail.Value -= op.value
			normalDetail.Height = op.height
			if err := setDetail(op.accountDB,addr, normalStakeKey, normalDetail); err != nil {
				ret.setError(err, types.RSFail)
				return ret
			}
		} else {
			remain := op.value - normalDetail.Value
			normalDetail.Value = 0
			removeDetail(op.accountDB,addr, normalStakeKey)

			// Need to sub frozen stake detail if remain > 0
			if remain > 0 {
				frozenKey := getDetailKey(addr, types.MinerTypeVerify, types.StakeFrozen)
				frozenDetail, err := getDetail(op.accountDB,addr, frozenKey)
				if err != nil {
					ret.setError(err, types.RSFail)
					return ret
				}
				if frozenDetail == nil {
					panic(fmt.Errorf("penalty can't find frozen detail of target:%v", addr.AddrPrefixString()))
				}
				if frozenDetail.Value < remain {
					panic(fmt.Errorf("frozen detail value less than remain punish value %v %v %v", frozenDetail.Value, remain, addr.AddrPrefixString()))
				}
				frozenDetail.Value -= remain
				frozenDetail.Height = op.height
				if err := setDetail(op.accountDB,addr, frozenKey, frozenDetail); err != nil {
					ret.setError(err, types.RSFail)
					return ret
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

	return ret
}
