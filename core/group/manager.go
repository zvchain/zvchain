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

package group

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/log"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

var logger *logrus.Logger
var Punishment *PunishmentContext

// Manager implements groupContextProvider in package consensus
type Manager struct {
	chain            chainReader
	checkerImpl      types.GroupCreateChecker
	storeReaderImpl  types.GroupStoreReader
	packetSenderImpl types.GroupPacketSender
	punishment       minerPunishment
	poolImpl         *pool
	skipCounter      groupSkipCounter
}

func (m *Manager) GetGroupStoreReader() types.GroupStoreReader {
	return m.storeReaderImpl
}

func (m *Manager) GetGroupPacketSender() types.GroupPacketSender {
	return m.packetSenderImpl
}

func (m *Manager) RegisterGroupCreateChecker(checker types.GroupCreateChecker) {
	m.checkerImpl = checker
}

func NewManager(chain chainReader, counter groupSkipCounter) *Manager {
	logger = log.GroupLogger
	gPool := newPool(chain)
	store := NewStore(chain)
	packetSender := NewPacketSender(chain)

	managerImpl := &Manager{
		chain:            chain,
		storeReaderImpl:  store,
		packetSenderImpl: packetSender,
		poolImpl:         gPool,
		skipCounter:      counter,
		//punishment:  reader,
	}
	return managerImpl
}

func (m *Manager) InitManager(punishment minerPunishment, gen *types.GenesisInfo) {
	m.punishment = punishment
	db, err := m.chain.LatestAccountDB()
	if err != nil {
		panic(fmt.Sprintf("failed to init group manager pool %v", err))
	}
	err = m.poolImpl.initPool(db, gen)
	if err != nil {
		panic(fmt.Sprintf("failed to init group manager pool %v", err))
	}

}

func (m *Manager) InitGenesis(db types.AccountDB, genesisInfo *types.GenesisInfo) {
	err := m.poolImpl.initGenesis(db, genesisInfo)
	if err != nil {
		panic(fmt.Sprintf("failed to init InitGenesis %v", err))
	}
}

// RegularCheck try to create group, do punishment and refresh active group
func (m *Manager) RegularCheck(db types.AccountDB, bh *types.BlockHeader) {
	ctx := &CheckerContext{bh.Height}
	Punishment = &PunishmentContext{}
	m.tryCreateGroup(db, m.checkerImpl, ctx, Punishment)
	m.tryDoPunish(db, m.checkerImpl, ctx, Punishment)
}

// OnBlockRemove resets group with top block with parameter bh
func (m *Manager) OnBlockRemove(bh *types.BlockHeader) {

}

// Height returns count of current group number
func (m *Manager) Height() uint64 {
	db, err := m.chain.LatestAccountDB()
	if err != nil {
		logger.Error("failed to get last db")
		return 0
	}
	return m.poolImpl.count(db)
}

func (m *Manager) GroupsAfter(height uint64) []types.GroupI {
	return m.poolImpl.groupsAfter(height, common.MaxInt64)
}

// Height returns count of current group number
func (m *Manager) ActiveGroupCount() int {
	return len(m.GetActivatedGroupsAt(m.chain.Height()))
	//return len(m.poolImpl.activeList)
}

func (m *Manager) GetActivatedGroupsAt(height uint64) []types.GroupI {
	gis := make([]types.GroupI, 0)
	m.poolImpl.iterateGroups(func(g *group) bool {
		if g.HeaderD.activatedAt(height) {
			gis = append(gis, g)
		}
		return g.HeaderD.livedAt(height)
	})
	revert(gis)
	// add genesis group
	gis = append(gis, m.poolImpl.genesis)
	return gis
}

func (m *Manager) GetLivedGroupsAt(height uint64) []types.GroupI {
	gis := make([]types.GroupI, 0)
	m.poolImpl.iterateGroups(func(g *group) bool {
		lived := g.HeaderD.livedAt(height)
		if lived {
			gis = append(gis, g)
		}
		return lived
	})
	revert(gis)
	// add genesis group
	gis = append(gis, m.poolImpl.genesis)
	return gis
}

// GetGroupBySeed returns group with given Seed
func (m *Manager) GetGroupBySeed(seedHash common.Hash) types.GroupI {
	gp := m.poolImpl.get(nil, seedHash)
	if gp == nil {
		return nil
	}
	return gp
}

// GetGroupBySeed returns group header with given Seed
func (m *Manager) GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI {
	g := m.GetGroupBySeed(seedHash)
	if g != nil {
		return g.Header()
	}
	return nil
}

// MinerJoinedLivedGroupCountFilter returns function to check if the miners joined live group count less than the
// maxCount in a given block height
func (m *Manager) MinerJoinedLivedGroupCountFilter(maxCount int, height uint64) func(addr common.Address) bool {
	lived := m.GetLivedGroupsAt(height)
	doFilter := func(addr common.Address) bool {
		count := 0
		for _, gi := range lived {
			g := gi.(*group)
			if g.hasMember(addr.Bytes()) {
				count++
			}
		}
		return count < maxCount
	}
	return doFilter
}

