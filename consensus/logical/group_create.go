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
	"strings"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/monitor"
)

// selectParentGroup determine the parent group randomly and the result is deterministic because of the base BlockHeader
func (gm *GroupManager) selectParentGroup(baseBH *types.BlockHeader, preGroupID []byte) (*StaticGroupInfo, error) {
	return gm.processor.globalGroups.getGenesisGroup()
}

func (gm *GroupManager) generateCreateGroupContext(baseHeight uint64) (*createGroupBaseContext, error) {
	lastGroup := gm.groupChain.LastGroup()
	baseBH := gm.mainChain.QueryBlockHeaderByHeight(baseHeight)
	if !checkCreate(baseHeight) {
		return nil, fmt.Errorf("cannot create group at the height")
	}
	if baseBH == nil {
		return nil, fmt.Errorf("base block is nil, height=%v", baseHeight)
	}
	sgi, err := gm.selectParentGroup(baseBH, lastGroup.ID)
	if sgi == nil || err != nil {
		return nil, fmt.Errorf("select parent group err %v", err)
	}
	enough, candidates := gm.checker.selectCandidates(baseBH)
	if !enough {
		return nil, fmt.Errorf("not enough candidates")
	}
	return newCreateGroupBaseContext(sgi, baseBH, lastGroup, candidates), nil
}

// checkCreateGroupRoutine check if the height meets the conditions for creating a group
// if so then start the group-create process
func (gm *GroupManager) checkCreateGroupRoutine(baseHeight uint64) {
	blog := newBizLog("checkCreateGroupRoutine")
	create := false
	var err error

	defer func() {
		ret := ""
		if err != nil {
			ret = err.Error()
		}
		blog.debug("baseBH height=%v, create=%v, ret=%v", baseHeight, create, ret)
	}()

	// The specified height has appeared on the group chain
	if gm.checker.heightCreated(baseHeight) {
		err = fmt.Errorf("topHeight already created")
		return
	}

	// generate the basic context
	baseCtx, err2 := gm.generateCreateGroupContext(baseHeight)
	if err2 != nil {
		err = err2
		return
	}

	// if current node doesn't belong to the selected parent group, it won't start the routine
	if !gm.processor.IsMinerGroup(baseCtx.parentInfo.GroupID) {
		err = fmt.Errorf("next select group id %v, not belong to the group", baseCtx.parentInfo.GroupID.GetHexString())
		return
	}

	kings, isKing := gm.checker.selectKing(baseCtx.baseBH, baseCtx.parentInfo)

	gm.setCreatingGroupContext(baseCtx, kings, isKing)
	groupLogger.Infof("createGroupContext info=%v", gm.getContext().logString())

	gm.pingNodes()
	create = true

}

// pingNodes send ping messages to the new members,
// in order to avoid too much ping messages, the current node does this only when he is one of kings.
func (gm *GroupManager) pingNodes() {
	ctx := gm.creatingGroupCtx
	if ctx == nil || !ctx.isKing() {
		return
	}
	msg := &model.CreateGroupPingMessage{
		FromGroupID: ctx.parentInfo.GroupID,
		PingID:      ctx.pingID,
		BaseHeight:  ctx.baseBH.Height,
	}
	blog := newBizLog("pingNodes")
	if msg.GenSign(gm.processor.getDefaultSeckeyInfo(), msg) {
		for _, id := range ctx.candidates {
			blog.debug("baseHeight=%v, pingID=%v, id=%v", ctx.baseBH.Height, msg.PingID, id.ShortS())
			gm.processor.NetServer.SendGroupPingMessage(msg, id)
		}
	}
}

