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
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/log"
	"io"
	"reflect"
	"sync"
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

// DirtyStateReader wraps the Get and Has method of a backing store for the dirty trie.
type DirtyStateReader interface {
	StoreDirtyTrie(root common.Hash, nb []byte) error
}

// NodeDatabase is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
type NodeDatabase struct {
	diskdb tasdb.Database // Persistent storage for matured trie nodes

	dirtydb tasdb.Database // Persistent storage for full trie nodes
	dirtyNodes  map[common.Hash]*dbNode
	nodes  map[common.Hash]*cachedNode // Data and references relationships of a node
	oldest common.Hash                 // Oldest tracked node, flush-list head
	newest common.Hash                 // Newest tracked node, flush-list tail

	preimages map[common.Hash][]byte // Preimages of nodes from the secure trie
	seckeybuf [secureKeyLength]byte  // Ephemeral buffer for calculating preimage keys

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	flushtime  time.Duration      // Time spent on data flushing since last commit
	flushnodes uint64             // Nodes flushed since last commit
	flushsize  common.StorageSize // Data storage flushed since last commit

	nodesSize     common.StorageSize // Storage size of the nodes cache (exc. flushlist)
	childrenSize  common.StorageSize // Storage size of the external children tracking
	preimagesSize common.StorageSize // Storage size of the preimages cache

	lock sync.RWMutex

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
	node node   // Cached collapsed trie node, or raw rlp data
	size uint16 // Byte size of the useful cached data

	parents  uint16                 // Number of live nodes referencing this one
	children map[common.Hash]uint16 // External children referenced by this node

	flushPrev common.Hash // Previous node in the flush-list
	flushNext common.Hash // Next node in the flush-list
}

type dbNode struct {
	node node
	parents  uint16
}