func (m *Manager) GetLivedGroupsByMember(address common.Address, height uint64) []types.GroupI {
	groups := m.GetLivedGroupsAt(height)
	groupIs := make([]types.GroupI, 0)
	for _, gi := range groups {
		g := gi.(*group)
		if g.hasMember(address.Bytes()) {
			groupIs = append(groupIs, g)
		}
	}
	return groupIs
}

func (m *Manager) tryCreateGroup(db types.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext, ctxPun *PunishmentContext) {
	createResult := checker.CheckGroupCreateResult(ctx)
	miners := createResult.FrozenMiners()
	addresslist := make([]string, 0, 0)
	if miners != nil && len(miners) > 0 {
		fmt.Println("for tryCreateGroup", util.ObjectTojson(createResult.FrozenMiners()))
		for _, miner := range miners {
			addr := common.ToAddrHex(miner)
			addresslist = append(addresslist, addr)
		}
		punishmentContent := &PunishmentContent{
			Height:      ctx.Height(),
			AddressList: addresslist,
		}
		ctxPun.GroupPiece = punishmentContent
	}

	if createResult == nil {
		return
	}
	//if createResult.Err() != nil {
	//	return
	//}
	switch createResult.Code() {
	case types.CreateResultSuccess:
		err := m.saveGroup(db, newGroup(createResult.GroupInfo(), m.poolImpl.getTopGroup(db)))
		if err != nil {
			// this case must not happen.
			logger.Panicf("saveGroup error: %v", err)
		}
	case types.CreateResultMarkEvil:
		markGroupFail(db, createResult)
	case types.CreateResultFail:
		// do nothing
	}
	if len(createResult.FrozenMiners()) > 0 {
		m.frozeMiner(db, createResult.FrozenMiners(), ctx)
	}

}

func (m *Manager) tryDoPunish(db types.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext, ctxPun *PunishmentContext) {
	msg, err := checker.CheckGroupCreatePunishment(ctx)
	addresslist := make([]string, 0, 0)
	if msg != nil && msg.PenaltyTarget() != nil && len(msg.PenaltyTarget()) > 0 {
		fmt.Println("for tryDoPunish", util.ObjectTojson(msg.PenaltyTarget()))
		for _, miner := range msg.PenaltyTarget() {
			addr := common.ToAddrHex(miner)
			addresslist = append(addresslist, addr)
		}
		punishmentContext := &PunishmentContent{
			Height:      ctx.Height(),
			AddressList: addresslist,
		}
		ctxPun.Punish = punishmentContext
	}

	if err != nil {
		return
	}
	_, err = m.punishment.MinerPenalty(db, msg, ctx.Height())
	if err != nil {
		logger.Errorf("MinerPenalty error: %v", err)
	}
}

func (m *Manager) saveGroup(db types.AccountDB, group *group) error {
	return m.poolImpl.add(db, group)
}

func (m *Manager) frozeMiner(db types.AccountDB, frozenMiners [][]byte, ctx types.CheckerContext) {
	logger.Debugf("frozeMiner: %v", frozenMiners)
	for _, p := range frozenMiners {
		addr := common.BytesToAddress(p)
		_, err := m.punishment.MinerFrozen(db, addr, ctx.Height())
		if err != nil {
			logger.Errorf("MinerFrozen error: %v", err)
		}
	}
}

// markGroupFail mark group member should upload origin piece
func markGroupFail(db types.AccountDB, seed types.SeedI) {
	db.SetData(common.HashToAddress(seed.Seed()), originPieceReqKey, []byte{1})
}

func (m *Manager) UpdateGroupSkipCounts(db types.AccountDB, bh *types.BlockHeader) {
	// remove skip counts for the current verify-group
	m.poolImpl.updateSkipCount(db, bh.Group, 0)
	pre := m.chain.QueryBlockHeaderByHash(bh.PreHash)
	if pre != nil {
		counts := m.skipCounter.GroupSkipCountsBetween(pre, bh.Height)
		for gSeed, cnt := range counts {
			m.poolImpl.updateSkipCount(db, gSeed, cnt)
		}
	}
}

func (m *Manager) GetGroupSkipCountsAt(h uint64, groups []types.GroupI) (map[common.Hash]uint16, error) {
	db, err := m.chain.AccountDBAt(h)
	if err != nil {
		return nil, err
	}
	ret := make(map[common.Hash]uint16)
	for _, g := range groups {
		cnt := m.poolImpl.getSkipCount(db, g.Header().Seed())
		if cnt > 0 {
			ret[g.Header().Seed()] = cnt
		}
	}
	return ret, nil
}

// GetAllGroupSeedsByHeight iterates all groups from the given block height to genesis group.
// It's expensive and time consuming, don't call it unless you know what you are doing!
// It's called by the offline tailor or verifier for pruning mode currently
func (m *Manager) GetAllGroupSeedsByHeight(h uint64) ([]common.Hash, error) {
	return m.poolImpl.getAllGroupSeedsByHeight(h)
}

func (m *Manager) GroupKey() []byte {
	return groupDataKey
}
