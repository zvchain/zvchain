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
	"bytes"
	"fmt"
	"math"
	"reflect"
	"sync"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

const doFreezeMissingRate = 0.5 //will do freeze directly if piece/mpk missing rate less than value in this round
const noFreezeDropRate = 0.5    //will not freeze miner if block drop rate bigger than value in this round

type createChecker struct {
	chain       types.BlockChain
	ctx         *createContext
	storeReader types.GroupStoreReader
	minerReader minerReader
	stat        *createStat
	lock        sync.RWMutex
}

func newCreateChecker(reader minerReader, chain types.BlockChain, store types.GroupStoreReader) *createChecker {
	return &createChecker{
		chain:       chain,
		storeReader: store,
		minerReader: reader,
		stat:        newCreateStat(),
	}
}

type createContext struct {
	era                      *era
	sentEncryptedPiecePacket types.EncryptedSharePiecePacket
	sentMpkPacket            types.MpkPacket
	sentOriginPiecePacket    types.OriginSharePiecePacket
	gInfo                    types.GroupI
	cands                    candidates
	selected                 bool
}

type candidates []*model.MinerDO

func (c candidates) has(id groupsig.ID) bool {
	return c.find(id) >= 0
}

func (c candidates) size() int {
	return len(c)
}

func (c candidates) get(id groupsig.ID) *model.MinerDO {
	idx := c.find(id)
	if idx >= 0 {
		return c[idx]
	}
	return nil
}

func (c candidates) find(id groupsig.ID) int {
	for i, m := range c {
		if m.ID.IsEqual(id) {
			return i
		}
	}
	return -1
}

func (c candidates) pubkeys() []groupsig.Pubkey {
	memPks := make([]groupsig.Pubkey, 0)
	for _, mem := range c {
		memPks = append(memPks, mem.PK)
	}
	return memPks
}

func (c candidates) ids() []groupsig.ID {
	ids := make([]groupsig.ID, 0)
	for _, mem := range c {
		ids = append(ids, mem.ID)
	}
	return ids
}

func (c candidates) threshold() int {
	return int(math.Ceil(float64(c.size()) * float64(threshold) / float64(100.0)))
}

func newCreateContext(era *era) *createContext {
	return &createContext{era: era}
}

func (checker *createChecker) currEra() *era {
	return checker.ctx.era
}

func (checker *createChecker) CheckEncryptedPiecePacket(packet types.EncryptedSharePiecePacket, ctx types.CheckerContext) error {
	checker.lock.RLock()
	defer checker.lock.RUnlock()

	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if era.Seed() != seed {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed().Hex(), seed.Hex())
	}
	if !era.encPieceRange.inRange(ctx.Height()) {
		return fmt.Errorf("not in the encrypted-piece round, curr %v, round %v", ctx.Height(), era.encPieceRange)
	}
	if !routine.shouldCreateGroup() {
		return fmt.Errorf("should not create group, candsize %v", checker.ctx.cands.size())
	}
	sender := groupsig.DeserializeID(packet.Sender())

	// Was selected
	if !checker.ctx.cands.has(sender) {
		return fmt.Errorf("current miner not selected:%v", sender.GetHexString())
	}

	minerInfo := checker.minerReader.GetLatestVerifyMiner(sender)
	if minerInfo == nil {
		return fmt.Errorf("miner info not exists:%v", sender.GetHexString())
	}
	if !minerInfo.CanJoinGroup() {
		return fmt.Errorf("miner cann't join group:%v", sender.GetHexString())
	}

	// Has sent piece
	if checker.storeReader.HasSentEncryptedPiecePacket(packet.Sender(), era) {
		return fmt.Errorf("has sent encrypted pieces")
	}
	logger.Debugf("receive encrypted share piece from %v at %v", common.ShortHex(common.ToHex(packet.Sender())), era.Seed())
	return nil
}

