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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"testing"
)

type group4GroupTest struct {
	header *groupHeader
}

func (g *group4GroupTest) Header() types.GroupHeaderI {
	return g.header
}

func (g *group4GroupTest) Members() []types.MemberI {
	return nil
}

type groupReader4Test struct {
	groups []types.GroupI
}

func (gr *groupReader4Test) GetGroupSkipCountsAt(h uint64, groups []types.GroupI) (map[common.Hash]uint16, error) {
	return nil, nil
}

func newGroupReader4Test(num uint64) *groupReader4Test {
	gr := &groupReader4Test{
		groups: make([]types.GroupI, 0),
	}
	gr.initGroups(num)
	return gr
}
func newGroupHeader4Test(wh, dh uint64) *groupHeader {
	return &groupHeader{
		seed:          common.BytesToHash(common.Uint64ToByte(wh)),
		workHeight:    wh,
		dismissHeight: dh,
	}
}

func newGroup4Test(wh, dh uint64) *group4GroupTest {
	return &group4GroupTest{
		header: newGroupHeader4Test(wh, dh),
	}
}

func (gr *groupReader4Test) initGroups(num uint64) {
	for n := uint64(0); n < num; n++ {
		gr.groups = append(gr.groups, newGroup4Test(n, 10*n))
	}
}

func (gr *groupReader4Test) GetActivatedGroupsAt(height uint64) []types.GroupI {
	gs := make([]types.GroupI, 0)
	for _, g := range gr.groups {
		if g.Header().DismissHeight() > height && g.Header().WorkHeight() <= height {
			gs = append(gs, g)
		}
	}
	return gs
}

func (gr *groupReader4Test) GetLivedGroupsAt(height uint64) []types.GroupI {
	gs := make([]types.GroupI, 0)
	for _, g := range gr.groups {
		if g.Header().DismissHeight() > height {
			gs = append(gs, g)
		}
	}
	return gs
}

func (gr *groupReader4Test) GetGroupBySeed(seedHash common.Hash) types.GroupI {
	for _, g := range gr.groups {
		if g.Header().Seed() == seedHash {
			return g
		}
	}
	return nil
}

func (gr *groupReader4Test) GetGroupHeaderBySeed(seedHash common.Hash) types.GroupHeaderI {
	g := gr.GetGroupBySeed(seedHash)
	if g != nil {
		return g.Header()
	}
	return nil
}

func (gr *groupReader4Test) Height() uint64 {
	return uint64(len(gr.groups)) + 1
}

func TestGroupReader_GetGroupBySeed(t *testing.T) {
	gr := newGroupReader(newGroupReader4Test(100), nil)

	for n := 0; n < 10; n++ {
		g := gr.getGroupBySeed(common.BytesToHash(common.Uint64ToByte(uint64(n))))
		t.Log(g.header.seed, g.header.workHeight, g.header.dismissHeight)
	}

	for n := 0; n < 10; n++ {
		g := gr.getGroupBySeed(common.BytesToHash(common.Uint64ToByte(uint64(n))))
		t.Log(g.header.seed, g.header.workHeight, g.header.dismissHeight)
	}
}

func TestGroupReader_GetActiveGroups(t *testing.T) {
	gr := newGroupReader(newGroupReader4Test(100), nil)

	for n := 0; n < 10; n++ {
		g := gr.getGroupBySeed(common.BytesToHash(common.Uint64ToByte(uint64(n))))
		t.Log(g.header.seed, g.header.workHeight, g.header.dismissHeight)
	}

	gs := gr.getActivatedGroupsByHeight(30)
	for _, g := range gs {
		t.Log(g.header.seed, g.header.workHeight, g.header.dismissHeight)
	}
}
