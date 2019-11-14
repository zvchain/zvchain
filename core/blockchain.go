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
	"github.com/zvchain/zvchain/storage/trie"
	"os"
	"sync"
	"sync/atomic"
	"time"

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
}

// FullBlockChain manages chain imports, reverts, chain reorganisations.
type FullBlockChain struct {
	blocks      *tasdb.PrefixedDatabase
	blockHeight *tasdb.PrefixedDatabase
	txDb        *tasdb.PrefixedDatabase
	stateDb     *tasdb.PrefixedDatabase
	cacheDb     *tasdb.PrefixedDatabase
	batch       tasdb.Batch

	stateCache account.AccountDatabase

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

func getBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		dbfile: common.GlobalConf.GetString(configSec, "db_blocks", "d_b"),
		block:  "bh",

		blockHeight: "hi",

		state: "st",

		reward: "nu",

		tx:      "tx",
		receipt: "rc",
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
		ts:               time2.TSInstance,
		futureRawBlocks:  common.MustNewLRUCache(100),
		verifiedBlocks:   common.MustNewLRUCache(10),
		topRawBlocks:     common.MustNewLRUCache(20),
		newBlockMessages: common.MustNewLRUCache(100),
		Account:          minerAccount,
	}

	types.DefaultPVFunc = helper.VRFProve2Value

	chain.initMessageHandler()

	// get the level db file cache size from config
	fileCacheSize := common.GlobalConf.GetInt(configSec, "db_file_cache", 500)
	// get the level db block cache size from config
	blockCacheSize := common.GlobalConf.GetInt(configSec, "db_block_cache", 512)
	// get the level db write cache size from config
	writeBufferSize := common.GlobalConf.GetInt(configSec, "db_write_cache", 512)

	iteratorNodeCacheSize := common.GlobalConf.GetInt(configSec, "db_node_cache", 30000)

	options := &opt.Options{
		OpenFilesCacheCapacity:        fileCacheSize,
		BlockCacheCapacity:            blockCacheSize * opt.MiB,
		WriteBuffer:                   writeBufferSize * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           4 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
		CompactionTotalSize:           16 * opt.MiB,
		BlockSize:                     64 * opt.KiB,
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
	chain.rewardManager = NewRewardManager()
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = newTransactionPool(chain, receiptdb)

	chain.txBatch = newTxBatchAdder(chain.transactionPool)

	chain.stateCache = account.NewDatabase(chain.stateDb)

	latestBH := chain.loadCurrentBlock()

	GroupManagerImpl = group.NewManager(chain, helper)

	chain.cpChecker = newCpChecker(GroupManagerImpl, chain)
	sp := newStateProcessor(chain)
	sp.addPostProcessor(GroupManagerImpl.RegularCheck)
	sp.addPostProcessor(chain.cpChecker.updateVotes)
	sp.addPostProcessor(MinerManagerImpl.GuardNodesCheck)
	sp.addPostProcessor(GroupManagerImpl.UpdateGroupSkipCounts)
	chain.stateProc = sp

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
	} else {
		chain.insertGenesisBlock()
	}

	chain.forkProcessor = initForkProcessor(chain, helper)

	BlockChainImpl = chain

	// db cache enabled
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

	return nil
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
func (chain *FullBlockChain) ResetTop(bh *types.BlockHeader) {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	chain.resetTop(bh)
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
