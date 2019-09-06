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
	*rpcBaseImpl
}

func (api *RpcDevImpl) Namespace() string {
	return "Dev"
}

func (api *RpcDevImpl) Version() string {
	return "1"
}

func (api *RpcDevImpl) ProposalTotalStake(height uint64) (uint64, error) {
	if core.MinerManagerImpl == nil {
		return 0, fmt.Errorf("status error")

	} else {
		totalStake := core.MinerManagerImpl.GetProposalTotalStake(height)
		return totalStake, nil
	}
}

// ConnectedNodes query the information of the connected node
func (api *RpcDevImpl) ConnectedNodes() ([]ConnInfo, error) {

	nodes := network.GetNetInstance().ConnInfo()
	conns := make([]ConnInfo, 0)
	for _, n := range nodes {
		conns = append(conns, ConnInfo{ID: n.ID, IP: n.IP, TCPPort: n.Port})
	}
	return conns, nil
}

// TransPool query buffer transaction information
func (api *RpcDevImpl) TransPool() ([]Transactions, error) {
	transactions := core.BlockChainImpl.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.Hex(),
			Source: v.Source.AddrPrefixString(),
			Target: v.Target.AddrPrefixString(),
			Value:  v.Value.String(),
		})
	}

	return transList, nil
}

func (api *RpcDevImpl) BalanceByHeight(height uint64, account string) (float64, error) {
	if !common.ValidateAddress(strings.TrimSpace(account)) {
		return 0, fmt.Errorf("wrong account address format")
	}
	db, err := core.BlockChainImpl.AccountDBAt(height)
	if err != nil {
		return 0, fmt.Errorf("this height is invalid")
	}
	b := db.GetBalance(common.StringToAddress(account))

	balance := common.RA2TAS(b.Uint64())
	return balance, nil
}

// get transaction by hash
func (api *RpcDevImpl) GetTransaction(hash string) (map[string]interface{}, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return nil, fmt.Errorf("wrong hash format")
	}
	transaction := core.BlockChainImpl.GetTransactionByHash(false, common.HexToHash(hash))
	if transaction == nil {
		return nil, nil
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

	return detail, nil
}

