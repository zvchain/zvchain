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
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/log"
	"sync"

	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	tas_middleware_pb "github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

const (
	reqTimeout                 = 10
	findAncestorReqPieceLength = 10
)

const tickerForkProcess = "fork_process_ticker"

type blockVerifier interface {
	VerifyBlockHeaders(pre, bh *types.BlockHeader) (ok bool, err error)
}

type peerCheckpoint interface {
	checkPointOf(chainSlice []*types.Block) *types.BlockHeader
}
type msgSender interface {
	Send(id string, msg network.Message) error
}

type forkSyncContext struct {
	target                     string
	targetTop                  *topBlockInfo
	lastReqPiece               *findAncestorPieceReq
	localTop                   *topBlockInfo
	localCP                    *types.BlockHeader
	ancestor                   *types.BlockHeader
	requestChainSliceEndHeight uint64 // request end height, excluded
	lastRequestEndHeight       uint64 // last requested end height
	receivedChainSlice         chainSlice
}

func (fctx *forkSyncContext) getLastHash() common.Hash {
	size := len(fctx.lastReqPiece.ChainPiece)
	return fctx.lastReqPiece.ChainPiece[size-1]
}

func (fctx *forkSyncContext) addChainSlice(slice chainSlice) {
	if fctx.receivedChainSlice == nil {
		fctx.receivedChainSlice = make(chainSlice, 0, len(slice))
	}
	fctx.receivedChainSlice = append(fctx.receivedChainSlice, slice...)
}

func (fctx *forkSyncContext) allChainSliceRequested() bool {
	return fctx.lastRequestEndHeight >= fctx.requestChainSliceEndHeight
}

type forkProcessor struct {
	chain     *FullBlockChain
	verifier  blockVerifier
	peerCP    peerCheckpoint
	msgSender msgSender

	syncCtx *forkSyncContext

	lock   sync.RWMutex
	logger *logrus.Logger
}

type findAncestorBlockResponse struct {
	Blocks       []*types.Block
	TopHeader    *types.BlockHeader
	FindAncestor bool
}

type findAncestorPieceReq struct {
	ChainPiece []common.Hash
	ReqCnt     int32
}

type chainSliceReq struct {
	begin, end uint64
}

type chainSlice []*types.Block

func (cs chainSlice) lastBlock() *types.BlockHeader {
	return cs[len(cs)-1].Header
}

func initForkProcessor(chain *FullBlockChain, verifier blockVerifier) *forkProcessor {
	fh := forkProcessor{
		chain:     chain,
		verifier:  verifier,
		peerCP:    chain.cpChecker,
		msgSender: network.GetNetInstance(),
	}
	fh.logger = log.ForkLogger
	notify.BUS.Subscribe(notify.ForkFindAncestorResponse, fh.onFindAncestorResponse)
	notify.BUS.Subscribe(notify.ForkFindAncestorReq, fh.onFindAncestorReq)
	notify.BUS.Subscribe(notify.ForkChainSliceReq, fh.onChainSliceRequest)
	notify.BUS.Subscribe(notify.ForkChainSliceResponse, fh.onChainSliceResponse)

	return &fh
}

func (fp *forkProcessor) targetTop(id string, bh *types.BlockHeader) *topBlockInfo {
	tb := newTopBlockInfo(bh)
	if blockSync == nil {
		return tb
	}
	targetTop := blockSync.getPeerTopBlock(id)
	if targetTop != nil && targetTop.BW.MoreWeight(&tb.BlockWeight) {
		return newTopBlockInfo(targetTop.BH)
	}
	return tb
}

func (fp *forkProcessor) updateContext(id string, targetTop *topBlockInfo) *forkSyncContext {
	top := fp.chain.QueryTopBlock()
	newCtx := &forkSyncContext{
		target:    id,
		targetTop: targetTop,
		localTop:  newTopBlockInfo(top),
		localCP:   fp.chain.CheckPointAt(top.Height),
	}
	fp.syncCtx = newCtx
	return newCtx
}

func (fp *forkProcessor) getLocalPieceInfo(topHash common.Hash) []common.Hash {
	bh := fp.chain.queryBlockHeaderByHash(topHash)
	pieces := make([]common.Hash, 0)
	for len(pieces) < findAncestorReqPieceLength && bh != nil {
		pieces = append(pieces, bh.Hash)
		bh = fp.chain.queryBlockHeaderByHash(bh.PreHash)
	}
	return pieces
}

