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
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
	time2 "github.com/zvchain/zvchain/middleware/time"

	"github.com/zvchain/zvchain/monitor"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

type batchAddBlockCallback func(b *types.Block, ret types.AddBlockResult) bool

type executePostState struct {
	state      *account.AccountDB
	receipts   types.Receipts
	evictedTxs []common.Hash
	txs        txSlice
	ts         *common.TimeStatCtx
}

// CastBlock cast a block, current casters synchronization operation in the group
func (chain *FullBlockChain) CastBlock(height uint64, proveValue []byte, qn uint64, castor []byte, gSeed common.Hash) *types.Block {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	block := new(types.Block)

	traceLog := monitor.NewPerformTraceLogger("CastBlock", common.Hash{}, height)
	traceLog.SetParent("blockProposal")
	defer func() {
		traceLog.Log("txs size %v", len(block.Transactions))
	}()

	latestBlock := chain.QueryTopBlock()
	if latestBlock == nil {
		Logger.Info("[BlockChain] fail to cast block: lastest block is nil")
		return nil
	}

	if height <= latestBlock.Height {
		Logger.Infof("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	begin := time.Now()

	defer func() {
		Logger.Debugf("cast block, height=%v, hash=%v, cost %v", block.Header.Height, block.Header.Hash.Hex(), time.Since(begin).String())
	}()

	block.Header = &types.BlockHeader{
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		Group:      gSeed,
		TotalQN:    latestBlock.TotalQN + qn,
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		PreHash:    latestBlock.Hash,
		Nonce:      common.ChainDataVersion,
	}

	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())

	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new stateDb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d error:", block.Header.Height))
		buffer.WriteString(fmt.Sprint(err))

		Logger.Error(buffer.String())
		return nil
	}

	Logger.Debugf("casting block height=%v,preHash=%v", height, preRoot)
	//taslog.Flush()

	packTraceLog := monitor.NewPerformTraceLogger("PackForCast", common.Hash{}, height)
	packTraceLog.SetParent("CastBlock")
	txs := chain.transactionPool.PackForCast()
	packTraceLog.SetEnd()
	defer packTraceLog.Log("")

	exeTraceLog := monitor.NewPerformTraceLogger("process", common.Hash{}, height)
	exeTraceLog.SetParent("CastBlock")
	defer exeTraceLog.Log("pack=true")
	block.Header.CurTime = chain.ts.Now()
	stateRoot, evictHashs, txSlice, receipts, gasFee, err := chain.stateProc.process(state, block.Header, txs, true, nil)
	exeTraceLog.SetEnd()

	block.Transactions = txSlice.txsToRaw()
	block.Header.GasFee = gasFee
	block.Header.TxTree = txSlice.calcTxTree()

	block.Header.StateTree = common.BytesToHash(stateRoot.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)

	elapsed := block.Header.CurTime.SinceMilliSeconds(latestBlock.CurTime)
	block.Header.Elapsed = int32(elapsed)
	if block.Header.Height == 1 && int64(block.Header.Elapsed) != elapsed {
		block.Header.Elapsed = math.MaxInt32 //overflow, may happen in first block
	}
	minElapse := chain.consensusHelper.GetBlockMinElapse(block.Header.Height)
	if block.Header.Elapsed < minElapse {
		block.Header.CurTime = latestBlock.CurTime.AddMilliSeconds(int64(minElapse))
		block.Header.Elapsed = minElapse
	}

	block.Header.Hash = block.Header.GenHash()

	traceLog.SetHash(block.Header.Hash)
	packTraceLog.SetHash(block.Header.Hash)
	exeTraceLog.SetHash(block.Header.Hash)

	// Blocks that you cast yourself do not need to be verified
	chain.verifiedBlocks.Add(block.Header.Hash, &executePostState{
		state:      state,
		receipts:   receipts,
		evictedTxs: evictHashs,
		txs:        txSlice,
	})
	return block
}

