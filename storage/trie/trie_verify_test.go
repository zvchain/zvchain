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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"os"
	"testing"
)

// Used for testing
func newEmptyFromDB(dir string) *Trie {
	db, _ := tasdb.NewLDBDatabase(dir, nil)
	trie, _ := NewTrie(common.Hash{}, NewDatabase(db,0,false))
	return trie
}

func TestTrie_VerifyIntegrity(t *testing.T) {
	trie := newEmptyFromDB("test_trie")

	trie.TryUpdate([]byte("1"), []byte("abc"))
	trie.TryUpdate([]byte("12"), []byte("abcd"))
	trie.TryUpdate([]byte("123"), []byte("abcdef"))

	root, err := trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}
	//root1 := root

	err = trie.db.Commit(root, false)
	if err != nil {
		t.Fatal("commit error", err)
	}

	trie.TryUpdate([]byte("1235"), []byte("abcfewgew"))
	trie.TryUpdate([]byte("3"), []byte("abcfewgew"))
	trie.TryUpdate([]byte("123235"), []byte("abcfewgew"))
	trie.TryUpdate([]byte("12f35"), []byte("abcfewgew"))

	root, err = trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = trie.db.Commit(root, false)
	if err != nil {
		t.Fatal("commit error", err)
	}

	trie.TryUpdate([]byte("af"), []byte("abcfewgew"))
	trie.TryUpdate([]byte("1"), []byte("abcfewgew"))
	trie.TryDelete([]byte("1235"))
	trie.TryDelete([]byte("12135"))

	root, err = trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = trie.db.Commit(root, false)
	if err != nil {
		t.Fatal("commit error", err)
	}

	ok, err := trie.VerifyIntegrity(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success ", ok, root.Hex())
}

func TestTrie_VerifyIntegrity_FromFile(t *testing.T) {
	if _, err := os.Stat("test_trie"); err != nil && os.IsNotExist(err) {
		return
	}
	trie := newEmptyFromDB("test_trie")
	ok, err := trie.VerifyIntegrity(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("verify fail %v", err)
	}
	t.Log("success ", ok)
}

func TestTrie_VerifyIntegrity_AfterDropKey(t *testing.T) {
	if _, err := os.Stat("test_trie"); err != nil && os.IsNotExist(err) {
		return
	}
	trie := newEmptyFromDB("test_trie")

	trie.db.diskdb.Delete(common.FromHex("0x79cf1279aa2a59f07e5bf539e6f63a86983c2341d9b851b16d55bb0e6cc539d3"))

	ok, _ := trie.VerifyIntegrity( nil)

	if ok {
		t.Fatalf("verify fail:should be missing node")
	}
	t.Log("success ", ok)
}
