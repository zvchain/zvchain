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
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

type mcacheStore struct {
	db tasdb.Database
}

type cacheItem struct {
	Addr  common.Address
	Miner *minerCache
}

var (
	storeKeyProposer = []byte("miner_proposer")
	storeKeyVerifier = []byte("miner_verifier")
)

func initCacheStore(db tasdb.Database) *mcacheStore {
	return &mcacheStore{
		db: db,
	}
}

func getStoreKey(mType types.MinerType) []byte {
	var key []byte
	if types.IsVerifyRole(mType) {
		key = storeKeyVerifier
	} else {
		key = storeKeyProposer
	}
	return key
}

func (ms *mcacheStore) doStore(mType types.MinerType, items []*cacheItem) {
	key := getStoreKey(mType)
	bs, err := msgpack.Marshal(items)
	if err != nil {
		return
	}
	err = ms.db.Put(key, bs)
	if err != nil {
		return
	}
	Logger.Debugf("store miner %v size %v", mType, len(items))
}

func (ms *mcacheStore) storeMiners(mType types.MinerType, cache *lru.Cache) {
	items := make([]*cacheItem, 0, cache.Len())
	for _, k := range cache.Keys() {
		addr := k.(common.Address)
		v, ok := cache.Peek(k)
		if !ok {
			continue
		}
		item := &cacheItem{
			Addr:  addr,
			Miner: v.(*minerCache),
		}
		items = append(items, item)
	}
	ms.doStore(mType, items)
}

func (ms *mcacheStore) doLoad(mType types.MinerType) []*cacheItem {
	key := getStoreKey(mType)
	bs, err := ms.db.Get(key)
	if err != nil {
		return nil
	}
	var items []*cacheItem
	err = msgpack.Unmarshal(bs, &items)
	if err != nil {
		return nil
	}
	Logger.Debugf("load miners size %v", len(items))
	return items
}

func (ms *mcacheStore) loadMiners(mType types.MinerType, cache *lru.Cache) {
	items := ms.doLoad(mType)
	if len(items) == 0 {
		return
	}
	for _, item := range items {
		cache.Add(item.Addr, item.Miner)
	}
}
