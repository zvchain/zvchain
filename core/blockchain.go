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

package core

import (
	"errors"
	"fmt"
	"github.com/zvchain/zvchain/common/prque"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zvchain/zvchain/storage/trie"

	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/log"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/ticker"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
)

const (
	blockStatusKey = "bcurrent"
	configSec      = "chain"
	prune          = "prune"
)

var (
	ErrBlockExist      = errors.New("block exist")
	ErrPreNotExist     = errors.New("pre block not exist")
	ErrLocalMoreWeight = errors.New("local more weight")
	ErrCommitBlockFail = errors.New("commit block fail")
	ErrBlockSizeLimit  = errors.New("block size exceed the limit")
)

var BlockChainImpl *FullBlockChain

var GroupManagerImpl *group.Manager

var Logger *logrus.Logger

// BlockChainConfig contains the configuration values of leveldb prefix string
type BlockChainConfig struct {
	dbfile      string
	block       string
	blockHeight string
	state       string
	reward      string
	tx          string
	receipt     string
	// Whether running node in pruning mode
	pruneMode bool
	// pruning mode config
	pruneConfig *PruneConfig
}

type PruneConfig struct {
	// max trie state memory limit,if over this value,will commit to big db and clear memory
	maxTriesInMemory common.StorageSize
	// if trie state over maxTriesInMemory,will commit to big db and clear memory size
	everyClearFromMemory common.StorageSize
	// prune count over this value,will commit next height block
	persistenceCount int
}

// FullBlockChain manages chain imports, reverts, chain reorganisations.
type FullBlockChain struct {
	blocks          *tasdb.PrefixedDatabase
	blockHeight     *tasdb.PrefixedDatabase
	txDb            *tasdb.PrefixedDatabase
	stateDb         *tasdb.PrefixedDatabase
	smallStateDb    *smallStateStore
	cacheDb         *tasdb.PrefixedDatabase
	batch           tasdb.Batch
	triegc          *prque.Prque // Priority queue mapping block numbers to tries to gc
	stateCache      account.AccountDatabase
	running         int32 // running must be called atomically
	transactionPool types.TransactionPool

	latestBlock   *types.BlockHeader // Latest block on chain
	latestStateDB *account.AccountDB
	latestCP      atomic.Value // Latest checkpoint *types.BlockHeader

	topRawBlocks *lru.Cache

	rwLock sync.RWMutex // Read-write lock

	mu      sync.Mutex // Mutex lock
	batchMu sync.Mutex // Batch mutex for add block on blockchain

	init bool // Init means where blockchain can work

	stateProc *stateProcessor

	futureRawBlocks  *lru.Cache
	verifiedBlocks   *lru.Cache
	newBlockMessages *lru.Cache

	isAdjusting bool // isAdjusting which means there may be a fork

	consensusHelper types.ConsensusHelper

	rewardManager *rewardManager

	forkProcessor *forkProcessor
	config        *BlockChainConfig

	txBatch *txBatchAdder

	ticker *ticker.GlobalTicker // Ticker is a global time ticker
	ts     time2.TimeService
	types.Account

	cpChecker *cpChecker
}

func getPruneConfig() *PruneConfig {
	pruneMode := common.GlobalConf.GetBool(configSec, "prune_mode", true)
	if !pruneMode {
		return nil
	}
	maxTriesInMem := common.StorageSize(common.GlobalConf.GetInt(prune, "max_tries_memory", defaultMaxTriesInMemory) * 1024 * 1024)
	everyClearFromMem := common.StorageSize(common.GlobalConf.GetInt(prune, "clear_tries_memory", defaultEveryClearFromMemory) * 1024 * 1024)
	persistenceCt := common.GlobalConf.GetInt(prune, "persistence_count", defaultPersistenceCount)

	if maxTriesInMem <= 0 {
		panic("config max_tries_memory must be more than 0")
	}
	if everyClearFromMem <= 0 {
		panic("config clear_tries_memory must be more than 0")
	}
	if persistenceCt < 0 {
		panic("config persistence_count must be more than 0")
	}
	if maxTriesInMem <= everyClearFromMem{
		panic("config max_tries_memory must be more than clear_tries_memory config")
	}
	return &PruneConfig{
		maxTriesInMemory:maxTriesInMem,
		everyClearFromMemory:everyClearFromMem,
		persistenceCount:persistenceCt,
	}
}

func getBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		dbfile:      common.GlobalConf.GetString(configSec, "db_blocks", "d_b"),
		block:       "bh",
		blockHeight: "hi",
		state:       "st",
		reward:      "nu",
		tx:          "tx",
		receipt:     "rc",
		pruneMode:   common.GlobalConf.GetBool(configSec, "prune_mode", true),
		pruneConfig: getPruneConfig(),
	}
}

func initBlockChain(helper types.ConsensusHelper, minerAccount types.Account) error {
	Logger = log.CoreLogger
	chain := &FullBlockChain{
		config:           getBlockChainConfig(),
		latestBlock:      nil,
		init:             true,
		isAdjusting:      false,
		consensusHelper:  helper,
		ticker:           ticker.NewGlobalTicker("chain"),
		triegc:           prque.NewPrque(),
		ts:               time2.TSInstance,
		futureRawBlocks:  common.MustNewLRUCache(100),
		verifiedBlocks:   common.MustNewLRUCache(10),
		topRawBlocks:     common.MustNewLRUCache(20),
		newBlockMessages: common.MustNewLRUCache(100),
		Account:          minerAccount,
	}

	types.DefaultPVFunc = helper.VRFProve2Value

	chain.initMessageHandler()

	conf := common.GlobalConf.GetSectionManager(configSec)
	// get the level db file cache size from config
	fileCacheSize := common.GlobalConf.GetInt(configSec, "db_file_cache", 5000)
	// get the level db block cache size from config
	blockCacheSize := conf.GetInt("db_block_cache", 512)
	// get the level db write cache size from config
	writeBufferSize := common.GlobalConf.GetInt(configSec, "db_write_cache", 512)
	stateCacheSize := common.GlobalConf.GetInt(configSec, "db_state_cache", 256)

	options := &opt.Options{
		OpenFilesCacheCapacity: fileCacheSize,
		BlockCacheCapacity:     blockCacheSize * opt.MiB,
		WriteBuffer:            writeBufferSize * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	}

	ds, err := tasdb.NewDataSource(chain.config.dbfile, options)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.txDb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.stateDb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return err
	}

	receiptdb, err := ds.NewPrefixDatabase(chain.config.receipt)
	if err != nil {
		Logger.Errorf("Init block chain error! Error:%s", err.Error())
		return err
	}
	smallStateDs, err := tasdb.NewDataSource(common.GlobalConf.GetString(configSec, "small_db", "d_small"), nil)
	if err != nil {
		Logger.Errorf("new small state datasource error:%v", err)
		return err
	}
	smallStateDb, err := smallStateDs.NewPrefixDatabase("")
	if err != nil {
		Logger.Errorf("new small state db error:%v", err)
		return err
	}
	chain.smallStateDb = initSmallStore(smallStateDb)
	chain.rewardManager = NewRewardManager()
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = newTransactionPool(chain, receiptdb)

	chain.txBatch = newTxBatchAdder(chain.transactionPool)

	chain.stateCache = account.NewDatabaseWithCache(chain.stateDb, chain.config.pruneMode, stateCacheSize, conf.GetString("state_cache_dir", "state_cache"))

	latestBH := chain.loadCurrentBlock()

	GroupManagerImpl = group.NewManager(chain, helper)

	chain.cpChecker = newCpChecker(GroupManagerImpl, chain)
	sp := newStateProcessor(chain)
	sp.addPostProcessor(GroupManagerImpl.RegularCheck)
	sp.addPostProcessor(chain.cpChecker.updateVotes)
	sp.addPostProcessor(MinerManagerImpl.GuardNodesCheck)
	sp.addPostProcessor(GroupManagerImpl.UpdateGroupSkipCounts)
	chain.stateProc = sp
	// merge small db state data to big db
	err = chain.mergeSmallDbDataToBigDB(latestBH)
	if err != nil {
		return err
	}
	if nil != latestBH {
		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		state, err := account.NewAccountDB(common.BytesToHash(latestBH.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.updateLatestBlock(state, latestBH)
			chain.buildCache(10)
			Logger.Debugf("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		} else {
			err = errors.New("initBlockChain NewAccountDB fail::" + err.Error())
			Logger.Error(err)
			return err
		}
		fmt.Printf("db height is %v at %v\n", latestBH.Height, latestBH.CurTime.Local().String())
	} else {
		chain.insertGenesisBlock()
	}

	chain.forkProcessor = initForkProcessor(chain, helper)

	BlockChainImpl = chain

	// db cache enabled
	iteratorNodeCacheSize := 30000
	if iteratorNodeCacheSize > 0 {
		cacheDs, err := tasdb.NewDataSource(common.GlobalConf.GetString(configSec, "db_cache", "d_cache"), nil)
		if err != nil {
			Logger.Errorf("new cache datasource error:%v", err)
			return err
		}
		cacheDB, err := cacheDs.NewPrefixDatabase("")
		if err != nil {
			Logger.Errorf("new cache db error:%v", err)
			return err
		}
		chain.cacheDb = cacheDB
		trie.CreateNodeCache(iteratorNodeCacheSize, cacheDB)
		initMinerManager(cacheDB)
	} else {
		initMinerManager(nil)
	}

	GroupManagerImpl.InitManager(MinerManagerImpl, chain.consensusHelper.GenerateGenesisInfo())

	chain.cpChecker.init()

	initStakeGetter(MinerManagerImpl, chain)

	chain.LogDbStats()
	return nil
}

func (chain *FullBlockChain) IsPruneMode() bool {
	return chain.config.pruneMode
}

func (chain *FullBlockChain) LogDbStats() {
	dbInterval := common.GlobalConf.GetInt(configSec, "meter_db_interval", 0)
	if dbInterval <= 0 {
		return
	}
	tc := time.NewTicker(time.Duration(dbInterval) * time.Second)
	go func() {
		for range tc.C {
			chain.stateDb.LogStats(log.MeterLogger)
		}
	}()
}

func (chain *FullBlockChain) buildCache(size int) {
	hash := chain.latestBlock.Hash
	for size > 0 {
		b := chain.queryBlockByHash(hash)
		if b != nil {
			chain.addTopBlock(b)
			size--
			hash = b.Header.PreHash
		} else {
			break
		}
	}
}

// insertGenesisBlock creates the genesis block and some necessary information，
// and commit it
func (chain *FullBlockChain) insertGenesisBlock() {
	stateDB, err := account.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil != err {
		panic("Init block chain error:" + err.Error())
	}

	block := new(types.Block)
	block.Header = &types.BlockHeader{
		Height:     0,
		ExtraData:  common.Sha256([]byte("zv")),
		CurTime:    time2.TimeToTimeStamp(time.Date(2019, 9, 28, 0, 0, 0, 0, time.UTC)),
		ProveValue: []byte{},
		Elapsed:    0,
		TotalQN:    0,
		Nonce:      common.ChainDataVersion,
	}

	block.Header.Signature = common.Sha256([]byte("zv"))
	block.Header.Random = common.Sha256([]byte("zv_initial_random"))

	genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
	setupGenesisStateDB(stateDB, genesisInfo)
	GroupManagerImpl.InitGenesis(stateDB, genesisInfo)

	miners := make([]*types.Miner, 0)
	for i, member := range genesisInfo.Group.Members() {
		miner := &types.Miner{ID: member.ID(), PublicKey: genesisInfo.Pks[i], VrfPublicKey: genesisInfo.VrfPKs[i], Stake: minimumStake()}
		miners = append(miners, miner)
	}
	MinerManagerImpl.addGenesesMiners(miners, stateDB)
	MinerManagerImpl.genFundGuardNodes(stateDB)

	// Create the global-use address
	stateDB.SetNonce(common.MinerPoolAddr, 1)
	stateDB.SetNonce(common.RewardStoreAddr, 1)
	stateDB.SetNonce(common.GroupTopAddress, 1)
	stateDB.SetNonce(cpAddress, 1)

	// mark group votes at 0
	chain.cpChecker.setGroupVotes(stateDB, []uint16{1})
	chain.cpChecker.setGroupEpoch(stateDB, types.EpochAt(0))

	root := stateDB.IntermediateRoot(true)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	ok, err := chain.commitBlock(block, &executePostState{state: stateDB})
	if !ok {
		panic("insert genesis block fail, err=" + err.Error())
	}

	Logger.Debugf("GenesisBlock %+v", block.Header)
}

// Clear clear blockchain all data. Not used now, should remove it latter
func (chain *FullBlockChain) Clear() error {
	return nil
}

func (chain *FullBlockChain) versionValidate() bool {
	genesisHeader := chain.queryBlockHeaderByHeight(uint64(0))
	if genesisHeader == nil {
		return false
	}
	version := genesisHeader.Nonce
	if version != common.ChainDataVersion {
		return false
	}
	return true
}

func (chain *FullBlockChain) compareChainWeight(bh2 *types.BlockHeader) int {
	bh1 := chain.getLatestBlock()
	return chain.compareBlockWeight(bh1, bh2)
}

func (chain *FullBlockChain) compareBlockWeight(bh1 *types.BlockHeader, bh2 *types.BlockHeader) int {
	bw1 := types.NewBlockWeight(bh1)
	bw2 := types.NewBlockWeight(bh2)
	return bw1.Cmp(bw2)
}

// Close the open levelDb files
func (chain *FullBlockChain) Close() {
	// Persist cache data
	if chain.stateCache != nil {
		chain.stateCache.TrieDB().SaveCache()
	}
	chain.PersistentState()
	if chain.blocks != nil {
		chain.blocks.Close()
	}
	if chain.blockHeight != nil {
		chain.blockHeight.Close()
	}

	if chain.stateDb != nil {
		chain.stateDb.Close()
	}
	if chain.cacheDb != nil {
		chain.cacheDb.Close()
	}
	if chain.smallStateDb != nil {
		chain.smallStateDb.Close()
	}
}

// GetRewardManager returns the reward manager
func (chain *FullBlockChain) GetRewardManager() types.RewardManager {
	return chain.rewardManager
}

// GetConsensusHelper returns consensus helper reference
func (chain *FullBlockChain) GetConsensusHelper() types.ConsensusHelper {
	return chain.consensusHelper
}

// ResetTop reset the current top block with parameter bh
func (chain *FullBlockChain) ResetTop(bh *types.BlockHeader) error {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	return chain.resetTop(bh)
}

// ResetNear reset the current top block with parameter bh,if parameter bh state  not exists,then find last restart point
func (chain *FullBlockChain) ResetNear(bh *types.BlockHeader) (restartBh *types.BlockHeader, err error) {
	localHeight := chain.Height()
	fmt.Printf("prepare reset to %v,local height is %v \n", bh.Height, localHeight)
	chain.mu.Lock()
	defer chain.mu.Unlock()

	lastRestartHeader, err := chain.findLastRestartPoint(bh)
	if err != nil {
		return nil, err
	}
	fmt.Printf("begin reset block,target height is %v \n", lastRestartHeader.Height)
	err = chain.resetTop(lastRestartHeader)
	if err != nil {
		return nil, fmt.Errorf("reset nil error,err is %v", err)
	}
	err = chain.smallStateDb.StoreStatePersistentHeight(lastRestartHeader.Height)
	if err != nil {
		return nil, fmt.Errorf("resetNear write persistentHeight to small db error,err is %v", err)
	}
	return lastRestartHeader, nil
}

// FindLastRestartPoint find last restart point,find 980 blocks state from parameter bh if state exists
func (chain *FullBlockChain) findLastRestartPoint(bh *types.BlockHeader) (restartBh *types.BlockHeader, err error) {
	var (
		cnt     uint64 = 0
		beginBh        = bh
	)
	for cnt < TriesInMemory {
		_, e := chain.accountDBAt(bh.Height)
		// if err not nil,then reset count
		if e != nil {
			cnt = 0
		} else {
			if cnt == 0 {
				beginBh = bh
			}
			cnt++
		}
		preHash := bh.PreHash
		bh = chain.queryBlockHeaderByHash(preHash)
		if bh == nil {
			return nil, fmt.Errorf("find block hash not exists,block hash is %v", preHash)
		}
		if bh.Height == 0 {
			return bh, nil
		}
		_, e = chain.accountDBAt(bh.Height)
	}
	return beginBh, nil
}

// Remove removes the block and blocks after it from the chain. Only used in a debug file, should be removed later
func (chain *FullBlockChain) Remove(block *types.Block) bool {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	pre := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if pre == nil {
		return chain.removeOrphan(block) == nil
	}
	return chain.resetTop(pre) == nil
}

func (chain *FullBlockChain) getLatestBlock() *types.BlockHeader {
	result := chain.latestBlock
	return result
}

// Version of chain Id
func (chain *FullBlockChain) Version() int {
	return common.ChainDataVersion
}

// Version of chain Id
func (chain *FullBlockChain) AddTransactionToPool(tx *types.Transaction) (bool, error) {
	return chain.GetTransactionPool().AddTransaction(tx)
}
