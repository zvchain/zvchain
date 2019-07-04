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
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/taslog"
	"math"
)

type createChecker struct {
	currentMiner currentMinerInfoReader
	chain        core.BlockChain
	ctx          *createContext
	storeReader  types.GroupStoreReader
	minerReader  minerReader
	logger       taslog.Logger
}

type createContext struct {
	era                      *era
	sentEncryptedPiecePacket types.EncryptedSharePiecePacket
	sentMpkPacket            types.MpkPacket
	sentOriginPiecePacket    types.OriginSharePiecePacket
	gInfo                    types.GroupI
	cands                    candidates
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
	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if era.Seed() != seed {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed().Hex(), seed.Hex())
	}
	if !era.encPieceRange.inRange(ctx.Height()) {
		return fmt.Errorf("height not in the encrypted-piece round")
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
	return nil
}

func (checker *createChecker) CheckMpkPacket(packet types.MpkPacket, ctx types.CheckerContext) error {
	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if seed != era.Seed() {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed().Hex(), seed.Hex())
	}
	if !era.mpkRange.inRange(ctx.Height()) {
		return fmt.Errorf("height not in the mpk round")
	}
	cands := checker.ctx.cands

	sender := groupsig.DeserializeID(packet.Sender())
	// Was selected
	if !cands.has(sender) {
		return fmt.Errorf("miner not selected:%v", sender.GetHexString())
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

	return nil
}

func (checker *createChecker) firstHeightOfRound(r *rRange) uint64 {
	firstBH := checker.chain.QueryBlockHeaderCeil(r.begin)
	if firstBH == nil {
		return 0
	}
	return firstBH.Height
}

func (checker *createChecker) CheckGroupCreateResult(ctx types.CheckerContext) types.CreateResult {
	era := checker.currEra()
	if !era.seedExist() {
		return errCreateResult(fmt.Errorf("seed not exists:%v", era.seedHeight))
	}

	sh := seedHeight(ctx.Height())
	if sh != era.seedHeight {
		return errCreateResult(fmt.Errorf("seed height not equal:expect %v, infact %v", era.seedHeight, sh))
	}
	first := checker.firstHeightOfRound(era.oriPieceRange)
	if era.oriPieceRange.inRange(first) {
		return errCreateResult(fmt.Errorf("not in the origin piece round"))
	}
	if first != ctx.Height() {
		return errCreateResult(fmt.Errorf("not the first height of the origin piece round"))
	}
	cands := checker.ctx.cands

	piecePkt, err := checker.storeReader.GetEncryptedPiecePackets(era)
	if err != nil {
		return errCreateResult(fmt.Errorf("get encrypted piece error"))
	}

	result := &createResult{}

	needFreeze := make([]groupsig.ID, 0)
	// Find those who didn't send encrypted share piece
	for _, mem := range cands {
		find := false
		for _, pkt := range piecePkt {
			if bytes.Equal(pkt.Sender(), mem.ID.Serialize()) {
				find = true
				break
			}
		}
		if !find {
			needFreeze = append(needFreeze, mem.ID)
		}
	}

	mpkPkt, err := checker.storeReader.GetMpkPackets(era)
	if err != nil {
		return errCreateResult(fmt.Errorf("get mpks error"))
	}

	availPieces := make([]types.EncryptedSharePiecePacket, 0)

	// Find those who sent the encrypted pieces and didn't send mpk
	for _, pkt := range piecePkt {
		find := false
		for _, mpk := range mpkPkt {
			if bytes.Equal(mpk.Sender(), pkt.Sender()) {
				find = true
				break
			}
		}
		if !find {
			needFreeze = append(needFreeze, groupsig.DeserializeID(pkt.Sender()))
		} else {
			availPieces = append(availPieces, pkt)
		}
	}
	// All of those who didn't send share piece and those who sent this but not mpk will be frozen
	result.frozenMiners = needFreeze
	// Not enough member count
	if !pieceEnough(len(availPieces), cands.size()) {
		result.err = fmt.Errorf("receives not enough available share piece(with mpk):%v", len(availPieces))
		result.code = types.CreateResultFail
	} else { // Success or evil encountered
		gpk := *aggrGroupPubKey(availPieces)
		gSign := aggrGroupSign(mpkPkt)
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
	seed := packet.Seed()
	era := checker.currEra()
	if !era.seedExist() {
		return fmt.Errorf("seed not exists:%v", era.seedHeight)
	}
	if era.Seed() != seed {
		return fmt.Errorf("seed not equal, expect %v infact %v", era.Seed().Hex(), seed.Hex())
	}
	if !era.oriPieceRange.inRange(ctx.Height()) {
		return fmt.Errorf("height not in the encrypted-piece round")
	}
	sender := groupsig.DeserializeID(packet.Sender())
	// Was selected
	if !checker.ctx.cands.has(sender) {
		return fmt.Errorf("miner not selected:%v", sender.GetHexString())
	}

	mInfo := checker.minerReader.GetLatestVerifyMiner(sender)
	if mInfo == nil {
		return fmt.Errorf("miner not exists:%v", sender.GetHexString())
	}
	if !mInfo.CanJoinGroup() {
		return fmt.Errorf("miner cann't join group")
	}

	// Whether origin piece required
	if !checker.storeReader.IsOriginPieceRequired(era) {
		return fmt.Errorf("don't need origin pieces")
	}
	// Whether sent encrypted pieces
	if !checker.storeReader.HasSentEncryptedPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("didn't sent encrypted share piece")
	}
	// Has sent piece
	if checker.storeReader.HasSentOriginPiecePacket(mInfo.ID.Serialize(), era) {
		return fmt.Errorf("has sent origin pieces")
	}
	return nil
}

func (checker *createChecker) CheckGroupCreatePunishment(ctx types.CheckerContext) (types.PunishmentMsg, error) {
	era := checker.currEra()
	if !era.seedExist() {
		return nil, fmt.Errorf("seed not exists:%v", era.seedHeight)
	}

	sh := seedHeight(ctx.Height())
	if sh != era.seedHeight {
		return nil, fmt.Errorf("seed height not equal:expect %v, infact %v", era.seedHeight, sh)
	}
	first := checker.firstHeightOfRound(era.endRange)
	if first != ctx.Height() {
		return nil, fmt.Errorf("not the first height of the origin piece round")
	}
	cands := checker.ctx.cands

	// Whether origin piece required
	if !checker.storeReader.IsOriginPieceRequired(era) {
		return nil, fmt.Errorf("origin piece not required")
	}

	originPacket, err := checker.storeReader.GetOriginPiecePackets(era)
	if err != nil {
		return nil, fmt.Errorf("get origin packet error:%v", err)
	}

	for _, oriPkt := range originPacket {
		oriPkt.
			encryptSharePieces()
		checkEvil()
	}

	piecePkt, err := checker.storeReader.GetEncryptedPiecePackets(era)
	if err != nil {
		return errCreateResult(fmt.Errorf("get encrypted piece error"))
	}
}
