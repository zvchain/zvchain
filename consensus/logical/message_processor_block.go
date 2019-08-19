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

package logical

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
)

func (p *Processor) thresholdPieceVerify(vctx *VerifyContext, slot *SlotContext) {
	p.reserveBlock(vctx, slot)
}

// verifyCastMessage verifies the message from proposal node
// Especially, as it takes the previous blockHeader as input, future proposal messages is not processable by the method,
// which may be triggered at the near future
// It returns true if and only if the message is legal
func (p *Processor) verifyCastMessage(msg *model.ConsensusCastMessage, preBH *types.BlockHeader) (ok bool, err error) {
	bh := &msg.BH
	si := &msg.SI
	castor := groupsig.DeserializeID(bh.Castor)
	gSeed := bh.Group

	// if the block already added on chain, then ignore this message
	if p.blockOnChain(bh.Hash) {
		err = fmt.Errorf("block onchain already")
		return
	}

	// check expire time, fail fast if expired
	expireTime := expireTime(bh, preBH)
	if p.ts.NowAfter(expireTime) {
		err = fmt.Errorf("cast verify expire, gseed=%v, preTime %v, expire %v", gSeed, preBH.CurTime, expireTime)
		return
	} else if bh.Height > 1 { // if the message comes early before the time it should begin, then deny it
		beginTime := expireTime.Add(-int64(model.Param.MaxGroupCastTime + 1))
		if !p.ts.NowAfter(beginTime) {
			err = fmt.Errorf("cast begin time illegal, expectBegin at %v, expire at %v", beginTime, expireTime)
			return
		}

	}
	if _, same := p.blockContexts.isHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("the block of this height has been cast %v", bh.Height)
		return
	}

	group := p.groupReader.getGroupBySeed(gSeed)
	if group == nil {
		err = fmt.Errorf("verifyGroup is nil: gseed=%v", gSeed)
		return
	}

	// get the verify context for the height
	// it won't create the context if not exist and just for fail fast
	vctx := p.blockContexts.getVctxByHeight(bh.Height)
	if vctx != nil {
		if vctx.blockSigned(bh.Hash) {
			err = fmt.Errorf("block signed")
			return
		}
		if vctx.prevBH.Hash == bh.PreHash {
			err = vctx.baseCheck(bh, si.GetID())
			if err != nil {
				return
			}
		}
	}
	castorDO := p.minerReader.getProposeMinerByHeight(castor, preBH.Height)
	if castorDO == nil {
		err = fmt.Errorf("castorDO nil id=%v", castor)
		return
	}
	pk := castorDO.PK

	// check the signature of the proposal
	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail")
		return
	}

	// check if the blockHeader is legal
	err = p.isCastLegal(bh, preBH)
	if err != nil {
		return
	}

	// get the verify context for the height, it will create the context if not exists
	vctx = p.blockContexts.getOrNewVerifyContext(group, bh, preBH)
	if vctx == nil {
		err = fmt.Errorf("get vctx is empty, maybe preBH has been deleted")
		return
	}

	// prepare the slot for the blockHeader, create if not exists
	slot, err := vctx.PrepareSlot(bh)
	if err != nil {
		return
	}
	if !slot.IsWaiting() {
		err = fmt.Errorf("slot status %v, not waiting", slot.GetSlotStatus())
		return
	}

	sKey := p.groupReader.getGroupSignatureSeckey(gSeed)
	var cvm model.ConsensusVerifyMessage
	cvm.BlockHash = bh.Hash

	// sign the message and send to other members in the verifyGroup
	if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), sKey), &cvm) {
		cvm.GenRandomSign(sKey, vctx.prevBH.Random)
		p.NetServer.SendVerifiedCast(&cvm, gSeed)
		slot.setSlotStatus(slSigned)
		p.blockContexts.attachVctx(bh, vctx)
		vctx.markSignedBlock(bh)

		stdLogger.Debugf("signdata: hash=%v, sk=%v, id=%v, sign=%v, seed=%v", bh.Hash.Hex(), sKey.GetHexString(), p.GetMinerID(), cvm.SI.DataSign.GetHexString(), gSeed)

		// trigger the cached messages from other members that come ahead of the proposal message
		p.castVerifyCh <- bh
		ok = true
	} else {
		err = fmt.Errorf("gen sign fail")
	}
	return
}

