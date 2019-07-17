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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"math/rand"
	"testing"
)

func genRandomMiners(n int) []*model.MinerDO {
	miners := make([]*model.MinerDO, 0)
	for len(miners) < n {
		miner := &model.MinerDO{
			ID:    groupsig.DeserializeID(common.Int32ToByte(int32(len(miners)))),
			Stake: uint64(rand.Int31n(1000000)),
		}
		miners = append(miners, miner)
	}
	return miners
}

func TestFts(t *testing.T) {
	rand := common.FromHex("0x1237")
	cands := genRandomMiners(100)

	selector := newCandidateSelector(cands, rand)
	selecteds := selector.fts(20)

	for _, m := range selecteds {
		t.Log(m.ID.GetHexString(), m.Stake)
	}
}

func TestFts_Distribution(t *testing.T) {
	cands := genRandomMiners(150)
	totalStake := uint64(0)
	for _, m := range cands {
		totalStake += m.Stake
	}
	selectedMap := make(map[string]int)
	testCount := 10000

	for i := 0; i < testCount; i++ {
		rand := common.Int32ToByte(int32(i))
		selector := newCandidateSelector(cands, rand)
		seleted := selector.fts(40)
		for _, m := range seleted {
			if v, ok := selectedMap[m.ID.GetHexString()]; ok {
				selectedMap[m.ID.GetHexString()] = v + 1
			} else {
				selectedMap[m.ID.GetHexString()] = 1
			}
		}
	}

	for _, mem := range cands {
		selected := selectedMap[mem.ID.GetHexString()]
		t.Log(mem.ID.GetHexString(), float64(mem.Stake)/float64(totalStake), float64(selected)/float64(testCount))
	}
}

func BenchmarkFts(b *testing.B) {
	cands := genRandomMiners(10000)
	for i := 0; i < b.N; i++ {
		rand := common.FromHex("0x1237")
		selector := newCandidateSelector(cands, rand)
		selector.fts(100)
	}
}
