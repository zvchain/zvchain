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
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

type Manager struct {
	checker types.GroupCreateChecker
}

// RegularCheck try to create group and do punishment
func RegularCheck(db *account.AccountDB, checker types.GroupCreateChecker, chain chainReader) {
	ctx := &CheckerContext{chain.Height()}
	tryCreateGroup(db,checker,ctx)
	tryDoPunish(db, checker,ctx)
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
	msg, err := checker.CheckGroupCreatePunishment(ctx)
	if err != nil {
		return
	}
	for _, p := range msg.PenaltyTarget() {
		//TODO: reduce stake
	}

	for _, r := range msg.RewardTarget() {
		// TODO: add balance
	}

}

type groupLife struct {
	begin uint64
	end   uint64
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
	life := &groupLife{group.HeaderD.WorkHeight(), group.HeaderD.DismissHeight()}
	lifeData, err := msgpack.Marshal(life)
	if err != nil {
		return err
	}

	db.SetData(common.GroupActiveAddress, group.HeaderD.Seed().Bytes(), lifeData)
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), groupDataKey, byteData)
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), groupHeaderKey, byteHeader)

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
	db.SetData(common.HashToAddress(group.HeaderD.Seed()), originPieceReqKey, []byte("1"))
	return nil
}
