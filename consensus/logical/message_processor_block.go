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
	"math"
	"sync/atomic"
	"time"

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

	// full book verification
	existHash := p.proveChecker.genProveHash(bh.Height, preBH.Random, p.GetMinerID())
	if msg.ProveHash != existHash {
		err = fmt.Errorf("check p rove hash fail, receive hash=%v, exist hash=%v", msg.ProveHash, existHash)
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
func (p *Processor) OnMessageCast(ccm *model.ConsensusCastMessage) {
	bh := &ccm.BH
	traceLog := monitor.NewPerformTraceLogger("OnMessageCast", bh.Hash, bh.Height)

	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeProposal,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: ccm.SI.GetID().GetHexString(),
		Verifier: bh.Group.Hex(),
		Ext:      fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	}
	group := p.groupReader.getGroupBySeed(bh.Group)
	var err error
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

	tlog.logStart("%height=%v, castor=%v", bh.Height, castor)

	defer func() {
		result := "signed"
		if err != nil {
			result = err.Error()
		}
		tlog.logEnd("height=%v, preHash=%v, gseed=%v, result=%v", bh.Height, bh.PreHash, bh.Group, result)
		traceLog.Log("PreHash=%v,castor=%v,result=%v", bh.PreHash, ccm.SI.GetID(), result)
	}()
	if ccm.GenHash() != ccm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", ccm.GenHash(), ccm.SI.DataHash)
		return
	}
	// Castor need to ignore his message
	if castor.IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}

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

	preBH := p.getBlockHeaderByHash(bh.PreHash)

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
	gSeed := vctx.group.header.Seed()

	if err = vctx.baseCheck(bh, cvm.SI.GetID()); err != nil {
		return
	}
	// Check if the current node is in the ingot verifyGroup
	if !vctx.group.hasMember(p.GetMinerID()) {
		err = fmt.Errorf("don't belong to verifyGroup, gseed=%v, hash=%v, id=%v", gSeed, bh.Hash, p.GetMinerID())
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
		err = fmt.Errorf("verify sign fail")
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
func (p *Processor) OnMessageVerify(cvm *model.ConsensusVerifyMessage) {
	blockHash := cvm.BlockHash
	tlog := newHashTraceLog("OMV", blockHash, cvm.SI.GetID())
	traceLog := monitor.NewPerformTraceLogger("OnMessageVerify", blockHash, 0)

	var (
		err  error
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

func (p *Processor) signCastRewardReq(msg *model.CastRewardTransSignReqMessage, bh *types.BlockHeader) (send bool, err error) {
	gSeed := bh.Group
	reward := &msg.Reward
	group := p.groupReader.getGroupBySeed(gSeed)
	if group == nil {
		err = fmt.Errorf("verifyGroup is nil")
		return
	}

	vctx := p.blockContexts.getVctxByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil,%v height=%v", vctx == nil, bh.Height)
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}

	// A dividend transaction has been sent, no longer signed for this
	if slot.IsRewardSent() {
		err = fmt.Errorf("already sent reward trans")
		return
	}

	if gSeed != reward.Group {
		err = fmt.Errorf("groupSeed not equal %v %v", bh.Group, reward.Group)
		return
	}
	if !slot.hasSignedTxHash(reward.TxHash) {
		rewardShare := p.MainChain.GetRewardManager().CalculateCastRewardShare(bh.Height, bh.GasFee)
		genReward, _, err2 := p.MainChain.GetRewardManager().GenerateReward(reward.TargetIds, bh.Hash, bh.Group, rewardShare.TotalForVerifier(), rewardShare.ForRewardTxPacking)
		if err2 != nil {
			err = err2
			return
		}
		if genReward.TxHash != reward.TxHash {
			err = fmt.Errorf("reward txHash diff %v %v", genReward.TxHash, reward.TxHash)
			return
		}

		if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
			err = fmt.Errorf("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
			return
		}

		mpk := group.getMemberPubkey(msg.SI.GetID())
		if !msg.VerifySign(mpk) {
			err = fmt.Errorf("verify sign fail, gseed=%v, uid=%v", gSeed, msg.SI.GetID())
			return
		}

		// Reuse the original generator to avoid duplicate signature verification
		gSignGenerator := slot.gSignGenerator

		for idx, idIndex := range msg.Reward.TargetIds {
			mem := group.getMemberAt(int(idIndex))
			if mem == nil {
				err = fmt.Errorf("member not exist, idx %v", idIndex)
				return
			}
			sign := msg.SignedPieces[idx]

			// If there is no local id signature, you need to verify the signature.
			if sig, ok := gSignGenerator.GetWitness(mem.id); !ok {
				pk := group.getMemberPubkey(mem.id)
				if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
					err = fmt.Errorf("verify member sign fail, id=%v", mem.id)
					return
				}
				// Join the generator
				gSignGenerator.AddWitnessForce(mem.id, sign)
			} else { // If the signature of the id already exists locally, just judge whether it is the same as the local signature.
				if !sign.IsEqual(sig) {
					err = fmt.Errorf("member sign different id=%v", mem.id)
					return
				}
			}
		}

		if !gSignGenerator.Recovered() {
			err = fmt.Errorf("recover verifyGroup sign fail")
			return
		}

		bhSign := groupsig.DeserializeSign(bh.Signature)
		aggSign := slot.GetAggregatedSign()
		if aggSign == nil {
			err = fmt.Errorf("obtain the Aggregated signature fail")
			return
		}
		if !aggSign.IsEqual(*bhSign) {
			err = fmt.Errorf("aggregated sign differ from bh sign, aggregated %v, bh %v", aggSign, bhSign)
			return
		}

		slot.addSignedTxHash(reward.TxHash)
	}

	send = true
	// Sign yourself
	signMsg := &model.CastRewardTransSignMessage{
		ReqHash:   reward.TxHash,
		BlockHash: reward.BlockHash,
		GSeed:     gSeed,
		Launcher:  msg.SI.GetID(),
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), p.groupReader.getGroupSignatureSeckey(gSeed))
	if signMsg.GenSign(ski, signMsg) {
		p.NetServer.SendCastRewardSign(signMsg)
	} else {
		err = fmt.Errorf("signCastRewardReq genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
	}
	return
}

// OnMessageCastRewardSignReq handles reward transaction signature requests
// It signs the message if and only if the block of the transaction already added on chain,
// otherwise the message will be cached util the condition met
func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mType := "OMCRSR"
	reward := &msg.Reward
	tLog := newHashTraceLog(mType, reward.BlockHash, msg.SI.GetID())
	tLog.logStart("txHash=%v", reward.TxHash)

	var (
		send bool
		err  error
	)

	defer func() {
		tLog.logEnd("txHash=%v, %v %v", reward.TxHash, send, err)
	}()

	// At this point the block is not necessarily on the chain
	// in case that, the message will be cached
	bh := p.getBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		err = fmt.Errorf("future reward request receive and cached, hash=%v", reward.BlockHash)
		msg.ReceiveTime = time.Now()
		p.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}

	send, err = p.signCastRewardReq(msg, bh)
	return
}

