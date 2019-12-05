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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/trie"
	"math"
	"sort"
)

const (
	groupThreshold  = 0.7
	groupNumMin     = 10
	cpMaxScanEpochs = 10 // max scan epoch when finding the check point
	cpBlockBuffer   = 20 // min blocks that a cp can occurs
)

var cpAddress = common.BytesToAddress([]byte("cp_votes"))
var cpVoteKey = []byte("votes")
var cpEpochKey = []byte("epoch")

type activatedGroupReader interface {
	GetActivatedGroupsAt(h uint64) []types.GroupI
}

type blockQuerier interface {
	Height() uint64
	AccountDBAt(height uint64) (types.AccountDB, error)
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader
	QueryBlockHeaderFloor(h uint64) *types.BlockHeader
	BatchGetBlockHeadersBetween(begin, end uint64) []*types.BlockHeader
}

type cpContext struct {
	epoch        types.Epoch
	groupIndexes map[common.Hash]int
	threshold    int
}

func newCpContext(ep types.Epoch, groups []types.GroupI) *cpContext {
	ctx := &cpContext{}
	ctx.epoch = ep
	ctx.groupIndexes = make(map[common.Hash]int)
	for i, g := range groups {
		ctx.groupIndexes[g.Header().Seed()] = i
	}
	ctx.threshold = cpGroupThreshold(ctx.groupSize())
	return ctx
}

func (ctx *cpContext) groupSize() int {
	return len(ctx.groupIndexes)
}
func (ctx *cpContext) groupIndex(g common.Hash) int {
	if v, ok := ctx.groupIndexes[g]; ok {
		return v
	}
	return -1
}

func (ctx *cpContext) groupsEnough() bool {
	return ctx.groupSize() >= groupNumMin
}

