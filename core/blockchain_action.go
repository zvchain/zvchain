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
	"github.com/zvchain/zvchain/monitor"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/taslog"
)

type batchAddBlockCallback func(b *types.Block, ret types.AddBlockResult) bool

type executePostState struct {
	state      *account.AccountDB
	receipts   types.Receipts
	evictedTxs []common.Hash
	txs        []*types.Transaction
	ts         *common.TimeStatCtx
}

// CastBlock cast a block, current casters synchronization operation in the group
func (chain *FullBlockChain) CastBlock(height uint64, proveValue []byte, qn uint64, castor []byte, groupid []byte) *types.Block {
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
		Logger.Info("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
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
		GroupID:    groupid,
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

	Logger.Infof("casting block height=%v,preHash=%x", height, preRoot)
	taslog.Flush()

	packTraceLog := monitor.NewPerformTraceLogger("PackForCast", common.Hash{}, height)
	packTraceLog.SetParent("CastBlock")
	txs := chain.transactionPool.PackForCast()
	packTraceLog.SetEnd()
	defer packTraceLog.Log("")

	exeTraceLog := monitor.NewPerformTraceLogger("Execute", common.Hash{}, height)
	exeTraceLog.SetParent("CastBlock")
	defer exeTraceLog.Log("pack=true")
	statehash, evitTxs, transactions, receipts, gasFee, err := chain.executor.Execute(state, block.Header, txs, true, nil)
	exeTraceLog.SetEnd()

	block.Transactions = transactions
	block.Header.GasFee = gasFee
	block.Header.TxTree = calcTxTree(block.Transactions)

	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)

	// Curtime setting after txs executed. More accuracy
	block.Header.CurTime = chain.ts.Now()
	block.Header.Elapsed = int32(block.Header.CurTime.Since(latestBlock.CurTime))
	if block.Header.Elapsed < 0 {
		Logger.Errorf("cur time is before pre time:height=%v, curtime=%v, pretime=%v", height, block.Header.CurTime, latestBlock.CurTime)
		return nil
	}

	block.Header.Hash = block.Header.GenHash()

	traceLog.SetHash(block.Header.Hash)
	packTraceLog.SetHash(block.Header.Hash)
	exeTraceLog.SetHash(block.Header.Hash)

	// Blocks that you cast yourself do not need to be verified
	chain.verifiedBlocks.Add(block.Header.Hash, &executePostState{
		state:      state,
		receipts:   receipts,
		evictedTxs: evitTxs,
		txs:        block.Transactions,
	})
	return block
}

func (chain *FullBlockChain) verifyTxs(bh *types.BlockHeader, txs []*types.Transaction) (ret int8) {
	begin := time.Now()

	traceLog := monitor.NewPerformTraceLogger("verifyTxs", bh.Hash, bh.Height)
	traceLog.SetParent("addBlockOnChain")

	var err error
	defer func() {
		Logger.Infof("verifyTxs hash:%v,height:%d,totalQn:%d,preHash:%v,len tx:%d, cost:%v, err=%v", bh.Hash.Hex(), bh.Height, bh.TotalQN, bh.PreHash.Hex(), len(txs), time.Since(begin).String(), err)
		traceLog.Log("err=%v", err)
	}()

	if !chain.validateTxs(bh, txs) {
		return -1
	}

	return 0
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
	return ret
}

func (chain *FullBlockChain) consensusVerifyBlock(bh *types.BlockHeader) (bool, error) {
	if chain.Height() == 0 {
		return true, nil
	}
	pre := chain.queryBlockByHash(bh.PreHash)
	if pre == nil {
		return false, errors.New("has no pre")
	}
	result, err := chain.GetConsensusHelper().VerifyNewBlock(bh, pre.Header)
	if err != nil {
		Logger.Errorf("consensusVerifyBlock error:%s", err.Error())
		return false, err
	}
	return result, err
}

func (chain *FullBlockChain) processFutureBlock(b *types.Block, source string) {
	chain.futureBlocks.Add(b.Header.PreHash, b)
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

	groupValidateResult, err := chain.consensusVerifyBlock(b.Header)
	if !groupValidateResult {
		if err == common.ErrSelectGroupNil || err == common.ErrSelectGroupInequal {
			Logger.Infof("Add block on chain failed: depend on group! trigger group sync")
			if groupSync != nil {
				go groupSync.trySyncRoutine()
			}
		} else {
			Logger.Errorf("Fail to validate group sig!Err:%s", err.Error())
		}
		return false, fmt.Errorf("consensus verify fail, err=%v", err.Error())
	}
	return true, nil
}

