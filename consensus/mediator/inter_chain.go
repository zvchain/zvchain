//   Copyright (C) 2018 ZVChain
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

// Package mediator provides some functions for use in other modules
package mediator

import (
	"fmt"
	"math"
	"math/big"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/logical"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

// ConsensusHelperImpl implements ConsensusHelper interface.
// It provides functions for chain use
type ConsensusHelperImpl struct {
	ID groupsig.ID
}

func NewConsensusHelper(id groupsig.ID) types.ConsensusHelper {
	return &ConsensusHelperImpl{ID: id}
}

// ProposalBonus returns bonus for packing one bonus transaction
func (helper *ConsensusHelperImpl) ProposalBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.ProposalBonus)
}

// PackBonus returns bonus for packing one bonus transaction
func (helper *ConsensusHelperImpl) PackBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.PackBonus)
}

// GenerateGenesisInfo generate genesis group and pk info of members
func (helper *ConsensusHelperImpl) GenerateGenesisInfo() *types.GenesisInfo {
	return logical.GenerateGenesis()
}

// VRFProve2Value convert the vrf prove to big int
func (helper *ConsensusHelperImpl) VRFProve2Value(prove []byte) *big.Int {
	if len(prove) == 0 {
		return big.NewInt(0)
	}
	return base.VRFProof2hash(base.VRFProve(prove)).Big()
}

// CalculateQN calculates the blockheader's qn
// It needs to be equal to the blockheader's totalQN - preHeader's totalQN
func (helper *ConsensusHelperImpl) CalculateQN(bh *types.BlockHeader) uint64 {
	return Proc.CalcBlockHeaderQN(bh)
}

// CheckProveRoot check the prove root hash for weight node when add block on chain
func (helper *ConsensusHelperImpl) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	// No longer check when going up, only check at consensus
	return true, nil
}

// VerifyNewBlock check the new block.
// Mainly verify the cast legality and the group signature
func (helper *ConsensusHelperImpl) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlock(bh, preBH)
}

// VerifyBlockHeader verify the blockheader: mainly verify the group signature
func (helper *ConsensusHelperImpl) VerifyBlockHeader(bh *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlockHeader(bh)
}

// CheckGroup check group legality
func (helper *ConsensusHelperImpl) CheckGroup(g *types.Group) (ok bool, err error) {
	return Proc.VerifyGroup(g)
}

// VerifyBonusTransaction verify bonus transaction
func (helper *ConsensusHelperImpl) VerifyBonusTransaction(tx *types.Transaction) (ok bool, err error) {
	signBytes := tx.Sign
	if len(signBytes) < common.SignLength {
		return false, fmt.Errorf("not enough bytes for bonus signature, sign =%v", signBytes)
	}

	groupID, members, blockHash, value, err := Proc.MainChain.GetBonusManager().ParseBonusTransaction(tx)
	if err != nil {
		return false, fmt.Errorf("failed to parse bonus transaction, err =%s", err)
	}

	if !Proc.MainChain.HasBlock(blockHash) {
		return false, fmt.Errorf("chain does not have this block, block hash=%v", blockHash)
	}

	if model.Param.VerifyBonus / uint64(len(members)) != value {
		return false, fmt.Errorf("invalid verify bonus, value=%v", value)
	}

	group := Proc.GroupChain.GetGroupByID(groupID)
	if group == nil {
		return false, common.ErrGroupNil
	}
	for _,id := range(members) {
		if !group.MemberExist(id) {
			return false, fmt.Errorf("invalid group member,id=%v",  groupsig.DeserializeID(id).GetHexString())
		}
	}

	gpk := groupsig.DeserializePubkeyBytes(group.PubKey)
	gsign := groupsig.DeserializeSign(signBytes[0:33]) //size of groupsig == 33
	if !groupsig.VerifySig(gpk, tx.Hash.Bytes(), *gsign) {
		return false, fmt.Errorf("verify bonus sign fail, gsign=%v", gsign.GetHexString())
	}

	return true, nil
}

// EstimatePreHeight estimate pre block's height
func (helper *ConsensusHelperImpl) EstimatePreHeight(bh *types.BlockHeader) uint64 {
	height := bh.Height
	if height == 1 {
		return 0
	}
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime)))
}
