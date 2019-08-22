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
	"sync/atomic"
	time2 "time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/monitor"
)

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) getBroadcastRoutineName() string {
	return "broadcast_" + p.getPrefix()
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}

// checkSelfCastRoutine check if the current verifyGroup cast block
func (p *Processor) checkSelfCastRoutine() bool {
	if !atomic.CompareAndSwapInt32(&p.isCasting, 0, 1) {
		return false
	}
	defer func() {
		p.isCasting = 0
	}()

	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")

	if p.MainChain.IsAdjusting() {
		blog.warn("isAdjusting, return...")
		p.triggerCastCheck()
		return false
	}

	top := p.MainChain.QueryTopBlock()

	var (
		expireTime  time.TimeStamp
		castHeight  uint64
		deltaHeight uint64
	)
	d := p.ts.Since(top.CurTime)
	if d < 0 {
		return false
	}

	deltaHeight = uint64(d)/uint64(model.Param.MaxGroupCastTime) + 1
	if top.Height > 0 {
		castHeight = top.Height + deltaHeight
	} else {
		castHeight = uint64(1)
	}
	expireTime = getCastExpireTime(top.CurTime, deltaHeight, castHeight)

	if !p.CanPropose() {
		return false
	}

	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		return false
	}
	blog.debug("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash, top.CurTime, castHeight, expireTime)
	worker = newVRFWorker(p.getSelfMinerDO(), top, castHeight, expireTime, p.ts)
	p.setVrfWorker(worker)
	p.blockProposal()
	return true
}

func (p *Processor) broadcastRoutine() bool {
	p.blockContexts.forEachReservedVctx(func(vctx *VerifyContext) bool {
		p.tryNotify(vctx)
		return true
	})
	return true
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height

	p.blockContexts.cleanVerifyContext(topHeight)

	blog := newBizLog("releaseRoutine")

	// Release futureVerifyMsg
	p.futureVerifyMsgs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.ConsensusCastMessage)
			if b.BH.Height+200 < topHeight {
				blog.debug("remove future verify msg, hash=%v", key.Hex())
				p.removeFutureVerifyMsgs(key)
				break
			}
		}
		return true
	})
	// Release futureRewardMsg
	p.rewardHandler.futureRewardReqs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.CastRewardTransSignReqMessage)

			// Can not be processed within 400s, are deleted
			if time2.Now().After(b.ReceiveTime.Add(400 * time2.Second)) {
				p.rewardHandler.futureRewardReqs.remove(key)
				blog.debug("remove future reward msg, hash=%v", key.Hex())
				break
			}
		}
		return true
	})

	for _, h := range p.blockContexts.verifyMsgCaches.Keys() {
		hash := h.(common.Hash)
		cache := p.blockContexts.getVerifyMsgCache(hash)
		if cache != nil && cache.expired() {
			blog.debug("remove verify cache msg, hash=%v", hash)
			p.blockContexts.removeVerifyMsgCache(hash)
		}
	}

	return true
}

func (p *Processor) getUpdateGlobalGroupsRoutineName() string {
	return "update_global_groups_routine_" + p.getPrefix()
}

func (p *Processor) getUpdateMonitorNodeInfoRoutine() string {
	return "update_monitor_node_routine_" + p.getPrefix()
}

func (p *Processor) updateMonitorInfo() bool {
	if !monitor.Instance.MonitorEnable() {
		return false
	}
	top := p.MainChain.Height()

	ni := &monitor.NodeInfo{
		BlockHeight: top,
		GroupHeight: p.groupReader.Height(),
		TxPoolCount: int(p.MainChain.GetTransactionPool().TxNum()),
	}
	proposer := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if proposer != nil {
		ni.Type |= monitor.NtypeProposal
		ni.PStake = proposer.Stake
		ni.VrfThreshold = p.GetVrfThreshold(ni.PStake)
	}
	verifier := p.minerReader.GetLatestVerifyMiner(p.GetMinerID())
	if verifier != nil {
		ni.Type |= monitor.NtypeVerifier
	}

	monitor.Instance.UpdateNodeInfo(ni)
	return true
}
