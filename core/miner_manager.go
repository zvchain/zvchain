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
	lru "github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	checkInterval = 1000
)

var MinerManagerImpl *MinerManager

// MinerManager manage all the miner related actions
type MinerManager struct {
	verifyAddrsCache   *minerAddressCache
	proposalAddrsCache *minerAddressCache
	verifyMinerData    *lru.Cache
	proposalMinerData  *lru.Cache
	store              *mcacheStore
	lock               sync.RWMutex
}

type minerAddressCache struct {
	rootHash common.Hash
	addrs    []common.Address
}

type minerCache struct {
	RootHash common.Hash
	Miner    *types.Miner
}

func initMinerManager(storeDB tasdb.Database) {
	MinerManagerImpl = &MinerManager{
		verifyMinerData:   common.MustNewLRUCache(10000),
		proposalMinerData: common.MustNewLRUCache(10000),
		store:             initCacheStore(storeDB),
	}

	if storeDB != nil {
		MinerManagerImpl.store.loadMiners(types.MinerTypeProposal, MinerManagerImpl.proposalMinerData)
		MinerManagerImpl.store.loadMiners(types.MinerTypeVerify, MinerManagerImpl.verifyMinerData)

		ticker := time.NewTicker(300 * time.Second)
		go func() {
			for range ticker.C {
				MinerManagerImpl.store.storeMiners(types.MinerTypeProposal, MinerManagerImpl.proposalMinerData)
				MinerManagerImpl.store.storeMiners(types.MinerTypeVerify, MinerManagerImpl.verifyMinerData)
			}
		}()
	}
}

// GuardNodesCheck check guard nodes is expired
func (mm *MinerManager) GuardNodesCheck(accountDB types.AccountDB, bh *types.BlockHeader) {
	snapshot := accountDB.Snapshot()
	expiredAddresses, err := mm.FullStakeGuardNodesCheck(accountDB, bh.Height)
	if err != nil {
		accountDB.RevertToSnapshot(snapshot)
		log.CoreLogger.Errorf("check full guard node error,error is %s", err.Error())
	} else {
		if expiredAddresses != nil && len(expiredAddresses) > 0 {
			for _, expiredAddr := range expiredAddresses {
				err = mm.processGuardNodeExpired(accountDB, expiredAddr, bh.Height)
				if err != nil {
					accountDB.RevertToSnapshot(snapshot)
					log.CoreLogger.Errorf("check full guard node error,error is %s", err.Error())
					break
				}
			}
		}
	}
	snapshot = accountDB.Snapshot()
	allExpiredFundAddresses, err := mm.FundGuardExpiredCheck(accountDB, bh.Height)
	if err != nil {
		accountDB.RevertToSnapshot(snapshot)
		log.CoreLogger.Errorf("check fund guard node error,error is %s", err.Error())
	} else {
		for _, expiredAddr := range allExpiredFundAddresses {
			err = guardNodeExpired(accountDB, expiredAddr, bh.Height, true)
			if err != nil {
				accountDB.RevertToSnapshot(snapshot)
				log.CoreLogger.Errorf("check fund guard node error,error is %s", err.Error())
				break
			}
		}
	}
}

func (mm *MinerManager) FundGuardExpiredCheck(accountDB types.AccountDB, height uint64) ([]common.Address, error) {
	allExpiredFundAddresses := []common.Address{}
	expiredAddresses, err := mm.fundGuardSixAddFiveNodesCheck(accountDB, height)
	if err != nil {
		return nil, err
	}
	if expiredAddresses != nil {
		allExpiredFundAddresses = append(allExpiredFundAddresses, expiredAddresses...)
	}
	expiredAddresses, err = mm.fundGuardSixAddSixNodesCheck(accountDB, height)
	if err != nil {
		return nil, err
	}
	if expiredAddresses != nil {
		allExpiredFundAddresses = append(allExpiredFundAddresses, expiredAddresses...)
	}
	return allExpiredFundAddresses, nil
}

func (mm *MinerManager) fundGuardSixAddFiveNodesCheck(accountDB types.AccountDB, height uint64) ([]common.Address, error) {
	if height < adjustWeightPeriod/2 || height > adjustWeightPeriod*2 {
		return nil, nil
	}
	if height%checkInterval != 0 {
		return nil, nil
	}
	log.CoreLogger.Infof("begin scan 6+5 mode")
	hasScanned := hasScanedSixAddFiveFundGuards(accountDB)
	if hasScanned {
		log.CoreLogger.Infof("begin scan 6+5 mode,find has scanned")
		return nil, nil
	}
	fds, err := mm.GetAllFundStakeGuardNodes(accountDB)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		expiredAddresses = append(expiredAddresses, fd.Address)
	}
	markScanedSixAddFiveFundGuards(accountDB)
	log.CoreLogger.Infof("scan 6+5 over")
	return expiredAddresses, nil
}

