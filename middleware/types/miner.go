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
	MinerNormal NodeIdentity 	   = 0 // this is normal miner node,cannot be stake by others and stake to others
	MinerGuard                     = 1 // this miner is gurad miner node,cannot be stake by others
	MinerPool                      = 2 // this is miner pool node,can stake by others
	InValidMinerPool			   = 3 // this is invalid miner pool.only can be reduce before meeting the minimum number of votes
)

const (
	pkSize    = 128
	vrfPkSize = 32
)

// Miner is the miner info including public keys and pledges
type Miner struct {
	ID                   []byte
	PublicKey            []byte
	VrfPublicKey         []byte
	ApplyHeight          uint64
	Stake                uint64
	StatusUpdateHeight   uint64
	Type                 MinerType
	Status               MinerStatus
	Identity	         NodeIdentity
	IdentityUpdateHeight uint64
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

func (m *Miner) IsMinerPool() bool {
	return m.Identity == MinerPool
}

func (m *Miner) IsInvalidMinerPool() bool {
	return m.Identity == InValidMinerPool
}

func (m *Miner) PksCompleted() bool {
	return len(m.PublicKey) == pkSize && len(m.VrfPublicKey) == vrfPkSize
}

func (m *Miner) UpdateStatus(status MinerStatus, height uint64) {
	m.StatusUpdateHeight = height
	m.Status = status
}

func (m *Miner) UpdateIdentity(identity NodeIdentity, height uint64) {
	m.IdentityUpdateHeight = height
	m.Identity = identity
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
	DisMissHeight uint64
}

type MinerPks struct {
	MType MinerType
	Pk    []byte
	VrfPk []byte
}

const PayloadVersion = 1

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
	totalLen := 2 + pkSize + vrfPkSize
	if len(bs) != 2 && len(bs) != totalLen {
		return nil, fmt.Errorf("length error")
	}
	version := bs[0]
	if version != PayloadVersion {
		return nil, fmt.Errorf("version error %v", version)
	}
	pks := &MinerPks{
		MType: MinerType(bs[1]),
	}
	if len(bs) == totalLen {
		pks.Pk = bs[2 : 2+pkSize]
		pks.VrfPk = bs[2+pkSize:]
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
