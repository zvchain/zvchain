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
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"

	"fmt"
	"github.com/gogo/protobuf/proto"
)

const (
	reqPieceTimeout  = 60
	chainPieceLength = 10
)

const tickerReqPieceBlock = "req_chain_piece_block"

type forkSyncContext struct {
	target       string
	targetTop    *topBlockInfo
	lastReqPiece *chainPieceReq
	localTop     *topBlockInfo
}

func (fctx *forkSyncContext) getLastHash() common.Hash {
	size := len(fctx.lastReqPiece.ChainPiece)
	return fctx.lastReqPiece.ChainPiece[size-1]
}

type forkProcessor struct {
	chain *FullBlockChain

	syncCtx *forkSyncContext

	lock   sync.RWMutex
	logger taslog.Logger
}

type chainPieceBlockMsg struct {
	Blocks       []*types.Block
	TopHeader    *types.BlockHeader
	FindAncestor bool
}

type chainPieceReq struct {
	ChainPiece []common.Hash
	ReqCnt     int32
}

func initForkProcessor(chain *FullBlockChain) *forkProcessor {
	fh := forkProcessor{
		chain: chain,
	}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceBlockReq, fh.chainPieceBlockReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlock, fh.chainPieceBlockHandler)

	return &fh
}

func (fp *forkProcessor) targetTop(id string, bh *types.BlockHeader) *topBlockInfo {
	targetTop := blockSync.getPeerTopBlock(id)
	tb := newTopBlockInfo(bh)
	if targetTop != nil && targetTop.MoreWeight(&tb.BlockWeight) {
		return targetTop
	}
	return tb
}

func (fp *forkProcessor) updateContext(id string, bh *types.BlockHeader) *forkSyncContext {
	targetTop := fp.targetTop(id, bh)

	newCtx := &forkSyncContext{
		target:    id,
		targetTop: targetTop,
		localTop:  newTopBlockInfo(fp.chain.QueryTopBlock()),
	}
	fp.syncCtx = newCtx
	return newCtx
}

func (fp *forkProcessor) getLocalPieceInfo(topHash common.Hash) []common.Hash {
	bh := fp.chain.queryBlockHeaderByHash(topHash)
	pieces := make([]common.Hash, 0)
	for len(pieces) < chainPieceLength && bh != nil {
		pieces = append(pieces, bh.Hash)
		bh = fp.chain.queryBlockHeaderByHash(bh.PreHash)
	}
	return pieces
}

func (fp *forkProcessor) tryToProcessFork(targetNode string, b *types.Block) {
	if blockSync == nil {
		return
	}
	if targetNode == "" {
		return
	}

	fp.lock.Lock()
	defer fp.lock.Unlock()

	bh := b.Header
	if fp.chain.HasBlock(bh.PreHash) {
		fp.chain.AddBlockOnChain(targetNode, b)
		return
	}
	ctx := fp.syncCtx

	if ctx != nil {
		fp.logger.Debugf("fork processing with %v: targetTop %v, local %v", ctx.target, ctx.targetTop.Height, ctx.localTop.Height)
		return
	}

	ctx = fp.updateContext(targetNode, bh)

	fp.logger.Debugf("fork process from %v: targetTop:%v-%v, local:%v-%v", ctx.target, ctx.targetTop.Hash.Hex(), ctx.targetTop.Height, ctx.localTop.Hash.Hex(), ctx.localTop.Height)
	fp.requestPieceBlock(fp.chain.QueryTopBlock().Hash)
}

func (fp *forkProcessor) reqPieceTimeout(id string) {
	fp.logger.Debugf("req piece from %v timeout", id)
	if fp.syncCtx == nil {
		return
	}
	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.syncCtx.target != id {
		return
	}
	peerManagerImpl.timeoutPeer(fp.syncCtx.target)
	peerManagerImpl.updateReqBlockCnt(fp.syncCtx.target, false)
	fp.reset()
}

func (fp *forkProcessor) reset() {
	fp.syncCtx = nil
}

func (fp *forkProcessor) timeoutTickerName(id string) string {
	return tickerReqPieceBlock + id
}

