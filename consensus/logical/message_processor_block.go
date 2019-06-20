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
	"bytes"
	"fmt"
	"math"
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
	groupID := groupsig.DeserializeID(bh.GroupID)

	// trigger the cached messages from other members that come ahead of the proposal message
	defer func() {
		if ok {
			go func() {
				verifys := p.blockContexts.getVerifyMsgCache(bh.Hash)
				if verifys != nil {
					for _, vmsg := range verifys.verifyMsgs {
						p.OnMessageVerify(vmsg)
					}
				}
				p.blockContexts.removeVerifyMsgCache(bh.Hash)
			}()
		}
	}()

	// if the block already added on chain, then ignore this message
	if p.blockOnChain(bh.Hash) {
		err = fmt.Errorf("block onchain already")
		return
	}

	// check expire time, fail fast if expired
	expireTime := expireTime(bh, preBH)
	if p.ts.NowAfter(expireTime) {
		err = fmt.Errorf("cast verify expire, gid=%v, preTime %v, expire %v", groupID.ShortS(), preBH.CurTime, expireTime)
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

	group := p.GetGroup(groupID)
	if group == nil {
		err = fmt.Errorf("group is nil:groupID=%v", groupID.GetHexString())
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
	castorDO := p.minerReader.getProposeMiner(castor)
	if castorDO == nil {
		err = fmt.Errorf("castorDO nil id=%v", castor.ShortS())
		return
	}
	pk := castorDO.PK

	// check the signature of the proposal
	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail")
		return
	}

	// check if the blockHeader is legal
	ok, _, err = p.isCastLegal(bh, preBH)
	if !ok {
		return
	}

	// full book verification
	existHash := p.proveChecker.genProveHash(bh.Height, preBH.Random, p.GetMinerID())
	if msg.ProveHash != existHash {
		err = fmt.Errorf("check p rove hash fail, receive hash=%v, exist hash=%v", msg.ProveHash.ShortS(), existHash.ShortS())
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

	skey := p.getSignKey(groupID)
	var cvm model.ConsensusVerifyMessage
	cvm.BlockHash = bh.Hash

	// sign the message and send to other members in the group
	if cvm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &cvm) {
		cvm.GenRandomSign(skey, vctx.prevBH.Random)
		p.NetServer.SendVerifiedCast(&cvm, groupID)
		slot.setSlotStatus(slSigned)
		p.blockContexts.attachVctx(bh, vctx)
		vctx.markSignedBlock(bh)
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
		Verifier: groupsig.DeserializeID(bh.GroupID).GetHexString(),
		Ext:      fmt.Sprintf("external:qn:%v,totalQN:%v", 0, bh.TotalQN),
	}
	group := p.GetGroup(groupsig.DeserializeID(bh.GroupID))
	var err error
	if group == nil {
		err = fmt.Errorf("GetSelfGroup failed")
		return
	}

	detalHeight := int(bh.Height - p.MainChain.Height())
	if common.AbsInt(detalHeight) < 100 && monitor.Instance.IsFirstNInternalNodesInGroup(group.GetMembers(), 3) {
		monitor.Instance.AddLogIfNotInternalNodes(le)
	}
	mtype := "OMC"
	blog := newBizLog(mtype)

	si := &ccm.SI
	tlog := newHashTraceLog(mtype, bh.Hash, si.GetID())
	castor := groupsig.DeserializeID(bh.Castor)
	groupID := groupsig.DeserializeID(bh.GroupID)

	tlog.logStart("%v:height=%v, castor=%v", mtype, bh.Height, castor.ShortS())
	blog.debug("proc(%v) begin hash=%v, height=%v, sender=%v, castor=%v, groupID=%v", p.getPrefix(), bh.Hash.ShortS(), bh.Height, si.GetID().ShortS(), castor.ShortS(), groupID.ShortS())

	defer func() {
		result := "signed"
		if err != nil {
			result = err.Error()
		}
		tlog.logEnd("%v:height=%v, hash=%v, preHash=%v,groupID=%v, result=%v", mtype, bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupID.ShortS(), result)
		blog.debug("height=%v, hash=%v, preHash=%v, groupID=%v, result=%v", bh.Height, bh.Hash.ShortS(), bh.PreHash.ShortS(), groupID.ShortS(), result)
		traceLog.Log("PreHash=%v,castor=%v,result=%v", bh.PreHash.ShortS(), ccm.SI.GetID().ShortS(), result)
	}()
	if ccm.GenHash() != ccm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", ccm.GenHash().ShortS(), ccm.SI.DataHash.ShortS())
		return
	}
	// Castor need to ignore his message
	if castor.IsEqual(p.GetMinerID()) && si.GetID().IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}

	// Check if the current node is in the group
	if !p.IsMinerGroup(groupID) {
		err = fmt.Errorf("don't belong to group, gid=%v, hash=%v, id=%v", groupID.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
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
	if slot.castor.IsEqual(p.GetMinerID()) && cvm.SI.GetID().IsEqual(p.GetMinerID()) {
		err = fmt.Errorf("ignore self message")
		return
	}
	bh := slot.BH
	groupID := vctx.group.GroupID

	if err = vctx.baseCheck(bh, cvm.SI.GetID()); err != nil {
		return
	}

	// Check if the current node is in the ingot group
	if !p.IsMinerGroup(groupID) {
		err = fmt.Errorf("don't belong to group, gid=%v, hash=%v, id=%v", groupID.ShortS(), bh.Hash.ShortS(), p.GetMinerID().ShortS())
		return
	}
	if !p.blockOnChain(vctx.prevBH.Hash) {
		err = fmt.Errorf("pre not on chain:hash=%v", vctx.prevBH.Hash.ShortS())
		return
	}

	if cvm.GenHash() != cvm.SI.DataHash {
		err = fmt.Errorf("msg genHash %v diff from si.DataHash %v", cvm.GenHash().ShortS(), cvm.SI.DataHash.ShortS())
		return
	}

	if _, same := p.blockContexts.isHeightCasted(bh.Height, bh.PreHash); same {
		err = fmt.Errorf("the block of this height has been cast %v", bh.Height)
		return
	}

	pk, ok := p.getMemberSignPubKey(model.NewGroupMinerID(groupID, cvm.SI.GetID()))
	if !ok {
		err = fmt.Errorf("get member sign pubkey fail: gid=%v, uid=%v", groupID.ShortS(), cvm.SI.GetID().ShortS())
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

// OnMessageVerify handles the verification messages from other members in the group for a specified block proposal
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
		tlog.logEnd("sender=%v, ret=%v %v", cvm.SI.GetID().ShortS(), ret, result)
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
	gid := groupsig.DeserializeID(bh.GroupID)
	group := p.GetGroup(gid)
	reward := &msg.Reward
	if group == nil {
		err = fmt.Errorf("group is nil")
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
		err = fmt.Errorf("alreayd sent reward trans")
		return
	}

	if !bytes.Equal(bh.GroupID, reward.GroupID) {
		err = fmt.Errorf("groupID error %v %v", bh.GroupID, reward.GroupID)
		return
	}
	if !slot.hasSignedTxHash(reward.TxHash) {

		genBonus, _, err2 := p.MainChain.GetBonusManager().GenerateBonus(reward.TargetIds, bh.Hash, bh.GroupID, model.Param.VerifyBonus)
		if err2 != nil {
			err = err2
			return
		}
		if genBonus.TxHash != reward.TxHash {
			err = fmt.Errorf("bonus txHash diff %v %v", genBonus.TxHash.ShortS(), reward.TxHash.ShortS())
			return
		}

		if len(msg.Reward.TargetIds) != len(msg.SignedPieces) {
			err = fmt.Errorf("targetId len differ from signedpiece len %v %v", len(msg.Reward.TargetIds), len(msg.SignedPieces))
			return
		}

		mpk, ok := p.getMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))
		if !ok {
			err = fmt.Errorf("getMemberSignPubKey not ok, ask id %v", gid.ShortS())
			return
		}
		if !msg.VerifySign(mpk) {
			err = fmt.Errorf("verify sign fail, gid=%v, uid=%v", gid.ShortS(), msg.SI.GetID().ShortS())
			return
		}

		// Reuse the original generator to avoid duplicate signature verification
		gSignGener := slot.gSignGenerator

		for idx, idIndex := range msg.Reward.TargetIds {
			id := group.GetMemberID(int(idIndex))
			sign := msg.SignedPieces[idx]

			// If there is no local id signature, you need to verify the signature.
			if sig, ok := gSignGener.GetWitness(id); !ok {
				pk, exist := p.getMemberSignPubKey(model.NewGroupMinerID(gid, id))
				if !exist {
					continue
				}
				if !groupsig.VerifySig(pk, bh.Hash.Bytes(), sign) {
					err = fmt.Errorf("verify member sign fail, id=%v", id.ShortS())
					return
				}
				// Join the generator
				gSignGener.AddWitnessForce(id, sign)
			} else { // If the signature of the id already exists locally, just judge whether it is the same as the local signature.
				if !sign.IsEqual(sig) {
					err = fmt.Errorf("member sign different id=%v", id.ShortS())
					return
				}
			}
		}

		if !gSignGener.Recovered() {
			err = fmt.Errorf("recover group sign fail")
			return
		}

		bhSign := groupsig.DeserializeSign(bh.Signature)
		if !gSignGener.GetGroupSign().IsEqual(*bhSign) {
			err = fmt.Errorf("recovered sign differ from bh sign, recover %v, bh %v", gSignGener.GetGroupSign().ShortS(), bhSign.ShortS())
			return
		}

		slot.addSignedTxHash(reward.TxHash)
	}

	send = true
	// Sign yourself
	signMsg := &model.CastRewardTransSignMessage{
		ReqHash:   reward.TxHash,
		BlockHash: reward.BlockHash,
		GroupID:   gid,
		Launcher:  msg.SI.GetID(),
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(gid))
	if signMsg.GenSign(ski, signMsg) {
		p.NetServer.SendCastRewardSign(signMsg)
	} else {
		err = fmt.Errorf("signCastRewardReq genSign fail, id=%v, sk=%v, %v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(gid))
	}
	return
}

// OnMessageCastRewardSignReq handles bonus transaction signature requests
// It signs the message if and only if the block of the transaction already added on chain,
// otherwise the message will be cached util the condition met
func (p *Processor) OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	mtype := "OMCRSR"
	blog := newBizLog(mtype)
	reward := &msg.Reward
	tlog := newHashTraceLog("OMCRSR", reward.BlockHash, msg.SI.GetID())
	blog.debug("begin, sender=%v, blockHash=%v, txHash=%v", msg.SI.GetID().ShortS(), reward.BlockHash.ShortS(), reward.TxHash.ShortS())
	tlog.logStart("txHash=%v", reward.TxHash.ShortS())

	var (
		send bool
		err  error
	)

	defer func() {
		tlog.logEnd("txHash=%v, %v %v", reward.TxHash.ShortS(), send, err)
		blog.debug("blockHash=%v, txHash=%v, result=%v %v", reward.BlockHash.ShortS(), reward.TxHash.ShortS(), send, err)
	}()

	// At this point the block is not necessarily on the chain
	// in case that, the message will be cached
	bh := p.getBlockHeaderByHash(reward.BlockHash)
	if bh == nil {
		err = fmt.Errorf("future reward request receive and cached, hash=%v", reward.BlockHash.ShortS())
		msg.ReceiveTime = time.Now()
		p.futureRewardReqs.addMessage(reward.BlockHash, msg)
		return
	}

	send, err = p.signCastRewardReq(msg, bh)
	return
}

