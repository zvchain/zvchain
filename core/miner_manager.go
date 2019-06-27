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
	"errors"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/trie"
	"github.com/zvchain/zvchain/storage/vm"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/middleware/ticker"
)

const (
	heavyMinerNetTriggerInterval = 10
	heavyMinerCountKey           = "heavy_miner_count"
	lightMinerCountKey           = "light_miner_count"
)

var (
	emptyValue         [0]byte
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
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

type stakeFlagByte = byte

var MinerManagerImpl *MinerManager

// MinerManager manage all the miner related actions
type MinerManager struct {
	hasNewHeavyMiner bool
	heavyMiners      []string

	ticker *ticker.GlobalTicker
	lock   sync.RWMutex
}

type MinerCountOperation struct {
	Code int
}

type MinerIterator struct {
	iterator *trie.Iterator
	cache    *lru.Cache
}

func initMinerManager(ticker *ticker.GlobalTicker) {
	MinerManagerImpl = &MinerManager{
		hasNewHeavyMiner: true,
		heavyMiners:      make([]string, 0),
		ticker:           ticker,
	}

	MinerManagerImpl.ticker.RegisterPeriodicRoutine("build_virtual_net", MinerManagerImpl.buildVirtualNetRoutine, heavyMinerNetTriggerInterval)
	MinerManagerImpl.ticker.StartTickerRoutine("build_virtual_net", false)
}

func (mm *MinerManager) GetMinerByID(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner {
	if accountdb == nil {
		accountdb = BlockChainImpl.LatestStateDB()
	}
	db := mm.getMinerDatabase(ttype)
	data := accountdb.GetData(db, string(id))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		err := msgpack.Unmarshal(data, &miner)
		if err != nil {
			Logger.Errorf("GetMinerByID Unmarshal error,msg= %s", err.Error())
			return nil
		}
		return &miner
	}
	return nil
}

// GetTotalStake returns the chain's total staked value when the specific block height
func (mm *MinerManager) GetTotalStake(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	iter := mm.minerIterator(types.MinerTypeHeavy, accountDB)
	var total uint64
	for iter.Next() {
		miner, _ := iter.Current()
		if height >= miner.ApplyHeight {
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight {
				total += miner.Stake
			}
		}
	}
	if total == 0 {
		iter = mm.minerIterator(types.MinerTypeHeavy, accountDB)
		for iter.Next() {
			miner, _ := iter.Current()
			Logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}
	return total
}

// GetHeavyMiners returns all heavy miners
func (mm *MinerManager) GetHeavyMiners() []string {
	mm.lock.RLock()
	defer mm.lock.RUnlock()
	mems := make([]string, len(mm.heavyMiners))
	copy(mems, mm.heavyMiners)
	return mems
}

func (mm *MinerManager) MinerIterator(minerType byte, height uint64) *MinerIterator {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	return mm.minerIterator(minerType, accountDB)
}

func (mm *MinerManager) HeavyMinerCount(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	heavyMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, heavyMinerCountKey)
	return common.ByteToUInt64(heavyMinerCountByte)

}

func (mm *MinerManager) LightMinerCount(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	lightMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, lightMinerCountKey)
	return common.ByteToUInt64(lightMinerCountByte)
}

func (mm *MinerManager) buildVirtualNetRoutine() bool {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	if mm.hasNewHeavyMiner {
		iterator := mm.minerIterator(types.MinerTypeHeavy, nil)
		array := make([]string, 0)
		for iterator.Next() {
			miner, _ := iterator.Current()
			gid := groupsig.DeserializeID(miner.ID)
			array = append(array, gid.GetHexString())
		}
		mm.heavyMiners = array
		network.GetNetInstance().BuildGroupNet(network.FullNodeVirtualGroupID, array)
		Logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(array))
		mm.hasNewHeavyMiner = false
	}
	return true
}

func (mm *MinerManager) getMinerDatabase(minerType byte) common.Address {
	switch minerType {
	case types.MinerTypeLight:
		return common.LightDBAddress
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress
	}
	return common.Address{}
}

func (mm *MinerManager) addMiner(id []byte, miner *types.Miner, accountdb vm.AccountDB) int {
	Logger.Debugf("Miner manager add miner %d", miner.Type)
	db := mm.getMinerDatabase(miner.Type)

	if accountdb.GetData(db, string(id)) != nil {
		return -1
	}
	if miner.Stake < mm.MinStake() || miner.Stake > mm.MaxStake(miner.ApplyHeight) {
		return -1
	}
	mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(id), data)
	if miner.Type == types.MinerTypeHeavy {
		mm.hasNewHeavyMiner = true
	}
	return 1
}

