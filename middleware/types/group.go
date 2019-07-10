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

type SenderI interface {
	Sender() []byte
}

// SharePiecePacket contains share piece  data generated during the group-create routine
type SharePiecePacket interface {
	SeedI
	SenderI
	Pieces() []byte // Encrypted pieces data
}

// EncryptedSharePiecePacket is encrypted share piece data and only the corresponding receiver can decrypt it
type EncryptedSharePiecePacket interface {
	SharePiecePacket
	Pubkey0() []byte // Initial Pubkey
}

// OriginSharePiecePacket is the origin share piece data which is not encrypted
type OriginSharePiecePacket interface {
	SharePiecePacket
	EncSeckey() []byte
}

// MpkPacket contains the signature pubkey of the group which is aggregated from the share pieces received
type MpkPacket interface {
	SeedI
	SenderI
	Mpk() []byte  // Pubkey aggregated
	Sign() []byte // Signature to the seed signed by the Mpk
}

// MemberI is the group member interface
type MemberI interface {
	ID() []byte
	PK() []byte
}

type CreateResultCode int

const (
	CreateResultSuccess  CreateResultCode = iota // Group create success
	CreateResultMarkEvil                         // Someone cheat, and mark the origin pieces required
	CreateResultFail                             // Error occurs
)

// GroupI is the group info interface
type GroupI interface {
	Header() GroupHeaderI
	Members() []MemberI
}

// CreateResult is the group-create result presentation
type CreateResult interface {
	Code() CreateResultCode // Seen in the CreateResultCode enum above
	GroupInfo() GroupI      // Return new groupInfo if create success
	FrozenMiners() [][]byte // Determines miners to be froze
	Err() error             // Error occurs
}

// GroupHeaderI is the group header info interface
type GroupHeaderI interface {
	SeedI
	WorkHeight() uint64
	DismissHeight() uint64
	PublicKey() []byte
	Threshold() uint32
}

// PunishmentMsg is the punishment message when someone cheats
type PunishmentMsg interface {
	PenaltyTarget() [][]byte // Determines miners to be punished
	RewardTarget() [][]byte  // Determines miners to be rewarded
}

type CheckerContext interface {
	Height() uint64
}

// GroupCreateChecker provides function to check if the group-create related packets are legal
type GroupCreateChecker interface {

	// CheckEncryptedPiecePacket checks the encrypted share piece packet
	CheckEncryptedPiecePacket(packet EncryptedSharePiecePacket, ctx CheckerContext) error

	// CheckMpkPacket checks if the mpk packet legal
	CheckMpkPacket(packet MpkPacket, ctx CheckerContext) error

	// CheckGroupCreateResult checks if the group-create is success
	CheckGroupCreateResult(ctx CheckerContext) CreateResult

	// CheckOriginPiecePacket checks the origin share pieces packet is legal
	CheckOriginPiecePacket(packet OriginSharePiecePacket, ctx CheckerContext) error

	// CheckGroupCreatePunishment determines miners to be punished or rewarded
	CheckGroupCreatePunishment(ctx CheckerContext) (PunishmentMsg, error)
}

// GroupStoreReader provides function to access the data generated during the routine or group info
type GroupStoreReader interface {

	// GetEncryptedPiecePackets Get the encrypted share packet of the given seed
	GetEncryptedPiecePackets(seed SeedI) ([]EncryptedSharePiecePacket, error)

	// HasSentEncryptedPiecePacket checks if the given sender has sent the packet yet
	HasSentEncryptedPiecePacket(sender []byte, seed SeedI) bool

	// HasSentMpkPacket checks if the given sender has sent the packet yet
	HasSentMpkPacket(sender []byte, seed SeedI) bool

	// GetMpkPackets Get the mpk packet of the given seed
	GetMpkPackets(seed SeedI) ([]MpkPacket, error)

	// IsOriginPieceRequired check if origin piece is required of the given seed
	IsOriginPieceRequired(seed SeedI) bool

	// GetOriginPiecePackets returns the origin share piece data of the given seed
	GetOriginPiecePackets(seed SeedI) ([]OriginSharePiecePacket, error)

	// HasSentOriginPiecePacket checks if the given sender has sent the packet yet
	HasSentOriginPiecePacket(sender []byte, seed SeedI) bool

	// GetAvailableGroupSeeds gets available groups' seed at the given height
	GetAvailableGroupSeeds(height uint64) []SeedI

	// GetGroupBySeed returns the group info of the given seed
	GetGroupBySeed(seedHash common.Hash) GroupI

	// GetGroupHeaderBySeed returns the group header info of the given seed
	GetGroupHeaderBySeed(seedHash common.Hash) GroupHeaderI

	// MinerLiveGroupCount returns the lived-group number the given address participates in on the given height
	MinerLiveGroupCount(addr common.Address, height uint64) int
}

// GroupPacketSender provides functions for sending packets
type GroupPacketSender interface {

	// SendEncryptedPiecePacket sends the encrypted packet to the pool
	SendEncryptedPiecePacket(packet EncryptedSharePiecePacket) error

	// SendMpkPacket sends the mpk packet to the pool
	SendMpkPacket(packet MpkPacket) error

	// SendOriginPiecePacket sends the origin packet to the pool
	SendOriginPiecePacket(packet OriginSharePiecePacket) error
}
