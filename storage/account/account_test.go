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
	"bytes"
	"github.com/zvchain/zvchain/taslog"
	"math/big"
	"sync"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/sha3"
	"github.com/zvchain/zvchain/storage/tasdb"
)

func init() {
	common.InitConf("../../cmd/gtas/cli/tas.ini")
	instance := common.GlobalConf.GetString("instance", "index", "")
	debugLog = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
}

type StateSuite struct {
	db    *tasdb.MemDatabase
	state *AccountDB
}

func setUp() *StateSuite {
	s := new(StateSuite)
	s.db, _ = tasdb.NewMemDatabase()
	s.state, _ = NewAccountDB(common.Hash{}, NewDatabase(s.db))
	return s
}

var toAddr = common.BytesToAddress

func TestNull(t *testing.T) {
	s := setUp()
	address := common.HexToAddress("0x823140710bf13990e4500136726d8b55")
	s.state.CreateAccount(address)
	s.state.SetData(address, "emptykey", []byte(""))
	s.state.Commit(false)
	value := s.state.GetData(address, "emptykey")
	if string(value) != "" {
		t.Errorf("expected empty hash. got %x", value)
	}
}

func TestSnapshot(t *testing.T) {
	s := setUp()
	stateObjAddr := toAddr([]byte("aa"))
	var storageAddr = "test"
	data1 := []byte("value1")
	data2 := []byte("value2")

	var addr = common.HexToAddress("0x12345")

	// set initial state object value
	s.state.SetData(stateObjAddr, storageAddr, data1[:])
	s.state.AddBalance(addr, new(big.Int).SetUint64(10000))

	t.Log(s.state.GetBalance(addr))
	// get snapshot of current state
	snapshot := s.state.Snapshot()

	// set new state object value
	s.state.SetData(stateObjAddr, storageAddr, data2[:])
	s.state.AddBalance(addr, new(big.Int).SetUint64(200))

	t.Log(s.state.GetBalance(addr))
	// restore snapshot
	s.state.RevertToSnapshot(snapshot)

	b := s.state.GetBalance(addr)
	t.Log(b)
	if b.Uint64() != 10000 {
		t.Fatal("balance error")
	}
	// get state storage value
	res := s.state.GetData(stateObjAddr, storageAddr)

	if string(res) != "value1" {
		t.Errorf("expected empty hash. got %s", res)
	}
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// use testing instead of checker because checker does not support
// printing/logging in tests (-check.vv does not work)
func TestSnapshot2(t *testing.T) {
	db, _ := tasdb.NewMemDatabase()
	state, _ := NewAccountDB(common.Hash{}, NewDatabase(db))

	stateobjaddr0 := "so0"
	stateobjaddr1 := "so1"

	data0 := common.BytesToHash([]byte{17})
	data1 := common.BytesToHash([]byte{18})

	state.SetData(toAddr([]byte(stateobjaddr0)), "", data0[:])
	state.SetData(toAddr([]byte(stateobjaddr1)), "", data1[:])

	// db, trie are already non-empty values
	so0 := state.getAccountObject(toAddr([]byte(stateobjaddr0)))
	so0.SetBalance(big.NewInt(42))
	so0.SetNonce(43)
	so0.SetCode(Keccak256Hash([]byte{'c', 'a', 'f', 'e'}), []byte{'c', 'a', 'f', 'e'})
	so0.suicided = false
	so0.deleted = false
	state.setAccountObject(so0)

	root, _ := state.Commit(false)
	state.Reset(root)

	// and one with deleted == true
	so1 := state.getAccountObject(toAddr([]byte(stateobjaddr1)))
	so1.SetBalance(big.NewInt(52))
	so1.SetNonce(53)
	so1.SetCode(Keccak256Hash([]byte{'c', 'a', 'f', 'e', '2'}), []byte{'c', 'a', 'f', 'e', '2'})
	so1.suicided = true
	so1.deleted = true
	state.setAccountObject(so1)

	so1 = state.getAccountObject(toAddr([]byte(stateobjaddr1)))
	if so1 != nil {
		t.Fatalf("deleted object not nil when getting")
	}

	snapshot := state.Snapshot()
	state.RevertToSnapshot(snapshot)

	so0Restored := state.getAccountObject(toAddr([]byte(stateobjaddr0)))
	// Update lazily-loaded values before comparing.
	so0Restored.GetData(state.db, "")
	so0Restored.Code(state.db)
	// non-deleted is equal (restored)
	compareStateObjects(so0Restored, so0, t)

	// deleted should be nil, both before and after restore of state copy
	so1Restored := state.getAccountObject(toAddr([]byte(stateobjaddr1)))
	if so1Restored != nil {
		t.Fatalf("deleted object not nil after restoring snapshot: %+v", so1Restored)
	}
}

func compareStateObjects(so0, so1 *accountObject, t *testing.T) {
	if so0.Address() != so1.Address() {
		t.Fatalf("Address mismatch: have %v, want %v", so0.address, so1.address)
	}
	if so0.Balance().Cmp(so1.Balance()) != 0 {
		t.Fatalf("Balance mismatch: have %v, want %v", so0.Balance(), so1.Balance())
	}
	if so0.Nonce() != so1.Nonce() {
		t.Fatalf("Nonce mismatch: have %v, want %v", so0.Nonce(), so1.Nonce())
	}
	if so0.data.Root != so1.data.Root {
		t.Errorf("Root mismatch: have %x, want %x", so0.data.Root[:], so1.data.Root[:])
	}
	if !bytes.Equal(so0.CodeHash(), so1.CodeHash()) {
		t.Fatalf("CodeHash mismatch: have %v, want %v", so0.CodeHash(), so1.CodeHash())
	}
	if !bytes.Equal(so0.code, so1.code) {
		t.Fatalf("Code mismatch: have %v, want %v", so0.code, so1.code)
	}

	if len(so1.dirtyStorage) != len(so0.dirtyStorage) {
		t.Errorf("Dirty storage size mismatch: have %d, want %d", len(so1.dirtyStorage), len(so0.dirtyStorage))
	}
	for k, v := range so1.dirtyStorage {
		if string(so0.dirtyStorage[k]) != string(v) {
			t.Errorf("Dirty storage key %x mismatch: have %v, want %v", k, so0.dirtyStorage[k], v)
		}
	}
	for k, v := range so0.dirtyStorage {
		if string(so1.dirtyStorage[k]) != string(v) {
			t.Errorf("Dirty storage key %x mismatch: have %v, want none.", k, v)
		}
	}
	if len(so1.cachedStorage) != len(so0.cachedStorage) {
		t.Errorf("Origin storage size mismatch: have %d, want %d", len(so1.cachedStorage), len(so0.cachedStorage))
	}
	for k, v := range so1.cachedStorage {
		if string(so0.cachedStorage[k]) != string(v) {
			t.Errorf("Origin storage key %x mismatch: have %v, want %v", k, so0.cachedStorage[k], v)
		}
	}
	for k, v := range so0.cachedStorage {
		if string(so1.cachedStorage[k]) != string(v) {
			t.Errorf("Origin storage key %x mismatch: have %v, want none.", k, v)
		}
	}
}

func TestGetData(t *testing.T) {
	s := setUp()

	wg := &sync.WaitGroup{}
	for a := 0; a < 100; a++ {
		addr := common.BytesToAddress(common.Int32ToByte(int32(a)))
		s.state.SetData(addr, "1", []byte("234444"))

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.state.GetData(addr, "1")
			}()
		}
		s.state.Commit(true)
	}
	wg.Wait()
}
