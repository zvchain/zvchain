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

// GenesisDefaultGroupInfo represent for the basic info of the genesis verifyGroup
const GenesisDefaultGroupInfo = `{"Seed":"0x6861736820666f72207a76636861696e27732067656e657369732067726f7570","Gpk":"0x12aaff2f5594b1e2f0a21430cbdae68c3933f4f0bd804179bdf34b5b08603e89296c42922ffa71754bc64a10c2b44095793758cb0a62302f5a269a99b69379012aac13b8c0f73a0f44724b8e504b7f7596d3e2229d9b9aadbec1de700a71299518d7e9c858467f7a5114c168a7aa959bc88083a90f5e0a5bf209a5b4bd7e7a4e","Threshold":5,"Members":[{"ID":"zvdd6ec7f81d9b989cdea55c0d8c7f45da92d0eef8b85bc66da66c27ec68ba016b","PK":"0x2c11b2aaadc33311c487c4d9fb4aea095b27d8d9b11e8d1c3e56e9fb5e4fb2da1e895e5d4be4ae5d0be0707c88a68d4796f21b86083725de6eb40db61954c7d82515a196caa8ed3ab241e5159cdb90e58d3ba9dbd6a105ab79e63d870d2e9f5a249d004225291ccd875a58e37f49b053dd08cd0d51f535ac6f7cfb467a0d7a4a"},{"ID":"zv9eae37dbc0c3a076fe9981fbb050ebd29dfa9f3a6b1a4a8ef91e9604356e8801","PK":"0x253dd87879d81d42ef54122ebd5738ee8a9ac6e5cb877d6298aa93ca989384550d6a98cb9c0b79052759985f92ec285b7ec21c7ea59f7affec0bf724c6156cc727d2fbf308dbf19c8ad5f5039e8b0ce56fe6e44c729cf3dd76ac86b65b204fe32f11222921f4aa9cb7472a14ecff5b012612a44f263cc5262ebd3b69fae14a0f"},{"ID":"zv2cd125d242305c01effc80b30f87c28920b65d91230985bd8efd5357c37dc6d5","PK":"0x24b8abda6308a20d6725dd978d55b24afab0f529a57b679b282e16aeffe8987622757ca9213e47bc39e2d02c6e2babba52026eae6f8cadce4acce9325d8c28ae0e600faee814f61f75e0bf750f014757ed4872a5e5fbe89f18a11af1236440261f783d9d501afc2b2b06b38a1e91fd1ab13fc82c1f0978e9d291c2505136c8bb"},{"ID":"zvd02a30936e503c2c3ba8f6049a876ecc188d871bb1e817eb56c54eee03ede766","PK":"0x118811f311347b16cf3a1c20186916ae3c27538e85f3cf474a1fbcf317f4d8f42ab9a1ee7882d8d111bdba64f370ce8b5a1949f3a570123ab7f82abbb22e2ddd15f157048ddec375d612439bd6f16740b87db3c4225ff43cbbb8ca6addbc862b0ec68a31283dd2d3b193b8d3248c755be2ce1ff5053038f3fcc99460dbd5bad9"},{"ID":"zvfda344891762be219e85ff16dfa0376df91b54b0753ee9a67c2e4f062fcf1cd7","PK":"0x13a4061bb264bf915a54417eae22f830ce32e4d4022b8e5fd69aaaf7cbd88a090fd5ac53ac5f1f3901c10a186de091622682e9e8638db3ab17c2eb41065aa72109725bd7d6b7333dc6936af1858eae000ca313a437510d5203132c36d79338cb2c4ea7079d60f8179f1509e9d6eae26c9db51e6b57aa0b1a6a5e343a620e8aba"},{"ID":"zvff4e03884d6acd5dd4c571ad7ad428e686cde516bb58eb1eaa8e4dd891a47863","PK":"0x0b212cb206ca359492edeb3522768f73738d937fc103eb275c7ecb5c1c6010dd271e7d6eceebdc32d97cc6fe6683d4e8f0598ec101564556d0bb4a926bc2b0480b3cec96282acb1e65572c75b82161d4d7b3f4a40a7c6d9025c5eb5d9eb3d3d504ee3a2649f2180db4216610704ea48c25367c1e4d4ffa529560d511867f16b3"},{"ID":"zvad0bde1096265978ab16afa1f63870077d237f9fd3f9e4b1bc3605b2c82dd0ae","PK":"0x050ade55ffeac024f32e6a5eae61fbe002dd69a5b5ce8d24b148d1270659814920f70e10a08c18252e4b55a827d4c0d550df4e7801b52a8fa40792db504fd0042ae61593ac8c641ce2e5981aea0d92ce30a7dcefdb578c420ab4868acdc05d9c1207e776fdc067f9efea1c1f82de8611d37db44fbb69dafc5f8fe48096ddb1b9"},{"ID":"zv4c037cf6962f336863e3462bb6540e500360d31b78442d0961f68d83335b454c","PK":"0x034a44e8edb405aec5097f7a9fe140561d017d1385ec265b375e99bed8be2765069e813320117b1f6cdd48ec987f3ecfd2303bfe8cd8553b41a2d25e91f5df441fdff77101c26b1b9b4ac2357cdb4010258b513971f95c44f147e4f85725dc29241a7f287be90a6b0ff16e0442200c01e35be50310bcaeb2a6c7d53c13f437d6"},{"ID":"zvc66b013a65d92b3ec0bfe42feb78ad70fbd562687da78c80d7c71e3d41235fc0","PK":"0x248dbcb10dc7f0d9b7391d1d0a418103587d535387bf3cc0ad14691a326e42261766ecb04acb7fdee556b2ba8aba568cf7b8e56bebadc3c044fe76daa9eeb4471b34d27564fd557b4c54a93fd851ba74a26fbd19d7b14181828dd5a9512883901a91c49a9d7dad5d56264bde3265b89eca61775854a648235ef0670467cefaea"}],"VrfPks":["0xf4a81acac8a8dd63687f9a8af075a00069ac5581279550b901586aed643f69d2","0xab08faa0f2b44700fcb5858212cf03f43e10084de9c1be3738bf175f701e3c3c","0x200ca069108ad82d671e64be4a538609c8790f82b69d0d73e992c79bda4bb584","0x11c3aa8f1630fea22af4cbb3d9b5b5e083b14cb6c4c33fff977d0e9a465e366a","0xa87de1b73ff9b2037832d9b09ac5dde01b6d0a99e18e15bac7bdeb87ba9fd486","0xd074974a4b794b53c63cedb07c5528fc97c5d6f5c5fa0d5606575cff7b4780a6","0xe45c1e6975b2868a4a2476ba7cd209c7a451ac74d6aecb4c0f67c9a64d26ecfe","0xe60e705ceb457a14a1c1040c638b1dd3702652892615c3fcd9808d2e9b0bd461","0x5cffa275721e128c5f33b63741cdfbeefe3d8c0037fe63ad749181c6e3601110"],"Pks":["0x15168614d5cc5582bb79a314ece852dd46fd66916796df7b8787419318882839298b667638424aa5e0f3b7dbb549890648e22e0c594651a0bfff11dd78deecb3185a533a3914590f43c0dd7ca4d96363bc2b664c0a02de4fb9cdad84ad4987221e291fce3303106e08338e69e44d4613392131bab32fed8a6aa82a3d459e8388","0x2db1a9723cea34561e72f3609f2fd4a9cad2d50d4c3ee5efca4f451c1539e8240cd9bc62964d9170902dad9b44f60590fd2360ddaef4991c15d1a6a44fc957a7111b70f0b119d5284f90aa1e2f0b9bddb9d153782fc981cc7e0ba761dd53619d15147bf42f4348c3d4c28d074a29150e2bc1ce1dcd18e51e1d7804b794b66d3e","0x0465b5c8daa3d3e51795b70a54a39697116379aa51abd008a03e05e16b0b0fa029f1b51527f87c9f2b88d4287d62798d24a43a6096f9085fef3ccca5ebeb5a540853181102c984a974e3bbb15c6474af11be38eef3fe9b086c09c57aed9681052e0b60f624b3b46952c61b2e6ff2c372e7014a992127d016c0f978dbe260c935","0x02cd59d6557af2510719bbe037bcb911127bbd2ea1e34a22818d6c26cb673dd5144e8cfceba0a4907bb3ef3fa2b2f8f3f49d1e95a49c44b4577f20ecda1c337d02928d9496ac55eadc3d5e8cde879c04277131c002eac430dd3a27dd84a908aa1a43b18fa9d829fee15242aaf5b09adfe3e90d6151bfdd511a88b51e609cc61c","0x1908a8e0f48b31864ae09686f41f9693c69ae4bdb312cfcaf990b34feb6bcc1d037fa98eb8c0cf64cf6af917b46bb3cdcace00db3ec9ce7c95eb22106b0cc22216add3a29fb517912945515acfee33e46eaa83b31127bfb15046bfbb375d20371ab23f196613a989d9380ea19b4d9dd6fc78fd1d239c8f78ab5bc335a0efb7dd","0x1d71168cabd9dac8708b5bac970c79a7320e722e1338a7705c8330726b86b72d1c0a6b0c68acbdeee43f4dc788b3f8c3e4477fbf94e5ff8a11ef360e388e2dd104ca6030496a624efe649941ea3f72b7ebd012596259c3235b7ebfbb5f64d03b06fb20b5b478b483efba67449c92aeb0e84c7efda3da57748b4be8df002c7b53","0x0e5e4789ed8a19003ac5def58a6023a741600945d5bd631e0d9dc3bb0a4308180f4ee623ad24aa57c381c67c196cf7b72a65d6a7efb148983230b99ca66c44861a3ff43db8b910573a20db072b7ab5840ad8caeab6f4a94badd35f85694d8d53168894ed8d5894082fc52b3e5daa059eb69b68eddd080a6a21a13fc6e8719579","0x2a6880bceed03323a21a02e094190316d4f48781bb7721a58eef25cb3520324d1d8edc7a7998c1ca4fb8c5ffe3fd9798f6687ddb79c1955a555faa81482f70da15c1f773926b63514be316007d457309c948f8117c24b1b86d517ace65d924a621affecf2f00f0359b3eb58c9c8c0098f534363746ccdd374b0d16c44e3a6bb6","0x25855b082bf9e0c707450fa0292e692aeac3736ebe4f35696532cee4cfe284fa0e6c7fd2d01bf7ab0e1a7ff20ddb0d6f9a5feb45faafcb46d731efe2291686fb12ae66b245e8cfb0718d4ffb972e48ff7260b2b57ea327008c078bfb381e1f690035c4df07a407fd82c8d0f79556b4dbf18d89e414d72513c4d5a88b4d5bc1b7"]}`

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