func (api *RpcDevImpl) GetBlocks(from uint64, end uint64) ([]*Block, error) {
	if end < from {
		return nil, fmt.Errorf("start must be bigger than the end,begin is %v,end is %v", from, end)
	}
	if end-from > 50 {
		return nil, fmt.Errorf("end can't be 50 more than the start,begin is %v,end is %v", from, end)
	}
	maxHeight := core.BlockChainImpl.QueryTopBlock().Height
	if from > maxHeight {
		from = maxHeight
	}
	if end > maxHeight {
		end = maxHeight
	}
	blocks := make([]*Block, 0)
	var preBH *types.BlockHeader
	for h := from; h <= end; h++ {
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
	return blocks, nil
}

func (api *RpcDevImpl) GetTopBlock() (map[string]interface{}, error) {
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
	blockDetail["group_id"] = bh.Group.Hex()
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(b.Transactions)
	blockDetail["elapsed"] = bh.Elapsed
	blockDetail["tps"] = math.Round(float64(len(b.Transactions)) / float64(bh.Elapsed*1e3))

	blockDetail["tx_pool_count"] = len(core.BlockChainImpl.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = core.BlockChainImpl.GetTransactionPool().TxNum()
	blockDetail["miner_id"] = mediator.Proc.GetMinerID().GetAddrString()
	return blockDetail, nil
}

func (api *RpcDevImpl) WorkGroupNum() (int, error) {
	groups := api.gr.ActiveGroupCount()
	return groups, nil
}

func (api *RpcDevImpl) GetGroupsAfter(height uint64) ([]*Group, error) {
	api2 := &RpcExplorerImpl{rpcBaseImpl: api.rpcBaseImpl}
	return api2.ExplorerGroupsAfter(height)
}

func (api *RpcDevImpl) GetCurrentWorkGroup() ([]*Group, error) {
	height := core.BlockChainImpl.Height()
	return api.GetWorkGroup(height)
}

func (api *RpcDevImpl) GetWorkGroup(height uint64) ([]*Group, error) {
	groups := api.gr.GetActivatedGroupsAt(height)
	ret := make([]*Group, 0)
	for _, group := range groups {
		if group != nil {
			g := convertGroup(group)
			ret = append(ret, g)
		}

	}
	return ret, nil
}

// CastStat cast block statistics
func (api *RpcDevImpl) CastStat(begin uint64, end uint64) (map[string]map[string]int32, error) {
	if end < begin {
		return nil, fmt.Errorf("start must be bigger than the end,begin is %v,end is %v", begin, end)
	}
	if end-begin > 100 {
		return nil, fmt.Errorf("end can't be 100 more than the start,begin is %v,end is %v", begin, end)
	}
	proposerStat := make(map[string]int32)
	groupStat := make(map[string]int32)
	chain := core.BlockChainImpl
	maxHeight := chain.QueryTopBlock().Height
	if begin > maxHeight {
		begin = maxHeight
	}
	if end > maxHeight {
		end = maxHeight
	}

	for h := begin; h <= end; h++ {
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
		g := bh.Group.Hex()
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
		pmap[id.GetAddrString()] = v
	}
	for key, v := range groupStat {
		gmap[key] = v
	}
	ret := make(map[string]map[string]int32)
	ret["proposer"] = pmap
	ret["group"] = gmap
	return ret, nil
}

func (api *RpcDevImpl) NodeInfo() (*NodeInfo, error) {
	ni := &NodeInfo{}
	p := mediator.Proc
	ni.ID = p.GetMinerID().GetAddrString()
	balance := core.BlockChainImpl.GetBalance(common.StringToAddress(p.GetMinerID().GetAddrString()))
	ni.Balance = common.RA2TAS(balance.Uint64())
	if !p.Ready() {
		ni.Status = "node not ready"
	} else {
		ni.Status = "running"
		ni.NType, ni.MortGages = getMorts(p)
		//wg, ag := p.GetJoinedWorkGroupNums()
		//ni.WGroupNum = wg
		//ni.AGroupNum = ag

		ni.TxPoolNum = int(core.BlockChainImpl.GetTransactionPool().TxNum())

	}
	return ni, nil
}

func (api *RpcDevImpl) Dashboard() (*Dashboard, error) {
	blockHeight := core.BlockChainImpl.Height()
	groupHeight := api.gr.Height()
	workNum := api.gr.ActiveGroupCount()
	nodeResult, _ := api.NodeInfo()
	consResult, _ := api.ConnectedNodes()
	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult,
		Conns:       consResult,
	}
	return dash, nil
}

func (api *RpcDevImpl) BlockDetail(h string) (*BlockDetail, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return nil, fmt.Errorf("wrong param format")
	}
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return nil, nil
	}
	bh := b.Header
	block := convertBlockHeader(b)

	preBH := chain.QueryBlockHeaderByHash(bh.PreHash)
	block.Qn = bh.TotalQN - preBH.TotalQN

	castor := block.Castor.GetAddrString()

	trans := make([]Transaction, 0)
	rewardTxs := make([]RewardTransaction, 0)
	minerReward := make(map[string]*MinerRewardBalance)
	uniqueRewardBlockHash := make(map[common.Hash]uint64)
	minerVerifyBlockHash := make(map[string][]common.Hash)
	blockVerifyReward := make(map[common.Hash]uint64)

	minerReward[castor] = genMinerBalance(block.Castor, bh)

	for _, raw := range b.Transactions {
		tx := types.NewTransaction(raw, raw.GenHash())
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
				if _, ok := minerReward[tid.GetAddrString()]; !ok {
					minerReward[tid.GetAddrString()] = genMinerBalance(tid, bh)
				}
				if !btx.Success {
					continue
				}
				if hs, ok := minerVerifyBlockHash[tid.GetAddrString()]; ok {
					find := false
					for _, h := range hs {
						if h == btx.BlockHash {
							find = true
							break
						}
					}
					if !find {
						hs = append(hs, btx.BlockHash)
						minerVerifyBlockHash[tid.GetAddrString()] = hs
					}
				} else {
					hs = make([]common.Hash, 0)
					hs = append(hs, btx.BlockHash)
					minerVerifyBlockHash[tid.GetAddrString()] = hs
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
	if rewardTx := chain.GetRewardManager().GetRewardTransactionByBlockHash(bh.Hash); rewardTx != nil {
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
	return bd, nil
}

func (api *RpcDevImpl) BlockReceipts(h string) (*BlockReceipt, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return nil, fmt.Errorf("wrong param format")
	}
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return nil, fmt.Errorf("block not found")
	}

	evictedReceipts := make([]*types.Receipt, 0)
	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range b.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.GenHash())
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}
	br := &BlockReceipt{EvictedReceipts: evictedReceipts, Receipts: receipts}
	return br, nil
}

