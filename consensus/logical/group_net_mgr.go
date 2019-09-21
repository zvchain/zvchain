//   Copyright (C) 2019 ZVChain
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
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"sync"
)

type netServer interface {
	// BuildGroupNet builds the group net in local for inter-group communication
	BuildGroupNet(groupIdentifier string, mems []groupsig.ID)

	// ReleaseGroupNet releases the group net in local
	ReleaseGroupNet(groupIdentifier string)

	FullBuildProposerGroupNet(proposers []groupsig.ID, stakes []uint64)

	IncrementBuildProposerGroupNet(proposers []groupsig.ID, stakes []uint64)
}

type groupNetMgr struct {
	groupNetBuilt             sync.Map // Store groups that have built group-network
	ns                        netServer
	gr                        *groupReader
	mr                        minerPool
	minerID                   groupsig.ID
	expectNextFullBuiltHeight uint64
}

func newGroupNetMgr(ns netServer, gr *groupReader, mr minerPool, id groupsig.ID) *groupNetMgr {
	return &groupNetMgr{
		ns:      ns,
		gr:      gr,
		mr:      mr,
		minerID: id,
	}
}

func (nm *groupNetMgr) dissolveGroupNet(h uint64) {
	nm.groupNetBuilt.Range(func(key, value interface{}) bool {
		g := value.(*verifyGroup)
		// DismissHeight()+100 may be overflow
		if g.header.DismissHeight() < h && g.header.DismissHeight()+100 < h {
			stdLogger.Debugf("release group net of %v at %v", g.header.Seed(), h)
			nm.ns.ReleaseGroupNet(g.header.Seed().Hex())
			nm.groupNetBuilt.Delete(key)
		}
		return true
	})
}

// buildGroupNetOfNextEpoch Builds group-network of those groups will be activated at next epoch
func (nm *groupNetMgr) buildGroupNetOfNextEpoch(h uint64) {
	nextEp := types.EpochAt(h).Next()
	// checks if now is at the last 100 blocks of current epoch
	if h+100 > nextEp.Start() {
		nm.buildGroupNetOfActivateEpochAt(nextEp)
	}
}

// buildGroupNetOfActivateEpochAt Builds group-network of those groups will be activated at given epoch
func (nm *groupNetMgr) buildGroupNetOfActivateEpochAt(ep types.Epoch) {
	gs := nm.gr.getActivatedGroupsByHeight(ep.Start())
	for _, g := range gs {
		if g.hasMember(nm.minerID) {
			if _, ok := nm.groupNetBuilt.Load(g.header.Seed()); ok {
				continue
			}
			stdLogger.Debugf("build group net of %v at epoch %v-%v", g.header.Seed(), ep.Start(), ep.End())
			nm.ns.BuildGroupNet(g.header.Seed().Hex(), g.getMembers())
			nm.groupNetBuilt.Store(g.header.Seed(), g)
		}
	}
}

func (nm *groupNetMgr) tryFullBuildProposerGroupNetAt(h uint64) bool {
	if h < nm.expectNextFullBuiltHeight {
		return false
	}
	proposers := nm.mr.GetAllMiners(types.MinerTypeProposal, h)
	ids := make([]groupsig.ID, 0)
	stakes := make([]uint64, 0)
	for _, m := range proposers {
		id := groupsig.DeserializeID(m.ID)
		ids = append(ids, id)
		stakes = append(stakes, m.Stake)
	}
	stdLogger.Debugf("buildProposerGroupNetAt %v size %v", h, len(ids))
	nm.ns.FullBuildProposerGroupNet(ids, stakes)
	// build the net at the middle of each epoch in order to stagger with the candidate selection of the verify-group-build routine
	nm.expectNextFullBuiltHeight = types.EpochAt(h).Next().Start() + types.EpochLength/2
	return true
}

func (nm *groupNetMgr) incrementBuildProposerGroupNet(b *types.Block) {
	ids := make([]groupsig.ID, 0)
	stakes := make([]uint64, 0)
	for _, tx := range b.Transactions {
		if tx.Type == types.TransactionTypeStakeAdd {
			// Only stake for self can change the miner status
			if !bytes.Equal(tx.Source.Bytes(), tx.Target.Bytes()) {
				continue
			}
			mpks, err := types.DecodePayload(tx.Data)
			if err != nil {
				stdLogger.Errorf("DecodePayload error %v ", err)
				continue
			}
			// Only care about the proposer role
			if mpks.MType == types.MinerTypeProposal {
				miner := nm.mr.GetMiner(*tx.Source, types.MinerTypeProposal, b.Header.Height)
				if miner == nil {
					continue
				}
				if miner.IsActive() {
					id := groupsig.DeserializeID(tx.Source.Bytes())
					ids = append(ids, id)
					stakes = append(stakes, miner.Stake)
				}
			}
		}
	}
	if len(ids) > 0 {
		stdLogger.Debugf("incrementBuildProposerGroupNet at %v, size %v[%v-%v]", b.Header.Height, len(ids), ids, stakes)
		nm.ns.IncrementBuildProposerGroupNet(ids, stakes)
	}
}

func (nm *groupNetMgr) updateGroupNetRoutine(b *types.Block) {
	h := b.Header.Height
	nm.buildGroupNetOfNextEpoch(h)
	fullBuilt := nm.tryFullBuildProposerGroupNetAt(h)
	if !fullBuilt {
		nm.incrementBuildProposerGroupNet(b)
	}
	nm.dissolveGroupNet(h)
}
