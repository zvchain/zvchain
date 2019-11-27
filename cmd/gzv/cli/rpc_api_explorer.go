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
	"fmt"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/group"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"math/big"
	"strings"
)

type TokenContract struct {
	ContractAddr  string            `json:"contract_addr" gorm:"index"`
	Creator       string            `json:"creator" gorm:"index"`
	Name          string            `json:"name"`
	Symbol        string            `json:"symbol"`
	Decimal       int64             `json:"decimal"`
	HolderNum     uint64            `json:"holder_num"`
	TransferTimes uint64            `json:"transfer_times"`
	TokenHolders  map[string]string `json:"token_holders"`
}

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
func (api *RpcExplorerImpl) ExplorerAccount(hash string) (*ExplorerAccount, error) {
	if !common.ValidateAddress(strings.TrimSpace(hash)) {
		return nil, fmt.Errorf("wrong param format")
	}
	impl := &RpcGzvImpl{}
	return impl.ViewAccount(hash)
}

// ExplorerBlockDetail is used in the blockchain browser to query block details
func (api *RpcExplorerImpl) ExplorerBlockDetail(height uint64) (*ExplorerBlockDetail, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return nil, fmt.Errorf("queryBlock error")
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
	return bd, nil
}

// ExplorerGroupsAfter is used in the blockchain browser to
// query groups after the specified height
func (api *RpcExplorerImpl) ExplorerGroupsAfter(height uint64) ([]*Group, error) {
	groups := api.gr.GroupsAfter(height)

	ret := make([]*Group, 0)
	for _, g := range groups {
		group := convertGroup(g)
		ret = append(ret, group)
	}
	return ret, nil
}

// ExplorerBlockReward export reward transaction by block height
func (api *RpcExplorerImpl) ExplorerBlockReward(height uint64) (*ExploreBlockReward, error) {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
	if b == nil {
		return nil, fmt.Errorf("nil block")
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
		ret.VerifierGasFeeReward = share.FeeForVerifier
	}
	return ret, nil
}

func (api *RpcExplorerImpl) ExplorerGetCandidates() (*[]ExploreCandidateList, error) {

	candidate := ExploreCandidateList{}
	candidateLists := make([]ExploreCandidateList, 0)
	candidates := group.GetCandidates()
	if candidates == nil {
		return nil, nil
	}
	for _, v := range candidates {
		candidate.ID = v.ID.ToAddress().AddrPrefixString()
		candidate.Stake = v.Stake
		candidateLists = append(candidateLists, candidate)
	}
	return &candidateLists, nil
}
func (api *RpcExplorerImpl) ExplorerTokenMsg(tokenAddr string) (*TokenContract, error) {
	if !common.ValidateAddress(strings.TrimSpace(tokenAddr)) {
		return nil, fmt.Errorf("wrong param format")
	}
	if !IsTokenContract(common.StringToAddress(tokenAddr)) {
		return nil, fmt.Errorf("this address is not a token address")
	}

	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		return nil, err
	}

	tokenContract := &TokenContract{}
	keyMap := []string{"name", "symbol", "decimal"}
	for times, key := range keyMap {
		data := db.GetData(common.StringToAddress(tokenAddr), []byte(key))
		if v, ok := tvm.VmDataConvert(data).(string); ok {
			switch times {
			case 0:
				tokenContract.Name = v
			case 1:
				tokenContract.Symbol = v
			}
		}
		if v, ok := tvm.VmDataConvert(data).(int64); ok {
			switch times {
			case 2:
				tokenContract.Decimal = v
			}
		}
	}

	iter := db.DataIterator(common.StringToAddress(tokenAddr), []byte{})
	if iter == nil {
		return nil, iter.Err
	}
	//balanceOf := make(map[string]interface{})
	tokenHolder := make(map[string]string, 0)
	for iter.Next() {
		if strings.HasPrefix(string(iter.Key[:]), "balanceOf@") {
			realAddr := strings.TrimPrefix(string(iter.Key[:]), "balanceOf@")
			if util.ValidateAddress(realAddr) {
				value := tvm.VmDataConvert(iter.Value[:])
				if value != nil {
					var valuestring string
					if value1, ok := value.(int64); ok {
						valuestring = big.NewInt(value1).String()
					} else if value2, ok := value.(*big.Int); ok {
						valuestring = value2.String()
					}
					tokenHolder[realAddr] = valuestring
				}
			}
		}
	}
	tokenContract.TokenHolders = tokenHolder
	tokenContract.HolderNum = uint64(len(tokenHolder))
	return tokenContract, err
}
