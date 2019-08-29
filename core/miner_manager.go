//   Copyright (C) 2018 ZVChain
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
	"github.com/zvchain/zvchain/log"
	"sync"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

const (
	heavyMinerNetTriggerInterval = 10
	buildVirtualNetRoutineName   = "build_virtual_net"
	checkInterval                = 1000
)

var MinerManagerImpl *MinerManager

// MinerManager manage all the miner related actions
type MinerManager struct {
	existingProposal map[string]struct{} // Existing proposal addresses

	proposalAddCh    chan common.Address // Received when miner active operation happens
	proposalRemoveCh chan common.Address // Receiver when miner deactive operation such as miner-abort or frozen happens

	ticker *ticker.GlobalTicker
	lock   sync.RWMutex
}

func initMinerManager(ticker *ticker.GlobalTicker) {
	MinerManagerImpl = &MinerManager{
		existingProposal: make(map[string]struct{}),
		proposalAddCh:    make(chan common.Address),
		proposalRemoveCh: make(chan common.Address),
		ticker:           ticker,
	}

	MinerManagerImpl.ticker.RegisterPeriodicRoutine(buildVirtualNetRoutineName, MinerManagerImpl.updateProposalAddressRoutine, heavyMinerNetTriggerInterval)

	go MinerManagerImpl.listenProposalUpdate()
}

// GuardNodesCheck check guard nodes is expired
func (mm *MinerManager) GuardNodesCheck(accountDB types.AccountDB, bh *types.BlockHeader) {
	snapshot := accountDB.Snapshot()
	expiredAddresses,err := mm.FullStakeGuardNodesCheck(accountDB, bh.Height)
	if err != nil {
		accountDB.RevertToSnapshot(snapshot)
		log.CoreLogger.Errorf("check full guard node error,error is %s", err.Error())
	}else{
		if expiredAddresses != nil && len(expiredAddresses) > 0{
			for _,expiredAddr := range expiredAddresses{
				err = mm.processGuardNodeExpired(accountDB, expiredAddr, bh.Height)
				if err != nil {
					accountDB.RevertToSnapshot(snapshot)
					log.CoreLogger.Errorf("check full guard node error,error is %s", err.Error())
				}
			}
		}
	}
	snapshot = accountDB.Snapshot()
	allExpiredFundAddresses,err := mm.FundGuardExpiredCheck(accountDB, bh.Height)
	if err != nil {
		accountDB.RevertToSnapshot(snapshot)
		log.CoreLogger.Errorf("check fund guard node error,error is %s", err.Error())
	}else{
		for _,expiredAddr  := range allExpiredFundAddresses{
			err = guardNodeExpired(accountDB, expiredAddr, bh.Height, true)
			if err !=  nil{
				accountDB.RevertToSnapshot(snapshot)
				log.CoreLogger.Errorf("check fund guard node error,error is %s", err.Error())
			}
		}
	}
}

func (mm *MinerManager) FundGuardExpiredCheck(accountDB types.AccountDB, height uint64) ([]common.Address,error) {
	allExpiredFundAddresses  := []common.Address{}
	expiredAddresses,err := mm.fundGuardSixAddFiveNodesCheck(accountDB, height)
	if err != nil {
		return nil,err
	}
	if expiredAddresses != nil{
		allExpiredFundAddresses = append(allExpiredFundAddresses,expiredAddresses...)
	}
	expiredAddresses,err = mm.fundGuardSixAddSixNodesCheck(accountDB, height)
	if err != nil {
		return nil,err
	}
	if expiredAddresses != nil{
		allExpiredFundAddresses = append(allExpiredFundAddresses,expiredAddresses...)
	}
	return allExpiredFundAddresses,nil
}

func (mm *MinerManager) fundGuardSixAddFiveNodesCheck(accountDB types.AccountDB, height uint64) ([]common.Address,error) {
	if height < adjustWeightPeriod/2 || height > adjustWeightPeriod*2 {
		return nil,nil
	}
	if height%checkInterval != 0 {
		return nil,nil
	}
	log.CoreLogger.Infof("begin scan 6+5 mode")
	hasScanned := hasScanedSixAddFiveFundGuards(accountDB)
	if hasScanned {
		log.CoreLogger.Infof("begin scan 6+5 mode,find has scanned")
		return nil,nil
	}
	fds, err := mm.GetAllFundStakeGuardNodes(accountDB)
	if err != nil {
		return nil,err
	}
	expiredAddresses := []common.Address{}
	for _, fd := range fds {
		if !fd.isFundGuard() {
			continue
		}
		if !fd.isSixAddFive() {
			continue
		}
		err = updateFundGuardPoolStatus(accountDB, fd.Address, normalNodeType, height)
		if err != nil {
			return nil,err
		}
		expiredAddresses = append(expiredAddresses,fd.Address)
	}
	markScanedSixAddFiveFundGuards(accountDB)
	log.CoreLogger.Infof("scan 6+5 over")
	return expiredAddresses,nil
}

