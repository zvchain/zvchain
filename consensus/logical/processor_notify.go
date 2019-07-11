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
	"github.com/zvchain/zvchain/monitor"

	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/types"
)

func (p *Processor) chLoop() {
	for {
		select {
		case bh := <-p.castVerifyCh:
			p.verifyCachedMsg(bh.Hash)
		case bh := <-p.futureVerifyCh:
			p.triggerFutureVerifyMsg(bh)
		case bh := <-p.futureRewardCh:
			p.triggerFutureRewardSign(bh)
		}
	}
}

func (p *Processor) triggerFutureVerifyMsg(bh *types.BlockHeader) {
	futures := p.getFutureVerifyMsgs(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(bh.Hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tLog := newHashTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
		tLog.logStart("size %v", len(futures))
		verifyTraceLog := monitor.NewPerformTraceLogger("verifyCastMessage", msg.BH.Hash, msg.BH.Height)
		verifyTraceLog.SetParent("triggerFutureVerifyMsg")
		ok, err := p.verifyCastMessage(msg, bh)
		verifyTraceLog.Log("result=%v %v", ok, err)
		tLog.logEnd("result=%v %v", ok, err)
	}

}

func (p *Processor) triggerFutureRewardSign(bh *types.BlockHeader) {
	futures := p.futureRewardReqs.getMessages(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.futureRewardReqs.remove(bh.Hash)
	mType := "CMCRSR-Future"
	for _, msg := range futures {
		tLog := newHashTraceLog(mType, bh.Hash, groupsig.ID{})
		send, err := p.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh)
		tLog.logEnd("send %v, result %v", send, err)
	}
}

// onBlockAddSuccess handle the event of block add-on-chain
func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(*types.Block)
	bh := block.Header

	tLog := newHashTraceLog("OnBlockAddSuccess", bh.Hash, groupsig.ID{})
	tLog.log("preHash=%v, height=%v", bh.PreHash, bh.Height)

	group := p.groupReader.getGroupBySeed(bh.Group)
	if group != nil && group.hasMember(p.GetMinerID()) {
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

	p.futureVerifyCh <- bh
	p.futureRewardCh <- bh

	p.blockContexts.removeProposed(bh.Hash)
}

// onGroupAddSuccess handles the event of verifyGroup add-on-chain
func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.GroupI)
	stdLogger.Infof("groupAddEventHandler receive message, gSeed=%v, workHeight=%v\n", group.Header().Seed(), group.Header().WorkHeight())

	memIds := make([]groupsig.ID, len(group.Members()))
	for _, mem := range group.Members() {
		memIds = append(memIds, groupsig.DeserializeID(mem.ID()))
	}
	p.NetServer.BuildGroupNet(group.Header().Seed().Hex(), memIds)
}
