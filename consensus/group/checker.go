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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
)

type createChecker struct {
	currentMiner currentMinerInfoReader
	chain        core.BlockChain
	ctx          *createContext
	storeReader  types.GroupStoreReader
	minerReader  minerReader
}

type createContext struct {
	era                *era
	sentEncryptedPiece types.EncryptedSenderPiecePacket
	cands              candidateSelector
}

func newCreateContext(era *era) *createContext {
	return &createContext{era: era}
}

func (checker *createChecker) currEra() *era {
	return checker.ctx.era
}

func (checker *createChecker) sameSeed(h uint64, seed common.Hash) bool {
	return checker.currEra().sameEra(h, seed)
}

func (checker *createChecker) hasSentEncryptedPiece(id groupsig.ID) bool {
	return checker.storeReader.HasSentEncryptedPiecePacket(id.Serialize())
}

func (checker *createChecker) CheckEncryptedPiecePacket(packet types.EncryptedSenderPiecePacket, ctx types.CheckerContext) error {
	panic("implement me")
}

func (checker *createChecker) CheckMpkPacket(packet types.MpkPacket, ctx types.CheckerContext) error {
	panic("implement me")
}

func (checker *createChecker) CheckGroupCreateResult(ctx types.CheckerContext) (resultCode int, data interface{}, err error) {
	panic("implement me")
}

func (checker *createChecker) CheckOriginPiecePacket(packet types.OriginSenderPiecePacket, ctx types.CheckerContext) error {
	panic("implement me")
}

func (checker *createChecker) CheckGroupCreatePunishment(ctx types.CheckerContext) (types.PunishmentMsg, error) {
	panic("implement me")
}
