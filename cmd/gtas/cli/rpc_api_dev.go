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

package cli

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/pmylund/sortutil"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
)

// RpcDevImpl provides api functions for those develop chain features.
// It is mainly for debug or test use
type RpcDevImpl struct {
}

func (api *RpcDevImpl) Namespace() string {
	return "Dev"
}

func (api *RpcDevImpl) Version() string {
	return "1"
}

// ConnectedNodes query the information of the connected node
func (api *RpcDevImpl) ConnectedNodes() (*Result, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{ID: n.ID, IP: n.IP, TCPPort: n.Port})
	}
	return successResult(conns)
}

// TransPool query buffer transaction information
func (api *RpcDevImpl) TransPool() (*Result, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.Hex(),
			Source: v.Source.Hex(),
			Target: v.Target.Hex(),
			Value:  v.Value.String(),
		})
	}

	return successResult(transList)
}

// get transaction by hash
func (api *RpcDevImpl) GetTransaction(hash string) (*Result, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return failResult("Wrong hash format")
	}
	transaction := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(hash))
	if transaction == nil {
		return failResult("transaction not exists")
	}
	detail := make(map[string]interface{})
	detail["hash"] = hash
	if transaction.Source != nil {
		detail["source"] = transaction.Source.Hash().Hex()
	}
	if transaction.Target != nil {
		detail["target"] = transaction.Target.Hash().Hex()
	}
	detail["value"] = transaction.Value

	return successResult(detail)
}

func (api *RpcDevImpl) GetBlocks(from uint64, to uint64) (*Result, error) {
	if from < to {
		return failResult("param error")
	}
	blocks := make([]*Block, 0)
	var preBH *types.BlockHeader
	for h := from; h <= to; h++ {
		b := core.BlockChainImpl.QueryBlockByHeight(h)
		if b != nil {
			block := convertBlockHeader(b)
			if preBH == nil {
				preBH = core.BlockChainImpl.QueryBlockHeaderByHash(b.Header.PreHash)
			}
			if preBH == nil {
				block.Qn = b.Header.TotalQN
			} else {
				block.Qn = b.Header.TotalQN - preBH.TotalQN
			}
			preBH = b.Header
			blocks = append(blocks, block)
		}
	}
	return successResult(blocks)
}

func (api *RpcDevImpl) GetTopBlock() (*Result, error) {
	bh := core.BlockChainImpl.QueryTopBlock()
	b := core.BlockChainImpl.QueryBlockByHash(bh.Hash)
	bh = b.Header

	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime().Local().Format("2006-01-02 15:04:05")
	blockDetail["total_qn"] = bh.TotalQN
	blockDetail["cur_time"] = bh.CurTime.Local().Format("2006-01-02 15:04:05")
	blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupID)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(b.Transactions)
	blockDetail["elapsed"] = bh.Elapsed
	blockDetail["tps"] = math.Round(float64(len(b.Transactions)) / float64(bh.Elapsed))

	blockDetail["tx_pool_count"] = len(core.BlockChainImpl.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = core.BlockChainImpl.GetTransactionPool().TxNum()
	blockDetail["miner_id"] = mediator.Proc.GetPubkeyInfo().ID.ShortS()
	return successResult(blockDetail)
}

