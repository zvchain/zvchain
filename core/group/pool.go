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
	genesis     *group       // genesis group
	groupCache  *lru.Cache   // cache for groups. key is types.Seedi; value is types.Groupi
}

func newPool() *pool {
	return &pool{
		groupCache:  common.MustNewLRUCache(120),
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

	for current := p.getTopGroup(db) ; current != nil && current.HeaderD.DismissHeightD > height; current = p.get(db, current.HeaderD.PreSeed) {
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

	for current := p.getTopGroup(db) ; current != nil && current.HeaderD.DismissHeightD > height; current = p.get(db, current.HeaderD.PreSeed) {
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
	rs := make([]types.GroupI, 0, limit)
	db := chain.LatestStateDB()
	current := p.getTopGroup(db)

	for iter := p.get(db, current.HeaderD.SeedD); iter != nil; current = iter {
		if iter.HeaderD.BlockHeight == 0 {
			break
		}
	}

	//add p.genesis
	rs = append(rs, p.genesis)
	return rs


	//for _, v := range p.waitingList {
	//	rs = append(rs, p.get(chain.LatestStateDB(), v.Seed()))
	//	if len(rs) >= limit {
	//		return rs
	//	}
	//}
	//for _, v := range p.activeList {
	//	rs = append(rs, p.get(chain.LatestStateDB(), v.Seed()))
	//	if len(rs) >= limit {
	//		return rs
	//	}
	//}

	return rs
}

// move the group to dismiss db
//func (p *pool) toDismiss(db types.AccountDB, gl *groupLife) {
//	db.RemoveData(common.GroupActiveAddress, gl.SeedD.Bytes())
//	db.SetData(common.GroupDismissAddress, gl.SeedD.Bytes(), []byte{1})
//}

func (p *pool) count(db types.AccountDB) uint64 {
	return p.getTopGroup(db).HeaderD.GroupHeight +1
	//rs := len(p.waitingList) + len(p.activeList)
	//iter := db.DataIterator(common.GroupDismissAddress, []byte{})
	//if iter != nil {
	//	for iter.Next() {
	//		rs++
	//	}
	//}
	//return uint64(rs)
}

func (p *pool) saveTopGroup(db types.AccountDB, g *group) {
	db.SetData(common.GroupTopAddress, topGroupKey, g.HeaderD.SeedD.Bytes())
}

func (p *pool) getTopGroup(db types.AccountDB) *group {
	bs := db.GetData(common.GroupTopAddress, topGroupKey)
	return p.get(db, common.BytesToHash(bs))
	//
	//if bs != nil {
	//	var g group
	//	err := msgpack.Unmarshal(bs, &g)
	//	if err != nil {
	//		logger.Errorf("getTopGroup error:%v, Height:%v", err)
	//		return nil
	//	}
	//	return &g
	//}
	//logger.Errorf("getTopGroup returns nil")
	//return nil
}

//
//func removeFirst(queue []*groupLife) []*groupLife {
//	// this case should never happen, we already use the sPeek to check if len is 0
//	if len(queue) == 0 {
//		return nil
//	}
//	return queue[1:]
//}
//
//func removeLast(queue []*groupLife) []*groupLife {
//	// this case should never happen, we already use the peek to check if len is 0
//	if len(queue) == 0 {
//		return nil
//	}
//	return queue[:len(queue)-1]
//
//}
//
//func push(queue []*groupLife, gl *groupLife) []*groupLife {
//	for _, v := range queue {
//		if v.SeedD == gl.SeedD {
//			return queue
//		}
//	}
//	return append(queue, gl)
//}
//
//func sPeek(queue []*groupLife) *groupLife {
//	if len(queue) == 0 {
//		return nil
//	}
//	return queue[0]
//}
//
//func peek(queue []*groupLife) *groupLife {
//	if len(queue) == 0 {
//		return nil
//	}
//	return queue[len(queue)-1]
//}
//
//func clone(queue []*groupLife) []*groupLife {
//	tmp := make([]*groupLife, len(queue))
//	copy(tmp, queue)
//	return tmp
//}