func (n *dbNode) rlp() []byte {
	if node, ok := n.node.(rawNode); ok {
		return node
	}
	blob, err := rlp.EncodeToBytes(n.node)
	if err != nil {
		panic(err)
	}
	return blob
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
func NewDatabase(diskdb tasdb.Database, dirtyDb tasdb.Database) *NodeDatabase {
	return &NodeDatabase{
		diskdb:    diskdb,
		dirtydb:   dirtyDb,
		nodes:     map[common.Hash]*cachedNode{{}: {}},
		dirtyNodes:make(map[common.Hash]*dbNode),
		preimages: make(map[common.Hash][]byte),
	}
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *NodeDatabase) DiskDB() DatabaseReader {
	return db.diskdb
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



func (db *NodeDatabase) ClearDitry() {
	db.lock.Lock()
	db.dirtyNodes = make(map[common.Hash]*dbNode)
	db.lock.Unlock()
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
			if vl,ok := db.dirtyNodes[child];ok{
				vl.parents++
			}else{
				db.dirtyNodes[child] = &dbNode{node:c.node,parents: 1}
			}
		}
	}
	db.nodes[hash] = entry
	db.dirtyNodes[hash] = &dbNode{node:entry.node,parents:0}

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

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (db *NodeDatabase) node(hash common.Hash, cachegen uint16) (node, []byte) {
	// Retrieve the node from cache if available
	db.lock.RLock()
	node := db.nodes[hash]
	db.lock.RUnlock()

	if node != nil {
		return node.obj(hash, cachegen), nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := db.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		enc,err = db.dirtydb.Get(hash[:])
		if err != nil || enc == nil {
			return nil, nil
		}else{
			enc = enc[2:]
		}
	}
	return mustDecodeNode(hash[:], enc, cachegen), enc
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *NodeDatabase) Node(hash common.Hash) ([]byte, error) {
	// Retrieve the node from cache if available
	db.lock.RLock()
	node := db.nodes[hash]
	db.lock.RUnlock()

	if node != nil {
		return node.rlp(), nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	 //enc,err := db.diskdb.Get(hash[:])
	enc, err := db.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		enc,err = db.dirtydb.Get(hash[:])
		if err != nil || enc == nil {
			return nil, nil
		}else{
			enc = enc[2:]
		}
	}
	return enc,err
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

	if vlupdate,ok := db.dirtyNodes[child];ok{
		vlupdate.parents++
	}else{
		db.dirtyNodes[child] = &dbNode{node:node.node,parents: 1}
	}
}


// Dereference removes an existing reference from a root node.
func (db *NodeDatabase) Dereference(height uint64, root common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	cl := make(map[common.Hash]*dbNode)
	nodes, storage, start := len(db.nodes), db.nodesSize, time.Now()
	db.dereference(root, common.Hash{}, cl)

	db.gcnodes += uint64(nodes - len(db.nodes))
	db.gcsize += storage - db.nodesSize

	if len(cl) > 0 {
		err := db.ReduceDirtyStateParents(cl)
		if err != nil {
			return fmt.Errorf("delete dirty state failed,err is %v", err)
		}
	} else {
		log.CorpLogger.Debugf("Dereferenced trie,find no changed,height is %v,root is %v", height, root.Hex())
	}
	db.gctime += time.Since(start)
	log.CorpLogger.Debugf("Dereferenced trie from memory database,nodes=%v,size=%v,time=%v,gcnodes=%v,gcsize=%v,gctime=%v,livenodes=%v,livesize=%v,height=%v,root=%v", nodes-len(db.nodes), (storage-db.nodesSize)/(1024*1024), time.Since(start),
		db.gcnodes, db.gcsize/(1024*1024), db.gctime, len(db.nodes), db.nodesSize/(1024*1024), height, root.Hex())
	return nil
}

// dereference is the private locked version of Dereference.
func (db *NodeDatabase) dereference(child common.Hash, parent common.Hash, cl map[common.Hash]*dbNode) {
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
		if vl,ok:=cl[child];ok{
			vl.parents++
		}else{
			cl[child] = &dbNode{node:node.node,parents:1}
		}
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
			db.dereference(hash, child, cl)
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

	log.CorpLogger.Debug("Persisted nodes from memory database", "nodes", nodes-len(db.nodes), "size", storage-db.nodesSize, "time", time.Since(start),
		"flushnodes", db.flushnodes, "flushsize", db.flushsize, "flushtime", db.flushtime, "livenodes", len(db.nodes), "livesize", db.nodesSize)

	return nil
}


func (db *NodeDatabase) ReduceDirtyStateParents(dirtyData map[common.Hash]*dbNode) error {
	begin:= time.Now()
	batch := db.dirtydb.NewBatch()
	deleteCount := 0
	updateCount :=0
	for k,v := range dirtyData{
		bts,err := db.dirtydb.Get(k[:])
		if bts == nil || err != nil{
			log.CorpLogger.Warnf("reduce find update node value is nil,key is %s",k.Hex())
		}else{
			count := common.ByteToUInt16(bts[0:2])
			if v.parents >= count{
				err := batch.Delete(k[:])
				if err != nil{
					return err
				}
				deleteCount++
			}else{
				count -= v.parents
				bytesBuffer := bytes.NewBuffer(common.UInt16ToByte(count))
				bytesBuffer.Write(bts[2:])
				err := batch.Put(k[:],bytesBuffer.Bytes())
				if err != nil{
					return err
				}
				updateCount++
			}
		}
	}
	err :=  batch.Write()
	if err !=  nil{
		return err
	}
	log.CorpLogger.Debugf("reduce parent,delete count is %v,update count is %v,cost=%v",deleteCount,updateCount,time.Since(begin))
	return nil
}

func (db *NodeDatabase) DeleteDirtyState(dirtyData []common.Hash) error {
	batch := db.dirtydb.NewBatch()
	for _, k := range dirtyData {
		err := batch.Delete(k[:])
		if err != nil {
			return err
		}
	}
	err := batch.Write()
	if err != nil {
		return err
	}
	batch.Reset()
	log.CorpLogger.Debugf("delete count is %v",len(dirtyData))
	return nil
}

func (db *NodeDatabase) CommitDirtyToDb() error {
	begin := time.Now()
	db.lock.Lock()
	defer db.lock.Unlock()
	batch := db.dirtydb.NewBatch()
	for k,v := range db.dirtyNodes{
		bts,err := db.dirtydb.Get(k[:])
		if err == nil || len(bts) > 0{
			count := common.ByteToUInt16(bts[0:2])
			count += v.parents
			bytesBuffer := bytes.NewBuffer(common.UInt16ToByte(count))
			bytesBuffer.Write(bts[2:])
			err := batch.Put(k[:],bytesBuffer.Bytes())
			if err != nil{
				return err
			}
		}else{
			bytesBuffer := bytes.NewBuffer(common.UInt16ToByte(v.parents))
			bytesBuffer.Write(v.rlp())
			err := batch.Put(k[:],bytesBuffer.Bytes())
			if err != nil{
				return err
			}
		}
	}
	err :=  batch.Write()
	if err !=  nil{
		return err
	}
	log.CorpLogger.Debugf("commit dirty to small db, size is %v,cost=%v",len(db.dirtyNodes),time.Since(begin))
	return nil

}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions.
//
// As a side effect, all pre-images accumulated up to this point are also written.
func (db *NodeDatabase) Commit(node common.Hash, report bool) (error,[]common.Hash) {
	toDeleteHashs := []common.Hash{}
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
			return err,toDeleteHashs
		}
		if batch.ValueSize() > tasdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err,toDeleteHashs
			}
			batch.Reset()
		}
	}
	// Move the trie itself into the batch, flushing if enough data is accumulated
	nodes, storage := len(db.nodes), db.nodesSize
	if err := db.commit(node, batch); err != nil {
		log.DefaultLogger.Error("Failed to commit trie from trie database", "err", err)
		db.lock.RUnlock()
		return err,toDeleteHashs
	}
	// Write batch ready, unlock for readers during persistence
	if err := batch.Write(); err != nil {
		log.DefaultLogger.Error("Failed to write trie to disk", "err", err)
		db.lock.RUnlock()
		return err,toDeleteHashs
	}
	db.lock.RUnlock()

	// Write successful, clear out the flushed data
	db.lock.Lock()
	defer db.lock.Unlock()

	db.preimages = make(map[common.Hash][]byte)
	db.preimagesSize = 0

	db.uncache(node,&toDeleteHashs)

	//memcacheCommitTimeTimer.Update(time.Since(start))
	//memcacheCommitSizeMeter.Mark(int64(storage - db.nodesSize))
	//memcacheCommitNodesMeter.Mark(int64(nodes - len(db.nodes)))

	log.DefaultLogger.Debug("Persisted trie from memory database", "nodes", nodes-len(db.nodes)+int(db.flushnodes), "size", storage-db.nodesSize+db.flushsize, "time", time.Since(start)+db.flushtime,
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.nodes), "livesize", db.nodesSize)

	// Reset the garbage collection statistics
	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
	db.flushnodes, db.flushsize, db.flushtime = 0, 0, 0

	return nil,toDeleteHashs
}

// commit is the private locked version of Commit.
func (db *NodeDatabase) commit(hash common.Hash, batch tasdb.Batch)error {
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
	if err := batch.Put(hash[:], node.rlp()); err != nil {
		return err
	}
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
func (db *NodeDatabase) uncache(hash common.Hash,toDeleteHashs *[]common.Hash) {
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
		db.uncache(child,toDeleteHashs)
	}
	delete(db.nodes, hash)
	*toDeleteHashs = append(*toDeleteHashs,hash)
	db.nodesSize -= common.StorageSize(common.HashLength + int(node.size))
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *NodeDatabase) Size() (common.StorageSize, common.StorageSize) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// db.nodesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted. For every useful node, we track 2 extra hashes as the flushlist.
	var flushlistSize = common.StorageSize((len(db.nodes) - 1) * 2 * common.HashLength)
	return db.nodesSize + flushlistSize, db.preimagesSize
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
