//   Copyright (C) 2019 ZVChain
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
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"sync/atomic"
)

type stakeCacheItem struct {
	byRoot *lru.Cache
}

func (item *stakeCacheItem) getStake(root common.Hash) uint64 {
	if v, ok := item.byRoot.Peek(root); ok {
		return v.(uint64)
	}
	return 0
}

func (item *stakeCacheItem) setStake(root common.Hash, st uint64) {
	item.byRoot.Add(root, st)
}

type rootCacheItem struct {
	byHash *lru.Cache
}

func (item *rootCacheItem) getRoot(block common.Hash) common.Hash {
	if v, ok := item.byHash.Peek(block); ok {
		return v.(common.Hash)
	}
	return common.Hash{}
}

func (item *rootCacheItem) setRoot(block common.Hash, root common.Hash) {
	item.byHash.Add(block, root)
}

func newStakeCacheItem() *stakeCacheItem {
	return &stakeCacheItem{
		byRoot: common.MustNewLRUCache(10),
	}
}

func newRootCacheItem() *rootCacheItem {
	return &rootCacheItem{
		byHash: common.MustNewLRUCache(20),
	}
}

type accountDBGetter interface {
	// Use the lockless one of the FullBlockChain in case of deadlock when adds block on chain
	accountDBAt(height uint64) (types.AccountDB, error)
}

type minerGetter interface {
	getMiner(db types.AccountDB, address common.Address, minerType types.MinerType) *types.Miner
}

type stakeQuerier struct {
	mGet       minerGetter
	dbGet      accountDBGetter
	stakeCache *lru.Cache
	rootCache  *lru.Cache
	stakeHit   uint64
	rootHit    uint64
	total      uint64
}

var querier *stakeQuerier

func initStakeGetter(mGet minerGetter, dbGet accountDBGetter) {
	querier = &stakeQuerier{
		mGet:       mGet,
		dbGet:      dbGet,
		stakeCache: common.MustNewLRUCache(500),
		rootCache:  common.MustNewLRUCache(500),
	}
	types.DefaultStakeGetter = querier.queryProposerStake
}

//func (sq *stakeQuerier) getRoot(addr common.Address, height uint64) common.Hash {
//	v, ok := sq.rootCache.Get(addr)
//	var item *rootCacheItem
//	if ok {
//		item = v.(*rootCacheItem)
//		if root := item.getRoot(hash); root != common.EmptyHash {
//			atomic.AddUint64(&sq.rootHit, 1)
//			return root
//		}
//	} else {
//		item = newRootCacheItem()
//		sq.rootCache.ContainsOrAdd(addr, item)
//	}
//	db, err := sq.dbGet.accountDBAt(hash)
//	if err != nil {
//		Logger.Errorf("get accountDB error:%v, hash:%v", err, hash)
//		return common.Hash{}
//	}
//	obj := db.GetStateObject(addr)
//	if obj == nil {
//		Logger.Errorf("get account object nil of %v", addr.AddrPrefixString())
//		return common.Hash{}
//	}
//	item.setRoot(hash, obj.GetRootHash())
//	return obj.GetRootHash()
//}

func (sq *stakeQuerier) getRoot(addr common.Address, height uint64) (types.AccountDB, common.Hash) {
	db, err := sq.dbGet.accountDBAt(height)
	if err != nil {
		Logger.Errorf("get accountDB error:%v, height:%v", err, height)
		return nil, common.Hash{}
	}
	obj := db.GetStateObject(addr)
	if obj == nil {
		Logger.Errorf("get account object nil of %v", addr.AddrPrefixString())
		return db, common.Hash{}
	}
	return db, obj.GetRootHash()
}

func (sq *stakeQuerier) getStake(db types.AccountDB, addr common.Address, height uint64, root common.Hash) uint64 {
	v, ok := sq.stakeCache.Get(addr)
	var item *stakeCacheItem
	if ok {
		item = v.(*stakeCacheItem)
		if stake := item.getStake(root); stake > 0 {
			atomic.AddUint64(&sq.stakeHit, 1)
			return stake
		}
	} else {
		item = newStakeCacheItem()
		sq.stakeCache.ContainsOrAdd(addr, item)
	}
	m := sq.mGet.getMiner(db, addr, types.MinerTypeProposal)
	if m == nil {
		return 0
	}
	item.setStake(root, m.Stake)
	return m.Stake
}

func (sq *stakeQuerier) queryProposerStake(addr common.Address, height uint64) uint64 {
	t := atomic.AddUint64(&sq.total, 1)
	if t == 0 {
		atomic.StoreUint64(&sq.stakeHit, 0)
		atomic.StoreUint64(&sq.rootHit, 0)
	}
	if t != 0 && t%10 == 0 {
		stakeHit := atomic.LoadUint64(&sq.stakeHit)
		rootHit := atomic.LoadUint64(&sq.rootHit)
		Logger.Debugf("queryProposerStake stake hit rate: %f(%v/%v), root hit rate: %f(%v/%v)", float64(stakeHit)/float64(t), stakeHit, t, float64(rootHit)/float64(t), rootHit, t)
	}

	db, root := sq.getRoot(addr, height)
	if db == nil || root == common.EmptyHash {
		return 0
	}
	return sq.getStake(db, addr, height, root)
}
