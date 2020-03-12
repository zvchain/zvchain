package core

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

type tickFullCallBack func(op *voteMinerPoolOp, targetMiner *types.Miner) (error, types.ReceiptStatus)
type becomeFullGuardNodeCallBack func(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus)
type reduceTicketCallBack func(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) (error, types.ReceiptStatus)

type baseIdentityOp interface {
	processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) (error, types.ReceiptStatus)
	processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) (error, types.ReceiptStatus)
	processStakeReduce(op *stakeReduceOp, targetMiner *types.Miner) (error, types.ReceiptStatus)
	processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) (error, types.ReceiptStatus)
	processApplyGuard(op *applyGuardMinerOp, targetMiner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus)
	processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) (error, types.ReceiptStatus)
	processChangeFundGuardMode(op *changeFundGuardMode, targetMiner *types.Miner) (error, types.ReceiptStatus)

	checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus)
	checkUpperBound(miner *types.Miner, height uint64) bool

	afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) (error, types.ReceiptStatus)
	afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus)
	afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) (error, types.ReceiptStatus)
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

func (i *InvalidProposalMiner) processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("invalid miner pool unSupported miner abort"), types.RSMinerUnSupportOp
}
func (i *InvalidProposalMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("invalid miner pool not support apply guard node"), types.RSMinerUnSupportOp
}

func (i *InvalidProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("invalid miner pool not support stake add"), types.RSMinerUnSupportOp
}

func (v *VerifyMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	//verify node must can stake by myself
	if op.addSource != op.addTarget {
		return fmt.Errorf("could not stake to other's verify node"), types.RSMinerUnSupportOp
	}
	return nil, types.RSSuccess
}

func (v *VerifyMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("verify node not support vote"), types.RSMinerUnSupportOp
}

func (v *VerifyMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("verify node not support apply guard node"), types.RSMinerUnSupportOp
}

func (n *NormalProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	if op.addSource == op.addTarget || op.addSource == types.GetStakePlatformAddr() {
		return nil, types.RSSuccess
	}
	sourceMiner, err := getMiner(op.accountDB, op.addSource, types.MinerTypeProposal)
	if err != nil {
		return err, types.RSFail
	}
	if sourceMiner != nil && sourceMiner.IsMinerPool() {
		return nil, types.RSSuccess
	}

	return fmt.Errorf("stake add to normal node only can be stake add by fund owner or miner pool"), types.RSMinerUnSupportOp
}

func (n *NormalProposalMiner) afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus) {
	detail.DisMissHeight = height + adjustWeightPeriod/2
	addFullStakeGuardPool(db, address)
	if err := setDetail(db, address, detailKey, detail); err != nil {
		return err, types.RSFail
	}
	Logger.Infof("normal guard upgrade full stake guard node success,addr =%v,height=%v,dismissHeight=%v", address.AddrPrefixString(), height, detail.DisMissHeight)
	return nil, types.RSSuccess
}

func (g *GuardProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	// guard miner node cannot be staked by others
	if op.addSource != op.addTarget {
		return fmt.Errorf("guard miner node cannot be staked by others"), types.RSMinerUnSupportOp
	}
	return nil, types.RSSuccess
}

func (g *GuardProposalMiner) afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus) {
	// it must be fund guard node
	if detail.DisMissHeight == 0 {
		detail.DisMissHeight = height + adjustWeightPeriod/2
		err := updateFundGuardPoolStatus(db, address, fullStakeGuardNodeType, height)
		if err != nil {
			return err, types.RSFail
		}
		addFullStakeGuardPool(db, address)
		Logger.Infof("fund guard upgrade full stake guard node success,addr =%v,height=%v,dismissHeight=%v", address.AddrPrefixString(), height, detail.DisMissHeight)
	} else {
		detail.DisMissHeight = detail.DisMissHeight + adjustWeightPeriod/2
		Logger.Infof("fund guard upgrade full stake guard node success,addr =%v,height=%v,dismissHeight=%v", address.AddrPrefixString(), height, detail.DisMissHeight)
	}
	if err := setDetail(db, address, detailKey, detail); err != nil {
		return err, types.RSFail
	}
	return nil, types.RSSuccess
}

func (g *GuardProposalMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("guard node not support vote"), types.RSMinerUnSupportOp
}

