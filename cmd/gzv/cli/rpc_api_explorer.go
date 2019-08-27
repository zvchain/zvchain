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
	*rpcBaseImpl
}

func (api *RpcExplorerImpl) Namespace() string {
	return "Explorer"
}

func (api *RpcExplorerImpl) Version() string {
	return "1"
}

// ExplorerAccount is used in the blockchain browser to query account information
func (api *RpcExplorerImpl) ExplorerAccount(hash string) (*Result, error) {
	if !common.ValidateAddress(strings.TrimSpace(hash)) {
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
		trans = append(trans, *convertTransaction(types.NewTransaction(tx, tx.GenHash())))
	}

	evictedReceipts := make([]*types.Receipt, 0)

	receipts := make([]*types.Receipt, len(b.Transactions))
	for i, tx := range trans {
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
	groups := api.gr.GroupsAfter(height)

	ret := make([]*Group, 0)
	for _, g := range groups {
		group := convertGroup(g)
		ret = append(ret, group)
	}
	return successResult(ret)
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
		ProposalID: groupsig.DeserializeID(bh.Castor).GetAddrString(),
	}
	packedReward := uint64(0)
	rm := chain.GetRewardManager()
	if b.Transactions != nil {
		for _, tx := range b.Transactions {
			if tx.IsReward() {
				block := chain.QueryBlockByHash(common.BytesToHash(tx.Data))
				receipt := chain.GetTransactionPool().GetReceipt(tx.GenHash())
				if receipt != nil && block != nil && receipt.Success() {
					share := rm.CalculateCastRewardShare(bh.Height, 0)
					packedReward += share.ForRewardTxPacking
				}
			}
		}
	}
	share := rm.CalculateCastRewardShare(bh.Height, bh.GasFee)
	ret.ProposalReward = share.ForBlockProposal + packedReward
	ret.ProposalGasFeeReward = share.FeeForProposer
	if rewardTx := chain.GetRewardManager().GetRewardTransactionByBlockHash(bh.Hash); rewardTx != nil {
		genReward := convertRewardTransaction(rewardTx)
		genReward.Success = true
		ret.VerifierReward = *genReward
	}
	return successResult(ret)
}