func (fp *forkProcessor) tryToProcessFork(targetNode string, b *types.Block) (ret bool) {
	if targetNode == "" {
		return
	}
	bh := b.Header
	if fp.chain.HasBlock(bh.PreHash) {
		fp.chain.AddBlockOnChain(targetNode, b)
		return
	}

	fp.lock.Lock()
	defer fp.lock.Unlock()

	ctx := fp.syncCtx

	if ctx != nil {
		fp.logger.Debugf("fork processing with %v: targetTop %v, local %v", ctx.target, ctx.targetTop.Height, ctx.localTop.Height)
		return
	}

	localTop := fp.chain.QueryTopBlock()
	targetTop := fp.targetTop(targetNode, bh)
	// local more weight, won't process
	if !targetTop.MoreWeight(types.NewBlockWeight(localTop)) {
		fp.logger.Debugf("local top more weight, won't process fork:local %v %v, target %v %v", localTop.Hash, localTop.Height, targetTop.Hash, targetTop.Height)
		return
	}
	ret = true
	ctx = fp.updateContext(targetNode, targetTop)

	fp.logger.Debugf("fork process from %v: targetTop:%v-%v, local:%v-%v", ctx.target, ctx.targetTop.Hash.Hex(), ctx.targetTop.Height, ctx.localTop.Hash.Hex(), ctx.localTop.Height)
	fp.findAncestorRequest(fp.chain.QueryTopBlock().Hash)
	return
}

func (fp *forkProcessor) reqPieceTimeout() {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.syncCtx == nil {
		return
	}
	fp.logger.Warnf("req piece from %v timeout, target top %v", fp.syncCtx.target, fp.syncCtx.targetTop.Height)
	peerManagerImpl.timeoutPeer(fp.syncCtx.target)
	peerManagerImpl.updateReqBlockCnt(fp.syncCtx.target, false)
	fp.reset()
}

func (fp *forkProcessor) reset() {
	fp.syncCtx = nil
	fp.chain.ticker.RemoveRoutine(fp.timeoutTickerName())
}

func (fp *forkProcessor) timeoutTickerName() string {
	return tickerForkProcess
}

func (fp *forkProcessor) findAncestorRequest(topHash common.Hash) {

	chainPieceInfo := fp.getLocalPieceInfo(topHash)
	if len(chainPieceInfo) == 0 {
		fp.reset()
		return
	}

	reqCnt := peerManagerImpl.getPeerReqBlockCount(fp.syncCtx.target)

	pieceReq := &findAncestorPieceReq{
		ChainPiece: chainPieceInfo,
		ReqCnt:     int32(reqCnt),
	}

	body, e := marshalFindAncestorReqInfo(pieceReq)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		fp.reset()
		return
	}
	fp.logger.Debugf("req piece from %v, reqCnt %v", fp.syncCtx.target, reqCnt)

	message := network.Message{Code: network.ForkFindAncestorReq, Body: body}
	fp.msgSender.Send(fp.syncCtx.target, message)

	fp.syncCtx.lastReqPiece = pieceReq

	// Start ticker
	fp.chain.ticker.RegisterOneTimeRoutine(fp.timeoutTickerName(), func() bool {
		fp.reqPieceTimeout()
		return true
	}, reqTimeout)
}

func (fp *forkProcessor) findCommonAncestor(piece []common.Hash) *common.Hash {
	for _, h := range piece {
		if fp.chain.HasBlock(h) {
			return &h
		}
	}
	return nil
}

func (fp *forkProcessor) onFindAncestorReq(msg notify.Message) error {
	m := notify.AsDefault(msg)

	source := m.Source()
	pieceReq, err := unmarshalFindAncestorPieceReqInfo(m.Body())
	if err != nil {
		err = fmt.Errorf("unmarshalFindAncestorPieceReqInfo err %v", err)
		fp.logger.Error(err)
		return err
	}

	fp.logger.Debugf("Rcv chain piece block req from:%s, pieceSize %v, reqCnt %v", source, len(pieceReq.ChainPiece), pieceReq.ReqCnt)
	if pieceReq.ReqCnt > maxReqBlockCount {
		pieceReq.ReqCnt = maxReqBlockCount
	}
	if len(pieceReq.ChainPiece) == 0 {
		return fmt.Errorf("chain pieces empty")
	}

	blocks := make([]*types.Block, 0)
	ancestor := fp.findCommonAncestor(pieceReq.ChainPiece)

	response := &findAncestorBlockResponse{
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
	fp.sendFindAncestorResponse(source, response)
	return nil
}

func (fp *forkProcessor) sendFindAncestorResponse(targetID string, msg *findAncestorBlockResponse) {
	fp.logger.Debugf("Send chain piece blocks to:%s, findAncestor=%v, blockSize=%v", targetID, msg.FindAncestor, len(msg.Blocks))
	body, e := marshalFindAncestorBlockResponseMsg(msg)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece block msg error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ForkFindAncestorResponse, Body: body}
	fp.msgSender.Send(targetID, message)
}