func (fp *forkProcessor) requestPieceBlock(topHash common.Hash) {

	chainPieceInfo := fp.getLocalPieceInfo(topHash)
	if len(chainPieceInfo) == 0 {
		fp.reset()
		return
	}

	reqCnt := peerManagerImpl.getPeerReqBlockCount(fp.syncCtx.target)

	pieceReq := &chainPieceReq{
		ChainPiece: chainPieceInfo,
		ReqCnt:     int32(reqCnt),
	}

	body, e := marshalChainPieceInfo(pieceReq)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		fp.reset()
		return
	}
	fp.logger.Debugf("req piece from %v, reqCnt %v", fp.syncCtx.target, reqCnt)

	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	network.GetNetInstance().Send(fp.syncCtx.target, message)

	fp.syncCtx.lastReqPiece = pieceReq

	// Start ticker
	fp.chain.ticker.RegisterOneTimeRoutine(fp.timeoutTickerName(fp.syncCtx.target), func() bool {
		if fp.syncCtx != nil {
			fp.reqPieceTimeout(fp.syncCtx.target)
		}
		return true
	}, reqPieceTimeout)
}

func (fp *forkProcessor) findCommonAncestor(piece []common.Hash) *common.Hash {
	for _, h := range piece {
		if fp.chain.HasBlock(h) {
			return &h
		}
	}
	return nil
}

func (fp *forkProcessor) chainPieceBlockReqHandler(msg notify.Message) {
	m := notify.AsDefault(msg)

	source := m.Source()
	pieceReq, err := unMarshalChainPieceInfo(m.Body())
	if err != nil {
		fp.logger.Errorf("unMarshalChainPieceInfo err %v", err)
		return
	}

	fp.logger.Debugf("Rcv chain piece block req from:%s, pieceSize %v, reqCnt %v", source, len(pieceReq.ChainPiece), pieceReq.ReqCnt)
	if pieceReq.ReqCnt > maxReqBlockCount {
		pieceReq.ReqCnt = maxReqBlockCount
	}

	blocks := make([]*types.Block, 0)
	ancestor := fp.findCommonAncestor(pieceReq.ChainPiece)

	response := &chainPieceBlockMsg{
		TopHeader:    fp.chain.QueryTopBlock(),
		FindAncestor: ancestor != nil,
		Blocks:       blocks,
	}

	if ancestor != nil { // Find a common ancestor
		ancestorBH := fp.chain.queryBlockHeaderByHash(*ancestor)
		// Maybe the ancestor were killed due to forks
		if ancestorBH != nil {
			blocks = fp.chain.BatchGetBlocksAfterHeight(ancestorBH.Height, int(pieceReq.ReqCnt))
			response.Blocks = blocks
		}
	}
	fp.sendChainPieceBlock(source, response)
}

func (fp *forkProcessor) sendChainPieceBlock(targetID string, msg *chainPieceBlockMsg) {
	fp.logger.Debugf("Send chain piece blocks to:%s, findAncestor=%v, blockSize=%v", targetID, msg.FindAncestor, len(msg.Blocks))
	body, e := marshalChainPieceBlockMsg(msg)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece block msg error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceBlock, Body: body}
	network.GetNetInstance().Send(targetID, message)
}

func (fp *forkProcessor) reqFinished(id string, reset bool) {
	if fp.syncCtx == nil || fp.syncCtx.target != id {
		return
	}
	peerManagerImpl.heardFromPeer(id)
	fp.chain.ticker.RemoveRoutine(fp.timeoutTickerName(id))
	peerManagerImpl.updateReqBlockCnt(id, true)
	if reset {
		fp.reset()
	}
	return
}

func (fp *forkProcessor) getNextSyncHash() *common.Hash {
	last := fp.syncCtx.getLastHash()
	bh := fp.chain.QueryBlockHeaderByHash(last)
	if bh != nil {
		h := bh.PreHash
		return &h
	}
	return nil
}

