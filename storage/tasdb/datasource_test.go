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
package tasdb

import (
	"testing"
)

func TestCreateLDB(t *testing.T) {
	// 创建ldb实例
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	if err != nil {
		t.Fatal(err)
		return
	}
	defer func() {
		if ldb != nil {
			ldb.Close()
		}
	}()

	// 测试put
	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		t.Fatal(err)
	}

	// 测试get
	result, err := ldb.Get([]byte("testkey"))
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Errorf("get key : testkey, value: %s \n", result)
	}

	// 测试has
	exist, err := ldb.Has([]byte("testkey"))
	if err != nil {
		t.Fatal(err)

	}
	if !exist {
		t.Errorf("get key : %s\n", "testkey")
	}

	// 测试delete
	err = ldb.Delete([]byte("testkey"))
	if err != nil {
		t.Fatal(err)

	}

	// 测试get空
	// key不存在，会返回err
	result, _ = ldb.Get([]byte("testkey"))

	if result != nil {
		t.Errorf("get key : testkey, value: %s \n", result)
	}

}

func TestLRUMemDatabase(t *testing.T) {
	mem, _ := NewLRUMemDatabase(10)
	for i := (byte)(0); i < 11; i++ {
		mem.Put([]byte{i}, []byte{i})
	}
	data, _ := mem.Get([]byte{0})
	if data != nil {
		t.Errorf("expected value nil")
	}
	data, _ = mem.Get([]byte{10})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	data, _ = mem.Get([]byte{5})
	if data == nil {
		t.Errorf("expected value not nil")
	}
	mem.Delete([]byte{5})
	data, _ = mem.Get([]byte{5})
	if data != nil {
		t.Errorf("expected value nil")
	}
}

func TestClearLDB(t *testing.T) {
	// 创建ldb实例
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	// 测试put
	err = ldb.Put([]byte("testkey"), []byte("testvalue"))
	if err != nil {
		t.Fatalf("failed to put key in testldb\n")
	}

	if err != nil {
		t.Fatalf("error to clear ldb : %s\n", "testldb")
		return
	}
}

func TestBatchPutVisiableBeforeWrite(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	key := []byte("test")
	batch := ldb.CreateLDBBatch()
	_, err = ldb.Get(key)

	ldb.AddKv(batch, key, []byte("i am handsome"))
	_, err = ldb.Get(key)

	err = batch.Write()
	if err != nil {
		t.Fatal("write fail", err)
	}
	_, err = ldb.Get(key)

	ldb.AddKv(batch, key, nil)
	err = batch.Write()
	if err != nil {
		t.Fatal("write fail", err)
	}
}

func TestIteratorWithPrefix(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
	}

}

func TestIteratorWithPrefix2(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
		return
	}

	_, err = ds.NewPrefixDatabase("testldb")
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")
	}
}

func TestGetAfter(t *testing.T) {
	ds, err := NewDataSource("test")
	if err != nil {
		t.Fatal(err)
	}
	ldb, err := ds.NewPrefixDatabase("testldb")
	defer ldb.Close()
	if err != nil {
		t.Fatalf("error to create ldb : %s\n", "testldb")

	}
}
