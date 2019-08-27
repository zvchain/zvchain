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
	"github.com/boltdb/bolt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
)

const (
	skEncVersion = 1
)

var (
	bucketHash         = "hash"
	bucketExpireHeight = "expire"
)

type skInfo struct {
	msk   groupsig.Seckey
	encSk groupsig.Seckey
}

func (si *skInfo) toBytes() []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(skEncVersion)
	buf.Write(si.msk.Serialize())
	buf.Write(si.encSk.Serialize())
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

type skStorage struct {
	file   string
	encKey []byte
	db     *bolt.DB

	blockAddCh chan uint64
}

func newSkStorage(file string, encKey []byte) *skStorage {
	db, err := bolt.Open(file, 0666, nil)
	if err != nil {
		logger.Error(fmt.Errorf("create db fail:%v in %v", err, file))
		if db == nil {
			panic(fmt.Errorf("create db fail:%v in %v", err, file))
		}
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

func (store *skStorage) removeExpires(height uint64) {
	delHash := make([]common.Hash, 0)
	delHeight := make([]uint64, 0)
	err := store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExpireHeight))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		min := common.Uint64ToByte(0)
		max := common.Uint64ToByte(height)
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			hash := common.BytesToHash(v)
			expireHeight := common.ByteToUInt64(k)
			if expireHeight > height {
				break
			}
			delHash = append(delHash, hash)
			delHeight = append(delHeight, expireHeight)
		}
		return nil
	})
	if err != nil {
		logger.Errorf("remove expire error:%v", err)
		return
	}
	if len(delHeight) > 0 {
		err = store.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucketExpireHeight))
			if b == nil {
				return nil
			}
			for _, h := range delHeight {
				e := b.Delete(common.Uint64ToByte(h))
				if e != nil {
					return e
				}
				logger.Debugf("remove expire height %v", h)
			}
			return nil
		})
		if err != nil {
			logger.Errorf("remove expire height error:%v", err)
		}
	}
	if len(delHash) > 0 {
		err = store.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucketHash))
			if b == nil {
				return nil
			}
			for _, h := range delHash {
				e := b.Delete(h.Bytes())
				if e != nil {
					return e
				}
				logger.Debugf("remove expire hash %v", h)
			}
			return nil
		})
		if err != nil {
			logger.Errorf("remove expire hash error:%v", err)
		}
	}
}

func (store *skStorage) storeSeckey(hash common.Hash, msk *groupsig.Seckey, encSk *groupsig.Seckey, expireHeight uint64) {

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
	err = store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(bucketHash))
		if e != nil {
			return e
		}
		return b.Put(hash.Bytes(), encryptedBytes)
	})
	if err != nil {
		logger.Errorf("store sk info error %v", err)
		return
	}
	err = store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(bucketExpireHeight))
		if e != nil {
			return e
		}
		return b.Put(common.Uint64ToByte(expireHeight), hash.Bytes())
	})
	if err != nil {
		logger.Errorf("store sk height error:%v", err)
		return
	}
	logger.Debugf("store seckey %v %v", hash, expireHeight)
}

func (store *skStorage) getSkInfo(hash common.Hash) *skInfo {
	var ski *skInfo
	err := store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketHash))
		if b == nil {
			return nil
		}
		bs := b.Get(hash.Bytes())
		if bs == nil {
			return nil
		}
		decyptedBytes, e := common.DecryptWithKey(store.encKey, bs)
		if e != nil {
			return e
		}
		ski = decodeSkInfoBytes(decyptedBytes)
		return nil
	})
	if err != nil {
		logger.Errorf("decrypt sk info error:%v", err)
		return nil
	}
	return ski
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
	logger.Debugf("closing db file %v", store.db.Path())

	return store.db.Close()
}
