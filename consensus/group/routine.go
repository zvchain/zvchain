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

// Package group implements the group creation protocol
package group

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
	"math"
)

const (
	threshold             = 51  // BLS threshold, percentage number which should divide by 100
	recvPieceMinRatio     = 0.8 // The minimum ratio of the number of participants in the final group-creation to the expected number of nodes
	memberMaxJoinGroupNum = 5   // Maximum number of group one miner can participate in
)

func candidateCount(totalN int) int {
	if totalN >= model.Param.GroupMemberMax {
		return model.Param.GroupMemberMax
	} else if totalN < model.Param.GroupMemberMin {
		return 0
	}
	return totalN
}

func candidateEnough(n int) bool {
	return n >= model.Param.GroupMemberMin
}

func pieceEnough(pieceNum, candidateNum int) bool {
	return pieceNum >= int(math.Ceil(float64(candidateNum)*recvPieceMinRatio))
}

type groupContextProvider interface {
	GetGroupStoreReader() types.GroupStoreReader

	GetGroupPacketSender() types.GroupPacketSender

	RegisterGroupCreateChecker(checker types.GroupCreateChecker)
}

type minerReader interface {
	SelfMinerInfo() *model.SelfMinerDO
	GetLatestVerifyMiner(id groupsig.ID) *model.MinerDO
	GetCanJoinGroupMinersAt(h uint64) []*model.MinerDO
}

type createRoutine struct {
	*createChecker
	packetSender types.GroupPacketSender
	store        *skStorage
}

var routine *createRoutine

func InitRoutine(reader minerReader, chain types.BlockChain, provider groupContextProvider, miner *model.SelfMinerDO) *skStorage {
	checker := newCreateChecker(reader, chain, provider.GetGroupStoreReader())
	routine = &createRoutine{
		createChecker: checker,
		packetSender:  provider.GetGroupPacketSender(),
		store:         newSkStorage(fmt.Sprintf("groupsk%v.store", common.GlobalConf.GetString("instance", "index", "")), base.Data2CommonHash(miner.SK.Serialize()).Bytes()),
	}
	top := chain.QueryTopBlock()
	routine.updateContext(top)

	routine.store.initAndLoad()
	go routine.store.loop()

	provider.RegisterGroupCreateChecker(checker)

	notify.BUS.Subscribe(notify.BlockAddSucc, routine.onBlockAddSuccess)
	return routine.store
}

func (routine *createRoutine) onBlockAddSuccess(message notify.Message) {
	block := message.GetData().(*types.Block)
	bh := block.Header

	routine.store.blockAddCh <- bh.Height

	routine.updateContext(bh)
	err := routine.checkAndSendEncryptedPiecePacket(bh)
	if err != nil {
		routine.logger.Errorf("check and send encrypted piece error:%v at %v-%v", err, bh.Height, bh.Hash.Hex())
	} else {
		routine.logger.Debugf("checkAndSendEncryptedPiecePacket sent encrypted packet at %v, seedHeight %v", bh.Height, routine.currEra().seedHeight)
	}
	err = routine.checkAndSendMpkPacket(bh)
	if err != nil {
		routine.logger.Errorf("check and send mpk error:%v at %v-%v", err, bh.Height, bh.Hash.Hex())
	} else {
		routine.logger.Debugf("checkAndSendMpkPacket sent mpk packet at %v, seedHeight %v", bh.Height, routine.currEra().seedHeight)
	}
	err = routine.checkAndSendOriginPiecePacket(bh)
	if err != nil {
		routine.logger.Errorf("check and send origin piece error:%v at %v-%v", err, bh.Height, bh.Hash.Hex())
	} else {
		routine.logger.Debugf("checkAndSendOriginPiecePacket sent origin packet at %v, seedHeight %v", bh.Height, routine.currEra().seedHeight)
	}

}

