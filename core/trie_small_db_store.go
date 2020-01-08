package core

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"sync"
)

var (
	smallDbRootData = []byte("dt") // this key used be store root data to small db
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

// iterateData iterates data with given func, and the key parameter doesn't contains common prefix(dt)
// it's removed by the prefixIter instance
func (store *smallStateStore) iterateData(iterFunc func(key, value []byte) (bool, error)) error {
	iter := store.db.NewIteratorWithPrefix(smallDbRootData)
	defer iter.Release()

	for iter.Next() {
		if ok, err := iterFunc(iter.Key(), iter.Value()); !ok {
			return err
		}
	}
	return nil
}

// DeleteHeights delete from small db if reset top
func (store *smallStateStore) DeleteHeights(heights []uint64) error {
	batch := store.db.NewBatch()
	for _, height := range heights {
		err := batch.Delete(store.generateDataKey(common.Uint64ToByte(height)))
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

// DeletePreviousOf iterates and deletes data from beginning to the given height
func (store *smallStateStore) DeletePreviousOf(height uint64) (uint64, error) {
	batch := store.db.NewBatch()
	beginHeight := uint64(0)
	err := store.iterateData(func(key, value []byte) (bool, error) {
		delHeight := store.parseHeightOfPrefixIterKey(key)
		if delHeight > height {
			return false, nil
		}
		if beginHeight == 0 {
			beginHeight = delHeight
		}
		if err := batch.Delete(store.generateDataKey(key)); err != nil {
			return false, fmt.Errorf("delete error at %v, err %v", delHeight, err)
		}
		if batch.ValueSize() >= tasdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return false, err
			}
			batch.Reset()
		}
		return true, nil
	})
	if err != nil {
		return beginHeight, err
	}
	if err := batch.Write(); err != nil {
		return beginHeight, err
	}
	return beginHeight, nil
}

// store current root data and height  to small db
func (store *smallStateStore) StoreDataToSmallDb(height uint64, nb []byte) error {
	err := store.db.Put(store.generateDataKey(common.Uint64ToByte(height)), nb)
	if err != nil {
		return fmt.Errorf("store state data to small db error %v", err)
	}
	return nil
}

// parseHeightOfPrefixIterKey parses height in the given iter key which doesn't contains prefix
func (store *smallStateStore) parseHeightOfPrefixIterKey(key []byte) uint64 {
	return common.ByteToUInt64(key)
}

// generateDataKey generate a prefixed key
func (store *smallStateStore) generateDataKey(heightBytes []byte) []byte {
	bytesBuffer := bytes.NewBuffer(smallDbRootData)
	bytesBuffer.Write(heightBytes)
	return bytesBuffer.Bytes()
}

func (store *smallStateStore) Close() {
	if store.db != nil {
		store.db.Close()
	}
}
