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
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/ticker"
	zvtime "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
)

const (
	sendLocalTopInterval       = 3   // Interval of sending local top block to neighbor
	syncNeightborsInterval     = 3   // Interval of requesting synchronize block from neighbor
	syncNeightborTimeout       = 5   // Timeout of requesting synchronize block from neighbor
	blockSyncCandidatePoolSize = 100 // Size of candidate peer pool for block synchronize
)

const (
	tickerSendLocalTop = "send_local_top"
	tickerSyncNeighbor = "sync_neightbor"
	tickerSyncTimeout  = "sync_timeout"
)

var (
	ErrorBlockHash  = errors.New("consensus verify fail,block hash error")
	ErrorGroupSign  = errors.New("consensus verify fail,group sign signature error")
	ErrorRandomSign = errors.New("consensus verify fail,random signature error")
	ErrPkNotExists  = errors.New("consensus verify fail,pk not exists")

	ErrPkNil          = errors.New("pk is nil")
	ErrGroupNotExists = errors.New("group not exists")
)

var blockSync *blockSyncer

type blockSyncer struct {
	chain *FullBlockChain

	candidatePool map[string]*types.CandidateBlockHeader
	syncingPeers  map[string]uint64

	ticker *ticker.GlobalTicker

	lock   sync.RWMutex
	logger taslog.Logger
}

type topBlockInfo struct {
	types.BlockWeight
	Height uint64
}

func newTopBlockInfo(topBH *types.BlockHeader) *topBlockInfo {
	return &topBlockInfo{
		BlockWeight: *types.NewBlockWeight(topBH),
		Height:      topBH.Height,
	}
}

func newBlockSyncer(chain *FullBlockChain) *blockSyncer {
	return &blockSyncer{
		candidatePool: make(map[string]*types.CandidateBlockHeader),
		chain:         chain,
		syncingPeers:  make(map[string]uint64),
	}
}

// InitBlockSyncer initialize the blockSyncer. Register the ticker for sending and requesting blocks to neighbors timely
// and also subscribe these events to handle requests from neighbors
func InitBlockSyncer(chain *FullBlockChain) {
	blockSync = newBlockSyncer(chain)
	blockSync.ticker = blockSync.chain.ticker
	blockSync.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	blockSync.ticker.RegisterPeriodicRoutine(tickerSendLocalTop, blockSync.notifyLocalTopBlockRoutine, sendLocalTopInterval)
	blockSync.ticker.StartTickerRoutine(tickerSendLocalTop, false)

	blockSync.ticker.RegisterPeriodicRoutine(tickerSyncNeighbor, blockSync.trySyncRoutine, syncNeightborsInterval)
	blockSync.ticker.StartTickerRoutine(tickerSyncNeighbor, false)

	notify.BUS.Subscribe(notify.BlockInfoNotify, blockSync.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockReq, blockSync.blockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, blockSync.blockResponseMsgHandler)

}

func (bs *blockSyncer) isSyncing() bool {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	delta := zvtime.TSInstance.Since(bs.chain.QueryTopBlock().CurTime)
	// return false if top block's curTime is in the range of recent 50 block's
	if delta < 3*50 {
		return false
	}
	localHeight := bs.chain.Height()
	_, candTop := bs.getCandidateById("")
	if candTop == nil {
		return false
	}
	return candTop.BH.Height > localHeight+50
}

// get blockheader by candidateID,if this candidateID is evil,then remove it from candidatePool
func (bs *blockSyncer) getCandidateByCandidateID(candidateID string) *types.CandidateBlockHeader {
	isEvil := bs.checkEvilAndDelete(candidateID)
	if isEvil {
		return nil
	}
	maxTop := bs.candidatePool[candidateID]
	if maxTop == nil {
		return nil
	}
	ok := bs.checkBlockHeaderAndAddBlack(candidateID, maxTop.BH)
	if !ok {
		return nil
	}
	return maxTop
}

