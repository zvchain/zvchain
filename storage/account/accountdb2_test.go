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
	"fmt"
	"math/big"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
)

func TestAccountDB_AddBalance(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	defer db.Close()
	triedb := NewDatabase(db, false)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	state.SetBalance(common.BytesToAddress([]byte("1")), big.NewInt(1000000))
	state.AddBalance(common.BytesToAddress([]byte("2")), big.NewInt(1))
	state.SubBalance(common.BytesToAddress([]byte("1")), big.NewInt(2))
	root, _ := state.Commit(true)
	triedb.TrieDB().Commit(0,root, true)
	state, _ = NewAccountDB(root, triedb)
	balance := state.GetBalance(common.BytesToAddress([]byte("1")))
	if balance.Cmp(big.NewInt(999998)) != 0 {
		t.Errorf("wrong value: %s,expect value 999998", balance)
	}
	balance = state.GetBalance(common.BytesToAddress([]byte("2")))
	if balance.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("wrong value: %s,expect value 1", balance)
	}
}

func TestAccountDB_SetData(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	defer db.Close()
	triedb := NewDatabase(db, false)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	state.SetData(common.BytesToAddress([]byte("1")), []byte("aa"), []byte("v1"))
	state.SetData(common.BytesToAddress([]byte("1")), []byte("bb"), []byte("v2"))
	snapshot := state.Snapshot()
	state.SetData(common.BytesToAddress([]byte("1")), []byte("bb"), []byte("v3"))
	state.RevertToSnapshot(snapshot)
	state.SetData(common.BytesToAddress([]byte("2")), []byte("cc"), []byte("v4"))
	root, _ := state.Commit(false)
	triedb.TrieDB().Commit(0,root, false)

	state, _ = NewAccountDB(root, triedb)
	sta := state.GetData(common.BytesToAddress([]byte("1")), []byte("aa"))
	if string(sta) != "v1" {
		t.Errorf("wrong value: %s,expect value v1", sta)
	}
	sta = state.GetData(common.BytesToAddress([]byte("1")), []byte("bb"))
	if string(sta) != "v2" {
		t.Errorf("wrong value: %s,expect value v2", sta)
	}
	sta = state.GetData(common.BytesToAddress([]byte("2")), []byte("cc"))
	if string(sta) != "v4" {
		t.Errorf("wrong value: %s,expect value v4", sta)
	}
}

func TestAccountDB_SetCode(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	defer db.Close()
	triedb := NewDatabase(db, false)
	state, _ := NewAccountDB(common.Hash{}, triedb)
	state.SetCode(common.BytesToAddress([]byte("2")), []byte("code"))
	root, _ := state.Commit(false)
	triedb.TrieDB().Commit(0,root, false)

	state, _ = NewAccountDB(root, triedb)
	sta := state.GetCode(common.BytesToAddress([]byte("2")))
	if string(sta) != "code" {
		t.Errorf("wrong value: %s,expect value code", sta)
	}
}

func TestRef(t *testing.T) {
	db, _ := tasdb.NewLDBDatabase("test", nil)
	defer db.Close()
	triedb := NewDatabase(db,false)
	state, _ := NewAccountDB(common.Hash{}, triedb)

	state.SetBalance(common.BytesToAddress([]byte("11")), new(big.Int).SetInt64(1))
	state.SetBalance(common.BytesToAddress([]byte("12")), new(big.Int).SetInt64(2))

	r, e := state.Commit(true)
	if e != nil {
		t.Fatalf("state commit err %v", e)
	}
	//e = triedb.TrieDB().Commit(r, false)
	//if e != nil {
	//	t.Fatalf("db commit err %v", e)
	//}
	fmt.Println("============1================")
	state, _ = NewAccountDB(r, triedb)
	state.SetBalance(common.BytesToAddress([]byte("12")), new(big.Int).SetInt64(3))
	r, e = state.Commit(true)
	if e != nil {
		t.Fatalf("state commit err %v", e)
	}

	state, _ = NewAccountDB(r, triedb)
	state.SetBalance(common.BytesToAddress([]byte("12")), new(big.Int).SetInt64(2))
	_, e = state.Commit(true)
	if e != nil {
		t.Fatalf("state commit err %v", e)
	}
}

