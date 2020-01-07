package core

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"sync"
)

var (
	smallDbRootData = "dt" // this key used be store root data to small db
)

type smallStateStore struct {
	db tasdb.Database
	mu sync.Mutex // Mutex lock
}

func initSmallStore(db tasdb.Database) *smallStateStore {
	return &smallStateStore{
		db: db,
	}
}

// DeleteSmallDbData delete from small db if reset top
func (store *smallStateStore) DeleteSmallDbData(data []uint64) error {
	batch := store.db.NewBatch()
	for _, height := range data {
		err := batch.Delete(store.generateKey(height, smallDbRootData))
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

func (store *smallStateStore) GetDeleteKeysByHeight(height uint64) ([][]byte, uint64) {
	var startHeight uint64
	iter := store.db.NewIterator()
	deleteKeys := [][]byte{}
	for iter.Next() {
		delHeight := store.GetHeight(iter.Key())
		// if db height >= persistenceHeight then break,key is dt(2 bytes)+height(8 bytes)
		if delHeight >= height {
			break
		}
		if startHeight == 0 {
			startHeight = delHeight
		}
		tmp := make([]byte, len(iter.Key()))
		copy(tmp, iter.Key())
		deleteKeys = append(deleteKeys, tmp)
	}
	return deleteKeys, startHeight
}

func (store *smallStateStore) GetIterator() iterator.Iterator {
	return store.db.NewIterator()
}

// DeleteSmallDbDataByKey delete data when cold start after merge data from small db
func (store *smallStateStore) DeleteSmallDbDataByKey(deleteKeys [][]byte) error {
	batch := store.db.NewBatch()
	for _, k := range deleteKeys {
		err := batch.Delete(k)
		if err != nil {
			err = fmt.Errorf("delete small db failed,error is %v", err)
			return err
		}
		if batch.ValueSize() > tasdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("delete small db failed,error is %v", err)
			}
			batch.Reset()
		}
	}
	err := batch.Write()
	if err != nil {
		err = fmt.Errorf("delete small db failed,error is %v", err)
		return err
	}
	return nil
}

// store current root data and height  to small db
func (store *smallStateStore) StoreDataToSmallDb(height uint64, nb []byte) error {
	err := store.db.Put(store.generateKey(height, smallDbRootData), nb)
	if err != nil {
		return fmt.Errorf("store state data to small db error %v", err)
	}
	return nil
}

func (store *smallStateStore) GetHeight(key []byte) uint64 {
	return common.ByteToUInt64(key[2:])
}

func (store *smallStateStore) GetStatePersistentHeight() uint64 {
	iter := store.db.NewIterator()
	defer iter.Release()
	hasValue := iter.Seek([]byte(smallDbRootData))
	if !hasValue {
		return 0
	}
	return store.GetHeight(iter.Key())
}

// generateKey generate a prefixed key
func (store *smallStateStore) generateKey(height uint64, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	bytesBuffer.Write(common.Uint64ToByte(height))
	return bytesBuffer.Bytes()
}

func (store *smallStateStore) Close() {
	if store.db != nil {
		store.db.Close()
	}
}