// OnMessageCastRewardSign receives signed messages for the reward transaction from verifyGroup members
// If threshold signature received and the verifyGroup signature recovered successfully, the node will submit the reward transaction to the pool
func (p *Processor) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) {
	mType := "OMCRS"

	tLog := newHashTraceLog(mType, msg.BlockHash, msg.SI.GetID())

	tLog.logStart("txHash=%v", msg.ReqHash)

	var (
		send bool
		err  error
	)

	defer func() {
		tLog.logEnd("reward send:%v, ret:%v", send, err)
	}()

	// If the block related to the reward transaction is not on the chain, then drop the messages
	// This could happened after one fork process
	bh := p.getBlockHeaderByHash(msg.BlockHash)
	if bh == nil {
		err = fmt.Errorf("block not exist, hash=%v", msg.BlockHash)
		return
	}

	gSeed := bh.Group
	group := p.groupReader.getGroupBySeed(gSeed)
	if group == nil {
		err = fmt.Errorf("verifyGroup is nil")
		return
	}
	pk := group.getMemberPubkey(msg.SI.GetID())
	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail, pk=%v, id=%v", pk, msg.SI.GetID())
		return
	}

	vctx := p.blockContexts.getVctxByHeight(bh.Height)
	if vctx == nil || vctx.prevBH.Hash != bh.PreHash {
		err = fmt.Errorf("vctx is nil")
		return
	}

	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		err = fmt.Errorf("slot is nil")
		return
	}

	// Try to add the signature to the verifyGroup sign generator of the slot related to the block
	accept, recover := slot.AcceptRewardPiece(&msg.SI)

	// Add the reward transaction to pool if the signature is accepted and the verifyGroup signature is recovered
	if accept && recover && slot.statusTransform(slRewardSignReq, slRewardSent) {
		_, err2 := p.MainChain.GetTransactionPool().AddTransaction(slot.rewardTrans)
		send = true
		err = fmt.Errorf("add rewardTrans to txPool, txHash=%v, ret=%v", slot.rewardTrans.Hash, err2)

	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}