// OnMessageCast handles the message from the proposer
// Note that, if the pre-block of the block present int the message isn't on the blockchain, it will caches the message
// and trigger it after the pre-block added on chain
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) (err error) {
	bh := &ccm.BH
	traceLog := monitor.NewPerformTraceLogger("OnMessageCast", bh.Hash, bh.Height)

	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeProposal,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: ccm.SI.GetID().GetAddrString(),
		Verifier: bh.Group.Hex(),
		Ext:      fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	}
	group := p.groupReader.getGroupBySeed(bh.Group)
	if group == nil {
		err = fmt.Errorf("GetSelfGroup failed")
		return
	}

	deltaHeight := int(bh.Height - p.MainChain.Height())
	if common.AbsInt(deltaHeight) < 100 && monitor.Instance.IsFirstNInternalNodesInGroup(group.getMembers(), 3) {
		monitor.Instance.AddLogIfNotInternalNodes(le)
	}
	mType := "OMC"

	si := &ccm.SI
	tlog := newHashTraceLog(mType, bh.Hash, si.GetID())
	castor := groupsig.DeserializeID(bh.Castor)

	tlog.logStart("height=%v, castor=%v", bh.Height, castor)

	defer func() {
		result := "signed"
		if err != nil {
			result = err.Error()
		}
		tlog.logEnd("height=%v, preHash=%v, gseed=%v, result=%v", bh.Height, bh.PreHash, bh.Group, result)
		traceLog.Log("PreHash=%v,castor=%v,result=%v", bh.PreHash, ccm.SI.GetID(), result)
	}()
	if ccm.GenHash() != ccm.SI.DataHash || ccm.GenHash() != bh.Hash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", ccm.GenHash(), ccm.SI.DataHash, bh.Hash)
		return
	}
	// Castor need to ignore his message
	//if castor.IsEqual(p.GetMinerID()) {
	//	err = fmt.Errorf("ignore self message")
	//	return
	//}

	// Check if the current node is in the verifyGroup
	if !group.hasMember(p.GetMinerID()) {
		err = fmt.Errorf("don't belong to verifyGroup, gseed=%v, hash=%v, id=%v", bh.Group, bh.Hash, p.GetMinerID())
		return
	}

	if bh.Elapsed <= 0 {
		err = fmt.Errorf("elapsed error %v", bh.Elapsed)
		return
	}

	if p.ts.Since(bh.CurTime) < -1 {
		err = fmt.Errorf("block too early: now %v, curtime %v", p.ts.Now(), bh.CurTime)
		return
	}

	if p.blockOnChain(bh.Hash) {
		err = fmt.Errorf("block onchain already")
		return
	}

	preBH := p.GetBlockHeaderByHash(bh.PreHash)

	// Cache the message due to the absence of the pre-block
	if preBH == nil {
		p.addFutureVerifyMsg(ccm)
		err = fmt.Errorf("parent block did not received")
		return
	}

	verifyTraceLog := monitor.NewPerformTraceLogger("verifyCastMessage", bh.Hash, bh.Height)
	verifyTraceLog.SetParent("OnMessageCast")
	defer verifyTraceLog.Log("")

	_, err = p.verifyCastMessage(ccm, preBH)
	return
}

func (p *Processor) verifyCachedMsg(hash common.Hash) {
	verifys := p.blockContexts.getVerifyMsgCache(hash)
	if verifys != nil {
		for _, vmsg := range verifys.verifyMsgs {
			p.OnMessageVerify(vmsg)
		}
	}
	p.blockContexts.removeVerifyMsgCache(hash)
}

