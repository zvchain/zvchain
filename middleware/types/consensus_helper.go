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

package types

import (
	"math/big"
)

// GenesisInfo define genesis group info
type GenesisInfo struct {
	Group  Group
	VrfPKs [][]byte
	Pks    [][]byte
}

// ConsensusHelper are consensus interface collection
type ConsensusHelper interface {

	// generate genesis group and pk info of members
	GenerateGenesisInfo() *GenesisInfo

	// convert the vrf prove to big int
	VRFProve2Value(prove []byte) *big.Int

	// calculate the blockheader's qn
	// it needs to be equal to the blockheader's totalQN - preHeader's totalQN
	CalculateQN(bh *BlockHeader) uint64

	// check the prove root hash for weight node when add block on chain
	CheckProveRoot(bh *BlockHeader) (bool, error)

	// check the new block
	// mainly verify the cast legality and the group signature
	VerifyNewBlock(bh *BlockHeader, preBH *BlockHeader) (bool, error)

	// verify the blockheader: mainly verify the group signature
	VerifyBlockHeader(bh *BlockHeader) (bool, error)

	// check group legality
	CheckGroup(g *Group) (bool, error)

	// verify reward transaction
	VerifyRewardTransaction(tx *Transaction) (bool, error)

	// estimate pre block's height
	EstimatePreHeight(bh *BlockHeader) uint64

}