func (api *RpcDevImpl) WorkGroupNum(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroups(height)
	return successResult(groups)
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.ID != nil && len(g.ID) != 0 {
		gmap["group_id"] = groupsig.DeserializeID(g.ID).GetHexString()
		gmap["g_hash"] = g.Header.Hash.Hex()
	}
	gmap["parent"] = groupsig.DeserializeID(g.Header.Parent).GetHexString()
	gmap["pre"] = groupsig.DeserializeID(g.Header.PreGroup).GetHexString()
	gmap["begin_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	gmap["create_height"] = g.Header.CreateHeight
	gmap["create_time"] = g.Header.BeginTime
	gmap["mem_size"] = len(g.Members)
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeID(mem).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	gmap["extends"] = g.Header.Extends
	return gmap
}

func (api *RpcDevImpl) GetGroupsAfter(height uint64) (*Result, error) {
	groups := core.GroupChainImpl.GetGroupsAfterHeight(height, math.MaxInt64)

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *RpcDevImpl) GetCurrentWorkGroup() (*Result, error) {
	height := core.BlockChainImpl.Height()
	return api.GetWorkGroup(height)
}

func (api *RpcDevImpl) GetWorkGroup(height uint64) (*Result, error) {
	groups := mediator.Proc.GetCastQualifiedGroupsFromChain(height)
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

// CastStat cast block statistics
func (api *RpcDevImpl) CastStat(begin uint64, end uint64) (*Result, error) {
	proposerStat := make(map[string]int32)
	groupStat := make(map[string]int32)

	chain := core.BlockChainImpl
	if end == 0 {
		end = chain.QueryTopBlock().Height
	}

	for h := begin; h < end; h++ {
		b := chain.QueryBlockByHeight(h)
		if b == nil {
			continue
		}
		bh := b.Header
		p := string(bh.Castor)
		if v, ok := proposerStat[p]; ok {
			proposerStat[p] = v + 1
		} else {
			proposerStat[p] = 1
		}
		g := string(bh.GroupID)
		if v, ok := groupStat[g]; ok {
			groupStat[g] = v + 1
		} else {
			groupStat[g] = 1
		}
	}
	pmap := make(map[string]int32)
	gmap := make(map[string]int32)

	for key, v := range proposerStat {
		id := groupsig.DeserializeID([]byte(key))
		pmap[id.GetHexString()] = v
	}
	for key, v := range groupStat {
		id := groupsig.DeserializeID([]byte(key))
		gmap[id.GetHexString()] = v
	}
	ret := make(map[string]map[string]int32)
	ret["proposer"] = pmap
	ret["group"] = gmap
	return successResult(ret)
}

func (api *RpcDevImpl) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := mediator.Proc
	ni.ID = p.GetMinerID().GetHexString()
	balance := core.BlockChainImpl.GetBalance(common.HexToAddress(p.GetMinerID().GetHexString()))
	ni.Balance = common.RA2TAS(balance.Uint64())
	if !p.Ready() {
		ni.Status = "node not ready"
	} else {
		ni.Status = "running"
		morts := make([]MortGage, 0)
		t := "--"
		addr := common.BytesToAddress(p.GetMinerID().Serialize())
		proposalInfo := core.MinerManagerImpl.GetLatestMiner(addr, types.MinerTypeProposal)
		if proposalInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(proposalInfo))
			if proposalInfo.IsActive() {
				t = "proposal role"
			}
		}
		verifyInfo := core.MinerManagerImpl.GetLatestMiner(addr, types.MinerTypeVerify)
		if verifyInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(verifyInfo))
			if verifyInfo.IsActive() {
				t += " verify role"
			}
		}
		ni.NType = t
		ni.MortGages = morts

		wg, ag := p.GetJoinedWorkGroupNums()
		ni.WGroupNum = wg
		ni.AGroupNum = ag

		ni.TxPoolNum = int(core.BlockChainImpl.GetTransactionPool().TxNum())

	}
	return successResult(ni)

}

func (api *RpcDevImpl) Dashboard() (*Result, error) {
	blockHeight := core.BlockChainImpl.Height()
	groupHeight := core.GroupChainImpl.Height()
	workNum := len(mediator.Proc.GetCastQualifiedGroups(blockHeight))
	nodeResult, _ := api.NodeInfo()
	consResult, _ := api.ConnectedNodes()
	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult.Data.(*NodeInfo),
		Conns:       consResult.Data.([]ConnInfo),
	}
	return successResult(dash)
}

