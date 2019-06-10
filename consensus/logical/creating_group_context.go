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
	"time"

	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

// status enum of the CreatingGroupContext
const (
	waitingPong = 1 // waitingPong indicates the context is waiting for pong response from nodes
	waitingSign = 2 // waitingSign indicates the context is waiting for the group signature for the group-creating proposal
	sendInit    = 3 // sendInit indicates the context has send group init message to the members who make up the new group
)

type createGroupBaseContext struct {
	parentInfo *StaticGroupInfo   // the parent group info
	baseBH     *types.BlockHeader // the blockHeader the group-create routine based on
	baseGroup  *types.Group       // the last group of the groupchain
	candidates []groupsig.ID      // the legal candidates
}

// CreatingGroupContext stores the context info when parent group starting group-create routine
type CreatingGroupContext struct {
	createGroupBaseContext

	gSignGenerator *model.GroupSignGenerator // group signature generator

	kings           []groupsig.ID // kings selected randomly from the parent group who responsible for node pings and new group proposal
	pingID          string        // identify one ping process
	createTime      time.Time     // create time for the context, used to make local timeout judgments
	createTopHeight uint64        // the blockchain height when starting the group-create routine

	gInfo   *model.ConsensusGroupInitInfo // new group info generated during the routine and will be sent to the new-group members for consensus
	pongMap map[string]byte               // pong response received from candidates
	memMask []byte                        // each non-zero bit indicates that the candidate at the subscript replied to the ping message and will become a full member of the new-group
	status  int8                          // the context status
	bKing   bool                          // whether the current node is one of the kings
	lock    sync.RWMutex
}

func newCreateGroupBaseContext(sgi *StaticGroupInfo, baseBH *types.BlockHeader, baseG *types.Group, cands []groupsig.ID) *createGroupBaseContext {
	return &createGroupBaseContext{
		parentInfo: sgi,
		baseBH:     baseBH,
		baseGroup:  baseG,
		candidates: cands,
	}
}

func newCreateGroupContext(baseCtx *createGroupBaseContext, kings []groupsig.ID, isKing bool, top uint64) *CreatingGroupContext {
	pingIDBytes := baseCtx.baseBH.Hash.Bytes()
	pingIDBytes = append(pingIDBytes, baseCtx.baseGroup.ID...)
	cg := &CreatingGroupContext{
		createGroupBaseContext: *baseCtx,
		kings:                  kings,
		status:                 waitingPong,
		createTime:             time.Now(),
		bKing:                  isKing,
		createTopHeight:        top,
		pingID:                 base.Data2CommonHash(pingIDBytes).Hex(),
		pongMap:                make(map[string]byte, 0),
		gSignGenerator:         model.NewGroupSignGenerator(model.Param.GetGroupK(baseCtx.parentInfo.GetMemberCount())),
	}

	return cg
}

func (ctx *createGroupBaseContext) hasCandidate(uid groupsig.ID) bool {
	for _, id := range ctx.candidates {
		if id.IsEqual(uid) {
			return true
		}
	}
	return false
}

func (ctx *createGroupBaseContext) readyHeight() uint64 {
	return ctx.baseBH.Height + model.Param.GroupReadyGap
}

func (ctx *createGroupBaseContext) readyTimeout(h uint64) bool {
	return h >= ctx.readyHeight()
}

func (ctx *createGroupBaseContext) recoverMemberSet(mask []byte) (ids []groupsig.ID) {
	ids = make([]groupsig.ID, 0)
	for i, id := range ctx.candidates {
		b := mask[i/8]
		if (b & (1 << byte(i%8))) != 0 {
			ids = append(ids, id)
		}
	}
	return
}

func (ctx *createGroupBaseContext) createGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
	pid := ctx.parentInfo.GroupID
	theBH := ctx.baseBH
	gn := fmt.Sprintf("%s-%v", pid.GetHexString(), theBH.Height)
	extends := fmt.Sprintf("baseBlock:%v|%v|%v", theBH.Hash.Hex(), theBH.CurTime, theBH.Height)

	gh := &types.GroupHeader{
		Parent:       ctx.parentInfo.GroupID.Serialize(),
		PreGroup:     ctx.baseGroup.ID,
		Name:         gn,
		Authority:    777,
		BeginTime:    theBH.CurTime,
		CreateHeight: theBH.Height,
		ReadyHeight:  ctx.readyHeight(),
		WorkHeight:   theBH.Height + model.Param.GroupWorkGap,
		MemberRoot:   model.GenMemberRootByIds(memIds),
		Extends:      extends,
	}
	gh.DismissHeight = gh.WorkHeight + model.Param.GroupworkDuration

	gh.Hash = gh.GenHash()
	return gh
}

func (ctx *createGroupBaseContext) createGroupInitInfo(mask []byte) *model.ConsensusGroupInitInfo {
	memIds := ctx.recoverMemberSet(mask)
	gh := ctx.createGroupHeader(memIds)
	return &model.ConsensusGroupInitInfo{
		GI:   model.ConsensusGroupInitSummary{GHeader: gh},
		Mems: memIds,
	}
}

func (ctx *CreatingGroupContext) pongDeadline(h uint64) bool {
	return h >= ctx.baseBH.Height+model.Param.GroupWaitPongGap
}

func (ctx *CreatingGroupContext) isKing() bool {
	return ctx.bKing
}

func (ctx *CreatingGroupContext) addPong(h uint64, uid groupsig.ID) (add bool, size int) {
	if ctx.pongDeadline(h) {
		return false, ctx.pongSize()
	}
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	if ctx.hasCandidate(uid) {
		ctx.pongMap[uid.GetHexString()] = 1
		add = true
	}
	size = len(ctx.pongMap)
	return
}

func (ctx *CreatingGroupContext) pongSize() int {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return len(ctx.pongMap)
}

func (ctx *CreatingGroupContext) getStatus() int8 {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return ctx.status
}

func (ctx *CreatingGroupContext) setStatus(st int8) {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	ctx.status = st
}

func (ctx *CreatingGroupContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(ctx.candidates)+7)/8)

	for i, id := range ctx.candidates {
		b := mask[i/8]
		if _, ok := ctx.pongMap[id.GetHexString()]; ok {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

func (ctx *CreatingGroupContext) generateGroupInitInfo(h uint64) bool {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	if ctx.gInfo != nil {
		return true
	}
	if len(ctx.pongMap) == len(ctx.candidates) || ctx.pongDeadline(h) {
		mask := ctx.generateMemberMask()
		gInfo := ctx.createGroupInitInfo(mask)
		ctx.gInfo = gInfo
		ctx.memMask = mask
		return true
	}

	return false
}

func (ctx *CreatingGroupContext) acceptPiece(from groupsig.ID, sign groupsig.Signature) (accept, recover bool) {
	accept, recover = ctx.gSignGenerator.AddWitness(from, sign)
	return
}

func (ctx *CreatingGroupContext) logString() string {
	return fmt.Sprintf("baseHeight=%v, topHeight=%v, candidates=%v, isKing=%v, parentGroup=%v, pongs=%v, elapsed=%v",
		ctx.baseBH.Height, ctx.createTopHeight, len(ctx.candidates), ctx.isKing(), ctx.parentInfo.GroupID.ShortS(), ctx.pongSize(), time.Since(ctx.createTime).String())

}
