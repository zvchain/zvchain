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
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

const (
	maxActivatedGroupSkipCounts = 5
)

type skipCounts map[common.Hash]uint16

func (sc skipCounts) count(seed common.Hash) uint16 {
	if v, ok := sc[seed]; ok {
		return v
	}
	return 0
}

func (sc skipCounts) addCount(seed common.Hash, delta uint16) {
	if v, ok := sc[seed]; ok {
		sc[seed] = v + delta
	} else {
		sc[seed] = delta
	}
}

type activatedGroupInfoReader interface {
	getActivatedGroupsByHeight(h uint64) []*verifyGroup
	getGroupSkipCountsByHeight(h uint64) map[common.Hash]uint16
}

type skipCounter struct {
	full      skipCounts            // full skip counts from storage
	increment map[uint64]skipCounts // increment skip counts grouped by height
}

type groupSelector struct {
	gr        activatedGroupInfoReader
	skipCache *lru.Cache // cache skip infos
}

func newGroupSelector(gr activatedGroupInfoReader) *groupSelector {
	return &groupSelector{
		gr:        gr,
		skipCache: common.MustNewLRUCache(50),
	}
}

func duplicateGroupSeed(g *verifyGroup, skipCount uint16) []common.Hash {
	// always expands for genesis group
	if g.header.WorkHeight() == 0 {
		skipCount = 0
	}
	if skipCount >= maxActivatedGroupSkipCounts {
		return []common.Hash{}
	} else {
		dup := maxActivatedGroupSkipCounts - int(skipCount)
		ret := make([]common.Hash, dup)
		for i := range ret {
			ret[i] = g.header.Seed()
		}
		return ret
	}
}

func (gs *groupSelector) getAllSkipCountsBy(pre *types.BlockHeader, height uint64) skipCounts {
	var (
		fullSkipCounts  skipCounts
		incrementCounts skipCounts
	)
	v, ok := gs.skipCache.Peek(pre.Hash)
	if ok {
		ct := v.(*skipCounter)
		fullSkipCounts = ct.full
		incrementCounts = ct.increment[height]
		if incrementCounts == nil {
			incrementCounts = gs.groupSkipCountsBetween(pre, height)
			ct.increment[height] = incrementCounts
		}
	} else {
		fullSkipCounts = gs.gr.getGroupSkipCountsByHeight(pre.Height)
		incrementCounts = gs.groupSkipCountsBetween(pre, height)
		increments := make(map[uint64]skipCounts)
		increments[height] = incrementCounts
		ct := &skipCounter{
			full:      fullSkipCounts,
			increment: increments,
		}
		gs.skipCache.Add(pre.Hash, ct)
	}

	// merge the skip counts
	ret := make(skipCounts)
	for seed, cnt := range fullSkipCounts {
		ret.addCount(seed, cnt)
	}
	for seed, cnt := range incrementCounts {
		ret.addCount(seed, cnt)
	}
	return ret
}

func (gs *groupSelector) getWorkGroupSeedsAt(pre *types.BlockHeader, height uint64) []common.Hash {
	groupIS := gs.gr.getActivatedGroupsByHeight(height)
	// Must not happen
	if len(groupIS) == 0 {
		panic("no available groupIS")
	}
	skipCounts := gs.getAllSkipCountsBy(pre, height)

	workSeedsWithDuplication := make([]common.Hash, 0)
	logs := make([]string, 0)
	for _, g := range groupIS {
		seed := g.header.Seed()
		skipCnt := skipCounts.count(seed)
		dupSeeds := duplicateGroupSeed(g, skipCnt)
		logs = append(logs, fmt.Sprintf("%v(%v)", seed, len(dupSeeds)))
		workSeedsWithDuplication = append(workSeedsWithDuplication, dupSeeds...)
	}
	stdLogger.Debugf("group selector candidates %v, size %v, at %v", logs, len(logs), height)
	return workSeedsWithDuplication
}

func (gs *groupSelector) doSelect(preBH *types.BlockHeader, height uint64) common.Hash {
	var hash = calcRandomHash(preBH, height)

	groupSeeds := gs.getWorkGroupSeedsAt(preBH, height)
	// Must not happen
	if len(groupSeeds) == 0 {
		panic("no available groups")
	}

	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(len(groupSeeds))))

	selectedGroup := groupSeeds[index.Int64()]

	stdLogger.Debugf("selected %v at %v", selectedGroup, height)
	return selectedGroup
}

// groupSkipCountsBetween calculates the group skip counts between the given block and the corresponding pre block
func (gs *groupSelector) groupSkipCountsBetween(preBH *types.BlockHeader, height uint64) skipCounts {
	skipMap := make(skipCounts)
	for h := preBH.Height + 1; h < height; h++ {
		s := gs.doSelect(preBH, h)
		skipMap.addCount(s, 1)
	}
	return skipMap
}
