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
	"sort"

	lru "github.com/hashicorp/golang-lru"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)

type groupLife struct {
	SeedD  common.Hash
	Begin  uint64
	End    uint64
	Height uint64 // Height of group created
}

func (gl *groupLife) Seed() common.Hash {
	return gl.SeedD
}

func newGroupLife(group group) *groupLife {
	return &groupLife{group.Header().Seed(), group.Header().WorkHeight(), group.Header().DismissHeight(), group.Height}
}

type pool struct {
	activeList       []*groupLife // list of active groups
	waitingList      []*groupLife // list of waiting groups
	groupCache       *lru.Cache   // cache for groups. key is types.Seedi; value is types.Groupi
	activeListCache  *lru.Cache   // cache for active group lists. key is Height; value is []*groupLife
	waitingListCache *lru.Cache   // cache for waiting group lists. key is Height; value is []*groupLife
	topGroup         *group
}

func newPool() *pool {
	return &pool{
		activeList:       make([]*groupLife, 0),
		waitingList:      make([]*groupLife, 0),
		groupCache:       common.MustNewLRUCache(500),
		activeListCache:  common.MustNewLRUCache(500),
		waitingListCache: common.MustNewLRUCache(500),
	}
}

func (p *pool) initPool(db types.AccountDB) error {
	iter := db.DataIterator(common.GroupActiveAddress, []byte{})
	if iter != nil {
		for iter.Next() {
			var life groupLife
			err := msgpack.Unmarshal(iter.Value, &life)
			if err != nil {
				return err
			}
			p.activeList = append(p.activeList, &life)
		}
	}
	sort.SliceStable(p.activeList, func(i, j int) bool {
		return p.activeList[i].End < p.activeList[j].End
	})

	iter = db.DataIterator(common.GroupWaitingAddress, []byte{})
	if iter != nil {
		for iter.Next() {
			var life groupLife
			err := msgpack.Unmarshal(iter.Value, &life)
			if err != nil {
				return err
			}
			p.waitingList = append(p.waitingList, &life)
		}
	}

	sort.SliceStable(p.waitingList, func(i, j int) bool {
		return p.waitingList[i].Begin < p.waitingList[j].Begin
	})

	return nil
}

func (p *pool) initGenesis(db types.AccountDB, genesis *types.GenesisInfo) error {
	exist := p.get(db, genesis.Group.Header().Seed())
	if exist == nil {
		g := newGroup(genesis.Group, 0)
		err := p.add(db, g)
		if err != nil {
			return err
		}
		p.adjust(db, 0)
		p.activeListCache.Add(uint64(0), clone(p.activeList))
	}
	return nil
}

func (p *pool) add(db types.AccountDB, group *group) error {
	byteData, err := msgpack.Marshal(group)
	if err != nil {
		return err
	}

	life := newGroupLife(*group)
	lifeData, err := msgpack.Marshal(life)
	if err != nil {
		return err
	}
	seed := group.Header().Seed().Bytes()
	p.waitingList = append(p.waitingList, life)
	db.SetData(common.GroupWaitingAddress, seed, lifeData)

	p.groupCache.Add(group.Header().Seed(), group)
	db.SetData(common.HashToAddress(group.Header().Seed()), groupDataKey, byteData)
	p.topGroup = group
	return nil
}

func (p *pool) resetToTop(db types.AccountDB, height uint64) {
	removed := make([]*groupLife, 0)
	// remove group from waitingList
	peeked := peek(p.waitingList)
	for peeked != nil && peeked.Height >= height {
		removed = append(removed, peeked)
		p.waitingList = removeLast(p.waitingList)
		peeked = peek(p.waitingList)
	}

	// remove group from activeGroup
	if len(p.waitingList) == 0 {
		peeked = peek(p.activeList)
		for peeked != nil && peeked.Height >= height {
			removed = append(removed, peeked)
			p.activeList = removeLast(p.activeList)
			peeked = peek(p.waitingList)
		}
	}

	// remove from groupCache
	for _, v := range removed {
		p.groupCache.Remove(v.Seed())
		p.activeListCache.Remove(v.Height)
		p.waitingListCache.Remove(v.Height)
	}
	p.topGroup = nil
}

