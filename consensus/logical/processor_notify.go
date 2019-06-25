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
	"fmt"
	"github.com/zvchain/zvchain/monitor"

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
)

func (p *Processor) triggerFutureVerifyMsg(bh *types.BlockHeader) {
	futures := p.getFutureVerifyMsgs(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(bh.Hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newHashTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
		tlog.logStart("size %v", len(futures))
		verifyTraceLog := monitor.NewPerformTraceLogger("verifyCastMessage", msg.BH.Hash, msg.BH.Height)
		verifyTraceLog.SetParent("triggerFutureVerifyMsg")
		ok, err := p.verifyCastMessage(msg, bh)
		verifyTraceLog.Log("result=%v %v", ok, err)
		tlog.logEnd("result=%v %v", ok, err)
	}

}

func (p *Processor) triggerFutureRewardSign(bh *types.BlockHeader) {
	futures := p.futureRewardReqs.getMessages(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.futureRewardReqs.remove(bh.Hash)
	mtype := "CMCRSR-Future"
	for _, msg := range futures {
		blog := newBizLog(mtype)
		send, err := p.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh)
		blog.debug("send %v, result %v", send, err)
	}
}

// onBlockAddSuccess handle the event of block add-on-chain
func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(*types.Block)
	bh := block.Header

	tlog := newMsgTraceLog("OnBlockAddSuccess", bh.Hash.ShortS(), "")
	tlog.log("preHash=%v, height=%v", bh.PreHash.ShortS(), bh.Height)

	gid := groupsig.DeserializeID(bh.GroupID)
	if p.IsMinerGroup(gid) {
		p.blockContexts.addCastedHeight(bh.Height, bh.PreHash)
		vctx := p.blockContexts.getVctxByHeight(bh.Height)
		if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
			if vctx.isWorking() {
				vctx.markCastSuccess()
			}
			p.reqRewardTransSign(vctx, bh)
		}
	}

	vrf := p.getVrfWorker()
	if vrf != nil && vrf.baseBH.Hash == bh.PreHash && vrf.castHeight == bh.Height {
		vrf.markSuccess()
	}

	traceLog := monitor.NewPerformTraceLogger("onBlockAddSuccess", bh.Hash, bh.Height)
	// start to check next proposal routine immediately
	go p.checkSelfCastRoutine()

	traceLog.Log("block onchain cost %v", p.ts.Now().Local().Sub(bh.CurTime.Local()).String())

	//p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh)
	p.triggerFutureRewardSign(bh)
	p.groupManager.CreateNextGroupRoutine()
	p.blockContexts.removeProposed(bh.Hash)
}

// onGroupAddSuccess handles the event of group add-on-chain
func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(*types.Group)
	stdLogger.Infof("groupAddEventHandler receive message, groupId=%v, workheight=%v\n", groupsig.DeserializeID(group.ID).GetHexString(), group.Header.WorkHeight)
	if group.ID == nil || len(group.ID) == 0 {
		return
	}
	sgi := newSGIFromCoreGroup(group)
	p.acceptGroup(sgi)

	p.groupManager.onGroupAddSuccess(sgi)
	p.joiningGroups.Clean(sgi.GInfo.GroupHash())
	p.globalGroups.removeInitedGroup(sgi.GInfo.GroupHash())

	beginHeight := group.Header.WorkHeight
	topHeight := p.MainChain.Height()

	// The current block height has exceeded the effective height, group may have a problem
	if beginHeight > 0 && beginHeight <= topHeight {
		stdLogger.Warnf("group add after can work! gid=%v, gheight=%v, beginHeight=%v, currentHeight=%v", sgi.GroupID.ShortS(), group.GroupHeight, beginHeight, topHeight)
		pre := p.MainChain.QueryBlockHeaderFloor(beginHeight - 1)
		if pre == nil {
			// hold it for now
			panic(fmt.Sprintf("block nil at height %v", beginHeight-1))
		}
		for h := beginHeight; h <= topHeight; {
			bh := p.MainChain.QueryBlockHeaderCeil(h)
			if bh == nil {
				break
			}
			if bh.PreHash != pre.Hash {
				// hold it for now
				panic(fmt.Sprintf("pre error:bh %v, prehash %v, height %v, real pre hash %v height %v", bh.Hash.Hex(), bh.PreHash.Hex(), bh.Height, pre.Hash.Hex(), pre.Height))
			}
			gid := p.calcVerifyGroupFromChain(pre, bh.Height)
			if !bytes.Equal(gid.Serialize(), bh.GroupID) {
				old := p.MainChain.QueryTopBlock()
				stdLogger.Errorf("adjust top block: old %v %v %v, new %v %v %v", old.Hash.Hex(), old.PreHash.Hex(), old.Height, pre.Hash.Hex(), pre.PreHash.Hex(), pre.Height)
				p.MainChain.ResetTop(pre)
				break
			}
			pre = bh
			h = bh.Height + 1
		}
	}
}
