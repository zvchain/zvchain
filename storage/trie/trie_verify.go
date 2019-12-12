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
)

type VerifyLeafCallback func(key []byte, value []byte) error

// VerifyIntegrity is a debug method to iterate over the entire trie stored in
// the disk and check whether every node is reachable from the meta root. The goal
// is to find any errors that might cause trie nodes missing during prune
//
// This method is extremely CPU and disk intensive, and time consuming, only use when must.
func (t *Trie) VerifyIntegrity(onleaf VerifyLeafCallback) (bool, error) {
	return t.verifyIntegrity(t.root, []byte{}, onleaf)
}

func (t *Trie) verifyIntegrity(node node, accumulateKey []byte, onleaf VerifyLeafCallback) (ok bool, err error) {
	switch n := (node).(type) {
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
		ok, err = t.verifyIntegrity(n.Val, append(accumulateKey, n.Key...), onleaf)
		if !ok {
			return
		}
	case *fullNode:
		for i, child := range n.Children {
			if child != nil {
				if ok, err = t.verifyIntegrity(child, append(accumulateKey, byte(i)), onleaf); !ok {
					return
				}
			}
		}
	case hashNode:
		r, e := t.resolve(n, accumulateKey)
		if e != nil {
			return false, e
		}
		if ok, err = t.verifyIntegrity(r, accumulateKey, onleaf); !ok {
			return
		}
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
	return true, nil
}
