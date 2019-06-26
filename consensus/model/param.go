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
	"math"

	"github.com/zvchain/zvchain/common"
)

// defines some const params of the consensus engine
const (
	// MaxWaitBlockTime is group cast block maximum allowable time, it's 10s
	MaxGroupBlockTime int = 20

	// MaxWaitBlockTime is Waiting for the maximum time before broadcasting the block,it's 2s
	MaxWaitBlockTime int = 2

	// ConsensusVersion is consensus version
	ConsensusVersion = 1

	// MaxUnknownBlocks means the memory saves the largest block that cannot be chained
	// (the middle block is not received)
	MaxUnknownBlocks = 5

	// GroupInitMaxSeconds means there initialization must completed within 10 minutes,
	// otherwise the group fails. There is no longer an opportunity for initialization.
	GroupInitMaxSeconds = 60 * 60 * 24

	// MaxSlotSize means number of slots per round
	MaxSlotSize = 5

	// SSSSThreshold means value range 1-100
	SSSSThreshold int = 51

	// GroupMaxMembers means the maximum number of members in a group
	GroupMaxMembers int = 100

	// GroupMinMembers means the minimum number of members in a group
	GroupMinMembers int = 10

	// MinerMaxJoinedGroup means the maximum number of groups each miner joins
	MinerMaxJoinedGroup = 5

	// CandidatesMinRatio means the multiple of the smallest candidate relative to the number of group members
	CandidatesMinRatio = 1

	Epoch int = 5

	GroupCreateGap = Epoch * 2

	GroupWaitPongGap = GroupCreateGap + Epoch*2

	// GroupReadyGap means the group is ready (built group) with an interval of 6 epoch
	GroupReadyGap = GroupCreateGap + Epoch*6

	// GroupWorkGap after the group is ready, wait for the interval of the to work is 8 epoch
	GroupWorkGap = GroupCreateGap + Epoch*8

	// GroupWorkDuration the group work cycle is 100 epoch
	GroupWorkDuration   = Epoch * 100
	GroupCreateInterval = Epoch * 10
)

// ConsensusParam defines all the params of the consensus engine
type ConsensusParam struct {
	GroupMemberMax      int // limit the maximum number of one group
	GroupMemberMin      int // limit the minimum number of one group
	MaxQN               int // limit the max qn of one block
	SSSSThreshold       int // the bls threshold for consensus
	MaxGroupCastTime    int // the time window length of the block consensus in seconds
	MaxWaitBlockTime    int // the waiting seconds for waiting the more weight block to broadcast
	MaxFutureBlock      int //
	GroupInitMaxSeconds int
	Epoch               uint64 // epoch length denoted by height
	CreateGroupInterval uint64 // group create routine triggered interval denoted by height
	MinerMaxJoinGroup   int    // limit the maximum group counts one node can joined in
	CandidatesMinRatio  int    // The lowest ratio of the total number of miners' pools to the number of group members
	GroupReadyGap       uint64
	GroupWorkGap        uint64
	GroupworkDuration   uint64
	GroupCreateGap      uint64
	GroupWaitPongGap    uint64
	PotentialProposal   int // Potential proposer count

	MaxSlotSize int
}

var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMemberMax:      cc.GetInt("group_member_max", GroupMaxMembers),
		GroupMemberMin:      cc.GetInt("group_member_min", GroupMinMembers),
		SSSSThreshold:       SSSSThreshold,
		MaxWaitBlockTime:    cc.GetInt("max_wait_block_time", MaxWaitBlockTime),
		MaxGroupCastTime:    cc.GetInt("max_group_cast_time", MaxGroupBlockTime),
		MaxQN:               5,
		MaxFutureBlock:      MaxUnknownBlocks,
		GroupInitMaxSeconds: GroupInitMaxSeconds,
		Epoch:               uint64(cc.GetInt("Epoch", Epoch)),
		MinerMaxJoinGroup:   cc.GetInt("miner_max_join_group", MinerMaxJoinedGroup),
		CandidatesMinRatio:  cc.GetInt("candidates_min_ratio", CandidatesMinRatio),
		GroupReadyGap:       uint64(cc.GetInt("group_ready_gap", GroupReadyGap)),
		GroupWorkGap:        uint64(cc.GetInt("group_cast_qualify_gap", GroupWorkGap)),
		GroupworkDuration:   uint64(cc.GetInt("group_cast_duration", GroupWorkDuration)),
		PotentialProposal:   10,
		CreateGroupInterval: uint64(GroupCreateInterval),
		GroupCreateGap:      uint64(GroupCreateGap),
		GroupWaitPongGap:    uint64(GroupWaitPongGap),
		MaxSlotSize:         MaxSlotSize,
	}
}

func (p *ConsensusParam) GetGroupK(max int) int {
	return int(math.Ceil(float64(max*p.SSSSThreshold) / 100))
}

func (p *ConsensusParam) IsGroupMemberCountLegal(cnt int) bool {
	return p.GroupMemberMin <= cnt && cnt <= p.GroupMemberMax
}
func (p *ConsensusParam) CreateGroupMinCandidates() int {
	return p.GroupMemberMin * p.CandidatesMinRatio
}

func (p *ConsensusParam) CreateGroupMemberCount(availCandidates int) int {
	cnt := int(math.Ceil(float64(availCandidates / p.CandidatesMinRatio)))
	if cnt > p.GroupMemberMax {
		cnt = p.GroupMemberMax
	} else if cnt < p.GroupMemberMin {
		cnt = 0
	}
	return cnt
}
