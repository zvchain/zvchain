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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"sync"
	"sync/atomic"
)

type ExtLeafCallback func(key []byte, value []byte) error
type ResolveNodeCallback func(hash common.Hash, data []byte)

type checkErrorFn func() error

// VerifyIntegrity is a debug method to iterate over the entire trie stored in
// the disk and check whether every node is reachable from the meta root. The goal
// is to find any errors that might cause trie nodes missing during prune
//
// This method is extremely CPU and disk intensive, and time consuming, only use when must.
func (t *Trie) VerifyIntegrity(onleaf ExtLeafCallback, resolve ResolveNodeCallback) (bool, error) {
	return t.verifyIntegrity(hashNode(t.originalRoot.Bytes()), []byte{}, onleaf, resolve, false, nil)
}

func (t *Trie) verifyFullNodeConcurrently(fn *fullNode, accumulateKey []byte, onleaf ExtLeafCallback, resolve ResolveNodeCallback) (bool, error) {
	wg := sync.WaitGroup{}
	errV := atomic.Value{}

	stopCheckFn := func() error {
		if e := errV.Load(); e != nil {
			return e.(error)
		}
		return nil
	}

	for i, child := range fn.Children {
		if child != nil {
			wg.Add(1)
			go func(n node) {
				defer wg.Done()
				if ok, err := t.verifyIntegrity(n, append(accumulateKey, byte(i)), onleaf, resolve, false, stopCheckFn); !ok {
					errV.Store(err)
					return
				}
			}(child)
		}
	}
	wg.Wait()
	err := errV.Load()
	if err == nil {
		return true, nil
	}
	return false, err.(error)
}

func (t *Trie) verifyIntegrity(nd node, accumulateKey []byte, onleaf ExtLeafCallback, resolve ResolveNodeCallback, concurrent bool, errCheckFn checkErrorFn) (ok bool, err error) {
	if errCheckFn != nil {
		if e := errCheckFn(); e != nil {
			return false, e
		}
	}
	switch n := (nd).(type) {
	case nil:
		return true, nil
	case valueNode:
		if onleaf != nil {
			uKey := hexToKeybytes(accumulateKey)
			if e := onleaf(uKey, n); e != nil {
				return false, e
			}
		}
		return true, nil
	case *shortNode:
		ok, err = t.verifyIntegrity(n.Val, append(accumulateKey, n.Key...), onleaf, resolve, false, errCheckFn)
		if !ok {
			return
		}
	case *fullNode:
		if concurrent {
			ok, err = t.verifyFullNodeConcurrently(n, accumulateKey, onleaf, resolve)
			if !ok {
				return
			}
		} else {
			for i, child := range n.Children {
				if child != nil {
					if ok, err = t.verifyIntegrity(child, append(accumulateKey, byte(i)), onleaf, resolve, false, errCheckFn); !ok {
						return
					}
				}
			}
		}
	case hashNode:
		hash := common.BytesToHash(n)
		var (
			resolvedNode node
			data         []byte
		)
		if hash != (common.Hash{}) && hash != emptyRoot {
			r, bs, e := t.resolveHashAndGetRawBytes(n, accumulateKey)
			if e != nil {
				fmt.Println("missing", common.ToHex(n), common.ToHex(hexToKeybytes(accumulateKey)), string(hexToKeybytes(accumulateKey)))
				return false, e
			}
			resolvedNode = r
			data = bs
		}
		if resolve != nil {
			resolve(common.BytesToHash(n), data)
		}
		if ok, err = t.verifyIntegrity(resolvedNode, accumulateKey, onleaf, resolve, concurrent, errCheckFn); !ok {
			return
		}
	default:
		panic(fmt.Sprintf("%T: invalid n: %v", n, n))
	}
	return true, nil
}
