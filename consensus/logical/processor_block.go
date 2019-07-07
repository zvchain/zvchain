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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"sync"
)

// FutureMessageHolder store some messages non-processable currently and may be processed in the future
type FutureMessageHolder struct {
	messages sync.Map
}

func NewFutureMessageHolder() *FutureMessageHolder {
	return &FutureMessageHolder{
		messages: sync.Map{},
	}
}
func (holder *FutureMessageHolder) addMessage(hash common.Hash, msg interface{}) {
	if vs, ok := holder.messages.Load(hash); ok {
		vsSlice := vs.([]interface{})
		vsSlice = append(vsSlice, msg)
		holder.messages.Store(hash, vsSlice)
	} else {
		slice := make([]interface{}, 0)
		slice = append(slice, msg)
		holder.messages.Store(hash, slice)
	}
}

func (holder *FutureMessageHolder) getMessages(hash common.Hash) []interface{} {
	if vs, ok := holder.messages.Load(hash); ok {
		return vs.([]interface{})
	}
	return nil
}

func (holder *FutureMessageHolder) remove(hash common.Hash) {
	holder.messages.Delete(hash)
}

func (holder *FutureMessageHolder) forEach(f func(key common.Hash, arr []interface{}) bool) {
	holder.messages.Range(func(key, value interface{}) bool {
		arr := value.([]interface{})
		return f(key.(common.Hash), arr)
	})
}

func (holder *FutureMessageHolder) size() int {
	cnt := 0
	holder.forEach(func(key common.Hash, value []interface{}) bool {
		cnt += len(value)
		return true
	})
	return cnt
}

func (p *Processor) doAddOnChain(block *types.Block) (result int8) {

	bh := block.Header

	rlog := newRtLog("doAddOnChain")
	result = int8(p.MainChain.AddBlockOnChain("", block))

	rlog.log("height=%v, hash=%v, result=%v.", bh.Height, bh.Hash.ShortS(), result)
	castor := groupsig.DeserializeID(bh.Castor)
	tlog := newHashTraceLog("doAddOnChain", bh.Hash, castor)
	tlog.log("result=%v,castor=%v", result, castor.ShortS())

	if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
		p.futureRewardReqs.remove(block.Header.Hash)
	}

	return result

}

func (p *Processor) blockOnChain(h common.Hash) bool {
	return p.MainChain.HasBlock(h)
}

func (p *Processor) getBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	b := p.MainChain.QueryBlockHeaderByHash(hash)
	return b
}

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusCastMessage) {
	b := msg.BH
	stdLogger.Debugf("future verifyMsg receive cached! h=%v, hash=%v, preHash=%v\n", b.Height, b.Hash.ShortS(), b.PreHash.ShortS())

	p.futureVerifyMsgs.addMessage(b.PreHash, msg)
}

func (p *Processor) getFutureVerifyMsgs(hash common.Hash) []*model.ConsensusCastMessage {
	if vs := p.futureVerifyMsgs.getMessages(hash); vs != nil {
		ret := make([]*model.ConsensusCastMessage, len(vs))
		for idx, m := range vs {
			ret[idx] = m.(*model.ConsensusCastMessage)
		}
		return ret
	}
	return nil
}

func (p *Processor) removeFutureVerifyMsgs(hash common.Hash) {
	p.futureVerifyMsgs.remove(hash)
}

func (p *Processor) blockPreview(bh *types.BlockHeader) string {
	return fmt.Sprintf("hash=%v, height=%v, curTime=%v, preHash=%v, preTime=%v", bh.Hash.ShortS(), bh.Height, bh.CurTime, bh.PreHash.ShortS(), bh.CurTime.Add(-int64(bh.Elapsed)))
}

func (p *Processor) prepareForCast(sgi *StaticGroupInfo) {
	// Establish a group network
	p.NetServer.BuildGroupNet(sgi.GroupID.GetHexString(), sgi.GetMembers())
}

// VerifyBlock check if the block is legal, it will take the pre-block into consideration
func (p *Processor) VerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (ok bool, err error) {
	tlog := newMsgTraceLog("VerifyBlock", bh.Hash.ShortS(), "")
	defer func() {
		tlog.log("preHash=%v, height=%v, result=%v %v", bh.PreHash.ShortS(), bh.Height, ok, err)
		newBizLog("VerifyBlock").info("hash=%v, preHash=%v, height=%v, result=%v %v", bh.Hash.ShortS(), bh.PreHash.ShortS(), bh.Height, ok, err)
	}()
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}
	if preBH.Hash != bh.PreHash {
		err = fmt.Errorf("preHash error")
		return
	}

	ok2, group, err2 := p.isCastLegal(bh, preBH)
	if !ok2 {
		err = err2
		return
	}

	gpk := group.GroupPK
	pPubkey := p.getProposerPubKeyInBlock(bh)
	if pPubkey == nil {
		err = fmt.Errorf("getProposerPubKeyInBlock fail in VerifyBlock")
		return
	}
	pubArray := [2]groupsig.Pubkey{*pPubkey, gpk}
	aggSign := groupsig.DeserializeSign(bh.Signature)
	b := groupsig.VerifyAggregateSig(pubArray[:], bh.Hash.Bytes(), *aggSign)
	if !b {
		err = fmt.Errorf("signature verify fail")
		return
	}
	rsig := groupsig.DeserializeSign(bh.Random)
	b = groupsig.VerifySig(gpk, preBH.Random, *rsig)
	if !b {
		err = fmt.Errorf("random verify fail")
		return
	}
	ok = true
	return
}

// VerifyBlockHeader mainly check the group signature of the block
func (p *Processor) VerifyBlockHeader(bh *types.BlockHeader) (ok bool, err error) {
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}

	gid := groupsig.DeserializeID(bh.Group)
	gpk := p.getGroupPubKey(gid)
	ppk := p.getProposerPubKeyInBlock(bh)
	if ppk == nil {
		err = fmt.Errorf("getProposerPubKeyInBlock fail in VerifyBlockHeader")
		return
	}
	pkArray := [2]groupsig.Pubkey{*ppk, gpk}
	aggSign := groupsig.DeserializeSign(bh.Signature)
	b := groupsig.VerifyAggregateSig(pkArray[:], bh.Hash.Bytes(), *aggSign)
	if !b {
		err = fmt.Errorf("signature verify fail")
		return
	}
	ok = true
	return
}

// VerifyGroup check whether the give group is legal
func (p *Processor) VerifyGroup(g *types.Group) (ok bool, err error) {
	if len(g.Signature) == 0 {
		return false, fmt.Errorf("sign is empty")
	}
	mems := make([]groupsig.ID, len(g.Members))
	for idx, mem := range g.Members {
		mems[idx] = groupsig.DeserializeID(mem)
	}
	gInfo := &model.ConsensusGroupInitInfo{
		GI: model.ConsensusGroupInitSummary{
			Signature: *groupsig.DeserializeSign(g.Signature),
			GHeader:   g.Header,
		},
		Mems: mems,
	}

	// Check head and signature
	if _, ok, err := p.groupManager.checkGroupInfo(gInfo); ok {
		gpk := groupsig.DeserializePubkeyBytes(g.PubKey)
		gid := groupsig.NewIDFromPubkey(gpk).Serialize()
		if !bytes.Equal(gid, g.ID) {
			return false, fmt.Errorf("gid error, expect %v, receive %v", gid, g.ID)
		}
	} else {
		return false, err
	}
	ok = true
	return
}