func (chain *FullBlockChain) verifyTxs(block *types.Block) (txs txSlice, ok bool) {
	begin := time.Now()
	bh := block.Header
	traceLog := monitor.NewPerformTraceLogger("verifyTxs", bh.Hash, bh.Height)
	traceLog.SetParent("addBlockOnChain")

	var err error
	defer func() {
		Logger.Debugf("verifyTxs hash:%v,height:%d,totalQn:%d,preHash:%v,len tx:%d, cost:%v, err=%v", bh.Hash.Hex(), bh.Height, bh.TotalQN, bh.PreHash.Hex(), len(block.Transactions), time.Since(begin).String(), err)
		traceLog.Log("err=%v", err)
	}()

	return chain.validateTxs(block)
}

// AddBlockOnChain add a block on blockchain, there are five cases of return value：
//
// 		0, successfully add block on blockchain
// 		-1, verification failed
//		1, the block already exist on the blockchain, then we should discard it
// 		2, the same height block with a larger QN value on the chain, then we should discard it
// 		3, need adjust the blockchain, there will be a fork
func (chain *FullBlockChain) AddBlockOnChain(source string, b *types.Block) types.AddBlockResult {
	ret, _ := chain.addBlockOnChain(source, b)
	if ret == types.AddBlockSucc {
		log.ELKLogger.WithFields(logrus.Fields{
			"blockHash": b.Header.Hash.Hex(),
			"caster":    common.BytesToAddress(b.Header.Castor).AddrPrefixString(),
			"height":    b.Header.Height,
			"logType":   "doAddOnChain",
			"now":       time2.TSInstance.Now().UTC(),
			"version":   common.GzvVersion,
		}).Info("doAddOnChain success")
	}
	return ret
}

func (chain *FullBlockChain) consensusVerifyBlock(bh *types.BlockHeader) (bool, error) {
	if chain.Height() == 0 {
		return true, nil
	}
	pre := chain.queryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return false, errors.New("has no pre")
	}

	// Checks if pre block lower than latest checkpoint
	latestCP := chain.LatestCheckPoint()
	if pre.Height < latestCP.Height {
		return false, fmt.Errorf("pre block lower than latest cp: pre %v, cp %v, comming %v-%v", pre.Height, latestCP.Height, bh.Height, bh.Hash)
	}

	result, err := chain.GetConsensusHelper().VerifyNewBlock(bh, pre)
	if err != nil {
		Logger.Errorf("consensusVerifyBlock error:%s", err.Error())
		return false, err
	}
	return result, err
}

func (chain *FullBlockChain) processFutureBlock(b *types.Block, source string) {
	chain.futureRawBlocks.Add(b.Header.PreHash, b)
	if source == "" {
		return
	}
	eh := chain.GetConsensusHelper().EstimatePreHeight(b.Header)
	if eh <= chain.Height() { // If pre height is less than the current height, it is judged to be fork
		bh := b.Header
		top := chain.latestBlock
		Logger.Warnf("detect fork. hash=%v, height=%v, preHash=%v, topHash=%v, topHeight=%v, topPreHash=%v", bh.Hash.Hex(), bh.Height, bh.PreHash.Hex(), top.Hash.Hex(), top.Height, top.PreHash.Hex())
		go chain.forkProcessor.tryToProcessFork(source, b)
	} else { //
		go blockSync.syncFrom(source)
	}
}

func (chain *FullBlockChain) validateBlock(source string, b *types.Block) (bool, error) {
	if b == nil {
		return false, fmt.Errorf("block is nil")
	}
	traceLog := monitor.NewPerformTraceLogger("validateBlock", b.Header.Hash, b.Header.Height)
	traceLog.SetParent("addBlockOnChain")
	defer traceLog.Log("")

	if !chain.HasBlock(b.Header.PreHash) {
		chain.processFutureBlock(b, source)
		return false, ErrPreNotExist
	}

	if chain.compareChainWeight(b.Header) > 0 {
		return false, ErrLocalMoreWeight
	}

	blockSize := 0
	for _, v := range b.Transactions {
		blockSize += v.Size()
	}
	if blockSize > txAccumulateSizeMaxPerBlock {
		return false, ErrBlockSizeLimit
	}

	groupValidateResult, err := chain.consensusVerifyBlock(b.Header)
	if !groupValidateResult {
		return false, fmt.Errorf("consensus verify fail, err=%v", err.Error())
	}
	return true, nil
}

