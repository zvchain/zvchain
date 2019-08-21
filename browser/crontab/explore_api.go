package crontab

import (
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/mediator"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
)

type Explore struct{}

type MortGage struct {
	Stake              uint64 `json:"stake"`
	ApplyHeight        uint64 `json:"apply_height"`
	Type               string `json:"type"`
	Status             string `json:"miner_status"`
	StatusUpdateHeight uint64 `json:"status_update_height"`
}

type ExploreBlockReward struct {
	ProposalID           string            `json:"proposal_id"`
	ProposalReward       uint64            `json:"proposal_reward"`
	ProposalGasFeeReward uint64            `json:"proposal_gas_fee_reward"`
	VerifierReward       RewardTransaction `json:"verifier_reward"`
	VerifierGasFeeReward uint64            `json:"verifier_gas_fee_reward"`
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

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "proposal node"
	if miner.IsVerifyRole() {
		t = "verify node"
	}
	status := "prepared"
	if miner.IsActive() {
		status = "normal"
	} else if miner.IsFrozen() {
		status = "frozen"
	}
	mg := &MortGage{
		Stake:              uint64(common.RA2TAS(miner.Stake)),
		ApplyHeight:        miner.ApplyHeight,
		Type:               t,
		Status:             status,
		StatusUpdateHeight: miner.StatusUpdateHeight,
	}
	return mg
}
func (api *Explore) GetRewardByHeight(height uint64) *ExploreBlockReward {
	chain := core.BlockChainImpl
	b := chain.QueryBlockCeil(height)
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
				receipt := chain.GetTransactionPool().GetReceipt(tx.Hash)
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
		ret.VerifierReward = *genReward
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
