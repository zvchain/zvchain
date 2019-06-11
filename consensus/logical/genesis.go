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
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	time2 "github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
)

// GenesisDefaultGroupInfo represent for the basic info of the genesis group
const GenesisDefaultGroupInfo = `{"GroupID":"0xa85a46d7fde553e2dbfe19750faee0144d361695","GroupPK":"0x1 161a9ae59369f30530f162890575713ba964249cadb38c4c3d4fc0f9c47b891317b8be0b2f1a87ca737819afd5621355 23aad05550c100e20ebba6accd4ee3e40c1c5be0feaeac26071c3e6bac22ae597b493f1e7d07421fb8e2db02f930226a 1915be8fbe1c8562d7ca42e0e035b5ca01e5bf5f90db1145b0655e749f3ad2ac4580d5b15f03cbaf279d5e667b3af50f b821db672b203385b9e55e5dabd666b7454049f53a2032b50202a292322f718965d7b274a35b56e2a4ee601b6b10fbd","Members":[{"ID":"0x77696c64206368696c6472656e","PK":"0x1 2108f2e814c7b1084d07c40a408cb07050094ebcd7a88e8764d3e577d9569a4f78b8917b370ee2295497bc966f6172ae 5a29c58eda4aa1c9121e708fd88752c4e0243052e809283e02c547a88c8267398616049aefc242acb7d9bdc3fa375d1 690ebf4faf2da06ee68ee8a16081f38ed26fc9206d13e906b36da3038339f427b11313fb45b880cba4ca7e6b6fae819 21bc936fb7c9fac7b5a6ae112e0f65cda4cb1746b054e0c04be22b40eb735605ca5d82f63d7b6c0ad59cdde75751295f"},{"ID":"0x67656261696e69","PK":"0x1 140a1d17b1750d446349c9386a70f3f91058ca6483fc36698de433ef93b4c5f9ed5c9eca6024ffce6f44fbb847a29441 6f3db140e6b72f9aa115a2829093d8692c526708a27e48d01347800878ea8d9e8cf4542a2f6d975e3fe68cae10a9cf7 1b5b725980ada2a0174297352d50d83f8c0a163af16c2a75a621887373a3e967d2e534cc7b16a04f713c7b3efbdb3735 dc63616d2c8fbc71581fd7e2511c58f199097c660b9dbda642b3e6a6cbec954812db8facd7614c5b417ccb26ba9a504"},{"ID":"0x77656e71696e","PK":"0x1 18a6db6582accb0103fc82649c614bb5243119b22d15e6517c79135915b6c9bb80ebe10ca30a3f76913d15940e4892e5 8d825a7b374084643c36a62300e71a7292d6df25b53c9394fe4ff52793edd1d87f8a6107e8ceddb00c3b41fc0b79f5f 4b701ef074bfd8af388c47b44cf4a97134c8783c12d82b9a6d1159a68eb118049d01b1346a4c518faf167651ecd16cc aeada3147866a862712d350421ac8578919c5d2e2900620b3ef89975227b653e7dd25cf2cde8b9ccf69095617b4cf65"},{"ID":"0x62616f7a6875","PK":"0x1 198ca0bef3a72bfde3707af5a30c241e62c4ec0f2ac7287c616c02c8543ffb63e3ee49c2e3eaea6dd78f326e76e5daf1 1ea6077383c21f666f36caef7c02db2f79a0c911a14db60ee9e1292e14c99d0326b18db0494a2a8d628ce7d3d8331b37 123f851696561912a136e0720bbc38cac062b4e89f515d2dc68361934da5a7e2867310112aa4926e4cd04d12f63be30c 1aff63e13e5f935631aa70aa45cd78a9799cab60e15aab128a354cb1bf560ccbd642746e0a6d2f9d21403460d23dc9d3"},{"ID":"0x74686965666f78","PK":"0x1 1ee1ca610b2bbf4d867f0ab4c0ad4e17cefda4c2e6587fc2245e29f8040b013e8433039454cf9bcbf870174bcf5f9ab 1024d85e64785bb522b5022d4593d1ecc528a835585bd8d5c8ffffdb8d0c396893fe1da9b78f3e25bb407d0660ad5290 20c4fe9f96db267f0c64576898b172fc2aa143f3fb5c0c6c35db32ec9197eba047a6e2f9a44cadec88071bf385fa9f2c 1dfdcd3272def12a54db5ba69712dc932b3f6001c9c8aef572b0aa84a829a72387d56c73f54b03129cb0cb8dc8a35e2d"},{"ID":"0x736972656e","PK":"0x1 1badfe7a1e3c5c32bef1e2f28af91f1f79a7cb5070a95677dc251edc75e81db178b98f3a05e222cf5828201401d09530 194396834ceb649fd6e562d7dbc3949e5289cea5bb833ec09967cbbaa67c29d36c5c3ad03c66c357744287b4d3fd9ed2 1d5e623e62c2f3b30675aa10599aa19ce7321f80d2551252f6bfbf5dd551d3e683e308c4d05426c5eb5119bd6fa3f744 4df93bf3cf3ae03aede6816270a7e2fa6294fb7cef78f75c95caa44953d1dceb834fff0a033a923872da3ef644e0220"},{"ID":"0x6a75616e7a69","PK":"0x1 31ac2d6c95916df11ba31e3cd4025162b55d1a8376038ee1c64ebbd2d3ede429bae81defc736a4b217a0129159c6172 106b4c114f86bf7aee6be577f7754a084bcb9cfebd0532a4f87d626d0d83b1195b8615e940623a4a4ecf2a8da54011a3 19ea0ff2a9452cf6d2e5b9e4f82e971e47f7fa935ef44361cf1fa6c2bac3a805c95b8c07f289096097ca5c8b9a3e28fb 192bf976636dfc8022687d6a4cdc01ae8235296fb6791d45000e5af1d494ca64d42e0ca31a305bde2ca9babff629c690"}],"MemIndex":{"0x62616f7a6875":3,"0x67656261696e69":1,"0x6a75616e7a69":6,"0x736972656e":5,"0x74686965666f78":4,"0x77656e71696e":2,"0x77696c64206368696c6472656e":0},"GIS":{"ParentID":"0x0","Signature":{},"Authority":777,"Name":[84,65,83,32,103,101,110,101,115,105,115,32,103,114,111,117,112,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"DummyID":"0x547275737420416d6f6e6720537472616e67657273","Members":7,"BeginTime":1554048000,"MemberHash":"0x07226e8b75714cb9f9f8f27fc0215c16621c83963499e9e94cd14a2cc3242222","TopHeight":0,"GetReadyHeight":1,"BeginCastHeight":0,"DismissHeight":18446744073709551615,"Extends":"room 1003, BLWJXXJS6KYHX"},"WorkHeight":0,"DismissHeight":18446744073709551615,"ParentId":"0x0","Signature":{},"Authority":777,"Name":"TAS genesis group\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000","Extends":"room 1003, BLWJXXJS6KYHX"}`