// UpdateEra updates the era info base on current block header
func (routine *createRoutine) updateContext(bh *types.BlockHeader) {
	routine.lock.Lock()
	defer routine.lock.Unlock()

	sh := seedHeight(bh.Height)
	seedBH := routine.chain.QueryBlockHeaderByHeight(sh)
	if routine.ctx != nil {
		curEra := routine.currEra()
		if curEra.sameEra(sh, seedBH) {
			return
		}
	}
	seedBlockHash := common.Hash{}
	if seedBH != nil {
		seedBlockHash = seedBH.Hash
	}
	routine.logger.Debugf("new create context: era:%v-%v", sh, seedBlockHash)
	routine.ctx = newCreateContext(newEra(sh, seedBH))
	err := routine.selectCandidates()
	if err != nil {
		routine.logger.Debugf("select candidates:%v", err)
	}
}

func (routine *createRoutine) selectCandidates() error {
	// Already selected
	if routine.ctx.cands != nil {
		return nil
	}

	routine.ctx.cands = make(candidates, 0)

	era := routine.currEra()
	h := era.seedHeight
	bh := era.seedBlock

	allVerifiers := routine.minerReader.GetCanJoinGroupMinersAt(h)
	if !candidateEnough(len(allVerifiers)) {
		return fmt.Errorf("not enought candidate in all:%v", len(allVerifiers))
	}

	availCandidates := make([]*model.MinerDO, 0)
	for _, m := range allVerifiers {
		cnt := routine.storeReader.MinerLiveGroupCount(m.ID.ToAddress(), h)
		if cnt < memberMaxJoinGroupNum {
			availCandidates = append(availCandidates, m)
		}
	}

	memberCnt := candidateCount(len(availCandidates))
	if !candidateEnough(memberCnt) {
		return fmt.Errorf("not enough candiates in availables:%v", len(availCandidates))
	}

	selector := newCandidateSelector(availCandidates, bh.Random)
	selectedCandidates := selector.fts(memberCnt)

	mems := make([]string, len(selectedCandidates))
	for _, m := range selectedCandidates {
		mems = append(mems, m.ID.GetHexString())
	}
	routine.logger.Debugf("selected candidates at seed %v-%v is %v", era.seedHeight, era.Seed().Hex(), mems)

	routine.ctx.cands = selectedCandidates
	return nil
}

func (routine *createRoutine) checkAndSendEncryptedPiecePacket(bh *types.BlockHeader) error {
	routine.lock.Lock()
	defer routine.lock.Unlock()

	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.encPieceRange.inRange(bh.Height) {
		return fmt.Errorf("not in the encrypted-piece round")
	}
	mInfo := routine.minerReader.SelfMinerInfo()
	if mInfo == nil {
		return fmt.Errorf("miner is nil")
	}
	if !mInfo.CanJoinGroup() {
		routine.logger.Debugf("miner info:%+v", mInfo.MinerDO)
		return fmt.Errorf("current miner cann't join group")
	}

	// Was selected
	if !routine.ctx.cands.has(mInfo.ID) {
		return fmt.Errorf("current miner not selected:%v", mInfo.ID.GetHexString())
	}
	// Has sent piece
	if routine.ctx.sentEncryptedPiecePacket != nil || routine.storeReader.HasSentEncryptedPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("has sent encrypted pieces")
	}

	encSk := generateEncryptedSeckey()

	// Generate encrypted share piece
	packet := generateEncryptedSharePiecePacket(mInfo, encSk, era.Seed(), routine.ctx.cands)
	routine.store.storeSeckey(era.Seed(), nil, &encSk, bh.Height+expireHeightGap)

	// Send the piece packet
	err := routine.packetSender.SendEncryptedPiecePacket(packet)
	if err != nil {
		return fmt.Errorf("send packet error:%v", err)
	}
	routine.ctx.sentEncryptedPiecePacket = packet

	return nil
}

