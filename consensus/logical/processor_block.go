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

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
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

	rlog.log("height=%v, hash=%v, result=%v.", bh.Height, bh.Hash, result)
	castor := groupsig.DeserializeID(bh.Castor)
	tlog := newHashTraceLog("doAddOnChain", bh.Hash, castor)
	tlog.log("result=%v,castor=%v", result, castor)

	if result == -1 {
		p.removeFutureVerifyMsgs(block.Header.Hash)
		p.rewardHandler.futureRewardReqs.remove(block.Header.Hash)
	}

	return result

}

func (p *Processor) blockOnChain(h common.Hash) bool {
	return p.MainChain.HasBlock(h)
}

func (p *Processor) GetBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	b := p.MainChain.QueryBlockHeaderByHash(hash)
	return b
}

func (p *Processor) addFutureVerifyMsg(msg *model.ConsensusCastMessage) {
	b := msg.BH
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

// VerifyBlock check if the block is legal, it will take the pre-block into consideration
func (p *Processor) VerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (ok bool, err error) {
	tLog := newHashTraceLog("VerifyBlock", bh.Hash, groupsig.ID{})
	defer func() {
		tLog.log("preHash=%v, height=%v, result=%v %v", bh.PreHash, bh.Height, ok, err)
	}()
	if bh.Hash != bh.GenHash() {
		err = core.ErrorBlockHash
		return
	}
	if preBH.Hash != bh.PreHash {
		err = fmt.Errorf("preHash error")
		return
	}

	minElapse := p.GetBlockMinElapse(bh.Height)
	if bh.Elapsed < minElapse {
		err = fmt.Errorf("min elapsed error %v", bh.Elapsed)
		return
	}
	if bh.Height > 1 && bh.CurTime.SinceMilliSeconds(preBH.CurTime) != int64(bh.Elapsed) {
		err = fmt.Errorf("elapsed error %v", bh.Elapsed)
		return
	}

	err = p.isCastLegal(bh, preBH)
	if err != nil {
		return
	}

	group := p.groupReader.getGroupHeaderBySeed(bh.Group)
	if group == nil {
		err = fmt.Errorf("get group is nil:%v", bh.Group)
		return
	}

	pPubkey := p.getProposerPubKeyInBlock(bh)
	if pPubkey == nil || !pPubkey.IsValid() {
		err = core.ErrPkNotExists
		return
	}
	gpk := group.gpk
	pubArray := [2]groupsig.Pubkey{*pPubkey, gpk}
	aggSign := groupsig.DeserializeSign(bh.Signature)
	b := groupsig.VerifyAggregateSig(pubArray[:], bh.Hash.Bytes(), *aggSign)
	if !b {
		err = core.ErrorGroupSign
		return
	}
	randomSig := groupsig.DeserializeSign(bh.Random)
	b = groupsig.VerifySig(gpk, preBH.Random, *randomSig)
	if !b {
		err = core.ErrorRandomSign
		return
	}
	ok = true
	return
}

// VerifyBlockSign mainly check the verifyGroup signature of the block
func (p *Processor) VerifyBlockSign(bh *types.BlockHeader) (ok bool, err error) {
	if bh.Hash != bh.GenHash() {
		err = fmt.Errorf("block hash error")
		return
	}

	group := p.groupReader.getGroupHeaderBySeed(bh.Group)
	if group == nil {
		err = core.ErrGroupNotExists
		return
	}

	gpk := group.gpk

	ppk := p.getProposerPubKeyInBlock(bh)
	if ppk == nil || !ppk.IsValid() {
		err = core.ErrPkNil
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

// VerifyBlockHeaders checks if the group is legal and the group signature is correct
func (p *Processor) VerifyBlockHeaders(pre, bh *types.BlockHeader) (ok bool, err error) {
	if bh.PreHash != pre.Hash {
		return false, fmt.Errorf("prehash not equal to pre")
	}
	gSeed := p.calcVerifyGroup(pre, bh.Height)
	if gSeed != bh.Group {
		return false, fmt.Errorf("verify group error: expect %v, infact %v", gSeed, bh.Group)
	}
	return p.VerifyBlockSign(bh)
}
