//   Copyright (C) 2018 ZVChain
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
	"strconv"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"github.com/zvchain/zvchain/storage/trie"
)

func getString(trie *trie.Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *trie.Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *trie.Trie, k string) {
	trie.Delete([]byte(k))
}
func TestExpandTrie(t *testing.T) {
	diskdb, _ := tasdb.NewMemDatabase()
	triedb := NewDatabase(diskdb, false)
	trie1, _ := trie.NewTrie(common.Hash{}, triedb.TrieDB())

	for i := 0; i < 100; i++ {
		updateString(trie1, strconv.Itoa(i), strconv.Itoa(i))
	}
	trie1.SetCacheLimit(10)
	for i := 0; i < 11; i++ {
		trie1.Commit(nil)
	}

	root, _ := trie1.Commit(nil)
	triedb.TrieDB().Commit(root, false)

	for i := 0; i < 100; i++ {
		vl := string(getString(trie1, strconv.Itoa(i)))
		if vl != strconv.Itoa(i) {
			t.Errorf("wrong value: %v", vl)
		}
	}
}