func (chain *FullBlockChain) resetBlockHeight() error {
	trieGc := common.GlobalConf.GetBool(configSec, "gcmode", GcMode)
	if !trieGc {
		return nil
	}
	latestBH := chain.loadCurrentBlock()
	if latestBH == nil {
		return nil
	}
	curHeight := dirtyState.GetCurrentHeight()
	if curHeight < latestBH.Height {
		err := dirtyState.StoreCurHeight(latestBH.Height)
		if err != nil {
			return err
		}
	}
	stateHeight := dirtyState.GetLastTrieHeight()
	if stateHeight < latestBH.Height {
		hash := chain.queryBlockHash(stateHeight)
		err := chain.storeBlockHash(*hash)
		if err != nil {
			return fmt.Errorf("reset block hash error,height is %v", stateHeight)
		}
	}
	return nil
}

func (chain *FullBlockChain) Stop() {
	trieGc := common.GlobalConf.GetBool(configSec, "gcmode", GcMode)
	if !trieGc {
		return
	}
	if !atomic.CompareAndSwapInt32(&chain.running, 0, 1) {
		return
	}
	chain.wg.Wait()
	begin := time.Now()
	defer func() {
		fmt.Printf("stop success,cost %v", time.Since(begin))
	}()

	bh := chain.QueryTopBlock()
	if bh == nil {
		return
	}

	triedb := chain.stateCache.TrieDB()
	var closen uint64 = 0
	if bh.Height > TriesInMemory{
		closen = bh.Height - uint64(TriesInMemory)
	}
	var lastBlockHeight uint64 = common.MaxUint64
	fmt.Printf("stop process begin,local height is %v,closen is%v \n",bh.Height,closen)
	for !chain.triegc.Empty() {
		root, number := chain.triegc.Pop()
		if uint64(-number) < closen {
			continue
		}
		err := triedb.Commit(root.(common.Hash), true)
		if err != nil {
			fmt.Printf("stopping trie commit statedb error:%s", err.Error())
			break
		}else{
			lastBlockHeight = uint64(-number)
		}
		fmt.Printf("commit data height is %v \n",(-number))
	}
	if lastBlockHeight != common.MaxUint64{
		dirtyState.StoreTriePureHeight(lastBlockHeight)
	}
	os.Exit(0)
}

func (chain *FullBlockChain) DeleteDirtyTrie(persistenceHeight uint64) {
	lastDeleteHeight := dirtyState.GetLastDeleteDirtyTrieHeight()
	if persistenceHeight <= lastDeleteHeight+TriesInMemory+1 {
		return
	}
	beginHeight := lastDeleteHeight + 1
	endHeight := persistenceHeight - TriesInMemory
	begin := time.Now()
	defer func() {
		log.CorpLogger.Debugf("delete dirty trie from db success,height is %v-%v,cost=%v", beginHeight, endHeight, time.Since(begin))
	}()
	log.CorpLogger.Debugf("begin delete dirty trie from db,height is %v-%v", beginHeight, endHeight)
	for i := beginHeight; i < endHeight; i++ {
		bh := chain.queryBlockHeaderByHeight(i)
		if bh == nil {
			continue
		}
		err := dirtyState.DeleteDirtyTrie(bh.StateTree, bh.Height)
		if err != nil {
			log.CoreLogger.Error(err)
			break
		}
		chain.stateCache.TrieDB().DeleteByRoot(bh.StateTree)
	}
}

