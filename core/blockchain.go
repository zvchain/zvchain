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
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/ticker"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/tasdb"
	"github.com/zvchain/zvchain/taslog"
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
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

var consensusLogger taslog.Logger

// BlockChainConfig contains the configuration values of leveldb prefix string
type BlockChainConfig struct {
	dbfile      string
	block       string
	blockHeight string
	state       string
	bonus       string
	tx          string
	receipt     string
}

// FullBlockChain manages chain imports, reverts, chain reorganisations.
type FullBlockChain struct {
	blocks      *tasdb.PrefixedDatabase
	blockHeight *tasdb.PrefixedDatabase
	txDb        *tasdb.PrefixedDatabase
	stateDb     *tasdb.PrefixedDatabase
	batch       tasdb.Batch

	stateCache account.AccountDatabase

	transactionPool TransactionPool

	latestBlock   *types.BlockHeader // Latest block on chain
	latestStateDB *account.AccountDB

	topBlocks *lru.Cache

	rwLock sync.RWMutex // Read-write lock

	mu      sync.Mutex // Mutex lock
	batchMu sync.Mutex // Batch mutex for add block on blockchain

	init bool // Init means where blockchain can work

	executor *TVMExecutor

	futureBlocks   *lru.Cache
	verifiedBlocks *lru.Cache

	isAdjusting bool // isAdjusting which means there may be a fork

	consensusHelper types.ConsensusHelper

	bonusManager *BonusManager

	forkProcessor *forkProcessor
	config        *BlockChainConfig

	txBatch *txBatchAdder

	ticker *ticker.GlobalTicker // Ticker is a global time ticker
	ts     time2.TimeService
}

func getBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		dbfile: common.GlobalConf.GetString(configSec, "db_blocks", "d_b") + common.GlobalConf.GetString("instance", "index", ""),
		block:  "bh",

		blockHeight: "hi",

		state: "st",

		bonus: "nu",

		tx:      "tx",
		receipt: "rc",
	}
}

func initBlockChain(helper types.ConsensusHelper) error {
	instance := common.GlobalConf.GetString("instance", "index", "")
	Logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, instance)
	chain := &FullBlockChain{
		config:          getBlockChainConfig(),
		latestBlock:     nil,
		init:            true,
		isAdjusting:     false,
		consensusHelper: helper,
		ticker:          ticker.NewGlobalTicker("chain"),
		ts:              time2.TSInstance,
		futureBlocks:    common.MustNewLRUCache(10),
		verifiedBlocks:  common.MustNewLRUCache(10),
		topBlocks:       common.MustNewLRUCache(20),
	}

	types.DefaultPVFunc = helper.VRFProve2Value

	chain.initMessageHandler()

	options := &opt.Options{
		OpenFilesCacheCapacity:        100,
		BlockCacheCapacity:            16 * opt.MiB,
		WriteBuffer:                   512 * opt.MiB, // Two of these are used internally
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
	chain.bonusManager = newBonusManager()
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = newTransactionPool(chain, receiptdb)

	chain.txBatch = newTxBatchAdder(chain.transactionPool)

	chain.stateCache = account.NewDatabase(chain.stateDb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain.ticker)

	chain.latestBlock = chain.loadCurrentBlock()
	if nil != chain.latestBlock {
		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(10)
		Logger.Debugf("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		state, err := account.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic("initBlockChain NewAccountDB fail:" + err.Error())
		}
	} else {
		chain.insertGenesisBlock()
	}

	chain.forkProcessor = initForkProcessor(chain)

	BlockChainImpl = chain
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
		ExtraData:  common.Sha256([]byte("tas")),
		CurTime:    time2.TimeToTimeStamp(time.Date(2019, 4, 25, 0, 0, 0, 0, time.UTC)),
		ProveValue: []byte{},
		Elapsed:    0,
		TotalQN:    0,
		Nonce:      common.ChainDataVersion,
	}

	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
	setupGenesisStateDB(stateDB, genesisInfo)

	miners := make([]*types.Miner, 0)
	for i, member := range genesisInfo.Group.Members {
		miner := &types.Miner{ID: member, PublicKey: genesisInfo.Pks[i], VrfPublicKey: genesisInfo.VrfPKs[i], Stake: common.TAS2RA(100)}
		miners = append(miners, miner)
	}
	MinerManagerImpl.addGenesesMiner(miners, stateDB)
	stateDB.SetNonce(common.BonusStorageAddress, 1)
	stateDB.SetNonce(common.HeavyDBAddress, 1)
	stateDB.SetNonce(common.LightDBAddress, 1)
	stateDB.SetNonce(common.MinerStakeDetailDBAddress, 1)

	root, _ := stateDB.Commit(true)
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
	chain.blocks.Close()
	chain.blockHeight.Close()
	chain.stateDb.Close()
}

// GetBonusManager returns the bonus manager
func (chain *FullBlockChain) GetBonusManager() *BonusManager {
	return chain.bonusManager
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