func (mm *MinerManager) fundGuardSixAddSixNodesCheck(accountDB types.AccountDB, height uint64) ([]common.Address,error) {
	if height < adjustWeightPeriod || height > adjustWeightPeriod*3 {
		return nil,nil
	}
	if height%checkInterval != 0 {
		return nil,nil
	}
	log.CoreLogger.Infof("begin scan 6+6 mode")
	hasScanned := hasScanedSixAddSixFundGuards(accountDB)
	if hasScanned {
		log.CoreLogger.Infof("begin scan 6+6 mode,has scanned")
		return nil,nil
	}
	fds, err := mm.GetAllFundStakeGuardNodes(accountDB)
	if err != nil {
		return nil,err
	}
	expiredAddresses := []common.Address{}
	for _, fd := range fds {
		if !fd.isFundGuard() {
			continue
		}
		if !fd.isSixAddSix() {
			continue
		}
		err = updateFundGuardPoolStatus(accountDB, fd.Address, normalNodeType, height)
		if err != nil {
			return nil,err
		}
		expiredAddresses = append(expiredAddresses,fd.Address)
	}
	markScanedSixAddSixFundGuards(accountDB)
	log.CoreLogger.Infof("scan 6+6 mode over")
	return expiredAddresses,nil
}

func (mm *MinerManager) FullStakeGuardNodesCheck(db types.AccountDB, height uint64) ([]common.Address,error){
	if height < adjustWeightPeriod/2 {
		return nil,nil
	}
	if height%checkInterval != 0 {
		return nil,nil
	}
	log.CoreLogger.Infof("begin process full stake guard nodes")
	fullStakeAddress := mm.GetAllFullStakeGuardNodes(db)
	if fullStakeAddress == nil || len(fullStakeAddress) == 0 {
		return nil,nil
	}
	expiredAddresses := []common.Address{}
	var err error
	var isExpired bool
	for _, addr := range fullStakeAddress {
		isExpired,err = mm.checkFullStakeGuardNodeExpired(db, addr, height)
		if err != nil {
			return nil,err
		}
		if isExpired{
			expiredAddresses = append(expiredAddresses,addr)
		}
	}
	return expiredAddresses,nil
}

func (mm *MinerManager) GetTickets(db types.AccountDB, address common.Address) uint64 {
	return getTickets(db, address)
}

func (mm *MinerManager) GetValidTicketsByHeight(height uint64) uint64 {
	return getValidTicketsByHeight(height)
}

func (mm *MinerManager) checkFullStakeGuardNodeExpired(db types.AccountDB, address common.Address, height uint64) (bool,error) {
	detailKey := getDetailKey(address, types.MinerTypeProposal, types.Staked)
	stakedDetail, err := getDetail(db, address, detailKey)
	if err != nil {
		return false,err
	}
	if stakedDetail == nil {
		return false,fmt.Errorf("check guard nodes,find stake detail is nil,address is %s", address.String())
	}
	if height > (stakedDetail.DisMissHeight + stakeBuffer) {
		return true,nil
	}
	if stakedDetail.MarkNotFullHeight > 0 {
		if height > stakedDetail.MarkNotFullHeight+stakeBuffer {
			return true,nil
		}
	} else {
		if !isFullStake(stakedDetail.Value, height) {
			stakedDetail.MarkNotFullHeight = height
			err = setDetail(db, address, detailKey, stakedDetail)
			if err != nil {
				return false,err
			}
			return false,nil
		}
	}
	return false,nil
}

func (mm *MinerManager) processGuardNodeExpired(db types.AccountDB, address common.Address, height uint64) error {
	err := guardNodeExpired(db, address, height, false)
	if err != nil {
		return fmt.Errorf("processGuardNodeExpired error :%v", err)
	}
	return nil
}

func (mm *MinerManager) executeOperation(operation mOperation, accountDB types.AccountDB) (success bool, err error) {
	if err = operation.ParseTransaction(); err != nil {
		return
	}
	snapshot := accountDB.Snapshot()
	if ret := operation.Transition(); ret.err != nil {
		accountDB.RevertToSnapshot(snapshot)
		return false, ret.err
	}
	return true, nil

}