func (chain *FullBlockChain) FixTrieDataFromDB() error {
	trieGc := common.GlobalConf.GetBool(configSec, "gcmode", GcMode)
	if !trieGc {
		return nil
	}
	topHeight := dirtyState.GetCurrentHeight()
	block := chain.QueryBlockByHeight(topHeight)
	if block == nil {
		return nil
	}
	lastTrieHeight := dirtyState.GetLastTrieHeight()
	if lastTrieHeight < topHeight {
		end := lastTrieHeight
		var begin uint64 = 1
		if end > TriesInMemory {
			begin = end - uint64(TriesInMemory)
		}
		if end < begin {
			return nil
		}
		start := time.Now()
		defer func() {
			fmt.Printf("fix dirty state data success,from %v-%v,cost %v \n", begin, end, time.Since(start))
		}()
		fmt.Printf("begin fix dirty state data,from %v-%v \n", begin, end)
		triedb := chain.stateCache.TrieDB()

		for i := begin; i <= end; i++ {
			bh := chain.queryBlockHeaderByHeight(i)
			if bh == nil {
				continue
			}

			data := dirtyState.GetDirtyByRoot(bh.StateTree)
			if len(data) == 0 {
				log.CorpLogger.Debugf("get dirty state data nil,height is %v，root is %v", bh.Height, bh.StateTree.Hex())
				continue
			}
			err, caches := triedb.DecodeStoreBlob(data)
			if err != nil {
				return err
			}
			triedb.AddDirtyStateToCache(bh.StateTree,caches)
		}
	}
	return nil
}

func (chain *FullBlockChain) FixState() error {
	trieGc := common.GlobalConf.GetBool(configSec, "gcmode", GcMode)
	if !trieGc {
		return nil
	}
	ProcessFixState = true
	begin := time.Now()
	defer func() {
		ProcessFixState = false
		fmt.Printf("fix state cost %v \n", time.Since(begin))
	}()
	topHeight := dirtyState.GetCurrentHeight()
	block := chain.QueryBlockByHeight(topHeight)
	if block == nil {
		return fmt.Errorf("find block is nil,height is %v", topHeight)
	}
	lastTrieHeight := dirtyState.GetLastTrieHeight()
	fmt.Printf("begin fix State from %v - %v \n", lastTrieHeight, topHeight)
	for lastTrieHeight < topHeight {
		lastTrieHeight++
		curBlock := chain.QueryBlockByHeight(lastTrieHeight)
		if curBlock == nil {
			continue
		}
		trans := make([]*types.Transaction, 0)
		for _, tx := range curBlock.Transactions {
			trans = append(trans, types.NewTransaction(tx, tx.GenHash()))
		}
		success, ps := chain.executeTransaction(curBlock, trans)
		if !success {
			return fmt.Errorf("fixState execute tx failed,height is %v", lastTrieHeight)
		}
		err := chain.storePartBlock(curBlock, ps)
		if err != nil {
			return fmt.Errorf("fixState storePartBlock failed,err=%v", err)
		}
		notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: curBlock})
	}
	return nil
}

