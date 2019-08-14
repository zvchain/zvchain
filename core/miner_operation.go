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
	oneHourBlocks = 86400 / onBlockSeconds / 24   // Blocks generated in one hour on average, used when status transforms from Frozen to Prepare
	oneDayBlocks  = 86400 / onBlockSeconds        // Blocks generated in one day on average
	twoDayBlocks  = 2 * oneDayBlocks // Blocks generated in two days on average, used when executes the miner refund
	stakeBuffer  = 15 * oneDayBlocks

)

// mOperation define some functions on miner operation
// Used when executes the miner related transactions or stake operations from contract
// Different from the stateTransition, it doesn't take care of the gas and only focus on the operation
// In mostly case, some functions can be reused in stateTransition
type mOperation interface {
	Validate() error         // Validate the input args
	ParseTransaction() error // Parse the input transaction
	Transition() *result     // Do the operation
	Source()common.Address
	Target()common.Address
	Value() uint64
	GetDb()types.AccountDB
	Height()uint64
	GetMinerType()types.MinerType
    GetBaseOperation()*baseOperation
}



// newOperation creates the mOperation instance base on msg type
func newOperation(db types.AccountDB, msg types.MinerOperationMessage, height uint64) mOperation {
	baseOp := newBaseOperation(db, msg, height,nil)
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
	case types.TransactionTypeApplyGuardMiner:
		operation = &applyGuardMinerOp{baseOperation: baseOp}
	case types.TransactionTypeVoteMinerPool:
		operation = &voteMinerPoolOp{baseOperation: baseOp}
	case types.TransactionTypeCancelGuard:
		operation = &cancelGuardOp{baseOperation: baseOp}
	default:
		operation = &unSupported{typ: msg.OpType()}
	}

	return operation
}

type voteMinerPoolOp struct {
	*baseOperation
	targetAddr common.Address
}

func (op *voteMinerPoolOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *voteMinerPoolOp)Height()uint64{
	return op.height
}
func (op *voteMinerPoolOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *voteMinerPoolOp)GetDb()types.AccountDB{
	return op.db
}

func (op *voteMinerPoolOp) Value() uint64 {
	return 0
}

func (op *voteMinerPoolOp) Source() common.Address {
	return *op.msg.Operator()
}

func (op *voteMinerPoolOp)Target()common.Address{
	return op.targetAddr
}

func (op *voteMinerPoolOp) Validate() error {
	if op.msg.OpTarget() == nil{
		return fmt.Errorf("target must be not nil")
	}
	if *op.msg.Operator() == op.targetAddr{
		return fmt.Errorf("could not vote myself")
	}
	return nil
}

func (op *voteMinerPoolOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil{
		return fmt.Errorf("target must be not nil")
	}
	op.targetAddr = *op.msg.OpTarget()
	op.minerType = types.MinerTypeProposal
	return nil
}

func (op *voteMinerPoolOp) Transition() *result {
	ret := newResult()
	targetMiner, err := op.getMiner(op.targetAddr)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,targetMiner)
	err = baseOp.processMinerOp(op,targetMiner,VoteMinerPoolOp)
	if err != nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}


type applyGuardMinerOp struct {
	*baseOperation
	targetAddr common.Address
}

func (op *applyGuardMinerOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *applyGuardMinerOp)Height()uint64{
	return op.height
}
func (op *applyGuardMinerOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *applyGuardMinerOp)GetDb()types.AccountDB{
	return op.db
}

func (op *applyGuardMinerOp) Value() uint64 {
	return 0
}

func (op *applyGuardMinerOp) Source() common.Address {
	return *op.msg.Operator()
}

func (op *applyGuardMinerOp)Target()common.Address{
	return op.targetAddr
}

func (op *applyGuardMinerOp) Validate() error {
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	return nil
}

func (op *applyGuardMinerOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil {
		return fmt.Errorf("target is nil")
	}
	op.targetAddr = *op.msg.Operator()
	op.minerType = types.MinerTypeProposal
	return nil
}

