// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"errors"
	"fmt"
	"github.com/VictoriaMetrics/fastcache"
	"github.com/zvchain/zvchain/log"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zvchain/zvchain/storage/tasdb"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/rlp"
)

// secureKeyPrefix is the database key prefix used to store trie node preimages.
var secureKeyPrefix = []byte("secure-key-")

// secureKeyLength is the length of the above prefix + 32byte hash.
const secureKeyLength = 11 + 32

// DatabaseReader wraps the Get and Has method of a backing store for the trie.
type DatabaseReader interface {
	// Get retrieves the value associated with key form the database.
	Get(key []byte) (value []byte, err error)

	// Has retrieves whether a key is present in the database.
	Has(key []byte) (bool, error)
}

type readMeter struct {
	readCount uint64
	nodeHit   uint64
	cacheHit  uint64
	miss      uint64
	missSize  uint64
	hitSize   uint64
}

// DirtyStateReader wraps the Get and Has method of a backing store for the dirty trie.
type DirtyStateReader interface {
	StoreDirtyTrie(root common.Hash, nb []byte) error
}

// NodeDatabase is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
type NodeDatabase struct {
	diskdb      tasdb.Database              // Persistent storage for matured trie nodes
	nodes       map[common.Hash]*cachedNode // Data and references relationships of a node
	cache       *fastcache.Cache
	meter       *readMeter // meter about node hit or cache hit

	oldest common.Hash // Oldest tracked node, flush-list head
	newest common.Hash // Newest tracked node, flush-list tail

	preimages map[common.Hash][]byte // Preimages of nodes from the secure trie
	seckeybuf [secureKeyLength]byte  // Ephemeral buffer for calculating preimage keys

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	flushtime  time.Duration      // Time spent on data flushing since last commit
	flushnodes uint64             // Nodes flushed since last commit
	flushsize  common.StorageSize // Data storage flushed since last commit

	childrenSize  common.StorageSize // Storage size of the external children tracking
	nodesSize     common.StorageSize // Storage size of the nodes cache (exc. flushlist)
	preimagesSize common.StorageSize // Storage size of the preimages cache
	lock            sync.RWMutex
	commitFullNodes []*storeBlob // CommitFullNodes strores commit nodes every time
	enableGc        bool         // enableGc crop dirty state if true
	lastGcHeight    uint64
	lastGcCount     uint64
}

// rawNode is a simple binary blob used to differentiate between collapsed trie
// nodes and already encoded RLP binary blobs (while at the same time store them
// in the same cache fields).
type rawNode []byte