func (checker *createChecker) CheckMpkPacket(packet types.MpkPacket, ctx types.CheckerContext) error {
	checker.lock.RLock()
	defer checker.lock.RUnlock()

	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if seed != era.Seed() {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed().Hex(), seed.Hex())
	}
	if !era.mpkRange.inRange(ctx.Height()) {
		return fmt.Errorf("not in the mpk round, curr %v, round %v", ctx.Height(), era.encPieceRange)
	}
	if !routine.shouldCreateGroup() {
		return fmt.Errorf("should not create group, candsize %v", checker.ctx.cands.size())
	}
	cands := checker.ctx.cands

	sender := groupsig.DeserializeID(packet.Sender())
	// Was selected
	if !cands.has(sender) {
		return fmt.Errorf("miner not selected:%v", sender.GetHexString())
	}

	// Verify sig
	if !groupsig.VerifySig(groupsig.DeserializePubkeyBytes(packet.Mpk()), packet.Seed().Bytes(), *groupsig.DeserializeSign(packet.Sign())) {
		return fmt.Errorf("verify sign fail:%v", common.ToHex(packet.Sign()))
	}

	mInfo := checker.minerReader.GetLatestVerifyMiner(sender)
	if mInfo == nil {
		return fmt.Errorf("miner not exist:%v", sender.GetHexString())
	}
	if !mInfo.CanJoinGroup() {
		return fmt.Errorf("miner cann't join group:%v", sender.GetHexString())
	}

	// Has sent mpk
	if checker.storeReader.HasSentMpkPacket(packet.Sender(), era) {
		return fmt.Errorf("has sent mpk packet")
	}
	// Has sent share piece
	if !checker.storeReader.HasSentEncryptedPiecePacket(packet.Sender(), era) {
		return fmt.Errorf("didn't send encrypted piece")
	}
	logger.Debugf("receive mpk from %v at %v", common.ShortHex(common.ToHex(packet.Sender())), era.Seed())

	return nil
}

func (checker *createChecker) firstHeightOfRound(r *rRange, curr uint64) uint64 {
	firstBH := checker.chain.QueryBlockHeaderCeil(r.begin)
	// If no block is higher than r.begin, then current height will be the first of the given round
	if firstBH == nil {
		return curr
	}
	return firstBH.Height
}

func findSender(senderArray interface{}, sender []byte) (bool, types.SenderI) {
	value := reflect.ValueOf(senderArray)
	for i := 0; i < value.Len(); i++ {
		v := value.Index(i)
		senderI := v.Interface().(types.SenderI)
		if bytes.Equal(senderI.Sender(), sender) {
			return true, senderI
		}
	}
	return false, nil
}

func (checker *createChecker) shouldFreeze(rang *rRange, missingNum int, requiredNum int) bool {
	if float32(missingNum)/float32(requiredNum) < doFreezeMissingRate {
		return true
	}
	dropRate := checker.calculateDropRate(rang.begin, rang.end)
	if dropRate > noFreezeDropRate {
		return false
	}
	return true
}

func (checker *createChecker) calculateDropRate(startHeight uint64, endHeight uint64) float32 {
	if endHeight < startHeight {
		return 0
	}
	realCount := checker.chain.CountBlocksInRange(startHeight, endHeight)
	rate := float32(realCount) / float32(endHeight-startHeight+1)
	return 1 - rate
}

func (checker *createChecker) shouldCreateGroup() bool {
	return checker.ctx != nil && checker.ctx.cands.size() > 0
}