func (gm *GroupManager) checkReqCreateGroupSign(topHeight uint64) bool {
	blog := newBizLog("checkReqCreateGroupSign")
	ctx := gm.creatingGroupCtx
	if ctx == nil {
		return false
	}

	var desc string
	defer func() {
		if desc != "" {
			groupLogger.Infof("checkReqCreateGroupSign info=%v, %v", ctx.logString(), desc)
		}
	}()

	if ctx.readyTimeout(topHeight) {
		return false
	}

	pongsize := ctx.pongSize()

	if ctx.getStatus() != waitingPong {
		return false
	}

	if !ctx.generateGroupInitInfo(topHeight) {
		desc = fmt.Sprintf("cannot generate group info, pongsize %v, pongdeadline %v", pongsize, ctx.pongDeadline(topHeight))
		return false
	}

	ctx.setStatus(waitingSign)
	gInfo := ctx.gInfo
	gh := gInfo.GI.GHeader

	desc = fmt.Sprintf("generateGroupInitInfo gHash=%v, memsize=%v, wait sign", gh.Hash.ShortS(), gInfo.MemberSize())

	if !ctx.isKing() {
		return false
	}
	if gInfo.MemberSize() < model.Param.GroupMemberMin {
		blog.warn("got not enough pongs!, got %v", pongsize)
		desc = "not enough pongs."
		return false
	}

	msg := &model.ConsensusCreateGroupRawMessage{
		GInfo: *gInfo,
	}
	ski := gm.processor.getInGroupSeckeyInfo(ctx.parentInfo.GroupID)
	if !msg.GenSign(ski, msg) {
		blog.error("genSign fail, id=%v, sk=%v", ski.ID.ShortS(), ski.SK.ShortS())
		return false
	}

	memIDStrs := make([]string, 0)
	for _, mem := range gInfo.Mems {
		memIDStrs = append(memIDStrs, mem.ShortS())
	}
	newHashTraceLog("checkReqCreateGroupSign", gh.Hash, gm.processor.GetMinerID()).log("parent %v, members %v", ctx.parentInfo.GroupID.ShortS(), strings.Join(memIDStrs, ","))

	// Send info
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeCreateGroup,
		Height:   gm.groupChain.Height(),
		Hash:     gh.Hash.Hex(),
		Proposer: gm.processor.GetMinerID().GetHexString(),
	}
	if monitor.Instance.IsFirstNInternalNodesInGroup(ctx.kings, 20) {
		monitor.Instance.AddLog(le)
	}

	gm.processor.NetServer.SendCreateGroupRawMessage(msg)
	desc += "req sign"
	return true
}

// checkGroupInfo check whether the group info is legal
func (gm *GroupManager) checkGroupInfo(gInfo *model.ConsensusGroupInitInfo) ([]groupsig.ID, bool, error) {
	gh := gInfo.GI.GHeader
	if gh.Hash != gh.GenHash() {
		return nil, false, fmt.Errorf("gh hash error, hash=%v, genHash=%v", gh.Hash.ShortS(), gh.GenHash().ShortS())
	}
	// check if the member count is legal
	if !model.Param.IsGroupMemberCountLegal(len(gInfo.Mems)) {
		return nil, false, fmt.Errorf("group member size error %v(%v-%v)", len(gInfo.Mems), model.Param.GroupMemberMin, model.Param.GroupMemberMax)
	}

	// check if the create height is legal
	if !checkCreate(gh.CreateHeight) {
		return nil, false, fmt.Errorf("cannot create at the height %v", gh.CreateHeight)
	}
	baseBH := gm.mainChain.QueryBlockHeaderByHeight(gh.CreateHeight)
	if baseBH == nil {
		return nil, false, common.ErrCreateBlockNil
	}
	// The previous group, whether the parent group exists
	preGroup := gm.groupChain.GetGroupByID(gh.PreGroup)
	if preGroup == nil {
		return nil, false, fmt.Errorf("preGroup is nil, gid=%v", groupsig.DeserializeID(gh.PreGroup).ShortS())
	}
	parentGroup := gm.groupChain.GetGroupByID(gh.Parent)
	if parentGroup == nil {
		return nil, false, fmt.Errorf("parentGroup is nil, gid=%v", groupsig.DeserializeID(gh.Parent).ShortS())
	}

	// check if it is the specified parent group
	sgi, err := gm.selectParentGroup(baseBH, gh.PreGroup)
	if err != nil {
		return nil, false, fmt.Errorf("select parent group err %v", err)
	}
	pid := groupsig.DeserializeID(parentGroup.ID)
	if !sgi.GroupID.IsEqual(pid) {
		return nil, false, fmt.Errorf("select parent group not equal, expect %v, recieve %v", sgi.GroupID.ShortS(), pid.ShortS())
	}
	gpk := gm.processor.getGroupPubKey(groupsig.DeserializeID(gh.Parent))

	// check the signature of the parent group
	if !groupsig.VerifySig(gpk, gh.Hash.Bytes(), gInfo.GI.Signature) {
		return nil, false, fmt.Errorf("verify parent sign fail")
	}

	// check if the candidates are legal
	enough, candidates := gm.checker.selectCandidates(baseBH)
	if !enough {
		return nil, false, fmt.Errorf("not enough candidates")
	}
	// Whether the selected member is in the designated candidate
	for _, mem := range gInfo.Mems {
		find := false
		for _, cand := range candidates {
			if mem.IsEqual(cand) {
				find = true
				break
			}
		}
		if !find {
			return nil, false, fmt.Errorf("mem error: %v is not a legal candidate", mem.ShortS())
		}
	}

	return candidates, true, nil
}

// recoverGroupInitInfo recover group info from mask
func (gm *GroupManager) recoverGroupInitInfo(baseHeight uint64, mask []byte) (*model.ConsensusGroupInitInfo, error) {
	ctx, err := gm.generateCreateGroupContext(baseHeight)
	if err != nil {
		return nil, err
	}
	return ctx.createGroupInitInfo(mask), nil
}
