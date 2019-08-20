package core

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

type BalanceOp = byte

const (
	BalanceReduce BalanceOp = iota
)

type tickFullCallBack func(op *voteMinerPoolOp, targetMiner *types.Miner) error
type becomeFullGuardNodeCallBack func(db types.AccountDB, detailKey []byte,detail *stakeDetail,address common.Address, height uint64) error
type reduceTicketCallBack func(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) error

type baseIdentityOp interface {
	processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) error
	processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) error
	processStakeReduce(op *stakeReduceOp, targetMiner *types.Miner) error
	processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) error
	processApplyGuard(op *applyGuardMinerOp, targetMiner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error
	processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) error
	processCancelGuard(op *cancelGuardOp, targetMiner *types.Miner) error

	checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error
	checkUpperBound(miner *types.Miner, height uint64) bool

	afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) error
	afterBecomeFullGuardNode(db types.AccountDB,detailKey []byte,detail *stakeDetail, address common.Address, height uint64) error
	afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) error
}

func geneBaseIdentityOp(opType types.MinerType, targetMiner *types.Miner) baseIdentityOp {
	if opType == types.MinerTypeVerify {
		return &VerifyMiner{BaseMiner: &BaseMiner{}}
	} else {
		if targetMiner == nil {
			return &NormalProposalMiner{BaseMiner: &BaseMiner{}}
		}
		switch targetMiner.Identity {
		case types.MinerNormal:
			return &NormalProposalMiner{BaseMiner: &BaseMiner{}}
		case types.MinerGuard:
			return &GuardProposalMiner{BaseMiner: &BaseMiner{}}
		case types.MinerPool:
			return &MinerPoolProposalMiner{BaseMiner: &BaseMiner{}}
		case types.InValidMinerPool:
			return &InvalidProposalMiner{BaseMiner: &BaseMiner{}}
		default:
			return &UnSupportMiner{}
		}
	}
}

type BaseMiner struct {
}

type VerifyMiner struct {
	*BaseMiner
}

type NormalProposalMiner struct {
	*BaseMiner
}

type GuardProposalMiner struct {
	*BaseMiner
}

type MinerPoolProposalMiner struct {
	*BaseMiner
}

type InvalidProposalMiner struct {
	*BaseMiner
}

type UnSupportMiner struct {
}

