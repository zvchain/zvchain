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

package core

import (
	"bytes"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math/big"
)

var (
	prefixMiner               = []byte("minfo")
	prefixPoolProposal        = []byte("p")
	prefixPoolVerifier        = []byte("v")
	keyPoolProposalTotalStake = []byte("totalstake")
)

var (
	minerPoolAddr = common.BigToAddress(big.NewInt(1)) // The Address storing total stakes of each roles and addresses of all active nodes
)

type stakeDetail struct {
	Value  uint64 // Stake operation amount
	Height uint64 // Operation height
}

func getDetailKey(address common.Address, typ types.MinerType, status types.StakeStatus) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(address.Bytes())
	buf.WriteByte(byte(typ))
	buf.WriteByte(byte(status))
	return buf.Bytes()
}

func setPks(miner *types.Miner, pks *types.MinerPks) *types.Miner {
	if len(pks.Pk) > 0 {
		miner.PublicKey = pks.Pk
	}
	if len(pks.VrfPk) > 0 {
		miner.VrfPublicKey = pks.VrfPk
	}

	return miner
}

// checkCanActivate if status can be set to types.MinerStatusActive
func checkCanActivate(miner *types.Miner, height uint64) bool {
	// pks not completed
	if !miner.PksCompleted() {
		return false
	}
	// If the verifier stake up to the bound, then activate the miner
	// todo how to avtivate the miner ? manual or auto ?
	// todo check miner pks completed
	return miner.Stake >= common.VerifyStake
}

func checkCanInActivate(miner *types.Miner) bool {
	return miner.Stake < common.VerifyStake
}

func checkUpperBound(miner *types.Miner, height uint64) bool {
	return true
}

func checkLowerBound(miner *types.Miner, height uint64) bool {
	return true
}

func getMinerKey(typ types.MinerType) []byte {
	buf := bytes.NewBuffer(prefixMiner)
	buf.WriteByte(byte(typ))
	return buf.Bytes()
}

func getPoolKey(prefix []byte, address common.Address) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func (op *baseOperation) opProposalRole() bool {
	return types.IsProposalRole(op.minerType)
}
func (op *baseOperation) opVerifyRole() bool {
	return types.IsVerifyRole(op.minerType)
}

func (op *baseOperation) addToPool(address common.Address, addStake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		op.addProposalTotalStake(addStake)
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.SetDataSafe(minerPoolAddr, key, []byte{1})
}

func (op *baseOperation) addProposalTotalStake(addStake uint64) {
	totalStakeBytes := op.minerPool.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
	totalStake := uint64(0)
	if len(totalStakeBytes) > 0 {
		totalStake = common.ByteToUInt64(totalStakeBytes)
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(addStake+totalStake))
}

func (op *baseOperation) subProposalTotalStake(subStake uint64) {
	totalStakeBytes := op.minerPool.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
	totalStake := uint64(0)
	if len(totalStakeBytes) > 0 {
		totalStake = common.ByteToUInt64(totalStakeBytes)
	}
	if totalStake < subStake {
		panic("total stake less than sub stake")
	}
	op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-subStake))
}

func (op *baseOperation) removeFromPool(address common.Address, stake uint64) {
	var key []byte
	if op.opProposalRole() {
		key = getPoolKey(prefixPoolProposal, address)
		totalStakeBytes := op.minerPool.GetDataSafe(minerPoolAddr, keyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		if totalStake < stake {
			panic(fmt.Errorf("totalStake less than stake: %v %v", totalStake, stake))
		}
		op.minerPool.SetDataSafe(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-stake))
	} else if op.opVerifyRole() {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.minerPool.RemoveDataSafe(minerPoolAddr, key)
}

func (op *baseOperation) getDetail(address common.Address, detailKey []byte) (*stakeDetail, error) {
	data := op.accountDB.GetData(address, detailKey)
	if data != nil && len(data) > 0 {
		var detail stakeDetail
		err := msgpack.Unmarshal(data, &detail)
		if err != nil {
			return nil, err
		}
		return &detail, nil
	}
	return nil, fmt.Errorf("no data")
}

func (op *baseOperation) setDetail(address common.Address, detailKey []byte, sd *stakeDetail) error {
	bs, err := msgpack.Marshal(sd)
	if err != nil {
		return err
	}
	op.accountDB.SetData(address, detailKey, bs)
	return nil
}

func (op *baseOperation) removeDetail(address common.Address, detailKey []byte) {
	op.accountDB.RemoveData(address, detailKey)
}

func (op *baseOperation) getMiner(address common.Address) (*types.Miner, error) {
	data := op.accountDB.GetData(address, getMinerKey(op.minerType))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		err := msgpack.Unmarshal(data, &miner)
		if err != nil {
			return nil, err
		}
		return &miner, nil
	}
	return nil, fmt.Errorf("no data")
}

func (op *baseOperation) setMiner(miner *types.Miner) error {
	bs, err := msgpack.Marshal(miner)
	if err != nil {
		return err
	}
	op.accountDB.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)
	return nil
}
