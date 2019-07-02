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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware"
)

const confPathPrefix = `genesis_test_file`
const procNum = 3

var minerInfo = map[string]string{
	"0xf5a20ed1efcd6f49c076725a490992ebd7775d1075863dad88ad7ae0f8ae5a3e": "0x04266970d0941f145bbb4f20b3d6260ac67b44b9df4783215b3e05b81b1866ea0605c3098f17c4862f276bb57add6f73d728fc66e3200876f76f152cb3e49eff5852b06e555ad9fa0df06d5bd257d7801279742b21667669ad0a46ce98ba76e5ae",
	"0xe6fc8e413f66fbcf676d05670471f35312d8888fb28b9bfcbdc1dcaa56287aaa": "0x049fd46abe4d9d3b05ae72093ed98352df517dc5c31c7f1e27938f5f1dade10b4b0eb10e6857f615ccef1c1e8ca3b9d92d6ff734ef406b91fa8ad44b229a1d8e0079ea9ef0e1b97619339ee2fd49bf637953fe1df0e51740be28d9a2c7d33b26b5",
	"0x67a9bcc768bbfc94adaf33fde12076aa544a4495f42833692f5bff373001f214": "0x049e4676e5e24868b25e59713adc3843ec50ebdae1f9c8ea9cbd6a45404df5db00ee3de29bbf4fb1e115ff125b7dee933d378eff3c74bec391725d50f86bfd97fdcebc9cefbc36e00704f9d70edd8efcf1fa6df5ed0b53abcd2ae4af6863bc8acd",
}

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	proc := new(Processor)
	addr := cm.GetString("gtas", "miner", "")

	gstore := fmt.Sprintf("%v/groupstore%v", confPathPrefix, cm.GetString("instance", "index", ""))
	cm.SetString("consensus", "groupstore", gstore)
	minerDO, _ := model.NewSelfMinerDO(common.HexToSecKey(minerInfo[addr]))
	proc.Init(minerDO, cm)
	log.Printf("%v", proc.mi.VrfPK)
	return proc
}

func processors() (map[string]*Processor, map[string]int) {
	maxProcNum := procNum
	procs := make(map[string]*Processor, maxProcNum)
	indexs := make(map[string]int, maxProcNum)

	for i := 1; i <= maxProcNum; i++ {
		proc := initProcessor(fmt.Sprintf("%v/tas%v.ini", confPathPrefix, i))
		//proc.belongGroups.storeFile = fmt.Sprintf("%v/joined_group.config.%v", CONF_PATH_PREFIX, i)
		procs[proc.GetMinerID().GetHexString()] = proc
		indexs[proc.getPrefix()] = i
	}

	return procs, indexs
}