func (chain *FullBlockChain) addBlockOnChain(source string, block *types.Block) (ret types.AddBlockResult, err error) {
	begin := time.Now()

	traceLog := monitor.NewPerformTraceLogger("addBlockOnChain", block.Header.Hash, block.Header.Height)

	defer func() {
		traceLog.Log("ret=%v, err=%v", ret, err)
		Logger.Debugf("addBlockOnchain hash=%v, height=%v, err=%v, cost=%v", block.Header.Hash, block.Header.Height, err, time.Since(begin).String())
	}()

	if block == nil {
		return types.AddBlockFailed, fmt.Errorf("nil block")
	}
	bh := block.Header

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		err = fmt.Errorf("hash diff")
		return types.AddBlockFailed, err
	}

	topBlock := chain.getLatestBlock()
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d, Local tophash=%v, topPreHash=%v, height=%v,totalQn:%d", block.Header.Hash, block.Header.PreHash, block.Header.Height, block.Header.TotalQN, topBlock.Hash, topBlock.PreHash, topBlock.Height, topBlock.TotalQN)

	if chain.HasBlock(bh.Hash) {
		return types.AddBlockExisted, ErrBlockExist
	}
	if ok, e := chain.validateBlock(source, block); !ok {
		if e == ErrorBlockHash || e == ErrorGroupSign || e == ErrorRandomSign || e == ErrPkNotExists {
			ret = types.AddBlockConsensusFailed
		} else {
			ret = types.AddBlockFailed
		}
		err = e
		return
	}

	txSlice, ok := chain.verifyTxs(block)
	if !ok {
		Logger.Errorf("Fail to verifyTxs")
		ret = types.AddBlockFailed
		err = fmt.Errorf("verify block fail")
		return
	}

	chain.mu.Lock()
	defer chain.mu.Unlock()

	defer func() {
		if ret == types.AddBlockSucc {
			chain.addTopBlock(block)
			chain.successOnChainCallBack(block)
		}
	}()

	topBlock = chain.getLatestBlock()

	if chain.HasBlock(bh.Hash) {
		ret = types.AddBlockExisted
		err = ErrBlockExist
		return
	}

	if !chain.HasBlock(bh.PreHash) {
		chain.processFutureBlock(block, source)
		ret = types.AddBlockFailed
		err = ErrPreNotExist
		return
	}

	// Add directly to the blockchain
	if bh.PreHash == topBlock.Hash {
		ok, e := chain.transitAndCommit(block, txSlice)
		if ok {
			ret = types.AddBlockSucc
			return
		}
		Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash, bh.Height, e)
		ret = types.AddBlockFailed
		err = ErrCommitBlockFail
		return
	}

	cmpWeight := chain.compareChainWeight(bh)
	if cmpWeight > 0 { // the local block's weight is greater, then discard the new one
		ret = types.AddBlockLessWeightThanLocal
		err = ErrLocalMoreWeight
		return
	} else if cmpWeight == 0 {
		ret = types.AddBlockExisted
		err = ErrBlockExist
		return
	} else { // there is a fork
		newTop := chain.queryBlockHeaderByHash(bh.PreHash)
		old := chain.latestBlock
		Logger.Debugf("simple fork reset top: old %v %v %v %v, coming %v %v %v %v", old.Hash, old.Height, old.PreHash, old.TotalQN, bh.Hash, bh.Height, bh.PreHash, bh.TotalQN)
		if e := chain.resetTop(newTop); e != nil {
			Logger.Warnf("reset top err, currTop %v, setTop %v, setHeight %v", topBlock.Hash, newTop.Hash, newTop.Height)
			ret = types.AddBlockFailed
			err = fmt.Errorf("reset top err:%v", e)
			return
		}

		if chain.getLatestBlock().Hash != bh.PreHash {
			Logger.Error("reset top error")
			return
		}

		ok, e := chain.transitAndCommit(block, txSlice)
		if ok {
			ret = types.AddBlockSucc
			return
		}
		Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash, bh.Height, e)
		ret = types.AddBlockFailed
		err = ErrCommitBlockFail
		return
	}
}

func (chain *FullBlockChain) transitAndCommit(block *types.Block, tSlice txSlice) (ok bool, err error) {
	if len(block.Transactions) != len(tSlice) {
		return false, fmt.Errorf("txslice len not equal to block transactions:%v %v", len(block.Transactions), len(tSlice))
	}
	// Check if all txs exist in the pool to ensure the basic validation is done
	for _, tx := range tSlice {
		if exist, _ := chain.GetTransactionPool().IsTransactionExisted(tx.Hash); !exist {
			return false, fmt.Errorf("tx is not in the pool %v", tx.Hash)
		}
	}

	// process the transactions. Must be serialized execution
	executeTxResult, ps := chain.executeTransaction(block, tSlice)
	if !executeTxResult {
		err = fmt.Errorf("execute transaction fail")
		return
	}

	// Commit to DB
	return chain.commitBlock(block, ps)
}

