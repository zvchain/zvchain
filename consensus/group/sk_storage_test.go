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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/taslog"
	"math/big"
	"testing"
)

func TestOpenDB(t *testing.T) {
	db, err := createOrOpenDB("testdb")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Set([]byte("123"), []byte("456"))
	if err != nil {
		t.Fatal(err)
	}

	v, err := db.Get(nil, []byte("123"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(v)
	db.Close()
}

func TestStoreSeckey(t *testing.T) {
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	logger = taslog.GetLoggerByName("testlog")
	defer skStore.Close()

	msk := groupsig.DeserializeSeckey(common.FromHex("0x456000000000000000000000000000000000000000000000000000000000789"))
	encSk := groupsig.DeserializeSeckey(common.FromHex("0x234000000000000000000000000000000000000000000000000000000000aaa"))
	t.Log(msk.GetHexString())

	hash := common.HexToHash("0x123")
	skStore.storeSeckey(hash, msk, nil, 0)
	skStore.storeSeckey(hash, nil, encSk, 0)

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

func TestSkStorage_Store(t *testing.T) {
	logger = taslog.GetLoggerByName("testlog")
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	go skStore.loop()

	defer skStore.Close()

	//msk := groupsig.DeserializeSeckey(common.FromHex("0x456000000000000000000000000000000000000000000000000000000000789"))
	//encSk := groupsig.DeserializeSeckey(common.FromHex("0x234000000000000000000000000000000000000000000000000000000000aaa"))
	//
	//for i:=0; i < 100; i++ {
	//	hash := common.BigToHash(new(big.Int).SetUint64(uint64(i)))
	//	skStore.storeSeckey(hash, msk, nil, uint64(100)*uint64(i))
	//	skStore.storeSeckey(hash, nil, encSk, uint64(100)*uint64(i))
	//}

	iter, err := skStore.db.SeekFirst()
	if err != nil {
		t.Error(err)
		return
	}
	for {
		if k, v, err := iter.Next(); err != nil {
			break
		} else {
			if bytes.HasPrefix(k, []byte(prefixHash)) {
				key := k[len(prefixHash):]
				decypted, err := common.DecryptWithKey(skStore.encKey, v)
				if err != nil {
					t.Error("decryption error ", err)
				}
				ski := decodeSkInfoBytes(decypted)
				t.Log("hash", common.BytesToHash(key), ski.msk.GetHexString(), ski.encSk.GetHexString())
			} else if bytes.HasPrefix(k, []byte(prefixExpireHeight)) {
				key := k[len(prefixExpireHeight):]
				hash := common.BytesToHash(v)
				t.Log("expire", common.ByteToUInt64(key), hash)
			}
		}
	}

}

func TestSeek(t *testing.T) {
	logger = taslog.GetLoggerByName("testlog")
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	go skStore.loop()

	defer skStore.Close()

	msk := groupsig.DeserializeSeckey(common.FromHex("0x456000000000000000000000000000000000000000000000000000000000789"))
	encSk := groupsig.DeserializeSeckey(common.FromHex("0x234000000000000000000000000000000000000000000000000000000000aaa"))

	for i := 0; i < 100; i++ {
		hash := common.BigToHash(new(big.Int).SetUint64(uint64(i)))
		skStore.storeSeckey(hash, msk, nil, uint64(100)*uint64(i))
		skStore.storeSeckey(hash, nil, encSk, uint64(100)*uint64(i))
		t.Log(uint64(100) * uint64(i))
	}

	prefix := []byte(prefixExpireHeight)
	iter, hit, err := skStore.db.Seek(append(prefix, common.Uint64ToByte(500)...))
	t.Log(hit)
	if err != nil {
		t.Error(err)
	}
	for {
		if k, v, err := iter.Prev(); err != nil {
			break
		} else {
			if !bytes.HasPrefix(k, prefix) {
				break
			}
			key := k[len(prefixExpireHeight):]
			hash := common.BytesToHash(v)
			t.Log("prev", common.ByteToUInt64(key), hash)
		}
	}
	iter, hit, err = skStore.db.Seek(append(prefix, common.Uint64ToByte(1200)...))
	for {
		if k, v, err := iter.Next(); err != nil {
			break
		} else {
			if !bytes.HasPrefix(k, prefix) {
				break
			}
			key := k[len(prefixExpireHeight):]
			hash := common.BytesToHash(v)
			t.Log("next", common.ByteToUInt64(key), hash)
		}
	}

}

func TestRemoveExpire(t *testing.T) {
	logger = taslog.GetLoggerByName("testlog")
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	go skStore.loop()

	defer skStore.Close()

	skStore.blockAddCh <- 300

	skStore.blockAddCh <- 301
	skStore.blockAddCh <- 302
	skStore.blockAddCh <- 750
	skStore.blockAddCh <- 100000

}

func TestWALName(t *testing.T) {
	logger = taslog.GetLoggerByName("testlog")
	skStore := newSkStorage("testdb", common.FromHex("0xb1aef01c1fa63ed58b2b845ddd77dc1a9a94cb7358664cb7210c7296c0d13361"))
	defer skStore.Close()

	t.Log(skStore.db.WALName())
}