// genesisGroup defines the data struct of genesis group info
type genesisGroup struct {
	Group StaticGroupInfo     // static group info
	VrfPK []base.VRFPublicKey // vrf public keys of each members in the group
	Pks   []groupsig.Pubkey   // the non-group bls public keys of each members in the group
}

// genesisGroupInfo parsed from GenesisDefaultGroupInfo or config file which is singleton
var genesisGroupInfo *genesisGroup

func getGenesisGroupInfo() *genesisGroup {
	if genesisGroupInfo == nil {
		f := common.GlobalConf.GetSectionManager("consensus").GetString("genesis_sgi_conf", "genesis_sgi.config")
		common.DefaultLogger.Debugf("generate genesis info %v", f)
		genesisGroupInfo = genGenesisStaticGroupInfo(f)
	}
	return genesisGroupInfo
}

// GenerateGenesis generate genesis group info for chain use
func GenerateGenesis() *types.GenesisInfo {
	genesis := getGenesisGroupInfo()
	sgi := &genesis.Group
	coreGroup := convertStaticGroup2CoreGroup(sgi)
	vrfPKs := make([][]byte, sgi.GetMemberCount())
	pks := make([][]byte, sgi.GetMemberCount())

	for i, vpk := range genesis.VrfPK {
		vrfPKs[i] = vpk
	}
	for i, vpk := range genesis.Pks {
		pks[i] = vpk.Serialize()
	}
	return &types.GenesisInfo{
		Group:  *coreGroup,
		VrfPKs: vrfPKs,
		Pks:    pks,
	}
}

// BeginGenesisGroupMember Generate Genesis member information
func (p *Processor) BeginGenesisGroupMember() {

	genesis := getGenesisGroupInfo()
	if !genesis.Group.MemExist(p.GetMinerID()) {
		return
	}
	sgi := &genesis.Group

	// join the genesis group if and only if current node belongs to the group
	jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
	if jg == nil {
		panic("genesisMember find join_group fail")
	}
	p.joinGroup(jg)

}

func generateGenesisGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
	gh := &types.GroupHeader{
		Name:          "TAS genesis group",
		Authority:     777,
		BeginTime:     time2.TimeToTimeStamp(time.Now()),
		CreateHeight:  0,
		ReadyHeight:   1,
		WorkHeight:    0,
		DismissHeight: common.MaxUint64,
		MemberRoot:    model.GenMemberRootByIds(memIds),
		Extends:       "",
	}

	gh.Hash = gh.GenHash()
	return gh
}

func genGenesisStaticGroupInfo(f string) *genesisGroup {
	sgiData := []byte(GenesisDefaultGroupInfo)
	if strings.TrimSpace(f) != "" {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			panic(err)
		}
		sgiData = data
	}

	genesis := new(genesisGroup)
	err := json.Unmarshal(sgiData, genesis)
	if err != nil {
		panic(err)
	}
	group := genesis.Group
	group.buildMemberIndex()
	return genesis
}
