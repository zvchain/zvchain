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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
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
		t.Log(m.ID.GetAddrString(), m.Stake)
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
			if v, ok := selectedMap[m.ID.GetAddrString()]; ok {
				selectedMap[m.ID.GetAddrString()] = v + 1
			} else {
				selectedMap[m.ID.GetAddrString()] = 1
			}
		}
	}

	for _, mem := range cands {
		selected := selectedMap[mem.ID.GetAddrString()]
		t.Log(mem.ID.GetAddrString(), float64(mem.Stake)/float64(totalStake), float64(selected)/float64(testCount))
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

func getMinerInfos(file string) []*model.MinerDO {
	bs, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(bs), "\r\n")
	miners := make([]*model.MinerDO, 0)
	for i, line := range lines {
		if i == 0 {
			continue
		}
		arry := strings.Split(line, ",")
		if len(arry) < 3 {
			continue
		}
		idBytes := common.StringToAddress(strings.TrimSpace(arry[1]))
		stake, err := strconv.ParseUint(arry[2], 10, 64)
		if err != nil {
			panic(err)
		}
		status, err := strconv.ParseInt(arry[3], 10, 32)
		if err != nil {
			panic(err)
		}
		var ntype types.MinerType
		if arry[0] == "outer" {
			ntype = types.MinerTypeProposal
		} else if arry[0] == "internal" {
			ntype = types.MinerTypeVerify
		}
		miner := &model.MinerDO{
			ID:     groupsig.DeserializeID(idBytes.Bytes()),
			Stake:  stake,
			NType:  ntype,
			Status: types.MinerStatus(status),
		}
		if miner.IsActive() {
			miners = append(miners, miner)
		}
	}
	return miners
}

func genInternalMiners(num int, stake uint64) []*model.MinerDO {
	miners := make([]*model.MinerDO, 0)
	for i := 0; i < num; i++ {
		miner := &model.MinerDO{
			ID:     groupsig.DeserializeID(common.BigToAddress(new(big.Int).SetUint64(uint64(i))).Bytes()),
			Stake:  stake,
			NType:  types.MinerTypeVerify,
			Status: types.MinerStatusActive,
		}
		miners = append(miners, miner)
	}
	return miners
}

func TestMockSelect(t *testing.T) {
	miners := getMinerInfos("/Users/pxf/Downloads/verify_info.csv")
	outStake, interStake := uint64(0), uint64(0)

	for _, miner := range miners {
		if miner.IsProposal() {
			outStake += miner.Stake
		} else if miner.IsVerifier() {
			interStake += miner.Stake
		}
	}

	log.Println("outStake", outStake, "internalSstake", interStake, "totalStake", outStake+interStake, "internalRate", float64(interStake)/float64(interStake+outStake))
	groups := make([][]*model.MinerDO, 0)

	num := 120
	for i := 0; i < num; i++ {
		r := make([]byte, 32)
		rand.Read(r)
		selector := newCandidateSelector(miners, r)
		miners := selector.fts(100)
		groups = append(groups, miners)
	}
	internalSelected := 0
	for _, g := range groups {
		for _, m := range g {
			if m.IsVerifier() {
				internalSelected++
			}
		}
	}

	log.Println("selectedRate", float64(internalSelected)/float64(num*100))
}

type groupTmp struct {
	members       []*model.MinerDO
	beginHeight   uint64
	dismissHeight uint64
}

func (g *groupTmp) dismissed(h uint64) bool {
	return g.dismissHeight <= h
}

func adjustNode(nodeNum int, stake uint64) bool {
	miners := getMinerInfos("/Users/pxf/Downloads/verify_info.csv")
	incrMiners := genInternalMiners(nodeNum, stake)
	miners = append(miners, incrMiners...)
	for _, m := range miners {
		if m.IsVerifier() {
			m.Stake = stake
		}
	}

	outStake, interStake := uint64(0), uint64(0)
	outNode, interNode := 0, 0
	for _, miner := range miners {
		if miner.IsProposal() {
			outStake += miner.Stake
			outNode++
		} else if miner.IsVerifier() {
			interStake += miner.Stake
			interNode++
		}
	}

	num := 240

	for joinGroupPerNode := 5; joinGroupPerNode <= 5; joinGroupPerNode++ {

		groups := make([]*groupTmp, 0)
		for ep := types.EpochAt(0); len(groups) < num; ep = ep.Next() {

			joinedGroupMap := make(map[string]int)
			for _, g := range groups {
				if g.dismissed(ep.Start()) {
					continue
				}
				for _, mem := range g.members {
					if v, ok := joinedGroupMap[mem.ID.GetAddrString()]; ok {
						joinedGroupMap[mem.ID.GetAddrString()] = v + 1
					} else {
						joinedGroupMap[mem.ID.GetAddrString()] = 1
					}
				}
			}

			avaliMiners := make([]*model.MinerDO, 0)
			for _, m := range miners {
				if v, ok := joinedGroupMap[m.ID.GetAddrString()]; !ok || v < joinGroupPerNode {
					avaliMiners = append(avaliMiners, m)
				}
			}
			if !candidateEnough(len(avaliMiners)) {
				continue
			}

			r := make([]byte, 32)
			rand.Read(r)
			selector := newCandidateSelector(avaliMiners, r)
			miners := selector.fts(100)
			g := &groupTmp{
				members:       miners,
				beginHeight:   types.ActivateEpochOfGroupsCreatedAt(ep.Start()).Start(),
				dismissHeight: types.DismissEpochOfGroupsCreatedAt(ep.Start()).Start(),
			}
			groups = append(groups, g)
		}
		internalSelected := 0
		for _, g := range groups {
			for _, m := range g.members {
				if m.IsVerifier() {
					internalSelected++
				}
			}
		}
		r := float64(internalSelected) / float64(num*100) / (float64(interStake) / float64(interStake+outStake))
		fmt.Println(fmt.Sprintf("内部节点数 %v(新增%v), 内部每节点质押 %v, 内部质押占比 %.2f, 实际进组占比 %.2f, 理论与实际比 %.2f", interNode, nodeNum, stake, float64(interStake)/float64(interStake+outStake), float64(internalSelected)/float64(num*100), r))
		if float64(internalSelected)/float64(num*100) > 0.51 {
			return true
		}
	}
	return false
}

func TestMockGroupBuild(t *testing.T) {
	model.Param.GroupMemberMax = 100
	model.Param.GroupMemberMin = 80

	//for stake := uint64(50000); stake <= 50000; stake += 5000 {
	//	for nodeNum := 80; nodeNum < 1000; nodeNum += 40 {
	//		adjustNode(nodeNum, stake)
	//	}
	//}
	for stake := uint64(50000); stake <= 90000; stake += 10000 {
		for nodeNum := 60; nodeNum < 480; nodeNum += 20 {
			if adjustNode(nodeNum, stake) {
				break
			}
		}
	}
}
