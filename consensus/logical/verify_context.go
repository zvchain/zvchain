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
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"gopkg.in/fatih/set.v0"
)

// status enum of the verification consensus
const (
	// svWorking indicates waiting for shards
	svWorking = iota
	// svNotified means block body requested from proposer already
	svNotified
	// svSuccess indicates block added on chain successfully
	svSuccess
	// svTimeout means the block consensus timeout
	svTimeout
)

const (
	// pieceNormal means received a normal piece
	pieceNormal = 1

	// pieceThreshold means received a piece and reached the threshold,
	pieceThreshold = 2

	// pieceFail meas piece denied
	pieceFail = -1
)

// VerifyContext stores the context of verification consensus of each height.
// It is unique to each height which means replacement will take place when two different instance
// created for only one height
type VerifyContext struct {
	prevBH           *types.BlockHeader // Pre-block header the height based on
	castHeight       uint64             // The height the context related to
	signedMaxWeight  atomic.Value       // type of *types.BlockWeight, stores max weight block info current node signed
	signedBlockHashs set.Interface      // block hash set the current node signed, used for duplicated signed judgement
	expireTime       time.TimeStamp     // block consensus deadline
	createTime       time.TimeStamp
	consensusStatus  int32                        // consensus status
	slots            map[common.Hash]*SlotContext // verification context of each block proposal related to the context, called slot
	proposers        map[string]common.Hash       // Record the block hash corresponding to each proposer, Used to limit a proposer to work only once at that height

	successSlot *SlotContext // The slot that has make the consensus finished
	group       *verifyGroup // Corresponding verification Group info
	signedNum   int32        // Numbers of signed blocks
	verifyNum   int32        // Numbers of verified signatures
	aggrNum     int32        // Numbers of blocks that verifyGroup-sign recovered
	lock        sync.RWMutex
	ts          time.TimeService
}

func newVerifyContext(group *verifyGroup, castHeight uint64, expire time.TimeStamp, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{
		prevBH:           preBH,
		castHeight:       castHeight,
		group:            group,
		expireTime:       expire,
		consensusStatus:  svWorking,
		slots:            make(map[common.Hash]*SlotContext),
		ts:               time.TSInstance,
		createTime:       time.TSInstance.Now(),
		proposers:        make(map[string]common.Hash),
		signedBlockHashs: set.New(set.ThreadSafe),
	}
	return ctx
}

func (vc *VerifyContext) isWorking() bool {
	status := atomic.LoadInt32(&vc.consensusStatus)
	return status != svTimeout
}

func (vc *VerifyContext) castSuccess() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == svSuccess
}

func (vc *VerifyContext) isNotified() bool {
	return atomic.LoadInt32(&vc.consensusStatus) == svNotified
}

func (vc *VerifyContext) markTimeout() {
	if !vc.castSuccess() {
		atomic.StoreInt32(&vc.consensusStatus, svTimeout)
	}
}

func (vc *VerifyContext) markCastSuccess() {
	atomic.StoreInt32(&vc.consensusStatus, svSuccess)
}

func (vc *VerifyContext) markNotified() {
	atomic.StoreInt32(&vc.consensusStatus, svNotified)
}

// castExpire means whether the ingot has expired
func (vc *VerifyContext) castExpire() bool {
	return vc.ts.NowAfter(vc.expireTime)
}

// castRewardSignExpire means whether the reward transaction signature expires
func (vc *VerifyContext) castRewardSignExpire() bool {
	return vc.ts.NowAfter(vc.expireTime.Add(int64(30 * model.Param.MaxGroupCastTime)))
}

func (vc *VerifyContext) findSlot(hash common.Hash) *SlotContext {
	if sc, ok := vc.slots[hash]; ok {
		return sc
	}
	return nil
}

func (vc *VerifyContext) getSignedMaxWeight() *types.BlockWeight {
	v := vc.signedMaxWeight.Load()
	if v == nil {
		return nil
	}
	return v.(*types.BlockWeight)
}

func (vc *VerifyContext) hasSignedMoreWeightThan(bh *types.BlockHeader) bool {
	bw := vc.getSignedMaxWeight()
	if bw == nil {
		return false
	}
	bw2 := types.NewBlockWeight(bh)
	return bw.MoreWeight(bw2)
}

func (vc *VerifyContext) updateSignedMaxWeightBlock(bh *types.BlockHeader) bool {
	bw := vc.getSignedMaxWeight()
	bw2 := types.NewBlockWeight(bh)
	if bw != nil && bw.MoreWeight(bw2) {
		return false
	}
	vc.signedMaxWeight.Store(bw2)
	return true
}

func (vc *VerifyContext) baseCheck(bh *types.BlockHeader, sender groupsig.ID) (err error) {
	if bh.Elapsed <= 0 {
		err = fmt.Errorf("elapsed error %v", bh.Elapsed)
		return
	}
	// Check time window
	if vc.ts.Since(bh.CurTime) < -1 {
		return fmt.Errorf("block too early: now %v, curtime %v", vc.ts.Now(), bh.CurTime)
	}
	begin := vc.expireTime.Add(-int64(model.Param.MaxGroupCastTime + 1))
	if bh.Height > 1 && !vc.ts.NowAfter(begin) {
		return fmt.Errorf("block too early: begin %v, now %v", begin, vc.ts.Now())
	}

	// Check verifyGroup id
	if vc.group.header.Seed() != bh.Group {
		return fmt.Errorf("groupId error:vc-%v, bh-%v", vc.group.header.Seed(), bh.Group)
	}

	// Only sign blocks with higher weights than that have been signed
	if vc.castHeight > 1 && vc.hasSignedMoreWeightThan(bh) {
		max := vc.getSignedMaxWeight()
		err = fmt.Errorf("have signed a higher qn block %v,This block qn %v", max, bh.TotalQN)
		return
	}

	if vc.castSuccess() || vc.isNotified() {
		err = fmt.Errorf("already blocked:%v", vc.consensusStatus)
		return
	}
	// Don't check the height 1
	if vc.castExpire() && vc.castHeight > 1 {
		vc.markTimeout()
		err = fmt.Errorf("timed out" + vc.expireTime.String())
		return
	}
	slot := vc.GetSlotByHash(bh.Hash)
	if slot != nil {
		if slot.GetSlotStatus() >= slRecovered {
			err = fmt.Errorf("slot does not accept piece,slot status %v", slot.slotStatus)
			return
		}
		if _, ok := slot.gSignGenerator.GetWitness(sender); ok {
			err = fmt.Errorf("duplicate message %v", sender)
			return
		}
	}

	return
}

