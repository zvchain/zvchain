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

package cli

import (
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/storage/tasdb"
	"golang.org/x/crypto/scrypt"
	"os"
	"sync"
)

type Account struct {
	Address  string
	Pk       string
	Sk       string
	Password string
	Miner    *MinerRaw
}

func (a *Account) MinerSk() string {
	return a.Sk
}

type MinerRaw struct {
	BPk   string
	BSk   string
	VrfPk string
	VrfSk string
}

type KeyStoreRaw struct {
	Key     []byte
	IsMiner bool
}

func passwordHash(password string) string {
	return common.ToHex(common.Sha256([]byte(password)))
}

func dirExists(dir string) bool {
	f, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return f.IsDir()
}

type KeyStoreMgr struct {
	Accounts   sync.Map
	Path       string
	Store      *tasdb.LDBDatabase
	AccountMsg *Account
}

func NewkeyStoreMgr(ks string) (*KeyStoreMgr, error) {
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
	return &KeyStoreMgr{
		Path:  ks,
		Store: db,
	}, nil
}

func (ksm *KeyStoreMgr) CheckMinerAccount(addr string, password string) (*Account, error) {

	acc := new(Account)
	var err error
	if v, ok := ksm.Accounts.Load(addr); ok {
		acc = v.(*Account)
		if passwordHash(password) != acc.Password {
			return nil, ErrPassword
		}
	} else {
		acc, err = ksm.LoadKeyStore(addr, password)
		if err != nil {
			return nil, err
		}
		ksm.Accounts.Store(addr, acc)
	}
	return acc, nil
}

func (ksm *KeyStoreMgr) LoadKeyStore(addr string, password string) (*Account, error) {
	v, err := ksm.Store.Get([]byte(addr))
	if err != nil {
		return nil, err
	}

	salt := common.Sha256([]byte(password))
	scryptPwd, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	bs, err := common.DecryptWithKey(scryptPwd, v)
	if err != nil {
		return nil, err
	}

	var ksr = new(KeyStoreRaw)
	if err = json.Unmarshal(bs, ksr); err != nil {
		return nil, err
	}

	secKey := new(common.PrivateKey)
	if !secKey.ImportKey(ksr.Key) {
		return nil, ErrInternal
	}

	return ksm.ConstructAccount(password, secKey, ksr.IsMiner)
}

func (ksm *KeyStoreMgr) ConstructAccount(password string, sk *common.PrivateKey, bMiner bool) (*Account, error) {
	account := &Account{
		Sk:       sk.Hex(),
		Pk:       sk.GetPubKey().Hex(),
		Address:  sk.GetPubKey().GetAddress().AddrPrefixString(),
		Password: passwordHash(password),
	}

	if bMiner {
		minerDO, err := model.NewSelfMinerDO(sk)
		if err != nil {
			return nil, err
		}

		minerRaw := &MinerRaw{
			BPk:   minerDO.PK.GetHexString(),
			BSk:   minerDO.SK.GetHexString(),
			VrfPk: minerDO.VrfPK.GetHexString(),
			VrfSk: minerDO.VrfSK.GetHexString(),
		}
		account.Miner = minerRaw
	}
	return account, nil
}

func (ksm *KeyStoreMgr) GetFirstMinerAccount(password string) *Account {
	iter := ksm.Store.NewIterator()
	for iter.Next() {
		addr := string(iter.Key())
		if v, ok := ksm.Accounts.Load(addr); ok {
			acc := v.(*Account)
			if passwordHash(password) == acc.Password && acc.Miner != nil {
				return acc
			}
		} else {
			acc, err := ksm.LoadKeyStore(addr, password)
			if err == nil && acc.Miner != nil {
				return acc
			}
		}
	}
	return nil
}

func (ksm *KeyStoreMgr) StoreAccount(addr string, ksr *KeyStoreRaw, password string) error {
	bs, err := json.Marshal(ksr)
	if err != nil {
		return err
	}

	salt := common.Sha256([]byte(password))
	scryptPwd, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return err
	}
	ct, err := common.EncryptWithKey(scryptPwd, bs)
	if err != nil {
		return err
	}

	err = ksm.Store.Put([]byte(addr), ct)
	return err
}

func (ksm *KeyStoreMgr) Close() {
	ksm.Store.Close()
}
