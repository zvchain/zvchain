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
	"strings"
	"sync"

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
)

// triggerCastCheck trigger once to check if you are next ingot group
func (p *Processor) triggerCastCheck() {
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) calcVerifyGroupFromCache(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash = calcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromCache(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromCache height=%v, err: %v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) calcVerifyGroupFromChain(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash = calcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromChain(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromChain height=%v, err:%v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, height uint64) *net.GroupBrief {
	nextID := p.calcVerifyGroupFromCache(bh, height)
	if nextID == nil {
		return nil
	}
	group := p.GetGroup(*nextID)
	g := &net.GroupBrief{
		Gid:    *nextID,
		MemIds: group.GetMembers(),
	}
	return g
}

// reserveBlock reserves the block in the context utils it can be broadcast
func (p *Processor) reserveBlock(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.debug("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash.ShortS(), slot.GetSlotStatus())

	traceLog := monitor.NewPerformTraceLogger("reserveBlock", bh.Hash, bh.Height)
	traceLog.SetParent("OnMessageVerify")
	defer traceLog.Log("threshold sign cost %v", p.ts.Now().Local().Sub(bh.CurTime.Local()).String())

	if slot.IsRecovered() {
		//vctx.markCastSuccess() //onBlockAddSuccess方法中也mark了，该处调用是异步的
		p.blockContexts.addReservedVctx(vctx)
		if !p.tryNotify(vctx) {
			blog.warn("reserved, height=%v", vctx.castHeight)
		}
	}

	return
}

func (p *Processor) tryNotify(vctx *VerifyContext) bool {
	if sc := vctx.checkNotify(); sc != nil {
		bh := sc.BH
		tlog := newHashTraceLog("tryNotify", bh.Hash, p.GetMinerID())
		tlog.log("try broadcast, height=%v, totalQN=%v, consuming %vs", bh.Height, bh.TotalQN, p.ts.Since(bh.CurTime))

		// Add on chain and out-of-group broadcasting
		p.consensusFinalize(vctx, sc)

		p.blockContexts.removeReservedVctx(vctx.castHeight)
		return true
	}
	return false
}

func (p *Processor) onBlockSignAggregation(block *types.Block, sign groupsig.Signature, random groupsig.Signature) error {

	if block == nil {
		return fmt.Errorf("block is nil")
	}
	block.Header.Signature = sign.Serialize()
	block.Header.Random = random.Serialize()

	r := p.doAddOnChain(block)

	// Fork adjustment or add on chain failure does not take the logic below
	if r != int8(types.AddBlockSucc) {
		return fmt.Errorf("onchain result %v", r)
	}

	bh := block.Header
	tlog := newHashTraceLog("onBlockSignAggregation", bh.Hash, p.GetMinerID())

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}
	gb := p.spreadGroupBrief(bh, bh.Height+1)
	if gb == nil {
		return fmt.Errorf("next group is nil")
	}
	p.NetServer.BroadcastNewBlock(cbm, gb)
	tlog.log("broadcasted height=%v, consuming %vs", bh.Height, p.ts.Since(bh.CurTime))

	// Send info
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeBlockBroadcast,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: groupsig.DeserializeID(bh.Castor).GetHexString(),
		Verifier: gb.Gid.GetHexString(),
	}
	monitor.Instance.AddLog(le)
	return nil
}

// consensusFinalize represents the final stage of the consensus process.
// It firstly verifies the group signature and then requests the block body from proposer
func (p *Processor) consensusFinalize(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH

	var result string

	traceLog := monitor.NewPerformTraceLogger("consensusFinalize", bh.Hash, bh.Height)
	traceLog.SetParent("OnMessageVerify")
	defer func() {
		traceLog.Log("result=%v. consensusFinalize cost %v", result, p.ts.Now().Local().Sub(bh.CurTime.Local()).String())
	}()
	blog := newBizLog("consensusFinalize-" + bh.Hash.ShortS())

	// Already on blockchain
	if p.blockOnChain(bh.Hash) {
		blog.warn("block already onchain!")
		return
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeID(bh.GroupID))

	// Group signature verification passed
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) {
		blog.error("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	// Ask the proposer for a complete block
	msg := &model.ReqProposalBlock{
		Hash: bh.Hash,
	}
	tlog := newHashTraceLog("consensusFinalize", bh.Hash, p.GetMinerID())
	tlog.log("send ReqProposalBlock msg to %v", slot.castor.ShortS())
	p.NetServer.ReqProposalBlock(msg, slot.castor.GetHexString())

	result = fmt.Sprintf("Request block body from %v", slot.castor.GetHexString())

	slot.setSlotStatus(slSuccess)
	vctx.markNotified()
	vctx.successSlot = slot
	return
}

// blockProposal starts a block proposing process
func (p *Processor) blockProposal() {
	blog := newBizLog("blockProposal")
	top := p.MainChain.QueryTopBlock()
	worker := p.getVrfWorker()

	traceLogger := monitor.NewPerformTraceLogger("blockProposal", common.Hash{}, worker.castHeight)

	if worker.getBaseBH().Hash != top.Hash {
		blog.warn("vrf baseBH differ from top!")
		return
	}
	if worker.isProposed() || worker.isSuccess() {
		blog.debug("vrf worker proposed/success, status %v", worker.getStatus())
		return
	}
	height := worker.castHeight

	if !p.ts.NowAfter(worker.baseBH.CurTime) {
		blog.error("not the time!now=%v, pre=%v, height=%v", p.ts.Now(), worker.baseBH.CurTime, height)
		return
	}

	totalStake := p.minerReader.getTotalStake(worker.baseBH.Height, false)
	blog.debug("totalStake height=%v, stake=%v", height, totalStake)
	pi, qn, err := worker.Prove(totalStake)
	if err != nil {
		blog.warn("vrf prove not ok! %v", err)
		return
	}

	if height > 1 && p.proveChecker.proveExists(pi) {
		blog.warn("vrf prove exist, not proposal")
		return
	}

	if worker.timeout() {
		blog.warn("vrf worker timeout")
		return
	}

	gb := p.spreadGroupBrief(top, height)
	if gb == nil {
		blog.error("spreadGroupBrief nil, bh=%v, height=%v", top.Hash.ShortS(), height)
		return
	}
	gid := gb.Gid

	var (
		block         *types.Block
		proveHashs    []common.Hash
		proveTraceLog *monitor.PerformTraceLogger
	)
	// Parallelize the CastBlock and genProveHashs process
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		block = p.MainChain.CastBlock(uint64(height), pi, qn, p.GetMinerID().Serialize(), gid.Serialize())
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		//生成全量账本hash
		proveTraceLog = monitor.NewPerformTraceLogger("genProveHashs", common.Hash{}, 0)
		proveTraceLog.SetParent("blockProposal")
		proveHashs = p.proveChecker.genProveHashs(height, worker.getBaseBH().Random, gb.MemIds)
		proveTraceLog.SetEnd()
	}()
	wg.Wait()
	if block == nil {
		blog.error("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	bh := block.Header

	traceLogger.SetHash(bh.Hash)
	traceLogger.SetTxNum(len(block.Transactions))
	proveTraceLog.SetHash(bh.Hash)
	proveTraceLog.SetHeight(bh.Height)
	proveTraceLog.Log("")

	tlog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.debug("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash.ShortS(), height, qn, gid.ShortS(), pi.ShortS())
	tlog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash.ShortS(), gid.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		// Here you need to use a normal private key, a non-group related private key.
		skey := p.mi.SK

		ccm := &model.ConsensusCastMessage{
			BH: *bh,
		}
		// The message hash sent to everyone is the same, the signature is the same
		if !ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), ccm) {
			blog.error("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		}

		traceLogger.Log("PreHash=%v,Qn=%v", bh.PreHash.ShortS(), qn)

		p.NetServer.SendCastVerify(ccm, gb, proveHashs)

		// ccm.GenRandomSign(skey, worker.baseBH.Random)
		// Castor cannot sign random numbers
		tlog.log("successful cast block, SendVerifiedCast, time interval %v, castor=%v, hash=%v, genHash=%v", bh.Elapsed, ccm.SI.GetID().ShortS(), bh.Hash.ShortS(), ccm.SI.DataHash.ShortS())

		// Send info
		le := &monitor.LogEntry{
			LogType:  monitor.LogTypeProposal,
			Height:   bh.Height,
			Hash:     bh.Hash.Hex(),
			PreHash:  bh.PreHash.Hex(),
			Proposer: p.GetMinerID().GetHexString(),
			Verifier: gb.Gid.GetHexString(),
			Ext:      fmt.Sprintf("qn:%v,totalQN:%v", qn, bh.TotalQN),
		}
		monitor.Instance.AddLog(le)
		p.proveChecker.addProve(pi)
		worker.markProposed()

		p.blockContexts.addProposed(block)

	} else {
		blog.debug("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}

// reqRewardTransSign generates a bonus transaction based on the signature pieces received locally,
// and broadcast it to other members of the group for signature.
//
// After the block verification consensus, the group should issue a corresponding bonus transaction consensus
// to make sure that 51% of the verified-member can get the bonus
func (p *Processor) reqRewardTransSign(vctx *VerifyContext, bh *types.BlockHeader) {
	blog := newBizLog("reqRewardTransSign")
	blog.debug("start, bh=%v", p.blockPreview(bh))
	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		blog.error("slot is nil")
		return
	}
	if !slot.gSignGenerator.Recovered() {
		blog.error("slot not recovered")
		return
	}
	if !slot.IsSuccess() && !slot.IsVerified() {
		blog.error("slot not verified or success,status=%v", slot.GetSlotStatus())
		return
	}
	// If you sign yourself, you don't have to send it again
	if slot.hasSignedRewardTx() {
		blog.warn("has signed reward tx")
		return
	}

	groupID := groupsig.DeserializeID(bh.GroupID)
	group := p.GetGroup(groupID)

	targetIDIndexs := make([]int32, 0)
	signs := make([]groupsig.Signature, 0)
	idHexs := make([]string, 0)

	threshold := model.Param.GetGroupK(group.GetMemberCount())
	for idx, mem := range group.GetMembers() {
		if sig, ok := slot.gSignGenerator.GetWitness(mem); ok {
			signs = append(signs, sig)
			targetIDIndexs = append(targetIDIndexs, int32(idx))
			idHexs = append(idHexs, mem.ShortS())
			if len(signs) >= threshold {
				break
			}
		}
	}

	bonus, tx, err := p.MainChain.GetBonusManager().GenerateBonus(targetIDIndexs, bh.Hash, bh.GroupID, model.Param.VerifyBonus)
	if err != nil {
		err = fmt.Errorf("failed to generate bonus %s", err)
		return
	}
	blog.debug("generate bonus txHash=%v, targetIds=%v, height=%v", bonus.TxHash.ShortS(), bonus.TargetIds, bh.Height)

	tlog := newHashTraceLog("REWARD_REQ", bh.Hash, p.GetMinerID())
	tlog.log("txHash=%v, targetIds=%v", bonus.TxHash.ShortS(), strings.Join(idHexs, ","))

	if slot.setRewardTrans(tx) {
		msg := &model.CastRewardTransSignReqMessage{
			Reward:       *bonus,
			SignedPieces: signs,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(groupID))
		if msg.GenSign(ski, msg) {
			p.NetServer.SendCastRewardSignReq(msg)
			blog.debug("reward req send height=%v, gid=%v", bh.Height, groupID.ShortS())
		} else {
			blog.error("genSign fail, id=%v, sk=%v, belong=%v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(groupID))
		}
	}

}
