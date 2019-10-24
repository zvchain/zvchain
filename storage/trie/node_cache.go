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

package trie

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/rlp"
	"github.com/zvchain/zvchain/storage/tasdb"
	"sync/atomic"
	"time"
)

type nodeCache struct {
	cache *lru.Cache
	hit   uint64
	total uint64
	store tasdb.Database
}

type storeBlob struct {
	Key, Value []byte
}

var ncache *nodeCache
var storeKey = []byte("node_iterator_cache")

func CreateNodeCache(size int, db tasdb.Database) {
	ncache = &nodeCache{
		cache: common.MustNewLRUCache(size),
		store: db,
	}
	ncache.loadFromDB()

	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			ncache.persistCache()
		}
	}()
}

func (nc *nodeCache) getNode(hash common.Hash) node {
	if nc == nil {
		return nil
	}
	defer func() {
		total := atomic.LoadUint64(&nc.total)
		if total%50 == 0 && total > 0 {
			log.CoreLogger.Debugf("node iterator getNode hit %.4f(%v/%v),cache size %v", float64(nc.hit)/float64(total), nc.hit, total, ncache.cache.Len())
		}
	}()
	atomic.AddUint64(&ncache.total, 1)
	if atomic.LoadUint64(&nc.total) == 0 {
		atomic.StoreUint64(&nc.total, 1)
		atomic.StoreUint64(&nc.hit, 0)
	}
	if v, ok := nc.cache.Get(hash); ok {
		atomic.AddUint64(&ncache.hit, 1)
		return v.(node)
	}
	return nil
}

func (nc *nodeCache) storeNode(hash common.Hash, n node) {
	if nc == nil {
		return
	}
	nc.cache.Add(hash, n)
}

func (nc *nodeCache) loadFromDB() {
	bytes, err := nc.store.Get(storeKey)
	if err != nil {
		log.CoreLogger.Errorf("get node cache blobs error:%v", err)
		return
	}
	var blobs []*storeBlob
	err = rlp.DecodeBytes(bytes, &blobs)
	if err != nil {
		log.CoreLogger.Errorf("decode node cache blobs error:%v", err)
		return
	}
	log.CoreLogger.Errorf("load node cache blobs size:%v", len(blobs))
	for _, b := range blobs {
		hash := common.BytesToHash(b.Key)
		node := mustDecodeNode(hash.Bytes(), b.Value, 0)
		nc.storeNode(hash, node)
	}
}

func (nc *nodeCache) persistCache() {
	blobs := make([]*storeBlob, 0, nc.cache.Len())
	vkeys := nc.cache.Keys()
	if len(vkeys) > 6000 {
		vkeys = vkeys[len(vkeys)-6000:]
	}
	for _, k := range vkeys {
		hash := k.(common.Hash)
		v, ok := nc.cache.Peek(hash)
		if !ok {
			continue
		}
		cnode := &cachedNode{node: simplifyNode(v.(node))}
		blob := &storeBlob{
			Key:   hash.Bytes(),
			Value: cnode.rlp(),
		}
		blobs = append(blobs, blob)
	}
	bytes, err := rlp.EncodeToBytes(blobs)
	if err != nil {
		log.CoreLogger.Errorf("encode node cache blobs error:%v", err)
		return
	}
	err = nc.store.Put(storeKey, bytes)
	if err != nil {
		log.CoreLogger.Errorf("put node cache blobs error:%v", err)
	}
	log.CoreLogger.Debugf("persist node cache len %v/%v, size %v", len(vkeys), nc.cache.Len(), len(bytes))
}