func (routine *createRoutine) checkAndSendMpkPacket(bh *types.BlockHeader) error {
	routine.lock.Lock()
	defer routine.lock.Unlock()

	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.mpkRange.inRange(bh.Height) {
		return fmt.Errorf("not in the mpk round")
	}

	mInfo := routine.minerReader.SelfMinerInfo()
	if mInfo == nil {
		return fmt.Errorf("miner is nil")
	}
	if !mInfo.CanJoinGroup() {
		return fmt.Errorf("current miner cann't join group")
	}

	cands := routine.ctx.cands

	// Was selected
	if !cands.has(mInfo.ID) {
		return fmt.Errorf("current miner not selected:%v", mInfo.ID.GetHexString())
	}

	// Has sent mpk
	if routine.ctx.sentMpkPacket != nil || routine.storeReader.HasSentMpkPacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("has sent mpk packet")
	}
	// Didn't sent share piece
	if routine.ctx.sentEncryptedPiecePacket == nil && !routine.storeReader.HasSentEncryptedPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("didn't send encrypted piece")
	}

	encryptedPackets, err := routine.storeReader.GetEncryptedPiecePackets(era)
	if err != nil {
		return fmt.Errorf("get receiver piece error:%v", err)
	}

	num := len(encryptedPackets)
	routine.logger.Debugf("get encrypted pieces size %v", num)

	// Check if the received pieces enough
	if !pieceEnough(num, cands.size()) {
		return fmt.Errorf("received piece not enough, recv %v, total %v", num, cands.size())
	}

	msk, err := aggrSignSecKeyWithMySK(encryptedPackets, cands.find(mInfo.ID), mInfo.SK)
	if err != nil {
		return fmt.Errorf("genearte msk error:%v", err)
	}
	routine.store.storeSeckey(era.Seed(), msk, nil, bh.Height+expireHeightGap)

	mpk := groupsig.NewPubkeyFromSeckey(*msk)
	// Generate encrypted share piece
	packet := &mpkPacket{
		sender: mInfo.ID,
		seed:   era.Seed(),
		mPk:    *mpk,
		sign:   groupsig.Sign(*msk, era.Seed().Bytes()),
	}

	// Send the piece packet
	err = routine.packetSender.SendMpkPacket(packet)
	if err != nil {
		return fmt.Errorf("send mpk packet error:%v", err)
	}
	routine.ctx.sentMpkPacket = packet
	return nil
}

func (routine *createRoutine) checkAndSendOriginPiecePacket(bh *types.BlockHeader) error {
	routine.lock.Lock()
	defer routine.lock.Unlock()

	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.oriPieceRange.inRange(bh.Height) {
		return fmt.Errorf("not in the encrypted-piece round")
	}
	mInfo := routine.minerReader.SelfMinerInfo()
	if mInfo == nil {
		return fmt.Errorf("miner is nil")
	}
	if !mInfo.CanJoinGroup() {
		return fmt.Errorf("current miner cann't join group")
	}

	// Was selected
	if !routine.ctx.cands.has(mInfo.ID) {
		return fmt.Errorf("current miner not selected:%v", mInfo.ID.GetHexString())
	}
	// Whether origin piece required
	if !routine.storeReader.IsOriginPieceRequired(era) {
		return fmt.Errorf("don't need origin pieces")
	}
	id := mInfo.ID.Serialize()
	// Whether sent encrypted pieces
	if !routine.storeReader.HasSentEncryptedPiecePacket(id, era) {
		return fmt.Errorf("didn't sent encrypted share piece")
	}
	// Whether sent mpk packet
	if !routine.storeReader.HasSentMpkPacket(id, era) {
		return fmt.Errorf("didn't sent mpk packet")
	}
	// Has sent piece
	if routine.ctx.sentOriginPiecePacket != nil || routine.storeReader.HasSentOriginPiecePacket(id, era) {
		return fmt.Errorf("has sent origin pieces")
	}

	ski := routine.store.getSkInfo(era.Seed())
	if ski == nil {
		return fmt.Errorf("has no encrypted seckey")
	}

	// Generate origin share piece
	sp := generateSharePiecePacket(mInfo, ski.encSk, era.Seed(), routine.ctx.cands)
	packet := &originSharePiecePacket{sharePiecePacket: sp}

	// Send the piece packet
	err := routine.packetSender.SendOriginPiecePacket(packet)
	if err != nil {
		return fmt.Errorf("send packet error:%v", err)
	}
	routine.ctx.sentOriginPiecePacket = packet

	return nil
}