func (checker *createChecker) CheckGroupCreateResult(ctx types.CheckerContext) types.CreateResult {
	checker.lock.RLock()
	defer checker.lock.RUnlock()
	era := checker.currEra()

	var result *createResult
	defer func() {
		frozeMiners := make([]string, 0)
		if len(result.frozenMiners) > 0 {
			for _, m := range result.frozenMiners {
				frozeMiners = append(frozeMiners, common.ShortHex(m.GetHexString()))
			}
			logger.Debugf("froze miners: seedHeight=%v,curr=%v %v", era.seedHeight, ctx.Height(), frozeMiners)
		}

		switch result.code {
		case types.CreateResultSuccess:
			if result.gInfo != nil {
				g := result.gInfo
				logger.Debugf("group create success: seedHeight=%v, seed=%v, workHeight=%v, dismissHeight=%v, threshold=%v, memsize=%v, candsize=%v", era.seedHeight, g.header.Seed(), g.header.WorkHeight(), g.header.DismissHeight(), g.header.Threshold(), len(g.members), checker.ctx.cands.size())
			}
			checker.stat.markStatus(era.seedHeight, createStatusSuccess)
		case types.CreateResultIdle:
			return
		case types.CreateResultMarkEvil:
			checker.stat.markStatus(era.seedHeight, createStatusFail)
			logger.Debugf("group create fail, mark evil, seedHeight=%v %v %v", era.seedHeight, era.Seed(), result.err)
		case types.CreateResultFail:
			checker.stat.markStatus(era.seedHeight, createStatusFail)
			logger.Debugf("group create fail, seedHeight=%v, err=%v", era.seedHeight, result.err)
		}
		checker.stat.outCh <- struct{}{}
	}()

	if !era.seedExist() {
		result = idleCreateResult(fmt.Errorf("seed not exists:%v", era.seedHeight))
		return result
	}
	if !routine.shouldCreateGroup() {
		result = idleCreateResult(fmt.Errorf("should not create group, candsize %v", checker.ctx.cands.size()))
		return result
	}
	sh := seedHeight(ctx.Height())
	if sh != era.seedHeight {
		result = idleCreateResult(fmt.Errorf("seed height not equal:expect %v, infact %v", era.seedHeight, sh))
		return result
	}

	first := checker.firstHeightOfRound(era.oriPieceRange, ctx.Height())
	if !era.oriPieceRange.inRange(first) {
		result = idleCreateResult(fmt.Errorf("not in the origin piece round, curr %v, round %v, first %v", ctx.Height(), era.oriPieceRange, first))
		return result
	}
	if first != ctx.Height() {
		result = idleCreateResult(fmt.Errorf("not the first height of the origin piece round, curr %v, round %v, first %v", ctx.Height(), era.oriPieceRange, first))
		return result
	}
	cands := checker.ctx.cands

	piecePkt, err := checker.storeReader.GetEncryptedPiecePackets(era)
	if err != nil {
		result = errCreateResult(fmt.Errorf("get encrypted piece error:%v", err))
		return result
	}

	needFreezePiece := make([]groupsig.ID, 0)

	// Find those who didn't send encrypted share piece
	if checker.shouldFreeze(era.encPieceRange, len(piecePkt), cands.size()) {
		for _, mem := range cands {
			if ok, _ := findSender(piecePkt, mem.ID.Serialize()); !ok {
				needFreezePiece = append(needFreezePiece, mem.ID)
			}
		}
	}

	result = &createResult{seed: era.Seed()}

	// If not enough share piece, then freeze those who didn't send share piece only
	if !pieceEnough(len(piecePkt), cands.size()) {
		result.frozenMiners = needFreezePiece
		result.err = fmt.Errorf("receives not enough share piece:%v, candsize:%v", len(piecePkt), cands.size())
		result.code = types.CreateResultFail
		return result
	}

	mpkPkt, err := checker.storeReader.GetMpkPackets(era)
	if err != nil {
		return errCreateResult(fmt.Errorf("get mpks error:%v", err))
	}

	// Find those who sent the encrypted pieces and didn't send mpk
	needFreezeMpk := make([]groupsig.ID, 0)
	if checker.shouldFreeze(era.mpkRange, len(mpkPkt), len(piecePkt)) {
		for _, pkt := range piecePkt {
			if ok, _ := findSender(mpkPkt, pkt.Sender()); !ok {
				needFreezeMpk = append(needFreezeMpk, groupsig.DeserializeID(pkt.Sender()))
			}
		}
	}

	// All of those who didn't send share piece and those who sent this but not mpk will be frozen
	result.frozenMiners = append(needFreezePiece, needFreezeMpk...)

	// Not enough mpk
	if len(mpkPkt) < cands.threshold() {
		result.err = fmt.Errorf("receives not enough mpk:%v, threshold:%v", len(mpkPkt), cands.threshold())
		result.code = types.CreateResultFail
	} else { // Success or evil encountered

		for _, piece := range piecePkt {
			logger.Debugf("check create result, avail pieces: %v %v %v %v %v", era.Seed(), common.ShortHex(common.ToHex(piece.Sender())), piece.Seed(), common.ToHex(piece.Pubkey0()), common.ToHex(piece.Pieces()))
		}
		for _, mpk := range mpkPkt {
			logger.Debugf("check create result, mpk:%v %v %v %v", common.ShortHex(common.ToHex(mpk.Sender())), mpk.Seed(), common.ToHex(mpk.Mpk()), common.ToHex(mpk.Sign()))
		}

		gpk := *aggrGroupPubKey(piecePkt)
		logger.Debugf("check create result, gpk:%v", gpk.GetHexString())
		gSign := aggrGroupSign(mpkPkt)
		logger.Debugf("check create result, gSign:%v", gSign.GetHexString())

		// Aggregate sign fail, somebody must cheat!
		if !groupsig.VerifySig(gpk, era.Seed().Bytes(), *gSign) {
			result.err = fmt.Errorf("verify group sig fail")
			result.code = types.CreateResultMarkEvil
		} else {
			result.code = types.CreateResultSuccess
			result.gInfo = generateGroupInfo(mpkPkt, era, gpk, cands.threshold())
			checker.ctx.gInfo = result.gInfo
		}
	}

	return result
}

