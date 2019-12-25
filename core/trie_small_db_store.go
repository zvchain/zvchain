package core

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"sync"
)

var (
	persistentHeight = "ph"
	smallDbRootDatas = "dt"
	lastDeleteHeight = "ldt"
)

type smallStateStore struct {
	db        tasdb.Database
	mu        sync.Mutex // Mutex lock
	hasStored bool       // first store state data will set true
}

func initSmallStore(db tasdb.Database) *smallStateStore {
	return &smallStateStore{
		db: db,
	}
}

func (store *smallStateStore) GetLastDeleteHeight() uint64 {
	data, _ := store.db.Get([]byte(lastDeleteHeight))
	return common.ByteToUInt64(data)
}

// GetSmallDbDataByRoot will get the data by root key from small db
func (store *smallStateStore) GetSmallDbDataByRoot(root common.Hash) []byte {
	data, _ := store.db.Get(store.generateKey(root[:], smallDbRootDatas))
	return data
}

// DeleteSmallDbDataByRootWithoutStoreHeight only delete root key from small db
func (store *smallStateStore) DeleteSmallDbDataByRootWithoutStoreHeight(root common.Hash) error {
	err := store.db.Delete(store.generateKey(root[:], smallDbRootDatas))
	if err != nil {
		return fmt.Errorf("delete dirty trie error %v", err)
	}
	return nil
}

// DeleteSmallDbDataByRoot will delete root key and then store the current height to small db
func (store *smallStateStore) DeleteSmallDbDataByRoot(root common.Hash, height uint64) error {
	err := store.db.Delete(store.generateKey(root[:], smallDbRootDatas))
	if err != nil {
		return fmt.Errorf("delete state data from small db error %v", err)
	}
	err = store.db.Put([]byte(lastDeleteHeight), common.UInt64ToByte(height))
	if err != nil {
		return fmt.Errorf("store last delete height error %v,height is %v", err, height)
	}
	return nil
}

// store current root data and height  to small db
func (store *smallStateStore) StoreDataToSmallDb(height uint64, root common.Hash, nb []byte) error {
	// if small db data is empty,delete height reset to current height,because from no prune mode to prune mode will scale too much blocks
	if !store.hasStored && !store.HasStateData() {
		err := store.db.Put([]byte(lastDeleteHeight), common.UInt64ToByte(height))
		if err != nil {
			return fmt.Errorf("store last delete height error %v,height is %v", err, height)
		}
	}
	err := store.db.Put(store.generateKey(root[:], smallDbRootDatas), nb)
	if err != nil {
		return fmt.Errorf("store state data to small db error %v", err)
	}
	store.hasStored = true
	return nil
}

// StoreStatePersistentHeight store the persistent height to small db
// This height is used for cold start
func (store *smallStateStore) StoreStatePersistentHeight(height uint64) error {
	err := store.db.Put([]byte(persistentHeight), common.UInt64ToByte(height))
	if err != nil {
		return fmt.Errorf("store trie pure copy info error %v", err)
	}
	return nil
}

// HasStateData check the small db exists data
func (store *smallStateStore) HasStateData() bool {
	iter := store.db.NewIterator()
	defer iter.Release()
	return iter.Seek([]byte(smallDbRootDatas))
}

func (store *smallStateStore) GetStatePersistentHeight() uint64 {
	data, _ := store.db.Get([]byte(persistentHeight))
	return common.ByteToUInt64(data)
}

// generateKey generate a prefixed key
func (store *smallStateStore) generateKey(raw []byte, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
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
