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

package logical

import (
	"bytes"
	"math"
	"sync"

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

// GroupCreateChecker is responsible for legality verification
type GroupCreateChecker struct {
	processor      *Processor
	access         *MinerPoolReader
	createdHeights [50]uint64 // Identifies whether the group height has already been created
	curr           int
	lock           sync.RWMutex // CreateHeightGroups mutex to prevent repeated writes
}

func newGroupCreateChecker(proc *Processor) *GroupCreateChecker {
	return &GroupCreateChecker{
		processor: proc,
		access:    proc.MinerReader,
		curr:      0,
	}
}
func checkCreate(h uint64) bool {
	return h > 0 && h%model.Param.CreateGroupInterval == 0
}

func (gchecker *GroupCreateChecker) heightCreated(h uint64) bool {
	gchecker.lock.RLock()
	defer gchecker.lock.RUnlock()
	for _, height := range gchecker.createdHeights {
		if h == height {
			return true
		}
	}
	return false
}

func (gchecker *GroupCreateChecker) addHeightCreated(h uint64) {
	gchecker.lock.RLock()
	defer gchecker.lock.RUnlock()
	gchecker.createdHeights[gchecker.curr] = h
	gchecker.curr = (gchecker.curr + 1) % len(gchecker.createdHeights)
}

// selectKing just choose half of the people. Each person's weight is decremented in order
func (gchecker *GroupCreateChecker) selectKing(theBH *types.BlockHeader, group *StaticGroupInfo) (kings []groupsig.ID, isKing bool) {
	num := int(math.Ceil(float64(group.GetMemberCount() / 2)))
	if num < 1 {
		num = 1
	}

	rand := base.RandFromBytes(theBH.Random)

	isKing = false

	selectIndexs := rand.RandomPerm(group.GetMemberCount(), num)
	kings = make([]groupsig.ID, len(selectIndexs))
	for i, idx := range selectIndexs {
		kings[i] = group.GetMemberID(idx)
		if gchecker.processor.GetMinerID().IsEqual(kings[i]) {
			isKing = true
		}
	}

	newBizLog("selectKing").info("king index=%v, ids=%v, isKing %v", selectIndexs, kings, isKing)
	return
}

func (gchecker *GroupCreateChecker) availableGroupsAt(h uint64) []*types.Group {
	iter := gchecker.processor.GroupChain.NewIterator()
	gs := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.DismissHeight > h {
			gs = append(gs, g)
		} else {
			genesis := gchecker.processor.GroupChain.GetGroupByHeight(0)
			gs = append(gs, genesis)
			break
		}
	}
	return gs
}

// selectCandidates randomly select a sufficient number of miners from the miners' pool as new group candidates
func (gchecker *GroupCreateChecker) selectCandidates(theBH *types.BlockHeader) (enough bool, cands []groupsig.ID) {
	min := model.Param.CreateGroupMinCandidates()
	blog := newBizLog("selectCandidates")
	height := theBH.Height
	allCandidates := gchecker.access.getCanJoinGroupMinersAt(height)

	ids := make([]string, len(allCandidates))
	for idx, can := range allCandidates {
		ids[idx] = can.ID.ShortS()
	}
	blog.debug("=======allCandidates height %v, %v size %v", height, ids, len(allCandidates))
	if len(allCandidates) < min {
		return
	}
	groups := gchecker.availableGroupsAt(theBH.Height)

	blog.debug("available groupsize %v", len(groups))

	candidates := make([]model.MinerDO, 0)
	for _, cand := range allCandidates {
		joinedNum := 0
		for _, g := range groups {
			for _, mem := range g.Members {
				if bytes.Equal(mem, cand.ID.Serialize()) {
					joinedNum++
					break
				}
			}
		}
		if joinedNum < model.Param.MinerMaxJoinGroup {
			candidates = append(candidates, cand)
		}
	}
	num := len(candidates)

	selectNum := model.Param.CreateGroupMemberCount(num)
	if selectNum <= 0 {
		blog.warn("not enough candidates, got %v", len(candidates))
		return
	}

	rand := base.RandFromBytes(theBH.Random)
	seqs := rand.RandomPerm(num, selectNum)

	result := make([]groupsig.ID, len(seqs))
	for i, seq := range seqs {
		result[i] = candidates[seq].ID
	}

	str := ""
	for _, id := range result {
		str += id.ShortS() + ","
	}
	blog.info("=============selectCandidates %v", str)
	return true, result
}
