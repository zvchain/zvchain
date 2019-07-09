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
	"sort"

	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
)

type groupLife struct {
	seed  common.Hash
	begin uint64
	end   uint64
	height uint64	// height of group created

}

func newGroupLife(g types.GroupI) *groupLife {
	group := g.(*Group)
	return &groupLife{group.Header().Seed(), group.Header().WorkHeight(), group.Header().DismissHeight(),group.height}
}

type pool struct {
	active  []*groupLife
	waiting []*groupLife
	cache   *lru.Cache
}

func newPool() *pool {
	return &pool{
		active:  make([]*groupLife, 0),
		waiting: make([]*groupLife, 0),
		cache:    common.MustNewLRUCache(500),
	}
}

func (p *pool) initPool(db *account.AccountDB) error {
	iter := db.DataIterator(common.GroupActiveAddress, []byte{})
	if iter != nil {
		for iter.Next() {
			var life groupLife
			err := msgpack.Unmarshal(iter.Value, &life)
			if err != nil {
				return err
			}
			p.active = append(p.active, &life)
		}
	}
	sort.SliceStable(p.active, func(i, j int) bool {
		return p.active[i].end < p.active[j].end
	})

	iter = db.DataIterator(common.GroupWaitingAddress, []byte{})
	if iter != nil {
		for iter.Next() {
			var life groupLife
			err := msgpack.Unmarshal(iter.Value, &life)
			if err != nil {
				return err
			}
			p.waiting = append(p.waiting, &life)
		}
	}

	sort.SliceStable(p.waiting, func(i, j int) bool {
		return p.waiting[i].begin < p.waiting[j].begin
	})

	return nil
}

func (p *pool) add(db *account.AccountDB, group types.GroupI) error {
	byteData, err := msgpack.Marshal(group)
	if err != nil {
		return err
	}

	byteHeader, err := msgpack.Marshal(group.Header().(*GroupHeader))
	if err != nil {
		return err
	}

	life := newGroupLife(group)
	lifeData, err := msgpack.Marshal(life)
	if err != nil {
		return err
	}
	seed := group.Header().Seed().Bytes()
	p.waiting = append(p.waiting, life)
	db.SetData(common.GroupWaitingAddress, seed, lifeData)

	p.cache.Add(group.Header().Seed(),group)
	db.SetData(common.HashToAddress(group.Header().Seed()), groupDataKey, byteData)
	db.SetData(common.HashToAddress(group.Header().Seed()), groupHeaderKey, byteHeader)
	return nil
}

func (p *pool) resetToTop(db *account.AccountDB, height uint64) {
	removed := make([]common.Hash,0)
	// move group from waiting to active
	peeked := peek(p.waiting)
	for peeked != nil && peeked.height >= height {
		removed = append(removed, peeked.seed)
		p.waiting = removeLast(p.waiting)
		peeked = peek(p.waiting)
	}

	// check the active group only if all waiting groups removed
	if len(p.waiting) == 0 {
		peeked = peek(p.active)
		for peeked != nil && peeked.height >= height {
			removed = append(removed, peeked.seed)
			p.waiting = removeLast(p.waiting)
			peeked = peek(p.waiting)
		}
	}

	// remove from cache
	for _, v := range removed {
		p.cache.Remove(v)
	}
}


func (p *pool) adjust(db *account.AccountDB, height uint64) error {
	// move group from waiting to active
	peeked := sPeek(p.waiting)
	for peeked != nil && peeked.begin >= height {
		p.waiting = removeFirst(p.waiting)
		p.toActive(db, peeked)
		peeked = sPeek(p.waiting)
	}

	// move group from active to dismiss
	peeked = sPeek(p.active)
	for peeked != nil && peeked.end >= height {
		p.active = removeFirst(p.active)
		p.toDismiss(db, peeked)
		peeked = sPeek(p.active)
	}
	return nil
}

func (p *pool) toActive(db *account.AccountDB, gl *groupLife) {
	byteData, err := msgpack.Marshal(gl)
	if err != nil {
		// this case must not happen
		panic("failed to marshal group life data")
	}
	push(p.active,gl)
	db.RemoveData(common.GroupWaitingAddress, gl.seed.Bytes())
	db.SetData(common.GroupActiveAddress, gl.seed.Bytes(), byteData)

}

// move the group to dismiss db
func (p *pool) toDismiss(db *account.AccountDB, gl *groupLife) {
	db.RemoveData(common.GroupActiveAddress, gl.seed.Bytes())
	db.SetData(common.GroupDismissAddress, gl.seed.Bytes(), []byte{1})
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
	return append(queue,gl)
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