func (mm *MinerManager) activateAndAddStakeMiner(miner *types.Miner, accountdb vm.AccountDB, height uint64) bool {
	db := mm.getMinerDatabase(miner.Type)
	minerData := accountdb.GetData(db, string(miner.ID))
	if minerData == nil || len(minerData) == 0 {
		return false
	}
	var dbMiner types.Miner
	err := msgpack.Unmarshal(minerData, &dbMiner)
	if err != nil {
		Logger.Errorf("activateMiner: Unmarshal %d error, ", miner.ID)
		return false
	}
	miner.Stake = dbMiner.Stake + miner.Stake
	if miner.Stake < mm.MinStake() || miner.Stake > mm.MaxStake(height) {
		return false
	}
	miner.Status = types.MinerStatusNormal
	miner.ApplyHeight = height
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(miner.ID), data)
	if miner.Type == types.MinerTypeHeavy {
		mm.hasNewHeavyMiner = true
	}
	mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
	return true
}

func (mm *MinerManager) addGenesesMiner(miners []*types.Miner, accountdb vm.AccountDB) {
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _, miner := range miners {
		if accountdb.GetData(dbh, string(miner.ID)) == nil {
			miner.Type = types.MinerTypeHeavy
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbh, string(miner.ID), data)
			mm.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb)
			mm.heavyMiners = append(mm.heavyMiners, groupsig.DeserializeID(miner.ID).GetHexString())
			mm.updateMinerCount(types.MinerTypeHeavy, minerCountIncrease, accountdb)
		}
		if accountdb.GetData(dbl, string(miner.ID)) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, string(miner.ID), data)
			mm.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb)
			mm.updateMinerCount(types.MinerTypeLight, minerCountIncrease, accountdb)
		}
	}
	mm.hasNewHeavyMiner = true
}

func (mm *MinerManager) removeMiner(id []byte, ttype byte, accountdb vm.AccountDB) {
	Logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabase(ttype)
	accountdb.SetData(db, string(id), emptyValue[:])
}

func (mm *MinerManager) abortMiner(id []byte, ttype byte, height uint64, accountdb vm.AccountDB) bool {
	miner := mm.GetMinerByID(id, ttype, accountdb)
	if miner != nil && miner.Status == types.MinerStatusNormal {
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height

		db := mm.getMinerDatabase(ttype)
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		if ttype == types.MinerTypeHeavy {
			mm.hasNewHeavyMiner = true
		}
		mm.updateMinerCount(ttype, minerCountDecrease, accountdb)
		Logger.Debugf("Miner manager abort miner update success %+v", miner)
		return true
	}
	Logger.Debugf("Miner manager abort miner update fail %+v", miner)
	return false
}

func (mm *MinerManager) minerIterator(minerType byte, accountdb vm.AccountDB) *MinerIterator {
	db := mm.getMinerDatabase(minerType)
	if accountdb == nil {
		accountdb = BlockChainImpl.LatestStateDB()
	}
	iterator := &MinerIterator{iterator: accountdb.DataIterator(db, "")}
	return iterator
}

func (mm *MinerManager) updateMinerCount(minerType byte, operation MinerCountOperation, accountdb vm.AccountDB) {
	if minerType == types.MinerTypeHeavy {
		heavyMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, heavyMinerCountKey)
		heavyMinerCount := common.ByteToUInt64(heavyMinerCountByte)
		if operation == minerCountIncrease {
			heavyMinerCount++
		} else if operation == minerCountDecrease {
			heavyMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, heavyMinerCountKey, common.UInt64ToByte(heavyMinerCount))
		return
	}

	if minerType == types.MinerTypeLight {
		lightMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, lightMinerCountKey)
		lightMinerCount := common.ByteToUInt64(lightMinerCountByte)
		if operation == minerCountIncrease {
			lightMinerCount++
		} else if operation == minerCountDecrease {
			lightMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, lightMinerCountKey, common.UInt64ToByte(lightMinerCount))
		return
	}
	Logger.Error("Unknown miner type:%d", minerType)
}

func (mm *MinerManager) getMinerStakeDetailDatabase() common.Address {
	return common.MinerStakeDetailDBAddress
}

func (mm *MinerManager) getDetailDBKey(from []byte, minerAddr []byte, _type byte, status types.StakeStatus) []byte {
	var pledgFlagByte = (_type << 4) | byte(status)
	key := []byte{stakeFlagByte(pledgFlagByte)}
	key = append(key, minerAddr...)
	key = append(key, from...)
	Logger.Debugf("getDetailDBKey: toHex-> %s", common.ToHex(key))

	/**
	 *	key's available values:
	 *	LightStaked      stakeFlagByte = (types.MinerTypeLight << 4) | byte(Staked)
	 *	LightStakeFrozen stakeFlagByte = (types.MinerTypeLight << 4) | byte(StakeFrozen)
	 *	HeavyStaked      stakeFlagByte = (types.MinerTypeHeavy << 4) | byte(Staked)
	 *	HeavyStakeFrozen stakeFlagByte = (types.MinerTypeHeavy << 4) | byte(StakeFrozen)
	 */

	return key
}

