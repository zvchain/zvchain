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
	"github.com/zvchain/zvchain/storage/account"
)

type chainReader interface {
	Height() uint64
	QueryTopBlock() *types.BlockHeader
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader
	QueryBlockHeaderByHeight(height uint64) *types.BlockHeader
	HasBlock(hash common.Hash) bool
	HasHeight(height uint64) bool

	LatestStateDB() *account.AccountDB
	MinerSk() string
	AddTransactionToPool(tx *types.Transaction) (bool, error)
}

// Round 1 tx data,implement common.EncryptedSharePiecePacket
type EncryptedSharePiecePacketImp struct {
	SeedD    common.Hash    `msgpack:"se"`           //当前轮的seed
	SenderD  []byte `msgpack:"sr,omitempty"` //发送者address. will set from transaction source
	Pubkey0D []byte         `msgpack:"pb"`           //the gpk share of the miner
	PiecesD  []byte  		`msgpack:"pi"`           //发送者对组内每个人的加密分片
}

func (e *EncryptedSharePiecePacketImp) Seed() common.Hash {
	return e.SeedD
}

func (e *EncryptedSharePiecePacketImp) Sender() []byte {
	return e.SenderD
}

func (e *EncryptedSharePiecePacketImp) Pieces() []byte {
	return e.PiecesD
}

func (e *EncryptedSharePiecePacketImp) Pubkey0() []byte {
	return e.PiecesD
}



//
//// Piece明文数据包接口
//type SenderPiece struct {
//	ReceiverD  []byte `msgpack:"re"` // piece的接收者
//	PieceDataD []byte `msgpack:"pd"` // Piece
//}
//
//func (s *SenderPiece) Receiver() []byte {
//	return s.ReceiverD
//}
//
//func (s *SenderPiece) PieceData() []byte {
//	return s.PieceDataD
//}

// 传给共识用
//type encryptedReceiverPiece struct {
//	sender []byte    // piece的发送者
//	pieceData []byte // Piece加密后的数据
//	pubkey0 []byte   //the gpk share of the miner
//}
//
//func (s *encryptedReceiverPiece) Sender() []byte {
//	return s.sender
//}
//func (s *encryptedReceiverPiece) PieceData() []byte {
//	return s.pieceData
//}
//func (s *encryptedReceiverPiece) Pubkey0() []byte {
//	return s.pubkey0
//}


// Round 2 tx data
// Mpk数据包接口, implement interface types.MpkPacket
type MpkPacketImpl struct {
	SeedD   common.Hash    `msgpack:"se"`           //当前轮的seed
	SourceD common.Address `msgpack:"sr,omitempty"` //发送者address
	MpkD    []byte         `msgpack:"mp"`           // 聚合出来的签名公钥
	SignD   []byte         `msgpack:"si"`           // 用签名公钥对seed进行签名
}

func (s *MpkPacketImpl) Seed() common.Hash {
	return s.SeedD
}

func (s *MpkPacketImpl) Sender() []byte {
	return s.SourceD.Bytes()
}

func (s *MpkPacketImpl) Mpk() []byte {
	return s.MpkD
}

func (s *MpkPacketImpl) Sign() []byte {
	return s.SignD
}

// OriginSharePiecePacket implements types.OriginSharePiecePacket.
type OriginSharePiecePacketImpl struct {
	SeedD    common.Hash    `msgpack:"se"`           //当前轮的seed
	SenderD  []byte `msgpack:"sr,omitempty"` //发送者address. will set from transaction source
	EncSeckeyD []byte         `msgpack:"es"`           //the gpk share of the miner
	PiecesD  []byte  		`msgpack:"pi"`           //发送者对组内每个人的加密分片
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



//type FullPacket struct {
//	mpks []types.MpkPacket
//	pieces []types.EncryptedSenderPiecePacket
//}
//
//func newFullPacket(m []types.MpkPacket,p []types.EncryptedSenderPiecePacket) *FullPacket {
//	return &FullPacket{m,p}
//}
//
//func (s *FullPacket) Mpks() []types.MpkPacket {
//	//rs := make([]types.MpkPacket,0, len(s.mpks))
//	//for i := range s.mpks {
//	//	rs[i] = s.mpks[i]
//	//}
//	//return rs
//	return s.mpks
//}
//
//func (s *FullPacket) Pieces() []types.EncryptedSenderPiecePacket {
//	return s.pieces
//}
//
