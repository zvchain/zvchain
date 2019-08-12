package core

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

type MinerOp = byte
type BalanceOp = byte
const (
	StakedAddOp          		MinerOp = iota
	StakeAbortOp
	ApplyGuardOp
	VoteMinerPoolOp
	ReduceTicketOp
	CancelGuardOp
)

const (
	BalanceReduce          		BalanceOp = iota
)

type baseIdentityOp interface {
	processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error
	updateMinerDetail(mop mOperation,targetMiner *types.Miner,stakeStatus types.StakeStatus)error
	updateBalance(mop mOperation,balanceOp BalanceOp)error
	processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error
}


type BaseMiner struct {

}

type ProposalMiner struct {
	*BaseMiner
}

type VerifyMiner struct {
	*BaseMiner
}

type NormalProposalMiner struct {
	*ProposalMiner
}

type GuardProposalMiner struct {
	*ProposalMiner
}

type MinerPoolProposalMiner struct {
	*ProposalMiner
}

type InvalidProposalMiner struct {
	*BaseMiner
}

type UnSupportMiner struct {

}

func geneBaseIdentityOp(opType types.MinerType, targetMiner *types.Miner) baseIdentityOp {
	if opType == types.MinerTypeVerify {
		return &VerifyMiner{BaseMiner:&BaseMiner{}}
	} else {
		if targetMiner == nil {
			return &NormalProposalMiner{ProposalMiner:&ProposalMiner{BaseMiner:&BaseMiner{}}}
		}
		switch targetMiner.Identity {
			case types.MinerNormal:
				return &NormalProposalMiner{ProposalMiner:&ProposalMiner{BaseMiner:&BaseMiner{}}}
			case types.MinerGuard:
				return &GuardProposalMiner{ProposalMiner:&ProposalMiner{BaseMiner:&BaseMiner{}}}
			case types.MinerPool:
				return &MinerPoolProposalMiner{ProposalMiner:&ProposalMiner{BaseMiner:&BaseMiner{}}}
			case types.InValidMinerPool:
				return &InvalidProposalMiner{BaseMiner:&BaseMiner{}}
			default:
				return &UnSupportMiner{}
		}
	}
}