// AddStakeDetail adds the stake detail information into database
func (mm *MinerManager) AddStakeDetail(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool {
	dbAddr := mm.getMinerStakeDetailDatabase()
	key := mm.getDetailDBKey(from, miner.ID, miner.Type, types.Staked)
	detailData := accountdb.GetData(dbAddr, string(key))
	if detailData == nil || len(detailData) == 0 {
		Logger.Debugf("MinerManager.AddStakeDetail set new key: %s, value: %s", common.ToHex(key), common.ToHex(common.Uint64ToByte(amount)))
		accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(amount))
	} else {
		preAmount := common.ByteToUint64(detailData)
		// if overflow
		if preAmount+amount < preAmount {
			Logger.Debug("MinerManager.AddStakeDetail return false(overflow)")
			return false
		}
		Logger.Debugf("MinerManager.AddStakeDetail set key: %s, value: %s", common.ToHex(key), common.ToHex(common.Uint64ToByte(amount)))
		accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(preAmount+amount))
	}
	return true
}

// CancelStake cancels the stake value and update the database
func (mm *MinerManager) CancelStake(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool {
	dbAddr := mm.getMinerStakeDetailDatabase()
	key := mm.getDetailDBKey(from, miner.ID, miner.Type, types.Staked)
	stakedData := accountdb.GetData(dbAddr, string(key))
	if stakedData == nil || len(stakedData) == 0 {
		Logger.Debug("MinerManager.CancelStake false(cannot find stake data)")
		return false
	}
	preStake := common.ByteToUint64(stakedData)
	frozenKey := mm.getDetailDBKey(from, miner.ID, miner.Type, types.StakeFrozen)
	frozenData := accountdb.GetData(dbAddr, string(frozenKey))
	var preFrozen, newFrozen, newStake uint64
	if frozenData == nil || len(frozenData) == 0 {
		preFrozen = 0
	} else {
		preFrozen = common.ByteToUint64(frozenData[:8])
	}
	newStake = preStake - amount
	newFrozen = preFrozen + amount
	if preStake < amount || newFrozen < preFrozen {
		Logger.Debugf("MinerManager.CancelStake return false(overflow or not enough staked: preStake: %d, "+
			"preFrozen: %d, newStake: %d, newFrozen: %d)", preStake, preFrozen, newStake, newFrozen)
		return false
	}
	if newStake == 0 {
		accountdb.RemoveData(dbAddr, string(key))
	} else {
		accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(newStake))
	}
	newFrozenData := common.Uint64ToByte(newFrozen)
	newFrozenData = append(newFrozenData, common.Uint64ToByte(height)...)
	accountdb.SetData(dbAddr, string(frozenKey), newFrozenData)
	Logger.Debugf("MinerManager.CancelStake success from:%s, to: %s, value: %d ", common.ToHex(from), common.ToHex(miner.ID), amount)
	return true
}

// GetLatestCancelStakeHeight returns the block height of the property owner cancel the pledge stake for a miner or
// a validator. The owner can refund the stake after several blocks later after cancel stake
func (mm *MinerManager) GetLatestCancelStakeHeight(from []byte, miner *types.Miner, accountdb vm.AccountDB) uint64 {
	dbAddr := mm.getMinerStakeDetailDatabase()
	frozenKey := mm.getDetailDBKey(from, miner.ID, miner.Type, types.StakeFrozen)
	frozenData := accountdb.GetData(dbAddr, string(frozenKey))
	if frozenData == nil || len(frozenData) == 0 {
		return common.MaxUint64
	}
	return common.ByteToUint64(frozenData[8:])
}

// RefundStake refund the property which was be pledged for a miner or a validator
func (mm *MinerManager) RefundStake(from []byte, miner *types.Miner, accountdb vm.AccountDB) (uint64, bool) {
	dbAddr := mm.getMinerStakeDetailDatabase()
	frozenKey := mm.getDetailDBKey(from, miner.ID, miner.Type, types.StakeFrozen)
	frozenData := accountdb.GetData(dbAddr, string(frozenKey))
	if frozenData == nil || len(frozenData) == 0 {
		Logger.Debug("MinerManager.RefundStake return false(cannot find frozen data)")
		return 0, false
	}
	preFrozen := common.ByteToUint64(frozenData[:8])
	accountdb.RemoveData(dbAddr, string(frozenKey))
	return preFrozen, true
}

