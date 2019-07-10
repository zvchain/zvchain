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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"sync"
)

const (
	maxItems        = 20000 // Maximum seckeys stored in the skStorage. It only need to keep the lived-group-msk which the number is far enough
	expireHeightGap = 2 * lifeWindow
)

type skInfo struct {
	msk          groupsig.Seckey
	encSk        groupsig.Seckey
	expireHeight uint64
}

type skStorage struct {
	file    string
	encKey  []byte
	skInfos map[common.Hash]*skInfo

	flushCh    chan struct{}
	blockAddCh chan uint64
	lock       sync.RWMutex
}

func newSkStorage(file string, encKey []byte) *skStorage {
	return &skStorage{
		file:       file,
		encKey:     encKey,
		skInfos:    make(map[common.Hash]*skInfo),
		flushCh:    make(chan struct{}, 5),
		blockAddCh: make(chan uint64, 5),
	}
}

func (store *skStorage) loop() {
	for {
		select {
		case <-store.flushCh:
			store.flush()
		case height := <-store.blockAddCh:
			store.removeExpires(height)
		}
	}
}

func (store *skStorage) flush() {

}

func (store *skStorage) removeExpires(height uint64) {

}

func (store *skStorage) initAndLoad() error {
	return nil
}

func (store *skStorage) encryptSeckey(sk groupsig.Seckey) ([]byte, error) {
	return common.EncryptWithKey(store.encKey, sk.Serialize())
}
func (store *skStorage) decryptSeckey(bs []byte) (groupsig.Seckey, error) {
	data, err := common.DecryptWithKey(store.encKey, bs)
	if err != nil {
		return groupsig.Seckey{}, err
	}
	return *groupsig.DeserializeSeckey(data), nil
}

func (store *skStorage) storeSeckey(hash common.Hash, msk *groupsig.Seckey, encSk *groupsig.Seckey, expireHeight uint64) {
	store.lock.Lock()
	defer store.lock.Unlock()
	if v, ok := store.skInfos[hash]; ok {
		if msk != nil {
			v.msk = *msk
		}
		if encSk != nil {
			v.encSk = *encSk
		}
		v.expireHeight = expireHeight
	} else {
		v = &skInfo{
			expireHeight: expireHeight,
		}
		if msk != nil {
			v.msk = *msk
		}
		if encSk != nil {
			v.encSk = *encSk
		}
		store.skInfos[hash] = v
	}
	store.flushCh <- struct{}{}
}

func (store *skStorage) getSkInfo(hash common.Hash) *skInfo {
	store.lock.RLock()
	defer store.lock.RUnlock()
	if v, ok := store.skInfos[hash]; ok {
		return v
	}
	return nil
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