func (checker *createChecker) CheckOriginPiecePacket(packet types.OriginSharePiecePacket, ctx types.CheckerContext) error {
	checker.lock.RLock()
	defer checker.lock.RUnlock()

	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if era.Seed() != seed {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed(), seed)
	}
	if !era.oriPieceRange.inRange(ctx.Height()) {
		return fmt.Errorf("height not in the encrypted-piece round, curr %v, round %v", ctx.Height(), era.encPieceRange)
	}
	if !routine.shouldCreateGroup() {
		return fmt.Errorf("should not create group, candsize %v", checker.ctx.cands.size())
	}
	sender := groupsig.DeserializeID(packet.Sender())
	// Was selected
	if !checker.ctx.cands.has(sender) {
		return fmt.Errorf("miner not selected:%v", sender)
	}

	mInfo := checker.minerReader.GetLatestVerifyMiner(sender)
	if mInfo == nil {
		return fmt.Errorf("miner not exists:%v", sender)
	}
	if !mInfo.CanJoinGroup() {
		return fmt.Errorf("miner cann't join group")
	}
	id := mInfo.ID.Serialize()
	// Whether origin piece required
	if !checker.storeReader.IsOriginPieceRequired(era) {
		return fmt.Errorf("don't need origin pieces")
	}
	// Whether sent encrypted pieces
	if !checker.storeReader.HasSentEncryptedPiecePacket(id, era) {
		return fmt.Errorf("didn't sent encrypted share piece")
	}
	// Whether sent mpk packet
	if !checker.storeReader.HasSentMpkPacket(id, era) {
		return fmt.Errorf("didn't sent mpk packet")
	}
	// Has sent piece
	if checker.storeReader.HasSentOriginPiecePacket(id, era) {
		return fmt.Errorf("has sent origin pieces")
	}
	logger.Debugf("receive origin share piece from %v at %v", common.ShortHex(common.ToHex(packet.Sender())), era.Seed())

	return nil
}

