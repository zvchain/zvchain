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

package core

import (
	"time"

	"github.com/vmihailenco/msgpack"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"gopkg.in/fatih/set.v0"
)

var (
	topKey  = []byte("topKey")
	listKey = []byte("listKey")
)

type timedHash struct {
	Hash  common.Hash
	Begin time.Time
}

type txBlackList struct {
	topHash   common.Hash
	blackList set.Interface
	db        tasdb.Database
	timeout   time.Duration
}

func newTxBlackList(db tasdb.Database, timeout time.Duration) *txBlackList {
	tbl := &txBlackList{
		blackList: set.New(set.ThreadSafe),
		db:        db,
		timeout:   timeout,
	}
	tbl.initFromDb()
	return tbl
}

func (t *txBlackList) initFromDb() {
	loadedList := t.loadList()

	filtered := make([]timedHash, 0)
	for _, item := range loadedList {
		if time.Since(item.Begin) < t.timeout {
			filtered = append(filtered, item)
		}
	}
	for _, v := range filtered {
		t.blackList.Add(v.Hash)
	}

	topHash := t.loadTop()
	if topHash != nil && !t.blackList.Has(*topHash) {
		t.blackList.Add(*topHash)
		tHash := timedHash{Hash: *topHash, Begin: time.Now()}
		filtered = append(filtered, tHash)
	}
	t.storeList(filtered)
	Logger.Debugf("all blacklist: %v, filtered: %v ",loadedList, filtered)
}

func (t *txBlackList) setTop(hash common.Hash) bool {
	err := t.db.Put(topKey, hash.Bytes())
	if err != nil {
		Logger.Errorf("failed to setTop: %v", err)
	}
	return true
}

func (t *txBlackList) removeTop(hash common.Hash) bool {
	_ = t.db.Delete(topKey)
	return true
}

func (t *txBlackList) has(hash common.Hash) bool {

	return t.blackList.Has(hash)
}

func (t *txBlackList) loadList() (rs []timedHash) {
	bs, err := t.db.Get(listKey)
	if err != nil {
		return
	}
	err = msgpack.Unmarshal(bs, &rs)
	if err != nil {
		Logger.Errorf("failed to unmarshal black list: %v", err)
	}
	return
}

func (t *txBlackList) loadTop() *common.Hash {
	top, err := t.db.Get(topKey)
	if err != nil {
		return nil
	}
	item := common.BytesToHash(top)
	return &item
}

func (t *txBlackList) storeList(items []timedHash) {
	bs, err := msgpack.Marshal(items)
	if err != nil {
		Logger.Errorf("failed to marshal tx black list: %v", err)
		return
	}
	err = t.db.Put(listKey, bs)
	if err != nil {
		Logger.Errorf("failed to save tx black list: %v", err)
		return
	}
	Logger.Debugf("store tx black list size: %v", len(items))
}

func (t *txBlackList) trace() {
	t.blackList.Each(func(i interface{}) bool {
		if Logger != nil {
			Logger.Debugf("black tx: %v", i)
		}
		return true
	})
}
