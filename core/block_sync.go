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
	"math/big"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/ticker"
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

var blockSync *blockSyncer

type blockSyncer struct {
	chain *FullBlockChain

	candidatePool map[string]*topBlockInfo
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

// InitBlockSyncer initialize the blockSyncer. Register the ticker for sending and requesting blocks to neighbors timely
// and also subscribe these events to handle requests from neighbors
func InitBlockSyncer(chain *FullBlockChain) {
	blockSync = &blockSyncer{
		candidatePool: make(map[string]*topBlockInfo),
		chain:         chain,
		syncingPeers:  make(map[string]uint64),
	}
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
	localHeight := bs.chain.Height()
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	_, candTop := bs.getBestCandidate("")
	if candTop == nil {
		return false
	}
	return candTop.Height > localHeight+50
}

func (bs *blockSyncer) getBestCandidate(candidateID string) (string, *topBlockInfo) {
	if candidateID == "" {
		for id := range bs.candidatePool {
			if peerManagerImpl.isEvil(id) {
				bs.logger.Debugf("peer meter evil id:%+v", peerManagerImpl.getOrAddPeer(id))
				delete(bs.candidatePool, id)
			}
		}
		if len(bs.candidatePool) == 0 {
			return "", nil
		}
		var maxWeightBlock *topBlockInfo

		for id, top := range bs.candidatePool {
			if maxWeightBlock == nil || top.MoreWeight(&maxWeightBlock.BlockWeight) {
				maxWeightBlock = top
				candidateID = id
			}
		}

	}
	maxTop := bs.candidatePool[candidateID]
	if maxTop == nil {
		return "", nil
	}

	return candidateID, maxTop
}

func (bs *blockSyncer) getPeerTopBlock(id string) *topBlockInfo {
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

	candidate, candidateTop := bs.getBestCandidate(from)
	if candidate == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		return false
	}
	bs.logger.Debugf("candidate info: id %v, top %v %v %v", candidate, candidateTop.Hash.Hex(), candidateTop.Height, candidateTop.TotalQN)

	if localTopBlock.MoreWeight(&candidateTop.BlockWeight) {
		bs.logger.Debugf("local top more weight: local:%v %v %v, candidate: %v %v %v", localTopBlock.Height, localTopBlock.Hash.Hex(), localTopBlock.BlockWeight, candidateTop.Height, candidateTop.Hash.Hex(), candidateTop.BlockWeight)
		return false
	}
	if bs.chain.HasBlock(candidateTop.Hash) {
		bs.logger.Debugf("local has block %v, won't sync", candidateTop.Hash.Hex())
		return false
	}
	beginHeight := uint64(0)
	localHeight := bs.chain.Height()
	if candidateTop.Height <= localHeight {
		beginHeight = candidateTop.Height
	} else {
		beginHeight = localHeight + 1
	}

	bs.logger.Debugf("beginHeight %v, candidateHeight %v", beginHeight, candidateTop.Height)
	if beginHeight > candidateTop.Height {
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
		CandidateHeight: candidateTop.Height,
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
	topBlockInfo := newTopBlockInfo(top)

	bs.logger.Debugf("Send local %d,%v to neighbor!", top.TotalQN, top.Hash.Hex())
	body, e := marshalTopBlockInfo(topBlockInfo)
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

	blockInfo, e := bs.unMarshalTopBlockInfo(bnm.Body())
	if e != nil {
		bs.logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	source := bnm.Source()
	peerManagerImpl.heardFromPeer(source)

	bs.addCandidatePool(source, blockInfo)
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
		if peerTop != nil && localTop.MoreWeight(&peerTop.BlockWeight) {
			bs.logger.Debugf("sync block from %v, local top hash %v, height %v, totalQN %v, peerTop hash %v, height %v, totalQN %v", localTop.Hash.Hex(), localTop.Height, localTop.TotalQN, peerTop.Hash.Hex(), peerTop.Height, peerTop.TotalQN)
			return
		}

		allSuccess := true

		bs.chain.batchAddBlockOnChain(source, "sync", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
			bs.logger.Debugf("sync block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.Hex(), b.Header.Height, ret)
			if ret == types.AddBlockSucc || ret == types.BlockExisted {
				return true
			}
			allSuccess = false
			return false
		})

		// The weight is still low, continue to synchronize (must add blocks
		// is successful, otherwise it will cause an infinite loop)
		if allSuccess && peerTop != nil && peerTop.MoreWeight(&localTop.BlockWeight) {
			bs.syncComplete(source, false)
			complete = true
			go bs.trySyncRoutine()
		}
	}
}

func (bs *blockSyncer) addCandidatePool(source string, topBlockInfo *topBlockInfo) {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[source] = topBlockInfo
		return
	}
	for id, tbi := range bs.candidatePool {
		if topBlockInfo.MoreWeight(&tbi.BlockWeight) {
			delete(bs.candidatePool, id)
			bs.candidatePool[source] = topBlockInfo
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
	if br.ReqHeight == 0 || br.ReqHeight > localHeight || br.ReqSize > maxReqBlockCount {
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
		bs.logger.Debugf("Candidate id:%s,totalQn:%d, pv:%v, height:%d,topHash:%s", id, topBlockInfo.TotalQN, topBlockInfo.PV, topBlockInfo.Height, topBlockInfo.Hash.Hex())
	}
}

func marshalTopBlockInfo(bi *topBlockInfo) ([]byte, error) {
	blockInfo := tas_middleware_pb.TopBlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQN, PVBig: bi.PV.Bytes(), Height: &bi.Height}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*topBlockInfo, error) {
	message := new(tas_middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	pv := big.NewInt(0).SetBytes(message.PVBig)
	bw := &types.BlockWeight{
		TotalQN: *message.TotalQn,
		PV:      pv,
		Hash:    common.BytesToHash(message.Hash),
	}
	blockInfo := topBlockInfo{BlockWeight: *bw, Height: *message.Height}
	return &blockInfo, nil
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
