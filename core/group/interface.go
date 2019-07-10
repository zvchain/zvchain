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
	"github.com/zvchain/zvchain/middleware/types"
)

type minerReader interface {
	MinerFrozen(accountDB types.AccountDB, miner common.Address, height uint64) (success bool, err error)
	MinerPenalty(accountDB types.AccountDB, penalty types.PunishmentMsg, height uint64) (success bool, err error)
}

type chainReader interface {
	Height() uint64
	QueryTopBlock() *types.BlockHeader
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader
	QueryBlockHeaderByHeight(height uint64) *types.BlockHeader
	HasBlock(hash common.Hash) bool
	HasHeight(height uint64) bool

	LatestStateDB() types.AccountDB
	MinerSk() string
	AddTransactionToPool(tx *types.Transaction) (bool, error)
	GetAccountDBByHeight(height uint64) (types.AccountDB, error)
}

// Round 1 tx data,implement common.EncryptedSharePiecePacket
type EncryptedSharePiecePacketImpl struct {
	SeedD    common.Hash `msgpack:"se"`           // Seed
	SenderD  []byte      `msgpack:"sr,omitempty"` // sender's address. will set from transaction source
	Pubkey0D []byte      `msgpack:"pb"`           // the gpk share of the miner
	PiecesD  []byte      `msgpack:"pi"`           // array of encrypted piece for every group member
}

func (e *EncryptedSharePiecePacketImpl) Seed() common.Hash {
	return e.SeedD
}

func (e *EncryptedSharePiecePacketImpl) Sender() []byte {
	return e.SenderD
}

func (e *EncryptedSharePiecePacketImpl) Pieces() []byte {
	return e.PiecesD
}

func (e *EncryptedSharePiecePacketImpl) Pubkey0() []byte {
	return e.PiecesD
}

// Round 2 tx data. implement interface types.MpkPacket
type MpkPacketImpl struct {
	SeedD   common.Hash `msgpack:"se"`           // Seed
	SenderD []byte      `msgpack:"sr,omitempty"` // sender's address
	MpkD    []byte      `msgpack:"mp"`           // mpk
	SignD   []byte      `msgpack:"si"`           // byte data of Seed signed by mpk
}

func (s *MpkPacketImpl) Seed() common.Hash {
	return s.SeedD
}

func (s *MpkPacketImpl) Sender() []byte {
	return s.SenderD
}

func (s *MpkPacketImpl) Mpk() []byte {
	return s.MpkD
}

func (s *MpkPacketImpl) Sign() []byte {
	return s.SignD
}

// OriginSharePiecePacket implements types.OriginSharePiecePacket.
type OriginSharePiecePacketImpl struct {
	SeedD      common.Hash `msgpack:"se"`           // Seed
	SenderD    []byte      `msgpack:"sr,omitempty"` // sender's address. will set from transaction source
	EncSeckeyD []byte      `msgpack:"es"`           // the gpk share of the miner
	PiecesD    []byte      `msgpack:"pi"`           // array of origin piece for every group member
}

func (e *OriginSharePiecePacketImpl) Seed() common.Hash {
	return e.SeedD
}

func (e *OriginSharePiecePacketImpl) Sender() []byte {
	return e.SenderD
}

func (e *OriginSharePiecePacketImpl) Pieces() []byte {
	return e.PiecesD
}

func (e *OriginSharePiecePacketImpl) EncSeckey() []byte {
	return e.EncSeckeyD
}

type FullPacketImpl struct {
	mpks   []types.MpkPacket
	pieces []types.EncryptedSharePiecePacket
}

func (s *FullPacketImpl) Mpks() []types.MpkPacket {
	return s.mpks
}

func (s *FullPacketImpl) Pieces() []types.EncryptedSharePiecePacket {
	return s.pieces
}

type Group struct {
	HeaderD  *GroupHeader
	MembersD []*Member
	Height   uint64 // the Height of group created
}

func (g *Group) Header() types.GroupHeaderI {
	return g.HeaderD
}

func (g *Group) Members() []types.MemberI {
	rs := make([]types.MemberI, 0, len(g.MembersD))
	for _, v := range g.MembersD {
		rs = append(rs, v)
	}
	return rs
}

type Member struct {
	Id []byte
	Pk []byte
}

func (m *Member) ID() []byte {
	return m.Id
}

func (m *Member) PK() []byte {
	return m.Pk
}

type GroupHeader struct {
	SeedD          common.Hash
	WorkHeightD    uint64
	DismissHeightD uint64
	PublicKeyD     []byte
	ThresholdD     uint32
}

func (g *GroupHeader) Seed() common.Hash {
	return g.SeedD
}

func (g *GroupHeader) WorkHeight() uint64 {
	return g.WorkHeightD
}

func (g *GroupHeader) DismissHeight() uint64 {
	return g.DismissHeightD
}

func (g *GroupHeader) PublicKey() []byte {
	return g.PublicKeyD
}
func (g *GroupHeader) Threshold() uint32 {
	return g.ThresholdD
}

func newGroup(i types.GroupI, height uint64) *Group {
	header := &GroupHeader{i.Header().Seed(),
		i.Header().WorkHeight(),
		i.Header().DismissHeight(),
		i.Header().PublicKey(),
		i.Header().Threshold()}
	members := make([]*Member, 0)
	for _, m := range i.Members() {
		mem := &Member{m.ID(), m.PK()}
		members = append(members, mem)
	}
	return &Group{header, members, height}
}
