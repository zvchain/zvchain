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
func (store *smallStateStore) DeleteSmallDbData(data map[uint64]common.Hash) error {
	batch := store.db.NewBatch()
	for k, v := range data {
		err := batch.Delete(store.generateKey(v[:], k, smallDbRootData))
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

// DeleteSmallDbDataByKey delete data when cold start after merge data from small db
func (store *smallStateStore) DeleteSmallDbDataByKey(keys [][]byte) error {
	batch := store.db.NewBatch()
	for _, v := range keys {
		err := batch.Delete(v)
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

// store current root data and height  to small db
func (store *smallStateStore) StoreDataToSmallDb(height uint64, root common.Hash, nb []byte) error {
	err := store.db.Put(store.generateKey(root[:], height, smallDbRootData), nb)
	if err != nil {
		return fmt.Errorf("store state data to small db error %v", err)
	}
	return nil
}

func (store *smallStateStore) GetBatch() tasdb.Batch {
	return store.db.NewBatch()
}

func (store *smallStateStore) GetIterator() iterator.Iterator {
	return store.db.NewIterator()
}

func (store *smallStateStore) GetHeight(key []byte) uint64 {
	return common.ByteToUInt64(key[2:10])
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
func (store *smallStateStore) generateKey(raw []byte, height uint64, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	bytesBuffer.Write(common.Uint64ToByte(height))
	if raw != nil {
		bytesBuffer.Write(raw)
	}
	return bytesBuffer.Bytes()
}

func (store *smallStateStore) Close() {
	if store.db != nil {
		store.db.Close()
	}
}