func (chain *FullBlockChain) addBlockOnChain(source string, b *types.Block) (ret types.AddBlockResult, err error) {
	begin := time.Now()

	traceLog := monitor.NewPerformTraceLogger("addBlockOnChain", b.Header.Hash, b.Header.Height)

	defer func() {
		traceLog.Log("ret=%v, err=%v", ret, err)
		Logger.Debugf("addBlockOnchain hash=%v, height=%v, err=%v, cost=%v", b.Header.Hash.Hex(), b.Header.Height, err, time.Since(begin).String())
	}()

	if b == nil {
		return types.AddBlockFailed, fmt.Errorf("nil block")
	}
	bh := b.Header

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		err = fmt.Errorf("hash diff")
		return types.AddBlockFailed, err
	}

	topBlock := chain.getLatestBlock()
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d, Local tophash=%v, topPreHash=%v, height=%v,totalQn:%d", b.Header.Hash.ShortS(), b.Header.PreHash.ShortS(), b.Header.Height, b.Header.TotalQN, topBlock.Hash.ShortS(), topBlock.PreHash.ShortS(), topBlock.Height, topBlock.TotalQN)

	if chain.HasBlock(bh.Hash) {
		return types.BlockExisted, ErrBlockExist
	}
	if ok, e := chain.validateBlock(source, b); !ok {
		ret = types.AddBlockFailed
		err = e
		return
	}

	verifyResult := chain.verifyTxs(bh, b.Transactions)
	if verifyResult != 0 {
		Logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		ret = types.AddBlockFailed
		err = fmt.Errorf("verify block fail")
		return
	}

	chain.mu.Lock()
	defer chain.mu.Unlock()

	defer func() {
		if ret == types.AddBlockSucc {
			chain.addTopBlock(b)
			chain.successOnChainCallBack(b)
		}
	}()

	topBlock = chain.getLatestBlock()

	if chain.HasBlock(bh.Hash) {
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	}

	if !chain.HasBlock(bh.PreHash) {
		chain.processFutureBlock(b, source)
		ret = types.AddBlockFailed
		err = ErrPreNotExist
		return
	}

	// Add directly to the blockchain
	if bh.PreHash == topBlock.Hash {
		ok, e := chain.transitAndCommit(b)
		if ok {
			ret = types.AddBlockSucc
			return
		}
		Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.Hex(), bh.Height, e)
		ret = types.AddBlockFailed
		err = ErrCommitBlockFail
		return
	}

	cmpWeight := chain.compareChainWeight(bh)
	if cmpWeight > 0 { // the local block's weight is greater, then discard the new one
		ret = types.BlockTotalQnLessThanLocal
		err = ErrLocalMoreWeight
		return
	} else if cmpWeight == 0 {
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	} else { // there is a fork
		newTop := chain.queryBlockHeaderByHash(bh.PreHash)
		old := chain.latestBlock
		Logger.Debugf("simple fork reset top: old %v %v %v %v, coming %v %v %v %v", old.Hash.ShortS(), old.Height, old.PreHash.ShortS(), old.TotalQN, bh.Hash.ShortS(), bh.Height, bh.PreHash.ShortS(), bh.TotalQN)
		if e := chain.resetTop(newTop); e != nil {
			Logger.Warnf("reset top err, currTop %v, setTop %v, setHeight %v", topBlock.Hash.Hex(), newTop.Hash.Hex(), newTop.Height)
			ret = types.AddBlockFailed
			err = fmt.Errorf("reset top err:%v", e)
			return
		}

		if chain.getLatestBlock().Hash != bh.PreHash {
			Logger.Error("reset top error")
			return
		}

		ok, e := chain.transitAndCommit(b)
		if ok {
			ret = types.AddBlockSucc
			return
		}
		Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.Hex(), bh.Height, e)
		ret = types.AddBlockFailed
		err = ErrCommitBlockFail
		return
	}
}

func (chain *FullBlockChain) transitAndCommit(block *types.Block) (ok bool, err error) {
	// Execute the transactions. Must be serialized execution
	executeTxResult, ps := chain.executeTransaction(block)
	if !executeTxResult {
		err = fmt.Errorf("execute transaction fail")
		return
	}

	// Commit to DB
	return chain.commitBlock(block, ps)
}

// validateTxs check tx sign and recover source
func (chain *FullBlockChain) validateTxs(bh *types.BlockHeader, txs []*types.Transaction) bool {
	if txs == nil || len(txs) == 0 {
		return true
	}

	traceLog := monitor.NewPerformTraceLogger("validateTxs", bh.Hash, bh.Height)
	traceLog.SetParent("verifyTxs")
	defer traceLog.Log("size=%v", len(txs))

	addTxs := make([]*types.Transaction, 0)
	for _, tx := range txs {
		if tx.Source != nil {
			continue
		}
		poolTx := chain.transactionPool.GetTransaction(tx.Type == types.TransactionTypeReward, tx.Hash)
		if poolTx != nil {
			if tx.Hash != tx.GenHash() {
				Logger.Debugf("fail to validate txs: hash diff at %v, expect hash %v", tx.Hash.Hex(), tx.GenHash().Hex())
				return false
			}
			if !bytes.Equal(tx.Sign, poolTx.Sign) {
				Logger.Debugf("fail to validate txs: sign diff at %v, [%v %v]", tx.Hash.Hex(), tx.Sign, poolTx.Sign)
				return false
			}
			tx.Source = poolTx.Source
		} else {
			addTxs = append(addTxs, tx)
			//recoverCnt++
			//TxSyncer.add(tx)
			//if err := chain.transactionPool.RecoverAndValidateTx(tx); err != nil {
			//	Logger.Debugf("fail to validate txs RecoverAndValidateTx err:%v at %v", err, tx.Hash.String())
			//	return false
			//}
		}
	}

	batchTraceLog := monitor.NewPerformTraceLogger("batchAdd", bh.Hash, bh.Height)
	batchTraceLog.SetParent("validateTxs")
	defer batchTraceLog.Log("size=%v", len(addTxs))
	chain.txBatch.batchAdd(addTxs)
	for _, tx := range addTxs {
		if err := tx.RecoverSource(); err != nil {
			Logger.Errorf("tx source recover fail:%s", tx.Hash.Hex())
			return false
		}
	}

	Logger.Debugf("block %v, validate txs size %v, recover cnt %v", bh.Hash.Hex(), len(txs), len(addTxs))
	return true
}