func (m *MinerPoolProposalMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return checkMinerPoolUpperBound(miner, height)
}

func (m *MinerPoolProposalMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("miner pool not support apply guard node"), types.RSMinerUnSupportOp
}

func (m *MinerPoolProposalMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	sourceMiner, err := getMiner(op.accountDB, op.addSource, types.MinerTypeProposal)
	if err != nil {
		return err, types.RSFail
	}
	if sourceMiner != nil && sourceMiner.IsMinerPool() && op.addSource != op.addTarget {
		return fmt.Errorf("miner pool can not stake add to other miner pool"), types.RSMinerUnSupportOp
	}
	return nil, types.RSSuccess
}

func (m *MinerPoolProposalMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) (error, types.ReceiptStatus) {
	isFull := isFullTickets(totalTickets, op.height)
	if !isFull {
		if miner == nil {
			return fmt.Errorf("find miner pool miner is nil,addr is %s", op.target.AddrPrefixString()), types.RSFail
		}
		Logger.Infof("downgrade invalid pool miner node,addr = %s,height = %v,currentTickets=%v", op.target.AddrPrefixString(), op.height, totalTickets)
		miner.UpdateIdentity(types.InValidMinerPool, op.height)
		// Remove from pool if active
		if miner.IsActive() {
			removeFromPool(op.accountDB, types.MinerTypeProposal, op.target, miner.Stake)
			miner.UpdateStatus(types.MinerStatusPrepare, op.height)
		}
		if err := setMiner(op.accountDB, miner); err != nil {
			return err, types.RSFail
		}
		Logger.Infof("downgrade invalid pool miner node,addr = %s,height = %v,currentTickets=%v", op.target.AddrPrefixString(), op.height, totalTickets)
	}
	return nil, types.RSSuccess

}