// get max headerBlock from adjacent nodes,if this node is evil,then remove it from candidatePool add black,and choose again
func (bs *blockSyncer) getCandidate() (string, *types.CandidateBlockHeader) {
	for id := range bs.candidatePool {
		bs.checkEvilAndDelete(id)
	}
	if len(bs.candidatePool) == 0 {
		bs.logger.Debugf("candidatePool length is 0")
		return "", nil
	}
	var maxWeightBlock *types.CandidateBlockHeader

	var currentCandidateId string
	for id, top := range bs.candidatePool {
		if maxWeightBlock == nil || top.BW.MoreWeight(maxWeightBlock.BW) {
			maxWeightBlock = top
			currentCandidateId = id
		}
	}
	if maxWeightBlock == nil {
		return "", nil
	}
	ok := bs.checkBlockHeaderAndAddBlack(currentCandidateId, maxWeightBlock.BH)
	if !ok {
		return bs.getCandidate()
	}
	return currentCandidateId, maxWeightBlock
}

func (bs *blockSyncer) checkEvilAndDelete(candidateID string) bool {
	if peerManagerImpl.isEvil(candidateID) {
		bs.logger.Debugf("peer meter evil id:%+v", peerManagerImpl.getOrAddPeer(candidateID))
		delete(bs.candidatePool, candidateID)
		return true
	}
	return false
}

func (bs *blockSyncer) checkBlockHeaderAndAddBlack(candidateID string, bh *types.BlockHeader) bool {
	_, err := BlockChainImpl.GetConsensusHelper().VerifyBlockHeader(bh)
	if err != nil && (err != ErrGroupNotExists && err != ErrPkNil) {
		bs.addBlack(candidateID)
		return false
	}
	return true
}

func (bs *blockSyncer)addBlack(candidateID string){
	delete(bs.candidatePool, candidateID)
	peerManagerImpl.addEvilCount(candidateID)
	bs.logger.Debugf("getBestCandidate verify blockHeader error!we will add it to evil,peer is %v", candidateID)
}


func (bs *blockSyncer) getCandidateById(candidateID string) (string, *types.CandidateBlockHeader) {
	if candidateID == "" {
		return bs.getCandidate()
	} else {
		bh := bs.getCandidateByCandidateID(candidateID)
		if bh == nil {
			return "", bs.getCandidateByCandidateID(candidateID)
		}
		return candidateID, bh
	}
}

func (bs *blockSyncer) getPeerTopBlock(id string) *types.CandidateBlockHeader {
	bs.lock.RLock()
	defer bs.lock.RUnlock()
	tb, ok := bs.candidatePool[id]
	if ok {
		return tb
	}
	return nil
}
func (bs *blockSyncer) trySyncRoutine() bool {
	return bs.syncFrom("")
}

func (bs *blockSyncer) syncFrom(from string) bool {
	if bs == nil {
		return false
	}
	topBH := bs.chain.QueryTopBlock()
	localTopBlock := newTopBlockInfo(topBH)

	if bs.chain.IsAdjusting() {
		bs.logger.Debugf("chain is adjusting, won't sync")
		return false
	}
	bs.logger.Debugf("Local Weight:%v, height:%d,topHash:%s", localTopBlock.BlockWeight.String(), localTopBlock.Height, localTopBlock.Hash.Hex())

	bs.lock.Lock()
	defer bs.lock.Unlock()

	candidate, candidateTop := bs.getCandidateById(from)
	if candidate == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		return false
	}
	bs.logger.Debugf("candidate info: id %v, top %v %v %v", candidate, candidateTop.BH.Hash.Hex(), candidateTop.BH.Height, candidateTop.BH.TotalQN)

	if localTopBlock.MoreWeight(candidateTop.BW) {
		bs.logger.Debugf("local top more weight: local:%v %v %v, candidate: %v %v %v", localTopBlock.Height, localTopBlock.Hash.Hex(), localTopBlock.BlockWeight, candidateTop.BH.Height, candidateTop.BH.Hash.Hex(), candidateTop.BW)
		return false
	}
	if bs.chain.HasBlock(candidateTop.BH.Hash) {
		bs.logger.Debugf("local has block %v, won't sync", candidateTop.BH.Hash.Hex())
		return false
	}
	beginHeight := uint64(0)
	localHeight := bs.chain.Height()
	if candidateTop.BH.Height <= localHeight {
		beginHeight = candidateTop.BH.Height
	} else {
		beginHeight = localHeight + 1
	}

	bs.logger.Debugf("beginHeight %v, candidateHeight %v", beginHeight, candidateTop.BH.Height)
	if beginHeight > candidateTop.BH.Height {
		return false
	}

	for syncID, h := range bs.syncingPeers {
		if h == beginHeight {
			bs.logger.Debugf("height %v in syncing from %v", beginHeight, syncID)
			return false
		}
	}

	candInfo := &SyncCandidateInfo{
		Candidate:       candidate,
		CandidateHeight: candidateTop.BH.Height,
		ReqHeight:       beginHeight,
	}

	notify.BUS.Publish(notify.BlockSync, &syncMessage{CandidateInfo: candInfo})

	bs.requestBlock(candInfo)
	return true
}