func (op *applyGuardMinerOp) Transition() *result {
	ret := newResult()
	miner, err := op.getMiner(op.targetAddr)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	if miner == nil{
		ret.setError(fmt.Errorf("no miner info"),types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,miner)
	err = baseOp.processMinerOp(op,miner,ApplyGuardOp)
	if err != nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}


// stakeAddOp is for the stake add operation, miner can add stake for himself or others
type reduceTicketsOp struct {
	*baseOperation
	addTarget common.Address
	source    common.Address
}

func newReduceTicketsOp(db types.AccountDB,targetAddress common.Address,source common.Address,height uint64)*reduceTicketsOp{
	base := newTransitionContext(db, nil, nil)
    return &reduceTicketsOp{
		baseOperation:newBaseOperation(db, nil, height,base),
		addTarget:targetAddress,
		source:source,
	}
}

func (op *reduceTicketsOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *reduceTicketsOp)Height()uint64{
	return op.height
}
func (op *reduceTicketsOp)GetMinerType()types.MinerType{
	return types.MinerTypeProposal
}

func (op *reduceTicketsOp)GetDb()types.AccountDB{
	return op.db
}

func (op *reduceTicketsOp) Value() uint64 {
	return 0
}

func (op *reduceTicketsOp) Source() common.Address {
	return op.source
}

func (op *reduceTicketsOp)Target()common.Address{
	return op.addTarget
}

func (op *reduceTicketsOp) Validate() error {
	return nil
}

func (op *reduceTicketsOp) ParseTransaction() error {
	op.minerType = types.MinerTypeProposal
	return nil
}

func (op *reduceTicketsOp) Transition() *result {
	ret := newResult()
	targetMiner, err := op.getMiner(op.addTarget)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,targetMiner)
	err = baseOp.processMinerOp(op,targetMiner,ReduceTicketOp)
	if err !=nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}

// stakeAddOp is for the stake add operation, miner can add stake for himself or others
type cancelGuardOp struct {
	*baseOperation
	cancelTarget common.Address
}

func (op *cancelGuardOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *cancelGuardOp)Height()uint64{
	return op.height
}
func (op *cancelGuardOp)GetMinerType()types.MinerType{
	return types.MinerTypeProposal
}

func (op *cancelGuardOp)GetDb()types.AccountDB{
	return op.db
}

func (op *cancelGuardOp) Value() uint64 {
	return 0
}

func (op *cancelGuardOp) Source() common.Address {
	return op.source
}

func (op *cancelGuardOp)Target()common.Address{
	return op.cancelTarget
}

func (op *cancelGuardOp) Validate() error {
	if *op.msg.Operator() != types.AdminAddrType{
		return fmt.Errorf("only admin can call")
	}
	if op.msg.OpTarget() == nil{
		return fmt.Errorf("target can not be nil")
	}
	if !types.IsInExtractGuardNodes(*op.msg.OpTarget()){
		return fmt.Errorf("operator addr is not in extract guard nodes")
	}
	return nil
}

func (op *cancelGuardOp) ParseTransaction() error {
	if op.msg.OpTarget() == nil{
		return fmt.Errorf("target can not be nil")
	}
	op.minerType = types.MinerTypeProposal
	op.cancelTarget = *op.msg.OpTarget()
	return nil
}

func (op *cancelGuardOp) Transition() *result {
	ret := newResult()
	targetMiner, err := op.getMiner(op.cancelTarget)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,targetMiner)
	err = baseOp.processMinerOp(op,targetMiner,CancelGuardOp)
	if err !=nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}


// stakeAddOp is for the stake add operation, miner can add stake for himself or others
type stakeAddOp struct {
	*baseOperation
	minerPks  *types.MinerPks
	value     uint64
	addSource common.Address
	addTarget common.Address
}

func (op *stakeAddOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *stakeAddOp)Height()uint64{
	return op.height
}
func (op *stakeAddOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *stakeAddOp)GetDb()types.AccountDB{
	return op.db
}

func (op *stakeAddOp) Value() uint64 {
	return op.value
}

func (op *stakeAddOp) Source() common.Address {
	return op.addSource
}