// AddStake adds the stake information into database
func (mm *MinerManager) AddStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool {
	Logger.Debugf("Miner manager addStake, minerid: %d", miner.ID)
	db := mm.getMinerDatabase(miner.Type)
	miner.Stake += amount
	if miner.Stake < amount {
		Logger.Debug("MinerManager.AddStake return false (overflow)")
		return false
	}
	if miner.Stake > mm.MaxStake(height) {
		Logger.Debug("MinerManager.AddStake return false (miner.Stake > mm.MaxStake)")
		return false
	}
	if miner.Status == types.MinerStatusAbort &&
		miner.Stake >= mm.MinStake() {
		miner.Status = types.MinerStatusNormal
		mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
	}
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(id), data)
	return true
}

// ReduceStake reduce the stake value and update the database.
func (mm *MinerManager) ReduceStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool {
	Logger.Debugf("Miner manager reduceStake, minerid: %d", miner.ID)
	db := mm.getMinerDatabase(miner.Type)
	if miner.Stake < amount {
		return false
	}
	miner.Stake -= amount
	if miner.Status == types.MinerStatusNormal && miner.Stake < mm.MinStake() {
		if miner.Type == types.MinerTypeLight && GroupChainImpl.WhetherMemberInActiveGroup(id, height) {
			Logger.Debugf("TVMExecutor Execute MinerRefund Light Fail(Still In Active Group) %s", common.ToHex(id))
			return false
		}
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height
		mm.updateMinerCount(miner.Type, minerCountDecrease, accountdb)
	}
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(id), data)
	return true
}

// Transaction2MinerParams parses a transaction's data field and try to found out the information of miner stake or
// miner cancel stake or miner refund
func (mm *MinerManager) Transaction2MinerParams(tx *types.Transaction) (_type byte, id []byte, value uint64) {
	data := common.FromHex(string(tx.Data))
	if len(data) == 0 {
		return
	}
	_type = data[0]
	if len(data) < common.AddressLength+1 {
		return
	}
	id = data[1 : common.AddressLength+1]
	if len(data) > common.AddressLength+1 {
		value = common.ByteToUint64(data[common.AddressLength+1:])
	}
	return
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

func (mi *MinerIterator) Current() (*types.Miner, error) {
	if mi.cache != nil {
		if result, ok := mi.cache.Get(string(mi.iterator.Key)); ok {
			return result.(*types.Miner), nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iterator.Value, &miner)
	if err != nil {
		Logger.Warnf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iterator.Key, err, mi.iterator.Value)
	}

	if len(miner.ID) == 0 {
		err = errors.New("empty miner")
	}
	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iterator.Next()
}

// Transaction2Miner parses a transcation and try to found out the information of miner apply
func (mm *MinerManager) Transaction2Miner(tx *types.Transaction) *types.Miner {
	data := common.FromHex(string(tx.Data))
	var miner types.Miner
	msgpack.Unmarshal(data, &miner)
	return &miner
}

func (mm *MinerManager) getStakeDetailByType(from, to common.Address, typ byte, db vm.AccountDB) []*types.StakeDetail {
	dbAddr := mm.getMinerStakeDetailDatabase()
	details := make([]*types.StakeDetail, 0)
	key := string(mm.getDetailDBKey(from.Bytes(), to.Bytes(), typ, types.Staked))
	stakedData := db.GetData(dbAddr, key)

	if len(stakedData) > 0 {
		detail := &types.StakeDetail{
			Source: from,
			Target: to,
			Value:  common.ByteToUint64(stakedData),
			Status: types.Staked,
		}
		details = append(details, detail)
	}
	key = string(mm.getDetailDBKey(from.Bytes(), to.Bytes(), typ, types.StakeFrozen))
	stakedData = db.GetData(dbAddr, key)

	if len(stakedData) > 0 {
		detail := &types.StakeDetail{
			Source:       from,
			Target:       to,
			Value:        common.ByteToUint64(stakedData[:8]),
			Status:       types.StakeFrozen,
			FrozenHeight: common.ByteToUint64(stakedData[8:]),
		}
		details = append(details, detail)
	}
	return details
}

// GetStakeDetail returns the stake details of the given address pair
func (mm *MinerManager) GetStakeDetail(from, to common.Address, db vm.AccountDB) []*types.StakeDetail {
	details := mm.getStakeDetailByType(from, to, types.MinerTypeHeavy, db)
	details = append(details, mm.getStakeDetailByType(from, to, types.MinerTypeLight, db)...)
	return details
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
