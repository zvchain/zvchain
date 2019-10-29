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

/*
	Package tasdb provides level db operations
*/
package tasdb

import (
	"bytes"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"

	lru "github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/zvchain/zvchain/common"
)

const (
	ConfigSec   = "chain"
	DefaultFile = "database"
)

var (
	ErrLDBInit = errors.New("LDB instance not inited")
)

type PrefixedDatabase struct {
	db     *LDBDatabase
	prefix string
}

type databaseConfig struct {
	database string
	cache    int
	handler  int
}

func getInstance(file string, options *opt.Options) (*LDBDatabase, error) {
	var (
		instanceInner *LDBDatabase
		err           error
	)

	defaultConfig := &databaseConfig{
		database: DefaultFile,
		cache:    128,
		handler:  1024,
	}

	if nil == common.GlobalConf {
		instanceInner, err = NewLDBDatabase(defaultConfig.database, options)
	} else {
		instanceInner, err = NewLDBDatabase(file, options)
	}

	return instanceInner, err
}

// Close close db connection
func (db *PrefixedDatabase) Close() {
	db.db.Close()
}

func (db *PrefixedDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(generateKey(key, db.prefix), value)
}

func (db *PrefixedDatabase) Get(key []byte) ([]byte, error) {
	return db.db.Get(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) Delete(key []byte) error {
	return db.db.Delete(generateKey(key, db.prefix))
}

func (db *PrefixedDatabase) newIterator(prefix []byte) iterator.Iterator {
	iter := db.db.NewIteratorWithPrefix([]byte(db.prefix))
	return &prefixIter{
		prefix: []byte(db.prefix),
		iter:   iter,
	}
}

func (db *PrefixedDatabase) NewIterator() iterator.Iterator {
	return db.NewIteratorWithPrefix(nil)
}

func (db *PrefixedDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	iterPrefix := generateKey(prefix, db.prefix)
	iter := db.db.NewIteratorWithPrefix(iterPrefix)
	return &prefixIter{
		prefix: iterPrefix,
		iter:   iter,
	}
}

func (db *PrefixedDatabase) NewBatch() Batch {
	return &prefixBatch{db: db.db.db, b: new(leveldb.Batch), prefix: db.prefix}
}

func (db *PrefixedDatabase) AddKv(batch Batch, k, v []byte) error {
	if v == nil {
		return db.addDeleteToBatch(batch, k)
	}
	return db.addKVToBatch(batch, k, v)
}
func (db *PrefixedDatabase) CreateLDBBatch() Batch {
	return db.db.NewBatch()
}

func (db *PrefixedDatabase) addKVToBatch(b Batch, k, v []byte) error {
	key := generateKey(k, db.prefix)
	return b.Put(key, v)
}

func (db *PrefixedDatabase) LogStats(logger *logrus.Logger) {
	s := &leveldb.DBStats{}
	err := db.db.db.Stats(s)
	if err != nil {
		logger.Error("failed to get leveldb stats", err)
	} else {
		logger.Debugf("leveldb stats: WriteDelayCount:%v,WriteDelayDuration:%v,WritePaused:%v,AliveSnapshots:%v,"+
			"AliveIterators:%v,IOWrite:%v,IORead:%v,BlockCacheSize:%v,OpenedTablesCount:%v,LevelSizes:%v,"+
			"LevelTablesCounts:%v,LevelRead:%v,LevelWrite:%v,LevelDurations:%v", s.WriteDelayCount,
			s.WriteDelayDuration, s.WritePaused, s.AliveSnapshots, s.AliveIterators, s.IOWrite, s.IORead, s.BlockCacheSize,
			s.OpenedTablesCount, s.LevelSizes, s.LevelTablesCounts, s.LevelRead, s.LevelWrite, s.LevelDurations)
	}
}

func (db *PrefixedDatabase) addDeleteToBatch(b Batch, k []byte) error {
	key := generateKey(k, db.prefix)
	return b.Delete(key)
}

type prefixIter struct {
	prefix []byte
	iter   iterator.Iterator
}

func (iter *prefixIter) First() bool {
	return iter.iter.First()
}

func (iter *prefixIter) Last() bool {
	return iter.iter.Last()
}

func (iter *prefixIter) Seek(key []byte) bool {
	buf := bytes.Buffer{}
	buf.Write(iter.prefix)
	buf.Write(key)
	return iter.iter.Seek(buf.Bytes())
}

func (iter *prefixIter) Next() bool {
	return iter.iter.Next()
}

func (iter *prefixIter) Prev() bool {
	return iter.iter.Prev()
}

func (iter *prefixIter) Release() {
	iter.iter.Release()
}

func (iter *prefixIter) SetReleaser(releaser util.Releaser) {
	iter.iter.SetReleaser(releaser)
}

func (iter *prefixIter) Valid() bool {
	return iter.iter.Valid()
}

func (iter *prefixIter) Error() error {
	return iter.iter.Error()
}

func (iter *prefixIter) Key() []byte {
	key := iter.iter.Key()
	return key[len(iter.prefix):]
}

func (iter *prefixIter) Value() []byte {
	return iter.iter.Value()
}

type prefixBatch struct {
	db     *leveldb.DB
	b      *leveldb.Batch
	size   int
	prefix string
}

func (b *prefixBatch) Delete(key []byte) error {
	b.b.Delete(generateKey(key, b.prefix))
	b.size++
	return nil
}

func (b *prefixBatch) Put(key, value []byte) error {
	b.b.Put(generateKey(key, b.prefix), value)
	b.size += len(value)
	return nil
}

func (b *prefixBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *prefixBatch) ValueSize() int {
	return b.size
}

func (b *prefixBatch) Reset() {
	b.b.Reset()
	b.size = 0
}

// generateKey generate a prefixed key
func generateKey(raw []byte, prefix string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte(prefix))
	if raw != nil {
		bytesBuffer.Write(raw)
	}
	return bytesBuffer.Bytes()
}