func (chain *FullBlockChain) executeTransaction(block *types.Block) (bool, *executePostState) {
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

	statehash, evitTxs, transactions, receipts, gasFee, err := chain.executor.Execute(state, block.Header, block.Transactions, false, nil)
	txTree := calcTxTree(transactions)
	if txTree != block.Header.TxTree{
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree, block.Header.TxTree)
		return false, nil
	}
	if gasFee != block.Header.GasFee {
		Logger.Errorf("Fail to verify GasFee, fee1: %d, fee1: %d", gasFee, block.Header.GasFee)
		return false, nil
	}
	if statehash != block.Header.StateTree {
		Logger.Errorf("Fail to verify statetrexecute transaction failee, hash1:%s hash2:%s", statehash.Hex(), block.Header.StateTree.Hex())
		return false, nil
	}
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != block.Header.ReceiptTree {
		Logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", receiptsTree.Hex(), block.Header.ReceiptTree.Hex())
		return false, nil
	}

	Logger.Infof("executeTransactions block height=%v,preHash=%x", block.Header.Height, preRoot)
	taslog.Flush()

	eps := &executePostState{state: state, receipts: receipts, evictedTxs: evitTxs, txs: block.Transactions}
	chain.verifiedBlocks.Add(block.Header.Hash, eps)
	return true, eps
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block) {
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: remoteBlock})
}

func (chain *FullBlockChain) onBlockAddSuccess(message notify.Message) {
	b := message.GetData().(*types.Block)
	if value, _ := chain.futureBlocks.Get(b.Header.Hash); value != nil {
		block := value.(*types.Block)
		Logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.Hex(), block.Header.Height)
		chain.addBlockOnChain("", block)
		chain.futureBlocks.Remove(b.Header.Hash)
		return
	}
}

func (chain *FullBlockChain) ensureBlocksChained(blocks []*types.Block) bool {
	if len(blocks) <= 1 {
		return true
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Header.PreHash != blocks[i-1].Header.Hash {
			return false
		}
	}
	return true
}

func (chain *FullBlockChain) batchAddBlockOnChain(source string, module string, blocks []*types.Block, callback batchAddBlockCallback) {
	if !chain.ensureBlocksChained(blocks) {
		Logger.Errorf("%v blocks not chained! size %v", module, len(blocks))
		return
	}
	// First pre-recovery transaction source
	for _, b := range blocks {
		if b.Transactions != nil && len(b.Transactions) > 0 {
			go chain.transactionPool.AsyncAddTxs(b.Transactions)
		}
	}

	chain.batchMu.Lock()
	defer chain.batchMu.Unlock()

	localTop := chain.latestBlock

	var addBlocks []*types.Block
	for i, b := range blocks {
		if !chain.hasBlock(b.Header.Hash) {
			addBlocks = blocks[i:]
			break
		}
	}
	if addBlocks == nil || len(addBlocks) == 0 {
		return
	}
	firstBH := addBlocks[0]
	if firstBH.Header.PreHash != localTop.Hash {
		pre := chain.QueryBlockHeaderByHash(firstBH.Header.PreHash)
		if pre != nil {
			last := addBlocks[len(addBlocks)-1].Header
			Logger.Debugf("%v batchAdd reset top:old %v %v %v, new %v %v %v, last %v %v %v", module, localTop.Hash.ShortS(), localTop.Height, localTop.TotalQN, pre.Hash.ShortS(), pre.Height, pre.TotalQN, last.Hash.ShortS(), last.Height, last.TotalQN)
			chain.ResetTop(pre)
		} else {
			// There will fork, we have to deal with it
			Logger.Debugf("%v batchAdd detect fork from %v: local %v %v, peer %v %v", module, source, localTop.Hash.ShortS(), localTop.Height, firstBH.Header.Hash.ShortS(), firstBH.Header.Height)
			go chain.forkProcessor.tryToProcessFork(source, firstBH)
			return
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
}
