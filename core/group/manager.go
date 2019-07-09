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

	"github.com/zvchain/zvchain/taslog"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

var logger taslog.Logger

// Manager implements groupContextProvider in package consensus
type Manager struct {
	chain            chainReader
	checkerImpl      types.GroupCreateChecker
	storeReaderImpl  types.GroupStoreReader
	packetSenderImpl types.GroupPacketSender
	minerReaderImpl  minerReader
	poolImpl         pool
	genesisGroup     types.GroupI
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

func NewManager(chain chainReader, reader minerReader, genesisInfo *types.GenesisInfo) Manager {
	logger = taslog.GetLoggerByIndex(taslog.GroupLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	gPool := newPool()
	err := gPool.initPool(chain.LatestStateDB(), genesisInfo)
	if err != nil {
		panic(fmt.Sprintf("failed to init group manager pool %v", err))
	}

	store := NewStore(chain, *gPool)
	packetSender := NewPacketSender(chain)

	managerImpl := Manager{
		storeReaderImpl:  store,
		packetSenderImpl: packetSender,
		poolImpl:         *gPool,
		minerReaderImpl:  reader,
	}
	return managerImpl
}

// RegularCheck try to create group, do punishment and refresh active group
func (m *Manager) RegularCheck(db types.AccountDB) {
	ctx := &CheckerContext{m.chain.Height()}
	m.tryCreateGroup(db, m.checkerImpl, ctx)
	m.tryDoPunish(db, m.checkerImpl, ctx)
	m.poolImpl.adjust(db, ctx.Height())
}

// ResetTop resets group with top block with parameter bh
func (m *Manager) ResetToTop(db types.AccountDB, bh *types.BlockHeader) {
	m.poolImpl.resetToTop(db, bh.Height)
}

// Height returns count of current group number
func (m *Manager) Height() uint64 {
	return m.poolImpl.count(m.chain.LatestStateDB())
}

func (m *Manager) GroupsBefore(height uint64, limit int) []types.GroupI {
	return m.poolImpl.groupsBefore(m.chain, height, limit)
}

// Height returns count of current group number
func (m *Manager) WaitingGroupCount() int {
	return len(m.poolImpl.waitingList)
}

// Height returns count of current group number
func (m *Manager) ActiveGroupCount() int {
	return len(m.poolImpl.activeList)
}

func (m *Manager) tryCreateGroup(db types.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext) {
	createResult := checker.CheckGroupCreateResult(ctx)
	if createResult == nil {
		return
	}
	if createResult.Err() != nil {
		return
	}
	switch createResult.Code() {
	case types.CreateResultSuccess:
		err := m.saveGroup(db, newGroup(createResult.GroupInfo(), ctx.Height()))
		if err != nil {
			// this case must not happen.
			panic(logger.Error("saveGroup error: %v", err))
		}
	case types.CreateResultMarkEvil:
		markGroupFail(db, newGroup(createResult.GroupInfo(), ctx.Height()))
	case types.CreateResultFail:
		// do nothing
	}
	m.frozeMiner(db, createResult.FrozenMiners(), ctx)

}

func (m *Manager) tryDoPunish(db types.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext) {
	msg, err := checker.CheckGroupCreatePunishment(ctx)
	if err != nil {
		return
	}
	_, err = m.minerReaderImpl.MinerPenalty(db, msg, ctx.Height())
	if err != nil {
		logger.Error("MinerPenalty error: %v", err)
	}
}

func (m *Manager) saveGroup(db types.AccountDB, group *Group) error {
	return m.poolImpl.add(db, group)
}

func (m *Manager) frozeMiner(db types.AccountDB, frozenMiners [][]byte, ctx types.CheckerContext) {
	for _, p := range frozenMiners {
		addr := common.BytesToAddress(p)
		_, err := m.minerReaderImpl.MinerFrozen(db, addr, ctx.Height())
		if err != nil {
			//todo: should remove this panic? or just print the error?
			panic(logger.Error("saveGroup error: %v", err))
		}
	}
}

// markGroupFail mark group member should upload origin piece
func markGroupFail(db types.AccountDB, group *Group) {
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), originPieceReqKey, []byte{1})
}