type LDBDatabase struct {
	db *leveldb.DB

	quitLock sync.Mutex
	quitChan chan chan error

	filename string
	//cacheConfig   int
	//handlesConfig int

	inited bool
}

// NewLDBDatabase create level db instance by file
func NewLDBDatabase(file string, options *opt.Options) (*LDBDatabase, error) {

	db, err := newLevelDBInstance(file, options)
	if err != nil {
		return nil, err
	}

	ldb := &LDBDatabase{
		filename: file,
		db:       db,
		//cacheConfig:   cache,
		//handlesConfig: handles,
		inited: true,
	}
	return ldb, nil
}

// newLevelDBInstance generate a leveldb instance
func newLevelDBInstance(file string, options *opt.Options) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(file, options)

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ldb *LDBDatabase) Clear() error {
	ldb.inited = false
	ldb.Close()

	os.RemoveAll(ldb.Path())

	db, err := newLevelDBInstance(ldb.Path(), nil)
	if err != nil {
		return err
	}

	ldb.db = db
	ldb.inited = true
	return nil
}

// Path returns the path to the database directory.
func (ldb *LDBDatabase) Path() string {
	return ldb.filename
}

// Put puts the given key / value to the queue
func (ldb *LDBDatabase) Put(key []byte, value []byte) error {
	if !ldb.inited {
		return ErrLDBInit
	}

	return ldb.db.Put(key, value, nil)
}

func (ldb *LDBDatabase) Has(key []byte) (bool, error) {
	if !ldb.inited {
		return false, ErrLDBInit
	}

	return ldb.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (ldb *LDBDatabase) Get(key []byte) ([]byte, error) {
	if !ldb.inited {
		return nil, ErrLDBInit
	}

	dat, err := ldb.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil

}

// Delete deletes the key from the queue and database
func (ldb *LDBDatabase) Delete(key []byte) error {
	if !ldb.inited {
		return ErrLDBInit
	}
	return ldb.db.Delete(key, nil)
}

func (ldb *LDBDatabase) NewIterator() iterator.Iterator {
	if !ldb.inited {
		return nil
	}
	return ldb.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (ldb *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return ldb.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (ldb *LDBDatabase) Close() {
	ldb.quitLock.Lock()
	defer ldb.quitLock.Unlock()

	if ldb.quitChan != nil {
		error := make(chan error)
		ldb.quitChan <- error
		if err := <-error; err != nil {
			log.DefaultLogger.Error("Metrics collection failed", "err", err)
		}
	}

	ldb.db.Close()
}

func (ldb *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: ldb.db, b: new(leveldb.Batch)}
	//return ldb.batch
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}
func (b *ldbBatch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size++
	return nil
}
func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}

type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() (*MemDatabase, error) {
	return &MemDatabase{
		db: make(map[string][]byte),
	}, nil
}

func (db *MemDatabase) Clear() error {
	db.db = make(map[string][]byte)
	return nil
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) NewIterator() iterator.Iterator {
	panic("Not support")
}

func (db *MemDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	panic("Not support")
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) Close() {}

func (db *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct {
	k, v []byte
	del  bool
}
type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

func (b *memBatch) Delete(key []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil, true})
	b.size++
	return nil
}

// Write flushes any accumulated data to disk.
func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.del {
			delete(b.db.db, string(kv.k))
			continue
		}
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

type LRUMemDatabase struct {
	db   *lru.Cache
	lock sync.RWMutex
}

func NewLRUMemDatabase(size int) (*LRUMemDatabase, error) {
	cache, _ := lru.New(size)
	return &LRUMemDatabase{
		db: cache,
	}, nil
}

// Put inserts the given value into the key-value data store.
func (db *LRUMemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.db.Add(string(key), common.CopyBytes(value))
	return nil
}

// Has retrieves if a key is present in the key-value data store.
func (db *LRUMemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db.Get(string(key))
	return ok, nil
}

// Get retrieves the given key if it's present in the key-value data store.
func (db *LRUMemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db.Get(string(key)); ok {
		vl, _ := entry.([]byte)
		return common.CopyBytes(vl), nil
	}
	return nil, nil
}

// Delete removes the key from the key-value data store.
func (db *LRUMemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.db.Remove(string(key))
	return nil
}

func (db *LRUMemDatabase) Close() {}

// NewBatch creates a write-only database that buffers changes to its host db
// until a final write is called.
func (db *LRUMemDatabase) NewBatch() Batch {
	return &LruMemBatch{db: db}
}

func (db *LRUMemDatabase) NewIterator() iterator.Iterator {
	panic("Not support")
}

func (db *LRUMemDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	panic("Not support")
}

type LruMemBatch struct {
	db     *LRUMemDatabase
	writes []kv
	size   int
}

// Put inserts the given value into the key-value data store.
func (b *LruMemBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

// Delete removes the key from the key-value data store.
func (b *LruMemBatch) Delete(key []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil, true})
	b.size++
	return nil
}

// Write flushes any accumulated data to disk.
func (b *LruMemBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.del {
			b.db.db.Remove(string(kv.k))
		} else {
			b.db.db.Add(string(kv.k), kv.v)
		}
	}
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *LruMemBatch) ValueSize() int {
	return b.size
}

// Reset resets the batch for reuse.
func (b *LruMemBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
