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
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strings"
)

// RpcExplorerImpl provides rpc service for blockchain explorer use
type RpcExplorerImpl struct {
}

func (api *RpcExplorerImpl) Namespace() string {
	return "Explorer"
}

func (api *RpcExplorerImpl) Version() string {
	return "1"
}

// ExplorerAccount is used in the blockchain browser to query account information
func (api *RpcExplorerImpl) ExplorerAccount(hash string) (*Result, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return failResult("Wrong param format")
	}
	impl := &RpcGtasImpl{}
	return impl.ViewAccount(hash)
}

// ExplorerBlockDetail is used in the blockchain browser to query block details
func (api *RpcExplorerImpl) ExplorerBlockDetail(height uint64) (*Result, error) {
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
func (api *RpcExplorerImpl) ExplorerGroupsAfter(height uint64) (*Result, error) {
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

// ExplorerBlockReward export reward transaction by block height
func (api *RpcExplorerImpl) ExplorerBlockReward(height uint64) (*Result, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return failResult("nil block")
	}
	bh := b.Header

	ret := &ExploreBlockReward{
		ProposalID: groupsig.DeserializeID(bh.Castor).GetHexString(),
	}
	packedReward := uint64(0)
	if b.Transactions != nil {
		for _, tx := range b.Transactions {
			if tx.Type == types.TransactionTypeReward {
				block := chain.QueryBlockByHash(common.BytesToHash(tx.Data))
				status, err := chain.GetTransactionPool().GetTransactionStatus(tx.Hash)
				if err != nil && block != nil && status == types.ReceiptStatusSuccessful {
					packedReward += chain.GetRewardManager().CalculatePackedRewards(block.Header.Height)
				}
			}
		}
	}
	ret.ProposalReward = chain.GetRewardManager().CalculateCastorRewards(bh.Height) + packedReward
	ret.ProposalGasFeeReward = chain.GetRewardManager().
		CalculateGasFeeCastorRewards(bh.GasFee)
	if rewardTx := chain.GetRewardManager().GetRewardTransactionByBlockHash(bh.Hash.Bytes()); rewardTx != nil {
		genReward := convertRewardTransaction(rewardTx)
		genReward.Success = true
		ret.VerifierReward = *genReward
	}
	return successResult(ret)
}
