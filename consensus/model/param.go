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

package model

import (
	"github.com/zvchain/zvchain/common"
)

// defines some const params of the consensus engine
const (
	// MaxWaitBlockTime is group cast block maximum allowable time, it's 10s
	MaxGroupBlockTime int = 10

	// MaxWaitBlockTime is Waiting for the maximum time before broadcasting the block,it's 2s
	MaxWaitBlockTime int = 2

	// MaxUnknownBlocks means the memory saves the largest block that cannot be chained
	// (the middle block is not received)
	MaxUnknownBlocks = 5

	// MaxSlotSize means number of slots per round
	MaxSlotSize = 5

	// GroupMaxMembers means the maximum number of members in a group
	GroupMaxMembers int = 100

	// GroupMinMembers means the minimum number of members in a group
	GroupMinMembers int = 10
)

// ConsensusParam defines all the params of the consensus engine
type ConsensusParam struct {
	GroupMemberMax    int    // limit the maximum number of one group
	GroupMemberMin    int    // limit the minimum number of one group
	MaxQN             int    // limit the max qn of one block
	MaxGroupCastTime  int    // the time window length of the block consensus in seconds
	MaxWaitBlockTime  int    // the waiting seconds for waiting the more weight block to broadcast
	MaxFutureBlock    int    //
	Epoch             uint64 // epoch length denoted by height
	PotentialProposal int    // Potential proposer count

	MaxSlotSize int
}

var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMemberMax:    cc.GetInt("group_member_max", GroupMaxMembers),
		GroupMemberMin:    cc.GetInt("group_member_min", GroupMinMembers),
		MaxWaitBlockTime:  cc.GetInt("max_wait_block_time", MaxWaitBlockTime),
		MaxGroupCastTime:  cc.GetInt("max_group_cast_time", MaxGroupBlockTime),
		MaxQN:             5,
		MaxFutureBlock:    MaxUnknownBlocks,
		PotentialProposal: 10,
		MaxSlotSize:       MaxSlotSize,
		Epoch:             5,
	}
}