func (fp *forkProcessor) reqFinished(reset bool) {
	if fp.syncCtx == nil {
		return
	}
	ctx := fp.syncCtx
	peerManagerImpl.heardFromPeer(ctx.target)
	peerManagerImpl.updateReqBlockCnt(ctx.target, true)

	if reset {
		fp.reset()
	}
	return
}

func (fp *forkProcessor) getNextSyncHash() *common.Hash {
	last := fp.syncCtx.getLastHash()
	bh := fp.chain.QueryBlockHeaderByHash(last)

	// If local cp higher than next block for sync, then stop syncing.
	if bh.Height <= fp.syncCtx.localCP.Height {
		fp.logger.Debugf("local cp higher than next block piece for finding ancestor: cp %v", fp.syncCtx.localCP.Height)
		return nil
	}
	if bh != nil {
		h := bh.PreHash
		return &h
	}
	return nil
}

// chainSliceRequestEndHeight returns end of chain slice request range which is excluded
func (fp *forkProcessor) chainSliceRequestEndHeight(peerTop, ancestor *types.BlockHeader) uint64 {
	noHandleEpoch := types.ActivateEpochOfGroupsCreatedAt(ancestor.Height)
	topEpoch := types.EpochAt(peerTop.Height)

	// Short forks, all blocks can be verified
	if topEpoch.End() < noHandleEpoch.End() {
		return peerTop.Height + 1
	}
	// Long forks, only blocks lower than noHandleEpoch can be verified
	return noHandleEpoch.Start()
}

