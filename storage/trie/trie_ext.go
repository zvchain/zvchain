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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zvchain/zvchain/common"
)

type ExtLeafCallback func(key []byte, value []byte) error
type ResolveNodeCallback func(hash common.Hash, data []byte)

type checkErrorFn func() error

// Traverse is a debug method to iterate over the entire trie stored in
// the disk and check whether every node is reachable from the meta root. The goal
// is to find any errors that might cause trie nodes missing during prune
//
// This method is extremely CPU and disk intensive, and time consuming, only use when must.
func (t *Trie) Traverse(onleaf ExtLeafCallback, resolve ResolveNodeCallback, checkHash bool) (bool, error) {
	return t.traverse(hashNode(t.originalRoot.Bytes()), []byte{}, onleaf, resolve, true, nil, checkHash)
}

func (t *Trie) traverseFullNodeConcurrently(fn *fullNode, accumulateKey []byte, onleaf ExtLeafCallback, resolve ResolveNodeCallback, checkHash bool) (bool, error) {
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
			go func(idx byte) {
				defer wg.Done()
				if ok, err := t.traverse(fn.Children[idx], append(accumulateKey, idx), onleaf, resolve, false, stopCheckFn, checkHash); !ok {
					errV.Store(err)
					return
				}
			}(byte(i))
		}
	}
	wg.Wait()
	err := errV.Load()
	if err == nil {
		return true, nil
	}
	return false, err.(error)
}

func (t *Trie) traverse(nd node, accumulateKey []byte, onleaf ExtLeafCallback, resolve ResolveNodeCallback, concurrent bool, errCheckFn checkErrorFn, checkHash bool) (ok bool, err error) {
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
		ok, err = t.traverse(n.Val, append(accumulateKey, n.Key...), onleaf, resolve, false, errCheckFn, checkHash)
		if !ok {
			return
		}
	case *fullNode:
		if concurrent {
			ok, err = t.traverseFullNodeConcurrently(n, accumulateKey, onleaf, resolve, checkHash)
			if !ok {
				return
			}
		} else {
			for i, child := range n.Children {
				if child != nil {
					if ok, err = t.traverse(child, append(accumulateKey, byte(i)), onleaf, resolve, false, errCheckFn, checkHash); !ok {
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
				return false, e
			}
			resolvedNode = r
			data = bs
			if checkHash {
				hasher := newHasher(0, 0, nil)
				if !bytes.Equal(n, hasher.makeHashNode(bs)) {
					fmt.Printf("hash check failed:  %v", common.Bytes2Hex(n))
					return false, errors.New(fmt.Sprintf("hash check failed:  %v", common.Bytes2Hex(n)))
				}
			}
		}
		if resolve != nil {
			resolve(hash, data)
		}
		if ok, err = t.traverse(resolvedNode, accumulateKey, onleaf, resolve, concurrent, errCheckFn, checkHash); !ok {
			return
		}
	default:
		panic(fmt.Sprintf("%T: invalid n: %v", n, n))
	}
	return true, nil
}

// TraverseKey traverses the path of the given key in the trie tree,
// An error is returned when the specified path cannot be found or some node is missing
func (t *Trie) TraverseKey(key []byte, onleaf ExtLeafCallback, resolve ResolveNodeCallback, checkHash bool) (ok bool, err error) {
	key = keybytesToHex(key)
	return t.traverseKey(hashNode(t.originalRoot.Bytes()), key, 0, onleaf, resolve, checkHash)
}

func (t *Trie) traverseKey(origNode node, key []byte, pos int, onleaf ExtLeafCallback, resolve ResolveNodeCallback, checkHash bool) (ok bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return true, nil
	case valueNode:
		if onleaf != nil {
			uKey := hexToKeybytes(key)
			if e := onleaf(uKey, n); e != nil {
				return false, e
			}
		}
		return true, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return false, fmt.Errorf("key not found %x", key)
		}
		return t.traverseKey(n.Val, key, pos+len(n.Key), onleaf, resolve, checkHash)
	case *fullNode:
		return t.traverseKey(n.Children[key[pos]], key, pos+1, onleaf, resolve, checkHash)
	case hashNode:
		hash := common.BytesToHash(n)

		var (
			resolvedNode node
			data         []byte
		)
		if hash != (common.Hash{}) && hash != emptyRoot {
			r, bs, e := t.resolveHashAndGetRawBytes(n, key[:pos])
			if e != nil {
				return false, e
			}
			resolvedNode = r
			data = bs
			if checkHash {
				hasher := newHasher(0, 0, nil)
				if !bytes.Equal(n, hasher.makeHashNode(bs)) {
					fmt.Printf("hash check failed:  %v", common.Bytes2Hex(n))
					return false, errors.New(fmt.Sprintf("hash check failed:  %v", common.Bytes2Hex(n)))
				}
			}
		}
		if resolve != nil {
			resolve(hash, data)
		}
		return t.traverseKey(resolvedNode, key, pos, onleaf, resolve, checkHash)
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}