func (i *InvalidProposalMiner) processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) error {
	return fmt.Errorf("invalid miner pool unSupported miner abort")
}
func (i *InvalidProposalMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error {
	return fmt.Errorf("invalid miner pool not support apply guard node")
}

func (v *VerifyMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error {
	//verify node must can stake by myself
	if op.addSource != op.addTarget {
		return fmt.Errorf("could not stake to other's verify node")
	}
	return nil
}

func (v *VerifyMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) error {
	return fmt.Errorf("verify node not support vote")
}

func (v *VerifyMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error {
	return fmt.Errorf("verify node not support apply guard node")
}

func (n *NormalProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error {
	if op.addSource != op.addTarget && op.addSource != types.MiningPoolAddr {
		return fmt.Errorf("only admin can stake to normal")
	}
	return nil
}

func (n *NormalProposalMiner) afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte,detail *stakeDetail,address common.Address, height uint64) error {
	detail.DisMissHeight = height + adjustWeightPeriod/2
	addFullStakeGuardPool(db, address)
	if err := setDetail(db, address, detailKey, detail); err != nil {
		return err
	}
	return nil
}

func (g *GuardProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error {
	// guard miner node cannot be staked by others
	if op.addSource != op.addTarget {
		return fmt.Errorf("guard miner node cannot be staked by others")
	}
	return nil
}

func (g *GuardProposalMiner) afterBecomeFullGuardNode(db types.AccountDB,detailKey []byte,detail *stakeDetail, address common.Address, height uint64) error {
	// it must be fund guard node
	if detail.DisMissHeight == 0 {
		detail.DisMissHeight = height + adjustWeightPeriod/2
		err := updateFundGuardPoolStatus(db, address, fundGuardNodeType, height)
		if err != nil {
			return err
		}
		addFullStakeGuardPool(db, address)
	} else {
		detail.DisMissHeight = detail.DisMissHeight + adjustWeightPeriod/2
	}
	if err := setDetail(db, address, detailKey, detail); err != nil {
		return err
	}
	return nil
}

func (g *GuardProposalMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) error {
	return fmt.Errorf("guard node not support vote")
}

func (g *GuardProposalMiner) processCancelGuard(op *cancelGuardOp, targetMiner *types.Miner) error {
	if op.height < adjustWeightPeriod/2-cancelFundGuardBuffer || op.height > adjustWeightPeriod/2+cancelFundGuardBuffer {
		return fmt.Errorf("cancel fund guard must be in suitable height,addr is %s", op.cancelTarget.String())
	}
	fn, err := getFundGuardNode(op.accountDB, op.cancelTarget)
	if err != nil {
		return err
	}
	if fn == nil {
		return fmt.Errorf("fund  guard info is nil,addr is %s", op.cancelTarget.String())
	}
	if !fn.isFundGuard() {
		return fmt.Errorf("only fund guard can do this operator ,addr is %s,type is %d", op.cancelTarget.String(), fn.Type)
	}
	err = updateFundGuardPoolStatus(op.accountDB, op.cancelTarget, normalNodeType, op.height)
	if err != nil {
		return err
	}
	return guardNodeExpired(op.accountDB, op.cancelTarget, op.height, true)
}

func (m *MinerPoolProposalMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return checkMinerPoolUpperBound(miner, height)
}

func (m *MinerPoolProposalMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error {
	return fmt.Errorf("miner pool not support apply guard node")
}

func (m *MinerPoolProposalMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) error {
	isFull := isFullTickets(totalTickets, op.height)
	if !isFull {
		if miner == nil {
			return fmt.Errorf("find miner pool miner is nil,addr is %s", op.target.String())
		}
		miner.UpdateIdentity(types.InValidMinerPool, op.height)
		remove := false
		// Remove from pool if active
		if miner.IsActive() {
			removeFromPool(op.accountDB, types.MinerTypeProposal, op.target, miner.Stake)
			miner.UpdateStatus(types.MinerStatusPrepare, op.height)
			remove = true
		}
		if err := setMiner(op.accountDB, miner); err != nil {
			return err
		}
		if remove && MinerManagerImpl != nil {
			// Informs MinerManager the removal address
			MinerManagerImpl.proposalRemoveCh <- op.target
		}
	}
	return nil

}

func (m *MinerPoolProposalMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) error {
	return nil
}

func (b *BaseMiner) processMinerAbort(op *minerAbortOp, miner *types.Miner) error {
	remove := false
	// Remove from pool if active
	if miner.IsActive() {
		removeFromPool(op.accountDB, op.minerType, op.addr, miner.Stake)
		if types.IsProposalRole(op.minerType) {
			remove = true
		}
	}
	// Update the miner status
	miner.UpdateStatus(types.MinerStatusPrepare, op.height)
	if err := setMiner(op.accountDB, miner); err != nil {
		return err
	}
	if remove && MinerManagerImpl != nil {
		// Informs MinerManager the removal address
		MinerManagerImpl.proposalRemoveCh <- op.addr
	}
	return nil
}

func (b *BaseMiner) checkStakeAbort(op *minerAbortOp, miner *types.Miner) error {
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
	return nil
}

func (b *BaseMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error {
	return nil
}

func (b *BaseMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return checkUpperBound(miner, height)
}

func (b *BaseMiner) afterBecomeFullGuardNode(db types.AccountDB,detailKey []byte,detail *stakeDetail, address common.Address, height uint64) error {
	return nil
}

func (b *BaseMiner) checkCanReduce(op *stakeReduceOp, minerType types.MinerType, miner *types.Miner) error {
	if miner.IsFrozen() {
		return fmt.Errorf("frozen miner must abort first")
	}
	// Proposal node can reduce lowerbound
	if !checkLowerBound(miner) && types.IsVerifyRole(minerType) {
		if miner.IsActive() {
			return fmt.Errorf("active verify miner cann't reduce stake to below bound")
		}
		// prepared status,check node is in live group
		if GroupManagerImpl.GetGroupStoreReader().MinerLiveGroupCount(op.cancelTarget, op.height) > 0 {
			return fmt.Errorf("miner still in active groups, cannot reduce stake")
		}
	}
	return nil
}

func (b *BaseMiner) processStakeReduce(op *stakeReduceOp, miner *types.Miner) error {
	remove := false
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
	if err := b.checkCanReduce(op, op.minerType, miner); err != nil {
		return err
	}

	// Sub the corresponding total stake of the proposals
	if miner.IsActive() && types.IsProposalRole(op.minerType) {
		if !checkLowerBound(miner) {
			removeFromPool(op.accountDB, op.minerType, op.cancelTarget, originStake)
			miner.UpdateStatus(types.MinerStatusPrepare, op.height)
			remove = true
		} else {
			subProposalTotalStake(op.accountDB, op.value)
		}
	}
	if err := setMiner(op.accountDB, miner); err != nil {
		return err
	}

	// Get Target account detail: staked-detail of who stakes for me
	stakedDetailKey := getDetailKey(op.cancelSource, op.minerType, types.Staked)
	stakedDetail, err := getDetail(op.accountDB, op.cancelTarget, stakedDetailKey)
	if err != nil {
		return err
	}
	if stakedDetail == nil {
		return fmt.Errorf("target account has no staked detail data")
	}

	if op.height < stakedDetail.DisMissHeight {
		return fmt.Errorf("current height can not be reduce,dismissHeight is %v,current height is %v", stakedDetail.DisMissHeight, op.height)
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
		removeDetail(op.accountDB, op.cancelTarget, stakedDetailKey)
	} else {
		if err := setDetail(op.accountDB, op.cancelTarget, stakedDetailKey, stakedDetail); err != nil {
			return err
		}
	}
	// Get Target account detail: frozen-detail of who stake for me
	frozenDetailKey := getDetailKey(op.cancelSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := getDetail(op.accountDB, op.cancelTarget, frozenDetailKey)
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
	if err := setDetail(op.accountDB, op.cancelTarget, frozenDetailKey, frozenDetail); err != nil {
		return err
	}
	if remove && MinerManagerImpl != nil {
		// Informs MinerManager the removal address
		MinerManagerImpl.proposalRemoveCh <- op.cancelTarget
	}
	return nil
}

func checkVote(op *voteMinerPoolOp,vf *voteInfo)error{
	sourceMiner, err := getMiner(op.accountDB,op.source,types.MinerTypeProposal)
	if err != nil{
		return err
	}
	if sourceMiner == nil{
		return fmt.Errorf("miner info is nil,cannot vote")
	}
	if !sourceMiner.IsGuard(){
		return fmt.Errorf("this miner is not guard node,can not vote")
	}
	var voteHeight uint64 =  0
	if vf != nil{
		voteHeight = vf.Height
	}
	canVote := checkCanVote(voteHeight,op.height)
	if !canVote{
		return fmt.Errorf("has voted in this round,can not vote")
	}
	return nil
}

func (b *BaseMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) error {
	vf, err := getVoteInfo(op.accountDB, op.source)
	if err != nil {
		return err
	}
	err = checkVote(op,vf)
	if err != nil{
		return err
	}
	// process base
	err, isFull := processVote(op,vf)
	if err != nil {
		return err
	}
	if isFull {
		if ticketsFullFunc != nil{
			err = ticketsFullFunc(op, targetMiner)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *BaseMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) error {
	Logger.Infof("address %s is upgrade miner pool", op.targetAddr.String())
	if targetMiner == nil {
		targetMiner = &types.Miner{
			ID:          op.targetAddr.Bytes(),
			Stake:       0,
			ApplyHeight: op.height,
			Type:        types.MinerTypeProposal,
			Status:      types.MinerStatusPrepare,
		}
	}
	targetMiner.UpdateIdentity(types.MinerPool, op.height)
	// Save miner
	if err := setMiner(op.accountDB, targetMiner); err != nil {
		return err
	}
	return nil
}

func (b *BaseMiner) processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) error {
	totalTickets := subTicket(op.accountDB, op.target)
	if afterTicketReduceFunc != nil {
		return afterTicketReduceFunc(op, targetMiner, totalTickets)
	}
	return nil
}

func (b *BaseMiner) checkApplyGuard(op *applyGuardMinerOp, miner *types.Miner,detailKey []byte,detail *stakeDetail) error {
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	if detail == nil {
		return fmt.Errorf("target account has no staked detail data")
	}
	if !isFullStake(detail.Value, op.height) {
		return fmt.Errorf("not full stake,apply guard faild")
	}
	if detail.DisMissHeight > op.height && detail.DisMissHeight-op.height > adjustWeightPeriod/2 {
		return fmt.Errorf("apply guard time too long,addr is %s", op.targetAddr.String())
	}
	return nil
}

func (b *BaseMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error {
	detailKey := getDetailKey(op.targetAddr, types.MinerTypeProposal, types.Staked)
	stakedDetail,err := getDetail(op.accountDB, op.targetAddr, detailKey)
	if err != nil {
		return err
	}
	err = b.checkApplyGuard(op, miner,detailKey,stakedDetail)
	if err != nil {
		return err
	}
	// update miner identity and set dismissHeight to detail
	miner.UpdateIdentity(types.MinerGuard, op.height)
	if err = setMiner(op.accountDB, miner); err != nil {
		return err
	}
	if becomeFullGuardNodeFunc != nil {
		err = becomeFullGuardNodeFunc(op.accountDB, detailKey,stakedDetail,op.targetAddr, op.height)
		if err != nil {
			return err
		}
	}
	vf, err := getVoteInfo(op.accountDB, op.targetAddr)
	if err != nil {
		return err
	}
	// if this node is guard node,its has vote info,only set true
	if vf != nil {
		vf.Height = op.height
		err = setVoteInfo(op.accountDB, op.targetAddr, vf)
		if err != nil {
			return err
		}
	}
	log.CoreLogger.Infof("apply guard success,address is %s,height is %v", op.targetAddr.String(), op.height)
	return nil
}

func (b *BaseMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) error {
	return nil
}

func (b *BaseMiner) processCancelGuard(op *cancelGuardOp, targetMiner *types.Miner) error {
	return fmt.Errorf("unSupported cancel guard")
}

func (b *BaseMiner) processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) error {
	err := updateBalance(op.accountDB, op.addSource, op.value, BalanceReduce)
	if err != nil {
		return err
	}
	add := false
	// Already exists
	if targetMiner != nil {
		if targetMiner.IsFrozen() { // Frozen miner must abort first
			return fmt.Errorf("miner is frozen, cannot add stake")
		}
		// check uint64 overflow
		if targetMiner.Stake+op.value < targetMiner.Stake {
			return fmt.Errorf("stake overflow:%v %v", targetMiner.Stake, op.value)
		}
		targetMiner.Stake += op.value
	} else {
		targetMiner = initMiner(op)
	}
	if op.addTarget == op.addSource {
		setPks(targetMiner, op.minerPks)
	}
	if !checkUpperBound(targetMiner, op.height) {
		return fmt.Errorf("stake more than upper bound:%v", targetMiner.Stake)
	}
	if targetMiner.IsActive() {
		// Update proposal total stake
		if types.IsProposalRole(op.minerType) {
			addProposalTotalStake(op.accountDB, op.value)
		}
	} else if checkCanActivate(targetMiner) { // Check if to active the miner
		targetMiner.UpdateStatus(types.MinerStatusActive, op.height)
		// Add to pool so that the miner can start working
		addToPool(op.accountDB, op.minerType, op.addTarget, targetMiner.Stake)
		if types.IsProposalRole(op.minerType) {
			add = true
		}
	}
	// Save miner
	if err := setMiner(op.accountDB, targetMiner); err != nil {
		return err
	}
	// Set detail of the target account: who stakes from me
	detailKey := getDetailKey(op.addSource, op.minerType, types.Staked)
	detail, err := getDetail(op.accountDB, op.addTarget, detailKey)
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
	if err := setDetail(op.accountDB, op.addTarget, detailKey, detail); err != nil {
		return err
	}

	if add && MinerManagerImpl != nil {
		// Inform added proposer address to minerManager
		MinerManagerImpl.proposalAddCh <- op.addTarget
	}
	return nil
}

func (u *UnSupportMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) error {
	return fmt.Errorf("unSupported stake add")
}

func (u *UnSupportMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return false
}

func (u *UnSupportMiner) processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) error {
	return fmt.Errorf("unSupported stake add")
}

func (u *UnSupportMiner) processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) error {
	return fmt.Errorf("unSupported miner abort")
}

func (u *UnSupportMiner) processStakeReduce(op *stakeReduceOp, targetMiner *types.Miner) error {
	return fmt.Errorf("unSupported miner abort")
}

func (u *UnSupportMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) error {
	return fmt.Errorf("unSupported miner abort")
}

func (u *UnSupportMiner) processCancelGuard(op *cancelGuardOp, targetMiner *types.Miner) error {
	return fmt.Errorf("unSupported cancel guard")
}

func (u *UnSupportMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) error {
	return nil
}

func (u *UnSupportMiner) processApplyGuard(op *applyGuardMinerOp, targetMiner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) error {
	return fmt.Errorf("unSupported apply guard")
}

func (u *UnSupportMiner) afterBecomeFullGuardNode(db types.AccountDB,detailKey []byte,detail *stakeDetail, address common.Address, height uint64) error {
	return nil
}

func (u *UnSupportMiner) processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) error {
	return fmt.Errorf("unSupported reduce ticket")
}

func (u *UnSupportMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64)error {
	return nil
}

func updateBalance(db types.AccountDB, target common.Address, value uint64, balanceOp BalanceOp) error {
	if BalanceReduce == balanceOp {
		amount := new(big.Int).SetUint64(value)
		if needTransfer(amount) {
			if !db.CanTransfer(target, amount) {
				return fmt.Errorf("balance not enough")
			}
			// Sub the balance of source account
			db.SubBalance(target, amount)
		}
	} else {
		return fmt.Errorf("unknow balance update opertation")
	}
	return nil
}