func (bs *blockSyncer) requestBlock(ci *SyncCandidateInfo) {
	id := ci.Candidate
	height := ci.ReqHeight
	if _, ok := bs.syncingPeers[id]; ok {
		return
	}

	bs.logger.Debugf("Req block to:%s,height:%d", id, height)

	br := &syncRequest{
		ReqHeight: height,
		ReqSize:   int32(peerManagerImpl.getPeerReqBlockCount(id)),
	}

	body, err := marshalSyncRequest(br)
	if err != nil {
		bs.logger.Errorf("marshalSyncRequest error %v", err)
		return
	}

	message := network.Message{Code: network.ReqBlock, Body: body}
	network.GetNetInstance().Send(id, message)

	bs.syncingPeers[id] = ci.ReqHeight

	bs.chain.ticker.RegisterOneTimeRoutine(bs.syncTimeoutRoutineName(id), func() bool {
		return bs.syncComplete(id, true)
	}, syncNeightborTimeout)
}

func (bs *blockSyncer) notifyLocalTopBlockRoutine() bool {
	top := bs.chain.QueryTopBlock()
	if top.Height == 0 {
		return false
	}
	bs.logger.Debugf("Send local %d,%v to neighbor!", top.TotalQN, top.Hash.Hex())
	body, e := marshalTopBlockInfo(top)
	if e != nil {
		bs.logger.Errorf("marshal blockInfo error:%s", e.Error())
		return false
	}
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
	return true
}

func (bs *blockSyncer) topBlockInfoNotifyHandler(msg notify.Message) {
	bnm := notify.AsDefault(msg)
	if peerManagerImpl.getOrAddPeer(bnm.Source()).isEvil() {
		bs.logger.Warnf("block sync this source is is in evil...source is is %v\n", bnm.Source())
		return
	}
	blockHeader, e := bs.unMarshalTopBlockInfo(bnm.Body())
	if e != nil {
		bs.logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	source := bnm.Source()
	peerManagerImpl.heardFromPeer(source)

	bs.addCandidatePool(source, blockHeader)
}

func (bs *blockSyncer) syncTimeoutRoutineName(id string) string {
	return tickerSyncTimeout + id
}

func (bs *blockSyncer) syncComplete(id string, timeout bool) bool {
	if timeout {
		peerManagerImpl.timeoutPeer(id)
		bs.logger.Warnf("sync block from %v timeout", id)
	} else {
		peerManagerImpl.heardFromPeer(id)
	}
	peerManagerImpl.updateReqBlockCnt(id, !timeout)
	bs.chain.ticker.RemoveRoutine(bs.syncTimeoutRoutineName(id))

	bs.lock.Lock()
	defer bs.lock.Unlock()
	delete(bs.syncingPeers, id)
	return true
}

func (bs *blockSyncer) blockResponseMsgHandler(msg notify.Message) {
	m := notify.AsDefault(msg)
	source := m.Source()
	if bs == nil {
		//do nothing
		return
	}
	var complete = false
	defer func() {
		if !complete {
			bs.syncComplete(source, false)
		}
	}()

	blockResponse, e := bs.unMarshalBlockMsgResponse(m.Body())
	if e != nil {
		bs.logger.Warnf("Discard block response msg because unMarshalBlockMsgResponse error:%d", e.Error())
		return
	}

	blocks := blockResponse.Blocks

	if blocks == nil || len(blocks) == 0 {
		bs.logger.Debugf("Rcv block response nil from:%s", source)
	} else {
		bs.logger.Debugf("blockResponseMsgHandler rcv from %s! [%v-%v]", source, blocks[0].Header.Height, blocks[len(blocks)-1].Header.Height)
		peerTop := bs.getPeerTopBlock(source)
		localTop := newTopBlockInfo(bs.chain.QueryTopBlock())

		// First compare weights
		if peerTop != nil && localTop.MoreWeight(peerTop.BW) {
			bs.logger.Debugf("sync block from %v, local top hash %v, height %v, totalQN %v, peerTop hash %v, height %v, totalQN %v", localTop.Hash.Hex(), localTop.Height, localTop.TotalQN, peerTop.BH.Hash.Hex(), peerTop.BH.Height, peerTop.BH.TotalQN)
			return
		}

		allSuccess := true
		hasAddBlack := false
		bs.chain.batchAddBlockOnChain(source, "sync", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
			bs.logger.Debugf("sync block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.Hex(), b.Header.Height, ret)
			if ret == types.AddBlockSucc || ret == types.BlockExisted {
				return true
			}
			if ret == types.AddBlockConsensusFailed && !hasAddBlack{
				hasAddBlack = true
				bs.addBlack(m.Source())
			}
			allSuccess = false
			return false
		})

		// The weight is still low, continue to synchronize (must add blocks
		// is successful, otherwise it will cause an infinite loop)
		if allSuccess && peerTop != nil && peerTop.BW.MoreWeight(&localTop.BlockWeight) {
			bs.syncComplete(source, false)
			complete = true
			go bs.trySyncRoutine()
		}
	}
}

func (bs *blockSyncer) addCandidatePool(source string, header *types.BlockHeader) {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	cbh := types.NewCandidateBlockHeader(header)
	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[source] = cbh
		return
	}
	for id, tbi := range bs.candidatePool {
		if cbh.BW.MoreWeight(tbi.BW) {
			delete(bs.candidatePool, id)
			bs.candidatePool[source] = cbh
		}
	}
}

