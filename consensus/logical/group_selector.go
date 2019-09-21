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
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
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

type activatedGroupInfoReader interface {
	// GetActivatedGroupsAt gets activated groups at the given height
	GetActivatedGroupsAt(height uint64) []types.GroupI
	GetGroupSkipCountsAt(h uint64, groups []types.GroupI) (map[common.Hash]uint16, error)
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

func (gs *groupSelector) getWorkGroupSeedsAt(pre *types.BlockHeader, height uint64) []common.Hash {
	groupIS := gs.gr.GetActivatedGroupsAt(height)
	// Must not happen
	if len(groupIS) == 0 {
		panic("no available groupIS")
	}

	sc := skipCounts{}
	// Read skip count infos
	ep := types.EpochAt(height)
	skipCountBeginHeight := ep.Add(-types.GroupActivateEpochGap).Start()
	if skipCountBeginHeight > 0 {
		skipCnts, err := gs.gr.GetGroupSkipCountsAt(skipCountBeginHeight-1, groupIS)
		if err != nil {
			stdLogger.Errorf("GetGroupSkipCountsAt %v error %v", height, err)
			return nil
		}
		sc = skipCounts(skipCnts)
	}

	workSeeds := make([]common.Hash, 0)
	for _, g := range groupIS {
		seed := g.Header().Seed()
		skipCnt := sc.count(seed)
		if g.Header().WorkHeight() == 0 || skipCnt < maxActivatedGroupSkipCounts {
			workSeeds = append(workSeeds, seed)
		}
	}
	stdLogger.Debugf("skip counts %v, group selector candidates %v, size %v, at %v", sc, workSeeds, len(workSeeds), height)
	return workSeeds
}

func (gs *groupSelector) doSelect(preBH *types.BlockHeader, height uint64) common.Hash {
	var hash = calcRandomHash(preBH, height)
	return gs.doSelectWithRandom(hash, preBH, height)
}

func (gs *groupSelector) doSelectWithRandom(rand common.Hash, preBH *types.BlockHeader, height uint64) common.Hash {
	groupSeeds := gs.getWorkGroupSeedsAt(preBH, height)
	// Must not happen
	if len(groupSeeds) == 0 {
		panic("no available groups")
	}

	value := rand.Big()
	index := value.Mod(value, big.NewInt(int64(len(groupSeeds))))

	selectedGroup := groupSeeds[index.Int64()]

	stdLogger.Debugf("selected %v at %v", selectedGroup, height)
	return selectedGroup
}

// groupSkipCountsBetween calculates the group skip counts between the given block and the corresponding pre block
func (gs *groupSelector) groupSkipCountsBetween(preBH *types.BlockHeader, height uint64) skipCounts {
	sc := make(skipCounts)
	h := preBH.Height + 1
	rand := calcRandomHash(preBH, h)
	for ; h < height; h++ {
		expectedSeed := gs.doSelectWithRandom(rand, preBH, h)
		sc.addCount(expectedSeed, 1)
		rand = base.Data2CommonHash(rand.Bytes())
	}
	return sc
}