func (fp *forkProcessor) chainPieceBlockHandler(msg notify.Message) {
	m := notify.AsDefault(msg)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	source := m.Source()

	ctx := fp.syncCtx
	if ctx == nil {
		fp.logger.Debugf("ctx is nil: source=%v", source)
		return
	}
	chainPieceBlockMsg, e := unmarshalChainPieceBlockMsg(m.Body())
	if e != nil {
		fp.logger.Warnf("Unmarshal chain piece block msg error:%d", e.Error())
		return
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	localTop := fp.chain.QueryTopBlock()
	s := "no blocks"
	if len(blocks) > 0 {
		s = fmt.Sprintf("%v-%v", blocks[0].Header.Height, blocks[len(blocks)-1].Header.Height)
	}
	fp.logger.Debugf("rev block piece from %v, top %v-%v, blocks %v, findAc %v, local %v-%v, ctx target:%v %v", source, topHeader.Hash.Hex(), topHeader.Height, s, chainPieceBlockMsg.FindAncestor, localTop.Hash.Hex(), localTop.Height, ctx.target, ctx.targetTop.Height)

	// Target changed
	if source != ctx.target {
		// If the received block contains a branch of the ctx request, you can continue the add on chain process.
		sameFork := false
		if topHeader != nil && topHeader.Hash == ctx.targetTop.Hash {
			sameFork = true
		} else if len(blocks) > 0 {
			for _, b := range blocks {
				if b.Header.Hash == ctx.targetTop.Hash {
					sameFork = true
					break
				}
			}
		}
		if !sameFork {
			fp.logger.Debugf("Unexpected chain piece block from %s, expect from %s, blocksize %v", source, ctx.target, len(blocks))
			return
		}
		fp.logger.Debugf("upexpected target blocks, buf same fork!target=%v, expect=%v, blocksize %v", source, ctx.target, len(blocks))
	}
	var reset = true
	defer func() {
		fp.reqFinished(source, reset)
	}()

	if ctx.lastReqPiece == nil {
		return
	}

	// Giving a piece to go is not enough to find a common ancestor, continue to request a piece
	if !chainPieceBlockMsg.FindAncestor {
		nextSync := fp.getNextSyncHash()
		if nextSync != nil {
			fp.logger.Debugf("cannot find common ancestor from %v, keep finding", source)
			fp.requestPieceBlock(*nextSync)
			reset = false
		}
	} else {
		if len(blocks) == 0 {
			fp.logger.Errorf("from %v, find ancesotr, but blocks is empty!", source)
			return
		}
		ancestorBH := blocks[0].Header
		if !fp.chain.HasBlock(ancestorBH.Hash) {
			fp.logger.Errorf("local ancestor block not exist, hash=%v, height=%v", ancestorBH.Hash.Hex(), ancestorBH.Height)
		} else if len(blocks) > 1 {
			fp.chain.batchAddBlockOnChain(source, "fork", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
				fp.logger.Debugf("sync fork block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.Hex(), b.Header.Height, ret)
				return ret == types.AddBlockSucc || ret == types.BlockExisted
			})
			// Start synchronization if the local weight is still below the weight of the other party
			if fp.chain.compareChainWeight(topHeader) < 0 {
				go blockSync.trySyncRoutine()
			}
		}
	}
}

func unMarshalChainPieceInfo(b []byte) (*chainPieceReq, error) {
	message := new(tas_middleware_pb.ChainPieceReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}

	chainPiece := make([]common.Hash, 0)
	for _, hashBytes := range message.Pieces {
		h := common.BytesToHash(hashBytes)
		chainPiece = append(chainPiece, h)
	}
	cnt := int32(maxReqBlockCount)
	if message.ReqCnt != nil {
		cnt = *message.ReqCnt
	}
	chainPieceInfo := &chainPieceReq{ChainPiece: chainPiece, ReqCnt: cnt}
	return chainPieceInfo, nil
}

func marshalChainPieceBlockMsg(cpb *chainPieceBlockMsg) ([]byte, error) {
	topHeader := types.BlockHeaderToPb(cpb.TopHeader)
	blocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range cpb.Blocks {
		blocks = append(blocks, types.BlockToPb(b))
	}
	message := tas_middleware_pb.ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks, FindAncestor: &cpb.FindAncestor}
	return proto.Marshal(&message)
}

func unmarshalChainPieceBlockMsg(b []byte) (*chainPieceBlockMsg, error) {
	message := new(tas_middleware_pb.ChainPieceBlockMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	blocks := make([]*types.Block, 0)
	for _, b := range message.Blocks {
		blocks = append(blocks, types.PbToBlock(b))
	}
	cpb := chainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks, FindAncestor: *message.FindAncestor}
	return &cpb, nil
}

func marshalChainPieceInfo(chainPieceInfo *chainPieceReq) ([]byte, error) {
	pieces := make([][]byte, 0)
	for _, hash := range chainPieceInfo.ChainPiece {
		pieces = append(pieces, hash.Bytes())
	}
	message := tas_middleware_pb.ChainPieceReq{Pieces: pieces, ReqCnt: &chainPieceInfo.ReqCnt}
	return proto.Marshal(&message)
}
