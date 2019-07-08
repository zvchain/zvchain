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
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

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

func (p *Processor) canPropose() bool {
	miner := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
	return miner.CanPropose()
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
		stdLogger.Warnf("CalcBHQN getMiner nil id=%v, bh=%v", castor, bh.Hash)
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
	//gid := groupsig.DeserializeID(bh.Group)
	//
	//slog.AddStage("getGroup")
	//verifyGroup := p.GetGroup(gid)
	//slog.EndStage()
	//if !verifyGroup.Group.isValid() {
	//	return false, errors.New(fmt.Sprintf("verifyGroup is invalid, gid %v", gid))
	//}

	//slog.AddStage("genProveHash")
	//if _, root := p.proveChecker.genProveHashs(bh.Height, preBH.Random, verifyGroup.GetMembers()); root == bh.ProveRoot {
	//	slog.EndStage()
	//	p.proveChecker.addPRootResult(bh.Hash, true, nil)
	//	return true, nil
	//} else {
	//	//panic(fmt.Errorf("check prove fail, hash=%v, height=%v", bh.Hash.String(), bh.Height))
	//	//return false, errors.New(fmt.Sprintf("proveRoot expect %v, receive %v", bh.ProveRoot.String(), root.String()))
	//}
	return true, nil
}