func (m *MinerPoolProposalMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func (b *BaseMiner) processMinerAbort(op *minerAbortOp, miner *types.Miner) (error, types.ReceiptStatus) {
	err, rs := b.checkStakeAbort(op, miner)
	if err != nil {
		return err, rs
	}
	// Remove from pool if active
	if miner.IsActive() {
		Logger.Infof("miner abort,remove from pool")
		removeFromPool(op.accountDB, op.minerType, op.addr, miner.Stake)
	}
	// Update the miner status
	miner.UpdateStatus(types.MinerStatusPrepare, op.height)
	if err := setMiner(op.accountDB, miner); err != nil {
		return err, types.RSFail
	}

	Logger.Infof("minerabort success,addr =%s,height=%v,left=%v", op.addr.AddrPrefixString(), op.height, miner.Stake)
	log.MarketLogger.Info("minerabort success,addr =%s,height=%v,left=%v", op.addr.AddrPrefixString(), op.height, miner.Stake)
	return nil, types.RSSuccess
}

func (b *BaseMiner) checkStakeAbort(op *minerAbortOp, miner *types.Miner) (error, types.ReceiptStatus) {
	if miner == nil {
		return fmt.Errorf("no miner info"), types.RSMinerNotExists
	}
	if miner.IsPrepare() {
		return fmt.Errorf("already in prepare status"), types.RSMinerAbortHasPrepared
	}
	// Frozen miner must wait for 1 hour after frozen
	if miner.IsFrozen() && op.height <= miner.StatusUpdateHeight+oneHourBlocks {
		return fmt.Errorf("frozen miner can't abort less than 1 hour since frozen"), types.RSMinerStakeFrozen
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return checkUpperBound(miner, height)
}

func (b *BaseMiner) afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func (b *BaseMiner) checkCanReduce(op *stakeReduceOp, minerType types.MinerType, miner *types.Miner) (error, types.ReceiptStatus) {
	if miner.IsFrozen() {
		return fmt.Errorf("frozen miner must abort first"), types.RSMinerStakeFrozen
	}
	// Proposal node can reduce lowerbound
	if !checkLowerBound(miner) && types.IsVerifyRole(minerType) {
		if miner.IsActive() {
			return fmt.Errorf("active verify miner cann't reduce stake to below bound"), types.RSMinerVerifyLowerStake
		}
		// prepared status,check node is in live group
		if !GroupManagerImpl.MinerJoinedLivedGroupCountFilter(1, op.height)(op.cancelTarget) {
			return fmt.Errorf("miner still in active groups, cannot reduce stake"), types.RSMinerVerifyInGroup
		}
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) processStakeReduce(op *stakeReduceOp, miner *types.Miner) (error, types.ReceiptStatus) {
	if miner == nil {
		return fmt.Errorf("no miner info"), types.RSMinerNotExists
	}
	if miner.Stake < op.value {
		return fmt.Errorf("miner stake not enough:%v %v", miner.Stake, op.value), types.RSMinerStakeLessThanReduce
	}
	originStake := miner.Stake
	// Update miner stake
	miner.Stake -= op.value

	// Check if can do the reduce operation
	if err, rs := b.checkCanReduce(op, op.minerType, miner); err != nil {
		return err, rs
	}

	// Sub the corresponding total stake of the proposals
	if miner.IsActive() && types.IsProposalRole(op.minerType) {
		if !checkLowerBound(miner) {
			Logger.Infof("stake reduce lower min bound,remove from pool")
			removeFromPool(op.accountDB, op.minerType, op.cancelTarget, originStake)
			miner.UpdateStatus(types.MinerStatusPrepare, op.height)
		} else {
			subProposalTotalStake(op.accountDB, op.value)
		}
	}
	if err := setMiner(op.accountDB, miner); err != nil {
		return err, types.RSFail
	}

	// Get Target account detail: staked-detail of who stakes for me
	stakedDetailKey := getDetailKey(op.cancelSource, op.minerType, types.Staked)
	stakedDetail, err := getDetail(op.accountDB, op.cancelTarget, stakedDetailKey)
	if err != nil {
		return err, types.RSFail
	}
	if stakedDetail == nil {
		return fmt.Errorf("target account has no staked detail data"), types.RSFail
	}

	if op.height < stakedDetail.DisMissHeight {
		return fmt.Errorf("current height can not be reduce,dismissHeight is %v,current height is %v", stakedDetail.DisMissHeight, op.height), types.RSMinerReduceHeightNotEnough
	}

	// Must not happened
	if stakedDetail.Value > originStake {
		panic(fmt.Errorf("detail stake more than total stake of the miner:%v %v %x", stakedDetail.Value, originStake, miner.ID))
	}

	if stakedDetail.Value < op.value {
		return fmt.Errorf("detail stake less than cancel amount:%v %v", stakedDetail.Value, op.value), types.RSMinerStakeLessThanReduce
	}

	// Decrease the stake of the staked-detail
	// Removal will be taken if decreasing to zero
	stakedDetail.Value -= op.value
	stakedDetail.Height = op.height
	if stakedDetail.Value == 0 {
		removeDetail(op.accountDB, op.cancelTarget, stakedDetailKey)
	} else {
		if err := setDetail(op.accountDB, op.cancelTarget, stakedDetailKey, stakedDetail); err != nil {
			return err, types.RSFail
		}
	}
	// Get Target account detail: frozen-detail of who stake for me
	frozenDetailKey := getDetailKey(op.cancelSource, op.minerType, types.StakeFrozen)
	frozenDetail, err := getDetail(op.accountDB, op.cancelTarget, frozenDetailKey)
	if err != nil {
		return err, types.RSFail
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
		return err, types.RSFail
	}
	Logger.Infof("stakereduce success,source=%s,to=%s,height=%v,value=%v,left=%v", op.cancelSource, op.cancelTarget, op.height, op.value, miner.Stake)
	return nil, types.RSSuccess
}

func checkVote(op *voteMinerPoolOp, vf *voteInfo) (error, types.ReceiptStatus) {
	sourceMiner, err := getMiner(op.accountDB, op.source, types.MinerTypeProposal)
	if err != nil {
		return err, types.RSFail
	}
	if sourceMiner == nil {
		return fmt.Errorf("miner info is nil,cannot vote"), types.RSMinerUnSupportOp
	}
	if !sourceMiner.IsGuard() {
		return fmt.Errorf("this miner is not guard node,can not vote"), types.RSMinerUnSupportOp
	}
	var voteHeight uint64 = 0
	if vf != nil {
		voteHeight = vf.Height
	}
	canVote := checkCanVote(voteHeight, op.height)
	if !canVote {
		return fmt.Errorf("has voted in this round,can not vote,last vote height = %v", voteHeight), types.RSVoteNotInRound
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) (error, types.ReceiptStatus) {
	vf, err := getVoteInfo(op.accountDB, op.source)
	if err != nil {
		return err, types.RSFail
	}
	var rs types.ReceiptStatus
	var isFull bool
	err, rs = checkVote(op, vf)
	if err != nil {
		return err, rs
	}
	// process base
	err, isFull, rs = processVote(op, vf)
	if err != nil {
		return err, rs
	}
	if isFull {
		if ticketsFullFunc != nil {
			err, rs = ticketsFullFunc(op, targetMiner)
			if err != nil {
				return err, rs
			}
		}
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	Logger.Infof("address %s is upgrade miner pool at height %v", op.targetAddr.AddrPrefixString(), op.height)
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
		return err, types.RSFail
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) (error, types.ReceiptStatus) {
	totalTickets := subTicket(op.accountDB, op.target)
	Logger.Infof("reduce ticket success,target is %s,height is %v,tickets = %d", op.target.AddrPrefixString(), op.height, totalTickets)
	if afterTicketReduceFunc != nil {
		return afterTicketReduceFunc(op, targetMiner, totalTickets)
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) checkApplyGuard(op *applyGuardMinerOp, miner *types.Miner, detailKey []byte, detail *stakeDetail) (error, types.ReceiptStatus) {
	if miner == nil {
		return fmt.Errorf("no miner info"), types.RSMinerNotExists
	}
	if detail == nil {
		return fmt.Errorf("target account has no staked detail data"), types.RSMinerNotFullStake
	}
	if !isFullStake(detail.Value, op.height) {
		return fmt.Errorf("not full stake,apply guard faild"), types.RSMinerNotFullStake
	}
	if detail.DisMissHeight > op.height && detail.DisMissHeight-op.height > adjustWeightPeriod/2 {
		return fmt.Errorf("apply guard time too long,addr is %s", op.targetAddr.String()), types.RSMinerMaxApplyGuard
	}
	return nil, types.RSSuccess
}

func (b *BaseMiner) processApplyGuard(op *applyGuardMinerOp, miner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus) {
	detailKey := getDetailKey(op.targetAddr, types.MinerTypeProposal, types.Staked)
	stakedDetail, err := getDetail(op.accountDB, op.targetAddr, detailKey)
	if err != nil {
		return err, types.RSFail
	}
	var rs types.ReceiptStatus
	err, rs = b.checkApplyGuard(op, miner, detailKey, stakedDetail)
	if err != nil {
		return err, rs
	}
	// update miner identity and set dismissHeight to detail
	miner.UpdateIdentity(types.MinerGuard, op.height)
	if err = setMiner(op.accountDB, miner); err != nil {
		return err, types.RSFail
	}
	if becomeFullGuardNodeFunc != nil {
		err, rs = becomeFullGuardNodeFunc(op.accountDB, detailKey, stakedDetail, op.targetAddr, op.height)
		if err != nil {
			return err, rs
		}
	}
	vf, err := getVoteInfo(op.accountDB, op.targetAddr)
	if err != nil {
		return err, types.RSFail
	}
	// if this node is guard node,its has vote info,only set true
	if vf != nil {
		vf.Height = op.height
		err = setVoteInfo(op.accountDB, op.targetAddr, vf)
		if err != nil {
			return err, types.RSFail
		}
	}
	Logger.Infof("apply guard success,address is %s,height is %v", op.targetAddr.AddrPrefixString(), op.height)
	return nil, types.RSSuccess
}

func (b *BaseMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func (b *BaseMiner) processChangeFundGuardMode(op *changeFundGuardMode, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	if op.height > adjustWeightPeriod/2 {
		return fmt.Errorf("changge fund guard mode must be in suitable height"), types.RSMinerChangeModeExpired
	}
	fn, err := getFundGuardNode(op.accountDB, op.source)
	if err != nil {
		return err, types.RSFail
	}
	if fn == nil {
		return fmt.Errorf("fund  guard info is nil"), types.RSMinerUnSupportOp
	}
	if !fn.isFundGuard() {
		return fmt.Errorf("only fund guard can do this operator"), types.RSMinerUnSupportOp
	}
	err = updateFundGuardMode(op.accountDB, fn, op.source, op.mode, op.height)
	if err == nil {
		return err, types.RSFail
	}
	Logger.Infof("change fund guard mode success,addr = %v,current mode is %v,height=%v", op.source, op.mode, op.height)
	return nil, types.RSSuccess
}

func (b *BaseMiner) processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) (error, types.ReceiptStatus) {
	err := reduceBalance(op.accountDB, op.addSource, op.value)
	if err != nil {
		return err, types.RSBalanceNotEnough
	}
	// Already exists
	if targetMiner != nil {
		if targetMiner.IsFrozen() { // Frozen miner must abort first
			return fmt.Errorf("miner is frozen, cannot add stake"), types.RSMinerStakeFrozen
		}
		// check uint64 overflow
		if targetMiner.Stake+op.value < targetMiner.Stake {
			return fmt.Errorf("stake overflow:%v %v", targetMiner.Stake, op.value), types.RSFail
		}
		targetMiner.Stake += op.value
	} else {
		targetMiner = initMiner(op)
	}
	if op.addTarget == op.addSource {
		setPks(targetMiner, op.minerPks)
		Logger.Infof("stakeadd set pks success,from=%v,to=%v,type=%d,height=%d,value=%v", op.addSource, op.addTarget, op.minerType, op.height, op.value)
	}
	if !checkUpperBound(targetMiner, op.height) {
		return fmt.Errorf("stake more than upper bound:%v", targetMiner.Stake), types.RSMinerStakeOverLimit
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
		Logger.Infof("stakeadd success,from=%v,to=%v,type=%d,height=%d,value=%v,add to pool", op.addSource, op.addTarget, op.minerType, op.height, op.value)
	}
	// Save miner
	if err := setMiner(op.accountDB, targetMiner); err != nil {
		return err, types.RSFail
	}
	// Set detail of the target account: who stakes from me
	detailKey := getDetailKey(op.addSource, op.minerType, types.Staked)
	detail, err := getDetail(op.accountDB, op.addTarget, detailKey)
	if err != nil {
		return err, types.RSFail
	}
	if detail != nil {
		if detail.Value+op.value < detail.Value {
			return fmt.Errorf("stake detail value overflow:%v %v", detail.Value, op.value), types.RSFail
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
		return err, types.RSFail
	}

	Logger.Infof("stakeadd success,from=%v,to=%v,type=%d,height=%d,value=%v", op.addSource, op.addTarget, op.minerType, op.height, op.value)

	return nil, types.RSSuccess
}

func (u *UnSupportMiner) checkStakeAdd(op *stakeAddOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported stake add"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) checkUpperBound(miner *types.Miner, height uint64) bool {
	return false
}

func (u *UnSupportMiner) processStakeAdd(op *stakeAddOp, targetMiner *types.Miner, checkUpperBound func(miner *types.Miner, height uint64) bool) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported stake add"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) processMinerAbort(op *minerAbortOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported miner abort"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) processStakeReduce(op *stakeReduceOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported miner abort"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) processVote(op *voteMinerPoolOp, targetMiner *types.Miner, ticketsFullFunc tickFullCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported miner abort"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) processChangeFundGuardMode(op *changeFundGuardMode, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported change fund guard mode"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) afterTicketsFull(op *voteMinerPoolOp, targetMiner *types.Miner) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func (u *UnSupportMiner) processApplyGuard(op *applyGuardMinerOp, targetMiner *types.Miner, becomeFullGuardNodeFunc becomeFullGuardNodeCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported apply guard"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) afterBecomeFullGuardNode(db types.AccountDB, detailKey []byte, detail *stakeDetail, address common.Address, height uint64) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func (u *UnSupportMiner) processReduceTicket(op *reduceTicketsOp, targetMiner *types.Miner, afterTicketReduceFunc reduceTicketCallBack) (error, types.ReceiptStatus) {
	return fmt.Errorf("unSupported reduce ticket"), types.RSMinerUnSupportOp
}

func (u *UnSupportMiner) afterTicketReduce(op *reduceTicketsOp, miner *types.Miner, totalTickets uint64) (error, types.ReceiptStatus) {
	return nil, types.RSSuccess
}

func reduceBalance(db types.AccountDB, target common.Address, value uint64) error {
	amount := new(big.Int).SetUint64(value)
	if needTransfer(amount) {
		if !db.CanTransfer(target, amount) {
			return fmt.Errorf("balance not enough")
		}
		// Sub the balance of source account
		db.SubBalance(target, amount)
	}
	return nil
}
