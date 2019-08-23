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
	"io/ioutil"
	"strings"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/middleware/types"
)

// GenesisDefaultGroupInfo represent for the basic info of the genesis verifyGroup
const GenesisDefaultGroupInfo = `{"Group":{"Group":"0x0e80fc3e813b5e48c2c7cd7f88a6c26dd58f9ef8fe305a99833ca82af2d661e9","GroupPK":"0x28e79e70b045269247c923b78458364af04dee6ea4d909486f9cd2d9186bd9e1199aacc5e2cf468c5c5b71005b343d3e6ccae81b19e6c54e77739549c54547111c6fb5829bab62c1ca16c9391a838b7f6d5fa7bc13e1e1427cb56fe0a8c5cdf81829d46a4eb5531e04a91b5a0dcdacca4ebdda568ecf9de53ab796cf2fe0f643","MemIndex":{"0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4":0,"0x0cc66654d0099fdc10709df3bdb4eab4a68c97ec5dc66d087361d0cabb37db80":3,"0x27af3b203839ba7d47ceaac70b81dbe57074ade843c87b0e5562fd2a3dd3c990":5,"0x586e580e7d3352d617f35189ed4995679729a1a8b53ed8d91e46d7f8970d4737":1,"0x69d959d090df8c77adc85b5294871a904b0294eb85fb8251ba903710805d64c2":2,"0x806ec4eb2d7a2ba0ebee40e2a39e3e8b1f3a09d91ae78bd7fdf30c77e543f545":6,"0xb2e882c6d59b37636d65cd6e023d4f2bd49f25947c37221ac52b3c9b60278813":4,"0xb4c45be38afcc52fcbc6aa6706897b0c0f943bed691be2ac2beb18e5f2a07890":7,"0xc7d83d1e57ac5e2df25df8c569e87f62fc1173039faf3cb30f65d0efab9ecc50":8},"GInfo":{"GI":{"Signature":{},"GHeader":{"Hash":"0x3e3465a2de3340d74eaeef6b5e2789cf14cc6057b30c875f7d26b75109e4cb3f","Parent":null,"PreGroup":null,"Authority":777,"Name":"TAS genesis verifyGroup","BeginTime":1560240827,"MemberRoot":"0xc35de90b38cdbcd82ade26083b3fce148e4d10357ce4859ef02358b85d53b976","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""}},"Mems":["0x085fdd8d70ed4af61918f267829d7df06d686633af002b55410daaa3e59b08a4","0x586e580e7d3352d617f35189ed4995679729a1a8b53ed8d91e46d7f8970d4737","0x69d959d090df8c77adc85b5294871a904b0294eb85fb8251ba903710805d64c2","0x0cc66654d0099fdc10709df3bdb4eab4a68c97ec5dc66d087361d0cabb37db80","0xb2e882c6d59b37636d65cd6e023d4f2bd49f25947c37221ac52b3c9b60278813","0x27af3b203839ba7d47ceaac70b81dbe57074ade843c87b0e5562fd2a3dd3c990","0x806ec4eb2d7a2ba0ebee40e2a39e3e8b1f3a09d91ae78bd7fdf30c77e543f545","0xb4c45be38afcc52fcbc6aa6706897b0c0f943bed691be2ac2beb18e5f2a07890","0xc7d83d1e57ac5e2df25df8c569e87f62fc1173039faf3cb30f65d0efab9ecc50"]},"ParentID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPK":["5HDFCAi0E7ykddiDohFQ6kn9RguwSEokvdE72vSDZjQ=","rtTY7hpzG853FTXdLyfpsK/MWCuVb1+lRvzvHfje57c=","LNN3nTDJyexruHsVDDq2zx5ljcGub69jH80YK0yOMHk=","VjJC7LohK12LIb5rgbgSRNYRCAsI8IT5kZad2iu8e28=","PXz1vfJgacXf05hZ09X3Ss6heAqpTmWKnu1fx8zMlyc=","LngVXdInf5zIwBs1jqenGHebZai1nDvdXOywibY+esg=","d1oc6U43PeDCor/A7gtEvYWu/eUgptF5GM4NDinlqmI=","r57nD4Bz6KFFln/6L1re1nztiKfXIFUtGLSeR4wRjLE=","4k8Ppw9t3HkTqTG5khN6sOiaYQM5eBNUyKSbumNci8U="],"Pks":["0x169dcf2f252322b117c500db3eb8f86c4266aae226b5550838a3f3c5496dfe4b21a082666c202d505acd0a892f7e49d112a45344a5476e0fc85cf8600238b0351d86b3b3a2ed06d8a2a4f7c5d3f772d9a77a54d7d60ec06e92bf717be4f1ee9e0204bcfba515b65c207cdbde5d15d371d8378ef64ef04c1f78e08eceb8b8068a","0x0b550540ae0bfbd4d69f16495a3066e4a55d9d96239941783593c72184aabd121dbd09d428e8cfdb5c409b00f702a83afa6651925ff8fbf53d1ac6dea66c77be265ccedb314bb69150e02362cbcf6cb0ee34850966564e522664720c31826b0d187fbece4332552f135aa4387a0f49e783ac29f288cb5c91b8426fcaf404443e","0x234615c9c39b00c4c3895b98e232a632fd0527554593f5e371fddeae28f2ff50168b3ab64588a41f40c01484307294240538507353ab41aea240d6bbeb7f09bd2b4d766bb9114b9d0039b368b896c90b4d80a9d863fc1142f0010960b2b69f2709aab7e4be98b97426a47ebb7f94a593ac0a1631228a36b15a27dac45551be24","0x0b1dbb2de54e7aa87a055d9ef055038fce9962b5554fc26dc17ee0d730dffa062061b7d93bdf8b4d37d58a78617d6b35e58f671d1a0f5829fbd2e4336a272b42181b712c4f1eab9ed256eac83cbfd56f190a2c0b82bf2ee8d312f2e510cdd39f04ecac2202fc5919df25195f4a275d103c1bd56ce9533df3769b2131291b9f16","0x2771ee70ec1da2df10360d90838224c4d849ddb2514fab297d3d16db763ef7441a7ca51c09cb909a03106132d5ce533022f3c3c70c746c922d60d8cba6549b4a2c612c42b416fb6fcec583a3f16ea04d01ab6dac3870977c6149ff2b5362ffe91d92ca1c01b85036f2211351f4073442b5d26ff450faf93c6cbfa5efd2eb409e","0x0fbdbe0f5c346b48b007dbdc3079a711890138f07d9b7d2a85242498a8bcada014e9c048eee3afbe1125118d998018d96eb301ce6e3504a516485ed2a49b41300e09c1080e44679116f99cee6855351efc0d50cadd310fc172b6821347decc8b20e16b55b3e6942e7ada3de07b227d3c5e8a2631b38eac8f0ac9f94f5e964206","0x19abce37f3d456eb8ff1eb79b90a088c39e60c203c4f418d9f691a4f67d85102174c40e9ca401c403dacdbcc0b83b76e596129e8fd62c25a739714da4a049fa014e88f7dce7a103b46fa6ed2d72a142e989aa215663e31a3743a99e91f4912fd16dc8ffc385bbaf82238c1733e21ba4ec0a11cfe125016f8480552320a2aeeba","0x0832e824241d2a7d59ee566dee24d468d1edaf398f28807dd0209e02064a8b812c39e518260015fa37bd65a5475f70293d97cded1534d936e3fb145be6fa5c200c494bc4bb37fe2d60a11cca4fd13a6835992fd9767c4a0279aefabc1a970911079f81c5335d7f8aacf8bb76e68bea368c2bd7632c3662ab1a8b37919cdce8d6","0x19e36562ccd2421c575c00e72296fe32d10ff25d019c1cfab06f9bec7897d49a1f32d9d71a2d3b21e4986f80cbceb887ff11adcbd7e8316bcf9410fce930f5db212f13e222214e9cc769a1049c9a581e501a4c47a586eda9c2a30439ae24f325044a29a79c97e6d39f98534b491c65acea483f1c0a10462db0a22a5c31d8fc66"]}`

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

	f := common.GlobalConf.GetSectionManager("consensus").GetString("genesis_group_info", "genesis_group.info")
	genesis := genGenesisStaticGroupInfo(f)
	gHeader := &groupHeader{
		seed:          genGenesisGroupSeed(genesis),
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
	sgiData := []byte(GenesisDefaultGroupInfo)
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

func genGenesisGroupSeed(genesis *genesisGroupMarshal) common.Hash {
	return common.BytesToHash(genesis.Gpk.Serialize())
}