func (n rawNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

// rawFullNode represents only the useful data content of a full node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawFullNode [17]node

func (n rawFullNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawFullNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawFullNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

func (n rawFullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node

	for i, child := range n {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

// rawShortNode represents only the useful data content of a short node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawShortNode struct {
	Key []byte
	Val node
}

func (n rawShortNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawShortNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawShortNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

// cachedNode is all the information we know about a single cached node in the
// memory database write layer.
type cachedNode struct {
	node      node                   // Cached collapsed trie node, or raw rlp data
	size      uint16                 // Byte size of the useful cached data
	parents   uint16                 // Number of live nodes referencing this one
	children  map[common.Hash]uint16 // External children referenced by this node
	flushPrev common.Hash            // Previous node in the flush-list
	flushNext common.Hash            // Next node in the flush-list
}

// cachedNodeSize is the raw size of a cachedNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var cachedNodeSize = int(reflect.TypeOf(cachedNode{}).Size())

// cachedNodeChildrenSize is the raw size of an initialized but empty external
// reference map.
const cachedNodeChildrenSize = 48

// rlp returns the raw rlp encoded blob of the cached node, either directly from
// the cache, or by regenerating it from the collapsed node.
func (n *cachedNode) rlp() []byte {
	if node, ok := n.node.(rawNode); ok {
		return node
	}
	blob, err := rlp.EncodeToBytes(n.node)
	if err != nil {
		panic(err)
	}
	return blob
}

// obj returns the decoded and expanded trie node, either directly from the cache,
// or by regenerating it from the rlp encoded blob.
func (n *cachedNode) obj(hash common.Hash, cachegen uint16) node {
	if node, ok := n.node.(rawNode); ok {
		return mustDecodeNode(hash[:], node, cachegen)
	}
	return expandNode(hash[:], n.node, cachegen)
}

// childs returns all the tracked children of this node, both the implicit ones
// from inside the node as well as the explicit ones from outside the node.
func (n *cachedNode) childs() []common.Hash {
	children := make([]common.Hash, 0, 16)
	for child := range n.children {
		children = append(children, child)
	}
	if _, ok := n.node.(rawNode); !ok {
		gatherChildren(n.node, &children)
	}
	return children
}

// gatherChildren traverses the node hierarchy of a collapsed storage node and
// retrieves all the hashnode children.
func gatherChildren(n node, children *[]common.Hash) {
	switch n := n.(type) {
	case *rawShortNode:
		gatherChildren(n.Val, children)

	case rawFullNode:
		for i := 0; i < 16; i++ {
			gatherChildren(n[i], children)
		}
	case hashNode:
		*children = append(*children, common.BytesToHash(n))

	case valueNode, nil:

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// simplifyNode traverses the hierarchy of an expanded memory node and discards
// all the internal caches, returning a node that only contains the raw data.
func simplifyNode(n node) node {
	switch n := n.(type) {
	case *shortNode:
		// Short nodes discard the flags and cascade
		return &rawShortNode{Key: n.Key, Val: simplifyNode(n.Val)}

	case *fullNode:
		// Full nodes discard the flags and cascade
		node := rawFullNode(n.Children)
		for i := 0; i < len(node); i++ {
			if node[i] != nil {
				node[i] = simplifyNode(node[i])
			}
		}
		return node

	case valueNode, hashNode, rawNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// expandNode traverses the node hierarchy of a collapsed storage node and converts
// all fields and keys into expanded memory form.
func expandNode(hash hashNode, n node, cachegen uint16) node {
	switch n := n.(type) {
	case *rawShortNode:
		// Short nodes need key and child expansion
		return &shortNode{
			Key: compactToHex(n.Key),
			Val: expandNode(nil, n.Val, cachegen),
			flags: nodeFlag{
				hash: hash,
				gen:  cachegen,
			},
		}

	case rawFullNode:
		// Full nodes need child expansion
		node := &fullNode{
			flags: nodeFlag{
				hash: hash,
				gen:  cachegen,
			},
		}
		for i := 0; i < len(node.Children); i++ {
			if n[i] != nil {
				node.Children[i] = expandNode(nil, n[i], cachegen)
			}
		}
		return node

	case valueNode, hashNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// NewDatabase creates a new trie database to store ephemeral trie content before
// its written out to disk or garbage collected.
func NewDatabase(diskdb tasdb.Database, cacheSize int, gcEnable bool) *NodeDatabase {
	var cache *fastcache.Cache
	if cacheSize > 0 {
		cache = fastcache.New(cacheSize * 1024 * 1024)
	}
	return &NodeDatabase{
		diskdb:    diskdb,
		nodes:     map[common.Hash]*cachedNode{{}: {}},
		preimages: make(map[common.Hash][]byte),
		enableGc:  gcEnable,
		cache:     cache,
		meter:     &readMeter{},
	}
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *NodeDatabase) DiskDB() DatabaseReader {
	return db.diskdb
}

// ResetNodeCache make new slice before commit.
func (db *NodeDatabase) ResetNodeCache() {
	if db.enableGc {
		db.commitFullNodes = []*storeBlob{}
	}
}

// InsertFullToDirtyDb insert nodes to dirty db
func (db *NodeDatabase) InsertFullToDirtyDb(root common.Hash, dbReader DirtyStateReader) error {
	if db.commitFullNodes != nil && len(db.commitFullNodes) > 0 {
		dts, err := rlp.EncodeToBytes(db.commitFullNodes)
		if err != nil {
			return fmt.Errorf("encode error，error is %v", err)
		}
		err = dbReader.StoreDirtyTrie(root, dts)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertBlob writes a new reference tracked blob to the memory database if it's
// yet unknown. This method should only be used for non-trie nodes that require
// reference counting, since trie nodes are garbage collected directly through
// their embedded children.
func (db *NodeDatabase) InsertBlob(hash common.Hash, blob []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.insert(hash, blob, rawNode(blob))
}

// insert inserts a collapsed trie node into the memory database. This method is
// a more generic version of InsertBlob, supporting both raw blob insertions as
// well ex trie node insertions. The blob must always be specified to allow proper
// size tracking.
func (db *NodeDatabase) insert(hash common.Hash, blob []byte, node node) {
	// If the node's already cached, skip
	if _, ok := db.nodes[hash]; ok {
		return
	}
	// Create the cached entry for this node
	entry := &cachedNode{
		node:      simplifyNode(node),
		size:      uint16(len(blob)),
		flushPrev: db.newest,
	}
	for _, child := range entry.childs() {
		if c := db.nodes[child]; c != nil {
			c.parents++
		}
	}
	db.nodes[hash] = entry
	if db.enableGc {
		db.commitFullNodes = append(db.commitFullNodes, &storeBlob{Key: hash, Raw: entry.rlp()})
	}
	// Update the flush-list endpoints
	if db.oldest == (common.Hash{}) {
		db.oldest, db.newest = hash, hash
	} else {
		db.nodes[db.newest].flushNext, db.newest = hash, hash
	}
	db.nodesSize += common.StorageSize(common.HashLength + entry.size)
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will make a copy of the slice.
//
// Note, this method assumes that the database's lock is held!
func (db *NodeDatabase) insertPreimage(hash common.Hash, preimage []byte) {
	if _, ok := db.preimages[hash]; ok {
		return
	}
	db.preimages[hash] = common.CopyBytes(preimage)
	db.preimagesSize += common.StorageSize(common.HashLength + len(preimage))
}

func (db *NodeDatabase) tryGetFromCache(hash common.Hash) []byte {
	// reset the meter if read count > 2000000, for more accuracy statistic about recent db read
	if db.meter.readCount >= 2000000 {
		db.meter = &readMeter{}
	}
	atomic.AddUint64(&db.meter.readCount, 1)
	if db.cache != nil {
		meter := db.meter
		if enc := db.cache.Get(nil, hash[:]); enc != nil {
			atomic.AddUint64(&meter.cacheHit, 1)
			atomic.AddUint64(&meter.hitSize, uint64(len(enc)))
			if meter.readCount%1000 == 0 {
				log.CoreLogger.Infof("fastcache total %v, cacheHit %v, cacheHitRate %v, nodeHit %v, nodeHitRate %v, hitSize %vMB, missRate %v, missSize %vMB",
					meter.readCount, meter.cacheHit, float64(meter.cacheHit)/float64(meter.readCount), meter.nodeHit, float64(meter.nodeHit)/float64(meter.readCount), meter.hitSize/1024.0/1024.0,
					float64(meter.miss)/float64(meter.readCount), meter.missSize/1024.0/1024.0)
			}
			return enc
		}
	}
	return nil
}

func (db *NodeDatabase) addToCache(hash common.Hash, data []byte) {
	atomic.AddUint64(&db.meter.miss, 1)
	atomic.AddUint64(&db.meter.missSize, uint64(len(data)))
	if db.cache != nil {
		db.cache.Set(hash[:], data)
	}
}

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (db *NodeDatabase) node(hash common.Hash, cachegen uint16) (node, []byte) {
	// Retrieve the node from the clean cache if available
	bs := db.tryGetFromCache(hash)
	if bs != nil {
		return mustDecodeNode(hash[:], bs, cachegen), bs
	}
	// Retrieve the node from cache if available
	db.lock.RLock()
	node := db.nodes[hash]
	db.lock.RUnlock()

	if node != nil {
		atomic.AddUint64(&db.meter.nodeHit, 1)
		return node.obj(hash, cachegen), nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := db.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		return nil, nil
	}
	db.addToCache(hash, enc)
	return mustDecodeNode(hash[:], enc, cachegen), enc
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *NodeDatabase) Node(hash common.Hash) ([]byte, error) {
	/// It doens't make sense to retrieve the metaroot
	if hash == (common.Hash{}) {
		return nil, errors.New("not found")
	}
	// Retrieve the node from the clean cache if available
	bs := db.tryGetFromCache(hash)
	if bs != nil {
		return bs, nil
	}

	// Retrieve the node from the dirty cache if available
	db.lock.RLock()
	dirty := db.nodes[hash]
	db.lock.RUnlock()

	if dirty != nil {
		atomic.AddUint64(&db.meter.nodeHit, 1)
		return dirty.rlp(), nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := db.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		return nil, nil
	}
	db.addToCache(hash, enc)
	return enc, nil
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *NodeDatabase) preimage(hash common.Hash) ([]byte, error) {
	// Retrieve the node from cache if available
	db.lock.RLock()
	preimage := db.preimages[hash]
	db.lock.RUnlock()

	if preimage != nil {
		return preimage, nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	return db.diskdb.Get(db.secureKey(hash[:]))
}

// secureKey returns the database key for the preimage of key, as an ephemeral
// buffer. The caller must not hold onto the return value because it will become
// invalid on the next call.
func (db *NodeDatabase) secureKey(key []byte) []byte {
	buf := append(db.seckeybuf[:0], secureKeyPrefix...)
	buf = append(buf, key...)
	return buf
}

// Nodes retrieves the hashes of all the nodes cached within the memory database.
// This method is extremely expensive and should only be used to validate internal
// states in test code.
func (db *NodeDatabase) Nodes() []common.Hash {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(db.nodes))
	for hash := range db.nodes {
		if hash != (common.Hash{}) { // Special case for "root" references/nodes
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

// Reference adds a new reference from a parent node to a child node.
func (db *NodeDatabase) Reference(child common.Hash, parent common.Hash) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	db.reference(child, parent)
}

// reference is the private locked version of Reference.
func (db *NodeDatabase) reference(child common.Hash, parent common.Hash) {
	// If the node does not exist, it's a node pulled from disk, skip
	node, ok := db.nodes[child]
	if !ok {
		return
	}
	// If the reference already exists, only duplicate for roots
	if db.nodes[parent].children == nil {
		db.nodes[parent].children = make(map[common.Hash]uint16)
		db.childrenSize += cachedNodeChildrenSize
	} else if _, ok = db.nodes[parent].children[child]; ok && parent != (common.Hash{}) {
		return
	}
	node.parents++
	db.nodes[parent].children[child]++
	if db.nodes[parent].children[child] == 1 {
		db.childrenSize += common.HashLength + 2 // uint16 counter
	}
}

func (db *NodeDatabase) CommitDirtyToDb(dirtyBlobs []*storeBlob) error {
	batch := db.diskdb.NewBatch()
	if len(dirtyBlobs) > 0 {
		for _, vl := range dirtyBlobs {
			if err := batch.Put(vl.Key[:], vl.Raw); err != nil {
				return err
			}
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	return nil
}

func (db *NodeDatabase) DecodeStoreBlob(data []byte) (err error, dirtyBlobs []*storeBlob) {
	err = rlp.DecodeBytes(data, &dirtyBlobs)
	if err != nil {
		err = fmt.Errorf("decode storeBlob error,error is %v", err)
	}
	return nil, dirtyBlobs
}

func (db *NodeDatabase) LastGcHeight() uint64 {
	return db.lastGcHeight
}

func (db *NodeDatabase) CanPersistent(persistentCount int) bool {
	return db.lastGcCount >= uint64(persistentCount)
}

func (db *NodeDatabase) StoreGcData(height, currentHeight, cpHeight, gcCount uint64) {
	db.lastGcHeight = height
	db.lastGcCount += gcCount
	log.CropLogger.Debugf("store gc height is %v,curHeight is %v,cphHeight is %v,cur gc count is %v", height, currentHeight, cpHeight, gcCount)
}

func (db *NodeDatabase) ResetGcCount() {
	db.lastGcCount = 0
}

// Dereference removes an existing reference from a root node.
func (db *NodeDatabase) Dereference(height uint64, root common.Hash) {
	db.lock.Lock()
	defer db.lock.Unlock()

	nodes, storage, start := len(db.nodes), db.nodesSize, time.Now()
	db.dereference(root, common.Hash{})

	db.gcnodes += uint64(nodes - len(db.nodes))
	db.gcsize += storage - db.nodesSize
	db.gctime += time.Since(start)

	//memcacheGCTimeTimer.Update(time.Since(start))
	//memcacheGCSizeMeter.Mark(int64(storage - db.nodesSize))
	//memcacheGCNodesMeter.Mark(int64(nodes - len(db.nodes)))

	log.CropLogger.Debugf("Dereferenced trie from memory database,nodes=%v,size=%v,time=%v,gcnodes=%v,gcsize=%v,gctime=%v,livenodes=%v,livesize=%v,height=%v,root=%v", nodes-len(db.nodes), (storage-db.nodesSize)/(1024*1024), time.Since(start),
		db.gcnodes, db.gcsize/(1024*1024), db.gctime, len(db.nodes), db.nodesSize/(1024*1024), height, root.Hex())
}

// dereference is the private locked version of Dereference.
func (db *NodeDatabase) dereference(child common.Hash, parent common.Hash) {
	// Dereference the parent-child
	node := db.nodes[parent]

	if node.children != nil && node.children[child] > 0 {
		node.children[child]--
		if node.children[child] == 0 {
			delete(node.children, child)
			db.childrenSize -= (common.HashLength + 2) // uint16 counter
		}
	}
	// If the child does not exist, it's a previously committed node.
	node, ok := db.nodes[child]
	if !ok {
		return
	}
	// If there are no more references to the child, delete it and cascade
	if node.parents > 0 {
		// This is a special cornercase where a node loaded from disk (i.e. not in the
		// memcache any more) gets reinjected as a new node (short node split into full,
		// then reverted into short), causing a cached node to have no parents. That is
		// no problem in itself, but don't make maxint parents out of it.
		node.parents--
	}
	if node.parents == 0 {
		// Remove the node from the flush-list
		switch child {
		case db.oldest:
			db.oldest = node.flushNext
			db.nodes[node.flushNext].flushPrev = common.Hash{}
		case db.newest:
			db.newest = node.flushPrev
			db.nodes[node.flushPrev].flushNext = common.Hash{}
		default:
			db.nodes[node.flushPrev].flushNext = node.flushNext
			db.nodes[node.flushNext].flushPrev = node.flushPrev
		}
		// Dereference all children and delete the node
		for _, hash := range node.childs() {
			db.dereference(hash, child)
		}
		delete(db.nodes, child)
		db.nodesSize -= common.StorageSize(common.HashLength + int(node.size))
		if node.children != nil {
			db.childrenSize -= cachedNodeChildrenSize
		}
	}
}

// Cap iteratively flushes old but still referenced trie nodes until the total
// memory usage goes below the given threshold.
func (db *NodeDatabase) Cap(limit common.StorageSize) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	db.lock.RLock()

	nodes, storage, start := len(db.nodes), db.nodesSize, time.Now()
	batch := db.diskdb.NewBatch()

	// db.nodesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted. For every useful node, we track 2 extra hashes as the flushlist.
	size := db.nodesSize + common.StorageSize((len(db.nodes)-1)*2*common.HashLength)

	// If the preimage cache got large enough, push to disk. If it's still small
	// leave for later to deduplicate writes.
	flushPreimages := db.preimagesSize > 4*1024*1024
	if flushPreimages {
		for hash, preimage := range db.preimages {
			if err := batch.Put(db.secureKey(hash[:]), preimage); err != nil {
				//core.Logger.Error("Failed to commit preimage from trie database", "err", err)
				db.lock.RUnlock()
				return err
			}
			if batch.ValueSize() > tasdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					db.lock.RUnlock()
					return err
				}
				batch.Reset()
			}
		}
	}
	// Keep committing nodes from the flush-list until we're below allowance
	oldest := db.oldest
	for size > limit && oldest != (common.Hash{}) {
		// Fetch the oldest referenced node and push into the batch
		node := db.nodes[oldest]
		if err := batch.Put(oldest[:], node.rlp()); err != nil {
			db.lock.RUnlock()
			return err
		}
		// If we exceeded the ideal batch size, commit and reset
		if batch.ValueSize() >= tasdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				log.DefaultLogger.Error("Failed to write flush list to disk", "err", err)
				db.lock.RUnlock()
				return err
			}
			batch.Reset()
		}
		// Iterate to the next flush item, or abort if the size cap was achieved. Size
		// is the total size, including both the useful cached data (hash -> blob), as
		// well as the flushlist metadata (2*hash). When flushing items from the cache,
		// we need to reduce both.
		size -= common.StorageSize(3*common.HashLength + int(node.size))
		oldest = node.flushNext
	}
	// Flush out any remainder data from the last batch
	if err := batch.Write(); err != nil {
		log.DefaultLogger.Error("Failed to write flush list to disk", "err", err)
		db.lock.RUnlock()
		return err
	}
	db.lock.RUnlock()

	// Write successful, clear out the flushed data
	db.lock.Lock()
	defer db.lock.Unlock()

	if flushPreimages {
		db.preimages = make(map[common.Hash][]byte)
		db.preimagesSize = 0
	}
	for db.oldest != oldest {
		node := db.nodes[db.oldest]
		delete(db.nodes, db.oldest)
		db.oldest = node.flushNext

		db.nodesSize -= common.StorageSize(common.HashLength + int(node.size))
	}
	if db.oldest != (common.Hash{}) {
		db.nodes[db.oldest].flushPrev = common.Hash{}
	}
	db.flushnodes += uint64(nodes - len(db.nodes))
	db.flushsize += storage - db.nodesSize
	db.flushtime += time.Since(start)

	//memcacheFlushTimeTimer.Update(time.Since(start))
	//memcacheFlushSizeMeter.Mark(int64(storage - db.nodesSize))
	//memcacheFlushNodesMeter.Mark(int64(nodes - len(db.nodes)))

	log.DefaultLogger.Debug("Persisted nodes from memory database", "nodes", nodes-len(db.nodes), "size", storage-db.nodesSize, "time", time.Since(start),
		"flushnodes", db.flushnodes, "flushsize", db.flushsize, "flushtime", db.flushtime, "livenodes", len(db.nodes), "livesize", db.nodesSize)

	return nil
}

func (db *NodeDatabase) ClearFromNodes(height uint64, limit common.StorageSize) {
	db.lock.Lock()
	defer db.lock.Unlock()
	nodes, storage, start := len(db.nodes), db.nodesSize, time.Now()
	size := db.nodesSize + common.StorageSize((len(db.nodes)-1)*cachedNodeSize)
	size += db.childrenSize - common.StorageSize(len(db.nodes[common.Hash{}].children)*(common.HashLength+2))
	// Keep delete nodes from the flush-list until we're below allowance
	oldest := db.oldest
	for size > limit && oldest != (common.Hash{}) {
		// Fetch the oldest referenced node and push into the batch
		node := db.nodes[oldest]
		// Iterate to the next flush item, or abort if the size cap was achieved. Size
		// is the total size, including the useful cached data (hash -> blob), the
		// cache item metadata, as well as external children mappings.
		size -= common.StorageSize(common.HashLength + int(node.size) + cachedNodeSize)
		if node.children != nil {
			size -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
		}
		oldest = node.flushNext
	}
	for db.oldest != oldest {
		node := db.nodes[db.oldest]
		delete(db.nodes, db.oldest)
		db.oldest = node.flushNext

		db.nodesSize -= common.StorageSize(common.HashLength + int(node.size))
		if node.children != nil {
			db.childrenSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
		}
	}
	if db.oldest != (common.Hash{}) {
		db.nodes[db.oldest].flushPrev = common.Hash{}
	}

	db.flushnodes += uint64(nodes - len(db.nodes))
	db.flushsize += storage - db.nodesSize
	db.flushtime += time.Since(start)

	log.CropLogger.Debugf("Clear nodes from memory clear ,height is %v,clear count %v,clear size %v,cost %v,total flush nodes %v,total flush size %v,total flush time is %v,live node %v,live size %v",
		height, nodes-len(db.nodes), storage-db.nodesSize, time.Since(start), db.flushnodes, db.flushsize, db.flushtime, len(db.nodes), db.nodesSize/(1024*1024))
}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions.
//
// As a side effect, all pre-images accumulated up to this point are also written.
func (db *NodeDatabase) Commit(node common.Hash, report bool) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	db.lock.RLock()

	start := time.Now()
	batch := db.diskdb.NewBatch()

	// Move all of the accumulated preimages into a write batch
	for hash, preimage := range db.preimages {
		if err := batch.Put(db.secureKey(hash[:]), preimage); err != nil {
			log.DefaultLogger.Error("Failed to commit preimage from trie database", "err", err)
			db.lock.RUnlock()
			return err
		}
		if batch.ValueSize() > tasdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
	// Move the trie itself into the batch, flushing if enough data is accumulated
	nodes, storage := len(db.nodes), db.nodesSize
	if err := db.commit(node, batch); err != nil {
		log.DefaultLogger.Error("Failed to commit trie from trie database", "err", err)
		db.lock.RUnlock()
		return err
	}
	// Write batch ready, unlock for readers during persistence
	if err := batch.Write(); err != nil {
		log.DefaultLogger.Error("Failed to write trie to disk", "err", err)
		db.lock.RUnlock()
		return err
	}
	db.lock.RUnlock()

	// Write successful, clear out the flushed data
	db.lock.Lock()
	defer db.lock.Unlock()

	db.preimages = make(map[common.Hash][]byte)
	db.preimagesSize = 0
	db.uncache(node)
	//memcacheCommitTimeTimer.Update(time.Since(start))
	//memcacheCommitSizeMeter.Mark(int64(storage - db.nodesSize))
	//memcacheCommitNodesMeter.Mark(int64(nodes - len(db.nodes)))

	log.DefaultLogger.Debug("Persisted trie from memory database", "nodes", nodes-len(db.nodes)+int(db.flushnodes), "size", storage-db.nodesSize+db.flushsize, "time", time.Since(start)+db.flushtime,
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.nodes), "livesize", db.nodesSize)

	// Reset the garbage collection statistics
	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
	db.flushnodes, db.flushsize, db.flushtime = 0, 0, 0

	return nil
}

// commit is the private locked version of Commit.
func (db *NodeDatabase) commit(hash common.Hash, batch tasdb.Batch) error {
	// If the node does not exist, it's a previously committed node
	node, ok := db.nodes[hash]
	if !ok {
		return nil
	}
	for _, child := range node.childs() {
		if err := db.commit(child, batch); err != nil {
			return err
		}
	}
	v := node.rlp()
	if err := batch.Put(hash[:], v); err != nil {
		return err
	}
	db.cache.Set(hash.Bytes(), v)
	// If we've reached an optimal batch size, commit and start over
	if batch.ValueSize() >= tasdb.IdealBatchSize {
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
	}
	return nil
}

// uncache is the post-processing step of a commit operation where the already
// persisted trie is removed from the cache. The reason behind the two-phase
// commit is to ensure consistent data availability while moving from memory
// to disk.
func (db *NodeDatabase) uncache(hash common.Hash) {
	// If the node does not exist, we're done on this path
	node, ok := db.nodes[hash]
	if !ok {
		return
	}
	// Node still exists, remove it from the flush-list
	switch hash {
	case db.oldest:
		db.oldest = node.flushNext
		db.nodes[node.flushNext].flushPrev = common.Hash{}
	case db.newest:
		db.newest = node.flushPrev
		db.nodes[node.flushPrev].flushNext = common.Hash{}
	default:
		db.nodes[node.flushPrev].flushNext = node.flushNext
		db.nodes[node.flushNext].flushPrev = node.flushPrev
	}
	// Uncache the node's subtries and remove the node itself too
	for _, child := range node.childs() {
		db.uncache(child)
	}
	delete(db.nodes, hash)
	db.nodesSize -= common.StorageSize(common.HashLength + int(node.size))
	if node.children != nil {
		db.nodesSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashLength+2))
	}
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *NodeDatabase) Size() (common.StorageSize, common.StorageSize) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// db.nodesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted.
	var metadataSize = common.StorageSize((len(db.nodes) - 1) * cachedNodeSize)
	var metarootRefs = common.StorageSize(len(db.nodes[common.Hash{}].children) * (common.HashLength + 2))
	return db.nodesSize + db.childrenSize + metadataSize - metarootRefs, db.preimagesSize
}

// verifyIntegrity is a debug method to iterate over the entire trie stored in
// memory and check whether every node is reachable from the meta root. The goal
// is to find any errors that might cause memory leaks and or trie nodes to go
// missing.
//
// This method is extremely CPU and memory intensive, only use when must.
func (db *NodeDatabase) verifyIntegrity() {
	// Iterate over all the cached nodes and accumulate them into a set
	reachable := map[common.Hash]struct{}{{}: {}}

	for child := range db.nodes[common.Hash{}].children {
		db.accumulate(child, reachable)
	}
	// Find any unreachable but cached nodes
	unreachable := []string{}
	for hash, node := range db.nodes {
		if _, ok := reachable[hash]; !ok {
			unreachable = append(unreachable, fmt.Sprintf("%x: {Node: %v, Parents: %d, Prev: %x, Next: %x}",
				hash, node.node, node.parents, node.flushPrev, node.flushNext))
		}
	}
	if len(unreachable) != 0 {
		panic(fmt.Sprintf("trie cache memory leak: %v", unreachable))
	}
}

// accumulate iterates over the trie defined by hash and accumulates all the
// cached children found in memory.
func (db *NodeDatabase) accumulate(hash common.Hash, reachable map[common.Hash]struct{}) {
	// Mark the node reachable if present in the memory cache
	node, ok := db.nodes[hash]
	if !ok {
		return
	}
	reachable[hash] = struct{}{}

	// Iterate over all the children and accumulate them too
	for _, child := range node.childs() {
		db.accumulate(child, reachable)
	}
}
