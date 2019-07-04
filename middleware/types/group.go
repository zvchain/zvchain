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

package types

import "github.com/zvchain/zvchain/common"

// SeedI is the seed block which the group-create routine based on
type SeedI interface {
	Seed() common.Hash
}

// Piece明文数据包接口
type SenderPiece interface {
	Receiver() []byte  // piece的接收者
	PieceData() []byte // Piece明文
}
type SenderPiecePacket interface {
	SeedI
	Sender() []byte        //发送者
	Pieces() []SenderPiece //发送者对组内每个人的明文分片
}

type EncryptedSenderPiecePacket interface {
	SenderPiecePacket
	Pubkey0() []byte
}

type EncryptedReceiverPiece interface {
	Sender() []byte    // piece的发送者
	PieceData() []byte // Piece加密后的数据
	Pubkey0() []byte   //piece的发送者的公钥
}

type OriginSenderPiecePacket SenderPiecePacket

// Mpk数据包接口
type MpkPacket interface {
	SeedI
	Sender() []byte //发送者
	Mpk() []byte    // 聚合出来的签名公钥
	Sign() []byte   // 用签名公钥对seed进行签名
}

// 建组数据接口
type FullPacket interface {
	Mpks() []MpkPacket
	Pieces() []EncryptedSenderPiecePacket
}

type MemberI interface {
	ID() []byte
	PK() []byte
}

// 组信息接口
type GroupI interface {
	Header() GroupHeaderI
	Members() []MemberI
}

// 组头部信息接口
type GroupHeaderI interface {
	SeedI
	Hash() common.Hash
	Height() uint64
	MemRoot() common.Hash
	WorkHeight() uint64
	DismissHeight() uint64
	Extends() string
	PublicKey() []byte
	Threshold() uint32
}

// 惩罚消息接口
type PunishmentMsg interface {
	PenaltyTarget() [][]byte //罚款矿工列表
	RewardTarget() [][]byte  // 奖励列表
	Value() uint64           //罚款金额
}

type CheckerContext interface {
	Height() uint64
}

// 链在执行相关交易时调用共识校验接口
type GroupCreateChecker interface {

	// 校验一个piece交易是否合法，如果合法，则返回该加密后的piece数据
	// 链需要把piece数据存储到db
	CheckEncryptedPiecePacket(packet EncryptedSenderPiecePacket, ctx CheckerContext) error

	// 校验一个mpk交易是否合法，如果合法，则返回mpk数据
	CheckMpkPacket(packet MpkPacket, ctx CheckerContext) error

	// 校验建组是否成功
	// 若建组成功，则返回数据
	CheckGroupCreateResult(ctx CheckerContext) (resultCode int, data interface{}, err error)

	// 检查origin piece
	CheckOriginPiecePacket(packet OriginSenderPiecePacket, ctx CheckerContext) error

	// 检查惩罚
	CheckGroupCreatePunishment(ctx CheckerContext) (PunishmentMsg, error)
}

// 建组数据读取接口
type GroupStoreReader interface {

	// 返回指定seed的piece数据
	//  共识通过此接口获取所有piece数据，生成自己的签名私钥和签名公钥
	GetEncryptedReceiverPiece(receiver []byte, seed SeedI) ([]EncryptedReceiverPiece, error)

	// 查询指定sender 和seed 是否有piece数据
	HasSentEncryptedPiecePacket(sender []byte) bool

	// 返回所有的建组数据
	// 共识在校验是否建组成时调用
	GetPieceAndMpkPackets(seed SeedI) (FullPacket, error)

	// 返回origin piece 是否需要的标志
	IsOriginPieceRequired(seed SeedI) bool

	// 获取所有明文分片
	GetAllOriginPiecePackets(seed SeedI) ([]OriginSenderPiecePacket, error)

	// Get available group infos at the given height
	GetAvailableGroupInfos(h uint64) []GroupI
}

// 负责建组相关消息转换成交易发送，共识不关注交易类型，只关注数据
type GroupPacketSender interface {

	// 发送加密piece分片
	SendEncryptedPiecePacket(packet EncryptedSenderPiecePacket) error

	//发送签名公钥包
	SendMpkPacket(packet MpkPacket) error

	// 发送明文piece包
	SendOriginPiecePacket(packet OriginSenderPiecePacket) error
}
