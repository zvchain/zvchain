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
	lru "github.com/hashicorp/golang-lru"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
	"sync"
)

const (
	maxActivatedGroupSkipCounts = 10
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

func (sc skipCounts) merge(sc2 skipCounts) skipCounts {
	ret := make(skipCounts)
	for seed, cnt := range sc {
		ret.addCount(seed, cnt)
	}
	for seed, cnt := range sc2 {
		ret.addCount(seed, cnt)
	}
	return ret
}

type activatedGroupInfoReader interface {
	getActivatedGroupsByHeight(h uint64) []*verifyGroup
	getGroupSkipCountsByHeight(h uint64) map[common.Hash]uint16
}

type skipCounter struct {
	full      skipCounts            // full skip counts from storage
	increment map[uint64]skipCounts // increment skip counts grouped by height
	lock      sync.RWMutex
}

func (sc *skipCounter) addIncrement(h uint64, cnts skipCounts) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	if sc.increment == nil {
		sc.increment = make(map[uint64]skipCounts)
	}
	if sc.increment[h] == nil {
		sc.increment[h] = cnts
	}
}

func (sc *skipCounter) getIncrement(h uint64) skipCounts {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
	if sc.increment == nil {
		return nil
	}
	return sc.increment[h]
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

func (gs *groupSelector) getAllSkipCountsBy(pre *types.BlockHeader, height uint64) skipCounts {
	var (
		fullSkipCounts  skipCounts
		incrementCounts skipCounts
		sc              *skipCounter
	)
	// Should not happen
	if height < pre.Height {
		stdLogger.Panicf("height error: %v %v", height, pre.Height)
		return nil
	}
	v, ok := gs.skipCache.Peek(pre.Hash)
	if ok {
		sc = v.(*skipCounter)
		fullSkipCounts = sc.full
	} else {
		fullSkipCounts = gs.gr.getGroupSkipCountsByHeight(pre.Height)
		sc = &skipCounter{
			full: fullSkipCounts,
		}
		gs.skipCache.ContainsOrAdd(pre.Hash, sc)
	}
	incrementCounts = gs.groupIncrementSkipCountsBetween(sc, fullSkipCounts, pre, height)

	// merge the skip counts
	return fullSkipCounts.merge(incrementCounts)
}

func (gs *groupSelector) getWorkGroupSeedsAt(pre *types.BlockHeader, height uint64, sc skipCounts) []common.Hash {
	groupIS := gs.gr.getActivatedGroupsByHeight(height)
	// Must not happen
	if len(groupIS) == 0 {
		panic("no available groupIS")
	}

	workSeeds := make([]common.Hash, 0)
	for _, g := range groupIS {
		seed := g.header.Seed()
		skipCnt := sc.count(seed)
		if g.header.WorkHeight() == 0 || skipCnt < maxActivatedGroupSkipCounts {
			workSeeds = append(workSeeds, seed)
		}
	}
	stdLogger.Debugf("group selector candidates %v, size %v, at %v", workSeeds, len(workSeeds), height)
	return workSeeds
}

func (gs *groupSelector) selectWithSkipCountInfo(preBH *types.BlockHeader, height uint64, sc skipCounts) common.Hash {
	var hash = calcRandomHash(preBH, height)

	groupSeeds := gs.getWorkGroupSeedsAt(preBH, height, sc)
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

func (gs *groupSelector) doSelect(preBH *types.BlockHeader, height uint64) common.Hash {
	skipCountInfo := gs.getAllSkipCountsBy(preBH, height-1)
	return gs.selectWithSkipCountInfo(preBH, height, skipCountInfo)
}

func (gs *groupSelector) groupIncrementSkipCountsBetween(sc *skipCounter, fullSc skipCounts, preBH *types.BlockHeader, height uint64) skipCounts {
	if height == preBH.Height {
		return skipCounts{}
	}
	skipCnts := sc.getIncrement(height)
	if skipCnts != nil {
		return skipCnts
	}
	// gets increment skip counts before height-1 recursively
	cnts := gs.groupIncrementSkipCountsBetween(sc, fullSc, preBH, height-1)
	// calculates the expect work group at the given height with all skip count infos before height-1
	s := gs.selectWithSkipCountInfo(preBH, height, cnts.merge(fullSc))
	temp := skipCounts{}
	temp.addCount(s, 1)
	ret := cnts.merge(temp)
	sc.addIncrement(height, ret)
	return ret
}

// groupSkipCountsBetween calculates the group skip counts between the given block and the corresponding pre block
func (gs *groupSelector) groupSkipCountsBetween(preBH *types.BlockHeader, height uint64) skipCounts {
	return gs.getAllSkipCountsBy(preBH, height-1)
}