func (fp *forkProcessor) onFindAncestorResponse(msg notify.Message) error {
	m := notify.AsDefault(msg)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	source := m.Source()

	ctx := fp.syncCtx
	if ctx == nil {
		fp.logger.Debugf("ctx is nil: source=%v", source)
		return nil
	}
	// Target changed
	if source != ctx.target {
		fp.logger.Debugf("unexpected target blocks, target=%v, expect=%v, ignored", source, ctx.target)
		return fmt.Errorf("unexpected target")
	}

	var (
		reset = true
	)
	defer func() {
		fp.reqFinished(reset)
	}()

	chainPieceBlockMsg, e := unmarshalFindAncestorBlockResponseMsg(m.Body())
	if e != nil {
		fp.logger.Error(e)
		return e
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	localTop := fp.chain.QueryTopBlock()
	s := "no blocks"
	if len(blocks) > 0 {
		s = fmt.Sprintf("%v-%v", blocks[0].Header.Height, blocks[len(blocks)-1].Header.Height)
	}
	fp.logger.Debugf("rev block piece from %v, top %v-%v, blocks %v, findAc %v, local %v-%v, ctx target:%v %v", source, topHeader.Hash.Hex(), topHeader.Height, s, chainPieceBlockMsg.FindAncestor, localTop.Hash.Hex(), localTop.Height, ctx.target, ctx.targetTop.Height)

	if ctx.lastReqPiece == nil {
		return fmt.Errorf("lastReqPiece is nil")
	}
	if topHeader == nil {
		err := fmt.Errorf("top header is nil")
		fp.logger.Error(err)
		return err
	}

	// Can't find common ancestor with the given piece and keep finding with next pieces.
	if !chainPieceBlockMsg.FindAncestor {
		nextSync := fp.getNextSyncHash()
		if nextSync != nil {
			fp.logger.Debugf("cannot find common ancestor from %v, keep finding", source)
			go fp.findAncestorRequest(*nextSync)
			reset = false
		}
	} else { // Common ancestor found
		if len(blocks) == 0 {
			err := fmt.Errorf("from %v, find ancestor, but blocks is empty", source)
			fp.logger.Error(err)
			return err
		}
		ancestorBH := blocks[0].Header

		// Won't process if local cp higher than ancestor
		if ctx.localCP.Height > ancestorBH.Height {
			err := fmt.Errorf("local checkpoint higher than ancestor: cp %v, ancestor %v", ctx.localCP.Height, ancestorBH.Height)
			fp.logger.Info(err)
			return err
		}
		ctx.ancestor = ancestorBH
		ctx.addChainSlice(blocks)
		ctx.lastRequestEndHeight = ctx.receivedChainSlice.lastBlock().Height + 1

		// height can be handled
		requestEndHeight := fp.chainSliceRequestEndHeight(topHeader, ancestorBH)
		ctx.requestChainSliceEndHeight = requestEndHeight

		fp.checkChainSlice()

		reset = false

	}
	return nil
}

func (fp *forkProcessor) chainSliceRequest(target string, beginHeight, end uint64) {
	br := &chainSliceReq{
		begin: beginHeight,
		end:   end,
	}

	body, err := marshalChainSliceReqMsg(br)
	if err != nil {
		fp.logger.Errorf("marshalChainSliceReqMsg error %v", err)
		return
	}

	message := network.Message{Code: network.ForkChainSliceReq, Body: body}
	fp.msgSender.Send(target, message)
	fp.logger.Debugf("keep requesting chain slice from %v, %v-%v", target, beginHeight, end)

	// Start ticker
	fp.chain.ticker.RegisterOneTimeRoutine(fp.timeoutTickerName(), func() bool {
		fp.reqPieceTimeout()
		return true
	}, reqTimeout)
}

func (fp *forkProcessor) checkChainSlice() {
	ctx := fp.syncCtx

	if ctx.allChainSliceRequested() {
		fp.allBlocksReceived()
	} else {
		cnt := uint64(peerManagerImpl.getPeerReqBlockCount(ctx.target))
		begin := ctx.lastRequestEndHeight
		end := begin + cnt
		if end > ctx.requestChainSliceEndHeight {
			end = ctx.requestChainSliceEndHeight
		}
		ctx.lastRequestEndHeight = end
		fp.chainSliceRequest(ctx.target, begin, end)
	}
}

func (fp *forkProcessor) onChainSliceRequest(msg notify.Message) error {
	bs := notify.AsDefault(msg)

	req, err := unmarshalChainSliceReqMsg(bs.Body())
	if err != nil {
		fp.logger.Errorf("unmarshalChainSliceReq err:%v", err)
		return err
	}

	response := &blockResponseMessage{
		Blocks: make([]*types.Block, 0),
	}
	localHeight := fp.chain.Height()
	if req.end > req.begin && req.begin <= localHeight && (req.end-req.begin) <= maxReqBlockCount {
		response.Blocks = fp.chain.BatchGetBlocksBetween(req.begin, req.end)
	}

	bytes, err := marshalBlockMsgResponse(response)
	if err != nil {
		fp.logger.Errorf("marshalBlockMsgResponse error:%v", err)
		return err
	}
	fp.logger.Debugf("onChainSliceRequest from %v, req %v-%v, response %v", bs.Source(), req.begin, req.end, len(response.Blocks))
	message := network.Message{Code: network.ForkChainSliceResponse, Body: bytes}
	fp.msgSender.Send(bs.Source(), message)
	return nil
}

func (fp *forkProcessor) onChainSliceResponse(msg notify.Message) error {
	defaultMessage := notify.AsDefault(msg)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	ctx := fp.syncCtx
	if ctx == nil {
		fp.logger.Debugf("ctx is nil: source=%v", defaultMessage.Source())
		return nil
	}
	// Target changed
	if defaultMessage.Source() != ctx.target {
		fp.logger.Debugf("unexpected target blocks, target=%v, expect=%v, ignored", defaultMessage.Source(), ctx.target)
		return fmt.Errorf("unexpected target")
	}

	resp, err := unMarshalBlockMsgResponse(defaultMessage.Body())
	if err != nil {
		fp.logger.Errorf("unMarshalBlockMsgResponse error:%v", err)
		return err
	}
	blocks := resp.Blocks
	if len(blocks) == 0 {
		fp.logger.Warnf("get empty blocks from %v", defaultMessage.Source())
		return nil
	}
	peerManagerImpl.heardFromPeer(defaultMessage.Source())

	fp.logger.Debugf("receive chain slice response from %v, size %v, heights %v-%v", defaultMessage.Source(), len(blocks), blocks[0].Header.Height, blocks[len(blocks)-1].Header.Height)
	ctx.addChainSlice(blocks)

	fp.checkChainSlice()

	return nil

}

func (fp *forkProcessor) allBlocksReceived() {
	defer func() {
		fp.reset()
	}()

	if !fp.chain.HasBlock(fp.syncCtx.ancestor.Hash) {
		fp.logger.Errorf("local ancestor block not exist, hash=%v, height=%v", fp.syncCtx.ancestor.Hash, fp.syncCtx.ancestor.Height)
		return
	}
	blocks := fp.syncCtx.receivedChainSlice
	ancestorPre := fp.chain.QueryBlockHeaderByHash(blocks[0].Header.PreHash)
	if ancestorPre == nil {
		fp.logger.Errorf("ancestor pre is nil:%v %v", blocks[0].Header.Hash, blocks[0].Header.Height)
		return
	}
	pre := ancestorPre
	// Ensure blocks are chained and heights are legal
	for _, block := range blocks {
		if pre.Hash != block.Header.PreHash {
			fp.logger.Errorf("blocks not chained: %v %v", pre.Height, block.Header.Height)
			return
		}
		if block.Header.Height >= fp.syncCtx.requestChainSliceEndHeight {
			fp.logger.Errorf("receives block higher than expect height: %v, expect %v", block.Header.Height, fp.syncCtx.requestChainSliceEndHeight)
			return
		}
		pre = block.Header
	}
	pre = ancestorPre
	// Checks the blocks legality
	for _, block := range blocks {
		if ok, err := fp.verifier.VerifyBlockHeaders(pre, block.Header); !ok {
			fp.logger.Errorf("verify block headers err:%v %v %v", block.Header.Hash, block.Header.Height, err)
			return
		}
		pre = block.Header
	}
	// Peer cp
	peerCP := fp.peerCP.checkPointOf(blocks)
	// When Peer cp lower than ancestor, compares the weight
	if peerCP == nil || peerCP.Height <= fp.syncCtx.ancestor.Height {
		peerLast := blocks[len(blocks)-1].Header
		localLast := fp.chain.QueryBlockHeaderFloor(types.EpochAt(peerLast.Height).End() - 1)
		if types.NewBlockWeight(localLast).MoreWeight(types.NewBlockWeight(peerLast)) {
			fp.logger.Infof("local more weight than peer, local %v %v, peer %v %v", localLast.Height, localLast.Hash, peerLast.Height, peerLast.Hash)
			return
		}
	}
	// Accept peer fork, and add the chain slice to local
	fp.chain.batchAddBlockOnChain(fp.syncCtx.target, "fork", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
		fp.logger.Debugf("sync fork block from %v, hash=%v,height=%v,addResult=%v", fp.syncCtx.target, b.Header.Hash, b.Header.Height, ret)
		return ret == types.AddBlockSucc || ret == types.BlockExisted
	})
}

