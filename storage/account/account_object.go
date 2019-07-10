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

/*
	Package account is used to do accounts or contract operations
*/
package account

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/zvchain/zvchain/taslog"

	"io"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/serialize"
	"github.com/zvchain/zvchain/storage/trie"
	"golang.org/x/crypto/sha3"
)

var emptyCodeHash = sha3.Sum256(nil)

type Code []byte

func (c Code) String() string {
	return string(c)
}

var debugLog taslog.Logger

func getLogger() taslog.Logger {
	if debugLog == nil {
		instance := common.GlobalConf.GetString("instance", "index", "")
		debugLog = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
	}
	return debugLog
}

type Storage map[string][]byte

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// accountObject represents an account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a account object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type accountObject struct {
	address  common.Address
	addrHash common.Hash // hash of address of the account
	data     Account
	db       *AccountDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract code, which gets set when code is loaded

	cachedLock    sync.RWMutex
	cachedStorage Storage // Storage cache of original entries to dedup rewrites
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	dirtyCode bool // true if the code was updated
	suicided  bool
	touched   bool
	deleted   bool
	onDirty   func(addr common.Address)
}

// empty returns whether the account is considered empty.
func (ao *accountObject) empty() bool {
	//return ao.data.Nonce == 0 && ao.data.Balance.Sign() == 0 && bytes.Equal(ao.data.CodeHash, emptyCodeHash[:])
	return false
}

