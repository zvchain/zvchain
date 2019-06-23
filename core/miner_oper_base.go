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
	prefixStakeFrom           = []byte("from")
	prefixStakeTo             = []byte("to")
	prefixPoolProposal        = []byte("p")
	prefixPoolVerifier        = []byte("v")
	keyPoolProposalTotalStake = []byte("totalstake")
)

var (
	minerPoolAddr             = common.BigToAddress(big.NewInt(1)) // The Address storing total stakes of proposal role, and (address,stake) paires of verify roles
	HeavyDBAddress            = common.BigToAddress(big.NewInt(2))
	MinerCountDBAddress       = common.BigToAddress(big.NewInt(3))
	MinerStakeDetailDBAddress = common.BigToAddress(big.NewInt(4))
)

type stakeDetail struct {
	Value  uint64
	Height uint64
}

func tx2Miner(tx *types.Transaction) (*types.Miner, error) {
	data := common.FromHex(string(tx.Data))
	var miner = new(types.Miner)
	err := msgpack.Unmarshal(data, miner)
	if err != nil {
		return nil, err
	}
	miner.ID = tx.Target.Bytes()
	return miner, nil
}

func getDetailKey(prefix []byte, address common.Address, typ byte, status types.StakeStatus) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	buf.WriteByte(typ)
	buf.WriteByte(byte(status))
	return buf.Bytes()
}

// mergeMinerInfo merge the new miner info to the old one.
// The existing public keys and id will be replaced by the new if set
func mergeMinerInfo(old *types.Miner, nm *types.Miner) *types.Miner {
	old.Stake += nm.Stake
	if len(nm.ID) > 0 {
		old.ID = nm.ID
	}
	if len(nm.PublicKey) > 0 {
		old.PublicKey = nm.PublicKey
	}
	if len(nm.VrfPublicKey) > 0 {
		old.VrfPublicKey = nm.VrfPublicKey
	}

	return old
}

// checkCanActivate if status can be set to types.MinerStatusNormal
func checkCanActivate(miner *types.Miner) bool {
	// If the verifier stake up to the bound, then activate the miner
	// todo how to avtivate the miner ? manual or auto ?
	// todo check miner pks completed
	return miner.Stake >= common.VerifyStake
}

func checkCanInActivate(miner *types.Miner) bool {
	return miner.Stake < common.VerifyStake
}

func getMinerKey(typ byte) []byte {
	buf := bytes.NewBuffer(prefixMiner)
	buf.WriteByte(typ)
	return buf.Bytes()
}

func getPoolKey(prefix []byte, address common.Address) []byte {
	buf := bytes.NewBuffer(prefix)
	buf.Write(address.Bytes())
	return buf.Bytes()
}

func (op *baseOperation) addToPool(address common.Address, typ byte, stake uint64) {
	key := []byte{}
	if typ == types.MinerTypeProposal {
		key = getPoolKey(prefixPoolProposal, address)
		totalStakeBytes := op.accountdb.GetData(minerPoolAddr, keyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		oldStake := uint64(0)
		oldStakeBytes := op.accountdb.GetData(minerPoolAddr, key)
		if len(oldStakeBytes) > 0 {
			oldStake = common.ByteToUInt64(oldStakeBytes)
		}
		op.accountdb.SetData(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(stake+totalStake-oldStake))
	} else if typ == types.MinerTypeVerify {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.accountdb.SetData(minerPoolAddr, key, common.Uint64ToByte(stake))
}

func (op *baseOperation) removeFromPool(address common.Address, typ byte, stake uint64) {
	key := []byte{}
	if typ == types.MinerTypeProposal {
		key = getPoolKey(prefixPoolProposal, address)
		totalStakeBytes := op.accountdb.GetData(minerPoolAddr, keyPoolProposalTotalStake)
		totalStake := uint64(0)
		if len(totalStakeBytes) > 0 {
			totalStake = common.ByteToUInt64(totalStakeBytes)
		}
		if totalStake < stake {
			panic(fmt.Errorf("totalStake less than stake: %v %v", totalStake, stake))
		}
		oldStake := uint64(0)
		oldStakeBytes := op.accountdb.GetData(minerPoolAddr, key)
		if len(oldStakeBytes) > 0 {
			oldStake = common.ByteToUInt64(oldStakeBytes)
		}
		op.accountdb.SetData(minerPoolAddr, keyPoolProposalTotalStake, common.Uint64ToByte(totalStake-oldStake))
	} else if typ == types.MinerTypeVerify {
		key = getPoolKey(prefixPoolVerifier, address)

	}
	op.accountdb.RemoveData(minerPoolAddr, key)
}

func (op *baseOperation) getDetail(address common.Address, detailKey []byte) (*stakeDetail, error) {
	data := op.accountdb.GetData(address, detailKey)
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
	op.accountdb.SetData(address, detailKey, bs)
	return nil
}

func (op *baseOperation) removeDetail(address common.Address, detailKey []byte) {
	op.accountdb.RemoveData(address, detailKey)
}

func (op *baseOperation) getMiner(address common.Address, typ byte) (*types.Miner, error) {
	data := op.accountdb.GetData(address, getMinerKey(typ))
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
	op.accountdb.SetData(common.BytesToAddress(miner.ID), getMinerKey(miner.Type), bs)
	return nil
}