func (p *Processor) doVerify(cvm *model.ConsensusVerifyMessage, vctx *VerifyContext) (ret int8, err error) {
	blockHash := cvm.BlockHash
	if p.blockOnChain(blockHash) {
		err = fmt.Errorf("block already on chain")
		return
	}

	slot := vctx.GetSlotByHash(blockHash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}
	// Castor need to ignore his message
	if slot.castor.IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}
	bh := slot.BH
	gSeed := vctx.group.header.seed

	if err = vctx.baseCheck(bh, cvm.SI.GetID()); err != nil {
		return
	}
	// Check if the current node is in the ingot verifyGroup
	if !vctx.group.hasMember(p.GetMinerID()) {
		err = fmt.Errorf("don't belong to verifyGroup, gseed=%v, hash=%v, id=%v", gSeed, bh.Hash, p.GetMinerID())
		return
	}
	if !vctx.group.hasMember(cvm.SI.GetID()) {
		err = fmt.Errorf("sender doesn't belong the verifyGroup, gseed=%v, hash=%v, id=%v", gSeed, bh.Hash, cvm.SI.GetID())
		return
	}

	if !p.blockOnChain(vctx.prevBH.Hash) {
		err = fmt.Errorf("pre not on chain:hash=%v", vctx.prevBH.Hash)
		return
	}

	if cvm.GenHash() != cvm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", cvm.GenHash(), cvm.SI.DataHash)
		return
	}

	if _, same := p.blockContexts.isHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("the block of this height has been cast %v", bh.Height)
		return
	}

	pk := vctx.group.getMemberPubkey(cvm.SI.GetID())
	if !pk.IsValid() {
		err = fmt.Errorf("get member sign pubkey fail: gseed=%v, uid=%v", gSeed, cvm.SI.GetID())
		return
	}

	if !cvm.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail, gseed=%v, id=%v", gSeed, cvm.SI.GetID())
		return
	}
	if !groupsig.VerifySig(pk, vctx.prevBH.Random, cvm.RandomSign) {
		err = fmt.Errorf("verify random sign fail")
		return
	}

	ret, err = slot.AcceptVerifyPiece(cvm.SI.GetID(), cvm.SI.DataSign, cvm.RandomSign)
	vctx.increaseVerifyNum()
	if err != nil {
		return
	}
	if ret == pieceThreshold {
		p.reserveBlock(vctx, slot)
		vctx.increaseAggrNum()
	}
	return
}

// OnMessageVerify handles the verification messages from other members in the verifyGroup for a specified block proposal
// Note that, it will cache the messages if the corresponding proposal message doesn't come yet and trigger them as long as the condition met
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) (err error) {
	blockHash := cvm.BlockHash
	tlog := newHashTraceLog("OMV", blockHash, cvm.SI.GetID())
	traceLog := monitor.NewPerformTraceLogger("OnMessageVerify", blockHash, 0)

	var (
		ret  int8
		slot *SlotContext
	)
	defer func() {
		result := "unknown"
		if err != nil {
			result = err.Error()
		} else if slot != nil {
			result = slot.gSignGenerator.Brief()
		}
		tlog.logEnd("sender=%v, ret=%v %v", cvm.SI.GetID(), ret, result)
		traceLog.Log("result=%v, %v", ret, err)
	}()

	// Cache the message in case of absence of the proposal message
	vctx := p.blockContexts.getVctxByHash(blockHash)
	if vctx == nil {
		err = fmt.Errorf("verify context is nil, cache msg")
		p.blockContexts.addVerifyMsg(cvm)
		return
	}
	traceLog.SetHeight(vctx.castHeight)

	// Do the verification work
	ret, err = p.doVerify(cvm, vctx)

	slot = vctx.GetSlotByHash(blockHash)

	return
}

// OnMessageCastRewardSignReq handles reward transaction signature requests
// It signs the message if and only if the block of the transaction already added on chain,
// otherwise the message will be cached util the condition met
func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) error {
	return p.rewardHandler.OnMessageCastRewardSignReq(msg)
}

// OnMessageCastRewardSign receives signed messages for the reward transaction from verifyGroup members
// If threshold signature received and the verifyGroup signature recovered successfully, the node will submit the reward transaction to the pool
func (p *Processor) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) error {
	return p.rewardHandler.OnMessageCastRewardSign(msg)
}