func (op *stakeAddOp)Target()common.Address{
	return op.addTarget
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

func (op *stakeAddOp) Transition() *result {
	ret := newResult()
	targetMiner, err := op.getMiner(op.addTarget)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,targetMiner)
	err = baseOp.processMinerOp(op,targetMiner,StakedAddOp)
	if err !=nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}

// minerAbortOp abort the miner, which can cause miner status transfer to Prepare
// and quit mining
type minerAbortOp struct {
	*baseOperation
	addr common.Address
}

func (op *minerAbortOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *minerAbortOp)Height()uint64{
	return op.height
}
func (op *minerAbortOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *minerAbortOp)GetDb()types.AccountDB{
	return op.db
}

func (op *minerAbortOp) Value() uint64 {
	return 0
}

func (op *minerAbortOp) Source() common.Address {
	return op.addr
}

func (op *minerAbortOp)Target()common.Address{
	return op.addr
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

func (op *minerAbortOp) Transition() *result {
	ret := newResult()
	miner, err := op.getMiner(op.addr)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	baseOp := geneBaseIdentityOp(op.minerType,miner)
	err = baseOp.processMinerOp(op,miner,StakeAbortOp)
	if err != nil{
		ret.setError(err,types.RSFail)
		return ret
	}
	return ret
}

// stakeReduceOp is for stake reduce operation
type stakeReduceOp struct {
	*baseOperation
	cancelTarget common.Address
	cancelSource common.Address
	value        uint64
}

func (op *stakeReduceOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *stakeReduceOp)AddToPool(address common.Address, addStake uint64){
	return
}

func (op *stakeReduceOp)Height()uint64{
	return op.height
}
func (op *stakeReduceOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *stakeReduceOp)GetDb()types.AccountDB{
	return op.db
}

func (op *stakeReduceOp) Value() uint64 {
	return op.value
}

func (op *stakeReduceOp) Source() common.Address {
	return op.cancelSource
}

func (op *stakeReduceOp)Target()common.Address{
	return op.cancelTarget
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
	} else if miner.IsActive(){
		if op.opVerifyRole(){
			if !checkLowerBound(miner) {
				return fmt.Errorf("active verify miner cann't reduce stake to below bound")
			}
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

func (op *stakeReduceOp) Transition() *result {
	ret := newResult()
	remove:=false
	miner, err := op.getMiner(op.cancelTarget)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	if miner == nil {
		ret.setError(fmt.Errorf("no miner info"),types.RSFail)
		return ret
	}
	if miner.Stake < op.value {
		ret.setError(fmt.Errorf("miner stake not enough:%v %v", miner.Stake, op.value),types.RSFail)
		return ret
	}
	originStake := miner.Stake
	// Update miner stake
	miner.Stake -= op.value

	// Check if can do the reduce operation
	if err := op.checkCanReduce(miner); err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}

	// Sub the corresponding total stake of the proposals
	if miner.IsActive() && op.opProposalRole() {
		if !checkLowerBound(miner){
			op.removeFromPool(op.cancelTarget, originStake)
			miner.UpdateStatus(types.MinerStatusPrepare, op.height)
			remove = true
		}else{
			op.subProposalTotalStake(op.value)
		}
	}
	if err := op.setMiner(miner); err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}

	// Get Target account detail: staked-detail of who stakes for me
	stakedDetailKey := getDetailKey(op.cancelSource, op.minerType, types.Staked)
	stakedDetail, err := op.getDetail(op.cancelTarget, stakedDetailKey)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	if stakedDetail == nil {
		ret.setError(fmt.Errorf("target account has no staked detail data"),types.RSFail)
		return ret
	}

	if op.height < stakedDetail.DisMissHeight{
		ret.setError(fmt.Errorf("current height can not be reduce,dismissHeight is %v,current height is %v",stakedDetail.DisMissHeight,op.height),types.RSFail)
		return ret
	}

	// Must not happened
	if stakedDetail.Value > originStake {
		panic(fmt.Errorf("detail stake more than total stake of the miner:%v %v %x", stakedDetail.Value, originStake, miner.ID))
	}

	if stakedDetail.Value < op.value {
		ret.setError(fmt.Errorf("detail stake less than cancel amount:%v %v", stakedDetail.Value, op.value),types.RSFail)
		return ret
	}

	// Decrease the stake of the staked-detail
	// Removal will be taken if decreasing to zero
	stakedDetail.Value -= op.value
	stakedDetail.Height = op.height
	if stakedDetail.Value == 0 {
		op.removeDetail(op.cancelTarget, stakedDetailKey)
	} else {
		if err := op.setDetail(op.cancelTarget, stakedDetailKey, stakedDetail); err != nil {
			ret.setError(err,types.RSFail)
			return ret
		}
	}
	// Get Target account detail: frozen-detail of who stake for me
	frozenDetailKey := getDetailKey(op.cancelSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := op.getDetail(op.cancelTarget, frozenDetailKey)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
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
		ret.setError(err,types.RSFail)
		return ret
	}
	if remove && MinerManagerImpl != nil {
		// Informs MinerManager the removal address
		MinerManagerImpl.proposalRemoveCh <- op.cancelTarget
	}
	return ret
}

// stakeRefundOp is for stake refund operation, it only happens after stake-reduce ops
type stakeRefundOp struct {
	*baseOperation
	refundTarget common.Address
	refundSource common.Address
}
func (op *stakeRefundOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}
func (op *stakeRefundOp)Height()uint64{
	return op.height
}
func (op *stakeRefundOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *stakeRefundOp)GetDb()types.AccountDB{
	return op.db
}

func (op *stakeRefundOp) Value() uint64 {
	return 0
}


func (op *stakeRefundOp) Source() common.Address {
	return op.refundSource
}

func (op *stakeRefundOp)Target()common.Address{
	return op.refundTarget
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

func (op *stakeRefundOp) Transition() *result {
	ret := newResult()
	// Get the detail in target account: frozen-detail of the source
	frozenDetailKey := getDetailKey(op.refundSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := op.getDetail(op.refundTarget, frozenDetailKey)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	if frozenDetail == nil {
		ret.setError(fmt.Errorf("target has no frozen detail"),types.RSFail)
		return ret
	}
	// Check reduce-height
	if op.height <= frozenDetail.Height+twoDayBlocks {
		ret.setError(fmt.Errorf("refund cann't happen util 2days after last reduce"),types.RSFail)
		return ret
	}

	// Remove frozen data
	op.removeDetail(op.refundTarget, frozenDetailKey)

	// Restore the balance
	op.db.AddBalance(op.refundSource, new(big.Int).SetUint64(frozenDetail.Value))
	return ret

}

// minerFreezeOp freeze the miner, which can cause miner status transfer to Frozen
// and quit mining.
// It was called by the group-create routine when the miner didn't participate in the process completely
type minerFreezeOp struct {
	*baseOperation
	addr common.Address
}

func (op *minerFreezeOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *minerFreezeOp)Height()uint64{
	return op.height
}
func (op *minerFreezeOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *minerFreezeOp)GetDb()types.AccountDB{
	return op.db
}

func (op *minerFreezeOp) Value() uint64 {
	return 0
}


func (op *minerFreezeOp) Source() common.Address {
	return op.addr
}

func (op *minerFreezeOp)Target()common.Address{
	return op.addr
}

func (op *minerFreezeOp) ParseTransaction() error {
	return nil
}

func (op *minerFreezeOp) Validate() error {
	return nil
}

func (op *minerFreezeOp) Transition() *result {
	ret := newResult()
	if !op.opVerifyRole() {
		ret.setError(fmt.Errorf("not operates a verifier:%v", op.addr.String()),types.RSFail)
		return ret
	}
	miner, err := op.getMiner(op.addr)
	if err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}
	if miner == nil {
		ret.setError(fmt.Errorf("no miner info"),types.RSFail)
		return ret
	}
	if miner.IsFrozen() {
		ret.setError(fmt.Errorf("already in forzen status"),types.RSFail)
		return ret
	}
	if !miner.IsVerifyRole() {
		ret.setError(fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID)),types.RSFail)
		return ret
	}

	// Remove from pool if active
	if miner.IsActive() {
		op.removeFromPool(op.addr, miner.Stake)
	}

	// Update the miner status
	miner.UpdateStatus(types.MinerStatusFrozen, op.height)
	if err := op.setMiner(miner); err != nil {
		ret.setError(err,types.RSFail)
		return ret
	}

	return ret
}

