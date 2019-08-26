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
	"encoding/json"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"io/ioutil"
	"strings"
	"testing"
)

func createMinerDOs(keyFile string) []*model.SelfMinerDO {
	bs, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(err)
	}
	keys := strings.Split(string(bs), "\n")
	dos := make([]*model.SelfMinerDO, 0)
	for _, priKey := range keys {
		privateKey := common.HexToSecKey(priKey)
		miner, _ := model.NewSelfMinerDO(privateKey)
		dos = append(dos, &miner)
	}
	return dos
}

func newCandidates(miners []*model.SelfMinerDO) candidates {
	cands := make(candidates, 0)
	for _, miner := range miners {
		cands = append(cands, &miner.MinerDO)
	}
	return cands
}

type sharePiece struct {
	pieces []groupsig.Seckey
	pubkey groupsig.Pubkey
}

func generateSharePiece(miner *model.SelfMinerDO, cands candidates, seed common.Hash) *sharePiece {
	rand := miner.GenSecretForGroup(seed)
	secs := make([]groupsig.Seckey, cands.threshold())
	for i := 0; i < len(secs); i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(rand.Deri(i))
	}

	sec0 := *groupsig.NewSeckeyFromRand(rand.Deri(0))
	pk := *groupsig.NewPubkeyFromSeckey(sec0)

	pieces := make([]groupsig.Seckey, 0)
	for _, mem := range cands {
		pieces = append(pieces, *groupsig.ShareSeckey(secs, mem.ID))
	}
	return &sharePiece{
		pieces: pieces,
		pubkey: pk,
	}
}

func TestGenerateGenesisGroup(t *testing.T) {
	keyFile := "key_file_test"
	seed := common.BytesToHash([]byte("zvchain genesis group"))
	miners := createMinerDOs(keyFile)
	if len(miners) == 0 {
		t.Errorf("no miners")
	}
	candidates := newCandidates(miners)

	// Generate all share pieces
	pieces := make([]*sharePiece, candidates.size())
	for i, miner := range miners {
		pieces[i] = generateSharePiece(miner, candidates, seed)
	}

	pk0s := make([]groupsig.Pubkey, candidates.size())
	for i, piece := range pieces {
		pk0s[i] = piece.pubkey
	}
	// Aggregates the group pubkey
	gpk := groupsig.AggregatePubkeys(pk0s)

	minerSks := make([]groupsig.Seckey, candidates.size())
	minerPks := make([]groupsig.Pubkey, candidates.size())
	signs := make([]groupsig.Signature, candidates.size())

	signDataBytes := common.Hex2Bytes("0x123")
	// Aggregate all miner sign seckey and pubkey
	for i, miner := range miners {
		shares := make([]groupsig.Seckey, candidates.size())
		for j, piece := range pieces {
			shares[j] = piece.pieces[i]
		}
		minerSks[i] = *groupsig.AggregateSeckeys(shares)
		minerPks[i] = *groupsig.NewPubkeyFromSeckey(minerSks[i])
		signs[i] = groupsig.Sign(minerSks[i], signDataBytes)
		if !groupsig.VerifySig(minerPks[i], signDataBytes, signs[i]) {
			t.Errorf("verify sign fail at %v", miner.ID.GetAddrString())
		}
	}

	groupSign := groupsig.RecoverSignature(signs, candidates.ids())
	// Verify group signature
	if !groupsig.VerifySig(*gpk, signDataBytes, *groupSign) {
		t.Fatalf("verify group signature fail")
	}

	// Build the group info
	members := make([]*genesisMemberMarshal, candidates.size())
	vrfPks := make([]base.VRFPublicKey, candidates.size())
	basePks := make([]groupsig.Pubkey, candidates.size())
	for i, miner := range miners {
		members[i] = &genesisMemberMarshal{ID: miner.ID, PK: minerPks[i]}
		vrfPks[i] = miner.VrfPK
		basePks[i] = miner.PK
	}

	groupInfo := &genesisGroupMarshal{
		Seed:      seed,
		Gpk:       *gpk,
		Threshold: uint32(candidates.threshold()),
		Members:   members,
		VrfPks:    vrfPks,
		Pks:       basePks,
	}

	jsonBytes, err := json.Marshal(groupInfo)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jsonBytes))

	msks := make([]string, candidates.size())
	for i, msk := range minerSks {
		msks[i] = msk.GetHexString()
	}
	mskString := strings.Join(msks, ",")
	t.Log(mskString)

	ioutil.WriteFile("genesis_group.info", jsonBytes, 0666)
	ioutil.WriteFile("genesis_msk.info", []byte(mskString), 0666)
}