// validateTxs check tx sign and recover source
func (chain *FullBlockChain) validateTxs(block *types.Block) (txSlice, bool) {
	bh := block.Header
	rawTxs := block.Transactions

	rets := make(txSlice, 0)
	if rawTxs == nil || len(rawTxs) == 0 {
		return rets, true
	}

	traceLog := monitor.NewPerformTraceLogger("validateTxs", bh.Hash, bh.Height)
	traceLog.SetParent("verifyTxs")
	defer traceLog.Log("size=%v", len(rawTxs))

	addTxs := make(txSlice, 0)
	for _, rawTx := range rawTxs {
		txHash := rawTx.GenHash()
		poolTx := chain.transactionPool.GetTransaction(rawTx.IsReward(), txHash)
		if poolTx != nil {
			if !bytes.Equal(rawTx.Sign, poolTx.Sign) {
				Logger.Debugf("fail to validate rawTxs: sign diff at %v, [%v %v]", txHash.Hex(), rawTx.Sign, poolTx.Sign)
				return nil, false
			}
			rets = append(rets, poolTx)
		} else {
			newTx := types.NewTransaction(rawTx, txHash)
			rets = append(rets, newTx)
			addTxs = append(addTxs, newTx)
		}
	}

	batchTraceLog := monitor.NewPerformTraceLogger("batchAdd", bh.Hash, bh.Height)
	batchTraceLog.SetParent("validateTxs")
	defer batchTraceLog.Log("size=%v", len(addTxs))

	err := chain.txBatch.batchAdd(addTxs)
	if err != nil {
		return nil, false
	}

	Logger.Debugf("block %v, validate rawTxs size %v, recover cnt %v", bh.Hash.Hex(), len(rawTxs), len(addTxs))
	return rets, true
}

func (chain *FullBlockChain) executeTransaction(block *types.Block, slice txSlice) (bool, *executePostState) {
	traceLog := monitor.NewPerformTraceLogger("executeTransaction", block.Header.Hash, block.Header.Height)
	traceLog.SetParent("commitBlock")
	defer traceLog.Log("size=%v", len(block.Transactions))

	cached, _ := chain.verifiedBlocks.Get(block.Header.Hash)
	if cached != nil {
		cb := cached.(*executePostState)
		return true, cb
	}
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		return false, nil
	}

	preRoot := preBlock.StateTree
	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("Fail to new stateDb, error:%s", err)
		return false, nil
	}

	stateTree, evictTxs, executedSlice, receipts, gasFee, err := chain.stateProc.process(state, block.Header, slice, false, nil)
	txTree := executedSlice.calcTxTree()
	if txTree != block.Header.TxTree {
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree, block.Header.TxTree)
		return false, nil
	}
	if len(executedSlice) != len(block.Transactions) {
		Logger.Errorf("Fail to verify executedSlice, length1: %d length2: %d", len(executedSlice), len(block.Transactions))
		return false, nil
	}
	if gasFee != block.Header.GasFee {
		Logger.Errorf("Fail to verify GasFee, fee1: %d, fee1: %d", gasFee, block.Header.GasFee)
		return false, nil
	}
	if stateTree != block.Header.StateTree {
		Logger.Errorf("Fail to verify state tree, execute fail, hash1:%s hash2:%s", stateTree.Hex(), block.Header.StateTree.Hex())
		return false, nil
	}
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != block.Header.ReceiptTree {
		Logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", receiptsTree.Hex(), block.Header.ReceiptTree.Hex())
		return false, nil
	}

	Logger.Infof("executeTransactions block height=%v,preHash=%v", block.Header.Height, preRoot)
	//taslog.Flush()

	eps := &executePostState{state: state, receipts: receipts, evictedTxs: evictTxs, txs: executedSlice}
	chain.verifiedBlocks.Add(block.Header.Hash, eps)
	return true, eps
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block) {
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: remoteBlock})
}

