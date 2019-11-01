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
	"strconv"
	"testing"
	"time"

	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/tasdb"
)

func TestTxBlackList(t *testing.T) {
	txs := make([]*types.Transaction, 0)
	for i := 1; i < 5; i++ {
		tx := genTx4Test("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff"+strconv.Itoa(i), uint64(i), types.NewBigInt(20000), gasLimit, &addr1)
		txs = append(txs, tx)
	}
	addTx(t, txs[0])
	addTx(t, txs[1])
	addTx(t, txs[2])

	cacheDs, err := tasdb.NewDataSource("adb_blacklist", nil)
	if err != nil {
		t.Error("failed to init db", err)
	}
	cacheDB, err := cacheDs.NewPrefixDatabase("")
	tb := newTxBlackList(cacheDB, time.Hour)
	if !tb.has(txs[0].Hash) {
		t.Error("should has this has but not")
	}
	if !tb.has(txs[1].Hash) {
		t.Error("should has this has but not")
	}
	if !tb.has(txs[2].Hash) {
		t.Error("should has this has but not")
	}
	if tb.has(txs[3].Hash) {
		t.Error("should not has this but got has")
	}
	cacheDB.Close()

	//test timeout
	time.Sleep(time.Second * 3)
	cacheDs, err = tasdb.NewDataSource("adb_blacklist", nil)
	if err != nil {
		t.Error("failed to init db", err)
	}
	cacheDB, err = cacheDs.NewPrefixDatabase("")
	tb = newTxBlackList(cacheDB, time.Second)
	if tb.has(txs[0].Hash) {
		t.Error("should be removed after timeout")
	}
	cacheDB.Close()
}

func addTx(t *testing.T, tx *types.Transaction) {
	cacheDs, err := tasdb.NewDataSource("adb_blacklist", nil)
	if err != nil {
		t.Error("failed to init db", err)
	}
	cacheDB, err := cacheDs.NewPrefixDatabase("")
	tb := newTxBlackList(cacheDB, time.Hour)
	tb.setTop(tx.Hash)
	cacheDB.Close()
}