type cpChecker struct {
	groupReader activatedGroupReader
	querier     blockQuerier
	ctx         *cpContext
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

func (cp *cpChecker) reset(ep types.Epoch) {
	ctx := newCpContext(ep, cp.groupReader.GetActivatedGroupsAt(ep.Start()))
	cp.ctx = ctx
	Logger.Debugf("checkpoint reset epoch %v-%v, groupsize %v, threshold %v", ep.Start(), ep.End(), ctx.groupSize(), ctx.threshold)
}

func (cp *cpChecker) init() {
	cp.reset(types.EpochAt(cp.querier.Height()))
}

func (cp *cpChecker) checkPointOf(chainSlice []*types.Block) *types.BlockHeader {
	if len(chainSlice) == 0 {
		return nil
	}
	top := chainSlice[len(chainSlice)-1].Header
	if top.Height <= cpBlockBuffer {
		return nil
	}
	firstHeight := top.Height - cpBlockBuffer
	// backtrack cpBlockBuffer blocks to calculate the checkpoint
	for i := len(chainSlice) - 1; i >= 0; i-- {
		if chainSlice[i].Header.Height <= firstHeight {
			chainSlice = chainSlice[:i+1]
			break
		}
	}

	top = chainSlice[len(chainSlice)-1].Header

	ep := types.EpochAt(top.Height)
	ctx := newCpContext(ep, cp.groupReader.GetActivatedGroupsAt(ep.Start()))
	votes := make(map[common.Hash]struct{})

	for visit := len(chainSlice) - 1; visit >= 0; visit-- {
		bh := chainSlice[visit].Header
		if types.EpochAt(bh.Height).Equal(ep) {
			if !ctx.groupsEnough() {
				continue
			}
			votes[bh.Group] = struct{}{}
			// Threshold-group height found
			if len(votes) >= ctx.threshold {
				return bh
			}
		} else {
			ep = types.EpochAt(bh.Height)
			ctx = newCpContext(ep, cp.groupReader.GetActivatedGroupsAt(ep.Start()))
			votes = make(map[common.Hash]struct{})
			votes[bh.Group] = struct{}{}
		}
	}
	return nil
}

func (cp *cpChecker) getGroupVotes(db types.AccountDB) []uint16 {
	latestVoteBytes := db.GetData(cpAddress, cpVoteKey)
	votes := make([]uint16, 0)
	for i := 0; i < len(latestVoteBytes); i += 2 {
		votes = append(votes, common.ByteToUInt16(latestVoteBytes[i:i+2]))
	}
	return votes
}

func (cp *cpChecker) setGroupVotes(db types.AccountDB, votes []uint16) {
	buf := bytes.Buffer{}
	for _, v := range votes {
		buf.Write(common.UInt16ToByte(v))
	}
	db.SetData(cpAddress, cpVoteKey, buf.Bytes())
}
func (cp *cpChecker) setGroupEpoch(db types.AccountDB, ep types.Epoch) {
	db.SetData(cpAddress, cpEpochKey, common.Uint64ToByte(ep.Start()))
}
func (cp *cpChecker) getGroupEpoch(db types.AccountDB) types.Epoch {
	bs := db.GetData(cpAddress, cpEpochKey)
	return types.EpochAt(common.ByteToUint64(bs))
}

func (cp *cpChecker) updateVotes(db types.AccountDB, bh *types.BlockHeader) {
	dbGroupEpoch := cp.getGroupEpoch(db)
	var votes []uint16

	ep := types.EpochAt(bh.Height)
	// different epoch
	if ep.Start() >= dbGroupEpoch.Next().Start() {
		cp.reset(ep)
		votes = make([]uint16, cp.ctx.groupSize())
		cp.setGroupEpoch(db, ep)
	} else if ep.Equal(dbGroupEpoch) {
		votes = cp.getGroupVotes(db)
		if !ep.Equal(cp.ctx.epoch) {
			cp.reset(ep)
		}
	} else { // Shouldn't happen
		Logger.Panicf("group epoch is before block epoch:%v %v", dbGroupEpoch.Start(), bh.Height)
		return
	}

	gIndex := cp.ctx.groupIndex(bh.Group)
	if gIndex < 0 {
		Logger.Debugf("current groups:%v", cp.ctx.groupIndexes)
		Logger.Panicf("cannot find group %v at %v, epoch %v-%v", bh.Group, bh.Height, cp.ctx.epoch.Start(), cp.ctx.epoch.End())
		return
	}
	if len(votes) != cp.ctx.groupSize() {
		Logger.Panicf("vote size %v not equal to group size %v at %v", len(votes), cp.ctx.groupSize(), bh.Height)
		return
	}
	// vote height: offset from the epoch start
	// Add 1 for that 0 is as the invalid value
	votes[gIndex] = uint16(bh.Height-ep.Start()) + 1
	cp.setGroupVotes(db, votes)

	Logger.Debugf("cp group votes updated at %v, votes %v", bh.Height, votes)
}

func (cp *cpChecker) calcCheckpointByDB(db types.AccountDB, ctx *cpContext) (cpHeight uint64, found bool) {
	// Get the group epoch start with the given accountDB
	gEp := cp.getGroupEpoch(db)
	// If epoch of the given db not equal to current epoch, means that the whole current epoch was skipped
	if gEp.Equal(ctx.epoch) {
		votes := cp.getGroupVotes(db)

		validVotes := make([]int, 0)
		for _, v := range votes {
			if v > 0 {
				validVotes = append(validVotes, int(v))
			}
		}
		// cp found
		if len(validVotes) >= ctx.threshold {
			sort.Ints(validVotes)
			thresholdHeight := uint64(validVotes[len(validVotes)-ctx.threshold]) + ctx.epoch.Start() - 1
			return thresholdHeight, true
		}
	}
	return 0, false
}

func (cp *cpChecker) calcCheckpointByBlocks(blocks []*types.BlockHeader, ctx *cpContext) (cpHeight uint64, found bool) {
	if len(blocks) == 0 {
		return 0, false
	}
	votes := make(map[common.Hash]struct{})
	for visit := len(blocks) - 1; visit >= 0; visit-- {
		bh := blocks[visit]
		if types.EpochAt(bh.Height).Equal(ctx.epoch) {
			votes[bh.Group] = struct{}{}
			// Threshold-group height found
			if len(votes) >= ctx.threshold {
				return bh.Height, true
			}
		}
	}
	return 0, false
}

func isMissingNodeError(err error) bool {
	_, ok := err.(*trie.MissingNodeError)
	return ok
}

func (cp *cpChecker) checkpointAt(h uint64) uint64 {
	if h <= cpBlockBuffer {
		return 0
	}
	h -= cpBlockBuffer
	if h > cp.querier.Height() {
		h = cp.querier.Height()
	} else {
		h = cp.querier.QueryBlockHeaderFloor(h).Height
	}

	calcByAccountDB := true

	for scan := 0; scan < cpMaxScanEpochs; {
		ep := types.EpochAt(h)
		ctx := newCpContext(ep, cp.groupReader.GetActivatedGroupsAt(ep.Start()))
		if ctx.groupsEnough() {
			if calcByAccountDB {
				// Get the accountDB of end of the epoch
				db, err := cp.querier.AccountDBAt(h)
				if err != nil {
					Logger.Errorf("get account db at %v error:%v", h, err)
					// Missing node error may occurs when running at pruning-mode and then
					// will continue calculating the checkpoint by reading blocks from chain
					if isMissingNodeError(err) {
						calcByAccountDB = false
						continue
					}
					return 0
				}
				if h, found := cp.calcCheckpointByDB(db, ctx); found {
					return h
				}
			} else {
				blocks := cp.querier.BatchGetBlockHeadersBetween(ep.Start(), ep.End())
				if h, found := cp.calcCheckpointByBlocks(blocks, ctx); found {
					return h
				}
			}
		} else {
			// Not enough groups
			Logger.Debugf("not enough groups at %v-%v, groupsize %v, or not enough blocks %v", ep.Start(), ep.End(), ctx.groupSize(), h)
		}
		if ep.Start() == 0 {
			break
		}
		h = ep.Start() - 1
		scan++
	}
	return 0
}