func (p *pool) minerLiveGroupCount(chain chainReader, addr common.Address, height uint64) int {
	waiting := p.getWaiting(chain, height)
	active := p.getActives(chain, height)

	lived := append(waiting, active...)
	count := 0
	for _, v := range lived {
		g := p.get(chain.LatestStateDB(), v.Seed())
		if g != nil {
			for _, mem := range g.Members() {
				if bytes.Equal(addr.Bytes(), mem.ID()) {
					count++
				}
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

func (p *pool) adjust(db types.AccountDB, height uint64) {
	// move group from waitingList to activeList
	peeked := sPeek(p.waitingList)
	for peeked != nil && peeked.Begin >= height {
		p.waitingList = removeFirst(p.waitingList)
		p.toActive(db, peeked)
		peeked = sPeek(p.waitingList)
	}

	// move group from activeList to dismiss
	peeked = sPeek(p.activeList)
	for peeked != nil && peeked.End <= height {
		p.activeList = removeFirst(p.activeList)
		p.toDismiss(db, peeked)
		peeked = sPeek(p.activeList)
	}

	p.waitingListCache.Add(height, p.waitingList)
	p.activeListCache.Add(height, clone(p.activeList))
}

func (p *pool) toActive(db types.AccountDB, gl *groupLife) {
	byteData, err := msgpack.Marshal(gl)
	if err != nil {
		// this case must not happen
		panic("failed to marshal group life data")
	}
	p.activeList = push(p.activeList, gl)
	db.RemoveData(common.GroupWaitingAddress, gl.SeedD.Bytes())
	db.SetData(common.GroupActiveAddress, gl.SeedD.Bytes(), byteData)
}

func (p *pool) getActives(chain chainReader, height uint64) []*groupLife {
	if g, ok := p.activeListCache.Get(height); ok {
		gl := g.([]*groupLife)
		return gl
	}
	db, err := chain.GetAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("GetAccountDBByHeight error:%v, Height:%v", err, height)
		return nil
	}
	iter := db.DataIterator(common.GroupActiveAddress, []byte{})
	if iter == nil {
		return nil
	}
	rs := make([]*groupLife, 0)
	for iter.Next() {
		var life groupLife
		err := msgpack.Unmarshal(iter.Value, &life)
		if err != nil {
			logger.Errorf("GetAccountDBByHeight error:%v, Height:%v", err, height)
			return nil
		}
		rs = append(rs, &life)
	}
	p.activeListCache.ContainsOrAdd(height, rs)
	return rs
}

func (p *pool) getWaiting(chain chainReader, height uint64) []*groupLife {
	if g, ok := p.waitingListCache.Get(height); ok {
		gl := g.([]*groupLife)
		return gl
	}
	db, err := chain.GetAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("GetAccountDBByHeight error:%v, Height:%v", err, height)
		return nil
	}
	iter := db.DataIterator(common.GroupWaitingAddress, []byte{})
	if iter == nil {
		return nil
	}
	rs := make([]*groupLife, 0)
	for iter.Next() {
		var life groupLife
		err := msgpack.Unmarshal(iter.Value, &life)
		if err != nil {
			logger.Errorf("GetAccountDBByHeight error:%v, Height:%v", err, height)
			return nil
		}
		rs = append(rs, &life)
	}
	p.waitingListCache.ContainsOrAdd(height, rs)
	return rs
}


func (p *pool) groupsAfter(chain chainReader, height uint64, limit int) []types.GroupI {
	//TODO: optimize it
	rs := make([]types.GroupI,0,limit)
	for _, v := range p.waitingList {
		rs = append(rs, p.get(chain.LatestStateDB(),v.Seed()))
		if len(rs) >= limit {
			return rs
		}
	}
	for _, v := range p.activeList {
		rs = append(rs, p.get(chain.LatestStateDB(),v.Seed()))
		if len(rs) >= limit {
			return rs
		}
	}

	return rs
}

// move the group to dismiss db
func (p *pool) toDismiss(db types.AccountDB, gl *groupLife) {
	db.RemoveData(common.GroupActiveAddress, gl.SeedD.Bytes())
	db.SetData(common.GroupDismissAddress, gl.SeedD.Bytes(), []byte{1})
}

func (p *pool) count(db types.AccountDB) uint64 {
	rs := len(p.waitingList) + len(p.activeList)
	iter := db.DataIterator(common.GroupDismissAddress, []byte{})
	if iter != nil {
		for iter.Next() {
			rs++
		}
	}
	return uint64(rs)
}

func removeFirst(queue []*groupLife) []*groupLife {
	// this case should never happen, we already use the sPeek to check if len is 0
	if len(queue) == 0 {
		return nil
	}
	return queue[1:]
}

func removeLast(queue []*groupLife) []*groupLife {
	// this case should never happen, we already use the peek to check if len is 0
	if len(queue) == 0 {
		return nil
	}
	return queue[:len(queue)-1]

}

func push(queue []*groupLife, gl *groupLife) []*groupLife {
	return append(queue, gl)
}

func sPeek(queue []*groupLife) *groupLife {
	if len(queue) == 0 {
		return nil
	}
	return queue[0]
}

func peek(queue []*groupLife) *groupLife {
	if len(queue) == 0 {
		return nil
	}
	return queue[len(queue)-1]
}

func clone(queue []*groupLife) []*groupLife {
	tmp := make([]*groupLife, len(queue))
	copy(tmp, queue)
	return tmp
}
