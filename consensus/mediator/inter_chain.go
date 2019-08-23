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
	"github.com/zvchain/zvchain/common/ed25519"
	"math"
	"math/big"

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/consensus/groupsig"
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

// GenerateGenesisInfo generate genesis group and pk info of members
func (helper *ConsensusHelperImpl) GenerateGenesisInfo() *types.GenesisInfo {
	return group.GenerateGenesis()
}

// VRFProve2Value convert the vrf prove to big int
func (helper *ConsensusHelperImpl) VRFProve2Value(prove []byte) *big.Int {
	if len(prove) != ed25519.ProveSize {
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

// VerifyBlockSign verify the blockheader: mainly verify the group signature
func (helper *ConsensusHelperImpl) VerifyBlockSign(bh *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlockSign(bh)
}

// VerifyRewardTransaction verify reward transaction
func (helper *ConsensusHelperImpl) VerifyRewardTransaction(tx *types.Transaction) (ok bool, err error) {
	return Proc.VerifyRewardTransaction(tx)
}

// EstimatePreHeight estimate pre block's height
func (helper *ConsensusHelperImpl) EstimatePreHeight(bh *types.BlockHeader) uint64 {
	height := bh.Height
	if height == 1 {
		return 0
	}
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime)))
}

func (helper *ConsensusHelperImpl) VerifyBlockHeaders(pre, bh *types.BlockHeader) (ok bool, err error) {
	return Proc.VerifyBlockHeaders(pre, bh)
}
