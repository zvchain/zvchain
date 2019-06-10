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

package cli

import (
	"github.com/pmylund/sortutil"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
)

// ExplorerAccount is used in the blockchain browser to query account information
func (api *GtasAPI) ExplorerAccount(hash string) (*Result, error) {

	accoundDb := core.BlockChainImpl.LatestStateDB()
	if accoundDb == nil {
		return nil, nil
	}
	address := common.HexToAddress(hash)
	if !accoundDb.Exist(address) {
		return failResult("Account not Exist!")
	}
	account := ExplorerAccount{}
	account.Balance = accoundDb.GetBalance(address)
	account.Nonce = accoundDb.GetNonce(address)
	account.CodeHash = accoundDb.GetCodeHash(address).Hex()
	account.Code = string(accoundDb.GetCode(address)[:])
	account.Type = 0
	if len(account.Code) > 0 {
		account.Type = 1
		account.StateData = make(map[string]interface{})

		iter := accoundDb.DataIterator(common.HexToAddress(hash), "")
		for iter.Next() {
			k := string(iter.Key[:])
			v := string(iter.Value[:])
			account.StateData[k] = v

		}
	}
	return successResult(account)
}

// ExplorerBlockDetail is used in the blockchain browser to query block details
func (api *GtasAPI) ExplorerBlockDetail(height uint64) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return failResult("QueryBlock error")
	}
	block := convertBlockHeader(b)

	trans := make([]Transaction, 0)

	for _, tx := range b.Transactions {
		trans = append(trans, *convertTransaction(tx))
	}

	evictedReceipts := make([]*types.Receipt, 0)

	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range b.Transactions {
		wrapper := chain.GetTransactionPool().GetReceipt(tx.Hash)
		if wrapper != nil {
			receipts[i] = wrapper
		}
	}

	bd := &ExplorerBlockDetail{
		BlockDetail:     BlockDetail{Block: *block, Trans: trans},
		EvictedReceipts: evictedReceipts,
		Receipts:        receipts,
	}
	return successResult(bd)
}

// ExplorerGroupsAfter is used in the blockchain browser to
// query groups after the specified height
func (api *GtasAPI) ExplorerGroupsAfter(height uint64) (*Result, error) {
	groups := core.GroupChainImpl.GetGroupsAfterHeight(height, common.MaxInt64)

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := explorerConvertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func explorerConvertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.ID != nil && len(g.ID) != 0 {
		gmap["id"] = groupsig.DeserializeID(g.ID).GetHexString()
		gmap["hash"] = g.Header.Hash
	}
	gmap["parent_id"] = groupsig.DeserializeID(g.Header.Parent).GetHexString()
	gmap["pre_id"] = groupsig.DeserializeID(g.Header.PreGroup).GetHexString()
	gmap["begin_time"] = g.Header.BeginTime
	gmap["create_height"] = g.Header.CreateHeight
	gmap["work_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeID(mem).GetHexString()
		mems = append(mems, memberStr)
	}
	gmap["members"] = mems
	return gmap
}

// ExplorerBlockBonus export bonus transaction by block height
func (api *GtasAPI) ExplorerBlockBonus(height uint64) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return failResult("nil block")
	}
	bh := b.Header

	ret := &ExploreBlockBonus{
		ProposalID: groupsig.DeserializeID(bh.Castor).GetHexString(),
	}
	bonusNum := uint64(0)
	if b.Transactions != nil {
		for _, tx := range b.Transactions {
			if tx.Type == types.TransactionTypeBonus {
				bonusNum++
			}
		}
	}
	ret.ProposalBonus = model.Param.ProposalBonus + bonusNum*model.Param.PackBonus
	if bonusTx := chain.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes()); bonusTx != nil {
		genBonus := convertBonusTransaction(bonusTx)
		genBonus.Success = true
		ret.VerifierBonus = *genBonus
	}
	return successResult(ret)
}

// MonitorBlocks monitoring platform calls block sync
func (api *GtasAPI) MonitorBlocks(begin, end uint64) (*Result, error) {
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

func (api *GtasAPI) MonitorNodeInfo() (*Result, error) {
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

func (api *GtasAPI) MonitorAllMiners() (*Result, error) {
	miners := mediator.Proc.GetAllMinerDOs()
	totalStake := uint64(0)
	maxStake := uint64(0)
	for _, m := range miners {
		if m.AbortHeight == 0 && m.NType == types.MinerTypeHeavy {
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