// Account is the consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// newObject creates a account object.
func newAccountObject(db *AccountDB, address common.Address, data Account, onDirty func(addr common.Address)) *accountObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash[:]
	}
	return &accountObject{
		db:            db,
		address:       address,
		addrHash:      sha3.Sum256(address[:]),
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (ao *accountObject) Encode(w io.Writer) error {
	return serialize.Encode(w, ao.data)
}

// setError remembers the first non-nil error it is called with.
func (ao *accountObject) setError(err error) {
	if ao.dbErr == nil {
		ao.dbErr = err
	}
}

// markSuicided only marked
func (ao *accountObject) markSuicided() {
	ao.suicided = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) touch() {
	ao.db.transitions = append(ao.db.transitions, touchChange{
		account:   &ao.address,
		prev:      ao.touched,
		prevDirty: ao.onDirty == nil,
	})
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
	ao.touched = true
}

func (ao *accountObject) getTrie(db AccountDatabase) Trie {
	if ao.trie == nil {
		tr, err := db.OpenStorageTrie(ao.addrHash, ao.data.Root)
		if err != nil {
			getLogger().Infof("access HeavyDBAddress2 find trie is nil and next get has err %v, errorMsg = %s,root is %x,addr is %p", err, err.Error(), ao.data.Root, ao)
			taslog.Flush()
			tr, _ = db.OpenStorageTrie(ao.addrHash, common.Hash{})
			ao.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
		ao.trie = tr
	}
	return ao.trie
}

// GetData retrieves a value from the account storage trie.
func (ao *accountObject) GetData(db AccountDatabase, key []byte) []byte {
	ao.cachedLock.RLock()
	// If we have the original value cached, return that
	value, exists := ao.cachedStorage[string(key)]
	ao.cachedLock.RUnlock()
	if exists {
		return value
	}
	// Otherwise load the value from the database
	value, err := ao.getTrie(db).TryGet(key)
	if err != nil {
		ao.setError(err)
		return nil
	}

	if value != nil {
		ao.cachedLock.Lock()
		ao.cachedStorage[string(key)] = value
		ao.cachedLock.Unlock()
	}
	return value
}

// SetData updates a value in account storage.
func (ao *accountObject) SetData(db AccountDatabase, key []byte, value []byte) {
	ao.db.transitions = append(ao.db.transitions, storageChange{
		account:  &ao.address,
		key:      key,
		prevalue: ao.GetData(db, key),
	})
	ao.setData(key, value)
}

func (ao *accountObject) RemoveData(db AccountDatabase, key []byte) {
	ao.SetData(db, key, nil)
}

func (ao *accountObject) setData(key []byte, value []byte) {
	ao.cachedLock.Lock()
	ao.cachedStorage[string(key)] = value
	ao.cachedLock.Unlock()
	ao.dirtyStorage[string(key)] = value

	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (ao *accountObject) updateTrie(db AccountDatabase) Trie {
	tr := ao.getTrie(db)
	// Update all the dirty slots in the trie
	for key, value := range ao.dirtyStorage {
		delete(ao.dirtyStorage, key)
		if value == nil {
			ao.setError(tr.TryDelete([]byte(key)))
			continue
		}

		ao.setError(tr.TryUpdate([]byte(key), value[:]))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (ao *accountObject) updateRoot(db AccountDatabase) {
	ao.updateTrie(db)
	ao.data.Root = ao.trie.Hash()
}

// CommitTrie the storage trie of the object to db.
// This updates the trie root.
func (ao *accountObject) CommitTrie(db AccountDatabase) error {
	ao.updateTrie(db)
	if ao.dbErr != nil {
		return ao.dbErr
	}
	root, err := ao.trie.Commit(nil)

	if err == nil {
		ao.data.Root = root
		//ao.db.db.PushTrie(root, ao.trie)
	}
	return err
}

//AddBalance is used to add funds to the destination account of a transfer.
func (ao *accountObject) AddBalance(amount *big.Int) {
	// We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if ao.empty() {
			ao.touch()
		}
		return
	}
	ao.SetBalance(new(big.Int).Add(ao.Balance(), amount))
}

// SubBalance is used to remove funds from the origin account of a transfer.
func (ao *accountObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	ao.SetBalance(new(big.Int).Sub(ao.Balance(), amount))
}

func (ao *accountObject) SetBalance(amount *big.Int) {
	ao.db.transitions = append(ao.db.transitions, balanceChange{
		account: &ao.address,
		prev:    new(big.Int).Set(ao.data.Balance),
	})
	ao.setBalance(amount)
}

func (ao *accountObject) setBalance(amount *big.Int) {
	ao.data.Balance = amount
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) deepCopy(db *AccountDB, onDirty func(addr common.Address)) *accountObject {
	accountObject := newAccountObject(db, ao.address, ao.data, onDirty)
	if ao.trie != nil {
		accountObject.trie = db.db.CopyTrie(ao.trie)
	}
	accountObject.code = ao.code
	accountObject.dirtyStorage = ao.dirtyStorage.Copy()
	accountObject.cachedStorage = ao.dirtyStorage.Copy()
	accountObject.suicided = ao.suicided
	accountObject.dirtyCode = ao.dirtyCode
	accountObject.deleted = ao.deleted
	return accountObject
}

// Returns the address of the contract/account
func (ao *accountObject) Address() common.Address {
	return ao.address
}

// Code returns the contract code associated with this object, if any.
func (ao *accountObject) Code(db AccountDatabase) []byte {
	if ao.code != nil {
		return ao.code
	}
	if bytes.Equal(ao.CodeHash(), emptyCodeHash[:]) {
		return nil
	}
	code, err := db.ContractCode(ao.addrHash, common.BytesToHash(ao.CodeHash()))
	if err != nil {
		ao.setError(fmt.Errorf("can't load code hash %x: %v", ao.CodeHash(), err))
	}
	ao.code = code
	return code
}

// DataIterator returns a new key-value iterator from a node iterator
func (ao *accountObject) DataIterator(db AccountDatabase, prefix []byte) *trie.Iterator {
	if ao.trie == nil {
		ao.getTrie(db)
	}
	return trie.NewIterator(ao.trie.NodeIterator(prefix))
}

// SetCode set a value in contract storage.
func (ao *accountObject) SetCode(codeHash common.Hash, code []byte) {
	prevCode := ao.Code(ao.db.db)
	ao.db.transitions = append(ao.db.transitions, codeChange{
		account:  &ao.address,
		prevhash: ao.CodeHash(),
		prevcode: prevCode,
	})
	ao.setCode(codeHash, code)
}

func (ao *accountObject) setCode(codeHash common.Hash, code []byte) {
	ao.code = code
	ao.data.CodeHash = codeHash[:]
	ao.dirtyCode = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

// SetCode update nonce in account storage.
func (ao *accountObject) SetNonce(nonce uint64) {
	ao.db.transitions = append(ao.db.transitions, nonceChange{
		account: &ao.address,
		prev:    ao.data.Nonce,
	})
	ao.setNonce(nonce)
}

func (ao *accountObject) setNonce(nonce uint64) {
	ao.data.Nonce = nonce
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

// CodeHash returns code's hash
func (ao *accountObject) CodeHash() []byte {
	return ao.data.CodeHash
}

func (ao *accountObject) Balance() *big.Int {
	return ao.data.Balance
}

func (ao *accountObject) Nonce() uint64 {
	return ao.data.Nonce
}

func (ao *accountObject) Value() *big.Int {
	panic("Value on accountObject should never be called")
}
