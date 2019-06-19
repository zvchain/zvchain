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
	Package cli provides client command line window
*/
package cli

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/storage/tasdb"

	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const accountUnLockTime = time.Second * 120

var encryptPrivateKey *common.PrivateKey
var encryptPublicKey *common.PublicKey

//Generate public and private keys based on passwords
func init() {
	encryptPrivateKey = common.HexToSecKey("0x04b851c3551779125a588b2274cfa6d71604fe6ae1f0df82175bcd6e6c2b23d92a69d507023628b59c15355f3cbc0d8f74633618facd28632a0fb3e9cc8851536c4b3f1ea7c7fd3666ce8334301236c2437d9bed14e5a0793b51a9a6e7a4c46e70")
	pk := encryptPrivateKey.GetPubKey()
	encryptPublicKey = &pk
}

const (
	statusLocked   int8 = 0
	statusUnLocked      = 1
)
const DefaultPassword = "123"

type AccountManager struct {
	store    *tasdb.LDBDatabase
	accounts sync.Map

	unlockAccount *AccountInfo
	mu            sync.Mutex
}

type AccountInfo struct {
	Account
	Status       int8
	UnLockExpire time.Time
}

func (ai *AccountInfo) unlocked() bool {
	return time.Now().Before(ai.UnLockExpire) && ai.Status == statusUnLocked
}

func (ai *AccountInfo) resetExpireTime() {
	ai.UnLockExpire = time.Now().Add(accountUnLockTime)
}

type Account struct {
	Address  string
	Pk       string
	Sk       string
	Password string
	Miner    *MinerRaw
}

type MinerRaw struct {
	BPk   string
	BSk   string
	VrfPk string
	VrfSk string
}

func dirExists(dir string) bool {
	f, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return f.IsDir()
}

func newAccountOp(ks string) (*AccountManager, error) {
	options := &opt.Options{
		OpenFilesCacheCapacity:        10,
		WriteBuffer:                   8 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           2 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
	}
	db, err := tasdb.NewLDBDatabase(ks, options)
	if err != nil {
		return nil, fmt.Errorf("new ldb fail:%v", err.Error())
	}
	return &AccountManager{
		store: db,
	}, nil
}

func initAccountManager(keystore string, readyOnly bool) (accountOp, error) {
	// Specify internal account creation when you deploy in bulk (just create it once)
	if readyOnly && !dirExists(keystore) {
		aop, err := newAccountOp(keystore)
		if err != nil {
			return nil, err
		}
		ret := aop.NewAccount(DefaultPassword, true)
		if !ret.IsSuccess() {
			fmt.Println(ret.Message)
			return nil, err
		}
		return aop, nil
	}

	aop, err := newAccountOp(keystore)
	if err != nil {
		return nil, err
	}
	return aop, nil
}

func (am *AccountManager) loadAccount(addr string) (*Account, error) {
	v, err := am.store.Get([]byte(addr))
	if err != nil {
		return nil, err
	}

	bs, err := encryptPrivateKey.Decrypt(rand.Reader, v)
	if err != nil {
		return nil, err
	}

	var acc = new(Account)
	err = json.Unmarshal(bs, acc)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func (am *AccountManager) storeAccount(account *Account) error {
	bs, err := json.Marshal(account)
	if err != nil {
		return err
	}

	ct, err := common.Encrypt(rand.Reader, encryptPublicKey, bs)
	if err != nil {
		return err
	}

	err = am.store.Put([]byte(account.Address), ct)
	return err
}

func (am *AccountManager) getFirstMinerAccount() *Account {
	iter := am.store.NewIterator()
	for iter.Next() {
		if ac, err := am.getAccountInfo(string(iter.Key())); err != nil {
			fmt.Printf("getAccountInfo err,addr=%v,err=%v", string(iter.Key()), err.Error())
			return nil
		} else {
			if ac.Miner != nil {
				return &ac.Account
			}
		}
	}
	return nil
}

func (am *AccountManager) resetExpireTime(addr string) {
	acc, err := am.getAccountInfo(addr)
	if err != nil {
		return
	}
	acc.resetExpireTime()
}

func (am *AccountManager) getAccountInfo(addr string) (*AccountInfo, error) {
	var aci *AccountInfo
	if v, ok := am.accounts.Load(addr); ok {
		aci = v.(*AccountInfo)
	} else {
		acc, err := am.loadAccount(addr)
		if err != nil {
			return nil, err
		}
		aci = &AccountInfo{
			Account: *acc,
		}
		am.accounts.Store(addr, aci)
	}
	return aci, nil
}

func (am *AccountManager) currentUnLockedAddr() string {
	if am.unlockAccount != nil && am.unlockAccount.unlocked() {
		return am.unlockAccount.Address
	}
	return ""
}

func passwordSha(password string) string {
	return common.ToHex(common.Sha256([]byte(password)))
}

// NewAccount create a new account by password
func (am *AccountManager) NewAccount(password string, miner bool) *Result {
	privateKey, err := common.GenerateKey("")
	if err != nil {
		return opError(err)
	}
	pubkey := privateKey.GetPubKey()
	address := pubkey.GetAddress()

	account := &Account{
		Address:  address.Hex(),
		Pk:       pubkey.Hex(),
		Sk:       privateKey.Hex(),
		Password: passwordSha(password),
	}

	if miner {
		minerDO := model.NewSelfMinerDO(&privateKey)

		minerRaw := &MinerRaw{
			BPk:   minerDO.PK.GetHexString(),
			BSk:   minerDO.SK.GetHexString(),
			VrfPk: minerDO.VrfPK.GetHexString(),
			VrfSk: minerDO.VrfSK.GetHexString(),
		}
		account.Miner = minerRaw
	}
	if err := am.storeAccount(account); err != nil {
		return opError(err)
	}

	return opSuccess(address.Hex())
}

// AccountList show account list
func (am *AccountManager) AccountList() *Result {
	iter := am.store.NewIterator()
	addrs := make([]string, 0)
	for iter.Next() {
		addrs = append(addrs, string(iter.Key()))
	}
	return opSuccess(addrs)
}

// Lock lock the account by address
func (am *AccountManager) Lock(addr string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	aci.Status = statusLocked
	return opSuccess(nil)
}

// UnLock unlock the account by address and password
func (am *AccountManager) UnLock(addr string, password string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if aci.Password != passwordSha(password) {
		return opError(ErrPassword)
	}
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.unlockAccount != nil && aci.Address != am.unlockAccount.Address {
		am.unlockAccount.Status = statusLocked
	}

	aci.Status = statusUnLocked
	aci.resetExpireTime()
	am.unlockAccount = aci

	return opSuccess(nil)
}

// AccountInfo show account info
func (am *AccountManager) AccountInfo() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	aci.resetExpireTime()
	return opSuccess(&aci.Account)
}

// DeleteAccount delete current unlocked account
func (am *AccountManager) DeleteAccount() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	am.accounts.Delete(addr)
	am.store.Delete([]byte(addr))
	return opSuccess(nil)
}

func (am *AccountManager) Close() {
	am.store.Close()
}
