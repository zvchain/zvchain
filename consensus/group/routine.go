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
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"math"
)

const (
	groupMemberMin        = 80
	groupMemberMax        = 100
	threshold             = 51
	recvPieceMinRatio     = 0.8
	memberMaxJoinGroupNum = 5
)

const (
	prefixMSK         = "msk_"
	prefixEncryptedSK = "enc_"
)

func candidateCount(totalN int) int {
	if totalN >= groupMemberMax {
		return groupMemberMax
	} else if totalN < groupMemberMin {
		return 0
	}
	return totalN
}

func candidateEnough(n int) bool {
	return n >= groupMemberMin
}

func pieceEnough(pieceNum, candidateNum int) bool {
	return pieceNum >= int(math.Ceil(float64(candidateNum)*recvPieceMinRatio))
}

type minerReader interface {
	GetLatestVerifyMiner(id groupsig.ID) *model.MinerDO
	GetCanJoinGroupMinersAt(h uint64) []*model.MinerDO
}

type currentMinerInfoReader interface {
	MInfo() *model.SelfMinerDO
}

type createRoutine struct {
	*createChecker
	packetSender types.GroupPacketSender
	store        *skStorage
}

// UpdateEra updates the era info base on current block header
func (routine *createRoutine) UpdateContext(bh *types.BlockHeader) {
	curEra := routine.currEra()
	sh := seedHeight(bh.Height)
	seedBH := routine.chain.QueryBlockHeaderByHeight(sh)
	if curEra.sameEra(sh, seedBH) {
		return
	}
	routine.ctx = newCreateContext(newEra(sh, seedBH))
	routine.selectCandidates()
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
	groups := routine.storeReader.GetAvailableGroupInfos(h)
	memJoinedCountMap := make(map[string]int)
	for _, g := range groups {
		for _, mem := range g.Members() {
			hex := common.ToHex(mem.ID())
			if c, ok := memJoinedCountMap[hex]; !ok {
				memJoinedCountMap[hex] = 1
			} else {
				memJoinedCountMap[hex] = c + 1
			}
		}
	}

	for _, verifier := range allVerifiers {
		if c, ok := memJoinedCountMap[verifier.ID.GetHexString()]; !ok || c < memberMaxJoinGroupNum {
			availCandidates = append(availCandidates, verifier)
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

func (routine *createRoutine) CheckAndSendEncryptedPiecePacket(bh *types.BlockHeader) error {
	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.encPieceRange.inRange(bh.Height) {
		return fmt.Errorf("height not in the encrypted-piece round")
	}
	mInfo := routine.currentMiner.MInfo()
	if !mInfo.CanJoinGroup() {
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
	routine.store.storeSeckey(prefixEncryptedSK, era.Seed(), encSk)

	// Send the piece packet
	err := routine.packetSender.SendEncryptedPiecePacket(packet)
	if err != nil {
		return fmt.Errorf("send packet error:%v", err)
	}
	routine.ctx.sentEncryptedPiecePacket = packet

	return nil
}

func (routine *createRoutine) CheckAndSendMpkPacket(bh *types.BlockHeader) error {
	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.mpkRange.inRange(bh.Height) {
		return fmt.Errorf("height not in the mpk round")
	}

	mInfo := routine.currentMiner.MInfo()
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

	msk, err := aggrSignSecKey(encryptedPackets, cands, mInfo)
	if err != nil {
		return fmt.Errorf("genearte msk error:%v", err)
	}
	routine.store.storeSeckey(prefixMSK, era.Seed(), *msk)

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

func (routine *createRoutine) CheckAndSendOriginPiecePacket(bh *types.BlockHeader) error {
	era := routine.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if !era.oriPieceRange.inRange(bh.Height) {
		return fmt.Errorf("height not in the encrypted-piece round")
	}
	mInfo := routine.currentMiner.MInfo()
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
	// Whether sent encrypted pieces
	if !routine.storeReader.HasSentEncryptedPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("didn't sent encrypted share piece")
	}
	// Has sent piece
	if routine.ctx.sentOriginPiecePacket != nil || routine.storeReader.HasSentOriginPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("has sent origin pieces")
	}

	encSk, ok := routine.store.getSeckey(prefixEncryptedSK, era.Seed())
	if !ok {
		return fmt.Errorf("has no encrypted seckey")
	}

	// Generate origin share piece
	sp := generateSharePiecePacket(mInfo, encSk, era.Seed(), routine.ctx.cands)
	packet := &originSharePiecePacket{sharePiecePacket: sp}

	// Send the piece packet
	err := routine.packetSender.SendOriginPiecePacket(packet)
	if err != nil {
		return fmt.Errorf("send packet error:%v", err)
	}
	routine.ctx.sentOriginPiecePacket = packet

	return nil
}