func (mm *MinerManager) fundGuardSixAddSixNodesCheck(accountDB types.AccountDB, height uint64) ([]common.Address, error) {
	if height < adjustWeightPeriod || height > adjustWeightPeriod*3 {
		return nil, nil
	}
	if height%checkInterval != 0 {
		return nil, nil
	}
	log.CoreLogger.Infof("begin scan 6+6 mode")
	hasScanned := hasScanedSixAddSixFundGuards(accountDB)
	if hasScanned {
		log.CoreLogger.Infof("begin scan 6+6 mode,has scanned")
		return nil, nil
	}
	fds, err := mm.GetAllFundStakeGuardNodes(accountDB)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		expiredAddresses = append(expiredAddresses, fd.Address)
	}
	markScanedSixAddSixFundGuards(accountDB)
	log.CoreLogger.Infof("scan 6+6 mode over")
	return expiredAddresses, nil
}

func (mm *MinerManager) FullStakeGuardNodesCheck(db types.AccountDB, height uint64) ([]common.Address, error) {
	if height < adjustWeightPeriod/2 {
		return nil, nil
	}
	if height%checkInterval != 0 {
		return nil, nil
	}
	log.CoreLogger.Infof("begin process full stake guard nodes")
	fullStakeAddress := mm.GetAllFullStakeGuardNodes(db)
	if fullStakeAddress == nil || len(fullStakeAddress) == 0 {
		return nil, nil
	}
	expiredAddresses := []common.Address{}
	var err error
	var isExpired bool
	for _, addr := range fullStakeAddress {
		isExpired, err = mm.checkFullStakeGuardNodeExpired(db, addr, height)
		if err != nil {
			return nil, err
		}
		if isExpired {
			expiredAddresses = append(expiredAddresses, addr)
		}
	}
	return expiredAddresses, nil
}

func (mm *MinerManager) GetTickets(db types.AccountDB, address common.Address) uint64 {
	return getTickets(db, address)
}

func (mm *MinerManager) GetValidTicketsByHeight(height uint64) uint64 {
	return getValidTicketsByHeight(height)
}

func (mm *MinerManager) checkFullStakeGuardNodeExpired(db types.AccountDB, address common.Address, height uint64) (bool, error) {
	detailKey := getDetailKey(address, types.MinerTypeProposal, types.Staked)
	stakedDetail, err := getDetail(db, address, detailKey)
	if err != nil {
		return false, err
	}
	if stakedDetail == nil {
		return false, fmt.Errorf("check guard nodes,find stake detail is nil,address is %s", address.String())
	}
	if height > (stakedDetail.DisMissHeight + stakeBuffer) {
		return true, nil
	}
	if stakedDetail.MarkNotFullHeight > 0 {
		if height > stakedDetail.MarkNotFullHeight+stakeBuffer {
			return true, nil
		}
	} else {
		if !isFullStake(stakedDetail.Value, height) {
			stakedDetail.MarkNotFullHeight = height
			err = setDetail(db, address, detailKey, stakedDetail)
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}
	return false, nil
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

func (mm *MinerManager) getMiner(db types.AccountDB, address common.Address, mType types.MinerType) *types.Miner {
	m, err := getMiner(db, address, mType)
	if err != nil {
		Logger.Errorf("get miner error:%v", err)
		return nil
	}
	return m
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
			if !bytes.HasPrefix(iter.Key, common.KeyGuardNodes) {
				break
			}
			addr := common.BytesToAddress(iter.Key[len(common.KeyGuardNodes):])
			addrs = append(addrs, addr)
		}
	}
	return addrs
}
func (mm *MinerManager) GetFundGuard(addr string) (*fundGuardNode, error) {
	db, err := BlockChainImpl.LatestAccountDB()
	if err != nil {
		return nil, err
	}

	return getFundGuardNode(db, common.StringToAddress(addr))
}

