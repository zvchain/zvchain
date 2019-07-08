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

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

// Start starts miner process
func (p *Processor) Start() bool {
	p.Ticker.RegisterPeriodicRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 1)
	p.Ticker.RegisterPeriodicRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.RegisterPeriodicRoutine(p.getBroadcastRoutineName(), p.broadcastRoutine, 1)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.Ticker.StartTickerRoutine(p.getBroadcastRoutineName(), false)

	p.Ticker.RegisterPeriodicRoutine(p.getUpdateGlobalGroupsRoutineName(), p.updateGlobalGroups, 60)
	p.Ticker.StartTickerRoutine(p.getUpdateGlobalGroupsRoutineName(), false)

	p.Ticker.RegisterPeriodicRoutine(p.getUpdateMonitorNodeInfoRoutine(), p.updateMonitorInfo, 3)
	p.Ticker.StartTickerRoutine(p.getUpdateMonitorNodeInfoRoutine(), false)

	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

// Stop is reserved interface
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.QueryTopBlock().Height

	stdLogger.Infof("prepareMiner get groups from groupchain")
	iterator := p.GroupChain.NewIterator()
	groups := make([]*StaticGroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre() {
		stdLogger.Debugf("get group from core, id=%+v", coreGroup.Header)
		if coreGroup.ID == nil || len(coreGroup.ID) == 0 {
			continue
		}
		needBreak := false
		sgi := newSGIFromCoreGroup(coreGroup)
		if sgi.Dismissed(topHeight) && len(groups) > 100 {
			needBreak = true
			genesis := p.GroupChain.GetGroupByHeight(0)
			if genesis == nil {
				// hold it for now
				panic("get genesis group nil")
			}
			sgi = newSGIFromCoreGroup(genesis)

		}
		groups = append(groups, sgi)
		stdLogger.Debugf("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.getGroupHeader().WorkHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				stdLogger.Debugf("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.joinGroup(jg)
			}
			if sgi.GInfo.GI.CreateHeight() == 0 {
				stdLogger.Debugf("genesis member start...id %v", p.GetMinerID().GetHexString())
				p.genesisMember = true
			}
		}
		if needBreak {
			break
		}
	}
	for i := len(groups) - 1; i >= 0; i-- {
		p.acceptGroup(groups[i])
	}
	stdLogger.Infof("prepare finished")
}

// Ready check if the processor engine is initialized and ready for message processing
func (p *Processor) Ready() bool {
	return p.ready
}

// GetCastQualifiedGroups returns all group infos can work at the given height through the cached group slices
func (p *Processor) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetCastQualifiedGroups(height)
}

// Finalize do some clean and release work after stop mining
func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.close()
	}
}

func (p *Processor) getVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) getSelfMinerDO() *model.SelfMinerDO {
	md := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerDO = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(h uint64) bool {
	miner := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
	return miner.CanCastAt(h)
}

// GetJoinedWorkGroupNums returns both work-group and avail-group num of current node
func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.QueryTopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.CastQualified(h) {
			work++
		}
		avail++
	}
	return
}

// CalcBlockHeaderQN calculates the qn value of the given block header
func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := base.VRFProve(bh.ProveValue)
	castor := groupsig.DeserializeID(bh.Castor)
	pre := p.MainChain.QueryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return 0
	}
	miner := p.minerReader.getProposeMinerByHeight(castor, pre.Height)
	if miner == nil {
		stdLogger.Warnf("CalcBHQN getMiner nil id=%v, bh=%v", castor.ShortS(), bh.Hash.ShortS())
		return 0
	}
	totalStake := p.minerReader.getTotalStake(pre.Height)
	_, qn := vrfSatisfy(pi, miner.Stake, totalStake)
	return qn
}

// GetVrfThreshold returns the vrf threshold of current node under the specified stake
func (p *Processor) GetVrfThreshold(stake uint64) float64 {
	totalStake := p.minerReader.getTotalStake(p.MainChain.Height())
	if totalStake == 0 {
		return 0
	}
	vs := vrfThreshold(stake, totalStake)
	f, _ := vs.Float64()
	return f
}