// ClearTicker clear the ticker routine
func (mm *MinerManager) ClearTicker() {
	if mm.ticker == nil {
		return
	}
	mm.ticker.ClearRoutines()
}

// ExecuteOperation execute the miner operation
func (mm *MinerManager) ExecuteOperation(accountDB types.AccountDB, msg types.TxMessage, height uint64) (success bool, err error) {
	ss := newTransitionContext(accountDB, msg, nil, height)
	op := getOpByType(ss, msg.OpType())
	return mm.executeOperation(op, accountDB)
}

// FreezeMiner execute the miner frozen operation
func (mm *MinerManager) MinerFrozen(accountDB types.AccountDB, miner common.Address, height uint64) (success bool, err error) {
	base := newTransitionContext(accountDB, nil, nil, height)
	operation := &minerFreezeOp{transitionContext: base, addr: miner}
	return mm.executeOperation(operation, accountDB)
}

func (mm *MinerManager) MinerPenalty(accountDB types.AccountDB, penalty types.PunishmentMsg, height uint64) (success bool, err error) {
	base := newTransitionContext(accountDB, nil, nil, height)
	operation := &minerPenaltyOp{
		transitionContext: base,
		targets:           make([]common.Address, len(penalty.PenaltyTarget())),
		rewards:           make([]common.Address, len(penalty.RewardTarget())),
		value:             minimumStake(),
	}
	for i, id := range penalty.PenaltyTarget() {
		operation.targets[i] = common.BytesToAddress(id)
	}
	for i, id := range penalty.RewardTarget() {
		operation.rewards[i] = common.BytesToAddress(id)
	}
	return mm.executeOperation(operation, accountDB)
}

// GetMiner return the latest miner info stored in db of the given address and the miner type
func (mm *MinerManager) GetLatestMiner(address common.Address, mType types.MinerType) *types.Miner {
	accontDB, err := BlockChainImpl.LatestAccountDB()
	if err != nil {
		Logger.Errorf("get accontDB failed,error = %v", err.Error())
		return nil
	}
	miner, err := getMiner(accontDB, address, mType)
	if err != nil {
		Logger.Errorf("get miner by id error:%v", err)
		return nil
	}
	return miner
}

func (mm *MinerManager) GetFullMinerPoolStake(height uint64) uint64 {
	return getFullMinerPoolStake(height)
}

// GetMiner return miner info stored in db of the given address and the miner type at the given height
func (mm *MinerManager) GetMiner(address common.Address, mType types.MinerType, height uint64) *types.Miner {
	db, err := BlockChainImpl.AccountDBAt(height)
	if err != nil {
		Logger.Errorf("AccountDBAt error:%v, height:%v", err, height)
		return nil
	}
	miner, err := getMiner(db, address, mType)
	if err != nil {
		Logger.Errorf("get miner by id error:%v", err)
		return nil
	}
	return miner
}

// GetProposalTotalStake returns the chain's total staked value of proposals at the specific block height
func (mm *MinerManager) GetProposalTotalStake(height uint64) uint64 {
	accountDB, err := BlockChainImpl.AccountDBAt(height)
	if err != nil {
		Logger.Errorf("Get account db by height %v error:%s", height, err.Error())
		return 0
	}

	return getProposalTotalStake(accountDB)
}

