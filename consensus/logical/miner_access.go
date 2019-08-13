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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

type minerPool interface {
	// GetLatestMiner returns the latest miner info of the given address and miner type
	GetLatestMiner(address common.Address, mType types.MinerType) *types.Miner

	// GetMiner returns the miner info of the given address and miner type at the given height
	GetMiner(address common.Address, mType types.MinerType, height uint64) *types.Miner

	// GetProposalTotalStake returns the total stake value of all proposals at the given height
	GetProposalTotalStake(height uint64) uint64

	// GetAllMinerAddress returns all miner addresses of the the specified type at the given height
	GetAllMiners(mType types.MinerType, height uint64) []*types.Miner
}

// MinerPoolReader provides some functions for access to the miner pool
type MinerPoolReader struct {
	mPool   minerPool
	process *Processor
	blog    *bizLog
}

func newMinerPoolReader(p *Processor, mp minerPool) *MinerPoolReader {
	return &MinerPoolReader{
		mPool:   mp,
		process: p,
		blog:    newBizLog("MinerPoolReader"),
	}
}

func convert2MinerDO(miner *types.Miner) *model.MinerDO {
	if miner == nil {
		return nil
	}
	md := &model.MinerDO{
		ID:          groupsig.DeserializeID(miner.ID),
		PK:          groupsig.DeserializePubkeyBytes(miner.PublicKey),
		VrfPK:       base.VRFPublicKey(miner.VrfPublicKey),
		Stake:       miner.Stake,
		NType:       miner.Type,
		ApplyHeight: miner.ApplyHeight,
		Status:      miner.Status,
	}
	if !md.ID.IsValid() {
		stdLogger.Errorf("invalid id %v, %v", miner.ID, md.ID.GetAddrString())
		return nil
	}
	if !md.PK.IsValid() {
		stdLogger.Errorf("invalid pubkey %v", miner.PublicKey)
		return nil
	}
	return md
}

func (access *MinerPoolReader) GetLatestVerifyMiner(id groupsig.ID) *model.MinerDO {
	miner := access.mPool.GetLatestMiner(id.ToAddress(), types.MinerTypeVerify)
	if miner == nil {
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getLatestProposeMiner(id groupsig.ID) *model.MinerDO {
	miner := access.mPool.GetLatestMiner(id.ToAddress(), types.MinerTypeProposal)
	if miner == nil {
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getProposeMinerByHeight(id groupsig.ID, height uint64) *model.MinerDO {
	miner := access.mPool.GetMiner(id.ToAddress(), types.MinerTypeProposal, height)
	if miner == nil {
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getAllMinerDOByType(minerType types.MinerType, h uint64) []*model.MinerDO {
	miners := access.mPool.GetAllMiners(minerType, h)
	if miners == nil {
		return []*model.MinerDO{}
	}
	mds := make([]*model.MinerDO, 0)
	for _, m := range miners {
		md := convert2MinerDO(m)
		mds = append(mds, md)
	}
	return mds
}

func (access *MinerPoolReader) GetCanJoinGroupMinersAt(h uint64) []*model.MinerDO {
	miners := access.getAllMinerDOByType(types.MinerTypeVerify, h)
	rets := make([]*model.MinerDO, 0)
	for _, md := range miners {
		if md.CanJoinGroup() {
			rets = append(rets, md)
		}
	}
	return rets
}

func (access *MinerPoolReader) getTotalStake(h uint64) uint64 {
	st := access.mPool.GetProposalTotalStake(h)
	return st
}

func (access *MinerPoolReader) SelfMinerInfo() *model.SelfMinerDO {
	mi := *access.process.mi
	mInfo := access.GetLatestVerifyMiner(mi.ID)
	if mInfo != nil {
		mi.MinerDO = *mInfo
		return &mi
	}
	return nil
}
