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

func TestDuplicateSeed(t *testing.T) {
	g := &verifyGroup{
		header: &groupHeader{
			seed:       common.HexToHash("0x1"),
			workHeight: 10,
		},
	}

	seeds := duplicateGroupSeed(g, 0)
	if len(seeds) != maxActivatedGroupSkipCounts {
		t.Errorf("duplicate error on count 0")
	}

	seeds = duplicateGroupSeed(g, 3)
	if len(seeds) != maxActivatedGroupSkipCounts-3 {
		t.Errorf("duplicate error on count 3")
	}
	t.Log(seeds)
	seeds = duplicateGroupSeed(g, 5)
	if len(seeds) != maxActivatedGroupSkipCounts-5 {
		t.Errorf("duplicate error on count 5")
	}

	seeds = duplicateGroupSeed(g, 15)
	if len(seeds) != 0 {
		t.Errorf("duplicate error on count 15")
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
