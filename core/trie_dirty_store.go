package core

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
)

var (
	trieHeightStore     = "trie_height_store"
	dirtyTrie           = "dt"
	lastDirtyTrieHeight = "ldt"
)

var dirtyState *dirtyStateStore

type dirtyStateStore struct {
	db tasdb.Database
}

func initDirtyStore(db tasdb.Database) {
	dirtyState = &dirtyStateStore{
		db: db,
	}
}

func (store *dirtyStateStore) GetLastDeleteDirtyTrieHeight() uint64 {
	data, _ := store.db.Get([]byte(lastDirtyTrieHeight))
	return common.ByteToUInt64(data)
}

func (store *dirtyStateStore) GetDirtyByRoot(root common.Hash) []byte {
	data, _ := store.db.Get(store.generateKey(root[:], dirtyTrie))
	return data
}

func (store *dirtyStateStore) DeleteDirtyRoot(root common.Hash) error {
	err := store.db.Delete(store.generateKey(root[:], dirtyTrie))
	if err != nil {
		return fmt.Errorf("delete dirty trie error %v", err)
	}
	return nil
}

func (store *dirtyStateStore) DeleteDirtyTrie(root common.Hash, height uint64) error {
	err := store.db.Delete(store.generateKey(root[:], dirtyTrie))
	if err != nil {
		return fmt.Errorf("delete dirty trie error %v", err)
	}
	err = store.db.Put([]byte(lastDirtyTrieHeight), common.UInt64ToByte(height))
	if err != nil {
		return fmt.Errorf("delete store diry trie error %v", err)
	}
	return nil
}

func (store *dirtyStateStore) StoreDirtyTrie(root common.Hash, nb []byte) error {
	err := store.db.Put(store.generateKey(root[:], dirtyTrie), nb)
	if err != nil {
		return fmt.Errorf("store diry trie error %v", err)
	}
	return nil
}

func (store *dirtyStateStore) StoreStatePersistentHeight(height uint64) error {
	err := store.db.Put([]byte(trieHeightStore), common.UInt64ToByte(height))
	if err != nil {
		return fmt.Errorf("store trie pure copy info error %v", err)
	}
	return nil
}

func (store *dirtyStateStore) GetStatePersistentHeight() uint64 {
	data, _ := store.db.Get([]byte(trieHeightStore))
	return common.ByteToUInt64(data)
}

// generateKey generate a prefixed key
func (store *dirtyStateStore) generateKey(raw []byte, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	if raw != nil {
		bytesBuffer.Write(raw)
	}
	return bytesBuffer.Bytes()
}
