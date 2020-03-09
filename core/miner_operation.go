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
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/params"
	"math/big"
)

// Duration definition related to miner operation
const (
	onBlockSeconds = 3
	oneHourBlocks  = 86400 / onBlockSeconds / 24 // Blocks generated in one hour on average, used when status transforms from Frozen to Prepare
	oneDayBlocks   = 86400 / onBlockSeconds      // Blocks generated in one day on average
	stakeBuffer    = 15 * oneDayBlocks

	// determined after community votes for the refund deadline
	refundDeadlineTwoDays       = 2 * oneDayBlocks // Blocks generated in two days on average, used when executes the miner refund
	refundDeadlineNinetyDays    = 90 * oneDayBlocks
	refundDeadlineHalfYear      = 180 * oneDayBlocks
	refundDeadlineOneDayForTest = oneDayBlocks
)

// mOperation define some functions on miner operation
// Used when executes the miner related transactions or stake operations from contract
// Different from the stateTransition, it doesn't take care of the gas and only focus on the operation
// In mostly case, some functions can be reused in stateTransition
type mOperation interface {
	ParseTransaction() error // Parse the input transaction
	Transition() *result     // Do the operation
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
	log.CoreLogger.Infof("vote begin,source=%s,target=%s,height=%v", op.source.AddrPrefixString(), op.targetAddr.AddrPrefixString(), op.height)
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB, op.targetAddr, types.MinerTypeProposal)
	if err != nil {
		err = fmt.Errorf("vote failed,source=%s,target=%s,height=%v,error=%v", op.source.AddrPrefixString(), op.targetAddr.AddrPrefixString(), op.height, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err, rs = baseOp.processVote(op, targetMiner, baseOp.afterTicketsFull)
	if err != nil {
		err = fmt.Errorf("vote failed,source=%s,target=%s,height=%v,error=%v", op.source.AddrPrefixString(), op.targetAddr.AddrPrefixString(), op.height, err)
		ret.setError(err, rs)
		return ret
	}
	return ret
}

type applyGuardMinerOp struct {
	*transitionContext
	targetAddr common.Address
}

func (op *applyGuardMinerOp) ParseTransaction() error {
	op.targetAddr = *op.msg.Operator()
	return nil
}

func (op *applyGuardMinerOp) Transition() *result {
	log.CoreLogger.Infof("apply guard begin,source=%s,height=%v", op.targetAddr.AddrPrefixString(), op.height)
	ret := newResult()
	miner, err := getMiner(op.accountDB, op.targetAddr, types.MinerTypeProposal)
	if err != nil {
		log.CoreLogger.Errorf("apply guard failed,source=%s,height=%v,error=%v", op.targetAddr.AddrPrefixString(), op.height, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	if miner == nil {
		err = fmt.Errorf("apply guard failed,source=%s,height=%v,error=no miner info", op.targetAddr.AddrPrefixString(), op.height)
		ret.setError(err, types.RSMinerNotExists)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, miner)
	err, rs = baseOp.processApplyGuard(op, miner, baseOp.afterBecomeFullGuardNode)
	if err != nil {
		err = fmt.Errorf("apply guard failed,source=%s,height=%v,error=%v", op.targetAddr.AddrPrefixString(), op.height, err)
		ret.setError(err, rs)
		return ret
	}
	return ret
}

// if guard is invalid,or change vote, then do this op
type reduceTicketsOp struct {
	*transitionContext
	target common.Address
}

func newReduceTicketsOp(db types.AccountDB, targetAddress common.Address, height uint64) *reduceTicketsOp {
	base := newTransitionContext(db, nil, nil, height)
	return &reduceTicketsOp{
		transitionContext: base,
		target:            targetAddress,
	}
}

func (op *reduceTicketsOp) ParseTransaction() error {
	return nil
}

func (op *reduceTicketsOp) Transition() *result {
	log.CoreLogger.Infof("reduce ticket begin,target=%s,height=%v", op.target.AddrPrefixString(), op.height)
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB, op.target, types.MinerTypeProposal)
	if err != nil {
		err = fmt.Errorf("reduce ticket failed,target=%s,height=%v,error=%v", op.target.AddrPrefixString(), op.height, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err, rs = baseOp.processReduceTicket(op, targetMiner, baseOp.afterTicketReduce)
	if err != nil {
		err = fmt.Errorf("reduce ticket failed,target=%s,height=%v,error=%v", op.target.AddrPrefixString(), op.height, err)
		ret.setError(err, rs)
		return ret
	}
	return ret
}

type changeFundGuardMode struct {
	*transitionContext
	source common.Address
	mode   common.FundModeType
}

func (op *changeFundGuardMode) ParseTransaction() error {
	if len(op.msg.Payload()) != 1 {
		return fmt.Errorf("data length should be 1")
	}
	if err := fundGuardModeCheck(common.FundModeType(op.msg.Payload()[0])); err != nil {
		return err
	}
	op.source = *op.msg.Operator()
	op.mode = common.FundModeType(op.msg.Payload()[0])
	return nil
}

func (op *changeFundGuardMode) Transition() *result {
	log.CoreLogger.Infof("begin change fund mode,source=%s,mode=%d,height=%v", op.source, op.mode, op.height)
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB, op.source, types.MinerTypeProposal)
	if err != nil {
		err = fmt.Errorf("change fund mode error,source=%s,mode=%d,height=%v,error=%v", op.source, op.mode, op.height, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(types.MinerTypeProposal, targetMiner)
	err, rs = baseOp.processChangeFundGuardMode(op, targetMiner)
	if err != nil {
		err = fmt.Errorf("change fund mode error,source=%s,mode=%d,height=%v,error=%v", op.source, op.mode, op.height, err)
		ret.setError(err, rs)
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
	log.CoreLogger.Infof("stake add begin,from=%s,to=%s,type=%d,height=%d,value=%v", op.addSource.AddrPrefixString(), op.addTarget.AddrPrefixString(), op.minerType, op.height, op.value)
	ret := newResult()
	targetMiner, err := getMiner(op.accountDB, op.addTarget, op.minerType)
	if err != nil {
		err = fmt.Errorf("stake add failed,from=%s,to=%s,type=%d,height=%d,value=%v,error=%v", op.addSource.AddrPrefixString(), op.addTarget.AddrPrefixString(), op.minerType, op.height, op.value, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(op.minerType, targetMiner)
	err, rs = baseOp.checkStakeAdd(op, targetMiner)
	if err != nil {
		err = fmt.Errorf("stake add failed,from=%s,to=%s,type=%d,height=%d,value=%v,error=%v", op.addSource.AddrPrefixString(), op.addTarget.AddrPrefixString(), op.minerType, op.height, op.value, err)
		ret.setError(err, rs)
		return ret
	}
	err, rs = baseOp.processStakeAdd(op, targetMiner, baseOp.checkUpperBound)
	if err != nil {
		err = fmt.Errorf("stake add failed,from=%s,to=%s,type=%d,height=%d,value=%v,error=%v", op.addSource.AddrPrefixString(), op.addTarget.AddrPrefixString(), op.minerType, op.height, op.value, err)
		ret.setError(err, rs)
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
	log.CoreLogger.Infof("miner abort begin,addr=%s,type=%d,height=%d", op.addr.AddrPrefixString(), op.minerType, op.height)
	ret := newResult()
	miner, err := getMiner(op.accountDB, op.addr, op.minerType)
	if err != nil {
		err = fmt.Errorf("miner abort failed,addr=%s,type=%d,height=%d,error=%v", op.addr.AddrPrefixString(), op.minerType, op.height, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(op.minerType, miner)
	err, rs = baseOp.processMinerAbort(op, miner)
	if err != nil {
		err = fmt.Errorf("miner abort failed,addr=%s,type=%d,height=%d,error=%v", op.addr.AddrPrefixString(), op.minerType, op.height, err)
		ret.setError(err, rs)
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
	log.CoreLogger.Infof("stake reduce begin,source=%s,target=%s,height=%v,type = %d,value=%v", op.cancelSource, op.cancelTarget, op.height, op.minerType, op.value)
	ret := newResult()
	miner, err := getMiner(op.accountDB, op.cancelTarget, op.minerType)
	if err != nil {
		err = fmt.Errorf("stake reduce failed,source=%s,target=%s,height=%v,type=%d,value=%v,error=%v", op.cancelSource, op.cancelTarget, op.height, op.minerType, op.value, err)
		ret.setError(err, types.RSFail)
		return ret
	}
	var rs types.ReceiptStatus
	baseOp := geneBaseIdentityOp(op.minerType, miner)
	err, rs = baseOp.processStakeReduce(op, miner)
	if err != nil {
		err = fmt.Errorf("stake reduce failed,source=%s,target=%s,height=%v,type=%d,value=%v,error=%v", op.cancelSource, op.cancelTarget, op.height, op.minerType, op.value, err)
		ret.setError(err, rs)
		return ret
	}
	return ret
}

// stakeRefundOp is for stake refund operation, it only happens after stake-reduce ops
type stakeRefundOp struct {
	*transitionContext
	refundTarget common.Address
	refundSource common.Address
	minerType    types.MinerType
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
	frozenDetail, err := getDetail(op.accountDB, op.refundTarget, frozenDetailKey)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	if frozenDetail == nil {
		ret.setError(fmt.Errorf("target has no frozen detail"), types.RSFail)
		return ret
	}

	// Check reduce-height
	if params.GetChainConfig().IsZIP003(frozenDetail.Height) && op.refundSource != types.GetStakePlatformAddr() {
		dl := uint64(refundDeadlineOneDayForTest)
		if op.height <= frozenDetail.Height+dl {
			ret.setError(fmt.Errorf("refund cann't happen util %vdays after last reduce", dl), types.RSMinerRefundHeightNotEnougn)
			return ret
		}
	} else {
		if op.height <= frozenDetail.Height+refundDeadlineTwoDays {
			ret.setError(fmt.Errorf("refund cann't happen util 2days after last reduce"), types.RSMinerRefundHeightNotEnougn)
			return ret
		}
	}

	// Remove frozen data
	removeDetail(op.accountDB, op.refundTarget, frozenDetailKey)

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
	miner, err := getMiner(op.accountDB, op.addr, types.MinerTypeVerify)
	if err != nil {
		ret.setError(err, types.RSFail)
		return ret
	}
	if miner == nil {
		ret.setError(fmt.Errorf("no miner info"), types.RSMinerNotExists)
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
		removeFromPool(op.accountDB, types.MinerTypeVerify, op.addr, miner.Stake)
	}

	// Update the miner status
	miner.UpdateStatus(types.MinerStatusFrozen, op.height)
	if err := setMiner(op.accountDB, miner); err != nil {
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
		miner, err := getMiner(op.accountDB, addr, types.MinerTypeVerify)
		if err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}
		if miner == nil {
			ret.setError(fmt.Errorf("no miner info"), types.RSMinerNotExists)
			return ret
		}
		if !miner.IsVerifyRole() {
			ret.setError(fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID)), types.RSFail)
			return ret
		}

		// Remove from pool if active
		if miner.IsActive() {
			removeFromPool(op.accountDB, types.MinerTypeVerify, addr, miner.Stake)
		}
		// Must not happen
		if miner.Stake < op.value {
			panic(fmt.Errorf("stake less than punish value:%v %v of %v", miner.Stake, op.value, addr.AddrPrefixString()))
		}

		// Sub total stake and update the miner status
		miner.Stake -= op.value
		miner.UpdateStatus(types.MinerStatusFrozen, op.height)
		if err := setMiner(op.accountDB, miner); err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}
		// Add punishment detail
		punishmentKey := getDetailKey(common.PunishmentDetailAddr, types.MinerTypeVerify, types.StakePunishment)
		punishmentDetail, err := getDetail(op.accountDB, addr, punishmentKey)
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
		if err := setDetail(op.accountDB, addr, punishmentKey, punishmentDetail); err != nil {
			ret.setError(err, types.RSFail)
			return ret
		}

		// Sub the stake detail
		normalStakeKey := getDetailKey(addr, types.MinerTypeVerify, types.Staked)
		normalDetail, err := getDetail(op.accountDB, addr, normalStakeKey)
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
			if err := setDetail(op.accountDB, addr, normalStakeKey, normalDetail); err != nil {
				ret.setError(err, types.RSFail)
				return ret
			}
		} else {
			remain := op.value - normalDetail.Value
			normalDetail.Value = 0
			removeDetail(op.accountDB, addr, normalStakeKey)

			// Need to sub frozen stake detail if remain > 0
			if remain > 0 {
				frozenKey := getDetailKey(addr, types.MinerTypeVerify, types.StakeFrozen)
				frozenDetail, err := getDetail(op.accountDB, addr, frozenKey)
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
				if err := setDetail(op.accountDB, addr, frozenKey, frozenDetail); err != nil {
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