func (n *NormalProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		err := n.checkStakeAdd(mop,targetMiner)
		if err != nil{
			return err
		}
		err = n.processStakeAdd(mop,targetMiner,checkUpperBound)
		if err!=nil{
			return err
		}
	case StakeAbortOp:
		err := n.checkStakeAbort(mop,targetMiner)
		if err != nil{
			return err
		}
		err = n.processStakeAbort(mop,targetMiner)
		if err!=nil{
			return err
		}
	case ApplyGuardOp:
		err := n.checkApplyGuard(mop,targetMiner)
		if err != nil{
			return err
		}
		err = n.processApplyGuard(mop,targetMiner)
		if err != nil{
			return err
		}
	case VoteMinerPoolOp:
		err := n.checkVoteMinerPool(mop)
		if err != nil{
			return err
		}
		err = n.processVoteMinerPool(mop,targetMiner)
		if err != nil{
			return err
		}
	case ReduceTicketOp:
		n.processReduceTicket(mop,mop.Target(),mop.GetBaseOperation().subTicket)
	case CancelGuardOp:
		fmt.Errorf("normal proposal can not be operated")
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func(n*NormalProposalMiner)checkStakeAdd(mop mOperation,targetMiner *types.Miner)error{
	// only admin can stake to normal miner node
	if mop.Source() !=  mop.Target() && mop.Source() !=  adminAddrType{
		return fmt.Errorf("only admin can stake to others")
	}
	return nil
}

func (n *NormalProposalMiner)processReduceTicket(mop mOperation,targetAddress common.Address,subTicketsFun func(address common.Address)uint64){
	subTicketsFun(targetAddress)
}

func (n *NormalProposalMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	err,isFull := processVote(mop)
	if err != nil{
		return err
	}
	// if full,only update nodeIdentity
	if isFull{
		Logger.Infof("address %s is upgrade miner pool",mop.Target().Hex())
		if targetMiner == nil{
			targetMiner = &types.Miner{
				ID:          mop.Target().Bytes(),
				Stake:       0,
				ApplyHeight: mop.Height(),
				Type:        mop.GetMinerType(),
				Status:      types.MinerStatusPrepare,
			}
		}
		targetMiner.UpdateIdentity(types.MinerPool,mop.Height())
		// Save miner
		if err := mop.GetBaseOperation().setMiner(targetMiner); err != nil {
			return err
		}
	}
	return nil
}


func (g *GuardProposalMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	return fmt.Errorf("unSupported vote")
}

func(g*GuardProposalMiner)checkApplyGuard(mop mOperation,miner *types.Miner)error{
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	detailKey := getDetailKey(mop.Source(), mop.GetMinerType(), types.Staked)
	stakedDetail, err := mop.GetBaseOperation().getDetail(mop.Source(), detailKey)
	if err != nil {
		return err
	}
	if stakedDetail == nil {
		return fmt.Errorf("target account has no staked detail data")
	}
	if mop.Height() <= stakedDetail.DisMissHeight{
		return fmt.Errorf("guard node only can apply guard node in buf days")
	}
	return nil
}


func (g *GuardProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		err := g.checkStakeAdd(mop,targetMiner)
		if err != nil{
			return err
		}
		err = g.processStakeAdd(mop,targetMiner,checkUpperBound)
		if err!=nil{
			return err
		}
	case StakeAbortOp:
		err := g.checkStakeAbort(mop,targetMiner)
		if err != nil{
			return err
		}
		err = g.processStakeAbort(mop,targetMiner)
		if err!=nil{
			return err
		}
	case ApplyGuardOp:
		err := g.checkApplyGuard(mop,targetMiner)
		if err != nil{
			return err
		}
		err = g.processApplyGuard(mop,targetMiner)
		if err != nil{
			return err
		}
	case VoteMinerPoolOp:
		return fmt.Errorf("guard node could not be voted by others")
	case ReduceTicketOp:
		g.processReduceTicket(mop,mop.Target(),mop.GetBaseOperation().subTicket)
	case CancelGuardOp:

	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func (m *MinerPoolProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		err := m.processStakeAdd(mop,targetMiner,checkMinerPoolUpperBound)
		if err!=nil{
			return err
		}
	case StakeAbortOp:
		err := m.checkStakeAbort(mop,targetMiner)
		if err != nil{
			return err
		}
		err = m.processStakeAbort(mop,targetMiner)
		if err!=nil{
			return err
		}
	case ApplyGuardOp:
		return fmt.Errorf("miner pool node is not support this operator")
	case VoteMinerPoolOp:
		err := m.checkVoteMinerPool(mop)
		if err != nil{
			return err
		}
		err = m.processVoteMinerPool(mop,targetMiner)
		if err != nil{
			return err
		}
	case ReduceTicketOp:
		err := m.processReduceTicket(mop,mop.Target(),mop.GetBaseOperation().subTicket)
		if err != nil{
			return err
		}
	case CancelGuardOp:
		fmt.Errorf("miner pool can not be operated")
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func (m *MinerPoolProposalMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	if targetMiner == nil{
		return fmt.Errorf("miner cannot be nil")
	}
	// sub vote count
	err := mop.GetBaseOperation().voteMinerPool(mop.Source(),mop.Target())
	if err != nil{
		return err
	}
	// add tickets count
	mop.GetBaseOperation().addTicket(mop.Target())
	return nil
}

func (i *InvalidProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		return fmt.Errorf("invalid miner pool could not support stake add")
	case StakeAbortOp:
		return fmt.Errorf("invalid miner pool could not support stake abort")
	case ApplyGuardOp:
		return fmt.Errorf("invalid pool node not support this operator")
	case VoteMinerPoolOp:
		err := i.checkVoteMinerPool(mop)
		if err != nil{
			return err
		}
		err = i.processVoteMinerPool(mop,targetMiner)
		if err != nil{
			return err
		}
	case ReduceTicketOp:
		i.processReduceTicket(mop,mop.Target(),mop.GetBaseOperation().subTicket)
	case CancelGuardOp:
		fmt.Errorf("invalid miner pool can not be operated")
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func (i *InvalidProposalMiner)processReduceTicket(mop mOperation,targetAddress common.Address,subTicketsFun func(address common.Address)uint64){
	subTicketsFun(targetAddress)
}

func (i *InvalidProposalMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	if targetMiner == nil{
		return fmt.Errorf("target miner can not be nil")
	}
	err,isFull := processVote(mop)
	if err != nil{
		return err
	}
	add := false
	// if full,only update nodeIdentity
	if isFull{
		Logger.Infof("address %s is from invalid miner pool upgrade miner pool",mop.Target().Hex())
		targetMiner.UpdateIdentity(types.MinerPool,mop.Height())
		// Check if to active the miner
		if checkCanActivate(targetMiner) {
			targetMiner.UpdateStatus(types.MinerStatusActive, mop.Height())
			// Add to pool so that the miner can start working
			mop.GetBaseOperation().addToPool(mop.Target(), targetMiner.Stake)
			add = true
		}
		// Save miner
		if err := mop.GetBaseOperation().setMiner(targetMiner); err != nil {
			return err
		}
	}

	if add && MinerManagerImpl != nil {
		// Inform added proposer address to minerManager
		MinerManagerImpl.proposalAddCh <- mop.Target()
	}
	return nil
}


func processVote(mop mOperation)(error,bool){
	vf,err := getVoteInfo(mop.GetDb(),mop.Source())
	if err != nil{
		return err,false
	}
	// sub vote count
	err = mop.GetBaseOperation().voteMinerPool(mop.Source(),mop.Target())
	if err != nil{
		return err,false
	}
	var totalTickets uint64 = 0
	var empty = common.Address{}
	// vote target is old target
	if vf.Target == mop.Target() && vf.Target != empty{
		totalTickets = mop.GetBaseOperation().getTickets(mop.Target())
	}else{
		if vf.Target != empty{
			//reduce ticket first
			mop:=newReduceTicketsOp(mop.GetDb(),vf.Target,mop.Source(),mop.Height())
			ret := mop.Transition()
			if ret.err != nil{
				return ret.err,false
			}
			// add tickets count
		}
		totalTickets = mop.GetBaseOperation().addTicket(mop.Target())
	}
	isFull := mop.GetBaseOperation().isFullTickets(mop.Target(),totalTickets)
	return nil,isFull
}

func (v *VerifyMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	return fmt.Errorf("unSupported vote")
}

func (v *VerifyMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
		case StakedAddOp:
			err := v.checkStakeAdd(mop,targetMiner)
			if err != nil{
				return err
			}
			err = v.processStakeAdd(mop,targetMiner,checkUpperBound)
			if err!=nil{
				return err
			}
		case StakeAbortOp:
			err := v.checkStakeAbort(mop,targetMiner)
			if err != nil{
				return  err
			}
			err = v.processStakeAbort(mop,targetMiner)
			if err!=nil{
				return err
			}
		case ApplyGuardOp:
			return fmt.Errorf("verify node not support this operator")
		case VoteMinerPoolOp:
			return fmt.Errorf("verify node could not be voted by others")
		case ReduceTicketOp:
			return fmt.Errorf("verify node could not be reduce tickets")
		case CancelGuardOp:
			return fmt.Errorf("verify node can not be operated")
	default:
		return fmt.Errorf("unknow operator %v",op)
	}

	return nil
}

func(g*GuardProposalMiner)checkStakeAdd(mop mOperation,targetMiner *types.Miner)error{
	// guard miner node cannot be staked by others
	if mop.Source() !=  mop.Target(){
		return fmt.Errorf("guard miner node cannot be staked by others")
	}
	return nil
}

func (g *GuardProposalMiner)processCancelGuardNode(mop mOperation){
	guardNodeExpired(mop.GetDb(),mop.Target(),mop.Height())
}

func (g *GuardProposalMiner)processReduceTicket(mop mOperation,targetAddress common.Address,subTicketsFun func(address common.Address)uint64){
	subTicketsFun(targetAddress)
}

func (m *MinerPoolProposalMiner)processReduceTicket(mop mOperation,targetAddress common.Address,subTicketsFun func(address common.Address)uint64)error{
	totalTickets := subTicketsFun(targetAddress)
	isFull := mop.GetBaseOperation().isFullTickets(mop.Target(),totalTickets)
	if !isFull{
		miner,err := getMiner(mop.GetDb(),targetAddress,mop.GetMinerType())
		if err != nil{
			return err
		}
		if miner == nil{
			return fmt.Errorf("find miner pool miner is nil,addr is %s",targetAddress.Hex())
		}
		miner.UpdateIdentity(types.InValidMinerPool,mop.Height())
		miner.UpdateStatus(types.MinerStatusPrepare, mop.Height())
		remove:=false
		// Remove from pool if active
		if miner.IsActive() {
			mop.GetBaseOperation().removeFromPool(mop.Source(), miner.Stake)
			if mop.GetBaseOperation().opProposalRole() {
				remove = true
			}
		}
		if err := mop.GetBaseOperation().setMiner(miner); err != nil {
			return err
		}
		if remove && MinerManagerImpl != nil {
			// Informs MinerManager the removal address
			MinerManagerImpl.proposalRemoveCh <- mop.Source()
		}
	}
	return nil
}


func(v*VerifyMiner)checkStakeAdd(mop mOperation,targetMiner *types.Miner)error{
	//verify node must can stake by myself
	if mop.Source() !=  mop.Target(){
		return fmt.Errorf("could not stake to other's verify node")
	}
	return nil
}



func initMiner(mop mOperation)*types.Miner{
	miner := &types.Miner{
		ID:          mop.Target().Bytes(),
		Stake:       mop.Value(),
		ApplyHeight: mop.Height(),
		Type:        mop.GetMinerType(),
		Status:      types.MinerStatusPrepare,
	}
	setPks(miner, mop.GetMinerPks())
	return miner
}


func(b*BaseMiner)checkVoteMinerPool(mop mOperation)error{
	sourceMiner, err := mop.GetBaseOperation().getMiner(mop.Source())
	if err != nil{
		return err
	}
	if !sourceMiner.IsGuard(){
		return fmt.Errorf("only guard node can vote to miner pool")
	}
	vf,err := getVoteInfo(mop.GetDb(),mop.Source())
	if err != nil{
		return err
	}
	if vf == nil || vf.Last < 1{
		return fmt.Errorf("this guard node has no tickets")
	}
	return nil
}

func(b*BaseMiner)checkApplyGuard(mop mOperation,miner *types.Miner)error{
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	detailKey := getDetailKey(mop.Source(), mop.GetMinerType(), types.Staked)
	stakedDetail, err := mop.GetBaseOperation().getDetail(mop.Source(), detailKey)
	if err != nil {
		return err
	}
	if stakedDetail == nil {
		return fmt.Errorf("target account has no staked detail data")
	}
	if !isFullStake(stakedDetail.Value,mop.Height()){
		return fmt.Errorf("not full stake,apply guard faild")
	}
	return nil
}

func(b*BaseMiner)processApplyGuard(mop mOperation,miner *types.Miner) error{
	// update miner identity and set dismissHeight to detail
	miner.UpdateIdentity(types.MinerGuard,mop.Height())
	disMissHeight := mop.Height() + halfOfYearBlocks
	var err error
	err = mop.GetBaseOperation().addGuardMinerInfo(mop.Source(),disMissHeight)
	if err != nil{
		return err
	}
	var detail *stakeDetail
	if err = mop.GetBaseOperation().setMiner(miner); err != nil {
		return err
	}
	detailKey := getDetailKey(mop.Source(), mop.GetMinerType(), types.Staked)
	detail,err = mop.GetBaseOperation().getDetail(mop.Source(), detailKey)
	if err != nil{
		return err
	}
	detail.DisMissHeight = disMissHeight
	if err := mop.GetBaseOperation().setDetail(mop.Source(), detailKey, detail); err != nil {
		return err
	}
	vf,err := getVoteInfo(mop.GetDb(),mop.Source())
	if err != nil{
		return err
	}
	// if this node is guard node,its has vote info,only set last
	if vf == nil{
		vf = NewVoteInfo(mop.Height())
	}else{
		vf.Last = 1
		vf.UpdateHeight = mop.Height()
	}
	err = setVoteInfo(mop.GetDb(),mop.Source(),vf)
	if err != nil{
		return err
	}
	log.CoreLogger.Infof("apply guard success,address is %s,height is %v",mop.Source().Hex(),mop.Height())
	return nil
}

func(b*BaseMiner)checkStakeAbort(mop mOperation,miner *types.Miner)error{
	if miner == nil {
		return fmt.Errorf("no miner info")
	}
	if miner.IsPrepare() {
		return fmt.Errorf("already in prepare status")
	}
	// Frozen miner must wait for 1 hour after frozen
	if miner.IsFrozen() && mop.Height() <= miner.StatusUpdateHeight+oneHourBlocks {
		return fmt.Errorf("frozen miner can't abort less than 1 hour since frozen")
	}
	return nil
}

func(b*BaseMiner)processStakeAbort(mop mOperation,miner *types.Miner) error{
	remove:=false
	// Remove from pool if active
	if miner.IsActive() {
		mop.GetBaseOperation().removeFromPool(mop.Source(), miner.Stake)
		if mop.GetBaseOperation().opProposalRole() {
			remove = true
		}
	}
	// Update the miner status
	miner.UpdateStatus(types.MinerStatusPrepare, mop.Height())
	if err := mop.GetBaseOperation().setMiner(miner); err != nil {
		return err
	}
	if remove && MinerManagerImpl != nil {
		// Informs MinerManager the removal address
		MinerManagerImpl.proposalRemoveCh <- mop.Source()
	}
	return nil
}

func(b*BaseMiner)processStakeAdd(mop mOperation,targetMiner *types.Miner,checkUpperBound func(miner *types.Miner, height uint64)bool) error{
	err := b.updateBalance(mop,BalanceReduce)
	if err != nil{
		return err
	}
	add := false
	// Already exists
	if targetMiner != nil {
		targetMiner.Stake += mop.Value()
		// check uint64 overflow
		if targetMiner.Stake+mop.Value() < targetMiner.Stake {
			return fmt.Errorf("stake overflow:%v %v", targetMiner.Stake, mop.Value())
		}
	} else {
		targetMiner = initMiner(mop)
	}
	if !checkUpperBound(targetMiner,mop.Height()){
		return fmt.Errorf("stake more than upper bound:%v", targetMiner.Stake)
	}
	if targetMiner.IsActive() {
		// Update proposal total stake
		if mop.GetBaseOperation().opProposalRole() {
			mop.GetBaseOperation().addProposalTotalStake(mop.Value())
		}
	} else if checkCanActivate(targetMiner) { // Check if to active the miner
		targetMiner.UpdateStatus(types.MinerStatusActive, mop.Height())
		// Add to pool so that the miner can start working
		mop.GetBaseOperation().addToPool(mop.Target(), targetMiner.Stake)
		add = true
	}
	// Save miner
	if err := mop.GetBaseOperation().setMiner(targetMiner); err != nil {
		return err
	}
	err = b.updateMinerDetail(mop,targetMiner,types.Staked)
	if err != nil{
		return err
	}

	if add && MinerManagerImpl != nil {
		// Inform added proposer address to minerManager
		MinerManagerImpl.proposalAddCh <- mop.Target()
	}
	return nil
}


func(b*BaseMiner)updateMinerDetail(mop mOperation,targetMiner *types.Miner,stakeStatus types.StakeStatus)error{
	// Set detail of the target account: who stakes from me
	detailKey := getDetailKey(mop.Source(), mop.GetMinerType(), stakeStatus)
	detail, err := mop.GetBaseOperation().getDetail(mop.Target(), detailKey)
	if err != nil {
		return fmt.Errorf("get target detail error:%v", err)
	}
	if detail != nil {
		if detail.Value+mop.Value() < detail.Value {
			return fmt.Errorf("stake detail value overflow:%v %v", detail.Value, mop.Value())
		}
		detail.Value += mop.Value()
	} else {
		detail = &stakeDetail{
			Value: mop.Value(),
		}
	}
	// Update height
	detail.Height = mop.Height()
	if err := mop.GetBaseOperation().setDetail(mop.Target(), detailKey, detail); err != nil {
		return err
	}
	return nil
}


func(b*BaseMiner)updateBalance(mop mOperation,balanceOp BalanceOp)error{
	if BalanceReduce == balanceOp{
		amount := new(big.Int).SetUint64(mop.Value())
		if needTransfer(amount) {
			if !mop.GetDb().CanTransfer(mop.Source(), amount) {
				return fmt.Errorf("balance not enough")
			}
			// Sub the balance of source account
			mop.GetDb().SubBalance(mop.Source(), amount)
		}
	}else{
		fmt.Errorf("unknow balance update opertation")
	}
	return nil
}

func (u *UnSupportMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	return fmt.Errorf("unSupported this op,%v",op)
}
func(u*UnSupportMiner)updateMinerInfo(mop mOperation,targetMiner *types.Miner,op MinerOp)error{
	return fmt.Errorf("unSupported updateMinerInfo")
}
func(u*UnSupportMiner)updateMinerDetail(mop mOperation,targetMiner *types.Miner,stakeStatus types.StakeStatus)error{
	return fmt.Errorf("unSupported updateMinerDetail")
}
func(u*UnSupportMiner)updateBalance(mop mOperation,balanceOp BalanceOp)error{
	return fmt.Errorf("unSupported update balance")
}

func (u *UnSupportMiner)processVoteMinerPool(mop mOperation,targetMiner *types.Miner)error{
	return fmt.Errorf("unSupported vote")
}