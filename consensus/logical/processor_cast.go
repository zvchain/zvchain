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
	"math/big"
	"strings"
	"sync"

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/consensus/net"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
)

// triggerCastCheck trigger once to check if you are next ingot verifyGroup
func (p *Processor) triggerCastCheck() {
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) calcVerifyGroup(preBH *types.BlockHeader, height uint64) common.Hash {
	var hash = calcRandomHash(preBH, height)

	groupSeeds := p.groupReader.getAvailableGroupSeedsByHeight(height)
	// Must not happen
	if len(groupSeeds) == 0 {
		panic("no available groupSeeds")
	}
	seeds := make([]string, len(groupSeeds))
	for _, seed := range groupSeeds {
		seeds = append(seeds, common.ShortHex(seed.Seed().Hex()))
	}

	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(len(groupSeeds))))

	selectedGroup := groupSeeds[index.Int64()]

	stdLogger.Debugf("verify groups at %v: %v, selected %v", height, seeds, selectedGroup.Seed())
	return selectedGroup.Seed()
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, height uint64) *net.GroupBrief {
	nextGroup := p.calcVerifyGroup(bh, height)
	group := p.groupReader.getGroupBySeed(nextGroup)
	g := &net.GroupBrief{
		GSeed:  nextGroup,
		MemIds: group.getMembers(),
	}
	return g
}

// reserveBlock reserves the block in the context utils it can be broadcast
func (p *Processor) reserveBlock(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.debug("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash, slot.GetSlotStatus())

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

		// Add on chain and out-of-verifyGroup broadcasting
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

	gb := p.spreadGroupBrief(bh, bh.Height+1)
	if gb == nil {
		return fmt.Errorf("next verifyGroup is nil")
	}
	p.NetServer.BroadcastNewBlock(block, gb)
	tlog.log("broadcasted height=%v, consuming %vs", bh.Height, p.ts.Since(bh.CurTime))

	// Send info
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeBlockBroadcast,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: groupsig.DeserializeID(bh.Castor).GetHexString(),
		Verifier: gb.GSeed.Hex(),
	}
	monitor.Instance.AddLog(le)
	return nil
}

// consensusFinalize represents the final stage of the consensus process.
// It firstly verifies the verifyGroup signature and then requests the block body from proposer
func (p *Processor) consensusFinalize(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH

	var result string

	traceLog := monitor.NewPerformTraceLogger("consensusFinalize", bh.Hash, bh.Height)
	traceLog.SetParent("OnMessageVerify")

	tLog := newHashTraceLog("consensusFinalize", bh.Hash, p.GetMinerID())
	defer func() {
		traceLog.Log("result=%v. consensusFinalize cost %v", result, p.ts.Now().Local().Sub(bh.CurTime.Local()).String())
		tLog.log("result=%v", result)
	}()

	// Already on blockchain
	if p.blockOnChain(bh.Hash) {
		result = "already on chain"
		return
	}

	gpk := groupsig.DeserializePubkeyBytes(vctx.group.header.PublicKey())

	// Group signature verification passed
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) {
		result = "verify group sig fail"
		return
	}

	// Ask the proposer for a complete block
	msg := &model.ReqProposalBlock{
		Hash: bh.Hash,
	}
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

	totalStake := p.minerReader.getTotalStake(worker.baseBH.Height)
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
		blog.error("spreadGroupBrief nil, bh=%v, height=%v", top.Hash, height)
		return
	}

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
		block = p.MainChain.CastBlock(uint64(height), pi, qn, p.GetMinerID().Serialize(), gb.GSeed)
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
	block.Header.Signature = groupsig.Sign(p.mi.SK, block.Header.Hash.Bytes()).Serialize() //proposer sign block after cast
	bh := block.Header

	traceLogger.SetHash(bh.Hash)
	traceLogger.SetTxNum(len(block.Transactions))
	proveTraceLog.SetHash(bh.Hash)
	proveTraceLog.SetHeight(bh.Height)
	proveTraceLog.Log("")

	tLog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.debug("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash, height, qn, gb.GSeed, pi)
	tLog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash, gb.GSeed)

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		// Here you need to use a normal private key, a non-verifyGroup related private key.
		skey := p.mi.SK

		ccm := &model.ConsensusCastMessage{
			BH: *bh,
		}
		// The message hash sent to everyone is the same, the signature is the same
		if !ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), ccm) {
			blog.error("sign fail, id=%v, sk=%v", p.GetMinerID(), skey)
			return
		}

		traceLogger.Log("PreHash=%v,Qn=%v", bh.PreHash, qn)

		p.NetServer.SendCastVerify(ccm, gb, proveHashs)

		// ccm.GenRandomSign(skey, worker.baseBH.Random)
		// Castor cannot sign random numbers
		tLog.log("successful cast block, SendVerifiedCast, time interval %v, castor=%v, hash=%v, genHash=%v", bh.Elapsed, ccm.SI.GetID(), bh.Hash, ccm.SI.DataHash)

		// Send info
		le := &monitor.LogEntry{
			LogType:  monitor.LogTypeProposal,
			Height:   bh.Height,
			Hash:     bh.Hash.Hex(),
			PreHash:  bh.PreHash.Hex(),
			Proposer: p.GetMinerID().GetHexString(),
			Verifier: gb.GSeed.Hex(),
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

// reqRewardTransSign generates a reward transaction based on the signature pieces received locally,
// and broadcast it to other members of the verifyGroup for signature.
//
// After the block verification consensus, the verifyGroup should issue a corresponding reward transaction consensus
// to make sure that 51% of the verified-member can get the reward
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

	group := vctx.group

	targetIDIndexs := make([]int32, 0)
	signs := make([]groupsig.Signature, 0)
	idHexs := make([]string, 0)

	threshold := group.header.Threshold()
	for idx, mem := range group.getMembers() {
		if sig, ok := slot.gSignGenerator.GetWitness(mem); ok {
			signs = append(signs, sig)
			targetIDIndexs = append(targetIDIndexs, int32(idx))
			idHexs = append(idHexs, mem.GetHexString())
			if len(signs) >= int(threshold) {
				break
			}
		}
	}
	rewardShare := p.MainChain.GetRewardManager().CalculateCastRewardShare(bh.Height, bh.GasFee)

	reward, tx, err := p.MainChain.GetRewardManager().GenerateReward(targetIDIndexs, bh.Hash, bh.Group, rewardShare.TotalForVerifier(), rewardShare.ForRewardTxPacking)
	if err != nil {
		err = fmt.Errorf("failed to generate reward %s", err)
		return
	}
	blog.debug("generate reward txHash=%v, targetIds=%v, height=%v", reward.TxHash, reward.TargetIds, bh.Height)

	tLog := newHashTraceLog("REWARD_REQ", bh.Hash, p.GetMinerID())
	tLog.log("txHash=%v, targetIds=%v", reward.TxHash, strings.Join(idHexs, ","))

	if slot.setRewardTrans(tx) {
		msg := &model.CastRewardTransSignReqMessage{
			Reward:       *reward,
			SignedPieces: signs,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.groupReader.getGroupSignatureSeckey(group.header.Seed()))
		if msg.GenSign(ski, msg) {
			p.NetServer.SendCastRewardSignReq(msg)
			blog.debug("reward req send height=%v, gseed=%v", bh.Height, group.header.Seed())
		} else {
			blog.error("genSign fail, id=%v, sk=%v", ski.ID, ski.SK)
		}
	}

}
