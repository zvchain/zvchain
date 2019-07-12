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
	"bytes"

	lru "github.com/hashicorp/golang-lru"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type pool struct {
	genesis    *group     // genesis group
	groupCache *lru.Cache // cache for groups. key is types.Seedi; value is types.Groupi
}

func newPool() *pool {
	return &pool{
		groupCache: common.MustNewLRUCache(120),
	}
}

func (p *pool) initPool(db types.AccountDB, gen *types.GenesisInfo) error {
	p.genesis = p.get(db, gen.Group.Header().Seed())
	return nil
}

func (p *pool) initGenesis(db types.AccountDB, genesis *types.GenesisInfo) error {
	p.genesis = p.get(db, genesis.Group.Header().Seed())
	if p.genesis == nil {
		p.genesis = newGroup(genesis.Group, 0, nil)
		err := p.add(db, p.genesis)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *pool) add(db types.AccountDB, group *group) error {
	logger.Debugf("save group, height is %d, seed is %s", group.HeaderD.BlockHeight, group.HeaderD.SeedD)
	byteData, err := msgpack.Marshal(group)
	if err != nil {
		return err
	}

	p.groupCache.Add(group.Header().Seed(), group)
	db.SetData(common.HashToAddress(group.Header().Seed()), groupDataKey, byteData)

	p.saveTopGroup(db, group)

	return nil
}

func (p *pool) resetToTop(db types.AccountDB, height uint64) {

}

func (p *pool) minerLiveGroupCount(chain chainReader, addr common.Address, height uint64) int {
	lived := p.getLives(chain, height)
	count := 0
	for _, g := range lived {
		for _, mem := range g.MembersD {
			if bytes.Equal(addr.Bytes(), mem.Id) {
				count++
				break
			}
		}
	}
	return count
}

func (p *pool) get(db types.AccountDB, seed common.Hash) *group {
	if g, ok := p.groupCache.Get(seed); ok {
		return g.(*group)
	}

	byteData := db.GetData(common.HashToAddress(seed), groupDataKey)
	if byteData != nil {
		var gr group
		err := msgpack.Unmarshal(byteData, &gr)
		if err != nil {
			return nil
		}
		p.groupCache.ContainsOrAdd(seed, &gr)
		return &gr
	}
	return nil
}

func (p *pool) getActives(chain chainReader, height uint64) []*group {
	rs := make([]*group, 0)
	db := chain.LatestStateDB()

	for current := p.getTopGroup(db); current != nil && current.HeaderD.DismissHeightD > height; current = p.get(db, current.HeaderD.PreSeed) {
		if current.HeaderD.BlockHeight == 0 {
			break
		}
		if current.HeaderD.WorkHeightD <= height && current.HeaderD.BlockHeight <= height {
			rs = append(rs, current)
		}
	}

	//add p.genesis
	rs = append(rs, p.genesis)
	return rs
}

func (p *pool) getLives(chain chainReader, height uint64) []*group {
	rs := make([]*group, 0)
	db := chain.LatestStateDB()

	for current := p.getTopGroup(db); current != nil && current.HeaderD.DismissHeightD > height; current = p.get(db, current.HeaderD.PreSeed) {
		if current.HeaderD.BlockHeight == 0 {
			break
		}
		if current.HeaderD.BlockHeight <= height {
			rs = append(rs, current)
		}
	}

	//add p.genesis
	rs = append(rs, p.genesis)
	return rs

}

func (p *pool) groupsAfter(chain chainReader, height uint64, limit int) []types.GroupI {
	//TODO: optimize it
	rs := make([]types.GroupI, 0)
	db := chain.LatestStateDB()
	for current := p.getTopGroup(db); current != nil; current = p.get(db, current.HeaderD.PreSeed) {
		if current.HeaderD.GroupHeight < height {
			break
		}
		rs = append(rs, current)
		if current.HeaderD.BlockHeight == 0 {
			break
		}
	}
	rs = revert(rs)
	if limit < len(rs) {
		return rs[0:limit]
	}
	return rs
}

func (p *pool) count(db types.AccountDB) uint64 {
	return p.getTopGroup(db).HeaderD.GroupHeight + 1
}

func (p *pool) saveTopGroup(db types.AccountDB, g *group) {
	db.SetData(common.GroupTopAddress, topGroupKey, g.HeaderD.SeedD.Bytes())
}

func (p *pool) getTopGroup(db types.AccountDB) *group {
	bs := db.GetData(common.GroupTopAddress, topGroupKey)
	return p.get(db, common.BytesToHash(bs))
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
