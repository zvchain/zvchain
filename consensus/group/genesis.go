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

package group

import (
	"encoding/json"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"strings"
)

type genesisMemberMarshal struct {
	ID groupsig.ID
	PK groupsig.Pubkey
}

// genesisGroupMarshal defines the data struct of genesis verifyGroup info for marshalling
type genesisGroupMarshal struct {
	Seed      common.Hash
	Gpk       groupsig.Pubkey
	Threshold uint32
	Members   []*genesisMemberMarshal
	VrfPks    []base.VRFPublicKey
	Pks       []groupsig.Pubkey
}

var genesisGroupInfo *types.GenesisInfo

// GenerateGenesis generate genesis verifyGroup info for chain use
func GenerateGenesis() *types.GenesisInfo {
	if genesisGroupInfo != nil {
		return genesisGroupInfo
	}

	f := common.GlobalConf.GetSectionManager("consensus").GetString("genesis_group_info", "")
	genesis := genGenesisStaticGroupInfo(f)
	gHeader := &groupHeader{
		seed:          genesis.Seed,
		workHeight:    0,
		dismissHeight: common.MaxUint64,
		gpk:           genesis.Gpk,
		threshold:     genesis.Threshold,
	}
	members := make([]types.MemberI, len(genesis.Members))
	for i, mem := range genesis.Members {
		members[i] = &member{id: mem.ID.Serialize(), pk: mem.PK.Serialize()}
	}
	coreGroup := &group{header: gHeader, members: members}

	vrfPKs := make([][]byte, len(genesis.VrfPks))
	pks := make([][]byte, len(genesis.Pks))

	for i, vpk := range genesis.VrfPks {
		vrfPKs[i] = vpk
	}
	for i, vpk := range genesis.Pks {
		pks[i] = vpk.Serialize()
	}
	info := &types.GenesisInfo{
		Group:  coreGroup,
		VrfPKs: vrfPKs,
		Pks:    pks,
	}
	genesisGroupInfo = info
	return info
}

func genGenesisStaticGroupInfo(f string) *genesisGroupMarshal {
	sgiData := []byte(types.GenesisDefaultGroupInfo)
	if strings.TrimSpace(f) != "" {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			// panic is allowed if only called in init function
			panic(err)
		}
		sgiData = data
	}

	genesis := new(genesisGroupMarshal)
	err := json.Unmarshal(sgiData, genesis)
	if err != nil {
		// panic is allowed if only called in init function
		panic(err)
	}
	return genesis
}