func (chain *FullBlockChain) onBlockAddSuccess(message notify.Message) error {
	b := message.GetData().(*types.Block)
	latestCP := chain.CheckPointAt(b.Header.Height)
	if latestCP != nil {
		Logger.Debugf("latest cp at %v is %v-%v", b.Header.Height, latestCP.Height, latestCP.Hash)
		chain.latestCP.Store(latestCP)
	}
	if ProcessFixState {
		return nil
	}
	if value, _ := chain.futureRawBlocks.Get(b.Header.Hash); value != nil {
		rawBlock := value.(*types.Block)
		Logger.Debugf("Get rawBlock from future blocks,hash:%s,height:%d", rawBlock.Header.Hash.Hex(), rawBlock.Header.Height)
		chain.addBlockOnChain("", rawBlock)
		chain.futureRawBlocks.Remove(b.Header.Hash)
	}
	log.ELKLogger.WithFields(logrus.Fields{
		"txNum":    chain.transactionPool.TxNum(),
		"queueNum": chain.transactionPool.TxQueueNum(),
		"now":      time2.TSInstance.Now().UTC(),
		"logType":  "txPoolLog",
		"version":  common.GzvVersion,
	}).Info("transaction pool log")
	return nil
}

func (chain *FullBlockChain) ensureBlocksChained(rawBlocks []*types.Block) bool {
	if len(rawBlocks) <= 1 {
		return true
	}
	for i := 1; i < len(rawBlocks); i++ {
		if rawBlocks[i].Header.PreHash != rawBlocks[i-1].Header.Hash {
			return false
		}
	}
	return true
}

func (chain *FullBlockChain) batchAddBlockOnChain(source string, canReset bool, rawBlocks []*types.Block, callback batchAddBlockCallback) error {
	if !chain.ensureBlocksChained(rawBlocks) {
		return fmt.Errorf("blocks not chained")
	}

	chain.batchMu.Lock()
	defer chain.batchMu.Unlock()

	localTop := chain.latestBlock

	var addBlocks []*types.Block
	for i, b := range rawBlocks {
		if !chain.hasBlock(b.Header.Hash) {
			addBlocks = rawBlocks[i:]
			break
		}
	}
	if addBlocks == nil || len(addBlocks) == 0 {
		return fmt.Errorf("nothing to add")
	}
	firstBlock := addBlocks[0]
	if firstBlock.Header.PreHash != localTop.Hash {
		lastBlock := addBlocks[len(addBlocks)-1]
		// cannot reset in the block sync situation, and start the fork process
		if !canReset {
			go chain.forkProcessor.tryToProcessFork(source, lastBlock)
			return fmt.Errorf("batchAdd fork found, local top %v %v, peer first %v %v", localTop.Hash, localTop.Height, firstBlock.Header.Hash, firstBlock.Header.Height)
		} else {
			pre := chain.QueryBlockHeaderByHash(firstBlock.Header.PreHash)
			if pre != nil {
				last := lastBlock.Header
				chain.ResetTop(pre)
				Logger.Debugf("batchAdd reset top:old %v %v %v, new %v %v %v, last %v %v %v", localTop.Hash, localTop.Height, localTop.TotalQN, pre.Hash, pre.Height, pre.TotalQN, last.Hash, last.Height, last.TotalQN)
			} else {
				// There will fork, we have to deal with it
				Logger.Debugf("batchAdd detect fork from %v: local %v %v, peer %v %v", source, localTop.Hash, localTop.Height, firstBlock.Header.Hash, firstBlock.Header.Height)
				go chain.forkProcessor.tryToProcessFork(source, lastBlock)
				return fmt.Errorf("batchAdd fork found, local top %v %v, peer first %v %v", localTop.Hash, localTop.Height, firstBlock.Header.Hash, firstBlock.Header.Height)
			}
		}
	}
	chain.isAdjusting = true
	defer func() {
		chain.isAdjusting = false
	}()

	for _, b := range addBlocks {
		ret := chain.AddBlockOnChain(source, b)
		if !callback(b, ret) {
			break
		}
	}
	return nil
}
