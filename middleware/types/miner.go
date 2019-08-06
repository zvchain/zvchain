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

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"math/big"
)

type MinerType byte

// defines the miner role type
const (
	MinerTypeVerify   MinerType = iota // Proposal role
	MinerTypeProposal                  // Verify role
)

type MinerStatus byte

type NodeIdentity byte

const (
	MinerStatusPrepare MinerStatus = 0 // Miner prepare status, maybe abort or not enough stake or without pks,cannot participated any mining process
	MinerStatusActive              = 1 // Miner is activated, can participated in mining
	MinerStatusFrozen              = 2 // Miner frozen by system, cannot participated any mining process
)

const (
	MinerGuard NodeIdentity 	   = 0 // this miner is gurad miner node,cannot be stake by others
	MinerNormal                    = 1 // this is normal miner node,cannot be stake by others and stake to others
	MinerPool                      = 2 // this is miner pool node,can stake by others
)

const (
	pkSize    = 128
	vrfPkSize = 32
	preFixLen  = 2
	fixLen  = 10  // 2 + 8
)

// Miner is the miner info including public keys and pledges
type Miner struct {
	ID                 []byte 		`json:"d"`
	PublicKey          []byte		`json:"p"`
	VrfPublicKey       []byte		`json:"v"`
	ApplyHeight        uint64		`json:"a"`
	Stake              uint64		`json:"stk"`
	StatusUpdateHeight uint64		`json:"suh"`
	Type               MinerType	`json:"tp"`
	Status             MinerStatus	`json:"sta"`
	Identity	       NodeIdentity `json:"ni"`
}

func (m *Miner) IsActive() bool {
	return m.Status == MinerStatusActive
}
func (m *Miner) IsFrozen() bool {
	return m.Status == MinerStatusFrozen
}
func (m *Miner) IsPrepare() bool {
	return m.Status == MinerStatusPrepare
}

func (m *Miner) IsGuard() bool {
	return m.Identity == MinerGuard
}

func (m *Miner) IsNormal() bool {
	return m.Identity == MinerNormal
}

func (m *Miner) MinerPool() bool {
	return m.Identity == MinerPool
}

func (m *Miner) PksCompleted() bool {
	return len(m.PublicKey) == pkSize && len(m.VrfPublicKey) == vrfPkSize
}

func (m *Miner) UpdateStatus(status MinerStatus, height uint64) {
	m.StatusUpdateHeight = height
	m.Status = status
}

func (m *Miner) IsProposalRole() bool {
	return IsProposalRole(m.Type)
}

func (m *Miner) IsVerifyRole() bool {
	return IsVerifyRole(m.Type)
}

// StakeStatus indicates the stake status
type StakeStatus = byte

const (
	Staked          StakeStatus = iota // Normal status
	StakeFrozen                        // Frozen status
	StakePunishment                    // Punishment status, can't never refund
)

// StakeDetail expresses the stake detail
type StakeDetail struct {
	Source       common.Address
	Target       common.Address
	Value        uint64
	Status       StakeStatus
	UpdateHeight uint64
	MType        MinerType
}

type MinerPks struct {
	MType 		MinerType
	Pk    		[]byte
	VrfPk 		[]byte
	AddHeight 	uint64
}

const PayloadVersion = 2

func (tx *Transaction) OpType() int8 {
	return tx.Type
}

func (tx *Transaction) Operator() *common.Address {
	return tx.Source
}

func (tx *Transaction) OpTarget() *common.Address {
	return tx.Target
}

func (tx *Transaction) Amount() *big.Int {
	if tx.Value == nil {
		return nil
	}
	return tx.Value.Value()
}

func (tx *Transaction) Payload() []byte {
	return tx.Data
}

// EncodePayload encodes pk and vrf pk into byte array storing in transaction data field
func EncodePayload(pks *MinerPks) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(PayloadVersion)
	buf.WriteByte(byte(pks.MType))
	buf.Write(common.UInt64ToByte(pks.AddHeight))
	pkLen := len(pks.Pk)
	vrfPkLen := len(pks.VrfPk)
	if pkLen == pkSize && vrfPkLen == vrfPkSize {
		buf.Write(pks.Pk)
		buf.Write(pks.VrfPk)
	}
	return buf.Bytes(), nil
}

// DecodePayload decodes the data field of the miner related transaction
func DecodePayload(bs []byte) (*MinerPks, error) {
	totalLen := fixLen +  pkSize + vrfPkSize
	if len(bs) != fixLen && len(bs) != totalLen {
		return nil, fmt.Errorf("length error")
	}
	version := bs[0]
	if version != PayloadVersion {
		return nil, fmt.Errorf("version error %v", version)
	}
	pks := &MinerPks{
		MType: MinerType(bs[1]),
		AddHeight:common.ByteToUInt64(bs[preFixLen:fixLen]),
	}
	if len(bs) == totalLen {
		pks.Pk = bs[fixLen : fixLen+pkSize]
		pks.VrfPk = bs[fixLen+pkSize:]
	}
	return pks, nil
}

func IsProposalRole(typ MinerType) bool {
	return typ == MinerTypeProposal
}
func IsVerifyRole(typ MinerType) bool {
	return typ == MinerTypeVerify
}

// CastRewardShare represents for the block generation reward
type CastRewardShare struct {
	ForBlockProposal   uint64
	ForBlockVerify     uint64
	ForRewardTxPacking uint64
	FeeForProposer     uint64
	FeeForVerifier     uint64
}

func (crs *CastRewardShare) TotalForVerifier() uint64 {
	return crs.FeeForVerifier + crs.ForBlockVerify
}
