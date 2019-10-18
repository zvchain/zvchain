//   Copyright (C) 2018 ZVChain
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

package logical

import (
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

func genMinerDO() *model.SelfMinerDO {
	addr := "0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a"
	var id groupsig.ID
	id.SetAddrString(addr)
	minerDO := model.MinerDO{
		ID:    id,
		VrfPK: base.Hex2VRFPublicKey("0x666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"),
		Stake: 100,
	}
	miner := &model.SelfMinerDO{
		MinerDO: minerDO,
		VrfSK:   base.Hex2VRFPrivateKey("0x7e7707df15aa16256d0c18e9ddd59b36d48759ec5b6404cfb6beceea9a798879666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"),
	}
	return miner
}

func genBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		Height:  100,
		Nonce:   2,
		Random:  common.FromHex("0x194b3d24ddb883a1fd7d3b1e0038ebf9bb739759719eb1093f40e489fdacf6c200"),
		TotalQN: 1,
	}
}

func genBlockHeader2() *types.BlockHeader {
	return &types.BlockHeader{
		ProveValue: common.FromHex("0x03556a119b69e52a6c8f676213e2184c588bc9731ec0ab1ed32a91a9a22155cdeb001fa9a2fd33c8660483f267050f0e72072658f16d485a1586fca736a50a423cbbb181870219af0c2c4fdbbb89832730"),
		Height:     101,
		TotalQN:    3,
	}
}

func init() {
	model.Param.MaxQN = 5
	model.Param.PotentialProposal = 5
}

func TestProve(t *testing.T) {

	worker := &vrfWorker{
		miner:      genMinerDO(),
		baseBH:     genBlockHeader(),
		castHeight: 101,
	}

	pi, qn, err := worker.Prove(300)
	t.Log(common.ToHex(pi), qn)
	if err != nil {
		t.Fatal(err)
	}
	if len(pi) != 81 {
		t.Errorf("size of prove error")
	}
	if qn > uint64(model.Param.MaxQN) {
		t.Errorf("qn error")
	}
}

func TestVrfVerifyBlock(t *testing.T) {
	bh := genBlockHeader2()
	miner := genMinerDO().MinerDO
	ok, err := vrfVerifyBlock(bh, genBlockHeader(), &miner, 300)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("vrf verify block not ok")
	}
}

func genRandomBlockHeader() *types.BlockHeader {
	b := make([]byte, 32)
	rand.Read(b)
	return &types.BlockHeader{
		Height:  rand.Uint64(),
		Nonce:   2,
		Random:  b,
		TotalQN: 1,
	}
}

func generateMiners(stakes []uint64) []*model.SelfMinerDO {
	miners := make([]*model.SelfMinerDO, 0)
	for i := 0; i < len(stakes); i++ {
		pk, sk, _ := base.VRFGenerateKey(nil)

		var id groupsig.ID
		id.SetBigInt(new(big.Int).SetInt64(int64(i)))
		m := model.MinerDO{
			ID:    id,
			VrfPK: pk,
			Stake: stakes[i],
		}
		sm := &model.SelfMinerDO{
			MinerDO: m,
			VrfSK:   sk,
		}
		miners = append(miners, sm)
	}
	return miners
}

func generateStakes() []uint64 {
	stakes := make([]uint64, 0)
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 10000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 50000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 200000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 500000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 1000000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 1500000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 1800000)
	}
	for i := 0; i < 4; i++ {
		stakes = append(stakes, 2100000)
	}
	for i := 0; i < 90; i++ {
		stakes = append(stakes, 2500000)
	}
	return stakes
}
func totalStake(stakes []uint64) uint64 {
	v := uint64(0)
	for _, s := range stakes {
		v += s
	}
	return v
}

type blockWeight2 struct {
	qn    uint64
	pv    base.VRFProve
	stake uint64
}

func (bw *blockWeight2) moreWeight(bw2 *blockWeight2) bool {
	if bw.qn > bw2.qn {
		return true
	} else if bw.qn < bw2.qn {
		return false
	}
	p := new(big.Rat).Quo(new(big.Rat).SetInt(base.VRFProof2hash(base.VRFProve(bw.pv)).Big()), new(big.Rat).SetInt64(int64(bw.stake)))
	p2 := new(big.Rat).Quo(new(big.Rat).SetInt(base.VRFProof2hash(base.VRFProve(bw2.pv)).Big()), new(big.Rat).SetInt64(int64(bw2.stake)))
	return p.Cmp(p2) > 0
}

type minerStat struct {
	id            groupsig.ID
	qualification int
	win           int
	win2          int
	stake         uint64
}

