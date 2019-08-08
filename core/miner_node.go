package core

import (
	"fmt"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

type MinerOp = byte
type BalanceOp = byte
const (
	StakedAddOp          		MinerOp = iota
	StakeAbortOp
	StakeReduceOp
	StakeRefundOp
)

const (
	BalanceAdd          		BalanceOp = iota
	BalanceReduce
	BalanceNochanged
)

type baseIdentityOp interface {
	processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error
	updateMinerDetail(mop mOperation,targetMiner *types.Miner,stakeStatus types.StakeStatus)error
	updateBalance(mop mOperation,balanceOp BalanceOp)error
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
	case StakeReduceOp:
	case StakeRefundOp:
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func(n*NormalProposalMiner)checkStakeAdd(mop mOperation,targetMiner *types.Miner)error{
	// only admin can stake to normal miner node
	if mop.Source() !=  mop.Target() && mop.Source() !=  adminAddrType{
		return fmt.Errorf("could not stake to other's verify node")
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
	case StakeReduceOp:
	case StakeRefundOp:
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func (m *MinerPoolProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		err := m.checkStakeAdd(mop,targetMiner)
		if err != nil{
			return err
		}
		err = m.processStakeAdd(mop,targetMiner,checkMinerPoolUpperBound)
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
	case StakeReduceOp:
	case StakeRefundOp:
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
}

func (i *InvalidProposalMiner) processMinerOp(mop mOperation,targetMiner *types.Miner,op MinerOp) error {
	switch op {
	case StakedAddOp:
		return fmt.Errorf("invalid miner pool could not support stake add")
	case StakeAbortOp:
		return fmt.Errorf("invalid miner pool could not support stake abort")
	case StakeReduceOp:
	case StakeRefundOp:
	default:
		return fmt.Errorf("unknow operator %v",op)
	}
	return nil
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
		case StakeReduceOp:
		case StakeRefundOp:
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

func(m*MinerPoolProposalMiner)checkStakeAdd(mop mOperation,targetMiner *types.Miner)error{
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
	if BalanceNochanged == balanceOp{
		return nil
	}else if BalanceReduce == balanceOp{
		amount := new(big.Int).SetUint64(mop.Value())
		if needTransfer(amount) {
			if !mop.GetDb().CanTransfer(mop.Source(), amount) {
				return fmt.Errorf("balance not enough")
			}
			// Sub the balance of source account
			mop.GetDb().SubBalance(mop.Source(), amount)
		}
	}
	return fmt.Errorf("unknow balance update opertation")
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