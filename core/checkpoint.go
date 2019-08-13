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
	"container/heap"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"math"
)

const (
	groupThreshold = 0.67
	groupNumMin    = 3
)

var cpAddress = common.BytesToAddress([]byte("cp"))
var cpKey = []byte("cp")

type voteHeap []uint64

func (h voteHeap) Less(i, j int) bool {
	return h[i] < h[j]
}

func (h voteHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h voteHeap) Len() int {
	return len(h)
}

func (h *voteHeap) Pop() (v interface{}) {
	*h, v = (*h)[:h.Len()-1], (*h)[h.Len()-1]
	return
}

func (h *voteHeap) Push(v interface{}) {
	*h = append(*h, v.(uint64))
}

type activatedGroupReader interface {
	GetActivatedGroupsAt(h uint64) []types.GroupI
}

type blockQuerier interface {
	QueryTopBlock() *types.BlockHeader
	AccountDBAt(height uint64) (types.AccountDB, error)
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader
}

type cpChecker struct {
	groupReader activatedGroupReader
	querier     blockQuerier
	epoch       types.Epoch
	groups      []types.GroupI
	threshold   int
	allVotes    map[common.Hash]uint64
}

func newCpChecker(reader activatedGroupReader, querier blockQuerier) *cpChecker {
	return &cpChecker{
		groupReader: reader,
		querier:     querier,
	}
}

func cpGroupThreshold(groupNum int) int {
	return int(math.Ceil(float64(groupNum) * groupThreshold))
}

func (cp *cpChecker) init() {
	top := cp.querier.QueryTopBlock()
	cp.reset(types.EpochAt(top.Height))
	if len(cp.groups) < groupNumMin {
		Logger.Debugf("cp checker init, not enough groups at the epoch %v", cp.epoch.Start())
		return
	}

	// init votes by scanning blocks
	for bh := top; bh != nil && len(cp.allVotes) < cp.threshold; bh = cp.querier.QueryBlockHeaderByHash(bh.PreHash) {
		cp.allVotes[bh.Group] = bh.Height
	}
}

func (cp *cpChecker) checkPointOf(chainSlice []*types.Block) *types.BlockHeader {
	if len(chainSlice) == 0 {
		return nil
	}
	ep := types.EpochAt(chainSlice[len(chainSlice)-1].Header.Height)
	groups := cp.groupReader.GetActivatedGroupsAt(ep.Start())
	threshold := cpGroupThreshold(len(groups))
	votes := make(map[common.Hash]struct{})
	for i := len(chainSlice) - 1; i >= 0; i-- {
		if len(groups) < groupNumMin {
			continue
		}
		bh := chainSlice[i].Header
		currEP := types.EpochAt(bh.Height)
		if currEP.Start() == ep.Start() {
			votes[chainSlice[i].Header.Group] = struct{}{}
			// CP found
			if len(votes) >= threshold {
				return bh
			}
		} else { // CP not found in current epoch, keep finding in prev epoch
			ep = currEP
			groups = cp.groupReader.GetActivatedGroupsAt(ep.Start())
			threshold = cpGroupThreshold(len(groups))
			votes = make(map[common.Hash]struct{})
			votes[chainSlice[i].Header.Group] = struct{}{}
		}
	}
	return nil
}

func (cp *cpChecker) reset(ep types.Epoch) {
	cp.epoch = ep
	cp.groups = cp.groupReader.GetActivatedGroupsAt(ep.Start())
	cp.threshold = cpGroupThreshold(len(cp.groups))
	cp.allVotes = make(map[common.Hash]uint64)
}

func (cp *cpChecker) markCheckpoint(db types.AccountDB, height uint64) {
	db.SetData(cpAddress, cpKey, common.Uint64ToByte(height))
}

func (cp *cpChecker) checkAndUpdate(db types.AccountDB, bh *types.BlockHeader) {
	// different epoch
	if bh.Height < cp.epoch.Start() {
		cp.reset(cp.epoch.Prev())
	} else if bh.Height >= cp.epoch.End() {
		cp.reset(cp.epoch.Next())
	}

	if len(cp.groups) < groupNumMin {
		return
	}

	cp.allVotes[bh.Group] = bh.Height
	if len(cp.allVotes) < cp.threshold {
		return
	}
	h := make(voteHeap, cp.threshold)
	i := 0
	for _, voteHeight := range cp.allVotes {
		if i < cp.threshold {
			h[i] = voteHeight
			i++
		} else {
			break
		}
	}
	heap.Init(&h)

	i = 0
	for _, voteHeight := range cp.allVotes {
		if i >= cp.threshold {
			if h[0] < voteHeight {
				h[0] = voteHeight
				heap.Fix(&h, 0)
			}
		}
		i++
	}

	cp.markCheckpoint(db, h[0])
	Logger.Infof("cp updated at %v, cp is %v", bh.Height, h[0])

}

func (cp *cpChecker) checkpointAt(h uint64) uint64 {
	db, err := cp.querier.AccountDBAt(h)
	if err != nil {
		Logger.Errorf("get account db at %v error:%v", h, err)
		return 0
	}
	bs := db.GetData(cpAddress, cpKey)
	return common.ByteToUint64(bs)
}
