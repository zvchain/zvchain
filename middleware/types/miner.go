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

const (
	MinerStatusPrepare MinerStatus = 0 // Miner prepare status, maybe abort or not enough stake or without pks,cannot participated any mining process
	MinerStatusActive              = 1 // Miner is activated, can participated in mining
	MinerStatusFrozen              = 2 // Miner frozen by system, cannot participated any mining process
)

const (
	pkSize    = 32
	vrfPkSize = 32
)

// Miner is the miner info including public keys and pledges
type Miner struct {
	ID                 []byte
	PublicKey          []byte
	VrfPublicKey       []byte
	ApplyHeight        uint64
	Stake              uint64
	StatusUpdateHeight uint64
	Type               MinerType
	Status             MinerStatus
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

func (m *Miner) PksCompleted() bool {
	return len(m.PublicKey) == pkSize && len(m.VrfPublicKey) == vrfPkSize
}

func (m *Miner) UpdateStatus(status MinerStatus, height uint64) {
	m.StatusUpdateHeight = height
	m.Status = status
}

// StakeStatus indicates the stake status
type StakeStatus = byte

const (
	Staked      StakeStatus = iota // Normal status
	StakeFrozen                    // Frozen status
)

// StakeDetail expresses the stake detail
type StakeDetail struct {
	Source       common.Address
	Target       common.Address
	Value        uint64
	Status       StakeStatus
	FrozenHeight uint64
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
	if len(bs) != 2 && len(bs) != 66 {
		return nil, fmt.Errorf("length error")
	}
	version := bs[0]
	if version != PayloadVersion {
		return nil, fmt.Errorf("version error %v", version)
	}
	pks := &MinerPks{
		MType: MinerType(bs[1]),
	}
	if len(bs) == 66 {
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