func unmarshalFindAncestorPieceReqInfo(b []byte) (*findAncestorPieceReq, error) {
	message := new(tas_middleware_pb.FindAncestorReq)
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
	chainPieceInfo := &findAncestorPieceReq{ChainPiece: chainPiece, ReqCnt: cnt}
	return chainPieceInfo, nil
}

func marshalFindAncestorBlockResponseMsg(cpb *findAncestorBlockResponse) ([]byte, error) {
	topHeader := types.BlockHeaderToPb(cpb.TopHeader)
	blocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range cpb.Blocks {
		blocks = append(blocks, types.BlockToPb(b))
	}
	message := tas_middleware_pb.FindAncestorBlockResponse{TopHeader: topHeader, Blocks: blocks, FindAncestor: &cpb.FindAncestor}
	return proto.Marshal(&message)
}

func unmarshalFindAncestorBlockResponseMsg(b []byte) (*findAncestorBlockResponse, error) {
	message := new(tas_middleware_pb.FindAncestorBlockResponse)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	blocks := make([]*types.Block, 0)
	for _, b := range message.Blocks {
		blocks = append(blocks, types.PbToBlock(b))
	}
	cpb := findAncestorBlockResponse{TopHeader: topHeader, Blocks: blocks, FindAncestor: *message.FindAncestor}
	return &cpb, nil
}

func marshalFindAncestorReqInfo(chainPieceInfo *findAncestorPieceReq) ([]byte, error) {
	pieces := make([][]byte, 0)
	for _, hash := range chainPieceInfo.ChainPiece {
		pieces = append(pieces, hash.Bytes())
	}
	message := tas_middleware_pb.FindAncestorReq{Pieces: pieces, ReqCnt: &chainPieceInfo.ReqCnt}
	return proto.Marshal(&message)
}

func marshalChainSliceReqMsg(msg *chainSliceReq) ([]byte, error) {
	m := &tas_middleware_pb.ChainSliceReq{
		Begin: &msg.begin,
		End:   &msg.end,
	}
	return proto.Marshal(m)
}

func unmarshalChainSliceReqMsg(b []byte) (*chainSliceReq, error) {
	m := new(tas_middleware_pb.ChainSliceReq)
	e := proto.Unmarshal(b, m)
	if e != nil {
		return nil, e
	}
	var (
		begin, end uint64
	)
	if m.Begin == nil {
		begin = 0
	} else {
		begin = *m.Begin
	}
	if m.End == nil {
		end = 0
	} else {
		end = *m.End
	}
	return &chainSliceReq{begin: begin, end: end}, nil
}
