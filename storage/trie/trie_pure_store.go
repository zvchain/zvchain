package trie

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
)

var (
	trieHeightStore = "trie_height_store"
	trieHeight      = "trie_height"
	TrieStore       *TrieStorage
)

type TrieStorage struct {
	file string
	db   *bolt.DB
}

func NewTrieStorage(file string) {
	db, err := bolt.Open(file, 0666, nil)
	if err != nil {
		panic(fmt.Errorf("create trie copy db fail:%v in %v", err, file))
	}
	TrieStore = &TrieStorage{
		file: file,
		db:   db,
	}
}

func (store *TrieStorage) StoreTriePureHeight(height uint64) {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(trieHeight))
		if e != nil {
			return e
		}
		return b.Put([]byte(trieHeightStore), common.UInt64ToByte(height))
	})
	if err != nil {
		log.CoreLogger.Errorf("store trie pure copy info error %v", err)
		return
	}
}

func (store *TrieStorage) GetLastTrieHeight() uint64 {
	var height uint64 = 0
	store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte([]byte(trieHeight)))
		if b == nil {
			height = 0
		} else {
			v := b.Get([]byte(trieHeightStore))
			height = common.ByteToUInt64(v)
		}
		return nil
	})
	return height
}

func (store *TrieStorage) Close() error {
	log.CorpLogger.Debugf("closing trie copy db file %v", store.db.Path())

	return store.db.Close()
}
