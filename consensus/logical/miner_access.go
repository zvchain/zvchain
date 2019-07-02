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
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
)

// MinerPoolReader provides some functions for access to the miner pool
type MinerPoolReader struct {
	minerPool       *core.MinerManager
	blog            *bizLog
	totalStakeCache uint64
}

func newMinerPoolReader(mp *core.MinerManager) *MinerPoolReader {
	return &MinerPoolReader{
		minerPool: mp,
		blog:      newBizLog("MinerPoolReader"),
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
		AbortHeight: miner.AbortHeight,
	}
	if !md.ID.IsValid() {
		stdLogger.Errorf("invalid id %v, %v", miner.ID, md.ID.GetHexString())
		return nil
	}
	return md
}

func (access *MinerPoolReader) getLightMiner(id groupsig.ID) *model.MinerDO {
	miner := access.minerPool.GetMinerByID(id.Serialize(), types.MinerTypeLight, nil)
	if miner == nil {
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getProposeMiner(id groupsig.ID) *model.MinerDO {
	miner := access.minerPool.GetMinerByID(id.Serialize(), types.MinerTypeHeavy, nil)
	if miner == nil {
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getAllMinerDOByType(ntype byte, h uint64) []*model.MinerDO {
	iter := access.minerPool.MinerIterator(ntype, h)
	mds := make([]*model.MinerDO, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			continue
		} else {
			md := convert2MinerDO(curr)
			mds = append(mds, md)
		}
	}
	return mds
}

func (access *MinerPoolReader) getCanJoinGroupMinersAt(h uint64) []model.MinerDO {
	miners := access.getAllMinerDOByType(types.MinerTypeLight, h)
	rets := make([]model.MinerDO, 0)
	access.blog.debug("all light nodes size %v", len(miners))
	for _, md := range miners {
		if md.CanJoinGroupAt(h) {
			rets = append(rets, *md)
		}
	}
	return rets
}

func (access *MinerPoolReader) getTotalStake(h uint64, cache bool) uint64 {
	if cache && access.totalStakeCache > 0 {
		return access.totalStakeCache
	}
	st := access.minerPool.GetTotalStake(h)
	access.totalStakeCache = st
	return st
}
