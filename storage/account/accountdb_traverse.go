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

package account

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/storage/rlp"
	"github.com/zvchain/zvchain/storage/trie"
	"sync"
	"sync/atomic"
	"time"
)

type VisitAccountCallback func(stat *TraverseStat)
type SubTreeKeyProvider func(address common.Address) [][]byte

type TraverseStat struct {
	Addr      common.Address
	Account   Account
	DataCount uint64
	DataSize  uint64
	NodeSize  uint64
	NodeCount uint64
	KeySize   uint64
	CodeSize  uint64
	Cost      time.Duration
}

func (vs *TraverseStat) String() string {
	s, _ := json.Marshal(vs)
	return string(s)
}

type TraverseConfig struct {
	VisitAccountCb      VisitAccountCallback
	ResolveNodeCb       trie.ResolveNodeCallback
	CheckHash           bool
	SubTreeKeysProvider SubTreeKeyProvider       // Provides concerned keys for the specified address, and only traverse the given keys for the address
	VisitedRoots        map[common.Hash]struct{} // Store visited roots， no more revisit for duplicate roots
	lock                sync.RWMutex
}

func (cfg *TraverseConfig) OnResolve(hash common.Hash, data []byte) {
	if cfg.ResolveNodeCb != nil {
		cfg.ResolveNodeCb(hash, data)
	}
}

func (cfg *TraverseConfig) OnVisitAccount(stat *TraverseStat) {
	if cfg.VisitAccountCb != nil {
		cfg.VisitAccountCb(stat)
	}
}

func (cfg *TraverseConfig) subTreeKeys(address common.Address) [][]byte {
	if cfg.SubTreeKeysProvider != nil {
		return cfg.SubTreeKeysProvider(address)
	}
	return nil
}

func (cfg *TraverseConfig) needTraverse(root common.Hash) bool {
	if cfg.VisitedRoots == nil {
		return true
	}
	cfg.lock.RLock()
	defer cfg.lock.RUnlock()
	_, ok := cfg.VisitedRoots[root]
	return !ok
}

func (cfg *TraverseConfig) addVisitedRoot(root common.Hash) {
	if cfg.VisitedRoots == nil {
		return
	}
	cfg.lock.Lock()
	defer cfg.lock.Unlock()
	cfg.VisitedRoots[root] = struct{}{}
}

func (adb *AccountDB) Traverse(config *TraverseConfig) (bool, error) {
	return adb.trie.Traverse(func(key []byte, value []byte) error {
		var account Account
		if err := rlp.DecodeBytes(value, &account); err != nil {
			return err
		}
		begin := time.Now()
		vs := &TraverseStat{Account: account, Addr: common.BytesToAddress(key)}

		leafCb := func(k []byte, v []byte) error {
			atomic.AddUint64(&vs.DataCount, 1)
			atomic.AddUint64(&vs.DataSize, uint64(len(v)))
			atomic.AddUint64(&vs.KeySize, uint64(len(k)))
			return nil
		}

		resolveCb := func(hash common.Hash, data []byte) {
			config.OnResolve(hash, data)
			atomic.AddUint64(&vs.NodeSize, uint64(len(data)))
			atomic.AddUint64(&vs.NodeCount, 1)
		}

		// Traverse the sub tree of the account if needed
		// Those traversed root won't traverse again
		if account.Root != emptyData && config.needTraverse(account.Root) {
			t, err := trie.NewTrie(account.Root, adb.db.TrieDB())
			if err != nil {
				return err
			}
			keys := config.subTreeKeys(vs.Addr)
			// Traverse the entire sub tree if no keys specified
			if len(keys) == 0 {
				if ok, err := t.Traverse(leafCb, resolveCb, config.CheckHash); !ok {
					return err
				}
			} else { // Only traverse the specified keys of the sub tree
				log.CoreLogger.Debugf("key of %v %v", vs.Addr.Hash().Hex(), keys)
				for _, key := range keys {
					if ok, err := t.TraverseKey(key, leafCb, resolveCb, config.CheckHash); !ok {
						return err
					}
				}
			}
			config.addVisitedRoot(account.Root)
		}

		codeHash := common.BytesToHash(account.CodeHash)
		// Check the contract code of the account
		if codeHash != emptyCode {
			code, err := adb.db.TrieDB().Node(codeHash)
			if err != nil {
				return fmt.Errorf("get code %v err %v", codeHash.Hex(), err)
			}
			vs.CodeSize = uint64(len(code))
			config.OnResolve(codeHash, code)
		}
		vs.Cost = time.Since(begin)
		config.OnVisitAccount(vs)
		return nil
	}, config.ResolveNodeCb, config.CheckHash)
}
