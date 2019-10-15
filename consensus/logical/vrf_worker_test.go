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
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/common/ed25519"
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

func TestVRFDistribute(t *testing.T) {
	stakeArr := [...]uint64{
		500,500,500,500,500,500,500,500,500,500,500,500,500,500,500,500,501,501,501,501,501,501,501,501,501,501,501,501,
		576,600,682,709,1649,2251,2413,2768,3681,4200,8094,14335,34488,160300,185174,264371,584615,1830082,2499958,
		2499958,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,2500000,
		2500000,2500000,2500000,2500000,2500000}

	var totalStake uint64 = 0
	nNodes := len(stakeArr)
	skArr := make([]ed25519.PrivateKey, nNodes)
	pkArr := make([]ed25519.PublicKey, nNodes)
	for i := 0; i < nNodes; i++ {
		pk, sk, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		skArr[i] = sk
		pkArr[i] = pk
		totalStake += stakeArr[i]
	}

	nCount := 1000
	nMiss := 0
	temp := new(big.Int)
	temp.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	maxValue := new(big.Rat).SetInt(temp)

	var msg [32]byte

	vs := vrfThreshold(1, 136)

	for c := 0; c < nCount; c++ {
		breakFlag := false
		io.ReadFull(rand.Reader, msg[:])
		for i := 0; i < nNodes; i++ {
			pi, err := ed25519.ECVRFProve(pkArr[i], skArr[i], msg[:])
			if err != nil {
				t.Fatal(err)
			}
			value := ed25519.ECVRFProof2hash(pi)
			br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
			pr := br.Quo(br, maxValue)
			//vs := vrfThreshold(stakeArr[i], totalStake)

			if pr.Cmp(vs) < 0 {
				breakFlag = true
				break
			}
		}
		if breakFlag == false {
			nMiss++
		}
	}
	fmt.Printf("nodes:%v, totalStake:%v , try times: %v, miss times: %v", nNodes, totalStake, nCount, nMiss)
}