func (mm *MinerManager) GetAllFundStakeGuardNodes(accountDB types.AccountDB) ([]*fundGuardNodeDetail, error) {
	var fds []*fundGuardNodeDetail
	iter := accountDB.DataIterator(common.FundGuardNodeAddr, common.KeyGuardNodes)
	for iter.Next() {
		if !bytes.HasPrefix(iter.Key, common.KeyGuardNodes) {
			break
		}
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

func (mm *MinerManager) iteratorAddrs(accountDB types.AccountDB, mType types.MinerType, height uint64) []common.Address {
	var prefix []byte
	if types.IsVerifyRole(mType) {
		prefix = common.PrefixPoolVerifier
	} else {
		prefix = common.PrefixPoolProposal
	}
	addrs := make([]common.Address, 0)
	iter := accountDB.DataIterator(common.MinerPoolAddr, prefix)
	iter.EnableNodeCache()
	for iter.Next() {
		// finish the iterator
		if !bytes.HasPrefix(iter.Key, prefix) {
			break
		}
		addr := common.BytesToAddress(iter.Key[len(prefix):])
		addrs = append(addrs, addr)
	}
	return addrs
}

func (mm *MinerManager) getAndSetAllMinerAddrs(accountDB types.AccountDB, mType types.MinerType, height uint64) []common.Address {
	begin := time.Now()
	defer func() {
		Logger.Debugf("get all miners addrs cost %v", time.Since(begin).Seconds())
	}()

	accesser := accountDB.GetStateObject(common.MinerPoolAddr)
	if accesser == nil {
		Logger.Debugf("get state object nil,height = %v", height)
		return nil
	}
	var mac *minerAddressCache
	if types.IsVerifyRole(mType) {
		mac = mm.verifyAddrsCache
	} else {
		mac = mm.proposalAddrsCache
	}
	if mac == nil {
		mac = &minerAddressCache{}
	}
	if accesser.GetRootHash() != mac.rootHash {
		addrs := mm.iteratorAddrs(accountDB, mType, height)
		mac.rootHash = accesser.GetRootHash()
		mac.addrs = addrs
		if types.IsVerifyRole(mType) {
			mm.verifyAddrsCache = mac
		} else {
			mm.proposalAddrsCache = mac
		}
	} else {
		Logger.Debugf("hit address cache,type is %v,height is %v", mType, height)
	}
	return mac.addrs
}

// GetAllMiners returns all miners of the the specified type at the given height
func (mm *MinerManager) GetAllMiners(mType types.MinerType, height uint64) []*types.Miner {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	begin := time.Now()
	defer func() {
		Logger.Debugf("get all miners cost %v", time.Since(begin).Seconds())
	}()
	accountDB, err := BlockChainImpl.AccountDBAt(height)
	if err != nil {
		Logger.Errorf("Get account db by height %v error:%s", height, err.Error())
		return nil
	}
	addrs := mm.getAndSetAllMinerAddrs(accountDB, mType, height)
	if len(addrs) == 0 {
		return nil
	}
	miners := make([]*types.Miner, len(addrs))
	Logger.Debugf("get all miners len %v, type %v, height %v", len(addrs), mType, height)
	atErr := atomic.Value{}
	var missCount uint64 = 0
	// Get miners concurrently, and the order of the result must be kept as it is in the addrs slice
	getMinerFunc := func(begin, end int) {
		for i := begin; i < end; i++ {
			var (
				miner       *types.Miner
				err         error
				needToCache bool
				mc          *minerCache
				cache       *lru.Cache
			)
			stateObject := accountDB.GetStateObject(addrs[i])
			if stateObject == nil {
				err = fmt.Errorf("get miner error,because stateObject is nil, addr %v", addrs[i].AddrPrefixString())
				atErr.Store(err)
				return
			}
			if types.IsVerifyRole(mType) {
				cache = mm.verifyMinerData
			} else {
				cache = mm.proposalMinerData
			}
			obj, ok := cache.Get(stateObject.GetAddr())
			if ok {
				mc = obj.(*minerCache)
				if mc.RootHash == stateObject.GetRootHash() && mc.Miner != nil {
					miner = mc.Miner
				} else {
					miner, err = getMinerFromStateObject(accountDB.Database(), stateObject, mType)
					needToCache = true
				}
			} else {
				miner, err = getMinerFromStateObject(accountDB.Database(), stateObject, mType)
				needToCache = true
			}
			if err != nil {
				Logger.Errorf("get miner error, addr %v, err %v", addrs[i].AddrPrefixString(), err)
				atErr.Store(err)
				return
			}
			if miner == nil {
				err = fmt.Errorf("get miner nil:%v", addrs[i].AddrPrefixString())
				Logger.Error(err)
				atErr.Store(err)
				return
			} else {
				miners[i] = miner
			}
			if atErr.Load() != nil {
				break
			}
			if needToCache {
				atomic.AddUint64(&missCount, 1)
				mc = &minerCache{RootHash: stateObject.GetRootHash(), Miner: miner}
				cache.Add(stateObject.GetAddr(), mc)
			}
		}
		return
	}

	parallel := runtime.NumCPU() * 2
	step := int(math.Ceil(float64(len(addrs)) / float64(parallel)))
	wg := sync.WaitGroup{}

	for begin := 0; begin < len(addrs); {
		end := begin + step
		if end > len(addrs) {
			end = len(addrs)
		}
		wg.Add(1)
		b, e := begin, end
		go func() {
			defer wg.Done()
			getMinerFunc(b, e)
			Logger.Debugf("get all miner partly finished, range %v-%v, err:%v, height:%v", b, e, err, height)
		}()
		begin = end
	}
	wg.Wait()

	if e := atErr.Load(); e != nil {
		Logger.Errorf("get all miner error:%v", e)
		return nil
	}
	Logger.Debugf("get all miner partly finished, height:%v,cost %v,misscount is %v", height, time.Since(begin).Seconds(), atomic.LoadUint64(&missCount))
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
	for _, addr := range types.GetGuardAddress() {
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
