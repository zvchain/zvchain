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
	"fmt"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/taslog"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
)


var (
	GroupMaxNum         uint64 = 20                                //最大组数
	GroupStableGapTime  uint64 = 20                                //稳定块数
	GroupRoundTime      uint64 = 100                               //建组时内部状态转换时间块数
	GroupCreateLoopTime uint64 = 300                               //单次建组循环时间块数 15 minutes
	GroupBaseLifeTime          = GroupCreateLoopTime * GroupMaxNum // 块存活时长 300 minutes

	// GroupMaxMembers means the maximum number of members in a group
	GroupMaxMembers int = 100

	// GroupMinMembers means the minimum number of members in a group
	GroupMinMembers int = 10
    logger taslog.Logger
)

// GroupManager is responsible for group creation
type Manager struct {
	chain     core.BlockChain
	minerUtil minerUtilI
}

func NewGroupManager() *Manager {
	return &Manager{
	}
}

func log() taslog.Logger  {
	if logger == nil {
		instance := common.GlobalConf.GetString("group", "index", "")
		logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
	}
	return logger
}

func (m *Manager) Init(c core.BlockChain) {
	m.chain = c
	//m.minerUtil = miner //TODO: set the minerUtil implement object
}

func (m *Manager) CheckSelfRound() {
	// need use lock?
	if !m.selfPreCheck() {
		return
	}
	m.checkSelfRound1()
	m.checkSelfRound2()
	m.checkSelfRound3()

}


func (m *Manager) DoGroupCreate(db *account.AccountDB) {
	seed := m.getSeedBlockHeader().Hash
	cData,err := m.getGroupCreatingDataFromDb(seed)
	if err != nil {
		return
	}
	if len(cData.pieces) == 0{
		return
	}

	if len(cData.mpk) < len(cData.pieces){
		return
	}
	//TODO: get group pk and id
	group := m.createGroup(db, seed)
	m.saveGroupDataToDb(seed, group)
}

func (m *Manager) DoGroupPunish() {
	if !m.selfPreCheck() {
		return
	}
	currentHeight := m.chain.Height()
	seed := m.getPreSeedBlockHeader().Hash
	if hasSent(seed, 4) {
		return
	}

	roundFirst := getRoundFirstBlockHeight(currentHeight, 4)
	if currentHeight < roundFirst {
		return
	}
	for i := roundFirst; i < currentHeight; i++ {
		exist := m.chain.QueryBlockHeaderByHeight(i)
		if exist != nil {
			return
		}
	}

	cData,err := m.getGroupCreatingDataFromDb(seed)
	if err != nil {
		return
	}
	if len(cData.pieces) == 0{
		return
	}
	if len(m.getGroupCandidatesByBh(seed)) < len(cData.pieces) {

	}

}


func (m *Manager) ExecuteGroupPieceTx() {

}
func (m *Manager) ExecuteGroupMpkTx() {

}
func (m *Manager) ExecuteGroupOriginPieceTx() {

}

//--------interface function end--------

func (m *Manager) checkSelfRound1() {
	currentHeight := m.chain.Height()
	seed := m.getSeedBlockHeader().Hash
	if hasSent(seed, 1) {
		return
	}

	roundFirst := getRoundFirstBlockHeight(currentHeight, 1)
	if currentHeight < roundFirst {
		return
	}
	for i := roundFirst; i < currentHeight; i++ {
		exist := m.chain.QueryBlockHeaderByHeight(i)
		if exist != nil {
			return
		}
	}
	// check self in the list

	_, err := m.chain.GetTransactionPool().AddTransaction(m.generatePieceTransaction())
	if err != nil {
		fmt.Errorf("failed to send group piece transaction", err)
		return
	}
	maskAsSent(seed, 1)
}

func (m *Manager) checkSelfRound2() {
	currentHeight := m.chain.Height()
	seed := m.getSeedBlockHeader().Hash
	if hasSent(seed, 2) {
		return
	}

	roundFirst := getRoundFirstBlockHeight(currentHeight, 2)
	if currentHeight < roundFirst {
		return
	}
	for i := roundFirst; i < currentHeight; i++ {
		exist := m.chain.QueryBlockHeaderByHeight(i)
		if exist != nil {
			return
		}
	}
	cData,err := m.getGroupCreatingDataFromDb(seed)
	if err != nil {
		return
	}
	if cData.pieces[m.minerUtil.getSelfAddress()] == nil {
		return
	}


	_, err = m.chain.GetTransactionPool().AddTransaction(m.generateMpkTransaction())
	if err != nil {
		fmt.Errorf("failed to send group piece transaction", err)
		return
	}
	maskAsSent(seed, 2)
}

func (m *Manager) checkSelfRound3() {
	currentHeight := m.chain.Height()
	seed := m.getSeedBlockHeader().Hash
	if hasSent(seed, 3) {
		return
	}

	roundFirst := getRoundFirstBlockHeight(currentHeight, 3)
	if currentHeight < roundFirst {
		return
	}
	for i := roundFirst; i < currentHeight; i++ {
		exist := m.chain.QueryBlockHeaderByHeight(i)
		if exist != nil {
			return
		}
	}
	group := m.getGroupBySeed(seed)
	if group != nil {
		return
	}
	cData,err := m.getGroupCreatingDataFromDb(seed)
	if err != nil {
		return
	}
	if cData.pieces[m.minerUtil.getSelfAddress()] == nil {
		return
	}
	if len(cData.mpk) < len(cData.pieces){
		return
	}

		_, err = m.chain.GetTransactionPool().AddTransaction(m.generateOriginPieceTransaction())
	if err != nil {
		fmt.Errorf("failed to send group piece transaction", err)
		return
	}
	maskAsSent(seed, 3)
}


func (m *Manager) selfPreCheck() bool {
	if m.chain.IsAdjusting() {
		return false
	}
	//if m.chain.syncing(){
	//	return false
	//}
	if !isSelfMiner() {
		return false
	}

	if !m.isSeedExist() {
		return false
	}
	return true
}

func (m *Manager) isSeedExist() bool {
	return m.getSeedBlockHeader() != nil
}

func (m *Manager) getSeedBlockHeader() *types.BlockHeader {
	return m.chain.QueryBlockHeaderByHeight(getSeedHeight(m.chain.Height()))
}

func (m *Manager) getPreSeedBlockHeader() *types.BlockHeader {
	return m.chain.QueryBlockHeaderByHeight(getSeedHeight(m.chain.Height()-GroupCreateLoopTime))
}


func (m *Manager) getGroupCandidatesByBh(height uint64) []common.Address {
	seed := m.chain.QueryBlockHeaderByHeight(getSeedHeight(height))
	if seed == nil {
		return nil
	}

	//m.minerUtil.getAllMiner()
	//TODO: 返回建组入选队列
	return nil
}

func (m *Manager) createGroup(db *account.AccountDB, seed common.Hash) *types.Group {
	return nil
}


func (m *Manager) getGroupBySeed(seed common.Hash) *types.Group {
	return nil
}