func (bs *blockSyncer) blockReqHandler(msg notify.Message) {
	m := notify.AsDefault(msg)

	br, err := unmarshalSyncRequest(m.Body())
	if err != nil {
		bs.logger.Errorf("unmarshalSyncRequest error %v", err)
		return
	}
	localHeight := bs.chain.Height()
	if br.ReqHeight <= 0 || br.ReqHeight > localHeight || br.ReqSize > maxReqBlockCount {
		return
	}

	bs.logger.Debugf("Rcv block request:reqHeight:%d, reqSize:%v, localHeight:%d", br.ReqHeight, br.ReqSize, localHeight)
	blocks := bs.chain.BatchGetBlocksAfterHeight(br.ReqHeight, int(br.ReqSize))
	responseBlocks(m.Source(), blocks)
}

func responseBlocks(targetID string, blocks []*types.Block) {
	body, e := marshalBlockMsgResponse(&blockResponseMessage{Blocks: blocks})
	if e != nil {
		return
	}
	message := network.Message{Code: network.BlockResponseMsg, Body: body}
	network.GetNetInstance().Send(targetID, message)
}

func marshalBlockMsgResponse(bmr *blockResponseMessage) ([]byte, error) {
	pbblocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range bmr.Blocks {
		pb := types.BlockToPb(b)
		pbblocks = append(pbblocks, pb)
	}
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	return proto.Marshal(&message)
}

func (bs *blockSyncer) candidatePoolDump() {
	bs.logger.Debugf("Candidate Pool Dump:")
	for id, topBlockInfo := range bs.candidatePool {
		bs.logger.Debugf("Candidate id:%s,totalQn:%d, pv:%v, height:%d,topHash:%s", id, topBlockInfo.BH.TotalQN, topBlockInfo.BW.PV, topBlockInfo.BH.Height, topBlockInfo.BH.Hash.Hex())
	}
}

func marshalTopBlockInfo(header *types.BlockHeader) ([]byte, error) {
	blockHeader := types.BlockHeaderToPb(header)
	blockInfo := tas_middleware_pb.TopBlockInfo{TopHeader: blockHeader}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*types.BlockHeader, error) {
	message := new(tas_middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	return types.PbToBlockHeader(message.TopHeader), nil
}

func (bs *blockSyncer) unMarshalBlockMsgResponse(b []byte) (*blockResponseMessage, error) {
	message := new(tas_middleware_pb.BlockResponseMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockMsgResponse error:%s", e.Error())
		return nil, e
	}
	blocks := make([]*types.Block, 0)
	for _, pb := range message.Blocks {
		b := types.PbToBlock(pb)
		blocks = append(blocks, b)
	}
	bmr := blockResponseMessage{Blocks: blocks}
	return &bmr, nil
}