// OnMessageReqProposalBlock handles block body request from the verify verifyGroup members
// It only happens in the proposal role and when the verifyGroup signature generated by the verify-verifyGroup
func (p *Processor) OnMessageReqProposalBlock(msg *model.ReqProposalBlock, sourceID string) {
	from := groupsig.ID{}
	from.SetHexString(sourceID)
	tLog := newHashTraceLog("OMRPB", msg.Hash, from)
	tLog.logStart("sender=%v", sourceID)

	var s string
	defer func() {
		tLog.log("result:%v", s)
	}()

	pb := p.blockContexts.getProposed(msg.Hash)
	if pb == nil || pb.block == nil {
		s = fmt.Sprintf("block is nil")
		return
	}

	if pb.maxResponseCount == 0 {
		group := p.groupReader.getGroupBySeed(pb.block.Header.Group)
		if group == nil {
			s = fmt.Sprintf("get verifyGroup nil:%v", pb.block.Header.Group)
			return
		}

		pb.maxResponseCount = uint64(math.Ceil(float64(group.memberSize()) / 3))
	}

	// Only response to limited members of the verifyGroup in case of network traffic
	if atomic.AddUint64(&pb.responseCount, 1) > pb.maxResponseCount {
		s = fmt.Sprintf("response count exceed:%v %v", pb.responseCount, pb.maxResponseCount)
		return
	}
	s = fmt.Sprintf("response txs size %v", len(pb.block.Transactions))

	m := &model.ResponseProposalBlock{
		Hash:         pb.block.Header.Hash,
		Transactions: pb.block.Transactions,
	}

	p.NetServer.ResponseProposalBlock(m, sourceID)
}

// OnMessageResponseProposalBlock handles block body response from proposal node
// It only happens in the verify roles and after block body request to the proposal node
// It will add the block on chain and then broadcast
func (p *Processor) OnMessageResponseProposalBlock(msg *model.ResponseProposalBlock) {
	tLog := newHashTraceLog("OMRSPB", msg.Hash, groupsig.ID{})
	tLog.logStart("")

	var s string
	defer func() {
		tLog.log("result:%v", s)
	}()

	if p.blockOnChain(msg.Hash) {
		s = "block onchain"
		return
	}
	vctx := p.blockContexts.getVctxByHash(msg.Hash)
	if vctx == nil {
		s = "vctx is nil"
		return
	}
	slot := vctx.GetSlotByHash(msg.Hash)
	if slot == nil {
		s = "slot is nil"
		return
	}
	block := types.Block{Header: slot.BH, Transactions: msg.Transactions}
	aggSign := slot.GetAggregatedSign()
	if aggSign == nil {
		s = "aggregated signature is nil"
		return
	}
	err := p.onBlockSignAggregation(&block, *aggSign, slot.rSignGenerator.GetGroupSign())
	if err != nil {
		slot.setSlotStatus(slFailed)
		s = fmt.Sprintf("on block fail err=%v", err)
		return
	}
	s = "success"
}
