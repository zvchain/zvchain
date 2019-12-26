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
	"log"
	"testing"
	"time"
)

func traverseGroup(db types.AccountDB, seed common.Hash, cachedb *trie.NodeDatabase, onlyTraverKey []byte) error {
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
		ok bool
	)

	if len(onlyTraverKey) == 0 {
		ok, err = trie.Traverse(func(key []byte, value []byte) error {
			log.Printf("all traverse seed %v, key %v, valuesize %v", seed, string(key), len(value))
			return nil
		}, nil, false)
	} else {
		ok, err = trie.TraverseKey(onlyTraverKey, func(key []byte, value []byte) error {
			log.Printf("only key traverse seed %v, key %v, valuesize %v", seed, string(key), len(value))
			return nil
		}, nil, false)
	}
	if !ok {
		return err
	}

	return nil
}

func TestTraverseGroupAfterPrune(t *testing.T) {
	group := newGroup4CPTest(0, common.MaxUint64)
	tailor, err := NewOfflineTailor(&types.GenesisInfo{Group: group}, "", "", 1024, "", "", true, 1024)
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
		b := time.Now()
		seed := groupSeeds[i]
		if err = traverseGroup(db, seed, tailor.chain.stateCache.TrieDB(), traverseKey); err != nil {
			if isMissingNodeError(err) && traverseKey == nil {
				traverseKey = tailor.groupReader.GroupKey()
				log.Printf("traverse group %v missing node, start traverse key %v", seed, traverseKey)
				continue
			} else {
				t.Fatal(err)
			}
		}
		i++
		log.Printf("traverse group %v success, cost %v", seed, time.Since(b).String())
	}
	t.Log("success")
}
