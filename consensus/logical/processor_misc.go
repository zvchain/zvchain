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

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
)

func (p *Processor) getVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) getSelfMinerDO() *model.SelfMinerDO {
	md := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerDO = *md
	}
	return p.mi
}

func (p *Processor) canPropose() bool {
	miner := p.minerReader.getLatestProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
	return miner.CanPropose()
}

// CalcBlockHeaderQN calculates the qn value of the given block header
func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := base.VRFProve(bh.ProveValue)
	castor := groupsig.DeserializeID(bh.Castor)
	pre := p.MainChain.QueryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return 0
	}
	miner := p.minerReader.getProposeMinerByHeight(castor, pre.Height)
	if miner == nil {
		stdLogger.Warnf("CalcBHQN getMiner nil id=%v, bh=%v", castor, bh.Hash)
		return 0
	}
	totalStake := p.minerReader.getTotalStake(pre.Height)
	_, qn := vrfSatisfy(pi, miner.Stake, totalStake)
	return qn
}

// GetVrfThreshold returns the vrf threshold of current node under the specified stake
func (p *Processor) GetVrfThreshold(stake uint64) float64 {
	totalStake := p.minerReader.getTotalStake(p.MainChain.Height())
	if totalStake == 0 {
		return 0
	}
	vs := vrfThreshold(stake, totalStake)
	f, _ := vs.Float64()
	return f
}

// GetAllMinerDOs returns all available miner infos
func (p *Processor) GetAllMinerDOs() []*model.MinerDO {
	h := p.MainChain.Height()
	dos := make([]*model.MinerDO, 0)
	miners := p.minerReader.getAllMinerDOByType(types.MinerTypeProposal, h)
	dos = append(dos, miners...)

	miners = p.minerReader.getAllMinerDOByType(types.MinerTypeVerify, h)
	dos = append(dos, miners...)
	return dos
}

func (p *Processor) VerifyRewardTransaction(tx *types.Transaction) (ok bool, err error) {
	signBytes := tx.Sign
	if len(signBytes) < common.SignLength {
		return false, fmt.Errorf("not enough bytes for reward signature, sign =%v", signBytes)
	}

	gSeed, targetIds, blockHash, packFee, err := p.MainChain.GetRewardManager().ParseRewardTransaction(tx)
	if err != nil {
		return false, fmt.Errorf("failed to parse reward transaction, err =%s", err)
	}

	var bh *types.BlockHeader
	if bh = p.MainChain.QueryBlockHeaderByHash(blockHash); bh == nil {
		return false, fmt.Errorf("chain does not have this block, block hash=%v", blockHash)
	}
	if gSeed != bh.Group {
		return false, fmt.Errorf("group seed not equal to the block verifier")
	}
	rewardShare := p.MainChain.GetRewardManager().CalculateCastRewardShare(bh.Height, bh.GasFee)

	if rewardShare.ForRewardTxPacking != packFee.Uint64() {
		return false, fmt.Errorf("pack fee error: receive %v, expect %v", packFee.Uint64(), rewardShare.ForRewardTxPacking)
	}
	verifyRewards := rewardShare.TotalForVerifier()
	if verifyRewards/uint64(len(targetIds)) != tx.Value.Uint64() {
		return false, fmt.Errorf("invalid verify reward, value=%v", tx.Value)
	}

	group := p.groupReader.getGroupBySeed(gSeed)
	if group == nil {
		return false, common.ErrGroupNil
	}
	marks := make([]int, group.memberSize())
	for _, id := range targetIds {
		mid := groupsig.DeserializeID(id)
		if !mid.IsValid() {
			return false, fmt.Errorf("invalid group member,id=%v", mid)
		}
		idx := group.getMemberIndex(mid)
		if idx < 0 {
			return false, fmt.Errorf("member id not exist:%v", mid)
		}
		// duplication check
		if marks[idx] == 0 {
			marks[idx] = 1
		} else {
			return false, fmt.Errorf("duplicated target id:%v at %v", mid, targetIds)
		}
	}

	gpk := group.header.gpk
	gSign := groupsig.DeserializeSign(signBytes[0:33]) //size of groupsig == 33
	if !groupsig.VerifySig(gpk, tx.Hash.Bytes(), *gSign) {
		return false, fmt.Errorf("verify reward sign fail, blockHash=%v, gSign=%v, txHash=%v, gpk=%v, tx=%+v", blockHash, gSign.GetHexString(), tx.Hash.Hex(), gpk.GetHexString(), tx.RawTransaction)
	}

	return true, nil
}

// GetBlockMinElapse return the min elapsed milliseconds for blocks
func (p *Processor) GetBlockMinElapse(height uint64) int32 {

	var result int32

	currentEpoch := types.EpochAt(height)
	startEpoch := currentEpoch.Add(-chasingSeekEpochs)
	endEpoch := currentEpoch.Add(-1)
	if startEpoch.End() == 0 { // not enter chasing mode in first {chasingSeekEpochs} epochs
		stdLogger.Debugf("epoch not enough. current epoch: %d, min elapse: %d", currentEpoch, normalMinElapse)
		return normalMinElapse
	}

	lastBlock := p.MainChain.QueryBlockHeaderCeil(endEpoch.Start())
	if lastBlock == nil || lastBlock.Height > endEpoch.End() {
		stdLogger.Debugf("last block not in end epoch, consider as chasing. current epoch: %d, min elapse: %d", currentEpoch, normalMinElapse)
		return chasingMinElapse
	}

	if v, ok := p.cachedMinElapseByEpoch.Get(lastBlock.Hash); ok {
		return v.(int32)
	}

	firstBlock := p.MainChain.QueryBlockHeaderCeil(startEpoch.Start())
	if firstBlock == nil || firstBlock.Height > startEpoch.End() {
		stdLogger.Debugf("first block not in start epoch, consider as chasing. current epoch: %d, min elapse: %d", currentEpoch, normalMinElapse)
		return chasingMinElapse
	}

	realCount := lastBlock.Height - firstBlock.Height

	spends := uint64(lastBlock.CurTime.SinceMilliSeconds(firstBlock.CurTime))
	if spends > realCount*uint64(normalMinElapse) {
		result = chasingMinElapse
	} else {
		result = normalMinElapse
	}
	p.cachedMinElapseByEpoch.Add(lastBlock.Hash, result)
	stdLogger.Debugf("current epoch: %d, min elapse: %d. first block height: %d, last block height: %d, spends: %d, real block count %d",
		currentEpoch, result, firstBlock.Height, lastBlock.Height, spends, realCount)
	return result
}