// GetJoinGroupInfo returns group-related info current node joined in with the given gid(in hex string)
func (p *Processor) GetJoinGroupInfo(gid string) *JoinedGroup {
	var id groupsig.ID
	id.SetHexString(gid)
	jg := p.belongGroups.getJoinedGroup(id)
	return jg
}

// GetAllMinerDOs returns all available miner infos
func (p *Processor) GetAllMinerDOs() []*model.MinerDO {
	h := p.MainChain.Height()
	dos := make([]*model.MinerDO, 0)
	miners := p.minerReader.getAllMinerDOByType(types.MinerTypeProposal, h)
	dos = append(dos, miners...)

	miners = p.minerReader.getAllMinerDOByType(types.MinerTypeVerify, h)
	dos = append(dos, miners...)
	return dos
}

// GetCastQualifiedGroupsFromChain returns all group infos can work at the given height through the group chain
func (p *Processor) GetCastQualifiedGroupsFromChain(height uint64) []*types.Group {
	return p.globalGroups.getCastQualifiedGroupFromChains(height)
}

func (p *Processor) checkProveRoot(bh *types.BlockHeader) (bool, error) {
	//exist, ok, err := p.proveChecker.getPRootResult(bh.Hash)
	//if exist {
	//	return ok, err
	//}
	//slog := taslog.NewSlowLog("checkProveRoot-" + bh.Hash.ShortS(), 0.6)
	//defer func() {
	//	slog.Log("hash=%v, height=%v", bh.Hash.String(), bh.Height)
	//}()
	//slog.AddStage("queryBlockHeader")
	//preBH := p.MainChain.QueryBlockHeaderByHash(bh.PreHash)
	//slog.EndStage()
	//if preBH == nil {
	//	return false, errors.New(fmt.Sprintf("preBlock is nil,hash %v", bh.PreHash.ShortS()))
	//}
	//gid := groupsig.DeserializeID(bh.GroupID)
	//
	//slog.AddStage("getGroup")
	//group := p.GetGroup(gid)
	//slog.EndStage()
	//if !group.GroupID.isValid() {
	//	return false, errors.New(fmt.Sprintf("group is invalid, gid %v", gid))
	//}

	//slog.AddStage("genProveHash")
	//if _, root := p.proveChecker.genProveHashs(bh.Height, preBH.Random, group.GetMembers()); root == bh.ProveRoot {
	//	slog.EndStage()
	//	p.proveChecker.addPRootResult(bh.Hash, true, nil)
	//	return true, nil
	//} else {
	//	//panic(fmt.Errorf("check prove fail, hash=%v, height=%v", bh.Hash.String(), bh.Height))
	//	//return false, errors.New(fmt.Sprintf("proveRoot expect %v, receive %v", bh.ProveRoot.String(), root.String()))
	//}
	return true, nil
}

// DebugPrintCheckProves print some message for debug use
func (p *Processor) DebugPrintCheckProves(preBH *types.BlockHeader, height uint64, gid groupsig.ID) []string {
	ss := make([]string, 0)
	group := p.GetGroup(gid)
	if group == nil {
		stdLogger.Debugf("failed to get group: groupID=%v", gid)
		s := fmt.Sprintf("failed to get group: groupID=%v", gid)
		ss = append(ss, s)
		return ss
	}

	for _, id := range group.GetMembers() {
		h := p.proveChecker.sampleBlockHeight(height, preBH.Random, id)
		bs := p.MainChain.QueryBlockBytesFloor(h)
		block := p.MainChain.QueryBlockFloor(h)
		hash := p.proveChecker.genVerifyHash(bs, id)

		var s string
		if block == nil {
			s = fmt.Sprintf("id %v, height %v, bytes %v, prove hash %v block nil", id.GetHexString(), h, bs, hash.Hex())
			stdLogger.Debugf("id %v, height %v, bytes %v, prove hash %v block nil", id.GetHexString(), h, bs, hash.Hex())
		} else {
			s = fmt.Sprintf("id %v, height %v, bytes %v, prove hash %v blockheader %+v, body %+v", id.GetHexString(), h, bs, hash.Hex(), block.Header, block.Transactions)
			stdLogger.Debugf("id %v, height %v, bytes %v, prove hash %v blockheader %+v, body %+v", id.GetHexString(), h, bs, hash.Hex(), block.Header, block.Transactions)
		}
		ss = append(ss, s)
	}
	return ss
}
