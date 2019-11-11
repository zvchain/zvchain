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
	"bytes"
	lru "github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
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

type cacheItem struct {
	n   node
	raw []byte
}

type storeBlob struct {
	Key common.Hash
	Raw []byte
}

var ncache *nodeCache
var storeKey = []byte("node_iterator_cache")

func CreateNodeCache(size int, db tasdb.Database) {
	ncache = &nodeCache{
		cache: common.MustNewLRUCache(size),
		store: db,
	}

	ncache.loadFromDB()

	ticker := time.NewTicker(300 * time.Second)
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
	atomic.AddUint64(&nc.total, 1)
	if atomic.LoadUint64(&nc.total) == 0 {
		atomic.StoreUint64(&nc.total, 1)
		atomic.StoreUint64(&nc.hit, 0)
	}
	if v, ok := nc.cache.Get(hash); ok {
		atomic.AddUint64(&nc.hit, 1)
		return v.(*cacheItem).n
	}
	return nil
}

func (nc *nodeCache) storeNode(hash common.Hash, n node, raw []byte) {
	if nc == nil {
		return
	}
	nc.cache.Add(hash, &cacheItem{n: n, raw: raw})
}

func (nc *nodeCache) loadFromDB() {
	bs, err := nc.store.Get(storeKey)
	if err != nil {
		log.CoreLogger.Errorf("get node cache blobs error:%v", err)
		return
	}
	var blobs []*storeBlob
	r := bytes.NewBuffer(bs)
	decoder := msgpack.NewDecoder(r)
	err = decoder.Decode(&blobs)
	if err != nil {
		log.CoreLogger.Errorf("decode node cache blobs error:%v", err)
		return
	}
	log.CoreLogger.Infof("load node cache blobs size:%v", len(blobs))
	for _, b := range blobs {
		n := mustDecodeNode(b.Key.Bytes(), b.Raw, 0)
		nc.storeNode(b.Key, n, b.Raw)
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
		raw := v.(*cacheItem).raw
		if raw == nil {
			continue
		}
		blob := &storeBlob{
			Key: hash,
			Raw: raw,
		}
		blobs = append(blobs, blob)
	}
	w := bytes.NewBuffer([]byte{})
	encode := msgpack.NewEncoder(w)
	err := encode.Encode(blobs)
	if err != nil {
		log.CoreLogger.Errorf("encode node cache blobs error:%v", err)
		return
	}
	err = nc.store.Put(storeKey, w.Bytes())
	if err != nil {
		log.CoreLogger.Errorf("put node cache blobs error:%v", err)
	}
	log.CoreLogger.Debugf("persist node cache len %v/%v, size %v", len(vkeys), nc.cache.Len(), w.Len())
}
