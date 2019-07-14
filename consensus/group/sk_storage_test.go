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
	"github.com/boltdb/bolt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/taslog"
	"math/big"
	"testing"
	"time"
)

func TestStoreSeckey(t *testing.T) {
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	logger = taslog.GetLoggerByName("testlog")
	go skStore.loop()

	defer skStore.Close()

	msk := groupsig.DeserializeSeckey(common.FromHex("0x64b59f9ff74d2143a70d7e3c18edaef5750974bc08e5e34b3c57f8b95ea2a8"))
	encSk := groupsig.DeserializeSeckey(common.FromHex("0x64b59f9ff74d2143a70d7e3c18edaef5750974bc08e5e34b3c57f8b95ea2a8"))
	t.Log(msk.GetHexString())

	hash := common.HexToHash("0x123")
	skStore.storeSeckey(hash, msk, nil, 1020)
	skStore.storeSeckey(hash, nil, encSk, 1030)

	ski := skStore.getSkInfo(hash)
	t.Log(ski.msk.GetHexString())
	if !ski.msk.IsEqual(*msk) {
		t.Errorf("msk not equal")
	}
	if !ski.encSk.IsEqual(*encSk) {
		t.Errorf("encsk error")
	}

	skiNotExist := skStore.getSkInfo(common.HexToHash("0x1"))
	t.Log(skiNotExist)
}

func TestStore(t *testing.T) {
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	logger = taslog.GetLoggerByName("testlog")
	go skStore.loop()

	defer skStore.Close()

	msk := groupsig.DeserializeSeckey(common.FromHex("0x456000000000000000000000000000000000000000000000000000000000789"))
	encSk := groupsig.DeserializeSeckey(common.FromHex("0x234000000000000000000000000000000000000000000000000000000000aaa"))

	for i := 0; i < 100; i++ {
		hash := common.BigToHash(new(big.Int).SetUint64(uint64(i)))
		skStore.storeSeckey(hash, msk, nil, 0)
		skStore.storeSeckey(hash, nil, encSk, 100*uint64(i))
	}

}

func TestGetSkInfo(t *testing.T) {
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	logger = taslog.GetLoggerByName("testlog")
	go skStore.loop()

	defer skStore.Close()
	for i := 0; i < 100; i++ {
		hash := common.BigToHash(new(big.Int).SetUint64(uint64(i)))
		ski := skStore.getSkInfo(hash)
		if ski == nil {
			continue
		}
		t.Log(ski.msk.GetHexString(), ski.encSk.GetHexString())
	}

	err := skStore.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExpireHeight))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		min := common.Uint64ToByte(0)
		max := common.Uint64ToByte(10000)
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			hash := common.BytesToHash(v)
			expireHeight := common.ByteToUInt64(k)
			t.Log(expireHeight, hash)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

}

func TestRemoveExpire(t *testing.T) {
	logger = taslog.GetLoggerByName("testlog")
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	go skStore.loop()

	defer skStore.Close()

	skStore.blockAddCh <- 7666

	time.Sleep(2 * time.Second)
}
