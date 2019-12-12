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
	"sync/atomic"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type hitMeter struct {
	total uint64
	hit   uint64
}

func (m *hitMeter) increase(hit bool) {
	nv := atomic.AddUint64(&m.total, 1)
	// Overflows and reset
	if nv == 0 {
		m.hit = uint64(0)
	}
	if hit {
		atomic.AddUint64(&m.hit, 1)
	}

	if nv%200 == 0 {
		logger.Debugf("group cache hit %v/%v(%v)", m.hit, nv, m.hitRate())
	}
}

func (m *hitMeter) hitRate() float64 {
	if m.total == 0 {
		return 0
	}
	return float64(m.hit) / float64(m.total)
}

type pool struct {
	chain        chainReader
	genesis      *group     // genesis group
	cachedBySeed *lru.Cache // cache for groups. kv: types.SeedI -> types.GroupI
	meter        *hitMeter
}

func newPool(chain chainReader) *pool {
	return &pool{
		chain:        chain,
		cachedBySeed: common.MustNewLRUCache(120),
		meter:        &hitMeter{},
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

func (p *pool) updateSkipCount(db types.AccountDB, seed common.Hash, cnt uint16) {
	skip := p.getSkipCount(db, seed)
	if cnt == 0 {
		if skip > 0 {
			db.RemoveData(common.HashToAddress(seed), skipCounterKey)
			logger.Debugf("remove skip count %v", seed)
		}
	} else {
		db.SetData(common.HashToAddress(seed), skipCounterKey, common.UInt16ToByte(cnt+skip))
		logger.Debugf("update skip count %v %v", seed, cnt+skip)
	}
}

func (p *pool) getSkipCount(db types.AccountDB, seed common.Hash) uint16 {
	bs := db.GetData(common.HashToAddress(seed), skipCounterKey)
	if len(bs) == 0 {
		return 0
	}
	return common.ByteToUInt16(bs)
}

func (p *pool) get(db types.AccountDB, seed common.Hash) *group {
	var (
		hit = false
		v   interface{}
	)
	defer func() {
		p.meter.increase(hit)
	}()

	// Get from cache
	if v, hit = p.cachedBySeed.Get(seed); hit {
		return v.(*group)
	}
	// Get from db
	if db == nil {
		adb, err := p.chain.LatestAccountDB()
		if err != nil {
			logger.Error("failed to get last db", err)
			return nil
		}
		db = adb
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

// iterateGroups visit the groups from top to beginning(genesis group excluded)
func (p *pool) iterateGroups(iterFunc func(g *group) bool) {
	db, err := p.chain.LatestAccountDB()
	if err != nil {
		logger.Error("failed to get last db", err)
		return
	}

	for current := p.getTopGroup(db); current != nil && current.Header().WorkHeight() > 0; current = p.get(db, current.HeaderD.PreSeed) {
		if !iterFunc(current) {
			break
		}
	}
}

func (p *pool) groupsAfter(height uint64, limit int) []types.GroupI {
	rs := make([]types.GroupI, 0)

	p.iterateGroups(func(g *group) bool {
		if g.Header().GroupHeight() < height {
			return false
		}
		rs = append(rs, g)
		return true
	})
	if height == 0 {
		rs = append(rs, p.genesis)
	}
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
