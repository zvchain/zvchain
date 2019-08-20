package transfer

import (
	"github.com/zvchain/zvchain/browser/models"
	"github.com/zvchain/zvchain/cmd/gzv/cli"
)

type Transfer struct {
}

func (transfer *Transfer) BlockRewardTOAccount(reward cli.ExploreBlockReward) []*models.Account {
	accounts := make([]*models.Account, 0)
	account := &models.Account{
		Address: reward.ProposalID,
		Rewards: reward.ProposalReward,
	}

	accounts = append(accounts, account)
	targets := reward.VerifierReward.TargetIDs
	for i := 0; i < len(targets); i++ {
		account := &models.Account{
			Address: targets[i].GetAddrString(),
			Rewards: reward.VerifierReward.Value,
		}
		accounts = append(accounts, account)
	}
	return accounts

}
