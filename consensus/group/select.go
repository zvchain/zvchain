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
	"container/list"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/model"
)

type candidateSelector struct {
	list        *list.List
	remainStake uint64
	rand        []byte
}

func newCandidateSelector(cands []*model.MinerDO, rand []byte) *candidateSelector {
	list := list.New()
	stake := uint64(0)
	for _, c := range cands {
		list.PushBack(c)
		stake += c.Stake
	}
	return &candidateSelector{list: list, remainStake: stake, rand: rand}
}

func (cs *candidateSelector) algSatoshi(num int) []*model.MinerDO {
	rand := base.RandFromBytes(cs.rand)
	result := make([]*model.MinerDO, 0)
	for len(result) < num {
		r := rand.Deri(len(result)).ModuloUint64(cs.remainStake)
		cumulativeStake := uint64(0)
		for e := cs.list.Front(); e != nil; e = e.Next() {
			m := e.Value.(*model.MinerDO)
			if m.Stake+cumulativeStake > r {
				cs.list.Remove(e)
				cs.remainStake -= m.Stake
				result = append(result, m)
				break
			}
			cumulativeStake += m.Stake
		}
	}
	return result
}