func testFun(miners []*model.SelfMinerDO, testNum int, totalStake uint64) []*minerStat {
	qualifiedStat := make(map[string]int)
	winStat := make(map[string]int)
	winStat2 := make(map[string]int)
	for i := 0; i < testNum; i++ {
		bh := genRandomBlockHeader()
		var (
			maxWeight         *types.BlockWeight
			maxWeight2        *blockWeight2
			maxWeightMinerId  groupsig.ID
			maxWeightMinerId2 groupsig.ID
		)
		for _, miner := range miners {
			w := newVRFWorker(miner, bh, bh.Height+1, 0, nil)
			pi, qn, err := w.Prove(totalStake)
			if err == nil {
				if c, ok := qualifiedStat[miner.ID.GetAddrString()]; ok {
					qualifiedStat[miner.ID.GetAddrString()] = c + 1
				} else {
					qualifiedStat[miner.ID.GetAddrString()] = 1
				}
				wb := types.NewBlockWeight(&types.BlockHeader{TotalQN: qn, ProveValue: pi})
				if maxWeight == nil || wb.MoreWeight(maxWeight) {
					maxWeight = wb
					maxWeightMinerId = miner.ID
				}
				wb2 := &blockWeight2{qn: qn, pv: pi, stake: miner.Stake}
				if maxWeight2 == nil || wb2.moreWeight(maxWeight2) {
					maxWeight2 = wb2
					maxWeightMinerId2 = miner.ID
				}
			}
		}
		if maxWeight == nil {
			continue
		}
		if c, ok := winStat[maxWeightMinerId.GetAddrString()]; ok {
			winStat[maxWeightMinerId.GetAddrString()] = c + 1
		} else {
			winStat[maxWeightMinerId.GetAddrString()] = 1
		}
		if c, ok := winStat2[maxWeightMinerId2.GetAddrString()]; ok {
			winStat2[maxWeightMinerId2.GetAddrString()] = c + 1
		} else {
			winStat2[maxWeightMinerId2.GetAddrString()] = 1
		}
	}
	ms := make([]*minerStat, 0)
	for _, miner := range miners {
		q := qualifiedStat[miner.ID.GetAddrString()]
		w := winStat[miner.ID.GetAddrString()]
		w2 := winStat2[miner.ID.GetAddrString()]
		m := &minerStat{
			id:            miner.ID,
			qualification: q,
			win:           w,
			win2:          w2,
		}
		ms = append(ms, m)
	}
	return ms
}

func TestProposalQualification(t *testing.T) {
	types.DefaultPVFunc = func(pvBytes []byte) *big.Int {
		return base.VRFProof2hash(base.VRFProve(pvBytes)).Big()
	}
	stakes := generateStakes()
	miners := generateMiners(stakes)
	testCnt := 100
	parallel := runtime.NumCPU()
	testCntPerCpu := testCnt / parallel

	overMs := make([]*minerStat, len(miners))
	for i, ms := range miners {
		overMs[i] = &minerStat{id: ms.ID, stake: ms.Stake}
	}
	mu := sync.Mutex{}

	wg := &sync.WaitGroup{}
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ms := testFun(miners, testCntPerCpu, totalStake(stakes))
			mu.Lock()
			for j, m := range ms {
				if m.id.GetAddrString() != overMs[j].id.GetAddrString() {
					t.Fatalf("id not equal")
				}
				overMs[j].qualification += m.qualification
				overMs[j].win += m.win
				overMs[j].win2 += m.win2
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	rate := func(v int, stake uint64) float64 {
		return float64(v) * 1000 / float64(stake)
	}

	rateString := func(v int, stake uint64) string {
		return fmt.Sprintf("%.3f", rate(v, stake))
	}
	var (
		lastStake uint64
		winRate   float64
		winRate2  float64
		cnt       int
	)
	for i, ms := range overMs {
		if lastStake == ms.stake || lastStake == 0 {
			lastStake = ms.stake
			winRate += rate(ms.win, ms.stake)
			winRate2 += rate(ms.win2, ms.stake)
			cnt++
		} else {
			t.Logf("avg:\t%v %v\t%.3f\t%.3f", lastStake, cnt, winRate/float64(cnt), winRate2/float64(cnt))
			lastStake = ms.stake
			winRate = rate(ms.win, ms.stake)
			winRate2 = rate(ms.win2, ms.stake)
			cnt = 1
		}
		if i == len(overMs)-1 {
			t.Logf("avg:\t%v %v\t%.3f\t%.3f", lastStake, cnt, winRate/float64(cnt), winRate2/float64(cnt))
		}
		fmt.Printf("%v\t\t q:%v %v\t\t\t w:%v %v\t\t\t w2:%v %v\n", ms.stake, ms.qualification, rateString(ms.qualification, ms.stake), ms.win, rateString(ms.win, ms.stake), ms.win2, rateString(ms.win2, ms.stake))
	}
}