func (checker *createChecker) CheckGroupCreatePunishment(ctx types.CheckerContext) (msg types.PunishmentMsg, err error) {
	checker.lock.RLock()
	defer checker.lock.RUnlock()

	needPunish := false
	defer func() {
		if needPunish && err != nil {
			logger.Error(err)
		}
	}()

	era := checker.currEra()
	if !era.seedExist() {
		err = fmt.Errorf("seed not exists:%v", era.seedHeight)
		return
	}

	sh := seedHeight(ctx.Height())
	if sh != era.seedHeight {
		err = fmt.Errorf("seed height not equal:expect %v, infact %v", era.seedHeight, sh)
		return
	}
	if !routine.shouldCreateGroup() {
		err = fmt.Errorf("should not create group, candsize %v", checker.ctx.cands.size())
		return
	}
	first := checker.firstHeightOfRound(era.endRange, ctx.Height())
	if !era.endRange.inRange(first) {
		err = fmt.Errorf("not in the end round, curr %v, round %v, first %v", ctx.Height(), era.endRange, first)
		return
	}
	if first != ctx.Height() {
		err = fmt.Errorf("not the first height of the end round, curr %v, round %v, first %v", ctx.Height(), era.endRange, first)
		return
	}

	cands := checker.ctx.cands

	// Whether origin piece required
	if !checker.storeReader.IsOriginPieceRequired(era) {
		err = fmt.Errorf("origin piece not required")
		return
	}
	needPunish = true

	originPacket, err := checker.storeReader.GetOriginPiecePackets(era)
	if err != nil {
		err = fmt.Errorf("get origin packet error:%v", err)
		return
	}

	piecePkt, err := checker.storeReader.GetEncryptedPiecePackets(era)
	if err != nil {
		err = fmt.Errorf("get encrypted piece error:%v", err)
		return
	}

	mpkPacket, err := checker.storeReader.GetMpkPackets(era)
	if err != nil {
		err = fmt.Errorf("get mpk packet error:%v", err)
		return
	}

	// Find those who sent mpk (and of course encrypted piece did) but not sent origin pieces.
	missOriPieceIds := make([][]byte, 0)
	for _, mpk := range mpkPacket {
		if ok, _ := findSender(originPacket, mpk.Sender()); !ok {
			missOriPieceIds = append(missOriPieceIds, mpk.Sender())
		}
	}

	wrongPiecesIds := make([][]byte, 0)
	// Find those who sent the wrong encrypted pieces
	for _, ori := range originPacket {
		// Must not happen
		if ok, enc := findSender(piecePkt, ori.Sender()); !ok {
			panic(fmt.Sprintf("cannot find enc packet of %v", common.ToHex(ori.Sender())))
		} else {
			sharePieces := deserializeSharePieces(ori.Pieces())
			if ok, err := groupsig.CheckSharePiecesValid(sharePieces, cands.ids(), cands.threshold()); err != nil || !ok {
				if err != nil {
					logger.Errorf("check evil error:%v %v", err, common.ShortHex(common.ToHex(ori.Sender())))
				}
				wrongPiecesIds = append(wrongPiecesIds, ori.Sender())
			}
			if evil, err := checkEvil(enc.(types.EncryptedSharePiecePacket).Pieces(), sharePieces, *groupsig.DeserializeSeckey(ori.EncSeckey()), cands.pubkeys()); evil || err != nil {
				if err != nil {
					logger.Errorf("check evil error:%v %v", err, common.ShortHex(common.ToHex(ori.Sender())))
				}
				wrongPiecesIds = append(wrongPiecesIds, ori.Sender())
			}
		}
	}

	wrongMpkIds := make([][]byte, 0)
	// If someone didn't send origin piece, then we can't decrypt the encrypted-share piece and so we can't find out those who
	// gave the wrong mpk
	if len(missOriPieceIds) == 0 {
		sks := make([]groupsig.Seckey, 0)
		for _, ori := range originPacket {
			sks = append(sks, *groupsig.DeserializeSeckey(ori.EncSeckey()))
		}
		// Find those who sent the wrong mpk
		for _, mpk := range mpkPacket {
			idx := cands.find(groupsig.DeserializeID(mpk.Sender()))
			if idx < 0 {
				panic(fmt.Sprintf("cannot find id:%v", common.ShortHex(common.ToHex(mpk.Sender()))))
			}

			msk, err := aggrSignSecKeyWithMyPK(piecePkt, idx, sks, cands[idx].PK)
			if err != nil {
				wrongMpkIds = append(wrongMpkIds, mpk.Sender())
				logger.Errorf("aggregate seckey error:%v %v", err, common.ShortHex(common.ToHex(mpk.Sender())))
			} else {
				pk := groupsig.NewPubkeyFromSeckey(*msk)
				if !bytes.Equal(pk.Serialize(), mpk.Mpk()) {
					wrongMpkIds = append(wrongMpkIds, mpk.Sender())
				}
			}

		}
	}

	// Take apart the penalty targets and reward targets
	penaltyTargets := make([][]byte, 0)
	penaltyTargets = append(penaltyTargets, missOriPieceIds...)
	penaltyTargets = append(penaltyTargets, wrongPiecesIds...)
	penaltyTargets = append(penaltyTargets, wrongMpkIds...)

	rewardTargets := make([][]byte, 0)
	for _, mpk := range mpkPacket {
		if ok, _ := findSender(penaltyTargets, mpk.Sender()); !ok {
			rewardTargets = append(rewardTargets, mpk.Sender())
		}
	}

	pm := &punishment{penaltyTargets: penaltyTargets, rewardTargets: rewardTargets}

	if len(penaltyTargets) > 0 {
		mems := make([]string, 0)
		for _, p := range penaltyTargets {
			mems = append(mems, common.ShortHex(common.ToHex(p)))
		}
		logger.Debugf("punishment at %v penalty target:%v", ctx.Height(), mems)
	}
	if len(rewardTargets) > 0 {
		mems := make([]string, 0)
		for _, p := range rewardTargets {
			mems = append(mems, common.ShortHex(common.ToHex(p)))
		}
		logger.Debugf("punishment at %v reward target:%v", ctx.Height(), mems)
	}
	return pm, nil
}
