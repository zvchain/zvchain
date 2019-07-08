/*
@Time : 2019-07-08
@Author : fenglei0814@gmail.com
*/
package group

import (
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
}

func newGroupLife(group types.GroupI) *groupLife {
	return &groupLife{group.Header().Seed(), group.Header().WorkHeight(), group.Header().DismissHeight()}
}

type pool struct {
	active  []*groupLife
	waiting []*groupLife
}

func newPool() *pool {
	return &pool{
		active:  make([]*groupLife, 0),
		waiting: make([]*groupLife, 0),
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
	life := newGroupLife(group)
	lifeData, err := msgpack.Marshal(life)
	if err != nil {
		return err
	}
	seed := group.Header().Seed().Bytes()
	p.waiting = append(p.waiting, life)
	db.SetData(common.GroupWaitingAddress, seed, lifeData)

	return nil
}


func (p *pool) adjust(db *account.AccountDB, height uint64) error {
	// move group from waiting to active
	snapshot := db.Snapshot()
	peeked := sPeek(p.waiting)
	//todo: check the bound height
	for peeked != nil && peeked.begin <= height {
		p.waiting = shift(p.waiting)
		err := p.toActive(db, peeked)
		if err != nil {
			db.RevertToSnapshot(snapshot)
			return err
		}
		peeked = sPeek(p.waiting)
	}
	// move group from active to dismiss
	peeked = sPeek(p.active)
	for peeked != nil && peeked.end >= height {
		p.active = shift(p.active)
		p.toDismiss(db, peeked)
		peeked = sPeek(p.active)
	}
	return nil
}

func (p *pool) toActive(db *account.AccountDB, gl *groupLife) error {
	byteData, err := msgpack.Marshal(gl)
	if err != nil {
		return err
	}

	db.RemoveData(common.GroupWaitingAddress, gl.seed.Bytes())
	db.SetData(common.GroupActiveAddress, gl.seed.Bytes(), byteData)

	return nil
}

// move the group to dismiss db
func (p *pool) toDismiss(db *account.AccountDB, gl *groupLife) {
	db.RemoveData(common.GroupActiveAddress, gl.seed.Bytes())
	db.SetData(common.GroupDismissAddress, gl.seed.Bytes(), []byte{1})
}

func shift(queue []*groupLife) []*groupLife {
	// this case should never happen, we already use the sPeek to check if len is 0
	if len(queue) == 0 {
		return nil
	}
	return queue[1:]
}

func sPeek(queue []*groupLife) *groupLife {
	if len(queue) == 0 {
		return nil
	}
	return queue[0]
}
