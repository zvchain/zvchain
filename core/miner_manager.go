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
	"fmt"
	"github.com/zvchain/zvchain/storage/account"
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/vm"

	"github.com/zvchain/zvchain/middleware/ticker"
)

const (
	heavyMinerNetTriggerInterval = 30
)

const (
	MinMinerStake             = 500 * common.TAS // minimal token of miner can stake
	MaxMinerStakeAdjustPeriod = 5000000          // maximal token of miner can stake
	initialMinerNodesAmount   = 200              // The number initial of miner nodes envisioned
	MoreMinerNodesPerHalfYear = 12               // The number of increasing nodes per half year
	initialTokenReleased      = 500000000        // The initial amount of tokens released
	tokenReleasedPerHalfYear  = 400000000        //  The amount of tokens released per half year
	stakeAdjustTimes          = 24               // stake adjust times
)

var MinerManagerImpl *MinerManager

// MinerManager manage all the miner related actions
type MinerManager struct {
	proposalAddresses map[string]struct{}
	proposalAddCh     chan common.Address
	proposalRemoveCh  chan common.Address
	ticker            *ticker.GlobalTicker
	lock              sync.RWMutex
}

func initMinerManager(ticker *ticker.GlobalTicker) {
	MinerManagerImpl = &MinerManager{
		proposalAddresses: make(map[string]struct{}, 0),
		proposalAddCh:     make(chan common.Address),
		proposalRemoveCh:  make(chan common.Address),
		ticker:            ticker,
	}

	MinerManagerImpl.ticker.RegisterPeriodicRoutine("build_virtual_net", MinerManagerImpl.updateProposalAddressRoutine, heavyMinerNetTriggerInterval)
	MinerManagerImpl.ticker.StartTickerRoutine("build_virtual_net", false)

	go MinerManagerImpl.listenProposalUpdate()
}

// ExecuteOperation execute the miner operation
func (mm *MinerManager) ExecuteOperation(accountDB vm.AccountDB, msg vm.MinerOperationMessage, height uint64) (success bool, err error) {
	operation := newOperation(accountDB.(*account.AccountDB), msg, height)
	if operation == nil {
		err = fmt.Errorf("new operation nil")
		return
	}
	if err = operation.Validate(); err != nil {
		return
	}
	if err = operation.ParseTransaction(); err != nil {
		return
	}
	snapshot := accountDB.Snapshot()
	if err = operation.Operation(); err != nil {
		accountDB.RevertToSnapshot(snapshot)
		return
	}
	return true, nil
}

// GetMiner return the latest miner info stored in db of the given address and the miner type
func (mm *MinerManager) GetLatestMiner(address common.Address, mType types.MinerType) *types.Miner {
	miner, err := getMiner(BlockChainImpl.LatestStateDB(), address, mType)
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

func (mm *MinerManager) GetStakeDetails(address common.Address, source common.Address) []*types.StakeDetail {
	result := make([]*types.StakeDetail, 0)

	db := BlockChainImpl.LatestStateDB()
	key := getDetailKey(address, types.MinerTypeVerify, types.Staked)
	detail, err := getDetail(db, address, key)
	if err != nil {
		Logger.Errorf("get detail error:", err)
	}
	if detail != nil {
		result = append(result, detail)
	}
	getDetailKey(address, types.MinerTypeVerify, types.StakeFrozen)
	getDetailKey(address, types.MinerTypeProposal, types.Staked)
	getDetailKey(address, types.MinerTypeProposal, types.StakeFrozen)

}

func (mm *MinerManager) loadAllProposalAddress() map[string]struct{} {
	accountDB := BlockChainImpl.LatestStateDB()

	prefix := prefixPoolProposal
	iter := accountDB.AsAccountDBTS().DataIteratorSafe(minerPoolAddr, prefix)
	mp := make(map[string]struct{})
	for iter.Next() {
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
	mems := make([]string, len(mm.proposalAddresses))
	for addr := range mm.proposalAddresses {
		mems = append(mems, addr)
	}
	return mems
}

func (mm *MinerManager) listenProposalUpdate() {
	for {
		select {
		case addr := <-mm.proposalAddCh:
			mm.lock.Lock()
			if _, ok := mm.proposalAddresses[addr.Hex()]; !ok {
				mm.proposalAddresses[addr.Hex()] = struct{}{}
				mm.buildVirtualNetRoutine()
			}
			mm.lock.Unlock()
		case addr := <-mm.proposalRemoveCh:
			mm.lock.Lock()
			if _, ok := mm.proposalAddresses[addr.Hex()]; ok {
				delete(mm.proposalAddresses, addr.Hex())
				mm.buildVirtualNetRoutine()
			}
			mm.lock.Unlock()
		}
	}
}

func (mm *MinerManager) buildVirtualNetRoutine() {
	addrs := mm.getAllProposalAddresses()
	network.GetNetInstance().BuildGroupNet(network.FullNodeVirtualGroupID, addrs)
	Logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(addrs))
}

func (mm *MinerManager) updateProposalAddressRoutine() bool {
	addresses := mm.loadAllProposalAddress()

	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.proposalAddresses = addresses
	mm.buildVirtualNetRoutine()
	return true
}

func (mm *MinerManager) addGenesesMiner(miners []*types.Miner, accountDB vm.AccountDB) {
	for _, miner := range miners {
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
		_, err = mm.ExecuteOperation(accountDB, tx, 0)
		if err != nil {
			panic(fmt.Errorf("add genesis miner error:%v", err))
		}
	}
}

// MinerManager shows miner can stake the min value
func (mm *MinerManager) MinStake() uint64 {
	return MinMinerStake
}

// MaxStake shows miner can stake the max value
func (mm *MinerManager) MaxStake(height uint64) uint64 {
	period := height / MaxMinerStakeAdjustPeriod
	if period > stakeAdjustTimes {
		period = stakeAdjustTimes
	}
	nodeAmount := initialMinerNodesAmount + period*MoreMinerNodesPerHalfYear
	return mm.tokenReleased(height) / nodeAmount * common.TAS
}

func (mm *MinerManager) tokenReleased(height uint64) uint64 {
	adjustTimes := height / MaxMinerStakeAdjustPeriod
	if adjustTimes > stakeAdjustTimes {
		adjustTimes = stakeAdjustTimes
	}

	var released uint64 = initialTokenReleased
	for i := uint64(0); i < adjustTimes; i++ {
		halveTimes := i * MaxMinerStakeAdjustPeriod / halveRewardsPeriod
		if halveTimes > halveRewardsTimes {
			halveTimes = halveRewardsTimes
		}
		released += tokenReleasedPerHalfYear >> halveTimes
	}
	return released
}