func TestGenesisGroup(t *testing.T) {
	//groupsig.Init(1)
	middleware.InitMiddleware()
	common.InitConf(confPathPrefix + "/tas1.ini")

	InitConsensus()

	procs, _ := processors()

	mems := make([]groupsig.ID, 0)
	for _, proc := range procs {
		mems = append(mems, proc.GetMinerID())
	}
	gh := generateGenesisGroupHeader(mems)
	gis := &model.ConsensusGroupInitSummary{
		GHeader: gh,
	}
	grm := &model.ConsensusGroupRawMessage{
		GInfo: model.ConsensusGroupInitInfo{GI: *gis, Mems: mems},
	}

	ids := make([]groupsig.ID, 0)
	for _, p := range procs {
		ids = append(ids, p.GetMinerID())
	}

	procSpms := make(map[string][]*model.ConsensusSharePieceMessage)

	model.Param.GroupMemberMax = len(mems)
	model.Param.GroupMemberMin = len(mems)

	for _, p := range procs {
		gc := p.joiningGroups.ConfirmGroupFromRaw(grm, ids, p.mi)
		shares := gc.GenSharePieces()
		for id, share := range shares {
			spms := procSpms[id]
			if spms == nil {
				spms = make([]*model.ConsensusSharePieceMessage, 0)
				procSpms[id] = spms
			}
			var dest groupsig.ID
			dest.SetHexString(id)
			spm := &model.ConsensusSharePieceMessage{
				GHash: grm.GInfo.GroupHash(),
				Dest:  dest,
				Share: share,
			}
			spm.SI.SignMember = p.GetMinerID()
			spms = append(spms, spm)
			procSpms[id] = spms
		}
	}

	spks := make(map[string]*model.ConsensusSignPubKeyMessage)
	initedMsgs := make(map[string]*model.ConsensusGroupInitedMessage)

	for id, spms := range procSpms {
		p := procs[id]
		for _, spm := range spms {
			gc := p.joiningGroups.GetGroup(spm.GHash)
			ret := gc.PieceMessage(spm.SI.GetID(), &spm.Share)
			if ret == 1 {
				jg := gc.GetGroupInfo()
				p.joinGroup(jg)
				msg := &model.ConsensusSignPubKeyMessage{
					GHash:   spm.GHash,
					GroupID: jg.GroupID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}
				//msg.GenGSign(jg.SignKey)
				msg.SI.SignMember = p.GetMinerID()
				spks[id] = msg

				var initedMsg = &model.ConsensusGroupInitedMessage{
					GHash:        spm.GHash,
					GroupID:      jg.GroupID,
					GroupPK:      jg.GroupPK,
					CreateHeight: 0,
				}
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
				initedMsg.GenSign(ski, initedMsg)

				initedMsgs[id] = initedMsg
			}
		}
	}

	for _, p := range procs {
		for _, spkm := range spks {
			jg := p.belongGroups.getJoinedGroup(spkm.GroupID)
			p.belongGroups.addMemSignPk(spkm.SI.GetID(), spkm.GroupID, spkm.SignPK)
			log.Printf("processor %v join group gid %v\n", p.getPrefix(), jg.GroupID.ShortS())
		}
	}

	for _, p := range procs {
		for _, msg := range initedMsgs {
			initingGroup := p.globalGroups.GetInitedGroup(msg.GHash)
			if initingGroup == nil {
				ginfo := &model.ConsensusGroupInitInfo{
					GI: model.ConsensusGroupInitSummary{
						Signature: groupsig.Signature{},
						GHeader:   gh,
					},
					Mems: mems,
				}
				initingGroup = createInitedGroup(ginfo)
				p.globalGroups.generator.addInitedGroup(initingGroup)
			}
			if initingGroup.receive(msg.SI.GetID(), msg.GroupPK) == InitSuccess {
				staticGroup := newSGIFromStaticGroupSummary(msg.GroupID, msg.GroupPK, initingGroup)
				p.globalGroups.AddStaticGroup(staticGroup)
			}
		}
	}

	write := false
	for id, p := range procs {
		//index := indexs[p.getPrefix()]

		sgi := p.globalGroups.GetAvailableGroups(0)[0]
		jg := p.belongGroups.getJoinedGroup(sgi.GroupID)
		if jg == nil {
			log.Printf("jg is nil!!!!!! p=%v, gid=%v\n", p.getPrefix(), sgi.GroupID.ShortS())
			continue
		}
		jgByte, _ := json.Marshal(jg)

		if !write {
			write = true

			genesis := new(genesisGroup)
			genesis.Group = *sgi

			vrfpks := make([]base.VRFPublicKey, sgi.GetMemberCount())
			pks := make([]groupsig.Pubkey, sgi.GetMemberCount())
			for i, mem := range sgi.GetMembers() {
				_p := procs[mem.GetHexString()]
				vrfpks[i] = _p.mi.VrfPK
				pks[i] = _p.mi.GetDefaultPubKey()
			}
			genesis.VrfPK = vrfpks
			genesis.Pks = pks

			log.Println("=======", id, "============")
			sgiByte, _ := json.Marshal(genesis)

			ioutil.WriteFile(fmt.Sprintf("%s/genesis_sgi.config", confPathPrefix), sgiByte, os.ModePerm)

			log.Println(string(sgiByte))
			log.Println("-----------------------")
			log.Println(string(jgByte))
		}

		log.Println()

		//ioutil.WriteFile(fmt.Sprintf("%s/genesis_jg.config.%v", CONF_PATH_PREFIX, index), jgByte, os.ModePerm)
	}

}

func TestGetGenesisGroupInfo(t *testing.T) {
	file := "genesis_sgi_test.config"
	gg := genGenesisStaticGroupInfo(file)

	if gg.Group.GroupID.GetHexString() != "0x015fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735" {
		t.Errorf("group id error")
	}
	if gg.Group.GroupPK.GetHexString() != "0x01367332311024aa875285ade328974e8645ebe2833de109c9960c81a97a333408d4f0b4f9876e80fead0f802a22507dc1b2ad5b75836d15e9ab55747c52107c02bd626518d6b03c6d4421d8f63768f8ed2cf511a1947a05402efae14c77e3db2b20e9182bd94ca4c90e330a8db6c27bd52ef44f86bf01afd25dae660592157d" {
		t.Errorf("group pk error")
	}

}