// OnMessageReqProposalBlock handles block body request from the verify verifyGroup members
// It only happens in the proposal role and when the verifyGroup signature generated by the verify-verifyGroup
func (p *Processor) OnMessageReqProposalBlock(msg *model.ReqProposalBlock, sourceID string) (err error) {
	from := groupsig.ID{}
	from.SetAddrString(sourceID)
	tLog := newHashTraceLog("OMRPB", msg.Hash, from)
	tLog.logStart("sender=%v", sourceID)

	defer func() {
		if err != nil {
			tLog.log("result:%v", err)
		} else {
			tLog.log("result: success")
		}
	}()

	pb := p.blockContexts.getProposed(msg.Hash)
	if pb == nil || pb.block == nil {
		err = fmt.Errorf("block is nil")
		return
	}
	group := p.groupReader.getGroupBySeed(pb.block.Header.Group)
	if group == nil {
		err = fmt.Errorf("get verifyGroup nil:%v", pb.block.Header.Group)
		return
	}

	if !group.hasMember(msg.SI.GetID()) {
		err = fmt.Errorf("reqProposa sender doesn't belong the verifyGroup, gseed=%v, hash=%v, id=%v",
			group.header.Seed(), pb.block.Header.Hash, msg.SI.GetID())
		return
	}

	if msg.GenHash() != msg.SI.DataHash {
		err = fmt.Errorf("reqProposa msg genHash %v diff from si.DataHash %v", msg.GenHash(), msg.SI.DataHash)
		return
	}

	pk := group.getMemberPubkey(msg.SI.GetID())
	if !pk.IsValid() {
		err = fmt.Errorf("reqProposa get member sign pubkey fail: gseed=%v, uid=%v", group.header.Seed(), msg.SI.GetID())
		return
	}

	if !msg.VerifySign(pk) {
		err = fmt.Errorf("reqProposa verify sign fail, gseed=%v, id=%v", group.header.Seed(), msg.SI.GetID())
		return
	}

	exist, size := pb.containsOrAddRequested(msg.SI.GetID())

	if exist {
		err = fmt.Errorf("reqProposa sender %v has already requested the block", msg.SI.GetID())
		return
	}

	// Only response to limited members of the verifyGroup in case of network traffic
	if size > pb.maxResponseCount {
		err = fmt.Errorf("response count exceed:%v %v", size, pb.maxResponseCount)
		return
	}

	//err = fmt.Sprintf("response txs size %v", len(pb.block.Transactions))

	m := &model.ResponseProposalBlock{
		Hash:         pb.block.Header.Hash,
		Transactions: pb.block.Transactions,
	}

	p.NetServer.ResponseProposalBlock(m, sourceID)

	return
}

// OnMessageResponseProposalBlock handles block body response from proposal node
// It only happens in the verify roles and after block body request to the proposal node
// It will add the block on chain and then broadcast
func (p *Processor) OnMessageResponseProposalBlock(msg *model.ResponseProposalBlock) (err error) {
	tLog := newHashTraceLog("OMRSPB", msg.Hash, groupsig.ID{})
	tLog.logStart("")

	defer func() {
		if err != nil {
			tLog.log("result:%v", err)
		} else {
			tLog.log("result: success")
		}
	}()

	if p.blockOnChain(msg.Hash) {
		err = fmt.Errorf("block onchain")
		return
	}
	vctx := p.blockContexts.getVctxByHash(msg.Hash)
	if vctx == nil {
		err = fmt.Errorf("vctx is nil")
		return
	}
	slot := vctx.GetSlotByHash(msg.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}
	block := types.Block{Header: slot.BH, Transactions: msg.Transactions}
	aggSign := slot.GetAggregatedSign()
	if aggSign == nil {
		err = fmt.Errorf("aggregated signature is nil")
		return
	}
	err = p.onBlockSignAggregation(&block, *aggSign, slot.rSignGenerator.GetGroupSign())
	if err != nil {
		slot.setSlotStatus(slFailed)
		err = fmt.Errorf("on block fail err=%v", err)
		return
	}
	return
}
