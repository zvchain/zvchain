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
	"encoding/gob"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"math/rand"
	"os"
	"testing"
)

func genShortNode() node {
	val := node(valueNode(common.Uint64ToByte(rand.Uint64())))
	return &shortNode{
		Key: common.Int64ToByte(rand.Int63()),
		Val: val,
	}
}

func genFullNode(level int) node {
	if level == 0 {
		return genShortNode()
	}
	var chds [17]node
	for i := 0; i < 17; i++ {
		if i%10 == 0 {
			chds[i] = genFullNode(level - 1)
		}
	}
	return &fullNode{
		Children: chds,
	}
}

func TestStoreAndLoad(t *testing.T) {
	defer os.RemoveAll("cache_test")

	gob.Register(&shortNode{})
	gob.Register(&fullNode{})
	gob.Register(&valueNode{})
	gob.Register(&hashNode{})

	db, err := tasdb.NewLDBDatabase("cache_test", nil)
	if err != nil {
		t.Error(err)
		return
	}
	ncache = &nodeCache{
		cache: common.MustNewLRUCache(100),
		store: db,
	}
	for i := 0; i < 10; i++ {
		hash := common.BytesToHash(common.Uint64ToByte(rand.Uint64()))

		ncache.storeNode(hash, genFullNode(3))
	}
	ncache.persistCache()

	ncache.loadFromDB()
}