func (mm *MinerManager) GetAllFullStakeGuardNodes(accountDB types.AccountDB) []common.Address {
	var addrs []common.Address
	iter := accountDB.DataIterator(common.FullStakeGuardNodeAddr, common.KeyGuardNodes)
	if iter != nil {
		for iter.Next() {
			addr := common.BytesToAddress(iter.Key[len(common.KeyGuardNodes):])
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

func (mm *MinerManager) GetAllFundStakeGuardNodes(accountDB types.AccountDB) ([]*fundGuardNodeDetail, error) {
	var fds []*fundGuardNodeDetail
	iter := accountDB.DataIterator(common.FundGuardNodeAddr, common.KeyGuardNodes)
	for iter.Next() {
		addr := common.BytesToAddress(iter.Key[len(common.KeyGuardNodes):])
		bytes := iter.Value
		var fn fundGuardNode
		err := msgpack.Unmarshal(bytes, &fn)
		if err != nil {
			return nil, fmt.Errorf("get fund guard nodes error,error = %s", err.Error())
		}
		fds = append(fds, &fundGuardNodeDetail{Address: addr, fundGuardNode: &fn})
	}
	return fds, nil
}

// GetAllMiners returns all miners of the the specified type at the given height
func (mm *MinerManager) GetAllMiners(mType types.MinerType, height uint64) []*types.Miner {
	accountDB, err := BlockChainImpl.AccountDBAt(height)
	if err != nil {
		Logger.Errorf("Get account db by height %v error:%s", height, err.Error())
		return nil
	}
	var prefix []byte
	if types.IsVerifyRole(mType) {
		prefix = common.PrefixPoolVerifier
	} else {
		prefix = common.PrefixPoolProposal
	}
	iter := accountDB.DataIterator(common.MinerPoolAddr, prefix)
	miners := make([]*types.Miner, 0)
	for iter.Next() {
		addr := common.BytesToAddress(iter.Key[len(prefix):])
		miner, err := getMiner(accountDB, addr, mType)
		if err != nil {
			Logger.Errorf("get all miner error:%v, addr:%v", err, addr.AddrPrefixString())
			return nil
		}
		if miner != nil {
			miners = append(miners, miner)
		}
	}
	return miners
}

func (mm *MinerManager) getStakeDetail(address, source common.Address, status types.StakeStatus, mType types.MinerType) *types.StakeDetail {
	db, error := BlockChainImpl.LatestAccountDB()
	if error != nil {
		Logger.Errorf("get accountdb failed,error = %v", error.Error())
		return nil
	}
	key := getDetailKey(source, mType, status)
	detail, err := getDetail(db, address, key)
	if err != nil {
		Logger.Error("get detail error:", err)
	}
	if detail != nil {
		return &types.StakeDetail{
			Source:        source,
			Target:        address,
			Value:         detail.Value,
			UpdateHeight:  detail.Height,
			Status:        status,
			MType:         mType,
			DisMissHeight: detail.DisMissHeight,
		}
	}
	return nil
}

// GetStakeDetails returns all the stake details of the given address pairs
func (mm *MinerManager) GetStakeDetails(address common.Address, source common.Address) []*types.StakeDetail {
	result := make([]*types.StakeDetail, 0)

	detail := mm.getStakeDetail(address, source, types.Staked, types.MinerTypeVerify)
	if detail != nil {
		result = append(result, detail)
	}
	detail = mm.getStakeDetail(address, source, types.StakeFrozen, types.MinerTypeVerify)
	if detail != nil {
		result = append(result, detail)
	}
	detail = mm.getStakeDetail(address, source, types.Staked, types.MinerTypeProposal)
	if detail != nil {
		result = append(result, detail)
	}
	detail = mm.getStakeDetail(address, source, types.StakeFrozen, types.MinerTypeProposal)
	if detail != nil {
		result = append(result, detail)
	}
	return result
}

// GetAllStakeDetails returns all stake details of the given account
func (mm *MinerManager) GetAllStakeDetails(address common.Address) map[string][]*types.StakeDetail {
	ret := make(map[string][]*types.StakeDetail)
	accontDB, error := BlockChainImpl.LatestAccountDB()
	if error != nil {
		Logger.Errorf("get accountdb failed,err = %v", error.Error())
		return ret
	}
	iter := accontDB.DataIterator(address, common.PrefixDetail)
	if iter == nil {
		return nil
	}
	for iter.Next() {
		// finish the iterator
		if !bytes.HasPrefix(iter.Key, common.PrefixDetail) {
			break
		}
		addr, mt, st := parseDetailKey(iter.Key)
		sd, err := parseDetail(iter.Value)
		if err != nil {
			Logger.Errorf("parse detail error:%v", err)
		}
		detail := &types.StakeDetail{
			Source:       addr,
			Target:       address,
			Value:        sd.Value,
			UpdateHeight: sd.Height,
			Status:       st,
			MType:        mt,
		}
		var (
			ds []*types.StakeDetail
			ok bool
		)
		if ds, ok = ret[addr.AddrPrefixString()]; !ok {
			ds = make([]*types.StakeDetail, 0)
		}
		ds = append(ds, detail)
		ret[addr.AddrPrefixString()] = ds
	}
	return ret
}

func (mm *MinerManager) loadAllProposalAddress() map[string]struct{} {
	mp := make(map[string]struct{})
	accountDB, error := BlockChainImpl.LatestAccountDB()
	if error != nil {
		Logger.Errorf("get accountdb failed,error = %v", error.Error())
		return mp
	}
	prefix := common.PrefixPoolProposal
	iter := accountDB.DataIterator(common.MinerPoolAddr, prefix)
	for iter != nil && iter.Next() {
		if !bytes.HasPrefix(iter.Key, prefix) {
			break
		}
		addr := common.BytesToAddress(iter.Key[len(prefix):])
		mp[addr.AddrPrefixString()] = struct{}{}
	}
	return mp
}

// GetAllProposalAddresses returns all proposal miner addresses
func (mm *MinerManager) GetAllProposalAddresses() []string {
	mm.lock.RLock()
	defer mm.lock.RUnlock()
	return mm.getAllProposalAddresses()
}

func (mm *MinerManager) getAllProposalAddresses() []string {
	mems := make([]string, 0)
	for addr := range mm.existingProposal {
		mems = append(mems, addr)
	}
	return mems
}

func (mm *MinerManager) listenProposalUpdate() {
	for {
		select {
		case addr := <-mm.proposalAddCh:
			mm.lock.Lock()
			if _, ok := mm.existingProposal[addr.AddrPrefixString()]; !ok {
				mm.existingProposal[addr.AddrPrefixString()] = struct{}{}
				Logger.Debugf("Add proposer %v", addr.AddrPrefixString())
			}
			mm.lock.Unlock()
		case addr := <-mm.proposalRemoveCh:
			mm.lock.Lock()
			if _, ok := mm.existingProposal[addr.AddrPrefixString()]; ok {
				delete(mm.existingProposal, addr.AddrPrefixString())
				Logger.Debugf("Remove proposer %v", addr.AddrPrefixString())
			}
			mm.lock.Unlock()
		}
	}
}

func (mm *MinerManager) buildVirtualNetRoutine() {
	addrs := mm.getAllProposalAddresses()
	Logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(addrs))
	if network.GetNetInstance() != nil {
		network.GetNetInstance().BuildGroupNet(network.FullNodeVirtualGroupID, addrs)
	}
}

func (mm *MinerManager) updateProposalAddressRoutine() bool {
	addresses := mm.loadAllProposalAddress()

	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.existingProposal = addresses
	mm.buildVirtualNetRoutine()
	return true
}

func (mm *MinerManager) addGenesisMinerStake(miner *types.Miner, db types.AccountDB) {
	pks := &types.MinerPks{
		MType: miner.Type,
		Pk:    miner.PublicKey,
		VrfPk: miner.VrfPublicKey,
	}
	data, err := types.EncodePayload(pks)
	if err != nil {
		panic(fmt.Errorf("encode payload error:%v", err))
	}
	addr := common.BytesToAddress(miner.ID)
	raw := &types.RawTransaction{
		Source: &addr,
		Value:  types.NewBigInt(miner.Stake),
		Target: &addr,
		Type:   types.TransactionTypeStakeAdd,
		Data:   data,
	}
	tx := types.NewTransaction(raw, raw.GenHash())
	err = mm.ValidateStakeAdd(tx)
	if err != nil {
		panic(fmt.Errorf("add genesis miner validate error:%v", err))
	}

	_, err = mm.ExecuteOperation(db, tx, 0)
	if err != nil {
		panic(fmt.Errorf("add genesis miner error:%v", err))
	}
	// Add nonce or else the account maybe marked as deleted because zero nonce, zero balance, empty data
	nonce := db.GetNonce(addr)
	db.SetNonce(addr, nonce+1)
}

func (mm *MinerManager) ValidateStakeAdd(tx *types.Transaction) error {
	if len(tx.Data) == 0 {
		return fmt.Errorf("payload length error")
	}
	if tx.Target == nil {
		return fmt.Errorf("target is nil")
	}
	if tx.Value == nil {
		return fmt.Errorf("amount is nil")
	}
	if !tx.Value.Value().IsUint64() {
		return fmt.Errorf("amount type not uint64")
	}
	return nil
}

func (mm *MinerManager) addGenesesMiners(miners []*types.Miner, accountDB types.AccountDB) {
	for _, miner := range miners {
		// Add as verifier
		miner.Type = types.MinerTypeVerify
		mm.addGenesisMinerStake(miner, accountDB)
		// Add as proposer
		miner.Type = types.MinerTypeProposal
		mm.addGenesisMinerStake(miner, accountDB)
	}
}

func (mm *MinerManager) genFundGuardNodes(accountDB types.AccountDB) {
	for _, addr := range types.ExtractGuardNodes {
		miner := &types.Miner{ID: addr.Bytes(), Type: types.MinerTypeProposal, Identity: types.MinerGuard, Status: types.MinerStatusPrepare, ApplyHeight: 0, Stake: 0}
		bs, err := msgpack.Marshal(miner)
		if err != nil {
			panic("encode miner failed")
		}
		accountDB.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)
		err = addFundGuardPool(accountDB, addr)
		if err != nil {
			panic("encode fund guard failed")
		}
		nonce := accountDB.GetNonce(addr)
		accountDB.SetNonce(addr, nonce+1)
	}
}
