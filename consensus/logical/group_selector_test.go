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
	"math/rand"
	"testing"
)

func TestSkipCounts(t *testing.T) {
	ret := make(skipCounts)

	ret.addCount(common.HexToHash("0x1"), 3)
	c := ret.count(common.HexToHash("0x1"))
	if c != 3 {
		t.Fatal("count error")
	}
}

type activatedGroupReader4Test struct {
	groups []*verifyGroup
}

func (r *activatedGroupReader4Test) getActivatedGroupsByHeight(h uint64) []*verifyGroup {
	gs := make([]*verifyGroup, 0)
	for _, g := range r.groups {
		if g.header.workHeight <= h && g.header.dismissHeight > h {
			gs = append(gs, g)
		}
	}
	return gs
}

func (r *activatedGroupReader4Test) getGroupSkipCountsByHeight(h uint64) map[common.Hash]uint16 {
	skip := make(skipCounts)
	for i, g := range r.groups {
		if i < 4 {
			skip.addCount(g.header.seed, uint16(2*i))
		}
	}
	return skip
}

func newActivatedGroupReader4Test() *activatedGroupReader4Test {
	return &activatedGroupReader4Test{
		groups: make([]*verifyGroup, 0),
	}
}

func (r *activatedGroupReader4Test) init() {
	for h := uint64(0); h < 1000; h += 10 {
		gh := newGroupHeader4Test(h, h+200)
		g := &verifyGroup{header: gh}
		r.groups = append(r.groups, g)
	}
}

func buildGroupSelector4Test() *groupSelector {
	gr := newActivatedGroupReader4Test()
	gr.init()
	return newGroupSelector(gr)
}

func TestGroupSelector_getWorkGroupSeedsAt(t *testing.T) {
	gs := buildGroupSelector4Test()
	rnd := make([]byte, 32)
	rand.Read(rnd)
	bh := &types.BlockHeader{
		Height: 100,
		Random: rnd,
	}
	bh.Hash = bh.GenHash()
	sc := skipCounts{}
	sc.addCount(common.BytesToHash(common.Uint64ToByte(10)), 10)
	sc.addCount(common.BytesToHash(common.Uint64ToByte(20)), 14)
	sc.addCount(common.BytesToHash(common.Uint64ToByte(30)), 4)
	sc.addCount(common.BytesToHash(common.Uint64ToByte(0)), 14)
	seeds := gs.getWorkGroupSeedsAt(bh, 102, sc)
	t.Log(seeds)
}

func TestGroupSelector_getAllSkipCountsBy(t *testing.T) {
	gs := buildGroupSelector4Test()
	rnd := make([]byte, 32)
	rand.Read(rnd)
	bh := &types.BlockHeader{
		Height: 100,
		Random: rnd,
	}
	bh.Hash = bh.GenHash()

	for h := bh.Height + 1; h < 1000; h++ {
		avil := gs.gr.getActivatedGroupsByHeight(h)
		sc := gs.getAllSkipCountsBy(bh, h)
		work := gs.getWorkGroupSeedsAt(bh, h, sc)
		t.Log(len(avil), len(work), len(sc))
		if len(avil) < len(work) {
			t.Errorf("work group num error")
		}
	}
}

func TestGroupSelector_doSelect(t *testing.T) {
	gs := buildGroupSelector4Test()
	rnd := make([]byte, 32)
	rand.Read(rnd)
	bh := &types.BlockHeader{
		Height: 100,
		Random: rnd,
	}
	bh.Hash = bh.GenHash()

	for h := bh.Height + 1; h < 1000; h++ {
		selected := gs.doSelect(bh, h)
		t.Log(selected)
	}
}
