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

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	proc := new(Processor)
	addr := common.HexToAddress(cm.GetString("gtas", "miner", ""))

	gstore := fmt.Sprintf("%v/groupstore%v", confPathPrefix, cm.GetString("instance", "index", ""))
	cm.SetString("consensus", "groupstore", gstore)

	proc.Init(model.NewSelfMinerDO(addr), cm)
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
			if initingGroup.receive(msg.SI.GetID(), msg.GroupPK) == INIT_SUCCESS {
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
