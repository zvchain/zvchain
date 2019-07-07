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

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
)

// GroupManager is responsible for group creation
type GroupManager struct {
	groupChain       *core.GroupChain
	mainChain        core.BlockChain
	processor        *Processor
	creatingGroupCtx *CreatingGroupContext
	checker          *GroupCreateChecker
}

func newGroupManager(processor *Processor) *GroupManager {
	gm := &GroupManager{
		processor:  processor,
		mainChain:  processor.MainChain,
		groupChain: processor.GroupChain,
		checker:    newGroupCreateChecker(processor),
	}
	return gm
}

func (gm *GroupManager) setCreatingGroupContext(baseCtx *createGroupBaseContext, kings []groupsig.ID, isKing bool) {
	ctx := newCreateGroupContext(baseCtx, kings, isKing, gm.mainChain.Height())
	gm.creatingGroupCtx = ctx
}

func (gm *GroupManager) getContext() *CreatingGroupContext {
	return gm.creatingGroupCtx
}

func (gm *GroupManager) removeContext() {
	gm.creatingGroupCtx = nil
}

// CreateNextGroupRoutine start the group-create routine
func (gm *GroupManager) CreateNextGroupRoutine() {
	if !gm.processor.genesisMember {
		return
	}
	top := gm.mainChain.QueryTopBlock()
	topHeight := top.Height

	gap := model.Param.GroupCreateGap
	if topHeight > gap {
		gm.checkReqCreateGroupSign(topHeight)

		pre := gm.mainChain.QueryBlockHeaderByHash(top.PreHash)
		if pre != nil {
			for h := top.Height; h > pre.Height && h > gap; h-- {
				baseHeight := h - gap
				if checkCreate(baseHeight) {
					gm.checkCreateGroupRoutine(baseHeight)
					break
				}
			}
		}
	}

}

func (gm *GroupManager) onMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) (bool, error) {
	blog := newBizLog("OMCGR")
	blog.debug("gHash=%v, sender=%v", msg.GInfo.GI.GetHash().ShortS(), msg.SI.SignMember.ShortS())

	ctx := gm.getContext()
	if ctx == nil {
		return false, fmt.Errorf("ctx is nil")
	}
	if ctx.getStatus() == sendInit {
		return false, fmt.Errorf("has send inited")
	}
	top := gm.mainChain.Height()
	if ctx.readyTimeout(top) {
		return false, fmt.Errorf("ready timeout")
	}
	if !ctx.generateGroupInitInfo(top) {
		return false, fmt.Errorf("generate group init info fail")
	}
	if ctx.gInfo.GroupHash() != msg.GInfo.GroupHash() {
		blog.error("expect gh %+v, real gh %+v", ctx.gInfo.GI.GHeader, msg.GInfo.GI.GHeader)
		return false, fmt.Errorf("grouphash diff")
	}
	return true, nil

}

func (gm *GroupManager) onMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage) (bool, error) {
	blog := newBizLog("OMCGS")
	blog.debug("gHash=%v, sender=%v", msg.GHash.ShortS(), msg.SI.SignMember.ShortS())
	ctx := gm.getContext()
	if ctx == nil {
		return false, fmt.Errorf("context is nil")
	}

	height := gm.processor.MainChain.QueryTopBlock().Height
	if ctx.readyTimeout(height) {
		return false, fmt.Errorf("ready timeout")
	}
	if ctx.gInfo.GroupHash() != msg.GHash {
		return false, fmt.Errorf("gHash diff")
	}

	accept, recover := ctx.acceptPiece(msg.SI.GetID(), msg.SI.DataSign)
	blog.debug("accept result %v %v", accept, recover)
	newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID()).log("onMessageCreateGroupSign ret %v, %v", recover, ctx.gSignGenerator.Brief())
	if recover {
		ctx.gInfo.GI.Signature = ctx.gSignGenerator.GetGroupSign()
		return true, nil
	}
	return false, fmt.Errorf("waiting piece")
}

func (gm *GroupManager) addGroupOnChain(sgi *StaticGroupInfo) {
	group := convertStaticGroup2CoreGroup(sgi)

	stdLogger.Infof("addGroupOnChain height:%d,id:%s\n", group.GroupHeight, sgi.GroupID.ShortS())

	var err error
	defer func() {
		var s string
		if err != nil {
			s = err.Error()
		}
		newHashTraceLog("addGroupOnChain", sgi.GInfo.GroupHash(), groupsig.ID{}).log("gid=%v, workHeight=%v, result %v", sgi.GroupID.ShortS(), group.Header.WorkHeight, s)
	}()

	if gm.groupChain.GetGroupByID(group.ID) != nil {
		stdLogger.Debugf("group already onchain, accept, id=%v\n", sgi.GroupID.ShortS())
		gm.processor.acceptGroup(sgi)
		err = fmt.Errorf("group already onchain")
	} else {
		top := gm.processor.MainChain.Height()
		if !sgi.GetReadyTimeout(top) {
			err1 := gm.groupChain.AddGroup(group)
			if err1 != nil {
				stdLogger.Errorf("ERROR:add group fail! hash=%v, gid=%v, err=%v\n", group.Header.Hash.ShortS(), sgi.GroupID.ShortS(), err1.Error())
				err = err1
				return
			}
			err = fmt.Errorf("success")
			gm.checker.addHeightCreated(group.Header.CreateHeight)
			stdLogger.Infof("addGroupOnChain success, ID=%v, height=%v\n", sgi.GroupID.ShortS(), gm.groupChain.Height())
		} else {
			err = fmt.Errorf("ready timeout, currentHeight %v", top)
			stdLogger.Infof("addGroupOnChain group ready timeout, gid %v, timeout height %v, top %v\n", sgi.GroupID.ShortS(), sgi.GInfo.GI.GHeader.ReadyHeight, top)
		}
	}

}

func (gm *GroupManager) onGroupAddSuccess(g *StaticGroupInfo) {
	ctx := gm.getContext()
	if ctx != nil && ctx.gInfo != nil && ctx.gInfo.GroupHash() == g.GInfo.GroupHash() {
		top := gm.mainChain.Height()
		gm.removeContext()
	}
}