// GetSlotByHash return the slot related to the given block hash
func (vc *VerifyContext) GetSlotByHash(hash common.Hash) *SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	return vc.findSlot(hash)
}

// PrepareSlot returns the slotContext related to the given blockHeader.
// Replacement will take place if the existing slots reaches the limit defined in the model.Param
func (vc *VerifyContext) PrepareSlot(bh *types.BlockHeader) (*SlotContext, error) {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if vc.slots == nil {
		return nil, fmt.Errorf("slots is nil")
	}

	if vc.hasSignedMoreWeightThan(bh) && vc.castHeight > 1 {
		return nil, fmt.Errorf("hasSignedMoreWeightThan:%v", vc.getSignedMaxWeight())
	}
	sc := createSlotContext(bh, int(vc.group.header.Threshold()))
	if v, ok := vc.proposers[sc.castor.GetHexString()]; ok && vc.castHeight > 1 {
		if v != bh.Hash {
			return nil, fmt.Errorf("too many proposals: castor %v", sc.castor.GetHexString())
		}
	} else {
		vc.proposers[sc.castor.GetHexString()] = bh.Hash
	}
	if len(vc.slots) >= model.Param.MaxSlotSize {
		var (
			minWeight     *types.BlockWeight
			minWeightHash common.Hash
		)
		for hash, slot := range vc.slots {
			bw := types.NewBlockWeight(slot.BH)
			if minWeight == nil || minWeight.MoreWeight(bw) {
				minWeight = bw
				minWeightHash = hash
			}
		}
		currBw := *types.NewBlockWeight(bh)
		if currBw.MoreWeight(minWeight) {
			delete(vc.slots, minWeightHash)
		} else {
			return nil, fmt.Errorf("comming block weight less than min block weight")
		}
	}
	vc.slots[bh.Hash] = sc
	return sc, nil

}

// Clear release the slot
func (vc *VerifyContext) Clear() {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	vc.slots = nil
	vc.successSlot = nil
}

// shouldRemove determine whether the context can be deleted,
// mainly consider whether to send a reward transaction
func (vc *VerifyContext) shouldRemove(topHeight uint64) bool {
	// Reward transaction consensus timed out, can be deleted
	if vc.castRewardSignExpire() {
		return true
	}

	// Self-broadcast and already sent reward transaction, can be deleted
	if vc.successSlot != nil && vc.successSlot.IsRewardSent() {
		return true
	}

	// No block, but has stayed more than 200 heights, can be deleted
	if vc.castHeight+200 < topHeight {
		return true
	}
	return false
}

// GetSlots gets all the slots
func (vc *VerifyContext) GetSlots() []*SlotContext {
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	slots := make([]*SlotContext, 0)
	for _, slot := range vc.slots {
		slots = append(slots, slot)
	}
	return slots
}

// checkNotify check and returns the slotContext the maximum weight of block stored in
func (vc *VerifyContext) checkNotify() *SlotContext {
	blog := newBizLog("checkNotify")
	if vc.isNotified() || vc.castSuccess() {
		return nil
	}
	if vc.ts.Since(vc.createTime) < int64(model.Param.MaxWaitBlockTime) {
		return nil
	}
	var (
		maxBwSlot *SlotContext
		maxBw     *types.BlockWeight
	)

	vc.lock.RLock()
	defer vc.lock.RUnlock()
	qns := make([]uint64, 0)

	for _, slot := range vc.slots {
		if !slot.IsRecovered() {
			continue
		}
		qns = append(qns, slot.BH.TotalQN)
		bw := types.NewBlockWeight(slot.BH)

		if maxBw == nil || bw.MoreWeight(maxBw) {
			maxBwSlot = slot
			maxBw = bw
		}
	}
	if maxBwSlot != nil {
		blog.debug("select max qn=%v, hash=%v, height=%v, hash=%v, size=%v", maxBwSlot.BH.TotalQN, maxBwSlot.BH.Hash, maxBwSlot.BH.Height, maxBwSlot.BH.Hash, len(qns))
	}
	return maxBwSlot
}

func (vc *VerifyContext) increaseVerifyNum() {
	atomic.AddInt32(&vc.verifyNum, 1)
}

func (vc *VerifyContext) increaseAggrNum() {
	atomic.AddInt32(&vc.aggrNum, 1)
}

func (vc *VerifyContext) markSignedBlock(bh *types.BlockHeader) {
	vc.signedBlockHashs.Add(bh.Hash)
	atomic.AddInt32(&vc.signedNum, 1)
	vc.updateSignedMaxWeightBlock(bh)
}

func (vc *VerifyContext) blockSigned(hash common.Hash) bool {
	return vc.signedBlockHashs.Has(hash)
}
