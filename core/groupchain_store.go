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

package core

import (
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

func (chain *GroupChain) getGroupByHeight(height uint64) *types.Group {
	groupID, _ := chain.groupsHeight.Get(common.UInt64ToByte(height))
	if nil != groupID {
		return chain.getGroupByID(groupID)
	}

	return nil
}

func (chain *GroupChain) hasGroup(id []byte) bool {
	ok, _ := chain.groups.Has(id)
	return ok
}

func (chain *GroupChain) getGroupByID(id []byte) *types.Group {
	data, _ := chain.groups.Get(id)
	if nil == data || 0 == len(data) {
		return nil
	}

	var group types.Group
	err := msgpack.Unmarshal(data, &group)
	if err != nil {
		return nil
	}
	return &group
}

func (chain *GroupChain) getGroupsAfterHeight(height uint64, limit int) []*types.Group {
	result := make([]*types.Group, 0)
	iter := chain.groupsHeight.NewIterator()
	defer iter.Release()

	if !iter.Seek(common.UInt64ToByte(height)) {
		return result
	}

	for limit > 0 {
		gid := iter.Value()
		g := chain.getGroupByID(gid)
		if g != nil {
			result = append(result, g)
			limit--
		}
		if !iter.Next() {
			break
		}
	}
	return result
}

func (chain *GroupChain) loadLastGroup() *types.Group {
	gid, err := chain.groups.Get([]byte(groupStatusKey))
	if err != nil {
		return nil
	}
	return chain.getGroupByID(gid)
}

func (chain *GroupChain) commitGroup(group *types.Group) error {
	data, err := msgpack.Marshal(group)
	if nil != err {
		return err
	}

	batch := chain.groups.CreateLDBBatch()
	defer batch.Reset()

	if err := chain.groups.AddKv(batch, group.ID, data); err != nil {
		return err
	}
	if err := chain.groupsHeight.AddKv(batch, common.UInt64ToByte(group.GroupHeight), group.ID); err != nil {
		return err
	}
	if err := chain.groups.AddKv(batch, []byte(groupStatusKey), group.ID); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		return err
	}

	chain.lastGroup = group

	return nil
}
