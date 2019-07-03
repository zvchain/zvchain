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
	"sync/atomic"
	time2 "time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
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

// checkSelfCastRoutine check if the current group cast block
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

	if !p.canPropose() {
		return false
	}

	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		return false
	}
	blog.debug("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime)
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
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}
	// Groups that are currently highly disbanded should not be deleted from
	// the cache immediately, delaying the deletion of a build cycle. Ensure that
	// the block built on the eve of the dissolution of the group is valid
	groups := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	ids := make([]groupsig.ID, 0)
	for _, g := range groups {
		ids = append(ids, g.GroupID)
	}

	p.blockContexts.cleanVerifyContext(topHeight)

	blog := newBizLog("releaseRoutine")

	if len(ids) > 0 {
		blog.debug("clean group %v\n", len(ids))
		p.globalGroups.removeGroups(ids)
		p.belongGroups.leaveGroups(ids)
		for _, g := range groups {
			gid := g.GroupID
			blog.debug("DissolveGroupNet staticGroup gid %v ", gid.ShortS())
			p.NetServer.ReleaseGroupNet(gid.GetHexString())
			p.joiningGroups.RemoveGroup(g.GInfo.GroupHash())
		}
	}

	// Release the group network of the uncompleted group and the corresponding dummy group
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gInfo == nil || gc.is == GisGroupInitDone {
			return true
		}
		gis := &gc.gInfo.GI
		gHash := gis.GetHash()
		if gis.ReadyTimeout(topHeight) {
			blog.debug("DissolveGroupNet dummyGroup from joiningGroups gHash %v", gHash.ShortS())
			p.NetServer.ReleaseGroupNet(gHash.Hex())

			initedGroup := p.globalGroups.GetInitedGroup(gHash)
			omgied := "nil"
			if initedGroup != nil {
				omgied = fmt.Sprintf("OMGIED:%v(%v)", initedGroup.receiveSize(), initedGroup.threshold)
			}

			waitPieceIds := make([]string, 0)
			waitIds := make([]groupsig.ID, 0)
			for _, mem := range gc.gInfo.Mems {
				if !gc.node.hasPiece(mem) {
					waitPieceIds = append(waitPieceIds, mem.ShortS())
					waitIds = append(waitIds, mem)
				}
			}
			// Send info
			le := &monitor.LogEntry{
				LogType:  monitor.LogTypeInitGroupRevPieceTimeout,
				Height:   p.GroupChain.Height(),
				Hash:     gHash.Hex(),
				Proposer: "00",
				Ext:      fmt.Sprintf("MemCnt:%v,Pieces:%v,wait:%v,%v", gc.gInfo.MemberSize(), gc.node.groupInitPool.GetSize(), waitPieceIds, omgied),
			}
			if !gc.sendLog && monitor.Instance.IsFirstNInternalNodesInGroup(gc.gInfo.Mems, 50) {
				monitor.Instance.AddLog(le)
				gc.sendLog = true
			}

			msg := &model.ReqSharePieceMessage{
				GHash: gc.gInfo.GroupHash(),
			}
			stdLogger.Debugf("reqSharePieceRoutine:req size %v, ghash=%v", len(waitIds), gc.gInfo.GroupHash().ShortS())
			if msg.GenSign(p.getDefaultSeckeyInfo(), msg) {
				for _, receiver := range waitIds {
					stdLogger.Debugf("reqSharePieceRoutine:req share piece msg from %v, ghash=%v", receiver, gc.gInfo.GroupHash().ShortS())
					p.NetServer.ReqSharePiece(msg, receiver)
				}
			} else {
				ski := p.getDefaultSeckeyInfo()
				stdLogger.Debugf("gen req sharepiece sign fail, ski=%v %v", ski.ID.ShortS(), ski.SK.ShortS())
			}

		}
		return true
	})
	gctx := p.groupManager.getContext()
	if gctx != nil && gctx.readyTimeout(topHeight) {
		groupLogger.Infof("releaseRoutine:info=%v, elapsed %v. ready timeout.", gctx.logString(), time2.Since(gctx.createTime))

		if gctx.isKing() {
			gHash := "0000"
			if gctx != nil && gctx.gInfo != nil {
				gHash = gctx.gInfo.GroupHash().Hex()
			}
			// Send info
			le := &monitor.LogEntry{
				LogType:  monitor.LogTypeCreateGroupSignTimeout,
				Height:   p.GroupChain.Height(),
				Hash:     gHash,
				Proposer: p.GetMinerID().GetHexString(),
				Ext:      fmt.Sprintf("%v", gctx.gSignGenerator.Brief()),
			}
			if monitor.Instance.IsFirstNInternalNodesInGroup(gctx.kings, 50) {
				monitor.Instance.AddLog(le)
			}
		}
		p.groupManager.removeContext()
	}

	p.globalGroups.generator.forEach(func(ig *InitedGroup) bool {
		hash := ig.gInfo.GroupHash()
		if ig.gInfo.GI.ReadyTimeout(topHeight) {
			blog.debug("remove InitedGroup, gHash %v", hash.ShortS())
			p.NetServer.ReleaseGroupNet(hash.Hex())
			p.globalGroups.removeInitedGroup(hash)
		}
		return true
	})

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
	p.futureRewardReqs.forEach(func(key common.Hash, arr []interface{}) bool {
		for _, msg := range arr {
			b := msg.(*model.CastRewardTransSignReqMessage)

			// Can not be processed within 400s, are deleted
			if time2.Now().After(b.ReceiveTime.Add(400 * time2.Second)) {
				p.futureRewardReqs.remove(key)
				blog.debug("remove future reward msg, hash=%v", key.Hex())
				break
			}
		}
		return true
	})

	// Clean up the timeout signature public key request
	cleanSignPkReqRecord()

	for _, h := range p.blockContexts.verifyMsgCaches.Keys() {
		hash := h.(common.Hash)
		cache := p.blockContexts.getVerifyMsgCache(hash)
		if cache != nil && cache.expired() {
			blog.debug("remove verify cache msg, hash=%v", hash.ShortS())
			p.blockContexts.removeVerifyMsgCache(hash)
		}
	}

	return true
}

func (p *Processor) getUpdateGlobalGroupsRoutineName() string {
	return "update_global_groups_routine_" + p.getPrefix()
}

func (p *Processor) updateGlobalGroups() bool {
	top := p.MainChain.Height()
	iter := p.GroupChain.NewIterator()
	for g := iter.Current(); g != nil && !isGroupDissmisedAt(g.Header, top); g = iter.MovePre() {
		gid := groupsig.DeserializeID(g.ID)
		if g, _ := p.globalGroups.getGroupFromCache(gid); g != nil {
			continue
		}
		sgi := newSGIFromCoreGroup(g)
		stdLogger.Infof("updateGlobalGroups:gid=%v, workHeight=%v, topHeight=%v", gid.ShortS(), g.Header.WorkHeight, top)
		p.acceptGroup(sgi)
	}
	return true
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
		GroupHeight: p.GroupChain.Height(),
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
