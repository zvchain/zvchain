package crontab

import (
	"encoding/json"
	common2 "github.com/zvchain/zvchain/browser/common"
	browserlog "github.com/zvchain/zvchain/browser/log"
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/browser/util"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"time"
)

type Explore struct{}

type ExploreBlockReward struct {
	BlockHash            string
	BlockHeight          uint64
	ProposalID           string             `json:"proposal_id"`
	ProposalReward       uint64             `json:"proposal_reward"`
	ProposalGasFeeReward uint64             `json:"proposal_gas_fee_reward"`
	CurTime              time.Time          `json:"cur_time" gorm:"index"`
	VerifierReward       *RewardTransaction `json:"verifier_reward"`
	VerifierGasFeeReward uint64             `json:"verifier_gas_fee_reward"`
}

type BlockReward struct {
	MapReward         map[string]float64
	MapBlockCount     map[string]map[string]uint64
	MapMineBlockCount map[string][]uint64
}

type RewardTransaction struct {
	Hash         common.Hash   `json:"hash"`
	BlockHash    common.Hash   `json:"block_hash"`
	GroupSeed    common.Hash   `json:"group_id"`
	TargetIDs    []groupsig.ID `json:"target_ids"`
	Value        uint64        `json:"value"`
	PackFee      uint64        `json:"pack_fee"`
	StatusReport string        `json:"status_report"`
	Success      bool          `json:"success"`
}

func (api *Explore) GetPreHightRewardByHeight(height uint64) []*ExploreBlockReward {
	chain := core.BlockChainImpl
	b := chain.QueryBlockByHeight(height)
	if b == nil {
		return nil

	}
	exploreBlockReward := make([]*ExploreBlockReward, 0, 0)
	if b.Transactions != nil {
		for _, tx := range b.Transactions {
			if tx.IsReward() {
				block := chain.QueryBlockByHash(common.BytesToHash(tx.Data))
				reward := api.GetRewardByBlock(block)
				if reward != nil {
					reward.BlockHeight = block.Header.Height
					reward.BlockHash = block.Header.Hash.Hex()
					reward.CurTime = block.Header.CurTime.Local()
					exploreBlockReward = append(exploreBlockReward, reward)
				}
			}
		}
	}
	return exploreBlockReward

}

func (api *Explore) ExplorerGroupsAfter(height uint64) []*models.Group {
	groups := core.GroupManagerImpl.GroupsAfter(height)

	ret := make([]*models.Group, 0)
	for _, g := range groups {
		group := common2.ConvertGroup(g)
		ret = append(ret, group)
	}
	return ret
}

func (api Explore) isTokenContract(contractAddr common.Address) bool {
	chain := core.BlockChainImpl
	db, err := chain.LatestAccountDB()
	if err != nil {
		browserlog.BrowserLog.Error("isTokenContract: ", err)
		return false
	}
	code := db.GetCode(contractAddr)
	contract := tvm.Contract{}
	err = json.Unmarshal(code, &contract)
	if err != nil {
		browserlog.BrowserLog.Error("isTokenContract: ", err)
		return false
	}
	if util.HasTransferFunc(contract.Code) {
		symbol := db.GetData(contractAddr, []byte("symbol"))
		if len(symbol) >= 1 && symbol[0] == 's' {
			return true
		}
	}
	return false
}

func (api *Explore) GetRewardByBlock(b *types.Block) *ExploreBlockReward {
	chain := core.BlockChainImpl
	if b == nil {
		return nil
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
		genReward := convertRewardTransaction1(rewardTx)
		genReward.Success = true
		ret.VerifierReward = genReward
		ret.VerifierGasFeeReward = share.FeeForVerifier
	}
	return ret
}

func convertRewardTransaction1(tx *types.Transaction) *RewardTransaction {
	if tx.Type != types.TransactionTypeReward {
		return nil
	}
	gSeed, ids, bhash, packFee, err := mediator.Proc.MainChain.GetRewardManager().ParseRewardTransaction(tx)
	if err != nil {
		return nil
	}
	targets := make([]groupsig.ID, len(ids))
	for i, id := range ids {
		targets[i] = groupsig.DeserializeID(id)
	}
	return &RewardTransaction{
		Hash:      tx.Hash,
		BlockHash: bhash,
		GroupSeed: gSeed,
		TargetIDs: targets,
		Value:     tx.Value.Uint64(),
		PackFee:   packFee.Uint64(),
	}
}
