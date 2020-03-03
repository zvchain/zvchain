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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/trie"
	"io"
	"log"
	"os"
	"sync/atomic"
	"testing"
)

func traverseGroup(i int, db types.AccountDB, seed common.Hash, cachedb *trie.NodeDatabase, onlyTraverKey []byte, out io.Writer) error {
	obj := db.GetStateObject(common.HashToAddress(seed))
	if obj == nil {
		return fmt.Errorf("seed address object not exist")
	}
	root := obj.GetRootHash()
	trie, err := trie.NewTrie(root, cachedb)
	if err != nil {
		return err
	}

	var (
		ok        bool
		keyCount  int32
		valueSize int32
	)

	if len(onlyTraverKey) == 0 {
		ok, err = trie.Traverse(func(key []byte, value []byte) error {
			//out.Write([]byte(fmt.Sprintf("all traverse seed %v, key %v, valuesize %v", seed, string(key), len(value))))
			atomic.AddInt32(&keyCount, 1)
			atomic.AddInt32(&valueSize, int32(len(value)))
			return nil
		}, nil, false)
	} else {
		ok, err = trie.TraverseKey(onlyTraverKey, func(key []byte, value []byte) error {
			//out.Write([]byte(fmt.Sprintf("only key traverse seed %v, key %v, valuesize %v", seed, string(key), len(value))))
			atomic.AddInt32(&keyCount, 1)
			atomic.AddInt32(&valueSize, int32(len(value)))
			return nil
		}, nil, false)
	}
	if !ok {
		return err
	}

	fmt.Printf("%v, traverse group %v success, key %v, vsize %v\n", i, seed, keyCount, valueSize)

	return nil
}

func TestTraverseGroupAfterPrune(t *testing.T) {
	fi,_ := os.Stat("/Users/admin/Desktop")
	if fi == nil{
		return
	}
	common.InitConf("/Users/admin/Desktop/gzv-prune/zv.ini")
	group := newGroup4CPTest(0, common.MaxUint64)
	group.h.seed = common.HexToHash("0x6861736820666f72207a76636861696e27732067656e657369732067726f7570")
	tailor, err := NewOfflineTailor(&types.GenesisInfo{Group: group}, "/Users/admin/Desktop/gzv-prune/d_b20191220_241w_pruned_opti", "", 1024, "", "", true, 1024)
	if err != nil {
		t.Error(err)
	}
	height := tailor.chain.Height()
	fmt.Println("height ", height)
	db, err := tailor.chain.AccountDBAt(height)
	if err != nil {
		t.Fatalf("get db error at %v, err %v", height, err)
	}

	groupSeeds, err := tailor.groupReader.GetAllGroupSeedsByHeight(height)
	if err != nil {
		t.Fatalf("load group seed error %v", err)
	}
	log.Printf("group size %v", len(groupSeeds))

	var traverseKey []byte

	for i := 0; i < len(groupSeeds); {
		seed := groupSeeds[i]
		if err = traverseGroup(i, db, seed, tailor.chain.stateCache.TrieDB(), traverseKey, tailor.out); err != nil {
			if isMissingNodeError(err) && traverseKey == nil {
				traverseKey = tailor.groupReader.GroupKey()
				fmt.Printf("traverse group %v missing node, start traverse key %v\n", seed, traverseKey)
				continue
			} else {
				t.Fatal(err)
			}
		}
		if traverseKey != nil {
			d := db.GetData(common.HashToAddress(seed), traverseKey)
			if d == nil {
				t.Fatalf("get group data nil %v", seed)
			}
		}
		i++
	}
	t.Log("success")
}