func (api *RpcDevImpl) BlockDetail(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong param format")
	}
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return successResult(nil)
	}
	bh := b.Header
	block := convertBlockHeader(b)

	preBH := chain.QueryBlockHeaderByHash(bh.PreHash)
	block.Qn = bh.TotalQN - preBH.TotalQN

	castor := block.Castor.GetHexString()

	trans := make([]Transaction, 0)
	rewardTxs := make([]RewardTransaction, 0)
	minerReward := make(map[string]*MinerRewardBalance)
	uniqueRewardBlockHash := make(map[common.Hash]uint64)
	minerVerifyBlockHash := make(map[string][]common.Hash)
	blockVerifyReward := make(map[common.Hash]uint64)

	minerReward[castor] = genMinerBalance(block.Castor, bh)

	for _, tx := range b.Transactions {
		if tx.IsReward() {
			btx := *convertRewardTransaction(tx)
			receipt := chain.GetTransactionPool().GetReceipt(tx.Hash)
			if receipt == nil {
				btx.StatusReport = "get status nil"
			} else {
				if receipt.Success() {
					btx.StatusReport = "success"
					btx.Success = true
				} else {
					btx.StatusReport = "fail"
				}
			}

			rewardTxs = append(rewardTxs, btx)
			blockVerifyReward[btx.BlockHash] = btx.Value
			for _, tid := range btx.TargetIDs {
				if _, ok := minerReward[tid.GetHexString()]; !ok {
					minerReward[tid.GetHexString()] = genMinerBalance(tid, bh)
				}
				if !btx.Success {
					continue
				}
				if hs, ok := minerVerifyBlockHash[tid.GetHexString()]; ok {
					find := false
					for _, h := range hs {
						if h == btx.BlockHash {
							find = true
							break
						}
					}
					if !find {
						hs = append(hs, btx.BlockHash)
						minerVerifyBlockHash[tid.GetHexString()] = hs
					}
				} else {
					hs = make([]common.Hash, 0)
					hs = append(hs, btx.BlockHash)
					minerVerifyBlockHash[tid.GetHexString()] = hs
				}
			}
			if btx.Success {
				block := chain.QueryBlockByHash(btx.BlockHash)
				if block != nil {
					uniqueRewardBlockHash[btx.BlockHash] = block.Header.Height
				}
			}
		} else {
			trans = append(trans, *convertTransaction(tx))
		}
	}

	rm := chain.GetRewardManager()

	mbs := make([]*MinerRewardBalance, 0)
	for id, mb := range minerReward {
		mb.Explain = ""
		increase := uint64(0)
		if id == castor {
			mb.Proposal = true
			var packedRewards uint64
			for _, height := range uniqueRewardBlockHash {
				share := rm.CalculateCastRewardShare(height, 0)
				packedRewards += share.ForRewardTxPacking
			}
			mb.PackRewardTx = len(uniqueRewardBlockHash)
			share := rm.CalculateCastRewardShare(bh.Height, bh.GasFee)
			increase += share.ForBlockProposal
			increase += packedRewards
			increase += share.FeeForProposer
			mb.Explain = fmt.Sprintf("proposal, pack %v bouns-txs", mb.PackRewardTx)
		}
		if hs, ok := minerVerifyBlockHash[id]; ok {
			for _, h := range hs {
				increase += blockVerifyReward[h]
			}
			mb.VerifyBlock = len(hs)
			mb.Explain = fmt.Sprintf("%v, verify %v blocks", mb.Explain, mb.VerifyBlock)
		}
		mb.ExpectBalance = new(big.Int).SetUint64(mb.PreBalance.Uint64() + increase)
		mbs = append(mbs, mb)
	}

	var genReward *RewardTransaction
	if rewardTx := chain.GetRewardManager().GetRewardTransactionByBlockHash(bh.Hash.Bytes()); rewardTx != nil {
		genReward = convertRewardTransaction(rewardTx)
	}

	bd := &BlockDetail{
		Block:         *block,
		GenRewardTx:   genReward,
		Trans:         trans,
		BodyRewardTxs: rewardTxs,
		MinerReward:   mbs,
		PreTotalQN:    preBH.TotalQN,
	}
	return successResult(bd)
}

func (api *RpcDevImpl) BlockReceipts(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong param format")
	}
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return failResult("block not found")
	}

	evictedReceipts := make([]*types.Receipt, 0)
	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range b.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.Hash)
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}
	br := &BlockReceipt{EvictedReceipts: evictedReceipts, Receipts: receipts}
	return successResult(br)
}

// MonitorBlocks monitoring platform calls block sync
func (api *RpcDevImpl) MonitorBlocks(begin, end uint64) (*Result, error) {
	chain := core.BlockChainImpl
	if begin > end {
		end = begin
	}
	var pre *types.Block

	blocks := make([]*BlockDetail, 0)
	for h := begin; h <= end; h++ {
		b := chain.QueryBlockCeil(h)
		if b == nil {
			continue
		}
		bh := b.Header
		block := convertBlockHeader(b)

		if pre == nil {
			pre = chain.QueryBlockByHash(bh.PreHash)
		}
		if pre == nil {
			block.Qn = bh.TotalQN
		} else {
			block.Qn = bh.TotalQN - pre.Header.TotalQN
		}

		trans := make([]Transaction, 0)

		for _, tx := range b.Transactions {
			trans = append(trans, *convertTransaction(tx))
		}

		bd := &BlockDetail{
			Block: *block,
			Trans: trans,
		}
		pre = b
		blocks = append(blocks, bd)
	}
	return successResult(blocks)
}

func (api *RpcDevImpl) MonitorNodeInfo() (*Result, error) {
	bh := core.BlockChainImpl.Height()
	gh := core.GroupChainImpl.LastGroup().GroupHeight

	ni := &NodeInfo{}

	ret, _ := api.NodeInfo()
	if ret != nil && ret.IsSuccess() {
		ni = ret.Data.(*NodeInfo)
	}
	ni.BlockHeight = bh
	ni.GroupHeight = gh
	if ni.MortGages != nil {
		for _, mg := range ni.MortGages {
			if mg.Type == "proposal role" {
				ni.VrfThreshold = mediator.Proc.GetVrfThreshold(common.TAS2RA(mg.Stake))
				break
			}
		}
	}
	return successResult(ni)
}

func (api *RpcDevImpl) MonitorAllMiners() (*Result, error) {
	miners := mediator.Proc.GetAllMinerDOs()
	totalStake := uint64(0)
	maxStake := uint64(0)
	for _, m := range miners {
		if m.IsActive() && m.IsProposal() {
			totalStake += m.Stake
			if maxStake < m.Stake {
				maxStake = m.Stake
			}
		}
	}
	sortutil.AscByField(miners, "Stake")
	data := make(map[string]interface{})
	data["miners"] = miners
	data["maxStake"] = maxStake
	data["totalStake"] = totalStake
	return successResult(data)
}
