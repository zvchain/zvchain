////   Copyright (C) 2018 ZVChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"fmt"
	"testing"

	"github.com/zvchain/zvchain/middleware/types"
)

func TestGroupChain_Add(t *testing.T) {
	err := initContext4Test()
	if err != nil {
		t.Fatalf("fail to initContext4Test")
	}

	defer clear()

	id1 := genHash("test1")
	group1 := &types.Group{
		ID:     id1,
		Header: &types.GroupHeader{},
	}

	err = GroupChainImpl.AddGroup(group1)
	if err != nil {
		t.Fatalf("fail to add group1: %s", err)
	}

	if 1 != GroupChainImpl.Height() {
		t.Fatalf("fail to add group1")
	}

	id2 := genHash("test2")
	group2 := &types.Group{
		ID: id2,
		Header: &types.GroupHeader{
			PreGroup: id1,
		},
	}

	err = GroupChainImpl.AddGroup(group2)
	if err != nil {
		t.Fatalf("fail to add group2")
	}

	id3 := genHash("test3")
	group3 := &types.Group{
		ID: id3,
		Header: &types.GroupHeader{
			PreGroup: id2,
		},
	}

	err = GroupChainImpl.AddGroup(group3)
	if err != nil {
		fmt.Printf("fail to add group: %s", err)
		t.Fatalf("fail to add group4")
	}

	h := GroupChainImpl.Height()
	fmt.Printf("h = %d\n", h)
	if 3 != GroupChainImpl.Height() {
		t.Fatalf("fail to add group4")
	}

	group := GroupChainImpl.getGroupByID(id2)
	if nil == group {
		t.Fatalf("fail to GetGroupById2")
	}

	group = GroupChainImpl.getGroupByID(id3)
	if nil == group {
		t.Fatalf("fail to GetGroupById3")
	}

	chain := GroupChainImpl
	iter := chain.groupsHeight.NewIterator()
	defer iter.Release()

	limit := 100
	for iter.Next() {
		gid := iter.Value()
		g := chain.getGroupByID(gid)
		if g != nil {
			t.Log(g.GroupHeight, iter.Key(), g.ID)
			limit--
		}
	}
}
