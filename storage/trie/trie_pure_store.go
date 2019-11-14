package trie

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
)

var (
	trieHeightStore = "trie_height_store"
	curHeightStore  = "cur_height_store"
	pureStore       = "pure_store"
	dirtyTrie       = "dt"
	lastDirtyTrieHeight   = "ldt"
	TrieStore       *TrieStorage
	GcMode          = true
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

func (store *TrieStorage) StoreCurHeight(height uint64) {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(pureStore))
		if e != nil {
			return e
		}
		return b.Put([]byte(curHeightStore), common.UInt64ToByte(height))
	})
	if err != nil {
		log.CoreLogger.Errorf("store current height info error %v", err)
		return
	}
}

func (store *TrieStorage) GetLastDeleteDirtyTrieHeight() uint64 {
	var height uint64 = 0
	store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte([]byte(pureStore)))
		if b == nil {
			height = 1
		} else {
			v := b.Get([]byte(lastDirtyTrieHeight))
			height = common.ByteToUInt64(v)
		}
		return nil
	})
	return height
}


func (store *TrieStorage) GetDirtyByRoot(root common.Hash) []byte {
	var bt []byte
	store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte([]byte(pureStore)))
		bt = b.Get(store.generateKey(root[:],dirtyTrie))
		return nil
	})
	return bt
}

func (store *TrieStorage) DeleteDirtyTrie(root common.Hash,height uint64) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(pureStore))
		if e != nil {
			return e
		}
		e = b.Delete(store.generateKey(root[:],dirtyTrie))
		if e != nil{
			return fmt.Errorf("delete dirty trie error %v",e)
		}
		e = b.Put([]byte(lastDirtyTrieHeight),common.UInt64ToByte(height))
		if  e != nil{
			return fmt.Errorf("put last dirty trie height error %v",e)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete store diry trie error %v", err)
	}
	return nil
}

func (store *TrieStorage) StoreDirtyTrie(root common.Hash, nb []byte) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(pureStore))
		if e != nil {
			return e
		}
		return b.Put(store.generateKey(root[:],dirtyTrie), nb)
	})
	if err != nil {
		return fmt.Errorf("store diry trie error %v", err)
	}
	return nil
}

func (store *TrieStorage) StoreTriePureHeight(height uint64) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte(pureStore))
		if e != nil {
			return e
		}
		return b.Put([]byte(trieHeightStore), common.UInt64ToByte(height))
	})
	if err != nil {
		return fmt.Errorf("store trie pure copy info error %v", err)
	}
	return nil
}

func (store *TrieStorage) GetCurrentHeight() uint64 {
	var height uint64 = 0
	store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte([]byte(pureStore)))
		if b == nil {
			height = 0
		} else {
			v := b.Get([]byte(curHeightStore))
			height = common.ByteToUInt64(v)
		}
		return nil
	})
	return height
}

func (store *TrieStorage) GetLastTrieHeight() uint64 {
	var height uint64 = 0
	store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte([]byte(pureStore)))
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

// generateKey generate a prefixed key
func (store *TrieStorage) generateKey(raw []byte, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	if raw != nil {
		bytesBuffer.Write(raw)
	}
	return bytesBuffer.Bytes()
}