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
	"github.com/zvchain/zvchain/storage/tasdb"
	"testing"
)

// Used for testing
func newTrieFromDB(dir string, root common.Hash) *Trie {
	db, _ := tasdb.NewLDBDatabase(dir, nil)
	return newTrieWithDB(db, root)
}

func newTrieWithDB(db tasdb.Database, root common.Hash) *Trie {
	trie, _ := NewTrie(root, NewDatabase(db, 0, "", false))
	return trie
}

func TestTrie_Traverse(t *testing.T) {
	trie := newTrieFromDB("test_trie", common.Hash{})

	trie.TryUpdate([]byte("1"), []byte("abc"))
	trie.TryUpdate([]byte("12"), []byte("abcd"))
	trie.TryUpdate([]byte("123"), []byte("abcdef"))

	root, err := trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}
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

	ok, err := trie.Traverse(nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success ", ok, root.Hex())
}

func TestTrie_VerifyIntegrity_FromFile(t *testing.T) {
	//if _, err := os.Stat("test_trie"); err != nil && os.IsNotExist(err) {
	//	return
	//}
	//trie := newTrieFromDB("test_trie", common.HexToHash("0x30d38a45ef853e4ea2477074b7a39d2608441e2268812dd0a35ba3413694656d"))
	//ok, err := trie.VerifyIntegrity(nil, nil,false)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//if !ok {
	//	t.Fatalf("verify fail %v", err)
	//}
	//t.Log("success ", ok)
}

func TestTrie_VerifyIntegrity_AfterDropKey(t *testing.T) {
	//if _, err := os.Stat("test_trie"); err != nil && os.IsNotExist(err) {
	//	return
	//}
	//trie := newTrieFromDB("test_trie", common.HexToHash("0x30d38a45ef853e4ea2477074b7a39d2608441e2268812dd0a35ba3413694656d"))
	//
	//trie.db.diskdb.Delete(common.FromHex("0x79cf1279aa2a59f07e5bf539e6f63a86983c2341d9b851b16d55bb0e6cc539d3"))
	//
	//ok, _ := trie.VerifyIntegrity(nil, nil,false)
	//
	//if ok {
	//	t.Fatalf("verify fail:should be missing node")
	//}
	//t.Log("success ", ok)
}

func TestTrie_Traverse2(t *testing.T) {
	trie := newTrieFromDB("test_trie", common.Hash{})

	trie.TryUpdate([]byte("1"), []byte("abc"))
	trie.TryUpdate([]byte("12"), []byte("abcd"))
	trie.TryUpdate([]byte("123"), []byte("abcdef"))

	root, err := trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}

	trie2 := newTrieWithDB(trie.db.diskdb, root)

	trie2.Traverse(func(key []byte, value []byte) error {
		fmt.Println(string(key), string(value))
		return nil
	}, nil, false)

}