// MonitorBlocks monitoring platform calls block sync
func (api *RpcDevImpl) MonitorBlocks(begin, end uint64) ([]*BlockDetail, error) {
	if end < begin {
		return nil, fmt.Errorf("start must be bigger than the end,begin is %v,end is %v", begin, end)
	}
	if end-begin > 50 {
		return nil, fmt.Errorf("end can't be 50 more than the start,begin is %v,end is %v", begin, end)
	}
	chain := core.BlockChainImpl
	maxHeight := chain.QueryTopBlock().Height
	if begin > maxHeight {
		begin = maxHeight
	}
	if end > maxHeight {
		end = maxHeight
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
			trans = append(trans, *convertTransaction(types.NewTransaction(tx, tx.GenHash())))
		}

		bd := &BlockDetail{
			Block: *block,
			Trans: trans,
		}
		pre = b
		blocks = append(blocks, bd)
	}
	return blocks, nil
}

func (api *RpcDevImpl) MonitorNodeInfo() (*NodeInfo, error) {
	bh := core.BlockChainImpl.Height()
	gh := api.gr.Height()

	ni := &NodeInfo{}

	ret, err := api.NodeInfo()
	if err != nil {
		ni = ret
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
	return ni, nil
}

func (api *RpcDevImpl) MonitorAllMiners() (map[string]interface{}, error) {
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
	return data, nil
}

func (api *RpcDevImpl) GetLivedGroup(height uint64) ([]*Group, error) {
	groups := api.gr.GetLivedGroupsAt(height)
	ret := make([]*Group, 0)
	for _, group := range groups {
		if group != nil {
			g := convertGroup(group)
			ret = append(ret, g)
		}

	}
	return ret, nil
}

func (api *RpcDevImpl) BlockDropInfo(b, e uint64) (map[string]interface{}, error) {
	if e == 0 {
		e = core.BlockChainImpl.Height()
	}
	if b > e {
		return nil, fmt.Errorf("begin larger than end")
	}
	heights := core.BlockChainImpl.ScanBlockHeightsInRange(b, e)
	drops := make([]uint64, 0)
	for i, h := 0, b; h <= e && i < len(heights); h++ {
		if heights[i] == h {
			i++
		} else {
			drops = append(drops, h)
		}
	}
	dropRate := float64(len(drops)) / float64(e-b+1)
	ret := make(map[string]interface{})
	ret["expect_heights"] = e - b + 1
	ret["real_heights"] = len(heights)
	ret["drop_rate"] = dropRate
	ret["drops"] = drops
	return ret, nil
}

func (api *RpcDevImpl) JumpBlockInfo(begin, end uint64) (map[string][2]int, error) {
	pre := core.BlockChainImpl.QueryBlockHeaderFloor(begin)
	if end == 0 {
		end = core.BlockChainImpl.Height()
	}
	ret := make(map[string][2]int)
	for h := pre.Height + 1; h < end; h++ {
		bh := core.BlockChainImpl.QueryBlockHeaderByHeight(h)
		if bh == nil {
			g := mediator.Proc.CalcVerifyGroup(pre, h)
			if v, ok := ret[g.Hex()]; ok {
				v[1]++
				ret[g.Hex()] = v
			} else {
				v = [2]int{0, 1}
				ret[g.Hex()] = v
			}
		} else {
			if v, ok := ret[bh.Group.Hex()]; ok {
				v[0]++
				ret[bh.Group.Hex()] = v
			} else {
				v = [2]int{1, 0}
				ret[bh.Group.Hex()] = v
			}
			pre = bh
		}
	}
	return ret, nil
}
