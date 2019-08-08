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
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/vmihailenco/msgpack"
)

const (
	heavyMinerNetTriggerInterval = 10
	buildVirtualNetRoutineName   = "build_virtual_net"
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

func (mm *MinerManager) executeOperation(operation mOperation, accountDB types.AccountDB) (success bool, err error) {
	if err = operation.Validate(); err != nil {
		return
	}
	if err = operation.ParseTransaction(); err != nil {
		return
	}
	snapshot := accountDB.Snapshot()
	if ret := operation.Transition(); ret.err != nil {
		accountDB.RevertToSnapshot(snapshot)
		return
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
func (mm *MinerManager) ExecuteOperation(accountDB types.AccountDB, msg types.MinerOperationMessage, height uint64) (success bool, err error) {
	operation := newOperation(accountDB, msg, height)
	return mm.executeOperation(operation, accountDB)
}


// FreezeMiner execute the miner frozen operation
func (mm *MinerManager) MinerFrozen(accountDB types.AccountDB, miner common.Address, height uint64) (success bool, err error) {
	base := newBaseOperation(accountDB, nil, height,nil)
	base.minerType = types.MinerTypeVerify
	operation := &minerFreezeOp{baseOperation: base, addr: miner}
	return mm.executeOperation(operation, accountDB)
}

func (mm *MinerManager) MinerPenalty(accountDB types.AccountDB, penalty types.PunishmentMsg, height uint64) (success bool, err error) {
	base := newBaseOperation(accountDB, nil, height,nil)
	base.minerType = types.MinerTypeVerify
	operation := &minerPenaltyOp{
		baseOperation: base,
		targets:       make([]common.Address, len(penalty.PenaltyTarget())),
		rewards:       make([]common.Address, len(penalty.RewardTarget())),
		value:         minimumStake(),
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
	accontDB,err := BlockChainImpl.LatestStateDB()
	if err != nil {
		Logger.Errorf("get accontDB failed,error = %v",err.Error())
		return nil
	}
	miner, err := getMiner(accontDB, address, mType)
	if err != nil {
		Logger.Errorf("get miner by id error:%v", err)
		return nil
	}
	return miner
}

// GetMiner return miner info stored in db of the given address and the miner type at the given height
func (mm *MinerManager) GetMiner(address common.Address, mType types.MinerType, height uint64) *types.Miner {
	db, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Errorf("GetAccountDBByHeight error:%v, height:%v", err, height)
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
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	return getProposalTotalStake(accountDB.AsAccountDBTS())
}

// GetAllMiners returns all miners of the the specified type at the given height
func (mm *MinerManager) GetAllMiners(mType types.MinerType, height uint64) []*types.Miner {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	var prefix []byte
	if types.IsVerifyRole(mType) {
		prefix = prefixPoolVerifier
	} else {
		prefix = prefixPoolProposal
	}
	iter := accountDB.AsAccountDBTS().DataIteratorSafe(minerPoolAddr, prefix)
	miners := make([]*types.Miner, 0)
	for iter.Next() {
		addr := common.BytesToAddress(iter.Key[len(prefix):])
		miner, err := getMiner(accountDB, addr, mType)
		if err != nil {
			Logger.Errorf("get all miner error:%v, addr:%v", err, addr.Hex())
			return nil
		}
		if miner != nil {
			miners = append(miners, miner)
		}
	}
	return miners
}

func (mm *MinerManager) getStakeDetail(address, source common.Address, status types.StakeStatus, mType types.MinerType) *types.StakeDetail {
	db,error := BlockChainImpl.LatestStateDB()
	if error != nil{
		Logger.Errorf("get accountdb failed,error = %v",error.Error())
		return  nil
	}
	key := getDetailKey(source, mType, status)
	detail, err := getDetail(db, address, key)
	if err != nil {
		Logger.Error("get detail error:", err)
	}
	if detail != nil {
		return &types.StakeDetail{
			Source:       source,
			Target:       address,
			Value:        detail.Value,
			UpdateHeight: detail.Height,
			Status:       status,
			MType:        mType,
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
	accontDB,error := BlockChainImpl.LatestStateDB()
	if error != nil{
		Logger.Errorf("get accountdb failed,err = %v",error.Error())
		return ret
	}
	iter := accontDB.DataIterator(address, prefixDetail)
	if iter == nil{
		return nil
	}
	for iter.Next() {
		// finish the iterator
		if !bytes.HasPrefix(iter.Key, prefixDetail) {
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
		if ds, ok = ret[addr.Hex()]; !ok {
			ds = make([]*types.StakeDetail, 0)
		}
		ds = append(ds, detail)
		ret[addr.Hex()] = ds
	}
	return ret
}

func (mm *MinerManager) loadAllProposalAddress() map[string]struct{} {
	mp := make(map[string]struct{})
	accountDB,error := BlockChainImpl.LatestStateDB()
	if error != nil{
		Logger.Errorf("get accountdb failed,error = %v",error.Error())
		return mp
	}
	prefix := prefixPoolProposal
	iter := accountDB.AsAccountDBTS().DataIteratorSafe(minerPoolAddr, prefix)
	for iter != nil && iter.Next() {
		if !bytes.HasPrefix(iter.Key, prefix) {
			break
		}
		addr := common.BytesToAddress(iter.Key[len(prefix):])
		mp[addr.Hex()] = struct{}{}
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
			if _, ok := mm.existingProposal[addr.Hex()]; !ok {
				mm.existingProposal[addr.Hex()] = struct{}{}
				Logger.Debugf("Add proposer %v", addr.Hex())
			}
			mm.lock.Unlock()
		case addr := <-mm.proposalRemoveCh:
			mm.lock.Lock()
			if _, ok := mm.existingProposal[addr.Hex()]; ok {
				delete(mm.existingProposal, addr.Hex())
				Logger.Debugf("Remove proposer %v", addr.Hex())
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
	tx := &types.Transaction{
		Source: &addr,
		Value:  types.NewBigInt(miner.Stake),
		Target: &addr,
		Type:   types.TransactionTypeStakeAdd,
		Data:   data,
	}
	_, err = mm.ExecuteOperation(db, tx, 0)
	if err != nil {
		panic(fmt.Errorf("add genesis miner error:%v", err))
	}
	// Add nonce or else the account maybe marked as deleted because zero nonce, zero balance, empty data
	nonce := db.GetNonce(addr)
	db.SetNonce(addr, nonce+1)
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

func (mm *MinerManager) genGuardNodes(accountDB types.AccountDB) {
	for _, addr := range extractGuardNodes {
		miner := &types.Miner{ID: addr.Bytes(),Type:types.MinerTypeProposal,Identity:types.MinerGuard,Status:types.MinerStatusActive,ApplyHeight:0,Stake:0}
		bs, err := msgpack.Marshal(miner)
		if err != nil {
			panic("encode miner failed")
		}
		accountDB.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)

		nonce := accountDB.GetNonce(addr)
		accountDB.SetNonce(addr, nonce+1)
	}
}