// OnMessageCastRewardSign receives signed messages for the bonus transaction from group members
// If threshold signature received and the group signature recovered successfully, the node will submit the bonus transaction to the pool
func (p *Processor) OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage) {
	mtype := "OMCRS"
	blog := newBizLog(mtype)

	blog.debug("begin, sender=%v, reqHash=%v", msg.SI.GetID().ShortS(), msg.ReqHash.ShortS())
	tlog := newHashTraceLog(mtype, msg.BlockHash, msg.SI.GetID())

	tlog.logStart("txHash=%v", msg.ReqHash.ShortS())

	var (
		send bool
		err  error
	)

	defer func() {
		tlog.logEnd("bonus send:%v, ret:%v", send, err)
		blog.debug("blockHash=%v, send=%v, result=%v", msg.BlockHash.ShortS(), send, err)
	}()

	// If the block related to the bonus transaction is not on the chain, then drop the messages
	// This could happened after one fork process
	bh := p.getBlockHeaderByHash(msg.BlockHash)
	if bh == nil {
		err = fmt.Errorf("block not exist, hash=%v", msg.BlockHash.ShortS())
		return
	}

	gid := groupsig.DeserializeID(bh.GroupID)
	group := p.GetGroup(gid)
	if group == nil {
		err = fmt.Errorf("group is nil")
		return
	}
	pk, ok := p.getMemberSignPubKey(model.NewGroupMinerID(gid, msg.SI.GetID()))
	if !ok {
		err = fmt.Errorf("getMemberSignPubKey not ok, ask id %v", gid.ShortS())
		return
	}
	if !msg.VerifySign(pk) {
		err = fmt.Errorf("verify sign fail")
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

	// Try to add the signature to the group sign generator of the slot related to the block
	accept, recover := slot.AcceptRewardPiece(&msg.SI)
	blog.debug("slot acceptRewardPiece %v %v status %v", accept, recover, slot.GetSlotStatus())

	// Add the bonus transaction to pool if the signature is accepted and the group signature is recovered
	if accept && recover && slot.statusTransform(slRewardSignReq, slRewardSent) {
		_, err2 := p.MainChain.GetTransactionPool().AddTransaction(slot.rewardTrans)
		send = true
		err = fmt.Errorf("add rewardTrans to txPool, txHash=%v, ret=%v", slot.rewardTrans.Hash.ShortS(), err2)

	} else {
		err = fmt.Errorf("accept %v, recover %v, %v", accept, recover, slot.rewardGSignGen.Brief())
	}
}

// OnMessageReqProposalBlock handles block body request from the verify group members
// It only happens in the proposal role and when the group signature generated by the verify-group
func (p *Processor) OnMessageReqProposalBlock(msg *model.ReqProposalBlock, sourceID string) {
	blog := newBizLog("OMRPB")
	blog.debug("hash %v", msg.Hash.ShortS())

	from := groupsig.ID{}
	from.SetHexString(sourceID)
	tlog := newHashTraceLog("OMRPB", msg.Hash, from)

	var s string
	defer func() {
		tlog.log("result:%v", s)
	}()

	pb := p.blockContexts.getProposed(msg.Hash)
	if pb == nil || pb.block == nil {
		s = fmt.Sprintf("block is nil")
		blog.warn("block is nil hash=%v", msg.Hash.ShortS())
		return
	}

	if pb.maxResponseCount == 0 {
		gid := groupsig.DeserializeID(pb.block.Header.GroupID)
		group, err := p.globalGroups.GetGroupByID(gid)
		if err != nil {
			s = fmt.Sprintf("get group error")
			blog.error("block proposal response, GetGroupByID err= %v,  hash=%v", err, msg.Hash.ShortS())
			return
		}

		pb.maxResponseCount = uint(math.Ceil(float64(group.GetMemberCount()) / 3))
	}

	// Only response to limited members of the group in case of network traffic
	if pb.responseCount >= pb.maxResponseCount {
		s = fmt.Sprintf("response count exceed")
		blog.debug("block proposal response count >= maxResponseCount(%v), not response, hash=%v", pb.maxResponseCount, msg.Hash.ShortS())
		return
	}

	pb.responseCount++

	s = fmt.Sprintf("response txs size %v", len(pb.block.Transactions))
	blog.debug("block proposal response, count=%v, max count=%v, hash=%v", pb.responseCount, pb.maxResponseCount, msg.Hash.ShortS())

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
	blog := newBizLog("OMRSPB")
	blog.debug("hash %v", msg.Hash.ShortS())

	tlog := newHashTraceLog("OMRSPB", msg.Hash, groupsig.ID{})

	var s string
	defer func() {
		tlog.log("result:%v", s)
	}()

	if p.blockOnChain(msg.Hash) {
		s = "block onchain"
		return
	}
	vctx := p.blockContexts.getVctxByHash(msg.Hash)
	if vctx == nil {
		blog.warn("verify context is nil, cache msg")
		s = "vctx is nil"
		return
	}
	slot := vctx.GetSlotByHash(msg.Hash)
	if slot == nil {
		blog.warn("slot is nil")
		s = "slot is nil"
		return
	}
	block := types.Block{Header: slot.BH, Transactions: msg.Transactions}
	err := p.onBlockSignAggregation(&block, slot.gSignGenerator.GetGroupSign(), slot.rSignGenerator.GetGroupSign())
	if err != nil {
		blog.error("onBlockSignAggregation fail: %v", err)
		slot.setSlotStatus(slFailed)
		s = fmt.Sprintf("on block fail err=%v", err)
		return
	}
	s = "success"
}