type minerPenaltyOp struct {
	*baseOperation
	targets []common.Address
	rewards []common.Address
	value   uint64
}

func (op *minerPenaltyOp)GetBaseOperation()*baseOperation{
	return op.baseOperation
}

func (op *minerPenaltyOp)Height()uint64{
	return op.height
}
func (op *minerPenaltyOp)GetMinerType()types.MinerType{
	return op.minerType
}

func (op *minerPenaltyOp)GetDb()types.AccountDB{
	return op.db
}

func (op *minerPenaltyOp) Value() uint64 {
	return op.value
}

func (op *minerPenaltyOp) Source() common.Address {
	return common.Address{}
}

func (op *minerPenaltyOp)Target()common.Address{
	return common.Address{}
}

func (op *minerPenaltyOp) ParseTransaction() error {
	return nil
}

func (op *minerPenaltyOp) Validate() error {
	return nil
}

func (op *minerPenaltyOp) Transition() *result {
	ret := newResult()
	if !op.opVerifyRole() {
		ret.setError(fmt.Errorf("not operates verifiers"),types.RSFail)
		return ret
	}
	// Firstly, frozen the targets
	for _, addr := range op.targets {
		miner, err := op.getMiner(addr)
		if err != nil {
			ret.setError(err,types.RSFail)
			return ret
		}
		if miner == nil {
			ret.setError(fmt.Errorf("no miner info"),types.RSFail)
			return ret
		}
		if !miner.IsVerifyRole() {
			ret.setError(fmt.Errorf("not a verifier:%v", common.ToHex(miner.ID)),types.RSFail)
			return ret
		}

		// Remove from pool if active
		if miner.IsActive() {
			op.removeFromPool(addr, miner.Stake)
		}
		// Must not happen
		if miner.Stake < op.value {
			panic(fmt.Errorf("stake less than punish value:%v %v of %v", miner.Stake, op.value, addr.AddrPrefixString()))
		}

		// Sub total stake and update the miner status
		miner.Stake -= op.value
		miner.UpdateStatus(types.MinerStatusFrozen, op.height)
		if err := op.setMiner(miner); err != nil {
			ret.setError(err,types.RSFail)
			return ret
		}
		// Add punishment detail
		punishmentKey := getDetailKey(common.PunishmentDetailAddr, op.minerType, types.StakePunishment)
		punishmentDetail, err := op.getDetail(addr, punishmentKey)
		if err != nil {
			ret.setError(err,types.RSFail)
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
		if err := op.setDetail(addr, punishmentKey, punishmentDetail); err != nil {
			ret.setError(err,types.RSFail)
			return ret
		}

		// Sub the stake detail
		normalStakeKey := getDetailKey(addr, op.minerType, types.Staked)
		normalDetail, err := op.getDetail(addr, normalStakeKey)
		if err != nil {
			ret.setError(err,types.RSFail)
			return ret
		}
		// Must not happen
		if normalDetail == nil {
			panic(fmt.Errorf("penalty can't find detail of the target:%v", addr.AddrPrefixString()))
		}
		if normalDetail.Value > op.value {
			normalDetail.Value -= op.value
			normalDetail.Height = op.height
			if err := op.setDetail(addr, normalStakeKey, normalDetail); err != nil {
				ret.setError(err,types.RSFail)
				return ret
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
					ret.setError(err,types.RSFail)
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
				if err := op.setDetail(addr, frozenKey, frozenDetail); err != nil {
					ret.setError(err,types.RSFail)
					return ret
				}
			}
		}
	}

	// Finally, add the penalty stake to the balance of rewards
	if len(op.rewards) > 0 {
		addEach := new(big.Int).SetUint64(op.value * uint64(len(op.targets)) / uint64(len(op.rewards)))
		for _, addr := range op.rewards {
			op.db.AddBalance(addr, addEach)
		}
	}

	return ret
}
