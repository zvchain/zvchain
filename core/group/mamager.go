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
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

// Manager implements groupContextProvider in package consensus
type Manager struct {
	checker types.GroupCreateChecker
	s types.GroupStoreReader
	p types.GroupPacketSender
}

func (m *Manager)GetGroupStoreReader()types.GroupStoreReader  {
	return m.s
}

func (m *Manager)GetGroupPacketSender()types.GroupPacketSender  {
	return m.p
}

func (m *Manager)RegisterGroupCreateChecker(checker types.GroupCreateChecker)  {
	m.checker = checker
}

func NewManager(chain chainReader,ticker *ticker.GlobalTicker) Manager{
	store := NewStore(chain.LatestStateDB())
	packetSender := NewPacketSender(chain)
	managerImpl := Manager{s:store,p:packetSender}
	return managerImpl
}

// RegularCheck try to create group and do punishment
func RegularCheck(db *account.AccountDB, checker types.GroupCreateChecker, chain chainReader) {
	ctx := &CheckerContext{chain.Height()}
	tryCreateGroup(db,checker,ctx)
	tryDoPunish(db, checker,ctx)
	freshActiveGroup(db)
}

func tryCreateGroup(db *account.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext) {
	createResult := checker.CheckGroupCreateResult(ctx)
	if createResult == nil {
		return
	}
	if createResult.Err() != nil {
		return
	}
	switch createResult.Code() {
	case types.CreateResultSuccess:
		_ = saveGroup(db, newGroup(createResult.GroupInfo()))
	case types.CreateResultMarkEvil:
		_ = markGroupFail(db, newGroup(createResult.GroupInfo()))
	case types.CreateResultFail:
		// do nothing
	}
	_ = markEvil(db, createResult.FrozenMiners())
}

func tryDoPunish(db *account.AccountDB, checker types.GroupCreateChecker, ctx types.CheckerContext) {
	_, err := checker.CheckGroupCreatePunishment(ctx)
	if err != nil {
		return
	}
	//for _, p := range msg.PenaltyTarget() {
	//	//TODO: reduce stake
	//}
	//
	//for _, r := range msg.RewardTarget() {
	//	// TODO: add balance
	//}

}



func saveGroup(db *account.AccountDB, group *Group) error {
	byteData, err := msgpack.Marshal(group)
	if err != nil {
		return err
	}

	byteHeader, err := msgpack.Marshal(group.Header().(*GroupHeader))
	if err != nil {
		return err
	}
	life := newGroupLife(group)
	lifeData, err := msgpack.Marshal(life)
	if err != nil {
		return err
	}

	db.SetData(common.GroupWaitingAddress, group.HeaderD.Seed().Bytes(), lifeData)
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), groupDataKey, byteData)
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), groupHeaderKey, byteHeader)

	return nil
}

func freshActiveGroup(db *account.AccountDB) error {

	return nil
}

func markEvil(db *account.AccountDB, frozenMiners [][]byte) error {
	if frozenMiners == nil || len(frozenMiners) == 0 {
		return nil
	}
	//TODO: call miner interface
	return nil
}

// markGroupFail mark group member should upload origin piece
func markGroupFail(db *account.AccountDB, group *Group) error {
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), originPieceReqKey, []byte{1})
	return nil
}
