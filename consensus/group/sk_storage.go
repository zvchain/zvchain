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

package group

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"io"
	"modernc.org/kv"
	"os"
	"path/filepath"
	"sync"
)

const (
	maxItems        = 20000 // Maximum seckeys stored in the skStorage. It only need to keep the lived-group-msk which the number is far enough
	expireHeightGap = 2 * lifeWindow
	skEncVersion    = 1
)

var (
	prefixHash         = "hash-"
	prefixExpireHeight = "expi-"
)

var opt = &kv.Options{
	VerifyDbBeforeOpen:  false,
	VerifyDbAfterOpen:   true,
	VerifyDbBeforeClose: true,
	VerifyDbAfterClose:  false,
}

type skInfo struct {
	msk   groupsig.Seckey
	encSk groupsig.Seckey
}

func (si *skInfo) toBytes() []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(skEncVersion)
	mskBytes := make([]byte, groupsig.SkLength)
	copy(mskBytes, si.msk.Serialize())
	buf.Write(mskBytes)
	encBytes := make([]byte, groupsig.SkLength)
	copy(encBytes, si.encSk.Serialize())
	buf.Write(encBytes)
	return buf.Bytes()
}

func decodeSkInfoBytes(bs []byte) *skInfo {
	reader := bytes.NewReader(bs)
	version, err := reader.ReadByte()
	if err != nil {
		return nil
	}
	if version != skEncVersion {
		return nil
	}
	msk := make([]byte, groupsig.SkLength)
	_, err = reader.Read(msk)
	if err != nil {
		return nil
	}
	encSk := make([]byte, groupsig.SkLength)
	_, err = reader.Read(encSk)
	if err != nil {
		return nil
	}
	return &skInfo{
		msk:   *groupsig.DeserializeSeckey(msk),
		encSk: *groupsig.DeserializeSeckey(encSk),
	}
}

func hashKey(hash common.Hash) []byte {
	return append([]byte(prefixHash), hash.Bytes()...)
}

func heightKey(h uint64) []byte {
	return append([]byte(prefixExpireHeight), common.Uint64ToByte(h)...)
}

type skStorage struct {
	file   string
	encKey []byte
	db     *kv.DB

	blockAddCh chan uint64
	mu         sync.Mutex
}

type lockCloser struct {
	f    *os.File
	abs  string
	once sync.Once
	err  error
}

func (lc *lockCloser) Close() error {
	lc.once.Do(lc.close)
	return lc.err
}

func (lc *lockCloser) close() {
	if err := lc.f.Close(); err != nil {
		lc.err = err
	}
	if err := os.Remove(lc.abs); err != nil {
		lc.err = err
	}
}

func createOrOpenDB(file string) (*kv.DB, error) {
	lockFile := file + ".lock"
	_, err := os.Stat(lockFile)
	if err == nil || os.IsExist(err) {
		logger.Debugf("remove lock file %v", lockFile)
		os.Remove(lockFile)
	}

	opt.Locker = func(name string) (closer io.Closer, e error) {
		lname := name + ".lock"
		abs, err := filepath.Abs(lname)
		if err != nil {
			return nil, err
		}
		f, err := os.OpenFile(abs, os.O_CREATE|os.O_EXCL|os.O_RDONLY, 0666)
		if os.IsExist(err) {
			return nil, fmt.Errorf("cannot access DB %q: lock file %q exists", name, abs)
		}
		if err != nil {
			return nil, err
		}
		return &lockCloser{f: f, abs: abs}, nil
	}

	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		return kv.Create(file, opt)
	} else {
		return kv.Open(file, opt)
	}
}

func newSkStorage(file string, encKey []byte) *skStorage {
	db, err := createOrOpenDB(file)
	if err != nil {
		panic(fmt.Errorf("create db fail:%v in %v", err, file))
	}
	return &skStorage{
		file:       file,
		encKey:     encKey,
		blockAddCh: make(chan uint64, 5),
		db:         db,
	}
}

func (store *skStorage) loop() {
	for {
		select {
		case height := <-store.blockAddCh:
			store.removeExpires(height)
		}
	}
}

func (store *skStorage) flush() {

}

func (store *skStorage) removeExpires(height uint64) {
	store.mu.Lock()
	defer store.mu.Unlock()

	iter, _, err := store.db.Seek([]byte(prefixExpireHeight))
	if err != nil {
		logger.Errorf("seek error:%v", err)
		return
	}
	err = store.db.BeginTransaction()
	if err != nil {
		store.db.Rollback()
		logger.Errorf("begin transaction error %v", err)
		return
	}
	defer store.db.Commit()
	for {
		if k, v, err := iter.Next(); err != nil {
			break
		} else {
			if !bytes.HasPrefix(k, []byte(prefixExpireHeight)) {
				break
			}
			key := k[len(prefixExpireHeight):]
			hash := common.BytesToHash(v)
			expireHeight := common.ByteToUInt64(key)
			if expireHeight > height {
				break
			}

			store.db.Extract(nil, hashKey(hash))
			store.db.Extract(nil, k)
			logger.Debugf("delete sk info: %v %v", expireHeight, hash)
		}
	}
}

func (store *skStorage) storeSeckey(hash common.Hash, msk *groupsig.Seckey, encSk *groupsig.Seckey, expireHeight uint64) {
	store.mu.Lock()
	defer store.mu.Unlock()

	v := store.getSkInfo(hash)
	if v == nil {
		v = &skInfo{}
	}
	if msk != nil {
		v.msk = *msk
	}
	if encSk != nil {
		v.encSk = *encSk
	}
	encryptedBytes, err := common.EncryptWithKey(store.encKey, v.toBytes())
	if err != nil {
		logger.Errorf("encrypt sk error %v, encSk %v", err, common.ToHex(store.encKey))
		return
	}
	err = store.db.BeginTransaction()
	if err != nil {
		store.db.Rollback()
		logger.Errorf("begin transaction error %v", err)
		return
	}
	defer store.db.Commit()

	err = store.db.Set(hashKey(hash), encryptedBytes)
	if err != nil {
		logger.Errorf("store sk info error:%v", err)
		return
	}
	err = store.db.Set(heightKey(expireHeight), hash.Bytes())
	if err != nil {
		logger.Errorf("store sk height error:%v", err)
		return
	}
	logger.Debugf("store seckey %v %v", hash, expireHeight)
}

func (store *skStorage) getSkInfo(hash common.Hash) *skInfo {
	bs, err := store.db.Get(nil, hashKey(hash))
	if err != nil {
		logger.Errorf("get skInfo error:%v %v", hash, err)
		return nil
	}
	if bs == nil {
		return nil
	}
	decyptedBytes, err := common.DecryptWithKey(store.encKey, bs)
	if err != nil {
		logger.Errorf("decrypt sk info error:%v", err)
		return nil
	}
	return decodeSkInfoBytes(decyptedBytes)
}

func (store *skStorage) GetGroupSignatureSeckey(seed common.Hash) groupsig.Seckey {
	sk := store.getSkInfo(seed)
	if sk != nil {
		return sk.msk
	}
	return groupsig.Seckey{}
}

func (store *skStorage) StoreGroupSignatureSeckey(seed common.Hash, sk groupsig.Seckey, expireHeight uint64) {
	store.storeSeckey(seed, &sk, nil, expireHeight)
}

func (store *skStorage) Close() error {
	logger.Debugf("closing db file %v", store.db.Name())
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.db.Close()
}
