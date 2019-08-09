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
	lru "github.com/hashicorp/golang-lru"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type pool struct {
	chain         chainReader
	genesis       *group     // genesis group
	cachedBySeed  *lru.Cache // cache for groups. kv: types.SeedI -> types.GroupI
	cachedByEpoch *lru.Cache
}

func newPool(chain chainReader) *pool {
	return &pool{
		chain:         chain,
		cachedBySeed:  common.MustNewLRUCache(120),
		cachedByEpoch: common.MustNewLRUCache(10),
	}
}

func (p *pool) initPool(db types.AccountDB, gen *types.GenesisInfo) error {
	p.genesis = p.get(db, gen.Group.Header().Seed())
	return nil
}

func (p *pool) initGenesis(db types.AccountDB, genesis *types.GenesisInfo) error {
	p.genesis = p.get(db, genesis.Group.Header().Seed())
	if p.genesis == nil {
		p.genesis = newGroup(genesis.Group, nil)
		err := p.add(db, p.genesis)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *pool) add(db types.AccountDB, group *group) error {
	logger.Debugf("save group, height is %d, seed is %s", group.HeaderD.GroupHeightD, group.HeaderD.SeedD)
	byteData, err := msgpack.Marshal(group)
	if err != nil {
		return err
	}

	db.SetData(common.HashToAddress(group.Header().Seed()), groupDataKey, byteData)
	p.cachedBySeed.Add(group.Header().Seed(), group)
	p.saveTopGroup(db, group)

	return nil
}

func (p *pool) invalidateEpochGroupCache(ep types.Epoch) {
	p.cachedByEpoch.Remove(ep.End())
}

func (p *pool) get(db types.AccountDB, seed common.Hash) *group {
	if g, ok := p.cachedBySeed.Get(seed); ok {
		return g.(*group)
	}

	byteData := db.GetData(common.HashToAddress(seed), groupDataKey)
	if byteData != nil {
		var gr group
		err := msgpack.Unmarshal(byteData, &gr)
		if err != nil {
			logger.Errorf("Unmarshal failed when get group from db. seed = %v", seed)
			return nil
		}
		p.cachedBySeed.ContainsOrAdd(seed, &gr)
		return &gr
	}
	return nil
}

func (p *pool) iterateByHeight(height uint64, iterFunc func(g *group) bool) {
	db, err := p.chain.AccountDBAt(height)
	if err != nil {
		logger.Error("failed to get last db", err)
		return
	}

	for current := p.getTopGroup(db); current != nil; current = p.get(db, current.HeaderD.PreSeed) {
		if !iterFunc(current) {
			break
		}
	}
}

func (p *pool) getGroupsByHeightRange(start, end uint64) []*group {
	startDB, err := p.chain.AccountDBAt(start)
	if err != nil {
		logger.Errorf("get account db fail at %v %v", start, err)
		return nil
	}
	startTopSeed := p.getTopGroupSeed(startDB)

	rs := make([]*group, 0)

	p.iterateByHeight(end, func(g *group) bool {
		if g.HeaderD.Seed() != startTopSeed {
			rs = append(rs, g)
			return true
		} else {
			return false
		}
	})

	return rs
}

func (p *pool) getGroupsByEpoch(ep types.Epoch) []*group {
	if ep.End() < p.chain.Height() {
		if v, ok := p.cachedByEpoch.Get(ep.End()); ok {
			return v.([]*group)
		}
	}
	gs := p.getGroupsByHeightRange(ep.Start(), ep.End())
	// groups of the epoch cached iff the epoch is complete
	if ep.End() < p.chain.Height() {
		p.cachedByEpoch.Add(ep.End(), gs)
	}
	return gs
}

func (p *pool) groupsAfter(height uint64, limit int) []types.GroupI {
	rs := make([]types.GroupI, 0)

	p.iterateByHeight(common.MaxUint64, func(g *group) bool {
		if g.Header().GroupHeight() > height {
			return false
		}
		rs = append(rs, g)
		return true
	})
	rs = revert(rs)
	if limit < len(rs) {
		return rs[0:limit]
	}
	return rs
}

func (p *pool) count(db types.AccountDB) uint64 {
	return p.getTopGroup(db).HeaderD.GroupHeight() + 1
}

func (p *pool) saveTopGroup(db types.AccountDB, g *group) {
	db.SetData(common.GroupTopAddress, topGroupKey, g.HeaderD.SeedD.Bytes())
}

func (p *pool) getTopGroup(db types.AccountDB) *group {
	return p.get(db, p.getTopGroupSeed(db))
}

func (p *pool) getTopGroupSeed(db types.AccountDB) common.Hash {
	bs := db.GetData(common.GroupTopAddress, topGroupKey)
	return common.BytesToHash(bs)
}

func revert(s []types.GroupI) []types.GroupI {
	if s == nil {
		return nil
	}
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
